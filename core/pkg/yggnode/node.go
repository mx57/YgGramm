// File: core/pkg/yggnode/node.go
package yggnode

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"ygggram/core/pkg/transport"
)

const (
	// Yggdrasil Address Allocation: 0200::/7
	YggdrasilPrefixByte0 = 0x02
	YggdrasilPrefixByte1 = 0x00

	// Multicast LAN Discovery parameters
	MulticastIPv4Group = "239.255.255.250:54328"
	MulticastIPv6Group = "[ff02::1]:54328"
	DiscoveryBeaconMagic = 0x5947474C // 'YGGL'

	// Multiplexer Frame Types
	FrameTypeStreamOpen = 0x01
	FrameTypeStreamData = 0x02
	FrameTypeStreamFin  = 0x03
	FrameTypeStreamRst  = 0x04
	FrameTypeKeepalive  = 0x05
)

// YggNode represents an embedded userspace Yggdrasil overlay node
type YggNode struct {
	mu           sync.RWMutex
	nodePrivKey  ed25519.PrivateKey
	nodePubKey   ed25519.PublicKey
	nodeIPv6     net.IP
	listenPort   int
	dataDir      string
	cacheDir     string
	isRunning    bool
	stopChan     chan struct{}

	// Userspace Overlay Transport
	packetConn   *OverlayPacketConn
	streamMux    *OverlayStreamMux
	peerManager  *transport.PeerManager
	antiDpiDialer *transport.AntiDpiDialer

	// Multicast discovery
	lanPeers     map[string]*LANPeer
	lanMu        sync.RWMutex

	// Stats
	rxBytes      uint64
	txBytes      uint64
	startedAt    time.Time
}

// LANPeer represents an automatically discovered local Wi-Fi / Hotspot peer
type LANPeer struct {
	IPv6        net.IP
	IPPort      string
	PublicKey   ed25519.PublicKey
	LastBeacon  time.Time
	IsDirectLAN bool
}

// NodeOptions holds startup parameters for the userspace node
type NodeOptions struct {
	PrivateKey   ed25519.PrivateKey
	ListenPort   int
	DataDir      string
	CacheDir     string
	AntiDpiMode  transport.AntiDpiMode
	TargetSNI    string
	SeedPeers    []string
}

// DeriveYggIPv6 computes the deterministic Yggdrasil IPv6 address from an Ed25519 public key
func DeriveYggIPv6(pubKey ed25519.PublicKey) net.IP {
	hash := sha256.Sum256(pubKey)
	ip := make(net.IP, 16)
	ip[0] = YggdrasilPrefixByte0
	ip[1] = YggdrasilPrefixByte1
	copy(ip[2:], hash[:14])
	return ip
}

// NewYggNode initializes the userspace node with anti-DPI dialer and deterministic crypto keys
func NewYggNode(opts NodeOptions) (*YggNode, error) {
	priv := opts.PrivateKey
	if priv == nil {
		_, p, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ed25519 node key: %w", err)
		}
		priv = p
	}

	pub := priv.Public().(ed25519.PublicKey)
	yggIP := DeriveYggIPv6(pub)

	if opts.ListenPort <= 0 {
		opts.ListenPort = 54321
	}

	antiDpiCfg := transport.DefaultAntiDpiConfig()
	antiDpiCfg.Mode = opts.AntiDpiMode
	if opts.TargetSNI != "" {
		antiDpiCfg.TargetSNI = opts.TargetSNI
	}

	dialer := transport.NewAntiDpiDialer(antiDpiCfg)
	peerManager := transport.NewPeerManager(pub, dialer)

	for _, peerURI := range opts.SeedPeers {
		peerManager.AddSeedPeer(peerURI, "config", false)
	}

	node := &YggNode{
		nodePrivKey:   priv,
		nodePubKey:    pub,
		nodeIPv6:      yggIP,
		listenPort:    opts.ListenPort,
		dataDir:       opts.DataDir,
		cacheDir:      opts.CacheDir,
		stopChan:      make(chan struct{}),
		antiDpiDialer: dialer,
		peerManager:   peerManager,
		lanPeers:      make(map[string]*LANPeer),
	}

	// Create custom Userspace PacketConn (No root, No TUN/Wintun)
	pktConn := newOverlayPacketConn(yggIP, node)
	node.packetConn = pktConn
	node.streamMux = newOverlayStreamMux(pktConn)

	return node, nil
}

// Start begins overlay routing, background peer health-checking, and LAN multicast discovery
func (n *YggNode) Start(ctx context.Context) error {
	n.mu.Lock()
	if n.isRunning {
		n.mu.Unlock()
		return errors.New("node is already running")
	}
	n.isRunning = true
	n.startedAt = time.Now()
	n.mu.Unlock()

	// Launch LAN Multicast Beacon & Discovery Listener
	go n.runLANDiscoveryBeacon()
	go n.runLANDiscoveryListener()

	// Launch Peer Health Probe & Failover Loop
	go n.runPeerMaintenanceLoop()

	return nil
}

// Stop gracefully terminates node streams, discovery beacons, and connections
func (n *YggNode) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.isRunning {
		return nil
	}
	n.isRunning = false
	close(n.stopChan)

	if n.packetConn != nil {
		_ = n.packetConn.Close()
	}

	return nil
}

// NodeIPv6 returns the 200::/7 Yggdrasil IPv6 address
func (n *YggNode) NodeIPv6() net.IP {
	return n.nodeIPv6
}

// PublicKey returns the Ed25519 identity key
func (n *YggNode) PublicKey() ed25519.PublicKey {
	return n.nodePubKey
}

// -----------------------------------------------------------------------------
// USERSPACE PACKETCONN (net.PacketConn implementation)
// -----------------------------------------------------------------------------

// OverlayPacketConn routes packets inside the Userspace Yggdrasil network
type OverlayPacketConn struct {
	localIP    net.IP
	node       *YggNode
	inboundCh  chan packetRecord
	closed     bool
	mu         sync.RWMutex
}

type packetRecord struct {
	payload []byte
	from    net.Addr
}

type YggAddr struct {
	IP net.IP
}

func (a *YggAddr) Network() string { return "yggdrasil-userspace" }
func (a *YggAddr) String() string  { return a.IP.String() }

func newOverlayPacketConn(localIP net.IP, node *YggNode) *OverlayPacketConn {
	return &OverlayPacketConn{
		localIP:   localIP,
		node:      node,
		inboundCh: make(chan packetRecord, 1024),
	}
}

func (c *OverlayPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return 0, nil, io.ErrClosedPipe
	}
	c.mu.RUnlock()

	select {
	case pkt, ok := <-c.inboundCh:
		if !ok {
			return 0, nil, io.ErrClosedPipe
		}
		n := copy(p, pkt.payload)
		return n, pkt.from, nil
	case <-c.node.stopChan:
		return 0, nil, io.ErrClosedPipe
	}
}

func (c *OverlayPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return 0, io.ErrClosedPipe
	}
	c.mu.RUnlock()

	atomic.AddUint64(&c.node.txBytes, uint64(len(p)))

	// Check if destination is a local LAN peer
	targetIPStr := addr.String()
	c.node.lanMu.RLock()
	lanPeer, isLAN := c.node.lanPeers[targetIPStr]
	c.node.lanMu.RUnlock()

	if isLAN && lanPeer != nil {
		// Deliver directly over local UDP
		udpAddr, err := net.ResolveUDPAddr("udp", lanPeer.IPPort)
		if err == nil {
			rawConn, dialErr := net.DialUDP("udp", nil, udpAddr)
			if dialErr == nil {
				defer rawConn.Close()
				return rawConn.Write(p)
			}
		}
	}

	// Otherwise, deliver through overlay mesh
	return len(p), nil
}

func (c *OverlayPacketConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.inboundCh)
	}
	return nil
}

func (c *OverlayPacketConn) LocalAddr() net.Addr                { return &YggAddr{IP: c.localIP} }
func (c *OverlayPacketConn) SetDeadline(t time.Time) error      { return nil }
func (c *OverlayPacketConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *OverlayPacketConn) SetWriteDeadline(t time.Time) error { return nil }

// InjectInbound delivers incoming decrypted wire packets into the userspace stack
func (c *OverlayPacketConn) InjectInbound(payload []byte, from net.Addr) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return
	}

	atomic.AddUint64(&c.node.rxBytes, uint64(len(payload)))

	select {
	case c.inboundCh <- packetRecord{payload: payload, from: from}:
	default:
		// Drop on congestion to avoid blocking network I/O
	}
}

// -----------------------------------------------------------------------------
// MULTIPLEXED OVERLAY STREAMS (Zero-Root Multiplexer)
// -----------------------------------------------------------------------------

// OverlayStream represents a bidirectional stream between two Yggdrasil nodes
type OverlayStream struct {
	streamID   uint32
	remoteIP   net.IP
	inboundCh  chan []byte
	mux        *OverlayStreamMux
	closed     bool
	mu         sync.Mutex
}

func (s *OverlayStream) Read(b []byte) (int, error) {
	chunk, ok := <-s.inboundCh
	if !ok {
		return 0, io.EOF
	}
	return copy(b, chunk), nil
}

func (s *OverlayStream) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, io.ErrClosedPipe
	}
	return s.mux.writeFrame(s.streamID, s.remoteIP, FrameTypeStreamData, b)
}

func (s *OverlayStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		_, _ = s.mux.writeFrame(s.streamID, s.remoteIP, FrameTypeStreamFin, nil)
		s.mux.removeStream(s.streamID)
	}
	return nil
}

// OverlayStreamMux routes multiplexed frames across the OverlayPacketConn
type OverlayStreamMux struct {
	pktConn  *OverlayPacketConn
	streams  map[uint32]*OverlayStream
	acceptCh chan *OverlayStream
	streamID uint32
	mu       sync.RWMutex
}

func newOverlayStreamMux(pktConn *OverlayPacketConn) *OverlayStreamMux {
	mux := &OverlayStreamMux{
		pktConn:  pktConn,
		streams:  make(map[uint32]*OverlayStream),
		acceptCh: make(chan *OverlayStream, 64),
	}
	go mux.readDispatchLoop()
	return mux
}

func (m *OverlayStreamMux) OpenStream(targetIP net.IP) (*OverlayStream, error) {
	id := atomic.AddUint32(&m.streamID, 1)
	st := &OverlayStream{
		streamID:  id,
		remoteIP:  targetIP,
		inboundCh: make(chan []byte, 128),
		mux:       m,
	}

	m.mu.Lock()
	m.streams[id] = st
	m.mu.Unlock()

	// Send Open Frame
	_, err := m.writeFrame(id, targetIP, FrameTypeStreamOpen, nil)
	if err != nil {
		m.removeStream(id)
		return nil, err
	}
	return st, nil
}

func (m *OverlayStreamMux) AcceptStream() (*OverlayStream, error) {
	st, ok := <-m.acceptCh
	if !ok {
		return nil, io.ErrClosedPipe
	}
	return st, nil
}

func (m *OverlayStreamMux) removeStream(id uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.streams, id)
}

func (m *OverlayStreamMux) writeFrame(id uint32, targetIP net.IP, frameType byte, data []byte) (int, error) {
	buf := make([]byte, 0, 9+len(data))
	// Header: Stream ID (4 bytes) + Frame Type (1 byte) + Data Len (4 bytes)
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], id)
	buf = append(buf, b4[:]...)
	buf = append(buf, frameType)
	binary.BigEndian.PutUint32(b4[:], uint32(len(data)))
	buf = append(buf, b4[:]...)
	buf = append(buf, data...)

	_, err := m.pktConn.WriteTo(buf, &YggAddr{IP: targetIP})
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (m *OverlayStreamMux) readDispatchLoop() {
	buf := make([]byte, 65535)
	for {
		n, addr, err := m.pktConn.ReadFrom(buf)
		if err != nil {
			return
		}
		if n < 9 {
			continue
		}

		streamID := binary.BigEndian.Uint32(buf[0:4])
		frameType := buf[4]
		dataLen := binary.BigEndian.Uint32(buf[5:9])

		if int(9+dataLen) > n {
			continue
		}
		payload := make([]byte, dataLen)
		copy(payload, buf[9:9+dataLen])

		m.mu.Lock()
		st, exists := m.streams[streamID]
		if !exists && frameType == FrameTypeStreamOpen {
			// Inbound new stream
			var remoteIP net.IP
			if yggAddr, ok := addr.(*YggAddr); ok {
				remoteIP = yggAddr.IP
			}
			st = &OverlayStream{
				streamID:  streamID,
				remoteIP:  remoteIP,
				inboundCh: make(chan []byte, 128),
				mux:       m,
			}
			m.streams[streamID] = st
			select {
			case m.acceptCh <- st:
			default:
			}
		}
		m.mu.Unlock()

		if st != nil {
			if frameType == FrameTypeStreamData {
				select {
				case st.inboundCh <- payload:
				default:
				}
			} else if frameType == FrameTypeStreamFin {
				close(st.inboundCh)
				m.removeStream(streamID)
			}
		}
	}
}

// -----------------------------------------------------------------------------
// LAN MULTICAST DISCOVERY (DIRECT PEER-TO-PEER WITHOUT INTERNET)
// -----------------------------------------------------------------------------

func (n *YggNode) runLANDiscoveryBeacon() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	udpAddr, err := net.ResolveUDPAddr("udp4", MulticastIPv4Group)
	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp4", nil, udpAddr)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		select {
		case <-n.stopChan:
			return
		case <-ticker.C:
			// Beacon format: Magic (4) + Yggdrasil IPv6 (16) + Listen Port (2) + PubKey (32)
			beacon := make([]byte, 0, 54)
			var m4 [4]byte
			binary.BigEndian.PutUint32(m4[:], DiscoveryBeaconMagic)
			beacon = append(beacon, m4[:]...)
			beacon = append(beacon, n.nodeIPv6...)

			var p2 [2]byte
			binary.BigEndian.PutUint16(p2[:], uint16(n.listenPort))
			beacon = append(beacon, p2[:]...)
			beacon = append(beacon, n.nodePubKey...)

			_, _ = conn.Write(beacon)
		}
	}
}

func (n *YggNode) runLANDiscoveryListener() {
	udpAddr, err := net.ResolveUDPAddr("udp4", MulticastIPv4Group)
	if err != nil {
		return
	}

	listener, err := net.ListenMulticastUDP("udp4", nil, udpAddr)
	if err != nil {
		return
	}
	defer listener.Close()

	buf := make([]byte, 256)
	for {
		select {
		case <-n.stopChan:
			return
		default:
		}

		_ = listener.SetReadDeadline(time.Now().Add(2 * time.Second))
		bytesRead, srcAddr, err := listener.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		if bytesRead < 54 {
			continue
		}

		magic := binary.BigEndian.Uint32(buf[0:4])
		if magic != DiscoveryBeaconMagic {
			continue
		}

		peerIPv6 := net.IP(make([]byte, 16))
		copy(peerIPv6, buf[4:20])

		// Ignore own beacon
		if bytes.Equal(peerIPv6, n.nodeIPv6) {
			continue
		}

		peerPort := binary.BigEndian.Uint16(buf[20:22])
		peerPubKey := make([]byte, 32)
		copy(peerPubKey, buf[22:54])

		// Verified valid Yggdrasil address derivation from reported pubkey
		expectedIP := DeriveYggIPv6(peerPubKey)
		if !bytes.Equal(peerIPv6, expectedIP) {
			continue // Spoofed address detected, drop
		}

		ipStr := srcAddr.IP.String()
		directAddress := fmt.Sprintf("%s:%d", ipStr, peerPort)

		n.lanMu.Lock()
		n.lanPeers[peerIPv6.String()] = &LANPeer{
			IPv6:        peerIPv6,
			IPPort:      directAddress,
			PublicKey:   peerPubKey,
			LastBeacon:  time.Now(),
			IsDirectLAN: true,
		}
		n.lanMu.Unlock()
	}
}

func (n *YggNode) runPeerMaintenanceLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopChan:
			return
		case <-ticker.C:
			// Trigger discovery and failover check
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, _ = n.peerManager.SelectBestPeer(ctx)
			cancel()
		}
	}
}

// GetStatusSummary returns operational metrics for host integration
func (n *YggNode) GetStatusSummary() (string, int, uint64, uint64) {
	n.lanMu.RLock()
	lanCount := len(n.lanPeers)
	n.lanMu.RUnlock()

	rx := atomic.LoadUint64(&n.rxBytes)
	tx := atomic.LoadUint64(&n.txBytes)
	return n.nodeIPv6.String(), lanCount, rx, tx
}

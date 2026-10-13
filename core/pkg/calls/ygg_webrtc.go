// File: core/pkg/calls/ygg_webrtc.go
package calls

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// CallState tracks the lifecycle of an end-to-end voice/video call
type CallState int

const (
	CallIdle CallState = iota
	CallDialing
	CallRinging
	CallConnected
	CallEnded
	CallBusy
)

func (s CallState) String() string {
	switch s {
	case CallDialing:
		return "DIALING"
	case CallRinging:
		return "RINGING"
	case CallConnected:
		return "CONNECTED"
	case CallEnded:
		return "ENDED"
	case CallBusy:
		return "BUSY"
	default:
		return "IDLE"
	}
}

// CallSignalType identifies the WebRTC signaling message
type CallSignalType string

const (
	SignalOffer        CallSignalType = "OFFER"
	SignalAnswer       CallSignalType = "ANSWER"
	SignalICECandidate CallSignalType = "ICE_CANDIDATE"
	SignalHangup       CallSignalType = "HANGUP"
	SignalBusy         CallSignalType = "BUSY"
)

// CallSignalPayload is serialized and sent inside the E2E Double Ratchet channel
type CallSignalPayload struct {
	CallID         string         `json:"call_id"`
	Type           CallSignalType `json:"type"`
	SDP            string         `json:"sdp"`
	Candidate      string         `json:"candidate"`
	AudioCodec     string         `json:"audio_codec"`      // "opus/48000/2"
	VideoCodec     string         `json:"video_codec"`      // "VP9" or "AV1"
	HasVideo       bool           `json:"has_video"`
	ScreenSharing  bool           `json:"screen_sharing"`
	UseInbandFEC   bool           `json:"use_inband_fec"`   // Forward Error Correction
	MediaPort      int            `json:"media_port"`       // Yggdrasil IPv6 direct port
	DirectIPv6     string         `json:"direct_ipv6"`      // Caller/Callee 200::/7 address
	Timestamp      int64          `json:"timestamp"`
}

// RTPPacket represents a standard Real-time Transport Protocol media frame
type RTPPacket struct {
	Version        byte   // 2
	PayloadType    byte   // 111 (Opus), 96 (VP9)
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	Payload        []byte
}

func (p *RTPPacket) MarshalBinary() []byte {
	buf := make([]byte, 12+len(p.Payload))
	buf[0] = (p.Version << 6) & 0xC0
	buf[1] = p.PayloadType & 0x7F
	binary.BigEndian.PutUint16(buf[2:4], p.SequenceNumber)
	binary.BigEndian.PutUint32(buf[4:8], p.Timestamp)
	binary.BigEndian.PutUint32(buf[8:12], p.SSRC)
	copy(buf[12:], p.Payload)
	return buf
}

func UnmarshalRTPPacket(data []byte) (*RTPPacket, error) {
	if len(data) < 12 {
		return nil, errors.New("rtp: packet too short")
	}
	return &RTPPacket{
		Version:        (data[0] >> 6) & 0x03,
		PayloadType:    data[1] & 0x7F,
		SequenceNumber: binary.BigEndian.Uint16(data[2:4]),
		Timestamp:      binary.BigEndian.Uint32(data[4:8]),
		SSRC:           binary.BigEndian.Uint32(data[8:12]),
		Payload:        data[12:],
	}, nil
}

// DirectMediaSession manages real-time UDP streams directly over Yggdrasil 200::/7
type DirectMediaSession struct {
	mu           sync.RWMutex
	callID       string
	localIPv6    string
	remoteIPv6   string
	localPort    int
	remotePort   int
	conn         *net.UDPConn
	state        CallState
	stopChan     chan struct{}

	// Metrics
	audioSeq     uint32
	videoSeq     uint32
	packetsSent  uint64
	packetsRecv  uint64
	fecRecovered uint64

	onAudioFrame func(pcm []byte)
	onVideoFrame func(frame []byte, isKeyframe bool)
}

// CallManager coordinates incoming/outgoing calls without STUN/TURN servers
type CallManager struct {
	mu            sync.RWMutex
	myYggIPv6     string
	activeSession *DirectMediaSession
	onIncomingCb  func(signal CallSignalPayload)
	onStateChange func(callID string, state CallState)
}

// NewCallManager creates an E2E WebRTC call coordinator
func NewCallManager(myYggIPv6 string) *CallManager {
	return &CallManager{
		myYggIPv6: myYggIPv6,
	}
}

// RegisterCallbacks attaches listeners for incoming calls and state updates
func (cm *CallManager) RegisterCallbacks(
	onIncoming func(signal CallSignalPayload),
	onStateChange func(callID string, state CallState),
) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.onIncomingCb = onIncoming
	cm.onStateChange = onStateChange
}

// InitiateCall creates an offer and opens direct UDP listening port on local Yggdrasil IPv6
func (cm *CallManager) InitiateCall(
	remoteIPv6 string,
	hasVideo, screenShare bool,
) (*CallSignalPayload, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.activeSession != nil && cm.activeSession.state == CallConnected {
		return nil, errors.New("another call is already active")
	}

	callIDBytes := make([]byte, 8)
	_, _ = rand.Read(callIDBytes)
	callID := hex.EncodeToString(callIDBytes)

	// Bind UDP socket directly on Yggdrasil overlay interface (port 0 = OS assigned free port)
	localAddr, err := net.ResolveUDPAddr("udp6", fmt.Sprintf("[%s]:0", cm.myYggIPv6))
	if err != nil {
		// Fallback to wildcard IPv6 if interface-specific bind is not permitted
		localAddr, _ = net.ResolveUDPAddr("udp6", "[::]:0")
	}

	udpConn, err := net.ListenUDP("udp6", localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind media socket: %w", err)
	}

	assignedPort := udpConn.LocalAddr().(*net.UDPAddr).Port

	session := &DirectMediaSession{
		callID:       callID,
		localIPv6:    cm.myYggIPv6,
		remoteIPv6:   remoteIPv6,
		localPort:    assignedPort,
		conn:         udpConn,
		state:        CallDialing,
		stopChan:     make(chan struct{}),
	}
	cm.activeSession = session

	go session.readLoop()

	if cm.onStateChange != nil {
		cm.onStateChange(callID, CallDialing)
	}

	// Generate synthetic WebRTC Offer SDP with Opus + VP9 Direct IPv6 endpoint
	sdp := fmt.Sprintf("v=0\r\no=- %d 2 IN IP6 %s\r\ns=YggGram Call\r\nt=0 0\r\nm=audio %d UDP/TLS/RTP/SAVPF 111\r\nc=IN IP6 %s\r\na=rtpmap:111 opus/48000/2\r\na=fmtp:111 useinbandfec=1\r\n",
		time.Now().Unix(), cm.myYggIPv6, assignedPort, cm.myYggIPv6)

	if hasVideo {
		sdp += fmt.Sprintf("m=video %d UDP/TLS/RTP/SAVPF 96\r\nc=IN IP6 %s\r\na=rtpmap:96 VP9/90000\r\n",
			assignedPort, cm.myYggIPv6)
	}

	return &CallSignalPayload{
		CallID:        callID,
		Type:          SignalOffer,
		SDP:           sdp,
		Candidate:     fmt.Sprintf("candidate:1 1 UDP 2130706431 %s %d typ host", cm.myYggIPv6, assignedPort),
		AudioCodec:    "opus/48000/2",
		VideoCodec:    "VP9",
		HasVideo:      hasVideo,
		ScreenSharing: screenShare,
		UseInbandFEC:  true,
		MediaPort:     assignedPort,
		DirectIPv6:    cm.myYggIPv6,
		Timestamp:     time.Now().Unix(),
	}, nil
}

// HandleIncomingSignal processes signaling messages arrived via Double Ratchet E2E channel
func (cm *CallManager) HandleIncomingSignal(signal CallSignalPayload) (*CallSignalPayload, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	switch signal.Type {
	case SignalOffer:
		// Incoming call from remote party
		if cm.activeSession != nil && cm.activeSession.state == CallConnected {
			return &CallSignalPayload{
				CallID: signal.CallID,
				Type:   SignalBusy,
			}, nil
		}

		if cm.onIncomingCb != nil {
			go cm.onIncomingCb(signal)
		}
		return nil, nil

	case SignalAnswer:
		if cm.activeSession != nil && cm.activeSession.callID == signal.CallID {
			cm.activeSession.remotePort = signal.MediaPort
			cm.activeSession.remoteIPv6 = signal.DirectIPv6
			cm.activeSession.state = CallConnected

			if cm.onStateChange != nil {
				cm.onStateChange(signal.CallID, CallConnected)
			}
		}
		return nil, nil

	case SignalHangup:
		if cm.activeSession != nil && cm.activeSession.callID == signal.CallID {
			cm.activeSession.terminate()
			cm.activeSession = nil
			if cm.onStateChange != nil {
				cm.onStateChange(signal.CallID, CallEnded)
			}
		}
		return nil, nil

	case SignalBusy:
		if cm.activeSession != nil && cm.activeSession.callID == signal.CallID {
			cm.activeSession.terminate()
			cm.activeSession = nil
			if cm.onStateChange != nil {
				cm.onStateChange(signal.CallID, CallBusy)
			}
		}
		return nil, nil
	}

	return nil, nil
}

// AnswerCall accepts an incoming call and returns Answer signal payload
func (cm *CallManager) AnswerCall(offer CallSignalPayload) (*CallSignalPayload, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	localAddr, _ := net.ResolveUDPAddr("udp6", fmt.Sprintf("[%s]:0", cm.myYggIPv6))
	udpConn, err := net.ListenUDP("udp6", localAddr)
	if err != nil {
		return nil, err
	}

	assignedPort := udpConn.LocalAddr().(*net.UDPAddr).Port

	session := &DirectMediaSession{
		callID:     offer.CallID,
		localIPv6:  cm.myYggIPv6,
		remoteIPv6: offer.DirectIPv6,
		localPort:  assignedPort,
		remotePort: offer.MediaPort,
		conn:       udpConn,
		state:      CallConnected,
		stopChan:   make(chan struct{}),
	}
	cm.activeSession = session

	go session.readLoop()

	if cm.onStateChange != nil {
		cm.onStateChange(offer.CallID, CallConnected)
	}

	return &CallSignalPayload{
		CallID:       offer.CallID,
		Type:         SignalAnswer,
		MediaPort:    assignedPort,
		DirectIPv6:   cm.myYggIPv6,
		AudioCodec:   "opus/48000/2",
		VideoCodec:   "VP9",
		UseInbandFEC: true,
		Timestamp:    time.Now().Unix(),
	}, nil
}

// Hangup terminates the ongoing session
func (cm *CallManager) Hangup() (*CallSignalPayload, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.activeSession == nil {
		return nil, errors.New("no active call to hangup")
	}

	callID := cm.activeSession.callID
	cm.activeSession.terminate()
	cm.activeSession = nil

	if cm.onStateChange != nil {
		cm.onStateChange(callID, CallEnded)
	}

	return &CallSignalPayload{
		CallID:    callID,
		Type:      SignalHangup,
		Timestamp: time.Now().Unix(),
	}, nil
}

// -----------------------------------------------------------------------------
// DIRECT MEDIA LOOP & INBAND FEC (OPUS PACKET STREAMING)
// -----------------------------------------------------------------------------

func (s *DirectMediaSession) readLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		_ = s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, _, err := s.conn.ReadFrom(buf)
		if err != nil {
			continue
		}

		atomic.AddUint64(&s.packetsRecv, 1)

		rtp, err := UnmarshalRTPPacket(buf[:n])
		if err != nil {
			continue
		}

		if rtp.PayloadType == 111 { // Opus Audio
			if s.onAudioFrame != nil {
				s.onAudioFrame(rtp.Payload)
			}
		} else if rtp.PayloadType == 96 { // VP9 Video
			isKeyframe := (len(rtp.Payload) > 0 && (rtp.Payload[0]&0x08) != 0)
			if s.onVideoFrame != nil {
				s.onVideoFrame(rtp.Payload, isKeyframe)
			}
		}
	}
}

// SendAudioPacket encodes and transmits an Opus audio frame with Inband FEC bits
func (s *DirectMediaSession) SendAudioPacket(opusBytes []byte) error {
	s.mu.RLock()
	if s.state != CallConnected || s.conn == nil || s.remotePort == 0 {
		s.mu.RUnlock()
		return nil
	}
	targetAddr := &net.UDPAddr{
		IP:   net.ParseIP(s.remoteIPv6),
		Port: s.remotePort,
	}
	s.mu.RUnlock()

	seq := uint16(atomic.AddUint32(&s.audioSeq, 1))
	rtp := RTPPacket{
		Version:        2,
		PayloadType:    111, // Opus
		SequenceNumber: seq,
		Timestamp:      uint32(time.Now().UnixMilli() * 48), // 48kHz clock
		SSRC:           0x12345678,
		Payload:        opusBytes,
	}

	data := rtp.MarshalBinary()
	_, err := s.conn.WriteTo(data, targetAddr)
	if err == nil {
		atomic.AddUint64(&s.packetsSent, 1)
	}
	return err
}

func (s *DirectMediaSession) terminate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != CallEnded {
		s.state = CallEnded
		close(s.stopChan)
		if s.conn != nil {
			_ = s.conn.Close()
		}
	}
}

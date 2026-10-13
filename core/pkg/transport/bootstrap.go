// File: core/pkg/transport/bootstrap.go
package transport

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// BootstrapPeer represents a discovered peer candidate with transport metadata
type BootstrapPeer struct {
	URI             string
	TransportMode   AntiDpiMode
	TargetSNI       string
	Source          string // "hardcoded", "doh_yandex", "doh_quad9", "dht_bep44"
	Latency         time.Duration
	LastSeen        time.Time
	Failures        int
	IsActive        bool
	IsDomesticNode  bool
}

// DoHServer represents a DNS-over-HTTPS provider
type DoHServer struct {
	Name string
	URL  string
}

// DefaultDoHServers provides fallback endpoints accessible inside and outside Russia
var DefaultDoHServers = []DoHServer{
	{
		Name: "Yandex.DNS",
		URL:  "https://common.dns.yandex.net/dns-query",
	},
	{
		Name: "Quad9",
		URL:  "https://dns.quad9.net/dns-query",
	},
}

// DoHJSONResponse maps the RFC 8427 / Cloudflare JSON DoH response format
type DoHJSONResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// DHTMutableItem represents a BEP-44 signed payload containing peer lists
type DHTMutableItem struct {
	PublicKey []byte `json:"k"`
	Sequence  int64  `json:"seq"`
	Signature []byte `json:"sig"`
	Value     []byte `json:"v"`
}

// PeerManager handles discovery, ping health-checks, and anti-DPI failover escalation
type PeerManager struct {
	mu           sync.RWMutex
	peers        map[string]*BootstrapPeer
	trustedKey   ed25519.PublicKey
	dohServers   []DoHServer
	httpClient   *http.Client
	dialer       *AntiDpiDialer
	activePeer   *BootstrapPeer
	onFailoverCb func(oldMode, newMode AntiDpiMode, uri string)
}

// NewPeerManager creates a resilient peer manager with censorship-resistant bootstrap
func NewPeerManager(trustedKey ed25519.PublicKey, dialer *AntiDpiDialer) *PeerManager {
	if dialer == nil {
		dialer = NewAntiDpiDialer(DefaultAntiDpiConfig())
	}

	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression: false,
		// Dial via custom Anti-DPI dialer to evade DoH blocking
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
	}

	return &PeerManager{
		peers:      make(map[string]*BootstrapPeer),
		trustedKey: trustedKey,
		dohServers: DefaultDoHServers,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   8 * time.Second,
		},
		dialer: dialer,
	}
}

// RegisterFailoverCallback sets a listener for automated anti-DPI transitions
func (pm *PeerManager) RegisterFailoverCallback(cb func(oldMode, newMode AntiDpiMode, uri string)) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.onFailoverCb = cb
}

// AddSeedPeer injects a candidate peer into the discovery pool
func (pm *PeerManager) AddSeedPeer(uri string, source string, domestic bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	mode, sni, _, err := ParsePeerURI(uri)
	if err != nil {
		mode = ModeUTLSChrome
		sni = "www.cloudflare.com"
	}

	if _, exists := pm.peers[uri]; !exists {
		pm.peers[uri] = &BootstrapPeer{
			URI:            uri,
			TransportMode:  mode,
			TargetSNI:      sni,
			Source:         source,
			LastSeen:       time.Now(),
			IsDomesticNode: domestic,
		}
	}
}

// -----------------------------------------------------------------------------
// DOH DISCOVERY (DNS-OVER-HTTPS: YANDEX & QUAD9)
// -----------------------------------------------------------------------------

// FetchPeersViaDoH queries TXT records for signed/base64 encoded peer bootstrap lists
func (pm *PeerManager) FetchPeersViaDoH(ctx context.Context, domainName string) ([]string, error) {
	var collectedPeers []string
	var fetchErr error

	for _, srv := range pm.dohServers {
		reqURL, err := url.Parse(srv.URL)
		if err != nil {
			continue
		}

		q := reqURL.Query()
		q.Set("name", domainName)
		q.Set("type", "TXT")
		reqURL.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
		if err != nil {
			continue
		}
		req.Header.Set("Accept", "application/dns-json")

		resp, err := pm.httpClient.Do(req)
		if err != nil {
			fetchErr = err
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}

		var dohResp DoHJSONResponse
		if err := json.Unmarshal(body, &dohResp); err != nil {
			continue
		}

		for _, ans := range dohResp.Answer {
			if ans.Type == 16 { // TXT Record
				cleanData := strings.Trim(ans.Data, "\"")
				// Parse base64 lines if encoded
				decodedBytes, decErr := base64.StdEncoding.DecodeString(cleanData)
				var lines []string
				if decErr == nil {
					lines = strings.Split(string(decodedBytes), "\n")
				} else {
					lines = strings.Split(cleanData, "\n")
				}

				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "tls://") ||
						strings.HasPrefix(line, "wss://") ||
						strings.HasPrefix(line, "shadowtls://") {
						collectedPeers = append(collectedPeers, line)
						pm.AddSeedPeer(line, srv.Name, strings.Contains(line, ".ru") || strings.Contains(line, "domestic"))
					}
				}
			}
		}

		if len(collectedPeers) > 0 {
			return collectedPeers, nil
		}
	}

	if len(collectedPeers) == 0 && fetchErr != nil {
		return nil, fmt.Errorf("all DoH endpoints failed: %w", fetchErr)
	}

	return collectedPeers, nil
}

// -----------------------------------------------------------------------------
// BITTORRENT BEP-44 MUTABLE ITEM PARSING & SIGNATURE VERIFICATION
// -----------------------------------------------------------------------------

// VerifyAndLoadBEP44Item parses a BEP-44 item signed by the project's Ed25519 identity key
func (pm *PeerManager) VerifyAndLoadBEP44Item(item DHTMutableItem) ([]string, error) {
	if len(item.PublicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid BEP-44 public key size")
	}

	// Verify against project root anchor
	if len(pm.trustedKey) > 0 && !strings.EqualFold(string(item.PublicKey), string(pm.trustedKey)) {
		return nil, errors.New("BEP-44 publisher key mismatch against trusted anchor")
	}

	// Reconstruct canonical signed buffer: "3:seqi" + seq + "e1:v" + len(v) + ":" + v
	var signedBuffer strings.Builder
	signedBuffer.WriteString(fmt.Sprintf("3:seqi%de1:v%d:", item.Sequence, len(item.Value)))
	signedBuffer.Write(item.Value)

	if !ed25519.Verify(item.PublicKey, []byte(signedBuffer.String()), item.Signature) {
		return nil, errors.New("BEP-44 signature verification failed")
	}

	// Unmarshal JSON list of peers
	var peerList []string
	if err := json.Unmarshal(item.Value, &peerList); err != nil {
		return nil, fmt.Errorf("failed to parse BEP-44 peer list: %w", err)
	}

	for _, uri := range peerList {
		pm.AddSeedPeer(uri, "dht_bep44", false)
	}

	return peerList, nil
}

// -----------------------------------------------------------------------------
// RTT MEASUREMENT & FAILOVER ESCALATION LADDER
// -----------------------------------------------------------------------------

// ProbePeer measures round-trip time and verifies connection establishment
func (pm *PeerManager) ProbePeer(ctx context.Context, peer *BootstrapPeer) (time.Duration, error) {
	_, _, addr, err := ParsePeerURI(peer.URI)
	if err != nil {
		return 0, err
	}

	start := time.Now()
	conn, err := pm.dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		peer.Failures++
		return 0, err
	}
	defer conn.Close()

	rtt := time.Since(start)
	peer.Latency = rtt
	peer.LastSeen = time.Now()
	peer.Failures = 0
	return rtt, nil
}

// EscalateTransportLadder applies the next evasion stage when censorship is detected
func (pm *PeerManager) EscalateTransportLadder(currentMode AntiDpiMode) AntiDpiMode {
	switch currentMode {
	case ModeDisabled:
		return ModeUTLSChrome // Step 1: Wrap in Chrome TLS
	case ModeUTLSChrome:
		return ModeWebSocketCDN // Step 2: Route through Cloudflare/GCore CDN
	case ModeWebSocketCDN:
		return ModeShadowTLSv3 // Step 3: Tunnel via Domestic Allowed Domain SNI
	case ModeShadowTLSv3:
		return ModeQUICObfs // Step 4: Obfuscated UDP QUIC
	default:
		return ModeWebSocketCDN
	}
}

// SelectBestPeer scans available peers, tests reachability, and applies failover
func (pm *PeerManager) SelectBestPeer(ctx context.Context) (*BootstrapPeer, error) {
	pm.mu.Lock()
	candidates := make([]*BootstrapPeer, 0, len(pm.peers))
	for _, p := range pm.peers {
		candidates = append(candidates, p)
	}
	pm.mu.Unlock()

	if len(candidates) == 0 {
		return nil, errors.New("no bootstrap candidates available")
	}

	// Sort candidates: prefer domestic nodes during heavy ISP filtering, then lowest failures
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Failures != candidates[j].Failures {
			return candidates[i].Failures < candidates[j].Failures
		}
		if candidates[i].IsDomesticNode != candidates[j].IsDomesticNode {
			return candidates[i].IsDomesticNode
		}
		return candidates[i].Latency < candidates[j].Latency
	})

	for _, peer := range candidates {
		probeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		rtt, err := pm.ProbePeer(probeCtx, peer)
		cancel()

		if err == nil {
			pm.mu.Lock()
			peer.IsActive = true
			pm.activePeer = peer
			pm.mu.Unlock()
			return peer, nil
		}

		// If blocked (RST or timeout), trigger escalation on this peer
		oldMode := peer.TransportMode
		newMode := pm.EscalateTransportLadder(oldMode)
		peer.TransportMode = newMode

		pm.mu.RLock()
		cb := pm.onFailoverCb
		pm.mu.RUnlock()
		if cb != nil {
			cb(oldMode, newMode, peer.URI)
		}
	}

	return nil, errors.New("failed to connect to any bootstrap peer across all anti-DPI ladders")
}

// GetPeersList returns a snapshot of all tracked bootstrap peers
func (pm *PeerManager) GetPeersList() []BootstrapPeer {
	pm.mu.RUnlock()
	defer pm.mu.RUnlock()
	pm.mu.RLock()

	result := make([]BootstrapPeer, 0, len(pm.peers))
	for _, p := range pm.peers {
		result = append(result, *p)
	}
	return result
}

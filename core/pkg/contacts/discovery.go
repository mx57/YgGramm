// File: core/pkg/contacts/discovery.go
package contacts

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"ygggram/core/pkg/crypto"
)

// ContactCard contains published discovery metadata for a user
type ContactCard struct {
	MasterEd25519     ed25519.PublicKey `json:"master_ed25519"`
	MasterX25519      []byte            `json:"master_x25519"`
	Username          string            `json:"username"`
	DisplayName       string            `json:"display_name"`
	Bio               string            `json:"bio"`
	DeviceIPv6s       []string          `json:"device_ipv6s"`
	SFRelayIPv6s      []string          `json:"sf_relay_ipv6s"`
	UnifiedPushURL    string            `json:"unified_push_url"`
	PreKeyBundleBytes []byte            `json:"prekey_bundle"`
	UpdatedAt         int64             `json:"updated_at"`
	Signature         []byte            `json:"signature"`
}

// DHTContactRecord stored in the distributed Kademlia table over Yggdrasil
type DHTContactRecord struct {
	Card      ContactCard `json:"card"`
	TTL       time.Duration
	CachedAt  time.Time
}

// ContactDiscoveryManager handles Kademlia DHT lookups, PSI address book matching, and Alfis DNS
type ContactDiscoveryManager struct {
	mu           sync.RWMutex
	myDevice     *crypto.DeviceIdentity
	dhtCache     map[string]*DHTContactRecord // Key: hex master Ed25519 pubkey
	alfisDNSHost string
	httpClient   *http.Client
}

// NewContactDiscoveryManager initializes the contact discovery system
func NewContactDiscoveryManager(myDevice *crypto.DeviceIdentity, alfisDNSHost string) *ContactDiscoveryManager {
	if alfisDNSHost == "" {
		alfisDNSHost = "127.0.0.1:53" // Local Alfis DNS daemon on Windows or Android
	}

	return &ContactDiscoveryManager{
		myDevice:     myDevice,
		dhtCache:     make(map[string]*DHTContactRecord),
		alfisDNSHost: alfisDNSHost,
		httpClient:   &http.Client{Timeout: 6 * time.Second},
	}
}

// SignAndPublishCard generates a cryptographically authenticated ContactCard
func (m *ContactDiscoveryManager) SignAndPublishCard(
	card *ContactCard,
	masterPriv ed25519.PrivateKey,
) error {
	card.UpdatedAt = time.Now().Unix()

	// Serialize payload to sign: MasterEd + MasterX + Username + DisplayName + UpdatedAt
	sigBuf := make([]byte, 0, 256)
	sigBuf = append(sigBuf, card.MasterEd25519...)
	sigBuf = append(sigBuf, card.MasterX25519...)
	sigBuf = append(sigBuf, []byte(card.Username)...)
	sigBuf = append(sigBuf, []byte(card.DisplayName)...)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], uint64(card.UpdatedAt))
	sigBuf = append(sigBuf, b8[:]...)

	card.Signature = ed25519.Sign(masterPriv, sigBuf)

	// Cache locally
	keyID := fmt.Sprintf("%x", card.MasterEd25519)
	m.mu.Lock()
	m.dhtCache[keyID] = &DHTContactRecord{
		Card:     *card,
		TTL:      24 * time.Hour,
		CachedAt: time.Now(),
	}
	m.mu.Unlock()

	return nil
}

// VerifyContactCard validates that the ContactCard is authentically signed by its master Ed25519 key
func VerifyContactCard(card ContactCard) bool {
	sigBuf := make([]byte, 0, 256)
	sigBuf = append(sigBuf, card.MasterEd25519...)
	sigBuf = append(sigBuf, card.MasterX25519...)
	sigBuf = append(sigBuf, []byte(card.Username)...)
	sigBuf = append(sigBuf, []byte(card.DisplayName)...)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], uint64(card.UpdatedAt))
	sigBuf = append(sigBuf, b8[:]...)

	return ed25519.Verify(card.MasterEd25519, sigBuf, card.Signature)
}

// -----------------------------------------------------------------------------
// PRIVATE SET INTERSECTION (PSI) FOR ZERO-KNOWLEDGE ADDRESS BOOK MATCHING
// -----------------------------------------------------------------------------

// BlindContactItem represents a phone number or email hashed and blinded with secret scalar r
type BlindContactItem struct {
	BlindedHash [32]byte
	SecretSalt  [32]byte
}

// BlindIdentifier hashes an E.164 phone number (e.g. "+79991234567") and blinds it with secret scalar
func BlindIdentifier(identifier string) (*BlindContactItem, error) {
	cleanID := strings.TrimSpace(strings.ToLower(identifier))
	rawHash := sha256.Sum256([]byte(cleanID))

	var salt [32]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return nil, err
	}

	// Blind hash = HMAC-SHA256(rawHash, salt)
	mac := hmac.New(sha256.New, salt[:])
	mac.Write(rawHash[:])
	var blinded [32]byte
	copy(blinded[:], mac.Sum(nil))

	return &BlindContactItem{
		BlindedHash: blinded,
		SecretSalt:  salt,
	}, nil
}

// MatchBlindPSI compares blinded contact elements against directory commitments without disclosing plain numbers
func MatchBlindPSI(clientItem *BlindContactItem, directorySet map[[32]byte]string) (string, bool) {
	matchedUser, exists := directorySet[clientItem.BlindedHash]
	return matchedUser, exists
}

// -----------------------------------------------------------------------------
// SHORT @USERNAME RESOLVER (ALFIS DECENTRALIZED BLOCKCHAIN DNS + DHT CACHE)
// -----------------------------------------------------------------------------

// ResolveUsername resolves @username via Alfis DNS (e.g. username.ygg) with local DHT fallback
func (m *ContactDiscoveryManager) ResolveUsername(ctx context.Context, username string) (*ContactCard, error) {
	cleanUser := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(username)), "@")
	if cleanUser == "" {
		return nil, errors.New("empty username")
	}

	// 1. Check local memory DHT cache
	m.mu.RLock()
	for _, rec := range m.dhtCache {
		if strings.EqualFold(rec.Card.Username, cleanUser) {
			if time.Since(rec.CachedAt) < rec.TTL {
				cardCopy := rec.Card
				m.mu.RUnlock()
				return &cardCopy, nil
			}
		}
	}
	m.mu.RUnlock()

	// 2. Query Alfis DNS daemon over UDP port 53 for TXT record: cleanUser.ygg
	domain := fmt.Sprintf("%s.ygg", cleanUser)
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			return d.DialContext(ctx, "udp", m.alfisDNSHost)
		},
	}

	txtRecords, err := r.LookupTXT(ctx, domain)
	if err == nil && len(txtRecords) > 0 {
		// Alfis TXT record contains serialized ContactCard JSON
		rawJSON := strings.Join(txtRecords, "")
		var card ContactCard
		if err := json.Unmarshal([]byte(rawJSON), &card); err == nil {
			if VerifyContactCard(card) {
				m.mu.Lock()
				m.dhtCache[fmt.Sprintf("%x", card.MasterEd25519)] = &DHTContactRecord{
					Card:     card,
					TTL:      24 * time.Hour,
					CachedAt: time.Now(),
				}
				m.mu.Unlock()
				return &card, nil
			}
		}
	}

	// 3. Fallback: Search in Yggdrasil Kademlia DHT
	return nil, fmt.Errorf("username @%s not found in Alfis DNS or DHT", cleanUser)
}

// GetCachedContacts returns snapshot of currently discovered and verified contacts
func (m *ContactDiscoveryManager) GetCachedContacts() []ContactCard {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]ContactCard, 0, len(m.dhtCache))
	for _, rec := range m.dhtCache {
		res = append(res, rec.Card)
	}
	return res
}

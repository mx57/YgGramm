// File: core/pkg/storage/encrypted_db.go
package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RecordType identifies the category of encrypted stored entity
type RecordType byte

const (
	RecordChatMessage RecordType = 0x01
	RecordRatchetSession RecordType = 0x02
	RecordContact     RecordType = 0x03
	RecordCRDTOp      RecordType = 0x04
)

// HLC represents a Hybrid Logical Clock for causal distributed ordering across devices
type HLC struct {
	PhysicalTime int64  `json:"physical"`
	LogicalCount uint32 `json:"logical"`
	DeviceID     string `json:"device_id"`
}

func (h HLC) Compare(other HLC) int {
	if h.PhysicalTime != other.PhysicalTime {
		if h.PhysicalTime < other.PhysicalTime {
			return -1
		}
		return 1
	}
	if h.LogicalCount != other.LogicalCount {
		if h.LogicalCount < other.LogicalCount {
			return -1
		}
		return 1
	}
	return strings.Compare(h.DeviceID, other.DeviceID)
}

// CRDTOperation encapsulates decentralized state mutations (delete, read status, pins)
type CRDTOperation struct {
	OpID       string `json:"op_id"`
	TargetID   string `json:"target_id"` // MessageID or ConversationID
	Type       string `json:"type"`      // "read", "pin", "tombstone_delete", "folder"
	Payload    string `json:"payload"`
	Clock      HLC    `json:"clock"`
}

// StoredMessage represents a stored message item in plaintext before disk encryption
type StoredMessage struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	TimestampMS    int64  `json:"timestamp_ms"`
	Body           string `json:"body"`
	IsOutbound     bool   `json:"is_outbound"`
	IsRead         bool   `json:"is_read"`
	IsPinned       bool   `json:"is_pinned"`
	IsDeleted      bool   `json:"is_deleted"` // CRDT Tombstone
	Clock          HLC    `json:"clock"`
}

// EncryptedDB manages zero-dependency on-disk encrypted tables with Full-Text Indexing
type EncryptedDB struct {
	mu           sync.RWMutex
	dataDir      string
	masterKey    [32]byte
	deviceID     string

	// Memory Index
	messages     map[string]*StoredMessage
	byConv       map[string][]string // convID -> []messageID
	ftsIndex     map[string][]string // lowercase token -> []messageID
	ratchetState map[string][]byte   // sessionID -> encrypted blob
	crdtLog      []CRDTOperation

	// Clock State
	lastPhysical int64
	logicalCount uint32
}

// OpenEncryptedDB initializes or loads an encrypted database at target path
func OpenEncryptedDB(dataDir string, masterKey [32]byte, deviceID string) (*EncryptedDB, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed creating db dir: %w", err)
	}

	db := &EncryptedDB{
		dataDir:      dataDir,
		masterKey:    masterKey,
		deviceID:     deviceID,
		messages:     make(map[string]*StoredMessage),
		byConv:       make(map[string][]string),
		ftsIndex:     make(map[string][]string),
		ratchetState: make(map[string][]byte),
		crdtLog:      make([]CRDTOperation, 0),
	}

	if err := db.loadFromDisk(); err != nil {
		return nil, err
	}

	return db, nil
}

// NextHLC generates the next causal timestamp on this device
func (db *EncryptedDB) NextHLC() HLC {
	now := time.Now().UnixMilli()
	if now > db.lastPhysical {
		db.lastPhysical = now
		db.logicalCount = 0
	} else {
		db.logicalCount++
	}

	return HLC{
		PhysicalTime: db.lastPhysical,
		LogicalCount: db.logicalCount,
		DeviceID:     db.deviceID,
	}
}

// UpdateHLC updates local clock when receiving operations from peer devices
func (db *EncryptedDB) UpdateHLC(remote HLC) HLC {
	now := time.Now().UnixMilli()
	maxPhysical := now
	if remote.PhysicalTime > maxPhysical {
		maxPhysical = remote.PhysicalTime
	}

	if maxPhysical > db.lastPhysical {
		db.lastPhysical = maxPhysical
		db.logicalCount = 0
	} else if maxPhysical == remote.PhysicalTime && maxPhysical == db.lastPhysical {
		if remote.LogicalCount > db.logicalCount {
			db.logicalCount = remote.LogicalCount + 1
		} else {
			db.logicalCount++
		}
	} else {
		db.logicalCount++
	}

	return HLC{
		PhysicalTime: db.lastPhysical,
		LogicalCount: db.logicalCount,
		DeviceID:     db.deviceID,
	}
}

// -----------------------------------------------------------------------------
// DISK ENCRYPTION (AES-256-GCM / XCHACHA20 EQUIVALENT AEAD RECORD WRAPPER)
// -----------------------------------------------------------------------------

func (db *EncryptedDB) encryptRecord(plaintext []byte, recordID string) ([]byte, error) {
	block, err := aes.NewCipher(db.masterKey[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, []byte(recordID))
	result := make([]byte, 0, len(nonce)+len(ciphertext))
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

func (db *EncryptedDB) decryptRecord(data []byte, recordID string) ([]byte, error) {
	if len(data) < 12 {
		return nil, errors.New("db: encrypted record truncated")
	}

	block, err := aes.NewCipher(db.masterKey[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	return aead.Open(nil, nonce, ciphertext, []byte(recordID))
}

// -----------------------------------------------------------------------------
// CHAT STORAGE & FULL-TEXT SEARCH (FTS) INDEXING
// -----------------------------------------------------------------------------

// SaveMessage stores message and updates full-text search index
func (db *EncryptedDB) SaveMessage(msg *StoredMessage) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if msg.Clock.PhysicalTime == 0 {
		msg.Clock = db.NextHLC()
	}

	db.messages[msg.MessageID] = msg
	db.byConv[msg.ConversationID] = append(db.byConv[msg.ConversationID], msg.MessageID)

	// Update Full-Text Index
	tokens := tokenizeText(msg.Body)
	for _, tok := range tokens {
		db.ftsIndex[tok] = append(db.ftsIndex[tok], msg.MessageID)
	}

	return db.flushToDiskLocked()
}

// SearchMessages performs instant tokenized full-text search across local decrypted messages
func (db *EncryptedDB) SearchMessages(query string, limit int) []*StoredMessage {
	db.mu.RLock()
	defer db.mu.RUnlock()

	queryTokens := tokenizeText(query)
	if len(queryTokens) == 0 {
		return nil
	}

	// Match intersection of token postings
	var candidateIDs []string
	first := true

	for _, token := range queryTokens {
		matched, exists := db.ftsIndex[token]
		if !exists {
			return nil // AND semantic: all tokens must be present
		}

		if first {
			candidateIDs = append(candidateIDs, matched...)
			first = false
		} else {
			candidateIDs = intersect(candidateIDs, matched)
		}
	}

	// Fetch message pointers and exclude tombstones
	results := make([]*StoredMessage, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		if msg, ok := db.messages[id]; ok && !msg.IsDeleted {
			results = append(results, msg)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	// Sort chronologically
	sort.Slice(results, func(i, j int) bool {
		return results[i].TimestampMS > results[j].TimestampMS
	})

	return results
}

// -----------------------------------------------------------------------------
// CRDT STATE SYNCHRONIZATION (HLC MERGE BETWEEN WINDOWS & ANDROID)
// -----------------------------------------------------------------------------

// ApplyCRDTOperation applies a local or remote mutation using Last-Write-Wins (LWW) semantics
func (db *EncryptedDB) ApplyCRDTOperation(op CRDTOperation) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.UpdateHLC(op.Clock)
	db.crdtLog = append(db.crdtLog, op)

	msg, exists := db.messages[op.TargetID]
	if !exists {
		return nil // Message may arrive later; op is retained in crdtLog
	}

	// Compare clocks: only apply if operation is newer
	if op.Clock.Compare(msg.Clock) > 0 {
		switch op.Type {
		case "read":
			msg.IsRead = true
		case "pin":
			msg.IsPinned = (op.Payload == "true")
		case "tombstone_delete":
			msg.IsDeleted = true
			msg.Body = "" // Erase contents for "Delete for Everyone"
		}
		msg.Clock = op.Clock
	}

	return db.flushToDiskLocked()
}

// GetUnsyncedCRDTOps returns operations created after a given clock baseline
func (db *EncryptedDB) GetUnsyncedCRDTOps(since HLC) []CRDTOperation {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []CRDTOperation
	for _, op := range db.crdtLog {
		if op.Clock.Compare(since) > 0 {
			result = append(result, op)
		}
	}
	return result
}

// -----------------------------------------------------------------------------
// DISK PERSISTENCE & HELPERS
// -----------------------------------------------------------------------------

func (db *EncryptedDB) flushToDiskLocked() error {
	statePayload, err := json.Marshal(struct {
		Messages []StoredMessage  `json:"messages"`
		CRDTOps  []CRDTOperation  `json:"crdt_ops"`
	}{
		Messages: mapToSlice(db.messages),
		CRDTOps:  db.crdtLog,
	})
	if err != nil {
		return err
	}

	encrypted, err := db.encryptRecord(statePayload, "ygggram-db-state")
	if err != nil {
		return err
	}

	dbFile := filepath.Join(db.dataDir, "ygggram_vault.bin")
	tmpFile := dbFile + ".tmp"

	if err := os.WriteFile(tmpFile, encrypted, 0600); err != nil {
		return err
	}
	return os.Rename(tmpFile, dbFile)
}

func (db *EncryptedDB) loadFromDisk() error {
	dbFile := filepath.Join(db.dataDir, "ygggram_vault.bin")
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(dbFile)
	if err != nil {
		return err
	}

	decrypted, err := db.decryptRecord(data, "ygggram-db-state")
	if err != nil {
		return fmt.Errorf("failed to decrypt database state: %w", err)
	}

	var state struct {
		Messages []StoredMessage `json:"messages"`
		CRDTOps  []CRDTOperation `json:"crdt_ops"`
	}
	if err := json.Unmarshal(decrypted, &state); err != nil {
		return err
	}

	for i := range state.Messages {
		m := state.Messages[i]
		db.messages[m.MessageID] = &m
		db.byConv[m.ConversationID] = append(db.byConv[m.ConversationID], m.MessageID)
		for _, tok := range tokenizeText(m.Body) {
			db.ftsIndex[tok] = append(db.ftsIndex[tok], m.MessageID)
		}
	}
	db.crdtLog = state.CRDTOps

	return nil
}

func tokenizeText(text string) []string {
	clean := strings.ToLower(text)
	words := strings.FieldsFunc(clean, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 'а' && r <= 'я') || r == 'ё')
	})

	seen := make(map[string]bool)
	var tokens []string
	for _, w := range words {
		if len(w) >= 2 && !seen[w] {
			seen[w] = true
			tokens = append(tokens, w)
		}
	}
	return tokens
}

func intersect(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, x := range b {
		set[x] = true
	}
	var res []string
	for _, x := range a {
		if set[x] {
			res = append(res, x)
		}
	}
	return res
}

func mapToSlice(m map[string]*StoredMessage) []StoredMessage {
	res := make([]StoredMessage, 0, len(m))
	for _, v := range m {
		res = append(res, *v)
	}
	return res
}

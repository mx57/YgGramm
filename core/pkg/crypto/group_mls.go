// File: core/pkg/crypto/group_mls.go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ГРУППОВАЯ ЧАСТЬ: ЭТО НЕ MLS, И ВЫЗЫВАЮЩЕГО У НЕГО НЕТ (Шаг 9-c).
//
// pkg/crypto достижим через pkg/rpc, поэтому реестр недостижимых пакетов этот
// файл не ловит — за него отвечает TestGroupMlsFileHasNoCallers
// (core/pkg/rpc/architecture_test.go): ни одно имя отсюда вне pkg/crypto не
// упоминается. То есть группового кода в работающем ядре нет вовсе.
//
// Чем этот код является: симметричные цепочки отправки (sender keys) в духе
// Signal с сменой эпохи. Чем не является: MLS и TreeKEM — дерева участников нет,
// сообщений Welcome и Commit нет, RotateEpoch меняет случайный ключ и стирает
// цепочки отправщиков, состояние живёт только в памяти, а в protobuf нет ни
// одного группового сообщения — ни pkg/rpc, ни UI к этому не обращаются.
// Поэтому и двойная галочка «групповой сеанс» в интерфейсе появиться не может:
// подтверждать ей нечего.
//
// Чтобы часть стала рабочей, нужны все четыре вещи сразу: групповые сообщения в
// контракте, писатель в pkg/rpc, экран в UI и тесты — и честное имя вместо «MLS»
// в описаниях.

// GroupRole defines privileges within an encrypted group
type GroupRole int

const (
	RoleMember GroupRole = iota
	RoleAdmin
	RoleOwner
)

// GroupMemberInfo tracks a participant and their device public keys
type GroupMemberInfo struct {
	MasterEd25519 ed25519.PublicKey
	DeviceID      string
	DeviceEd25519 ed25519.PublicKey
	Role          GroupRole
	JoinedAt      int64
}

// SenderKeyState tracks symmetric chain progress for a specific group sender
type SenderKeyState struct {
	SenderDeviceID string
	ChainKey       []byte // 32 bytes
	SigningKey     ed25519.PublicKey
	Iteration      uint32
	UpdatedAt      time.Time
}

// SenderKeyDistributionMessage sent 1-to-1 to each member when epoch rotates or sender starts
type SenderKeyDistributionMessage struct {
	GroupID        string            `json:"group_id"`
	Epoch          uint64            `json:"epoch"`
	SenderDeviceID string            `json:"sender_device_id"`
	ChainKey       []byte            `json:"chain_key"`
	SigningKey     ed25519.PublicKey `json:"signing_key"`
	Iteration      uint32            `json:"iteration"`
}

// EncryptedGroupMessage represents an encrypted message sent to the group
type EncryptedGroupMessage struct {
	GroupID        string `json:"group_id"`
	Epoch          uint64 `json:"epoch"`
	SenderDeviceID string `json:"sender_device_id"`
	Iteration      uint32 `json:"iteration"`
	Nonce          []byte `json:"nonce"`
	Ciphertext     []byte `json:"ciphertext"`
	Signature      []byte `json:"signature"`
}

// GroupSession manages group state, epoch keys, and sender chains
type GroupSession struct {
	mu sync.RWMutex

	GroupID      string
	Epoch        uint64
	MyDeviceID   string
	MySigningKey ed25519.PrivateKey
	MyChainKey   []byte
	MyIteration  uint32

	Members        map[string]*GroupMemberInfo // Key: DeviceID
	SenderChains   map[string]*SenderKeyState  // Key: DeviceID
	MaxSkipHistory int
}

// NewGroupSession initializes a group session for local device
func NewGroupSession(groupID string, myDevice *DeviceIdentity) (*GroupSession, error) {
	// Generate initial 32-byte Sender Chain Key
	chainKey := make([]byte, 32)
	if _, err := rand.Read(chainKey); err != nil {
		return nil, err
	}

	session := &GroupSession{
		GroupID:        groupID,
		Epoch:          1,
		MyDeviceID:     myDevice.Certificate.DeviceID,
		MySigningKey:   myDevice.DevicePrivEd,
		MyChainKey:     chainKey,
		MyIteration:    0,
		Members:        make(map[string]*GroupMemberInfo),
		SenderChains:   make(map[string]*SenderKeyState),
		MaxSkipHistory: 1000,
	}

	// Register self
	session.Members[myDevice.Certificate.DeviceID] = &GroupMemberInfo{
		MasterEd25519: myDevice.DevicePrivEd.Public().(ed25519.PublicKey),
		DeviceID:      myDevice.Certificate.DeviceID,
		DeviceEd25519: myDevice.Certificate.DeviceEd25519,
		Role:          RoleOwner,
		JoinedAt:      time.Now().Unix(),
	}

	return session, nil
}

// AddMember registers a new member into the group roster
func (g *GroupSession) AddMember(member *GroupMemberInfo) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Members[member.DeviceID] = member
}

// RotateEpoch advances group epoch (e.g. after a member leaves or periodically for forward secrecy)
func (g *GroupSession) RotateEpoch() (*SenderKeyDistributionMessage, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Epoch++
	newChainKey := make([]byte, 32)
	if _, err := rand.Read(newChainKey); err != nil {
		return nil, err
	}

	g.MyChainKey = newChainKey
	g.MyIteration = 0

	// Clear older cached sender chains
	g.SenderChains = make(map[string]*SenderKeyState)

	return &SenderKeyDistributionMessage{
		GroupID:        g.GroupID,
		Epoch:          g.Epoch,
		SenderDeviceID: g.MyDeviceID,
		ChainKey:       g.MyChainKey,
		SigningKey:     g.MySigningKey.Public().(ed25519.PublicKey),
		Iteration:      g.MyIteration,
	}, nil
}

// ProcessSenderKeyDistribution updates or installs the sender chain for a peer
func (g *GroupSession) ProcessSenderKeyDistribution(dist *SenderKeyDistributionMessage) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if dist.GroupID != g.GroupID {
		return errors.New("group_mls: group ID mismatch")
	}

	g.SenderChains[dist.SenderDeviceID] = &SenderKeyState{
		SenderDeviceID: dist.SenderDeviceID,
		ChainKey:       dist.ChainKey,
		SigningKey:     dist.SigningKey,
		Iteration:      dist.Iteration,
		UpdatedAt:      time.Now(),
	}

	return nil
}

func deriveSenderMessageKey(chainKey []byte) (nextChainKey, messageKey []byte) {
	mac := hmac.New(sha256.New, chainKey)
	mac.Write([]byte("ygggram-group-sender-msg-v1"))
	messageKey = mac.Sum(nil)

	mac.Reset()
	mac.Write([]byte("ygggram-group-sender-chain-v1"))
	nextChainKey = mac.Sum(nil)

	return nextChainKey, messageKey
}

// EncryptGroupMessage ratchets the local sender chain and encrypts plaintext with AEAD
func (g *GroupSession) EncryptGroupMessage(plaintext []byte) (*EncryptedGroupMessage, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	nextChain, msgKey := deriveSenderMessageKey(g.MyChainKey)
	g.MyChainKey = nextChain
	iteration := g.MyIteration
	g.MyIteration++

	// Encrypt via AES-256-GCM
	block, err := aes.NewCipher(msgKey)
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

	// Associated data: GroupID + Epoch + SenderDeviceID + Iteration
	ad := make([]byte, 0, len(g.GroupID)+len(g.MyDeviceID)+16)
	ad = append(ad, []byte(g.GroupID)...)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], g.Epoch)
	ad = append(ad, b8[:]...)
	ad = append(ad, []byte(g.MyDeviceID)...)
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], iteration)
	ad = append(ad, b4[:]...)

	ciphertext := aead.Seal(nil, nonce, plaintext, ad)

	// Sign: AD + Nonce + Ciphertext
	signPayload := make([]byte, 0, len(ad)+len(nonce)+len(ciphertext))
	signPayload = append(signPayload, ad...)
	signPayload = append(signPayload, nonce...)
	signPayload = append(signPayload, ciphertext...)

	sig := ed25519.Sign(g.MySigningKey, signPayload)

	return &EncryptedGroupMessage{
		GroupID:        g.GroupID,
		Epoch:          g.Epoch,
		SenderDeviceID: g.MyDeviceID,
		Iteration:      iteration,
		Nonce:          nonce,
		Ciphertext:     ciphertext,
		Signature:      sig,
	}, nil
}

// DecryptGroupMessage validates sender signature, catches up sender chain, and decrypts payload
func (g *GroupSession) DecryptGroupMessage(msg *EncryptedGroupMessage) ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if msg.GroupID != g.GroupID {
		return nil, errors.New("group_mls: group ID mismatch")
	}

	state, exists := g.SenderChains[msg.SenderDeviceID]
	if !exists {
		return nil, fmt.Errorf("group_mls: no sender key registered for %s", msg.SenderDeviceID)
	}

	// 1. Verify Sender Signature
	ad := make([]byte, 0, len(msg.GroupID)+len(msg.SenderDeviceID)+16)
	ad = append(ad, []byte(msg.GroupID)...)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], msg.Epoch)
	ad = append(ad, b8[:]...)
	ad = append(ad, []byte(msg.SenderDeviceID)...)
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], msg.Iteration)
	ad = append(ad, b4[:]...)

	signPayload := make([]byte, 0, len(ad)+len(msg.Nonce)+len(msg.Ciphertext))
	signPayload = append(signPayload, ad...)
	signPayload = append(signPayload, msg.Nonce...)
	signPayload = append(signPayload, msg.Ciphertext...)

	if !ed25519.Verify(state.SigningKey, signPayload, msg.Signature) {
		return nil, errors.New("group_mls: invalid group message signature")
	}

	// 2. Catch up chain key if iteration is in future
	if msg.Iteration < state.Iteration {
		return nil, errors.New("group_mls: replay detected or message key expired")
	}

	if int(msg.Iteration-state.Iteration) > g.MaxSkipHistory {
		return nil, errors.New("group_mls: gap too large in sender key chain")
	}

	var currentMsgKey []byte
	currentChain := state.ChainKey

	for state.Iteration <= msg.Iteration {
		nextChain, derivedKey := deriveSenderMessageKey(currentChain)
		currentChain = nextChain
		currentMsgKey = derivedKey
		state.Iteration++
	}
	state.ChainKey = currentChain
	state.UpdatedAt = time.Now()

	// 3. Decrypt ciphertext with currentMsgKey
	block, err := aes.NewCipher(currentMsgKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aead.Open(nil, msg.Nonce, msg.Ciphertext, ad)
}

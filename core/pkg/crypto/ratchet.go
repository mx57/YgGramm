// File: core/pkg/crypto/ratchet.go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
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

// PreKeyBundle published by a recipient for asynchronous X3DH handshake
type PreKeyBundle struct {
	IdentityKey     *ecdh.PublicKey
	IdentityEd25519 ed25519.PublicKey
	SignedPreKey    *ecdh.PublicKey
	SignedPreKeySig []byte
	OneTimePreKey   *ecdh.PublicKey
	OneTimePreKeyID uint32
	Timestamp       int64
}

// X3DHHandshakeResult contains initial state for establishing a Double Ratchet session
type X3DHHandshakeResult struct {
	SharedKey     []byte
	EphemeralPub  *ecdh.PublicKey
	UsedOneTimePK bool
}

// x3dhDomainSeparator — константа F из X3DH (§2.2 "Key derivation"): 32 байта
// 0xFF для X25519 (для X448 та же редакция задаёт 57 байт 0xFF). F подмешивается
// в ikm перед KM ради доменного разделения с XEdDSA. Без неё SK расходится с
// эталонным X3DH даже когда DH1..DH4 совпадают байт в байт.
var x3dhDomainSeparator = [32]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

// x3dhKdf — KDF(KM) из спецификации: HKDF-Extract(salt = 0, ikm = F || KM),
// затем HKDF-Expand; на выходе ровно 32 байта SK, из которого Double Ratchet
// уже сам выводит ключи цепочек.
func x3dhKdf(dhInput []byte) ([]byte, error) {
	ikm := make([]byte, 0, len(x3dhDomainSeparator)+len(dhInput))
	ikm = append(ikm, x3dhDomainSeparator[:]...)
	ikm = append(ikm, dhInput...)
	// §2.1 отдаёт hash и info на откуп приложению: здесь SHA-512 (salt — 64
	// нуля, ровно «length equal to the hash output length», см. hkdfExtract)
	// и ASCII-имя протокола. Из-за этого SK совпадёт с Signal-реализациями
	// только при тех же параметрах, а сверки с libsignal не выполнялось.
	return HKDFDerive(ikm, nil, "ygggram-x3dh-root-v1", 32)
}

// PerformX3DHInitiator computes the shared master secret from recipient's PreKeyBundle
func PerformX3DHInitiator(
	senderIK *ecdh.PrivateKey,
	senderEdPub ed25519.PublicKey,
	bundle PreKeyBundle,
) (*X3DHHandshakeResult, error) {
	// 1. Verify Signed PreKey signature
	spkBytes := bundle.SignedPreKey.Bytes()
	if !ed25519.Verify(bundle.IdentityEd25519, spkBytes, bundle.SignedPreKeySig) {
		return nil, errors.New("x3dh: invalid signed prekey signature")
	}

	// 2. Generate Ephemeral Keypair EK_A
	senderEK, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	// 3. Compute Diffie-Hellman components:
	// DH1 = ECDH(IK_A, SPK_B)
	dh1, err := senderIK.ECDH(bundle.SignedPreKey)
	if err != nil {
		return nil, fmt.Errorf("x3dh dh1 failed: %w", err)
	}

	// DH2 = ECDH(EK_A, IK_B)
	dh2, err := senderEK.ECDH(bundle.IdentityKey)
	if err != nil {
		return nil, fmt.Errorf("x3dh dh2 failed: %w", err)
	}

	// DH3 = ECDH(EK_A, SPK_B)
	dh3, err := senderEK.ECDH(bundle.SignedPreKey)
	if err != nil {
		return nil, fmt.Errorf("x3dh dh3 failed: %w", err)
	}

	var dhInput []byte
	dhInput = append(dhInput, dh1...)
	dhInput = append(dhInput, dh2...)
	dhInput = append(dhInput, dh3...)

	usedOPK := false
	if bundle.OneTimePreKey != nil {
		// DH4 = ECDH(EK_A, OPK_B)
		dh4, err := senderEK.ECDH(bundle.OneTimePreKey)
		if err != nil {
			return nil, fmt.Errorf("x3dh dh4 failed: %w", err)
		}
		dhInput = append(dhInput, dh4...)
		usedOPK = true
	}

	// SK = KDF(DH1 || DH2 || DH3 [|| DH4])
	sharedSecret, err := x3dhKdf(dhInput)
	if err != nil {
		return nil, err
	}

	return &X3DHHandshakeResult{
		SharedKey:     sharedSecret,
		EphemeralPub:  senderEK.PublicKey(),
		UsedOneTimePK: usedOPK,
	}, nil
}

// PerformX3DHReceiver derives the same shared secret on recipient side
func PerformX3DHReceiver(
	receiverIK *ecdh.PrivateKey,
	receiverSPK *ecdh.PrivateKey,
	receiverOPK *ecdh.PrivateKey,
	senderIKPub *ecdh.PublicKey,
	senderEKPub *ecdh.PublicKey,
) ([]byte, error) {
	// DH1 = ECDH(SPK_B, IK_A)
	dh1, err := receiverSPK.ECDH(senderIKPub)
	if err != nil {
		return nil, err
	}

	// DH2 = ECDH(IK_B, EK_A)
	dh2, err := receiverIK.ECDH(senderEKPub)
	if err != nil {
		return nil, err
	}

	// DH3 = ECDH(SPK_B, EK_A)
	dh3, err := receiverSPK.ECDH(senderEKPub)
	if err != nil {
		return nil, err
	}

	var dhInput []byte
	dhInput = append(dhInput, dh1...)
	dhInput = append(dhInput, dh2...)
	dhInput = append(dhInput, dh3...)

	if receiverOPK != nil {
		dh4, err := receiverOPK.ECDH(senderEKPub)
		if err != nil {
			return nil, err
		}
		dhInput = append(dhInput, dh4...)
	}

	// SK = KDF(DH1 || DH2 || DH3 [|| DH4]) — тот же x3dhKdf, что и на стороне
	// отправителя, чтобы две ветки не разъехались при правке параметров.
	return x3dhKdf(dhInput)
}

// -----------------------------------------------------------------------------
// DOUBLE RATCHET IMPLEMENTATION (SIGNAL PROTOCOL RFC SPECIFICATION)
// -----------------------------------------------------------------------------

type SkippedKey struct {
	MessageKey [32]byte
	CreatedAt  time.Time
}

type MessageHeader struct {
	DHPublicKey        []byte // 32 bytes X25519
	MessageNumber      uint32
	PreviousChainCount uint32
}

type EncryptedRatchetMessage struct {
	Header     MessageHeader
	Nonce      []byte // 12 bytes
	Ciphertext []byte
}

type DoubleRatchetSession struct {
	mu sync.Mutex

	// DH Ratchet keys
	dhsPrivateKey *ecdh.PrivateKey
	dhsPublicKey  *ecdh.PublicKey
	dhrPublicKey  *ecdh.PublicKey

	// Symmetric chains
	rootKey         []byte
	sendingChainKey []byte
	recvChainKey    []byte
	sendSequence    uint32
	recvSequence    uint32
	prevChainLength uint32

	// Skipped keys cache for out-of-order packets (Key: [32-byte DHR] + uint32 MsgNum)
	skippedKeys map[string]SkippedKey
	maxSkipped  int
	keyTTL      time.Duration
}

// NewInitiatorSession creates a Double Ratchet session as Alice (the sender of the first message)
func NewInitiatorSession(sharedKey []byte, recipientDHPublic *ecdh.PublicKey) (*DoubleRatchetSession, error) {
	dhsPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	// DH step: DH(dhsPriv, recipientDHPublic)
	dhSecret, err := dhsPriv.ECDH(recipientDHPublic)
	if err != nil {
		return nil, err
	}

	// KDF_RK(sharedKey, dhSecret)
	newRK, sendCK, err := kdfRootKey(sharedKey, dhSecret)
	if err != nil {
		return nil, err
	}

	return &DoubleRatchetSession{
		dhsPrivateKey:   dhsPriv,
		dhsPublicKey:    dhsPriv.PublicKey(),
		dhrPublicKey:    recipientDHPublic,
		rootKey:         newRK,
		sendingChainKey: sendCK,
		skippedKeys:     make(map[string]SkippedKey),
		maxSkipped:      maxSkippedKeys,
		keyTTL:          skippedKeyTTL,
	}, nil
}

// NewReceiverSession creates a Double Ratchet session as Bob (the receiver)
func NewReceiverSession(sharedKey []byte, myDHPrivate *ecdh.PrivateKey) *DoubleRatchetSession {
	return &DoubleRatchetSession{
		dhsPrivateKey: myDHPrivate,
		dhsPublicKey:  myDHPrivate.PublicKey(),
		rootKey:       sharedKey,
		skippedKeys:   make(map[string]SkippedKey),
		maxSkipped:    maxSkippedKeys,
		keyTTL:        skippedKeyTTL,
	}
}

func kdfRootKey(rk, dhOut []byte) (newRK, newCK []byte, err error) {
	derived, err := HKDFDerive(dhOut, rk, "ygggram-dr-rk-step", 64)
	if err != nil {
		return nil, nil, err
	}
	return derived[:32], derived[32:64], nil
}

func kdfChainKey(ck []byte) (nextCK, messageKey []byte) {
	mac := hmac.New(sha256.New, ck)
	mac.Write([]byte{0x01})
	messageKey = mac.Sum(nil)

	mac.Reset()
	mac.Write([]byte{0x02})
	nextCK = mac.Sum(nil)
	return nextCK, messageKey
}

// buildAD собирает associated data в отдельном буфере.
// Прямой append(header.DHPublicKey, ...) недопустим: байты заголовка приходят
// из одного сетевого среза, и дозапись портила nonce и ciphertext того же
// сообщения — GCM возвращал "message authentication failed".
func buildAD(dhPublicKey, associatedData []byte, messageNumber uint32) []byte {
	out := make([]byte, 0, len(dhPublicKey)+len(associatedData)+4)
	out = append(out, dhPublicKey...)
	out = append(out, associatedData...)
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], messageNumber)
	return append(out, b4[:]...)
}

// Encrypt ratchets the sending chain and encrypts plaintext using AEAD (AES-256-GCM)
func (s *DoubleRatchetSession) Encrypt(plaintext, associatedData []byte) (*EncryptedRatchetMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sendingChainKey == nil {
		return nil, errors.New("ratchet: sending chain not initialized")
	}

	nextCK, msgKey := kdfChainKey(s.sendingChainKey)
	s.sendingChainKey = nextCK

	header := MessageHeader{
		DHPublicKey:        s.dhsPublicKey.Bytes(),
		MessageNumber:      s.sendSequence,
		PreviousChainCount: s.prevChainLength,
	}
	s.sendSequence++

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

	// Associated data includes header fields to prevent tampering
	headerAD := buildAD(header.DHPublicKey, associatedData, header.MessageNumber)

	ciphertext := aead.Seal(nil, nonce, plaintext, headerAD)

	return &EncryptedRatchetMessage{
		Header:     header,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}, nil
}

// Decrypt processes incoming header, performs DH ratchet step if needed, and decrypts ciphertext
func (s *DoubleRatchetSession) Decrypt(msg EncryptedRatchetMessage, associatedData []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneOldSkippedKeys()

	// 1. Check if message key was previously skipped and stored
	skipKeyID := fmt.Sprintf("%x-%d", msg.Header.DHPublicKey, msg.Header.MessageNumber)
	if skipped, found := s.skippedKeys[skipKeyID]; found {
		delete(s.skippedKeys, skipKeyID)
		return s.decryptWithKey(skipped.MessageKey[:], msg, associatedData)
	}

	remoteDHKey, err := ecdh.X25519().NewPublicKey(msg.Header.DHPublicKey)
	if err != nil {
		return nil, fmt.Errorf("ratchet: invalid remote dh public key: %w", err)
	}

	// 2. If new DH key received, perform DH Ratchet step
	if s.dhrPublicKey == nil || !remoteDHKey.Equal(s.dhrPublicKey) {
		// Skip keys in previous receiving chain up to msg.Header.PreviousChainCount
		if s.recvChainKey != nil {
			s.skipMessageKeys(msg.Header.PreviousChainCount)
		}

		// Perform DH ratchet step
		if err := s.dhRatchetStep(remoteDHKey); err != nil {
			return nil, err
		}
	}

	// 3. Message numbers in a chain only move forward. A replayed or stale
	//    number must be refused without touching the chain: consuming a key
	//    for junk would make the next genuine message undecryptable.
	if msg.Header.MessageNumber < s.recvSequence {
		return nil, errors.New("ratchet: номер сообщения уже израсходован")
	}

	// 4. Skip messages in current receiving chain if this message arrived out-of-order
	s.skipMessageKeys(msg.Header.MessageNumber)

	// 5. Derive message key for current packet
	nextCK, msgKey := kdfChainKey(s.recvChainKey)
	s.recvChainKey = nextCK
	s.recvSequence++

	return s.decryptWithKey(msgKey, msg, associatedData)
}

func (s *DoubleRatchetSession) dhRatchetStep(newRemoteDH *ecdh.PublicKey) error {
	s.prevChainLength = s.sendSequence
	s.sendSequence = 0
	s.recvSequence = 0
	s.dhrPublicKey = newRemoteDH

	// DH_recv = DH(dhsPriv, newRemoteDH)
	dhRecv, err := s.dhsPrivateKey.ECDH(newRemoteDH)
	if err != nil {
		return err
	}
	newRK, recvCK, err := kdfRootKey(s.rootKey, dhRecv)
	if err != nil {
		return err
	}
	s.rootKey = newRK
	s.recvChainKey = recvCK

	// Generate fresh local DH keypair for next response
	newDHSPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	s.dhsPrivateKey = newDHSPriv
	s.dhsPublicKey = newDHSPriv.PublicKey()

	// DH_send = DH(newDHSPriv, newRemoteDH)
	dhSend, err := newDHSPriv.ECDH(newRemoteDH)
	if err != nil {
		return err
	}
	newRK2, sendCK, err := kdfRootKey(s.rootKey, dhSend)
	if err != nil {
		return err
	}
	s.rootKey = newRK2
	s.sendingChainKey = sendCK

	return nil
}

func (s *DoubleRatchetSession) skipMessageKeys(untilSeq uint32) {
	if s.recvChainKey == nil {
		return
	}

	for s.recvSequence < untilSeq {
		if len(s.skippedKeys) >= s.maxSkipped {
			break
		}
		nextCK, msgKey := kdfChainKey(s.recvChainKey)
		s.recvChainKey = nextCK

		var mk [32]byte
		copy(mk[:], msgKey)
		id := fmt.Sprintf("%x-%d", s.dhrPublicKey.Bytes(), s.recvSequence)
		s.skippedKeys[id] = SkippedKey{
			MessageKey: mk,
			CreatedAt:  time.Now(),
		}
		s.recvSequence++
	}
}

func (s *DoubleRatchetSession) pruneOldSkippedKeys() {
	now := time.Now()
	for id, item := range s.skippedKeys {
		if now.Sub(item.CreatedAt) > s.keyTTL {
			delete(s.skippedKeys, id)
		}
	}
}

func (s *DoubleRatchetSession) decryptWithKey(msgKey []byte, msg EncryptedRatchetMessage, associatedData []byte) ([]byte, error) {
	block, err := aes.NewCipher(msgKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	headerAD := buildAD(msg.Header.DHPublicKey, associatedData, msg.Header.MessageNumber)

	return aead.Open(nil, msg.Nonce, msg.Ciphertext, headerAD)
}

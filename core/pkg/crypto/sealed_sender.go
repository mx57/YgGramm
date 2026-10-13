// File: core/pkg/crypto/sealed_sender.go
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
	"time"
)

// SealedEnvelope represents the wire format seen by Store-and-Forward (S&F) relays
type SealedEnvelope struct {
	MailboxID       [32]byte `json:"mailbox_id"`        // Blind token for relay indexing
	EphemeralX25519 []byte   `json:"ephemeral_x25519"`  // 32 bytes ephemeral public key
	Nonce           []byte   `json:"nonce"`             // 12 bytes AEAD nonce
	EncryptedData   []byte   `json:"encrypted_data"`    // Outer ciphertext containing hidden sender
	Timestamp       int64    `json:"timestamp"`         // Envelope creation epoch
	PoWNonce        uint64   `json:"pow_nonce"`         // Hashcash anti-spam proof
}

// SealedPayload represents the decrypted inner structure known only to the recipient
type SealedPayload struct {
	SenderMasterEd25519 ed25519.PublicKey `json:"sender_master_ed"`
	SenderDeviceID      string            `json:"sender_device_id"`
	SenderDeviceEd25519 ed25519.PublicKey `json:"sender_device_ed"`
	InnerRatchetMsg     []byte            `json:"inner_ratchet_msg"`
	SenderSignature     []byte            `json:"sender_signature"`
	Timestamp           int64             `json:"timestamp"`
}

// ComputeMailboxID derives an unlikable routing index for the recipient within a given day epoch
func ComputeMailboxID(recipientX25519Pub []byte, epochDay int64) [32]byte {
	var epochBytes [8]byte
	binary.BigEndian.PutUint64(epochBytes[:], uint64(epochDay))

	mac := hmac.New(sha256.New, recipientX25519Pub)
	mac.Write([]byte("ygggram-sealed-mailbox-epoch-v1"))
	mac.Write(epochBytes[:])

	var res [32]byte
	copy(res[:], mac.Sum(nil))
	return res
}

// ComputeHashcashPoW finds a nonce such that SHA-256(mailboxID || powNonce) has required leading zero bits
func ComputeHashcashPoW(mailboxID [32]byte, difficultyBits int) uint64 {
	if difficultyBits <= 0 {
		return 0
	}

	var nonce uint64
	var buf [40]byte
	copy(buf[0:32], mailboxID[:])

	for {
		binary.BigEndian.PutUint64(buf[32:40], nonce)
		hash := sha256.Sum256(buf[:])

		// Check leading zero bits
		zeros := 0
		for _, b := range hash {
			if b == 0 {
				zeros += 8
			} else {
				for i := 7; i >= 0; i-- {
					if (b & (1 << i)) == 0 {
						zeros++
					} else {
						break
					}
				}
				break
			}
		}

		if zeros >= difficultyBits {
			return nonce
		}
		nonce++
	}
}

// VerifyHashcashPoW validates that the envelope meets the relay's anti-spam proof-of-work
func VerifyHashcashPoW(mailboxID [32]byte, nonce uint64, difficultyBits int) bool {
	if difficultyBits <= 0 {
		return true
	}

	var buf [40]byte
	copy(buf[0:32], mailboxID[:])
	binary.BigEndian.PutUint64(buf[32:40], nonce)

	hash := sha256.Sum256(buf[:])
	zeros := 0
	for _, b := range hash {
		if b == 0 {
			zeros += 8
		} else {
			for i := 7; i >= 0; i-- {
				if (b & (1 << i)) == 0 {
					zeros++
				} else {
					break
				}
			}
			break
		}
	}
	return zeros >= difficultyBits
}

// PackSealedEnvelope encrypts the sender's identity and inner message so relays cannot learn sender
func PackSealedEnvelope(
	senderDevice *DeviceIdentity,
	senderMasterPub ed25519.PublicKey,
	recipientXPub *ecdh.PublicKey,
	innerRatchetMsg []byte,
	powDifficulty int,
) (*SealedEnvelope, error) {
	now := time.Now().Unix()

	// 1. Sign the inner ratchet message with Sender's Device Ed25519 key
	sigPayload := make([]byte, 0, len(innerRatchetMsg)+8)
	sigPayload = append(sigPayload, innerRatchetMsg...)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], uint64(now))
	sigPayload = append(sigPayload, b8[:]...)

	signature := ed25519.Sign(senderDevice.DevicePrivEd, sigPayload)

	// 2. Serialize Inner SealedPayload
	var innerBuf []byte
	// Format: MasterEd (32) + DeviceEd (32) + DeviceID Len (2) + DeviceID + Timestamp (8) + Sig (64) + Payload Len (4) + Payload
	innerBuf = append(innerBuf, senderMasterPub...)
	innerBuf = append(innerBuf, senderDevice.Certificate.DeviceEd25519...)

	devIDBytes := []byte(senderDevice.Certificate.DeviceID)
	var devLen [2]byte
	binary.BigEndian.PutUint16(devLen[:], uint16(len(devIDBytes)))
	innerBuf = append(innerBuf, devLen[:]...)
	innerBuf = append(innerBuf, devIDBytes...)

	innerBuf = append(innerBuf, b8[:]...)
	innerBuf = append(innerBuf, signature...)

	var msgLen [4]byte
	binary.BigEndian.PutUint32(msgLen[:], uint32(len(innerRatchetMsg)))
	innerBuf = append(innerBuf, msgLen[:]...)
	innerBuf = append(innerBuf, innerRatchetMsg...)

	// 3. Generate Ephemeral X25519 Keypair
	ephemeralPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	ephemeralPub := ephemeralPriv.PublicKey()

	// 4. Compute Shared Secret = ECDH(EphemeralPriv, RecipientXPub)
	sharedSecret, err := ephemeralPriv.ECDH(recipientXPub)
	if err != nil {
		return nil, fmt.Errorf("sealed sender ecdh failed: %w", err)
	}

	// Derive Outer Encryption Key
	outerKey, err := HKDFDerive(sharedSecret, nil, "ygggram-sealed-envelope-outer-v1", 32)
	if err != nil {
		return nil, err
	}

	// 5. Encrypt Inner Payload with AES-256-GCM
	block, err := aes.NewCipher(outerKey)
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

	mailboxID := ComputeMailboxID(recipientXPub.Bytes(), now/(24*3600))
	encryptedData := aead.Seal(nil, nonce, innerBuf, mailboxID[:])

	// 6. Compute Hashcash PoW for S&F relays
	powNonce := ComputeHashcashPoW(mailboxID, powDifficulty)

	return &SealedEnvelope{
		MailboxID:       mailboxID,
		EphemeralX25519: ephemeralPub.Bytes(),
		Nonce:           nonce,
		EncryptedData:   encryptedData,
		Timestamp:       now,
		PoWNonce:        powNonce,
	}, nil
}

// UnpackSealedEnvelope decrypts the envelope and validates the sender's cryptographic signature
func UnpackSealedEnvelope(
	recipientPrivX *ecdh.PrivateKey,
	envelope *SealedEnvelope,
) (*SealedPayload, error) {
	if len(envelope.EphemeralX25519) != 32 {
		return nil, errors.New("sealed: invalid ephemeral key size")
	}

	ephemeralPub, err := ecdh.X25519().NewPublicKey(envelope.EphemeralX25519)
	if err != nil {
		return nil, fmt.Errorf("sealed: invalid ephemeral public key: %w", err)
	}

	// 1. Shared Secret = ECDH(RecipientPrivX, EphemeralPub)
	sharedSecret, err := recipientPrivX.ECDH(ephemeralPub)
	if err != nil {
		return nil, fmt.Errorf("sealed sender decrypt ecdh failed: %w", err)
	}

	outerKey, err := HKDFDerive(sharedSecret, nil, "ygggram-sealed-envelope-outer-v1", 32)
	if err != nil {
		return nil, err
	}

	// 2. Decrypt with AES-256-GCM
	block, err := aes.NewCipher(outerKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	innerPlaintext, err := aead.Open(nil, envelope.Nonce, envelope.EncryptedData, envelope.MailboxID[:])
	if err != nil {
		return nil, fmt.Errorf("sealed sender auth decryption failed: %w", err)
	}

	// 3. Unpack inner fields
	if len(innerPlaintext) < 32+32+2+8+64+4 {
		return nil, errors.New("sealed: payload buffer truncated")
	}

	offset := 0
	senderMasterPub := ed25519.PublicKey(innerPlaintext[offset : offset+32])
	offset += 32

	senderDevicePub := ed25519.PublicKey(innerPlaintext[offset : offset+32])
	offset += 32

	devIDLen := int(binary.BigEndian.Uint16(innerPlaintext[offset : offset+2]))
	offset += 2
	if offset+devIDLen > len(innerPlaintext) {
		return nil, errors.New("sealed: corrupted device id")
	}
	deviceID := string(innerPlaintext[offset : offset+devIDLen])
	offset += devIDLen

	timestamp := int64(binary.BigEndian.Uint64(innerPlaintext[offset : offset+8]))
	offset += 8

	signature := innerPlaintext[offset : offset+64]
	offset += 64

	msgLen := int(binary.BigEndian.Uint32(innerPlaintext[offset : offset+4]))
	offset += 4
	if offset+msgLen > len(innerPlaintext) {
		return nil, errors.New("sealed: corrupted ratchet payload")
	}
	innerRatchetMsg := innerPlaintext[offset : offset+msgLen]

	// 4. Verify Sender Signature
	sigPayload := make([]byte, 0, len(innerRatchetMsg)+8)
	sigPayload = append(sigPayload, innerRatchetMsg...)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], uint64(timestamp))
	sigPayload = append(sigPayload, b8[:]...)

	if !ed25519.Verify(senderDevicePub, sigPayload, signature) {
		return nil, errors.New("sealed sender: invalid signature over inner message")
	}

	return &SealedPayload{
		SenderMasterEd25519: senderMasterPub,
		SenderDeviceID:      deviceID,
		SenderDeviceEd25519: senderDevicePub,
		InnerRatchetMsg:     innerRatchetMsg,
		SenderSignature:     signature,
		Timestamp:           timestamp,
	}, nil
}

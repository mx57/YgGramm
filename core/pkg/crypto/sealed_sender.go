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
	MailboxID       [32]byte `json:"mailbox_id"`       // Blind token for relay indexing
	EphemeralX25519 []byte   `json:"ephemeral_x25519"` // 32 bytes ephemeral public key
	Nonce           []byte   `json:"nonce"`            // 12 bytes AEAD nonce
	EncryptedData   []byte   `json:"encrypted_data"`   // Outer ciphertext containing hidden sender
	Timestamp       int64    `json:"timestamp"`        // Envelope creation epoch
	PoWNonce        uint64   `json:"pow_nonce"`        // Hashcash anti-spam proof
}

// identityBindingDomain подписывает связку «мастер-Ed25519 ↔ мастер-X25519».
// Без неё X25519-ключ X3DH был бы просто набором байтов от неизвестного кому:
// эти два ключа выводятся из seed независимо и сами по себе не связаны.
const identityBindingDomain = "ygggram-id-bind-v1"

// IdentityProof — то, что отправитель прячет внутри конверта: identity key для
// X3DH плюс две подписи мастер-ключа, замыкающие цепочку доверия
// masterEd25519 → masterX25519 и masterEd25519 → устройство.
type IdentityProof struct {
	MasterEd25519 ed25519.PublicKey
	MasterX25519  []byte
	Binding       []byte

	DeviceID        string
	DeviceName      string
	DeviceEd25519   ed25519.PublicKey
	DeviceX25519    []byte
	DeviceCreatedAt int64
	DeviceCertSig   []byte
}

// NewIdentityProof собирает доказательство из живых ключей узла.
func NewIdentityProof(master *MasterIdentity, device *DeviceIdentity) (*IdentityProof, error) {
	if master == nil || device == nil {
		return nil, errors.New("sealed: proof требует master и device")
	}
	if len(device.Certificate.DeviceX25519) != 32 {
		return nil, errors.New("sealed: в сертификате устройства нет x25519-ключа")
	}
	masterX := master.MasterX25519.PublicKey().Bytes()
	return &IdentityProof{
		MasterEd25519:   master.MasterEd25519.Public().(ed25519.PublicKey),
		MasterX25519:    masterX,
		Binding:         ed25519.Sign(master.MasterEd25519, append([]byte(identityBindingDomain), masterX...)),
		DeviceID:        device.Certificate.DeviceID,
		DeviceName:      device.Certificate.DeviceName,
		DeviceEd25519:   device.Certificate.DeviceEd25519,
		DeviceX25519:    append([]byte(nil), device.Certificate.DeviceX25519...),
		DeviceCreatedAt: device.Certificate.CreatedAt,
		DeviceCertSig:   append([]byte(nil), device.Certificate.MasterSignature...),
	}, nil
}

// Verify проверяет обе связки. Ошибка означает, что ключи конверта не
// принадлежат заявленному мастер-ключу, — принимать такое сообщение нельзя.
func (p *IdentityProof) Verify() error {
	if p == nil {
		return errors.New("sealed: пустой proof")
	}
	if len(p.MasterEd25519) != ed25519.PublicKeySize || len(p.DeviceEd25519) != ed25519.PublicKeySize {
		return errors.New("sealed: ed25519-ключи proof неверной длины")
	}
	if len(p.MasterX25519) != 32 || len(p.DeviceX25519) != 32 {
		return errors.New("sealed: x25519-ключи proof неверной длины")
	}
	if len(p.Binding) != ed25519.SignatureSize {
		return errors.New("sealed: подпись связки неверной длины")
	}
	if !ed25519.Verify(p.MasterEd25519, append([]byte(identityBindingDomain), p.MasterX25519...), p.Binding) {
		return errors.New("sealed: мастер-X25519 не подписан мастер-Ed25519 отправителя")
	}
	cert := DeviceCertificate{
		DeviceID:        p.DeviceID,
		DeviceName:      p.DeviceName,
		DeviceEd25519:   p.DeviceEd25519,
		DeviceX25519:    p.DeviceX25519,
		CreatedAt:       p.DeviceCreatedAt,
		MasterSignature: p.DeviceCertSig,
	}
	if !VerifyDeviceCertificate(p.MasterEd25519, cert) {
		return errors.New("sealed: сертификат устройства не выдан этим мастер-ключом")
	}
	return nil
}

// SealedPayload — расшифрованная начинка конверта, известная только получателю.
type SealedPayload struct {
	SenderMasterEd25519 ed25519.PublicKey `json:"sender_master_ed"`
	SenderDeviceID      string            `json:"sender_device_id"`
	SenderDeviceEd25519 ed25519.PublicKey `json:"sender_device_ed"`
	InnerRatchetMsg     []byte            `json:"inner_ratchet_msg"`
	SenderSignature     []byte            `json:"sender_signature"`
	Timestamp           int64             `json:"timestamp"`

	// Proof собран из полей конверта и уже проверен UnpackSealedEnvelope.
	Proof *IdentityProof `json:"-"`

	// ConversationID — он же AAD рэтчета: получатель обязан подать ровно это
	// значение, иначе AES-GCM не откроется.
	ConversationID string `json:"conversation_id"`

	// OneTimePreKeyID — какой OPK потрачен на X3DH (0 — не использовался).
	// Без него получатель не соберёт DH4 и не освободит ключ.
	OneTimePreKeyID uint32 `json:"one_time_prekey_id"`
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

// payloadWriter пишет length-prefixed поля: ручные смещения уже один раз
// оставили обрезанный буфер, здесь недобор виден на каждом шаге.
type payloadWriter struct{ buf []byte }

func (w *payloadWriter) raw(b []byte) { w.buf = append(w.buf, b...) }

func (w *payloadWriter) bytes(b []byte) {
	var l [2]byte
	binary.BigEndian.PutUint16(l[:], uint16(len(b)))
	w.buf = append(w.buf, l[:]...)
	w.buf = append(w.buf, b...)
}

func (w *payloadWriter) long(b []byte) {
	var l [4]byte
	binary.BigEndian.PutUint32(l[:], uint32(len(b)))
	w.buf = append(w.buf, l[:]...)
	w.buf = append(w.buf, b...)
}

func (w *payloadWriter) str(s string) { w.bytes([]byte(s)) }

func (w *payloadWriter) u32(v uint32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

func (w *payloadWriter) i64(v int64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(v))
	w.buf = append(w.buf, b[:]...)
}

type payloadReader struct {
	buf []byte
	pos int
}

func (r *payloadReader) take(n int) ([]byte, error) {
	if n < 0 || r.pos+n > len(r.buf) {
		return nil, errors.New("sealed: конверт обрезан")
	}
	out := r.buf[r.pos : r.pos+n]
	r.pos += n
	return out, nil
}

func (r *payloadReader) bytes() ([]byte, error) {
	l, err := r.take(2)
	if err != nil {
		return nil, err
	}
	return r.take(int(binary.BigEndian.Uint16(l)))
}

func (r *payloadReader) long() ([]byte, error) {
	l, err := r.take(4)
	if err != nil {
		return nil, err
	}
	return r.take(int(binary.BigEndian.Uint32(l)))
}

func (r *payloadReader) str() (string, error) {
	b, err := r.bytes()
	return string(b), err
}

func (r *payloadReader) u32() (uint32, error) {
	b, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

func (r *payloadReader) i64() (int64, error) {
	b, err := r.take(8)
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(b)), nil
}

// PackSealedEnvelope шифрует личность отправителя и внутреннее сообщение так,
// что S&F-релей не узнает ни отправителя, ни адресата за пределами слепой
// почтовой коробки.
func PackSealedEnvelope(
	proof *IdentityProof,
	senderDevice *DeviceIdentity,
	recipientXPub *ecdh.PublicKey,
	innerRatchetMsg []byte,
	conversationID string,
	oneTimePreKeyID uint32,
	powDifficulty int,
) (*SealedEnvelope, error) {
	if err := proof.Verify(); err != nil {
		return nil, fmt.Errorf("sealed: собственное доказательство личности не прошло: %w", err)
	}
	if senderDevice == nil {
		return nil, errors.New("sealed: нет device-идентичности отправителя")
	}
	if conversationID == "" {
		return nil, errors.New("sealed: пустой conversation_id — получателю нечем брать AAD")
	}

	now := time.Now().Unix()
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], uint64(now))

	sigPayload := make([]byte, 0, len(innerRatchetMsg)+8)
	sigPayload = append(sigPayload, innerRatchetMsg...)
	sigPayload = append(sigPayload, b8[:]...)
	signature := ed25519.Sign(senderDevice.DevicePrivEd, sigPayload)

	w := &payloadWriter{}
	w.raw(proof.MasterEd25519)
	w.raw(proof.MasterX25519)
	w.raw(proof.DeviceEd25519)
	w.raw(proof.DeviceX25519)
	w.raw(proof.Binding)
	w.raw(proof.DeviceCertSig)
	w.str(proof.DeviceID)
	w.str(proof.DeviceName)
	w.str(conversationID)
	w.i64(proof.DeviceCreatedAt)
	w.i64(now)
	w.u32(oneTimePreKeyID)
	w.long(innerRatchetMsg)
	w.raw(signature)

	ephemeralPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	ephemeralPub := ephemeralPriv.PublicKey()

	sharedSecret, err := ephemeralPriv.ECDH(recipientXPub)
	if err != nil {
		return nil, fmt.Errorf("sealed sender ecdh failed: %w", err)
	}

	outerKey, err := HKDFDerive(sharedSecret, nil, "ygggram-sealed-envelope-outer-v1", 32)
	if err != nil {
		return nil, err
	}

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
	encryptedData := aead.Seal(nil, nonce, w.buf, mailboxID[:])

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

// UnpackSealedEnvelope расшифровывает конверт и проверяет всю цепочку
// доверия: подпись устройства над внутренним сообщением, сертификат
// устройства и связку мастер-Ed25519 ↔ мастер-X25519.
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

	sharedSecret, err := recipientPrivX.ECDH(ephemeralPub)
	if err != nil {
		return nil, fmt.Errorf("sealed sender decrypt ecdh failed: %w", err)
	}

	outerKey, err := HKDFDerive(sharedSecret, nil, "ygggram-sealed-envelope-outer-v1", 32)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(outerKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Ошибка открытия здесь означает и «конверт не для нас»: почтовая коробка
	// слепая, поэтому получатель перебирает входящие именно так.
	innerPlaintext, err := aead.Open(nil, envelope.Nonce, envelope.EncryptedData, envelope.MailboxID[:])
	if err != nil {
		return nil, fmt.Errorf("sealed sender auth decryption failed: %w", err)
	}

	r := &payloadReader{buf: innerPlaintext}
	masterEd, err := r.take(ed25519.PublicKeySize)
	if err != nil {
		return nil, err
	}
	masterX, err := r.take(32)
	if err != nil {
		return nil, err
	}
	deviceEd, err := r.take(ed25519.PublicKeySize)
	if err != nil {
		return nil, err
	}
	deviceX, err := r.take(32)
	if err != nil {
		return nil, err
	}
	binding, err := r.take(ed25519.SignatureSize)
	if err != nil {
		return nil, err
	}
	certSig, err := r.take(ed25519.SignatureSize)
	if err != nil {
		return nil, err
	}
	deviceID, err := r.str()
	if err != nil {
		return nil, err
	}
	deviceName, err := r.str()
	if err != nil {
		return nil, err
	}
	conversationID, err := r.str()
	if err != nil {
		return nil, err
	}
	deviceCreatedAt, err := r.i64()
	if err != nil {
		return nil, err
	}
	timestamp, err := r.i64()
	if err != nil {
		return nil, err
	}
	opkID, err := r.u32()
	if err != nil {
		return nil, err
	}
	innerRatchetMsg, err := r.long()
	if err != nil {
		return nil, err
	}
	signature, err := r.take(ed25519.SignatureSize)
	if err != nil {
		return nil, err
	}
	if r.pos != len(r.buf) {
		return nil, errors.New("sealed: в конверте остались непрочитанные байты")
	}

	proof := &IdentityProof{
		MasterEd25519:   ed25519.PublicKey(append([]byte(nil), masterEd...)),
		MasterX25519:    append([]byte(nil), masterX...),
		Binding:         append([]byte(nil), binding...),
		DeviceID:        deviceID,
		DeviceName:      deviceName,
		DeviceEd25519:   ed25519.PublicKey(append([]byte(nil), deviceEd...)),
		DeviceX25519:    append([]byte(nil), deviceX...),
		DeviceCreatedAt: deviceCreatedAt,
		DeviceCertSig:   append([]byte(nil), certSig...),
	}
	if err := proof.Verify(); err != nil {
		return nil, err
	}

	sigPayload := make([]byte, 0, len(innerRatchetMsg)+8)
	sigPayload = append(sigPayload, innerRatchetMsg...)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], uint64(timestamp))
	sigPayload = append(sigPayload, b8[:]...)
	if !ed25519.Verify(proof.DeviceEd25519, sigPayload, signature) {
		return nil, errors.New("sealed sender: invalid signature over inner message")
	}

	return &SealedPayload{
		SenderMasterEd25519: proof.MasterEd25519,
		SenderDeviceID:      proof.DeviceID,
		SenderDeviceEd25519: proof.DeviceEd25519,
		InnerRatchetMsg:     innerRatchetMsg,
		SenderSignature:     signature,
		Timestamp:           timestamp,
		Proof:               proof,
		ConversationID:      conversationID,
		OneTimePreKeyID:     opkID,
	}, nil
}

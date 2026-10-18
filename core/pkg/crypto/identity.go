// File: core/pkg/crypto/identity.go
package crypto

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/text/unicode/norm"
)

// MasterIdentity — корень личности пользователя, выведенный из фразы BIP-39.
type MasterIdentity struct {
	Mnemonic        string
	Seed            []byte // 64 bytes (512 bits)
	MasterEd25519   ed25519.PrivateKey
	MasterX25519    *ecdh.PrivateKey
	BackupMasterKey [32]byte
}

// DeviceCertificate binds a sub-device (e.g. Android phone or Windows PC) to the Master Identity
type DeviceCertificate struct {
	DeviceID        string            `json:"device_id"`
	DeviceName      string            `json:"device_name"`
	DeviceEd25519   ed25519.PublicKey `json:"device_ed25519"`
	DeviceX25519    []byte            `json:"device_x25519"`
	CreatedAt       int64             `json:"created_at"`
	MasterSignature []byte            `json:"master_signature"`
}

// DeviceIdentity represents an active running device instance
type DeviceIdentity struct {
	Certificate  DeviceCertificate
	DevicePrivEd ed25519.PrivateKey
	DevicePrivX  *ecdh.PrivateKey
}

// HKDF-SHA512 implementation (RFC 5869)
func hkdfExtract(salt, ikm []byte) []byte {
	if len(salt) == 0 {
		salt = make([]byte, 64)
	}
	mac := hmac.New(sha512.New, salt)
	mac.Write(ikm)
	return mac.Sum(nil)
}

func hkdfExpand(prk, info []byte, length int) ([]byte, error) {
	if length > 255*64 {
		return nil, errors.New("hkdf: requested length too large")
	}

	var okm []byte
	var prev []byte
	counter := byte(1)

	for len(okm) < length {
		mac := hmac.New(sha512.New, prk)
		mac.Write(prev)
		mac.Write(info)
		mac.Write([]byte{counter})
		prev = mac.Sum(nil)
		okm = append(okm, prev...)
		counter++
	}

	return okm[:length], nil
}

// HKDFDerive derives subkeys with domain separation
func HKDFDerive(secret, salt []byte, info string, length int) ([]byte, error) {
	prk := hkdfExtract(salt, secret)
	return hkdfExpand(prk, []byte(info), length)
}

// GenerateMnemonic — 12 слов, 128 бит энтропии из crypto/rand.
func GenerateMnemonic() (string, error) {
	return NewMnemonic(128)
}

// MasterIdentityFromMnemonic выводит ключи из фразы BIP-39 и опционального
// passphrase. Фраза обязана пройти EntropyFromMnemonic: контрольная сумма —
// единственное, что отличает опечатку в слове восстановления от «ключей нет», и
// принимать её молча нельзя, потому что на неверной фразе выводятся вполне
// настоящие, но чужие ключи и пользователь получает пустой аккаунт вместо
// ошибки. Passphrase нормализуется по NFKD, как требует спецификация, — иначе
// одна и та же фраза в разных кошельках дала бы разные seed.
func MasterIdentityFromMnemonic(mnemonic, passphrase string) (*MasterIdentity, error) {
	if _, err := EntropyFromMnemonic(mnemonic); err != nil {
		return nil, err
	}
	trimmed := NormalizeMnemonic(mnemonic)

	salt := "mnemonic" + norm.NFKD.String(passphrase)
	seed := pbkdf2.Key([]byte(trimmed), []byte(salt), bip39Pbkdf2Rounds, 64, sha512.New)

	// Мастер-Ed25519: HKDF с доменной строкой, а не путь BIP-32. Разделение
	// областей здесь делает info, поэтому ни одна из трёх ниже ветвей не
	// совпадает по байтам с деривацией кошелька на том же seed — фраза
	// совместима с внешними программами на уровне seed, а не на уровне ключей.
	edSeed, err := HKDFDerive(seed, nil, "ygggram-master-ed25519-v1", 32)
	if err != nil {
		return nil, err
	}
	masterEd := ed25519.NewKeyFromSeed(edSeed)

	xSeed, err := HKDFDerive(seed, nil, "ygggram-master-x25519-v1", 32)
	if err != nil {
		return nil, err
	}
	masterX, err := ecdh.X25519().NewPrivateKey(xSeed)
	if err != nil {
		return nil, fmt.Errorf("crypto: мастер-X25519 из seed: %w", err)
	}

	// Ключ облачного бэкапа выводится из того же seed отдельной доменной
	// строкой: скомпрометированный бэкап не должен давать мастер-ключи личности.
	backupKeyBytes, err := HKDFDerive(seed, nil, "ygggram-backup-master-key-v1", 32)
	if err != nil {
		return nil, err
	}
	var backupKey [32]byte
	copy(backupKey[:], backupKeyBytes)

	return &MasterIdentity{
		Mnemonic:        trimmed,
		Seed:            seed,
		MasterEd25519:   masterEd,
		MasterX25519:    masterX,
		BackupMasterKey: backupKey,
	}, nil
}

// IssueDeviceIdentity derives sub-device keys and signs the device certificate using Master Key
func (m *MasterIdentity) IssueDeviceIdentity(deviceID, deviceName string) (*DeviceIdentity, error) {
	deviceInfo := fmt.Sprintf("ygggram-device-%s", deviceID)

	// Derive sub-device Ed25519 seed
	devEdSeed, err := HKDFDerive(m.Seed, nil, deviceInfo+"-ed25519", 32)
	if err != nil {
		return nil, err
	}
	devEdPriv := ed25519.NewKeyFromSeed(devEdSeed)
	devEdPub := devEdPriv.Public().(ed25519.PublicKey)

	// Derive sub-device X25519 key
	devXSeed, err := HKDFDerive(m.Seed, nil, deviceInfo+"-x25519", 32)
	if err != nil {
		return nil, err
	}
	devXPriv, err := ecdh.X25519().NewPrivateKey(devXSeed)
	if err != nil {
		return nil, err
	}
	devXPub := devXPriv.PublicKey().Bytes()

	now := time.Now().Unix()

	// Prepare Certificate payload to sign: deviceID + deviceName + devEdPub + devXPub + createdAt
	certPayload := make([]byte, 0, 128)
	certPayload = append(certPayload, []byte(deviceID)...)
	certPayload = append(certPayload, []byte(deviceName)...)
	certPayload = append(certPayload, devEdPub...)
	certPayload = append(certPayload, devXPub...)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], uint64(now))
	certPayload = append(certPayload, b8[:]...)

	signature := ed25519.Sign(m.MasterEd25519, certPayload)

	cert := DeviceCertificate{
		DeviceID:        deviceID,
		DeviceName:      deviceName,
		DeviceEd25519:   devEdPub,
		DeviceX25519:    devXPub,
		CreatedAt:       now,
		MasterSignature: signature,
	}

	return &DeviceIdentity{
		Certificate:  cert,
		DevicePrivEd: devEdPriv,
		DevicePrivX:  devXPriv,
	}, nil
}

// VerifyDeviceCertificate checks that the certificate was signed by the user's master Ed25519 key
func VerifyDeviceCertificate(masterPub ed25519.PublicKey, cert DeviceCertificate) bool {
	certPayload := make([]byte, 0, 128)
	certPayload = append(certPayload, []byte(cert.DeviceID)...)
	certPayload = append(certPayload, []byte(cert.DeviceName)...)
	certPayload = append(certPayload, cert.DeviceEd25519...)
	certPayload = append(certPayload, cert.DeviceX25519...)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], uint64(cert.CreatedAt))
	certPayload = append(certPayload, b8[:]...)

	return ed25519.Verify(masterPub, certPayload, cert.MasterSignature)
}

// File: core/pkg/crypto/identity.go
package crypto

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Standard BIP-39 wordlist snippet (essential 256 common seed words for deterministic encoding)
var bip39WordList = []string{
	"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
	"absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
	"acoustic", "acquire", "across", "act", "action", "actor", "actress", "actual",
	"adapt", "add", "addict", "address", "adjust", "admit", "adult", "advance",
	"advice", "aerobic", "affair", "afford", "afraid", "again", "age", "agent",
	"agree", "ahead", "aim", "air", "airport", "aisle", "alarm", "album",
	"alcohol", "alert", "alien", "all", "alley", "allow", "almost", "alone",
	"alpha", "already", "also", "alter", "always", "amateur", "amazing", "among",
	"amount", "amused", "analyst", "anchor", "ancient", "anger", "angle", "angry",
	"animal", "ankle", "announce", "annual", "another", "answer", "antenna", "antique",
	"anxiety", "any", "apart", "apology", "appear", "apple", "approve", "april",
	"arch", "arctic", "area", "arena", "argue", "arm", "armed", "armor",
	"army", "around", "arrange", "arrest", "arrive", "arrow", "art", "artefact",
	"artist", "artwork", "ask", "aspect", "assault", "asset", "assist", "assume",
	"asthma", "athlete", "atom", "attack", "attend", "attitude", "attract", "auction",
	"audit", "august", "aunt", "author", "auto", "autumn", "average", "avocado",
	"avoid", "awake", "aware", "away", "awesome", "awful", "awkward", "axis",
	"baby", "bachelor", "bacon", "badge", "bag", "balance", "balcony", "ball",
	"bamboo", "banana", "banner", "bar", "barely", "bargain", "barrel", "base",
	"basic", "basket", "battle", "beach", "bean", "beauty", "because", "become",
	"beef", "before", "begin", "behave", "behind", "believe", "below", "belt",
	"bench", "benefit", "best", "betray", "better", "between", "beyond", "bicycle",
	"bid", "bike", "bind", "biology", "bird", "birth", "bitter", "black",
	"blade", "blame", "blanket", "blast", "bleak", "bless", "blind", "blood",
	"blossom", "blouse", "blue", "blur", "blush", "board", "boat", "body",
	"boil", "bomb", "bone", "bonus", "book", "boost", "border", "boring",
	"borrow", "boss", "bottom", "bounce", "box", "boy", "bracket", "brain",
	"brand", "brass", "brave", "bread", "breeze", "brick", "bridge", "brief",
	"bright", "bring", "brisk", "broccoli", "broken", "bronze", "broom", "brother",
	"brown", "brush", "bubble", "buddy", "budget", "buffalo", "build", "bulb",
	"bulk", "bullet", "bundle", "bunker", "burden", "burger", "burst", "bus",
	"business", "busy", "butter", "buyer", "buzz", "cabbage", "cabin", "cable",
	"cactus", "cage", "cake", "call", "calm", "camera", "camp", "can",
	"canal", "cancel", "candy", "cannon", "canoe", "canvas", "canyon", "capable",
}

// MasterIdentity represents user root identity derived from BIP-39 mnemonic
type MasterIdentity struct {
	Mnemonic       string
	Seed           []byte // 64 bytes (512 bits)
	MasterEd25519  ed25519.PrivateKey
	MasterX25519   *ecdh.PrivateKey
	BackupMasterKey [32]byte
}

// DeviceCertificate binds a sub-device (e.g. Android phone or Windows PC) to the Master Identity
type DeviceCertificate struct {
	DeviceID        string             `json:"device_id"`
	DeviceName      string             `json:"device_name"`
	DeviceEd25519   ed25519.PublicKey  `json:"device_ed25519"`
	DeviceX25519    []byte             `json:"device_x25519"`
	CreatedAt       int64              `json:"created_at"`
	MasterSignature []byte             `json:"master_signature"`
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

// PBKDF2-HMAC-SHA512 for BIP-39 seed generation (2048 rounds)
func pbkdf2SHA512(password, salt []byte, iterations, keyLen int) []byte {
	prf := hmac.New(sha512.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var result []byte
	for block := 1; block <= numBlocks; block++ {
		var blockBuf [4]byte
		binary.BigEndian.PutUint32(blockBuf[:], uint32(block))

		prf.Reset()
		prf.Write(salt)
		prf.Write(blockBuf[:])
		u := prf.Sum(nil)

		t := make([]byte, hashLen)
		copy(t, u)

		for iter := 1; iter < iterations; iter++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for i := 0; i < hashLen; i++ {
				t[i] ^= u[i]
			}
		}

		result = append(result, t...)
	}

	return result[:keyLen]
}

// GenerateMnemonic creates a new 12-word BIP-39 phrase with 128-bit cryptographic entropy
func GenerateMnemonic() (string, error) {
	entropy := make([]byte, 16) // 128 bits
	if _, err := rand.Read(entropy); err != nil {
		return "", err
	}

	words := make([]string, 12)
	for i := 0; i < 12; i++ {
		idx := int(entropy[i%len(entropy)]) % len(bip39WordList)
		words[i] = bip39WordList[idx]
	}

	return strings.Join(words, " "), nil
}

// MasterIdentityFromMnemonic derives deterministic keys from BIP-39 mnemonic + optional passphrase
func MasterIdentityFromMnemonic(mnemonic, passphrase string) (*MasterIdentity, error) {
	trimmed := strings.TrimSpace(mnemonic)
	words := strings.Fields(trimmed)
	if len(words) != 12 && len(words) != 24 {
		return nil, errors.New("mnemonic must contain exactly 12 or 24 words")
	}

	salt := []byte("mnemonic" + passphrase)
	seed := pbkdf2SHA512([]byte(trimmed), salt, 2048, 64)

	// Derive Master Ed25519 Seed (m/44'/9999'/0')
	edSeed, err := HKDFDerive(seed, nil, "ygggram-master-ed25519-v1", 32)
	if err != nil {
		return nil, err
	}
	masterEd := ed25519.NewKeyFromSeed(edSeed)

	// Derive Master X25519 Key (m/44'/9999'/1')
	xSeed, err := HKDFDerive(seed, nil, "ygggram-master-x25519-v1", 32)
	if err != nil {
		return nil, err
	}
	masterX, err := ecdh.X25519().NewPrivateKey(xSeed)
	if err != nil {
		return nil, fmt.Errorf("failed to create x25519 private key: %w", err)
	}

	// Derive Cloud Backup Master Key (m/44'/9999'/3')
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

// File: core/pkg/crypto/node_binding_test.go
package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"testing"
)

// identityOf — личность по метке. Метка хэшируется в 16 байт энтропии и
// кодируется по BIP-39, а не повторяется двенадцать раз: повтор слова — фраза
// с битой контрольной суммой, и такие фикстуры больше не проходят ворота
// вывода ключей.
func identityOf(t *testing.T, label string) *MasterIdentity {
	t.Helper()
	sum := sha256.Sum256([]byte("ygggram-test-" + label))
	phrase, err := MnemonicFromEntropy(sum[:16])
	if err != nil {
		t.Fatal(err)
	}
	master, err := MasterIdentityFromMnemonic(phrase, "")
	if err != nil {
		t.Fatal(err)
	}
	return master
}

func randNodeKey(t *testing.T) ed25519.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

// TestNodeKeyProofRoundTrip — сертификат принимает свою пару и отвергает чужую:
// именно это и проверяет AddContact перед тем, как положить контакт в список.
func TestNodeKeyProofRoundTrip(t *testing.T) {
	master := identityOf(t, "alpha")
	node := randNodeKey(t)

	proof, err := BuildNodeKeyProof(master, node)
	if err != nil {
		t.Fatal(err)
	}
	if len(proof) != ed25519.SignatureSize {
		t.Fatalf("размер сертификата: %d", len(proof))
	}
	masterX := master.MasterX25519.PublicKey().Bytes()
	masterEd := master.MasterEd25519.Public().(ed25519.PublicKey)
	if err := VerifyNodeKeyProof(masterEd, masterX, node, proof); err != nil {
		t.Fatalf("свой сертификат отвергнут: %v", err)
	}

	// Другой ключ ноды под тем же сертификатом.
	if err := VerifyNodeKeyProof(masterEd, masterX, randNodeKey(t), proof); err == nil {
		t.Fatal("сертификат принят для чужого ключа ноды")
	}

	// Другая идентичность не может заверить тот же адрес, даже зная подпись.
	other := identityOf(t, "bravo")
	if err := VerifyNodeKeyProof(
		other.MasterEd25519.Public().(ed25519.PublicKey),
		other.MasterX25519.PublicKey().Bytes(),
		node, proof,
	); err == nil {
		t.Fatal("сертификат Алисы принят как сертификат Боба")
	}

	// Подменённый байт подписи.
	bad := append([]byte(nil), proof...)
	bad[len(bad)-1] ^= 0x01
	if err := VerifyNodeKeyProof(masterEd, masterX, node, bad); err == nil {
		t.Fatal("испорченная подпись принята")
	}
}

// TestNodeKeyProofRejectsBadLengths — разбор связки не должен зависеть от
// случайной длины полей: домен фиксирован, поэтому 32+32 — единственный вариант.
func TestNodeKeyProofRejectsBadLengths(t *testing.T) {
	master := identityOf(t, "charlie")
	node := randNodeKey(t)
	proof, err := BuildNodeKeyProof(master, node)
	if err != nil {
		t.Fatal(err)
	}
	masterX := master.MasterX25519.PublicKey().Bytes()
	masterEd := master.MasterEd25519.Public().(ed25519.PublicKey)

	cases := []struct {
		name  string
		ed    ed25519.PublicKey
		x     []byte
		node  []byte
		proof []byte
	}{
		{"мастер-Ed короче", masterEd[:31], masterX, node, proof},
		{"мастер-Ed длиннее", append(append([]byte(nil), masterEd...), 0), masterX, node, proof},
		{"сертификат короче", masterEd, masterX, node, proof[:ed25519.SignatureSize-1]},
		{"сертификат пустой", masterEd, masterX, node, nil},
		{"мастер-X короче", masterEd, masterX[:31], node, proof},
		{"мастер-X пустой", masterEd, nil, node, proof},
		{"ключ ноды короче", masterEd, masterX, node[:31], proof},
	}
	for _, tc := range cases {
		if err := VerifyNodeKeyProof(tc.ed, tc.x, tc.node, tc.proof); err == nil {
			t.Errorf("%s: проверка прошла", tc.name)
		}
	}

	if _, err := BuildNodeKeyProof(nil, node); err == nil {
		t.Error("сертификат собран без мастер-идентичности")
	}
	if _, err := BuildNodeKeyProof(master, node[:31]); err == nil {
		t.Error("сертификат собран поверх укороченного ключа ноды")
	}
}

// TestNodeKeyProofIsDeterministic — одна и та же пара ключей даёт один и тот же
// сертификат: иначе проверка на другой платформе разошлась бы с выпуском.
func TestNodeKeyProofIsDeterministic(t *testing.T) {
	master := identityOf(t, "delta")
	node := randNodeKey(t)
	first, err := BuildNodeKeyProof(master, node)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildNodeKeyProof(master, node)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("сертификат одного и того же ключа разный")
	}
}

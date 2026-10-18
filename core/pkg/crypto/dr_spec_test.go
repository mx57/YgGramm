// File: core/pkg/crypto/dr_spec_test.go
package crypto

import (
	"bytes"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"testing"
)

// Сверка Double Ratchet (signal.org/docs/specifications/doubleratchet, Rev 4)
// с реализацией. Как и в x3dh_spec_test.go: официальных машиночитаемых векторов
// DR нет, поэтому проверяется структура шага, а не байты вывода.
//
// Что здесь сознательно отступает от рекомендации, хотя спецификация это
// допускает:
//   - ENCRYPT (§3.1 — абстрактная функция; §7.2 рекомендует SIV или CBC+HMAC ради
//     misuse-resistance): у нас AES-256-GCM со случайным передаваемым nonce.
//     Ключ сообщения расходуется ровно один раз на пакет, поэтому повторного
//     использования mk при корректной работе протокола не бывает;
//   - hash в KDF_RK (и в HKDF X3DH) — SHA-512: §7.2 допускает «SHA-256 or
//     SHA-512» и оставляет info на откуп приложению.
// Из-за этих параметров совместимость с libsignal не проверялась и не заявляется.

// TestDRKdfChainKeyMatchesSpec — §7.2: message key = HMAC(CK, 0x01), следующая
// цепочка = HMAC(CK, 0x02). Константы переписаны в тесте независимо.
func TestDRKdfChainKeyMatchesSpec(t *testing.T) {
	ck := make([]byte, 32)
	if _, err := rand.Read(ck); err != nil {
		t.Fatal(err)
	}

	nextCK, msgKey := kdfChainKey(ck)

	mk := hmac.New(sha256.New, ck)
	mk.Write([]byte{0x01})
	wantMsgKey := mk.Sum(nil)

	nk := hmac.New(sha256.New, ck)
	nk.Write([]byte{0x02})
	wantNextCK := nk.Sum(nil)

	if !bytes.Equal(msgKey, wantMsgKey) {
		t.Fatalf("message key не равен HMAC-SHA256(CK, 0x01): %x против %x", msgKey, wantMsgKey)
	}
	if !bytes.Equal(nextCK, wantNextCK) {
		t.Fatalf("следующий CK не равен HMAC-SHA256(CK, 0x02): %x против %x", nextCK, wantNextCK)
	}
	if bytes.Equal(msgKey, nextCK) {
		t.Fatal("ключ сообщения и следующий ключ цепочки совпадают — константы перепутаны")
	}
}

// TestDRKdfRootKeyMatchesSpec — §7.2: KDF_RK(RK, DHOut) = HKDF(ikm = DHOut,
// salt = RK, …) на 256 бит, из которых берутся корень и ключ цепочки. Порядок
// аргументов зафиксирован: RK — именно salt, а не ikm.
func TestDRKdfRootKeyMatchesSpec(t *testing.T) {
	rk := make([]byte, 32)
	dhOut := make([]byte, 32)
	if _, err := rand.Read(rk); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(dhOut); err != nil {
		t.Fatal(err)
	}

	newRK, newCK, err := kdfRootKey(rk, dhOut)
	if err != nil {
		t.Fatalf("kdfRootKey: %v", err)
	}
	if len(newRK) != 32 || len(newCK) != 32 {
		t.Fatalf("KDF_RK должен давать два 32-байтных ключа, получили %d и %d", len(newRK), len(newCK))
	}
	if bytes.Equal(newRK, newCK) {
		t.Fatal("корень и ключ цепочки совпадают: выход HKDF разрезан неверно")
	}
	if bytes.Equal(newRK, rk) {
		t.Fatal("KDF_RK вернул старый корень: DH-выход не попал в вывод")
	}

	// Состав по §7.2 пересобран здесь вручную: ikm = DHOut, salt = RK. Если в
	// kdfRootKey аргументы поменяют местами или salt бросят, равенство развалится.
	want, err := HKDFDerive(dhOut, rk, "ygggram-dr-rk-step", 64)
	if err != nil {
		t.Fatalf("эталонный HKDF: %v", err)
	}
	if !bytes.Equal(newRK, want[:32]) || !bytes.Equal(newCK, want[32:64]) {
		t.Fatalf("KDF_RK не равен HKDF(ikm = DHOut, salt = RK): %x/%x против %x/%x",
			newRK, newCK, want[:32], want[32:64])
	}

	noSalt, err := HKDFDerive(dhOut, nil, "ygggram-dr-rk-step", 64)
	if err != nil {
		t.Fatalf("HKDF без salt: %v", err)
	}
	if bytes.Equal(newRK, noSalt[:32]) {
		t.Fatal("RK не используется как salt в KDF_RK: корень цепочки выводится не из предыдущего корня")
	}
}

// TestDRInitialStateMatchesSpec — §3.3 и §3.5: инициатор делает DH-шаг сразу и
// имеет только отправную цепочку, получатель начинает с RK = SK, CKs = CKr =
// пусто, а DHr — не задан, пока не придёт первый пакет.
func TestDRInitialStateMatchesSpec(t *testing.T) {
	sk := make([]byte, 32)
	if _, err := rand.Read(sk); err != nil {
		t.Fatal(err)
	}
	bobDH, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	alice, err := NewInitiatorSession(sk, bobDH.PublicKey())
	if err != nil {
		t.Fatalf("сессия инициатора: %v", err)
	}
	bob := NewReceiverSession(sk, bobDH)

	if alice.recvChainKey != nil {
		t.Fatal("у инициатора есть приёмная цепочка до первого принятого пакета (§3.3 требует отсутствующую)")
	}
	if alice.sendingChainKey == nil {
		t.Fatal("у инициатора нет отправной цепочки: DH-шаг из §3.3 не выполнен")
	}
	if bytes.Equal(alice.rootKey, sk) {
		t.Fatal("корень инициатора равен SK: DH-шаг при инициализации пропущен")
	}
	if alice.dhrPublicKey == nil || !alice.dhrPublicKey.Equal(bobDH.PublicKey()) {
		t.Fatal("инициатор не задал DHr чужим DH-ключом")
	}
	if alice.sendSequence != 0 || alice.prevChainLength != 0 {
		t.Fatalf("счётчики инициатора должны стартовать с нуля: %d/%d", alice.sendSequence, alice.prevChainLength)
	}

	if !bytes.Equal(bob.rootKey, sk) {
		t.Fatal("корень получателя не равен SK (§3.5: RK = SK до первого пакета)")
	}
	if bob.sendingChainKey != nil || bob.recvChainKey != nil {
		t.Fatal("у получателя есть цепочки до первого принятого пакета")
	}
	if bob.dhrPublicKey != nil {
		t.Fatal("у получателя задан DHr до прихода пакета")
	}
	if bob.sendSequence != 0 || bob.recvSequence != 0 {
		t.Fatalf("счётчики получателя должны стартовать с нуля: %d/%d", bob.sendSequence, bob.recvSequence)
	}
}

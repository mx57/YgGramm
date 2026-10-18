// File: core/pkg/crypto/vectors_test.go
package crypto

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"os"
	"testing"
)

// Регрессионные векторы детерминированных путей криптографии.
//
// Что они защищают: все существующие тесты проверяют пару «отправитель ↔
// получатель», то есть обе стороны меняются синхронно. Если кто-то тронет
// параметр KDF (info-строку, число итераций PBKDF2, порядок конкатенации DH,
// префикс подписи), такой тест не упадёт — а две сборки, между которыми
// параметр разошёлся, перестанут расшифровывать друг друга. Векторы ловят
// именно это: значение замерено кодом и зафиксировано байт в байт.
//
// Чего в них НЕТ: это не тест-векторы спецификации Signal — машиночитаемых
// векторов X3DH/Double Ratchet Signal не публикует (ни на странице спецификации,
// ни в libsignal). Значит, зафиксированное значение — «наше значение на эту
// дату», а не «значение по стандарту». Три вещи при этом проверены извне:
// структура X3DH-вывода (состав ikm, порядок DH, длина SK) сверена с текстом
// спецификации в x3dh_spec_test.go, а сами байты SK пересчитаны вторым
// независимым HKDF — см. TestVectorX3DHReceiverSecret. Третья — путь
// «фраза → seed»: с Шага 11 кодирование сверено с каноническими примерами
// BIP-39, а seed переснят второй реализацией PBKDF2 на фикстурах из
// bip39_test.go (TestBip39VectorsEncodeAndDecode,
// TestBip39SeedMatchesIndependentImplementation). Фикстура-фраза здесь тоже
// вектор спецификации (энтропия 0x7f…7f), но с нашей passphrase, поэтому
// master.seed ниже — применение уже проверенной извне функции к нашему входу, а
// не опубликованное число. Мастер-ключи из seed выводятся HKDF по нашим доменам,
// и вот они уже только наши: BIP-32/SLIP-0010 для них не реализован, и сверять
// их было бы не с чем.
//
// Хеш (SHA-512), salt и info-строка — параметры, которые §2.1/§2.2 отдают
// приложению; у libsignal они другие (SHA-256, info «WhisperText», 64 байта с
// разбивкой на корень и цепочку), поэтому совместимость с ним не проверялась и
// её нет. Домены S&F, PoW и сертификата ключа ноды — наши, аналогов в
// стандарте нет.
//
// Обновление: `YGGGRAM_VECTORS=1 go test ./pkg/crypto -run Vector -v` печатает
// текущие значения — их и переносить в вызовы assertVector.

const (
	// Каноническая фраза из векторов самой спецификации (энтропия 0x7f…7f):
	// регрессионные числа снимаются с фразы, чья годность не зависит от нашего
	// кодера.
	vecMnemonic   = "legal winner thank year wave sausage worth useful legal winner thank yellow"
	vecPassphrase = "ygggram-test-passphrase"

	// Фиксированные сиды по 32 байта: X25519 принимает любые байты верной
	// длины, а Ed25519-ключ собирается из сида — весь набор ключей
	// воспроизводится без генератора случайных чисел.
	vecSenderIKSeed  = "1111111111111111111111111111111111111111111111111111111111111111"
	vecSenderEKSeed  = "2222222222222222222222222222222222222222222222222222222222222222"
	vecReceiverSeed  = "3333333333333333333333333333333333333333333333333333333333333333"
	vecSPKSeed       = "4444444444444444444444444444444444444444444444444444444444444444"
	vecOPKSeed       = "5555555555555555555555555555555555555555555555555555555555555555"
	vecNodeKeySeed   = "6666666666666666666666666666666666666666666666666666666666666666"
	vecMailboxPubHex = "7777777777777777777777777777777777777777777777777777777777777777"
	vecEpochDay      = int64(20588)
)

func unhexSeed(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("сид %s: %v", s, err)
	}
	return b
}

func xKey(t *testing.T, seedHex string) *ecdh.PrivateKey {
	t.Helper()
	k, err := ecdh.X25519().NewPrivateKey(unhexSeed(t, seedHex))
	if err != nil {
		t.Fatalf("x25519 ключ: %v", err)
	}
	return k
}

// assertVector сверяет байты с зафиксированным hex-вектором. С YGGGRAM_VECTORS=1
// проверка не делается — вместо неё печатается текущее значение, чтобы векторы
// обновлял сам код, а не глаз.
func assertVector(t *testing.T, name string, got []byte, wantHex string) {
	t.Helper()
	if os.Getenv("YGGGRAM_VECTORS") != "" {
		t.Logf("%s = %x", name, got)
		return
	}
	want, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatalf("%s: вектор повреждён: %v", name, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s: получили %x, вектор зафиксировал %s\nЕсли изменение намеренное — переснимите вектор: YGGGRAM_VECTORS=1 go test ./pkg/crypto -run Vector -v", name, got, wantHex)
	}
}

// TestVectorMasterIdentity — мнемоника → мастер-ключи. Всё выводится из фразы
// детерминированно: seed — PBKDF2-HMAC-SHA512(2048) по BIP-39, сами ключи —
// HKDF поверх него, поэтому вся цепочка воспроизводится на любой машине.
// Фикстура-фраза — канонический вектор спецификации (энтропия 0x7f…7f), так что
// эти числа привязаны к фразе, годность которой проверена вне нашего кода.
func TestVectorMasterIdentity(t *testing.T) {
	m, err := MasterIdentityFromMnemonic(vecMnemonic, vecPassphrase)
	if err != nil {
		t.Fatal(err)
	}
	again, err := MasterIdentityFromMnemonic(vecMnemonic, vecPassphrase)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(m.Seed, again.Seed) || !m.MasterX25519.PublicKey().Equal(again.MasterX25519.PublicKey()) {
		t.Fatal("вывод ключей из фразы не детерминирован")
	}

	assertVector(t, "master.seed", m.Seed, "b4bcc72776d6c81c619fd2afe39eb5ec46d415f0d6f7be9785f467fdeca74fc9fd73e44210237a0120ed8a2d09a6d5be07f3ba9987e05ed8441d06140f169875")
	assertVector(t, "master.ed25519.pub", m.MasterEd25519.Public().(ed25519.PublicKey), "fadd88ef03028817f68d9678116ffba91dcdb57a035088db33599270d56fa509")
	assertVector(t, "master.x25519.pub", m.MasterX25519.PublicKey().Bytes(), "164c6d4e7cd18b49ddae7802b0c25f1460c01e9157fc1de58bef5d10e9d2f17c")
	assertVector(t, "master.backup.key", m.BackupMasterKey[:], "714fc9ed9dfcad28b300b6b3036bda69ac270bd007c8c39be60b639a0b464ed7")
}

// TestVectorHKDF — базовый KDF, из которого собраны и мастер-ключи, и корень
// X3DH: один символ в info-строке меняет весь выход.
func TestVectorHKDF(t *testing.T) {
	out, err := HKDFDerive([]byte("ygggram-hkdf-input"), []byte("salt"), "ygggram-x3dh-root-v1", 32)
	if err != nil {
		t.Fatal(err)
	}
	assertVector(t, "hkdf.root", out, "2380d6c010399a28d2b33d5cb425ebb662bb3683856fb123bdc483e92de728e4")

	other, err := HKDFDerive([]byte("ygggram-hkdf-input"), []byte("salt"), "ygggram-x3dh-root-v2", 32)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(out, other) {
		t.Fatal("info-строка не влияет на HKDF: разделение контекста сломано")
	}
}

// TestVectorX3DHReceiverSecret — корневой секрет X3DH на захардкоженных ключах.
// Это ровно те DH, ровно тот порядок их конкатенации и ровно тот префикс F, по
// которым собираются сессии: сдвинуть любой — диалоги перестанут расшифровываться
// между сборками. Оба значения дополнительно сверены с расчётом сторонним HKDF
// (OpenSSL) и независимой сборкой RFC 5869 в TestX3DHSKMatchesIndependentHkdf.
func TestVectorX3DHReceiverSecret(t *testing.T) {
	ikB := xKey(t, vecReceiverSeed)
	spkB := xKey(t, vecSPKSeed)
	opkB := xKey(t, vecOPKSeed)
	ikA := xKey(t, vecSenderIKSeed).PublicKey()
	ekA := xKey(t, vecSenderEKSeed).PublicKey()

	withOPK, err := PerformX3DHReceiver(ikB, spkB, opkB, ikA, ekA)
	if err != nil {
		t.Fatal(err)
	}
	assertVector(t, "x3dh.shared.with.opk", withOPK, "07c67bba8a8a80e519dd3210110605b35d60d75b6a3b381837af761fea114d27")

	withoutOPK, err := PerformX3DHReceiver(ikB, spkB, nil, ikA, ekA)
	if err != nil {
		t.Fatal(err)
	}
	assertVector(t, "x3dh.shared.no.opk", withoutOPK, "2b83709b20c4fadd729465215e3d938428b93cd9e483bed96f7a3b6a5fae454d")
	if bytes.Equal(withOPK, withoutOPK) {
		t.Fatal("one-time prekey не влияет на корень: DH4 не участвует в выводе")
	}
}

// TestVectorNodeKeyProof — сертификат ключа ноды: подпись Ed25519
// детерминирована (RFC 8032), поэтому байты фиксируются напрямую.
func TestVectorNodeKeyProof(t *testing.T) {
	m, err := MasterIdentityFromMnemonic(vecMnemonic, vecPassphrase)
	if err != nil {
		t.Fatal(err)
	}
	node := ed25519.NewKeyFromSeed(unhexSeed(t, vecNodeKeySeed))
	pub := node.Public().(ed25519.PublicKey)
	proof, err := BuildNodeKeyProof(m, pub)
	if err != nil {
		t.Fatal(err)
	}
	assertVector(t, "node.key.proof", proof, "a7775fc7282699c7fe7254d8da5d8c3816e3c72ab0095672245c56dde9481f4d4d30ae155571d4910097236f7eb23a2795832ddc63e67533a502dc75afbecb0b")
	assertVector(t, "node.key.pub", pub, "34b4d9043156cb6dcf0beb0a2949b7559c940d2bcb6dbe8c53a9b30278e3a746")

	if err := VerifyNodeKeyProof(
		m.MasterEd25519.Public().(ed25519.PublicKey),
		m.MasterX25519.PublicKey().Bytes(),
		pub,
		proof,
	); err != nil {
		t.Fatalf("свой сертификат не прошёл проверку: %v", err)
	}
}

// TestVectorMailboxIDAndHashcash — слепой индекс ящика и PoW: от них зависит,
// придёт ли конверт в нужный ящик и примет ли его релеи, поэтому значения
// зафиксированы.
func TestVectorMailboxIDAndHashcash(t *testing.T) {
	pub := unhexSeed(t, vecMailboxPubHex)
	id := ComputeMailboxID(pub, vecEpochDay)
	assertVector(t, "mailbox.id", id[:], "a94acc73f1f9d5d1e9277731b12d000a7a6450190aad243f4b41db17b9cd08f4")

	nonce := ComputeHashcashPoW(id, 16)
	assertVector(t, "hashcash.nonce", binary.BigEndian.AppendUint64(nil, nonce), "0000000000007ef5")
	if !VerifyHashcashPoW(id, nonce, 16) {
		t.Fatalf("найденный PoW не проходит проверку (nonce=%d)", nonce)
	}
	if VerifyHashcashPoW(id, nonce+1, 16) {
		t.Logf("следующий nonce тоже проходит: сложность ниже заявленной")
	}
}

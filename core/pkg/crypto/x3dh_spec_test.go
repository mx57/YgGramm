// File: core/pkg/crypto/x3dh_spec_test.go
package crypto

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"testing"
)

// Тесты-«ворота» к спецификации X3DH (signal.org/docs/specifications/x3dh,
// §2.2 и §3.3). Официальных машиночитаемых тест-векторов X3DH у Signal нет,
// поэтому здесь проверяется не значение байта, а структура вывода: состав ikm,
// порядок склейки DH, длина SK и сходимость двух сторон. Сами байты зафиксированы
// отдельно в vectors_test.go — как регрессия «наше значение на эту дату»,
// а не как соответствие стандарту.

func concatBytes(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// specF — F из §2.2, переписанная здесь независимо от кода: тест проверяет и
// значение константы, а не только то, что она куда-то подмешивается.
var specF = [32]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

// TestX3DHIkmComposition — SK обязан равняться HKDF(F || DH1 || DH2 || DH3 || DH4)
// с нулевым salt и info-строкой приложения. DH-выходы пересчитываются в тесте
// заново, поэтому он ловит сразу три поломки: потерю доменного разделителя F,
// перестановку DH и склейку с разделителями.
func TestX3DHIkmComposition(t *testing.T) {
	ikB := xKey(t, vecReceiverSeed)
	spkB := xKey(t, vecSPKSeed)
	opkB := xKey(t, vecOPKSeed)
	ikA := xKey(t, vecSenderIKSeed)
	ekA := xKey(t, vecSenderEKSeed)

	dh1, err := ikA.ECDH(spkB.PublicKey())
	if err != nil {
		t.Fatalf("DH1: %v", err)
	}
	dh2, err := ekA.ECDH(ikB.PublicKey())
	if err != nil {
		t.Fatalf("DH2: %v", err)
	}
	dh3, err := ekA.ECDH(spkB.PublicKey())
	if err != nil {
		t.Fatalf("DH3: %v", err)
	}
	dh4, err := ekA.ECDH(opkB.PublicKey())
	if err != nil {
		t.Fatalf("DH4: %v", err)
	}

	km := concatBytes(dh1, dh2, dh3, dh4)
	sk, err := PerformX3DHReceiver(ikB, spkB, opkB, ikA.PublicKey(), ekA.PublicKey())
	if err != nil {
		t.Fatalf("receiver: %v", err)
	}

	if !bytes.Equal(x3dhDomainSeparator[:], specF[:]) {
		t.Fatalf("доменный разделитель в коде расходится с F из §2.2: %x", x3dhDomainSeparator)
	}

	want, err := HKDFDerive(concatBytes(specF[:], km), nil, "ygggram-x3dh-root-v1", 32)
	if err != nil {
		t.Fatalf("эталонный HKDF: %v", err)
	}
	if !bytes.Equal(sk, want) {
		t.Fatalf("SK не совпадает с KDF(KM) по спецификации: получили %x, ожидалось %x", sk, want)
	}
	if len(sk) != 32 {
		t.Fatalf("KDF(KM) по спецификации отдаёт 32 байта, а получилось %d", len(sk))
	}

	// salt = «zero-filled byte sequence длиной с выход хеша»: nil в нашем HKDF
	// — ровно это, и проверка не даёт формулировке разойтись с кодом.
	zeroSalt, err := HKDFDerive(concatBytes(specF[:], km), make([]byte, 64), "ygggram-x3dh-root-v1", 32)
	if err != nil {
		t.Fatalf("HKDF с явными нулями: %v", err)
	}
	if !bytes.Equal(want, zeroSalt) {
		t.Fatal("nil-salt в HKDFDerive — не 64 нуля: salt расходится с текстом спецификации")
	}

	// Разделитель F обязателен: без него это уже не X3DH.
	noSeparator, err := HKDFDerive(km, nil, "ygggram-x3dh-root-v1", 32)
	if err != nil {
		t.Fatalf("HKDF без F: %v", err)
	}
	if bytes.Equal(sk, noSeparator) {
		t.Fatal("F не подмешивается в ikm: доменное разделение XEdDSA сломано")
	}

	// Порядок DH зафиксирован: при склейке без разделителей перестановка обязана
	// давать другой ключ.
	reordered, err := HKDFDerive(concatBytes(specF[:], dh4, dh3, dh2, dh1), nil, "ygggram-x3dh-root-v1", 32)
	if err != nil {
		t.Fatalf("HKDF с переставленными DH: %v", err)
	}
	if bytes.Equal(sk, reordered) {
		t.Fatal("перестановка DH1..DH4 не меняет SK: KM склеен с разделителями, порядок не зафиксирован")
	}
}

// TestX3DHInitiatorReceiverAgree — стороны §3.3 и §3.4 обязаны сойтись на одном
// SK и с one-time prekey, и без него (bundle без OPK — штатный случай пустого
// склада).
func TestX3DHInitiatorReceiverAgree(t *testing.T) {
	ikB := xKey(t, vecReceiverSeed)
	spkB := xKey(t, vecSPKSeed)
	opkB := xKey(t, vecOPKSeed)
	ikA := xKey(t, vecSenderIKSeed)

	_, identityEd, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519 для подписи SPK: %v", err)
	}
	identityPub := identityEd.Public().(ed25519.PublicKey)
	spkSig := ed25519.Sign(identityEd, spkB.PublicKey().Bytes())

	for _, useOPK := range []bool{true, false} {
		bundle := PreKeyBundle{
			IdentityKey:     ikB.PublicKey(),
			IdentityEd25519: identityPub,
			SignedPreKey:    spkB.PublicKey(),
			SignedPreKeySig: spkSig,
		}
		if useOPK {
			bundle.OneTimePreKey = opkB.PublicKey()
			bundle.OneTimePreKeyID = 7
		}

		handshake, err := PerformX3DHInitiator(ikA, identityPub, bundle)
		if err != nil {
			t.Fatalf("инициатор (OPK=%v): %v", useOPK, err)
		}
		if handshake.UsedOneTimePK != useOPK {
			t.Fatalf("OPK=%v: инициатор вернул UsedOneTimePK=%v", useOPK, handshake.UsedOneTimePK)
		}

		var receiverOPK *ecdh.PrivateKey
		if useOPK {
			receiverOPK = opkB
		}
		sk, err := PerformX3DHReceiver(ikB, spkB, receiverOPK, ikA.PublicKey(), handshake.EphemeralPub)
		if err != nil {
			t.Fatalf("получатель (OPK=%v): %v", useOPK, err)
		}
		if !bytes.Equal(handshake.SharedKey, sk) {
			t.Fatalf("OPK=%v: стороны разошлись: %x против %x", useOPK, handshake.SharedKey, sk)
		}
		if len(sk) != 32 {
			t.Fatalf("OPK=%v: SK %d байт, нужно 32", useOPK, len(sk))
		}
	}
}

// TestX3DHBundleRejectsBadSPKSignature — §3.3: «Alice verifies the prekey
// signature and aborts». Подпись чужого identity-ключа должна отсесть до
// вычисления DH: иначе подмена SPK в bundle дала бы рабочий сеанс с атакующим.
func TestX3DHBundleRejectsBadSPKSignature(t *testing.T) {
	ikB := xKey(t, vecReceiverSeed)
	spkB := xKey(t, vecSPKSeed)
	ikA := xKey(t, vecSenderIKSeed)

	stranger := ed25519.NewKeyFromSeed(make([]byte, 32))
	strangerPub := stranger.Public().(ed25519.PublicKey)

	_, realEd, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519: %v", err)
	}

	_, err = PerformX3DHInitiator(ikA, strangerPub, PreKeyBundle{
		IdentityKey:     ikB.PublicKey(),
		IdentityEd25519: strangerPub,
		SignedPreKey:    spkB.PublicKey(),
		SignedPreKeySig: ed25519.Sign(realEd, spkB.PublicKey().Bytes()),
	})
	if err == nil {
		t.Fatal("bundle с подписью чужого identity-ключа принят")
	}
}

// independentHkdf — RFC 5869 «в лоб»: Extract и первый блок Expand сведены
// напрямую из HMAC-SHA-512, без общего HKDFDerive приложения. Нужен именно как
// второй, независимый расчёт того же SK.
func independentHkdf(km []byte) []byte {
	prkMac := hmac.New(sha512.New, make([]byte, 64))
	prkMac.Write(concatBytes(specF[:], km))

	block := hmac.New(sha512.New, prkMac.Sum(nil))
	block.Write([]byte("ygggram-x3dh-root-v1"))
	block.Write([]byte{0x01})
	return block.Sum(nil)[:32]
}

// TestX3DHSKMatchesIndependentHkdf — зафиксированные байты SK сверены тремя
// способами: общим HKDF приложения, независимой пересборкой RFC 5869 выше и
// OpenSSL (node crypto.hkdfSync("sha512", …), замер внешним процессом). Все три
// дают один и тот же вывод, так что значение в векторах — не «разделяемая ошибка»
// одной реализации.
func TestX3DHSKMatchesIndependentHkdf(t *testing.T) {
	ikB := xKey(t, vecReceiverSeed)
	spkB := xKey(t, vecSPKSeed)
	opkB := xKey(t, vecOPKSeed)
	ikA := xKey(t, vecSenderIKSeed)
	ekA := xKey(t, vecSenderEKSeed)

	dh1, err := ikA.ECDH(spkB.PublicKey())
	if err != nil {
		t.Fatalf("DH1: %v", err)
	}
	dh2, err := ekA.ECDH(ikB.PublicKey())
	if err != nil {
		t.Fatalf("DH2: %v", err)
	}
	dh3, err := ekA.ECDH(spkB.PublicKey())
	if err != nil {
		t.Fatalf("DH3: %v", err)
	}
	dh4, err := ekA.ECDH(opkB.PublicKey())
	if err != nil {
		t.Fatalf("DH4: %v", err)
	}

	cases := []struct {
		name    string
		km      []byte
		opk     *ecdh.PrivateKey
		wantHex string
	}{
		{"без OPK", concatBytes(dh1, dh2, dh3), nil, "2b83709b20c4fadd729465215e3d938428b93cd9e483bed96f7a3b6a5fae454d"},
		{"с OPK", concatBytes(dh1, dh2, dh3, dh4), opkB, "07c67bba8a8a80e519dd3210110605b35d60d75b6a3b381837af761fea114d27"},
	}

	for _, c := range cases {
		want, err := hex.DecodeString(c.wantHex)
		if err != nil {
			t.Fatalf("%s: вектор повреждён: %v", c.name, err)
		}
		got, err := PerformX3DHReceiver(ikB, spkB, c.opk, ikA.PublicKey(), ekA.PublicKey())
		if err != nil {
			t.Fatalf("%s: receiver: %v", c.name, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s: SK расходится с внешним замером: %x против %s", c.name, got, c.wantHex)
		}
		if local := independentHkdf(c.km); !bytes.Equal(got, local) {
			t.Fatalf("%s: HKDF приложения и независимая сборка RFC 5869 разошлись: %x против %x", c.name, got, local)
		}
	}
}

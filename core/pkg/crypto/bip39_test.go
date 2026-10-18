// File: core/pkg/crypto/bip39_test.go
package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// Числа ниже — не снимаются этим же кодом. Их выдала вторая, независимая по
// сборке реализация (Node.js: поток битов как строка из '0'/'1', режется по 11,
// sha256 и pbkdf2 — из OpenSSL), а три фразы из списка — канонические векторы
// самой спецификации BIP-39, которые угадать по памяти нельзя:
//
//	0x7f…7f → «legal winner thank year … yellow»
//	0x80…80 → «letter advice cage absurd … above»
//	0xff…ff → «zoo … wrong»
//
// Pbkdf2-значения seed — тот же замер (2048 итераций, HMAC-SHA512, 64 байта,
// salt = «mnemonic» + NFKD(passphrase)); для контрольной фразы из одиннадцати
// «abandon» и «about» с passphrase «TREZOR» независимая реализация даёт
// c55257c360c07c72029aebc1b53c05ed…, то есть совпадает с публикуемым вектором
// спецификации.
var bip39Vectors = []struct {
	entropy string
	phrase  string
}{
	{"00000000000000000000000000000000",
		"abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"},
	{"01010101010101010101010101010101",
		"absurd amount doctor acoustic avoid letter advice cage absurd amount doctor adjust"},
	{"0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f",
		"audit journey sense bulk valley maple destroy tiger audit journey sense cable"},
	{"7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f",
		"legal winner thank year wave sausage worth useful legal winner thank yellow"},
	{"80808080808080808080808080808080",
		"letter advice cage absurd amount doctor acoustic avoid letter advice cage above"},
	{"ffffffffffffffffffffffffffffffff",
		"zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo wrong"},
	{"1234567890abcdef1234567890abcdef12345678",
		"banana pencil owner candy rubber wasp emotion member joy luxury vibrant taste cash clinic token"},
	{"000000000000000000000000000000000000000000000000",
		"abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon agent"},
	{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"primary fetch primary fetch primary fetch primary fetch primary fetch primary fetch primary fetch primary fetch primary fetch primary fetch primary fetch primary foster"},
	{"00112233445566778899aabbccddeeff00112233445566778899aabb",
		"abandon math mimic master filter design carbon crystal rookie group knife wrap absurd much snack melt grid rough chapter fever reason"},
	{"3e14e9d412cef289371d6c77b9b09e7a958080f19e0e1b4e",
		"dignity point insect cereal upset duty symbol pulse jeans snow chest vivid fix document shoe scrub brave include"},
	{"f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0",
		"valley maple destroy tiger audit journey sense bulk valley maple destroy tiger audit journey sense bulk valley maple destroy tiger audit journey sense aware"},
}

// Фикстуры seed привязаны к энтропии, а не к позиции в списке: правка списка не
// должна молча перевесить тест на другую фразу. Замер — независимая реализация;
// третья строка проверяет NFKD на кириллице и умлауте, потому что без
// нормализации соль состоит из других байт и seed разошёлся бы с любым другим
// кошельком на той же фразе.
const (
	bip39SeedVectorEntropy = "3e14e9d412cef289371d6c77b9b09e7a958080f19e0e1b4e"
	bip39SeedNoPassphrase  = "643c86abf501ad39ff6907d7fb351d7211286405dd66e5554438a9e965bb02bd883c016b6c137a6d89aabadfcbc339bc8fd6b3375afaaf1009351cb1f48a3ada"
	bip39SeedTrezor        = "046598cb81c81707e7ea3bb99e1023747e3a4ec07d7241c0e45b8c1399f60816e941a37a52c1bc2f51e311efa5fa338f5ec8d63b6e09d171136c7cf562112f55"
	bip39SeedNonAscii      = "c67968b11c65656024fb63fc88298111fd96c3f5cdf10a8ee8b1d39d29fa14637c0e2b7f4b737dd04d587c03eae2f1122e414954dabba6d9131730db17cfdbfc"
)

// Хэш кодовой книги: каноническая форма — слова через \n с одним переводом в
// конце. Проверка на joined-байтах, а не на сырых байтах файла, потому что
// рабочий каталог Windows с autocrlf отдал бы CRLF и иначе падал бы тест, а не код.
const bip39WordlistSHA256 = "2f5eed53a4727b4bf8880d8f3f199efc90e58503646d9ff8eff3a2ed3b24dbda"

func TestBip39WordlistIsOfficial(t *testing.T) {
	if len(bip39Words) != 2048 {
		t.Fatalf("слов %d, по спецификации 2048", len(bip39Words))
	}
	if bip39Words[0] != "abandon" || bip39Words[2047] != "zoo" {
		t.Fatalf("края списка: %q … %q", bip39Words[0], bip39Words[2047])
	}
	seen := make(map[string]bool, len(bip39Words))
	for i, w := range bip39Words {
		if seen[w] {
			t.Fatalf("слово %q повторилось на позиции %d: индексы разошлись бы с эталоном", w, i)
		}
		seen[w] = true
		if strings.ToLower(w) != w {
			t.Fatalf("слово %q не в нижнем регистре", w)
		}
		for _, r := range w {
			if r < 'a' || r > 'z' {
				t.Fatalf("слово %q содержит не-ASCII-букву %q", w, r)
			}
		}
		if i > 0 && bip39Words[i-1] >= w {
			t.Fatalf("порядок слов нарушен: %q стоит раньше %q", bip39Words[i-1], w)
		}
	}

	canonical := strings.Join(bip39Words, "\n") + "\n"
	sum := sha256.Sum256([]byte(canonical))
	if got := hex.EncodeToString(sum[:]); got != bip39WordlistSHA256 {
		t.Fatalf("хэш кодовой книги %s — это не список из спецификации (ожидался %s)", got, bip39WordlistSHA256)
	}
}

func TestBip39VectorsEncodeAndDecode(t *testing.T) {
	for _, v := range bip39Vectors {
		entropy, err := hex.DecodeString(v.entropy)
		if err != nil {
			t.Fatalf("фикстура %s: %v", v.entropy, err)
		}
		got, err := MnemonicFromEntropy(entropy)
		if err != nil {
			t.Fatalf("кодирование %s: %v", v.entropy, err)
		}
		if got != v.phrase {
			t.Errorf("энтропия %s:\n получено %q\n ожидалось %q", v.entropy, got, v.phrase)
		}
		back, err := EntropyFromMnemonic(v.phrase)
		if err != nil {
			t.Errorf("разбор %q: %v", v.phrase, err)
			continue
		}
		if hex.EncodeToString(back) != v.entropy {
			t.Errorf("разбор %q дал %x, ожидалось %s", v.phrase, back, v.entropy)
		}
	}
}

// Round-trip по случайным энтропиям всех допустимых размеров: кодирование и
// разбор обязаны быть точными обратами друг друга, иначе часть фраз
// восстанавливалась бы в другие ключи.
func TestBip39RoundTripAllSizes(t *testing.T) {
	for _, size := range []int{16, 20, 24, 28, 32} {
		for i := 0; i < 200; i++ {
			entropy := make([]byte, size)
			if _, err := rand.Read(entropy); err != nil {
				t.Fatal(err)
			}
			phrase, err := MnemonicFromEntropy(entropy)
			if err != nil {
				t.Fatalf("размер %d: %v", size, err)
			}
			if want := (size*8 + size*8/32) / 11; len(strings.Fields(phrase)) != want {
				t.Fatalf("размер %d дал %d слов, нужно %d", size, len(strings.Fields(phrase)), want)
			}
			back, err := EntropyFromMnemonic(phrase)
			if err != nil {
				t.Fatalf("размер %d, фраза %q: %v", size, phrase, err)
			}
			if string(back) != string(entropy) {
				t.Fatalf("round-trip размера %d вернул %x вместо %x", size, back, entropy)
			}
		}
	}
}

func TestBip39RejectsGarbage(t *testing.T) {
	valid := bip39Vectors[0].phrase
	cases := []struct {
		name   string
		phrase string
	}{
		{"пусто", ""},
		{"одно слово", "abandon"},
		{"одиннадцать слов", strings.Join(strings.Fields(valid)[:11], " ")},
		{"тринадцать слов", valid + " zoo"},
		{"не из списка", "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon horseshoe"},
		{"битая сумма (последнее слово)", "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"},
		{"битая сумма (первое слово)", "ability abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"},
		{"двойное значение", valid + " " + valid},
	}
	for _, c := range cases {
		if _, err := EntropyFromMnemonic(c.phrase); err == nil {
			t.Errorf("%s: фраза %q принята, хотя принята быть не могла", c.name, c.phrase)
		}
		if err := ValidateMnemonic(c.phrase); err == nil {
			t.Errorf("%s: ValidateMnemonic промолчал на %q", c.name, c.phrase)
		}
	}
}

// Опечатка в слове ловится контрольной суммой не всегда — по спецификации
// вероятность пропуска 2^-CS, для 128 бит это 1/16. Тест проверяет именно это
// утверждение, а не «любая правка отвергается», что было бы неверно.
func TestBip39TypoDetectionRate(t *testing.T) {
	const trials = 2000
	accepted := 0
	for i := 0; i < trials; i++ {
		entropy := make([]byte, 16)
		if _, err := rand.Read(entropy); err != nil {
			t.Fatal(err)
		}
		phrase, err := MnemonicFromEntropy(entropy)
		if err != nil {
			t.Fatal(err)
		}
		words := strings.Fields(phrase)
		pos := i % len(words)
		words[pos] = bip39Words[(bip39WordIndex[words[pos]]+1)%2048]
		if _, err := EntropyFromMnemonic(strings.Join(words, " ")); err == nil {
			accepted++
		}
	}
	// Ожидаем ~1/16; граница 1/8 оставляет место на флуктуацию, нижняя — на то,
	// что сумма вообще не проверяется (тогда принял бы все замены).
	if accepted > trials/8 || accepted < trials/64 {
		t.Fatalf("из %d испорченных фраз принято %d: контрольная сумма работает не по спецификации", trials, accepted)
	}
}

func TestBip39SeedMatchesIndependentImplementation(t *testing.T) {
	phrase := ""
	for _, v := range bip39Vectors {
		if v.entropy == bip39SeedVectorEntropy {
			phrase = v.phrase
		}
	}
	if phrase == "" {
		t.Fatalf("фикстуры seed висят на энтропии %s, которой нет в списке векторов", bip39SeedVectorEntropy)
	}
	cases := []struct {
		passphrase string
		want       string
	}{
		{"", bip39SeedNoPassphrase},
		{"TREZOR", bip39SeedTrezor},
		{"порода ü", bip39SeedNonAscii},
	}
	for _, c := range cases {
		master, err := MasterIdentityFromMnemonic(phrase, c.passphrase)
		if err != nil {
			t.Fatalf("passphrase %q: %v", c.passphrase, err)
		}
		if got := hex.EncodeToString(master.Seed); got != c.want {
			t.Errorf("passphrase %q:\n получено %s\n ожидалось %s", c.passphrase, got, c.want)
		}
	}
}

// Нормализация не имеет права менять ключи: пользователь, который ввёл фразу
// капсом или вставил перенос строки, получает ту же личность, а не новый
// пустой аккаунт.
func TestBip39NormalizationKeepsSameKeys(t *testing.T) {
	phrase := bip39Vectors[3].phrase
	canonical, err := MasterIdentityFromMnemonic(phrase, "пароль")
	if err != nil {
		t.Fatal(err)
	}
	variants := []string{
		strings.ToUpper(phrase),
		"   " + strings.ReplaceAll(phrase, " ", "\t") + "\n",
		strings.ReplaceAll(strings.ToUpper(phrase), " ", "  "),
	}
	for _, v := range variants {
		other, err := MasterIdentityFromMnemonic(v, "пароль")
		if err != nil {
			t.Fatalf("вариант %q: %v", v, err)
		}
		if hex.EncodeToString(other.Seed) != hex.EncodeToString(canonical.Seed) {
			t.Errorf("вариант %q дал другой seed", v)
		}
		if !bytes.Equal(other.MasterEd25519.Public().(ed25519.PublicKey),
			canonical.MasterEd25519.Public().(ed25519.PublicKey)) {
			t.Errorf("вариант %q дал другой мастер-Ed25519", v)
		}
		if !other.MasterX25519.PublicKey().Equal(canonical.MasterX25519.PublicKey()) {
			t.Errorf("вариант %q дал другой мастер-X25519", v)
		}
		if other.BackupMasterKey != canonical.BackupMasterKey {
			t.Errorf("вариант %q дал другой ключ бэкапа", v)
		}
		if other.Mnemonic != NormalizeMnemonic(phrase) {
			t.Errorf("в структуре осталась неканоническая форма %q", other.Mnemonic)
		}
	}
}

func TestNewMnemonicIsUsable(t *testing.T) {
	for _, tc := range []struct {
		bits  int
		words int
	}{{128, 12}, {256, 24}} {
		first, err := NewMnemonic(tc.bits)
		if err != nil {
			t.Fatalf("%d бит: %v", tc.bits, err)
		}
		if n := len(strings.Fields(first)); n != tc.words {
			t.Fatalf("%d бит дали %d слов, нужно %d", tc.bits, n, tc.words)
		}
		if err := ValidateMnemonic(first); err != nil {
			t.Fatalf("собственная фраза не прошла проверку: %v", err)
		}
		second, err := NewMnemonic(tc.bits)
		if err != nil {
			t.Fatal(err)
		}
		if first == second {
			t.Fatalf("две фразы на %d бит совпали: %q", tc.bits, first)
		}
	}

	for _, bits := range []int{0, 127, 129, 96, 512} {
		if _, err := NewMnemonic(bits); err == nil {
			t.Errorf("NewMnemonic(%d) принял недопустимую длину", bits)
		}
	}
	for _, size := range []int{0, 4, 15, 33, 64} {
		if _, err := MnemonicFromEntropy(make([]byte, size)); err == nil {
			t.Errorf("энтропия %d байт принята, хотя спецификация её не определяет", size)
		}
	}
}

// GenerateMnemonic — то, что зовёт ядро при создании личности: обязано выдавать
// фразу, которую ворота вывода ключей принимают.
func TestGenerateMnemonicPassesGate(t *testing.T) {
	phrase, err := GenerateMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Fields(phrase)) != 12 {
		t.Fatalf("GenerateMnemonic дал %d слов", len(strings.Fields(phrase)))
	}
	if _, err := MasterIdentityFromMnemonic(phrase, ""); err != nil {
		t.Fatalf("собственная фраза не выводит ключей: %v", err)
	}

	// И наоборот: прежние «условные» фикстуры тестов, не проходящие сумму,
	// обязаны отвергаться, а не выводывать молча несуществующую личность.
	if _, err := MasterIdentityFromMnemonic(
		"alpha bravo charlie delta echo foxtrot golf hotel india juliet kilo lima", ""); err == nil {
		t.Error("фраза вне кодовой книги вывела ключи")
	}
	if _, err := MasterIdentityFromMnemonic(
		strings.Repeat("abandon ", 11)+"abandon", ""); err == nil {
		t.Error("фраза с битой контрольной суммой вывела ключи")
	}
}

// File: core/pkg/crypto/bip39.go
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Английский список слов BIP-39 целиком, 2048 строк. Файл лежит рядом с кодом и
// попадает в бинарник на этапе сборки: спецификация определяет кодовую книгу как
// часть стандарта, поэтому «свой» список здесь — не деталь реализации, а поломка
// совместимости. Инварианты файла (2048 строк, порядок, уникальность, ASCII)
// сторожит TestBip39WordlistIsOfficial.
//
//go:embed bip39_english.txt
var bip39WordlistFile string

var bip39Words = strings.Split(strings.TrimSuffix(bip39WordlistFile, "\n"), "\n")

var bip39WordIndex = func() map[string]int {
	m := make(map[string]int, len(bip39Words))
	for i, w := range bip39Words {
		m[w] = i
	}
	return m
}()

// Числа из спецификации: энтропия кратна 32 битам от 128 до 256, контрольная
// сумма — первые ENT/32 бит SHA-256 от энтропии, слово кодирует 11 бит.
const (
	bip39EntropyBytesMin = 16
	bip39EntropyBytesMax = 32
	bip39BitsPerWord     = 11
	// Число итераций PBKDF2-HMAC-SHA512 задаёт спецификация; 2048 — не настройка
	// безопасности по вкусу, а часть совместимости (другое значение дало бы
	// другой seed на той же фразе).
	bip39Pbkdf2Rounds = 2048
	// Один бит суммы на каждые 32 бита энтропии, то есть 33 бита потока дают три
	// слова (3 × 11). Отсюда «длина фразы кратна трём словам».
	bip39EntropyBitsPerChecksum = 32
	bip39BitsPerGroup           = 33
)

// NormalizeMnemonic приводит фразу к каноническому виду так, как требует
// спецификация: NFKD, регистр не значим, разделители — любые пробельные
// символы. Без нормализации passphrase и слова с диакритикой давали бы другой
// seed в разных кошельках, а проверка контрольной суммы ловила бы собственную
// фразу как чужую.
func NormalizeMnemonic(mnemonic string) string {
	normalized := norm.NFKD.String(strings.TrimSpace(mnemonic))
	words := strings.Fields(normalized)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, " ")
}

// MnemonicFromEntropy кодирует энтропию в фразу BIP-39. Длина энтропии обязана
// быть кратна 32 битам в диапазоне 128..256 — вне этого диапазона контрольная
// сумма не определена, а слово «случайная строка любой длины» в стандарте нет.
func MnemonicFromEntropy(entropy []byte) (string, error) {
	if len(entropy) < bip39EntropyBytesMin || len(entropy) > bip39EntropyBytesMax || len(entropy)%4 != 0 {
		return "", fmt.Errorf("bip39: энтропия %d байт: нужно 16, 20, 24, 28 или 32", len(entropy))
	}
	checksum := sha256.Sum256(entropy)

	entropyBits := len(entropy) * 8
	checksumBits := entropyBits / bip39EntropyBitsPerChecksum
	words := make([]string, (entropyBits+checksumBits)/bip39BitsPerWord)
	for i := range words {
		idx := 0
		for bit := 0; bit < bip39BitsPerWord; bit++ {
			idx = idx<<1 | int(bip39StreamBit(entropy, checksum[:], i*bip39BitsPerWord+bit))
		}
		words[i] = bip39Words[idx]
	}
	return strings.Join(words, " "), nil
}

// EntropyFromMnemonic разбирает фразу и возвращает энтропию. Это единственные
// ворота проверки подлинности фразы: слово из списка ещё ничего не значит, а
// вот контрольная сумма отсекает опечатку в любом слове с вероятностью 31/32 на
// одну замену. Возвращаемая энтропия — те самые байты, из которых фраза
// родилась, поэтому вывод ключей воспроизводится по фразе любого кошелька.
func EntropyFromMnemonic(mnemonic string) ([]byte, error) {
	words := strings.Fields(NormalizeMnemonic(mnemonic))
	n := len(words)
	if n < 12 || n > 24 || n%3 != 0 {
		return nil, fmt.Errorf("bip39: %d слов: должно быть 12, 15, 18, 21 или 24", n)
	}

	indices := make([]int, n)
	for i, w := range words {
		idx, ok := bip39WordIndex[w]
		if !ok {
			return nil, fmt.Errorf("bip39: слова %q нет в английском списке", w)
		}
		indices[i] = idx
	}

	totalBits := n * bip39BitsPerWord
	checksumBits := totalBits / bip39BitsPerGroup
	entropyBits := totalBits - checksumBits
	entropy := make([]byte, entropyBits/8)
	for i := 0; i < entropyBits; i++ {
		bit := (indices[i/bip39BitsPerWord] >> uint(bip39BitsPerWord-1-i%bip39BitsPerWord)) & 1
		entropy[i/8] |= byte(bit) << uint(7-i%8)
	}

	checksum := sha256.Sum256(entropy)
	for i := 0; i < checksumBits; i++ {
		bit := (indices[(entropyBits+i)/bip39BitsPerWord] >>
			uint(bip39BitsPerWord-1-(entropyBits+i)%bip39BitsPerWord)) & 1
		want := (checksum[i/8] >> uint(7-i%8)) & 1
		if byte(bit) != want {
			return nil, errors.New("bip39: контрольная сумма не сходится: фраза испорчена или из другой кодовой книги")
		}
	}
	return entropy, nil
}

// NewMnemonic генерирует фразу заданной длины из криптографического генератора.
// 128 бит дают 12 слов (стандарт YggGram для экрана восстановления), 256 — 24.
func NewMnemonic(bits int) (string, error) {
	if bits%32 != 0 || bits < 128 || bits > 256 {
		return "", fmt.Errorf("bip39: %d бит энтропии: нужно 128, 160, 192, 224 или 256", bits)
	}
	entropy := make([]byte, bits/8)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("bip39: генерация энтропии: %w", err)
	}
	return MnemonicFromEntropy(entropy)
}

// ValidateMnemonic проверяет фразу, не выводя ключей: так UI отличает
// «опечатка в слове» от «фразы вообще нет», не считая PBKDF2 на каждом вводе.
func ValidateMnemonic(mnemonic string) error {
	_, err := EntropyFromMnemonic(mnemonic)
	return err
}

// bip39StreamBit — i-й бит потока «энтропия || первые CS бит суммы». Старший бит
// байта идёт первым: спецификация перечисляет биты именно так, и обратный
// порядок дал бы другую фразу при тех же байтах.
func bip39StreamBit(entropy, checksum []byte, i int) byte {
	if i < len(entropy)*8 {
		return (entropy[i/8] >> uint(7-i%8)) & 1
	}
	j := i - len(entropy)*8
	return (checksum[j/8] >> uint(7-j%8)) & 1
}

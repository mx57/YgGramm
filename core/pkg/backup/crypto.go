// File: core/pkg/backup/crypto.go
//
// Криптографическая часть бэкапа. Отдельным файлом — потому что именно она в
// прежнем коде не совпадала с описанием: «Argon2id + XChaCha20-Poly1305» в
// README и поле argon2id_salt в ygggram.proto при сырых 32 байтах ключа,
// переданных конструктору, и AES-256-GCM. Здесь оба этих алгоритма выполняются
// кодом, и имена алгоритмов в незашифрованном заголовке обязаны совпадать с
// тем, что код делает (это проверяет тест, а не внимание автора строки).
//
// Два ключа из одного вывода Argon2id, а не один: прежний код тем же backupKey
// и запечатывал чанки, и подписывал манифест HMAC-ом. Разделение по домену —
// стандартная гигиена, и оно исключает ситуацию, когда компрометация одного
// примитива утаскивает за собой второй.
package backup

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	// SaltSize — salt Argon2id из ygggram.proto (поле argon2id_salt).
	SaltSize = 16

	// KdfName, CompressionName, EncryptionName — строки, которые ложатся в
	// незашифрованный заголовок пакета и обязаны соответствовать коду ниже.
	KdfName         = "argon2id"
	CompressionName = "zlib"
	EncryptionName  = "xchacha20poly1305"

	manifestAAD  = "ygggram-backup-manifest-v1"
	keyMaterial  = 64
	encKeyLength = 32
	macKeyLength = 32
)

// KDFParams — параметры Argon2id. Едут в заголовке открытым текстом, потому что
// без них восстановление невозможно, и потому что смена параметров должна быть
// видна, а не выведена догадкой.
type KDFParams struct {
	Time      uint32 `json:"time"`
	MemoryKiB uint32 `json:"memory_kib"`
	Threads   uint8  `json:"threads"`
}

// ProductionKDF — профиль RFC 9106 §7.3 для интерактивного применения: t=3,
// 64 МиБ. Дорожек одна, а не четыре: вывод Argon2id зависит от числа дорожек, и
// одинаковые параметры на телефоне и на десктопе важнее нескольких десятков
// миллисекунд, которые экономит параллелизм.
var ProductionKDF = KDFParams{Time: 3, MemoryKiB: 64 * 1024, Threads: 1}

// TestKDF — заведомо дешёвый профиль для тестов формата и путей отказа.
// Параметры ProductionKDF в тестах тоже измеряются, но отдельно (см.
// TestProductionKdfIsMeasured), чтобы проверка «восстановление работает» не
// превращалась в ожидание на полчаса.
var TestKDF = KDFParams{Time: 1, MemoryKiB: 64, Threads: 1}

func (p KDFParams) validate() error {
	if p.Time == 0 || p.MemoryKiB == 0 || p.Threads == 0 {
		return fmt.Errorf("backup: нулевой параметр Argon2id: %+v", p)
	}
	// Обязательство Argon2: m >= 8 * p^2.
	if uint64(p.MemoryKiB) < 8*uint64(p.Threads)*uint64(p.Threads) {
		return fmt.Errorf("backup: памятью %d КиБ при %d дорожках не обойтись", p.MemoryKiB, p.Threads)
	}
	return nil
}

type keySet struct {
	enc [encKeyLength]byte
	mac [macKeyLength]byte
}

// deriveKeys — Argon2id(passphrase, salt) → 64 байта, делящиеся на ключ
// шифрования и ключ MAC. Ошибка на пустой passphrase или кривом salt — не
// косметика: с ними вывод вырождается в детерминированный по пустому входу, и
// «защита паролем» становится фикцией.
func deriveKeys(passphrase string, salt []byte, p KDFParams) (keySet, error) {
	var ks keySet
	if passphrase == "" {
		return ks, errors.New("backup: пустая passphrase")
	}
	if len(salt) != SaltSize {
		return ks, fmt.Errorf("backup: salt должен быть %d байт, а не %d", SaltSize, len(salt))
	}
	if err := p.validate(); err != nil {
		return ks, err
	}

	out := argon2.IDKey([]byte(passphrase), salt, p.Time, p.MemoryKiB, p.Threads, keyMaterial)
	copy(ks.enc[:], out[:encKeyLength])
	copy(ks.mac[:], out[encKeyLength:])
	return ks, nil
}

// newSalt — salt обязан быть случайным: два бэкапа одной passphrase не имеют
// права давать одинаковый ключ.
func newSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("backup: salt не задать: %w", err)
	}
	return salt, nil
}

// sealChunk — XChaCha20-Poly1305 с 24-байтовым случайным nonce в начале кадра.
// AAD — контент-хэш чанка, то есть шифртекст привязан к тому хэшу, под которым
// чанк лежит в хранилище: подмена имени файла с содержимым не проходит.
func sealChunk(key *[encKeyLength]byte, plaintext, aad []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("backup: AEAD не собрать: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("backup: nonce не задать: %w", err)
	}
	return aead.Seal(nonce, nonce, plaintext, aad), nil
}

func openChunk(key *[encKeyLength]byte, blob, aad []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("backup: AEAD не собрать: %w", err)
	}
	if len(blob) < aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("backup: кадр короче, чем nonce с тегом")
	}
	nonce, ciphertext := blob[:aead.NonceSize()], blob[aead.NonceSize():]
	out, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("backup: чанк не расшифровывается: %w", err)
	}
	return out, nil
}

// macManifest — encrypt-then-MAC: подпись считается от шифртекста манифеста,
// поэтому проверка не требует сначала расшифровать то, чему мы не доверяем.
func macManifest(key *[macKeyLength]byte, ciphertext []byte) []byte {
	mac := hmac.New(sha256.New, key[:])
	mac.Write([]byte(manifestAAD))
	mac.Write(ciphertext)
	return mac.Sum(nil)
}

func checkManifestMAC(key *[macKeyLength]byte, ciphertext, signature []byte) error {
	want := macManifest(key, ciphertext)
	if !hmac.Equal(want, signature) {
		return errors.New("backup: подпись манифеста не сходится")
	}
	return nil
}

// MerkleRoot — двоичное дерево SHA-256 над контент-хэшами чанков в порядке
// сборки. Нечётный элемент уровня дублируется (тоже задокументировано в тесте),
// иначе перестановка последних чанков не меняла бы корень. Один-единственный
// чанк не отдаёт свой хэш как есть, а проходит хотя бы один раунд связывания:
// иначе корень совпадал бы с хэшем самого листа, и потеря или подмена
// единственного чанка переставала бы отличаться от дерева вообще.
func MerkleRoot(hashes [][]byte) ([]byte, error) {
	if len(hashes) == 0 {
		return nil, errors.New("backup: пустой список хэшей")
	}
	for i, h := range hashes {
		if len(h) != sha256.Size {
			return nil, fmt.Errorf("backup: хэш чанка %d длиной %d, а не %d", i, len(h), sha256.Size)
		}
	}

	level := make([][]byte, len(hashes))
	for i, h := range hashes {
		level[i] = h
	}
	// Раунд выполняется всегда, даже когда листьев один-единственный: иначе
	// корень одночанкового пакета равнялся бы хэшу этого чанка.
	for {
		next := make([][]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, pairHash(left, right))
		}
		level = next
		if len(level) == 1 {
			return level[0], nil
		}
	}
}

func pairHash(left, right []byte) []byte {
	buf := make([]byte, 0, len(left)+len(right))
	buf = append(buf, left...)
	buf = append(buf, right...)
	sum := sha256.Sum256(buf)
	return sum[:]
}

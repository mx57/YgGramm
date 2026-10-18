// File: core/pkg/backup/backup_test.go
//
// Первый тестовый файл в пакете. До него криптография бэкапа не выполнялась ни
// разу: её не звал ни один вызывающий, и «работает» было описанием, а не
// фактом. Здесь проверяется ровно то, что обещают README, AGENTS.md и
// ygggram.proto (argon2id_salt, compression_algo, encryption_algo,
// root_merkle_hash), — так, чтобы враньё в любом из этих мест уронило прогон.
//
// Три границы, которые нельзя «улучшить» молча:
//   - ключ выводится из passphrase, поэтому единственный способ восстановить
//     пакет в другом процессе — прочесть salt и параметры KDF из заголовка;
//     тесты идут через RestoreBundle, а не через живой движок, иначе
//     проверка превратилась бы в вызов того же объекта;
//   - манифест подписан HMAC по шифртексту (encrypt-then-MAC), поэтому
//     подмена порядка чанков без ключа невозможна: перепечатать манифест может
//     только тот, у кого есть MAC-ключ. Отдельно проверяется, что заголовок,
//     склеенный другим ключом, падает именно на подписи;
//   - шифртекст чанка привязан AAD к своему контент-хэшу, поэтому файл,
//     выложенный под чужим именем, не расшифровывается.
package backup

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testPassphrase = "верная passphrase для теста бэкапа"

func randomPayload(t *testing.T, n int) []byte {
	t.Helper()

	// Байты из SHA-256 со счётчиком: детерминированы (падающий тест должен
	// повторяться) и не схлопываются в один чанк, как bytes.Repeat.
	out := make([]byte, 0, n)
	counter := uint64(1)
	for len(out) < n {
		var buf [8]byte
		for i := range buf {
			buf[i] = byte(counter >> (8 * i))
		}
		h := sha256.Sum256(append(buf[:], byte(len(out))))
		out = append(out, h[:]...)
		counter++
	}
	return out[:n]
}

func mustSalt(t *testing.T) []byte {
	t.Helper()

	salt, err := newSalt()
	if err != nil {
		t.Fatalf("salt не задать: %v", err)
	}
	return salt
}

func newTestEngine(t *testing.T, passphrase, dir string, salt []byte) *SnapshotEngine {
	t.Helper()

	e, err := NewSnapshotEngineWithKDF(passphrase, dir, salt, TestKDF)
	if err != nil {
		t.Fatalf("движок не собрался: %v", err)
	}
	return e
}

// createBundle — тот же путь, что и CreateBundle, но с дешёвым KDF: боевой
// профиль в этих тестах означал бы минуту ожидания на каждый случай.
func createBundle(t *testing.T, e *SnapshotEngine, dir string, payload []byte) (*Header, []ChunkDescriptor) {
	t.Helper()

	descriptors, totalChunk, err := e.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatalf("чанки не легли: %v", err)
	}
	h, err := e.Seal(descriptors, uint64(len(payload)), totalChunk)
	if err != nil {
		t.Fatalf("манифест не собрался: %v", err)
	}
	if err := WriteHeader(dir, h); err != nil {
		t.Fatalf("заголовок не записан: %v", err)
	}
	return h, descriptors
}

func chunkFiles(t *testing.T, e *SnapshotEngine) []string {
	t.Helper()

	dir, err := e.localChunkDir()
	if err != nil {
		t.Fatalf("каталог чанков не вывести: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "??", "*.bin"))
	if err != nil {
		t.Fatalf("каталог чанков не перечисляется: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("чанков на диске нет — проверка их не касается")
	}
	return files
}

func TestDeriveKeysSeparatesEncryptionFromMAC(t *testing.T) {
	salt := mustSalt(t)
	otherSalt := mustSalt(t)

	first, err := deriveKeys(testPassphrase, salt, TestKDF)
	if err != nil {
		t.Fatalf("ключ не выведен: %v", err)
	}
	same, err := deriveKeys(testPassphrase, salt, TestKDF)
	if err != nil {
		t.Fatalf("ключ не выведен повторно: %v", err)
	}
	if first.enc != same.enc || first.mac != same.mac {
		t.Fatal("Argon2id дал разные ключи на одинаковом входе")
	}

	shifted, err := deriveKeys(testPassphrase, otherSalt, TestKDF)
	if err != nil {
		t.Fatalf("ключ на втором salt не выведен: %v", err)
	}
	if shifted.enc == first.enc {
		t.Fatal("salt не влияет на ключ: два пакета одной фразы шифровались бы одинаково")
	}
	if bytes.Equal(first.enc[:], first.mac[:]) {
		t.Fatal("ключи шифрования и MAC совпали — ровно то, что делал прежний код одним backupKey")
	}

	for _, bad := range []struct {
		name   string
		pass   string
		salt   []byte
		params KDFParams
	}{
		{"пустая passphrase", "", salt, TestKDF},
		{"salt короче", testPassphrase, salt[:SaltSize-1], TestKDF},
		{"salt длиннее", testPassphrase, append(salt, 1), TestKDF},
		{"нулевой time", testPassphrase, salt, KDFParams{Time: 0, MemoryKiB: 64, Threads: 1}},
		{"мало памяти для четырёх дорожек", testPassphrase, salt, KDFParams{Time: 1, MemoryKiB: 8, Threads: 4}},
	} {
		if _, err := deriveKeys(bad.pass, bad.salt, bad.params); err == nil {
			t.Errorf("%s: ключ вывелся, хотя вход негодный", bad.name)
		}
	}
}

func TestHeaderAlgoNamesMatchWhatCodeActuallyDoes(t *testing.T) {
	dir := t.TempDir()
	e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
	h, _ := createBundle(t, e, dir, randomPayload(t, 200*1024))

	// Эти строки читаются как протокол и в README, и в полях protobuf:
	// расхождение имени с кодом — тот класс вранья, из-за которого пакет
	// числился готовым.
	if h.Kdf != "argon2id" || h.Encryption != "xchacha20poly1305" || h.Compression != "zlib" {
		t.Fatalf("заголовок называет %s/%s/%s", h.Kdf, h.Encryption, h.Compression)
	}
	if len(h.Salt) != SaltSize {
		t.Fatalf("salt %d байт, а в контракте поле называется argon2id_salt", len(h.Salt))
	}

	// XChaCha20-Poly1305 = 24-байтовый nonce + 16-байтовый тег. Кадры короче
	// значат, что пишет другой примитив, чем заявлено в заголовке.
	blobs := chunkFiles(t, e)
	raw, err := os.ReadFile(blobs[0])
	if err != nil {
		t.Fatalf("чанк не читается: %v", err)
	}
	if len(raw) < 24+16 {
		t.Fatalf("кадр %d байт — меньше, чем nonce с тегом", len(raw))
	}
}

func TestBundleRestoresInAFreshEngineFromPassphrase(t *testing.T) {
	dir := t.TempDir()
	payload := randomPayload(t, 700*1024)

	e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
	h, _ := createBundle(t, e, dir, payload)
	if h.TotalChunks < 2 {
		t.Fatalf("чанков %d: связка файлов не проверяется", h.TotalChunks)
	}

	// Новый движок знает только passphrase: salt и параметры он обязан прочесть
	// из заголовка, иначе «восстановление на другом устройстве» — фикция.
	got, err := RestoreBundle(testPassphrase, dir)
	if err != nil {
		t.Fatalf("восстановление не удалось: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("собрано %d байт, не совпадающих с оригиналом (%d)", len(got), len(payload))
	}
}

func TestWrongPassphraseAndTamperingAreErrors(t *testing.T) {
	payload := randomPayload(t, 300*1024)

	t.Run("чужая passphrase", func(t *testing.T) {
		dir := t.TempDir()
		e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
		createBundle(t, e, dir, payload)
		if _, err := RestoreBundle("другая фраза", dir); err == nil {
			t.Fatal("пакет открылся чужой passphrase")
		}
	})

	t.Run("подмена байта в чанке", func(t *testing.T) {
		dir := t.TempDir()
		e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
		createBundle(t, e, dir, payload)

		files := chunkFiles(t, e)
		raw, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("чанк не читается: %v", err)
		}
		raw[len(raw)-1] ^= 0x01
		if err := os.WriteFile(files[0], raw, 0600); err != nil {
			t.Fatalf("чанк не перезаписан: %v", err)
		}
		if _, err := RestoreBundle(testPassphrase, dir); err == nil {
			t.Fatal("подменённый чанк прошёл как годный")
		}
	})

	t.Run("чанк, выложенный под чужим именем", func(t *testing.T) {
		dir := t.TempDir()
		e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
		createBundle(t, e, dir, payload)

		files := chunkFiles(t, e)
		if len(files) < 2 {
			t.Fatalf("чанков %d — менять имена нечем", len(files))
		}
		a, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("первый чанк не читается: %v", err)
		}
		b, err := os.ReadFile(files[1])
		if err != nil {
			t.Fatalf("второй чанк не читается: %v", err)
		}
		// Содержимое меняется местами, имена остаются: AAD по контент-хэшу
		// обязан это заметить.
		if err := os.WriteFile(files[0], b, 0600); err != nil {
			t.Fatalf("первый чанк не перезаписан: %v", err)
		}
		if err := os.WriteFile(files[1], a, 0600); err != nil {
			t.Fatalf("второй чанк не перезаписан: %v", err)
		}
		if _, err := RestoreBundle(testPassphrase, dir); err == nil {
			t.Fatal("файл под чужим именем расшифровался — AAD по хэшу не работает")
		}
	})

	t.Run("подмена зашифрованного манифеста", func(t *testing.T) {
		dir := t.TempDir()
		e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
		h, _ := createBundle(t, e, dir, payload)

		h.Manifest[len(h.Manifest)-1] ^= 0x01
		if err := WriteHeader(dir, h); err != nil {
			t.Fatalf("заголовок не перезаписан: %v", err)
		}
		// MAC считается по шифртексту, поэтому падение обязано быть до
		// расшифровки: иначе подменённый манифест был бы оракулом.
		if _, err := RestoreBundle(testPassphrase, dir); err == nil {
			t.Fatal("манифест с оторванной подписью восстановился")
		} else if !strings.Contains(err.Error(), "подпись") {
			t.Fatalf("ошибка не про подпись: %v", err)
		}
	})

	t.Run("заголовок, подписанный другим salt", func(t *testing.T) {
		dir := t.TempDir()
		e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
		_, descriptors := createBundle(t, e, dir, payload)

		// Та же passphrase, другой salt: чужой движок легально подписывает свой
		// манифест, и проверка MAC проходит — зато чанки остались в каталоге
		// первого ключа, и до них восстановление не доходит. Порядок чанков без
		// MAC-ключа оригинала не изменить (см. случай выше), а каталоги,
		// привязанные к ключу, ловят подмену заголовка целиком.
		other := newTestEngine(t, testPassphrase, dir, mustSalt(t))
		forged, err := other.Seal(descriptors, uint64(len(payload)), sumEncrypted(descriptors))
		if err != nil {
			t.Fatalf("поддельный заголовок не собирается: %v", err)
		}
		if err := WriteHeader(dir, forged); err != nil {
			t.Fatalf("заголовок не перезаписан: %v", err)
		}
		if _, err := RestoreBundle(testPassphrase, dir); err == nil {
			t.Fatal("пакет открылся по заголовку, подписанному чужим ключом")
		} else if !strings.Contains(err.Error(), "лежит не в хранилище") {
			t.Fatalf("ошибка не про отсутствующие чанки: %v", err)
		}
	})

	t.Run("манифест ссылается на отсутствующий чанк", func(t *testing.T) {
		dir := t.TempDir()
		e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
		createBundle(t, e, dir, payload)

		ghost := ChunkDescriptor{
			ChunkHash:     hex.EncodeToString(bytes.Repeat([]byte{0xAB}, sha256.Size)),
			OriginalSize:  100,
			EncryptedSize: 140,
		}
		forged, err := e.Seal([]ChunkDescriptor{ghost}, 100, 140)
		if err != nil {
			t.Fatalf("манифест с призрачным чанком не собирается: %v", err)
		}
		if err := WriteHeader(dir, forged); err != nil {
			t.Fatalf("заголовок не перезаписан: %v", err)
		}
		if _, err := RestoreBundle(testPassphrase, dir); err == nil {
			t.Fatal("манифест, ссылающийся на отсутствующий чанк, восстановился")
		} else if !strings.Contains(err.Error(), "лежит не в хранилище") {
			// Формулировка важна: «чанка нет в хранилище» — это не «не
			// расшифровывается», и вызывающий по ней понимает, что пакет неполный,
			// а не что passphrase мимо.
			t.Fatalf("отсутствующий чанк назван иначе: %v", err)
		}
	})
}

func sumEncrypted(descriptors []ChunkDescriptor) uint64 {
	var total uint64
	for _, d := range descriptors {
		total += uint64(d.EncryptedSize)
	}
	return total
}

func TestDeduplicationLooksAtDiskNotOnlyMemory(t *testing.T) {
	dir := t.TempDir()
	salt := mustSalt(t)
	payload := randomPayload(t, 200*1024)

	first := newTestEngine(t, testPassphrase, dir, salt)
	descriptorsA, _, err := first.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatalf("первая раскладка не удалась: %v", err)
	}
	afterFirst := len(chunkFiles(t, first))

	// Новый движок с тем же salt и фразой: память пустая, диск полный.
	second := newTestEngine(t, testPassphrase, dir, salt)
	descriptorsB, _, err := second.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatalf("вторая раскладка не удалась: %v", err)
	}
	if afterSecond := len(chunkFiles(t, second)); afterSecond != afterFirst {
		t.Fatalf("чанков было %d, после перезапуска стало %d: дедупликация по диску не работает",
			afterFirst, afterSecond)
	}

	// Повторный чанк обязан нести честный размер шифртекста: ноль в манифесте
	// значит, что пакет врёт про объём хранилища.
	for i := range descriptorsA {
		if descriptorsA[i].EncryptedSize == 0 || descriptorsB[i].EncryptedSize == 0 {
			t.Fatalf("чанк %d записан с нулевым размером шифртекста", i)
		}
	}

	// Другая passphrase — другой ключ — другой каталог чанков: файлы первого
	// пакета не переиспользуются, иначе восстановление читало бы чанки,
	// зашифрованные чужим ключом.
	third := newTestEngine(t, "иная фраза", dir, mustSalt(t))
	if _, _, err := third.ProcessFileIntoChunks(payload); err != nil {
		t.Fatalf("раскладка другим ключом не удалась: %v", err)
	}
	if afterThird := len(chunkFiles(t, third)); afterThird != afterFirst {
		t.Fatalf("в каталоге второго ключа %d файлов, ожидалось %d", afterThird, afterFirst)
	}
	total := 0
	err = filepath.WalkDir(filepath.Join(dir, "chunks"), func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".bin") {
			total++
		}
		return err
	})
	if err != nil {
		t.Fatalf("обход чанков сорвался: %v", err)
	}
	if total != 2*afterFirst {
		t.Fatalf("всего файлов %d, ожидалось %d: ключевые каталоги не разделились", total, 2*afterFirst)
	}
}

func TestMerkleRootOrderSensitivityAndSingleLeaf(t *testing.T) {
	a := sha256.Sum256([]byte("a"))
	b := sha256.Sum256([]byte("b"))

	root, err := MerkleRoot([][]byte{a[:], b[:]})
	if err != nil {
		t.Fatalf("корень не посчитан: %v", err)
	}
	want := sha256.Sum256(append(append([]byte{}, a[:]...), b[:]...))
	if !bytes.Equal(root, want[:]) {
		t.Fatal("корень из двух листьев не равен sha256(a||b): дерево собрано не по описанию")
	}

	swapped, err := MerkleRoot([][]byte{b[:], a[:]})
	if err != nil {
		t.Fatalf("корень не посчитан: %v", err)
	}
	if bytes.Equal(root, swapped) {
		t.Fatal("перестановка листьев не меняет корень: merkle-дерево ничего не сторожит")
	}

	single, err := MerkleRoot([][]byte{a[:]})
	if err != nil {
		t.Fatalf("корень одного листа не посчитан: %v", err)
	}
	if bytes.Equal(single, a[:]) {
		t.Fatal("один лист отдаёт себя: нечётный узел не дублируется, и потеря листа не видна")
	}

	three, err := MerkleRoot([][]byte{a[:], b[:], a[:]})
	if err != nil {
		t.Fatalf("корень из трёх листьев не посчитан: %v", err)
	}
	if bytes.Equal(three, root) {
		t.Fatal("три листа дали корень двух: лишний лист в дереве проигнорирован")
	}

	if _, err := MerkleRoot(nil); err == nil {
		t.Fatal("пустой список хэшей дал корень")
	}
	if _, err := MerkleRoot([][]byte{{1, 2, 3}}); err == nil {
		t.Fatal("хэш не той длины принят")
	}
}

func TestProductionKdfIsMeasured(t *testing.T) {
	// Граница проверки: этот тест меряет стоимость профиля, а не корректность
	// самого Argon2id — её даёт x/crypto/argon2, и ни одна независимая
	// реализация в этом репозитории вывод не переснимала (тест-векторов RFC 9106
	// здесь нет, а локального инструмента для их съёма не нашлось). Что вывод
	// хотя бы детерминирован и зависит от salt, сверяет
	// TestDeriveKeysSeparatesEncryptionFromMAC.
	salt := mustSalt(t)
	start := time.Now()
	if _, err := deriveKeys(testPassphrase, salt, ProductionKDF); err != nil {
		t.Fatalf("ключ боевого профиля не выведен: %v", err)
	}
	cost := time.Since(start)
	t.Logf("Argon2id %+v: %s на этой машине", ProductionKDF, cost.Round(time.Millisecond))

	// Верхняя граница — не «быстро», а «не ловушка»: десятки секунд на
	// восстановление были бы непригодны для телефона.
	if cost > 10*time.Second {
		t.Fatalf("вывод ключа занял %s: профиль непригоден для интерактивного восстановления", cost)
	}
}

// TestHeaderCarriesPlaintextIdentity — открытый заголовок обязан называть тот же
// пакет, что и подписанный манифест. С Шага 13 у пакета появился вызывающий
// (core/pkg/rpc/backup_service.go), и он читает только заголовок: из него берутся
// id для событий прогресса и поля protobuf-ответа (BackupManifest), которые
// клиент видит, не умея расшифровывать манифест.
func TestHeaderCarriesPlaintextIdentity(t *testing.T) {
	dir := t.TempDir()
	payload := randomPayload(t, 300*1024)

	e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
	h, descriptors := createBundle(t, e, dir, payload)

	if h.BackupID == 0 {
		t.Fatal("в заголовке нулевой id пакета")
	}
	if h.BackupID != e.BackupID() {
		t.Fatalf("заголовок называет id %d, движок — %d: прогресс и ответ про разные пакеты", h.BackupID, e.BackupID())
	}
	if len(h.RootMerkleHash) != sha256.Size {
		t.Fatalf("merkle-корень в заголовке %d байт, ожидалось %d", len(h.RootMerkleHash), sha256.Size)
	}
	if h.TotalChunks != uint32(len(descriptors)) {
		t.Fatalf("заголовок обещает %d чанков, уложено %d", h.TotalChunks, len(descriptors))
	}
	if h.TotalPlain != uint64(len(payload)) {
		t.Fatalf("открытых байтов %d, а payload был %d", h.TotalPlain, len(payload))
	}
	if h.TotalEncrypted != sumEncrypted(descriptors) {
		t.Fatalf("зашированных байт %d, по дескрипторам %d", h.TotalEncrypted, sumEncrypted(descriptors))
	}
	// created_at принадлежит движку, а не моменту Seal: id берётся оттуда же, и
	// вызывающему он нужен на первое же событие — до сборки манифеста.
	if diff := time.Now().Unix() - h.CreatedAt; diff < 0 || diff > 30 {
		t.Fatalf("created_at %d отличается от текущего на %d с: время пакета берётся не там", h.CreatedAt, diff)
	}

	again, err := ReadHeader(dir)
	if err != nil {
		t.Fatalf("заголовок после записи не читается: %v", err)
	}
	if again.BackupID != h.BackupID || again.CreatedAt != h.CreatedAt {
		t.Fatalf("id/время после чтения другие: %d/%d против %d/%d",
			again.BackupID, again.CreatedAt, h.BackupID, h.CreatedAt)
	}
	if !bytes.Equal(again.RootMerkleHash, h.RootMerkleHash) {
		t.Fatal("merkle-корень после чтения не тот")
	}
	if again.TotalChunks != h.TotalChunks || again.TotalPlain != h.TotalPlain ||
		again.TotalEncrypted != h.TotalEncrypted {
		t.Fatalf("размеры после чтения разошлись с записанными: %+v против %+v", again, h)
	}
}

// TestPlaintextHeaderCannotLieAboutBundle — открытая часть пакета подписи не
// имеет, поэтому хотя бы не должна расходиться с подписанной: иначе вызывающий
// отдаст клиенту корень и id, которые чанки не сторожат.
func TestPlaintextHeaderCannotLieAboutBundle(t *testing.T) {
	payload := randomPayload(t, 300*1024)

	t.Run("корень заголовка не тот, что в манифесте", func(t *testing.T) {
		dir := t.TempDir()
		e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
		h, _ := createBundle(t, e, dir, payload)

		h.RootMerkleHash = bytes.Repeat([]byte{0x11}, sha256.Size)
		if err := WriteHeader(dir, h); err != nil {
			t.Fatalf("заголовок не перезаписан: %v", err)
		}
		// Значение правдоподобной длины, поэтому ReadHeader обязан пропустить
		// его дальше — и упасть уже на сверке с подписанным манифестом.
		if _, err := RestoreBundle(testPassphrase, dir); err == nil {
			t.Fatal("пакет собрался с чужим открытым корнем")
		} else if !strings.Contains(err.Error(), "открытом заголовке") {
			t.Fatalf("ошибка не про расхождение заголовка с манифестом: %v", err)
		}
	})

	t.Run("заголовок без осмысленных значений", func(t *testing.T) {
		dir := t.TempDir()
		e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
		h, _ := createBundle(t, e, dir, payload)

		cases := []struct {
			name string
			dist func(*Header)
		}{
			{"нулевой id", func(x *Header) { x.BackupID = 0 }},
			{"корень не той длины", func(x *Header) { x.RootMerkleHash = x.RootMerkleHash[:8] }},
			{"ноль чанков", func(x *Header) { x.TotalChunks = 0 }},
		}
		for _, c := range cases {
			bad := *h
			c.dist(&bad)
			if err := WriteHeader(dir, &bad); err != nil {
				t.Fatalf("заголовок не перезаписан: %v", err)
			}
			if _, err := ReadHeader(dir); err == nil {
				t.Fatalf("%s: заголовок принят — вызывающий построил бы прогресс по несуществующему пакету", c.name)
			}
		}
	})
}

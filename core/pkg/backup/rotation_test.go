// File: core/pkg/backup/rotation_test.go
//
// Шаг 15: несколько пакетов в одном хранилище, дедупликация между ними и ротация
// с удалением старых чанков. Прежний пакет проверялся одним способом: «один пакет
// на хранилище, повтор его перезаписывает». Здесь проверяются ровно те обещания,
// без которых «инкрементальный бэкап» остаётся словом:
//
//   - второй пакет не стирает первый: у каждого свой слот headers/<ключ>/<id>.json,
//     а header.json — алиас самого свежего, и старый читатель по-прежнему находит
//     актуальный пакет по старому имени;
//   - salt нового пакета берётся у самого свежего, поэтому каталог чанков тот же и
//     неизменное не выкладывается заново. Это проверяется по числу объектов в
//     хранилище, а не по флагу в памяти: «движок решил не выкладывать» видно
//     только по тому, чего в хранилище не прибавилось;
//   - сборщик мусора удаляет только то, что может назвать без догадки: каталог,
//     выведенный из (passphrase, salt удаляемого заголовка) и совпавший с именем
//     слота, и только хэши, на которые не ссылается ни один оставшийся пакет.
//
// Чего здесь нет намеренно: живых облаков. Ротация поверх сети проверяется в
// webdav_test.go, и только против эталонного сервера golang.org/x/net/webdav в
// этом процессе.

package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// sealInto — пакет в хранилище тем же путём, что CreateBundleToProvider, но с
// дешёвым KDF и с явным salt: тестам ротации нужно управлять ключом, а не получать
// «новый на каждый вызов».
func sealInto(t *testing.T, provider CloudProvider, passphrase string, salt []byte, payload []byte) (*Header, []ChunkDescriptor) {
	t.Helper()

	e, err := NewSnapshotEngineWithProvider(passphrase, provider, salt, TestKDF)
	if err != nil {
		t.Fatalf("движок поверх %s не собрался: %v", provider.Name(), err)
	}
	descriptors, totalChunk, err := e.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatalf("чанки не выложились: %v", err)
	}
	h, err := e.Seal(descriptors, uint64(len(payload)), totalChunk)
	if err != nil {
		t.Fatalf("манифест не собрался: %v", err)
	}
	if err := e.WriteBundle(h); err != nil {
		t.Fatalf("записи пакета не выложились: %v", err)
	}
	return h, descriptors
}

func engineOf(t *testing.T, provider CloudProvider, passphrase string, salt []byte) *SnapshotEngine {
	t.Helper()

	e, err := NewSnapshotEngineWithProvider(passphrase, provider, salt, TestKDF)
	if err != nil {
		t.Fatalf("движок не собрался: %v", err)
	}
	return e
}

// treeOf — все файлы хранилища как «slash-имя → размер». Снимается обходом
// диска, а не через провайдер: проверка «удалён ровно этот объект» не должна
// опираться на тот же листинг, чью корректность она проверяет.
func treeOf(t *testing.T, root string) map[string]int64 {
	t.Helper()

	out := map[string]int64{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = info.Size()
		return nil
	})
	if err != nil {
		t.Fatalf("состояние %s не снимается: %v", root, err)
	}
	return out
}

func namesOf(tree map[string]int64) []string {
	out := make([]string, 0, len(tree))
	for name := range tree {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// chunkObject — путь чанка в пакете, собранный здесь заново, а не взятый у
// провайдера: иначе «чанк лежит по такому-то пути» проверялся бы тем же кодом,
// который его туда положил.
func chunkObject(chunkDir, hash string) string {
	return fmt.Sprintf("%s/%s/%s/%s.bin", chunksRootName, chunkDir, hash[:2], hash)
}

// chunkSet — множество хэшей чанков, на которые ссылается пакет или несколько
// пакетов сразу. Строится из манифестов теста, а не из листинга хранилища:
// «удалён ровно этот чанк» нельзя проверять тем же списком, который ротация
// прочитала из хранилища.
func chunkSet(sets ...[]ChunkDescriptor) map[string]bool {
	out := make(map[string]bool)
	for _, set := range sets {
		for _, d := range set {
			out[d.ChunkHash] = true
		}
	}
	return out
}

func mustLocal(t *testing.T, dir string) CloudProvider {
	t.Helper()

	provider, err := NewLocalProvider(dir)
	if err != nil {
		t.Fatalf("локальный провайдер %s не собрался: %v", dir, err)
	}
	return provider
}

func mustStore(t *testing.T, provider CloudProvider) BundleStore {
	t.Helper()

	store, err := AsBundleStore(provider)
	if err != nil {
		t.Fatalf("%s не хранилище пакетов: %v", provider.Name(), err)
	}
	return store
}

func mustSlotName(t *testing.T, chunkDir string, backupID uint64) string {
	t.Helper()

	name, err := headerSlotName(chunkDir, backupID)
	if err != nil {
		t.Fatalf("слот пакета %d в %s не собрать: %v", backupID, chunkDir, err)
	}
	return name
}

func mustMarshal(t *testing.T, h *Header) []byte {
	t.Helper()

	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("заголовок не сериализуется: %v", err)
	}
	return data
}

func writeAt(t *testing.T, root, name string, data []byte) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("файл %s не положить: %v", name, err)
	}
}

func slotList(bundles []Bundle) []string {
	out := make([]string, 0, len(bundles))
	for _, b := range bundles {
		out = append(out, fmt.Sprintf("%s#%d", b.Slot, b.BackupID))
	}
	return out
}

// spread раздвигает следующий пакет по CreatedAt. Порядок ротации определён этим
// полем, а оно считается с точностью до секунды; id различает пакеты и внутри
// секунды, но полагаться на это значило бы проверять не правило, а совпадение.
func spread(t *testing.T, prev *int64) {
	t.Helper()

	for {
		now := time.Now().Unix()
		if now > *prev {
			*prev = now
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// payloadSeeded — тот же детерминированный поток байт, но с собственным счётчиком
// вместо единицы. Нужен там, где невыгодно, что randomPayload даёт вложенные
// префиксы: у надстройки общий начальный отрезок, поэтому границы FastCDC
// сходятся и наборы чанков почти совпадают — удалять было бы нечего.
func payloadSeeded(t *testing.T, n int, seed uint64) []byte {
	t.Helper()

	out := make([]byte, 0, n)
	counter := seed
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

// sortedNames — отсортированный список имён для сравнения списков листинга:
// порядок листинга не определён, а «ровно эти объекты» проверяется именно как
// множество.
func sortedNames(names ...string) []string {
	out := append([]string(nil), names...)
	sort.Strings(out)
	return out
}

func TestHeaderSlotNameIsCanonical(t *testing.T) {
	const dir = "0123abcd"
	const id = uint64(0x1122334455667788)

	name, err := headerSlotName(dir, id)
	if err != nil {
		t.Fatalf("годное имя слота отвергнуто: %v", err)
	}
	if want := "headers/" + dir + "/1122334455667788.json"; name != want {
		t.Fatalf("слот назван %q, ожидалось %q", name, want)
	}
	back, backID, ok := parseHeaderSlot(name)
	if !ok || back != dir || backID != id {
		t.Fatalf("имя %q разобрано как %q/%d/%v", name, back, backID, ok)
	}

	// Нулевой id — не пакет: он неотличим от «поля нет», и слот с таким именем
	// перечислялся бы как точка восстановления, которой нет.
	if _, err := headerSlotName(dir, 0); err == nil {
		t.Fatal("слот с нулевым id собран")
	}
	for _, bad := range []string{dir[:7] + "g", "0123ABCD", "abcd123", "abcd12345"} {
		if _, err := headerSlotName(bad, id); err == nil {
			t.Errorf("каталог %q принят как имя слота", bad)
		}
	}

	// Обратный разбор требует каноничности: верхний регистр, чужая длина,
	// лишние сегменты и ведущие нули в имени id — не слоты.
	for _, bad := range []string{
		HeaderName,
		"chunks/" + dir + "/11/aa.bin",
		"headers/" + dir + "/1122334455667788",
		"headers/" + dir + "/AB22334455667788.json",
		"headers/" + dir + "/01122334455667788.json",
		"headers/" + strings.ToUpper(dir) + "/1122334455667788.json",
		"headers/" + dir + "/nested/1122334455667788.json",
		"headers/" + dir + "/.json",
		"headers//1122334455667788.json",
		"headers/" + dir + "/0000000000000000.json",
	} {
		if _, _, ok := parseHeaderSlot(bad); ok {
			t.Errorf("%q принят списком как слот", bad)
		}
	}

	// Имя, заданное не канонически, пересобранное не даёт: в листинге облака
	// встречается и «%/01/…» вместо «0123abcd».
	if _, _, ok := parseHeaderSlot("headers/abcd1234/1122334455667788.json"); !ok {
		t.Fatal("годное имя не разобралось")
	}
}

func TestNamedHeaderObjectsAreGatedByName(t *testing.T) {
	// Имя приходит строкой (от вызывающего и из листинга облака), поэтому «..»
	// обязано быть отвергнуто до любого обращения к диску: иначе ротация читала бы
	// и удаляла файлы вне пакета.
	root := t.TempDir()
	store := mustStore(t, mustLocal(t, root))
	ctx := context.Background()

	for _, bad := range []string{
		"",
		HeaderName + ".tmp",
		"../header.json",
		"../../etc/passwd",
		"headers/../header.json",
		"headers/abcd1234/../..%2f.json",
		"headers/abcd1234/1122334455667788.JSON",
		"headers/abcd1234/1122334455667788.json/../0000000000000002.json",
		"chunks/abcd1234/0a/" + testHash + ".bin",
		"ygggram/whatever",
	} {
		if err := store.UploadHeaderNamed(ctx, bad, []byte("данные")); err == nil {
			t.Errorf("кладка под именем %q прошла", bad)
		}
		if _, err := store.DownloadHeaderNamed(ctx, bad); err == nil {
			t.Errorf("чтение под именем %q прошла", bad)
		}
		if err := store.DeleteHeaderNamed(ctx, bad); err == nil {
			t.Errorf("удаление под именем %q прошло", bad)
		}
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("отказы всё-таки оставили в %s %d записей (ошибка %v)", root, len(entries), err)
	}

	// Годное имя проходит и кладёт файл ровно туда, где тот и должен лежать.
	good := mustSlotName(t, testChunkDir, 1)
	if err := store.UploadHeaderNamed(ctx, good, []byte("заголовок")); err != nil {
		t.Fatalf("слот не выложился: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(good))); err != nil {
		t.Fatalf("слот лёг не по имени пакета: %v", err)
	}
	if got := treeOf(t, root); len(got) != 1 {
		t.Fatalf("хранилище после одной кладки: %v", namesOf(got))
	}

	// Удаление отсутствующего объекта — успех, а не ошибка: прерванный обход
	// перечитывается с начала, и второе удаление того, чего уже нет, не имеет
	// права быть отказом.
	for i := 0; i < 2; i++ {
		if err := store.DeleteHeaderNamed(ctx, good); err != nil {
			t.Fatalf("удаление слота (заход %d) сочтено ошибкой: %v", i+1, err)
		}
	}
	if err := store.DeleteChunk(ctx, testChunkDir, testHash); err != nil {
		t.Fatalf("удаление несуществующего чанка сочтено ошибкой: %v", err)
	}
}

func TestAsBundleStoreRefusesProvidersWithoutRotation(t *testing.T) {
	// Провайдер без ротации обязан быть назван отказом, а не «тихим no-op»: иначе
	// CreateBundle молча потерял бы слот, и второй пакет стёр бы первый.
	if _, err := AsBundleStore(nil); err == nil {
		t.Fatal("пустой провайдер принят")
	}
	for _, plain := range []CloudProvider{
		&memoryProvider{objects: map[string][]byte{}},
		&listingFailsProvider{},
		&nilMapProvider{},
	} {
		if _, err := AsBundleStore(plain); err == nil ||
			!strings.Contains(err.Error(), "не поддерживает ротацию") {
			t.Errorf("провайдер %s принят хранилищем пакетов: %v", plain.Name(), err)
		}
	}
	for _, provider := range []CloudProvider{mustLocal(t, t.TempDir()), &WebDAVProvider{}} {
		if _, err := AsBundleStore(provider); err != nil {
			t.Errorf("провайдер %s отвергнут как не хранящий пакеты: %v", provider.Name(), err)
		}
	}
}

func TestListBundlesReadsSlotsAndAlias(t *testing.T) {
	root := t.TempDir()
	salt := mustSalt(t)
	payload := randomPayload(t, 200*1024)
	dirName := chunkDirFor(t, testPassphrase, salt)

	// Хранилище пустое: записей нет, и это не ошибка.
	bundles, err := ListBundles(mustLocal(t, root))
	if err != nil {
		t.Fatalf("пустое хранилище названо ошибкой: %v", err)
	}
	if len(bundles) != 0 {
		t.Fatalf("пустое хранилище перечислило %d пакетов", len(bundles))
	}
	if _, err := NewestBundle(mustLocal(t, root)); !errors.Is(err, ErrNoBundles) {
		t.Fatalf("пустое хранилище дало %v, а обязан дать ErrNoBundles", err)
	}
	// «Не умеет листать» и «листнул пусто» — разные отказы: первый обязан быть
	// ошибкой, иначе ротация молча проехала бы по провайдеру без слотов.
	if _, err := NewestBundle(&memoryProvider{objects: map[string][]byte{}}); err == nil ||
		!strings.Contains(err.Error(), "не поддерживает ротацию") {
		t.Fatalf("не-хранилище пакетов описано как пустое хранилище: %v", err)
	}

	// Первый пакет: слот и алиас — это один пакет, а не две точки восстановления.
	provider := mustLocal(t, root)
	header, _ := sealInto(t, provider, testPassphrase, salt, payload)
	bundles, err = ListBundles(provider)
	if err != nil {
		t.Fatalf("один пакет не перечисляется: %v", err)
	}
	if len(bundles) != 1 {
		t.Fatalf("записей %d: алиас и слот должны сойтись в один пакет: %v", len(bundles), slotList(bundles))
	}
	if bundles[0].BackupID != header.BackupID || bundles[0].Header.TotalChunks != header.TotalChunks {
		t.Fatalf("запись названа %d/%d, пакет %d/%d",
			bundles[0].BackupID, bundles[0].Header.TotalChunks, header.BackupID, header.TotalChunks)
	}
	if want := mustSlotName(t, dirName, header.BackupID); bundles[0].Slot != want {
		t.Fatalf("алиас пересилил слот: читатель получил %q, а авторитетнее %q", bundles[0].Slot, want)
	}
	if !bytes.Equal(bundles[0].Header.Manifest, header.Manifest) {
		t.Fatal("заголовок из слота не совпадает с выложенным")
	}

	// Второй пакет того же ключа: две записи, порядок — от нового к старому.
	spread(t, &header.CreatedAt)
	next, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 260*1024))
	bundles, err = ListBundles(provider)
	if err != nil {
		t.Fatalf("два пакета не перечисляются: %v", err)
	}
	if len(bundles) != 2 {
		t.Fatalf("записей %d, ожидалось две: %v", len(bundles), slotList(bundles))
	}
	if bundles[0].BackupID != next.BackupID || bundles[1].BackupID != header.BackupID {
		t.Fatalf("порядок не от нового к старому: %v", slotList(bundles))
	}
	if bundles[0].ChunkDir != dirName || bundles[1].ChunkDir != dirName {
		t.Fatalf("каталоги чанков не разобраны из имён слотов: %q и %q, ожидалось %q",
			bundles[0].ChunkDir, bundles[1].ChunkDir, dirName)
	}

	newest, err := NewestBundle(provider)
	if err != nil || newest.BackupID != next.BackupID {
		t.Fatalf("NewestBundle вернул %d (%v), свежий пакет %d", newest.BackupID, err, next.BackupID)
	}
}

func TestListBundlesRefusesUnreadableAndContradictoryRecords(t *testing.T) {
	// Молча пропущенная запись означала бы, что ротация посчитала пакетов меньше,
	// чем их лежит, и удалила чанки, на которые оставшийся пакет ещё ссылается.
	t.Run("испорченный слот", func(t *testing.T) {
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		header, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 128*1024))
		writeAt(t, root, mustSlotName(t, chunkDirFor(t, testPassphrase, salt), header.BackupID),
			[]byte("не json вовсе"))
		if _, err := ListBundles(provider); err == nil {
			t.Fatal("неразбираемый слот пропущен")
		}
	})

	t.Run("слот врёт про id", func(t *testing.T) {
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		dirName := chunkDirFor(t, testPassphrase, salt)
		header, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 128*1024))

		store := mustStore(t, provider)
		fake := mustSlotName(t, dirName, header.BackupID+1)
		if err := store.UploadHeaderNamed(context.Background(), fake, mustMarshal(t, header)); err != nil {
			t.Fatalf("подставной слот не выложить: %v", err)
		}
		_, err := ListBundles(provider)
		if err == nil || !strings.Contains(err.Error(), "какой каталог чанков этот пакет сторожит") {
			t.Fatalf("расхождение имени слота и id внутри не названо: %v", err)
		}
	})

	t.Run("один пакет в двух слотах", func(t *testing.T) {
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		header, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 128*1024))

		store := mustStore(t, provider)
		// Второй слот с другим именем каталога: содержимое то же, id тот же.
		other := mustSlotName(t, "00000001", header.BackupID)
		if err := store.UploadHeaderNamed(context.Background(), other, mustMarshal(t, header)); err != nil {
			t.Fatalf("второй слот не выложить: %v", err)
		}
		_, err := ListBundles(provider)
		if err == nil || !strings.Contains(err.Error(), "выбирать один из них ротация не вправе") {
			t.Fatalf("дубль слота принят: %v", err)
		}
	})

	t.Run("чужие имена в дереве заголовков", func(t *testing.T) {
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		dirName := chunkDirFor(t, testPassphrase, salt)
		header, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 128*1024))

		for _, junk := range []string{
			"headers/readme.txt",
			"headers/" + dirName + "/notes.txt",
			"headers/" + dirName + "/zzzzzzzzzzzzzzzz.json",
			"headers/junk/" + fmt.Sprintf("%016x", header.BackupID) + ".json",
			"headers/" + dirName + "/" + fmt.Sprintf("%016x", header.BackupID) + ".json.tmp",
		} {
			writeAt(t, root, junk, []byte("не запись пакета"))
		}
		bundles, err := ListBundles(provider)
		if err != nil {
			t.Fatalf("сорняки в дереве заголовков сломали листинг: %v", err)
		}
		if len(bundles) != 1 {
			t.Fatalf("записей %d, ожидалась одна: %v", len(bundles), slotList(bundles))
		}
	})
}

func TestReuseBundleKeyFollowsTheNewestBundle(t *testing.T) {
	root := t.TempDir()
	provider := mustLocal(t, root)

	// Пакетов нет: salt новый, параметры производственные, reused=false.
	salt, params, reused, err := ReuseBundleKey(provider)
	if err != nil {
		t.Fatalf("пустое хранилище названо ошибкой: %v", err)
	}
	if reused {
		t.Fatal("ключ переиспользован там, где переиспользовать нечего")
	}
	if len(salt) != SaltSize {
		t.Fatalf("salt %d байт, а нужно %d", len(salt), SaltSize)
	}
	if params != ProductionKDF {
		t.Fatalf("параметры первого пакета %+v, ожидался производственный профиль %+v", params, ProductionKDF)
	}

	// Свежий заголовок диктует и salt, и параметры: именно так чанки второго
	// пакета попадают в тот же каталог.
	header, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 128*1024))
	gotSalt, gotParams, reused, err := ReuseBundleKey(provider)
	if err != nil {
		t.Fatalf("ключ существующего пакета не взять: %v", err)
	}
	if !reused {
		t.Fatal("второй пакет объявлен первым")
	}
	if !bytes.Equal(gotSalt, header.Salt) {
		t.Fatalf("salt %x, а у свежего пакета %x", gotSalt, header.Salt)
	}
	if gotParams.Time != header.Time || gotParams.MemoryKiB != header.MemoryKiB ||
		gotParams.Threads != header.Threads {
		t.Fatalf("параметры KDF %+v не совпадают с заголовком %+v/%d/%d",
			gotParams, header.Time, header.MemoryKiB, header.Threads)
	}

	// Граница риска: алиас переписан на заголовок, который не читается, — и
	// ReuseBundleKey обязан отказать, а не «молча взять новый ключ». С новым
	// ключом пакет лёг бы в новый каталог и выложил бы всё заново, а вызывающий
	// узнал об этом только по размеру хранилища.
	writeAt(t, root, HeaderName, []byte(`{"magic":"ygggram-backup-v1","backup_id":7}`))
	if _, _, _, err := ReuseBundleKey(provider); err == nil {
		t.Fatal("неполный алиас обошёл проверку ключа")
	}
}

func TestSecondBundleReusesChunksOfTheFirst(t *testing.T) {
	// Главный факт Шага 15, измеренный числом объектов в хранилище, а не флагом в
	// памяти процесса.
	root := t.TempDir()
	salt := mustSalt(t)
	payload := randomPayload(t, 300*1024)

	_, first := sealInto(t, mustLocal(t, root), testPassphrase, salt, payload)
	afterFirst := treeOf(t, root)
	if len(afterFirst) != len(first)+2 {
		t.Fatalf("первый пакет: файлов %d, ожидалось %d (чанки + слот + алиас): %v",
			len(afterFirst), len(first)+2, namesOf(afterFirst))
	}

	// Тот же ключ, тот же контент, новый движок: ни одного чанка не выложено,
	// прибавились бы только две записи о пакете.
	e := engineOf(t, mustLocal(t, root), testPassphrase, salt)
	if e.ReusedKey() {
		t.Fatal("движок, построенный с явным salt, объявил ключ переиспользованным")
	}
	descriptors, _, err := e.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatalf("раскладка не удалась: %v", err)
	}
	uploaded, already := e.DedupStats()
	if uploaded != 0 || already != uint32(len(descriptors)) {
		t.Fatalf("выложено %d чанков при %d переиспользованных: дедуп между пакетами не работает",
			uploaded, already)
	}
	if err := e.WriteBundle(mustSeal(t, e, descriptors, uint64(len(payload)))); err != nil {
		t.Fatalf("второй пакет не выложился: %v", err)
	}
	afterSecond := treeOf(t, root)
	// Прибавился ровно один объект — слот второго пакета: алиас уже лежит, и
	// второй пакет его перезаписывает, а не множит. Чанки не добавились ни одним:
	// это и есть измеренный в хранилище дедуп между пакетами.
	if len(afterSecond) != len(afterFirst)+1 {
		t.Fatalf("второй пакет добавил %d файлов, ожидался 1 (слот): %v",
			len(afterSecond)-len(afterFirst), namesOf(afterSecond))
	}
	dirName := chunkDirFor(t, testPassphrase, salt)
	if _, ok := afterSecond[mustSlotName(t, dirName, e.BackupID())]; !ok {
		t.Fatalf("слот второго пакета в хранилище не лёг: %v", namesOf(afterSecond))
	}
	if alias, err := os.ReadFile(filepath.Join(root, HeaderName)); err != nil {
		t.Fatalf("алиас пропал: %v", err)
	} else if ah, err := parseHeader(alias); err != nil || ah.BackupID != e.BackupID() {
		t.Fatalf("алиас после второго пакета указывает не на него (%+v): %v", ah, err)
	}
	for name, size := range afterFirst {
		if name == HeaderName {
			continue // алиас перезаписывается — в этом и состоит его роль
		}
		if afterSecond[name] != size {
			t.Fatalf("файл %s изменился с %d до %d: повторный пакет тронул чужие байты",
				name, size, afterSecond[name])
		}
	}

	// Тот же итог через полный путь CreateBundleToProvider: вызывающий видит
	// переиспользование по счётчикам движка, а не по названию команды.
	dirB := t.TempDir()
	providerB := mustLocal(t, dirB)
	engine, err := NewSnapshotEngineToProvider(testPassphrase, providerB)
	if err != nil {
		t.Fatalf("первый пакет не завёлся: %v", err)
	}
	if engine.ReusedKey() {
		t.Fatal("пакет в пустом хранилище назван переиспользующим ключ")
	}
	dB, totalB, err := engine.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatal(err)
	}
	if up, _ := engine.DedupStats(); up != uint32(len(dB)) {
		t.Fatalf("первый пакет выложил %d чанков из %d", up, len(dB))
	}
	if err := engine.WriteBundle(mustSeal(t, engine, dB, uint64(len(payload)))); err != nil {
		t.Fatalf("первый пакет не выложен: %v", err)
	}

	second, err := NewSnapshotEngineToProvider(testPassphrase, mustLocal(t, dirB))
	if err != nil {
		t.Fatalf("второй пакет не завёлся: %v", err)
	}
	if !second.ReusedKey() {
		t.Fatal("второй пакет в непустом хранилище объявлен первым")
	}
	d2, total2, err := second.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatal(err)
	}
	if up, already := second.DedupStats(); up != 0 || already != uint32(len(d2)) {
		t.Fatalf("повторный пакет выложил %d чанков из %d при том же контенте", up, len(d2))
	}
	if total2 != totalB {
		t.Fatalf("объём шифртекста изменился: %d против %d при тех же чанках", total2, totalB)
	}
	if err := second.WriteBundle(mustSeal(t, second, d2, uint64(len(payload)))); err != nil {
		t.Fatalf("второй пакет по полному пути не выложился: %v", err)
	}
	if treeB := treeOf(t, dirB); len(treeB) != len(dB)+3 {
		t.Fatalf("в %s %d файлов при %d чанках (ожидалось чанки + два слота + алиас): %v",
			dirB, len(treeB), len(dB), namesOf(treeB))
	}
}

func TestForeignPassphraseGetsItsOwnChunkDir(t *testing.T) {
	// Разделение каталогов по ключу — то, что мешает второму пакету прочитать
	// чужой шифртекст вместо своего.
	root := t.TempDir()
	salt := mustSalt(t)
	payload := randomPayload(t, 256*1024)
	own := chunkDirFor(t, testPassphrase, salt)

	_, mine := sealInto(t, mustLocal(t, root), testPassphrase, salt, payload)
	foreignSalt := mustSalt(t)
	_, theirs := sealInto(t, mustLocal(t, root), "совсем другая фраза", foreignSalt, payload)

	foreignDir := chunkDirFor(t, "совсем другая фраза", foreignSalt)
	if foreignDir == own {
		t.Fatal("две passphrase дали один каталог чанков")
	}
	tree := treeOf(t, root)
	for _, d := range mine {
		if _, ok := tree[chunkObject(own, d.ChunkHash)]; !ok {
			t.Fatalf("чанк %s своего пакета не лежит в своём каталоге", d.ChunkHash)
		}
	}
	for _, d := range theirs {
		if _, ok := tree[chunkObject(foreignDir, d.ChunkHash)]; !ok {
			t.Fatalf("чанк %s чужого пакета не лежит в чужом каталоге", d.ChunkHash)
		}
	}

	// Алиас указывает на чужой пакет, и своя passphrase обязана его отвергнуть:
	// ключ выводится из фразы, а не угадывается по содержимому.
	if _, err := RestoreBundleFromProvider("совсем другая фраза", mustLocal(t, root)); err != nil {
		t.Fatalf("свежий чужой пакет не читается своей фразой: %v", err)
	}
	if _, err := RestoreBundleFromProvider(testPassphrase, mustLocal(t, root)); err == nil {
		t.Fatal("алиас указывает на чужой пакет, и своя passphrase его открыла")
	}
}

func TestWriteBundleRefusesForeignHeader(t *testing.T) {
	// Слот называется id заголовка, и заголовок от другого движка дал бы слот,
	// под которым лежит чужое содержимое.
	root := t.TempDir()
	salt := mustSalt(t)
	e := engineOf(t, mustLocal(t, root), testPassphrase, salt)

	if err := e.WriteBundle(&Header{Magic: headerMagic, BackupID: e.BackupID() + 1}); err == nil ||
		!strings.Contains(err.Error(), "слот был бы назван id, который внутри не живёт") {
		t.Fatalf("чужой заголовок принят: %v", err)
	}
	if err := e.WriteBundle(nil); err == nil {
		t.Fatal("пустой заголовок принят")
	}
	if tree := treeOf(t, root); len(tree) != 0 {
		t.Fatalf("отказ оставил в хранилище %v", namesOf(tree))
	}
}

func TestPruneKeepsNewestAndDeletesOnlyOrphans(t *testing.T) {
	// Ровно тот случай, ради которого введена ротация: три пакета, keep=2, и
	// удалиться должен только самый старый — вместе с чанками, на которые больше
	// никто не ссылается, и ни одним чанком больше.
	root := t.TempDir()
	salt := mustSalt(t)
	provider := mustLocal(t, root)
	dirName := chunkDirFor(t, testPassphrase, salt)

	v1 := randomPayload(t, 300*1024)
	h1, d1 := sealInto(t, provider, testPassphrase, salt, v1)
	spread(t, &h1.CreatedAt)

	v2 := append(append([]byte(nil), v1...), randomPayload(t, 120*1024)...)
	h2, d2 := sealInto(t, provider, testPassphrase, salt, v2)
	spread(t, &h2.CreatedAt)

	v3 := append(append([]byte(nil), v2...), randomPayload(t, 100*1024)...)
	h3, d3 := sealInto(t, provider, testPassphrase, salt, v3)

	before := treeOf(t, root)
	if _, ok := before[mustSlotName(t, dirName, h1.BackupID)]; !ok {
		t.Fatalf("самого старого слота в хранилище нет до ротации: %v", namesOf(before))
	}
	for _, d := range d1 {
		if _, ok := before[chunkObject(dirName, d.ChunkHash)]; !ok {
			t.Fatalf("чанк %s первого пакета не лежит в хранилище до ротации: проверять нечего", d.ChunkHash)
		}
	}
	rep, err := PruneBundles(testPassphrase, provider, 2)
	if err != nil {
		t.Fatalf("ротация не удалась: %v", err)
	}
	if rep.Bundles != 3 || rep.Dropped != 1 || rep.Kept != 2 {
		t.Fatalf("ротация отчиталась %d/%d/%d, ожидалось 3/1/2", rep.Bundles, rep.Dropped, rep.Kept)
	}
	if len(rep.Skipped) != 0 {
		t.Fatalf("ротация пропустила то, что обязана была убрать: %+v", rep.Skipped)
	}
	if want := mustSlotName(t, dirName, h3.BackupID); rep.AliasSlot != want {
		t.Fatalf("алиасом назван %q, свежий слот %q", rep.AliasSlot, want)
	}

	// Удалиться обязано ровно то, на что не ссылается ни h2, ни h3.
	refs := chunkSet(d2, d3)
	wantDeleted := 0
	var wantBytes uint64
	for _, d := range d1 {
		if refs[d.ChunkHash] {
			continue
		}
		wantDeleted++
		wantBytes += uint64(d.EncryptedSize)
	}
	if wantDeleted == 0 {
		t.Fatal("первый пакет целиком переиспользован вторым: проверять нечего")
	}
	if rep.ChunksDeleted != wantDeleted || rep.BytesFreed != wantBytes {
		t.Fatalf("ротация убрала %d чанков (%d байт), а не %d (%d байт)",
			rep.ChunksDeleted, rep.BytesFreed, wantDeleted, wantBytes)
	}

	after := treeOf(t, root)
	for _, d := range d1 {
		name := chunkObject(dirName, d.ChunkHash)
		_, exists := after[name]
		if want := refs[d.ChunkHash]; want != exists {
			t.Fatalf("чанк %s: в хранилище=%v, на него ссылается оставленный пакет=%v",
				d.ChunkHash, exists, want)
		}
	}
	// Ни один чанк оставленных пакетов не пострадал, и дерево чанков — ровно их
	// объединение: ни живого чанка не пропало, ни одного сироты не осталось.
	want := chunkSet(d2, d3)
	for hash := range want {
		if _, ok := after[chunkObject(dirName, hash)]; !ok {
			t.Fatalf("чанк %s оставленного пакета удалён", hash)
		}
	}
	now := map[string]bool{}
	for name := range after {
		if !strings.HasPrefix(name, chunksRootName+"/") {
			continue
		}
		hash := strings.TrimSuffix(filepath.Base(name), ".bin")
		if _, ok := want[hash]; !ok {
			t.Fatalf("сирота %s пережил ротацию: %v", hash, name)
		}
		if name != chunkObject(dirName, hash) {
			t.Fatalf("чанк %s лежит не в своём каталоге: %q", hash, name)
		}
		now[hash] = true
	}
	if len(now) != len(want) {
		t.Fatalf("в хранилище %d чанков, оставленные пакеты ссылаются на %d: %+v", len(now), len(want), namesOf(after))
	}

	// Записи: слот самого старого пакета убран, два остальных целы.
	for _, id := range []uint64{h1.BackupID, h2.BackupID, h3.BackupID} {
		slot := mustSlotName(t, dirName, id)
		_, exists := after[slot]
		if want := id != h1.BackupID; exists != want {
			t.Fatalf("слот %s: есть=%v, ожидалось %v", slot, exists, want)
		}
	}

	// Алиас — байты самого свежего слота, и пакет по-прежнему восстанавливается.
	alias, err := os.ReadFile(filepath.Join(root, HeaderName))
	if err != nil {
		t.Fatalf("алиас пропал: %v", err)
	}
	if !bytes.Equal(alias, mustMarshal(t, h3)) {
		t.Fatal("алиас после ротации не совпадает с самым свежим пакетом")
	}
	bundles, err := ListBundles(mustLocal(t, root))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundles) != 2 {
		t.Fatalf("после ротации записей %d: %v", len(bundles), slotList(bundles))
	}
	e, err := NewSnapshotEngineWithProvider(testPassphrase, mustLocal(t, root), salt, TestKDF)
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.Restore(bundles[1].Header)
	if err != nil {
		t.Fatalf("второй по свежести пакет не раскрылся: %v", err)
	}
	if !bytes.Equal(got, v2) {
		t.Fatalf("раскрыто %d байт, в пакете %d", len(got), len(v2))
	}
	// Тот же пакет читался и до ротации — иначе «читается после» ничего не стоило
	// бы: сравнение одними и теми же байтами по обе стороны удаления.
	gotBefore, err := e.Restore(h2)
	if err != nil {
		t.Fatalf("второй по свежести пакет до ротации не раскрылся: %v", err)
	}
	if !bytes.Equal(gotBefore, v2) {
		t.Fatalf("до ротации раскрыто %d байт, в пакете %d", len(gotBefore), len(v2))
	}
}

func TestPruneProtectsSharedAndUnreadable(t *testing.T) {
	t.Run("общий чанк двух удаляемых пакетов считается один раз", func(t *testing.T) {
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		dirName := chunkDirFor(t, testPassphrase, salt)
		payload := randomPayload(t, 300*1024)

		h1, d1 := sealInto(t, provider, testPassphrase, salt, payload)
		spread(t, &h1.CreatedAt)
		h2, d2 := sealInto(t, provider, testPassphrase, salt, payload)
		spread(t, &h2.CreatedAt)

		// Третий пакет — не надстройка первых 300К: randomPayload детерминирован, и
		// 400К байт начинаются ровно с тех же байт, поэтому его манифест держал бы
		// почти все чанки первых двух и считать однократно было бы нечего. Первые
		// другой поток байт, а не надстройка первых 300К: у надстройки общий
		// начальный отрезок, поэтому её манифест держал бы почти все чанки первых
		// двух и считать однократно было бы нечего.
		other := payloadSeeded(t, 400*1024, 917)
		_, d3 := sealInto(t, provider, testPassphrase, salt, other)

		if len(d1) != len(d2) || len(chunkSet(d1, d2)) != len(d1) {
			t.Fatalf("два пакета одного содержимого дали разные наборы чанков: %d против %d", len(d1), len(d2))
		}

		// Ожидаемое удаление: каждый хэш набора двух одинаковых пакетов ровно один
		// раз, кроме тех, что остались в манифесте третьего.
		kept := chunkSet(d3)
		wanted := map[string]uint32{}
		for _, d := range append(append([]ChunkDescriptor{}, d1...), d2...) {
			if kept[d.ChunkHash] {
				continue
			}
			if _, seen := wanted[d.ChunkHash]; !seen {
				wanted[d.ChunkHash] = d.EncryptedSize
			}
		}
		if len(wanted) < 2 {
			t.Fatalf("сирот %d: однократный счёт проверять нечем", len(wanted))
		}

		rep, err := PruneBundles(testPassphrase, provider, 1)
		if err != nil {
			t.Fatalf("ротация не удалась: %v", err)
		}
		if rep.Dropped != 2 {
			t.Fatalf("убрано записей %d, ожидалось две: %+v", rep.Dropped, rep.Skipped)
		}
		// Общий чанк двух удаляемых пакетов удалён и сосчитан один раз: при двойном
		// счёте число было бы вдвое больше.
		if rep.ChunksDeleted != len(wanted) {
			t.Fatalf("удалено %d чанков, ожидалось %d: общий чанк двух пакетов сосчитан дважды",
				rep.ChunksDeleted, len(wanted))
		}
		var wantBytes uint64
		for _, size := range wanted {
			wantBytes += uint64(size)
		}
		if rep.BytesFreed != wantBytes {
			t.Fatalf("освобождено %d байт, а по манифесту %d", rep.BytesFreed, wantBytes)
		}
		tree := treeOf(t, root)
		for _, d := range d1 {
			_, present := tree[chunkObject(dirName, d.ChunkHash)]
			if want := kept[d.ChunkHash]; want != present {
				t.Fatalf("чанк %s: в хранилище=%v, оставленный пакет ссылается=%v", d.ChunkHash, present, want)
			}
		}
		for _, d := range d3 {
			if _, ok := tree[chunkObject(dirName, d.ChunkHash)]; !ok {
				t.Fatalf("чанк %s оставленного пакета удалён", d.ChunkHash)
			}
		}
	})

	t.Run("не вскрытый оставленный пакет защищает весь каталог", func(t *testing.T) {
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		dirName := chunkDirFor(t, testPassphrase, salt)

		old, dOld := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 300*1024))
		spread(t, &old.CreatedAt)
		header, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 200*1024))

		// Свежий пакет испорчен по части манифеста: подпись не сходится, ключ-то
		// выводится. Оставлять такой пакет нельзя, но и удалять его чанки нечем —
		// значит под защитой оказывается весь каталог, включая чанки старого.
		store := mustStore(t, provider)
		header.Manifest = bytes.Repeat([]byte{0xaa}, 96)
		header.ManifestMAC = bytes.Repeat([]byte{0xbb}, 32)
		ctx := context.Background()
		if err := store.UploadHeaderNamed(ctx, mustSlotName(t, dirName, header.BackupID),
			mustMarshal(t, header)); err != nil {
			t.Fatalf("испорченный слот не выложить: %v", err)
		}
		if err := store.UploadHeaderNamed(ctx, HeaderName, mustMarshal(t, header)); err != nil {
			t.Fatalf("алиас не переписать: %v", err)
		}

		rep, err := PruneBundles(testPassphrase, provider, 1)
		if err != nil {
			t.Fatalf("ротация с непросматриваемым манифестом названа ошибкой: %v", err)
		}
		if rep.Dropped != 0 || len(rep.Skipped) == 0 {
			t.Fatalf("убрано %d записей при %d пропусках: защита каталога не сработала",
				rep.Dropped, len(rep.Skipped))
		}
		tree := treeOf(t, root)
		for _, d := range dOld {
			if _, ok := tree[chunkObject(dirName, d.ChunkHash)]; !ok {
				t.Fatalf("чанк %s удалён, хотя каталог под защитой: %+v", d.ChunkHash, rep.Skipped)
			}
		}
		if !strings.Contains(rep.Skipped[0].Reason, "защищён целиком") &&
			!strings.Contains(rep.Skipped[0].Reason, "не трогать") {
			t.Fatalf("пропуск не объяснён: %q", rep.Skipped[0].Reason)
		}
	})

	t.Run("расхождение имени слота и ключа", func(t *testing.T) {
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		dirName := chunkDirFor(t, testPassphrase, salt)

		old, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 200*1024))
		spread(t, &old.CreatedAt)
		header, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 200*1024))

		// Слот с чужим именем каталога: по такому имени удаление началось бы с
		// догадки, какой каталог настоящий. Настоящий слот убирается, иначе
		// ListBundles законно откажется выбирать между двумя записями одного id.
		store := mustStore(t, provider)
		ctx := context.Background()
		if err := store.UploadHeaderNamed(ctx, mustSlotName(t, "ffffffff", header.BackupID),
			mustMarshal(t, header)); err != nil {
			t.Fatalf("подозрительный слот не выложить: %v", err)
		}
		if err := store.DeleteHeaderNamed(ctx, mustSlotName(t, dirName, header.BackupID)); err != nil {
			t.Fatalf("настоящий слот не убрать: %v", err)
		}
		if err := store.UploadHeaderNamed(ctx, HeaderName, mustMarshal(t, header)); err != nil {
			t.Fatalf("алиас не переписать: %v", err)
		}

		rep, err := PruneBundles(testPassphrase, provider, 1)
		if err != nil {
			t.Fatalf("ротация с подозрительным слотом названа ошибкой: %v", err)
		}
		if rep.Dropped != 0 {
			t.Fatalf("пакет с расходящимся именем слота всё-таки убран: %+v", rep)
		}
		found := false
		for _, s := range rep.Skipped {
			if strings.Contains(s.Reason, "выводит") {
				found = true
			}
		}
		if !found {
			t.Fatalf("расхождение не названо среди пропусков: %+v", rep.Skipped)
		}
	})
}

func TestPruneBoundaries(t *testing.T) {
	t.Run("keep меньше единицы отвергается", func(t *testing.T) {
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		sealInto(t, provider, testPassphrase, salt, randomPayload(t, 200*1024))

		before := treeOf(t, root)
		for _, keep := range []int{0, -1} {
			_, err := PruneBundles(testPassphrase, provider, keep)
			if err == nil || !strings.Contains(err.Error(), "хотя бы один пакет") {
				t.Fatalf("keep=%d принят: %v", keep, err)
			}
		}
		if after := treeOf(t, root); len(after) != len(before) {
			t.Fatalf("отказ тронул хранилище: %v", namesOf(after))
		}
	})

	t.Run("keep не меньше числа пакетов — чистый no-op", func(t *testing.T) {
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		header, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 200*1024))
		spread(t, &header.CreatedAt)
		sealInto(t, provider, testPassphrase, salt, randomPayload(t, 260*1024))

		before := treeOf(t, root)
		for _, keep := range []int{2, 3, 100} {
			rep, err := PruneBundles(testPassphrase, provider, keep)
			if err != nil {
				t.Fatalf("keep=%d: %v", keep, err)
			}
			if rep.Dropped != 0 || rep.ChunksDeleted != 0 || rep.Kept != 2 || rep.Bundles != 2 {
				t.Fatalf("keep=%d дал %+v", keep, rep)
			}
			if rep.AliasSlot == "" {
				t.Fatalf("keep=%d: отчёт не называет, на что указывает алиас", keep)
			}
		}
		if after := treeOf(t, root); len(after) != len(before) {
			t.Fatalf("no-op тронул хранилище: %v", namesOf(after))
		}
	})

	t.Run("пустое хранилище — не ошибка", func(t *testing.T) {
		rep, err := PruneBundles(testPassphrase, mustLocal(t, t.TempDir()), 1)
		if err != nil {
			t.Fatalf("ротация по пустому хранилищу названа ошибкой: %v", err)
		}
		if rep.Bundles != 0 || rep.Dropped != 0 || rep.Kept != 0 {
			t.Fatalf("пустое хранилище отчиталось %+v", rep)
		}
	})

	t.Run("чужая passphrase не удаляет ничего", func(t *testing.T) {
		// Удаление возможно только по манифесту, который вскрылся данной
		// passphrase. Чужой каталог для этого кода не вычислим — значит, не может
		// быть и удалён.
		root := t.TempDir()
		salt := mustSalt(t)
		provider := mustLocal(t, root)
		for i := 0; i < 3; i++ {
			header, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, (200+40*i)*1024))
			spread(t, &header.CreatedAt)
		}
		before := treeOf(t, root)

		rep, err := PruneBundles("не та passphrase", provider, 1)
		if err != nil {
			t.Fatalf("ротация чужой фразой названа ошибкой: %v", err)
		}
		if rep.Dropped != 0 || rep.ChunksDeleted != 0 || rep.BytesFreed != 0 {
			t.Fatalf("ротация чужой фразой что-то удалила: %+v", rep)
		}
		if after := treeOf(t, root); len(after) != len(before) {
			t.Fatalf("хранилище изменилось: %v", namesOf(after))
		}
		for _, s := range rep.Skipped {
			if s.Reason == "" {
				t.Fatal("пропуск без причины")
			}
		}
	})

	t.Run("провайдер без ротации отвергается", func(t *testing.T) {
		_, err := PruneBundles(testPassphrase, &memoryProvider{objects: map[string][]byte{}}, 1)
		if err == nil {
			t.Fatal("ротация по провайдеру без слотов прошла")
		}
	})
}

func TestPruneIsRepeatableAfterInterruption(t *testing.T) {
	// Обрыв посреди удаления обязан сходить «с начала» до конца: чанки убираются
	// первыми, слот — последним. Если бы было наоборот, осиротевшие чанки нечем
	// было бы перечислить, и их место осталось бы занято навсегда.
	root := t.TempDir()
	salt := mustSalt(t)
	provider := mustLocal(t, root)
	dirName := chunkDirFor(t, testPassphrase, salt)

	v1 := payloadSeeded(t, 1600*1024, 404)
	h1, d1 := sealInto(t, provider, testPassphrase, salt, v1)
	spread(t, &h1.CreatedAt)
	// Ни один пакет не надстройка другого: randomPayload детерминирован и длинный
	// набор начинается с короткого, поэтому у общего начала сирот было бы двое и
	// обрывать посреди удаления было бы негде. Разные затравки дают непересекающиеся
	// чанки.
	v2 := payloadSeeded(t, 400*1024, 733)
	h2, d2 := sealInto(t, provider, testPassphrase, salt, v2)
	spread(t, &h2.CreatedAt)
	v3 := payloadSeeded(t, 500*1024, 991)
	h3, d3 := sealInto(t, provider, testPassphrase, salt, v3)

	kept := chunkSet(d3)
	// Обрыв приходится на удаление чанков самого старого обречённого пакета: до его
	// слота очередь не доходит, поэтому запись обязана пережить обрыв.
	firstOrphans := 0
	for hash := range chunkSet(d1) {
		if !kept[hash] {
			firstOrphans++
		}
	}
	if firstOrphans < 3 {
		t.Fatalf("у первого пакета %d сирот: обрыва посреди его удаления не получится устроить", firstOrphans)
	}
	orphans := firstOrphans
	for hash := range chunkSet(d2) {
		if !kept[hash] {
			orphans++
		}
	}

	failing := &failOnceStore{BundleStore: mustStore(t, provider), failOn: firstOrphans/2 + 1}
	if _, err := PruneBundles(testPassphrase, failing, 1); err == nil {
		t.Fatal("ротация не заметила отказ хранилища")
	}
	// Прерванный заход не потерял ни одной записи: чанки уходят первыми, слот —
	// последним, поэтому перечислимость пакета переживает обрыв и есть чем
	// перечитать оставшиеся чанки.
	bundles, err := ListBundles(provider)
	if err != nil {
		t.Fatalf("записи после прерванной ротации не перечисляются: %v", err)
	}
	if len(bundles) != 3 {
		t.Fatalf("прерванная ротация потеряла запись: записей %d, ожидалось три: %v", len(bundles), slotList(bundles))
	}

	// Обрыв попал в середину удаления, а не в его начало или конец: часть сирот уже
	// стёрта, часть ещё лежит, а слот самого старого пакета цел. Иначе проверка
	// «перечитать с начала» ничего не стоила бы.
	partial := treeOf(t, root)
	if _, ok := partial[mustSlotName(t, dirName, h1.BackupID)]; !ok {
		t.Fatalf("прерванная ротация успела стереть слот самого старого пакета: %v", namesOf(partial))
	}
	deleted := 0
	for hash := range chunkSet(d1, d2) {
		if kept[hash] {
			continue
		}
		if _, ok := partial[chunkObject(dirName, hash)]; !ok {
			deleted++
		}
	}
	if deleted == 0 || deleted >= orphans {
		t.Fatalf("обрыв не попал в середину удаления: стёрто %d сирот из %d", deleted, orphans)
	}

	// Тот же заход ещё раз, уже по нормальному хранилищу: доводит дело до конца.
	if _, err := PruneBundles(testPassphrase, provider, 1); err != nil {
		t.Fatalf("доводка ротации не удалась: %v", err)
	}
	tree := treeOf(t, root)
	if _, ok := tree[mustSlotName(t, dirName, h3.BackupID)]; !ok {
		t.Fatalf("свежий пакет потерял слот после прерванной ротации: %v", namesOf(tree))
	}
	if _, ok := tree[mustSlotName(t, dirName, h2.BackupID)]; ok {
		t.Fatalf("старый слот пережил доведённую ротацию")
	}
	// Дерево чанков сошлось на набор оставленного пакета: ни сироты, ни потерянного
	// чанка.
	now := map[string]bool{}
	for name := range tree {
		if !strings.HasPrefix(name, chunksRootName+"/") {
			continue
		}
		hash := strings.TrimSuffix(filepath.Base(name), ".bin")
		if name != chunkObject(dirName, hash) {
			t.Fatalf("чанк %s лежит не в своём каталоге: %q", hash, name)
		}
		if !kept[hash] {
			t.Fatalf("сирота %s пережил доведённую ротацию", hash)
		}
		now[hash] = true
	}
	for hash := range kept {
		if !now[hash] {
			t.Fatalf("чанк %s оставленного пакета удалён", hash)
		}
	}
	// Третий заход — чистый no-op: ротация сошлась, ничего не осталось.
	again, err := PruneBundles(testPassphrase, provider, 1)
	if err != nil {
		t.Fatalf("повторная ротация: %v", err)
	}
	if again.Dropped != 0 || again.ChunksDeleted != 0 {
		t.Fatalf("ротация не сошлась: %+v", again)
	}
	if _, err := RestoreBundleFromProvider(testPassphrase, mustLocal(t, root)); err != nil {
		t.Fatalf("свежий пакет после прерванной ротации не восстанавливается: %v", err)
	}
}

func TestAliasTracksTheNewestBundleAfterPrune(t *testing.T) {
	// Алиас — единственный объект, который ротация передвигает, и он обязан
	// указывать на оставшийся свежий пакет: прежний код, который читает по имени,
	// не должен получить «пакета нет» там, где пакет есть.
	root := t.TempDir()
	salt := mustSalt(t)
	provider := mustLocal(t, root)
	dirName := chunkDirFor(t, testPassphrase, salt)

	first, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 200*1024))
	spread(t, &first.CreatedAt)
	second, _ := sealInto(t, provider, testPassphrase, salt, randomPayload(t, 240*1024))

	// Алиас отстал: на нём лежит запись первого пакета, хотя свежий — второй.
	store := mustStore(t, provider)
	ctx := context.Background()
	if err := store.UploadHeaderNamed(ctx, HeaderName, mustMarshal(t, first)); err != nil {
		t.Fatalf("алиас не переписать: %v", err)
	}

	rep, err := PruneBundles(testPassphrase, provider, 1)
	if err != nil {
		t.Fatalf("ротация не удалась: %v", err)
	}
	if rep.Dropped != 1 {
		t.Fatalf("убрана %d запись, ожидалась одна", rep.Dropped)
	}
	if want := mustSlotName(t, dirName, second.BackupID); rep.AliasSlot != want {
		t.Fatalf("отчёт называет алиасом %q, свежий слот %q", rep.AliasSlot, want)
	}
	alias, err := os.ReadFile(filepath.Join(root, HeaderName))
	if err != nil {
		t.Fatalf("алиас пропал: %v", err)
	}
	if !bytes.Equal(alias, mustMarshal(t, second)) {
		t.Fatal("алиас не стал копией самого свежего слота")
	}
	if _, err := RestoreBundleFromProvider(testPassphrase, mustLocal(t, root)); err != nil {
		t.Fatalf("восстановление по алиасу после ротации: %v", err)
	}
}

func TestLegacySingleHeaderStorageStillReads(t *testing.T) {
	// Пакет, снятый прежним кодом: один header.json, ни одного слота. Новый код
	// обязан прочесть его, иначе обновление ядра сделало бы единственный имеющийся
	// бэкап нечитаемым.
	root := t.TempDir()
	salt := mustSalt(t)
	payload := randomPayload(t, 300*1024)

	provider := mustLocal(t, root)
	e := engineOf(t, provider, testPassphrase, salt)
	descriptors, total, err := e.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatal(err)
	}
	if total != sumEncrypted(descriptors) {
		t.Fatalf("движок назвал объём шифртекста %d, а по манифесту %d", total, sumEncrypted(descriptors))
	}
	header := mustSeal(t, e, descriptors, uint64(len(payload)))
	if err := WriteHeaderTo(provider, header); err != nil {
		t.Fatalf("заголовок прежним способом не выложить: %v", err)
	}
	if tree := treeOf(t, root); len(tree) != len(descriptors)+1 {
		t.Fatalf("прежняя раскладка дала %d файлов: %v", len(tree), namesOf(tree))
	}

	bundles, err := ListBundles(provider)
	if err != nil {
		t.Fatalf("пакет без слота не перечисляется: %v", err)
	}
	if len(bundles) != 1 || bundles[0].Slot != HeaderName {
		t.Fatalf("прежний пакет прочитан как %+v", bundles)
	}
	if bundles[0].ChunkDir != "" {
		t.Fatalf("у алиаса не может быть каталога в имени, а получено %q", bundles[0].ChunkDir)
	}
	got, err := RestoreBundleFromProvider(testPassphrase, provider)
	if err != nil {
		t.Fatalf("прежний пакет не восстанавливается: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("прежний пакет раскрылся другими байтами")
	}

	// Ротация одного прежнего пакета ничего не делает и ничего не портит: он и
	// есть последняя точка восстановления, а убрать её — значит снять обещание.
	before := treeOf(t, root)
	rep, err := PruneBundles(testPassphrase, provider, 1)
	if err != nil {
		t.Fatalf("ротация прежнего пакета названа ошибкой: %v", err)
	}
	if rep.Dropped != 0 || rep.Bundles != 1 || rep.Kept != 1 {
		t.Fatalf("ротация прежнего пакета: %+v", rep)
	}
	if after := treeOf(t, root); len(after) != len(before) {
		t.Fatalf("ротация тронула прежний пакет: %v", namesOf(after))
	}

	// Новый пакет рядом с прежним: прежний получает слот и остаётся читаемой
	// точкой, а алиас переставляется на новый. Без переноса в слот первый же
	// Шаг 15 бэкап стёр бы единственную запись о прежнем пакете — то есть сделал
	// бы ровно то, от чего этот шаг бережёт.
	spread(t, &header.CreatedAt)
	legacyID := header.BackupID
	newHeader, _ := sealInto(t, provider, testPassphrase, salt, append(payload, randomPayload(t, 100*1024)...))
	bundles, err = ListBundles(provider)
	if err != nil {
		t.Fatalf("прежний и новый пакет не сосуществуют: %v", err)
	}
	if len(bundles) != 2 {
		t.Fatalf("записей %d, ожидалось две: %v", len(bundles), slotList(bundles))
	}
	dirName := chunkDirFor(t, testPassphrase, salt)
	if bundles[0].Slot != mustSlotName(t, dirName, newHeader.BackupID) {
		t.Fatalf("свежая запись %q, ожидался слот нового пакета: %v", bundles[0].Slot, slotList(bundles))
	}
	if want := mustSlotName(t, dirName, legacyID); bundles[1].Slot != want {
		t.Fatalf("прежний пакет прочитан по %q, а его перенесённый слот %q", bundles[1].Slot, want)
	}
	if bundles[1].ChunkDir != dirName {
		t.Fatalf("перенесённый слот называет каталог %q, ожидалось %q", bundles[1].ChunkDir, dirName)
	}
	if _, err := RestoreBundleFromProvider(testPassphrase, mustLocal(t, root)); err != nil {
		t.Fatalf("алиас после второго пакета не ведёт к свежему: %v", err)
	}
	e2 := engineOf(t, mustLocal(t, root), testPassphrase, salt)
	legacy, err := e2.Restore(bundles[1].Header)
	if err != nil {
		t.Fatalf("прежний пакет после переноса в слот не раскрылся: %v", err)
	}
	if !bytes.Equal(legacy, payload) {
		t.Fatalf("прежний пакет раскрылся другими байтами: %d против %d", len(legacy), len(payload))
	}
	// Прежние чанки на месте: новый пакет их переиспользует, а не затирает.
	tree := treeOf(t, root)
	for _, d := range descriptors {
		if _, ok := tree[chunkObject(dirName, d.ChunkHash)]; !ok {
			t.Fatalf("чанк %s прежнего пакета пропал после второго пакета", d.ChunkHash)
		}
	}
}

// Граница, названная честно: чужой алиас в слот не переносится. Имя слота —
// каталог чанков, а его выводит только та passphrase, при которой salt записи
// даёт этот же ключ; подписывать чужую запись своим именем ротация не вправе,
// потому что потом не отличит её чанки от чужих. Такой header.json перезаписывается
// — как перезаписывался до Шага 15.
func TestLegacyAliasOfForeignBundleIsNotMigrated(t *testing.T) {
	root := t.TempDir()
	foreignSalt := mustSalt(t)
	foreignProvider := mustLocal(t, root)

	// Прежний пакет чужой passphrase: ровно один header.json, ни одного слота —
	// так, как его записал код до Шага 15.
	foreign := engineOf(t, foreignProvider, "не та passphrase", foreignSalt)
	foreignChunks, _, err := foreign.ProcessFileIntoChunks(randomPayload(t, 200*1024))
	if err != nil {
		t.Fatal(err)
	}
	foreignHeader := mustSeal(t, foreign, foreignChunks, uint64(200*1024))
	if err := WriteHeaderTo(foreignProvider, foreignHeader); err != nil {
		t.Fatalf("чужой заголовок прежним способом не выложить: %v", err)
	}
	if tree := treeOf(t, root); len(tree) != len(foreignChunks)+1 {
		t.Fatalf("прежняя раскладка чужого пакета дала %d файлов: %v", len(tree), namesOf(tree))
	}

	ownSalt := mustSalt(t)
	ownDir := chunkDirFor(t, testPassphrase, ownSalt)
	foreignDir := chunkDirFor(t, "не та passphrase", foreignSalt)
	if ownDir == foreignDir {
		t.Fatal("две passphrase дали один каталог чанков")
	}

	provider := mustLocal(t, root)
	e := engineOf(t, provider, testPassphrase, ownSalt)
	descriptors, _, err := e.ProcessFileIntoChunks(randomPayload(t, 240*1024))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.WriteBundle(mustSeal(t, e, descriptors, uint64(240*1024))); err != nil {
		t.Fatalf("свой пакет не выложился поверх чужого алиаса: %v", err)
	}

	bundles, err := ListBundles(provider)
	if err != nil {
		t.Fatalf("листинг после подмены чужого алиаса: %v", err)
	}
	if len(bundles) != 1 || bundles[0].BackupID != e.BackupID() {
		t.Fatalf("записей %d: чужой алиас не должен был пережить запись своего пакета: %v",
			len(bundles), slotList(bundles))
	}
	// Чужие чанки при этом целы: под чужой passphrase они в своём каталоге, а этот
	// код его имени не вычислит — значит, и удалить их нечем.
	tree := treeOf(t, root)
	if _, ok := tree[mustSlotName(t, ownDir, e.BackupID())]; !ok {
		t.Fatalf("свой слот в хранилище не лёг: %v", namesOf(tree))
	}
	for _, d := range foreignChunks {
		if _, ok := tree[chunkObject(foreignDir, d.ChunkHash)]; !ok {
			t.Fatalf("чанк %s чужого пакета затронут записью своего", d.ChunkHash)
		}
	}
	// Чужая запись после подмены алиаса не читается больше ничем: слота у неё нет,
	// и это названная граница переноса, а не случайная потеря.
	if _, err := RestoreBundleFromProvider("не та passphrase", mustLocal(t, root)); err == nil {
		t.Fatal("алиас указывает на свой пакет, и чужая passphrase его открыла")
	}
}

func TestPruneKeepsTheNewestEvenIfUnreadable(t *testing.T) {
	// Самый свежий пакет не удаляется никогда: он мог быть написан другой
	// passphrase, и тогда он чужой, но живой.
	root := t.TempDir()
	ownSalt := mustSalt(t)
	provider := mustLocal(t, root)
	own, _ := sealInto(t, provider, testPassphrase, ownSalt, randomPayload(t, 200*1024))
	spread(t, &own.CreatedAt)
	own2, _ := sealInto(t, provider, testPassphrase, ownSalt, randomPayload(t, 300*1024))
	spread(t, &own2.CreatedAt)

	foreignSalt := mustSalt(t)
	foreign, _ := sealInto(t, provider, "чужая passphrase", foreignSalt, randomPayload(t, 260*1024))

	before := treeOf(t, root)
	rep, err := PruneBundles(testPassphrase, provider, 1)
	if err != nil {
		t.Fatalf("ротация с чужим свежим пакетом названа ошибкой: %v", err)
	}
	if rep.Dropped == 0 {
		t.Fatalf("ротация не убрала ничего: %+v", rep)
	}
	tree := treeOf(t, root)
	foreignSlot := mustSlotName(t, chunkDirFor(t, "чужая passphrase", foreignSalt), foreign.BackupID)
	if _, ok := tree[foreignSlot]; !ok {
		t.Fatalf("чужой свежий пакет удалён чужой passphrase: %+v", rep)
	}
	// Он остался самой свежей записью, и алиас ведёт к нему.
	if alias, err := os.ReadFile(filepath.Join(root, HeaderName)); err != nil {
		t.Fatalf("алиас пропал: %v", err)
	} else if !bytes.Equal(alias, mustMarshal(t, foreign)) {
		t.Fatal("алиас не указывает на чужой свежий пакет")
	}
	if _, err := RestoreBundleFromProvider("чужая passphrase", mustLocal(t, root)); err != nil {
		t.Fatalf("чужой пакет после ротации не читается: %v", err)
	}
	if len(tree) >= len(before) {
		t.Fatalf("ротация не убрала ни одного файла: %d против %d", len(tree), len(before))
	}
}

// -----------------------------------------------------------------------------
// Вспомогательное второго ряда
// -----------------------------------------------------------------------------

func mustSeal(t *testing.T, e *SnapshotEngine, descriptors []ChunkDescriptor, totalPlain uint64) *Header {
	t.Helper()

	h, err := e.Seal(descriptors, totalPlain, sumEncrypted(descriptors))
	if err != nil {
		t.Fatalf("манифест не собрался: %v", err)
	}
	return h
}

// failOnceStore — хранилище, которое отказывает ровно на одном удалении чанка:
// имитация обрыва связи посреди GC. Отказ обязан прервать обход, а не превратиться
// в «успех с неполным отчётом».
type failOnceStore struct {
	BundleStore
	seen   int
	failOn int
}

func (s *failOnceStore) DeleteChunk(ctx context.Context, chunkDir, hashHex string) error {
	s.seen++
	if s.seen == s.failOn {
		return errors.New("хранилище оборвалось")
	}
	return s.BundleStore.DeleteChunk(ctx, chunkDir, hashHex)
}

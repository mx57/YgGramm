// File: core/pkg/backup/webdav_test.go
//
// Шаг 14: единственный подключённый удалённый провайдер сверяется не с самим
// собой, а с независимой реализацией того же протокола — эталонным
// WebDAV-сервером golang.org/x/net/webdav поверх MemFS в этом же процессе. Это
// тот же приём, что Шаг 4-a с X3DH и Шаг 11 с BIP-39: второй код, написанный не
// здесь, отвечает за то, что «клиент понимает протокол, а не собственное
// представление о нём».
//
// Что именно сверяется с эталонным сервером:
//   - PUT в несуществующую коллекцию — отказ (RFC 4918 §9.7.1 допускает 409 или
//     507; эталонный сервер отвечает 409 Conflict), поэтому клиент обязан сам
//     построить цепочку MKCOL, включая сегменты названного подкаталога: цель
//     вида «…/ygggram/2026/10» сервер за клиента не создаёт;
//   - MKCOL по уже существующему пути — 405, и это «уже есть», а не ошибка;
//   - PROPFIND Depth: infinity по каталогу ключа обязан вернуть размеры,
//     совпадающие с тем, что легло в MemFS: это единственный источник размеров
//     для дедупликации;
//   - 404 на GET — «чанка нет», а не «чанк пустой».
//
// Что к этому добавил Шаг 15 (ротация поверх сети, а не только по диску):
//   - два пакета с разными passphrase получают по слоту, и каталог слотов
//     строится теми же MKCOL, то есть имена слотов переносимы;
//   - листинг заголовков отвечает ровно двумя PROPFIND (алиас + Depth: infinity
//     по headers/) и называет алиас и все слоты, не обходя чанки;
//   - сироты убираются DELETE-ом по журналу запросов, повторный DELETE уже
//     убранного — успех, а не ошибка, иначе прерванный обход нельзя перечитать.
//
// Чего здесь нет намеренно: ни одного обращения к живому облаку. Nextcloud,
// Mail.ru и WebDAV-доступ Yandex.Disk требуют учётных данных, и чем именно они
// отвечают на редирект, на Depth: infinity и на отсутствующий файл, этот тест не
// утверждает.
package backup

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/webdav"
)

// davLog — журнал обращений к серверу. Без него «провайдер ничего не выкладывал»
// было бы утверждением на глаз.
type davLog struct {
	mu      sync.Mutex
	entries []string
}

func (l *davLog) add(r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()
	auth := ""
	if r.Header.Get("Authorization") != "" {
		auth = " auth"
	}
	l.entries = append(l.entries, r.Method+" "+r.URL.Path+auth)
}

func (l *davLog) count(prefix string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, e := range l.entries {
		if strings.HasPrefix(e, prefix) {
			n++
		}
	}
	return n
}

func (l *davLog) dump() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.entries, "; ")
}

// startReferenceDAV — эталонный сервер поверх MemFS. Prefix пустой: путь запроса
// становится путём в памяти сервера как есть, и «что там легло» проверяется
// через файловую систему сервера, а не через разбор ответа клиента.
func startReferenceDAV(t *testing.T) (*httptest.Server, webdav.FileSystem, *davLog) {
	t.Helper()

	fs := webdav.NewMemFS()
	handler := &webdav.Handler{FileSystem: fs, LockSystem: webdav.NewMemLS()}
	log := &davLog{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.add(r)
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts, fs, log
}

// davTarget — цель «webdav+http://127.0.0.1:порт/подкаталог»: httptest слушает
// loopback, поэтому незащищённая схема здесь законна (OpenProvider отвергает её
// на любом другом хосте).
func davTarget(ts *httptest.Server, sub string) string {
	return "webdav+http://" + strings.TrimPrefix(ts.URL, "http://") + sub
}

func davProvider(t *testing.T, ts *httptest.Server, sub string) CloudProvider {
	t.Helper()

	provider, err := OpenProvider(davTarget(ts, sub))
	if err != nil {
		t.Fatalf("webdav-цель %q отвергнута: %v", sub, err)
	}
	if provider.Name() != ProviderWebDAV {
		t.Fatalf("цель %q назвалась %q", sub, provider.Name())
	}
	return provider
}

// chunkDirFor — имя каталога чанков, выведенное из ключа заново, а не взятое у
// движка: иначе проверка «файл лёг куда надо» опиралась бы на тот же код, который
// её кладёт.
func chunkDirFor(t *testing.T, passphrase string, salt []byte) string {
	t.Helper()

	keys, err := deriveKeys(passphrase, salt, TestKDF)
	if err != nil {
		t.Fatalf("ключ не выведен: %v", err)
	}
	return chunkDirName(keys)
}

// objectPath — путь объекта пакета в MemFS: подкаталог + раскладка objectName.
func objectPath(sub, name string) string {
	return "/" + strings.Trim(sub, "/") + "/" + name
}

func chunkPath(sub, chunkDir, hash string) string {
	return objectPath(sub, fmt.Sprintf("%s/%s/%s/%s.bin", chunksRootName, chunkDir, hash[:2], hash))
}

// sealToProvider — пакет в хранилище тем же путём, что CreateBundle, но с дешёвым
// KDF: боевой профиль означал бы минуту ожидания на каждый случай.
func sealToProvider(t *testing.T, provider CloudProvider, passphrase string, salt []byte, payload []byte) (*Header, []ChunkDescriptor) {
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
	if err := WriteHeaderTo(provider, h); err != nil {
		t.Fatalf("заголовок не выложен: %v", err)
	}
	return h, descriptors
}

// restoreFromProvider — чтение пакета «с нуля»: новый клиент, новый движок, salt
// и параметры KDF берутся из открытого заголовка, как на другом устройстве.
func restoreFromProvider(t *testing.T, provider CloudProvider, passphrase string) ([]byte, error) {
	t.Helper()

	h, err := ReadHeaderFrom(provider)
	if err != nil {
		return nil, err
	}
	e, err := NewSnapshotEngineWithProvider(passphrase, provider, h.Salt, KDFParams{
		Time: h.Time, MemoryKiB: h.MemoryKiB, Threads: h.Threads,
	})
	if err != nil {
		return nil, err
	}
	return e.Restore(h)
}

func TestWebDAVBundleRoundTripAgainstReferenceServer(t *testing.T) {
	ts, fs, log := startReferenceDAV(t)
	const sub = "/ygggram/2026/10"
	salt := mustSalt(t)
	payload := randomPayload(t, 300*1024)
	dirName := chunkDirFor(t, testPassphrase, salt)

	// Коллекций нет вовсе: клиент обязан построить цепочку MKCOL сам, включая
	// сегменты названного подкаталога.
	provider := davProvider(t, ts, sub)
	header, descriptors := sealToProvider(t, provider, testPassphrase, salt, payload)
	if len(descriptors) < 2 {
		t.Fatalf("чанков %d: сквозной путь проверяется на нескольких", len(descriptors))
	}
	if header.TotalChunks != uint32(len(descriptors)) || header.TotalPlain != uint64(len(payload)) {
		t.Fatalf("заголовок облачного пакета врёт: %+v при %d чанках и %d байт",
			header, len(descriptors), len(payload))
	}
	if len(header.Salt) != len(salt) || !bytes.Equal(header.Salt, salt) {
		t.Fatalf("salt в заголовке не тот: %x", header.Salt)
	}

	// Цепочка: подкаталог, chunks, каталог ключа, шард.
	for _, want := range []string{
		"MKCOL /ygggram", "MKCOL /ygggram/2026", "MKCOL /ygggram/2026/10",
		"MKCOL /ygggram/2026/10/chunks",
		"MKCOL /ygggram/2026/10/chunks/" + dirName,
		"PUT /ygggram/2026/10/header.json",
	} {
		if log.count(want) == 0 {
			t.Fatalf("обращения %q в журнале нет: %s", want, log.dump())
		}
	}

	// Каждый чанк лежит под своим именем и ровно с тем размером шифртекста,
	// который назван в манифесте.
	for _, d := range descriptors {
		info, err := fs.Stat(context.Background(), chunkPath(sub, dirName, d.ChunkHash))
		if err != nil {
			t.Fatalf("чанк %s не лежит по пути, который обещает раскладка: %v", d.ChunkHash, err)
		}
		if info.Size() != int64(d.EncryptedSize) {
			t.Fatalf("чанк %s весит %d, манифест обещает %d", d.ChunkHash, info.Size(), d.EncryptedSize)
		}
	}

	// Список чанков, который сервер отдаёт сам, совпадает с тем, что легло: это
	// единственный источник размеров для следующего пакета.
	existing, err := provider.ExistingChunks(context.Background(), dirName)
	if err != nil {
		t.Fatalf("PROPFIND не удался: %v", err)
	}
	if len(existing) != len(descriptors) {
		t.Fatalf("PROPFIND назвал %d чанков, выложено %d", len(existing), len(descriptors))
	}
	for _, d := range descriptors {
		if existing[d.ChunkHash] != d.EncryptedSize {
			t.Fatalf("чанк %s: сервер отдаёт размер %d, манифест пишет %d",
				d.ChunkHash, existing[d.ChunkHash], d.EncryptedSize)
		}
	}

	got, err := restoreFromProvider(t, davProvider(t, ts, sub), testPassphrase)
	if err != nil {
		t.Fatalf("пакет из webdav не восстанавливается: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("восстановлено %d байт, ожидалось %d", len(got), len(payload))
	}

	// Чужая passphrase обязана быть отвергнута, а не «вернуть мусор»: ключ
	// выводится из неё, а не угадывается.
	if _, err := restoreFromProvider(t, davProvider(t, ts, sub), "не та фраза"); err == nil {
		t.Fatal("пакет открылся чужой passphrase")
	}
	// Другая фраза — другой каталог чанков: пустой PROPFIND, а не чужие файлы.
	if _, err := restoreFromProvider(t, davProvider(t, ts, "/другой-пакет"), "не та фраза"); err == nil {
		t.Fatal("пакет в несуществующем подкаталоге прочитался")
	}
}

func TestWebDAVProvidersAgreeOnTheSameBundle(t *testing.T) {
	// Тот же контент, те же ключи, два хранилища: merkle-корень и набор
	// контент-хэшей обязаны совпасть. Расхождение означало бы, что раскладка
	// зависит от способа доставки, — и «проверено на диске» не переносилось бы в
	// облако ни на байт.
	ts, _, _ := startReferenceDAV(t)
	salt := mustSalt(t)
	payload := randomPayload(t, 400*1024)

	disk := t.TempDir()
	local, err := NewLocalProvider(disk)
	if err != nil {
		t.Fatalf("локальный провайдер не собрался: %v", err)
	}
	localHeader, localChunks := sealToProvider(t, local, testPassphrase, salt, payload)

	remote := davProvider(t, ts, "/agree")
	remoteHeader, remoteChunks := sealToProvider(t, remote, testPassphrase, salt, payload)

	if !bytes.Equal(localHeader.RootMerkleHash, remoteHeader.RootMerkleHash) {
		t.Fatal("merkle-корень пакета зависит от хранилища")
	}
	if localHeader.TotalChunks != remoteHeader.TotalChunks ||
		localHeader.TotalPlain != remoteHeader.TotalPlain ||
		localHeader.TotalEncrypted != remoteHeader.TotalEncrypted {
		t.Fatalf("заголовки расходятся: диск %+v, облако %+v", localHeader, remoteHeader)
	}
	if len(localChunks) != len(remoteChunks) {
		t.Fatalf("чанков: диск %d, облако %d", len(localChunks), len(remoteChunks))
	}
	for i := range localChunks {
		if localChunks[i].ChunkHash != remoteChunks[i].ChunkHash ||
			localChunks[i].EncryptedSize != remoteChunks[i].EncryptedSize {
			t.Fatalf("чанк %d различается: %+v против %+v", i, localChunks[i], remoteChunks[i])
		}
	}

	fromRemote, err := restoreFromProvider(t, remote, testPassphrase)
	if err != nil {
		t.Fatalf("облачный пакет не читается: %v", err)
	}
	fromDisk, err := RestoreBundle(testPassphrase, disk)
	if err != nil {
		t.Fatalf("пакет на диске не читается: %v", err)
	}
	if !bytes.Equal(fromRemote, fromDisk) {
		t.Fatal("два хранилища отдали разные байты")
	}

	// Каталог пакета копируется как есть: имена объектов одинаковы, поэтому
	// «перенести бэкап» = «скопировать дерево», без пересборки формата.
	other := t.TempDir()
	copyTree(t, disk, other)
	cloned, err := RestoreBundle(testPassphrase, other)
	if err != nil {
		t.Fatalf("скопированный пакет не читается: %v", err)
	}
	if !bytes.Equal(cloned, fromRemote) {
		t.Fatal("копия пакета на диске разошлась с облаком")
	}
}

func TestWebDAVDeduplicationUsesServerListing(t *testing.T) {
	ts, _, log := startReferenceDAV(t)
	const sub = "/dedup"
	salt := mustSalt(t)
	payload := randomPayload(t, 300*1024)

	_, first := sealToProvider(t, davProvider(t, ts, sub), testPassphrase, salt, payload)
	puts := log.count("PUT ")
	if puts != len(first)+1 {
		t.Fatalf("PUT %d при %d чанках и одном заголовке: журнал %s", puts, len(first), log.dump())
	}

	// Новый клиент, тот же ключ: список приходит из PROPFIND, поэтому повторный
	// пакет не выкладывает ни одного чанка заново.
	before := log.count("PUT ")
	_, again := sealToProvider(t, davProvider(t, ts, sub), testPassphrase, salt, payload)
	if got := log.count("PUT ") - before; got != 1 {
		t.Fatalf("повторный пакет выложил %d объектов вместо одного заголовка: дедупликация не видит хранилище (%s)",
			got, log.dump())
	}
	if len(again) != len(first) {
		t.Fatalf("раскладка изменилась: %d чанков против %d", len(again), len(first))
	}

	// Другая passphrase — другой каталог чанков: файлы первого пакета не
	// переиспользуются, иначе восстановление читало бы чужой шифртекст.
	otherSalt := mustSalt(t)
	before = log.count("PUT ")
	_, foreign := sealToProvider(t, davProvider(t, ts, sub), "иная фраза", otherSalt, payload)
	if got := log.count("PUT ") - before; got != len(foreign)+1 {
		t.Fatalf("второй ключ выложил %d объектов, ожидалось %d: каталоги чанков не разделились",
			got, len(foreign)+1)
	}
}

func TestWebDAVAbsentAndTamperedChunksAreNamed(t *testing.T) {
	ts, fs, _ := startReferenceDAV(t)
	const sub = "/tamper"
	salt := mustSalt(t)
	payload := randomPayload(t, 300*1024)
	dirName := chunkDirFor(t, testPassphrase, salt)

	_, descriptors := sealToProvider(t, davProvider(t, ts, sub), testPassphrase, salt, payload)
	if len(descriptors) < 2 {
		t.Fatalf("чанков %d: проверке нужна потеря одного при живых остальных", len(descriptors))
	}

	// Потерянный чанк — «его нет в хранилище», а не «он пустой»: иначе вызывающий
	// получил бы ошибку AEAD вместо понятной.
	lost := chunkPath(sub, dirName, descriptors[0].ChunkHash)
	if err := fs.RemoveAll(context.Background(), lost); err != nil {
		t.Fatalf("чанк не убрать: %v", err)
	}
	_, err := restoreFromProvider(t, davProvider(t, ts, sub), testPassphrase)
	if err == nil {
		t.Fatal("пакет с пропавшим чанком восстановился")
	}
	if !strings.Contains(err.Error(), "лежит не в хранилище") {
		t.Fatalf("отсутствие чанка названо иначе: %v", err)
	}
	// Тот же случай на уровне провайдера: ErrChunkAbsent, а не пустые байты.
	if _, err := davProvider(t, ts, sub).DownloadChunk(context.Background(), dirName, descriptors[0].ChunkHash); !errors.Is(err, ErrChunkAbsent) {
		t.Fatalf("404 от webdav дал %v, а обязан дать ErrChunkAbsent", err)
	}

	// Подмена байта в чанке ловится аутентификацией, а не «молча другим
	// содержимым». Объект портится не изнутри MemFS, а перезаписью через PUT по
	// HTTP: это путь
	// «у противника есть право записи в хранилище», и подменённые байты приходят
	// клиенту по тому же каналу, по которому он их читает.
	header, fresh := sealToProvider(t, davProvider(t, ts, "/tamper2"), testPassphrase, salt, payload)
	if header.TotalChunks == 0 {
		t.Fatal("пустой пакет")
	}
	tamperFirstByte(t, ts, chunkPath("/tamper2", dirName, fresh[0].ChunkHash))

	if _, err := restoreFromProvider(t, davProvider(t, ts, "/tamper2"), testPassphrase); err == nil {
		t.Fatal("подменённый чанк раскрылся без ошибки")
	}
}

// tamperFirstByte — инверсия первого байта объекта через HTTP. Точечная правка
// «внутри сервера» недоступна: webdav.File — это http.File плюс io.Writer, без
// записи по смещению, а противнику с правом записи никто не мешает перезаписать
// объект целиком тем же размером.
func tamperFirstByte(t *testing.T, ts *httptest.Server, path string) {
	t.Helper()

	resp, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatalf("объект %s не скачать: %v", path, err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("объект %s не читается: %v", path, err)
	}
	if len(body) == 0 {
		t.Fatalf("объект %s пуст: портить нечего", path)
	}
	body[0] ^= 0xff

	req, err := http.NewRequest(http.MethodPut, ts.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("PUT не собрать: %v", err)
	}
	req.ContentLength = int64(len(body))
	put, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("PUT %s сорвался: %v", path, err)
	}
	_ = put.Body.Close()
	switch put.StatusCode {
	case http.StatusCreated, http.StatusOK, http.StatusNoContent, http.StatusMultiStatus:
	default:
		t.Fatalf("PUT %s ответил %s", path, put.Status)
	}
}

func TestWebDAVMKCOLFailureStopsTheUpload(t *testing.T) {
	// Сервер, который не даёт создавать коллекции (например, право на запись
	// только в чужие пути), обязан породить ошибку с именем коллекции, а не
	// «успешный пакет из несуществующих файлов».
	var mu sync.Mutex
	uploads := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			w.WriteHeader(http.StatusForbidden)
		case http.MethodPut:
			mu.Lock()
			uploads++
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	provider, err := OpenProvider("webdav+http://" + strings.TrimPrefix(ts.URL, "http://") + "/dst")
	if err != nil {
		t.Fatalf("цель отвергнута: %v", err)
	}
	err = provider.UploadChunk(context.Background(), testChunkDir, testHash, []byte("шифртекст"))
	if err == nil {
		t.Fatal("кладка при недоступном MKCOL прошла")
	}
	if !strings.Contains(err.Error(), "коллекция") {
		t.Fatalf("отказ не называет причину: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if uploads != 0 {
		t.Fatalf("PUT всё-таки ушёл %d раз: клиент полез в несуществующую коллекцию", uploads)
	}
}

func TestWebDAVRefusesRedirectsWithCredentialsOnTheLine(t *testing.T) {
	// Второй сервер считает, сколько запросов доехало с Authorization. Прежний
	// клиент следовал редиректу молча — и пароль уезжал на хост из Location.
	var mu sync.Mutex
	arrived := 0
	trap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if r.Header.Get("Authorization") != "" {
			arrived++
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(trap.Close)

	targeted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, trap.URL+"/stolen", http.StatusFound)
	}))
	t.Cleanup(targeted.Close)

	provider, err := OpenProvider("webdav+http://" + strings.TrimPrefix(targeted.URL, "http://") + "/dst")
	if err != nil {
		t.Fatalf("цель отвергнута: %v", err)
	}
	err = provider.UploadChunk(context.Background(), testChunkDir, testHash, []byte("данные"))
	if err == nil {
		t.Fatal("клиент последовал редиректу")
	}
	if !strings.Contains(err.Error(), "редирект") {
		t.Fatalf("отказ не называет редирект: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if arrived != 0 {
		t.Fatalf("на чужой хост пришло %d запросов с паролем", arrived)
	}
}

func TestWebDAVTimeoutIsNamed(t *testing.T) {
	// Потолок времени обязана быть названа: «облако не ответило» и «облако
	// ответило ошибкой» для вызывающего — разные ситуации, а зависнуть под e.mu
	// значит остановить приём почты на всём ядре.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(ts.Close)

	provider, err := NewWebDAVProvider(WebDAVConfig{
		BaseURL:  ts.URL + "/dst",
		Username: "user",
		Password: "pass",
		Timeout:  50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("провайдер не собрался: %v", err)
	}
	start := time.Now()
	err = provider.UploadChunk(context.Background(), testChunkDir, testHash, []byte("данные"))
	if err == nil {
		t.Fatal("запрос без ответа принят успехом")
	}
	if !strings.Contains(err.Error(), "не ответил за") {
		t.Fatalf("таймаут назван иначе: %v", err)
	}
	if spent := time.Since(start); spent > time.Second {
		t.Fatalf("потолок %v не сработал: запрос шёл %v", provider.timeout, spent)
	}
}

func TestWebDAVClientConfigIsCheckedBeforeFirstRequest(t *testing.T) {
	for _, bad := range []struct{ url, why string }{
		{"", "пустой BaseURL"},
		{"https://user:pass@host/dst", "userinfo в адресе: пароль уехал бы в лог"},
		{"ftp://host/dst", "не http(s)"},
		{"https:///dst", "нет хоста"},
		{"https://host/dst?fields=x", "query"},
		{"https://host/dst#frag", "fragment"},
	} {
		if _, err := NewWebDAVProvider(WebDAVConfig{BaseURL: bad.url}); err == nil {
			t.Errorf("%s принят: %q", bad.why, bad.url)
		}
	}

	// Завершающий слэш дописывается, а не склеивается «на глаз»: без него
	// первый же запрос потерял бы последний сегмент подкаталога.
	provider, err := NewWebDAVProvider(WebDAVConfig{BaseURL: "https://host/dst"})
	if err != nil {
		t.Fatalf("годный конфиг отвергнут: %v", err)
	}
	if got := provider.base.String(); got != "https://host/dst/" {
		t.Fatalf("BaseURL нормализован криво: %q", got)
	}
	if provider.timeout != defaultProviderTimeout {
		t.Fatalf("по умолчанию нет потолка времени: %v", provider.timeout)
	}
	// Имя объекта проверяется до любого обращения к сети.
	if _, err := provider.urlFor("../etc/passwd"); err == nil {
		t.Fatal("имя с «..» дошло до сборки адреса")
	}
	if _, err := provider.urlFor("chunks//x.bin"); err == nil {
		t.Fatal("пустой сегмент дошло до сборки адреса")
	}
}

func TestMultistatusParsingMatchesWhatServersActuallySend(t *testing.T) {
	// encoding/xml сопоставляет по локальным именам, поэтому разбор не зависит от
	// префикса пространства имён. Ответы живых облаков отличаются именно
	// префиксами и формой href — здесь обе формы.
	longHash := testHash
	otherHash := strings.Repeat("f4", 32)
	cases := []struct {
		name string
		body string
		want map[string]uint32
		err  bool
	}{
		{
			name: "с префиксом d и абсолютными href",
			body: `<?xml version="1.0"?><d:multistatus xmlns:d="DAV:">` +
				`<d:response><d:href>https://host/dst/chunks/abcd1234/0a/` + longHash + `.bin</d:href>` +
				`<d:propstat><d:prop><d:getcontentlength>4096</d:getcontentlength><d:resourcetype/></d:prop></d:propstat></d:response>` +
				`</d:multistatus>`,
			want: map[string]uint32{longHash: 4096},
		},
		{
			name: "без префикса, href путём",
			body: `<?xml version="1.0"?><multistatus xmlns="DAV:">` +
				`<response><href>/dst/chunks/abcd1234/f4/` + otherHash + `.bin</href>` +
				`<propstat><prop><getcontentlength>7</getcontentlength></prop></propstat></response>` +
				`</multistatus>`,
			want: map[string]uint32{otherHash: 7},
		},
		{
			name: "каталог и чанк без длины",
			body: `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:">` +
				`<D:response><D:href>/dst/chunks/abcd1234/</D:href><D:propstat><D:prop>` +
				`<D:getcontentlength>0</D:getcontentlength><D:resourcetype><D:collection/></D:resourcetype></D:prop></D:propstat></D:response>` +
				`<D:response><D:href>/dst/chunks/abcd1234/0a/` + longHash + `.bin</D:href><D:propstat><D:prop>` +
				`<D:resourcetype/></D:prop></D:propstat></D:response>` +
				`</D:multistatus>`,
			want: map[string]uint32{},
		},
		{
			name: "несколько propstat, длина во втором",
			body: `<?xml version="1.0"?><multistatus xmlns="DAV:">` +
				`<response><href>/x/` + otherHash + `.bin</href>` +
				`<propstat><prop><resourcetype/></prop></propstat>` +
				`<propstat><prop><getcontentlength>123</getcontentlength></prop></propstat></response>` +
				`</multistatus>`,
			want: map[string]uint32{otherHash: 123},
		},
		{name: "пустое тело", body: ``, err: true},
		{name: "не xml", body: `this is not xml`, err: true},
		{name: "длина не число", body: `<multistatus xmlns="DAV:"><response><href>/a/` + longHash +
			`.bin</href><propstat><prop><getcontentlength>abc</getcontentlength></prop></propstat></response></multistatus>`, err: true},
		{name: "длина за потолком", body: `<multistatus xmlns="DAV:"><response><href>/a/` + longHash +
			`.bin</href><propstat><prop><getcontentlength>` + fmt.Sprint(maxChunkBlob+1) +
			`</getcontentlength></prop></propstat></response></multistatus>`, err: true},
	}
	for _, c := range cases {
		got, err := parseMultistatus([]byte(c.body))
		if c.err {
			if err == nil {
				t.Errorf("%s: разбор не отвергнут", c.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: разбор не удался: %v", c.name, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: список %v, ожидалось %v", c.name, got, c.want)
			continue
		}
		for hash, size := range c.want {
			if got[hash] != size {
				t.Errorf("%s: чанк %s получил размер %d, ожидалось %d", c.name, hash, got[hash], size)
			}
		}
	}
}

func TestExistingChunksDistinguishesFirstBundleFromBadServer(t *testing.T) {
	// 404 на PROPFIND — единственный случай, имеющий право дать пустое множество:
	// каталога ключа ещё нет. Всё остальное — ошибка, иначе движок перевыложил бы
	// пакет поверх непонятного ответа или потерял чанки.
	codes := map[string]func(http.ResponseWriter, *http.Request){
		"501 без PROPFIND": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) },
		"207 пустое тело":  func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(207) },
		"200 не xml":       func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) },
	}
	for name, serve := range codes {
		ts := httptest.NewServer(http.HandlerFunc(serve))
		provider, err := NewWebDAVProvider(WebDAVConfig{BaseURL: ts.URL + "/dst"})
		if err != nil {
			t.Fatalf("конфиг отвергнут: %v", err)
		}
		existing, err := provider.ExistingChunks(context.Background(), testChunkDir)
		ts.Close()
		if err == nil {
			t.Errorf("%s: ответ принят как %v, а это не список", name, existing)
		}
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	provider, err := NewWebDAVProvider(WebDAVConfig{BaseURL: ts.URL + "/dst"})
	if err != nil {
		t.Fatalf("конфиг отвергнут: %v", err)
	}
	existing, err := provider.ExistingChunks(context.Background(), testChunkDir)
	if err != nil {
		t.Fatalf("отсутствие каталога стало ошибкой: %v", err)
	}
	if len(existing) != 0 {
		t.Fatalf("несуществующий каталог перечислился: %v", existing)
	}
}

func TestWebDAVReplyAndUploadCaps(t *testing.T) {
	// Ответ больше потолка — ошибка, а не «прочитано до конца»: на нарисованном
	// Content-Length клиент не обязан выделять память. Потолок работает и в
	// обратную сторону: шифртекст за пределом не выкладывается вовсе.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		size := maxChunkBlob + 1024
		if strings.HasSuffix(r.URL.Path, HeaderName) {
			size = maxHeaderBlob + 1024
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte{0x7a}, size))
	}))
	t.Cleanup(ts.Close)

	provider, err := NewWebDAVProvider(WebDAVConfig{BaseURL: ts.URL + "/dst"})
	if err != nil {
		t.Fatalf("конфиг отвергнут: %v", err)
	}
	if _, err := provider.DownloadHeader(context.Background()); err == nil {
		t.Fatal("заголовок за потолком принят")
	}
	if _, err := provider.DownloadChunk(context.Background(), testChunkDir, testHash); err == nil {
		t.Fatal("чанк за потолком принят")
	}
	if err := provider.UploadChunk(context.Background(), testChunkDir, testHash,
		bytes.Repeat([]byte{1}, maxChunkBlob+1)); err == nil {
		t.Fatal("шифртекст за потолком выложен")
	}
	if err := provider.UploadHeader(context.Background(), bytes.Repeat([]byte{1}, maxHeaderBlob+1)); err == nil {
		t.Fatal("заголовок за потолком выложен")
	}
}

func TestWebDAVCredentialsReachTheServerAsBasicAuth(t *testing.T) {
	// Пароль обязан дойти до сервера (иначе клиент бесполезен) и не появиться в
	// адресе и в тексте ошибки (иначе он утекает в логи).
	var mu sync.Mutex
	got := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if w2, ok := r.Header["Authorization"]; ok && len(w2) > 0 {
			got = w2[0]
		}
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(ts.Close)

	// Пароль в userinfo обязан быть percent-encoded: net/url не принимает «сырые»
	// нелатинные символы в учётных данных. Здесь пара проверяется дважды: что
	// декодированный пароль доходит до сервера ровно своим байтом и что ни сырой,
	// ни закодированный секрет не остаётся в адресе клиента.
	const secret = "S3cr3t pw"
	provider, err := OpenProvider("webdav+http://user:S3cr3t%20pw@" +
		strings.TrimPrefix(ts.URL, "http://") + "/dst")
	if err != nil {
		t.Fatalf("цель отвергнута: %v", err)
	}
	if err := provider.UploadChunk(context.Background(), testChunkDir, testHash, []byte("данные")); err != nil {
		t.Fatalf("кладка не удалась: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.HasPrefix(got, "Basic ") {
		t.Fatalf("учётные данные не дошли: %q", got)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(got, "Basic "))
	if err != nil {
		t.Fatalf("Basic-заголовок не декодируется: %v", err)
	}
	if want := "user:" + secret; string(raw) != want {
		t.Fatalf("на линию ушла пара %q, ожидалась %q", raw, want)
	}
	if addr := provider.(*WebDAVProvider).base.String(); strings.Contains(addr, "S3cr3t") {
		t.Fatalf("пароль остался в адресе: %q", addr)
	}

	// Сырой (не закодированный) нелатинский пароль цель отвергает — и отказ не
	// должен повторять ни его, ни percent-форму.
	_, err = OpenProvider("webdav+http://user:пароль-для-basic@" +
		strings.TrimPrefix(ts.URL, "http://") + "/dst")
	if err == nil {
		t.Fatal("цель с недопустимым userinfo принята")
	}
	if strings.Contains(err.Error(), "пароль-для-basic") {
		t.Fatalf("отказ повторил пароль: %v", err)
	}
}

func TestWebDAVTwoPassphrasesGetTheirOwnSlots(t *testing.T) {
	// Шаг 15 поверх сети: два пакета от разных passphrase в одном подкаталоге больше
	// не затирают друг друга. У каждого свой слот headers/<ключ>/<id>.json, и
	// прежняя точка читается по своему слоту, хотя алиас указывает на свежую.
	// Открытый заголовок больше не «один на хранилище», и это проверяется не по
	// флагу в памяти, а по MemFS эталонного сервера.
	ts, fs, log := startReferenceDAV(t)
	const sub = "/multi"
	payload := randomPayload(t, 200*1024)
	saltA, saltB := mustSalt(t), mustSalt(t)
	dirA := chunkDirFor(t, "фраза первая", saltA)
	dirB := chunkDirFor(t, "фраза вторая", saltB)
	if dirA == dirB {
		t.Fatal("ключевые каталоги совпали")
	}

	hA, chunksA := sealInto(t, davProvider(t, ts, sub), "фраза первая", saltA, payload)
	spread(t, &hA.CreatedAt)
	// Каталог второго ключа пуст: имена чанков разведены по ключу, и дедупликация
	// второго пакета не возьмёт шифртекст первого.
	existing, err := davProvider(t, ts, sub).ExistingChunks(context.Background(), dirB)
	if err != nil {
		t.Fatalf("PROPFIND по чужому каталогу не удался: %v", err)
	}
	if len(existing) != 0 {
		t.Fatalf("каталог второго ключа увидел %d чанков первого: имена не разделились", len(existing))
	}

	hB, chunksB := sealInto(t, davProvider(t, ts, sub), "фраза вторая", saltB, payload)
	if log.count("PUT "+sub+"/chunks/"+dirB+"/") == 0 {
		t.Fatalf("чанков второго ключа в хранилище нет: %s", log.dump())
	}

	// Цепочка MKCOL строится и под дерево слотов: «…/headers/<ключ>» сервер за
	// клиента не создаёт, а PUT в несуществующую коллекцию — 409.
	for _, want := range []string{
		"MKCOL " + sub + "/headers",
		"MKCOL " + sub + "/headers/" + dirA,
		"MKCOL " + sub + "/headers/" + dirB,
	} {
		if log.count(want) == 0 {
			t.Fatalf("обращения %q в журнале нет: %s", want, log.dump())
		}
	}

	// Оба слота лежат в MemFS под своими именами, и листинг видит все три объекта.
	ctx := context.Background()
	slotA, slotB := mustSlotName(t, dirA, hA.BackupID), mustSlotName(t, dirB, hB.BackupID)
	for _, pair := range [][2]string{{slotA, dirA}, {slotB, dirB}} {
		if _, err := fs.Stat(ctx, objectPath(sub, pair[0])); err != nil {
			t.Fatalf("слот %s не лежит в хранилище: %v", pair[0], err)
		}
	}
	names, err := mustStore(t, davProvider(t, ts, sub)).ListHeaderNames(ctx)
	if err != nil {
		t.Fatalf("ListHeaderNames поверх сети не удался: %v", err)
	}
	sort.Strings(names)
	if want := sortedNames(HeaderName, slotA, slotB); !equalStrings(names, want) {
		t.Fatalf("листинг заголовков дал %v, ожидалось %v", names, want)
	}

	// Алиас и NewestBundle ведут на свежий пакет: чужая passphrase обязана
	// споткнуться именно на нём, а не «вернуть мусор».
	newest, err := NewestBundle(davProvider(t, ts, sub))
	if err != nil || newest.BackupID != hB.BackupID {
		t.Fatalf("свежим назван %d (%v), а записан последним %d", newest.BackupID, err, hB.BackupID)
	}
	if _, err := RestoreBundleFromProvider("фраза первая", davProvider(t, ts, sub)); err == nil {
		t.Fatal("алиас указывает на второй пакет, и первая passphrase его открыла")
	}
	gotB, err := RestoreBundleFromProvider("фраза вторая", davProvider(t, ts, sub))
	if err != nil {
		t.Fatalf("второй пакет не читается: %v", err)
	}
	if !bytes.Equal(gotB, payload) {
		t.Fatal("второй пакет раскрылся чужим содержимым")
	}

	// Прежняя точка восстановима по своему слоту: ровно то, что до Шага 15 было
	// невозможно, потому что header.json перезаписывался.
	eA, err := NewSnapshotEngineWithProvider("фраза первая", davProvider(t, ts, sub), saltA, TestKDF)
	if err != nil {
		t.Fatalf("движок прежнего пакета не собрался: %v", err)
	}
	gotA, err := eA.Restore(hA)
	if err != nil {
		t.Fatalf("прежний пакет по своему слоту не раскрылся: %v", err)
	}
	if !bytes.Equal(gotA, payload) {
		t.Fatal("прежний пакет раскрылся другими байтами")
	}
	for _, d := range append(append([]ChunkDescriptor{}, chunksA...), chunksB...) {
		dir := dirA
		if !chunkSet(chunksA)[d.ChunkHash] {
			dir = dirB
		}
		if _, err := fs.Stat(ctx, chunkPath(sub, dir, d.ChunkHash)); err != nil {
			t.Fatalf("чанк %s пропал из %s: %v", d.ChunkHash, dir, err)
		}
	}
}

func TestWebDAVSecondBundleReusesChunksAndPrunesOverNetwork(t *testing.T) {
	// Обещания Шага 15, измеренные против эталонного сервера, а не против диска:
	// второй пакет не выкладывает ни одного чанка, листинг заголовков отвечает двумя
	// PROPFIND, а ротация убирает сироты DELETE-ом и идемпотентна к повтору.
	ts, fs, log := startReferenceDAV(t)
	const sub = "/rotation"
	salt := mustSalt(t)
	payload := randomPayload(t, 300*1024)
	dirName := chunkDirFor(t, testPassphrase, salt)

	first, d1 := sealInto(t, davProvider(t, ts, sub), testPassphrase, salt, payload)
	spread(t, &first.CreatedAt)
	chunkPUTs := log.count("PUT " + sub + "/chunks/")
	if chunkPUTs == 0 {
		t.Fatalf("первый пакет не выложил чанков: %s", log.dump())
	}

	// Второй пакет идёт полным путём вызывающего: ключ берётся у самого свежего,
	// поэтому каталог тот же и список провайдера уже знает эти хэши.
	second, err := NewSnapshotEngineToProvider(testPassphrase, davProvider(t, ts, sub))
	if err != nil {
		t.Fatalf("второй пакет не завёлся: %v", err)
	}
	if !second.ReusedKey() {
		t.Fatal("второй пакет в непустом хранилище объявлен первым")
	}
	if second.ChunkDir() != dirName {
		t.Fatalf("второй пакет собрался в каталог %s, а первый в %s: дедуп между пакетами невозможен",
			second.ChunkDir(), dirName)
	}
	d2, _, err := second.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatalf("раскладка второго пакета не удалась: %v", err)
	}
	if uploaded, already := second.DedupStats(); uploaded != 0 || already != uint32(len(d2)) {
		t.Fatalf("второй пакет выложил %d чанков из %d при том же контенте", uploaded, len(d2))
	}
	if got := log.count("PUT "+sub+"/chunks/") - chunkPUTs; got != 0 {
		t.Fatalf("второй пакет послал %d PUT чанков поверх списка провайдера: %s", got, log.dump())
	}
	h2 := mustSeal(t, second, d2, uint64(len(payload)))
	if err := second.WriteBundle(h2); err != nil {
		t.Fatalf("второй пакет не выложился: %v", err)
	}
	slot2 := mustSlotName(t, dirName, h2.BackupID)
	if log.count("PUT "+sub+"/"+slot2) == 0 {
		t.Fatalf("слот второго пакета не выложен PUT-ом: %s", log.dump())
	}

	// Третий пакет — другое содержимое, иначе его чанки удержали бы первый от
	// удаления и удалять по сети было бы нечего.
	spread(t, &h2.CreatedAt)
	v3 := payloadSeeded(t, 400*1024, 4211)
	third, d3 := sealInto(t, davProvider(t, ts, sub), testPassphrase, salt, v3)

	provider := davProvider(t, ts, sub)
	names, err := mustStore(t, provider).ListHeaderNames(context.Background())
	if err != nil {
		t.Fatalf("ListHeaderNames не удался: %v", err)
	}
	sort.Strings(names)
	if want := sortedNames(HeaderName, mustSlotName(t, dirName, first.BackupID), slot2,
		mustSlotName(t, dirName, third.BackupID)); !equalStrings(names, want) {
		t.Fatalf("листинг заголовков дал %v, ожидалось %v", names, want)
	}

	// keep=1: остаются только чанки третьего пакета. Первый и второй описывают одно
	// и то же содержимое, поэтому их общий набор уходит один раз — это и есть
	// однократный счёт общего чанка, только поверх сети.
	kept := chunkSet(d3)
	orphans := []string{}
	for hash := range chunkSet(d1, d2) {
		if !kept[hash] {
			orphans = append(orphans, hash)
		}
	}
	sort.Strings(orphans)
	if len(orphans) == 0 {
		t.Fatal("общий набор чанков первого и второго пакета целиком нужен третьему: удалять по сети нечего")
	}

	ctx := context.Background()
	rep, err := PruneBundles(testPassphrase, provider, 1)
	if err != nil {
		t.Fatalf("ротация поверх webdav не удалась: %v", err)
	}
	if rep.Dropped != 2 || rep.ChunksDeleted != len(orphans) {
		t.Fatalf("ротация отчиталась %+v, ожидалось два пакета и %d чанков", rep, len(orphans))
	}
	if log.count("DELETE "+sub+"/chunks/"+dirName+"/") < len(orphans) {
		t.Fatalf("удалённых чанков по журналу меньше, чем названо отчётом: %s", log.dump())
	}
	for _, hash := range orphans {
		if _, err := fs.Stat(ctx, chunkPath(sub, dirName, hash)); err == nil {
			t.Fatalf("чанк %s пережил DELETE, хотя на него никто не ссылается", hash)
		}
	}
	for hash := range kept {
		if _, err := fs.Stat(ctx, chunkPath(sub, dirName, hash)); err != nil {
			t.Fatalf("чанк %s оставленного пакета удалён: %v", hash, err)
		}
	}
	for _, id := range []uint64{first.BackupID, h2.BackupID} {
		if _, err := fs.Stat(ctx, objectPath(sub, mustSlotName(t, dirName, id))); err == nil {
			t.Fatalf("слот пакета %d остался в хранилище после ротации", id)
		}
	}

	// Идемпотентность поверх сети: повторное удаление уже убранного — успех, иначе
	// прерванный обход нельзя перечитать с начала.
	store := mustStore(t, davProvider(t, ts, sub))
	if err := store.DeleteChunk(ctx, dirName, orphans[0]); err != nil {
		t.Fatalf("повторное удаление чанка сочтено ошибкой: %v", err)
	}
	if err := store.DeleteHeaderNamed(ctx, mustSlotName(t, dirName, first.BackupID)); err != nil {
		t.Fatalf("повторное удаление слота сочтено ошибкой: %v", err)
	}

	// Алиас ведёт на самого свежего, и пакет читается «с нуля» новым клиентом.
	if alias, err := fs.Stat(ctx, objectPath(sub, HeaderName)); err != nil || alias.Size() == 0 {
		t.Fatalf("алиас после ротации пропал: %v", err)
	}
	got, err := RestoreBundleFromProvider(testPassphrase, davProvider(t, ts, sub))
	if err != nil {
		t.Fatalf("свежий пакет после ротации не восстанавливается: %v", err)
	}
	if !bytes.Equal(got, v3) {
		t.Fatal("свежий пакет раскрылся другими байтами")
	}
}

// equalStrings — сравнение отсортированных списков имён: отчёт обязан называть
// ровно те объекты, что легли, без одного лишнего и без одного пропущенного.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// copyTree — файловая копия каталога пакета: проверка, что имена объектов
// переносимы между хранилищами без пересборки.
func copyTree(t *testing.T, from, to string) {
	t.Helper()

	err := filepath.WalkDir(from, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0600)
	})
	if err != nil {
		t.Fatalf("копия %s -> %s не собралась: %v", from, to, err)
	}
}

// File: core/pkg/backup/providers_test.go
//
// Провайдер — граница, за которой пакет перестаёт быть байтами в памяти процесса.
// Здесь проверяются ровно те обещания, которые CloudProvider даёт движку, и ровно
// те, что Шаг 14 снял с трёх прежних «облачных» клиентов.
//
// Почему без этого нельзя:
//   - дедупликация смотрит в хранилище, поэтому ExistingChunks обязан отличать
//     «чанков нет» от «хранилище не ответило». Прежний WebDAV рисовал пустое
//     множество на месте несделанного PROPFIND, и это означало бы молча
//     перевыложить всё либо потерять чанки;
//   - размеры шифртекстов в манифесте берутся оттуда же, поэтому нулевой или
//     выдуманный размер — враньё в пакете, а не в коде;
//   - имя объекта уходит и в путь на диске, и в URL облака, и оно же — часть
//     AAD. Слэш, «..» или верхний регистр в hex превратили бы один чанк в два
//     файла с разным смыслом;
//   - атомарная кладка: оборванный черновик не имеет права сойти за готовый чанк,
//     иначе восстановление падает на AEAD у файла, который «существует».
package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testChunkDir = "abcd1234"
	testHash     = "0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a"
)

func TestObjectNameIsTheSingleLayout(t *testing.T) {
	name, err := objectName(testChunkDir, testHash)
	if err != nil {
		t.Fatalf("годное имя не собралось: %v", err)
	}
	if want := chunksRootName + "/" + testChunkDir + "/0a/" + testHash + ".bin"; name != want {
		t.Fatalf("раскладка %q, ожидалось %q", name, want)
	}

	for _, bad := range []struct{ chunkDir, hash string }{
		{"", testHash},
		{"abcdefg", testHash},
		{"..", testHash},
		{"ABCD1234", testHash},
		{"abcd123e", testHash + "/"},
		{testChunkDir, ""},
		{testChunkDir, "nothexnothexnothexnothexnothexnothexnothexnothex"},
		{testChunkDir, strings.ToUpper(testHash)},
		{testChunkDir, testHash[:62] + "g9"},
	} {
		if _, err := objectName(bad.chunkDir, bad.hash); err == nil {
			t.Errorf("имя %q/%q принято: его записали бы куда попало", bad.chunkDir, bad.hash)
		}
	}

	// Локальный провайдер обязан держать файл внутри корня пакета: то, что имя не
	// прошло objectName, сюда вообще не доходит, а пройденное не имеет права
	// выбраться за root.
	root := t.TempDir()
	provider, err := NewLocalProvider(root)
	if err != nil {
		t.Fatalf("провайдер не собрался: %v", err)
	}
	if err := provider.UploadChunk(context.Background(), testChunkDir, testHash, []byte("x")); err != nil {
		t.Fatalf("чанк не выложен: %v", err)
	}
	written := filepath.Join(root, chunksRootName, testChunkDir, testHash[:2], testHash+".bin")
	if _, err := os.Stat(written); err != nil {
		t.Fatalf("чанк лежит не там, где обещает раскладка: %v", err)
	}
	if rel, err := filepath.Rel(root, written); err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("путь чанка %q вышел за корень %q", written, root)
	}
}

func TestLocalProviderRequiresAbsolutePath(t *testing.T) {
	for _, bad := range []string{"", "bundle", "bundles/one", "..", filepath.Join("relative", "dir")} {
		if _, err := NewLocalProvider(bad); err == nil {
			t.Fatalf("цель %q принята как каталог пакета: у FFI-сборки текущая директория не определена", bad)
		}
	}
	if _, err := NewLocalProvider(t.TempDir()); err != nil {
		t.Fatalf("абсолютный путь отвергнут: %v", err)
	}
}

func TestLocalProviderDistinguishesEmptyFromUnreachable(t *testing.T) {
	root := t.TempDir()
	provider, err := NewLocalProvider(filepath.Join(root, "bundle"))
	if err != nil {
		t.Fatalf("провайдер не собрался: %v", err)
	}
	ctx := context.Background()

	// Каталога чанков нет — это первый пакет, а не ошибка.
	existing, err := provider.ExistingChunks(ctx, testChunkDir)
	if err != nil {
		t.Fatalf("отсутствие каталога чанков стало ошибкой: %v", err)
	}
	if len(existing) != 0 {
		t.Fatalf("несуществующий каталог перечислился как %d чанков", len(existing))
	}
	if _, err := provider.DownloadChunk(ctx, testChunkDir, testHash); !errors.Is(err, ErrChunkAbsent) {
		t.Fatalf("отсутствующий чанк дал %v, а обязан дать ErrChunkAbsent", err)
	}
	// Заголовок — не чанк: его отсутствие называет файл, иначе вызывающий решит,
	// что пакет просто неполный.
	if _, err := provider.DownloadHeader(ctx); err == nil || errors.Is(err, ErrChunkAbsent) {
		t.Fatalf("отсутствующий заголовок дал %v", err)
	}

	// Каталог, занятый файлом, — ошибка, а не пустое множество: пустое значит
	// «чанков нет, можно писать», и движок выложил бы поверх чужого файла.
	blocked, err := NewLocalProvider(root)
	if err != nil {
		t.Fatalf("провайдер не собрался: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, chunksRootName), 0700); err != nil {
		t.Fatalf("каталог чанков не создать: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, chunksRootName, testChunkDir), []byte("не каталог"), 0600); err != nil {
		t.Fatalf("файл на месте каталога ключа не записать: %v", err)
	}
	if _, err := blocked.ExistingChunks(ctx, testChunkDir); err == nil {
		t.Fatal("каталог чанков, занятый файлом, перечислился как успех")
	}
}

func TestLocalListingIgnoresScratchAndRejectsOversize(t *testing.T) {
	root := t.TempDir()
	provider, err := NewLocalProvider(root)
	if err != nil {
		t.Fatalf("провайдер не собрался: %v", err)
	}
	ctx := context.Background()
	blob := []byte("шифртекст")
	if err := provider.UploadChunk(ctx, testChunkDir, testHash, blob); err != nil {
		t.Fatalf("чанк не выложен: %v", err)
	}
	shard := filepath.Join(root, chunksRootName, testChunkDir, testHash[:2])

	// Черновик атомарной кладки — мусор обрыва, не чанк.
	if err := os.WriteFile(filepath.Join(shard, testHash+".bin.tmp"), []byte("недокладка"), 0600); err != nil {
		t.Fatalf("черновик не записать: %v", err)
	}
	// Чужое имя в шарде — не чанк: оно не проходит validChunkHash.
	if err := os.WriteFile(filepath.Join(shard, "readme.txt"), []byte("x"), 0600); err != nil {
		t.Fatalf("чужой файл не записать: %v", err)
	}
	existing, err := provider.ExistingChunks(ctx, testChunkDir)
	if err != nil {
		t.Fatalf("листинг не удался: %v", err)
	}
	if len(existing) != 1 || existing[testHash] != uint32(len(blob)) {
		t.Fatalf("листинг дал %v: черновик или чужой файл сошёл за чанк, либо размер не настоящий", existing)
	}

	// Файл за потолок шифртекста — ошибка, а не «чанк неизвестного размера»:
	// вызывающий прочитал бы его в память.
	if err := os.Truncate(filepath.Join(shard, testHash+".bin"), int64(maxChunkBlob)+1); err != nil {
		t.Fatalf("файл не удлинить: %v", err)
	}
	if _, err := provider.ExistingChunks(ctx, testChunkDir); err == nil {
		t.Fatal("чанк за потолком принят списком")
	}
	if _, err := provider.DownloadChunk(ctx, testChunkDir, testHash); err == nil || errors.Is(err, ErrChunkAbsent) {
		t.Fatalf("чтение чанка за потолком дало %v", err)
	}
}

func TestLocalWriteIsAtomicAndRepeatable(t *testing.T) {
	root := t.TempDir()
	provider, err := NewLocalProvider(root)
	if err != nil {
		t.Fatalf("провайдер не собрался: %v", err)
	}
	ctx := context.Background()
	if err := provider.UploadChunk(ctx, testChunkDir, testHash, []byte("тело")); err != nil {
		t.Fatalf("чанк не выложен: %v", err)
	}
	if err := provider.UploadHeader(ctx, []byte(`{"magic":"x"}`)); err != nil {
		t.Fatalf("заголовок не выложен: %v", err)
	}

	tmp := 0
	err = filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".tmp") {
			tmp++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("обход не удался: %v", err)
	}
	if tmp != 0 {
		t.Fatalf("после успешной кладки осталось %d черновиков", tmp)
	}
	// Перекладка тех же байт — успех: дедупликация оптимизация, а не уникальный
	// путь, и ошибка здесь означала бы «пакет не пересобрать».
	if err := provider.UploadChunk(ctx, testChunkDir, testHash, []byte("тело")); err != nil {
		t.Fatalf("перекладка чанка стала ошибкой: %v", err)
	}
}

func TestProviderMethodsHonourContext(t *testing.T) {
	provider, err := NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("провайдер не собрался: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := provider.UploadChunk(ctx, testChunkDir, testHash, []byte("x")); !errors.Is(err, context.Canceled) {
		t.Errorf("отменённый контекст не остановил кладку: %v", err)
	}
	if _, err := provider.DownloadChunk(ctx, testChunkDir, testHash); !errors.Is(err, context.Canceled) {
		t.Errorf("отменённый контекст не остановил чтение: %v", err)
	}
	if _, err := provider.ExistingChunks(ctx, testChunkDir); !errors.Is(err, context.Canceled) {
		t.Errorf("отменённый контекст не остановил листинг: %v", err)
	}
	if err := provider.UploadHeader(ctx, []byte("x")); !errors.Is(err, context.Canceled) {
		t.Errorf("отменённый контекст не остановил заголовок: %v", err)
	}
	if _, err := provider.DownloadHeader(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("отменённый контекст не остановил чтение заголовка: %v", err)
	}
}

func TestOpenProviderAcceptsOnlyExecutedTargets(t *testing.T) {
	const secret = "sekretnyj-parol"

	good := []struct{ target, name string }{
		{"webdav+https://cloud.example/dst", ProviderWebDAV},
		{"webdav+https://user@" + "cloud.example/dst", ProviderWebDAV},
		{"webdav+https://user:" + secret + "@cloud.example/dst", ProviderWebDAV},
		{"WEBDAV+HTTPS://cloud.example/dst", ProviderWebDAV},
		{"webdav+HTTPS://cloud.example/a/b", ProviderWebDAV},
		{"webdav+http://127.0.0.1:8080/dst", ProviderWebDAV},
		{"webdav+http://localhost:8080/dst", ProviderWebDAV},
		{"webdav+https://[::1]:8443/dst", ProviderWebDAV},
	}
	for _, c := range good {
		provider, err := OpenProvider(c.target)
		if err != nil {
			t.Fatalf("цель %q отвергнута: %v", c.target, err)
		}
		if provider.Name() != c.name {
			t.Fatalf("цель %q назвалась %q", c.target, provider.Name())
		}
	}

	bad := []string{
		"", "   ", "\t",
		"bundle", "bundles/one", "..",
		"gdrive://token@appdata", "yandex://token@app:/backup", "lan://192.168.1.5:54331",
		"webdav://host/dst", "https://cloud.example/x", "s3://bucket/key", "://x",
		"webdav+https://", "webdav+https:///dst", "webdav+https://cloud.example",
		"webdav+https://cloud.example/", "webdav+https://cloud.example//dst",
		"webdav+https://cloud.example/a/../b", `webdav+https://cloud.example\a\b`,
		"webdav+https://cloud.example/dst?fields=x", "webdav+https://cloud.example/dst#frag",
		"webdav+http://cloud.example/dst", "webdav+http://10.0.0.5/dst",
		"webdav+https://user:pass@cloud.example/dst?x=1",
	}
	for _, target := range bad {
		provider, err := OpenProvider(target)
		if err == nil {
			t.Fatalf("цель %q принята как %q: её обязан отвергнуть разбор", target, provider.Name())
		}
		if strings.Contains(err.Error(), "pass") || strings.Contains(err.Error(), secret) {
			t.Fatalf("отказ по %q процитировал учётные данные: %v", target, err)
		}
	}

	// Абсолютный путь — локальный каталог, и только абсолютный.
	root := t.TempDir()
	local, err := OpenProvider(root)
	if err != nil {
		t.Fatalf("абсолютный путь отвергнут: %v", err)
	}
	if local.Name() != ProviderLocal {
		t.Fatalf("путь назвал себя %q", local.Name())
	}

	// Разбор цели кладёт адрес в поле, а учётные данные — в отдельное: в URL
	// пароль попал бы в каждый лог и в каждую ошибку клиента.
	provider, err := OpenProvider("webdav+https://user:" + secret + "@cloud.example/dst")
	if err != nil {
		t.Fatalf("цель с учётными данными отвергнута: %v", err)
	}
	wd, ok := provider.(*WebDAVProvider)
	if !ok {
		t.Fatalf("webdav+https дал %T", provider)
	}
	if got := wd.base.String(); got != "https://cloud.example/dst/" {
		t.Fatalf("BaseURL собран из цели криво: %q", got)
	}
	if strings.Contains(wd.base.String(), secret) {
		t.Fatal("пароль остался в адресе")
	}
	if wd.username != "user" || wd.password != secret {
		t.Fatalf("учётные данные разобраны криво: %q/%q", wd.username, wd.password)
	}
	if wd.timeout != defaultProviderTimeout {
		t.Fatalf("у webdav-клиента нет потолка времени: %v", wd.timeout)
	}
}

func TestRedactedTargetNeverRepeatsThePassword(t *testing.T) {
	const secret = "пароль-в-userinfo"
	for _, target := range []string{
		"webdav+https://user:" + secret + "@cloud.example/dst",
		"webdav+http://user:" + secret + "@127.0.0.1:8080/dst",
		"/plain/path",
		"webdav+https://cloud.example/dst",
	} {
		if got := RedactedTarget(target); strings.Contains(got, secret) {
			t.Fatalf("RedactedTarget(%q) вернул %q с паролем", target, got)
		}
	}
	if got := RedactedTarget("webdav+https://user:" + secret + "@cloud.example/dst"); got != "webdav+https://user:xxxxxx@cloud.example/dst" {
		t.Fatalf("редактура собрала другой адрес: %q", got)
	}
}

func TestLegacyProvidersAreNotWired(t *testing.T) {
	// Все три заготовки — CloudProvider по типу, но не по содержанию: каждый
	// метод обязан называть отказ. Любая «тихая удача» (nil, пустое множество)
	// превратилась бы в пакет, который восстанавливается пустым.
	skeletons := []CloudProvider{
		NewGoogleDriveProvider("token", "proxy:443"),
		NewYandexDiskProvider("token"),
		NewLanDeviceSync("127.0.0.1:54331"),
	}
	ctx := context.Background()
	for _, p := range skeletons {
		if _, err := p.ExistingChunks(ctx, testChunkDir); !errors.Is(err, errProviderNotWired) {
			t.Errorf("%s: ExistingChunks дал %v", p.Name(), err)
		}
		if err := p.UploadChunk(ctx, testChunkDir, testHash, []byte("x")); !errors.Is(err, errProviderNotWired) {
			t.Errorf("%s: UploadChunk дал %v", p.Name(), err)
		}
		if _, err := p.DownloadChunk(ctx, testChunkDir, testHash); !errors.Is(err, errProviderNotWired) {
			t.Errorf("%s: DownloadChunk дал %v", p.Name(), err)
		}
		if err := p.UploadHeader(ctx, []byte("x")); !errors.Is(err, errProviderNotWired) {
			t.Errorf("%s: UploadHeader дал %v", p.Name(), err)
		}
		if _, err := p.DownloadHeader(ctx); !errors.Is(err, errProviderNotWired) {
			t.Errorf("%s: DownloadHeader дал %v", p.Name(), err)
		}

		// Ни одна заготовка не должна появляться из OpenProvider: иначе
		// «не подключено» превратилось бы в «работает, но пусто».
		provider, err := OpenProvider(p.Name() + "://host/dst")
		if err == nil {
			t.Errorf("цель схемы %q принята: %s", p.Name(), provider.Name())
			continue
		}
		if !strings.Contains(err.Error(), p.Name()) {
			t.Errorf("отказ по схеме %q её не называет: %v", p.Name(), err)
		}
	}
}

func TestEngineStopsBeforeFirstUploadIfListingFails(t *testing.T) {
	// Единственный легитимный ответ «чанков нет» — пустое множество. Провайдер,
	// который ошибается при листинге, обязан остановить движок до первой кладки:
	// иначе «пакет почти собрался», а в хранилище половина.
	listing := &listingFailsProvider{}
	if _, err := NewSnapshotEngineWithProvider(testPassphrase, listing, mustSalt(t), TestKDF); err == nil {
		t.Fatal("движок собрался поверх провайдера, который не может перечислить своё содержимое")
	}
	if listing.uploads != 0 {
		t.Fatalf("до отказа листинга уехало %d чанков", listing.uploads)
	}

	// nil-провайдер — отказ, а не паника на первом чанке.
	if _, err := NewSnapshotEngineWithProvider(testPassphrase, nil, mustSalt(t), TestKDF); err == nil {
		t.Fatal("движок собрался без хранилища")
	}

	// nil-карта при успешном листинге — это «чанков нет», а не ошибка.
	empty := &nilMapProvider{}
	e, err := NewSnapshotEngineWithProvider(testPassphrase, empty, mustSalt(t), TestKDF)
	if err != nil {
		t.Fatalf("пустой листинг стал ошибкой: %v", err)
	}
	if _, _, err := e.ProcessFileIntoChunks(randomPayload(t, 128)); err != nil {
		t.Fatalf("раскладка по пустому хранилищу не удалась: %v", err)
	}
	if empty.uploads == 0 {
		t.Fatal("раскладка ничего не выложила")
	}
}

func TestHeaderCheckIsProviderIndependent(t *testing.T) {
	// parseHeader — одна проверка на диск и на облако: разобранный, но
	// бессмысленный заголовок не обязан проходить только потому, что его
	// принёс HTTP.
	dir := t.TempDir()
	e := newTestEngine(t, testPassphrase, dir, mustSalt(t))
	payload := randomPayload(t, 200*1024)
	descriptors, totalChunk, err := e.ProcessFileIntoChunks(payload)
	if err != nil {
		t.Fatalf("чанки не легли: %v", err)
	}
	header, err := e.Seal(descriptors, uint64(len(payload)), totalChunk)
	if err != nil {
		t.Fatalf("манифест не собрался: %v", err)
	}

	provider := &memoryProvider{objects: make(map[string][]byte)}
	if err := WriteHeaderTo(provider, header); err != nil {
		t.Fatalf("заголовок не выложен: %v", err)
	}
	again, err := ReadHeaderFrom(provider)
	if err != nil {
		t.Fatalf("заголовок не читается: %v", err)
	}
	if again.BackupID != header.BackupID || again.TotalChunks != header.TotalChunks ||
		!equalBytes(again.RootMerkleHash, header.RootMerkleHash) {
		t.Fatal("заголовок после выкладки в провайдер изменился")
	}

	shortSalt, err := json.Marshal(&Header{Magic: headerMagic, Salt: make([]byte, SaltSize-1),
		BackupID: 1, TotalChunks: 1, RootMerkleHash: make([]byte, 32)})
	if err != nil {
		t.Fatalf("фикстура не собирается: %v", err)
	}
	cases := map[string][]byte{
		"пустой":      nil,
		"не JSON":     []byte("header.json"),
		"чужая магия": []byte(`{"magic":"другой","argon2id_salt":"AAAAAAAAAAAAAAAA"}`),
		"нулевой id":  []byte(`{"magic":"` + headerMagic + `","argon2id_salt":"AAAAAAAAAAAAAAAA","total_chunks":1,"root_merkle_hash":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}`),
		"нет чанков":  []byte(`{"magic":"` + headerMagic + `","argon2id_salt":"AAAAAAAAAAAAAAAA","backup_id":7,"root_merkle_hash":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}`),
		"salt короче": shortSalt,
	}
	for name, blob := range cases {
		if err := WriteHeaderTo(provider, &Header{}); err != nil {
			t.Fatalf("черновой заголовок не выложить: %v", err)
		}
		provider.objects[HeaderName] = blob
		if _, err := ReadHeaderFrom(provider); err == nil {
			t.Errorf("заголовок %q принят", name)
		}
	}
}

// listingFailsProvider — хранилище, которое отвечает на листинг только «нет».
type listingFailsProvider struct{ uploads int }

func (p *listingFailsProvider) Name() string { return "listing-fails" }

func (p *listingFailsProvider) ExistingChunks(context.Context, string) (map[string]uint32, error) {
	return nil, errors.New("хранилище не отвечает")
}

func (p *listingFailsProvider) UploadChunk(_ context.Context, _, hashHex string, data []byte) error {
	p.uploads++
	if len(data) == 0 {
		return fmt.Errorf("чанк %s прилетел пустым", hashHex)
	}
	return nil
}

func (p *listingFailsProvider) DownloadChunk(context.Context, string, string) ([]byte, error) {
	return nil, errors.New("хранилище не отвечает")
}

func (p *listingFailsProvider) UploadHeader(context.Context, []byte) error { return nil }

func (p *listingFailsProvider) DownloadHeader(context.Context) ([]byte, error) {
	return nil, errors.New("хранилище не отвечает")
}

// nilMapProvider — провайдер, который отвечает «пусто» nil-картой.
type nilMapProvider struct{ uploads int }

func (p *nilMapProvider) Name() string { return "nil-map" }

func (p *nilMapProvider) ExistingChunks(context.Context, string) (map[string]uint32, error) {
	return nil, nil
}

func (p *nilMapProvider) UploadChunk(context.Context, string, string, []byte) error {
	p.uploads++
	return nil
}

func (p *nilMapProvider) DownloadChunk(context.Context, string, string) ([]byte, error) {
	return nil, errors.New("чанков нет")
}

func (p *nilMapProvider) UploadHeader(context.Context, []byte) error { return nil }

func (p *nilMapProvider) DownloadHeader(context.Context) ([]byte, error) {
	return nil, errors.New("заголовка нет")
}

// memoryProvider — хранилище в памяти карты, чтобы проверка заголовка не
// зависела одновременно от файловой системы и от сети.
type memoryProvider struct {
	objects map[string][]byte
}

func (p *memoryProvider) Name() string { return "memory" }

func (p *memoryProvider) ExistingChunks(context.Context, string) (map[string]uint32, error) {
	return map[string]uint32{}, nil
}

func (p *memoryProvider) UploadChunk(_ context.Context, chunkDir, hashHex string, data []byte) error {
	name, err := objectName(chunkDir, hashHex)
	if err != nil {
		return err
	}
	p.objects[name] = append([]byte(nil), data...)
	return nil
}

func (p *memoryProvider) DownloadChunk(_ context.Context, chunkDir, hashHex string) ([]byte, error) {
	name, err := objectName(chunkDir, hashHex)
	if err != nil {
		return nil, err
	}
	data, ok := p.objects[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrChunkAbsent, hashHex)
	}
	return data, nil
}

func (p *memoryProvider) UploadHeader(_ context.Context, data []byte) error {
	p.objects[HeaderName] = append([]byte(nil), data...)
	return nil
}

func (p *memoryProvider) DownloadHeader(context.Context) ([]byte, error) {
	data, ok := p.objects[HeaderName]
	if !ok {
		return nil, errors.New("заголовка нет")
	}
	return data, nil
}

func equalBytes(a, b []byte) bool {
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

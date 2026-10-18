// File: core/pkg/backup/providers.go
//
// Провайдер — то, где пакет лежит. С Шага 14 их два настоящих: каталог на диске
// (localProvider) и WebDAV-клиент (webdavProvider, см. webdav.go). Оба вызываются
// одним кодом: SnapshotEngine работает через этот интерфейс, а команды
// CreateBackup/RestoreBackup выбирают провайдер из destination_target через
// OpenProvider. До этого шага файл содержал четыре клиента, из которых ни один не
// выкладывал ни одного реального файла, а «успех» ListExistingChunks у WebDAV —
// пустое множество, нарисованное на месте несделанного PROPFIND.
//
// Что здесь решено и почему:
//
//   - Интерфейс оперирует «объектами пакета» (чанк по контент-хэшу в каталоге,
//     привязанном к ключу, плюс заголовок), а не путями: layout собирает одна
//     функция objectName, поэтому локальный каталог и облако имеют одинаковую
//     структуру, и пакет можно перенести между ними простым копированием.
//   - ExistingChunks обязан различать «каталога нет» (пустое множество — законно,
//     это первый пакет) и «провайдер ответил ерунду» (ошибка). Прежний WebDAV
//     возвращал пустое множество как успех при любом раскладе, и это означало бы
//     молча перевыложить всё либо — хуже — не заметить потерянные чанки.
//   - Размеры шифртекстов приходят вместе со списком: манифест обязан хранить
//     настоящую длину шифртекста повторного чанка (Шаг 12 приучил брать её с
//     диска, а не писать ноль), поэтому map[string]bool тут не хватает.
//   - Имя чанка проверяется до любого обращения к хранилищу: это всегда 64
//     hex-символа sha256, и ни «..», ни слэш, ни scheme в него не пройдут.
//
// С Шага 14 «один пакет на хранилище» сменился на «пакетов столько, сколько
// создавалось»: у заголовка появился именованный слот (headers/<ключ>/<id>.json),
// а header.json стал алиасом самого свежего, поэтому повторный пакет больше не
// стирает предыдущую точку восстановления. Чтобы это не кончилось вечным ростком
// хранилища, интерфейс получил именованные чтения, листинг слотов и удаление —
// именно их использует сборщик мусора (rotation.go), и именно поэтому delete
// обязан быть повторяемым: прерванный обход хранилища перечитывается заново.
//
// Три прежних «облачных» клиента (Google Drive, Yandex.Disk, LAN P2P Sync)
// оставлены поимённо, но их HTTP-код заменён названным отказом: он не выполнялся
// ни разу и в нескольких местах был неработоспособен по построению (подробности —
// у каждого). Исходный текст этих клиентов — в HEAD:
// `git show HEAD:core/pkg/backup/providers.go`
package backup

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CloudProvider — хранилище одного пакета. chunkDir — имя каталога чанков,
// привязанное к ключу (chunkDirName): провайдер не знает ключ и не должен его
// угадывать, иначе два пакета в одном каталоге снова переиспользовали бы файлы
// друг друга.
type CloudProvider interface {
	// Name — machine-friendly имя схемы для поля storage_provider в
	// BackupManifest: вызывающий обязан называть то хранилище, куда реально
	// выложил чанк.
	Name() string

	// ExistingChunks возвращает размеры шифртекстов чанков, которые провайдер уже
	// хранит в этом каталоге. Пустое множество легально только когда каталога нет.
	ExistingChunks(ctx context.Context, chunkDir string) (map[string]uint32, error)

	// UploadChunk кладёт шифртекст чанка. Повторная кладка тех же байт под тем же
	// именем обязана быть успехом, а не ошибкой: дедупликация — оптимизация.
	UploadChunk(ctx context.Context, chunkDir, hashHex string, data []byte) error

	// DownloadChunk читает шифртекст чанка. Отсутствующий чанк — ошибка с
	// указанием имени, а не пустые байты: пустой чанк раскрылся бы ошибкой AEAD и
	// спрятал бы настоящую причину.
	DownloadChunk(ctx context.Context, chunkDir, hashHex string) ([]byte, error)

	// UploadHeader/DownloadHeader — незашифрованный заголовок пакета (HeaderName)
	// в корне хранилища. С Шага 15 это алиас самого свежего пакета: он обязан
	// остаться, иначе прежние вызывающие (и пакет, снятый до ротации) перестанут
	// читаться. Настоящий список пакетов лежит в именованных слотах ниже.
	UploadHeader(ctx context.Context, data []byte) error
	DownloadHeader(ctx context.Context) ([]byte, error)
}

// BundleStore — CloudProvider плюс то, что нужно ротации: несколько именованных
// слотов заголовка и удаление. Оба исполняемых провайдера (localProvider,
// WebDAVProvider) реализуют его; три незадействованные заготовки — нет, и
// AsBundleStore даёт отказ, а не «пустое хранилище»: молча проигнорированная
// невозможность удалить чанк означала бы, что GC отчитался успехом, ничего не
// почистив.
//
// Слот — это header.json, но названный по ключу и id пакета
// (headers/<chunkDir>/<backupID>.json), поэтому повторный пакет не стирает
// предыдущую точку восстановления. header.json остаётся копией самого свежего
// слота, и именно его читают те, кто не знает о слотах.
//
// Все именованные операции принимают только имена, собранные headerSlotName (или
// алиас HeaderName): ни «..», ни чужой путь, ни имя чанка в них не проходят
// (validHeaderName).
type BundleStore interface {
	CloudProvider

	// ListHeaderNames возвращает имена всех объектов-заголовков хранилища: и
	// алиас header.json, и каждый слот headers/<ключ>/<id>.json. Пустой список
	// легален (хранилище пустое), ошибка обязана быть ошибкой, а не пустотой.
	ListHeaderNames(ctx context.Context) ([]string, error)

	// UploadHeaderNamed/DownloadHeaderNamed/DeleteHeaderNamed — то же, что
	// UploadHeader/DownloadHeader, но по точному имени объекта (HeaderName или
	// слот). DeleteHeaderNamed обязан быть идемпотентным: отсутствующий объект —
	// успех, а не ошибка, иначе прерванный GC не перечитать.
	UploadHeaderNamed(ctx context.Context, name string, data []byte) error
	DownloadHeaderNamed(ctx context.Context, name string) ([]byte, error)
	DeleteHeaderNamed(ctx context.Context, name string) error

	// DeleteChunk убирает шифртекст чанка. Идемпотентен по той же причине.
	DeleteChunk(ctx context.Context, chunkDir, hashHex string) error
}

// AsBundleStore — приводит провайдер к BundleStore. Отказ называет провайдера:
// вызывающий обязан понять, что ротация с этим хранилищем невозможна, а не что
// пакетов нет.
func AsBundleStore(provider CloudProvider) (BundleStore, error) {
	if provider == nil {
		return nil, errors.New("backup: провайдер не задан: ротации нужно хранилище")
	}
	store, ok := provider.(BundleStore)
	if !ok {
		return nil, fmt.Errorf("backup: провайдер %s не поддерживает ротацию пакетов: у него нет именованных слотов заголовка и удаления чанков", provider.Name())
	}
	return store, nil
}

const (
	// ProviderLocal — имя схемы локального каталога.
	ProviderLocal = "local"
	// ProviderWebDAV — имя схемы WebDAV.
	ProviderWebDAV = "webdav"

	// chunksRootName — каталог чанков в пакете (тот же, что вела прежняя
	// локальная раскладка).
	chunksRootName = "chunks"

	// headersRootName — каталог именованных слотов заголовка (Шаг 15). Разведён с
	// chunks по-разному нарочно: алиас header.json лежит в корне и читается тем
	// же кодом, что и до ротации, а слоты — то, что листит ListBundles.
	headersRootName = "headers"

	// maxChunkBlob — потолок шифртекста одного чанка: открытый чанк не больше
	// MaxChunkSize, сжатие может добавить байт на заголовки, XChaCha20-Poly1305 —
	// nonce+тег. За этим числом лежит либо чужой файл, либо провайдер, который
	// хочет сожрать память.
	maxChunkBlob = 2 * MaxChunkSize

	// maxHeaderBlob — потолок заголовка: манифест в шифртексте + открытые поля.
	maxHeaderBlob = 1 << 20

	// defaultProviderTimeout — на сколько затягивает одна операция провайдера.
	// Ограничение обязательно: CreateBackup/RestoreBackup исполняются внутри
	// Engine.Handle, который держит e.mu, а он нужен приёмному циклу. Неограниченная
	// сетевая операция остановила бы приём почты на всем ядре.
	defaultProviderTimeout = 20 * time.Second
)

// ErrChunkAbsent — чанка в хранилище нет. Отдельный тип ошибки нужен, чтобы
// вызывающий (и тест) мог отличить «провайдер не знает такого» от «провайдер
// недоступен»: Restore обязан сообщать первое, а не «не расшифровывается».
var ErrChunkAbsent = errors.New("backup: чанк в хранилище отсутствует")

// localProvider обязан оставаться BundleStore: ротация работает через этот
// интерфейс, поэтому потеря любого из шести новых методов — ошибка сборки, а не
// «повторный пакет снова перезапишет первый», что выяснилось бы только на
// восстановлении.
var _ BundleStore = (*localProvider)(nil)

// objectName — единственный сборщик имени объекта пакета:
// chunks/<chunkDir>/<hash[0:2]>/<hash>.bin. Одна функция на все провайдеры,
// поэтому и проверка имени здесь же: в objectName не может просочиться путь
// выше корня хранилища.
func objectName(chunkDir, hashHex string) (string, error) {
	if err := validChunkDirName(chunkDir); err != nil {
		return "", err
	}
	if err := validChunkHash(hashHex); err != nil {
		return "", err
	}
	return chunksRootName + "/" + chunkDir + "/" + hashHex[:2] + "/" + hashHex + ".bin", nil
}

func validChunkDirName(chunkDir string) error {
	if len(chunkDir) != 8 {
		return fmt.Errorf("backup: имя каталога чанков %q должно быть 8 hex-символами", chunkDir)
	}
	raw, err := hex.DecodeString(chunkDir)
	if err != nil {
		return fmt.Errorf("backup: имя каталога чанков %q не hex: %w", chunkDir, err)
	}
	// Канонический нижний регистр, как у имени чанка: каталог выводится из ключа
	// через hex.EncodeToString, и «ABCD1234» дал бы второй каталог под тем же
	// ключом — дедупликация его не видит, восстановление ищет в первом.
	if hex.EncodeToString(raw) != chunkDir {
		return fmt.Errorf("backup: имя каталога чанков %q написано не в нижнем регистре hex", chunkDir)
	}
	return nil
}

func validChunkHash(hashHex string) error {
	if len(hashHex) != 2*sha256.Size {
		return fmt.Errorf("backup: имя чанка %q должно быть %d hex-символов", hashHex, 2*sha256.Size)
	}
	raw, err := hex.DecodeString(hashHex)
	if err != nil {
		return fmt.Errorf("backup: имя чанка %q не hex: %w", hashHex, err)
	}
	// Регистр имеет значение: имя объекта — часть AAD, и «AB..» с «ab..» были бы
	// двумя разными файлами с одним шифртекстом внутри.
	if hex.EncodeToString(raw) != hashHex {
		return fmt.Errorf("backup: имя чанка %q написано не в нижнем регистре hex", hashHex)
	}
	return nil
}

// chunkHashFromName — обратное разбирание имени файла листинга: "…<hash>.bin" →
// hash. Всё, что не проходит validChunkHash, листинг просто не считает чанком.
func chunkHashFromName(base string) (string, bool) {
	if !strings.HasSuffix(base, ".bin") {
		return "", false
	}
	hash := strings.TrimSuffix(base, ".bin")
	if validChunkHash(hash) != nil {
		return "", false
	}
	return hash, true
}

// headerSlotName — имя слота заголовка одного пакета:
// headers/<chunkDir>/<backupID hex>.json. ID пишется 16 hex-символами
// (big-endian uint64) в нижнем регистре: два пакета одного ключа различаются
// только id, поэтому имя обязано быть каноническим — «AB..» и «ab..» дали бы один
// и тот же пакет за два разных слота, и листинг посчитал бы их двумя точками
// восстановления.
func headerSlotName(chunkDir string, backupID uint64) (string, error) {
	if err := validChunkDirName(chunkDir); err != nil {
		return "", err
	}
	if backupID == 0 {
		return "", errors.New("backup: заголовок пакета с нулевым id в слот не кладётся: id обязан отличать этот пакет от прочих")
	}
	return fmt.Sprintf("%s/%s/%016x.json", headersRootName, chunkDir, backupID), nil
}

// parseHeaderSlot разбирает имя слота обратно. ok=false для всего, что слотом не
// является (включая алиас header.json и чанки): листинг такие имена молча
// пропускает, а явный вызов на неразобранном имени получает ошибку.
func parseHeaderSlot(name string) (chunkDir string, backupID uint64, ok bool) {
	if !strings.HasSuffix(name, ".json") {
		return "", 0, false
	}
	prefix := headersRootName + "/"
	if !strings.HasPrefix(name, prefix) {
		return "", 0, false
	}
	rest := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".json")
	slash := strings.IndexByte(rest, '/')
	if slash <= 0 || slash != len(rest)-17 {
		return "", 0, false
	}
	dir, idHex := rest[:slash], rest[slash+1:]
	if err := validChunkDirName(dir); err != nil {
		return "", 0, false
	}
	raw, err := hex.DecodeString(idHex)
	if err != nil || len(raw) != 8 {
		return "", 0, false
	}
	if hex.EncodeToString(raw) != idHex {
		return "", 0, false
	}
	id := binary.BigEndian.Uint64(raw)
	if id == 0 {
		return "", 0, false
	}
	return dir, id, true
}

// validHeaderName — ворота для UploadHeaderNamed/DownloadHeaderNamed/
// DeleteHeaderNamed: по имени заголовка проходят только алиас и слот заголовка.
// Имя сюда приходит строкой от вызывающего (и от листинга облака), поэтому без
// этого «..\\..\\» или чужой путь превратились бы в запись и удаление вне каталога
// пакета.
//
// Чанки в этих воротах не просто так отсутствуют: у них нет операции по имени —
// к ним обращаются только через objectName, который собирает путь из проверенных
// частей. Пропусти слот-ворота чужое «chunks/...» имя, и ротация получила бы
// право стереть шифртекст, прочитав вместо него мусор, или положить заголовок туда,
// где листинг чанков ждёт бинарник.
func validHeaderName(name string) error {
	if name == HeaderName {
		return nil
	}
	if _, _, ok := parseHeaderSlot(name); ok {
		return nil
	}
	return fmt.Errorf("backup: имя %q не является заголовком пакета: допускаются %q и %s/<ключ>/<id>.json",
		name, HeaderName, headersRootName)
}

// -----------------------------------------------------------------------------
// ЛОКАЛЬНЫЙ КАТАЛОГ
// -----------------------------------------------------------------------------

// localProvider — пакет в каталоге файловой системы. Это тот же путь, которым
// бэкап жил всегда, но теперь он реализует интерфейс наравне с облаком: проверка
// «удалённый провайдер не врёт» распространяется и на него.
type localProvider struct {
	root string
}

// NewLocalProvider — провайдер-каталог. Каталог обязан быть абсолютным: у
// FFI-сборки (особенно на Android) текущая директория процесса неизвестна, и
// относительный путь означал бы пакет в случайном месте.
func NewLocalProvider(dir string) (CloudProvider, error) {
	if dir == "" {
		return nil, errors.New("backup: каталог пакета пуст")
	}
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("backup: каталог пакета %q обязан быть абсолютным путём: относительный разрешался бы по текущей директории процесса", dir)
	}
	return &localProvider{root: filepath.Clean(dir)}, nil
}

func (p *localProvider) Name() string { return ProviderLocal }

func (p *localProvider) dirFor(chunkDir string) (string, error) {
	if err := validChunkDirName(chunkDir); err != nil {
		return "", err
	}
	return filepath.Join(p.root, chunksRootName, chunkDir), nil
}

func (p *localProvider) ExistingChunks(ctx context.Context, chunkDir string) (map[string]uint32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := p.dirFor(chunkDir)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]uint32)
	// Сначала — чем вообще является путь каталога ключа. Отсутствие значит «первый
	// пакет», а файл на его месте значит «писаться сюда нельзя»: пустой листинг во
	// втором случае означал бы «чанков нет, можно класть», и первая же запись
	// ушла бы в чужой файл.
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return existing, nil
	}
	if err != nil {
		return nil, fmt.Errorf("backup: каталог чанков %s не перечисляется: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("backup: %s — файл, а не каталог чанков: пустой список по такому пути был бы ложью о хранилище", dir)
	}
	// Обход, а не Glob по «??»: подкаталог шарда мог не создаться, а мог создаться
	// и с другим именем — важно не принять каталог за чанк и не пропустить чанк.
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Черновик атомарной записи — не чанк: кладка пишет в *.bin.tmp и
		// переименовывает, оборванный черновик обязан оставаться мусором.
		if strings.HasSuffix(info.Name(), ".tmp") {
			return nil
		}
		hash, ok := chunkHashFromName(info.Name())
		if !ok {
			return nil
		}
		if info.Size() < 0 || info.Size() > maxChunkBlob {
			return fmt.Errorf("backup: чанк %s весит %d байт, а это вне потолка шифртекста %d", hash, info.Size(), maxChunkBlob)
		}
		existing[hash] = uint32(info.Size())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("backup: каталог чанков %s не перечисляется: %w", dir, err)
	}
	return existing, nil
}

func (p *localProvider) UploadChunk(ctx context.Context, chunkDir, hashHex string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name, err := objectName(chunkDir, hashHex)
	if err != nil {
		return err
	}
	return p.writeAtomic(filepath.Join(p.root, filepath.FromSlash(name)), data)
}

func (p *localProvider) DownloadChunk(ctx context.Context, chunkDir, hashHex string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	name, err := objectName(chunkDir, hashHex)
	if err != nil {
		return nil, err
	}
	return readCapped(filepath.Join(p.root, filepath.FromSlash(name)), maxChunkBlob, hashHex, ErrChunkAbsent)
}

func (p *localProvider) UploadHeader(ctx context.Context, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.writeAtomic(filepath.Join(p.root, HeaderName), data)
}

func (p *localProvider) DownloadHeader(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return readCapped(filepath.Join(p.root, HeaderName), maxHeaderBlob, HeaderName, nil)
}

// -----------------------------------------------------------------------------
// Ротация: именованные слоты заголовка и удаление (Шаг 15)
// -----------------------------------------------------------------------------

// headerPath — абсолютный путь объекта-заголовка внутри корня пакета. Имя
// проходит validHeaderName ДО сборки пути: листинг облака мог принести что
// угодно, а «..» в имени означал бы чтение и удаление вне каталога пакета.
func (p *localProvider) headerPath(name string) (string, error) {
	if err := validHeaderName(name); err != nil {
		return "", err
	}
	return filepath.Join(p.root, filepath.FromSlash(name)), nil
}

func (p *localProvider) UploadHeaderNamed(ctx context.Context, name string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := p.headerPath(name)
	if err != nil {
		return err
	}
	if len(data) > maxHeaderBlob {
		return fmt.Errorf("backup: заголовок %s весит %d байт, а потолок %d", name, len(data), maxHeaderBlob)
	}
	return p.writeAtomic(path, data)
}

func (p *localProvider) DownloadHeaderNamed(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := p.headerPath(name)
	if err != nil {
		return nil, err
	}
	return readCapped(path, maxHeaderBlob, name, nil)
}

// DeleteHeaderNamed убирает заголовок. Отсутствующий файл — успех: прерванный
// сборщик мусора перечитывается с начала, и второе удаление того, чего уже нет, не
// имеет права быть ошибкой.
func (p *localProvider) DeleteHeaderNamed(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := p.headerPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("backup: заголовок %s не удалить: %w", name, err)
	}
	return nil
}

// DeleteChunk убирает шифртекст чанка; отсутствующий чанк — успех (см.
// DeleteHeaderNamed про повторяемость удаления).
func (p *localProvider) DeleteChunk(ctx context.Context, chunkDir, hashHex string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name, err := objectName(chunkDir, hashHex)
	if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(p.root, filepath.FromSlash(name))); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("backup: чанк %s не удалить: %w", hashHex, err)
	}
	return nil
}

// ListHeaderNames собирает имена всех заголовков хранилища: алиас, если он есть,
// и каждый слот из headers/<ключ>/<id>.json. Порядок не гарантируется и вызывающему
// не нужен — ListBundles сортирует сам.
//
// Обход, а не Glob: у каталога ключа мог не создаться шард, а мог создаться
// каталог с другим именем; важно не принять каталог за слот и не пропустить
// годный слот. Черновик *.tmp — мусор обрыва, не заголовок.
func (p *localProvider) ListHeaderNames(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]string, 0, 8)

	// Алиас лежит в корне: его присутствие проверяется отдельно, потому что у
	// несуществующего header.json нет каталога, по которому можно пройтись.
	alias := filepath.Join(p.root, HeaderName)
	if info, err := os.Stat(alias); err == nil && !info.IsDir() {
		out = append(out, HeaderName)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("backup: алиас пакета %s не перечисляется: %w", alias, err)
	}

	dir := filepath.Join(p.root, headersRootName)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("backup: каталог заголовков %s не перечисляется: %w", dir, err)
	}
	if !info.IsDir() {
		// Файл на месте каталога слотов — ложь о хранилище, а не «слотов нет»:
		// пустой список разрешил бы писать слоты поверх чужого файла.
		return nil, fmt.Errorf("backup: %s — файл, а не каталог заголовков: пустой список по такому пути был бы враньём", dir)
	}
	err = filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if fi.IsDir() || strings.HasSuffix(fi.Name(), ".tmp") {
			return nil
		}
		rel, err := filepath.Rel(p.root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if _, _, ok := parseHeaderSlot(name); ok {
			out = append(out, name)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("backup: каталог заголовков %s не перечисляется: %w", dir, err)
	}
	return out, nil
}

// writeAtomic — запись через черновик и переименование. Обычный WriteFile оставил
// бы на диске оборванный чанк с настоящим именем, и дедупликация приняла бы его за
// годный: восстановление потом падает на AEAD, а починить его нельзя, не зная,
// что файл битый.
func (p *localProvider) writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("backup: каталог %s не создать: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("backup: черновик %s не записан: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("backup: %s не занял своё имя: %w", path, err)
	}
	return nil
}

// readCapped — чтение файла пакета с потолком размера. absent говорит, чем
// считается отсутствующий файл: для чанка это ErrChunkAbsent (вызывающий обязан
// отличить «нет в хранилище» от «не читается»), для заголовка — названный файл,
// потому что пустой манифест — это не «чанка нет», а «пакета нет».
func readCapped(path string, limit int64, what string, absent error) ([]byte, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		if absent != nil {
			return nil, fmt.Errorf("%w: %s", absent, what)
		}
		return nil, fmt.Errorf("backup: файл пакета %s отсутствует", what)
	}
	if err != nil {
		return nil, fmt.Errorf("backup: %s не читается: %w", path, err)
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("backup: %s весит %d байт, а потолок для него %d", path, info.Size(), limit)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("backup: %s не читается: %w", path, err)
	}
	return data, nil
}

// -----------------------------------------------------------------------------
// ВЫБОР ПРОВАЙДЕРА ПО ЦЕЛИ
// -----------------------------------------------------------------------------

// schemeRefusals — схемы, которые вызывающий обязан получить отказом, а не
// «тихой удачей». Причина у каждой своя и названа: тот, кто решит это
// подключать, должен знать, что именно не сверялось с живой стороной сервиса.
var schemeRefusals = map[string]string{
	"gdrive": "клиент Google Drive строил POST на uploadType=multipart без multipart-тела (метаданные файла не отправлялись вовсе), статус ответов на скачивании не сверялся, а живые API не трогались ни разу",
	"yandex": "клиент Yandex.Disk не создавал каталог app:/chunks и не сверял статус первого ответа, поэтому загрузка в несуществующий путь превращалась в ошибку второй руки; живые API не трогались ни разу",
	"lan":    "LAN-провайдер стучился на /sync/chunk и /sync/manifest, которых не реализует ни один сервер репозитория, и считал успехом любой ответ, включая 404",
}

// OpenProvider разбирает destination_target/source_target в провайдер:
//   - путь в файловой системе (обязательно абсолютный) — localProvider;
//   - webdav+http:// или webdav+https://user:pass@host/path — настоящий WebDAV;
//     незащищённый http разрешён только на loopback: пакет летит в чужое облако,
//     и Basic-пароль по открытому тексту через проксирующий интернет — это утечка
//     учётных данных, а не «экономия на сертификате»;
//   - gdrive/yandex/lan — названный отказ (см. конец файла);
//   - всё остальное со «://» — неизвестная схема.
//
// Тексты ошибок никогда не содержат пароль: цель в них возвращается через
// RedactedTarget.
func OpenProvider(raw string) (CloudProvider, error) {
	target := strings.TrimSpace(raw)
	if target == "" {
		return nil, errors.New("backup: цель пуста: провайдер обязан знать, куда класть пакет")
	}

	if strings.Contains(target, "://") {
		scheme, rest, err := splitTargetScheme(target)
		if err != nil {
			return nil, err
		}
		if reason, refused := schemeRefusals[scheme]; refused {
			return nil, fmt.Errorf("backup: цель %q называет схему %q, а она не подключена: %s", RedactedTarget(target), scheme, reason)
		}
		switch scheme {
		case "webdav+http", "webdav+https":
			cfg, err := parseWebDAVTarget(target, rest, scheme == "webdav+http")
			if err != nil {
				return nil, fmt.Errorf("backup: webdav-цель %q не разобрана: %w", RedactedTarget(target), err)
			}
			return NewWebDAVProvider(cfg)
		default:
			return nil, fmt.Errorf("backup: схема %q цели %q этому коду не знакома: OpenProvider знает абсолютные пути и webdav+http(s)", scheme, RedactedTarget(target))
		}
	}

	return NewLocalProvider(target)
}

// splitTargetScheme — схема до «://» в нижнем регистре + остальное. Отдельная
// функция потому, что «C:\Users\…» не содержит «://» и попадает в ветку путей, а
// «webdav+HTTP://» с заглавной буквы обязана читаться так же, как строчная.
func splitTargetScheme(target string) (scheme, rest string, err error) {
	at := strings.Index(target, "://")
	if at <= 0 {
		return "", "", fmt.Errorf("backup: цель %q содержит «://» без схемы слева", RedactedTarget(target))
	}
	scheme = strings.ToLower(target[:at])
	rest = target[at+len("://"):]
	if scheme == "" || rest == "" {
		return "", "", fmt.Errorf("backup: цель %q разбрана на схему и адрес криво", RedactedTarget(target))
	}
	return scheme, rest, nil
}

// parseWebDAVTarget — разбор host/port/path/userinfo из «webdav+http(s)://…».
// rest передаётся без схемы: url.Parse требует scheme, какой именно — решает
// вызывающий (он же решает, разрешён ли http).
func parseWebDAVTarget(target, rest string, cleartext bool) (WebDAVConfig, error) {
	parsed, err := url.Parse("http://" + rest)
	if err != nil {
		// Без «%w»: net/url в тексте ошибки повторяет адрес целиком, userinfo
		// включительно, а цель сюда приходит ровно та, что написал вызывающий.
		// Причина отказа названа словами, пароль в ней не воспроизводится.
		return WebDAVConfig{}, errors.New("адрес не парсится: недопустимые символы в userinfo или пути, учётные данные обязаны быть percent-encoded")
	}
	if parsed.Host == "" {
		return WebDAVConfig{}, errors.New("адрес без хоста")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return WebDAVConfig{}, errors.New("у webdav-цели не бывает query и fragment: они были бы потеряны при раскладке в путь")
	}
	path := parsed.Path
	if path == "" || path == "/" {
		return WebDAVConfig{}, errors.New("путь цели обязан вести в подкаталог, а не в корень хранилища: пакет писал бы header.json и chunks/ в корень чужого диска")
	}
	if strings.HasPrefix(path, "//") {
		return WebDAVConfig{}, fmt.Errorf("путь цели %q начинается с двойного слэша: это другой host по RFC", path)
	}
	if strings.Contains(path, "\\") {
		return WebDAVConfig{}, fmt.Errorf("путь цели %q содержит обратный слэш: webdav оперирует slash-разделителями", path)
	}
	if strings.Contains(path, "..") {
		return WebDAVConfig{}, fmt.Errorf("путь цели %q содержит «..»: пакет обязан лежать внутри названного подкаталога", path)
	}
	if cleartext && !loopbackHost(parsed.Hostname()) {
		return WebDAVConfig{}, fmt.Errorf("хост %q не loopback, а webdav+http пересылал бы Basic-пароль открытым текстом: нужен webdav+https", parsed.Hostname())
	}
	user := parsed.User.Username()
	pass, _ := parsed.User.Password()
	scheme := "https"
	if cleartext {
		scheme = "http"
	}
	return WebDAVConfig{
		BaseURL:  scheme + "://" + parsed.Host + path,
		Username: user,
		Password: pass,
	}, nil
}

// loopbackHost — адреса, куда разрешён незащищённый HTTP: их слушает эта же
// машина (тесты и будущая синхронизация устройств), а не сеть.
func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	return host == "127.0.0.1" || host == "::1" || strings.HasPrefix(host, "127.")
}

// RedactedTarget — цель для текстов ошибок: пароль из userinfo заменён на
// «xxxxxx», запрос и якорь отброшены целиком.
//
// Разбор url здесь не опора, а ловушка: net/url отвергает userinfo с нелатинными
// символами, а такая цель приходит сюда ровно тем путём, которым она отвергнута,
// и сообщение о неразобранной цели выдал бы пароль открытым текстом. Поэтому
// границы authority ищутся по символам-разделителям, а не по результатам парсинга.
func RedactedTarget(target string) string {
	trimmed := strings.TrimSpace(target)
	at := strings.Index(trimmed, "://")
	if at < 0 {
		return trimmed
	}
	scheme, rest := trimmed[:at], trimmed[at+3:]

	// Authority кончается первым «/», «?» или «#». Хвост после неё — путь, запрос
	// или якорь; в облачных целях запрос бывал носителем токена, поэтому в
	// редактуре остаются только путь без запроса и якоря.
	end := len(rest)
	for i := 0; i < len(rest); i++ {
		if c := rest[i]; c == '/' || c == '?' || c == '#' {
			end = i
			break
		}
	}
	tail := rest[end:]
	if cut := strings.IndexAny(tail, "?#"); cut >= 0 {
		tail = tail[:cut]
	}

	authority := rest[:end]
	if sep := strings.LastIndexByte(authority, '@'); sep >= 0 {
		host := authority[sep+1:]
		userinfo := authority[:sep]
		if colon := strings.IndexByte(userinfo, ':'); colon >= 0 {
			authority = userinfo[:colon] + ":xxxxxx@" + host
		} else {
			authority = userinfo + "@" + host
		}
	}
	return scheme + "://" + authority + tail
}

// -----------------------------------------------------------------------------
// ЗАГОТОВКИ: СХЕМЫ, КОТОРЫЕ НИКТО НЕ ПОДКЛЮЧАЕТ
// -----------------------------------------------------------------------------

// errProviderNotWired — ответ незадействованного провайдера. Методы не возвращают «успех с
// пустым результатом» ровно по той же причине, по которой был выброшен прежний
// WebDAV ListExistingChunks: молчаливый успех в бэкапе означает потерянные файлы,
// которые обнаруживаются только при восстановлении.
var errProviderNotWired = errors.New("backup: провайдер не подключён: ни один байт через него не выкладывался и не скачивался, см. пакетное описание")

// GoogleDriveProvider — скелет. Прежняя реализация (её видно в git) отправляла
// тело на «upload/drive/v3/files?uploadType=multipart» без multipart-блока с
// метаданными, то есть Google такой запрос принять не мог; статус ответов при
// скачивании не сверялся, поэтому 404 превращался в «пустой файл», а поиск по
// имени подставлялся в запрос без экранирования. Живые API не трогались ни разу,
// и сверить этот путь нечем.
type GoogleDriveProvider struct {
	accessToken string
	proxyHost   string
}

// NewGoogleDriveProvider сохраняем как документ: конструктор не вызывает
// OpenProvider, и тест TestLegacyProvidersAreNotWired следит, чтобы так и было.
func NewGoogleDriveProvider(accessToken, yggdrasilProxyHost string) *GoogleDriveProvider {
	return &GoogleDriveProvider{accessToken: accessToken, proxyHost: yggdrasilProxyHost}
}

func (p *GoogleDriveProvider) Name() string { return "gdrive" }

func (p *GoogleDriveProvider) ExistingChunks(ctx context.Context, chunkDir string) (map[string]uint32, error) {
	return nil, errProviderNotWired
}

func (p *GoogleDriveProvider) UploadChunk(ctx context.Context, chunkDir, hashHex string, data []byte) error {
	return errProviderNotWired
}

func (p *GoogleDriveProvider) DownloadChunk(ctx context.Context, chunkDir, hashHex string) ([]byte, error) {
	return nil, errProviderNotWired
}

func (p *GoogleDriveProvider) UploadHeader(ctx context.Context, data []byte) error {
	return errProviderNotWired
}

func (p *GoogleDriveProvider) DownloadHeader(ctx context.Context) ([]byte, error) {
	return nil, errProviderNotWired
}

// YandexDiskProvider — скелет. Прежняя реализация брала upload-href и клала
// байты методом PUT, но каталог app:/chunks не создавала, а ответ первого запроса
// (который как раз и сообщает, что пути нет) не сверялась ни со статусом, ни с
// наличием href. Живые API не трогались ни разу.
type YandexDiskProvider struct {
	oauthToken string
}

func NewYandexDiskProvider(oauthToken string) *YandexDiskProvider {
	return &YandexDiskProvider{oauthToken: oauthToken}
}

func (p *YandexDiskProvider) Name() string { return "yandex" }

func (p *YandexDiskProvider) ExistingChunks(ctx context.Context, chunkDir string) (map[string]uint32, error) {
	return nil, errProviderNotWired
}

func (p *YandexDiskProvider) UploadChunk(ctx context.Context, chunkDir, hashHex string, data []byte) error {
	return errProviderNotWired
}

func (p *YandexDiskProvider) DownloadChunk(ctx context.Context, chunkDir, hashHex string) ([]byte, error) {
	return nil, errProviderNotWired
}

func (p *YandexDiskProvider) UploadHeader(ctx context.Context, data []byte) error {
	return errProviderNotWired
}

func (p *YandexDiskProvider) DownloadHeader(ctx context.Context) ([]byte, error) {
	return nil, errProviderNotWired
}

// LanDeviceSync — скелет. Прежняя реализация ходила на /sync/chunk и
// /sync/manifest, которых не реализует ни один сервер репозитория, и считала
// успехом любой ответ, включая 404: «устройство синхронизировано» означало бы
// «запрос ушёл». Если этот путь и понадобится, начинать надо с серверной стороны
// (она же — будущее окно синхронизации Windows ↔ Android), а не с клиента.
type LanDeviceSync struct {
	remoteAddress string
}

func NewLanDeviceSync(remoteAddress string) *LanDeviceSync {
	return &LanDeviceSync{remoteAddress: remoteAddress}
}

func (p *LanDeviceSync) Name() string { return "lan" }

func (p *LanDeviceSync) ExistingChunks(ctx context.Context, chunkDir string) (map[string]uint32, error) {
	return nil, errProviderNotWired
}

func (p *LanDeviceSync) UploadChunk(ctx context.Context, chunkDir, hashHex string, data []byte) error {
	return errProviderNotWired
}

func (p *LanDeviceSync) DownloadChunk(ctx context.Context, chunkDir, hashHex string) ([]byte, error) {
	return nil, errProviderNotWired
}

func (p *LanDeviceSync) UploadHeader(ctx context.Context, data []byte) error {
	return errProviderNotWired
}

func (p *LanDeviceSync) DownloadHeader(ctx context.Context) ([]byte, error) {
	return nil, errProviderNotWired
}

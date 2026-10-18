// Файл: core/pkg/backup/webdav.go
//
// Настоящий WebDAV-клиент пакета. До Шага 14 «WebDAVProvider» в этом пакете делал
// PUT и GET, но: список чанков возвращал пустое множество на месте несделанного
// PROPFIND, ответ GET не сверялся со статусом (404 от провайдера превращался в
// «пустой чанк», который потом не расшифровывается), каталоги не создавались (PUT
// в несуществующую коллекцию по RFC 4918 §9.7.1 — отказ (эталонный сервер отвечает
// 409, спецификация допускает и 507), а клиент молча следовал
// редиректам, пересылая Basic-пароль на чужой хост.
//
// Проверка этого клиента — не самописная фикстура: тесты гоняют его против эталонного
// WebDAV-сервера golang.org/x/net/webdav (MemFS) в этом же процессе, то есть с той же
// стороны границы, с которой Шаг 4-a и Шаг 11 сверяли X3DH и BIP-39 вторым независимым
// кодом. Живые облака (Nextcloud, Mail.ru, WebDAV у Yandex.Disk) отсюда не трогались:
// они требуют учётных данных, и что именно там отвечает каждый — не заявляется.
package backup

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// WebDAVConfig — параметры подключения. Пароль живёт отдельным полем, а не
// внутри BaseURL: в URL он попал бы в каждый лог и в каждую ошибку.
type WebDAVConfig struct {
	// BaseURL — «http(s)://host[:port]/подкаталог» без userinfo, без query.
	BaseURL string
	// Username/Password — Basic-учётные данные (WebDAV не знает других).
	Username string
	Password string
	// Timeout — потолок одной операции. 0 означает defaultProviderTimeout.
	Timeout time.Duration
}

// WebDAVProvider — CloudProvider поверх WebDAV-коллекции.
type WebDAVProvider struct {
	base     *url.URL
	username string
	password string
	timeout  time.Duration
	client   *http.Client

	// made — коллекции, которые этот провайдер уже создавал. MKCOL на каждый
	// PUT означал бы три запроса на чанк вместо одного.
	mu   sync.Mutex
	made map[string]bool
}

// NewWebDAVProvider собирает клиент и проверяет конфиг: ошибки здесь обязаны
// быть, а не отложены до первого запроса, потому что вызывающий (Engine.Handle)
// держит замок приёмного цикла.
func NewWebDAVProvider(cfg WebDAVConfig) (*WebDAVProvider, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("backup: webdav-конфиг без BaseURL")
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		// Без «%w»: net/url повторяет адрес вместе с userinfo, а текст отказа
		// уходит в логи и в ответ вызывающему.
		return nil, fmt.Errorf("backup: webdav BaseURL %q не разбирается: недопустимые символы в userinfo или пути", RedactedTarget(cfg.BaseURL))
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("backup: webdav BaseURL %q по схеме %q не http(s)", RedactedTarget(cfg.BaseURL), parsed.Scheme)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("backup: в BaseURL %q лежит userinfo: учётные данные задаются полями Username/Password, а не адресом", RedactedTarget(cfg.BaseURL))
	}
	if parsed.Host == "" {
		return nil, errors.New("backup: webdav BaseURL без хоста")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("backup: webdav BaseURL не обязан нести query или fragment")
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		// Без слэша склейка отрезала бы последний сегмент: «…/ygggram» превратил
		// первый же запрос в «…/ygggramchunks/…».
		parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultProviderTimeout
	}
	// CheckRedirect отвергает редиректы наотрез. Следовать им значило бы
	// переслать Basic-пароль на хост, который назвал сервер: именно так утекают
	// учётные данные через доверенный редирект, и именно этим грешил прежний код.
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("backup: webdav-сервер ответил редиректом, а клиент их не следует: вместе с Location на чужой хост уехал бы Basic-пароль")
		},
	}
	return &WebDAVProvider{
		base:     parsed,
		username: cfg.Username,
		password: cfg.Password,
		timeout:  timeout,
		client:   client,
		made:     make(map[string]bool),
	}, nil
}

func (p *WebDAVProvider) Name() string { return ProviderWebDAV }

// WebDAVProvider — тоже BundleStore: ротация обязана работать и в облаке, а не
// только на диске, иначе «повторный пакет не трогает первый» было бы обещанием
// только для локального каталога.
var _ BundleStore = (*WebDAVProvider)(nil)

// checkName — то, что имя объекта не пытается уйти из подкаталога пакета ни
// одним сегментом. Одинаковая проверка для PUT, GET и PROPFIND: у листинга имя
// собирается так же, и пропуск здесь означал бы чтение чужой коллекции.
func checkName(name string) error {
	for _, seg := range strings.Split(name, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("backup: имя объекта %q содержит пустой или относительный сегмент", name)
		}
	}
	return nil
}

// joinPath — склейка сегментов в путь с одним слэшем между ними. Без path.Join:
// он чистит «..», а нам такие имена нужно отвергать, а не «починить».
func joinPath(segs ...string) string {
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if s != "" {
			out = append(out, strings.Trim(s, "/"))
		}
	}
	return "/" + strings.Join(out, "/")
}

// baseSegments — сегменты подкаталога пакета из BaseURL («/backup/2026-10» →
// backup, 2026-10). Они же — первые коллекции, которые обязан создать клиент.
func (p *WebDAVProvider) baseSegments() []string {
	segs := make([]string, 0, 4)
	for _, s := range strings.Split(strings.Trim(p.base.Path, "/"), "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	return segs
}

// urlFor — адрес объекта внутри подкаталога пакета. Сегменты кладутся в
// url.URL.Path (не в EscapedPath), поэтому Go сам закодирует то, что требует
// кодирования, а «%2e%2e» не пройдёт: имена проверяются до сборки.
func (p *WebDAVProvider) urlFor(name string) (string, error) {
	if err := checkName(name); err != nil {
		return "", err
	}
	u := *p.base
	u.Path = joinPath(append(p.baseSegments(), name)...)
	return u.String(), nil
}

// collectionChain — пути коллекций, которые обязаны существовать перед PUT
// объекта: PUT в несуществующую коллекцию — отказ (RFC 4918 §9.7.1 допускает 409
// и 507, эталонный golang.org/x/net/webdav отвечает 409 Conflict), а не «тихая
// удача». Цепочка начинается с сегментов самого BaseURL — цель вида
// «…/backup/2026-10» сервер за клиента не создаёт, и без этого первый же PUT
// падал бы, а вызывающий назвал бы это «облако недоступно».
func (p *WebDAVProvider) collectionChain(name string) []string {
	segs := p.baseSegments()
	objSegs := strings.Split(name, "/")
	segs = append(segs, objSegs[:len(objSegs)-1]...)

	out := make([]string, 0, len(segs))
	for i := 1; i <= len(segs); i++ {
		out = append(out, joinPath(segs[:i]...))
	}
	return out
}

// ensureCollections создаёт недостающие коллекции. 405 Method Not Allowed — это
// «уже есть», и он годный: проверять существование отдельным HEAD-запросом
// означало бы удвоить число запросов на каждый чанк.
func (p *WebDAVProvider) ensureCollections(ctx context.Context, name string) error {
	for _, path := range p.collectionChain(name) {
		u := *p.base
		u.Path = path
		addr := u.String()

		p.mu.Lock()
		known := p.made[addr]
		p.mu.Unlock()
		if known {
			continue
		}
		if _, _, err := p.do(ctx, "MKCOL", addr, nil, []int{
			http.StatusCreated, http.StatusOK, http.StatusNoContent, http.StatusMethodNotAllowed,
		}, 0); err != nil {
			return fmt.Errorf("backup: коллекция %q не создана: %w", path, err)
		}
		p.mu.Lock()
		p.made[addr] = true
		p.mu.Unlock()
	}
	return nil
}

// do — один запрос с потолком по времени, статусом из списка годных и телом
// ответа, ограниченным по размеру.
func (p *WebDAVProvider) do(ctx context.Context, method, addr string, body []byte, okStatus []int, maxReply int64) ([]byte, *http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, addr, reader)
	if err != nil {
		return nil, nil, fmt.Errorf("backup: запрос %s к %s не собрать: %w", method, addr, err)
	}
	if p.username != "" || p.password != "" {
		req.SetBasicAuth(p.username, p.password)
	}
	switch method {
	case "PROPFIND":
		req.Header.Set("Depth", "infinity")
		req.Header.Set("Content-Type", "application/xml")
	case http.MethodPut:
		req.Header.Set("Content-Type", "application/octet-stream")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		// Таймаут называется отдельно: «сервер не ответил за N» и «сервер ответил
		// ошибкой» для вызывающего — разные ситуации.
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, nil, fmt.Errorf("backup: webdav %s %s не ответил за %s: %w", method, addr, p.timeout, err)
		}
		return nil, nil, fmt.Errorf("backup: webdav %s %s не выполнен: %w", method, addr, err)
	}
	defer resp.Body.Close()

	accepted := false
	for _, s := range okStatus {
		if resp.StatusCode == s {
			accepted = true
			break
		}
	}
	if !accepted {
		return nil, resp, fmt.Errorf("backup: webdav %s %s ответил %s (ожидались %v)", method, addr, resp.Status, okStatus)
	}
	if maxReply <= 0 {
		return nil, resp, nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReply+1))
	if err != nil {
		return nil, resp, fmt.Errorf("backup: тело ответа webdav %s не дочитано: %w", addr, err)
	}
	if int64(len(data)) > maxReply {
		return nil, resp, fmt.Errorf("backup: webdav %s отдал больше %d байт: это либо чужой файл, либо провайдер, который хочет сожрать память", addr, maxReply)
	}
	return data, resp, nil
}

func (p *WebDAVProvider) UploadChunk(ctx context.Context, chunkDir, hashHex string, data []byte) error {
	name, err := objectName(chunkDir, hashHex)
	if err != nil {
		return err
	}
	if len(data) > maxChunkBlob {
		return fmt.Errorf("backup: шифртекст чанка %s весит %d байт, а потолок %d", hashHex, len(data), maxChunkBlob)
	}
	if err := p.ensureCollections(ctx, name); err != nil {
		return err
	}
	addr, err := p.urlFor(name)
	if err != nil {
		return err
	}
	if _, _, err := p.do(ctx, http.MethodPut, addr, data, []int{
		http.StatusCreated, http.StatusOK, http.StatusNoContent,
	}, 0); err != nil {
		return fmt.Errorf("backup: чанк %s не выложен: %w", hashHex, err)
	}
	return nil
}

func (p *WebDAVProvider) DownloadChunk(ctx context.Context, chunkDir, hashHex string) ([]byte, error) {
	name, err := objectName(chunkDir, hashHex)
	if err != nil {
		return nil, err
	}
	addr, err := p.urlFor(name)
	if err != nil {
		return nil, err
	}
	data, resp, err := p.do(ctx, http.MethodGet, addr, nil, []int{http.StatusOK}, maxChunkBlob)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %s (webdav ответил 404)", ErrChunkAbsent, hashHex)
		}
		return nil, fmt.Errorf("backup: чанк %s не скачан: %w", hashHex, err)
	}
	return data, nil
}

func (p *WebDAVProvider) UploadHeader(ctx context.Context, data []byte) error {
	return p.UploadHeaderNamed(ctx, HeaderName, data)
}

func (p *WebDAVProvider) DownloadHeader(ctx context.Context) ([]byte, error) {
	return p.DownloadHeaderNamed(ctx, HeaderName)
}

// UploadHeaderNamed кладёт заголовок по имени объекта (алиас HeaderName или слот
// headers/<ключ>/<id>.json). Имя проходит validHeaderName до сборки адреса: PUT
// обязан уйти внутри подкаталога пакета, а не по пути, который принёс листинг.
func (p *WebDAVProvider) UploadHeaderNamed(ctx context.Context, name string, data []byte) error {
	if err := validHeaderName(name); err != nil {
		return err
	}
	if len(data) > maxHeaderBlob {
		return fmt.Errorf("backup: заголовок %s весит %d байт, а потолок %d", name, len(data), maxHeaderBlob)
	}
	if err := p.ensureCollections(ctx, name); err != nil {
		return err
	}
	addr, err := p.urlFor(name)
	if err != nil {
		return err
	}
	if _, _, err := p.do(ctx, http.MethodPut, addr, data, []int{
		http.StatusCreated, http.StatusOK, http.StatusNoContent,
	}, 0); err != nil {
		return fmt.Errorf("backup: заголовок %s не выложен: %w", name, err)
	}
	return nil
}

func (p *WebDAVProvider) DownloadHeaderNamed(ctx context.Context, name string) ([]byte, error) {
	if err := validHeaderName(name); err != nil {
		return nil, err
	}
	addr, err := p.urlFor(name)
	if err != nil {
		return nil, err
	}
	data, resp, err := p.do(ctx, http.MethodGet, addr, nil, []int{http.StatusOK}, maxHeaderBlob)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("backup: заголовок %s на webdav не найден (404): каталог пакета пуст или цель указана неверно", name)
		}
		return nil, fmt.Errorf("backup: заголовок %s не скачан: %w", name, err)
	}
	return data, nil
}

// DeleteHeaderNamed убирает заголовок. 404 — успех (объекта и так нет): GC
// перечитывается после обрыва, и второе удаление не имеет права быть ошибкой.
func (p *WebDAVProvider) DeleteHeaderNamed(ctx context.Context, name string) error {
	if err := validHeaderName(name); err != nil {
		return err
	}
	return p.deleteObject(ctx, name)
}

func (p *WebDAVProvider) DeleteChunk(ctx context.Context, chunkDir, hashHex string) error {
	name, err := objectName(chunkDir, hashHex)
	if err != nil {
		return err
	}
	return p.deleteObject(ctx, name)
}

// deleteObject — DELETE по имени объекта пакета. Годные статусы включают 404:
// удаление отсутствующего объекта — это «уже чисто», а не отказ. Редиректы
// отвергаются тем же CheckRedirect, что и остальные методы: вместе с Location на
// чужой хост уехал бы Basic-пароль, и для DELETE это верно не меньше, чем для PUT.
func (p *WebDAVProvider) deleteObject(ctx context.Context, name string) error {
	addr, err := p.urlFor(name)
	if err != nil {
		return err
	}
	if _, _, err := p.do(ctx, http.MethodDelete, addr, nil, []int{
		http.StatusNoContent, http.StatusOK, http.StatusNotFound, http.StatusMultiStatus,
	}, 0); err != nil {
		return fmt.Errorf("backup: %s не удалён: %w", name, err)
	}
	return nil
}

// ListHeaderNames собирает имена заголовков: алиас header.json (PROPFIND по нему
// самому) и каждый слот (PROPFIND Depth: infinity по headers/). Два запроса, а не
// один по корню, потому что корневой infinity перечислил бы и все чанки — за
// список заголовков это стоило бы обхода всего хранилища.
//
// Отсутствующий header.json (404) — это «алиаса нет», а ошибка в headers/ — нет:
// каталог слотов мог не создаться (первый пакет), но испорченный ответ обязан
// остаться ошибкой, иначе ротация решит, что пакетов нет, и оставит старые чанки
// сиротами вместо того, чтобы сказать о недоступности хранилища.
func (p *WebDAVProvider) ListHeaderNames(ctx context.Context) ([]string, error) {
	out := make([]string, 0, 8)

	aliasAddr, err := p.urlFor(HeaderName)
	if err != nil {
		return nil, err
	}
	aliasBody := []byte(`<?xml version="1.0" encoding="utf-8"?>` +
		`<d:propfind xmlns:d="DAV:"><d:prop><d:resourcetype/></d:prop></d:propfind>`)
	aliasData, resp, err := p.do(ctx, "PROPFIND", aliasAddr, aliasBody, []int{207, http.StatusOK}, 1<<20)
	switch {
	case err == nil:
		names, perr := p.parsePropfindNames(aliasData)
		if perr != nil {
			return nil, fmt.Errorf("backup: список заголовков на webdav не разобран: %w", perr)
		}
		for _, n := range names {
			if n == HeaderName {
				out = append(out, HeaderName)
			}
		}
	case resp != nil && resp.StatusCode == http.StatusNotFound:
		// Алиаса нет — законно для пакета, снятого ротацией, где header.json
		// убрали, а слоты остались.
	default:
		return nil, fmt.Errorf("backup: алиас заголовка на webdav не перечисляется: %w", err)
	}

	dirAddr, err := p.urlFor(headersRootName)
	if err != nil {
		return nil, err
	}
	dirBody := []byte(`<?xml version="1.0" encoding="utf-8"?>` +
		`<d:propfind xmlns:d="DAV:"><d:prop><d:resourcetype/></d:prop></d:propfind>`)
	dirData, resp, err := p.do(ctx, "PROPFIND", dirAddr, dirBody, []int{207, http.StatusOK}, 8<<20)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// Каталога слотов нет — пакетов, снятых ротацией, тоже нет.
			return out, nil
		}
		return nil, fmt.Errorf("backup: каталог слотов заголовков на webdav не получен: %w", err)
	}
	names, err := p.parsePropfindNames(dirData)
	if err != nil {
		return nil, fmt.Errorf("backup: список слотов заголовков на webdav не разобран: %w", err)
	}
	for _, n := range names {
		if _, _, ok := parseHeaderSlot(n); ok {
			out = append(out, n)
		}
	}
	return out, nil
}

// parsePropfindNames превращает 207 Multi-Status в имена объектов пакета,
// отнесённые от подкаталога пакета: «/base/chunks/…/x.bin» → «chunks/…/x.bin»,
// «/base/headers/<dir>/<id>.json» → «headers/<dir>/<id>.json». Коллекции (href с
// завершающим слэшем) пропускаются: это каталоги, а не заголовки.
func (p *WebDAVProvider) parsePropfindNames(data []byte) ([]string, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("backup: PROPFIND ответил пустым телом: 207 без multistatus — это не список, а непонятный ответ")
	}
	var parsed propfindResult
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("ответ PROPFIND не разбирается как multistatus: %w", err)
	}
	basePath := p.base.Path
	names := make([]string, 0, len(parsed.Responses))
	for _, r := range parsed.Responses {
		href := r.Href
		if u, err := url.Parse(href); err == nil && (u.IsAbs() || strings.HasPrefix(href, "/")) {
			href = u.Path
		}
		decoded, err := url.PathUnescape(href)
		if err != nil {
			return nil, fmt.Errorf("href %q из PROPFIND не декодируется: %w", r.Href, err)
		}
		if strings.HasSuffix(decoded, "/") {
			continue // коллекция, не объект
		}
		rel := strings.TrimPrefix(decoded, basePath)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
			continue // вне подкаталога пакета — чужой ответ, молча пропускаем
		}
		names = append(names, rel)
	}
	return names, nil
}

// propfindResult — разбор 207 Multi-Status. Пространство имён DAV: в структуре
// не пишется намеренно: encoding/xml сопоставляет по локальным именам, поэтому
// разбор не зависит от префикса («d:», «D:», без префикса).
type propfindResult struct {
	XMLName   xml.Name `xml:"multistatus"`
	Responses []struct {
		Href     string `xml:"href"`
		Propstat []struct {
			Prop struct {
				ContentLength string `xml:"getcontentlength"`
				ResourceType  struct {
					Collection string `xml:"collection"`
				} `xml:"resourcetype"`
			} `xml:"prop"`
		} `xml:"propstat"`
	} `xml:"response"`
}

// ExistingChunks — PROPFIND Depth: infinity по каталогу ключа. Глубина
// бесконечна осознанно: чанки лежат в шардах по две буквы, и Depth: 1 вернул бы
// список каталогов вместо файлов, то есть пустое множество — ровно ту ложь,
// которую исправляет этот шаг. Сервер, который не умеет infinity, обязан
// ответить ошибкой, а не пустым списком.
func (p *WebDAVProvider) ExistingChunks(ctx context.Context, chunkDir string) (map[string]uint32, error) {
	if err := validChunkDirName(chunkDir); err != nil {
		return nil, err
	}
	name := chunksRootName + "/" + chunkDir
	addr, err := p.urlFor(name)
	if err != nil {
		return nil, err
	}
	body := []byte(`<?xml version="1.0" encoding="utf-8"?>` +
		`<d:propfind xmlns:d="DAV:"><d:prop><d:getcontentlength/><d:resourcetype/></d:prop></d:propfind>`)
	data, resp, err := p.do(ctx, "PROPFIND", addr, body, []int{207, http.StatusOK}, 8<<20)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// Каталога ключа нет — это первый пакет. Именно этот случай и только
			// он имеет право дать пустое множество.
			return map[string]uint32{}, nil
		}
		return nil, fmt.Errorf("backup: список чанков на webdav не получен: %w", err)
	}
	return parseMultistatus(data)
}

// parseMultistatus разбирает ответ PROPFIND в карту «хэш чанка -> размер
// шифртекста». Вынесено из метода, чтобы разбор можно было сверять и с ответом
// эталонного сервера, и с рукописным XML: живые облака отвечают каждый по-своему.
func parseMultistatus(data []byte) (map[string]uint32, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("backup: PROPFIND ответил пустым телом: 207 без multistatus — это не список, а непонятный ответ")
	}
	var parsed propfindResult
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("backup: ответ PROPFIND не разбирается как multistatus: %w", err)
	}
	existing := make(map[string]uint32)
	for _, r := range parsed.Responses {
		base := r.Href
		if u, err := url.Parse(base); err == nil && (u.IsAbs() || strings.HasPrefix(base, "/")) {
			base = u.Path
		}
		base = strings.TrimSuffix(base, "/")
		if at := strings.LastIndex(base, "/"); at >= 0 {
			base = base[at+1:]
		}
		decoded, err := url.PathUnescape(base)
		if err != nil {
			return nil, fmt.Errorf("backup: href %q из PROPFIND не декодируется: %w", r.Href, err)
		}
		hash, ok := chunkHashFromName(decoded)
		if !ok {
			continue
		}
		var size *uint32
		for _, ps := range r.Propstat {
			if ps.Prop.ContentLength == "" {
				continue
			}
			n, err := parseContentLength(ps.Prop.ContentLength)
			if err != nil {
				return nil, fmt.Errorf("backup: getcontentlength %q у чанка %s: %w", ps.Prop.ContentLength, hash, err)
			}
			size = &n
			break
		}
		if size == nil {
			// Без длины файл считается незнакомым, а не «нулевым»: нуль уехал бы
			// в манифест и соврал о размере шифртекста. Такой чанк будет выложен
			// заново теми же байтами — дороже, но честно.
			continue
		}
		existing[hash] = *size
	}
	return existing, nil
}

func parseContentLength(raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.New("пустое значение")
	}
	var n uint64
	if _, err := fmt.Sscanf(raw, "%d", &n); err != nil {
		return 0, fmt.Errorf("не число")
	}
	if n > uint64(maxChunkBlob) {
		return 0, fmt.Errorf("%d байт вне потолка шифртекста %d", n, maxChunkBlob)
	}
	return uint32(n), nil
}

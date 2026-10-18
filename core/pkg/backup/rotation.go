// File: core/pkg/backup/rotation.go
//
// Шаг 15: в одном хранилище живёт несколько пакетов, повторный пакет переиспользует
// чанки предыдущего, а ротация убирает за собой. До этого шага хранилище имело
// ровно один слот заголовка (header.json) и каждый CreateBundleToProvider брал
// новый salt. Следствия были два, и оба назывались в описании пакета как «чего нет»:
//
//   - второй пакет перезаписывал header.json, то есть стирал единственную запись о
//     первом, а его чанки оставались в хранилище сиротами: восстановить первую
//     точку было уже нечем;
//   - каталог чанков выводится из ключа, ключ — из (passphrase, salt), а salt новый
//     ⇒ каталог новый ⇒ дедупликация между пакетами не работала в принципе:
//     «инкрементальный бэкап» означал «полный пакет заново, в другой подкаталог».
//
// Что здесь решено и почему:
//
//   - Заголовок каждого пакета лежит в собственном слоте
//     headers/<ключ>/<id>.json, а header.json стал алиасом самого свежего. Старые
//     точки восстановления перестают быть перезаписанными, а тот, кто ничего не
//     знает о слотах (прежний код, любой читатель по имени), по-прежнему находит
//     актуальный пакет по старому имени.
//   - Хранилище, созданное прежним кодом (один header.json, ни одного слота),
//     перед первым новым пакетом получает слот для лежащей в алиасе записи — иначе
//     первый же Шаг 15 бэкап стёр бы прежнюю точку восстановления вместе с
//     алиасом. Переносится только то, что открывается ключом этого движка; чужую
//     или неразбираемую запись под чужим именем этот код подписывать не вправе, и
//     такой алиас перезаписывается, как перезаписывался всегда.
//   - Salt и параметры KDF нового пакета берутся у самого свежего существующего
//     (ReuseBundleKey). Это и есть переиспользование чанков: тот же ключ ⇒ тот же
//     каталог ⇒ ExistingChunks видит выложенное. Цена названа честно: все
//     хранимые пакеты одной passphrase расшифровываются одним ключом, поэтому
//     защитой остаётся сама passphrase, а не соль. Соляное разнообразие здесь не
//     защищает: вывод Argon2id от salt не «заменяет» секрет, он только разводит
//     одинаковые passphrase по разным ключам, а здесь passphrase одна и та же по
//     замыслу.
//   - Сборщик мусора (PruneBundles) удаляет чанки только по манифесту, который
//     вскрылся данной passphrase, и только в каталоге, который вывелся из этого
//     же ключа и совпал с именем слота. Всё остальное пропускается с названной
//     причиной. Поэтому «удаляем по догадке» в этом коде невозможно по построению:
//     чужой каталог (другая passphrase, другой ключ) не может быть вычислен,
//     значит не может быть и удалён.
//   - Удаление идёт чанками вперёд, слот последним. Порядок выбран по
//     сходимости, а не по красоте: оборванный обход оставляет пакет, который
//     по-прежнему перечисляется и чей манифест читается, поэтому повторный заход
//     доводит дело до конца (удаление идемпотентно). Если бы сначала исчез слот,
//     осиротевшие чанки стало бы нечем перечислить, и их место осталось бы занято
//     навсегда.
package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// ErrNoBundles — в хранилище нет ни одной записи пакета. Называется и алиас, и
// слоты, потому что вызывающий (и человек, и тест) должен понимать, что именно
// он искал: «пакета нет» и «хранилище не ответило» — разные отказы, а второй
// обязан быть ошибкой провайдера, а не этим пустым ответом.
var ErrNoBundles = errors.New("backup: в хранилище нет ни одного пакета: ни алиаса header.json, ни слотов headers/<ключ>/<id>.json")

// Bundle — одна запись о пакете в хранилище.
//
// Slot — имя объекта, из которого прочитан заголовок: алиас HeaderName или слот.
// ChunkDir пуст у алиаса (в его имени каталога нет) — и именно поэтому GC
// вычисляет каталог из ключа и сверяет с именем слота, а не наоборот.
type Bundle struct {
	Slot     string
	ChunkDir string
	BackupID uint64
	Header   *Header

	// raw — байты заголовка как есть: ротация переставляет алиас на самого
	// свежего именно ими, а не пересобранными, иначе «алиас = копия слота»
	// держалась бы на том, что json.Marshal детерминирован.
	raw []byte
}

// ListBundles перечисляет пакеты хранилища от нового к старому.
//
// Разбор отделён от удаления нарочно: «что в хранилище есть» вызывающий обязан
// уметь получить и при чужой passphrase (в ней нет секрета, нужного для открытой
// части заголовка), а вот «что можно удалить» — только с passphrase.
//
// Прочитанный, но неразбираемый заголовок — ошибка, а не пропущенный элемент.
// Молча проигнорированная запись означала бы, что ротация посчитала пакетов
// меньше, чем их лежит, и удалила чанки, на которые оставшийся пакет ещё ссылается.
func ListBundles(provider CloudProvider) ([]Bundle, error) {
	store, err := AsBundleStore(provider)
	if err != nil {
		return nil, err
	}
	ctx := providerContext()
	names, err := store.ListHeaderNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("backup: заголовки пакетов в %s не перечисляются: %w", provider.Name(), err)
	}

	byID := make(map[uint64]Bundle, len(names))
	order := make([]uint64, 0, len(names))
	for _, name := range names {
		dir, slotID, isSlot := parseHeaderSlot(name)
		if !isSlot && name != HeaderName {
			// Чужое имя в дереве заголовков — не запись пакета (так же листинг
			// чанков пропускает файлы, не похожие на чанк).
			continue
		}
		data, err := store.DownloadHeaderNamed(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("backup: запись пакета %s не читается: %w", name, err)
		}
		h, err := parseHeader(data)
		if err != nil {
			return nil, fmt.Errorf("backup: запись пакета %s испорчена: %w", name, err)
		}

		chunkDir := ""
		if isSlot {
			// Имя слота и id внутри обязаны совпасть: по имени GC решает, какой
			// каталог чанков принадлежит этому пакету, а по содержимому — какие
			// чанки в нём лежат. Расхождение означало бы, что одно из двух врёт.
			if slotID != h.BackupID {
				return nil, fmt.Errorf("backup: слот %s называет пакет %d, а внутри него %d: какой каталог чанков этот пакет сторожит — неизвестно",
					name, slotID, h.BackupID)
			}
			chunkDir = dir
		}
		b := Bundle{Slot: name, ChunkDir: chunkDir, BackupID: h.BackupID, Header: h, raw: data}

		prev, seen := byID[h.BackupID]
		switch {
		case !seen:
			order = append(order, h.BackupID)
		case prev.ChunkDir != "" && chunkDir != "":
			return nil, fmt.Errorf("backup: пакет %d лежит в двух слотах (%s и %s): выбирать один из них ротация не вправе",
				h.BackupID, prev.Slot, b.Slot)
		case prev.ChunkDir != "":
			// Алиас того же пакета: слот авторитетнее (алиас перезаписывается, слот
			// — нет), поэтому оставляем прочитанное ранее.
			continue
		}
		byID[h.BackupID] = b
	}

	out := make([]Bundle, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	// CreatedAt с точностью до секунды: два пакета, созданные в одной секунде,
	// различаются id (он считается из UnixNano), поэтому вторичный ключ обязателен,
	// иначе порядок двух соседних пакетов зависел бы от порядка листинга.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Header.CreatedAt != out[j].Header.CreatedAt {
			return out[i].Header.CreatedAt > out[j].Header.CreatedAt
		}
		return out[i].BackupID > out[j].BackupID
	})
	return out, nil
}

// NewestBundle — самый свежий пакет хранилища: то, что читает восстановление и
// у чего ReuseBundleKey берёт ключ.
func NewestBundle(provider CloudProvider) (Bundle, error) {
	bundles, err := ListBundles(provider)
	if err != nil {
		return Bundle{}, err
	}
	if len(bundles) == 0 {
		return Bundle{}, fmt.Errorf("%w (%s)", ErrNoBundles, provider.Name())
	}
	return bundles[0], nil
}

// WriteBundle кладёт заголовок построенного пакета: сначала в слот, потом в алиас
// header.json. Порядок — единственный осмысленный: алиас — то, по чему пакет
// начинают искать, и он обязан указывать на запись, которая уже существует.
//
// Две записи одного содержимого — это не дублирование данных: слот неизменяем, а
// алиас — указатель, который ротация передвигает.
func (e *SnapshotEngine) WriteBundle(h *Header) error {
	if h == nil {
		return errors.New("backup: пустой заголовок")
	}
	if h.BackupID != e.backupID {
		return fmt.Errorf("backup: заголовок называет пакет %d, а этот движок строит %d: слот был бы назван id, который внутри не живёт",
			h.BackupID, e.backupID)
	}
	store, err := AsBundleStore(e.provider)
	if err != nil {
		return err
	}
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	slot, err := headerSlotName(e.chunkDir, h.BackupID)
	if err != nil {
		return err
	}
	ctx := providerContext()
	if err := e.migrateAliasSlot(ctx, store); err != nil {
		return err
	}
	if err := store.UploadHeaderNamed(ctx, slot, data); err != nil {
		return fmt.Errorf("backup: слот пакета %s не выложен: %w", slot, err)
	}
	if err := store.UploadHeaderNamed(ctx, HeaderName, data); err != nil {
		return fmt.Errorf("backup: алиас %s не обновлён (слот %s уже лежит): %w", HeaderName, slot, err)
	}
	return nil
}

// migrateAliasSlot даёт слот пакету, который лежит в хранилище только под
// алиасом header.json. Нужно это ровно для одного перехода: хранилище, созданное
// прежним кодом (один заголовок, ни одного слота), при первом Шаге 15 бэкапе
// потеряло бы свою единственную запись о прежней точке восстановления вместе с
// алиасом — то есть повторный пакет снова стёр бы первый, ради отмены чего Шаг 15
// и затеян.
//
// Переносится только то, что открывается ключом этого движка. Имя слота — это
// каталог чанков, а его выводит только та passphrase, при которой solt заголовка
// даёт этот же ключ; раскрытый манифест доказывает, что salt алиаса совпал, а
// значит e.chunkDir и есть его настоящий каталог. Нераскрытый алиас (другая
// passphrase или испорченный заголовок) под чужим именем не подписывается:
// ротация потом не смогла бы отличить его чанки от чужих. Такой алиас остаётся
// перезаписанным — как и до Шага 15, и это названо в README.
//
// Ошибка переноса прерывает запись пакета: положить новый слот и переставить
// алиас, потеряв прежнюю запись, значило бы сделать то самое, от чего этот код
// бережёт.
func (e *SnapshotEngine) migrateAliasSlot(ctx context.Context, store BundleStore) error {
	data, err := store.DownloadHeaderNamed(ctx, HeaderName)
	if err != nil {
		return nil // алиаса нет — переносить нечего
	}
	h, err := parseHeader(data)
	if err != nil {
		return nil
	}
	if h.BackupID == e.backupID {
		return nil // этот же пакет: его слот сейчас перезапишется целиком
	}
	if _, err := manifestChunksWithKeys(e.keys, h); err != nil {
		return nil
	}
	slot, err := headerSlotName(e.chunkDir, h.BackupID)
	if err != nil {
		return err
	}
	if err := store.UploadHeaderNamed(ctx, slot, data); err != nil {
		return fmt.Errorf("backup: прежняя запись %s не перенесена в слот %s, поэтому алиас не перезаписывается: %w",
			HeaderName, slot, err)
	}
	return nil
}

// ReuseBundleKey — salt и параметры KDF, которые обязан взять новый пакет, чтобы
// его чанки легли в тот же каталог, что уже заполнен: дедупликация между пакетами
// существует ровно в этом объёме.
//
// Пакетов нет — новый salt и производственные параметры (reused=false). Самая
// свежая запись не читается — ошибка, а не «возьмём новый ключ»: с новым ключом
// пакет лег бы в новый каталог и выложил бы всё содержимое заново, а вызывающий
// узнал бы об этом только по размеру хранилища.
func ReuseBundleKey(provider CloudProvider) (salt []byte, params KDFParams, reused bool, err error) {
	newest, err := NewestBundle(provider)
	if err != nil {
		if !errors.Is(err, ErrNoBundles) {
			return nil, KDFParams{}, false, err
		}
		fresh, serr := newSalt()
		if serr != nil {
			return nil, KDFParams{}, false, serr
		}
		return fresh, ProductionKDF, false, nil
	}
	return append([]byte(nil), newest.Header.Salt...),
		KDFParams{Time: newest.Header.Time, MemoryKiB: newest.Header.MemoryKiB, Threads: newest.Header.Threads},
		true, nil
}

// PruneSkip — почему пакет остался в хранилище, хотя ротация до него дошла.
type PruneSkip struct {
	Slot   string
	Reason string
}

// PruneReport — что ротация сделала и чего не сделала. Причина у каждого пропуска
// названа: «молча удалила меньше» и «сказала, почему не удалила» — это разные
// обещания, и второе проверяется тестом.
type PruneReport struct {
	Bundles       int // записей пакета в хранилище до ротации
	Kept          int // осталось записей после
	Dropped       int
	ChunksDeleted int
	BytesFreed    uint64
	Skipped       []PruneSkip
	AliasSlot     string // слот, на который теперь указывает header.json
}

// PruneBundles оставляет keep самых свежих пакетов, а за ними убирает и их слоты,
// и чанки, на которые не ссылается ни один оставшийся пакет.
//
// Правила, каждое из которых проверяется тестом:
//
//   - keep < 1 отвергается: «удалить все пакеты» не значит «очистить хранилище»,
//     это значит «снять все точки восстановления», и такой отказ обязан быть
//     назван, а не исполнен;
//   - keep >= числа пакетов — чистый no-op (ни одного обращения на запись):
//     вызывающий, который зовёт ротацию после каждого бэкапа, не обязан считать,
//     доросло ли хранилище до удаления;
//   - самый свежий пакет не удаляется никогда, даже если его манифест не читается
//     (он мог быть написан другой passphrase, и тогда он чужой, но живой);
//   - удаляются только чанки каталога, вычисленного из (passphrase, salt
//     удаляемого заголовка) и совпавшего с именем слота. Ни до одного чужого
//     каталога этот код не дотянется по построению: чтобы назвать его, нужна
//     passphrase;
//   - если в каталоге остаётся пакет, чей манифест не вскрыт, весь каталог
//     защищается: его хэши неизвестны, значит неизвестно и то, что можно удалить.
//
// Ошибка прерывает обход (отчёт с частичными итогами возвращается вместе с ней):
// хранилище, которое перестало отвечать на удалении, не повод продолжать и решить
// за него.
func PruneBundles(passphrase string, provider CloudProvider, keep int) (PruneReport, error) {
	var rep PruneReport
	store, err := AsBundleStore(provider)
	if err != nil {
		return rep, err
	}
	if keep < 1 {
		return rep, fmt.Errorf("backup: keep=%d: ротация обязана оставлять хотя бы один пакет, иначе удалёнными окажутся все точки восстановления", keep)
	}
	bundles, err := ListBundles(provider)
	if err != nil {
		return rep, err
	}
	rep.Bundles = len(bundles)
	if len(bundles) == 0 {
		return rep, nil
	}
	if keep >= len(bundles) {
		rep.Kept = len(bundles)
		rep.AliasSlot = bundles[0].Slot
		return rep, nil
	}

	ctx := providerContext()
	retained, doomed := bundles[:keep], bundles[keep:]
	refs := make(map[string]map[string]bool)
	protected := make(map[string]bool)

	// Пропуск с защитой каталога: dir здесь — любое из двух имён каталога,
	// которые у этого пакета могут быть (у слота и у ключа), потому что при их
	// расхождении непонятно, какой настоящий, а рисковать нельзя ни одним.
	skip := func(b Bundle, dir, why string) {
		rep.Skipped = append(rep.Skipped, PruneSkip{Slot: b.Slot, Reason: why})
		if b.ChunkDir != "" {
			protected[b.ChunkDir] = true
		}
		if dir != "" {
			protected[dir] = true
		}
	}

	for _, b := range retained {
		_, dir, chunks, err := bundleKeyAndManifest(passphrase, b.Header)
		if err != nil {
			skip(b, dir, fmt.Sprintf("оставленный пакет %d не вскрыт этой passphrase (%s): его каталог чанков защищён целиком", b.BackupID, err))
			continue
		}
		if b.ChunkDir != "" && b.ChunkDir != dir {
			skip(b, dir, fmt.Sprintf("оставленный пакет %d: слот называет каталог %s, ключ выводит %s — чанки этого пакета неотличимы от чужих",
				b.BackupID, b.ChunkDir, dir))
			continue
		}
		set := refs[dir]
		if set == nil {
			set = make(map[string]bool)
			refs[dir] = set
		}
		for _, d := range chunks {
			set[d.ChunkHash] = true
		}
	}

	// Обход от старых к новым. Порядок важен ровно в одном случае: старый пакет,
	// который не вскрылся, оставляет свой каталог под защитой, и новый пакет того
	// же каталога обязан быть пропущен, а не обработан первым (обработанный первым
	// удалил бы чанки, на которые старый ещё ссылается, и оставил бы его битым).
	deleted := make(map[string]bool)
	for i := len(doomed) - 1; i >= 0; i-- {
		b := doomed[i]
		_, dir, chunks, err := bundleKeyAndManifest(passphrase, b.Header)
		if err != nil {
			skip(b, dir, fmt.Sprintf("манифест пакета %d не вскрывается этой passphrase (%s): удалять его чанки нечем", b.BackupID, err))
			continue
		}
		if b.ChunkDir != "" && b.ChunkDir != dir {
			skip(b, dir, fmt.Sprintf("имя слота %q называет каталог %s, а ключ пакета %d выводит %s: по такому имени удаление началось бы с догадки",
				b.Slot, b.ChunkDir, b.BackupID, dir))
			continue
		}
		if protected[dir] {
			skip(b, dir, fmt.Sprintf("в каталоге %s остался пакет, чьи чанки этой passphrase не перечислить: каталог целиком не трогать", dir))
			continue
		}
		for _, d := range chunks {
			if refs[dir][d.ChunkHash] {
				continue // на него ссылается оставленный пакет
			}
			key := dir + "/" + d.ChunkHash
			if deleted[key] {
				continue // общий чанк двух удаляемых пакетов: считается один раз
			}
			if err := store.DeleteChunk(ctx, dir, d.ChunkHash); err != nil {
				return rep, fmt.Errorf("backup: чанк %s пакета %d не удалён: %w", d.ChunkHash, b.BackupID, err)
			}
			deleted[key] = true
			rep.ChunksDeleted++
			rep.BytesFreed += uint64(d.EncryptedSize)
		}
		if err := store.DeleteHeaderNamed(ctx, b.Slot); err != nil {
			return rep, fmt.Errorf("backup: запись пакета %d (%s) не удалена: %w", b.BackupID, b.Slot, err)
		}
		rep.Dropped++
	}

	// Алиас — на самого свежего из оставшихся. Он им и был бы, если бы запись о
	// новом пакете никогда не прерывалась, но ровно этого здесь гарантировать
	// нельзя: этот шаг и создан для случая «алиас отстал от содержимого».
	newest := bundles[0]
	if rep.Dropped > 0 && newest.Slot != HeaderName {
		if err := store.UploadHeaderNamed(ctx, HeaderName, newest.raw); err != nil {
			return rep, fmt.Errorf("backup: алиас %s не переставлен на пакет %d: %w", HeaderName, newest.BackupID, err)
		}
	}
	rep.Kept = rep.Bundles - rep.Dropped
	rep.AliasSlot = newest.Slot
	return rep, nil
}

// bundleKeyAndManifest — ключ этого заголовка при данной passphrase, каталог
// чанков, который из ключа выводится, и список чанков из манифеста.
//
// Каталог возвращается и когда манифест не вскрылся (ключ-то выведен): вызывающий
// обязан защитить каталог от удаления и в этом случае, а не остаться без имени,
// которое можно внести в список защищённых. Ошибка при выводе ключа даёт пустой
// dir — защищать нечего, чужого каталога этот код не увидит.
func bundleKeyAndManifest(passphrase string, h *Header) (keySet, string, []ChunkDescriptor, error) {
	if h == nil {
		return keySet{}, "", nil, errors.New("backup: пустой заголовок")
	}
	keys, err := deriveKeys(passphrase, h.Salt, KDFParams{Time: h.Time, MemoryKiB: h.MemoryKiB, Threads: h.Threads})
	if err != nil {
		return keySet{}, "", nil, err
	}
	dir := chunkDirName(keys)
	chunks, err := manifestChunksWithKeys(keys, h)
	if err != nil {
		return keys, dir, nil, err
	}
	return keys, dir, chunks, nil
}

// manifestChunksWithKeys — список чанков заголовка, раскрытый уже выведенным
// ключом. Отдельно от bundleKeyAndManifest потому, что у движка ключ есть, а
// passphrase нет: она не хранится в движке намеренно, и перенос прежней записи в
// слот не обязан её вызывать.
//
// Проверки здесь — те же, что делает Restore (подпись, раскрытие, согласованность
// числа чанков и обоих копий id), плюс каноничность каждого имени: GC собирает по
// ним путь удаления, и одно «..» в манифесте означало бы удаление файла вне
// хранилища пакета.
func manifestChunksWithKeys(keys keySet, h *Header) ([]ChunkDescriptor, error) {
	if err := checkManifestMAC(&keys.mac, h.Manifest, h.ManifestMAC); err != nil {
		return nil, err
	}
	raw, err := openChunk(&keys.enc, h.Manifest, []byte(manifestAAD))
	if err != nil {
		return nil, fmt.Errorf("манифест не раскрылся: %w", err)
	}
	var m BackupManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("манифест разобран криво: %w", err)
	}
	if m.BackupID != h.BackupID {
		return nil, fmt.Errorf("манифест называет пакет %d, открытый заголовок — %d", m.BackupID, h.BackupID)
	}
	if len(m.Chunks) == 0 {
		return nil, errors.New("backup: в манифесте нет ни одного чанка")
	}
	if uint32(len(m.Chunks)) != h.TotalChunks {
		return nil, fmt.Errorf("манифест перечисляет %d чанков, заголовок обещает %d", len(m.Chunks), h.TotalChunks)
	}
	for _, d := range m.Chunks {
		if err := validChunkHash(d.ChunkHash); err != nil {
			return nil, err
		}
	}
	return m.Chunks, nil
}

// File: core/pkg/backup/snapshot_engine.go
//
// pkg/backup — криптография и раскладка снимка состояния. С Шага 13 у пакета
// есть вызывающий: core/pkg/rpc/backup_service.go исполняет команды
// CreateBackup и RestoreBackup, собирает protobuf-ответ BackupResp и публикует
// события backup_progress. Достижимость машинная: тест
// TestUnreachablePackagesMatchRegistry (core/pkg/rpc/architecture_test.go)
// падает, если цепочка импортов от cmd/* до pkg/backup снова пропадёт.
//
// Что в пакете выполняется кодом, а не описанием:
//   - ключ выводится из passphrase по Argon2id (параметры и salt лежат в
//     незашифрованном заголовке), а не передаётся сырыми 32 байтами;
//   - чанки и манифест — XChaCha20-Poly1305, манифест дополнительно подписан
//     HMAC-SHA256 по шифртексту (encrypt-then-MAC: проверять подпись до
//     расшифровки, а не после);
//   - есть сборка файла из чанков (RestoreBundle) — раньше в пакете не
//     существовало вообще ничего, что читает чанк обратно;
//   - хэши чанков складываются в merkle-дерево, корень лежит и в манифесте, и в
//     заголовке, а при сборке сверяются обе копии с пересчитанной;
//   - дедупликация смотрит в хранилище (провайдер), а не только в память
//     процесса, поэтому перезапущенный движок не выкладывает тот же чанк заново;
//   - в одном хранилище живёт несколько пакетов: заголовок каждого лежит в
//     собственном слоте headers/<ключ>/<id>.json, а header.json — алиас самого
//     свежего (rotation.go). Salt нового пакета берётся у самого свежего
//     (ReuseBundleKey), поэтому чанки между пакетами переиспользуются, а не
//     укладываются заново под новым ключом;
//   - ротация (PruneBundles) убирает слоты устаревших пакетов и те их чанки, на
//     которые не ссылается ни один оставшийся пакет, и переставляет алиас.
//
// Чего в пакете по-прежнему нет (и тесты этого не проверяют, потому что
// проверять нечего):
//   - сжатие осталось zlib/DEFLATE: заявленный в описаниях Zstd — отдельная
//     зависимость, и имя алгоритма в заголовке честно говорит «zlib»;
//   - из облачных схем подключена одна — WebDAV (webdav.go). Google Drive,
//     Yandex.Disk и LAN P2P Sync остались в providers.go поимённо, но каждый их
//     метод отвечает названным отказом, а OpenProvider отвергает их схемы;
//     живые API этих трёх сервисов не трогались ни разу, поэтому сверять их
//     клиент нечем. Клиент WebDAV мерился против эталонного сервера
//     golang.org/x/net/webdav в этом процессе — см. webdav_test.go;
//   - ключ пакета выводится из passphrase, а не из MasterIdentity.BackupMasterKey,
//     который pkg/crypto считает HKDF из seed по домену
//     «ygggram-backup-master-key-v1». Это решение, а не недосмотр: в бандле
//     лежат файлы, которые сами по себе зашифрованы ключом хранилища из seed,
//     так что для чтения почты нужны и passphrase пакета, и фраза кошелька.
//     BackupMasterKey в этой цепи не участвует и пока никем не читается.
package backup

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	MinChunkSize    = 64 * 1024          // 64 KB
	TargetChunkSize = 256 * 1024         // 256 KB
	MaxChunkSize    = 1024 * 1024        // 1 MB
	FastCDCMaskS    = 0x0000d90f03530000 // Cutpoint mask for small chunks
	FastCDCMaskL    = 0x0000d90303530000 // Cutpoint mask for larger chunks

	// HeaderName — имя файла незашифрованного заголовка пакета. Открытым
	// текстом он обязан быть потому, что без salt и параметров KDF ключ не
	// вывести вообще: восстановление на другом устройстве начинается с него.
	HeaderName = "header.json"

	headerMagic = "ygggram-backup-v1"

	// chunkDirLabel — доменная строка для имени каталога чанков (см.
	// chunkDirName).
	chunkDirLabel = "ygggram-backup-chunks-v1"
)

// ChunkDescriptor identifies an encrypted deduplicated slice
type ChunkDescriptor struct {
	ChunkHash     string `json:"hash"`
	OriginalSize  uint32 `json:"orig_size"`
	EncryptedSize uint32 `json:"enc_size"`
}

// BackupManifest describes the encrypted snapshot hierarchy. Живёт внутри
// зашифрованного блока Header.Manifest: порядок чанков и их количество — то,
// что не должен уметь менять тот, кто держит файл.
type BackupManifest struct {
	BackupID        uint64            `json:"backup_id"`
	CreatedAt       int64             `json:"created_at"`
	TotalPlainBytes uint64            `json:"total_plain_bytes"`
	TotalChunkBytes uint64            `json:"total_chunk_bytes"`
	RootMerkleHash  []byte            `json:"root_merkle_hash"`
	Chunks          []ChunkDescriptor `json:"chunks"`
}

// Header — незашифрованная часть пакета: salt и параметры KDF, имена
// алгоритмов, зашифрованный манифест и его HMAC.
//
// BackupID, CreatedAt и RootMerkleHash дублируются из зашифрованного манифеста
// открытым текстом, и вот зачем. Без id вызывающий не связал бы события прогресса
// с пакетом: он нужен ещё до чтения манифеста. Без root и created_at он не
// заполнил бы protobuf-ответ (CoreResponse_Backup) при восстановлении, где
// содержимое манифеста доступно только внутри Restore. Расхождение двух копий
// исключено: и в манифест, и в заголовок значения пишутся из одного места в Seal.
type Header struct {
	Magic          string `json:"magic"`
	Kdf            string `json:"kdf"`
	Salt           []byte `json:"argon2id_salt"`
	Time           uint32 `json:"kdf_time"`
	MemoryKiB      uint32 `json:"kdf_memory_kib"`
	Threads        uint8  `json:"kdf_threads"`
	Compression    string `json:"compression_algo"`
	Encryption     string `json:"encryption_algo"`
	Manifest       []byte `json:"manifest"`
	ManifestMAC    []byte `json:"manifest_mac"`
	BackupID       uint64 `json:"backup_id"`
	CreatedAt      int64  `json:"created_at"`
	RootMerkleHash []byte `json:"root_merkle_hash"`
	TotalChunks    uint32 `json:"total_chunks"`
	TotalPlain     uint64 `json:"total_plain_bytes"`
	TotalEncrypted uint64 `json:"total_encrypted_bytes"`
}

// SnapshotEngine укладывает байты снимка в чанки и собирает заголовок пакета.
// Хранилище ему даёт вызывающий через CloudProvider: с Шага 14 тот же код пишет
// пакет и в каталог на диске, и в облако по WebDAV, и ни одна из проверок
// (подпись манифеста, merkle-корень, контент-хэш каждого чанка) от выбора
// хранилища не зависит.
type SnapshotEngine struct {
	mu     sync.RWMutex
	salt   []byte
	params KDFParams
	keys   keySet

	// provider — куда кладутся чанки и заголовок. chunkDir — имя каталога чанков,
	// привязанное к ключу (см. chunkDirName): оно известно только здесь, потому
	// что ключ знает движок, а провайдер не должен его угадывать.
	provider CloudProvider
	chunkDir string

	// known — дедупликация: хэш чанка -> размер шифртекста в хранилище.
	// Наполняется списком провайдера при создании движка, поэтому перезапуска
	// «памяти процесса» для этого больше не нужно.
	known map[string]uint32

	// uploaded и reused — счётчики последней раскладки: сколько чанков реально
	// выложено и сколько уже лежали в хранилище. Нужны вызывающему, чтобы отчёт
	// об «инкрементальном» бэкапе называл число, а не намерение.
	uploaded uint32
	reused   uint32

	// backupID и createdAt принадлежат этому движку, то есть одному пакету:
	// они попадают и в зашифрованный манифест, и в открытый заголовок, и в
	// события прогресса вызывающего.
	backupID  uint64
	createdAt int64

	// reusedKey — salt и параметры взяты у уже лежащего в хранилище пакета, то
	// есть чанки этого движка лягут в тот же каталог и дедупликация сработает
	// между пакетами. Ставится только NewSnapshotEngineToProvider.
	reusedKey bool
}

// NewSnapshotEngine создаёт движок для НОВОГО пакета в локальном каталоге: salt
// генерируется, параметры KDF — производственные. Ключ из passphrase, а не
// «принесите 32 байта»: тот, кто делает бэкап, знает пароль, а не ключ.
func NewSnapshotEngine(passphrase, storageDir string) (*SnapshotEngine, error) {
	provider, err := NewLocalProvider(storageDir)
	if err != nil {
		return nil, err
	}
	salt, err := newSalt()
	if err != nil {
		return nil, err
	}
	return NewSnapshotEngineWithProvider(passphrase, provider, salt, ProductionKDF)
}

// NewSnapshotEngineWithKDF — движок с указанными salt и параметрами в локальном
// каталоге: им пользуются восстановление (значения берутся из заголовка) и тесты
// формата, которым нужен дешёвый KDF.
func NewSnapshotEngineWithKDF(passphrase, storageDir string, salt []byte, params KDFParams) (*SnapshotEngine, error) {
	provider, err := NewLocalProvider(storageDir)
	if err != nil {
		return nil, err
	}
	return NewSnapshotEngineWithProvider(passphrase, provider, salt, params)
}

// NewSnapshotEngineToProvider создаёт движок НОВОГО пакета в произвольном
// хранилище: это тот же путь, что NewSnapshotEngine, но хранилище выбирает
// вызывающий, а не «каталог на диске».
//
// Ключ берётся у самого свежего пакета этого хранилища (ReuseBundleKey), а не
// генерируется заново: новый salt означал бы новый каталог чанков, а вместе с ним
// и весь пакет выложенным заново, то есть «инкрементальный бэкап» был бы ложью.
// Пакетов в хранилище нет — salt новый, параметры производственные.
func NewSnapshotEngineToProvider(passphrase string, provider CloudProvider) (*SnapshotEngine, error) {
	salt, params, reused, err := ReuseBundleKey(provider)
	if err != nil {
		return nil, err
	}
	e, err := NewSnapshotEngineWithProvider(passphrase, provider, salt, params)
	if err != nil {
		return nil, err
	}
	e.reusedKey = reused
	return e, nil
}

// NewSnapshotEngineWithProvider — единственный настоящий конструктор: остальные
// — обёртки над локальным каталогом.
//
// Дедупликация запрашивается у провайдера здесь, а не на первом чанке: отказ
// провайдера перечислить содержимое обязан появиться до того, как хоть один байт
// уехал, иначе движок молча перевыложил бы весь пакет поверх чужого.
func NewSnapshotEngineWithProvider(passphrase string, provider CloudProvider, salt []byte, params KDFParams) (*SnapshotEngine, error) {
	if provider == nil {
		return nil, errors.New("backup: провайдер не задан: пакету нужно хранилище")
	}
	keys, err := deriveKeys(passphrase, salt, params)
	if err != nil {
		return nil, err
	}
	dirName := chunkDirName(keys)

	known, err := provider.ExistingChunks(providerContext(), dirName)
	if err != nil {
		return nil, fmt.Errorf("backup: провайдер %s не перечислил свои чанки: %w", provider.Name(), err)
	}
	if known == nil {
		known = make(map[string]uint32)
	}

	saltCopy := make([]byte, len(salt))
	copy(saltCopy, salt)

	// Один движок — один пакет, поэтому id и время создаются здесь, а не в Seal:
	// вызывающему id нужен на первое же событие прогресса, до раскладки чанков.
	created := time.Now()

	return &SnapshotEngine{
		salt:      saltCopy,
		params:    params,
		keys:      keys,
		provider:  provider,
		chunkDir:  dirName,
		known:     known,
		backupID:  uint64(created.UnixNano()),
		createdAt: created.Unix(),
	}, nil
}

// Provider — хранилище, в которое этот движок кладёт пакет. Нужен вызывающему,
// чтобы назвать в ответе тот storage_provider, куда чанки реально легли.
func (e *SnapshotEngine) Provider() CloudProvider { return e.provider }

// localChunkDir — путь каталога чанков, если пакет лежит на диске. Нужен тестам,
// которые правят файлы чанков в обход провайдера: подмена байта в облаке без
// такого пути проверялась бы только через сам клиент, а это круг.
func (e *SnapshotEngine) localChunkDir() (string, error) {
	p, ok := e.provider.(*localProvider)
	if !ok {
		return "", fmt.Errorf("backup: провайдер %s не локальный каталог, пути к чанкам у него нет", e.provider.Name())
	}
	return p.dirFor(e.chunkDir)
}

// BackupID — идентификатор пакета, который строит этот движок. Он же лежит в
// зашифрованном манифесте и в открытом заголовке, поэтому события прогресса и
// protobuf-ответ называют одно и то же.
func (e *SnapshotEngine) BackupID() uint64 { return e.backupID }

// ChunkDir — имя каталога чанков этого пакета. Нужен вызывающему, чтобы назвать
// в отчёте, где именно лежат файлы, не выводя ключ заново.
func (e *SnapshotEngine) ChunkDir() string { return e.chunkDir }

// ReusedKey — правда ли, что ключ (и salt, и параметры KDF) взят у уже лежащего в
// хранилище пакета. Если false, движок строит первый пакет: чанкам ещё негде
// переиспользоваться, и «дедуп между пакетами» в отчёте было бы ложью.
func (e *SnapshotEngine) ReusedKey() bool { return e.reusedKey }

// DedupStats — сколько чанков этой раскладки уже лежали в хранилище (reused) и
// сколько для неё выложено (uploaded). Это единственный честный ответ на вопрос
// «бэкап-то инкрементальный?»: по одному лишь TotalChunks невозможно отличить
// пакет, повторивший предыдущий байт в байт, от пакета, который весь выслан
// заново в новый каталог.
func (e *SnapshotEngine) DedupStats() (uploaded, reused uint32) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.uploaded, e.reused
}

// -----------------------------------------------------------------------------
// FASTCDC (CONTENT-DEFINED CHUNKING)
// -----------------------------------------------------------------------------

// FastCDCSplit partitions arbitrary stream data into variable-size deduplicated chunks
func FastCDCSplit(data []byte) [][]byte {
	n := len(data)
	if n <= MinChunkSize {
		return [][]byte{data}
	}

	var chunks [][]byte
	cursor := 0

	for cursor < n {
		remaining := n - cursor
		if remaining <= MinChunkSize {
			chunks = append(chunks, data[cursor:])
			break
		}

		maxBoundary := cursor + MaxChunkSize
		if maxBoundary > n {
			maxBoundary = n
		}

		splitPos := maxBoundary
		fingerprint := uint64(0)
		targetLimit := cursor + TargetChunkSize
		if targetLimit > maxBoundary {
			targetLimit = maxBoundary
		}

		// Phase 1: Small mask scanning
		i := cursor + MinChunkSize
		for ; i < targetLimit; i++ {
			fingerprint = (fingerprint << 1) + uint64(data[i])
			if (fingerprint & FastCDCMaskS) == 0 {
				splitPos = i
				break
			}
		}

		// Phase 2: Large mask scanning if not cut in Phase 1
		if splitPos == maxBoundary {
			for ; i < maxBoundary; i++ {
				fingerprint = (fingerprint << 1) + uint64(data[i])
				if (fingerprint & FastCDCMaskL) == 0 {
					splitPos = i
					break
				}
			}
		}

		chunks = append(chunks, data[cursor:splitPos])
		cursor = splitPos
	}

	return chunks
}

// -----------------------------------------------------------------------------
// COMPRESSION & ENCRYPTION
// -----------------------------------------------------------------------------

func compressChunk(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressChunk(compressed []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func (e *SnapshotEngine) encryptChunk(compressed []byte, hashHex string) ([]byte, error) {
	return sealChunk(&e.keys.enc, compressed, []byte(hashHex))
}

func (e *SnapshotEngine) decryptChunk(encData []byte, hashHex string) ([]byte, error) {
	compressed, err := openChunk(&e.keys.enc, encData, []byte(hashHex))
	if err != nil {
		return nil, err
	}
	return decompressChunk(compressed)
}

// -----------------------------------------------------------------------------
// SNAPSHOT CREATION & MANIFEST PACKING
// -----------------------------------------------------------------------------

// chunkDirName — имя каталога чанков, привязанное к ключу. Без этого два пакета
// в одном хранилище (одна passphrase, разные salt) переиспользовали бы файлы друг
// друга по контент-хэшу, и восстановление второго читало бы чанки, зашифрованные
// первым ключом. Имя — усечённый хэш ключа, а не кусок самого ключа: каталог
// видит и провайдер, и любой, кто листит хранилище.
func chunkDirName(keys keySet) string {
	sum := sha256.Sum256(append([]byte(chunkDirLabel), keys.enc[:]...))
	return hex.EncodeToString(sum[:4])
}

// providerContext — родитель для всех обращений движка к провайдеру. Потолок
// времени одной операции находится в самом провайдере (webdav.go: по одному
// defaultProviderTimeout на запрос), а не здесь: ограничивать здесь весь проход
// означало бы отменять бэкап на половине выложенных чанков.
func providerContext() context.Context { return context.Background() }

// ProcessFileIntoChunks раскладывает байты на чанки, отбрасывает те, что
// провайдер уже хранит, и возвращает дескрипторы пакета.
func (e *SnapshotEngine) ProcessFileIntoChunks(rawBytes []byte) ([]ChunkDescriptor, uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ctx := providerContext()

	rawChunks := FastCDCSplit(rawBytes)
	descriptors := make([]ChunkDescriptor, 0, len(rawChunks))
	var totalChunkBytes uint64

	for _, rawChunk := range rawChunks {
		h := sha256.Sum256(rawChunk)
		hashHex := hex.EncodeToString(h[:])

		// Дедупликация обязана видеть хранилище: прежняя память процесса
		// «забывала» выложенные чанки после перезапуска, и тот же контент
		// укладывался вторым файлом.
		encSize, exists := e.known[hashHex]

		if !exists {
			compressed, err := compressChunk(rawChunk)
			if err != nil {
				return nil, 0, err
			}
			encrypted, err := e.encryptChunk(compressed, hashHex)
			if err != nil {
				return nil, 0, err
			}
			if err := e.provider.UploadChunk(ctx, e.chunkDir, hashHex, encrypted); err != nil {
				return nil, 0, err
			}
			e.uploaded++
			encSize = uint32(len(encrypted))
			e.known[hashHex] = encSize
		} else {
			e.reused++
		}

		totalChunkBytes += uint64(encSize)
		descriptors = append(descriptors, ChunkDescriptor{
			ChunkHash:     hashHex,
			OriginalSize:  uint32(len(rawChunk)),
			EncryptedSize: encSize,
		})
	}

	return descriptors, totalChunkBytes, nil
}

// Seal собирает манифест, шифрует его и подписывает. Возвращает готовый
// заголовок пакета, который ещё не лежит на диске — запись отдельная, чтобы
// вызывающий мог выложить сначала чанки, потом заголовок.
func (e *SnapshotEngine) Seal(descriptors []ChunkDescriptor, totalPlainBytes, totalChunkBytes uint64) (*Header, error) {
	if len(descriptors) == 0 {
		return nil, errors.New("backup: пустой список чанков")
	}

	hashes := make([][]byte, 0, len(descriptors))
	var encryptedSum uint64
	for _, d := range descriptors {
		raw, err := hex.DecodeString(d.ChunkHash)
		if err != nil || len(raw) != sha256.Size {
			return nil, fmt.Errorf("backup: хэш чанка %q не годится", d.ChunkHash)
		}
		hashes = append(hashes, raw)
		encryptedSum += uint64(d.EncryptedSize)
	}

	root, err := MerkleRoot(hashes)
	if err != nil {
		return nil, err
	}

	manifest := BackupManifest{
		BackupID:        e.backupID,
		CreatedAt:       e.createdAt,
		TotalPlainBytes: totalPlainBytes,
		TotalChunkBytes: totalChunkBytes,
		RootMerkleHash:  root,
		Chunks:          descriptors,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	encManifest, err := sealChunk(&e.keys.enc, manifestBytes, []byte(manifestAAD))
	if err != nil {
		return nil, err
	}

	return &Header{
		Magic:          headerMagic,
		Kdf:            KdfName,
		Salt:           append([]byte(nil), e.salt...),
		Time:           e.params.Time,
		MemoryKiB:      e.params.MemoryKiB,
		Threads:        e.params.Threads,
		Compression:    CompressionName,
		Encryption:     EncryptionName,
		Manifest:       encManifest,
		ManifestMAC:    macManifest(&e.keys.mac, encManifest),
		BackupID:       e.backupID,
		CreatedAt:      e.createdAt,
		RootMerkleHash: root,
		TotalChunks:    uint32(len(descriptors)),
		TotalPlain:     totalPlainBytes,
		TotalEncrypted: encryptedSum,
	}, nil
}

// ReadChunk читает шифртекст чанка из хранилища движка.
func (e *SnapshotEngine) ReadChunk(hashHex string) ([]byte, error) {
	return e.provider.DownloadChunk(providerContext(), e.chunkDir, hashHex)
}

// Restore раскручивает пакет обратно: подпись манифеста, расшифровка манифеста,
// merkle-корень, затем каждый чанк — расшифровка, распаковка и сверка
// контент-хэша. Порча любого байта в любом чанке обязана быть ошибкой, а не
// «молча обрезанным файлом».
func (e *SnapshotEngine) Restore(h *Header) ([]byte, error) {
	if h == nil {
		return nil, errors.New("backup: пустой заголовок")
	}
	if h.Magic != headerMagic {
		return nil, fmt.Errorf("backup: заголовок с магией %q нам не знаком", h.Magic)
	}
	if h.Kdf != KdfName || h.Encryption != EncryptionName || h.Compression != CompressionName {
		return nil, fmt.Errorf("backup: заголовок обещает %s/%s/%s, а код умеет %s/%s/%s",
			h.Kdf, h.Encryption, h.Compression, KdfName, EncryptionName, CompressionName)
	}
	if h.Time != e.params.Time || h.MemoryKiB != e.params.MemoryKiB || h.Threads != e.params.Threads {
		return nil, fmt.Errorf("backup: движок построен с параметрами %+v, а заголовок требует %+v/%d/%d",
			e.params, h.Time, h.MemoryKiB, h.Threads)
	}

	if err := checkManifestMAC(&e.keys.mac, h.Manifest, h.ManifestMAC); err != nil {
		return nil, err
	}
	manifestBytes, err := openChunk(&e.keys.enc, h.Manifest, []byte(manifestAAD))
	if err != nil {
		return nil, fmt.Errorf("backup: манифест не читается: %w", err)
	}

	var manifest BackupManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("backup: манифест разобран криво: %w", err)
	}
	if len(manifest.Chunks) == 0 {
		return nil, errors.New("backup: в манифесте нет ни одного чанка")
	}
	if uint32(len(manifest.Chunks)) != h.TotalChunks {
		return nil, fmt.Errorf("backup: манифест перечисляет %d чанков, заголовок обещает %d",
			len(manifest.Chunks), h.TotalChunks)
	}
	if manifest.TotalPlainBytes != h.TotalPlain {
		return nil, fmt.Errorf("backup: объём открытых байт в манифесте (%d) не совпадает с заголовком (%d)",
			manifest.TotalPlainBytes, h.TotalPlain)
	}

	out := make([]byte, 0, manifest.TotalPlainBytes)
	hashes := make([][]byte, 0, len(manifest.Chunks))
	for _, d := range manifest.Chunks {
		blob, err := e.ReadChunk(d.ChunkHash)
		if err != nil {
			if errors.Is(err, ErrChunkAbsent) {
				return nil, fmt.Errorf("backup: чанк %s лежит не в хранилище (%s): %w",
					d.ChunkHash, e.provider.Name(), err)
			}
			return nil, fmt.Errorf("backup: чанк %s не читается: %w", d.ChunkHash, err)
		}
		compressed, err := openChunk(&e.keys.enc, blob, []byte(d.ChunkHash))
		if err != nil {
			return nil, err
		}
		raw, err := decompressChunk(compressed)
		if err != nil {
			return nil, fmt.Errorf("backup: чанк %s не распаковывается: %w", d.ChunkHash, err)
		}

		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != d.ChunkHash {
			return nil, fmt.Errorf("backup: чанк %s раскрылся содержимым с другим хэшем", d.ChunkHash)
		}
		if uint32(len(raw)) != d.OriginalSize {
			return nil, fmt.Errorf("backup: чанк %s длиной %d, манифест ждёт %d", d.ChunkHash, len(raw), d.OriginalSize)
		}

		hashes = append(hashes, sum[:])
		out = append(out, raw...)
	}

	root, err := MerkleRoot(hashes)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(root, manifest.RootMerkleHash) {
		return nil, errors.New("backup: merkle-корень не сходится: чанки идут не в том порядке или часть потеряна")
	}
	// Открытый заголовок и подписанный манифест обязаны называть один и тот же
	// корень: иначе вызывающий, который читает заголовок (он не умеет
	// расшифровывать), отдаст клиенту другое значение, чем сторожат чанки.
	if !bytes.Equal(root, h.RootMerkleHash) {
		return nil, errors.New("backup: merkle-корень в открытом заголовке не совпадает с манифестом")
	}
	if uint64(len(out)) != manifest.TotalPlainBytes {
		return nil, fmt.Errorf("backup: собрано %d байт, ожидалось %d", len(out), manifest.TotalPlainBytes)
	}
	return out, nil
}

// -----------------------------------------------------------------------------
// ПАКЕТ В ХРАНИЛИЩЕ
// -----------------------------------------------------------------------------

// WriteHeaderTo кладёт заголовок в хранилище. Отдельная от Seal операция нужна
// ровно поэтому: чанки обязаны выложиться раньше, чем появится файл, по которому
// их начнут искать.
//
// Пишет только алиас самого свежего, то есть header.json, и потому годится там,
// где пакету не нужен собственный слот (тесты формата, перекладка готового
// заголовка). Живой бэкап вызывает SnapshotEngine.WriteBundle: он кладёт слот, а
// потом алиас, иначе второй пакет стёр бы запись о первом.
func WriteHeaderTo(provider CloudProvider, h *Header) error {
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	if err := provider.UploadHeader(providerContext(), data); err != nil {
		return fmt.Errorf("backup: заголовок пакета не выложен в %s: %w", provider.Name(), err)
	}
	return nil
}

// ReadHeaderFrom читает и проверяет незашифрованную часть пакета из хранилища.
// Имя фиксированное — алиас header.json; если в хранилище несколько пакетов и
// нужен самый свежий именно как запись ротации, зови NewestBundle.
func ReadHeaderFrom(provider CloudProvider) (*Header, error) {
	data, err := provider.DownloadHeader(providerContext())
	if err != nil {
		return nil, fmt.Errorf("backup: заголовок пакета в %s не читается: %w", provider.Name(), err)
	}
	return parseHeader(data)
}

// parseHeader разбирает и проверяет открытую часть. Разбор отделён от способа
// доставки, чтобы проверка «заголовок не врёт» была одна на диск и облако.
func parseHeader(data []byte) (*Header, error) {
	h := &Header{}
	if err := json.Unmarshal(data, h); err != nil {
		return nil, fmt.Errorf("backup: заголовок разобран криво: %w", err)
	}
	if h.Magic != headerMagic {
		return nil, fmt.Errorf("backup: заголовок с магией %q нам не знаком", h.Magic)
	}
	if len(h.Salt) != SaltSize {
		return nil, fmt.Errorf("backup: salt в заголовке %d байт, а нужно %d", len(h.Salt), SaltSize)
	}
	// Открытая часть не подписана ничем, кроме совпадения с манифестом, поэтому
	// хотя бы требует осмысленных значений: пустой корень или id нулевой стоят
	// ровно «заголовок исказили», и молча передавать такое вызывающему нельзя.
	if h.BackupID == 0 {
		return nil, errors.New("backup: в заголовке нулевой id пакета")
	}
	if len(h.RootMerkleHash) != sha256.Size {
		return nil, fmt.Errorf("backup: merkle-корень в заголовке %d байт, а нужно %d",
			len(h.RootMerkleHash), sha256.Size)
	}
	if h.TotalChunks == 0 {
		return nil, errors.New("backup: заголовок обещает ноль чанков")
	}
	return h, nil
}

// WriteHeader кладёт заголовок в локальный каталог пакета.
func WriteHeader(dir string, h *Header) error {
	provider, err := NewLocalProvider(dir)
	if err != nil {
		return err
	}
	return WriteHeaderTo(provider, h)
}

// ReadHeader читает незашифрованную часть пакета из локального каталога.
func ReadHeader(dir string) (*Header, error) {
	provider, err := NewLocalProvider(dir)
	if err != nil {
		return nil, err
	}
	return ReadHeaderFrom(provider)
}

// CreateBundle — полный путь «есть байты → есть пакет» в локальном каталоге:
// чанки, манифест, заголовок на диске. Возвращает заголовок, чтобы вызывающий мог
// переложить его в облако отдельно.
func CreateBundle(passphrase, dir string, payload []byte) (*Header, error) {
	provider, err := NewLocalProvider(dir)
	if err != nil {
		return nil, err
	}
	return CreateBundleToProvider(passphrase, provider, payload)
}

// CreateBundleToProvider — тот же путь, но хранилище выбирает вызывающий.
//
// С Шага 15 ключ берётся не «новый salt на каждый пакет», а у самого свежего
// пакета этого хранилища (NewSnapshotEngineToProvider → ReuseBundleKey): без этого
// дедупликация между пакетами невозможна, а «инкрементальный бэкап» означал бы
// полный пакет заново в другой подкаталог. Заголовок пишется через
// engine.WriteBundle, то есть в собственный слот и в алиас, а не поверх
// единственного header.json: прежний путь стирал запись о предыдущей точке
// восстановления.
func CreateBundleToProvider(passphrase string, provider CloudProvider, payload []byte) (*Header, error) {
	e, err := NewSnapshotEngineToProvider(passphrase, provider)
	if err != nil {
		return nil, err
	}

	descriptors, totalChunkBytes, err := e.ProcessFileIntoChunks(payload)
	if err != nil {
		return nil, err
	}
	h, err := e.Seal(descriptors, uint64(len(payload)), totalChunkBytes)
	if err != nil {
		return nil, err
	}
	if err := e.WriteBundle(h); err != nil {
		return nil, err
	}
	return h, nil
}

// RestoreBundle — восстановление в другом процессе и на другом устройстве из
// локального каталога.
func RestoreBundle(passphrase, dir string) ([]byte, error) {
	provider, err := NewLocalProvider(dir)
	if err != nil {
		return nil, err
	}
	return RestoreBundleFromProvider(passphrase, provider)
}

// RestoreBundleFromProvider — то же, но пакет берётся из любого провайдера: из
// заголовка читаются salt и параметры KDF, ключ выводится заново из passphrase,
// дальше идёт полная сверка каждого чанка.
//
// Берётся самый свежий пакет хранилища (NewestBundle), а не «файл с фиксированным
// именем»: с Шага 15 записей о пакетах несколько, и восстановление по умолчанию
// обязано читать последнюю точку, а не ту, чей слот оказался раньше в листинге.
func RestoreBundleFromProvider(passphrase string, provider CloudProvider) ([]byte, error) {
	newest, err := NewestBundle(provider)
	if err != nil {
		return nil, err
	}
	e, err := NewSnapshotEngineWithProvider(passphrase, provider, newest.Header.Salt, KDFParams{
		Time:      newest.Header.Time,
		MemoryKiB: newest.Header.MemoryKiB,
		Threads:   newest.Header.Threads,
	})
	if err != nil {
		return nil, err
	}
	return e.Restore(newest.Header)
}

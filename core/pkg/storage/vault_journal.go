// File: core/pkg/storage/vault_journal.go
//
// Дневник хранилища (Шаг 10). Прежний код на каждую запись переписывал весь
// ygggram_vault.bin: это O(размер базы) на сообщение, и база в несколько тысяч
// диалогов становилась непригодной — об этом прямо писалось в AGENTS.md. Теперь
// база лежит снапшотом плюс append-only дневник: запись дописывает один кадр, а
// полный снимок собирается только когда дневник догоняет снапшот по размеру.
// Средняя стоимость записи перестаёт зависеть от размера базы, а открытие
// по-прежнему читает всё подряд — но один раз на запуск, а не на сообщение.
//
// Почему не bbolt: правило «только Pure-Go базы» соблюдается и так, а замена
// движка — это миграция формата живых баз и другой профиль шифрования (ключ
// ложится на весь файл, а не на запись). Это отдельная задача, и молча подменять
// её здесь нельзя.
//
// Порядок записи и восстановление после аварии. Кадр дневника вскрывается своим
// AEAD, поэтому битый хвост отличается от годной записи и не читается как мусор:
//  1. удаление сначала ложится кадром в ygggram_vault.jrl и лишь потом убирается
//     из памяти — иначе память осталась бы без записи, которую диск ещё хранит,
//     и после перезапуска она воскресла бы (для чужой почты S&F это значило бы
//     «письмо вернулось адресату дважды»);
//  2. set-запись пишется в том же порядке, но здесь отказ дописи безопаснее:
//     строка живёт в памяти, вызывающий получает ошибку, а ближайшая компакция
//     всё равно унесёт её на диск. Инвариант держится в одну сторону — «диск не
//     хранит того, чего нет в памяти», — и обратного он не обещает;
//  3. компакция пишет новый снапшот (tmp, sync, rename) с уже включёнными в него
//     записями дневника и с номером journal_seq;
//  4. только после этого дневник переименовывается в .old-<seq> и удаляется, а
//     следующий кадр создаст пустой файл заново.
//
// Любая авария между этими шагами оставляет состояние, которое восстанавливается
// однозначно: снапшот всегда старше или равен дневнику, поэтому при открытии
// переигрываются только кадры с seq больше snapshot.journal_seq, а осиротевший
// файл ротации пропускается по тому же правилу.
package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	// Имена файлов в dataDir. Старое имя снапшота сохранено намеренно: база,
	// созданная прежним кодом, открывается без миграции.
	//
	// Названия экспортированы, потому что их читает вызывающий извне: бэкап
	// (core/pkg/rpc/backup_service.go) забирает ровно эти файлы, а восстановление
	// принимает только их. Список живёт здесь, а не в pkg/rpc: файл, которого тут
	// нет, в снимок состояния не попадёт — и тест
	// TestBackupCoversEveryVaultFile удерживает это соответствие.
	VaultSnapshotFile = "ygggram_vault.bin"
	VaultJournalFile  = "ygggram_vault.jrl"

	// AAD снапшота — тот же, что был до Шага 10: иначе старые базы не откроются.
	snapshotRecordID = "ygggram-db-state"
	// Домен ключа дневника. Ключ не совпадает с ключом снапшота, поэтому
	// столкнуть nonce двух слоёв при одном мастере нельзя в принципе.
	journalKeyDomain = "ygggram-vault-journal-v1"
	// Nonce кадра = 4 байта домена + 8 байт seq (12 байт — стандартный размер
	// GCM). Seq увеличивается ровно на единицу перед каждой записью, так что
	// повтор nonce в пределах ключа невозможен по построению, а не «маловероятен».
	journalNonceDomain = "YGGJ"
	// Потолок одного кадра: за ним стоит либо повреждение, либо чужой файл, и ни
	// то, ни другое нельзя принимать за запись молча.
	journalFrameMax = 8 << 20
	// Нижний порог компакции: пока дневник меньше и этого числа, и размера
	// снапшота, полная перезапись невыгодна.
	journalCompactFloor = 64 << 10

	journalOpSet = "set"
	journalOpDel = "del"
)

// Таблицы, для которых дневник хранит записи. messages и crdt_ops поддерживают
// только set: удаление сообщения в этой базе — tombstone-флаг, а не стёртая
// строка, и притворяться иначе дневник не вправе.
const (
	tableMessages = "messages"
	tableCRDTOps  = "crdt_ops"
	tableOutbound = "outbound"
	tableSessions = "sessions"
	tableContacts = "contacts"
	tableSF       = "sf"
	tablePrekeys  = "prekeys"
)

var errVaultClosed = errors.New("db: хранилище закрыто")

// journalEntry — одна запись дневника. Value — JSON самой сущности, а не обёртка:
// хранилище не знает форматов pkg/rpc и переписывает байты дословно.
type journalEntry struct {
	Seq   uint64          `json:"seq"`
	Op    string          `json:"op"`
	Table string          `json:"table"`
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value,omitempty"`
}

// vaultSnapshot — содержимое ygggram_vault.bin. Порядок срезов детерминирован
// (см. mapToSlice и outboundToSlice), иначе каждая компакция писала бы файл, не
// равный прежнему, а сравнение снимков в тестах превращалось бы в шум.
type vaultSnapshot struct {
	Messages   []StoredMessage   `json:"messages"`
	CRDTOps    []CRDTOperation   `json:"crdt_ops"`
	Outbound   []StoredOutbound  `json:"outbound"`
	Sessions   map[string][]byte `json:"sessions"`
	Contacts   map[string][]byte `json:"contacts"`
	Sf         []StoredSF        `json:"sf"`
	PrekeySet  []*StoredPrekeys  `json:"prekeys"`
	JournalSeq uint64            `json:"journal_seq,omitempty"`
}

// openVaults — блокировка «один экземпляр хранилища на каталог в процессе».
// Дневник ведёт seq в памяти экземпляра, поэтому второй живой экземпляр на тот же
// каталог писал бы кадры с теми же номерами, и открытие отбросило бы часть из них
// как устаревшие. Молча потерять почту хуже, чем не открыться.
var (
	openVaultsMu sync.Mutex
	openVaults   = map[string]bool{}
)

func vaultDirKey(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("db: каталог %s не приводится к абсолютному пути: %w", dir, err)
	}
	key := filepath.Clean(abs)
	// Windows считает регистр букв в пути незначимым, Linux — значимым.
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key, nil
}

func claimVaultDir(dir string) (string, error) {
	key, err := vaultDirKey(dir)
	if err != nil {
		return "", err
	}
	openVaultsMu.Lock()
	defer openVaultsMu.Unlock()
	if _, busy := openVaults[key]; busy {
		return "", fmt.Errorf("db: каталог %s уже открыт другим экземпляром хранилища в этом процессе: у дневника обязан быть один владелец, иначе записи с одинаковыми номерами будут отброшены при открытии", key)
	}
	openVaults[key] = true
	return key, nil
}

func releaseVaultDir(key string) {
	openVaultsMu.Lock()
	defer openVaultsMu.Unlock()
	delete(openVaults, key)
}

// deriveJournalKey — подключ мастер-ключа по домену (HMAC-SHA256). Отдельный ключ
// нужен, чтобы кадры дневника не вскрывались ключом снапшота и наоборот.
func deriveJournalKey(master [32]byte) [32]byte {
	m := hmac.New(sha256.New, master[:])
	m.Write([]byte(journalKeyDomain))
	var out [32]byte
	copy(out[:], m.Sum(nil))
	return out
}

// newJournalAEAD — AES-256-GCM на подключе дневника. Тот же алгоритм, что у
// снапшота, но другой ключ: два слоя одного каталога не должны вскрываться
// шифром друг друга, а «шифрование диска» здесь означает не «один ключ на весь
// файл», потому что файл дописывается по записям.
func newJournalAEAD(master [32]byte) (cipher.AEAD, error) {
	key := deriveJournalKey(master)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("db: ключ дневника не принимается: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("db: AEAD дневника не собирается: %w", err)
	}
	return aead, nil
}

func (db *EncryptedDB) journalNonce(seq uint64) []byte {
	nonce := make([]byte, 12)
	copy(nonce, journalNonceDomain)
	binary.BigEndian.PutUint64(nonce[len(journalNonceDomain):], seq)
	return nonce
}

func (db *EncryptedDB) journalAAD(seq uint64) []byte {
	aad := make([]byte, 8)
	binary.BigEndian.PutUint64(aad, seq)
	return aad
}

// journalSetLocked дописывает в дневник запись table/key со значением value.
func (db *EncryptedDB) journalSetLocked(table, key string, value any) error {
	if !db.open {
		return errVaultClosed
	}
	if key == "" {
		return fmt.Errorf("db: запись в таблицу %s с пустым ключом не дописывается", table)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("db: значение %s/%s не сериализуется: %w", table, key, err)
	}
	return db.appendFrameLocked(journalEntry{
		Seq:   db.journalSeq + 1,
		Op:    journalOpSet,
		Table: table,
		Key:   key,
		Value: raw,
	})
}

// journalDelLocked дописывает удаление. Для messages и crdt_ops его нет — см.
// комментарий у констант таблиц.
func (db *EncryptedDB) journalDelLocked(table, key string) error {
	if !db.open {
		return errVaultClosed
	}
	if key == "" {
		return fmt.Errorf("db: удаление из %s с пустым ключом не дописывается", table)
	}
	switch table {
	case tableMessages, tableCRDTOps:
		return fmt.Errorf("db: таблица %s не удаляет записей: удаление сообщения — это tombstone-флаг, а не стёртая строка", table)
	}
	return db.appendFrameLocked(journalEntry{
		Seq:   db.journalSeq + 1,
		Op:    journalOpDel,
		Table: table,
		Key:   key,
	})
}

func (db *EncryptedDB) appendFrameLocked(e journalEntry) error {
	plain, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("db: кадр дневника %s/%s не сериализуется: %w", e.Table, e.Key, err)
	}
	nonce := db.journalNonce(e.Seq)
	sealed := db.journalAEAD.Seal(nil, nonce, plain, db.journalAAD(e.Seq))
	if len(nonce)+len(sealed) > journalFrameMax {
		return fmt.Errorf("db: кадр дневника %s/%s в %d байт длиннее потолка %d", e.Table, e.Key, len(nonce)+len(sealed), journalFrameMax)
	}

	frame := make([]byte, 4, 4+len(nonce)+len(sealed))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(nonce)+len(sealed)))
	frame = append(frame, nonce...)
	frame = append(frame, sealed...)

	path := filepath.Join(db.dataDir, VaultJournalFile)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("db: дневник %s не открыт на допись: %w", path, err)
	}
	if _, err := f.Write(frame); err != nil {
		_ = f.Close()
		return fmt.Errorf("db: кадр дневника не дописан: %w", err)
	}
	// Без синхронизации «успешная» запись означала бы лишь то, что байты лежат в
	// кэше ОС, тогда как память уже считает её состоявшейся.
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("db: дневник не синхронизирован на диске: %w", err)
	}
	size, statErr := func() (int64, error) {
		st, err := f.Stat()
		if err != nil {
			return 0, err
		}
		return st.Size(), nil
	}()
	if closeErr := f.Close(); closeErr != nil {
		return fmt.Errorf("db: дневник не закрыт после дописи: %w", closeErr)
	}
	if statErr != nil {
		return fmt.Errorf("db: размер дневника после дописи неизвестен: %w", statErr)
	}

	db.journalSeq = e.Seq
	return db.compactLocked(size)
}

// compactLocked меняет дневник на снапшот, когда дневник догоняет снапшот по
// размеру. Порог геометрический: средняя стоимость записи сверху ограничена
// константой, а не растёт линейно с размером базы.
func (db *EncryptedDB) compactLocked(journalSize int64) error {
	threshold := int64(journalCompactFloor)
	if db.snapBytes > threshold {
		threshold = db.snapBytes
	}
	if journalSize < threshold {
		return nil
	}
	if err := db.writeSnapshotLocked(); err != nil {
		return err
	}

	cur := filepath.Join(db.dataDir, VaultJournalFile)
	stale := filepath.Join(db.dataDir, fmt.Sprintf("%s.old-%d", VaultJournalFile, db.journalSeq))
	if err := os.Rename(cur, stale); err != nil {
		return fmt.Errorf("db: дневник не переименован в %s: %w", stale, err)
	}
	db.compactions++
	// Удаление — гигиена, а не часть протокола восстановления: переименованный
	// дневник целиком помещается в только что записанный снапшот, поэтому его
	// остаток безопаснее потерять, чем посчитать нужным. Мусор добирается при
	// следующем открытии.
	if err := os.Remove(stale); err != nil && !os.IsNotExist(err) {
		db.jrnStale++
	}
	return nil
}

// sweepStaleJournals убирает остатки ротации, которые не удалось удалить при
// компакции (например, файл держал другой процесс).
func (db *EncryptedDB) sweepStaleJournals() {
	matches, err := filepath.Glob(filepath.Join(db.dataDir, VaultJournalFile+".old-*"))
	if err != nil {
		return
	}
	for _, name := range matches {
		db.jrnStale++
		_ = os.Remove(name)
	}
}

// openFrame вскрывает кадр дневника. Ошибка — это либо повреждение, либо чужой
// файл: AEAD-тег не даёт подставить в дневник произвольные байты.
func (db *EncryptedDB) openFrame(rec []byte) (journalEntry, error) {
	if len(rec) < 12 {
		return journalEntry{}, errors.New("db: кадр дневника короче nonce")
	}
	nonce := rec[:12]
	if string(nonce[:len(journalNonceDomain)]) != journalNonceDomain {
		return journalEntry{}, errors.New("db: nonce кадра не из нашего дневника")
	}
	seq := binary.BigEndian.Uint64(nonce[len(journalNonceDomain):])
	plain, err := db.journalAEAD.Open(nil, nonce, rec[12:], db.journalAAD(seq))
	if err != nil {
		return journalEntry{}, fmt.Errorf("db: кадр дневника %d не проходит AEAD: %w", seq, err)
	}
	var e journalEntry
	if err := json.Unmarshal(plain, &e); err != nil {
		return journalEntry{}, fmt.Errorf("db: кадр дневника %d не разбирается: %w", seq, err)
	}
	// Номер обязан совпасть с nonce: кадр, вырезанный из середины потока и
	// перенесённый в другое место, должен отваливаться, а не съезжать состоянием.
	if e.Seq != seq {
		return journalEntry{}, fmt.Errorf("db: кадр дневника заявляет seq %d, а лежит под nonce %d", e.Seq, seq)
	}
	return e, nil
}

// replayJournal переигрывает дневник поверх снапшота. Битый хвост отбрасывается с
// обрезкой файла: потеря последнего кадра означает «запись не состоялась», тогда
// как молча принять мусор значило бы применить состояние, которого никто не писал.
func (db *EncryptedDB) replayJournal() error {
	path := filepath.Join(db.dataDir, VaultJournalFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("db: дневник %s не читается: %w", path, err)
	}

	good := 0
	for good < len(data) {
		if len(data)-good < 4 {
			break
		}
		n := int(binary.BigEndian.Uint32(data[good : good+4]))
		if n <= 0 || n > journalFrameMax {
			// На первом кадре это не обрыв: писатель выдаёт длину и тело одним
			// вызовом записи, поэтому оборваться может только хвост последнего
			// кадра. Длина, которой не могло быть, на нулевом смещении означает
			// «файл нам не принадлежит».
			if good == 0 {
				return fmt.Errorf("db: дневник %s начинается с длины кадра %d, которой наш писатель выдать не мог: это не оборванная запись, а чужой или испорченный файл", path, n)
			}
			break
		}
		if len(data)-good-4 < n {
			break
		}
		e, err := db.openFrame(data[good+4 : good+4+n])
		if err != nil {
			// Тот же довод, что и выше: целый первый кадр, который не вскрывается
			// этим ключом, — дневник другой базы. Отрезать его и молча открыть
			// пустое состояние означало бы потерять чужую запись и сделать вид, что
			// восстановления удалось.
			if good == 0 {
				return fmt.Errorf("db: первый кадр дневника %s не вскрывается этим ключом: %w", path, err)
			}
			break
		}
		good += 4 + n
		if e.Seq <= db.journalSeq {
			// Уже внутри снапшота: пропуск, а не ошибка — именно так работает
			// ротация после компакции.
			db.jrnSkipped++
			continue
		}
		if err := db.applyEntryLocked(e); err != nil {
			return fmt.Errorf("db: запись дневника %d (%s/%s) не применяется: %w", e.Seq, e.Table, e.Key, err)
		}
		db.journalSeq = e.Seq
		db.jrnApplied++
	}

	if tail := len(data) - good; tail > 0 {
		db.jrnCorrupt += tail
		if err := os.Truncate(path, int64(good)); err != nil {
			return fmt.Errorf("db: битый хвост дневника (%d байт) не отрезан: %w", tail, err)
		}
	}
	return nil
}

// applyEntryLocked применяет одну запись дневника к памяти. Проверка «ключ
// записи совпадает с ключом внутри сущности» обязательна: иначе дневник мог бы
// переписать чужую строку, а вызывающий никогда не узнал бы, что база расходится с
// тем, что в неё клали.
func (db *EncryptedDB) applyEntryLocked(e journalEntry) error {
	if e.Seq == 0 {
		return errors.New("db: запись дневника без номера")
	}
	switch e.Op {
	case journalOpDel:
		return db.deleteLocked(e.Table, e.Key)
	case journalOpSet:
	default:
		return fmt.Errorf("db: неизвестная операция дневника %q", e.Op)
	}
	if len(e.Value) == 0 {
		return fmt.Errorf("db: set-запись %s/%s без значения", e.Table, e.Key)
	}

	switch e.Table {
	case tableMessages:
		var m StoredMessage
		if err := json.Unmarshal(e.Value, &m); err != nil {
			return err
		}
		if m.MessageID != e.Key {
			return fmt.Errorf("db: ключ %s не совпадает с message_id %s", e.Key, m.MessageID)
		}
		db.storeMessageLocked(&m)
		return nil
	case tableCRDTOps:
		var op CRDTOperation
		if err := json.Unmarshal(e.Value, &op); err != nil {
			return err
		}
		if op.OpID != e.Key {
			return fmt.Errorf("db: ключ %s не совпадает с op_id %s", e.Key, op.OpID)
		}
		db.appendCRDTOpLocked(op)
		return nil
	case tableOutbound:
		var rec StoredOutbound
		if err := json.Unmarshal(e.Value, &rec); err != nil {
			return err
		}
		if rec.MessageID != e.Key {
			return fmt.Errorf("db: ключ %s не совпадает с message_id в очереди", e.Key)
		}
		db.outbound[rec.MessageID] = &rec
		return nil
	case tableSessions:
		var blob []byte
		if err := json.Unmarshal(e.Value, &blob); err != nil {
			return err
		}
		if len(blob) == 0 {
			return fmt.Errorf("db: сеанс %s без состояния", e.Key)
		}
		db.sessions[e.Key] = blob
		return nil
	case tableContacts:
		var blob []byte
		if err := json.Unmarshal(e.Value, &blob); err != nil {
			return err
		}
		if len(blob) == 0 {
			return fmt.Errorf("db: контакт %s без карты", e.Key)
		}
		db.contacts[e.Key] = blob
		return nil
	case tableSF:
		var rec StoredSF
		if err := json.Unmarshal(e.Value, &rec); err != nil {
			return err
		}
		if sfKey(rec.MailboxID, rec.EnvelopeID) != e.Key {
			return fmt.Errorf("db: ключ почты %s не совпадает с её ящиком и содержимым", e.Key)
		}
		db.sf[e.Key] = &rec
		return nil
	case tablePrekeys:
		var pk StoredPrekeys
		if err := json.Unmarshal(e.Value, &pk); err != nil {
			return err
		}
		if prekeyKey(pk.Owner) != e.Key {
			return fmt.Errorf("db: ключ prekey-набора %s не совпадает с его владельцем", e.Key)
		}
		db.prekeys[e.Key] = &pk
		return nil
	default:
		return fmt.Errorf("db: неизвестная таблица дневника %q", e.Table)
	}
}

func (db *EncryptedDB) deleteLocked(table, key string) error {
	switch table {
	case tableOutbound:
		delete(db.outbound, key)
	case tableSessions:
		delete(db.sessions, key)
	case tableContacts:
		delete(db.contacts, key)
	case tableSF:
		delete(db.sf, key)
	case tablePrekeys:
		delete(db.prekeys, key)
	default:
		return fmt.Errorf("db: удаление из таблицы %s не поддерживается", table)
	}
	return nil
}

// JournalStats — счётчики восстановления дневника: сколько записей переиграно,
// сколько пропущено как уже снятые, сколько байт отброшено как битый хвост,
// сколько было компакций и сколько осиротевших файлов ротации найдено.
type JournalStats struct {
	Applied    int
	Skipped    int
	Corrupt    int
	Compaction int
	Stale      int
	JournalSeq uint64
	SnapBytes  int64
}

func (db *EncryptedDB) JournalStats() JournalStats {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return JournalStats{
		Applied:    db.jrnApplied,
		Skipped:    db.jrnSkipped,
		Corrupt:    db.jrnCorrupt,
		Compaction: db.compactions,
		Stale:      db.jrnStale,
		JournalSeq: db.journalSeq,
		SnapBytes:  db.snapBytes,
	}
}

// File: core/pkg/storage/encrypted_db.go
package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RecordType identifies the category of encrypted stored entity
type RecordType byte

const (
	RecordChatMessage    RecordType = 0x01
	RecordRatchetSession RecordType = 0x02
	RecordContact        RecordType = 0x03
	RecordCRDTOp         RecordType = 0x04
)

// HLC represents a Hybrid Logical Clock for causal distributed ordering across devices
type HLC struct {
	PhysicalTime int64  `json:"physical"`
	LogicalCount uint32 `json:"logical"`
	DeviceID     string `json:"device_id"`
}

func (h HLC) Compare(other HLC) int {
	if h.PhysicalTime != other.PhysicalTime {
		if h.PhysicalTime < other.PhysicalTime {
			return -1
		}
		return 1
	}
	if h.LogicalCount != other.LogicalCount {
		if h.LogicalCount < other.LogicalCount {
			return -1
		}
		return 1
	}
	return strings.Compare(h.DeviceID, other.DeviceID)
}

// CRDTOperation encapsulates decentralized state mutations (delete, read status, pins)
type CRDTOperation struct {
	OpID     string `json:"op_id"`
	TargetID string `json:"target_id"` // MessageID or ConversationID
	Type     string `json:"type"`      // "read", "pin", "tombstone_delete", "folder"
	Payload  string `json:"payload"`
	Clock    HLC    `json:"clock"`
}

// StoredMessage represents a stored message item in plaintext before disk encryption
type StoredMessage struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	TimestampMS    int64  `json:"timestamp_ms"`
	Body           string `json:"body"`
	IsOutbound     bool   `json:"is_outbound"`
	IsRead         bool   `json:"is_read"`
	IsPinned       bool   `json:"is_pinned"`
	IsDeleted      bool   `json:"is_deleted"` // CRDT Tombstone
	Clock          HLC    `json:"clock"`
	// DeliveredMS — время сквозного ACK получателя (0, подтверждения нет).
	// Отдельное поле, а не IsRead: «адресат расшифровал» и «пользователь
	// открыл диалог» — разные утверждения, первое даёт только рэтчет.
	DeliveredMS int64 `json:"delivered_ms"`
}

// StoredOutbound — исходящий конверт, который ещё не ушёл в оверлей. Хранятся
// ровно те байты, что были собраны при отправке: повторный seal двигал бы
// отправную цепочку ради пакетов, которые никуда не ушли, и получатель платил
// за это skipped-ключами (ratchet.maxSkipped=2000) там, где достаточно одной
// дедупликации по message_id.
//
// Запись живёт в базе до подтверждения доставки (её снимает CompleteOutbound)
// либо до исчерпания лимита отправок.
// StoredSF — конверт, который этот узел хранит чужому адресату как слепой
// S&F-релей. Байты для хранилища непрозрачны: правомочен ли запросивший,
// решает pkg/rpc.
type StoredSF struct {
	// MailboxID — hex слепому ящику, по которому конверт ищут.
	MailboxID string `json:"mailbox_id"`
	// EnvelopeID — hex содержимого: повторная кладка того же конверта не должна
	// удваивать почту.
	EnvelopeID string `json:"envelope_id"`
	Data       []byte `json:"data"`
	PlacedMS   int64  `json:"placed_ms"`
	ExpiresMS  int64  `json:"expires_ms"`
	// HandedMS — когда письмо в последний раз выдавали адресату. Пакт ответа
	// релей доставляет так же слепо, как и живой конверт: первый пакет на ещё не
	// прогретой сессии оверлей теряет. Списывать почту по первой выдаче
	// означало бы терять её вместе с тем пакетом, поэтому выдача помечает запись,
	// а удаляет её только окно повторной выдачи (решает pkg/rpc).
	HandedMS int64 `json:"handed_ms"`
}

// StoredPrekey — один one-time prekey с идентификатором: тот же ID, что уходит
// в бандле предключей и в конверте X3DH.
type StoredPrekey struct {
	ID   uint32 `json:"id"`
	Priv []byte `json:"priv"`
}

// StoredPrekeys — prekey-материал ядра: signed prekey и склад one-time
// prekeys. Хранилище их не разбирает — ключами оперирует pkg/rpc; смысл здесь
// только один: материал обязан пережить рестарт. Контакт держит бандл,
// выданный однажды, и если после перезапуска SPK/OPK станут другими, X3DH этого
// контакта соберёт не тот корень, и первое же сообщение (в том числе почта,
// оставленная слепым релеем) отвалится на расшифровывании.
type StoredPrekeys struct {
	// Owner — мастер-X25519 идентичности, которой принадлежит набор. Тот же
	// хранилищный файл может служить нескольким личностям (явно заданный
	// master_key_256), а prekey-материал — не строка диалога, а сам ключ:
	// надеть чужой значит начать расшифровывать чужие диалоги. Поэтому набор
	// хранится под своим владельцем и другому не отдаётся.
	Owner   []byte         `json:"owner"`
	SpkPriv []byte         `json:"spk_priv"`
	OpkID   uint32         `json:"opk_id"`
	Opk     []StoredPrekey `json:"opk"`
	// OpkConsumed — ключи, уже потраченные на собранные сессии: они ушли из
	// склада, но забывать их нельзя, иначе следующий бандл выдаст ключ, по
	// которому у кого-то уже открыта сессия.
	OpkConsumed []uint32 `json:"opk_consumed"`
}

type StoredOutbound struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	// NodeKey — Ed25519-ключ узла получателя: он же адрес сессии ironwood и
	// то, о чём спрашивает HasRouteTo.
	NodeKey    []byte `json:"node_key"`
	Envelope   []byte `json:"envelope"`
	EnqueuedMS int64  `json:"enqueued_ms"`
	// Sends — сколько раз конверт лёг в сокет. Первый пакет на ещё не прогретой
	// сессии ironwood оверлей теряет регулярно, поэтому один дубль оправдан.
	// Бесконечная переотправка не оправдана: дословно повторённый конверт
	// получатель раскроет лишь один раз, а остальные отвергнет как replay —
	// номер сообщения в приёмной цепочке движется только вперёд.
	Sends      uint32 `json:"sends"`
	LastSendMS int64  `json:"last_send_ms"`
	// RelayedMS — когда конверт последний раз лёг слепым S&F-релеям, RelaySends —
	// сколько раз. Одной кладки не хватает по той же причине, что и в живой
	// отправке: первый пакет на ещё не прогретой сессии ironwood теряется. Без
	// счётчика сканер каждые две секунды сбрасывал бы на чужой узел те же байты.
	// Обнуляется при перепостановке: тогда на хранителя обязан лечь новый конверт,
	// а не тот, что лежал до перезапуска.
	RelayedMS  int64  `json:"relayed_ms"`
	RelaySends uint32 `json:"relay_sends"`
}

// EncryptedDB manages zero-dependency on-disk encrypted tables with Full-Text Indexing
type EncryptedDB struct {
	mu        sync.RWMutex
	dataDir   string
	masterKey [32]byte
	deviceID  string

	// Memory Index
	messages map[string]*StoredMessage
	byConv   map[string][]string // convID -> []messageID
	ftsIndex map[string][]string // lowercase token -> []messageID
	sessions map[string][]byte   // имя сеанса -> состояние, собранное pkg/rpc
	// prekeys — prekey-материал ядра (SPK + склад OPK) по владельцу: hex
	// мастер-X25519 -> набор. Байты ключей разбирает и проверяет только pkg/rpc;
	// хранилище обязано отдать набор дословно и ровно тому, кому он принадлежит.
	prekeys map[string]*StoredPrekeys
	// contacts — identity-hex контакта -> байты ContactCard. Хранилище их не
	// разбирает: проверка карты целиком в pkg/rpc.
	contacts map[string][]byte
	// sf — чужие конверты, которые этот узел хранит как слепой S&F-релей:
	// ключ "mailboxhex|envelopehex". Начинку хранилище не видит и не оценивает,
	// квоты и права на выдачу решает pkg/rpc.
	sf       map[string]*StoredSF
	outbound map[string]*StoredOutbound
	crdtLog  []CRDTOperation
	// crdtSeen — множество уже применённых op_id. Дневник переигрывается при
	// каждом открытии, а повторный set того же операция удвоил бы лог.
	crdtSeen map[string]bool

	// Clock State
	lastPhysical int64
	logicalCount uint32

	// Дневник (Шаг 10): снапшот + append-only журнал вместо полной перезаписи.
	journalAEAD cipher.AEAD
	journalSeq  uint64
	snapBytes   int64
	dirKey      string
	open        bool
	// Счётчики восстановления — см. JournalStats.
	jrnApplied, jrnSkipped, jrnCorrupt, jrnStale, compactions int
}

// OpenEncryptedDB initializes or loads an encrypted database at target path
func OpenEncryptedDB(dataDir string, masterKey [32]byte, deviceID string) (*EncryptedDB, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed creating db dir: %w", err)
	}

	dirKey, err := claimVaultDir(dataDir)
	if err != nil {
		return nil, err
	}
	aead, err := newJournalAEAD(masterKey)
	if err != nil {
		releaseVaultDir(dirKey)
		return nil, err
	}

	db := &EncryptedDB{
		dataDir:     dataDir,
		masterKey:   masterKey,
		deviceID:    deviceID,
		messages:    make(map[string]*StoredMessage),
		byConv:      make(map[string][]string),
		ftsIndex:    make(map[string][]string),
		sessions:    make(map[string][]byte),
		contacts:    make(map[string][]byte),
		sf:          make(map[string]*StoredSF),
		prekeys:     make(map[string]*StoredPrekeys),
		outbound:    make(map[string]*StoredOutbound),
		crdtLog:     make([]CRDTOperation, 0),
		crdtSeen:    make(map[string]bool),
		journalAEAD: aead,
		dirKey:      dirKey,
		open:        true,
	}

	if err := db.loadFromDisk(); err != nil {
		releaseVaultDir(dirKey)
		db.open = false
		return nil, err
	}
	return db, nil
}

// Close отпускает каталог: дневник принадлежит одному экземпляру хранилища, и
// второй живой экземпляр на тот же путь писал бы кадры с уже использованными
// номерами. Записи после Close отклоняются, а не теряются молча.
func (db *EncryptedDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.open {
		return nil
	}
	db.open = false
	releaseVaultDir(db.dirKey)
	return nil
}

// NextHLC generates the next causal timestamp on this device
func (db *EncryptedDB) NextHLC() HLC {
	now := time.Now().UnixMilli()
	if now > db.lastPhysical {
		db.lastPhysical = now
		db.logicalCount = 0
	} else {
		db.logicalCount++
	}

	return HLC{
		PhysicalTime: db.lastPhysical,
		LogicalCount: db.logicalCount,
		DeviceID:     db.deviceID,
	}
}

// UpdateHLC updates local clock when receiving operations from peer devices
func (db *EncryptedDB) UpdateHLC(remote HLC) HLC {
	now := time.Now().UnixMilli()
	maxPhysical := now
	if remote.PhysicalTime > maxPhysical {
		maxPhysical = remote.PhysicalTime
	}

	if maxPhysical > db.lastPhysical {
		db.lastPhysical = maxPhysical
		db.logicalCount = 0
	} else if maxPhysical == remote.PhysicalTime && maxPhysical == db.lastPhysical {
		if remote.LogicalCount > db.logicalCount {
			db.logicalCount = remote.LogicalCount + 1
		} else {
			db.logicalCount++
		}
	} else {
		db.logicalCount++
	}

	return HLC{
		PhysicalTime: db.lastPhysical,
		LogicalCount: db.logicalCount,
		DeviceID:     db.deviceID,
	}
}

// -----------------------------------------------------------------------------
// DISK ENCRYPTION (AES-256-GCM / XCHACHA20 EQUIVALENT AEAD RECORD WRAPPER)
// -----------------------------------------------------------------------------

func (db *EncryptedDB) encryptRecord(plaintext []byte, recordID string) ([]byte, error) {
	block, err := aes.NewCipher(db.masterKey[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, []byte(recordID))
	result := make([]byte, 0, len(nonce)+len(ciphertext))
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

func (db *EncryptedDB) decryptRecord(data []byte, recordID string) ([]byte, error) {
	if len(data) < 12 {
		return nil, errors.New("db: encrypted record truncated")
	}

	block, err := aes.NewCipher(db.masterKey[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	return aead.Open(nil, nonce, ciphertext, []byte(recordID))
}

// -----------------------------------------------------------------------------
// CHAT STORAGE & FULL-TEXT SEARCH (FTS) INDEXING
// -----------------------------------------------------------------------------

// SaveMessage stores message and updates full-text search index
func (db *EncryptedDB) SaveMessage(msg *StoredMessage) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if msg.Clock.PhysicalTime == 0 {
		msg.Clock = db.NextHLC()
	}

	// Повторная запись того же message_id — переотправка: запись обновляется,
	// но в списке диалога и в полнотекстовом индексе второй экземпляр не
	// появляется. Часы и отметку доставки берём от первой записи, иначе
	// пересланное сообщение перепрыгнуло бы соседей в CRDT-порядке и потеряло
	// полученный ACK.
	if prev, dup := db.messages[msg.MessageID]; dup {
		msg.Clock = prev.Clock
		msg.DeliveredMS = prev.DeliveredMS
		// Флаги «прочитано», «закреплено» и «удалено» могут только загораться:
		// пересылка того же сообщения их не тушит.
		msg.IsRead = msg.IsRead || prev.IsRead
		msg.IsPinned = msg.IsPinned || prev.IsPinned
		msg.IsDeleted = msg.IsDeleted || prev.IsDeleted
	}

	db.storeMessageLocked(msg)
	return db.journalSetLocked(tableMessages, msg.MessageID, msg)
}

// storeMessageLocked кладёт сообщение в память и заводит для него индексы —
// ровно один раз на message_id. Тот же путь используется при загрузке снапшота и
// при переигрывании дневника, поэтому повторная запись переотправленного
// сообщения не плодит второй экземпляр в списке диалога и в полнотекстовом
// индексе.
func (db *EncryptedDB) storeMessageLocked(msg *StoredMessage) {
	if _, dup := db.messages[msg.MessageID]; dup {
		db.messages[msg.MessageID] = msg
		return
	}
	db.messages[msg.MessageID] = msg
	db.byConv[msg.ConversationID] = append(db.byConv[msg.ConversationID], msg.MessageID)
	for _, tok := range tokenizeText(msg.Body) {
		db.ftsIndex[tok] = append(db.ftsIndex[tok], msg.MessageID)
	}
}

// HasMessage — есть ли сообщение с таким ID в локальной базе. Приёмный цикл
// использует это как дедупликацию: S&F-релей может доставить один конверт
// несколько раз, а повторный рэтчет-шаг сломал бы сессию.
func (db *EncryptedDB) HasMessage(messageID string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	_, ok := db.messages[messageID]
	return ok
}

// IsDelivered — есть ли у сообщения отметка сквозной доставки. Приёмный цикл
// спрашивает это до MarkDelivered, чтобы повторный ACK не объявлял событие
// заново.
func (db *EncryptedDB) IsDelivered(messageID string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	msg, ok := db.messages[messageID]
	return ok && msg.DeliveredMS != 0
}

// MarkDelivered проставляет время сквозного ACK получателя и отдаёт
// conversation_id найденного сообщения — статус показываем по тому, что лежит
// в базе, а не по тексту с провода.
//
// false означает «отметить нечего»: сообщения нет в базе, оно уже удалено, или
// это входящее сообщение (его доставка — не наше обещание, а чужое).
func (db *EncryptedDB) MarkDelivered(messageID string, atMS int64) (string, bool, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	msg, ok := db.messages[messageID]
	if !ok || msg.IsDeleted || !msg.IsOutbound {
		return "", false, nil
	}
	if msg.DeliveredMS == 0 {
		msg.DeliveredMS = atMS
		return msg.ConversationID, true, db.journalSetLocked(tableMessages, msg.MessageID, msg)
	}
	// Повторный ACK не переписывает время: доставкой считается первое
	// подтверждение.
	return msg.ConversationID, true, nil
}

// -----------------------------------------------------------------------------
// ИСХОДЯЩАЯ ОЧЕРЕДЬ (Шаг 3-c-2)
// -----------------------------------------------------------------------------

// QueueOutbound кладёт (или обновляет) конверт, ожидающий маршрута. Ключ —
// message_id: повторная отправка того же сообщения заменяет байты, а не заводит
// вторую запись, иначе очередь росла бы с каждой неудачей.
func (db *EncryptedDB) QueueOutbound(rec *StoredOutbound) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if rec.MessageID == "" || len(rec.Envelope) == 0 {
		return errors.New("db: в исходящую очередь просят положить пустой конверт")
	}
	if rec.EnqueuedMS == 0 {
		rec.EnqueuedMS = time.Now().UnixMilli()
	}
	if prev, ok := db.outbound[rec.MessageID]; ok {
		// Время постановки и счётчик отправок принадлежат записи, а не вызову:
		// перепостановка не передвигает сообщение в очереди и не обнуляет
		// сделанные попытки.
		rec.EnqueuedMS = prev.EnqueuedMS
		rec.Sends = prev.Sends
		rec.LastSendMS = prev.LastSendMS
	}
	db.outbound[rec.MessageID] = rec
	return db.journalSetLocked(tableOutbound, rec.MessageID, rec)
}

// PendingOutbound отдаёт копию очереди от старых сообщений к новым — так
// диалог восстанавливается в том порядке, в котором пользователь печатал.
func (db *EncryptedDB) PendingOutbound() []StoredOutbound {
	db.mu.RLock()
	defer db.mu.RUnlock()

	out := make([]StoredOutbound, 0, len(db.outbound))
	for _, rec := range db.outbound {
		out = append(out, *rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].EnqueuedMS != out[j].EnqueuedMS {
			return out[i].EnqueuedMS < out[j].EnqueuedMS
		}
		return out[i].MessageID < out[j].MessageID
	})
	return out
}

// RecordOutboundSend отмечает отправку конверта в сокет. Счётчик определяет,
// когда запись пора убрать: бесконечно слать одни и те же байты нельзя.
func (db *EncryptedDB) RecordOutboundSend(messageID string, atMS int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	rec, ok := db.outbound[messageID]
	if !ok {
		return nil
	}
	rec.Sends++
	rec.LastSendMS = atMS
	return db.journalSetLocked(tableOutbound, messageID, rec)
}

// RecordRelaySend засчитывает попытку кладки конверта слепым релеям и запоминает
// её время. Отсутствие записи — не ошибка: сообщение могли подтвердить и снять из
// очереди раньше, чем дошёл черёд кладки.
func (db *EncryptedDB) RecordRelaySend(messageID string, atMS int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	rec, ok := db.outbound[messageID]
	if !ok {
		return nil
	}
	rec.RelaySends++
	rec.RelayedMS = atMS
	return db.journalSetLocked(tableOutbound, messageID, rec)
}

// CompleteOutbound убирает сообщение из очереди. Отсутствие записи — не
// ошибка: так выглядит первая удачная отправка, которой нечего чистить.
func (db *EncryptedDB) CompleteOutbound(messageID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.outbound[messageID]; !ok {
		return nil
	}
	delete(db.outbound, messageID)
	return db.journalDelLocked(tableOutbound, messageID)
}

// -----------------------------------------------------------------------------
// СЕАНСЫ ДВОЙНОГО РЭТЧЕТА (Шаг 3-c-3)
// -----------------------------------------------------------------------------

// SaveSession кладёт состояние сеанса (или заменяет его). Байты для хранилища
// непрозрачны: их собирает pkg/rpc, включая привязку к мастер-ключу владельца.
//
// Писать нужно при каждом сдвиге цепочки. Молча принятое сюда устаревшее
// состояние означало бы, что после рестарта ядро отправит номер, который
// собеседник уже израсходовал, — и следующее сообщение потеряется.
func (db *EncryptedDB) SaveSession(name string, state []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if name == "" || len(state) == 0 {
		return errors.New("db: просят сохранить сеанс без имени или без состояния")
	}
	db.sessions[name] = state
	return db.journalSetLocked(tableSessions, name, state)
}

// Sessions отдаёт копию всех сохранённых состояний: имя сеанса -> байты. Годное
// это состояние или чужое — решает вызывающий, хранилище содержимое не
// интерпретирует.
func (db *EncryptedDB) Sessions() map[string][]byte {
	db.mu.RLock()
	defer db.mu.RUnlock()

	out := make(map[string][]byte, len(db.sessions))
	for name, state := range db.sessions {
		out[name] = append([]byte(nil), state...)
	}
	return out
}

// SaveContact кладёт карту контакта (или заменяет её по тому же ключу). Байты
// для хранилища непрозрачны: ContactCard собирает и проверяет pkg/rpc.
//
// Пустой ключ или пустые байты не принимаются: сохранённая «карта ничего»
// воскреснет после рестарта как контакт, которому нельзя доверить адрес.
func (db *EncryptedDB) SaveContact(key string, card []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if key == "" || len(card) == 0 {
		return errors.New("db: просят сохранить контакт без ключа или без карты")
	}
	db.contacts[key] = card
	return db.journalSetLocked(tableContacts, key, card)
}

// Contacts отдаёт копию всех сохранённых карт: ключ -> байты. Годная ли это
// карта и того ли контакта ключ — решает вызывающий.
func (db *EncryptedDB) Contacts() map[string][]byte {
	db.mu.RLock()
	defer db.mu.RUnlock()

	out := make(map[string][]byte, len(db.contacts))
	for key, card := range db.contacts {
		out[key] = append([]byte(nil), card...)
	}
	return out
}

// sfKey — составной ключ записи почты: один ящик может держать несколько
// конвертов, а один и тот же конверт не должен ложиться дважды.
func sfKey(mailboxID, envelopeID string) string { return mailboxID + "|" + envelopeID }

// SFPut кладёт чужой конверт в слепое хранилище релея. Повторная кладка того
// же содержимого в тот же ящик — не ошибка и не второй экземпляр: почту
// перекладывают, когда отправитель повторяет попытку.
func (db *EncryptedDB) SFPut(rec *StoredSF) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if rec.MailboxID == "" || rec.EnvelopeID == "" || len(rec.Data) == 0 {
		return errors.New("db: слепая почта без ящика, идентификатора или тела не принимается")
	}
	key := sfKey(rec.MailboxID, rec.EnvelopeID)
	if prev, ok := db.sf[key]; ok {
		// Срок хранения принадлежит первой записи: переотправка не продлевает
		// чужое письмо до бесконечности.
		rec.PlacedMS = prev.PlacedMS
		rec.ExpiresMS = prev.ExpiresMS
	}
	db.sf[key] = rec
	return db.journalSetLocked(tableSF, key, rec)
}

// SFHandOut отдаёт всю почту ящика и помечает её выданной, но из хранилища не
// убирает. Семантика — «выдачу можно повторить»: релей не знает, дошёл ли
// адресат, и пакет ответа теряется на оверлее точно так же, как живой конверт.
// Список, по которому запись всё же уходит, держит SFDropHanded; отправитель тем
// временем не выбрасывает свою копию из исходящей очереди до ACK, а получатель
// дедуплицирует по message_id.
func (db *EncryptedDB) SFHandOut(mailboxID string, atMS int64) ([]StoredSF, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var out []StoredSF
	for _, rec := range db.sf {
		if rec.MailboxID != mailboxID {
			continue
		}
		rec.HandedMS = atMS
		out = append(out, *rec)
	}
	if len(out) == 0 {
		return nil, nil
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PlacedMS != out[j].PlacedMS {
			return out[i].PlacedMS < out[j].PlacedMS
		}
		return out[i].EnvelopeID < out[j].EnvelopeID
	})
	for i := range out {
		key := sfKey(out[i].MailboxID, out[i].EnvelopeID)
		if err := db.journalSetLocked(tableSF, key, out[i]); err != nil {
			// Метки уже передвинулись в памяти: ошибку возвращаем как есть, иначе
			// релей вообразит, что окно повторной выдачи открылось на диске.
			return out, err
		}
	}
	return out, nil
}

// SFDropHanded убирает письма, выданные давно: olderThanMS — граница окна
// повторной выдачи, которую назначает pkg/rpc. Возвращает число удалённых:
// молча терять чужую почту можно только по опубликованному правилу.
func (db *EncryptedDB) SFDropHanded(olderThanMS int64) int {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.dropSFLocked(func(rec *StoredSF) bool {
		return rec.HandedMS != 0 && rec.HandedMS <= olderThanMS
	})
}

// dropSFLocked удаляет письма, прошедшие отбор. Дневник дописывается до того, как
// запись исчезнет из памяти: удаление, которого диск не принял, не имеет права
// выглядеть состоявшимся, иначе при следующем открытии почта воскреснет молча.
// Ключи собираются и сортируются отдельно — порядок кадров в дневнике обязан быть
// детерминированным, а обход карты его не даёт.
func (db *EncryptedDB) dropSFLocked(matched func(*StoredSF) bool) int {
	var keys []string
	for key, rec := range db.sf {
		if matched(rec) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	dropped := 0
	for _, key := range keys {
		if err := db.journalDelLocked(tableSF, key); err != nil {
			break
		}
		delete(db.sf, key)
		dropped++
	}
	// Обрыв возвращает ровно то, что диск принял: это меньше, чем подошло, и
	// вызывающий видит расхождение по счётчику, а не по молчаливо пропавшей почте.
	return dropped
}

// SFMailbox — сколько писем и байтов лежит в ящике: квоты релея считаются по
// живому состоянию, а не по доверию вызывающему.
func (db *EncryptedDB) SFMailbox(mailboxID string) (count int, bytes int64) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, rec := range db.sf {
		if rec.MailboxID == mailboxID {
			count++
			bytes += int64(len(rec.Data))
		}
	}
	return count, bytes
}

// SFStoredTotal — сколько чужой почты этот узел держит сейчас (для статуса).
func (db *EncryptedDB) SFStoredTotal() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return len(db.sf)
}

// SFDropExpired выбрасывает письма, срок которых истёк. Возвращает количество
// удалённых: молча терять чужую почту можно только по опубликованному TTL.
func (db *EncryptedDB) SFDropExpired(nowMS int64) int {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.dropSFLocked(func(rec *StoredSF) bool {
		return rec.ExpiresMS > 0 && rec.ExpiresMS <= nowMS
	})
}

// SearchMessages performs instant tokenized full-text search across local decrypted messages
func (db *EncryptedDB) SearchMessages(query string, limit int) []*StoredMessage {
	db.mu.RLock()
	defer db.mu.RUnlock()

	queryTokens := tokenizeText(query)
	if len(queryTokens) == 0 {
		return nil
	}

	// Match intersection of token postings
	var candidateIDs []string
	first := true

	for _, token := range queryTokens {
		matched, exists := db.ftsIndex[token]
		if !exists {
			return nil // AND semantic: all tokens must be present
		}

		if first {
			candidateIDs = append(candidateIDs, matched...)
			first = false
		} else {
			candidateIDs = intersect(candidateIDs, matched)
		}
	}

	// Fetch message pointers and exclude tombstones
	results := make([]*StoredMessage, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		if msg, ok := db.messages[id]; ok && !msg.IsDeleted {
			results = append(results, msg)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	// Sort chronologically
	sort.Slice(results, func(i, j int) bool {
		return results[i].TimestampMS > results[j].TimestampMS
	})

	return results
}

// ConversationSummary — агрегатор по диалогу для списка чатов в UI.
type ConversationSummary struct {
	ConversationID  string `json:"conversation_id"`
	LastBody        string `json:"last_body"`
	LastTimestampMS int64  `json:"last_timestamp_ms"`
	MessageCount    int    `json:"message_count"`
	UnreadCount     int    `json:"unread_count"`
}

// ListConversations возвращает диалоги от новых к старым, без tombstone'ов.
func (db *EncryptedDB) ListConversations() []ConversationSummary {
	db.mu.RLock()
	defer db.mu.RUnlock()

	summaries := make([]ConversationSummary, 0, len(db.byConv))
	for convID, messageIDs := range db.byConv {
		s := ConversationSummary{ConversationID: convID}
		for _, id := range messageIDs {
			msg, ok := db.messages[id]
			if !ok || msg.IsDeleted {
				continue
			}
			s.MessageCount++
			if !msg.IsRead && !msg.IsOutbound {
				s.UnreadCount++
			}
			if msg.TimestampMS >= s.LastTimestampMS {
				s.LastTimestampMS = msg.TimestampMS
				s.LastBody = msg.Body
			}
		}
		if s.MessageCount == 0 {
			continue
		}
		summaries = append(summaries, s)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].LastTimestampMS > summaries[j].LastTimestampMS
	})
	return summaries
}

// GetMessages — последние сообщения диалога, от новых к старым.
func (db *EncryptedDB) GetMessages(conversationID string, limit int) []*StoredMessage {
	db.mu.RLock()
	defer db.mu.RUnlock()

	ids := db.byConv[conversationID]
	results := make([]*StoredMessage, 0, len(ids))
	for _, id := range ids {
		msg, ok := db.messages[id]
		if !ok || msg.IsDeleted {
			continue
		}
		results = append(results, msg)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TimestampMS > results[j].TimestampMS
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// -----------------------------------------------------------------------------
// CRDT STATE SYNCHRONIZATION (HLC MERGE BETWEEN WINDOWS & ANDROID)// -----------------------------------------------------------------------------

// ApplyCRDTOperation applies a local or remote mutation using Last-Write-Wins (LWW) semantics
func (db *EncryptedDB) ApplyCRDTOperation(op CRDTOperation) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if op.OpID == "" {
		return errors.New("db: операция CRDT без op_id не применяется: лог опознать нельзя")
	}
	db.UpdateHLC(op.Clock)
	if err := db.journalSetLocked(tableCRDTOps, op.OpID, op); err != nil {
		return err
	}
	db.appendCRDTOpLocked(op)

	msg, exists := db.messages[op.TargetID]
	if !exists {
		return nil // Message may arrive later; op is retained in crdtLog
	}

	// Compare clocks: only apply if operation is newer
	if op.Clock.Compare(msg.Clock) > 0 {
		switch op.Type {
		case "read":
			msg.IsRead = true
		case "pin":
			msg.IsPinned = (op.Payload == "true")
		case "tombstone_delete":
			msg.IsDeleted = true
			msg.Body = "" // Erase contents for "Delete for Everyone"
		}
		msg.Clock = op.Clock
		return db.journalSetLocked(tableMessages, msg.MessageID, msg)
	}
	return nil
}

// GetUnsyncedCRDTOps returns operations created after a given clock baseline
func (db *EncryptedDB) GetUnsyncedCRDTOps(since HLC) []CRDTOperation {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []CRDTOperation
	for _, op := range db.crdtLog {
		if op.Clock.Compare(since) > 0 {
			result = append(result, op)
		}
	}
	return result
}

// -----------------------------------------------------------------------------
// DISK PERSISTENCE & HELPERS
// -----------------------------------------------------------------------------

// writeSnapshotLocked пишет полный снимок базы: tmp, sync, rename. С Шага 10 её
// вызывает только компакция дневника — на каждую запись база целиком больше не
// переписывается.
func (db *EncryptedDB) writeSnapshotLocked() error {
	payload, err := json.Marshal(vaultSnapshot{
		Messages:   mapToSlice(db.messages),
		CRDTOps:    db.crdtLog,
		Outbound:   outboundToSlice(db.outbound),
		Sessions:   db.sessions,
		Contacts:   db.contacts,
		Sf:         sfToSlice(db.sf),
		PrekeySet:  prekeysToSlice(db.prekeys),
		JournalSeq: db.journalSeq,
	})
	if err != nil {
		return fmt.Errorf("db: снапшот не сериализуется: %w", err)
	}

	encrypted, err := db.encryptRecord(payload, snapshotRecordID)
	if err != nil {
		return err
	}

	dbFile := filepath.Join(db.dataDir, VaultSnapshotFile)
	tmpFile := dbFile + ".tmp"

	f, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("db: черновик снапшота %s не открыт: %w", tmpFile, err)
	}
	if _, err := f.Write(encrypted); err != nil {
		_ = f.Close()
		return fmt.Errorf("db: снапшот не записан: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("db: снапшот не синхронизирован: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("db: черновик снапшота не закрыт: %w", err)
	}
	if err := os.Rename(tmpFile, dbFile); err != nil {
		return fmt.Errorf("db: снапшот не заменён на %s: %w", dbFile, err)
	}
	st, err := os.Stat(dbFile)
	if err != nil {
		return fmt.Errorf("db: размер снапшота неизвестен: %w", err)
	}
	db.snapBytes = st.Size()
	return nil
}

// loadFromDisk поднимает снапшот, убирает остатки ротации и переигрывает дневник
// поверх. Порядок именно такой: дневник по построению не старше снапшота, и
// пропуск кадров со старым номером — единственное, что отличает «уже снято» от
// «ещё не применено».
func (db *EncryptedDB) loadFromDisk() error {
	if err := db.loadSnapshot(); err != nil {
		return err
	}
	db.sweepStaleJournals()
	return db.replayJournal()
}

func (db *EncryptedDB) loadSnapshot() error {
	dbFile := filepath.Join(db.dataDir, VaultSnapshotFile)
	st, err := os.Stat(dbFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("db: снапшот %s не видно: %w", dbFile, err)
	}
	db.snapBytes = st.Size()

	data, err := os.ReadFile(dbFile)
	if err != nil {
		return err
	}

	decrypted, err := db.decryptRecord(data, snapshotRecordID)
	if err != nil {
		return fmt.Errorf("failed to decrypt database state: %w", err)
	}

	var state vaultSnapshot
	if err := json.Unmarshal(decrypted, &state); err != nil {
		return err
	}
	// База, созданная до Шага 10, поля journal_seq не имеет: ноль означает «пока
	// переигрываются все кадры», и это верно, потому что дневника у той базы нет.
	db.journalSeq = state.JournalSeq

	for i := range state.Messages {
		db.storeMessageLocked(&state.Messages[i])
	}
	for _, op := range state.CRDTOps {
		db.appendCRDTOpLocked(op)
	}
	// Очередь переживает рестарт: без неё перезапуск приложения означал бы
	// молчаливую потерю неотправленного.
	for i := range state.Outbound {
		rec := state.Outbound[i]
		if rec.MessageID == "" || len(rec.Envelope) == 0 {
			continue
		}
		db.outbound[rec.MessageID] = &rec
	}
	// Состояния сеансов: без них перезапуск ядра убивал бы каждый открытый
	// диалог — собеседник прислал бы номер из цепочки, которой здесь больше нет.
	for name, blob := range state.Sessions {
		if name == "" || len(blob) == 0 {
			continue
		}
		db.sessions[name] = blob
	}
	// Карты контактов: без них перезапуск оставлял бы сохранённые диалоги без
	// единственного адреса, по которому сообщение может уйти.
	for key, card := range state.Contacts {
		if key == "" || len(card) == 0 {
			continue
		}
		db.contacts[key] = card
	}
	// Чужая почта S&F обязана пережить рестарт релея: иначе перезапуск сервиса
	// означал бы молчаливое исчезновение писем, которые адресат ещё не забрал.
	for _, rec := range state.Sf {
		if rec.MailboxID == "" || rec.EnvelopeID == "" || len(rec.Data) == 0 {
			continue
		}
		r := rec
		db.sf[sfKey(r.MailboxID, r.EnvelopeID)] = &r
	}
	// Prekey-материал: пустая или безключевая запись — это «материала ещё нет»,
	// а не битый снимок. Годный набор отдаём pkg/rpc дословно, под его же
	// владельцем: чужой комплект ключей чужой личности не надевается.
	for _, pk := range state.PrekeySet {
		if pk == nil || len(pk.SpkPriv) != 32 || len(pk.Owner) != 32 {
			continue
		}
		db.prekeys[prekeyKey(pk.Owner)] = pk
	}

	return nil
}

// SavePrekeys кладёт prekey-материал ядра под его владельца. Замена полная, а
// не дополняющая: выбрасываемый one-time prekey обязан исчезнуть и с диска,
// иначе чужой снимок ключей пережил бы свою жизнь в памяти.
func (db *EncryptedDB) SavePrekeys(pk *StoredPrekeys) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if pk == nil || len(pk.SpkPriv) != 32 {
		return errors.New("db: prekey-материал без signed prekey не сохраняется")
	}
	if len(pk.Owner) != 32 {
		return errors.New("db: prekey-материал без владельца не сохраняется")
	}
	stored := clonePrekeys(pk)
	db.prekeys[prekeyKey(stored.Owner)] = stored
	return db.journalSetLocked(tablePrekeys, prekeyKey(stored.Owner), stored)
}

// Prekeys отдаёт копию prekey-материала указанной идентичности или nil, если её
// набора на диске нет (первый запуск ядра либо другой владелец хранилища).
func (db *EncryptedDB) Prekeys(owner []byte) *StoredPrekeys {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if len(owner) != 32 {
		return nil
	}
	stored := db.prekeys[prekeyKey(owner)]
	if stored == nil {
		return nil
	}
	return clonePrekeys(stored)
}

// prekeyKey — индекс набора по владельцу. Hex, а не сырые байты: ключом JSON
// карты служит строка, и не-UTF-8 в ней переписалась бы при сохранении.
func prekeyKey(owner []byte) string { return fmt.Sprintf("%x", owner) }

func prekeysToSlice(byOwner map[string]*StoredPrekeys) []*StoredPrekeys {
	out := make([]*StoredPrekeys, 0, len(byOwner))
	for _, pk := range byOwner {
		out = append(out, pk)
	}
	sort.Slice(out, func(i, j int) bool {
		return prekeyKey(out[i].Owner) < prekeyKey(out[j].Owner)
	})
	return out
}

// clonePrekeys копирует набор целиком, до байтов приватных ключей. Копия нужна
// в обе стороны: ядро держит те же ключи в своей памяти, и случайная правка
// выданного или входного снимка не должна переписывать живое ключевое вещество.
func clonePrekeys(pk *StoredPrekeys) *StoredPrekeys {
	out := &StoredPrekeys{
		Owner:       append([]byte(nil), pk.Owner...),
		SpkPriv:     append([]byte(nil), pk.SpkPriv...),
		OpkID:       pk.OpkID,
		Opk:         make([]StoredPrekey, 0, len(pk.Opk)),
		OpkConsumed: append([]uint32(nil), pk.OpkConsumed...),
	}
	for _, opk := range pk.Opk {
		out.Opk = append(out.Opk, StoredPrekey{ID: opk.ID, Priv: append([]byte(nil), opk.Priv...)})
	}
	return out
}

func tokenizeText(text string) []string {
	clean := strings.ToLower(text)
	words := strings.FieldsFunc(clean, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 'а' && r <= 'я') || r == 'ё')
	})

	seen := make(map[string]bool)
	var tokens []string
	for _, w := range words {
		if len(w) >= 2 && !seen[w] {
			seen[w] = true
			tokens = append(tokens, w)
		}
	}
	return tokens
}

func intersect(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, x := range b {
		set[x] = true
	}
	var res []string
	for _, x := range a {
		if set[x] {
			res = append(res, x)
		}
	}
	return res
}

func mapToSlice(m map[string]*StoredMessage) []StoredMessage {
	res := make([]StoredMessage, 0, len(m))
	for _, v := range m {
		res = append(res, *v)
	}
	// Порядок на диске стабилен по той же причине, что и у двух срезов ниже:
	// равные по содержанию снапшоты обязаны быть равны по байтам.
	sort.Slice(res, func(i, j int) bool { return res[i].MessageID < res[j].MessageID })
	return res
}

// appendCRDTOpLocked кладёт операцию в лог ровно один раз на op_id: дневник
// переигрывается при каждом открытии, а повторный set удвоил бы историю.
// Операция без op_id (такая появилась бы только из базы до Шага 10) в лог
// попадает, но вывести её повтор нельзя — поэтому и не дедуплицируется.
func (db *EncryptedDB) appendCRDTOpLocked(op CRDTOperation) {
	if op.OpID == "" {
		db.crdtLog = append(db.crdtLog, op)
		return
	}
	if db.crdtSeen[op.OpID] {
		return
	}
	db.crdtSeen[op.OpID] = true
	db.crdtLog = append(db.crdtLog, op)
}

func sfToSlice(m map[string]*StoredSF) []StoredSF {
	res := make([]StoredSF, 0, len(m))
	for _, v := range m {
		res = append(res, *v)
	}
	// Порядок на диске стабилен: равные по содержанию снимки не должны
	// расходиться байтами.
	sort.Slice(res, func(i, j int) bool {
		if res[i].PlacedMS != res[j].PlacedMS {
			return res[i].PlacedMS < res[j].PlacedMS
		}
		return res[i].EnvelopeID < res[j].EnvelopeID
	})
	return res
}

func outboundToSlice(m map[string]*StoredOutbound) []StoredOutbound {
	res := make([]StoredOutbound, 0, len(m))
	for _, v := range m {
		res = append(res, *v)
	}
	// Порядок на диске стабилен, иначе каждая перезапись вела бы себе равный
	// JSON и diff в тестах над файлом был бы шумом.
	sort.Slice(res, func(i, j int) bool { return res[i].MessageID < res[j].MessageID })
	return res
}

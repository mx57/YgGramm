// Package rpc реализует команды FFI-контракта (yggproto.CoreRequest) поверх
// настоящих подсистем ядра: crypto, storage и встроенной yggdrasil-go (yggnode).
//
// До появления этого пакета cmd/libygggram сам декодировал байты запроса и на
// StartNode/StopNode/SendMessage лишь менял локальные флаги, из-за чего весь
// слой криптографии и хранения оставался недостижимым из приложения.
package rpc

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"ygggram/core/pkg/crypto"
	"ygggram/core/pkg/storage"
	"ygggram/core/pkg/yggnode"
	yggproto "ygggram/core/proto"
)

// powDifficulty — сложность Hashcash для анонимного ящика (16 ведущих нулей).
const powDifficulty = 16

// Engine — состояние ядра, доступное через C-ABI.
type Engine struct {
	mu sync.Mutex

	dataDir  string
	cacheDir string

	master  *crypto.MasterIdentity
	device  *crypto.DeviceIdentity
	nodeKey ed25519.PrivateKey

	// Подписанный prekey этой ноды: публикуется контактам для X3DH.
	spkPriv *ecdh.PrivateKey
	opkPriv map[uint32]*ecdh.PrivateKey
	// opkID — какой one-time prekey опубликован сейчас, opkConsumed — какие
	// уже потрачены. Израсходованный ключ не выбрасывается сразу: отправитель
	// пользуется статичным бандлом и может повторить X3DH старым OPK, поэтому
	// мы публикуем свежий, а старый держим для расширяемости.
	opkID       uint32
	opkConsumed map[uint32]bool

	db   *storage.EncryptedDB
	node *yggnode.YggNode
	// vaultKey — ключ хранилища, которым реально открыта база этого запуска.
	// Он не выводится заново по требованию, а запоминается, потому что
	// StartNode принимает мастер-ключ из конфигурации (master_key_256) и тогда
	// база открыта не seed-ом кошелька. Восстановление обязано проверять
	// пакет ровно этим ключом: иначе «проверка прошла» врёт об одном томе, а
	// живое ядро откроет другой. Поле nil до StartNode и остаётся заданным
	// после StopNode — смена тома без перезапуска ядра невозможна.
	vaultKey *[32]byte
	// contactKey -> карта контакта.
	contacts map[string]*yggproto.ContactCard
	// Один сеанс Double Ratchet на диалог в обе стороны, ключ —
	// sessionKey(peerX, conversationID). Инициаторский и приёмный сеансы
	// различает сам рэтчет: после первого ответа получателя его отправная
	// цепочка строится тем же DH-шагом, поэтому держать два склада сессий
	// нельзя — ACK не нашёл бы сеанс, на котором пришло сообщение.
	sessions map[string]*crypto.DoubleRatchetSession
	// sessionKey -> эфемерный X25519 ключ X3DH и потраченный one-time prekey.
	ekPub   map[string][]byte
	opkUsed map[string]uint32

	// proof — связка мастер-ключей этого узла; кладётся в каждый конверт,
	// иначе получатель не сможет ни собрать X3DH, ни проверить личность.
	proof *crypto.IdentityProof

	// Приёмный путь (Шаг 3-b): подписка на оверлей и канал событий в UI.
	inbound   <-chan yggnode.Packet
	cancelInb func()
	events    chan *yggproto.CoreEvent
	eventSeq  uint64
	rxStats   inboundStats

	// Исходящая очередь (Шаг 3-c-2): сканер будется таймером и пинком из
	// ConnectPeer/SendMessage, а outDone гасит его в stopNode.
	outKick  chan struct{}
	outDone  chan struct{}
	outStats outboxStats

	// Персистентность сеансов (Шаг 3-c-3).
	sessStats sessionStats

	// Персистентность контактов (Шаг 5-b).
	contactStats contactStats

	// Персистентность prekey-материала (Шаг 6-e).
	prekeyStats prekeyStats

	// Слепой store-and-forward (Шаг 6). sfRelays — Ed25519-ключи нод-хранителей,
	// sfRelayEnabled — готовность самому держать чужую почту. Ключ узла и есть
	// его адрес в оверлее, поэтому отдельного IPv6 для релея нет.
	sfRelayEnabled bool
	sfRelays       [][]byte
	sfStats        sfStats
	// sfNextPoll — когда сканеру снова ходить на релеи за своей почтой.
	sfNextPoll time.Time
	// sfTaken — следы уже выданных этому узлу конвертов (свежесть по
	// sfTakenOrder). Релей пересылает письмо в окне повторной выдачи, а
	// вторичное раскрытие того же шифртекста приёмной цепочке только вредит:
	// шаг рэтчета ушёл вперёд, и повтор отвергается как мусор — уже счётчиком
	// отказов, которых на самом деле не было.
	sfTaken      map[string]bool
	sfTakenOrder []string

	// Раздача и обмен PreKeyBundle поверх оверлея (Шаг 7). serveBundles —
	// готовность узла отвечать на bundle_req; по умолчанию включена, иначе
	// контакт нельзя завести без пересылки карты байтами вне сети.
	serveBundles bool
	// bundleLastServed — когда узлу с этим ключом в последний раз отдавали
	// бандл. Запрос ничем не оплачен (он не кладёт почту и не шифруется),
	// поэтому одна нода не вправе выуживать prekey-материал чаще, чем раз в
	// bundleServeInterval: каждый ответ выпускает новый one-time prekey.
	bundleLastServed map[string]time.Time
	// bundlePending — незакрытые fetch-запросы. Ответ приходит в приёмный цикл
	// асинхронно, как и почта с релея, поэтому запрос живёт до ответа или до
	// истечения bundleFetchTimeout.
	bundlePending map[uint64]*bundleFetch
	bundleSeq     uint64
	bundleStats   bundleStats

	antiDpi yggproto.AntiDpiMode
	running bool

	// peersSig — сигнатура того набора пиров, который ядро уже объявило событием
	// peers_updated, а peersSigSet — что такое объявление вообще было (пустой
	// список тоже нужно донести до UI ровно один раз).
	peersSig    string
	peersSigSet bool

	// dpiSig — то же для anti_dpi_status: событие описывает исходящие линки, и
	// объявляется только когда их вершина (URI, SNI, поднялся ли, ошибка)
	// меняется. Пустой список линков тоже объявляется ровно один раз.
	dpiSig    string
	dpiSigSet bool
}

// sessionKey — имя сеанса в карте. peerX — мастер-X25519 собеседника,
// conversationID — имя диалога ровно то, что лежит в конверте на проводе:
// получатель обязан вернуть отправителю его имя, иначе тот не найдёт сеанс.
func sessionKey(peerX []byte, conversationID string) string {
	return hexKey(peerX) + "|" + conversationID
}

// inboundStats — честный счётчик того, что приёмный цикл не расшифровал или
// не сумел обработать; молча терять пакеты нельзя.
type inboundStats struct {
	received uint64
	ignored  uint64 // не наш конверт (AEAD не открылся) или мусор
	rejected uint64 // не прошёл проверку proof/PoW/подписи
	// routeMismatch — пакет заявленной идентичности пришёл с узла, который не
	// заверен в карте этого контакта. Криптография тут в порядке (proof.open
	// работает), разъезжается именно адрес доставки, поэтому такие пакеты
	// отбрасываются молча и без движения рэтчета.
	routeMismatch uint64
	applied       uint64 // сообщение распознано и сохранено
	duplicated    uint64 // повтор по message_id
	acked         uint64 // принято DeliveryAckContent
	ackSent       uint64 // отправлено подтверждение доставки
	ackSkipped    uint64 // подтверждение не ушло: нет маршрута либо ошибка записи
	evDropped     uint64 // событийный канал переполнен
}

// outboxStats — честный счётчик исходящей очереди: «сообщение не ушло» должно
// быть видно так же, как отказ приёма.
type outboxStats struct {
	queued     uint64 // конверт лёг в очередь: маршрута нет или запись не прошла
	written    uint64 // отложенный конверт ушёл в сеть
	failed     uint64 // маршрут был, но WriteTo не принял столько байт, сколько надо
	invalid    uint64 // в записи очереди нет годного ключа ноды — такое не отправить
	persistErr uint64 // хранилище не дало изменить запись очереди: учёт попыток поехал
}

// sessionStats — учёт состояний сеансов на диске. Кривой снимок нельзя ни
// принять молча, ни выбросить молча: это диалог, который пользователь ждёт.
type sessionStats struct {
	saved      uint64 // шаг цепочки записан в хранилище
	saveErr    uint64 // хранилище не приняло запись: отправка на этом остановилась
	restored   uint64 // сеанс поднят из хранилища при старте
	restoreErr uint64 // состояние не разобралось или не прошло проверку
	// skippedOwner — строки, привязанные к другой мастер-идентичности (смена
	// фразы): грузить их нельзя, но и молча выбрасывать незачем.
	skippedOwner uint64
}

// contactStats — учёт карт контактов на диске. Карта, которую не удалось
// сохранить или поднять, — это диалог без адреса: врать про «контакт добавлен»
// нельзя, поэтому каждая ветка имеет свой счётчик.
type contactStats struct {
	persisted  uint64 // карта записана в хранилище
	restoreErr uint64 // снимок не разобрался или не прошёл verifyContact
	// skippedKey — ключ строки не совпал с x25519-ключом в самой карте: база
	// подделана или повреждена, доверять такой связке нельзя.
	skippedKey uint64
	restored   uint64
}

// New создаёт движок без запуска сети — активация происходит в StartNode.
func New() *Engine {
	return &Engine{
		opkPriv:          make(map[uint32]*ecdh.PrivateKey),
		opkConsumed:      make(map[uint32]bool),
		contacts:         make(map[string]*yggproto.ContactCard),
		sessions:         make(map[string]*crypto.DoubleRatchetSession),
		ekPub:            make(map[string][]byte),
		opkUsed:          make(map[string]uint32),
		sfTaken:          make(map[string]bool),
		bundleLastServed: make(map[string]time.Time),
		bundlePending:    make(map[uint64]*bundleFetch),
		events:           make(chan *yggproto.CoreEvent, eventQueue),
		outKick:          make(chan struct{}, 1),
	}
}

// eventQueue и inboundQueue — глубины очередей приёма. Переполнение не должно
// блокировать ни оверлейный насос, ни приёмный цикл: лишнее считается
// потерянным и остаётся в счётчике.
const (
	eventQueue   = 64
	inboundQueue = 256
)

func defaultDataDir() string {
	return filepath.Join(os.TempDir(), "ygggram", "data")
}

func defaultCacheDir() string {
	return filepath.Join(os.TempDir(), "ygggram", "cache")
}

// Handle исполняет protobuf-запрос и возвращает protobuf-ответ.
func (e *Engine) Handle(req *yggproto.CoreRequest) *yggproto.CoreResponse {
	resp := &yggproto.CoreResponse{RequestId: req.GetRequestId()}

	e.mu.Lock()
	defer e.mu.Unlock()

	var err error
	switch r := req.Request.(type) {
	case *yggproto.CoreRequest_StartNode:
		out := &yggproto.NodeStatusResp{}
		resp.Response, err = &yggproto.CoreResponse_NodeStatus{NodeStatus: out}, e.startNode(r.StartNode.GetConfig())
		if err == nil {
			err = e.nodeStatus(out)
		}
	case *yggproto.CoreRequest_StopNode:
		err = e.stopNode()
	case *yggproto.CoreRequest_GetNodeStatus:
		out := &yggproto.NodeStatusResp{}
		resp.Response, err = &yggproto.CoreResponse_NodeStatus{NodeStatus: out}, e.nodeStatus(out)
	case *yggproto.CoreRequest_GetPeers:
		out := &yggproto.PeerListResp{}
		resp.Response, err = &yggproto.CoreResponse_PeerList{PeerList: out}, e.peerList(out)
	case *yggproto.CoreRequest_ConnectPeer:
		out := &yggproto.PeerListResp{}
		resp.Response, err = &yggproto.CoreResponse_PeerList{PeerList: out}, e.connectPeer(r.ConnectPeer)
		if err == nil {
			err = e.peerList(out)
		}
	case *yggproto.CoreRequest_DisconnectPeer:
		out := &yggproto.PeerListResp{}
		resp.Response, err = &yggproto.CoreResponse_PeerList{PeerList: out}, e.disconnectPeer(r.DisconnectPeer)
		if err == nil {
			err = e.peerList(out)
		}
	case *yggproto.CoreRequest_AddContact:
		err = e.addContact(r.AddContact.GetContact())
	case *yggproto.CoreRequest_GetPrekeyBundle:
		out := &yggproto.PreKeyBundle{}
		resp.Response, err = &yggproto.CoreResponse_PrekeyBundle{PrekeyBundle: out}, e.preKeyBundle(out)
	case *yggproto.CoreRequest_GenerateIdentity:
		out := &yggproto.IdentityResp{}
		resp.Response, err = &yggproto.CoreResponse_Identity{Identity: out}, generateIdentity(out)
	case *yggproto.CoreRequest_ListConversations:
		out := &yggproto.ConversationListResp{}
		resp.Response, err = &yggproto.CoreResponse_Conversations{Conversations: out}, e.listConversations(r.ListConversations.GetLimit(), out)
	case *yggproto.CoreRequest_GetMessages:
		out := &yggproto.MessageListResp{}
		resp.Response, err = &yggproto.CoreResponse_Messages{Messages: out}, e.getMessages(r.GetMessages, out)
	case *yggproto.CoreRequest_ListContacts:
		out := &yggproto.ContactListResp{}
		resp.Response, err = &yggproto.CoreResponse_Contacts{Contacts: out}, e.listContacts(out)
	case *yggproto.CoreRequest_SfCollect:
		out := &yggproto.SfCollectResp{}
		resp.Response, err = &yggproto.CoreResponse_SfCollect{SfCollect: out}, e.sfCollect(out)
	case *yggproto.CoreRequest_FetchContact:
		out := &yggproto.FetchContactResp{}
		resp.Response, err = &yggproto.CoreResponse_FetchContact{FetchContact: out}, e.fetchContact(r.FetchContact, out)
	case *yggproto.CoreRequest_SendMessage:
		out := &yggproto.SendMessageResp{}
		resp.Response, err = &yggproto.CoreResponse_SendMessage{SendMessage: out}, e.sendMessage(r.SendMessage, out)
	case *yggproto.CoreRequest_CreateBackup:
		out := &yggproto.BackupResp{}
		resp.Response, err = &yggproto.CoreResponse_Backup{Backup: out}, e.createBackup(r.CreateBackup, out)
	case *yggproto.CoreRequest_RestoreBackup:
		out := &yggproto.BackupResp{}
		resp.Response, err = &yggproto.CoreResponse_Backup{Backup: out}, e.restoreBackup(r.RestoreBackup, out)
	default:
		err = fmt.Errorf("неизвестный тип запроса %T", req.Request)
	}

	if err != nil {
		resp.Success = false
		resp.ErrorMessage = err.Error()
		return resp
	}
	resp.Success = true
	return resp
}

// InvokeBytes — точечный вход для cmd/libygggram: protobuf на входе, protobuf на выходе.
func (e *Engine) InvokeBytes(payload []byte) ([]byte, error) {
	req := &yggproto.CoreRequest{}
	if err := proto.Unmarshal(payload, req); err != nil {
		return nil, fmt.Errorf("rpc: разбор CoreRequest: %w", err)
	}
	out, err := proto.Marshal(e.Handle(req))
	if err != nil {
		return nil, fmt.Errorf("rpc: сборка CoreResponse: %w", err)
	}
	return out, nil
}

func (e *Engine) startNode(cfg *yggproto.NodeConfig) error {
	if cfg == nil {
		return errors.New("rpc: StartNode без NodeConfig")
	}
	if e.running {
		return nil
	}

	e.dataDir = value(cfg.GetDataDir(), defaultDataDir())
	e.cacheDir = value(cfg.GetCacheDir(), defaultCacheDir())
	e.antiDpi = cfg.GetAntiDpiMode()

	mnemonic := cfg.GetMnemonicSeed()
	if mnemonic == "" {
		return errors.New("rpc: пустой mnemonic_seed — идентичность нельзя вывести")
	}
	master, err := crypto.MasterIdentityFromMnemonic(mnemonic, "")
	if err != nil {
		return fmt.Errorf("rpc: вывод мастер-идентичности: %w", err)
	}
	e.master = master

	e.device, err = master.IssueDeviceIdentity("primary", "ygggram-core")
	if err != nil {
		return fmt.Errorf("rpc: выпуск device-идентичности: %w", err)
	}

	// Связка мастер-ключей идёт в каждый конверт: без неё получатель не сможет
	// ни собрать X3DH, ни проверить, что Ed25519 и X25519 одного владельца.
	e.proof, err = crypto.NewIdentityProof(master, e.device)
	if err != nil {
		return fmt.Errorf("rpc: связка мастер-ключей: %w", err)
	}

	e.nodeKey = nodeKeyFromSeed(master, cfg.GetEd25519NodePrivkey())

	// Перезапуск может сменить mnemonic, а с ним и все ключи: старые prekey
	// обязаны остаться в прошлом, иначе чужое состояние сессий утекло бы в новый
	// контекст. Сеансы при этом грузятся из базы по фильтру владельца — под другую
	// фразу они сами собой не подпадают. Карты контактов чужой идентичности не
	// несут, но держим их в контексте той же базы: при другой фразе хранилище
	// вообще не откроется, поэтому память очищаем и наполняем заново. Сам
	// prekey-материал живёт в базе и поднимается после её открытия (Шаг 6-e):
	// до хранилища непонятно, выпускать новый набор или продолжать старый.
	e.sessions = make(map[string]*crypto.DoubleRatchetSession)
	e.ekPub = make(map[string][]byte)
	e.opkUsed = make(map[string]uint32)
	e.contacts = make(map[string]*yggproto.ContactCard)
	// Роль в S&F и список релеев принадлежат конкретному запуску: держать их
	// через рестарт нельзя, иначе почта клалась бы релеям из прежнего конфига, а
	// счётчики отчитывались бы за прошлый заход. Негодный ключ — ошибка старта:
	// «релеи настроены, но письмо никуда не легло» пользователь заметил бы
	// только по недоставленному сообщению.
	relays, err := sfRelayKeys(cfg)
	if err != nil {
		return err
	}
	e.sfRelayEnabled = cfg.GetSfRelayEnabled()
	e.sfRelays = relays
	e.sfStats = sfStats{}
	e.sfNextPoll = time.Time{}
	// Раздача бандлов и незакрытые fetch-запросы принадлежат этому запуску:
	// держать их через рестарт нельзя — запрос ушёл в сеть прошлого сокета, и
	// ответ на него уже не придёт никогда.
	// Геттер у optional-поля отдаёт false вместо «не задано», поэтому роль
	// читается из самого поля: nil — значит по умолчанию, а по умолчанию
	// раздача включена.
	e.serveBundles = cfg.ServePrekeyBundles == nil || *cfg.ServePrekeyBundles
	e.bundleLastServed = make(map[string]time.Time)
	e.bundlePending = make(map[uint64]*bundleFetch)
	e.bundleStats = bundleStats{}
	// Неудачный заход (нода не собралась или старт не прошёл) оставлял хранилище
	// открытым, а повторный StartNode работал бы по тому же каталогу: у дневника
	// один владелец, и второй экземпляр был бы отвергнут. Освобождаем то, что
	// держали, прежде чем открывать заново.
	if e.db != nil {
		if err := e.db.Close(); err != nil {
			return fmt.Errorf("rpc: освобождение хранилища после неудачного старта: %w", err)
		}
		e.db = nil
	}

	key := storageKey(master, cfg.GetMasterKey_256())
	e.db, err = storage.OpenEncryptedDB(e.dataDir, key, e.device.Certificate.DeviceID)
	if err != nil {
		return fmt.Errorf("rpc: открытие хранилища: %w", err)
	}
	// Ключ запоминается именно здесь, а не вычисляется по требованию: бэкап
	// обязан проверять восстановленную базу тем же ключом, которым открыта
	// живая. cfg.GetMasterKey_256() умеет задать его явно, так что вывод «из
	// seed кошелька» был бы правдой не всегда.
	e.vaultKey = &key
	if err := e.restorePrekeys(); err != nil {
		return err
	}
	e.restoreSessions()
	e.restoreContacts()

	listen, err := listenURIs(cfg, e.antiDpi)
	if err != nil {
		return fmt.Errorf("rpc: listen_address: %w", err)
	}
	peers, err := peerURIs(cfg.GetBootstrapPeers(), e.antiDpi)
	if err != nil {
		return fmt.Errorf("rpc: bootstrap_peers: %w", err)
	}

	node, err := yggnode.NewYggNode(yggnode.NodeOptions{
		PrivateKey:    e.nodeKey,
		Listen:        listen,
		Peers:         peers,
		GroupPassword: cfg.GetAntiDpiPassword(),
		Multicast:     cfg.GetEnableLanMulticast(),
		NodeInfo:      map[string]interface{}{"app_name": "YggGram", "platform": runtime.GOOS},
	})
	if err != nil {
		return fmt.Errorf("rpc: сборка оверлей-ноды: %w", err)
	}
	if err := node.Start(); err != nil {
		return fmt.Errorf("rpc: старт оверлей-ноды: %w", err)
	}

	e.node = node
	e.running = true
	// Новая нода — новая вершина: прошлая сигнатура пиров к ней не относится, и
	// первое объявление (пусть даже пустой список) обязано состояться. Тем же
	// сбросом начинается и работа сканера: он живёт на той же сигнатуре.
	e.peersSig, e.peersSigSet = "", false
	e.dpiSig, e.dpiSigSet = "", false

	// Приёмный путь обязателен: yggnode уже крутит ReadFrom (без него
	// ironwood не отправляет ничего), а сюда долетают готовые пакеты.
	e.inbound, e.cancelInb = node.Subscribe(inboundQueue)
	go e.receiveLoop()

	// Сканер исходящей очереди: конверты, не ушедшие в сеть, лежат в
	// зашифрованной базе и должны выйти, как только маршрут появится, — в том
	// числе после перезапуска ядра.
	e.outDone = make(chan struct{})
	go e.outboxLoop(e.outDone)
	e.kickOutbox()
	return nil
}

func (e *Engine) stopNode() error {
	if !e.running {
		return nil
	}
	// Оба цикла гасим до остановки ноды: отписка — единственная гарантия, что
	// приёмный цикл не получит пакет после Stop, а сканер очереди не тронет
	// сокет закрытой ноды. running=false стоит раньше Stop именно потому, что
	// drainOutbox проверяет его под тем же e.mu.
	e.running = false
	if e.cancelInb != nil {
		e.cancelInb()
		e.cancelInb = nil
		e.inbound = nil
	}
	if e.outDone != nil {
		close(e.outDone)
		e.outDone = nil
	}
	if err := e.node.Stop(); err != nil {
		return fmt.Errorf("rpc: остановка ноды: %w", err)
	}
	// Хранилище освобождается в самом конце: приёмный цикл и сканер очереди к
	// этому моменту уже остановлены, а все записи в базе идут под e.mu, который
	// этот вызов держит. Пока файл дневника занят, повторный StartNode на том же
	// каталоге не откроет его — у дневника один владелец, и второй экземпляр писал
	// бы кадры с уже использованными номерами.
	stored := e.db
	e.db = nil
	if stored != nil {
		if err := stored.Close(); err != nil {
			return fmt.Errorf("rpc: освобождение хранилища: %w", err)
		}
	}
	return nil
}

// Events — канал асинхронных событий ядра. C-обёртка вычитывает его и шлёт в
// UI; никто не читает — события считаются потерянными, приёмный цикл не
// встанет.
func (e *Engine) Events() <-chan *yggproto.CoreEvent { return e.events }

// receiveLoop читает подписку оверлея, пока ядро не остановлено (cancel
// закрывает канал).
func (e *Engine) receiveLoop() {
	for pkt := range e.inbound {
		e.handleInbound(pkt)
	}
}

// outboxInterval — как часто сканер сам, без пинка, смотрит очередь. Пир мог
// появиться и без ConnectPeer (bootstrap, мультикаст, чужое входящее
// пирование), поэтому одного пинка не хватает.
const outboxInterval = 2 * time.Second

// outboxMaxSends — сколько раз отложенный конверт ложится в сокет, после чего
// запись из очереди уходит. Один дубль нужен: первый пакет на ещё не прогретой
// сессии ironwood оверлей теряет регулярно (потому и переотправляют SendMessage
// в тестах Шага 3). Больше одного дубля не нужно: дословно повторённый конверт
// получатель раскроет лишь один раз, а остальные сочтёт replay.
const outboxMaxSends = 2

// kickOutbox будит сканер, не блокируясь: пинкуют его пути, которые уже держат
// e.mu, а сканер как раз этот лок и ждёт — ожидание здесь было бы
// взаимоблокировкой.
func (e *Engine) kickOutbox() {
	select {
	case e.outKick <- struct{}{}:
	default:
	}
}

// outboxLoop — единственная горутина, которая переотправляет очередь и делает
// фоновую работу S&F: выбросить просроченную чужую почту и сходить на релеи за
// своей. done закрывает stopNode, поэтому цикл гарантированно умирает до Stop
// ноды. Тем же тиком он объявляет изменение списка пиров (publishPeers).
func (e *Engine) outboxLoop(done <-chan struct{}) {
	ticker := time.NewTicker(outboxInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
		case <-e.outKick:
		}
		e.drainOutbox()
		e.sfTick()
		e.publishPeers()
		e.publishAntiDpi()
	}
}

// drainOutbox пересылает всё, что ждёт маршрута. Байты конверта берутся из базы
// дословно: повторный seal сдвигал бы отправную цепочку ради пакетов, которые
// никуда не ушли, и раскручивал бы у получателя skipped-ключи — при том что
// дедупликация по message_id уже встроена в приёмный путь.
//
// Порядок — от старых к новым: сообщение обязано дойти в той же очерёдности, в
// которой его напечатали.
//
// Запись, до адресата которой маршрута по-прежнему нет, попутно кладётся слепым
// релеям: хранитель мог появиться в таблице маршрутов позже самого адресата, а
// без этого почта жила бы ровно до ближайшей остановки процесса.
func (e *Engine) drainOutbox() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running || e.node == nil || e.db == nil {
		return
	}
	for _, rec := range e.db.PendingOutbound() {
		if len(rec.NodeKey) != ed25519.PublicKeySize {
			// Ключ ноды в записи испорчен — такое возможно лишь в базе,
			// написанной другой версией. Отправить это нечем, и крутить
			// попытку впустую бессмысленно.
			e.outStats.invalid++
			e.removeOutbound(rec.MessageID)
			continue
		}
		nodeKey := ed25519.PublicKey(append([]byte(nil), rec.NodeKey...))
		if !e.node.HasRouteTo(nodeKey) {
			// Адресата в сети нет. Конверт всё это время может лежать у
			// слепых релеев — если до них маршрут есть уже сейчас, а при первой
			// попытке не был.
			e.relayPending(rec)
			continue
		}
		written, err := e.node.PacketConn().WriteTo(rec.Envelope, yggnode.AddrForPublicKey(nodeKey))
		if err != nil || written != len(rec.Envelope) {
			// Маршрут есть, а пакет не лёг. Попыткой это не считается:
			// в сокет ничего не ушло, и дубль тут не нужен.
			e.outStats.failed++
			continue
		}
		e.outStats.written++
		sends := rec.Sends + 1
		if sends >= outboxMaxSends {
			e.removeOutbound(rec.MessageID)
		} else if err := e.db.RecordOutboundSend(rec.MessageID, time.Now().UnixMilli()); err != nil {
			// Счётчик отправок не записан: сообщение ушло, но учёт поехал.
			e.outStats.persistErr++
		}
		if rec.Sends == 0 {
			// Статус «ушло в сеть» объявляется по первой отправке: повторный
			// дубль ничего нового UI не сообщает.
			e.emit(&yggproto.CoreEvent{
				EventId:     e.nextEventID(),
				TimestampMs: uint64(time.Now().UnixMilli()),
				Event: &yggproto.CoreEvent_MessageStatus{MessageStatus: &yggproto.MessageStatusEvent{
					MessageId:        rec.MessageID,
					ConversationId:   rec.ConversationID,
					WrittenToOverlay: true,
				}},
			})
		}
	}
}

// removeOutbound гасит запись очереди. Ошибка хранилища молча проходит мимо:
// терять из-за неё отправленное сообщение нельзя, а счётчик честно говорит,
// что учёт поехал.
func (e *Engine) removeOutbound(messageID string) {
	if err := e.db.CompleteOutbound(messageID); err != nil {
		e.outStats.persistErr++
	}
}

// handleInbound — разбор кадра оверлея и разметка по его вариантам: разговорный
// конверт, кладка на релей, запрос почты и ответ релея. Мусор и неизвестный
// кадр считаются здесь же: пройти мимо пакета, не прибавив счётчик, нельзя.
func (e *Engine) handleInbound(pkt yggnode.Packet) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rxStats.received++
	if !e.running || e.master == nil {
		e.rxStats.ignored++
		return
	}

	frame, err := deframe(pkt.Payload)
	if err != nil {
		e.rxStats.ignored++
		return
	}
	switch body := frame.Body.(type) {
	case *yggproto.WireFrame_Chat:
		e.acceptChat(body.Chat, pkt.From, false)
	case *yggproto.WireFrame_SfPut:
		e.acceptSFPut(body.SfPut)
	case *yggproto.WireFrame_SfPull:
		e.acceptSFPull(body.SfPull, pkt.From)
	case *yggproto.WireFrame_SfReply:
		e.acceptSFReply(body.SfReply, pkt.From)
	case *yggproto.WireFrame_BundleReq:
		e.acceptBundleReq(body.BundleReq, pkt.From)
	case *yggproto.WireFrame_BundleResp:
		e.acceptBundleResp(body.BundleResp, pkt.From)
	default:
		// Кадр без варианта — чужая версия протокола. Счесть его «не разобрался
		// конверт» значило бы соврать про шифрование, хотя дело в оболочке.
		e.rxStats.ignored++
	}
}

// acceptChat — весь путь принятого конверта: проверка личности,
// X3DH/Double Ratchet, хранилище и событие в UI; для DeliveryAckContent вместо
// хранилища — отметка доставки. Чужие пакеты (открывание AEAD не удалось) молча
// проходят мимо — слепая почтовая коробка не даёт иначе понять, что конверт
// был не нам.
//
// viaRelay означает, что байты принёс S&F-хранитель, а не сам отправитель: у
// чужого узла адреса собеседника нет в помине, поэтому адресные ворота Шага 5-a
// тут пропускаются (см. acceptSFReply).
func (e *Engine) acceptChat(wire *yggproto.SealedEnvelope, from ed25519.PublicKey, viaRelay bool) {
	if !crypto.VerifyHashcashPoW(mailboxOf(wire.GetBlindToken()), wire.GetPowNonce(), powDifficulty) {
		// Без доказательства работы конверт — кандидат на спам-очередь S&F,
		// расшифровывать его незачем.
		e.rxStats.rejected++
		return
	}

	payload, err := crypto.UnpackSealedEnvelope(e.master.MasterX25519, envelopeFromWire(wire))
	if err != nil {
		e.rxStats.ignored++
		return
	}

	senderX := payload.Proof.MasterX25519

	// Связка «идентичность ↔ адрес» обязана остаться той же, что заверена при
	// AddContact (Шаг 3-c-4 работает в обе стороны). Конверт из другой точки
	// сети криптографически честный: proof.open подписан мастер-ключом
	// отправителя. Но если знакомый человек заявлен в карте на узле N, а
	// байты пришли с узла M, продолжение диалога означает, что мы отвечаем
	// одному адресу, а слушаем другой — и обратно. Разъезд возможен только
	// когда узел настроен явно другим ключом (из той же фразы он выводится
	// детерминированно), поэтому такой пакет отбрасывается до шага рэтчета:
	// ни сессия, ни база, ни ACK не должны его увидеть.
	if !viaRelay {
		if card, ok := e.contacts[hexKey(senderX)]; ok && !bytes.Equal(card.GetEd25519Pubkey(), from) {
			e.rxStats.routeMismatch++
			return
		}
	}

	key := sessionKey(senderX, payload.ConversationID)
	session, known := e.sessions[key]
	if !known {
		session, err = e.openReceiverSession(payload)
		if err != nil {
			e.rxStats.rejected++
			return
		}
	}

	inner, ekPub, err := DecodeRatchetMessage(payload.InnerRatchetMsg)
	if err != nil {
		e.rxStats.rejected++
		return
	}
	plaintext, err := session.Decrypt(*inner, []byte(payload.ConversationID))
	if err != nil {
		// Сессия в map не появляется, пока сообщение не открылось: атака
		// мусором не может закрепить битый рэтчет на настоящем диалоге.
		e.rxStats.rejected++
		return
	}

	decoded := &yggproto.ChatMessage{}
	if err := proto.Unmarshal(plaintext, decoded); err != nil {
		e.rxStats.rejected++
		return
	}

	// Подтверждение доставки — служебное сообщение того же сеанса: оно
	// обновляет статус исходящего, но не становится строкой диалога.
	if ack := decoded.GetDeliveryAck(); ack != nil {
		e.rxStats.acked++
		e.markDelivered(ack.GetAcknowledgesMessageId(), payload.ConversationID, decoded.GetTimestampMs())
		return
	}

	// Диалог на стороне получателя именуется по identity key отправителя:
	// conversation_id в конверте — это имя отправителя, и двум клиентам оно
	// не принадлежит.
	localConv := inboundConversationID(senderX)
	msgID := decoded.GetMessageId()
	if msgID == "" {
		msgID = fmt.Sprintf("%s:%d", key, payload.Timestamp)
	}
	if e.db.HasMessage(msgID) {
		e.rxStats.duplicated++
		return
	}

	// Ключ one-time считаем потраченным до записи в базу: X3DH-сессия уже
	// собрана и открылась, а сам ключ остаётся в opkPriv (maxKeptOPKs), так что
	// повторная доставка тем же prekey расшифруется и погасится дедупликацией.
	//
	// Сеанс регистрируется сразу после удачного Decrypt: шаг рэтчета уже
	// сдвинул цепочку, и выброшенная сессия означала бы вечный рассинхрон с
	// отправителем. В этой же сессии у получателя появляется отправная
	// цепочка — ею и шифруется ACK.
	if !known {
		if err := e.rotateOPK(payload.OneTimePreKeyID); err != nil {
			e.rxStats.rejected++
			return
		}
		e.sessions[key] = session
		e.ekPub[key] = ekPub
	}
	// Приёмный шаг цепочки тоже фиксируется до того, как что-то увидит UI.
	// Ошибка записи уже учтена в sessStats.saveErr и обработку не срывает:
	// приёмная цепочка догоняет пропуск сама (skipMessageKeys), а выбросить уже
	// раскрытое сообщение было бы хуже, чем отстать на один счётчик.
	_ = e.persistSession(key)

	// Узел, который считаем источником и которому отвечаем ACK'ом. Живой пакет
	// приходит от собеседника, поэтому адрес берём у оверлея. Конверт с релея
	// назад тем же путём не дойдёт: хранитель выбросил ящик и каналом отправителя
	// не был. Тогда адресатом ACK делаем узел из карты контакта — связка
	// «идентичность ↔ узел» заверена при AddContact, а сама личность
	// подтверждена proof внутри конверта. Без карты ответа нет: выдумывать адрес
	// нельзя, и честнее признать ACK пропущенным.
	ackNode := from
	if viaRelay {
		ackNode = nil
		if card, ok := e.contacts[hexKey(senderX)]; ok && len(card.GetEd25519Pubkey()) == ed25519.PublicKeySize {
			ackNode = card.GetEd25519Pubkey()
		}
	}
	reportNode := ackNode
	if reportNode == nil {
		reportNode = from
	}

	stored := &storage.StoredMessage{
		MessageID:      msgID,
		ConversationID: localConv,
		// Источник — ключ узла, с которого пришёл пакет: его аутентифицирует
		// сам оверлей, в отличие от поля внутри сообщения. Для почты с релея это
		// заверенный узел отправителя, а если он неизвестен — адрес носителя:
		// он хотя бы правда принёс байты.
		SenderID:    yggnode.AddressForKey(reportNode).String(),
		TimestampMS: inboundTimestamp(decoded, payload.Timestamp),
		Body:        preview(decoded),
		IsOutbound:  false,
	}
	if err := e.db.SaveMessage(stored); err != nil {
		e.rxStats.rejected++
		return
	}

	decoded.ConversationId = localConv
	decoded.IsOutbound = false
	decoded.SenderNodeIpv6 = stored.SenderID
	decoded.SenderX25519Pubkey = senderX
	e.rxStats.applied++
	e.emit(&yggproto.CoreEvent{
		EventId:     e.nextEventID(),
		TimestampMs: uint64(time.Now().UnixMilli()),
		Event: &yggproto.CoreEvent_IncomingMessage{IncomingMessage: &yggproto.IncomingMessageEvent{
			Message: decoded,
			// proof прошёл UnpackSealedEnvelope: и устройство, и связка
			// мастер-ключей отправителя проверены подписями.
			SealedSenderVerified: true,
		}},
	})

	if ackNode == nil {
		// Почту принёс релей, а отправителя в контактах нет: шифровать ACK
		// нечем и некому. Сообщение при этом сохранено — доставка в UI всё
		// равно видна, а вот у отправителя статус останется «не подтверждён».
		e.rxStats.ackSkipped++
		return
	}
	// ACK уходит после события: получатель уже видел сообщение, а отправитель
	// получит подтверждение отдельным пакетом тем же сеансом.
	e.sendDeliveryAck(senderX, ackNode, payload.ConversationID, key, msgID)
}

// markDelivered применяет сквозное подтверждение доставки к исходящему
// сообщению и объявляет MessageStatusEvent. Вызывается под e.mu.
func (e *Engine) markDelivered(messageID, wireConv string, ackTimestampMs uint64) {
	if messageID == "" {
		e.rxStats.ignored++
		return
	}
	if e.db.IsDelivered(messageID) {
		e.rxStats.duplicated++
		return
	}
	at := int64(ackTimestampMs)
	if at == 0 {
		at = time.Now().UnixMilli()
	}
	conv, known, err := e.db.MarkDelivered(messageID, at)
	if err != nil {
		e.rxStats.rejected++
		return
	}
	if !known {
		// ACK не к нашему исходящему: чужой message_id, уже удалённое или
		// входящее сообщение. Статус выдумывать нельзя.
		e.rxStats.ignored++
		return
	}
	if conv != wireConv {
		// Имя диалога в конверте и база разошлись: событие показываем по базе,
		// счётчик честно отмечает несоответствие.
		e.rxStats.ignored++
	}
	// Подтверждённая доставка снимает сообщение из исходящей очереди: дублировать
	// его больше незачем.
	e.removeOutbound(messageID)
	e.emit(&yggproto.CoreEvent{
		EventId:     e.nextEventID(),
		TimestampMs: uint64(time.Now().UnixMilli()),
		Event: &yggproto.CoreEvent_MessageStatus{MessageStatus: &yggproto.MessageStatusEvent{
			MessageId:      messageID,
			ConversationId: conv,
			Delivered:      true,
			AckTimestampMs: uint64(at),
		}},
	})
}

// sendDeliveryAck отвечает отправителю тем же сеансом Double Ratchet, что
// открыл пришедшее сообщение. ACK — обычное шифрованное сообщение, поэтому
// подделать «доставлено» может только тот, кто имеет получательские ключи.
// Вызывается под e.mu; молча не отправляет, если маршрута обратно нет.
func (e *Engine) sendDeliveryAck(peerX []byte, from ed25519.PublicKey, wireConv, key, messageID string) {
	ack := &yggproto.ChatMessage{
		MessageId:      "ack:" + messageID,
		ConversationId: wireConv,
		TimestampMs:    uint64(time.Now().UnixMilli()),
		Content:        &yggproto.ChatMessage_DeliveryAck{DeliveryAck: &yggproto.DeliveryAckContent{AcknowledgesMessageId: messageID}},
	}
	plaintext, err := proto.Marshal(ack)
	if err != nil {
		e.rxStats.ackSkipped++
		return
	}
	sealed, err := e.seal(peerX, wireConv, key, plaintext)
	if err != nil {
		e.rxStats.ackSkipped++
		return
	}
	// Адрес обратно берём у оверлея (from — ключ узла, с которого реально
	// пришёл пакет), а не из контактной карты: получателя может и не быть в
	// списке контактов.
	if !e.node.HasRouteTo(from) {
		e.rxStats.ackSkipped++
		return
	}
	written, err := e.node.PacketConn().WriteTo(sealed, yggnode.AddrForPublicKey(from))
	if err != nil || written != len(sealed) {
		e.rxStats.ackSkipped++
		return
	}
	e.rxStats.ackSent++
}

// openReceiverSession собирает X3DH на стороне получателя и отдаёт сессию, не
// регистрируя её: в map она попадает только когда сообщение открылось. Иначе
// один неудачный конверт (мусор, чужой prekey) занял бы ключ настоящего
// диалога битым рэтчетом.
func (e *Engine) openReceiverSession(payload *crypto.SealedPayload) (*crypto.DoubleRatchetSession, error) {
	_, ekPub, err := DecodeRatchetMessage(payload.InnerRatchetMsg)
	if err != nil {
		return nil, fmt.Errorf("rpc: разбор рэтчет-сообщения: %w", err)
	}

	senderIK, err := ecdh.X25519().NewPublicKey(payload.Proof.MasterX25519)
	if err != nil {
		return nil, fmt.Errorf("rpc: identity key отправителя: %w", err)
	}
	ek, err := ecdh.X25519().NewPublicKey(ekPub)
	if err != nil {
		return nil, fmt.Errorf("rpc: эфемерный ключ X3DH: %w", err)
	}

	var opk *ecdh.PrivateKey
	if id := payload.OneTimePreKeyID; id != 0 {
		opk = e.opkPriv[id]
		if opk == nil {
			return nil, fmt.Errorf("rpc: one-time prekey %d неизвестен", id)
		}
	}

	shared, err := crypto.PerformX3DHReceiver(e.master.MasterX25519, e.spkPriv, opk, senderIK, ek)
	if err != nil {
		return nil, fmt.Errorf("rpc: X3DH получателя: %w", err)
	}
	return crypto.NewReceiverSession(shared, e.spkPriv), nil
}

// maxKeptOPKs — сколько израсходованных one-time prekey держим. Отправитель
// пользуется опубликованным бандлом и может собрать сессию на уже потраченном
// OPK, поэтому сразу удалять нельзя; но и бесконечный склад ключей не нужен.
const maxKeptOPKs = 4

// rotateOPK помечает ключ потраченным и публикует следующий.
//
// Состояние кладётся на диск до возврата: собеседник пользуется опубликованным
// бандлом и придёт на этот ключ снова, а неразглашённый наружу OPK после
// перезапуска означал бы, что настоящий диалог перестал открываться. Поэтому
// ошибка хранилища здесь срывает обработку принятых байтов — потерять одно
// сообщение дешевле, чем рассинхронизировать prekey-материал всего ядра.
func (e *Engine) rotateOPK(consumed uint32) error {
	if consumed == 0 {
		return nil
	}
	e.opkConsumed[consumed] = true
	if consumed == e.opkID {
		next, err := ecdh.X25519().GenerateKey(rand.Reader)
		if err != nil {
			return fmt.Errorf("rpc: выпуск нового one-time prekey: %w", err)
		}
		e.opkID = consumed + 1
		e.opkPriv[e.opkID] = next
	}
	if err := e.trimOPK(); err != nil {
		return err
	}
	return e.persistPrekeys()
}

func (e *Engine) trimOPK() error {
	// Границы считаются по классам: опубликованный ключ не трогаем никогда,
	// потраченные живут до maxKeptOPKs, выданные но не потраченные — до
	// maxHandedOPKs. Общескладный лимит вместо этого сбрасывал бы все
	// потраченные ключи разом, как только пятая запись появилась бы на диске, и
	// отправитель с прежним бандлом больше не собрал бы сессию.
	var consumedIDs, handedIDs []uint32
	for id := range e.opkPriv {
		switch {
		case id == e.opkID:
		case e.opkConsumed[id]:
			consumedIDs = append(consumedIDs, id)
		default:
			handedIDs = append(handedIDs, id)
		}
	}
	dropOldest := func(ids []uint32, keep int) {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for len(ids) > keep {
			delete(e.opkPriv, ids[0])
			delete(e.opkConsumed, ids[0])
			ids = ids[1:]
		}
	}
	dropOldest(consumedIDs, maxKeptOPKs)
	dropOldest(handedIDs, maxHandedOPKs)
	return nil
}

// maxHandedOPKs — сколько выданных, но ещё не потраченных one-time prekey
// держим. Выдача бандла (ручная карта или ответ на bundle_req) помечает ключ
// выданным и публикует следующий;sender придёт на него когда-нибудь, и к тому
// моменту ключ обязан остаться в складе.
const maxHandedOPKs = 8

// rotateOnHandout публикует следующий one-time prekey перед тем, как отдать
// нынешний. Когда склад заполнен, ротации нет: отдать повторно тот же ключ
// честнее, чем отказать человеку в бандле, — one-time свойство при этом
// деградирует, и это ровно та цена, которую платит bounded pool.
//
// Состояние уходит на диск до возврата вызывающему: выданный и не записанный
// OPK означает X3DH, который после рестарта перестанет открываться.
func (e *Engine) rotateOnHandout() error {
	if len(e.opkPriv) >= maxKeptOPKs+maxHandedOPKs {
		return nil
	}
	next, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("rpc: выпуск следующего one-time prekey: %w", err)
	}
	e.opkID++
	e.opkPriv[e.opkID] = next
	if err := e.trimOPK(); err != nil {
		return err
	}
	return e.persistPrekeys()
}

// prekeyStats — учёт prekey-материала на диске. Ошибка здесь стоит дороже
// обычной: выпущенный заново signed prekey ломает чужие X3DH-сессии, собранные
// по прежнему бандлу, и для пользователя это выглядит как «сообщения перестали
// открываться» без единой подсказки, почему.
type prekeyStats struct {
	persisted  uint64 // снимок материала записан в хранилище
	persistErr uint64 // хранилище не приняло запись
	restored   uint64 // материал поднят из хранилища при старте
	// skippedBroken — на диске был набор, но он не разбирается или не годится:
	// ядро вынуждено выпустить новый, и старые бандлы контактов перестают
	// открываться. Молча это делать нельзя, поэтому случай считается отдельно.
	skippedBroken uint64
	regenerated   uint64 // материал выпушен заново (первый запуск или битый снимок)
}

// restorePrekeys поднимает signed prekey и склад one-time prekeys из
// хранилища, а если годного набора там нет — выпускает новый и тут же
// сохраняет. Вызывается из startNode после открытия базы и до того, как нода
// кому-либо что-либо опубликует.
//
// Порядок фиксирован: без хранилища непонятно, продолжаем ли мы прежний материал
// или начинаем новую жизнь, а молча сменить SPK — значит обречь всех, кто
// застал старый бандл, на вечный отказ расшифровки.
//
// Набор берётся строго по владельцу: в одном хранилище (явно заданный
// master_key_256) могут лежать ключи нескольких идентичностей, и надеть чужой
// signed prekey — это начать расшифровывать чужие диалоги.
func (e *Engine) restorePrekeys() error {
	e.opkID = 0
	e.opkPriv = make(map[uint32]*ecdh.PrivateKey)
	e.opkConsumed = make(map[uint32]bool)

	if stored := e.db.Prekeys(e.master.MasterX25519.PublicKey().Bytes()); stored != nil {
		if e.loadPrekeySet(stored) {
			e.prekeyStats.restored++
			return nil
		}
		e.prekeyStats.skippedBroken++
	}

	spk, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("rpc: генерация signed prekey: %w", err)
	}
	opk, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("rpc: генерация one-time prekey: %w", err)
	}
	e.spkPriv = spk
	e.opkID = 1
	e.opkPriv = map[uint32]*ecdh.PrivateKey{e.opkID: opk}
	e.prekeyStats.regenerated++
	return e.persistPrekeys()
}

// loadPrekeySet применяет сохранённый набор к памяти и говорит, годится ли он.
// Годным считается только полный: published-ключ обязан присутствовать в складе
// (его ищет openReceiverSession), а все приватные части — разбираться.
func (e *Engine) loadPrekeySet(stored *storage.StoredPrekeys) bool {
	if stored == nil || len(stored.SpkPriv) != 32 || stored.OpkID == 0 {
		return false
	}
	spk, err := ecdh.X25519().NewPrivateKey(stored.SpkPriv)
	if err != nil {
		return false
	}
	priv := make(map[uint32]*ecdh.PrivateKey, len(stored.Opk))
	for _, rec := range stored.Opk {
		key, err := ecdh.X25519().NewPrivateKey(rec.Priv)
		if err != nil || rec.ID == 0 {
			return false
		}
		priv[rec.ID] = key
	}
	published, ok := priv[stored.OpkID]
	if !ok || published == nil {
		return false
	}
	consumed := make(map[uint32]bool, len(stored.OpkConsumed))
	for _, id := range stored.OpkConsumed {
		consumed[id] = true
	}
	e.spkPriv = spk
	e.opkID = stored.OpkID
	e.opkPriv = priv
	e.opkConsumed = consumed
	return true
}

// persistPrekeys пишет текущий материал целиком: выброшенный one-time prekey
// обязан исчезнуть и с диска, а не пережить свою жизнь в памяти.
func (e *Engine) persistPrekeys() error {
	if e.db == nil || e.master == nil || e.spkPriv == nil {
		e.prekeyStats.persistErr++
		return errors.New("rpc: prekey-материал не записан — ядро ещё не собрало ключи")
	}
	pk := &storage.StoredPrekeys{
		Owner:   e.master.MasterX25519.PublicKey().Bytes(),
		SpkPriv: e.spkPriv.Bytes(),
		OpkID:   e.opkID,
	}
	ids := make([]uint32, 0, len(e.opkPriv))
	for id := range e.opkPriv {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		pk.Opk = append(pk.Opk, storage.StoredPrekey{ID: id, Priv: e.opkPriv[id].Bytes()})
	}
	for id := range e.opkConsumed {
		pk.OpkConsumed = append(pk.OpkConsumed, id)
	}
	// Порядок списков фиксирован: иначе два снимка одного состояния различались бы
	// побайтово, и тест на «пережило рестарт» ловил бы собственную неотсортированность.
	sort.Slice(pk.OpkConsumed, func(i, j int) bool { return pk.OpkConsumed[i] < pk.OpkConsumed[j] })

	if err := e.db.SavePrekeys(pk); err != nil {
		e.prekeyStats.persistErr++
		return fmt.Errorf("rpc: сохранение prekey-материала: %w", err)
	}
	e.prekeyStats.persisted++
	return nil
}

func (e *Engine) nextEventID() uint64 {
	e.eventSeq++
	return e.eventSeq
}

// emit публикует событие, не блокируясь: читатель в UI может быть занят, а
// приёмный цикл стоять не вправе.
func (e *Engine) emit(ev *yggproto.CoreEvent) {
	select {
	case e.events <- ev:
	default:
		e.rxStats.evDropped++
	}
}

// inboundConversationID — имя диалога у получателя. Относится к identity key
// отправителя, а не к conversation_id в конверте: два клиента не договаривались
// об одном тексте, и полагаться на имя, выбранное другой стороной, нельзя.
func inboundConversationID(senderX25519 []byte) string {
	return "1:" + hexKey(senderX25519)
}

// inboundTimestamp — метка входящего сообщения. Конверт несёт эпоху в секундах,
// и два сообщения внутри одной секунды получили бы одинаковый порядок;
// собственная метка ChatMessage точнее, поэтому приоритет у неё.
func inboundTimestamp(msg *yggproto.ChatMessage, envelopeSeconds int64) int64 {
	if ms := msg.GetTimestampMs(); ms != 0 {
		return int64(ms)
	}
	return envelopeSeconds * 1000
}

func mailboxOf(b []byte) (out [32]byte) {
	copy(out[:], b)
	return out
}

func envelopeFromWire(env *yggproto.SealedEnvelope) *crypto.SealedEnvelope {
	return &crypto.SealedEnvelope{
		MailboxID:       mailboxOf(env.GetBlindToken()),
		EphemeralX25519: env.GetEphemeralX25519Pubkey(),
		Nonce:           env.GetNonce(),
		EncryptedData:   env.GetEncryptedPayload(),
		Timestamp:       int64(env.GetTimestamp()),
		PoWNonce:        env.GetPowNonce(),
	}
}

func (e *Engine) nodeStatus(out *yggproto.NodeStatusResp) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	var up uint32
	for _, p := range e.node.Peers() {
		if p.Up {
			up++
		}
	}
	out.IsRunning = true
	out.MyIpv6 = e.node.NodeIPv6().String()
	out.MyEd25519Pubkey = e.node.PublicKey()
	out.MyX25519Pubkey = e.master.MasterX25519.PublicKey().Bytes()
	out.ConnectedPeersCount = up
	out.UptimeSeconds = uint64(e.node.Uptime().Seconds())
	out.ActiveAntiDpi = e.antiDpi
	out.RoutingTableNodes = e.node.RoutingEntries()
	// Участие в S&F показывается всегда, даже при неработающих релеях: «ноль
	// релеев в конфиге» и «релеи есть, но почта к ним не легла» — разная
	// диагностика.
	out.Sf = &yggproto.SfStats{
		RelayEnabled:     e.sfRelayEnabled,
		RelaysConfigured: uint32(len(e.sfRelays)),
		Pushed:           e.sfStats.pushed,
		PushFailed:       e.sfStats.pushFailed,
		Stored:           e.sfStats.stored,
		Refused:          e.sfStats.refused,
		Pulled:           e.sfStats.pulled,
		HandedOut:        e.sfStats.handedOut,
		LastRefusal:      e.sfStats.lastRefusal,
		Expired:          e.sfStats.expired,
	}
	return nil
}

func (e *Engine) peerList(out *yggproto.PeerListResp) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	out.Peers = e.peerInfos()
	return nil
}

// peerInfos — снимок пиров ровно так, как его видит оверлей. Общ один у ответа
// на GetPeers и у события peers_updated: два пути не вправе показывать UI
// разные списки.
func (e *Engine) peerInfos() []*yggproto.PeerInfo {
	infos := make([]*yggproto.PeerInfo, 0, len(e.node.Peers()))
	for _, p := range e.node.Peers() {
		info := &yggproto.PeerInfo{
			Uri:           p.URI,
			Transport:     transportFromURI(p.URI),
			IsInbound:     p.Inbound,
			IsConnected:   p.Up,
			BytesSent:     p.TXBytes,
			BytesReceived: p.RXBytes,
			LatencyMs:     uint64(p.Latency.Milliseconds()),
		}
		// remote_address оставлен пустым: transport-адрес уже лежит в URI, а
		// выдавать за него ключ ноды было бы неверно.
		if len(p.RemoteKey) == ed25519.PublicKeySize {
			info.YggdrasilIpv6 = yggnode.AddressForKey(p.RemoteKey).String()
		}
		infos = append(infos, info)
	}
	if len(infos) == 0 {
		return nil
	}
	return infos
}

// publishPeers объявляет peers_updated, только если вершина оверлея изменилась
// с прошлого объявления. Сигнатура намеренно структурная (адрес, транспорт,
// входящее ли соединение, поднят ли линк): счётчики байт и латентность
// меняются от каждого пакета, и событие иначе превратилось бы в опрос, который
// UI и так делает через GetPeers.
//
// Вызывается из сканера очереди, поэтому задержка объявления — один тик, а
// пинк от ConnectPeer превращает его в доли секунды.
func (e *Engine) publishPeers() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running || e.node == nil {
		return
	}
	infos := e.peerInfos()
	sig := peerSignature(infos)
	if e.peersSigSet && sig == e.peersSig {
		return
	}
	e.peersSig, e.peersSigSet = sig, true
	e.emit(&yggproto.CoreEvent{
		EventId:     e.nextEventID(),
		TimestampMs: uint64(time.Now().UnixMilli()),
		Event: &yggproto.CoreEvent_PeersUpdated{
			PeersUpdated: &yggproto.PeerListUpdatedEvent{Peers: infos},
		},
	})
}

// peerSignature — свёрнутая структурная вершина. Порядок пиров не фиксирован
// (оверлей отдаёт их как попало), поэтому поле сортируются перед склейкой.
func peerSignature(infos []*yggproto.PeerInfo) string {
	entries := make([]string, 0, len(infos))
	for _, p := range infos {
		entries = append(entries, fmt.Sprintf("%s|%s|%t|%t",
			p.GetUri(), p.GetYggdrasilIpv6(), p.GetIsInbound(), p.GetIsConnected()))
	}
	sort.Strings(entries)
	return strings.Join(entries, ";")
}

// publishAntiDpi объявляет anti_dpi_status — единственную ветку контракта,
// отвечающую на вопрос «что с обходом происходит на живых ссылках». Границы
// честности заданы измерением, а не желанием:
//
//   - mode показывает *намерение* из конфига. К байтам рукопожатия он не
//     прилагается: ссылки dial'ит yggdrasil своим crypto/tls (Шаг 4-b), а
//     техники pkg/transport подключены только к замерке cmd/tspuprobe;
//   - current_sni считается из URI ссылки, а не из anti_dpi_sni: для
//     литерального адреса crypto/tls SNI не отправляет вовсе — это замерено на
//     живом линке, и посреднику сопоставить с таким именем нечего;
//   - handshake_success — состояние линка по данным оверлея (PeerStatus.Up), а
//     не «наш dialer смог»;
//   - fallback_triggered всегда false: переключения на другой путь в живом коде
//     нет, а поле есть. Выдумывать срабатывание события нечем.
//
// Объявление, как и у peers_updated, только по изменению структурной вершины:
// иначе событие превратилось бы в опрос.
func (e *Engine) publishAntiDpi() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running || e.node == nil {
		return
	}
	ev, sig := e.antiDpiStatus()
	if e.dpiSigSet && sig == e.dpiSig {
		return
	}
	e.dpiSig, e.dpiSigSet = sig, true
	e.emit(&yggproto.CoreEvent{
		EventId:     e.nextEventID(),
		TimestampMs: uint64(time.Now().UnixMilli()),
		Event: &yggproto.CoreEvent_AntiDpiStatus{
			AntiDpiStatus: ev,
		},
	})
}

// antiDpiStatus — снимок исходящих ссылок и свёрнутая вершина для дедупа.
// Входящие линки не участвуют: SNI отправляет та сторона, что соединяется, а
// событие описывает ровно то, что посредник видит на нашем исходящем пути.
func (e *Engine) antiDpiStatus() (*yggproto.AntiDpiStatusEvent, string) {
	peers := e.node.Peers()
	entries := make([]string, 0, len(peers))
	type linkCount struct {
		total int
		up    int
		snis  map[string]bool
	}
	byScheme := map[string]*linkCount{}
	out, up := 0, 0
	lastErr := ""

	for _, p := range peers {
		if p.Inbound {
			continue
		}
		out++
		if p.Up {
			up++
		} else if p.LastError != "" {
			lastErr = p.LastError
		}
		scheme, _ := splitLink(p.URI)
		sni := linkSNI(p.URI)
		c := byScheme[scheme]
		if c == nil {
			c = &linkCount{snis: map[string]bool{}}
			byScheme[scheme] = c
		}
		c.total++
		if p.Up {
			c.up++
		}
		c.snis[sni] = true

		entries = append(entries, fmt.Sprintf("%s|%s|%t|%s", p.URI, sni, p.Up, p.LastError))
	}
	sort.Strings(entries)

	ev := &yggproto.AntiDpiStatusEvent{
		Mode:              e.antiDpi,
		HandshakeSuccess:  out > 0 && up > 0,
		FallbackTriggered: false,
	}

	schemes := make([]string, 0, len(byScheme))
	for s := range byScheme {
		schemes = append(schemes, s)
	}
	sort.Strings(schemes)

	sniSeen := map[string]bool{}
	var snis []string
	for _, s := range schemes {
		for name := range byScheme[s].snis {
			if name != "" && !sniSeen[name] {
				sniSeen[name] = true
				snis = append(snis, name)
			}
		}
	}
	sort.Strings(snis)
	ev.CurrentSni = strings.Join(snis, ",")

	switch {
	case out == 0:
		ev.TransportDetails = "исходящих ссылок нет: сравнивать нечего"
	default:
		parts := make([]string, 0, len(schemes))
		for _, s := range schemes {
			c := byScheme[s]
			sniNote := "SNI нет"
			switch {
			case !carriesTLS(s):
				sniNote = "TLS нет"
			case len(c.snis) == 1 && !c.snis[""]:
				for n := range c.snis {
					sniNote = "SNI = " + n
				}
			case len(c.snis) > 1:
				names := make([]string, 0, len(c.snis))
				for n := range c.snis {
					if n != "" {
						names = append(names, n)
					}
				}
				sort.Strings(names)
				sniNote = "SNI: " + strings.Join(names, ",") + ", у части ссылок нет"
			}
			parts = append(parts, fmt.Sprintf("%s:// ×%d (поднято %d; %s; %s)",
				s, c.total, c.up, sniNote, linkHandshakeNote(s)))
		}
		ev.TransportDetails = fmt.Sprintf("исходящих %d, поднято %d: %s", out, up, strings.Join(parts, "; "))
		if lastErr != "" {
			ev.TransportDetails += "; последняя ошибка: " + lastErr
		}
	}
	// Режим входит в вершину: сам по себе он фиксируется при старте ноды, но
	// без него событие не пересказалось бы, если режим когда-нибудь научат
	// менять на живом соединении.
	return ev, fmt.Sprintf("%d;%s", e.antiDpi, strings.Join(entries, ";"))
}

// carriesTLS — схемы, где yggdrasil оборачивает поток в TLS и, значит, в сети
// появляется ClientHello. Остальные описаны отдельно: называть рукопожатие там,
// где его нет, событие не вправе.
func carriesTLS(scheme string) bool {
	return scheme == "tls" || scheme == "wss"
}

// linkHandshakeNote — что именно посредник видит на ссылке этой схемы. Формулировки
// соответствуют замерам: Шага 4-b (байты tls://-ссылки) и Шага 16 (что делает и
// чего не делает pkg/transport).
func linkHandshakeNote(scheme string) string {
	switch scheme {
	case "tcp":
		return "TLS-рукопожатия нет: видно только TCP и полезную нагрузку"
	case "tls":
		return "рукопожатие делает yggdrasil своим crypto/tls"
	case "wss":
		return "рукопожатие делает yggdrasil своим crypto/tls поверх WebSocket"
	case "ws":
		return "TLS-рукопожатия нет: WebSocket поверх открытого HTTP"
	case "quic":
		return "рукопожатие QUIC; отпечаток этим кодом не замерен"
	case "socks":
		return "TLS делает прокси, не мы"
	default:
		return "описание рукопожатия для этой схемы неизвестно"
	}
}

// splitLink — схема и хост URI без порта. Скобки вокруг IPv6 снимаются: они
// часть записи в URI, а не имени хоста.
func splitLink(uri string) (scheme, host string) {
	rest := uri
	scheme = "tcp"
	if i := strings.Index(uri, "://"); i >= 0 {
		scheme = uri[:i]
		rest = uri[i+3:]
	}
	h, _, err := net.SplitHostPort(rest)
	if err != nil {
		h = rest
	}
	return scheme, strings.Trim(h, "[]")
}

// linkSNI — имя, которое уходит в SNI этой ссылки. Пустой ответ означает «SNI
// нет», а не «неизвестно»: crypto/tls не отправляет SNI для литерального адреса,
// и это замерено на живом линке (Шаг 4-b).
func linkSNI(uri string) string {
	scheme, host := splitLink(uri)
	if !carriesTLS(scheme) || host == "" || net.ParseIP(host) != nil {
		return ""
	}
	return host
}

func (e *Engine) connectPeer(req *yggproto.ConnectPeerReq) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	uri := strings.TrimSpace(req.GetPeerUri())
	if uri == "" {
		return errors.New("rpc: пустой peer_uri")
	}
	uri, err := peerURI(uri, e.antiDpi)
	if err != nil {
		return fmt.Errorf("rpc: peer_uri: %w", err)
	}
	if err := e.node.AddPeer(uri); err != nil {
		return err
	}
	// Пир только что добавили: возможно, это первый маршрут для очереди, и
	// ждать следующего тика сканера незачем.
	e.kickOutbox()
	return nil
}

// disconnectPeer вычёркивает постоянного пира (добавленного ли connect_peer,
// ли поднятого из bootstrap_peers при старте) и рвёт живой линк. URI проходит
// ту же нормализацию, что и при добавлении: оверлей ищет линк по строке URI,
// и «127.0.0.1:9009» без схемы обязан сойтись с tcp://127.0.0.1:9009, иначе
// честный запрос выдасться ответом «пира нет».
//
// Принятую снаружи ссылку так убрать нельзя — её адрес не мы конфигурировали,
// и yggdrasil законно отвечает ErrLinkNotConfigured. Это видно в тексте ошибки,
// а не молчаливым success=true: список пиров в ответе возвращается всегда,
// когда команда прошла.
func (e *Engine) disconnectPeer(req *yggproto.DisconnectPeerReq) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	uri := strings.TrimSpace(req.GetPeerUri())
	if uri == "" {
		return errors.New("rpc: пустой peer_uri")
	}
	uri, err := peerURI(uri, e.antiDpi)
	if err != nil {
		return fmt.Errorf("rpc: peer_uri: %w", err)
	}
	return e.node.RemovePeer(uri)
}

func (e *Engine) preKeyBundle(out *yggproto.PreKeyBundle) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	// Отдаваемый бандл сразу уступает опубликованный one-time prekey: и ручная
	// карта (GetPrekeyBundle), и сетевой ответ (bundle_req) несут ключ, которым
	// воспользуются без нас. Если бы один и тот же OPK раздавался всем подряд,
	// «one-time» перестал бы значить что-либо, а X3DH-корень двух разных
	// отправителей совпал бы.
	id := e.opkID
	if err := e.rotateOnHandout(); err != nil {
		return err
	}
	spk := e.spkPriv.PublicKey()
	sig := ed25519.Sign(e.master.MasterEd25519, spk.Bytes())
	out.IdentityEd25519Pubkey = e.master.MasterEd25519.Public().(ed25519.PublicKey)
	out.IdentityX25519Pubkey = e.master.MasterX25519.PublicKey().Bytes()
	out.SignedPrekey = spk.Bytes()
	out.SignedPrekeySignature = sig
	out.OneTimePrekeys = [][]byte{e.opkPriv[id].PublicKey().Bytes()}
	out.PrekeyTimestamp = uint64(time.Now().Unix())
	out.KeyId = id
	// Ключ ноды в бандле — не справочная строка, а заверенный адрес доставки:
	// контакту, который не может показать эту подпись, AddContact откажет.
	out.NodeEd25519Pubkey = append([]byte(nil), e.node.PublicKey()...)
	proof, err := crypto.BuildNodeKeyProof(e.master, e.node.PublicKey())
	if err != nil {
		return fmt.Errorf("rpc: сертификат ключа ноды: %w", err)
	}
	out.NodeKeyProof = proof
	return nil
}

// verifyContact — единые ворота доверия для карты контакта. Проверка стоит
// отдельной функцией потому, что карту обязаны проходить и живая добавка
// (AddContact), и восстановление из базы после рестарта: сохранённая карта —
// это байты на диске, и доверять им «потому что мы же сами записали» нельзя.
// Порядок фиксирован: дешёвые формальные проверки, затем адрес (ловит подделку
// «ключ ↔ ipv6»), затем связка с идентичностью.
func verifyContact(card *yggproto.ContactCard) error {
	if card == nil {
		return errors.New("rpc: ContactCard отсутствует")
	}
	if len(card.GetX25519Pubkey()) != 32 {
		return errors.New("rpc: в ContactCard нет валидного x25519-ключа")
	}
	bundle := card.GetBundle()
	if bundle == nil {
		return errors.New("rpc: ContactCard без PreKeyBundle: X3DH невозможен")
	}
	nodeKey := card.GetEd25519Pubkey()
	if len(nodeKey) != ed25519.PublicKeySize {
		return errors.New("rpc: у контакта нет ed25519-ключа ноды: писать в оверлей некуда")
	}
	// Адрес Yggdrasil однозначно выводится из ключа ноды. Если прислали другой,
	// контакт подделан или устарел — принимать его нельзя.
	if card.GetNodeIpv6() != "" {
		want := yggnode.AddressForKey(nodeKey)
		got := net.ParseIP(card.GetNodeIpv6())
		if got == nil || !want.Equal(got) {
			return fmt.Errorf("rpc: node_ipv6 %s не соответствует ключу ноды (ожидается %s)",
				card.GetNodeIpv6(), want)
		}
	}
	// Дальше — связка «адрес доставки ↔ идентичность». Сеанс Double Ratchet
	// ключуется по card.x25519_pubkey, а X3DH собирается по бандлу: разнобой
	// означал бы шифрование одному адресату и запись в диалог другого.
	if !bytes.Equal(card.GetX25519Pubkey(), bundle.GetIdentityX25519Pubkey()) {
		return errors.New("rpc: x25519-ключ контакта не совпадает с identity-ключом бандла")
	}
	certified := bundle.GetNodeEd25519Pubkey()
	if len(certified) != ed25519.PublicKeySize {
		return errors.New("rpc: в PreKeyBundle нет ключа ноды: связать адрес с идентичностью нечем")
	}
	if !bytes.Equal(certified, nodeKey) {
		return errors.New("rpc: ключ ноды в карте контакта не тот, что заверен в бандле")
	}
	// Сертификат выдаёт тот же мастер-Ed25519, что подписывает signed prekey, и
	// подписывает ровно эту пару ключей. Без него любой, кто знает чужой
	// identity-ключ, подставил бы в карту собственный адрес узла: шифротекст
	// честной идентичности ушёл бы к нему.
	if err := crypto.VerifyNodeKeyProof(
		bundle.GetIdentityEd25519Pubkey(),
		bundle.GetIdentityX25519Pubkey(),
		certified,
		bundle.GetNodeKeyProof(),
	); err != nil {
		return fmt.Errorf("rpc: ключ ноды не привязан к идентичности контакта: %w", err)
	}
	return nil
}

func (e *Engine) addContact(card *yggproto.ContactCard) error {
	// Карта живёт в том же хранилище, что и сообщения, а оно открывается вместе
	// с идентичностью в StartNode. Без запущенного ядра непонятно, к какому
	// владельцу относится контакт, а «добавить в память и потерять при
	// перезапуске» — это ровно тот дефект, который Шаг 5-b и убирает.
	if err := e.requireStarted(); err != nil {
		return err
	}
	if err := verifyContact(card); err != nil {
		return err
	}
	key := hexKey(card.GetX25519Pubkey())
	// Сначала байты в базе, потом карта в памяти. Если запись отклонили, контакта
	// не существует ни там, ни тут; обратный порядок дал бы диалог, который
	// работает до первого рестарта и потом исчезает.
	durable, err := proto.Marshal(card)
	if err != nil {
		return fmt.Errorf("rpc: кодировка ContactCard: %w", err)
	}
	if err := e.db.SaveContact(key, durable); err != nil {
		return fmt.Errorf("rpc: сохранение контакта: %w", err)
	}
	e.contactStats.persisted++
	e.contacts[key] = card
	return nil
}

func (e *Engine) sendMessage(req *yggproto.SendMessageReq, out *yggproto.SendMessageResp) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	msg := req.GetMessage()
	if msg == nil {
		return errors.New("rpc: SendMessage без ChatMessage")
	}
	contact := e.contacts[hexKey(req.GetDestinationX25519Pubkey())]
	if contact == nil {
		return errors.New("rpc: контакта с таким x25519-ключом нет, сначала AddContact с PreKeyBundle")
	}
	if msg.GetConversationId() == "" {
		return errors.New("rpc: пустой conversation_id — сессии Double Ratchet не на чем различать")
	}

	key := sessionKey(contact.GetX25519Pubkey(), msg.GetConversationId())
	// Сеанс заводит sessionFor, а шифрует seal по тому же ключу: держаться за
	// указатель сессии тут незачем.
	if _, err := e.sessionFor(contact.GetX25519Pubkey(), msg.GetConversationId(), contact); err != nil {
		return err
	}

	if msg.GetMessageId() == "" {
		// Идентификатор фиксируется до шифрования: получатель назовёт его в
		// ACK, и отправитель должен узнать собственное сообщение.
		msg.MessageId = messageID(msg)
	}

	plaintext, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("rpc: сборка ChatMessage: %w", err)
	}

	env, err := e.sealEnvelope(contact.GetX25519Pubkey(), msg.GetConversationId(), key, plaintext)
	if err != nil {
		return err
	}
	sealed, err := chatFrame(env)
	if err != nil {
		return err
	}

	stored := &storage.StoredMessage{
		MessageID:      messageID(msg),
		ConversationID: msg.GetConversationId(),
		SenderID:       e.node.NodeIPv6().String(),
		TimestampMS:    int64(msg.GetTimestampMs()),
		Body:           preview(msg),
		IsOutbound:     true,
	}
	if err := e.db.SaveMessage(stored); err != nil {
		return fmt.Errorf("rpc: сохранение сообщения: %w", err)
	}
	out.MessageId = stored.MessageID
	// Клиенту отдаём реальные байты, которые ушли в сеть: без них сообщение
	// нельзя переложить в S&F-очередь или переотправить при появлении пира.
	out.SealedEnvelope = sealed

	nodeKey := contact.GetEd25519Pubkey()
	target := yggnode.AddrForPublicKey(nodeKey)
	if target == nil {
		return errors.New("rpc: ключ ноды получателя негодный")
	}
	// WriteTo в ironwood принимает пакет всегда (он лишь заводит сессию),
	// поэтому «доставлено» по ошибке WriteTo определить нельзя. Сначала
	// спрашиваем, знает ли spanning-дерево маршрут до ключа.
	var written int
	var writeErr error
	if e.node.HasRouteTo(nodeKey) {
		written, writeErr = e.node.PacketConn().WriteTo(sealed, target)
	}
	if writeErr != nil || written != len(sealed) {
		// Маршрута нет или пакет не лёг. Байты уходят в зашифрованную исходящую
		// очередь: сканер перешлёт их дословно, как только увидит Known-маршрут,
		// в том числе после перезапуска ядра.
		if err := e.db.QueueOutbound(&storage.StoredOutbound{
			MessageID:      stored.MessageID,
			ConversationID: stored.ConversationID,
			NodeKey:        append([]byte(nil), nodeKey...),
			Envelope:       append([]byte(nil), sealed...),
			EnqueuedMS:     time.Now().UnixMilli(),
		}); err != nil {
			return fmt.Errorf("rpc: пакет не ушёл в сеть и в исходящую очередь не лёг: %w", err)
		}
		e.outStats.queued++
		e.kickOutbox()
		out.QueuedInOutbox = true
		// Адресат офлайн — конверт кладётся и слепым релеям, чтобы пережить
		// исчезновение этого процесса. Попытка здесь не засчитывается намеренно:
		// первый пакет на ещё не прогретой сессии ironwood теряется (о таком же
		// риске молчит и живая отправка), а одна кладка — это один шанс. Сканер
		// очереди повторит handoff при следующих проходах и засчитает их: больше
		// четырёх раз те же байты на хранителя не лягут, а повтор для релея
		// безвреден — ящик дедуплицируется по содержимому конверта.
		if handed := e.pushToRelays(env); handed > 0 {
			out.RelaysPushed = true
			out.RelaysHanded = uint32(handed)
		}
		return nil
	}
	// Успех: убираем остаток прошлой неудачной попытки. Текущие байты ушли, а
	// старые превратились бы у получателя в replay.
	e.removeOutbound(stored.MessageID)
	// Честная формулировка: оверлей принял пакет, и маршрут до ключа известен.
	// Это ещё не «доставлено» — настоящим подтверждением считает только
	// DeliveryAckContent, открыт сеансом получателя.
	out.WrittenToOverlay = true
	e.emit(&yggproto.CoreEvent{
		EventId:     e.nextEventID(),
		TimestampMs: uint64(time.Now().UnixMilli()),
		Event: &yggproto.CoreEvent_MessageStatus{MessageStatus: &yggproto.MessageStatusEvent{
			MessageId:        stored.MessageID,
			ConversationId:   stored.ConversationID,
			WrittenToOverlay: true,
		}},
	})
	return nil
}

func (e *Engine) sessionFor(peerX []byte, convID string, contact *yggproto.ContactCard) (*crypto.DoubleRatchetSession, error) {
	key := sessionKey(peerX, convID)
	if s, ok := e.sessions[key]; ok {
		return s, nil
	}
	bundle := contact.GetBundle()
	identityX, err := ecdh.X25519().NewPublicKey(bundle.GetIdentityX25519Pubkey())
	if err != nil {
		return nil, fmt.Errorf("rpc: ключ идентичности получателя: %w", err)
	}
	spk, err := ecdh.X25519().NewPublicKey(bundle.GetSignedPrekey())
	if err != nil {
		return nil, fmt.Errorf("rpc: signed prekey получателя: %w", err)
	}
	bundleGo := crypto.PreKeyBundle{
		IdentityKey:     identityX,
		IdentityEd25519: ed25519.PublicKey(bundle.GetIdentityEd25519Pubkey()),
		SignedPreKey:    spk,
		SignedPreKeySig: bundle.GetSignedPrekeySignature(),
		Timestamp:       int64(bundle.GetPrekeyTimestamp()),
	}
	if opks := bundle.GetOneTimePrekeys(); len(opks) > 0 {
		opk, err := ecdh.X25519().NewPublicKey(opks[0])
		if err != nil {
			return nil, fmt.Errorf("rpc: one-time prekey получателя: %w", err)
		}
		bundleGo.OneTimePreKey = opk
		bundleGo.OneTimePreKeyID = bundle.GetKeyId()
	}

	handshake, err := crypto.PerformX3DHInitiator(e.master.MasterX25519, e.master.MasterEd25519.Public().(ed25519.PublicKey), bundleGo)
	if err != nil {
		return nil, fmt.Errorf("rpc: X3DH инициатора: %w", err)
	}
	session, err := crypto.NewInitiatorSession(handshake.SharedKey, spk)
	if err != nil {
		return nil, fmt.Errorf("rpc: сессия Double Ratchet: %w", err)
	}
	e.sessions[key] = session
	// Эфемерный ключ X3DH должен доехать до получателя, иначе он не сможет
	// посчитать те же DH-пары.
	e.ekPub[key] = handshake.EphemeralPub.Bytes()
	// ID потраченного one-time prekey: получатель по нему найдёт свой приватный
	// ключ, без этого X3DH на его стороне не соберётся.
	if handshake.UsedOneTimePK {
		e.opkUsed[key] = bundleGo.OneTimePreKeyID
	}
	return session, nil
}

// persistedSession — что ядро кладёт в хранилище по одному сеансу.
//
// Помимо состояния рэтчета тут живут эфемерный ключ X3DH и ID потраченного
// one-time prekey: конверт обязан нести их и во втором сообщении после
// перезапуска. Потерялись бы — получатель, который сам потерял сеанс, не смог бы
// собрать X3DH заново и остался бы с пустым диалогом.
//
// Owner — мастер-X25519 этого узла. Он нужен потому, что сеанс выводится из
// него: перезапуск с другой фразой обязан увидеть несвоё состояние и не
// использовать его, а не расшифровывать чужой диалог.
type persistedSession struct {
	Owner   []byte              `json:"owner"`
	State   crypto.SessionState `json:"state"`
	EkPub   []byte              `json:"ek,omitempty"`
	OpkUsed uint32              `json:"opk,omitempty"`
}

// persistSession сериализует сеанс и пишет его в зашифрованную базу. Вызывается
// под e.mu сразу после шага рэтчета — до того, как байты увидят сеть: запись
// позже означала бы, что пережит краш между отправкой и фиксацией счётчика.
func (e *Engine) persistSession(key string) error {
	session, ok := e.sessions[key]
	if !ok {
		return fmt.Errorf("rpc: сеанса %s нет в памяти — сохранять нечего", key)
	}
	if e.db == nil || e.master == nil {
		return fmt.Errorf("rpc: сеанс %s не сохранён: ядро без хранилища", key)
	}
	blob, err := json.Marshal(persistedSession{
		Owner:   e.master.MasterX25519.PublicKey().Bytes(),
		State:   session.ExportState(),
		EkPub:   e.ekPub[key],
		OpkUsed: e.opkUsed[key],
	})
	if err != nil {
		e.sessStats.saveErr++
		return fmt.Errorf("rpc: сериализация сеанса %s: %w", key, err)
	}
	if err := e.db.SaveSession(key, blob); err != nil {
		e.sessStats.saveErr++
		return fmt.Errorf("rpc: запись сеанса %s: %w", key, err)
	}
	e.sessStats.saved++
	return nil
}

// restoreSessions поднимает сеансы из хранилища сразу после его открытия.
// Разобрать не удалось — значит состояние битое или чужое; в памяти сеанса не
// появляется, и следующий шаг диалога будет честно отвергнут, а не расшифрован
// чем попало. Строки другого владельца не удаляются: пусть лежат под замком, но
// чужой личностью шифровать нельзя.
func (e *Engine) restoreSessions() {
	if e.db == nil || e.master == nil {
		return
	}
	owner := e.master.MasterX25519.PublicKey().Bytes()
	for name, blob := range e.db.Sessions() {
		var rec persistedSession
		if err := json.Unmarshal(blob, &rec); err != nil {
			e.sessStats.restoreErr++
			continue
		}
		if !bytes.Equal(rec.Owner, owner) {
			e.sessStats.skippedOwner++
			continue
		}
		session, err := crypto.SessionFromState(rec.State)
		if err != nil {
			e.sessStats.restoreErr++
			continue
		}
		e.sessions[name] = session
		if len(rec.EkPub) > 0 {
			e.ekPub[name] = rec.EkPub
		}
		if rec.OpkUsed != 0 {
			e.opkUsed[name] = rec.OpkUsed
		}
		e.sessStats.restored++
	}
}

// restoreContacts поднимает карты контактов из хранилища сразу после его
// открытия. Байты лежат под тем же замком, что и сообщения, но «файл же свой» —
// не доказательство: диск правят руками, а карта — единственное, чем ядро
// доверяет адрес доставки. Поэтому каждая проходит ровно те же ворота, что и
// AddContact, плюс сверку ключа строки с идентичностью внутри самой карты:
// связка «ключ A -> карта B» означала бы шифрование одному адресату и отправку
// на узел другого.
func (e *Engine) restoreContacts() {
	if e.db == nil {
		return
	}
	for key, blob := range e.db.Contacts() {
		var card yggproto.ContactCard
		if err := proto.Unmarshal(blob, &card); err != nil {
			e.contactStats.restoreErr++
			continue
		}
		if err := verifyContact(&card); err != nil {
			e.contactStats.restoreErr++
			continue
		}
		if hexKey(card.GetX25519Pubkey()) != key {
			e.contactStats.skippedKey++
			continue
		}
		e.contacts[key] = &card
		e.contactStats.restored++
	}
}

// seal шифрует plaintext сеансом (peerX, convID) и упаковывает его в конверт
// Sealed Sender, а затем в кадр оверлея: ровно эти байты ложатся в сокет и в
// исходящую очередь. convID — имя диалога ровно то, что видит получатель: по
// нему он ищет сеанс, поэтому менять его на локальный нельзя.
func (e *Engine) seal(peerX []byte, convID, key string, plaintext []byte) ([]byte, error) {
	env, err := e.sealEnvelope(peerX, convID, key, plaintext)
	if err != nil {
		return nil, err
	}
	return chatFrame(env)
}

// sealEnvelope — та же отправка, но до оболочки: кладущему конверт слепому
// релею нужен сам SealedEnvelope, потому что индекс ящика лежит в его полях, а
// не в кадре.
func (e *Engine) sealEnvelope(peerX []byte, convID, key string, plaintext []byte) (*yggproto.SealedEnvelope, error) {
	session, ok := e.sessions[key]
	if !ok {
		return nil, fmt.Errorf("rpc: сеанса %s нет, ACK нечем шифровать", key)
	}
	enc, err := session.Encrypt(plaintext, []byte(convID))
	if err != nil {
		return nil, fmt.Errorf("rpc: шифрование рэтчетом: %w", err)
	}
	// Состояние фиксируется до упаковки и до сокета. Не запишись оно, пакет всё
	// равно ушёл бы, а после рестарта цепочка откатилась на шаг назад — и
	// следующее сообщение собеседник отверг бы как повтор. Поэтому здесь
	// отказываем отправке, а не делаем вид, что всё в порядке.
	if err := e.persistSession(key); err != nil {
		return nil, err
	}
	inner := encodeRatchetMessage(enc, e.ekPub[key])

	recipientX, err := ecdh.X25519().NewPublicKey(peerX)
	if err != nil {
		return nil, fmt.Errorf("rpc: ключ получателя: %w", err)
	}
	env, err := crypto.PackSealedEnvelope(e.proof, e.device, recipientX, inner, convID, e.opkUsed[key], powDifficulty)
	if err != nil {
		return nil, fmt.Errorf("rpc: упаковка Sealed Sender: %w", err)
	}

	return &yggproto.SealedEnvelope{
		RecipientX25519Pubkey: peerX,
		EphemeralX25519Pubkey: env.EphemeralX25519,
		EncryptedPayload:      env.EncryptedData,
		Nonce:                 env.Nonce,
		Timestamp:             uint64(env.Timestamp),
		PowNonce:              env.PoWNonce,
		BlindToken:            env.MailboxID[:],
	}, nil
}

// DecodeRatchetMessage — обратная к encodeRatchetMessage часть протокола.
func DecodeRatchetMessage(inner []byte) (*crypto.EncryptedRatchetMessage, []byte, error) {
	if len(inner) < 32 {
		return nil, nil, errors.New("rpc: внутренний блок слишком короткий")
	}
	ekPub := inner[:32]
	body := inner[32:]
	if len(body) < 40 {
		return nil, nil, errors.New("rpc: обрезанный заголовок рэтчета")
	}
	msg := &crypto.EncryptedRatchetMessage{
		Header: crypto.MessageHeader{
			DHPublicKey:        body[:32],
			MessageNumber:      binary.BigEndian.Uint32(body[32:36]),
			PreviousChainCount: binary.BigEndian.Uint32(body[36:40]),
		},
		Nonce: body[40:52],
	}
	body = body[52:]
	if len(body) < 4 {
		return nil, nil, errors.New("rpc: обрезанный ciphertext")
	}
	n := binary.BigEndian.Uint32(body[:4])
	body = body[4:]
	if uint32(len(body)) < n {
		return nil, nil, errors.New("rpc: ciphertext короче заявленного")
	}
	msg.Ciphertext = body[:n]
	return msg, ekPub, nil
}

func (e *Engine) listConversations(limit uint32, out *yggproto.ConversationListResp) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	summaries := e.db.ListConversations()
	if limit > 0 && uint32(len(summaries)) > limit {
		summaries = summaries[:limit]
	}
	for _, s := range summaries {
		out.Conversations = append(out.Conversations, &yggproto.ConversationInfo{
			ConversationId:  s.ConversationID,
			LastMessage:     s.LastBody,
			LastTimestampMs: s.LastTimestampMS,
			MessageCount:    uint32(s.MessageCount),
			UnreadCount:     uint32(s.UnreadCount),
		})
	}
	return nil
}

func (e *Engine) getMessages(req *yggproto.GetMessagesReq, out *yggproto.MessageListResp) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	if req.GetConversationId() == "" {
		return errors.New("rpc: пустой conversation_id в GetMessages")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}
	for _, stored := range e.db.GetMessages(req.GetConversationId(), limit) {
		out.Messages = append(out.Messages, &yggproto.ChatMessage{
			MessageId:      stored.MessageID,
			ConversationId: stored.ConversationID,
			SenderNodeIpv6: stored.SenderID,
			TimestampMs:    uint64(stored.TimestampMS),
			IsOutbound:     stored.IsOutbound,
			Delivered:      stored.DeliveredMS != 0,
			Content:        &yggproto.ChatMessage_Text{Text: &yggproto.TextContent{Body: stored.Body}},
		})
	}
	return nil
}

func (e *Engine) listContacts(out *yggproto.ContactListResp) error {
	for _, card := range e.contacts {
		out.Contacts = append(out.Contacts, card)
	}
	sort.Slice(out.Contacts, func(i, j int) bool {
		return out.Contacts[i].GetNodeIpv6() < out.Contacts[j].GetNodeIpv6()
	})
	return nil
}

func generateIdentity(out *yggproto.IdentityResp) error {
	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		return fmt.Errorf("rpc: генерация фразы: %w", err)
	}
	out.Mnemonic = mnemonic
	return nil
}

func (e *Engine) requireStarted() error {
	if !e.running || e.node == nil {
		return errors.New("rpc: ядро не запущено (отправьте StartNode)")
	}
	return nil
}

// encodeRatchetMessage: EphemeralX(32) || DH(32) || MsgNum(4) || PrevChain(4) || Nonce(12) || Len(4) || Ciphertext
func encodeRatchetMessage(m *crypto.EncryptedRatchetMessage, ekPub []byte) []byte {
	if len(ekPub) != 32 {
		ekPub = make([]byte, 32)
	}
	out := make([]byte, 0, 32+32+4+4+12+4+len(m.Ciphertext))
	out = append(out, ekPub...)
	out = append(out, m.Header.DHPublicKey...)
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], m.Header.MessageNumber)
	out = append(out, b4[:]...)
	binary.BigEndian.PutUint32(b4[:], m.Header.PreviousChainCount)
	out = append(out, b4[:]...)
	out = append(out, m.Nonce...)
	binary.BigEndian.PutUint32(b4[:], uint32(len(m.Ciphertext)))
	out = append(out, b4[:]...)
	return append(out, m.Ciphertext...)
}

func nodeKeyFromSeed(master *crypto.MasterIdentity, explicit []byte) ed25519.PrivateKey {
	if len(explicit) == ed25519.PrivateKeySize {
		return ed25519.PrivateKey(explicit)
	}
	seed := sha256.Sum256(append([]byte("ygggram-node-v1:"), master.Seed...))
	return ed25519.NewKeyFromSeed(seed[:])
}

func storageKey(master *crypto.MasterIdentity, explicit []byte) [32]byte {
	if len(explicit) == 32 {
		var k [32]byte
		copy(k[:], explicit)
		return k
	}
	return sha256.Sum256(append([]byte("ygggram-storage-v1:"), master.Seed...))
}

func messageID(msg *yggproto.ChatMessage) string {
	if id := msg.GetMessageId(); id != "" {
		return id
	}
	var b [16]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%x", b)
}

func preview(msg *yggproto.ChatMessage) string {
	if t := msg.GetText(); t != nil {
		return t.GetBody()
	}
	if r := msg.GetReply(); r != nil {
		return r.GetBody()
	}
	if m := msg.GetMedia(); m != nil {
		return "[медиа] " + m.GetFileName()
	}
	if c := msg.GetCallSignal(); c != nil {
		return "[звонок] " + c.GetCallId()
	}
	return ""
}

func hexKey(b []byte) string { return fmt.Sprintf("%x", b) }

func value(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// listenURIs собирает слушатели ноды. Молча открывать порт «по умолчанию»
// нельзя: приложение явно просит listen_address, и без него нода остаётся
// исключительно исходящей.
func listenURIs(cfg *yggproto.NodeConfig, mode yggproto.AntiDpiMode) ([]string, error) {
	raw := strings.TrimSpace(cfg.GetListenAddress())
	if raw == "" {
		return nil, nil
	}
	uri, err := endpointURI(raw, listenScheme(mode), listenPort(cfg))
	if err != nil {
		return nil, err
	}
	return []string{uri}, nil
}

// listenPort — порт из NodeConfig; 0 означает «не задан», а не «слушай 0».
func listenPort(cfg *yggproto.NodeConfig) uint32 {
	if p := cfg.GetListenPort(); p > 0 {
		return uint32(p)
	}
	return uint32(yggnode.DefaultListenPort)
}

// peerURIs нормализует bootstrap-пиры. Пустой список — это валидный конфиг
// (только мультикаст или последующий ConnectPeer), ошибкой он не считается.
func peerURIs(raw []string, mode yggproto.AntiDpiMode) ([]string, error) {
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		uri, err := peerURI(p, mode)
		if err != nil {
			return nil, err
		}
		out = append(out, uri)
	}
	return out, nil
}

func peerURI(raw string, mode yggproto.AntiDpiMode) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("пустая строка пира")
	}
	// Без схемы оставляем tcp — так же поступает сам yggdrasil, и так пир
	// остаётся совместимым с публичными мостами. Молча «повышать» его до
	// tls/wss нельзя: рукопожатие с обычным демоном не сорвётся только у нас.
	scheme := "tcp"
	if strings.Contains(s, "://") {
		scheme = "" // схема уже внутри, endpointURI её разберёт
	}
	port := uint32(yggnode.DefaultListenPort)
	return endpointURI(s, scheme, port)
}

// endpointURI дополняет URI схемой и портом. expectedScheme=="" означает
// «схему взял из самого URI».
func endpointURI(raw, scheme string, defaultPort uint32) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("пустой URI")
	}
	if i := strings.Index(s, "://"); i >= 0 {
		scheme = s[:i]
		s = s[i+3:]
	}
	switch scheme {
	case "tcp", "tls", "socks", "ws", "wss", "quic":
	default:
		return "", fmt.Errorf("неизвестный транспорт %q (ожидается tcp/tls/socks/ws/wss/quic)", scheme)
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		// Порт не указан: "a.b.c.d" или "[::1]". IPv6 без скобок неразборчив —
		// "::1:8666" невозможно однозначно разделить, поэтому просим скобки.
		if strings.Count(s, ":") > 1 && !strings.HasPrefix(s, "[") {
			return "", fmt.Errorf("адрес %q неоднозначен: IPv6 нужно в скобках, например [::1]:8666", raw)
		}
		host, port = s, ""
	}
	// Скобки — часть записи IPv6 в URI, а не хоста: JoinHostPort добавит их
	// сам, иначе получится [[::1]]:8666.
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}
	if port == "" {
		if defaultPort == 0 {
			return "", fmt.Errorf("в %q нет порта, а порт по умолчанию неизвестен", raw)
		}
		port = strconv.FormatUint(uint64(defaultPort), 10)
	}
	if host == "" {
		return "", fmt.Errorf("в %q пустой хост", raw)
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "", fmt.Errorf("невалидный порт в %q: %w", raw, err)
	}
	return scheme + "://" + net.JoinHostPort(host, port), nil
}

// listenScheme — транспорт, который реально поднимает встроенная yggdrasil.
// uTLS-подделку Chrome/ iOS и ShadowTLS в дата-пути больше не применять:
// ссылки идут через crypto/tls и QUIC из yggdrasil-go, поэтому JA3 у нас
// стандартный для Go (замер — Шаг 4).
func listenScheme(mode yggproto.AntiDpiMode) string {
	switch mode {
	case yggproto.AntiDpiMode_ANTI_DPI_WEBSOCKET_CDN:
		return "wss"
	case yggproto.AntiDpiMode_ANTI_DPI_UDP_QUIC_OBFS:
		return "quic"
	case yggproto.AntiDpiMode_ANTI_DPI_DISABLED:
		return "tcp"
	default:
		return "tls"
	}
}

func transportFromURI(uri string) yggproto.PeerTransportType {
	scheme := uri
	if i := strings.Index(uri, "://"); i >= 0 {
		scheme = uri[:i]
	} else {
		scheme = "tcp"
	}
	switch scheme {
	case "wss", "ws":
		return yggproto.PeerTransportType_TRANSPORT_WSS
	case "quic":
		return yggproto.PeerTransportType_TRANSPORT_QUIC_OBFS
	case "tls":
		return yggproto.PeerTransportType_TRANSPORT_TLS_UTLS
	default:
		return yggproto.PeerTransportType_TRANSPORT_TCP
	}
}

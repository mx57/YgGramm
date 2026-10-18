// File: core/pkg/yggnode/node.go
//
// Узел оверлея Yggdrasil. Реализован поверх настоящей yggdrasil-go
// (github.com/yggdrasil-network/yggdrasil-go/src/core), поэтому адреса,
// рукопожатия и формат пакетов wire-совместимы с публичной сетью: к ноде
// можно пироваться настоящими `yggdrasil`-демонами и через публичные мосты.
//
// TUN/Wintun не используется: пакеты ходят через Core.ReadFrom/WriteTo
// (userspace, без root).
package yggnode

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	iwt "github.com/Arceliar/ironwood/types"
	gl "github.com/gologme/log"
	yggaddr "github.com/yggdrasil-network/yggdrasil-go/src/address"
	yggcfg "github.com/yggdrasil-network/yggdrasil-go/src/config"
	yggcore "github.com/yggdrasil-network/yggdrasil-go/src/core"
	yggmc "github.com/yggdrasil-network/yggdrasil-go/src/multicast"
)

// DefaultListenPort — порт, который YggGram держит открытым для входящих
// пирований по умолчанию.
const DefaultListenPort = 54321

// NodeOptions — параметры запуска userspace-узла.
type NodeOptions struct {
	// PrivateKey — Ed25519-ключ ноды, выведенный из мнемоники. yggdrasil
	// строит из него и self-signed сертификат, и IPv6-адрес.
	PrivateKey ed25519.PrivateKey

	// Listen — URI вида "tcp://[::]:54321", "tls://0.0.0.0:443",
	// "wss://[::]:443", "quic://[::]:443". Пусто — только исходящие соединения.
	Listen []string

	// Peers — постоянные пиры в той же URI-форме ("tcp://1.2.3.4:54321").
	Peers []string

	// GroupPassword — приватная подсеть: узлы без такого же пароля не получат
	// от нас трафик.
	GroupPassword string

	// Multicast — wire-совместимое LAN-обнаружение Yggdrasil (ff02::114:9001).
	Multicast bool

	// NodeInfo — необязательная видимая сети информация (имя, версия).
	NodeInfo map[string]interface{}
}

// PeerStatus — состояние пиров-соединения, взятое из ноды yggdrasil.
type PeerStatus struct {
	URI       string
	Up        bool
	Inbound   bool
	Latency   time.Duration
	Uptime    time.Duration
	RXBytes   uint64
	TXBytes   uint64
	RemoteKey ed25519.PublicKey
	LastError string
}

// YggNode — обёртка над *yggcore.Core с userspace-доступом к пакетам.
type YggNode struct {
	core          *yggcore.Core
	multicast     *yggmc.Multicast
	logger        *gl.Logger
	wantMulticast bool
	startedAt     time.Time
	stopped       bool

	// Входящие пакеты: read-насос раздаёт их подписчикам. Без подписчика насос
	// всё равно должен крутиться — иначе встанет и отправка.
	inboundMu sync.Mutex
	inbound   map[int]chan Packet
	nextSub   int
}

// AddressForKey возвращает IPv6 узла Yggdrasil по его Ed25519-ключу — ровно
// так же, как эталонная реализация (0200::/7 + старшие биты инвертированного
// ключа).
func AddressForKey(pub ed25519.PublicKey) net.IP {
	a := yggaddr.AddrForKey(pub)
	if a == nil {
		return nil
	}
	return net.IP(a[:])
}

// SubnetForKey возвращает /64-подсеть, маршрутизируемую на ключ узла.
func SubnetForKey(pub ed25519.PublicKey) *net.IPNet {
	s := yggaddr.SubnetForKey(pub)
	if s == nil {
		return nil
	}
	base := make([]byte, 16)
	copy(base, s[:])
	return &net.IPNet{IP: net.IP(base), Mask: net.CIDRMask(64, 128)}
}

// AddrForPublicKey превращает ключ узла в адрес для записи в PacketConn:
// yggdrasil адресует сессии по Ed25519-ключу, а не по IPv6.
func AddrForPublicKey(pub ed25519.PublicKey) net.Addr {
	if len(pub) != ed25519.PublicKeySize {
		return nil
	}
	key := make([]byte, ed25519.PublicKeySize)
	copy(key, pub)
	return iwt.Addr(key)
}

// NewYggNode собирает и запускает ноду yggdrasil. Ошибка возвращается, если
// ключ негодный или URI разборены криво: молча игнорировать кривой конфиг
// нельзя — пользователь увидит «онлайн» без сети.
func NewYggNode(opts NodeOptions) (*YggNode, error) {
	if len(opts.PrivateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("yggnode: ключ ноды должен быть ed25519 (%d байт), получено %d",
			ed25519.PrivateKeySize, len(opts.PrivateKey))
	}

	cfg := yggcfg.NodeConfig{PrivateKey: yggcfg.KeyBytes(opts.PrivateKey)}
	if err := cfg.GenerateSelfSignedCertificate(); err != nil {
		return nil, fmt.Errorf("yggnode: self-signed сертификат ноды: %w", err)
	}
	// ed25519 принимает любые 64 байта, не проверяя согласование половин. Если
	// приватная часть не соответствует публичной, узел получит адрес, под
	// который сеть не примет его подпись, — ловим это до старта.
	probe := []byte("yggnode:key-consistency")
	sig := ed25519.Sign(opts.PrivateKey, probe)
	if !ed25519.Verify(opts.PrivateKey.Public().(ed25519.PublicKey), probe, sig) {
		return nil, errors.New("yggnode: приватная и публичная части ключа ноды не согласованы")
	}

	coreOpts := make([]yggcore.SetupOption, 0, len(opts.Listen)+len(opts.Peers)+2)
	for _, l := range opts.Listen {
		uri, err := normalizeURI(l)
		if err != nil {
			return nil, fmt.Errorf("yggnode: слушатель %q: %w", l, err)
		}
		coreOpts = append(coreOpts, yggcore.ListenAddress(uri))
	}
	for _, p := range opts.Peers {
		uri, err := normalizePeerURI(p)
		if err != nil {
			return nil, fmt.Errorf("yggnode: пир %q: %w", p, err)
		}
		coreOpts = append(coreOpts, yggcore.Peer{URI: uri})
	}
	if opts.GroupPassword != "" {
		coreOpts = append(coreOpts, yggcore.GroupPassword(opts.GroupPassword))
	}
	if opts.NodeInfo != nil {
		coreOpts = append(coreOpts, yggcore.NodeInfo(opts.NodeInfo))
	}

	logger := newLogger()
	core, err := yggcore.New(cfg.Certificate, logger, coreOpts...)
	if err != nil {
		return nil, fmt.Errorf("yggnode: запуск ядра yggdrasil: %w", err)
	}

	return &YggNode{
		core:          core,
		logger:        logger,
		wantMulticast: opts.Multicast,
		inbound:       make(map[int]chan Packet),
	}, nil
}

// Start отмечает время запуска, поднимает read-насос и LAN-мультикаст.
// Слушатели и пиры yggdrasil стартуют вместе с ядром, в NewYggNode.
//
// Насос обязателен: ironwood отдаёт входящие кадры только пока кто-то зовёт
// ReadFrom (канал приёмного буфера ёмкостью 1), и без читателя на НАШЕЙ
// стороне не устанавливается даже сессия, то есть отправка молча стоит.
// Проверено экспериментально: без насоса пакет не доходит ни одним способом.
func (n *YggNode) Start() error {
	if !n.startedAt.IsZero() {
		return errors.New("yggnode: узел уже запущен")
	}
	n.startedAt = time.Now()
	go n.readPump()

	if !n.wantMulticast {
		return nil
	}
	mc, err := yggmc.New(n.core, n.logger, yggmc.MulticastInterface{
		Regex:  regexp.MustCompile(".*"),
		Beacon: true,
		Listen: true,
	})
	if err != nil {
		return fmt.Errorf("yggnode: LAN-мультикаст: %w", err)
	}
	n.multicast = mc
	return nil
}

// readPump читает пакеты оверлея, пока ядро не закрыто. Ошибка (в т.ч.
// types.ErrClosed после Stop) завершает цикл.
func (n *YggNode) readPump() {
	buf := make([]byte, n.MTU()+1)
	for {
		n2, from, err := n.core.ReadFrom(buf)
		if err != nil {
			return
		}
		key, ok := from.(iwt.Addr)
		if !ok || len(key) != ed25519.PublicKeySize || n2 == 0 {
			continue
		}
		pkt := Packet{
			From:    append([]byte(nil), key...),
			Payload: append([]byte(nil), buf[:n2]...),
		}
		n.inboundMu.Lock()
		for _, ch := range n.inbound {
			select {
			case ch <- pkt:
			default:
				// Подписчик не справляется: дешевле сбросить, чем тормозить
				// сетевой ввод всего узла.
			}
		}
		n.inboundMu.Unlock()
	}
}

// Packet — кадр, пришедший из оверлея. From — Ed25519-ключ узла-отправителя
// (это и есть адрес сессии ironwood), Payload — байты, записанные отправителем
// через WriteTo.
type Packet struct {
	From    ed25519.PublicKey
	Payload []byte
}

// Subscribe подписывает вызывающего на входящие пакеты. Вернувшаяся функция
// отписывает; после отписки канал больше не получает пакетов. Буфер size —
// сколько пакетов переживёт паузу подписчика, остальное отбрасывается.
func (n *YggNode) Subscribe(size int) (<-chan Packet, func()) {
	if size <= 0 {
		size = 64
	}
	ch := make(chan Packet, size)
	n.inboundMu.Lock()
	id := n.nextSub
	n.nextSub++
	n.inbound[id] = ch
	n.inboundMu.Unlock()

	return ch, func() {
		n.inboundMu.Lock()
		if _, ok := n.inbound[id]; ok {
			delete(n.inbound, id)
			close(ch)
		}
		n.inboundMu.Unlock()
	}
}

// Stop гасит мультикаст и ядро.
func (n *YggNode) Stop() error {
	if n.stopped {
		return nil
	}
	n.stopped = true
	if n.multicast != nil {
		if err := n.multicast.Stop(); err != nil {
			return fmt.Errorf("yggnode: остановка мультикаста: %w", err)
		}
	}
	n.core.Stop()
	return nil
}

// NodeIPv6 — адрес узла в 0200::/7, вычисленный yggdrasil из ключа ноды.
func (n *YggNode) NodeIPv6() net.IP { return n.core.Address() }

// PublicKey — Ed25519-ключ ноды: он же адрес сессии для записи в PacketConn.
func (n *YggNode) PublicKey() ed25519.PublicKey { return n.core.PublicKey() }

// MTU — максимальный размер полезной нагрузки пакета оверлея.
func (n *YggNode) MTU() uint64 { return n.core.MTU() }

// PacketConn отдаёт userspace-доступ к сети без TUN.
func (n *YggNode) PacketConn() *PacketConn { return &PacketConn{core: n.core} }

// Peers — реальный список пиров с состоянием соединения.
func (n *YggNode) Peers() []PeerStatus {
	infos := n.core.GetPeers()
	out := make([]PeerStatus, 0, len(infos))
	for _, p := range infos {
		status := PeerStatus{
			URI:       p.URI,
			Up:        p.Up,
			Inbound:   p.Inbound,
			Latency:   p.Latency,
			Uptime:    p.Uptime,
			RXBytes:   p.RXBytes,
			TXBytes:   p.TXBytes,
			RemoteKey: p.Key,
		}
		if p.LastError != nil {
			status.LastError = p.LastError.Error()
		}
		out = append(out, status)
	}
	return out
}

// AddPeer добавляет постоянный пир в рантайме: yggdrasil сама переподключится
// при обрыве.
func (n *YggNode) AddPeer(uri string) error {
	u, err := normalizePeerURI(uri)
	if err != nil {
		return fmt.Errorf("yggnode: пир %q: %w", uri, err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("yggnode: разбор URI пира: %w", err)
	}
	if err := n.core.AddPeer(parsed, ""); err != nil {
		return fmt.Errorf("yggnode: AddPeer(%s): %w", u, err)
	}
	return nil
}

// RemovePeer вычёркивает постоянного пира и рвёт уже поднятый линк: ядро
// yggdrasil отменяет контекст линка и закрывает соединение, поэтому повторы
// после вызова не возвращаются. URI нормализуется ровно так же, как в AddPeer,
// — линк ищется по строке URI, и различие в схеме или дополненном порте
// означало бы «пира нет», хотя он стоит.
//
// Отданной наружно ссылки это не касается: принятое соединение приходит со
// стороны собеседника, и его адрес мы не конфигурировали. Для него yggdrasil
// отвечает ErrLinkNotConfigured.
func (n *YggNode) RemovePeer(uri string) error {
	u, err := normalizePeerURI(uri)
	if err != nil {
		return fmt.Errorf("yggnode: пир %q: %w", uri, err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("yggnode: разбор URI пира: %w", err)
	}
	if err := n.core.RemovePeer(parsed, ""); err != nil {
		return fmt.Errorf("yggnode: RemovePeer(%s): %w", u, err)
	}
	return nil
}

// Uptime — сколько живёт нода (0 до Start).
func (n *YggNode) Uptime() time.Duration {
	if n.startedAt.IsZero() {
		return 0
	}
	return time.Since(n.startedAt)
}

// RoutingEntries — сколько узлов известно spanning-дереву этой ноды. Это не
// «соседи», а записей маршрутизации: 0 у изолированной ноды.
func (n *YggNode) RoutingEntries() uint64 {
	return uint64(n.core.GetSelf().RoutingEntries)
}

// HasRouteTo отвечает на вопрос «знает ли оверлей, куда слать пакет этому ключу».
// WriteTo в ironwood всегда принимает пакет (он лишь заводит сессию), поэтому
// без этой проверки ядро выдавало бы «доставлено» там, где маршрута нет.
// Учтите: это знание о маршруте, а не квитанция от получателя.
func (n *YggNode) HasRouteTo(pub ed25519.PublicKey) bool {
	if len(pub) != ed25519.PublicKeySize {
		return false
	}
	for _, t := range n.core.GetTree() {
		if bytes.Equal(t.Key, pub) {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// USERSPACE PACKETCONN
// -----------------------------------------------------------------------------

// PacketConn — чтение/запись пакетов оверлея поверх ядра yggdrasil. Адрес
// получателя — iwt.Addr(Ed25519 ключ ноды), см. AddrForPublicKey.
type PacketConn struct {
	core *yggcore.Core
}

func (c *PacketConn) ReadFrom(p []byte) (int, net.Addr, error) { return c.core.ReadFrom(p) }

func (c *PacketConn) WriteTo(p []byte, addr net.Addr) (int, error) { return c.core.WriteTo(p, addr) }

// LocalAddr — наш собственный адрес в оверлее.
func (c *PacketConn) LocalAddr() net.Addr {
	return &net.IPAddr{IP: c.core.Address()}
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// normalizeURI проверяет транспорт и наличие host:port для слушателя: пустой
// хост здесь означает «bind на всех интерфейсах» и легален, а вот порт
// обязателен — иначе ядро подняло бы ничего не принимающий узел.
func normalizeURI(raw string) (string, error) {
	return normalizeEndpoint(raw, false)
}

// normalizePeerURI строже: пиру нужен конкретный адрес, "tcp://:54321" — это
// не адрес пира, а забытый хост.
func normalizePeerURI(raw string) (string, error) {
	return normalizeEndpoint(raw, true)
}

func normalizeEndpoint(raw string, needHost bool) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "tcp", "tls", "socks", "ws", "wss", "quic":
	default:
		return "", fmt.Errorf("неизвестный транспорт %q (ожидается tcp/tls/socks/ws/wss/quic)", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("пустой host:port")
	}
	if u.Port() == "" {
		return "", fmt.Errorf("в %q нет порта", raw)
	}
	if needHost && u.Hostname() == "" {
		return "", fmt.Errorf("в %q нет адреса пира", raw)
	}
	return u.String(), nil
}

func newLogger() *gl.Logger {
	w := io.Discard
	if os.Getenv("YGGGRAM_DEBUG") != "" {
		w = os.Stderr
	}
	l := gl.New(w, "yggnode: ", 0)
	if os.Getenv("YGGGRAM_DEBUG") != "" {
		for _, lvl := range []string{"error", "warn", "info", "debug"} {
			l.EnableLevel(lvl)
		}
	}
	return l
}

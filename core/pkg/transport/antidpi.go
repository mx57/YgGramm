// File: core/pkg/transport/antidpi.go
//
// pkg/transport — каркас вне дата-пути, а не транспорт приложения. Вызывающий у
// него появился на Шаге 16 и это измерительный вход cmd/tspuprobe: он собирает
// отсюда ClientHello, считает им печатку из pkg/fingerprint и смотрит, на какой фазе
// обрывается соединение. До pkg/rpc, pkg/yggnode и приложения байты отсюда не
// доходят по-прежнему — это измеряет TestUnreachablePackagesMatchRegistry
// (core/pkg/rpc/architecture_test.go), а не память. С Шага 2 рукопожатие ссылок
// делает встроенная yggdrasil-go через crypto/tls и QUIC.
//
// Что обещано и чего в коде нет — построчно, чтобы файл не читался как описание
// работающего обхода фильтров:
//   - Фрагментация ClientHello на замерянной сети (Шаг 16, матрица
//     «9 целей × 6 профилей × 3 прогона») работает против нас: там, где целый
//     hello проходит 3/3, фрагментированный с SNI получает активный RST 0/3, а
//     там, где дроп по SNI тихий, фрагментация не обходит его ни разу. Значит
//     «разрезать hello» — не обход, а признак, по которому этот фильтр рвёт
//     соединение; включение в дата-путь требует пересъёмки на целевой сети.
//   - ModeUTLSChrome не эмулирует Chrome: зависимости refraction-networking/utls
//     в go.mod нет, рукопожатие делает stdlib crypto/tls. Замер Шага 4-b тем же
//     pkg/fingerprint даёт «голый» Go-клиент — без ALPN, без GREASE, одна
//     TLS-запись версии 0x0301. Отличия от браузерного отпечатка лежат ровно в
//     тех признаках, по которым такой трафик ищут.
//   - dialShadowTLS — собственный протокол (8 байт времени, 16 байт соли,
//     HMAC-SHA256 поверх них и ack «YG»), он несовместим с эталонным ShadowTLS v3
//     и к чужим мостам не подключится.
//   - Внешнее рукопожатие идёт с InsecureSkipVerify: true. Пока пакет вне
//     дата-пути, это мёртвый код; включать его в живое соединение можно только
//     вместе с реальной проверкой сертификата, иначе маскировка превращается в
//     готовый канал для человека посередине.
//   - TrafficMorpher (статистический паддинг и джиттер) не реализован вообще.
//   - bootstrap.go (DoH, DHT BEP-44) не вызывается: пиры ядру отдаёт конфиг и
//     multicast-обнаружение из pkg/yggnode.
//
// Чтобы пакет стал рабочим, нужно выбрать одну технику, подключить её к
// link-конфигу yggnode вместо crypto/tls и замерить получившийся отпечаток, а не
// описывать её в README как подключённую.
package transport

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// AntiDpiMode defines the evasion technique used for TCP/TLS connections
type AntiDpiMode int

const (
	ModeDisabled AntiDpiMode = iota
	ModeUTLSChrome
	ModeUTLSIOS
	ModeShadowTLSv3
	ModeWebSocketCDN
	ModeQUICObfs
)

// AntiDpiConfig stores parameters for censorship circumvention
type AntiDpiConfig struct {
	Mode               AntiDpiMode
	TargetSNI          string
	Password           string
	EnableFragmentTCP  bool
	FragmentSplitPoint int // Offset where ClientHello is split (e.g. 5, 20, or SNI boundary)
	WSSPath            string
	CustomHeaders      map[string]string
	ConnectTimeout     time.Duration
}

// DefaultAntiDpiConfig returns recommended settings for bypassing Russian TSPU filters
func DefaultAntiDpiConfig() AntiDpiConfig {
	return AntiDpiConfig{
		Mode:               ModeUTLSChrome,
		TargetSNI:          "www.cloudflare.com",
		Password:           "ygggram-anti-dpi-v1",
		EnableFragmentTCP:  true,
		FragmentSplitPoint: 5, // Split after 5-byte TLS record header
		WSSPath:            "/ygg-overlay",
		CustomHeaders:      make(map[string]string),
		ConnectTimeout:     10 * time.Second,
	}
}

// AntiDpiDialer encapsulates dialing logic for filtered environments
type AntiDpiDialer struct {
	cfg AntiDpiConfig
	mu  sync.RWMutex
}

// NewAntiDpiDialer creates a new dialer with specified evasion parameters
func NewAntiDpiDialer(cfg AntiDpiConfig) *AntiDpiDialer {
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.FragmentSplitPoint <= 0 {
		cfg.FragmentSplitPoint = 5
	}
	return &AntiDpiDialer{
		cfg: cfg,
	}
}

// DialContext connects to remote peer address using the configured anti-DPI strategy
func (d *AntiDpiDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d.mu.RLock()
	mode := d.cfg.Mode
	d.mu.RUnlock()

	switch mode {
	case ModeWebSocketCDN:
		return d.dialWebSocketCDN(ctx, addr)
	case ModeShadowTLSv3:
		return d.dialShadowTLS(ctx, addr)
	case ModeUTLSChrome, ModeUTLSIOS:
		return d.dialFragmentedTLS(ctx, addr, mode)
	default:
		// Direct TCP or Fragmented TLS if flag is set
		if d.cfg.EnableFragmentTCP {
			return d.dialFragmentedTLS(ctx, addr, ModeUTLSChrome)
		}
		var dialer net.Dialer
		dialer.Timeout = d.cfg.ConnectTimeout
		return dialer.DialContext(ctx, network, addr)
	}
}

// -----------------------------------------------------------------------------
// 1. TCP RECORD FRAGMENTATION & CHROME TLS CLIENTHELLO SPOOFING
// -----------------------------------------------------------------------------

// fragmentedConn wraps net.Conn to perform byte-level record splitting on write
type fragmentedConn struct {
	net.Conn
	splitPoint int
	wroteFirst bool
	mu         sync.Mutex
}

func (fc *fragmentedConn) Write(b []byte) (int, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Only fragment the very first packet (TLS ClientHello)
	if fc.wroteFirst || len(b) <= fc.splitPoint || fc.splitPoint <= 0 {
		return fc.Conn.Write(b)
	}

	fc.wroteFirst = true

	// Split 1: First chunk (e.g. 5 bytes TLS Record Header)
	n1, err := fc.Conn.Write(b[:fc.splitPoint])
	if err != nil {
		return n1, err
	}

	// Micro-sleep to force TCP stack to flush first segment separately
	time.Sleep(2 * time.Millisecond)

	// Split 2: Remaining ClientHello payload (including SNI)
	n2, err := fc.Conn.Write(b[fc.splitPoint:])
	return n1 + n2, err
}

func (d *AntiDpiDialer) dialFragmentedTLS(ctx context.Context, addr string, mode AntiDpiMode) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
		port = "443"
		addr = net.JoinHostPort(host, port)
	}

	var rawDialer net.Dialer
	rawDialer.Timeout = d.cfg.ConnectTimeout
	tcpConn, err := rawDialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("antidpi tcp dial failed to %s: %w", addr, err)
	}

	sni := d.cfg.TargetSNI
	if sni == "" {
		sni = host
	}

	// Wrap raw TCP connection with record fragmenter
	splitPoint := d.cfg.FragmentSplitPoint
	if splitPoint <= 0 {
		splitPoint = 5
	}
	fragConn := &fragmentedConn{
		Conn:       tcpConn,
		splitPoint: splitPoint,
	}

	// Конфиг собирается тем же кодом, что и у замерки (probe.go): два отдельных
	// описания tls.Config расходились бы молча, и измеряли бы «похожий»
	// handshake, а не этот. ALPN и списки шифров прежние.
	tlsConfig := clientTLSConfig(sni, []string{"h2", "http/1.1"}, true)

	tlsClient := tls.Client(fragConn, tlsConfig)
	handshakeDeadline := time.Now().Add(d.cfg.ConnectTimeout)
	if err := tlsClient.SetDeadline(handshakeDeadline); err != nil {
		_ = tcpConn.Close()
		return nil, err
	}

	if err := tlsClient.HandshakeContext(ctx); err != nil {
		_ = tcpConn.Close()
		return nil, fmt.Errorf("antidpi fragmented tls handshake failed (sni: %s): %w", sni, err)
	}

	// Reset deadline after successful handshake
	_ = tlsClient.SetDeadline(time.Time{})
	return tlsClient, nil
}

// -----------------------------------------------------------------------------
// 2. WEBSOCKET OVER CDN (wss://) TRANSPORT
// -----------------------------------------------------------------------------

// wsStreamConn wraps a WebSocket connection to implement net.Conn interface
type wsStreamConn struct {
	rawConn net.Conn
	reader  io.Reader
	mu      sync.Mutex
}

func (w *wsStreamConn) Read(b []byte) (int, error) {
	for {
		if w.reader != nil {
			n, err := w.reader.Read(b)
			if err == nil {
				return n, nil
			}
			if !errors.Is(err, io.EOF) {
				return n, err
			}
			w.reader = nil
			if n > 0 {
				return n, nil
			}
		}

		// Read next WebSocket frame header
		var header [2]byte
		if _, err := io.ReadFull(w.rawConn, header[:]); err != nil {
			return 0, err
		}

		opcode := header[0] & 0x0F
		isMasked := (header[1] & 0x80) != 0
		payloadLen := int64(header[1] & 0x7F)

		if payloadLen == 126 {
			var ext [2]byte
			if _, err := io.ReadFull(w.rawConn, ext[:]); err != nil {
				return 0, err
			}
			payloadLen = int64(binary.BigEndian.Uint16(ext[:]))
		} else if payloadLen == 127 {
			var ext [8]byte
			if _, err := io.ReadFull(w.rawConn, ext[:]); err != nil {
				return 0, err
			}
			payloadLen = int64(binary.BigEndian.Uint64(ext[:]))
		}

		var maskKey [4]byte
		if isMasked {
			if _, err := io.ReadFull(w.rawConn, maskKey[:]); err != nil {
				return 0, err
			}
		}

		// Handle Close / Ping / Pong control frames
		if opcode == 0x08 { // Close frame
			return 0, io.EOF
		} else if opcode == 0x09 { // Ping
			// Consume ping payload and respond with pong
			pingData := make([]byte, payloadLen)
			_, _ = io.ReadFull(w.rawConn, pingData)
			_ = w.writeFrame(0x0A, pingData)
			continue
		} else if opcode == 0x0A { // Pong
			_, _ = io.CopyN(io.Discard, w.rawConn, payloadLen)
			continue
		} else if opcode != 0x02 && opcode != 0x00 { // We expect Binary frame (0x02)
			_, _ = io.CopyN(io.Discard, w.rawConn, payloadLen)
			continue
		}

		// Read payload bytes
		buf := make([]byte, payloadLen)
		if _, err := io.ReadFull(w.rawConn, buf); err != nil {
			return 0, err
		}

		if isMasked {
			for i := 0; i < len(buf); i++ {
				buf[i] ^= maskKey[i%4]
			}
		}

		w.reader = bytes.NewReader(buf)
	}
}

func (w *wsStreamConn) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	err := w.writeFrame(0x02, b) // 0x02: Binary frame
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (w *wsStreamConn) writeFrame(opcode byte, data []byte) error {
	header := make([]byte, 0, 14)
	header = append(header, 0x80|opcode) // FIN + opcode

	length := len(data)
	// Client-to-server frames MUST be masked (RFC 6455 Section 5.3)
	if length < 126 {
		header = append(header, byte(0x80|length))
	} else if length <= 0xFFFF {
		header = append(header, byte(0x80|126))
		var b2 [2]byte
		binary.BigEndian.PutUint16(b2[:], uint16(length))
		header = append(header, b2[:]...)
	} else {
		header = append(header, byte(0x80|127))
		var b8 [8]byte
		binary.BigEndian.PutUint64(b8[:], uint64(length))
		header = append(header, b8[:]...)
	}

	// 4-byte random mask
	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return err
	}
	header = append(header, mask[:]...)

	maskedData := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		maskedData[i] = data[i] ^ mask[i%4]
	}

	if _, err := w.rawConn.Write(header); err != nil {
		return err
	}
	_, err := w.rawConn.Write(maskedData)
	return err
}

func (w *wsStreamConn) Close() error                       { return w.rawConn.Close() }
func (w *wsStreamConn) LocalAddr() net.Addr                { return w.rawConn.LocalAddr() }
func (w *wsStreamConn) RemoteAddr() net.Addr               { return w.rawConn.RemoteAddr() }
func (w *wsStreamConn) SetDeadline(t time.Time) error      { return w.rawConn.SetDeadline(t) }
func (w *wsStreamConn) SetReadDeadline(t time.Time) error  { return w.rawConn.SetReadDeadline(t) }
func (w *wsStreamConn) SetWriteDeadline(t time.Time) error { return w.rawConn.SetWriteDeadline(t) }

func (d *AntiDpiDialer) dialWebSocketCDN(ctx context.Context, addr string) (net.Conn, error) {
	// First establish TLS connection to CDN edge with SNI
	tlsConn, err := d.dialFragmentedTLS(ctx, addr, ModeUTLSChrome)
	if err != nil {
		return nil, fmt.Errorf("websocket cdn tls stage failed: %w", err)
	}

	host, _, _ := net.SplitHostPort(addr)
	if host == "" {
		host = addr
	}
	sni := d.cfg.TargetSNI
	if sni == "" {
		sni = host
	}

	wsPath := d.cfg.WSSPath
	if wsPath == "" {
		wsPath = "/ygg-overlay"
	}

	// Generate Sec-WebSocket-Key
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		_ = tlsConn.Close()
		return nil, err
	}
	wsKey := base64.StdEncoding.EncodeToString(nonce)

	// Build HTTP Upgrade request
	var req strings.Builder
	req.WriteString(fmt.Sprintf("GET %s HTTP/1.1\r\n", wsPath))
	req.WriteString(fmt.Sprintf("Host: %s\r\n", sni))
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	req.WriteString(fmt.Sprintf("Sec-WebSocket-Key: %s\r\n", wsKey))
	req.WriteString("Sec-WebSocket-Version: 13\r\n")
	req.WriteString("User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36\r\n")

	for k, v := range d.cfg.CustomHeaders {
		req.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	req.WriteString("\r\n")

	if _, err := tlsConn.Write([]byte(req.String())); err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("failed to send websocket upgrade: %w", err)
	}

	// Read HTTP 101 Switching Protocols response
	respBuf := make([]byte, 1024)
	n, err := tlsConn.Read(respBuf)
	if err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("failed reading websocket handshake response: %w", err)
	}

	respStr := string(respBuf[:n])
	if !strings.Contains(respStr, "101 Switching Protocols") {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("websocket upgrade rejected by CDN: %s", strings.Split(respStr, "\r\n")[0])
	}

	return &wsStreamConn{
		rawConn: tlsConn,
	}, nil
}

// -----------------------------------------------------------------------------
// 3. SHADOWTLS V3 / REALITY-STYLE HANDSHAKE MASKING
// -----------------------------------------------------------------------------

func (d *AntiDpiDialer) dialShadowTLS(ctx context.Context, addr string) (net.Conn, error) {
	// ShadowTLS establishes a dual-session:
	// 1. Valid TLS ClientHello with whitelisted Russian SNI (e.g. yandex.ru or vk.com).
	// 2. An HMAC-SHA256 authentication payload derived from pre-shared secret.
	// 3. The remote relay verifies the HMAC, disconnects the SNI target, and splices the Yggdrasil stream.

	tlsConn, err := d.dialFragmentedTLS(ctx, addr, ModeUTLSChrome)
	if err != nil {
		return nil, fmt.Errorf("shadowtls outer handshake failed: %w", err)
	}

	// Prepare challenge: Timestamp (8 bytes) + Salt (16 bytes) + HMAC (32 bytes)
	timestamp := uint64(time.Now().Unix())
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		_ = tlsConn.Close()
		return nil, err
	}

	challengeData := make([]byte, 24)
	binary.BigEndian.PutUint64(challengeData[0:8], timestamp)
	copy(challengeData[8:24], salt)

	mac := hmac.New(sha256.New, []byte(d.cfg.Password))
	mac.Write(challengeData)
	expectedHMAC := mac.Sum(nil)

	authPacket := make([]byte, 0, 56)
	authPacket = append(authPacket, challengeData...)
	authPacket = append(authPacket, expectedHMAC...)

	// Send authenticated handshake token
	if _, err := tlsConn.Write(authPacket); err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("shadowtls auth write error: %w", err)
	}

	// Await 4-byte acknowledgment
	var ack [4]byte
	_ = tlsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadFull(tlsConn, ack[:]); err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("shadowtls server failed to verify token: %w", err)
	}
	_ = tlsConn.SetReadDeadline(time.Time{})

	if ack[0] != 0x59 || ack[1] != 0x47 { // 'YG'
		_ = tlsConn.Close()
		return nil, errors.New("shadowtls invalid relay magic signature")
	}

	return tlsConn, nil
}

// ParsePeerURI extracts transport protocol, SNI, and target host:port
func ParsePeerURI(rawURI string) (AntiDpiMode, string, string, error) {
	// Formats supported:
	// - tls://peer.example.org:54321?sni=www.cloudflare.com
	// - wss://cdn.example.org:443/ws?sni=cdn.example.org
	// - shadowtls://bridge.example.org:8443?sni=yandex.ru&pwd=secret
	u, err := url.Parse(rawURI)
	if err != nil {
		return ModeDisabled, "", "", err
	}

	q := u.Query()
	sni := q.Get("sni")
	port := u.Port()
	host := u.Hostname()

	switch u.Scheme {
	case "tls":
		if port == "" {
			port = "54321"
		}
		return ModeUTLSChrome, sni, net.JoinHostPort(host, port), nil
	case "wss":
		if port == "" {
			port = "443"
		}
		return ModeWebSocketCDN, sni, net.JoinHostPort(host, port), nil
	case "shadowtls":
		if port == "" {
			port = "8443"
		}
		return ModeShadowTLSv3, sni, net.JoinHostPort(host, port), nil
	default:
		if port == "" {
			port = "54321"
		}
		return ModeDisabled, sni, net.JoinHostPort(host, port), nil
	}
}

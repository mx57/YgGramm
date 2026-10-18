// File: core/pkg/transport/probe.go
//
// Шаг 16: поле. До этого файла про транспорт можно было только рассказывать:
// рукопожатие ссылок делает встроенная yggdrasil, а байты отсюда в сети не
// бывали. Здесь появляется то, что делает утверждения проверяемыми: одинаковые
// профили исходящего ClientHello, честное разделение отказа по фазам (DNS, TCP,
// TLS) и снимок отпечатка именно того, что отправил dialer.
//
// Три правила, без которых замерка врёт:
//
//   - Профиль собирает тот же tls.Config, что и живой dialer. dialFragmentedTLS
//     больше не строит конфиг у себя в теле, а зовёт clientTLSConfig — иначе
//     измерялся бы «похожий» handshake, а не тот, который код действительно
//     отправляет. Равенство держит тест: отпечаток пути dialer и отпечаток пути
//     замерки обязаны совпасть на одном сервере.
//   - Разрез считается по смещению SNI, снятому с ДЕМОНА. Дать имя хоста в
//     первом куске или нет — это и есть предмет спора с посредником, поэтому
//     резать надо ровно по границе имени, а не по числу «обычно 174 байта».
//     Смещение берётся из холостого handshake в трубу (байты hello от адреса не
//     зависят, длины полей постоянны), и на живом снимке оно перепроверяется:
//     если живая длина не совпала с холостой, результат помечается названной
//     причиной, а не выдаётся за «разрез по SNI».
//   - Отказ называется по фазе. «Не открылось» за DNS, за зависшим SYN и за
//     разорванным рукопожатием — три разные диагностики, и сводить их к одной
//     строчке ошибки нельзя: именно различие фаз и отвечает на вопрос, режет ли
//     посредник по имени, по адресу или по самому факту TLS.
//
// Что замерка не утверждает: она не проверяет сертификаты (InsecureSkipVerify нужен
// ровно для одного: посчитать handshake с произвольной целью) и не передаёт
// полезных данных. В дата-путь приложения этот код не подключён и не подключается
// этим шагом: ссылки по-прежнему делает yggdrasil.
package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	"ygggram/core/pkg/fingerprint"
)

// SplitMode — где резать первую запись handshake.
type SplitMode int

const (
	SplitNone         SplitMode = iota // одна запись одним вызовом, как stdlib
	SplitRecordHeader                  // первые 5 байт (заголовок TLS-записи) отдельно
	SplitBeforeSNI                     // отдельно всё, что идёт до первого байта имени
)

// HelloProfile — именованный набор того, что уходит в сеть. Имена попадают в
// отчёт, поэтому переименовывать их молча нельзя: «go-alpn» и «go-alpn-frag-sni»
// — это разные утверждения о посреднике.
type HelloProfile struct {
	Name          string
	SendSNI       bool
	ALPN          []string
	BrowserSuites bool
	Split         SplitMode
	// ForceSNI — имя, которое уходит в SNI, когда адрес цели на него не похож.
	// Без этого поля нельзя отделить «посредник режет по имени» от «посредник
	// режет по адресу»: crypto/tls не отправляет SNI для IP-литерала вовсе.
	ForceSNI string
}

// browserCipherSuites — тот самый список, которым dialer прикрывает «Chrome-
// совместимость». Он ничего не эмулирует: порядок и набор выбирает stdlib, а
// список здесь задан руками. Назван переменной, чтобы у dialer и у замерки была
// одна источник истина.
var browserCipherSuites = []uint16{
	tls.TLS_AES_128_GCM_SHA256,
	tls.TLS_AES_256_GCM_SHA384,
	tls.TLS_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
}

// clientTLSConfig — единственное место, где собирается конфиг исходящего
// handshake. dialFragmentedTLS и замерка зовут её обе.
func clientTLSConfig(sni string, alpn []string, useBrowserSuites bool) *tls.Config {
	cfg := &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	}
	if len(alpn) > 0 {
		cfg.NextProtos = append([]string(nil), alpn...)
	}
	if useBrowserSuites {
		cfg.CipherSuites = append([]uint16(nil), browserCipherSuites...)
	}
	return cfg
}

// Profiles — набор, которым меряется различимость. Порядок зафиксирован: отчёт по нему
// сверяется в README, и перестановка изменила бы таблицы без смены смысла.
func Profiles() []HelloProfile {
	return []HelloProfile{
		{Name: "go-bare", SendSNI: true, Split: SplitNone},
		{Name: "go-alpn", SendSNI: true, ALPN: []string{"h2", "http/1.1"}, BrowserSuites: true, Split: SplitNone},
		{Name: "go-alpn-frag5", SendSNI: true, ALPN: []string{"h2", "http/1.1"}, BrowserSuites: true, Split: SplitRecordHeader},
		{Name: "go-alpn-frag-sni", SendSNI: true, ALPN: []string{"h2", "http/1.1"}, BrowserSuites: true, Split: SplitBeforeSNI},
		{Name: "no-sni", SendSNI: false, Split: SplitNone},
		{Name: "no-sni-frag5", SendSNI: false, Split: SplitRecordHeader},
	}
}

// Фаза, на которой встал handshake. Именно это различие читает посредник, и
// именно его нельзя терять в одной строке ошибки.
const (
	PhaseResolve = "resolve"
	PhaseTCP     = "tcp"
	PhaseTLS     = "tls"
	PhaseTLSOK   = "tls-ok"
)

// Outcome — одна строка отчёта. Пустой JA3/JA4 означает «до отпечатка дело не
// дошло», а не «отпечатка нет»: фазы различимы.
type Outcome struct {
	Profile string
	Target  string
	// Host — имя или адрес, как его дал вызывающий; IP — то, куда реально
	// приехали. Разница между ними и есть ответ на вопрос «резолвилось или нет».
	Host       string
	IP         string
	Phase      string
	JA3        string
	JA4        string
	SNI        string
	Segments   []int
	HelloBytes int
	Elapsed    time.Duration
	Err        string
	Notes      []string
}

// String — одна строка таблицы. Числа печатаются всегда, кроме пустого отпечатка.
func (o Outcome) String() string {
	seg := "-"
	if len(o.Segments) > 0 {
		parts := make([]string, len(o.Segments))
		for i, s := range o.Segments {
			parts[i] = fmt.Sprintf("%d", s)
		}
		seg = strings.Join(parts, "+")
	}
	// Время в отчёте не косметика: «тихий дроп после ClientHello» и «мгновенный
	// RST» различаются именно им, а фазы у обоих одна — tls.
	line := fmt.Sprintf("%-24s %-26s %-15s phase=%-8s ja3=%s ja4=%s segments=%s hello=%d t=%s",
		o.Profile, o.Target, o.IP, o.Phase, dash(o.JA3), dash(o.JA4), seg, o.HelloBytes,
		o.Elapsed.Round(time.Millisecond))
	if o.Err != "" {
		line += " err=" + o.Err
	}
	for _, n := range o.Notes {
		line += " note=" + n
	}
	return line
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// RunProbe поднимает один handshake данного профиля к addr и возвращает, чем он
// кончился. addr — это host:port без схемы; если host не адрес, он резолвится
// отдельно, и отказ резолва не путается с отказом соединения.
func RunProbe(ctx context.Context, addr string, p HelloProfile) Outcome {
	start := time.Now()
	out := Outcome{Profile: p.Name, Target: addr}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		out.Host = addr
		host, port = addr, "443"
		out.Target = net.JoinHostPort(host, port)
		out.Notes = append(out.Notes, "порт не задан — взят 443")
	}
	out.Host = host

	ip := host
	if net.ParseIP(host) == nil {
		ips, err := lookupHost(ctx, host)
		if err != nil {
			out.Phase = PhaseResolve
			out.Err = err.Error()
			out.Elapsed = time.Since(start)
			return out
		}
		ip = ips[0]
		out.IP = ip
		out.Notes = append(out.Notes, fmt.Sprintf("адресов по имени: %d", len(ips)))
	} else {
		out.IP = ip
	}

	dialAddr := net.JoinHostPort(ip, port)
	splitAt, note, err := splitPointFor(ctx, p, profileSNI(host, p))
	if err != nil {
		out.Phase = PhaseTLS
		out.Err = err.Error()
		out.Elapsed = time.Since(start)
		return out
	}
	if note != "" {
		out.Notes = append(out.Notes, note)
	}

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", dialAddr)
	if err != nil {
		out.Phase = PhaseTCP
		out.Err = classify(err)
		out.Elapsed = time.Since(start)
		return out
	}
	defer conn.Close()

	rec := fingerprint.NewRecorder(conn)
	var wire net.Conn = rec
	if splitAt > 0 {
		wire = &recordSplitter{Conn: rec, first: splitAt}
	}

	sni := profileSNI(host, p)
	cli := tls.Client(wire, clientTLSConfig(sni, p.ALPN, p.BrowserSuites))
	hsCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		hsCtx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	if err := cli.HandshakeContext(hsCtx); err != nil {
		out.Phase = PhaseTLS
		out.Err = classify(err)
		// Отпечаток снимается и при неудаче: посредник мог проглотить ответ,
		// но наши байты он увидел, и именно они предмет спора.
		if h, helloErr := rec.Hello(); helloErr == nil {
			fillHello(&out, h, rec)
		}
		out.Elapsed = time.Since(start)
		return out
	}
	out.Phase = PhaseTLSOK
	h, err := rec.Hello()
	if err != nil {
		out.Notes = append(out.Notes, "отпечаток не снят: "+err.Error())
	} else {
		fillHello(&out, h, rec)
		if p.Split == SplitBeforeSNI && h.ServerName != "" {
			if off, err := fingerprint.SNIByteOffset(rec.Captured(), h.ServerName); err == nil && off != splitAt {
				out.Notes = append(out.Notes, fmt.Sprintf("разрез=%d а SNI=%d", splitAt, off))
			}
		}
	}
	_ = cli.Close()
	out.Elapsed = time.Since(start)
	return out
}

// profileSNI — то имя, которое пойдёт в SNI. Порог честности: пустая строка
// означает «имя не отправляется», и это отдельный профиль, а не недосмотр.
func profileSNI(host string, p HelloProfile) string {
	if !p.SendSNI {
		return ""
	}
	if p.ForceSNI != "" {
		return p.ForceSNI
	}
	return host
}

func fillHello(o *Outcome, h *fingerprint.ClientHello, rec *fingerprint.Recorder) {
	o.JA3 = h.JA3()
	o.JA4 = h.JA4()
	o.SNI = h.ServerName
	o.Segments = rec.Segments()
	o.HelloBytes = len(rec.Captured())
	if h.Fragmented() {
		o.Notes = append(o.Notes, fmt.Sprintf("записей=%d", len(h.RecordVersions)))
	}
}

// splitPointFor — сколько байт отдать первым куском. Для SplitBeforeSNI смещение
// снимается с холостого handshake в трубу: имя и списки задаются конфигом, а
// адрес на них не влияет, так что разрез можно спланировать до соединения.
func splitPointFor(ctx context.Context, p HelloProfile, sni string) (int, string, error) {
	switch p.Split {
	case SplitNone:
		return 0, "", nil
	case SplitRecordHeader:
		return 5, "", nil
	case SplitBeforeSNI:
		if sni == "" {
			return 0, "разрез по SNI без SNI невозможен — профиль идёт целиком", nil
		}
		raw, err := captureHelloLocally(ctx, p, sni)
		if err != nil {
			return 0, "", err
		}
		off, err := fingerprint.SNIByteOffset(raw, sni)
		if err != nil {
			return 0, "смещение SNI не нашлось: " + err.Error(), nil
		}
		return off, "", nil
	}
	return 0, "", fmt.Errorf("неизвестный режим разреза %d", int(p.Split))
}

// captureHelloLocally собирает байты hello этого профиля, не выходя в сеть:
// tls.Client поверх net.Pipe, читатель которого всё выбрасывает. Ждём не конца
// рукопожатия (отвечать нечем, оно бы встало до дедлайна), а момента, когда
// снимок стал целым ClientHello: это миллисекунды, и холостой проход не должен
// стоить как таймаут.
func captureHelloLocally(ctx context.Context, p HelloProfile, host string) ([]byte, error) {
	client, server := net.Pipe()
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := server.Read(buf); err != nil {
				return
			}
		}
	}()
	rec := fingerprint.NewRecorder(client)
	cli := tls.Client(rec, clientTLSConfig(profileSNI(host, p), p.ALPN, p.BrowserSuites))
	_ = cli.SetDeadline(time.Now().Add(time.Second))
	hsDone := make(chan struct{})
	go func() {
		_ = cli.HandshakeContext(ctx)
		close(hsDone)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		if _, err := rec.Hello(); err == nil {
			break
		}
		select {
		case <-hsDone:
		case <-time.After(5 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			raw := rec.Captured()
			_ = cli.Close()
			_ = client.Close()
			_ = server.Close()
			return nil, fmt.Errorf("холостой hello не снялся: байт %d", len(raw))
		}
	}
	raw := rec.Captured()
	_ = cli.Close()
	_ = client.Close()
	_ = server.Close()
	return raw, nil
}

// lookupHost запрашивает имя у DNS, но только для того, чтобы отделить «домен режут по
// DNS» от «адрес недоступен». Первый ответ используется как цель замера.
func lookupHost(ctx context.Context, host string) ([]string, error) {
	r := net.Resolver{PreferGo: false}
	ips, err := r.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", classify(err), err)
	}
	if len(ips) == 0 {
		return nil, errors.New("resolver вернул пустой список")
	}
	return ips, nil
}

// classify раскладывает ошибку по фазам не по тексту сообщения, а по типу:
// таймаут соединения, отказ соединения, обрыв посередине и «сервер ответил
// ерундой» — разные диагнозы.
func classify(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "отменено"
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return "обрыв"
	}
	if name := errnoDiagnosis(err); name != "" {
		return name
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return "nxdomain"
		}
		return "dns"
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return "tcp"
		}
	}
	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return "сертификат"
	}
	var alert tls.AlertError
	if errors.As(err, &alert) {
		return "alert:" + strings.TrimPrefix(alert.Error(), "tls: ")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	text := strings.ReplaceAll(err.Error(), " ", "_")
	if len(text) > 80 {
		text = text[:80]
	}
	return text
}

// errnoDiagnosis сопоставляет код сокета с диагнозом, и здесь нужны оба набора
// по замеренной причине: на Windows winsock кладёт в ошибку raw WSA-код (10061
// «подключиться отказались»), а syscall.ECONNREFUSED в Go на Windows — это
// 536870934 из собственного диапазона, и errors.Is на них не сходится. На
// POSIX, наоборот, работает errors.Is. Один только второй набор читался бы на
// десктопе как «диагноза нет», и это ровно тот случай, когда отказ посредника
// потерялся бы в пустой строке.
func errnoDiagnosis(err error) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return ""
	}
	switch int(errno) {
	case 10061: // WSAECONNREFUSED
		return "refused"
	case 10054, 10053: // WSAECONNRESET, WSAECONNABORTED
		return "reset"
	case 10065, 10051: // WSAEHOSTUNREACH, WSAENETUNREACH
		return "недостижимо"
	case 10060: // WSAETIMEDOUT
		return "timeout"
	case 10013: // WSAEACCES — соединение зарубил локальный фильтр
		return "запрещено"
	}
	switch {
	case errors.Is(errno, syscall.ECONNREFUSED):
		return "refused"
	case errors.Is(errno, syscall.ECONNRESET):
		return "reset"
	case errors.Is(errno, syscall.EHOSTUNREACH), errors.Is(errno, syscall.ENETUNREACH):
		return "недостижимо"
	case errors.Is(errno, syscall.ETIMEDOUT):
		return "timeout"
	}
	return ""
}

// recordSplitter — то же, что fragmentedConn, но с разрезом по любому смещению и
// без состояния «первый пакет»: замерке нужно резать ровно одну запись.
type recordSplitter struct {
	net.Conn
	first int
	done  bool
}

func (c *recordSplitter) Write(b []byte) (int, error) {
	if c.done || len(b) <= c.first || c.first <= 0 {
		c.done = true
		return c.Conn.Write(b)
	}
	c.done = true
	n, err := c.Conn.Write(b[:c.first])
	if err != nil {
		return n, err
	}
	time.Sleep(2 * time.Millisecond)
	m, err := c.Conn.Write(b[c.first:])
	return n + m, err
}

// File: core/pkg/transport/probe_test.go
//
// Проверки замерки — целиком на loopback, без сети и без внешних хостов. Они
// сторожат четыре вещи, без которых полевой отчёт был бы набором слов:
//
//   - фазы отказа не сводятся к одной строке: «домен не резолвится», «порт
//     закрыт», «рукопожатие порвалось» и «всё открылось» — четыре разных
//     значения, и тест вызывает каждое намеренно;
//   - профиль меняет отпечаток ровно там, где заявлено: ALPN и списки шифров
//     сдвигают JA3, разрез по TCP-потоку — нет;
//   - dialer и замерка собирают один и тот же handshake. Снимок берётся на
//     сервере, а не вторым Recorder той же стороны: иначе «общий конфиг»
//     остался бы верой в один вызов функции;
//   - смещение SNI, снятое с холостого handshake, совпадает с живым разрезом.
//
// Живых посредников здесь нет намеренно. Пройдёт ли разрез через фильтрующую
// сеть — меряет cmd/tspuprobe, и ни один тест этого файла того не обещает.

package transport

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"ygggram/core/pkg/fingerprint"
)

func probeCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ключ не сгенерирован: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "transport-probe"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(10 * time.Minute),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:         true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("сертификат не собрался: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// peekConn отдаёт прочитанное наперёд, а потом читает сокет. Без него снимок
// hello, снятый сервером до tls.Server, съел бы байты рукопожатия.
type peekConn struct {
	net.Conn
	buf []byte
	off int
}

func (c *peekConn) Read(p []byte) (int, error) {
	if c.off < len(c.buf) {
		n := copy(p, c.buf[c.off:])
		c.off += n
		return n, nil
	}
	return c.Conn.Read(p)
}

// probeServer — TLS-сервер на всех интерфейсах, складывающий разобранный
// ClientHello каждого соединения в канал. Await забирает следующий по порядку:
// проверки идут последовательно, поэтому «N-й запрос — N-й снимок» честно.
type probeServer struct {
	addr   string
	stop   func()
	hellos chan *fingerprint.ClientHello
}

func newProbeServer(t *testing.T) *probeServer {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("листенер не поднялся: %v", err)
	}
	cert := probeCert(t)
	hellos := make(chan *fingerprint.ClientHello, 16)
	go func() {
		for {
			raw, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer raw.Close()
				_ = raw.SetReadDeadline(time.Now().Add(15 * time.Second))
				var stream []byte
				var hello *fingerprint.ClientHello
				chunk := make([]byte, 4096)
				for hello == nil {
					n, err := raw.Read(chunk)
					if n > 0 {
						stream = append(stream, chunk[:n]...)
						if h, ok, serr := fingerprint.Scan(stream); serr == nil && ok {
							hello = h
						}
					}
					if err != nil {
						return
					}
				}
				hellos <- hello
				tc := tls.Server(&peekConn{Conn: raw, buf: stream}, &tls.Config{Certificates: []tls.Certificate{cert}})
				defer tc.Close()
				_, _ = io.Copy(tc, tc)
			}()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port
	return &probeServer{
		// Имя, а не адрес: с IP-литералом crypto/tls не отправляет SNI вовсе, и
		// профили со SNI проверялись бы на безымянном handshake.
		addr:   fmt.Sprintf("localhost:%d", port),
		stop:   func() { _ = ln.Close() },
		hellos: hellos,
	}
}

func (s *probeServer) await(t *testing.T) *fingerprint.ClientHello {
	t.Helper()
	select {
	case h := <-s.hellos:
		return h
	case <-time.After(10 * time.Second):
		t.Fatal("сервер не увидел целого ClientHello")
		return nil
	}
}

// plainTCPServer принимает соединение и тут же обрывает: рукопожатие не
// состоится, и это ветка PhaseTLS с «обрыв».
func plainTCPServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("листенер не поднялся: %v", err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	t.Cleanup(func() { _ = ln.Close() })
	return ln.Addr().String()
}

func closedPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("листенер не поднялся: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func TestProbeClassifiesFailureByPhase(t *testing.T) {
	ctx := context.Background()

	t.Run("резолв", func(t *testing.T) {
		o := RunProbe(ctx, "yggram-nerazreshaemogo.invalid:443", HelloProfile{Name: "test", SendSNI: true})
		if o.Phase != PhaseResolve {
			t.Fatalf("отказ резолва попал в другую фазу: %+v", o)
		}
		if o.Err == "" {
			t.Fatalf("фаза названа, а причины нет")
		}
		if o.JA3 != "" {
			t.Fatalf("дело до отправки не дошло, а отпечаток проставлен: %s", o.JA3)
		}
	})

	t.Run("tcp", func(t *testing.T) {
		o := RunProbe(ctx, closedPort(t), HelloProfile{Name: "test", SendSNI: true})
		if o.Phase != PhaseTCP {
			t.Fatalf("отказ соединения попал в другую фазу: %+v", o)
		}
		if !strings.Contains(o.Err, "refused") && !strings.Contains(o.Err, "недостижимо") {
			t.Fatalf("отказ TCP назван непонятно: %q", o.Err)
		}
	})

	t.Run("обрыв на рукопожатии", func(t *testing.T) {
		o := RunProbe(ctx, plainTCPServer(t), HelloProfile{Name: "test", SendSNI: true})
		if o.Phase != PhaseTLS {
			t.Fatalf("обрыв после отправки hello записан другой фазой: %+v", o)
		}
		// Наши байты ушли до обрыва — снимок обязан остаться: предмет спора с
		// посредником именно в них, а не в том, что ответил сервер.
		if o.JA3 == "" || len(o.Segments) == 0 {
			t.Fatalf("рукопожатие порвалось, а снимок наших байтов пропал: %+v", o)
		}
	})

	t.Run("рукопожатие удалось", func(t *testing.T) {
		srv := newProbeServer(t)
		defer srv.stop()
		o := RunProbe(ctx, srv.addr, HelloProfile{Name: "test", SendSNI: true})
		if o.Phase != PhaseTLSOK {
			t.Fatalf("удачное рукопожатие записано отказом: %+v", o)
		}
		if len(o.JA3) != 32 || o.JA4 == "" {
			t.Fatalf("печатки невозможной длины: %+v", o)
		}
		if got := srv.await(t); got.JA3() != o.JA3 {
			t.Fatalf("серверный и клиентский снимки одного handshake разошлись: %s против %s", got.JA3(), o.JA3)
		}
	})
}

func TestProbeProfilesMoveFingerprintWhereClaimed(t *testing.T) {
	srv := newProbeServer(t)
	defer srv.stop()
	bare := HelloProfile{Name: "go-bare", SendSNI: true}
	alpn := HelloProfile{Name: "go-alpn", SendSNI: true, ALPN: []string{"h2", "http/1.1"}, BrowserSuites: true}
	frag5 := alpn
	frag5.Name = "go-alpn-frag5"
	frag5.Split = SplitRecordHeader
	noSNI := HelloProfile{Name: "no-sni", SendSNI: false}

	ctx := context.Background()
	a := RunProbe(ctx, srv.addr, bare)
	aSeen := srv.await(t)
	b := RunProbe(ctx, srv.addr, alpn)
	bSeen := srv.await(t)
	c := RunProbe(ctx, srv.addr, frag5)
	cSeen := srv.await(t)
	d := RunProbe(ctx, srv.addr, noSNI)
	dSeen := srv.await(t)

	for _, o := range []Outcome{a, b, c, d} {
		if o.Phase != PhaseTLSOK {
			t.Fatalf("профиль %s не дошёл до рукопожатия: %+v", o.Profile, o)
		}
	}
	if aSeen.JA3() == bSeen.JA3() {
		t.Fatalf("ALPN и другой список шифров не сдвинули JA3: %s", aSeen.JA3())
	}
	if !bSeen.HasALPN() || aSeen.HasALPN() {
		t.Fatalf("ALPN дошёл не туда: bare=%v alpn=%v", aSeen.Protocols, bSeen.Protocols)
	}
	// Разрез по TCP-потоку: сервер видит ровно те же байты, значит печатка та же.
	if cSeen.JA3() != bSeen.JA3() || cSeen.JA4() != bSeen.JA4() {
		t.Fatalf("разрез сдвинул отпечаток на серверной стороне: %s/%s против %s/%s",
			cSeen.JA3(), cSeen.JA4(), bSeen.JA3(), bSeen.JA4())
	}
	if len(c.Segments) < 2 || c.Segments[0] != 5 {
		t.Fatalf("разрез по 5 байт не виден в границах: %v", c.Segments)
	}
	if len(b.Segments) != 1 {
		t.Fatalf("целый профиль ушёл не одной записью: %v", b.Segments)
	}
	if dSeen.ServerName != "" || d.SNI != "" {
		t.Fatalf("в профиле без SNI имя всё-таки ушло: %q / %q", dSeen.ServerName, d.SNI)
	}
}

// TestDialerHelloEqualsProbeHello — общий конфиг, измеренный, а не заявленный.
// Dialer идёт через AntiDpiDialer (режим «Chrome» с разрезом по заголовку),
// замерка — через RunProbe с тем же профилем; снимок берётся на сервере, так что
// ни одна сторона не сверяется сама с собой.
func TestDialerHelloEqualsProbeHello(t *testing.T) {
	srv := newProbeServer(t)
	defer srv.stop()

	dialer := NewAntiDpiDialer(AntiDpiConfig{
		Mode:               ModeUTLSChrome,
		TargetSNI:          "localhost",
		EnableFragmentTCP:  true,
		FragmentSplitPoint: 5,
		ConnectTimeout:     8 * time.Second,
	})
	c, err := dialer.DialContext(context.Background(), "tcp", srv.addr)
	if err != nil {
		t.Fatalf("dialer не поднялся: %v", err)
	}
	_ = c.Close()
	fromDialer := srv.await(t)

	probe := RunProbe(context.Background(), srv.addr, HelloProfile{
		Name:          "go-alpn-frag5",
		SendSNI:       true,
		ALPN:          []string{"h2", "http/1.1"},
		BrowserSuites: true,
		Split:         SplitRecordHeader,
	})
	if probe.Phase != PhaseTLSOK {
		t.Fatalf("замерка не дошла до рукопожатия: %+v", probe)
	}
	if fromDialer.JA3() != probe.JA3 || fromDialer.JA4() != probe.JA4 {
		t.Fatalf("dialer и замерка собирают разные handshake: %s/%s против %s/%s",
			fromDialer.JA3(), fromDialer.JA4(), probe.JA3, probe.JA4)
	}
	if fromDialer.ServerName != "localhost" {
		t.Fatalf("TargetSNI не дошёл до сети: %q", fromDialer.ServerName)
	}
}

// TestSplitBeforeSNIAgreesWithOfflineCapture — смещение, снятое с холостого
// handshake, обязано оказаться ровно там, где живое рукопожатие разрывает поток.
func TestSplitBeforeSNIAgreesWithOfflineCapture(t *testing.T) {
	srv := newProbeServer(t)
	defer srv.stop()

	prof := HelloProfile{
		Name:          "go-alpn-frag-sni",
		SendSNI:       true,
		ALPN:          []string{"h2", "http/1.1"},
		BrowserSuites: true,
		Split:         SplitBeforeSNI,
	}
	dry, err := captureHelloLocally(context.Background(), prof, "localhost")
	if err != nil {
		t.Fatalf("холостой hello не снялся: %v", err)
	}
	off, err := fingerprint.SNIByteOffset(dry, "localhost")
	if err != nil {
		t.Fatalf("смещение SNI на холостом снимке не нашлось: %v", err)
	}
	if off <= 5 {
		t.Fatalf("смещение %d попало в заголовок записи", off)
	}
	if strings.Contains(string(dry[:off]), "localhost") {
		t.Fatalf("в первый кусок попало имя хоста: разрез по %d неверен", off)
	}

	o := RunProbe(context.Background(), srv.addr, prof)
	if o.Phase != PhaseTLSOK {
		t.Fatalf("рукопожатие через разрез по SNI не состоялось: %+v", o)
	}
	if len(o.Segments) < 2 || o.Segments[0] != off {
		t.Fatalf("живой разрез (%v) не совпал с холостым смещением %d", o.Segments, off)
	}
	for _, n := range o.Notes {
		if strings.Contains(n, "разрез") {
			t.Fatalf("живой снимок сам сообщил о расхождении: %v", o.Notes)
		}
	}
}

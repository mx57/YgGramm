// File: core/pkg/fingerprint/measure_test.go
//
// Офлайновые проверки снимщика — на loopback, без сети и без внешних хостов.
// Проверяются ровно те обещания, которые measure.go даёт полям:
//
//   - снятые байты — это и есть целый ClientHello, и они разбираются;
//   - после того как hello собран, снимок перестаёт расти: прикладные байты в
//     него не попадают (иначе «отпечаток» со временем стал бы просто дампом
//     соединения);
//   - разрез внутри одной TLS-записи меняет границы записей, но не байтовый
//     поток, а значит не меняет ни JA3, ни JA4, ни разбор полей. Это центральный
//     инвариант всего обхода: если он нарушится, «фрагментация» превратится в
//     «другой клиент», и сравнение в поле станет бессмысленным;
//   - смещение SNI находится только по уникальному вхождению имени, перед которым
//     стоит структура host_name; неоднозначный случай называется ошибкой, а не
//     режется наугад.
//
// Чего здесь нет намеренно: живых посредников. Пройдёт ли разрез через
// действующую фильтрующую сеть — это измеряет cmd/tspuprobe, и ни один тест в
// этом файле того утверждения не держит.

package fingerprint

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// testCert — самоподписанный сертификат на десять минут. Нужен только чтобы
// рукопожатие на loopback дошло до конца; проверкой сертификата этот тест не
// занимается и потому клиент идёт с InsecureSkipVerify.
func testCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ключ не сгенерирован: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "fingerprint-loopback"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(10 * time.Minute),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"example.test"},
		IsCA:         true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("сертификат не собрался: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// loopbackServer — эхо-сервер поверх tls.Server на 127.0.0.1. Возвращает адрес и
// остановку; соединение живёт до дедлайна, читатель на сервере обязателен, иначе
// рукопожатие TLS 1.3 не завершится.
func loopbackServer(t *testing.T) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("loopback-листенер не поднялся: %v", err)
	}
	cert := testCert(t)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				tc := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{cert}})
				defer tc.Close()
				_ = tc.SetDeadline(time.Now().Add(15 * time.Second))
				buf := make([]byte, 4096)
				for {
					n, err := tc.Read(buf)
					if err != nil {
						return
					}
					if _, err := tc.Write(buf[:n]); err != nil {
						return
					}
				}
			}()
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

// dialRecorded поднимает клиент поверх Recorder и возвращает снимок. splitPoint>0
// ставит разрезатель первой записи между TLS-слоем и снимщиком — ровно то, что
// делает фрагментатор в поле.
func dialRecorded(t *testing.T, addr string, splitPoint int) (*ClientHello, []int, []byte) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("до loopback не достучались: %v", err)
	}
	rec := NewRecorder(conn)
	var wire net.Conn = rec
	if splitPoint > 0 {
		wire = &splitConn{Conn: rec, first: splitPoint}
	}
	cli := tls.Client(wire, &tls.Config{
		ServerName:         "example.test",
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	})
	_ = cli.SetDeadline(time.Now().Add(10 * time.Second))
	if err := cli.Handshake(); err != nil {
		t.Fatalf("рукопожатие на loopback не состоялось: %v", err)
	}
	// Прикладной обмен после рукопожатия: он обязан остаться за пределами снимка.
	if _, err := cli.Write([]byte("ping")); err != nil {
		t.Fatalf("прикладная запись не прошла: %v", err)
	}
	buf := make([]byte, 16)
	if _, err := cli.Read(buf); err != nil {
		t.Fatalf("прикладное чтение не прошло: %v", err)
	}
	hello, err := rec.Hello()
	if err != nil {
		t.Fatalf("снятый hello не разбирается: %v", err)
	}
	segs := rec.Segments()
	raw := rec.Captured()
	_ = cli.Close()
	return hello, segs, raw
}

// splitConn режет ПЕРВУЮ запись на два вызова с паузой, остальное отдаёт целиком.
type splitConn struct {
	net.Conn
	first int
	mu    sync.Mutex
	split bool
}

func (c *splitConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.split || len(b) <= c.first {
		c.split = true
		return c.Conn.Write(b)
	}
	c.split = true
	if _, err := c.Conn.Write(b[:c.first]); err != nil {
		return 0, err
	}
	time.Sleep(2 * time.Millisecond)
	n, err := c.Conn.Write(b[c.first:])
	return c.first + n, err
}

// sinkConn — соединение-воронка: Write всегда принимает байты, чтения нет. Нужно
// для отказных веток снимщика, где соединение как таковое не проверяется.
type sinkConn struct{ net.Conn }

func (sinkConn) Write(b []byte) (int, error) { return len(b), nil }
func (sinkConn) Close() error                { return nil }

func TestRecorderCapturesOutgoingHello(t *testing.T) {
	addr, stop := loopbackServer(t)
	defer stop()

	hello, segs, raw := dialRecorded(t, addr, 0)

	if hello.ServerName != "example.test" {
		t.Fatalf("SNI снялся не тот: %q", hello.ServerName)
	}
	if len(hello.JA3()) != 32 || len(hello.JA4()) < 4 {
		t.Fatalf("печатки невозможной длины: ja3=%q ja4=%q", hello.JA3(), hello.JA4())
	}
	if len(raw) < 5 || raw[0] != recordTypeHandshake {
		t.Fatalf("снимок начинается не с handshake-записи: %d байт", len(raw))
	}
	sum := 0
	for _, s := range segs {
		sum += s
	}
	if sum != len(raw) {
		t.Fatalf("границы записей не сходятся с снимком: %d против %d", sum, len(raw))
	}
	// Второй разбор тех же байт обязан дать ту же печатку: снимщик ничего в поток
	// не додумывает.
	again, ok, err := Scan(raw)
	if err != nil || !ok || again.JA3() != hello.JA3() || again.JA4() != hello.JA4() {
		t.Fatalf("повторный разбор снимка разошёлся: ok=%v err=%v", ok, err)
	}
	if hello.Fragmented() {
		t.Fatalf("целая запись помечена фрагментированной: %v", hello.RecordVersions)
	}
}

func TestFragmentationInsideOneRecordKeepsFingerprint(t *testing.T) {
	addr, stop := loopbackServer(t)
	defer stop()

	// Один и тот же поток байтов, разложенный по-разному. Сравнивать два
	// разные рукопожатия байт в байт нельзя: в hello есть client_random,
	// session_id и key_share, и они различны по построению — поэтому сначала
	// инвариант проверяется на одном снятом потоке.
	_, _, raw := dialRecorded(t, addr, 0)

	whole := NewRecorder(sinkConn{})
	if _, err := whole.Write(raw); err != nil {
		t.Fatalf("целая запись не прошла: %v", err)
	}
	split := NewRecorder(sinkConn{})
	if _, err := split.Write(raw[:5]); err != nil {
		t.Fatalf("первый кусок не прошёл: %v", err)
	}
	if _, err := split.Write(raw[5:]); err != nil {
		t.Fatalf("второй кусок не прошёл: %v", err)
	}

	hWhole, err := whole.Hello()
	if err != nil {
		t.Fatalf("целой поток не разобрался: %v", err)
	}
	hSplit, err := split.Hello()
	if err != nil {
		t.Fatalf("разрезанный поток не разобрался: %v", err)
	}
	if hWhole.JA3() != hSplit.JA3() || hWhole.JA4() != hSplit.JA4() {
		t.Fatalf("разрез внутри записи сдвинул печатку: %s/%s против %s/%s",
			hWhole.JA3(), hWhole.JA4(), hSplit.JA3(), hSplit.JA4())
	}
	if hWhole.JA3Spec() != hSplit.JA3Spec() {
		t.Fatalf("разрез сдвинул спецификацию JA3")
	}
	if len(hWhole.RecordVersions) != len(hSplit.RecordVersions) {
		t.Fatalf("разрез по TCP-потоку отразился в версиях записей: %v против %v",
			hWhole.RecordVersions, hSplit.RecordVersions)
	}
	if got := whole.Segments(); len(got) != 1 || got[0] != len(raw) {
		t.Fatalf("целой поток записан не одной границей: %v", got)
	}
	if got := split.Segments(); len(got) != 2 || got[0] != 5 || got[1] != len(raw)-5 {
		t.Fatalf("разрез по 5 байт не виден в границах записей: %v", got)
	}

	// Живой прогон сквозь разрезатель: рукопожатие обязано состояться, а
	// печатка — совпасть с целой (тот же tls.Config, значит те же списки).
	wholeHello, _, _ := dialRecorded(t, addr, 0)
	splitHello, splitSegs, _ := dialRecorded(t, addr, 5)
	if wholeHello.JA3() != splitHello.JA3() || wholeHello.JA4() != splitHello.JA4() {
		t.Fatalf("живое рукопожатие через разрезатель дало другой отпечаток: %s/%s против %s/%s",
			wholeHello.JA3(), wholeHello.JA4(), splitHello.JA3(), splitHello.JA4())
	}
	if len(splitSegs) < 2 || splitSegs[0] != 5 {
		t.Fatalf("живой разрез не виден в границах: %v", splitSegs)
	}
}

func TestRecorderNamesIncompleteAndForeignStreams(t *testing.T) {
	t.Run("недописанный hello", func(t *testing.T) {
		rec := NewRecorder(sinkConn{})
		if _, err := rec.Write([]byte{recordTypeHandshake, 3, 3, 0, 200}); err != nil {
			t.Fatalf("запись не прошла: %v", err)
		}
		if _, err := rec.Hello(); err == nil || !strings.Contains(err.Error(), "нет целого") {
			t.Fatalf("недописанный hello обязан называться, а не молчать: %v", err)
		}
	})

	t.Run("первая запись не handshake", func(t *testing.T) {
		rec := NewRecorder(sinkConn{})
		if _, err := rec.Write([]byte{0x17, 3, 3, 0, 4, 1, 2, 3, 4}); err != nil {
			t.Fatalf("запись не прошла: %v", err)
		}
		if _, err := rec.Hello(); err == nil || !strings.Contains(err.Error(), "не handshake") {
			t.Fatalf("чужой тип первой записи назван не был: %v", err)
		}
	})

	t.Run("снимок упирается в потолок", func(t *testing.T) {
		rec := NewRecorder(sinkConn{})
		chunk := make([]byte, MaxHelloCapture/2)
		for i := range chunk {
			chunk[i] = byte(i % 251)
		}
		// Заголовок handshake с невозможной длиной: разбор не соберётся никогда, и
		// потолок обязан сработать до того, как буфер станет дампом соединения.
		chunk[0], chunk[1], chunk[2], chunk[3], chunk[4] = recordTypeHandshake, 3, 3, 0xff, 0xff
		for i := 0; i < 3; i++ {
			if _, err := rec.Write(chunk); err != nil {
				t.Fatalf("запись %d не прошла: %v", i, err)
			}
		}
		if _, err := rec.Hello(); err == nil || !strings.Contains(err.Error(), "потолок") {
			t.Fatalf("потолок снимка не назван: %v", err)
		}
		if len(rec.Captured()) > MaxHelloCapture {
			t.Fatalf("после отказа снимок продолжил расти: %d байт", len(rec.Captured()))
		}
		if got := len(rec.Segments()); got != 2 {
			t.Fatalf("после отказа границы продолжали писаться: %d записей", got)
		}
	})
}

func TestSNIByteOffsetRequiresUniqueHostname(t *testing.T) {
	addr, stop := loopbackServer(t)
	defer stop()
	_, _, raw := dialRecorded(t, addr, 0)

	off, err := SNIByteOffset(raw, "example.test")
	if err != nil {
		t.Fatalf("смещение SNI не нашлось: %v", err)
	}
	if got := string(raw[off : off+len("example.test")]); got != "example.test" {
		t.Fatalf("смещение указывает не на имя: %q", got)
	}
	// Структура по RFC 6066: байт типа host_name (0x00), затем двухбайтная длина
	// имени, затем само имя. Именно это SNIByteOffset требует, а не «любой байт
	// перед хостом».
	if raw[off-3] != 0x00 {
		t.Fatalf("перед именем нет байта типа host_name: 0x%02x", raw[off-3])
	}
	if got := int(uint16(raw[off-2])<<8 | uint16(raw[off-1])); got != len("example.test") {
		t.Fatalf("длина имени в структуре SNI не совпала: %d", got)
	}
	if off < 5 {
		t.Fatalf("смещение %d попало в заголовок записи", off)
	}

	// Имя, встречающееся дважды, — неоднозначный разрез, и он обязан быть ошибкой,
	// а не «режем по первому вхождению».
	dup := append(append([]byte{}, raw...), []byte("example.test")...)
	if _, err := SNIByteOffset(dup, "example.test"); err == nil || !strings.Contains(err.Error(), "дважды") {
		t.Fatalf("повтор имени не назван ошибкой: %v", err)
	}
	if _, err := SNIByteOffset(raw, "net.example"); err == nil || !strings.Contains(err.Error(), "не найдено") {
		t.Fatalf("отсутствующее имя принято молча: %v", err)
	}
	if _, err := SNIByteOffset(raw, ""); err == nil || !strings.Contains(err.Error(), "пустое") {
		t.Fatalf("пустое имя принято: %v", err)
	}
	// Имя без предшествующего 0x00 — совпадение байтов, а не SNI.
	if _, err := SNIByteOffset([]byte("example.test"), "example.test"); err == nil ||
		!strings.Contains(err.Error(), "не SNI") {
		t.Fatalf("совпадение без байта типа принято как SNI: %v", err)
	}
}

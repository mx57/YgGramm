package yggnode

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"ygggram/core/pkg/fingerprint"
)

// Замеры Шага 4-b: какой TLS ClientHello реально уходит в сеть, когда ссылка
// yggdrasil построена по tls://. Прокси ничего не портит — он только копирует
// байты и подсматривает в поток, поэтому разобранный hello это ровно то, что
// увидит наблюдатель на пути (и ровно то, по чему считаются JA3 и JA4).
//
// Числа (хэши обеих печаток) в проверки не кладутся намеренно: они зависят от
// версии Go, а CI и локальная машина используют разные. Проверяются только
// структурные инварианты и согласованность печатка с разобранными полями, а
// измеренное печатается в журнал и руками переносится в README/AGENTS.md.

// helloRecorder — TCP-зеркало: принимает соединение, соединяется с целевым
// узлом, качает трафик в обе стороны и разбирает первый client_hello.
type helloRecorder struct {
	ln     net.Listener
	target string

	mu      sync.Mutex
	buf     []byte
	hello   *fingerprint.ClientHello
	err     error
	closed  bool
	done    chan struct{}
	dialers int
}

func newHelloRecorder(t *testing.T, target string) *helloRecorder {
	t.Helper()
	// Пустой хост = wildcard: приём и по IPv4, и по IPv6, чтобы «localhost»
	// не зависел от того, какой семьи адрес резолвер поставил первым.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	r := &helloRecorder{
		ln:     ln,
		target: target,
		done:   make(chan struct{}),
	}
	go r.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return r
}

func (r *helloRecorder) port() int {
	return r.ln.Addr().(*net.TCPAddr).Port
}

func (r *helloRecorder) serve() {
	for {
		client, err := r.ln.Accept()
		if err != nil {
			return
		}
		go r.handle(client)
	}
}

func (r *helloRecorder) handle(client net.Conn) {
	up, err := net.Dial("tcp", r.target)
	if err != nil {
		r.fail(fmt.Errorf("proxy: uplink %s: %w", r.target, err))
		_ = client.Close()
		return
	}
	r.mu.Lock()
	r.dialers++
	r.mu.Unlock()

	go func() {
		_, _ = io.Copy(client, up)
		_ = up.Close()
		_ = client.Close()
	}()

	buf := make([]byte, 4096)
	for {
		n, err := client.Read(buf)
		if n > 0 {
			r.observe(buf[:n])
			if _, werr := up.Write(buf[:n]); werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	_ = client.Close()
	_ = up.Close()
}

func (r *helloRecorder) observe(in []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.buf = append(r.buf, in...)
	h, ok, err := fingerprint.Scan(r.buf)
	switch {
	case ok:
		r.hello = h
		r.closed = true
		close(r.done)
	case err != nil && !errors.Is(err, fingerprint.ErrShort):
		r.err = err
		r.closed = true
		close(r.done)
	}
}

func (r *helloRecorder) fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.err = err
	r.closed = true
	close(r.done)
}

// await ждёт разобранный hello: либо он приходит, либо прокси видит ошибку.
func (r *helloRecorder) await(t *testing.T, limit time.Duration) *fingerprint.ClientHello {
	t.Helper()
	select {
	case <-r.done:
	case <-time.After(limit):
		t.Fatal("client_hello так и не прошёл через прокси")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		t.Fatalf("proxy: %v", r.err)
	}
	if r.hello == nil {
		t.Fatal("hello не разобран")
	}
	return r.hello
}

// captureTLSHello поднимает узел B за прокси и узел A, который спирится к нему
// по tls:// через это прокси, и возвращает байтовый слепок hello. Пир задаётся
// именем или адресом — от этого зависит, будет ли SNI.
func captureTLSHello(t *testing.T, peerHost string) *fingerprint.ClientHello {
	t.Helper()

	targetPort := freeTCPPort(t)
	b, err := NewYggNode(NodeOptions{
		PrivateKey: nodeKey(t),
		Listen:     []string{fmt.Sprintf("tls://:%d", targetPort)},
	})
	if err != nil {
		t.Fatalf("сборка узла B: %v", err)
	}
	if err := b.Start(); err != nil {
		t.Fatalf("старт B: %v", err)
	}
	t.Cleanup(func() { _ = b.Stop() })

	rec := newHelloRecorder(t, fmt.Sprintf("127.0.0.1:%d", targetPort))

	a, err := NewYggNode(NodeOptions{
		PrivateKey: nodeKey(t),
		Peers:      []string{fmt.Sprintf("tls://%s:%d", peerHost, rec.port())},
	})
	if err != nil {
		t.Fatalf("сборка узла A: %v", err)
	}
	if err := a.Start(); err != nil {
		t.Fatalf("старт A: %v", err)
	}
	t.Cleanup(func() { _ = a.Stop() })

	h := rec.await(t, 30*time.Second)

	// Замер имеет смысл только если ссылка живая: иначе наблюдаем пробник, а
	// не рабочий транспорт.
	if !waitFor(30*time.Second, func() bool {
		return b.HasRouteTo(a.PublicKey()) && a.HasRouteTo(b.PublicKey())
	}) {
		t.Fatalf("рукопожатие прошло, но маршрут не построился: peers A=%v B=%v",
			a.Peers(), b.Peers())
	}

	t.Logf("Go: %s", runtime.Version())
	t.Logf("JA3 spec: %s", h.JA3Spec())
	t.Logf("JA3 hash: %s", h.JA3())
	t.Logf("JA4:      %s", h.JA4())
	t.Logf("JA4_r:    %s", h.JA4Raw())
	t.Logf("summary:  %s", h.Summary())
	return h
}

func hasU16(list []uint16, v uint16) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// TestTLSLinkHelloByIPAddressHasNoSNI — пир задан адресом: Go обязан не класть
// server_name (RFC 6066 запрещает IP в SNI), и для DPI это значит, что на
// tls://-ссылке с адресом имени узла не видно.
func TestTLSLinkHelloByIPAddressHasNoSNI(t *testing.T) {
	if testing.Short() {
		t.Skip("замер требует реального рукопожатия")
	}
	h := captureTLSHello(t, "127.0.0.1")

	if h.HasSNI() {
		t.Fatalf("SNI появился при адресе пира: %q", h.ServerName)
	}
	if h.HasExtension(0) {
		t.Fatal("расширение server_name присутствует, хотя имени нет")
	}
	checkHelloInvariants(t, h)
}

// TestTLSLinkHelloByHostnameSendsSNI — тот же транспорт, но пир задан именем:
// имя уходит открытым текстом. Это ровно то, что надо знать перед тем, как
// советовать «пишите пирам имена».
func TestTLSLinkHelloByHostnameSendsSNI(t *testing.T) {
	if testing.Short() {
		t.Skip("замер требует реального рукопожатия")
	}
	h := captureTLSHello(t, "localhost")

	if !h.HasSNI() {
		t.Fatal("SNI нет, хотя пир задан именем")
	}
	if !strings.EqualFold(h.ServerName, "localhost") {
		t.Fatalf("SNI = %q, ожидалось localhost", h.ServerName)
	}
	checkHelloInvariants(t, h)
}

// TestTLSLinkAddressAndNameShareExtensionHash — два живых замера одной и той же
// сборки: пир адресом и пир именем. По JA4 у них обязаны совпасть оба хэша и
// раздвинуться только приставки (d/i). Это и есть ответ на вопрос «видно ли имя
// пира наблюдателю»: имя видно в приставке и в открытом SNI, а весь остальной
// печаток от способа задания пира не зависит — в отличие от JA3, где хэш от
// появления server_name уезжает целиком.
func TestTLSLinkAddressAndNameShareExtensionHash(t *testing.T) {
	if testing.Short() {
		t.Skip("замер требует реального рукопожатия")
	}
	byAddr := captureTLSHello(t, "127.0.0.1")
	byName := captureTLSHello(t, "localhost")

	addrChunks := strings.Split(byAddr.JA4(), "_")
	nameChunks := strings.Split(byName.JA4(), "_")
	if len(addrChunks) != 3 || len(nameChunks) != 3 {
		t.Fatalf("печаток разобрался неверно: %q и %q", byAddr.JA4(), byName.JA4())
	}
	for i, label := range []string{"_s", "_c"} {
		if addrChunks[i+1] != nameChunks[i+1] {
			t.Errorf("хэш %s поехал при смене способа задания пира: %s против %s",
				label, addrChunks[i+1], nameChunks[i+1])
		}
	}
	if addrChunks[0] == nameChunks[0] {
		t.Errorf("приставки совпали (%s), хотя SNI в этих замерах разный", addrChunks[0])
	}
	if byAddr.JA3() == byName.JA3() {
		t.Errorf("JA3 совпали (%s), хотя в одном hello есть server_name, а в другом нет", byAddr.JA3())
	}
	t.Logf("JA4 адресная ссылка: %s", byAddr.JA4())
	t.Logf("JA4 именная ссылка:  %s", byName.JA4())
}

// checkHelloInvariants — то, что не зависит от версии Go и от транзитных
// рандомизаций: порядок полей hello разобран, транспорт TLS 1.3, GREASE нет,
// ALPN нет, X25519 заявлен.
func checkHelloInvariants(t *testing.T, h *fingerprint.ClientHello) {
	t.Helper()

	if h.LegacyVersion != 0x0303 {
		t.Errorf("legacy_version = %d, ожидалось 771", h.LegacyVersion)
	}
	if !hasU16(h.SupportedVersions, 0x0304) {
		t.Errorf("TLS 1.3 не предложен: supported_versions=%v", h.SupportedVersions)
	}
	for _, v := range h.SupportedVersions {
		if v > 0x0304 {
			t.Errorf("версия %d не может быть в supported_versions клиента", v)
		}
	}
	if !hasU16(h.CipherSuites, 0x1301) {
		t.Errorf("шифр TLS_AES_128_GCM_SHA256 отсутствует: %v", h.CipherSuites)
	}
	if !hasU16(h.SupportedGroups, 29) {
		t.Errorf("X25519 не заявлен в supported_groups: %v", h.SupportedGroups)
	}
	if len(h.KeyShareGroups) == 0 {
		t.Error("key_share пуст — рукопожатие 1.3 не могло бы состояться")
	} else if !hasU16(h.KeyShareGroups, 29) && !hasU16(h.KeyShareGroups, 0x11ec) {
		t.Errorf("в key_share нет ни X25519, ни гибридной группы: %v", h.KeyShareGroups)
	}
	if h.HasALPN() {
		t.Errorf("ALPN заявлен (%v), хотя yggdrasil его не ставит", h.Protocols)
	}
	if h.HasGrease() {
		t.Error("GREASE в Go-клиенте быть не может — проверка отпечатка потеряла смысл")
	}
	if len(h.Compression) != 1 || h.Compression[0] != 0 {
		t.Errorf("методы сжатия: %v, ожидался только null", h.Compression)
	}
	if len(h.Extensions) == 0 || len(h.CipherSuites) == 0 {
		t.Error("пустые списки: hello разобран неверно")
	}
	if len(h.RecordVersions) == 0 {
		t.Error("версии TLS-записей не записаны")
	}
	if n := strings.Count(h.JA3Spec(), ","); n != 4 {
		t.Errorf("в описании JA3 %d полей, ожидалось 5: %q", n+1, h.JA3Spec())
	}
	if hash := h.JA3(); len(hash) != 32 || strings.ContainsAny(hash, "ghijklmnopqrstuvwxyz") {
		t.Errorf("JA3-хэш не похож на hex MD5: %q", hash)
	}
	checkJA4Invariants(t, h)
}

// checkJA4Invariants — форма JA4-печатка на живом рукопожатии. Значения хэшей тут
// не сравниваются с фиксированными по той же причине, что и у JA3: они зависят от
// версии Go. Проверяется то, что от версии не зависит: протокол «t» (Scan видит
// только TCP-записи), маркер SNI как факт разбора, отсутствие ALPN, длины списков
// из приставки, hex-форма хэшей и их независимый пересчёт из разобранных полей.
// Отдельный тест сравнивает адресную и именную ссылки между собой.
func checkJA4Invariants(t *testing.T, h *fingerprint.ClientHello) {
	t.Helper()

	ja4 := h.JA4()
	chunks := strings.Split(ja4, "_")
	if len(chunks) != 3 {
		t.Fatalf("JA4 = %q, ожидалось три куска через подчёркивание", ja4)
	}
	prefix, ciphHash, extHash := chunks[0], chunks[1], chunks[2]
	if len(prefix) != 10 {
		t.Fatalf("приставка JA4 = %q, по спецификации 10 символов", prefix)
	}
	if prefix[0] != 't' {
		t.Errorf("первый символ JA4 = %q, ожидался «t» (разбирается только TCP)", prefix[0])
	}
	if !knownJA4Version(prefix[1:3]) {
		t.Errorf("версия в JA4 = %q, в таблице спецификации такого нет", prefix[1:3])
	}
	// Маркер SNI сверяется с другим разбором того же hello: приставка строится по
	// наличию расширения, а HasSNI — по извлечённому имени. Это разные пути, и
	// на живом трафике они обязаны совпадать.
	wantSNI := byte('i')
	if h.HasSNI() {
		wantSNI = 'd'
	}
	if prefix[3] != wantSNI {
		t.Errorf("маркер SNI = %q, по расширению ожидался %q", prefix[3], wantSNI)
	}
	if got, want := prefix[4:6], fmt.Sprintf("%02d", len(h.CipherSuites)); got != want {
		t.Errorf("число шифров в JA4 = %s, в hello %s", got, want)
	}
	if got, want := prefix[6:8], fmt.Sprintf("%02d", len(h.Extensions)); got != want {
		t.Errorf("число расширений в JA4 = %s, в hello %s", got, want)
	}
	if prefix[8:10] != "00" {
		t.Errorf("маркер ALPN = %q, ожидалось 00 (yggdrasil ALPN не заявляет)", prefix[8:10])
	}
	for name, hash := range map[string]string{"_s": ciphHash, "_c": extHash} {
		if len(hash) != 12 {
			t.Errorf("хэш %s JA4 = %q, ожидалось 12 hex-символов", name, hash)
			continue
		}
		if strings.ContainsAny(hash, "ghijklmnopqrstuvwxyz") {
			t.Errorf("хэш %s JA4 не похож на hex: %q", name, hash)
		}
		if hash == "000000000000" {
			t.Errorf("хэш %s = 000000000000, хотя списки непустые: список пустой или считать нечего", name)
		}
	}

	// Смысловая проверка: хэш расширений считается по списку без SNI и ALPN, а
	// не по сырым байтам hello. Сверяем _c с hex-строкой, собранной здесь же из
	// разобранных полей — если парсер или порядок сортировки поедут, это видно.
	var wantExts []string
	for _, e := range h.Extensions {
		if e == ja4ExtSNI || e == ja4ExtALPN {
			continue
		}
		wantExts = append(wantExts, fmt.Sprintf("%04x", e))
	}
	sort.Strings(wantExts)
	wantC := strings.Join(wantExts, ",")
	if len(h.SignatureAlgs) > 0 {
		sigs := make([]string, len(h.SignatureAlgs))
		for i, s := range h.SignatureAlgs {
			sigs[i] = fmt.Sprintf("%04x", s)
		}
		wantC += "_" + strings.Join(sigs, ",")
	}
	if got, want := ja4Hash(wantC), extHash; got != want {
		t.Errorf("хэш расширений JA4 не совпадает с независимым пересчётом: %s против %s", got, want)
	}
	var wantCiph []string
	for _, c := range h.CipherSuites {
		wantCiph = append(wantCiph, fmt.Sprintf("%04x", c))
	}
	sort.Strings(wantCiph)
	if got, want := ja4Hash(strings.Join(wantCiph, ",")), ciphHash; got != want {
		t.Errorf("хэш шифров JA4 не совпадает с независимым пересчётом: %s против %s", got, want)
	}

	// JA4_r длиннее самого печатка: в нём хэши заменены списками, и подчёркиваний
	// три (приставка_шифры_расширения_алгоритмы), а не два.
	if raw := h.JA4Raw(); strings.Count(raw, "_") != 3 {
		t.Errorf("JA4_r = %q, в нём должно быть ровно три подчёркивания", raw)
	}
}

// knownJA4Version — две цифры версии из таблицы спецификации.
func knownJA4Version(code string) bool {
	switch code {
	case "13", "12", "11", "10", "s3", "s2", "d1", "d2", "d3", "00":
		return true
	}
	return false
}

// Коды расширений, по которым JA4 строит приставку. В pkg/fingerprint они не
// экспортируются (разбор их прячет), а здесь нужен именно голый код.
const (
	ja4ExtSNI  uint16 = 0x0000
	ja4ExtALPN uint16 = 0x0010
)

// ja4Hash — вторая, независимая от pkg/fingerprint сборка того же усечённого
// sha256: первые 6 байт в hex, пустая строка показывается как двенадцать нулей.
func ja4Hash(s string) string {
	if s == "" {
		return "000000000000"
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

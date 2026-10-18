// File: core/pkg/fingerprint/hello_test.go
package fingerprint

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
	"testing"
)

// construction-помощники: hello собирается байт в байт, чтобы ожидание в тесте
// было видно буквально, а не вычислялось тем же кодом, который проверяется.

func u16(v uint16) []byte {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], v)
	return b[:]
}

func u16List(vals ...uint16) []byte {
	out := u16(uint16(len(vals) * 2))
	for _, v := range vals {
		out = append(out, u16(v)...)
	}
	return out
}

func ext(id uint16, payload []byte) []byte {
	return append(append(u16(id), u16(uint16(len(payload)))...), payload...)
}

type helloOpts struct {
	legacy     uint16
	extensions [][]byte
	// recordSplit — режет рукопожатие на N TLS-записей (эмуляция фрагментации).
	recordSplit int
	recordVer   uint16
}

// buildHello собирает поток TCP-байтов с одним client_hello.
func buildHello(t *testing.T, o helloOpts) []byte {
	t.Helper()
	if o.legacy == 0 {
		o.legacy = 0x0303
	}
	if o.recordVer == 0 {
		o.recordVer = 0x0301
	}
	var exts []byte
	for _, e := range o.extensions {
		exts = append(exts, e...)
	}
	body := u16(o.legacy)
	body = append(body, make([]byte, 32)...) // random
	body = append(body, 0)                   // session id: пусто
	body = append(body, u16List(4865, 4866, 4867, 49196)...)
	body = append(body, 1, 0) // compression: null
	body = append(body, u16(uint16(len(exts)))...)
	body = append(body, exts...)

	msg := append([]byte{typeClientHello}, byte(len(body)>>16), byte(len(body)>>8), byte(len(body)))
	msg = append(msg, body...)

	split := o.recordSplit
	if split < 1 {
		split = 1
	}
	var out []byte
	chunk := (len(msg) + split - 1) / split
	for off := 0; off < len(msg); off += chunk {
		end := off + chunk
		if end > len(msg) {
			end = len(msg)
		}
		rec := []byte{recordTypeHandshake, byte(o.recordVer >> 8), byte(o.recordVer)}
		rec = append(rec, u16(uint16(end-off))...)
		out = append(out, rec...)
		out = append(out, msg[off:end]...)
	}
	return out
}

// svBlock — supported_versions с длиной списка в 1 байт (единственное
// разбираемое расширение именно с таким префиксом).
func svBlock(versions ...uint16) []byte {
	body := make([]byte, 0, len(versions)*2)
	for _, v := range versions {
		body = append(body, u16(v)...)
	}
	return append([]byte{byte(len(body))}, body...)
}

func standardExts() [][]byte {
	return [][]byte{
		ext(extSupportedVersions, svBlock(0x0304, 0x0303)),
		ext(extSupportedGroups, u16List(29, 23)),
		ext(extPointFormats, []byte{1, 0}),
		ext(extSignatureAlgs, u16List(0x0403, 0x0503)),
	}
}

// alpnBlock — [длина списка][длина строки]['h','2']: ALPN собирается вручную,
// потому что это список строк, а не пар чисел.
func alpnBlock(proto string) []byte {
	one := append([]byte{byte(len(proto))}, []byte(proto)...)
	return append(u16(uint16(len(one))), one...)
}

// TestParseClientHelloFields — разобранные поля обязаны совпадать с тем, что
// реально лежит в байтах, включая порядок расширений.
func TestParseClientHelloFields(t *testing.T) {
	stream := buildHello(t, helloOpts{extensions: append(standardExts(),
		ext(extALPN, alpnBlock("h2")),
	)})
	h, ok, err := Scan(stream)
	if err != nil || !ok {
		t.Fatalf("Scan: ok=%v err=%v", ok, err)
	}
	if h.LegacyVersion != 0x0303 {
		t.Fatalf("legacy version: %d", h.LegacyVersion)
	}
	if got := strings.Join(mapList(h.CipherSuites), "-"); got != "4865-4866-4867-49196" {
		t.Fatalf("шифры: %s", got)
	}
	if got := strings.Join(mapList(h.Extensions), "-"); got != "43-10-11-13-16" {
		t.Fatalf("порядок расширений потерялся: %s", got)
	}
	if got := strings.Join(mapList(h.SupportedGroups), "-"); got != "29-23" {
		t.Fatalf("группы: %s", got)
	}
	if got := strings.Join(mapBytes(h.PointFormats), "-"); got != "0" {
		t.Fatalf("форматы точек: %s", got)
	}
	if got := strings.Join(mapList(h.SupportedVersions), "-"); got != "772-771" {
		t.Fatalf("supported versions: %s", got)
	}
	if len(h.SignatureAlgs) != 2 || h.SignatureAlgs[0] != 0x0403 {
		t.Fatalf("signature algorithms: %v", h.SignatureAlgs)
	}
	if strings.Join(h.Protocols, ",") != "h2" {
		t.Fatalf("ALPN: %q", h.Protocols)
	}
	if h.HasSNI() {
		t.Fatal("SNI появился там, где его не было")
	}
	if len(h.RecordVersions) != 1 || h.RecordVersions[0] != 0x0301 {
		t.Fatalf("версии записей: %v", h.RecordVersions)
	}
}

func mapList(in []uint16) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strconv.Itoa(int(v))
	}
	return out
}

func mapBytes(in []byte) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strconv.Itoa(int(v))
	}
	return out
}

// TestJA3SpecIsLiteral — описание печатка пишется буквально: версия из тела
// hello, списки в порядке пакета, пустое поле форматов не съедает запятую.
func TestJA3SpecIsLiteral(t *testing.T) {
	h, ok, err := Scan(buildHello(t, helloOpts{extensions: standardExts()}))
	if err != nil || !ok {
		t.Fatal(err)
	}
	want := "771,4865-4866-4867-49196,43-10-11-13,29-23,0"
	if got := h.JA3Spec(); got != want {
		t.Fatalf("JA3 spec:\n получено %s\n ожидалось %s", got, want)
	}
	if got := h.JA3(); len(got) != 32 || strings.ContainsAny(got, "ghijklmnopqrstuvwxyz") {
		t.Fatalf("JA3 hash не похож на hex MD5: %q", got)
	}

	// Тот же набор расширений в другом порядке — уже другой печаток. Сортировка
	// здесь была бы грубой ошибкой: DPI смотрит на порядок.
	shuffled := buildHello(t, helloOpts{extensions: [][]byte{
		standardExts()[2], standardExts()[0], standardExts()[3], standardExts()[1],
	}})
	h2, ok, err := Scan(shuffled)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if want2 := "771,4865-4866-4867-49196,11-43-13-10,29-23,0"; h2.JA3Spec() != want2 {
		t.Fatalf("порядок расширений в JA3 потерялся: %s", h2.JA3Spec())
	}
	if h2.JA3() == h.JA3() {
		t.Fatal("разный порядок расширений дал тот же JA3")
	}

	// hello без расширения форматов точек: поле пустое, запятая остаётся.
	nopf, ok, err := Scan(buildHello(t, helloOpts{extensions: [][]byte{
		ext(extSupportedVersions, svBlock(0x0304)),
		ext(extSupportedGroups, u16List(29)),
	}}))
	if err != nil || !ok {
		t.Fatal(err)
	}
	if got := nopf.JA3Spec(); got != "771,4865-4866-4867-49196,43-10,29," {
		t.Fatalf("пустые форматы точек: %q", got)
	}
}

// TestParseFragmentedHello — рукопожатие, разложенное по нескольким
// TLS-записям, читается как одно, а количество записей видно отдельно.
func TestParseFragmentedHello(t *testing.T) {
	stream := buildHello(t, helloOpts{extensions: standardExts(), recordSplit: 3})
	h, ok, err := Scan(stream)
	if err != nil || !ok {
		t.Fatalf("фрагментированный hello не разобран: ok=%v err=%v", ok, err)
	}
	if len(h.RecordVersions) != 1 {
		t.Fatalf("версии записей должны схлопнуться в одну: %v", h.RecordVersions)
	}
	if strings.Join(mapList(h.Extensions), "-") != "43-10-11-13" {
		t.Fatalf("поля после склейки записей разъехались: %v", h.Extensions)
	}
}

// TestScanRejectsJunk — битый поток обязан отваливаться ошибкой разбора, а не
// молча возвращать «недочитано»: молчание здесь означало бы замер, который
// никогда не завершится.
func TestScanRejectsJunk(t *testing.T) {
	whole := buildHello(t, helloOpts{extensions: standardExts()})

	// Оборванное расширение: заголовок обещает 10 байт, в записи их нет.
	badExt := buildHello(t, helloOpts{extensions: [][]byte{
		ext(extSupportedVersions, svBlock(0x0304)),
		append(append(u16(extALPN), u16(10)...), 1, 2),
	}})
	// Длина рукопожатия больше, чем лежит в TLS-записи: hello никогда не соберётся.
	badMsgLen := func() []byte {
		b := buildHello(t, helloOpts{extensions: standardExts()})
		b[6] = 0x7f
		return b
	}()

	cases := []struct {
		name string
		in   []byte
	}{
		{"не handshake", append([]byte{0x17, 0x03, 0x03, 0x00, 0x04}, whole[5:9]...)},
		{"другой тип рукопожатия", func() []byte {
			b := buildHello(t, helloOpts{extensions: standardExts()})
			b[5] = 0x02 // server_hello
			return b
		}()},
		{"оборванное расширение", badExt},
		{"невозможная версия записи", buildHello(t, helloOpts{extensions: standardExts(), recordVer: 0x0309})},
		{"рукопожатие длиннее записи", badMsgLen},
	}
	for _, tc := range cases {
		h, ok, err := Scan(tc.in)
		if ok || err == nil || errors.Is(err, ErrShort) || h != nil {
			t.Errorf("%s: ok=%v err=%v, ожидалась ошибка разбора", tc.name, ok, err)
		}
	}

	// Обрезка посередине — это «дочитай», а не битый поток.
	for _, cut := range []int{0, 2, 5, len(whole) / 2, len(whole) - 1} {
		h, ok, err := Scan(whole[:cut])
		if h != nil || ok || !errors.Is(err, ErrShort) {
			t.Errorf("обрезок %d байт: ok=%v err=%v, ожидался ErrShort", cut, ok, err)
		}
	}
}

// TestGreaseAndSNI — GREASE и SNI: два признака, по которым живой браузер
// отличается от Go-клиента.
func TestGreaseAndSNI(t *testing.T) {
	var name []byte
	name = append(name, u16(uint16(len("example.com")+3))...)
	name = append(name, 0)
	name = append(name, u16(uint16(len("example.com")))...)
	name = append(name, []byte("example.com")...)

	withSNI, ok, err := Scan(buildHello(t, helloOpts{extensions: append(standardExts(), ext(extServerName, name))}))
	if err != nil || !ok {
		t.Fatal(err)
	}
	if !withSNI.HasSNI() || withSNI.ServerName != "example.com" {
		t.Fatalf("SNI: %q", withSNI.ServerName)
	}
	if withSNI.HasGrease() {
		t.Fatal("GREASE найден там, где его нет")
	}

	greased := buildHello(t, helloOpts{extensions: [][]byte{
		ext(extSupportedVersions, svBlock(0x0a0a, 0x0304)),
		ext(extSupportedGroups, u16List(29)),
	}})
	h, ok, err := Scan(greased)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if !h.HasGrease() {
		t.Fatal("GREASE в supported_versions не замечен")
	}

	// Пустое имя сервера SNI не означает.
	empty := ext(extServerName, u16(0))
	h2, ok, err := Scan(buildHello(t, helloOpts{extensions: [][]byte{empty, ext(extSupportedGroups, u16List(29))}}))
	if err != nil || !ok {
		t.Fatal(err)
	}
	if h2.HasSNI() {
		t.Fatal("пустой server_name засчитан как SNI")
	}
	if !h2.HasExtension(extServerName) {
		t.Fatal("расширение server_name потерялось из списка")
	}
}

// TestKeyShareGroups — key_share устроен сложнее плоских списков, группы из
// него обязаны доставаться по одной.
func TestKeyShareGroups(t *testing.T) {
	var payload []byte
	var inner []byte
	for _, g := range []uint16{29, 23, 30} {
		inner = append(inner, u16(g)...)
		inner = append(inner, u16(32)...)
		inner = append(inner, make([]byte, 32)...)
	}
	payload = append(payload, u16(uint16(len(inner)))...)
	payload = append(payload, inner...)

	h, ok, err := Scan(buildHello(t, helloOpts{extensions: [][]byte{ext(extKeyShare, payload), ext(extSupportedGroups, u16List(29, 23, 30))}}))
	if err != nil || !ok {
		t.Fatal(err)
	}
	if strings.Join(mapList(h.KeyShareGroups), "-") != "29-23-30" {
		t.Fatalf("группы key_share: %v", h.KeyShareGroups)
	}
	if strings.Join(mapList(h.SupportedGroups), "-") != "29-23-30" {
		t.Fatalf("supported groups попали в key_share: %v", h.SupportedGroups)
	}
}

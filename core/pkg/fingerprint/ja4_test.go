// File: core/pkg/fingerprint/ja4_test.go
package fingerprint

import (
	"encoding/hex"
	"strings"
	"testing"
)

// Тесты собираются вокруг единственного внешнего источника: пример из
// technical_details/JA4.md репозитория FoxIO-LLC/ja4 — там же, где и сама
// спецификация. Ожидаемые хэши (8daaf6152771, e5627efa2ab1, 6d807ffa2a79) взяты
// из документа, а не из кода, так что падение означает «мы считаем не по JA4».

func specExampleHello() *ClientHello {
	return &ClientHello{
		RecordVersions:  []uint16{0x0301},
		LegacyVersion:   0x0303,
		CipherSuites:    parseHexCodes("1301,1302,1303,c02b,c02f,c02c,c030,cca9,cca8,c013,c014,009c,009d,002f,0035"),
		Extensions:      parseHexCodes("001b,0000,0033,0010,4469,0017,002d,000d,0005,0023,0012,002b,ff01,000b,000a,0015"),
		SupportedGroups: parseHexCodes("001d,0017,0018,0019"),
		PointFormats:    []byte{0x00},
		SignatureAlgs:   parseHexCodes("0403,0804,0401,0503,0805,0501,0806,0601"),
		SupportedVersions: []uint16{
			0x0304, 0x0303,
		},
		Protocols:  []string{"h2", "http/1.1"},
		ServerName: "example.com",
	}
}

func parseHexCodes(list string) []uint16 {
	parts := strings.Split(list, ",")
	out := make([]uint16, 0, len(parts))
	for _, p := range parts {
		b, err := hex.DecodeString(p)
		if err != nil || len(b) != 2 {
			panic("тестовый список кодов испорчен: " + p)
		}
		out = append(out, uint16(b[0])<<8|uint16(b[1]))
	}
	return out
}

// TestJA4OfficialExample — печаток из примера спецификации, целиком: и приставка,
// и оба хэша, и raw-строка до хэширования.
func TestJA4OfficialExample(t *testing.T) {
	h := specExampleHello()

	wantRaw := "t13d1516h2_" +
		"002f,0035,009c,009d,1301,1302,1303,c013,c014,c02b,c02c,c02f,c030,cca8,cca9_" +
		"0005,000a,000b,000d,0012,0015,0017,001b,0023,002b,002d,0033,4469,ff01" +
		"_0403,0804,0401,0503,0805,0501,0806,0601"
	if got := h.JA4Raw(); got != wantRaw {
		t.Fatalf("JA4_r расходится с примером:\nесть   %s\nдолжен %s", got, wantRaw)
	}

	if got, want := h.JA4(), "t13d1516h2_8daaf6152771_e5627efa2ab1"; got != want {
		t.Fatalf("JA4 = %s, по спецификации %s", got, want)
	}
}

// TestJA4ExtensionHashWithoutSignatureAlgs — вторая половинка примера: без
// signature_algorithms строка заканчивается без подчёркивания и хэшируется иначе
// (6d807ffa2a79 — тоже из документа).
func TestJA4ExtensionHashWithoutSignatureAlgs(t *testing.T) {
	h := specExampleHello()
	h.SignatureAlgs = nil

	if got, want := h.JA4(), "t13d1516h2_8daaf6152771_6d807ffa2a79"; got != want {
		t.Fatalf("JA4 без signature_algorithms = %s, ожидалось %s", got, want)
	}
	if raw := h.JA4Raw(); strings.HasSuffix(raw, "_") {
		t.Fatalf("в raw-строке осталось висячее подчёркивание: %s", raw)
	}
}

// TestJA4IgnoresGrease — GREASE не считается в длинах, не попадает в списки и не
// двигает версию: без этого печаток браузера был бы неустойчив, а наш Go-клиент
// GREASE не вставляет вовсе, и проверка держит именно правило разбора.
func TestJA4IgnoresGrease(t *testing.T) {
	clean := specExampleHello()
	dirty := specExampleHello()
	dirty.CipherSuites = append([]uint16{0x0a0a, 0xfafa}, dirty.CipherSuites...)
	dirty.Extensions = append([]uint16{0x1a1a}, dirty.Extensions...)
	dirty.SignatureAlgs = append([]uint16{0x3a3a}, dirty.SignatureAlgs...)
	dirty.SupportedVersions = append([]uint16{0x2a2a}, dirty.SupportedVersions...)

	if clean.JA4() != dirty.JA4() {
		t.Fatalf("GREASE сдвинул печаток: %s против %s", clean.JA4(), dirty.JA4())
	}
	// 15 шифров и 16 расширений остались как в примере, несмотря на добавленные значения.
	if !strings.HasPrefix(dirty.JA4(), "t13d1516") {
		t.Fatalf("после GREASE приставка поехала: %s", dirty.JA4())
	}
}

// TestJA4NoSNIKeepsExtensionHash — смысл выброса SNI и ALPN из хэша расширений:
// переход на домен и на адрес дают одинаковый _c.
func TestJA4NoSNIKeepsExtensionHash(t *testing.T) {
	withName := specExampleHello()

	noName := specExampleHello()
	var exts []uint16
	for _, e := range noName.Extensions {
		if e == extServerName || e == extALPN {
			continue
		}
		exts = append(exts, e)
	}
	noName.Extensions = exts
	noName.ServerName = ""
	noName.Protocols = nil

	withNameC := strings.Split(withName.JA4(), "_")[2]
	noNameC := strings.Split(noName.JA4(), "_")[2]
	if withNameC != noNameC {
		t.Fatalf("хэш расширений зависит от SNI/ALPN: %s против %s", withNameC, noNameC)
	}
	// А приставка обязана различать: d/i и наличие ALPN. Приставка — 10 символов
	// (t + версия + SNI + две цифры шифров + две цифры расширений + ALPN), поэтому
	// берём её из Split, а не срезом по пальцу.
	withPrefix := strings.Split(withName.JA4(), "_")[0]
	noPrefix := strings.Split(noName.JA4(), "_")[0]
	if !strings.HasPrefix(withPrefix, "t13d") || !strings.HasPrefix(noPrefix, "t13i") {
		t.Fatalf("маркер SNI сломан: %s и %s", withPrefix, noPrefix)
	}
	if !strings.HasSuffix(withPrefix, "h2") || !strings.HasSuffix(noPrefix, "00") {
		t.Fatalf("маркер ALPN сломан: %q против %q", withPrefix, noPrefix)
	}
}

// TestJA4EmptyLists — пустые списы заменяются на двенадцать нулей, а не на хэш от
// пустой строки (спецификация: иначе наблюдатель не отличит «полей не было»).
func TestJA4EmptyLists(t *testing.T) {
	h := &ClientHello{LegacyVersion: 0x0303}
	if got := h.JA4(); got != "t12i000000_000000000000_000000000000" {
		t.Fatalf("пустой hello печатается как %s", got)
	}
	if got := hash12(""); got != "000000000000" {
		t.Fatalf("hash12 от пустой строки = %s", got)
	}
	if got := hash12("551d0f,551d25,551d11"); got != "aae71e8db6d7" {
		t.Fatalf("hash12 расходится с эталонной реализацией: %s", got)
	}
}

// TestJA4ALPNMarker — восемь примеров из спецификации на разбор первого ALPN.
func TestJA4ALPNMarker(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{"h2", "h2"},
		{"http/1.1", "h1"},
		{"h", "hh"},
		{"", "00"},
		{string([]byte{0xAB}), "ab"},
		{string([]byte{0x20}), "20"},
		{string([]byte{0xAB, 0xCD}), "ad"},
		{string([]byte{0x20, 0x61}), "21"},
		{string([]byte{0x30, 0xAB}), "3b"},
		{string([]byte{0x61, 0x20}), "60"},
		{string([]byte{0x30, 0x31, 0xAB, 0xCD}), "3d"},
		{string([]byte{0x30, 0xAB, 0xCD, 0x31}), "01"},
	}
	for _, c := range cases {
		if got := alpnMarker(c.value); got != c.want {
			t.Errorf("ALPN %x: получили %q, по спецификации %q", []byte(c.value), got, c.want)
		}
	}

	h := &ClientHello{Protocols: []string{"h2", "http/1.1"}}
	if got := h.ja4ALPNMarker(); got != "h2" {
		t.Fatalf("маркер берётся не из первого значения: %s", got)
	}
	h.Protocols = []string{""}
	if got := h.ja4ALPNMarker(); got != "00" {
		t.Fatalf("пустое первое значение должно печататься как 00, получили %s", got)
	}
}

// TestJA4VersionCode — таблица версий слово в слово из спецификации. Черновые коды
// TLS 1.3 (0x1301…0x1303) в таблице отсутствуют, и эталонная реализация даёт для
// них «00»; закрепляем это как факт, а не чиним молча.
func TestJA4VersionCode(t *testing.T) {
	cases := []struct {
		value uint16
		want  string
	}{
		{0x0304, "13"},
		{0x0303, "12"},
		{0x0302, "11"},
		{0x0301, "10"},
		{0x0300, "s3"},
		{0x0002, "s2"},
		{0xfeff, "d1"},
		{0xfefd, "d2"},
		{0xfefc, "d3"},
		{0x1301, "00"},
		{0x0000, "00"},
	}
	for _, c := range cases {
		if got := tlsVersionCode(c.value); got != c.want {
			t.Errorf("версия 0x%04x: %s, ожидалось %s", c.value, got, c.want)
		}
	}

	// supported_versions важнее legacy_version, и выбирается старшее значение.
	h := specExampleHello()
	h.LegacyVersion = 0x0301
	h.SupportedVersions = []uint16{0x0303, 0x0304}
	if got := h.versionForJA4(); got != "13" {
		t.Fatalf("версия взята не из supported_versions: %s", got)
	}
	h.SupportedVersions = nil
	h.Extensions = withoutCodes(h.Extensions, extSupportedVersions)
	if got := h.versionForJA4(); got != "10" {
		t.Fatalf("без supported_versions версия берётся не из тела hello: %s", got)
	}
}

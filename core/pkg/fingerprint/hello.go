// Package fingerprint разбирает TLS ClientHello и считает JA3.
//
// Смысл пакета в том, чтобы перестать описывать отпечаток словами: он снимает
// реальные байты, которые уходят в сеть при tls://-ссылке, и превращает их в
// проверяемые числа. Всё остальное в репозитории (uTLS-подобный dialer,
// фрагментация) к этим байтам отношения не имеет.
package fingerprint

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Типы TLS-записей и рукопожатий, нужные для разбора.
const (
	recordTypeHandshake byte = 0x16
	typeClientHello     byte = 0x01
)

// Коды расширений ClientHello, которые разбираются по содержимому.
const (
	extServerName        uint16 = 0x0000
	extSupportedGroups   uint16 = 0x000a
	extPointFormats      uint16 = 0x000b
	extSignatureAlgs     uint16 = 0x000d
	extALPN              uint16 = 0x0010
	extSupportedVersions uint16 = 0x002b
	extKeyShare          uint16 = 0x0033
)

// ErrShort — байтов пока меньше, чем нужно для целого ClientHello. Сканирующему
// читателю это сигнал досчитать, а не ошибка соединения.
var ErrShort = errors.New("fingerprint: данных пока недостаточно")

// ClientHello — разбор первого полёта рукопожатия. Поля сохраняют порядок из
// пакета: и JA3, и DPI-сигнатуры считают именно последовательность, а не множество.
type ClientHello struct {
	// RecordVersions — версии в заголовках TLS-записей, которыми hello реально
	// пришёл. Один hello в нескольких записях = фрагментация, и это отдельный
	// от печатка факт.
	RecordVersions []uint16
	LegacyVersion  uint16
	SessionID      []byte
	CipherSuites   []uint16
	Compression    []byte
	Extensions     []uint16

	SupportedVersions []uint16
	SupportedGroups   []uint16
	KeyShareGroups    []uint16
	PointFormats      []byte
	SignatureAlgs     []uint16
	Protocols         []string
	ServerName        string
}

// Scan разбирает поток байтов TCP, начиная с первой TLS-записи. Возвращает
// (hello, true, nil), когда ClientHello собран целиком, (nil, false, ErrShort),
// когда нужно дочитать, и ошибку, когда это точно не ClientHello.
func Scan(stream []byte) (*ClientHello, bool, error) {
	var body []byte
	versions := []uint16{}

	p := stream
	for {
		if len(p) < 5 {
			if len(body) == 0 {
				return nil, false, ErrShort
			}
			return nil, false, fmt.Errorf("fingerprint: обломок TLS-записи в %d байт", len(p))
		}
		if p[0] != recordTypeHandshake {
			if len(body) == 0 {
				return nil, false, fmt.Errorf("fingerprint: первая запись не handshake (тип 0x%02x)", p[0])
			}
			return nil, false, fmt.Errorf("fingerprint: handshake прерван записью типа 0x%02x", p[0])
		}
		recVer := uint16(p[1])<<8 | uint16(p[2])
		// Версия в заголовке TLS-записи лежит в диапазоне 0x0300–0x0303 (SSL 3.0
		// … TLS 1.3, где запись формально остаётся 0x0303). Всё прочее — не TLS,
		// и ждать из такого потока hello бессмысленно.
		if recVer>>8 != 3 || recVer&0xff > 3 {
			return nil, false, fmt.Errorf("fingerprint: невозможная версия TLS-записи 0x%04x", recVer)
		}
		recLen := int(uint16(p[3])<<8 | uint16(p[4]))
		p = p[5:]
		if len(p) < recLen {
			if len(body) == 0 && recLen > 0 {
				return nil, false, ErrShort
			}
			return nil, false, fmt.Errorf("fingerprint: запись обещает %d байт, есть %d", recLen, len(p))
		}
		if len(versions) == 0 || versions[len(versions)-1] != recVer {
			versions = append(versions, recVer)
		}
		payload := p[:recLen]
		p = p[recLen:]

		if len(body) == 0 {
			if len(payload) < 4 {
				return nil, false, fmt.Errorf("fingerprint: handshake-заголовок короче 4 байт")
			}
			if payload[0] != typeClientHello {
				return nil, false, fmt.Errorf("fingerprint: первое рукопожатие не client_hello (тип 0x%02x)", payload[0])
			}
			want := int(uint32(payload[1])<<16|uint32(payload[2])<<8|uint32(payload[3])) + 4
			if len(payload) >= want {
				return parse(payload, versions)
			}
			body = append(body, payload...)
			continue
		}
		body = append(body, payload...)
		want := int(uint32(body[1])<<16|uint32(body[2])<<8|uint32(body[3])) + 4
		if len(body) >= want {
			return parse(body[:want], versions)
		}
	}
}

// parse разбирает ровно одно сообщение client_hello.
func parse(msg []byte, versions []uint16) (*ClientHello, bool, error) {
	if msg[0] != typeClientHello {
		return nil, false, fmt.Errorf("fingerprint: ожидался client_hello, получен тип 0x%02x", msg[0])
	}
	h := &ClientHello{RecordVersions: versions}
	p := msg[4:]

	if len(p) < 34 {
		return nil, false, errors.New("fingerprint: client_hello короче фиксированной части")
	}
	h.LegacyVersion = uint16(p[0])<<8 | uint16(p[1])
	// random (32 байт) в отпечатке не участвует — он случайный по определению.
	p = p[34:]

	sidLen := int(p[0])
	p = p[1:]
	if len(p) < sidLen {
		return nil, false, errors.New("fingerprint: session id оборван")
	}
	h.SessionID = append([]byte(nil), p[:sidLen]...)
	p = p[sidLen:]

	if len(p) < 2 {
		return nil, false, errors.New("fingerprint: оборван список шифров")
	}
	csLen := int(uint16(p[0])<<8 | uint16(p[1]))
	p = p[2:]
	if len(p) < csLen || csLen%2 != 0 {
		return nil, false, fmt.Errorf("fingerprint: список шифров некорректен (%d байт)", csLen)
	}
	for i := 0; i < csLen; i += 2 {
		h.CipherSuites = append(h.CipherSuites, uint16(p[i])<<8|uint16(p[i+1]))
	}
	p = p[csLen:]

	if len(p) < 1 {
		return nil, false, errors.New("fingerprint: оборван метод сжатия")
	}
	cmpLen := int(p[0])
	p = p[1:]
	if len(p) < cmpLen {
		return nil, false, errors.New("fingerprint: оборван список методов сжатия")
	}
	h.Compression = append([]byte(nil), p[:cmpLen]...)
	p = p[cmpLen:]

	if len(p) < 2 {
		return nil, false, errors.New("fingerprint: оборвана длина расширений")
	}
	extTotal := int(uint16(p[0])<<8 | uint16(p[1]))
	p = p[2:]
	if len(p) < extTotal {
		return nil, false, fmt.Errorf("fingerprint: расширений обещано %d байт, есть %d", extTotal, len(p))
	}
	p = p[:extTotal]
	for len(p) > 0 {
		if len(p) < 4 {
			return nil, false, errors.New("fingerprint: оборван заголовок расширения")
		}
		id := uint16(p[0])<<8 | uint16(p[1])
		n := int(uint16(p[2])<<8 | uint16(p[3]))
		p = p[4:]
		if len(p) < n {
			return nil, false, fmt.Errorf("fingerprint: расширение 0x%04x короче заявленных %d байт", id, n)
		}
		data := p[:n]
		p = p[n:]
		h.Extensions = append(h.Extensions, id)
		if err := h.readExtension(id, data); err != nil {
			return nil, false, err
		}
	}
	return h, true, nil
}

func (h *ClientHello) readExtension(id uint16, data []byte) error {
	switch id {
	case extServerName:
		name, err := parseServerName(data)
		if err != nil {
			return err
		}
		h.ServerName = name
	case extSupportedGroups, extSignatureAlgs, extKeyShare:
		// key_share устроен как [длина списка][группа, длина, данные]..., поэтому
		// группы здесь вытаскиваются по шагу 2 для плоских списков и вручную для
		// key_share.
		if id == extKeyShare {
			h.KeyShareGroups = []uint16{}
			p := data
			if len(p) < 2 {
				return errors.New("fingerprint: оборвана длина key_share")
			}
			total := int(uint16(p[0])<<8 | uint16(p[1]))
			p = p[2:]
			if len(p) < total {
				return errors.New("fingerprint: key_share короче заявленного")
			}
			p = p[:total]
			for len(p) >= 4 {
				group := uint16(p[0])<<8 | uint16(p[1])
				kl := int(uint16(p[2])<<8 | uint16(p[3]))
				p = p[4:]
				if len(p) < kl {
					return errors.New("fingerprint: ключ key_share оборван")
				}
				p = p[kl:]
				h.KeyShareGroups = append(h.KeyShareGroups, group)
			}
			return nil
		}
		list, err := parseUint16List(data)
		if err != nil {
			return err
		}
		if id == extSupportedGroups {
			h.SupportedGroups = list
		} else {
			h.SignatureAlgs = list
		}
	case extPointFormats:
		if len(data) < 1 {
			return errors.New("fingerprint: пустой список форматов точек")
		}
		n := int(data[0])
		if len(data) < 1+n {
			return errors.New("fingerprint: форматы точек оборваны")
		}
		h.PointFormats = append([]byte(nil), data[1:1+n]...)
	case extSupportedVersions:
		// Единственное разбираемое расширение с длиной в 1 байт: версии — это
		// пары байтов, а их количество клиент считает в одном байте.
		if len(data) < 1 {
			return errors.New("fingerprint: оборвана длина supported_versions")
		}
		n := int(data[0])
		if len(data) < 1+n || n%2 != 0 {
			return fmt.Errorf("fingerprint: supported_versions на %d байт не бьётся по парам", n)
		}
		list, err := parseFlatUint16(data[1 : 1+n])
		if err != nil {
			return err
		}
		h.SupportedVersions = list
	case extALPN:
		if len(data) < 2 {
			return errors.New("fingerprint: список ALPN оборван")
		}
		total := int(uint16(data[0])<<8 | uint16(data[1]))
		p := data[2:]
		if len(p) < total {
			return errors.New("fingerprint: ALPN короче заявленного")
		}
		p = p[:total]
		for len(p) > 0 {
			n := int(p[0])
			p = p[1:]
			if len(p) < n {
				return errors.New("fingerprint: строка ALPN оборвана")
			}
			h.Protocols = append(h.Protocols, string(p[:n]))
			p = p[n:]
		}
	}
	return nil
}

func parseServerName(data []byte) (string, error) {
	if len(data) < 2 {
		return "", errors.New("fingerprint: список имён сервера оборван")
	}
	total := int(uint16(data[0])<<8 | uint16(data[1]))
	p := data[2:]
	if len(p) < total {
		return "", errors.New("fingerprint: имена сервера короче заявленного")
	}
	p = p[:total]
	for len(p) >= 3 {
		kind := p[0]
		n := int(uint16(p[1])<<8 | uint16(p[2]))
		p = p[3:]
		if len(p) < n {
			return "", errors.New("fingerprint: имя сервера оборвано")
		}
		if kind == 0 {
			return string(p[:n]), nil
		}
		p = p[n:]
	}
	return "", nil
}

func parseUint16List(data []byte) ([]uint16, error) {
	if len(data) < 2 {
		return nil, errors.New("fingerprint: длина списка меньше 2 байт")
	}
	total := int(uint16(data[0])<<8 | uint16(data[1]))
	p := data[2:]
	if len(p) < total || total%2 != 0 {
		return nil, fmt.Errorf("fingerprint: список на %d байт не бьётся по парам", total)
	}
	return parseFlatUint16(p[:total])
}

// parseFlatUint16 читает список пар байтов без префикса длины.
func parseFlatUint16(p []byte) ([]uint16, error) {
	if len(p)%2 != 0 {
		return nil, fmt.Errorf("fingerprint: %d байт не раскладываются по парам", len(p))
	}
	out := make([]uint16, 0, len(p)/2)
	for i := 0; i < len(p); i += 2 {
		out = append(out, uint16(p[i])<<8|uint16(p[i+1]))
	}
	return out, nil
}

// HasExtension — присутствовало ли расширение в hello, независимо от того,
// разобрали ли мы его содержимое. Порядок в Extensions сохраняется, поэтому по
// этой проверке видно и сам факт, и место в списке.
func (h *ClientHello) HasExtension(id uint16) bool {
	for _, e := range h.Extensions {
		if e == id {
			return true
		}
	}
	return false
}

// HasSNI — расширение server_name с непустым именем. Для DPI это важнее, чем
// кажется: SNI у yggdrasil-ссылки появляется только когда пир задан именем, а
// не адресом.
func (h *ClientHello) HasSNI() bool { return h.ServerName != "" }

// HasALPN — заявлен ли в hello прикладной протокол (h2, http/1.1, h3). Его
// отсутствие — отдельный признак: браузеры почти всегда что-то называют.
func (h *ClientHello) HasALPN() bool { return len(h.Protocols) > 0 }

// isGrease — значения 0x?a?a, которыми браузеры намеренно ломают жёсткие
// сигнатуры. Go-клиент GREASE не вставляет, и это один из главных признаков, по
// которому его видно.
func isGrease(v uint16) bool { return v&0x0f0f == 0x0a0a }

// HasGrease — есть ли GREASE в шифрах, расширениях, группах или версиях.
func (h *ClientHello) HasGrease() bool {
	for _, set := range [][]uint16{h.CipherSuites, h.Extensions, h.SupportedGroups, h.SupportedVersions, h.KeyShareGroups} {
		for _, v := range set {
			if isGrease(v) {
				return true
			}
		}
	}
	return false
}

// VersionForJA3 — первое поле описания JA3: legacy_version из тела hello
// (0x0303 = 771 у всех современных клиентов, включая браузеры). Если брать
// старшую supported_versions, published-строки Chrome перестанут сходиться.
func (h *ClientHello) VersionForJA3() uint16 { return h.LegacyVersion }

// JA3Spec — текстовое описание печатка по исходной спецификации JA3:
//
//	SSLVersion,Ciphers,Extensions,Curves,PointFormats
//
// где списки — десятичные коды в том порядке, в каком они лежат в hello.
// Возвращает строку и её MD5 (собственно JA3).
func (h *ClientHello) JA3Spec() string {
	parts := []string{
		strconv.Itoa(int(h.VersionForJA3())),
		joinUint16(h.CipherSuites),
		joinUint16(h.Extensions),
		joinUint16(h.SupportedGroups),
		joinBytes(h.PointFormats),
	}
	return strings.Join(parts, ",")
}

// JA3 — MD5 от описания печатка, hex-строка.
func (h *ClientHello) JA3() string {
	sum := md5.Sum([]byte(h.JA3Spec()))
	return hex.EncodeToString(sum[:])
}

func joinUint16(in []uint16) string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strconv.Itoa(int(v))
	}
	return strings.Join(out, "-")
}

func joinBytes(in []byte) string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strconv.Itoa(int(v))
	}
	return strings.Join(out, "-")
}

// Summary — одна строка с фактами, по которым hello отличается от браузерного:
// она же попадает в журнал замерного теста.
func (h *ClientHello) Summary() string {
	sni := "нет"
	if h.HasSNI() {
		sni = strconv.Quote(h.ServerName)
	}
	alpn := "нет"
	if h.HasALPN() {
		alpn = strings.Join(h.Protocols, ",")
	}
	records := make([]string, len(h.RecordVersions))
	for i, v := range h.RecordVersions {
		records[i] = strconv.Itoa(int(v))
	}
	return fmt.Sprintf("legacy=%d supported=[%s] шифры=[%s] расшир=[%s] группы=[%s] форматы=[%s] SNI=%s ALPN=%s GREASE=%v записи=%d[%s]",
		h.LegacyVersion, joinUint16(h.SupportedVersions), joinUint16(h.CipherSuites),
		joinUint16(h.Extensions), joinUint16(h.SupportedGroups), joinBytes(h.PointFormats),
		sni, alpn, h.HasGrease(), len(h.RecordVersions), strings.Join(records, ","))
}

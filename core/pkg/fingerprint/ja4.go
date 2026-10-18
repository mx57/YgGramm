// File: core/pkg/fingerprint/ja4.go
package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// JA4 считается по спецификации FoxIO (github.com/FoxIO-LLC/ja4, файл
// technical_details/JA4.md) и сверено с эталонной реализацией того же репозитория
// (rust/ja4/src/tls.rs). Смысл второй печатки рядом с JA3: JA3 целиком держится на
// порядке списков, а браузеры перемешивают расширения и вставляют GREASE, поэтому
// у реальных клиентов один браузер даёт десятки разных хэшей. JA4 от этого свободен:
// списки сортируются по hex, GREASE отбрасывается, SNI и ALPN вынесены в читаемую
// приставку. Для нас это не академия: именно эта приставка говорит, что у
// yggdrasil-ссылки на пир-адрес SNI нет вообще, а ALPN нет ни в каком варианте.
//
// Ограничение, которое надо помнить: Scan() разбирает только TLS-записи из TCP, так
// что первый символ печатка всегда «t». QUIC-ссылки yggdrasil шифрует своим
// рукопожатием поверх UDP, и в этот пакет байты не приходят — для них печатка тут
// не снимается совсем, а не снимается «как t».

// JA4 — печаток вида t13i121000_… (приставка и два хэша через подчёркивание).
func (h *ClientHello) JA4() string {
	prefix, ciphers, extsSigs := h.ja4Parts()
	return fmt.Sprintf("%s_%s_%s", prefix, hash12(ciphers), hash12(extsSigs))
}

// JA4Raw — то же самое до хэширования: приставка, отсортированный список шифров и
// список расширений с подписанными алгоритмами. Нужен для проверки: по нему чужой
// реализацией или вручную пересчитывается любой из двух хэшей.
func (h *ClientHello) JA4Raw() string {
	prefix, ciphers, extsSigs := h.ja4Parts()
	return fmt.Sprintf("%s_%s_%s", prefix, ciphers, extsSigs)
}

func (h *ClientHello) ja4Parts() (prefix, ciphers, extsSigs string) {
	cipherList := filterGrease(h.CipherSuites)
	extList := filterGrease(h.Extensions)

	prefix = fmt.Sprintf("t%s%c%02d%02d%s",
		h.versionForJA4(),
		h.ja4SNIMarker(),
		min(len(cipherList), 99),
		min(len(extList), 99),
		h.ja4ALPNMarker(),
	)

	sortedCiphers := hexCodes(cipherList)
	// SNI и ALPN из списка расширений выбрасываются: они уже стоят в приставке,
	// иначе один и тот же клиент печатался бы по-разному на домен и на адрес.
	sortedExts := hexCodes(withoutCodes(extList, extServerName, extALPN))
	// signature_algorithms остаются в порядке пакета: спецификация требует
	// «in the order that they appear (not sorted)», и эталонная реализация их
	// не сортирует. Сортировка здесь превратила бы разные клиенты в один
	// печаток и сломала бы пример из документации.
	sigs := hexCodesOrdered(filterGrease(h.SignatureAlgs))

	// Спецификация: хэшируется склейка «расширения_алгоритмы», а если алгоритмов
	// нет — строка заканчивается без нижнего подчёркивания.
	if len(sigs) > 0 {
		extsSigs = strings.Join(sortedExts, ",") + "_" + strings.Join(sigs, ",")
	} else {
		extsSigs = strings.Join(sortedExts, ",")
	}
	return prefix, strings.Join(sortedCiphers, ","), extsSigs
}

// versionForJA4 — две цифры версии. Если есть supported_versions (0x002b), берётся
// старшее не-GREASE значение из него, иначе legacy_version тела hello. Версия
// заголовка TLS-записи не смотрятся: они у всех современных клиентов 0x0303 и
// ничего не различают.
func (h *ClientHello) versionForJA4() string {
	if !h.HasExtension(extSupportedVersions) {
		return tlsVersionCode(h.LegacyVersion)
	}
	var best uint16
	for _, v := range filterGrease(h.SupportedVersions) {
		if v > best {
			best = v
		}
	}
	return tlsVersionCode(best)
}

// tlsVersionCode — таблица из спецификации, слово в слово. Черновые коды TLS 1.3
// (0x1301…0x1303) в ней отсутствуют, и эталонная реализация даёт для них «00»;
// Go-клиент такие значения не отправляет, так что додумывать таблицу незачем.
func tlsVersionCode(v uint16) string {
	switch v {
	case 0x0304:
		return "13"
	case 0x0303:
		return "12"
	case 0x0302:
		return "11"
	case 0x0301:
		return "10"
	case 0x0300:
		return "s3"
	case 0x0002:
		return "s2"
	case 0xfeff:
		return "d1"
	case 0xfefd:
		return "d2"
	case 0xfefc:
		return "d3"
	default:
		return "00"
	}
}

// ja4SNIMarker — «d», если расширение server_name в hello есть, иначе «i».
// Берётся именно наличие расширения, а не непустое имя: так сказано в
// спецификации и так же делает эталонная реализация.
func (h *ClientHello) ja4SNIMarker() byte {
	if h.HasExtension(extServerName) {
		return 'd'
	}
	return 'i'
}

// ja4ALPNMarker — первая и последняя ASCII-буква/цифра первого ALPN-значения,
// «00» если ALPN нет или первое значение пустое. Если крайний байт не
// буква/цифра, берутся первый и последний символы hex-представления всего
// значения — примеры этой ветки закреплены тестом.
func (h *ClientHello) ja4ALPNMarker() string {
	if len(h.Protocols) == 0 {
		return "00"
	}
	return alpnMarker(h.Protocols[0])
}

func alpnMarker(value string) string {
	if value == "" {
		return "00"
	}
	first, last := value[0], value[len(value)-1]
	if !isASCIIAlnum(first) || !isASCIIAlnum(last) {
		raw := hex.EncodeToString([]byte(value))
		return string(raw[0]) + string(raw[len(raw)-1])
	}
	return string([]byte{first, last})
}

func isASCIIAlnum(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// filterGrease — GREASE-значения отбрасываются везде, где на них натыкаемся, и
// не считаются в длинах (спецификация: «They don't count»).
func filterGrease(in []uint16) []uint16 {
	out := make([]uint16, 0, len(in))
	for _, v := range in {
		if isGrease(v) {
			continue
		}
		out = append(out, v)
	}
	return out
}

// withoutCodes — список без указанных кодов (SNI и ALPN из расширений).
func withoutCodes(in []uint16, drop ...uint16) []uint16 {
	out := make([]uint16, 0, len(in))
	for _, v := range in {
		skip := false
		for _, d := range drop {
			if v == d {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, v)
		}
	}
	return out
}

// hexCodes — 4-символьные hex-коды в нижнем регистре, отсортированные по hex. Для
// фиксированной ширины это тот же порядок, что и по значению, но сортировка
// делается именно по строкам: так печаток совпадает с эталонной реализацией.
func hexCodes(in []uint16) []string {
	out := hexCodesOrdered(in)
	sort.Strings(out)
	return out
}

// hexCodesOrdered — те же 4-символьные hex-коды, но без сортировки: порядок
// появления нужен для signature_algorithms.
func hexCodesOrdered(in []uint16) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = fmt.Sprintf("%04x", v)
	}
	return out
}

// hash12 — sha256 от строки, первые 6 байт в нижнем регистре. Пустая строка даёт
// не хэш от ничего, а «000000000000»: по спецификации это способ показать
// наблюдателю, что полей в списке не было.
func hash12(s string) string {
	if s == "" {
		return "000000000000"
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

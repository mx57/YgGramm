// File: core/cmd/tspuprobe/main_test.go
//
// Проверки сборщика отчёта. Сеть здесь не трогается: живые строки даёт RunProbe,
// а этот файл сторожит ровно то, по чему отчёт врал без единой ошибки в замерке:
// одна цель не должна считаться дважды, а повторные прогоны не должны сливаться
// в одну строку.
package main

import (
	"strings"
	"testing"

	"ygggram/core/pkg/transport"
)

func outcome(profile, addr, phase, err string) transport.Outcome {
	return transport.Outcome{Profile: profile, Target: addr, Phase: phase, Err: err}
}

// Две цели на одном адресе — разные строки итога. Именно здесь первая версия
// отчёта печатала 9/12 там, где было 3/6 и 6/6: группировка шла по addr, а
// различие целей живёт в принудительном SNI.
func TestMatrixKeepsTargetsApartOnSameAddress(t *testing.T) {
	m := newMatrix()
	for _, p := range []string{"go-bare @discord.com", "go-alpn @discord.com", "no-sni @discord.com"} {
		m.add("1.1.1.1:443#discord.com", outcome(p, "1.1.1.1:443", transport.PhaseTLS, "timeout"))
	}
	for _, p := range []string{"go-bare @discord.com", "no-sni @discord.com"} {
		m.add("1.1.1.1:443#discord.com", outcome(p, "1.1.1.1:443", transport.PhaseTLSOK, ""))
	}
	for _, p := range []string{"go-bare @example.com", "go-alpn @example.com", "no-sni @example.com"} {
		m.add("1.1.1.1:443#example.com", outcome(p, "1.1.1.1:443", transport.PhaseTLSOK, ""))
	}

	lines := m.targetLines()
	if len(lines) != 2 {
		t.Fatalf("целей должно быть две строки, получено %d: %q", len(lines), lines)
	}
	if !strings.Contains(lines[0], "tls-ok 2/5") {
		t.Fatalf("discord посчитан по чужим строкам: %q", lines[0])
	}
	if !strings.Contains(lines[1], "tls-ok 3/3") {
		t.Fatalf("example.com посчитан по чужим строкам: %q", lines[1])
	}
	if strings.Contains(lines[0], "12") || strings.Contains(lines[1], "12") {
		t.Fatalf("итог по цели снова склеил обе цели: %q", lines)
	}
}

// Повторы одной ячейки: доля успехов и набор фаз считаются по всем прогонам, а
// порядок фаз стабилен — отчёт сравнивают между запусками.
func TestMatrixAggregatesRepeatsPerCell(t *testing.T) {
	m := newMatrix()
	label := "one.one.one.one:443#ntc.party"
	m.add(label, outcome("go-alpn-frag5 @ntc.party", "one.one.one.one:443", transport.PhaseTLS, "reset: connection reset by peer"))
	m.add(label, outcome("go-alpn-frag5 @ntc.party", "one.one.one.one:443", transport.PhaseTLS, "reset: connection reset by peer"))
	m.add(label, outcome("go-alpn-frag5 @ntc.party", "one.one.one.one:443", transport.PhaseTLSOK, ""))

	cells := m.cellLines()
	if len(cells) != 1 {
		t.Fatalf("повторы одной пары не должны плодить ячейки: %q", cells)
	}
	got := cells[0]
	if !strings.Contains(got, "tls-ok 1/3") {
		t.Fatalf("доля успехов неверна: %q", got)
	}
	if !strings.Contains(got, "tls=2 tls-ok=1") {
		t.Fatalf("фазы посчитаны или упорядочены неверно: %q", got)
	}
	if !strings.Contains(got, "ошибки: reset: connection reset by peer=2") {
		t.Fatalf("причина отказа не собрана в ячейке: %q", got)
	}
}

func TestParseTargetsKeepsLabels(t *testing.T) {
	list, err := parseTargets("1.1.1.1:443#discord.com, example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("целей не две: %+v", list)
	}
	if list[0].addr != "1.1.1.1:443" || list[0].sni != "discord.com" || list[0].label != "1.1.1.1:443#discord.com" {
		t.Fatalf("метка цели потеряна: %+v", list[0])
	}
	if list[1].sni != "" {
		t.Fatalf("у цели без SNI появилось принудительное имя: %+v", list[1])
	}
	if _, err := parseTargets("1.1.1.1"); err == nil {
		t.Fatalf("цель без порта принята молча")
	}
}

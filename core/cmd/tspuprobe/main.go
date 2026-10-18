// File: core/cmd/tspuprobe/main.go
//
// Полевая замерка различимости: матрица «профиль исходящего ClientHello × цель»
// против реальной сети, и вывод, который можно перенести в README как числа.
//
// Почему это отдельный бинарник, а не тест. Ответ на вопрос «видит ли посредник
// нас и мешает ли ему наш handshake» даёт не наша реализация, а чужая сеть:
// тест, которому нужен живой фильтрующий посредник, либо пропускается всегда,
// либо зеленеет только там, где ему и положено падать. Здесь же инструмент,
// который честно печатает фазы и отпечатки, а вывод делает читающий.
//
// Что он НЕ утверждает:
//   - он не проверяет сертификаты целей (InsecureSkipVerify в профидах нужен,
//     чтобы считать рукопожатие с произвольной целью, и это не про доверие);
//   - он не отправляет полезных данных — дальше первого рукопожатия дело не
//     идёт, и это видно по фазе;
//   - он не доказывает, что отказ дал именно ТСПУ: фаза и текст ошибки говорят,
//     ГДЕ встал путь, а КТО это сделал — отдельный вопрос. Поэтому в отчёте
//     рядом печатается контроль (разрешённая цель тем же профилем);
//   - один прогон не доказывает детерминизм: правило посредника может срабатывать
//     через раз, и строка «tls-ok» из одного запуска не отличие профиля, а случай.
//     Для выводов матрицу прогоняют флагом -repeat N и смотрят на долю успехов
//     в ячейке, а не на отдельную строку.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"ygggram/core/pkg/transport"
)

// defaultTargets — набор, подобранный под три разных диагноза, а не под «проверим
// десять сайтов». Разрешённая цель обязана быть в списке: без контроля нельзя
// отличить «наш handshake не прошёл» от «в этот момент сеть не работала».
var defaultTargets = []string{
	"example.com:443",               // контроль: адресат открыт для любого клиента
	"one.one.one.one:443",           // контроль: чужой TLS-хост на Cloudflare
	"discord.com:443",               // DNS открывается, handshake по имени рвётся
	"x.com:443",                     // то же на другой сети
	"www.youtube.com:443",           // блокировка на уровне DNS (NXDOMAIN)
	"162.159.136.232:443",           // адрес того же discord без имени: проверка, есть ли блок по IP
	"1.1.1.1:443#discord.com",       // адрес открыт, имя из чёрного списка
	"1.1.1.1:443#example.com",       // тот же адрес, имя из белого списка
	"one.one.one.one:443#ntc.party", // имя, по которому режут TLS, при открытой цели
}

func main() {
	var (
		targets  = flag.String("targets", strings.Join(defaultTargets, ","), "список целей через запятую; addr или addr#sni")
		profileF = flag.String("profiles", "", "имена профилей через запятую (пусто = все из transport.Profiles)")
		timeout  = flag.Duration("timeout", 12*time.Second, "на один handshake")
		repeat   = flag.Int("repeat", 1, "сколько раз прогнать матрицу целиком")
		asJSON   = flag.Bool("json", false, "машиночитаемый вывод вместо таблицы")
	)
	flag.Parse()

	if *repeat < 1 {
		fmt.Fprintln(os.Stderr, "-repeat должен быть не меньше 1")
		os.Exit(2)
	}

	profSet, err := pickProfiles(*profileF)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	list, err := parseTargets(*targets)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if !*asJSON {
		fmt.Printf("# tspuprobe %s: go %s %s/%s, таймаут %s\n",
			time.Now().Format(time.RFC3339), runtime.Version(), runtime.GOOS, runtime.GOARCH, timeout)
		fmt.Printf("# целей %d, профилей %d, прогонов %d, запросов %d. Контроль — разрешённая цель тем же профилем.\n",
			len(list), len(profSet), *repeat, len(list)*len(profSet)*(*repeat))
	}

	base := context.Background()
	rows := make([]transport.Outcome, 0, len(list)*len(profSet)**repeat)
	m := newMatrix()

	for round := 0; round < *repeat; round++ {
		for _, t := range list {
			for _, p := range profSet {
				if t.sni != "" {
					// Адрес открыт, имя — из списка: именно так отделяется фильтрация
					// по SNI от фильтрации по адресу.
					p.ForceSNI = t.sni
					p.Name = p.Name + " @" + t.sni
				}
				// Таймаут на каждый handshake свой: один висящий цель-профиль не
				// должен отнимать бюджет у остальных.
				runCtx, cancel := context.WithTimeout(base, *timeout)
				res := transport.RunProbe(runCtx, t.addr, p)
				cancel()
				rows = append(rows, res)
				m.add(t.label, res)
				if !*asJSON {
					fmt.Println(res.String())
				}
			}
			if !*asJSON {
				fmt.Println()
			}
		}
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{
			"at":       time.Now().Format(time.RFC3339),
			"go":       runtime.Version(),
			"os":       runtime.GOOS + "/" + runtime.GOARCH,
			"timeouts": timeout.String(),
			"repeat":   *repeat,
			"outcomes": rows,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if *repeat > 1 {
		fmt.Println("# Ячейки: сколько из прогонов handshake дошёл до конца. Доля ниже 1 — правило посредника срабатывает через раз.")
		for _, line := range m.cellLines() {
			fmt.Println(line)
		}
		fmt.Println()
	}

	fmt.Println("# Итог по целям: сколько профилей дошло до рукопожатия.")
	for _, line := range m.targetLines() {
		fmt.Println(line)
	}
}

// cell — агрегат по паре «профиль × цель» за несколько прогонов одной матрицы.
type cell struct {
	profile string
	target  string
	phases  map[string]int
	errs    map[string]int
	total   int
	ok      int
}

func (c *cell) String() string {
	line := fmt.Sprintf("%-24s %-26s tls-ok %d/%d  %s",
		c.profile, c.target, c.ok, c.total, phaseSummary(c.phases))
	if len(c.errs) > 0 {
		line += "  ошибки: " + phaseSummary(c.errs)
	}
	return line
}

// matrix собирает ячейки и итоги по целям. Ключ ячейки — имя профиля вместе с
// меткой цели: у двух целей на одном адресе (1.1.1.1#discord.com и
// 1.1.1.1#example.com) иначе получалась одна строка с двойным счётом, и отчёт
// врал без всякой ошибки в замерке.
type matrix struct {
	cells   map[string]*cell
	order   []string
	targets map[string]*cell
	tOrder  []string
}

func newMatrix() *matrix {
	return &matrix{cells: map[string]*cell{}, targets: map[string]*cell{}}
}

func (m *matrix) add(label string, res transport.Outcome) {
	key := res.Profile + "\x00" + label
	c := m.cells[key]
	if c == nil {
		c = &cell{profile: res.Profile, target: label, phases: map[string]int{}, errs: map[string]int{}}
		m.cells[key] = c
		m.order = append(m.order, key)
	}
	t := m.targets[label]
	if t == nil {
		t = &cell{profile: "", target: label, phases: map[string]int{}, errs: map[string]int{}}
		m.targets[label] = t
		m.tOrder = append(m.tOrder, label)
	}
	for _, x := range []*cell{c, t} {
		x.phases[res.Phase]++
		if res.Err != "" {
			x.errs[res.Err]++
		}
		x.total++
		if res.Phase == transport.PhaseTLSOK {
			x.ok++
		}
	}
}

func (m *matrix) cellLines() []string {
	out := make([]string, 0, len(m.order))
	for _, key := range m.order {
		out = append(out, m.cells[key].String())
	}
	return out
}

func (m *matrix) targetLines() []string {
	out := make([]string, 0, len(m.tOrder))
	for _, label := range m.tOrder {
		c := m.targets[label]
		out = append(out, fmt.Sprintf("%-34s tls-ok %d/%d  %s", c.target, c.ok, c.total, phaseSummary(c.phases)))
	}
	return out
}

type target struct {
	addr  string
	sni   string
	label string
}

func parseTargets(raw string) ([]target, error) {
	parts := strings.Split(raw, ",")
	out := make([]target, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		addr, sni, _ := strings.Cut(s, "#")
		if !strings.Contains(addr, ":") {
			return nil, fmt.Errorf("цель %q без порта: нужно host:port", s)
		}
		out = append(out, target{addr: addr, sni: sni, label: s})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("список целей пуст")
	}
	return out, nil
}

func pickProfiles(raw string) ([]transport.HelloProfile, error) {
	all := transport.Profiles()
	if strings.TrimSpace(raw) == "" {
		return all, nil
	}
	want := map[string]bool{}
	for _, name := range strings.Split(raw, ",") {
		if n := strings.TrimSpace(name); n != "" {
			want[n] = true
		}
	}
	out := make([]transport.HelloProfile, 0, len(want))
	for _, p := range all {
		if want[p.Name] {
			out = append(out, p)
		}
	}
	if len(out) != len(want) {
		missing := make([]string, 0)
		seen := map[string]bool{}
		for _, p := range out {
			seen[p.Name] = true
		}
		for n := range want {
			if !seen[n] {
				missing = append(missing, n)
			}
		}
		return nil, fmt.Errorf("профилей нет в наборе: %s (есть: %s)",
			strings.Join(missing, ", "), profileNames(all))
	}
	return out, nil
}

func profileNames(all []transport.HelloProfile) string {
	names := make([]string, len(all))
	for i, p := range all {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

func phaseSummary(phases map[string]int) string {
	keys := make([]string, 0, len(phases))
	for k := range phases {
		keys = append(keys, k)
	}
	// Порядок нужен стабильный: отчёт сравнивается между запусками.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, phases[k]))
	}
	return strings.Join(parts, " ")
}

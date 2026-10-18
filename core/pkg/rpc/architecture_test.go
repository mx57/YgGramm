// File: core/pkg/rpc/architecture_test.go
package rpc

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Этот тест — не про сообщения и не про сеть. Он держит реестр пакетов, до
// которых дата-путь не доходит, чтобы «каркас» и «работает» не расходились
// снова, как разошлись после Шага 0..8: семь пакетов были описаны в README
// рабочими фичами, хотя не импортировались ни одним живым слоем.
//
// Определение честное и жёсткое: пакет достижим, если до него есть цепочка
// НЕтестовых импортов от одного из исполняемых корней (cmd/*). Тестовый импорт
// достижимостью не считается — тест доказывает код, а не существование
// вызывающего.
//
// Что делать при падении:
//   - пакет пропал из списка недостижимых — у него появился вызывающий. Это
//     хороший исход, но тогда пакет обязан обзавестись собственными тестами,
//     а описания в README и AGENTS должны перестать называть его каркасом;
//     добавь пакет в archRoots-достижимое состояние (то есть убери из
//     archUnreachable) вместе с документацией;
//   - пакет появился в списке — новая подсистема лёглась без вызывающего, а
//     значит она каркас. Либо добавь вызывающего, либо внеси пакет сюда и
//     опиши его как каркас.

// archModule — путь модуля из core/go.mod. Сверяется явно: при переименовании
// модуля поиск импортов молча перестал бы находить рёбра, и тест бы «позеленел»
// враньём, а не правдой.
const archModule = "ygggram/core"

// archRoots — исполняемые корни: то, что запускает человек или приложение.
// cmd/tspuprobe — корень-инструмент: он не участвует в обмене сообщениями, но
// вызывает pkg/transport по-настоящему, и прятать это в «каркасы» больше нельзя.
var archRoots = []string{"cmd/libygggram", "cmd/ygggramnode", "cmd/hellomirror", "cmd/tspuprobe"}

// archUnreachable — реестр пакетов без вызывающего. Проверено этим тестом, а не
// памятью: см. README (раздел «Монорепо») и AGENTS.md (тот же реестр словами).
// pkg/transport из него вычеркнут Шаг 16: вызывает его cmd/tspuprobe.
var archUnreachable = []string{
	"pkg/calls",
	"pkg/channels",
	"pkg/contacts",
	"pkg/delivery",
	"pkg/media",
}

// archGroupMlsIdents — имена из pkg/crypto/group_mls.go. Отдельная проверка, а
// не часть реестра пакетов: сам pkg/crypto достижим и удалён быть не может, а
// вот один его файл никто не вызывает.
var archGroupMlsIdents = []string{
	"GroupSession",
	"GroupMemberInfo",
	"SenderKeyState",
	"SenderKeyDistributionMessage",
	"EncryptedGroupMessage",
	"NewGroupSession",
	"RotateEpoch",
}

type archGraph struct {
	// edges: пакет -> импортируемые пакеты этого модуля (нетестовые файлы).
	edges     map[string]map[string]bool
	importers map[string]map[string]bool
	packages  map[string]bool
}

// archModuleRoot поднимается от рабочей директории теста до каталога с go.mod.
func archModuleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("рабочая директория недоступна: %v", err)
	}
	for {
		if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
			first := ""
			for _, line := range strings.Split(string(data), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					first = line
					break
				}
			}
			want := "module " + archModule
			if first != want {
				t.Fatalf("модуль в go.mod называется %q, а тест ищет %q: обнови константу", first, want)
			}
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("не найден go.mod выше по дереву от теста")
		}
		dir = parent
	}
}

func archBuildGraph(t *testing.T) archGraph {
	t.Helper()

	root := archModuleRoot(t)
	g := archGraph{
		edges:     map[string]map[string]bool{},
		importers: map[string]map[string]bool{},
		packages:  map[string]bool{},
	}

	// Первый проход: какие пакеты вообще существуют (по каталогам с .go-файлами).
	// Второй: рёбра. Раздельно, потому что импорт может указывать на пакет,
	// которого нет на диске, — такое ребро отбрасывается, а не создаёт фантом.
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "build" || name == "gen") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		g.packages[filepath.ToSlash(rel)] = true
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("обход дерева сорвался: %v", err)
	}

	fset := token.NewFileSet()
	for _, path := range files {
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			t.Fatalf("путь %q не относительный к %q: %v", path, root, err)
		}
		from := filepath.ToSlash(rel)

		// ImportsOnly: нужен именно список импортов, а разбор тел тысяч строк
		// сгенерированного protobuf замедлил бы тест в разы.
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("файл %q не разбирается: %v", path, err)
		}
		for _, imp := range file.Imports {
			to := strings.Trim(imp.Path.Value, `"`)
			if to == archModule || !strings.HasPrefix(to, archModule+"/") {
				continue // сторонняя зависимость или сам модуль целиком
			}
			to = strings.TrimPrefix(to, archModule+"/")
			if !g.packages[to] {
				t.Fatalf("%s импортирует %s, а такого пакета на диске нет", from, to)
			}
			if g.edges[from] == nil {
				g.edges[from] = map[string]bool{}
			}
			g.edges[from][to] = true
			if g.importers[to] == nil {
				g.importers[to] = map[string]bool{}
			}
			g.importers[to][from] = true
		}
	}
	return g
}

// archReachable — транзитивное замыкание от корней по нетестовым импортам.
func archReachable(g archGraph) map[string]bool {
	seen := map[string]bool{}
	var visit func(string)
	visit = func(p string) {
		if seen[p] {
			return
		}
		seen[p] = true
		for next := range g.edges[p] {
			visit(next)
		}
	}
	for _, root := range archRoots {
		visit(root)
	}
	return seen
}

func archSorted(pkgs []string) []string {
	out := append([]string(nil), pkgs...)
	sort.Strings(out)
	return out
}

func TestUnreachablePackagesMatchRegistry(t *testing.T) {
	g := archBuildGraph(t)
	for _, root := range archRoots {
		if !g.packages[root] {
			t.Fatalf("корень %q не найден среди пакетов модуля: реестр устарел вместе с деревом", root)
		}
	}

	reachable := archReachable(g)
	var orphaned []string
	for p := range g.packages {
		if !strings.HasPrefix(p, "pkg/") {
			continue // cmd/* — корни, proto/* — генерация на границе UI↔ядро
		}
		if !reachable[p] {
			orphaned = append(orphaned, p)
		}
	}
	orphaned = archSorted(orphaned)
	want := archSorted(archUnreachable)

	if strings.Join(orphaned, ",") != strings.Join(want, ",") {
		inFact := map[string]bool{}
		for _, p := range orphaned {
			inFact[p] = true
		}
		for _, p := range want {
			if !inFact[p] {
				t.Errorf("пакет %s помечен каркасом, но до него есть цепочка импортов от %v: его импортируют %v. Убери его из archUnreachable, добавь ему тесты и перестань называть его каркасом в README и AGENTS",
					p, archRoots, sortedKeys(g.importers[p]))
			}
		}
		wanted := map[string]bool{}
		for _, p := range want {
			wanted[p] = true
		}
		for _, p := range orphaned {
			if !wanted[p] {
				t.Errorf("пакет %s не вызывает никто, а в реестре его нет: это новая подсистема без вызывающего. Либо добавь ей вызывающего, либо внеси в archUnreachable и опиши в README и AGENTS как каркас", p)
			}
		}
		t.Fatalf("реестр недостижимых пакетов разошёлся с деревом: в реестре %v, по факту %v", want, orphaned)
	}

	// Обратная сторона: достижимые пакеты не имеют права притворяться
	// недосягаемыми, и ни один из них не обязан знать про реестр.
	for _, p := range archUnreachable {
		if len(g.importers[p]) != 0 {
			t.Errorf("пакет %s в реестре «без вызывающего», но его импортируют %v", p, sortedKeys(g.importers[p]))
		}
	}
}

func TestGroupMlsFileHasNoCallers(t *testing.T) {
	root := archModuleRoot(t)

	hits := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if strings.HasPrefix(slash, "pkg/crypto/") {
			return nil // здесь живут сами определения
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		for _, ident := range archGroupMlsIdents {
			if strings.Contains(text, ident) {
				hits[ident] = append(hits[ident], slash)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("обход дерева сорвался: %v", err)
	}
	for _, ident := range sortedKeys(hits) {
		t.Errorf("%s упоминается вне pkg/crypto в %v: sender keys получили потребителя. Это по-прежнему не MLS (дерева участников, Welcome/Commit и групповых protobuf-сообщений нет), так что описания обязаны остаться честными, а реестр и тесты групповой части — появиться",
			ident, archSorted(hits[ident]))
	}
}

func sortedKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

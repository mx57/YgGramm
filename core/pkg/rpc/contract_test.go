// File: core/pkg/rpc/contract_test.go
//
// Контракт UI ↔ ядро живёт в proto/ygggram.proto, а не в памяти того, кто
// писал диспетчер. Три ветки этого контракта молча расходятся с кодом, и ни
// компилятор, ни go vet это не ловят:
//
//   - у CoreRequest нет обязательной полноты: switch в Handle оканчивается
//     default-ом, поэтому новая команда из protobuf превращается в «неизвестный
//     тип запроса» на пустом месте. Так и случилось с disconnect_peer — ветка
//     была в контракте, кейса в ядре не было;
//   - у CoreEvent реестр «что ядро публикует, а что нет» — единственное, чем
//     docs отвечают на «UI молча выбрасывает три ветки». Без проверки это
//     снова утверждение со словами;
//   - у CoreResponse та же яма, только тише: команда может быть разборной, а
//     наполняемой ветки `resp.Response` у неё нет — клиент прочитает пустой
//     oneof вместо результата и не получит ни ошибки, ни ответа.
//
// Список веток берётся из дескриптора сгенерированного protobuf, а не из
// хардкода: добавят поле в .proto — оно появится здесь само, и тест потребует
// либо кейс в Handle, либо осознанную запись в реестре.
package rpc

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	yggproto "ygggram/core/proto"
)

// contractUnknownDispatch — маркер ветки, до которой в Handle не дошёл ни один
// case. Сверяется по тексту ошибки: отдельного кода ответа у «неизвестного
// запроса» нет, и тест обязан упасть, если формулировку поменяют, а не пройти
// впустую.
const contractUnknownDispatch = "неизвестный тип запроса"

// contractEventPublished — ветки CoreEvent, которые ядро действительно строит.
// contractEventUnbuilt — те, для которых protobuf-сообщение есть в контракте, а
// писателя нет: UI их разбирает, но дожидаться нечего. Обе группы обязаны
// покрыть все ветки дескриптора, иначе тест валится.
var (
	contractEventPublished = []string{
		"CoreEvent_NetworkStatus",
		"CoreEvent_PeersUpdated",
		"CoreEvent_IncomingMessage",
		"CoreEvent_MessageStatus",
		"CoreEvent_ContactUpdated",
		"CoreEvent_BackupProgress",
		"CoreEvent_AntiDpiStatus",
	}
	contractEventUnbuilt = []string{
		"CoreEvent_IncomingCall",
	}
)

// contractResponseFilled — ветки CoreResponse, которые ядро действительно
// наполняет ответом. contractResponseUnfilled — те, что в контракте есть, но в
// `resp.Response` не кладётся никогда: клиент, который ждёт их, получит пустой
// oneof и молча разойдётся с ядром.
var (
	contractResponseFilled = []string{
		"CoreResponse_NodeStatus",
		"CoreResponse_PeerList",
		"CoreResponse_SendMessage",
		"CoreResponse_PrekeyBundle",
		"CoreResponse_Identity",
		"CoreResponse_Conversations",
		"CoreResponse_Messages",
		"CoreResponse_Contacts",
		"CoreResponse_SfCollect",
		"CoreResponse_FetchContact",
		"CoreResponse_Backup",
	}
	// contractResponseUnfilled пуст: с Шага 13 наполняется каждая ветка
	// CoreResponse. Реестр оставлен намеренно — без него тест молча простил бы
	// команде «разбирается, но ответа нет», а это ровно та яма из шапки файла.
	contractResponseUnfilled = []string{}
)

// contractOneofFields — поля именованной oneof-ветки сообщения. Пустой список
// означает, что контракт переписали (oneof исчезла или переименовалась), и
// молча «зелёный» тест в это время ничего не проверял бы.
func contractOneofFields(t *testing.T, msg proto.Message, oneof string) []protoreflect.FieldDescriptor {
	t.Helper()

	fields := msg.ProtoReflect().Descriptor().Oneofs().ByName(protoreflect.Name(oneof))
	if fields == nil {
		t.Fatalf("в %s нет oneof %q: контракт переписан, обнови тест",
			msg.ProtoReflect().Descriptor().FullName(), oneof)
	}
	members := fields.Fields()
	if members.Len() == 0 {
		t.Fatalf("oneof %q пуста — проверять нечего", oneof)
	}
	out := make([]protoreflect.FieldDescriptor, 0, members.Len())
	for i := 0; i < members.Len(); i++ {
		out = append(out, members.Get(i))
	}
	return out
}

// contractWrapper — имя Go-обёртки oneof-поля по правилу protoc-gen-go:
// нижнее подчёркивание → CamelCase, префикс — имя сообщения. Правило
// проверяется по сгенерированному файлу (contractStub), а не принимается на
// веру: сгенерированные имена могут разойтись, и тогда поиск по тексту ниже
// искал бы то, чего в коде нет, и вечно промахивался бы мимо.
func contractWrapper(t *testing.T, stub, structName string, fd protoreflect.FieldDescriptor) string {
	t.Helper()

	parts := strings.Split(string(fd.Name()), "_")
	for i, p := range parts {
		if p == "" {
			t.Fatalf("имя поля %q с пустым сегментом: правило именования обёрток пора пересматривать", fd.Name())
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	wrapper := structName + "_" + strings.Join(parts, "")
	if !strings.Contains(stub, "type "+wrapper+" struct") {
		t.Fatalf("обёртка %s в сгенерированном protobuf не найдена: protoc-gen-go называет её иначе, чем выводит тест", wrapper)
	}
	return wrapper
}

// contractStub читает core/proto/ygggram.pb.go — единственный источник того,
// как генератор назвал обёртки.
func contractStub(t *testing.T) string {
	t.Helper()

	path := filepath.Join(archModuleRoot(t), "proto", "ygggram.pb.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("сгенерированный контракт не читается (%s): %v", path, err)
	}
	return string(data)
}

// contractEmptyRequest собирает запрос с пустым телом выбранной ветки. Путь
// через dynamicpb и marshal/ unmarshal, а не через протерефлекс-сеттеры:
// именно разбор байтов порождает Go-обёртку oneof, а значит тест идёт тем же
// путём, что и живой FFI-вызов с пустым protobuf.
func contractEmptyRequest(t *testing.T, fd protoreflect.FieldDescriptor) *yggproto.CoreRequest {
	t.Helper()

	md := (&yggproto.CoreRequest{}).ProtoReflect().Descriptor()
	dm := dynamicpb.NewMessage(md)
	dm.Set(fd, protoreflect.ValueOfMessage(dynamicpb.NewMessage(fd.Message())))
	raw, err := proto.Marshal(dm)
	if err != nil {
		t.Fatalf("пустой запрос %s не сериализуется: %v", fd.Name(), err)
	}
	req := &yggproto.CoreRequest{}
	if err := proto.Unmarshal(raw, req); err != nil {
		t.Fatalf("пустой запрос %s не разбирается: %v", fd.Name(), err)
	}
	if req.Request == nil {
		t.Fatalf("разбор пустого %s не наполнил oneof: тест потерял смысл", fd.Name())
	}
	return req
}

// TestCoreRequestCommandsAllDispatched — ни одна ветка контракта не имеет права
// падать в default. Ответ не обязан быть успешным: пустое тело, нода не
// поднята, транспорт неизвестен — всё это честные отказы. Валится тест только
// когда команда вообще не разобралась диспетчером.
func TestCoreRequestCommandsAllDispatched(t *testing.T) {
	stub := contractStub(t)
	e := New()
	t.Cleanup(func() {
		e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}})
	})

	for _, fd := range contractOneofFields(t, &yggproto.CoreRequest{}, "request") {
		wrapper := contractWrapper(t, stub, "CoreRequest", fd)
		resp := e.Handle(contractEmptyRequest(t, fd))
		if resp == nil {
			t.Fatalf("%s: Handle вернул nil", wrapper)
		}
		if strings.Contains(resp.GetErrorMessage(), contractUnknownDispatch) {
			t.Errorf("%s: запрос до диспетчера не дошёл (%q). Добавь case в Engine.Handle с настоящим исполнителем, а не оставляй ветку контракта мёртвой",
				wrapper, resp.GetErrorMessage())
		}
		if resp.GetErrorMessage() != "" && resp.GetSuccess() {
			t.Errorf("%s: success=true вместе с текстом ошибки %q — UI покажет и результат, и сбой", wrapper, resp.GetErrorMessage())
		}
	}
}

// contractSources — нетестовые .go модуля: rel-путь → содержимое. Каталог с
// генерацией (proto/) исключается: там есть имя каждой обёртки, и поиск по
// всему дереву никогда бы не увидел отсутствия писателя.
func contractSources(t *testing.T) map[string]string {
	t.Helper()

	root := archModuleRoot(t)
	out := map[string]string{}
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
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if strings.HasPrefix(slash, "proto/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[slash] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("обход дерева сорвался: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("нетестовых исходников не найдено — тест ничего не проверяет")
	}
	return out
}

func contractFound(sources map[string]string, wrapper string) []string {
	var hits []string
	for path, text := range sources {
		if strings.Contains(text, wrapper) {
			hits = append(hits, path)
		}
	}
	sort.Strings(hits)
	return hits
}

// TestCoreEventBranchesAreClassified — реестр публикуемых веток сверяется с
// дескриптором в обе стороны и с кодом: опубликованная ветка обязана
// конструироваться хотя бы в одном нетестовом исходнике модуля, неопубликованная
// — не встречаться там нигде.
func TestCoreEventBranchesAreClassified(t *testing.T) {
	stub := contractStub(t)
	sources := contractSources(t)

	published := map[string]bool{}
	for _, w := range contractEventPublished {
		published[w] = true
	}
	unbuilt := map[string]bool{}
	for _, w := range contractEventUnbuilt {
		unbuilt[w] = true
	}
	for _, w := range contractEventPublished {
		if unbuilt[w] {
			t.Fatalf("%s числится и публикуемой, и нестроящейся", w)
		}
		if !strings.Contains(stub, "type "+w+" struct") {
			t.Errorf("обёртки %s в сгенерированном protobuf нет: реестр живёт по памяти, а не по контракту", w)
		}
	}

	var known []string
	for _, fd := range contractOneofFields(t, &yggproto.CoreEvent{}, "event") {
		wrapper := contractWrapper(t, stub, "CoreEvent", fd)
		known = append(known, wrapper)

		switch {
		case published[wrapper]:
			// Область поиска — весь модуль, а не только pkg/rpc: heartbeat
			// network_status строит слой FFI (cmd/libygggram), и это его
			// законное место. Пустой результат значит, что ветка названа
			// публикуемой впустую.
			if len(contractFound(sources, wrapper)) == 0 {
				t.Errorf("%s помечена публикуемой, но не строится ни в одном нетестовом исходнике: UI будет ждать события, которого ядро не шлёт (%s не объявлена)",
					wrapper, fd.Name())
			}
		case unbuilt[wrapper]:
			if hits := contractFound(sources, wrapper); len(hits) != 0 {
				t.Errorf("%s заявлена как «не строится нигде», но найдена в %v: у ветки появился писатель. Перенеси её в contractEventPublished, добавь ей тест и убери из README/AGENTS слова о «разобранном, но не публикуемом»",
					wrapper, hits)
			}
		default:
			t.Errorf("ветка CoreEvent %s (%s) не отнесена ни к публикуемым, ни к нестроящим: реестр отстал от контракта",
				wrapper, fd.Name())
		}
	}

	// Обратная сторона: реестр не имеет права описывать ветки, которой в
	// контракте больше нет.
	inContract := map[string]bool{}
	for _, w := range known {
		inContract[w] = true
	}
	for _, w := range append(append([]string(nil), contractEventPublished...), contractEventUnbuilt...) {
		if !inContract[w] {
			t.Errorf("%s в реестре есть, а в oneof event — нет: поле из protobuf убрали, а описание не почистили", w)
		}
	}
}

// TestCoreResponseBranchesAreClassified — третья нога того же контракта:
// ответы. Здесь ветка без писателя опаснее, чем кажется: `Handle` возвращает
// готовый `CoreResponse` с nil-овым `Response`, поэтому клиент читает пустой
// oneof и получает «успех без результата», а не ошибку. Список наполняемых
// веток сверяется с исходниками в обе стороны ровно как реестр событий.
func TestCoreResponseBranchesAreClassified(t *testing.T) {
	stub := contractStub(t)
	sources := contractSources(t)

	filled := map[string]bool{}
	for _, w := range contractResponseFilled {
		filled[w] = true
	}
	unfilled := map[string]bool{}
	for _, w := range contractResponseUnfilled {
		unfilled[w] = true
	}
	for _, w := range contractResponseFilled {
		if unfilled[w] {
			t.Fatalf("%s числится и наполняемой, и пустой", w)
		}
		if !strings.Contains(stub, "type "+w+" struct") {
			t.Errorf("обёртка %s в сгенерированном protobuf не найдена: список живёт по памяти, а не по контракту", w)
		}
	}

	var known []string
	for _, fd := range contractOneofFields(t, &yggproto.CoreResponse{}, "response") {
		wrapper := contractWrapper(t, stub, "CoreResponse", fd)
		known = append(known, wrapper)

		switch {
		case filled[wrapper]:
			if len(contractFound(sources, wrapper)) == 0 {
				t.Errorf("%s помечена наполняемой, но в нетестовых исходниках не встречается: %s не кладётся в resp.Response, а клиент её читает",
					wrapper, fd.Name())
			}
		case unfilled[wrapper]:
			if hits := contractFound(sources, wrapper); len(hits) != 0 {
				t.Errorf("%s заявлена пустой, но найдена в %v: у ветки появился писатель — перенеси её в contractResponseFilled, добавь команде тест ответа и перестань называть её отказом в README/AGENTS",
					wrapper, hits)
			}
		default:
			t.Errorf("ветка CoreResponse %s (%s) не отнесена ни к наполняемым, ни к пустым: список отстал от контракта",
				wrapper, fd.Name())
		}
	}

	inContract := map[string]bool{}
	for _, w := range known {
		inContract[w] = true
	}
	for _, w := range append(append([]string(nil), contractResponseFilled...), contractResponseUnfilled...) {
		if !inContract[w] {
			t.Errorf("%s в списке есть, а в oneof response — нет: поле из protobuf убрали, а список не почистили", w)
		}
	}
}

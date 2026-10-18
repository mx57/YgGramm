// File: core/pkg/rpc/process_test.go
//
// Два клиента в отдельных процессах. До этого файла «два узла» в тестах
// означали один процесс с двумя движками в общей памяти: они делили кучу,
// имели прямой доступ друг к другу (e.node, e.db), а «перезапуск» был вызовом
// stopNode/startNode в той же программе. Здесь каждый клиент живёт своим
// процессом, видит только protobuf и свой зашифрованный каталог, и встречается
// с другим исключительно в оверлее yggdrasil по настоящему TCP.
package rpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"ygggram/core/pkg/nodepipe"
	yggproto "ygggram/core/proto"
)

// childEnv — единственный признак, по которому тестовый бинарник понимает, что
// его запустили не ради тестов, а как процесс ядра.
const childEnv = "YGGGRAM_NODE_CHILD"

// nodeBinEnv — необязательный путь к настоящему core/cmd/ygggramnode. По
// умолчанию дочерний процесс поднимается перезапуском этого же тестового
// бинарника: так `go test` не требует отдельной сборки. Переменная позволяет
// прогнать ровно тот же сценарий против собранного CLI и тем самым проверить,
// что cmd/ygggramnode — рабочий вход, а не декорация.
const nodeBinEnv = "YGGGRAM_NODE_BIN"

// TestNodeChildProcess — тело процесса-клиента. Родитель перезапускает этот же
// тестовый бинарник с -test.run по данному имени и childEnv=1, поэтому отдельная
// сборка под e2e не нужна: тот же код и тот же go.mod, только в другом процессе.
//
// Кадры идут по stdin и stderr, а не по stdout: тестовый фреймворк пишет в
// stdout итоги прогона, и примешивать к ним рамку означало бы ловить гонку
// байтов вместо теста. Запросы приходят в stdin, ответы и события уходят в
// stderr, stdout остаётся фреймворку — его родитель показывает, когда процесс
// падает.
func TestNodeChildProcess(t *testing.T) {
	if os.Getenv(childEnv) != "1" {
		t.Skip("это не дочерний процесс: запускается только из теста на два процесса")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	if err := nodepipe.Serve(ctx, New(), os.Stdin, os.Stderr); err != nil {
		fmt.Fprintf(os.Stdout, "процесс ядра: %v\n", err)
		os.Exit(1)
	}
	// Выходим сразу, не возвращаясь в фреймворк: он дописал бы в stdout итоги
	// прогона, а родителю они уже не нужны.
	os.Exit(0)
}

// nodeProc — взгляд родителя на процесс-клиент: сам процесс, рамочный канал до
// него и буфер его stdout (только для отладочного вывода).
type nodeProc struct {
	name   string
	cmd    *exec.Cmd
	conn   *nodepipe.Conn
	stdin  io.WriteCloser
	stdout *lockedBuffer

	mu     sync.Mutex
	closed bool
}

// lockedBuffer пишет вывод процесса из горутины os/exec, а читает его тест из
// своей: без защёлки это гонка.
type lockedBuffer struct {
	mu sync.Mutex
	b  []byte
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.b = append(l.b, p...)
	return len(p), nil
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return string(l.b)
}

// launchNodeProc поднимает процесс ядра и рамочный канал до него.
//
// Рамка живёт на stderr, когда ребёнком идёт тестовый бинарник: фреймворк
// пишет свои итоги в stdout, и два потока байтов в одной трубе дали бы гонку
// вместо теста. У собранного CLI stdout свободен, и кадры идут в него — ровно
// как задумано в cmd/ygggramnode. Второй поток в обоих случаях собирается в
// буфер и показывается, когда процесс ведёт себя не так.
func launchNodeProc(t *testing.T, name string) *nodeProc {
	t.Helper()

	cli := os.Getenv(nodeBinEnv)
	var (
		cmd    *exec.Cmd
		frames io.ReadCloser
		err    error
	)
	out := &lockedBuffer{}
	if cli != "" {
		cmd = exec.Command(cli)
		cmd.Env = os.Environ()
		if frames, err = cmd.StdoutPipe(); err != nil {
			t.Fatalf("%s: stdout-труба CLI: %v", name, err)
		}
		cmd.Stderr = out
	} else {
		cmd = exec.Command(os.Args[0], "-test.run=^TestNodeChildProcess$", "-test.timeout=25m")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		if frames, err = cmd.StderrPipe(); err != nil {
			t.Fatalf("%s: stderr-труба: %v", name, err)
		}
		cmd.Stdout = out
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("%s: stdin-труба: %v", name, err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("%s: запуск процесса: %v", name, err)
	}

	p := &nodeProc{
		name:   name,
		cmd:    cmd,
		conn:   nodepipe.NewConn(frames, stdin),
		stdin:  stdin,
		stdout: out,
	}
	t.Cleanup(p.stop)
	return p
}

// stop завершает процесс штатно: закрытый stdin означает для Serve «клиент
// ушёл», и процесс выходит сам. Wait вызывается только после того, как читатель
// рамки отключился, — иначе os/exec закрыл бы трубу у нас из-под ног.
func (p *nodeProc) stop() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	_ = p.stdin.Close()
	select {
	case <-p.conn.Done():
	case <-time.After(20 * time.Second):
		_ = p.cmd.Process.Kill()
		<-p.conn.Done()
	}
	_ = p.cmd.Wait()
}

// kill рвёт процесс на ходу: так выглядит падение клиента, а не остановка.
// Сокеты закрывает ядро системы, а очередь, сеансы и база остаются на диске.
func (p *nodeProc) kill(t *testing.T) {
	t.Helper()
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	if err := p.cmd.Process.Kill(); err != nil {
		t.Fatalf("%s: убить процесс: %v", p.name, err)
	}
	select {
	case <-p.conn.Done():
	case <-time.After(30 * time.Second):
		t.Fatalf("%s: после Kill рамочный канал не закрылся: %s", p.name, p.stdout.String())
	}
	_ = p.stdin.Close()
	// Код возврата тут не «успешный выход»: процесс убит, а ждать его нужно
	// только ради того, чтобы os/exec закрыл свои ресурсы.
	_ = p.cmd.Wait()
}

// must шлёт запрос и требует успешного ответа: отказ ядра — провал теста, а не
// повод разбирать пустой ответ дальше.
func (p *nodeProc) must(t *testing.T, req *yggproto.CoreRequest) *yggproto.CoreResponse {
	t.Helper()
	resp, err := p.conn.Call(req, 30*time.Second)
	if err != nil {
		t.Fatalf("%s: запрос до процесса не дошёл: %v\nвывод процесса: %s", p.name, err, p.stdout.String())
	}
	if !resp.GetSuccess() {
		t.Fatalf("%s: ядро отказало: %s", p.name, resp.GetErrorMessage())
	}
	return resp
}

// waitEvent ждёт событие нужного вида. Чужие события по дороге выбрасываются
// намеренно: канал один, и «первое попавшееся» ничего не гарантирует.
func (p *nodeProc) waitEvent(t *testing.T, limit time.Duration, want func(*yggproto.CoreEvent) bool, why string) *yggproto.CoreEvent {
	t.Helper()
	deadline := time.After(limit)
	for {
		select {
		case ev := <-p.conn.Events():
			if ev != nil && want(ev) {
				return ev
			}
		case <-deadline:
			t.Fatalf("%s: за %s не дождался: %s; вывод процесса: %s",
				p.name, limit, why, p.stdout.String())
			return nil
		}
	}
}

// startNodeProc запускает ядро в процессе и возвращает его статус. Транспорт по
// умолчанию tcp: тест проверяет обмен через оверлей, а не режимы обхода DPI, и
// смешивать их в одном тесте незачем.
func startNodeProc(t *testing.T, p *nodeProc, mnemonic, dir string, mut func(*yggproto.NodeConfig)) *yggproto.NodeStatusResp {
	t.Helper()
	cfg := &yggproto.NodeConfig{
		DataDir:      filepath.Join(dir, "data"),
		CacheDir:     filepath.Join(dir, "cache"),
		MnemonicSeed: mnemonic,
		AntiDpiMode:  yggproto.AntiDpiMode_ANTI_DPI_DISABLED,
	}
	if mut != nil {
		mut(cfg)
	}
	st := p.must(t, &yggproto.CoreRequest{Request: &yggproto.CoreRequest_StartNode{
		StartNode: &yggproto.StartNodeReq{Config: cfg},
	}}).GetNodeStatus()
	if st == nil {
		t.Fatalf("%s: StartNode вернул пустой NodeStatusResp", p.name)
	}
	return st
}

// bundleProc запрашивает бандл предключей процесса — ровно то, что клиент
// получил бы по protobuf-границе, без доступа к движку.
func bundleProc(t *testing.T, p *nodeProc) *yggproto.PreKeyBundle {
	t.Helper()
	b := p.must(t, &yggproto.CoreRequest{
		Request: &yggproto.CoreRequest_GetPrekeyBundle{GetPrekeyBundle: &yggproto.GetPreKeyBundleReq{}},
	}).GetPrekeyBundle()
	if b == nil || len(b.GetSignedPrekey()) != 32 || len(b.GetOneTimePrekeys()) == 0 {
		t.Fatalf("%s: бандл предключей неполный: %v", p.name, b)
	}
	return b
}

// addContactProc кладёт карту собеседника: адрес и ключ узла берутся из его
// NodeStatusResp, предключи — из бандла.
func addContactProc(t *testing.T, p *nodeProc, nickname string, st *yggproto.NodeStatusResp, b *yggproto.PreKeyBundle) {
	t.Helper()
	p.must(t, &yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			NodeIpv6:      st.GetMyIpv6(),
			Ed25519Pubkey: st.GetMyEd25519Pubkey(),
			X25519Pubkey:  b.GetIdentityX25519Pubkey(),
			Nickname:      nickname,
			Bundle:        b,
		}},
	}})
}

func sendProc(t *testing.T, p *nodeProc, destX []byte, msg *yggproto.ChatMessage) *yggproto.SendMessageResp {
	t.Helper()
	resp := p.must(t, &yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{DestinationX25519Pubkey: destX, Message: msg},
	}}).GetSendMessage()
	if resp == nil {
		t.Fatalf("%s: SendMessage без SendMessageResp", p.name)
	}
	return resp
}

// messagesProc читает диалог так, как его читает UI: только через GetMessages.
func messagesProc(t *testing.T, p *nodeProc, conv string) []*yggproto.ChatMessage {
	t.Helper()
	resp := p.must(t, &yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetMessages{
		GetMessages: &yggproto.GetMessagesReq{ConversationId: conv, Limit: 200},
	}})
	return resp.GetMessages().GetMessages()
}

func bodiesProc(t *testing.T, p *nodeProc, conv string) []string {
	t.Helper()
	var out []string
	for _, m := range messagesProc(t, p, conv) {
		out = append(out, m.GetText().GetBody())
	}
	return out
}

func hasBodyProc(t *testing.T, p *nodeProc, conv, body string) bool {
	t.Helper()
	for _, got := range bodiesProc(t, p, conv) {
		if got == body {
			return true
		}
	}
	return false
}

func deliveredProc(t *testing.T, p *nodeProc, conv, messageID string) bool {
	t.Helper()
	for _, m := range messagesProc(t, p, conv) {
		if m.GetMessageId() == messageID {
			return m.GetDelivered()
		}
	}
	return false
}

// peerCountProc — сколько пирует этот процесс. Ровно то, что видит клиент:
// прямого доступа к spanning-дереву чужого процесса нет.
func peerCountProc(t *testing.T, p *nodeProc) int {
	t.Helper()
	resp := p.must(t, &yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetPeers{
		GetPeers: &yggproto.GetPeersReq{},
	}}).GetPeerList()
	n := 0
	for _, peer := range resp.GetPeers() {
		if peer.GetIsConnected() {
			n++
		}
	}
	return n
}

// deliverProc переотправляет сообщение с тем же message_id, пока адресат не
// положит его в свой диалог: первый пакет на ещё не прогретой сессии ironwood
// оверлей теряет регулярно, а дедупликация по идентификатору гасит повторы.
// Повтор — это ровно то, что делает пользователь, нажав «отправить» ещё раз.
func deliverProc(t *testing.T, from *nodeProc, destX []byte, msg *yggproto.ChatMessage,
	to *nodeProc, toConv, body string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for attempts := 1; ; attempts++ {
		sendProc(t, from, destX, msg)
		if waitUntil(t, 5*time.Second, func() bool { return hasBodyProc(t, to, toConv, body) }) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: %s не дошло до %s за %d попыток; вывод %s: %s",
				from.name, msg.GetMessageId(), to.name, attempts, to.name, to.stdout.String())
		}
	}
}

func procMessage(id, conv, body string) *yggproto.ChatMessage {
	return &yggproto.ChatMessage{
		MessageId:      id,
		ConversationId: conv,
		TimestampMs:    1700000000000,
		Content:        textContent(body),
	}
}

// TestTwoClientsInSeparateProcessesOverYggdrasil — главный тест Шага 8: два
// клиента обмениваются сквозными сообщениями, будучи отдельными процессами, и
// переживают гибель и перезапуск одного из них.
//
// Проверяется всё, что до этого держалось на «в одном процессе два движка»:
//   - рамочный канал донёс и запросы с ответами, и асинхронные события;
//   - отправка без маршрута лёг в исходящую очередь одного процесса и только
//     его, а переотправилась сама, когда появился пир, — дословными байтами;
//   - получатель раскрыл конверт, записал строку в свою зашифрованную базу и
//     прислал ACK, отправитель увидел delivered;
//   - диалог пошёл в обратную сторону по тому же сеансу Double Ratchet;
//   - после Kill и запуска НОВОГО процесса с тем же каталогом живы контакты,
//     история и сеанс, и продолжение диалога расшифровывается.
func TestTwoClientsInSeparateProcessesOverYggdrasil(t *testing.T) {
	if testing.Short() {
		t.Skip("два процесса с живым оверлеем: в -short не запускается")
	}

	root := t.TempDir()
	aliceDir := filepath.Join(root, "alice")
	bobDir := filepath.Join(root, "bob")
	alicePort := freeTCPPort(t)
	bobPort := freeTCPPort(t)

	alice := launchNodeProc(t, "Алиса")
	bob := launchNodeProc(t, "Боб")

	// Оба слушают, но пирами пока не связаны: у Алисы нет ни одного адреса
	// Боба, и первое сообщение обязано лечь в очередь, а не в пустоту.
	aSt := startNodeProc(t, alice, seedAlice, aliceDir, func(cfg *yggproto.NodeConfig) {
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(alicePort)
	})
	bSt := startNodeProc(t, bob, seedBob, bobDir, func(cfg *yggproto.NodeConfig) {
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(bobPort)
	})
	if aSt.GetMyIpv6() == "" || bSt.GetMyIpv6() == "" {
		t.Fatalf("узел не сообщил свой адрес: алиса=%q боб=%q", aSt.GetMyIpv6(), bSt.GetMyIpv6())
	}
	if aSt.GetMyIpv6() == bSt.GetMyIpv6() {
		t.Fatalf("два процесса вывели один адрес: %s", aSt.GetMyIpv6())
	}
	if peerCountProc(t, alice) != 0 {
		t.Fatal("Алиса без bootstrap-пиров уже с кем-то пирует")
	}

	aBundle := bundleProc(t, alice)
	bBundle := bundleProc(t, bob)
	addContactProc(t, alice, "Боб", bSt, bBundle)
	addContactProc(t, bob, "Алиса", aSt, aBundle)

	// Имя диалога у получателя выводится из identity key отправителя, а не из
	// conversation_id в конверте: чужое имя читать нельзя.
	bobConv := inboundConversationID(aBundle.GetIdentityX25519Pubkey())
	aliceConv := inboundConversationID(bBundle.GetIdentityX25519Pubkey())

	const conv = "два процесса"

	// --- Фаза 1: маршрута нет — конверт остаётся в очереди этого процесса ---
	first := procMessage("msg-proc-1", conv, "первое — из очереди в живом оверлее")
	sent := sendProc(t, alice, bBundle.GetIdentityX25519Pubkey(), first)
	if sent.GetWrittenToOverlay() {
		t.Fatal("без единого пира SendMessage отчитался written_to_overlay")
	}
	if !sent.GetQueuedInOutbox() {
		t.Fatalf("конверт без маршрута не лёг в исходящую очередь: %+v", sent)
	}
	if got := bodiesProc(t, bob, bobConv); len(got) != 0 {
		t.Fatalf("у Боба без маршрута уже есть сообщения: %v", got)
	}

	// --- Фаза 2: пир появляется, и очередь переотправляется сама ---
	// Никакого повторного SendMessage: ровно тот ConnectPeer, который делает
	// клиент, когда пользователь вбивает адрес узла.
	alice.must(t, &yggproto.CoreRequest{Request: &yggproto.CoreRequest_ConnectPeer{
		ConnectPeer: &yggproto.ConnectPeerReq{PeerUri: fmt.Sprintf("127.0.0.1:%d", bobPort)},
	}})

	status := alice.waitEvent(t, 60*time.Second, func(ev *yggproto.CoreEvent) bool {
		ms := ev.GetMessageStatus()
		return ms != nil && ms.GetMessageId() == "msg-proc-1" && ms.GetWrittenToOverlay()
	}, "событие written_to_overlay от сканера очереди")
	if status.GetMessageStatus().GetDelivered() {
		t.Fatal("written_to_overlay объявлено вместе с delivered: этапы обязаны разъезжать")
	}

	incoming := bob.waitEvent(t, 60*time.Second, func(ev *yggproto.CoreEvent) bool {
		im := ev.GetIncomingMessage()
		return im != nil && im.GetMessage().GetText().GetBody() == first.GetText().GetBody()
	}, "IncomingMessageEvent от Алисы")
	msg := incoming.GetIncomingMessage().GetMessage()
	if msg.GetConversationId() != bobConv {
		t.Fatalf("Боб положил сообщение в %q, ожидалось %q", msg.GetConversationId(), bobConv)
	}
	if msg.GetMessageId() != "msg-proc-1" {
		t.Fatalf("у Боба чужой идентификатор сообщения: %s", msg.GetMessageId())
	}
	if !incoming.GetIncomingMessage().GetSealedSenderVerified() {
		t.Fatal("событие не подтверждает проверку Sealed Sender")
	}
	if !waitUntil(t, 60*time.Second, func() bool {
		return deliveredProc(t, alice, conv, "msg-proc-1")
	}) {
		t.Fatalf("ACK от второго процесса не дошёл: %v", bodiesProc(t, alice, conv))
	}

	// --- Фаза 3: ответ по тому же сеансу, но уже от Боба ---
	second := procMessage("msg-proc-2", conv, "ответ из второго процесса")
	deliverProc(t, bob, aBundle.GetIdentityX25519Pubkey(), second, alice, aliceConv,
		second.GetText().GetBody())
	if !waitUntil(t, 60*time.Second, func() bool {
		return deliveredProc(t, bob, conv, "msg-proc-2")
	}) {
		t.Fatalf("ACK в обратную сторону не дошёл: %v", bodiesProc(t, bob, conv))
	}

	// --- Фаза 4: Боб падает, его процесс стартует заново с тем же каталогом ---
	bob.kill(t)

	bob2 := launchNodeProc(t, "Боб-2")
	bSt2 := startNodeProc(t, bob2, seedBob, bobDir, func(cfg *yggproto.NodeConfig) {
		// Слушателя у второго Боба нет: он сам идёт к Алисе, как делает
		// мобильный клиент, вернувшийся в сеть.
		cfg.BootstrapPeers = []string{fmt.Sprintf("127.0.0.1:%d", alicePort)}
	})
	if bSt2.GetMyIpv6() != bSt.GetMyIpv6() {
		t.Fatalf("тот же каталог и та же фраза дали другой адрес: %s против %s",
			bSt2.GetMyIpv6(), bSt.GetMyIpv6())
	}

	// Новый процесс, старые данные: история и контакт живут на диске.
	if got := bodiesProc(t, bob2, bobConv); len(got) != 1 || got[0] != first.GetText().GetBody() {
		t.Fatalf("после перезапуска процесса история Боба поехала: %v", got)
	}
	contacts := bob2.must(t, &yggproto.CoreRequest{
		Request: &yggproto.CoreRequest_ListContacts{ListContacts: &yggproto.ListContactsReq{}},
	}).GetContacts().GetContacts()
	if len(contacts) != 1 || contacts[0].GetNickname() != "Алиса" {
		t.Fatalf("контакты не пережили перезапуск процесса: %v", contacts)
	}

	// Диалог продолжается в новом процессе тем же сеансом: расшифровать
	// msg-proc-3 можно, только если состояние рэтчета действительно пережило
	// смену процесса, — номер цепочки у Алисы уже сдвинут на два шага.
	third := procMessage("msg-proc-3", conv, "пришло после перезапуска процесса")
	deliverProc(t, alice, bBundle.GetIdentityX25519Pubkey(), third, bob2, bobConv,
		third.GetText().GetBody())
	if !waitUntil(t, 60*time.Second, func() bool {
		return deliveredProc(t, alice, conv, "msg-proc-3")
	}) {
		t.Fatalf("после перезапуска процесса доставка не подтвердилась: %v", bodiesProc(t, alice, conv))
	}

	// Итог: у Алисы два исходящих сообщения, оба доставлены; у Боба два
	// входящих. Ответ Боба лежит в его собственной исходящей строке и в
	// диалоге Алисы под локальным именем.
	outbox := messagesProc(t, alice, conv)
	if len(outbox) != 2 {
		t.Fatalf("у Алисы в своём диалоге %d строк, ожидалось 2: %v", len(outbox), bodiesProc(t, alice, conv))
	}
	for _, m := range outbox {
		if !m.GetDelivered() {
			t.Fatalf("сообщение %s так и не помечено доставленным", m.GetMessageId())
		}
	}
	if got := bodiesProc(t, alice, aliceConv); len(got) != 1 || got[0] != second.GetText().GetBody() {
		t.Fatalf("ответ Боба в диалоге Алисы поехал: %v", got)
	}
	if got := bodiesProc(t, bob2, bobConv); len(got) != 2 {
		t.Fatalf("у Боба %d входящих строк, ожидалось 2: %v", len(got), got)
	}
}

// pollEvent вычитывает канал до события нужного вида и отдаёт nil по истечении
// окна. Нужен там, где «пока ничего» — рабочий вариант, а не провал теста: у
// FetchContact итог может и не прийти, и тогда запрос повторяют, как это делает
// живой клиент, а не валят прогон.
func (p *nodeProc) pollEvent(limit time.Duration, want func(*yggproto.CoreEvent) bool) *yggproto.CoreEvent {
	deadline := time.After(limit)
	for {
		select {
		case ev := <-p.conn.Events():
			if ev != nil && want(ev) {
				return ev
			}
		case <-deadline:
			return nil
		}
	}
}

// fetchContactProc заводит контакт тем путём, которым его заводит живой
// клиент: UI знает только Ed25519-ключ узла собеседника, зовёт FetchContact, а
// бандл и сертификат ноды приходят по оверлею. Отказ «маршрута нет» здесь —
// норма первых секунд после ConnectPeer, поэтому попытка повторяется, пока
// запрос не принят; итог берётся по request_id, чтобы не принять за ответ
// прошлого запроса его же просрочку.
func fetchContactProc(t *testing.T, p *nodeProc, nodeKey []byte, nickname, wantIPv6 string) *yggproto.ContactCard {
	t.Helper()
	deadline := time.Now().Add(3 * time.Minute)
	var reason string
	for attempts := 1; ; attempts++ {
		resp, err := p.conn.Call(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_FetchContact{
			FetchContact: &yggproto.FetchContactReq{NodeEd25519Pubkey: nodeKey, Nickname: nickname},
		}}, 30*time.Second)
		if err != nil {
			t.Fatalf("%s: FetchContact до процесса не дошёл: %v\nвывод процесса: %s", p.name, err, p.stdout.String())
		}
		if !resp.GetSuccess() {
			reason = "запрос не принят: " + resp.GetErrorMessage()
		} else {
			fr := resp.GetFetchContact()
			if !fr.GetRequested() {
				t.Fatalf("%s: FetchContact ответил requested=false: %+v", p.name, fr)
			}
			if fr.GetNodeIpv6() != wantIPv6 {
				t.Fatalf("%s: адрес узла в ответе %s, в статусе ноды %s", p.name, fr.GetNodeIpv6(), wantIPv6)
			}
			ev := p.pollEvent(45*time.Second, func(ev *yggproto.CoreEvent) bool {
				ce := ev.GetContactUpdated()
				return ce != nil && ce.GetRequestId() == fr.GetRequestId()
			})
			ce := ev.GetContactUpdated()
			switch {
			case ev == nil:
				reason = "итог не пришёл за окно"
			case ce.GetContact() != nil && len(ce.GetContact().GetX25519Pubkey()) == 32:
				return ce.GetContact()
			default:
				reason = "узел не разрешился: " + ce.GetError()
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: контакт по ключу %x не разрешился за %d попыток; последняя причина: %s\nвывод процесса: %s",
				p.name, nodeKey, attempts, reason, p.stdout.String())
		}
	}
}

// sfOf читает участие процесса в S&F так, как его видит клиент: только через
// NodeStatusResp, без доступа к счётчикам движка.
func sfOf(t *testing.T, p *nodeProc) *yggproto.SfStats {
	t.Helper()
	st := p.must(t, &yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetNodeStatus{
		GetNodeStatus: &yggproto.GetNodeStatusReq{},
	}}).GetNodeStatus()
	if st.GetSf() == nil {
		t.Fatalf("%s: в статусе ноды нет раздела S&F", p.name)
	}
	return st.GetSf()
}

// TestContactFetchedOverOverlayInSeparateProcesses — та часть Шага 8, которую
// главный тест сознательно не трогал: там карта собеседника передаётся процессу
// байтами вне сети, а в живой программе контакта по ключу узла добивается
// FetchContact. Проверяется всё, что держится на этом пути между процессами:
// бандл пришёл по оверлею, прошёл ворота verifyContact, лег в зашифрованную
// базу, дал настоящий сквозной диалог и пережил перезапуск того же процесса.
func TestContactFetchedOverOverlayInSeparateProcesses(t *testing.T) {
	if testing.Short() {
		t.Skip("два процесса с живым оверлеем: в -short не запускается")
	}

	root := t.TempDir()
	aliceDir := filepath.Join(root, "alice")
	bobDir := filepath.Join(root, "bob")
	bobPort := freeTCPPort(t)

	alice := launchNodeProc(t, "Алиса")
	bob := launchNodeProc(t, "Боб")

	aSt := startNodeProc(t, alice, seedAlice, aliceDir, nil)
	bSt := startNodeProc(t, bob, seedBob, bobDir, func(cfg *yggproto.NodeConfig) {
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(bobPort)
	})

	alice.must(t, &yggproto.CoreRequest{Request: &yggproto.CoreRequest_ConnectPeer{
		ConnectPeer: &yggproto.ConnectPeerReq{PeerUri: fmt.Sprintf("127.0.0.1:%d", bobPort)},
	}})

	// Единственный вход — ключ узла. Ни бандла, ни адреса Алиса не знает.
	card := fetchContactProc(t, alice, bSt.GetMyEd25519Pubkey(), "Боб по сети", bSt.GetMyIpv6())
	if card.GetIsVerified() {
		t.Fatal("сетевой контакт размечен проверенным: сеть доказала связь ключей, а не человека за ними")
	}
	if card.GetNickname() != "Боб по сети" {
		t.Fatalf("имя контакта приехало чужое: %q", card.GetNickname())
	}
	if !bytes.Equal(card.GetX25519Pubkey(), bSt.GetMyX25519Pubkey()) {
		t.Fatalf("идентичность в карте не ключу ноды: %x против %x", card.GetX25519Pubkey(), bSt.GetMyX25519Pubkey())
	}
	if bundle := card.GetBundle(); bundle == nil || len(bundle.GetOneTimePrekeys()) == 0 {
		t.Fatalf("карта без бандла: X3DH по ней невозможен: %v", bundle)
	}

	bobConv := inboundConversationID(aSt.GetMyX25519Pubkey())
	const conv = "по сети"

	// Карта, добытая по сети, обязана быть годной не «в принципе», а для
	// настоящего сквозного сообщения этому процессу.
	first := procMessage("msg-net-1", conv, "привет от добытого контакта")
	sent := sendProc(t, alice, card.GetX25519Pubkey(), first)
	if !sent.GetWrittenToOverlay() {
		t.Fatalf("сообщение на сетевой контакт не легло в оверлей: %+v", sent)
	}
	deliverProc(t, alice, card.GetX25519Pubkey(), first, bob, bobConv, first.GetText().GetBody())
	if !waitUntil(t, 60*time.Second, func() bool {
		return deliveredProc(t, alice, conv, "msg-net-1")
	}) {
		t.Fatalf("ACK от процесса, добавленного по сети, не дошёл: %v", bodiesProc(t, alice, conv))
	}

	// Перезапуск процесса: карта лежит в базе, а не в памяти, и обязана
	// пережить смену процесса — включая ворота verifyContact на снимке из базы.
	alice.kill(t)
	alice2 := launchNodeProc(t, "Алиса-2")
	startNodeProc(t, alice2, seedAlice, aliceDir, func(cfg *yggproto.NodeConfig) {
		cfg.BootstrapPeers = []string{fmt.Sprintf("127.0.0.1:%d", bobPort)}
	})
	contacts := alice2.must(t, &yggproto.CoreRequest{
		Request: &yggproto.CoreRequest_ListContacts{ListContacts: &yggproto.ListContactsReq{}},
	}).GetContacts().GetContacts()
	if len(contacts) != 1 {
		t.Fatalf("после перезапуска процесса сетевых контактов %d, ожидался 1: %v", len(contacts), contacts)
	}
	restored := contacts[0]
	if restored.GetNickname() != "Боб по сети" || restored.GetIsVerified() {
		t.Fatalf("карта после перезапуска поехала: %q verified=%v", restored.GetNickname(), restored.GetIsVerified())
	}
	if !bytes.Equal(restored.GetX25519Pubkey(), card.GetX25519Pubkey()) {
		t.Fatalf("в базе лежит карта другой идентичности: %x против %x", restored.GetX25519Pubkey(), card.GetX25519Pubkey())
	}

	// Восстановленная карта годится для диалога: бандл пережил запись в
	// зашифрованную базу и обратно, а сеанс — смену процесса.
	second := procMessage("msg-net-2", conv, "после перезапуска процесса")
	deliverProc(t, alice2, restored.GetX25519Pubkey(), second, bob, bobConv, second.GetText().GetBody())
	if !waitUntil(t, 60*time.Second, func() bool {
		return deliveredProc(t, alice2, conv, "msg-net-2")
	}) {
		t.Fatalf("после перезапуска доставка не подтвердилась: %v", bodiesProc(t, alice2, conv))
	}
}

// TestOfflineMailIsHeldByAnotherProcess — store-and-forward между процессами.
// Внутри одного процесса релей и клиенты делят память, а здесь почта лежит в
// зашифрованной базе третьего процесса, адресат забирает её из НОВОГО процесса
// с той же директорией, и расшифровка удаётся только если prekey-материал и
// состояние сеанса действительно пережили смену процесса.
func TestOfflineMailIsHeldByAnotherProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("три процесса с живым оверлеем: в -short не запускается")
	}

	root := t.TempDir()
	relayDir := filepath.Join(root, "relay")
	aliceDir := filepath.Join(root, "alice")
	bobDir := filepath.Join(root, "bob")
	relayPort := freeTCPPort(t)
	relayURI := fmt.Sprintf("127.0.0.1:%d", relayPort)

	relay := launchNodeProc(t, "Хранитель")
	rSt := startNodeProc(t, relay, seedCarol, relayDir, func(cfg *yggproto.NodeConfig) {
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(relayPort)
		cfg.SfRelayEnabled = true
	})
	if !rSt.GetSf().GetRelayEnabled() {
		t.Fatal("хранитель не объявил себя релеем в статусе")
	}
	relayKey := append([]byte(nil), rSt.GetMyEd25519Pubkey()...)

	// Боб существует как идентичность, но его процесса в сети нет: бандл снят,
	// процесс убит, и кладка обязана переждать его отсутствие на диске хранителя.
	bob := launchNodeProc(t, "Боб-офлайн")
	bSt := startNodeProc(t, bob, seedBob, bobDir, func(cfg *yggproto.NodeConfig) {
		cfg.BootstrapPeers = []string{relayURI}
	})
	bBundle := bundleProc(t, bob)
	bob.kill(t)

	alice := launchNodeProc(t, "Алиса")
	aSt := startNodeProc(t, alice, seedAlice, aliceDir, func(cfg *yggproto.NodeConfig) {
		cfg.BootstrapPeers = []string{relayURI}
		cfg.SfRelayNodeKeys = [][]byte{relayKey}
	})
	if got := sfOf(t, alice).GetRelaysConfigured(); got != 1 {
		t.Fatalf("релей не виден в статусе клиента: %d", got)
	}
	addContactProc(t, alice, "Боб", bSt, bBundle)

	const body = "дождётся адресата в другом процессе"
	msg := procMessage("mail-proc-1", "офлайн по сети", body)
	sent := sendProc(t, alice, bBundle.GetIdentityX25519Pubkey(), msg)
	if sent.GetWrittenToOverlay() {
		t.Fatal("письмо «доставлено» адресату, которого нет в сети")
	}
	if !sent.GetQueuedInOutbox() {
		t.Fatalf("офлайновое письмо не легло в исходящую очередь: %+v", sent)
	}

	// Кладка на хранителя — асинхронная (её делает сканер очереди), и «передано
	// оверлею» не значит «хранится»: хранение подтверждает только счётчик самого
	// хранителя.
	if !waitUntil(t, 90*time.Second, func() bool {
		return sfOf(t, relay).GetStored() == 1 && sfOf(t, alice).GetPushed() >= 1
	}) {
		t.Fatalf("хранитель не держит письмо: он %+v, отправитель %+v", sfOf(t, relay), sfOf(t, alice))
	}

	bob2 := launchNodeProc(t, "Боб-онлайн")
	startNodeProc(t, bob2, seedBob, bobDir, func(cfg *yggproto.NodeConfig) {
		cfg.BootstrapPeers = []string{relayURI}
		cfg.SfRelayNodeKeys = [][]byte{relayKey}
	})
	// Диалог у адресата именуется по идентичности отправителя, а не по имени
	// диалога из конверта: имя выбрал отправитель, и адресату оно не принадлежит.
	conv := inboundConversationID(aSt.GetMyX25519Pubkey())

	// Запрос почты повторяется: пакет запроса — первый на ещё не прогретом
	// маршруте, и оверлей такие теряет, а хранитель по неудачной попытке ничего
	// не списывает. «Ноль запрошенных ящиков» здесь — тоже штатный ответ:
	// collectFromRelays идёт только к тем релеям, чей адрес уже известен
	// spanning-дереву, а оно узнаёт его на секунду-другую позже StartNode.
	deadline := time.Now().Add(2 * time.Minute)
	for attempts := 1; ; attempts++ {
		collect := bob2.must(t, &yggproto.CoreRequest{Request: &yggproto.CoreRequest_SfCollect{
			SfCollect: &yggproto.SfCollectReq{},
		}}).GetSfCollect()
		if collect.GetRequested() > 0 && waitUntil(t, 5*time.Second, func() bool {
			return hasBodyProc(t, bob2, conv, body)
		}) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("за %d попыток почта не открылась у адресата в новом процессе: запрос %+v, он %+v, хранитель %+v, тела %v\nвывод процессов: %s / %s",
				attempts, collect, sfOf(t, bob2), sfOf(t, relay), bodiesProc(t, bob2, conv),
				bob2.stdout.String(), relay.stdout.String())
		}
	}

	if got := sfOf(t, bob2); got.GetPulled() == 0 {
		t.Fatalf("забранное не посчитано: %+v", got)
	}
	if got := sfOf(t, relay); got.GetHandedOut() == 0 || got.GetRefused() != 0 {
		t.Fatalf("выдача посчитана неверно: %+v", got)
	}
	// Выданное письмо остаётся на хранении до окна повторной выдачи: пакет
	// ответа идёт тем же оверлеем и может потеряться.
	if got := sfOf(t, relay); got.GetStored() != 1 {
		t.Fatalf("хранитель списал почту по первой выдаче: %+v", got)
	}
}

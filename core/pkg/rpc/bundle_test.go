// File: core/pkg/rpc/bundle_test.go
package rpc

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
	"time"

	yggproto "ygggram/core/proto"
)

// Обмен PreKeyBundle поверх оверлея (Шаг 7). Проверяются обе стороны и ровно то,
// на чём такой обмен ломается: ответ обязан прийти от того же узла, у которого
// спрашивали, карта — пройти те же ворота, что и AddContact, а два бандла,
// розданные разным людям, — нести разные one-time prekey.

func bundleStatsOf(t *testing.T, e *Engine) bundleStats {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.bundleStats
}

// opkSnapshot — состояние склада one-time prekey под блокировкой: пул растёт и
// сжимается из приёмного цикла, читать его без lock'а было бы гонкой.
func opkSnapshot(t *testing.T, e *Engine) (published uint32, held int) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.opkID, len(e.opkPriv)
}

// nextResolved ждёт событие contact_updated. Ожидание с предельным сроком, а не
// навсегда: канал событий в тесте никто не вычитывает, и потерянное событие
// должно уронить тест, а не зависить его.
func nextResolved(t *testing.T, e *Engine, within time.Duration) *yggproto.ContactResolvedEvent {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case ev := <-e.Events():
			if resolved := ev.GetContactUpdated(); resolved != nil {
				return resolved
			}
		case <-deadline:
			t.Fatalf("за %s события contact_updated не было", within)
			return nil
		}
	}
}

// pendingFetch регистрирует незакрытый запрос руками: ворота ответа проверяются
// на точных данных, а не на гонке с настоящим сокетом.
func pendingFetch(t *testing.T, e *Engine, id uint64, nodeKey ed25519.PublicKey, nickname string) {
	t.Helper()
	frame, err := bundleReqFrame(&yggproto.BundleReq{RequestId: id})
	if err != nil {
		t.Fatal(err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if id > e.bundleSeq {
		e.bundleSeq = id
	}
	e.bundlePending[id] = &bundleFetch{
		nodeKey:  append([]byte(nil), nodeKey...),
		nickname: nickname,
		frame:    frame,
		sentAt:   time.Now(),
		sends:    1,
	}
}

func respFrameOf(t *testing.T, id uint64, bundle *yggproto.PreKeyBundle, errMsg string) []byte {
	t.Helper()
	return frameOf(t, &yggproto.WireFrame{Body: &yggproto.WireFrame_BundleResp{
		BundleResp: &yggproto.BundleResp{RequestId: id, Bundle: bundle, Error: errMsg},
	}})
}

func cardCount(t *testing.T, e *Engine) int {
	t.Helper()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_ListContacts{}})
	if !resp.GetSuccess() {
		t.Fatalf("ListContacts: %s", resp.GetErrorMessage())
	}
	return len(resp.GetContacts().GetContacts())
}

// addContactFrom — карта из конкретного бандла: тесту важно, каким именно
// one-time prekey собрали сессию, а не каким он оказался на момент вызова.
func addContactFrom(t *testing.T, from, to *Engine, bundle *yggproto.PreKeyBundle) {
	t.Helper()
	resp := from.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			NodeIpv6:      bundleNodeIPv6(to),
			Ed25519Pubkey: to.node.PublicKey(),
			X25519Pubkey:  bundle.GetIdentityX25519Pubkey(),
			Bundle:        bundle,
		}},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("AddContact: %s", resp.GetErrorMessage())
	}
}

// TestFetchContactCommandRejectsBadInput — команда отказывает до сети на каждом
// из четырёх сломанных входов: ядро не запущено, ключ не 32 байта, ключ свой,
// маршрута нет.
func TestFetchContactCommandRejectsBadInput(t *testing.T) {
	cold := New()
	resp := cold.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_FetchContact{
		FetchContact: &yggproto.FetchContactReq{NodeEd25519Pubkey: make([]byte, ed25519.PublicKeySize)},
	}})
	if resp.GetSuccess() || !strings.Contains(resp.GetErrorMessage(), "StartNode") {
		t.Fatalf("FetchContact без запущенного ядра прошёл: %q", resp.GetErrorMessage())
	}

	e := startEngine(t, seedAlice, t.TempDir())
	handle := func(key []byte) *yggproto.CoreResponse {
		return e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_FetchContact{
			FetchContact: &yggproto.FetchContactReq{NodeEd25519Pubkey: key},
		}})
	}
	if r := handle(make([]byte, 31)); r.GetSuccess() || !strings.Contains(r.GetErrorMessage(), "32 байта") {
		t.Fatalf("кривой ключ ноды принят: %q", r.GetErrorMessage())
	}
	if r := handle(e.node.PublicKey()); r.GetSuccess() || !strings.Contains(r.GetErrorMessage(), "самому узлу") {
		t.Fatalf("запрос собственного бандла прошёл: %q", r.GetErrorMessage())
	}

	_, stranger, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	r := handle(stranger.Public().(ed25519.PublicKey))
	if r.GetSuccess() || !strings.Contains(r.GetErrorMessage(), "маршрута") {
		t.Fatalf("запрос без маршрута обязан называть причину: %q", r.GetErrorMessage())
	}
	if st := bundleStatsOf(t, e); st.requested != 0 {
		t.Fatalf("отклонённый запрос посчитан отправленным: %+v", st)
	}
	e.mu.Lock()
	pending := len(e.bundlePending)
	e.mu.Unlock()
	if pending != 0 {
		t.Fatalf("отклонённый запрос оставил ожидание: %d", pending)
	}
}

// TestBundleRespGates — ответ проверяется поворотами, которые сеть умеет
// подставлять: чужой узел, чужой бандл под честным ключом, отказ словами и
// опоздавший дубликат.
func TestBundleRespGates(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	carol := startEngine(t, seedCarol, t.TempDir())

	bobKey := bob.node.PublicKey()
	honest := bundleOf(t, bob)
	foreign := bundleOf(t, carol)
	const id = uint64(77)

	// 1. Ответ на запрос, которого нет: дубликат после истечения окна.
	pendingFetch(t, alice, id, bobKey, "Боб")
	inject(t, alice, bobKey, respFrameOf(t, id+1, honest, ""))
	if st := bundleStatsOf(t, alice); st.unmatched != 1 || st.resolved != 0 {
		t.Fatalf("ответ на чужой request_id принят: %+v", st)
	}
	if got := cardCount(t, alice); got != 0 {
		t.Fatalf("несовпадающий ответ завёл контакт: %d карт", got)
	}

	// 2. Отказ словами — это отказ, а не «контакт добавлен».
	inject(t, alice, bobKey, respFrameOf(t, id, nil, "узел не раздаёт PreKeyBundle"))
	if st := bundleStatsOf(t, alice); st.failed != 1 || st.resolved != 0 {
		t.Fatalf("отказ посчитан иначе: %+v", st)
	}
	if got := cardCount(t, alice); got != 0 {
		t.Fatalf("после отказа в базе %d карт", got)
	}

	// 3. Честный бандл, но принёс его другой узел. Без этой проверки вопрос
	// «у кого спрашивали» повисает: криптография бандла отвечает на «чья это
	// идентичность», а не на «этот ли узел мне ответил».
	pendingFetch(t, alice, id, bobKey, "Боб")
	inject(t, alice, carol.node.PublicKey(), respFrameOf(t, id, honest, ""))
	if st := bundleStatsOf(t, alice); st.failed != 2 || st.resolved != 0 {
		t.Fatalf("ответ от чужого узла принят в работу: %+v", st)
	}

	// 4. Идентичность Каролы под заверенным ключом Боба: ворота verifyContact.
	pendingFetch(t, alice, id, bobKey, "Боб")
	tampered := newBundleOf(honest)
	tampered.IdentityEd25519Pubkey = foreign.GetIdentityEd25519Pubkey()
	tampered.IdentityX25519Pubkey = foreign.GetIdentityX25519Pubkey()
	tampered.SignedPrekey = foreign.GetSignedPrekey()
	tampered.SignedPrekeySignature = foreign.GetSignedPrekeySignature()
	inject(t, alice, bobKey, respFrameOf(t, id, tampered, ""))
	if st := bundleStatsOf(t, alice); st.failed != 3 || st.resolved != 0 {
		t.Fatalf("перетёртый бандл прошёл ворота: %+v", st)
	}
	if got := cardCount(t, alice); got != 0 {
		t.Fatalf("перетёртый бандл записан в базу: %d карт", got)
	}

	// 5. Пустой бандл без ошибки.
	pendingFetch(t, alice, id, bobKey, "Боб")
	inject(t, alice, bobKey, respFrameOf(t, id, nil, ""))
	if st := bundleStatsOf(t, alice); st.failed != 4 {
		t.Fatalf("пустой бандл посчитан иначе: %+v", st)
	}

	// 6. Честный ответ: карта проходит ворота, сохраняется и показывается.
	pendingFetch(t, alice, id, bobKey, "Боб")
	inject(t, alice, bobKey, respFrameOf(t, id, honest, ""))
	if st := bundleStatsOf(t, alice); st.resolved != 1 || st.failed != 4 {
		t.Fatalf("честный ответ не принят: %+v", st)
	}
	if got := cardCount(t, alice); got != 1 {
		t.Fatalf("честный ответ дал %d карт, ожидалась одна", got)
	}
	// Дубликат того же ответа: запрос уже закрыт, значит карта не раздвоится.
	inject(t, alice, bobKey, respFrameOf(t, id, honest, ""))
	if st := bundleStatsOf(t, alice); st.unmatched != 2 || st.resolved != 1 {
		t.Fatalf("дубликат ответа закрылся повторно: %+v", st)
	}
	if got := cardCount(t, alice); got != 1 {
		t.Fatalf("дубликат ответа добавил карту: %d", got)
	}
}

// TestBundleRespCardCarriesLocalNicknameAndAddress — имя выбирает тот, кто
// добавляет, адрес выводится из запрошенного ключа, а флаг «проверен» сам
// собой не вспыхивает.
func TestBundleRespCardCarriesLocalNicknameAndAddress(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	bundle := bundleOf(t, bob)
	bobKey := bob.node.PublicKey()

	pendingFetch(t, alice, 5, bobKey, "Сосед по Wi-Fi")
	inject(t, alice, bobKey, respFrameOf(t, 5, bundle, ""))

	resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_ListContacts{}})
	cards := resp.GetContacts().GetContacts()
	if len(cards) != 1 {
		t.Fatalf("карт %d, ожидалась одна: %s", len(cards), resp.GetErrorMessage())
	}
	card := cards[0]
	if card.GetNickname() != "Сосед по Wi-Fi" {
		t.Fatalf("имя контакта потерялось: %q", card.GetNickname())
	}
	if card.GetNodeIpv6() != bundleNodeIPv6(bob) {
		t.Fatalf("адрес контакта не выведен из ключа: %s", card.GetNodeIpv6())
	}
	if card.GetIsVerified() {
		t.Fatal("сетевой fetch не подтверждает личность — флаг не должен ставиться сам")
	}
	if !bytes.Equal(card.GetEd25519Pubkey(), bobKey) {
		t.Fatalf("ключ ноды в карте другой: %x", card.GetEd25519Pubkey())
	}
	if card.GetX25519Pubkey() == nil {
		t.Fatal("в карте нет identity-X25519 ключа: сессии не на чем строить")
	}
}

// TestBundleServeRefusesAndRateLimits — раздача выключена и частота: отказ
// назван, prekey-склад при этом не трогается.
func TestBundleServeRefusesAndRateLimits(t *testing.T) {
	off := startEngineWith(t, seedCarol, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		disabled := false
		cfg.ServePrekeyBundles = &disabled
	})
	alice := startEngine(t, seedAlice, t.TempDir())
	req := frameOf(t, &yggproto.WireFrame{Body: &yggproto.WireFrame_BundleReq{
		BundleReq: &yggproto.BundleReq{RequestId: 1},
	}})

	beforeID, beforeHeld := opkSnapshot(t, off)
	inject(t, off, alice.node.PublicKey(), req)
	st := bundleStatsOf(t, off)
	if st.serves != 0 || st.refused != 1 {
		t.Fatalf("выключенная раздача отдала бандл: %+v", st)
	}
	if id, held := opkSnapshot(t, off); id != beforeID || held != beforeHeld {
		t.Fatalf("отказ сдвинул prekey-материал: было %d/%d, стало %d/%d", beforeID, beforeHeld, id, held)
	}

	bob := startEngine(t, seedBob, t.TempDir())
	published, _ := opkSnapshot(t, bob)
	inject(t, bob, alice.node.PublicKey(), req)
	if st := bundleStatsOf(t, bob); st.serves != 1 || st.refused != 0 {
		t.Fatalf("первый запрос бандла не обслужен: %+v", st)
	}
	if id, _ := opkSnapshot(t, bob); id != published+1 {
		t.Fatalf("раздача не сдвинула опубликованный one-time prekey: было %d, стало %d", published, id)
	}
	// Тот же узел повторно, не дожидаясь интервала.
	inject(t, bob, alice.node.PublicKey(), req)
	if st := bundleStatsOf(t, bob); st.serves != 1 || st.refused != 1 {
		t.Fatalf("частоту раздачи не ограничили: %+v", st)
	}
	// Другой узел спрашивает — ему ответ положен сразу.
	dave := startEngine(t, seedDave, t.TempDir())
	inject(t, bob, dave.node.PublicKey(), req)
	if st := bundleStatsOf(t, bob); st.serves != 2 {
		t.Fatalf("честный второй спрашивающий получил отказ: %+v", st)
	}
}

// TestPrekeyHandoutIsOneTimePerPeer — два бандла, розданные разным людям, несут
// разные one-time prekey, и оба первых сообщения открываются. Иначе «одноразовый»
// ключ означал бы, что второй отправитель собрал тот же X3DH-корень.
func TestPrekeyHandoutIsOneTimePerPeer(t *testing.T) {
	bob := startEngine(t, seedBob, t.TempDir())
	alice := startEngine(t, seedAlice, t.TempDir())
	carol := startEngine(t, seedCarol, t.TempDir())

	first := bundleOf(t, bob)
	second := bundleOf(t, bob)
	if first.GetKeyId() == second.GetKeyId() {
		t.Fatalf("разданы два одинаковых one-time prekey: id=%d", first.GetKeyId())
	}
	if bytes.Equal(first.GetOneTimePrekeys()[0], second.GetOneTimePrekeys()[0]) {
		t.Fatal("байты выданного one-time prekey совпали")
	}

	addContactFrom(t, alice, bob, first)
	addContactFrom(t, carol, bob, second)

	envA := wireOf(t, alice, bob, first, &yggproto.ChatMessage{
		MessageId: "handout-a", ConversationId: "диалог А", TimestampMs: 1700000000000,
		Content: textContent("письмо А"),
	})
	envB := wireOf(t, carol, bob, second, &yggproto.ChatMessage{
		MessageId: "handout-b", ConversationId: "диалог Б", TimestampMs: 1700000000000,
		Content: textContent("письмо Б"),
	})

	inject(t, bob, alice.node.PublicKey(), frameOf(t, &yggproto.WireFrame{
		Body: &yggproto.WireFrame_Chat{Chat: envA},
	}))
	inject(t, bob, carol.node.PublicKey(), frameOf(t, &yggproto.WireFrame{
		Body: &yggproto.WireFrame_Chat{Chat: envB},
	}))

	if st := statsOf(t, bob); st.rejected != 0 || st.applied != 2 {
		t.Fatalf("оба первых сообщения не открылись: %+v", st)
	}
	aliceX := alice.master.MasterX25519.PublicKey().Bytes()
	carolX := carol.master.MasterX25519.PublicKey().Bytes()
	if got := storedBodies(t, bob, inboundConversationID(aliceX)); len(got) != 1 || got[0] != "письмо А" {
		t.Fatalf("диалог Алисы: %v", got)
	}
	if got := storedBodies(t, bob, inboundConversationID(carolX)); len(got) != 1 || got[0] != "письмо Б" {
		t.Fatalf("диалог Каролы: %v", got)
	}

	// Склад ограничен: сколько бы бандлов ни раздавали, приватных ключей не
	// становится больше договорённого пула.
	for i := 0; i < 4*maxHandedOPKs; i++ {
		_ = bob.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetPrekeyBundle{}})
	}
	if _, held := opkSnapshot(t, bob); held > maxKeptOPKs+maxHandedOPKs {
		t.Fatalf("склад one-time prekey разросся: %d ключей при пуле %d", held, maxKeptOPKs+maxHandedOPKs)
	}
}

// TestExpireBundleFetchesTimesOutAndKeepsExhausted — запрос без ответа обязан
// захлопнуться с причиной, а израсходовавший повторы — дожить до срока: раньше
// времени объявить узла молчащим нельзя.
func TestExpireBundleFetchesTimesOutAndKeepsExhausted(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	pendingFetch(t, alice, 11, bob.node.PublicKey(), "Боб")

	alice.mu.Lock()
	alice.bundlePending[11].sentAt = time.Now().Add(-bundleFetchTimeout - time.Second)
	alice.expireBundleFetches(time.Now())
	pendingLeft := len(alice.bundlePending)
	st := alice.bundleStats
	alice.mu.Unlock()
	if pendingLeft != 0 || st.expired != 1 {
		t.Fatalf("просроченный запрос не закрыт: ожиданий %d, %+v", pendingLeft, st)
	}
	resolved := nextResolved(t, alice, 2*time.Second)
	if resolved.GetContact() != nil || !strings.Contains(resolved.GetError(), "не ответил") {
		t.Fatalf("таймаут назван неверно: %q", resolved.GetError())
	}
	if resolved.GetRequestId() != 11 {
		t.Fatalf("событие не про тот запрос: %d", resolved.GetRequestId())
	}

	// Повторы исчерпаны, срок ещё не вышел — запрос живёт дальше.
	pendingFetch(t, alice, 12, bob.node.PublicKey(), "Боб")
	alice.mu.Lock()
	alice.bundlePending[12].sends = bundleMaxSends
	alice.bundlePending[12].sentAt = time.Now().Add(-bundleRetryInterval)
	alice.expireBundleFetches(time.Now())
	pendingLeft = len(alice.bundlePending)
	st = alice.bundleStats
	alice.mu.Unlock()
	if pendingLeft != 1 || st.expired != 1 {
		t.Fatalf("запрос с исчерпанными повторами сброшен раньше срока: ожиданий %d, %+v", pendingLeft, st)
	}
}

// TestFetchContactOverRealOverlay — настоящий сценарий: узлы связаны TCP, контакта
// в базе нет, байтов вне сети не было. После FetchContact диалог открывается, а
// письмо попадает к адресату.
func TestFetchContactOverRealOverlay(t *testing.T) {
	port := freeTCPPort(t)
	alice := startEngineWith(t, seedAlice, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(port)
	})
	bob := startEngineWith(t, seedBob, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.BootstrapPeers = []string{fmt.Sprintf("127.0.0.1:%d", port)}
	})
	waitForRoute(t, alice, bob.node.PublicKey())
	waitForRoute(t, bob, alice.node.PublicKey())

	if got := cardCount(t, alice); got != 0 {
		t.Fatalf("до fetch у Алисы уже %d контактов", got)
	}

	// Первый пакет на холодном маршруте теряется: ядро повторяет запрос само,
	// а тест ждёт событие с запасом и при необходимости переспрашивает.
	deadline := time.Now().Add(60 * time.Second)
	var resolved *yggproto.ContactResolvedEvent
	for attempts := 1; ; attempts++ {
		resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_FetchContact{
			FetchContact: &yggproto.FetchContactReq{
				NodeEd25519Pubkey: bob.node.PublicKey(),
				Nickname:          "Боб по ту сторону Wi-Fi",
			},
		}})
		if !resp.GetSuccess() {
			t.Fatalf("FetchContact #%d: %s", attempts, resp.GetErrorMessage())
		}
		if !resp.GetFetchContact().GetRequested() {
			t.Fatalf("FetchContact #%d вернул requested=false", attempts)
		}
		timeout := 12 * time.Second
		if left := time.Until(deadline); left < timeout {
			timeout = left
		}
		resolved = nextResolved(t, alice, timeout)
		if resolved.GetContact() != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("за %d попыток бандл не получен: %+v, у Боба %+v, карт %d",
				attempts, resolved, bundleStatsOf(t, bob), cardCount(t, alice))
		}
	}

	card := resolved.GetContact()
	if card.GetNickname() != "Боб по ту сторону Wi-Fi" {
		t.Fatalf("имя не дошло: %q", card.GetNickname())
	}
	if card.GetNodeIpv6() != bundleNodeIPv6(bob) {
		t.Fatalf("адрес в карте чужой: %s", card.GetNodeIpv6())
	}
	if st := bundleStatsOf(t, bob); st.serves == 0 || st.refused != 0 {
		t.Fatalf("Боб не раздавал бандл: %+v", st)
	}
	if st := bundleStatsOf(t, alice); st.resolved == 0 {
		t.Fatalf("fetch не засчитан: %+v", st)
	}

	// Диалог обязан открыться на добытом бандле: без этого шаг бессмыслен.
	// Имя строки у получателя — его собственное (по identity key отправителя),
	// а не conversation_id Алисы.
	body := "привет без бумажки"
	conv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	sendDeadline := time.Now().Add(60 * time.Second)
	for attempts := 1; ; attempts++ {
		resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
			SendMessage: &yggproto.SendMessageReq{
				DestinationX25519Pubkey: card.GetX25519Pubkey(),
				Message: &yggproto.ChatMessage{
					MessageId:      fmt.Sprintf("fetch-%d", attempts),
					ConversationId: "fetch-overlay",
					TimestampMs:    1700000000000,
					Content:        textContent(body),
				},
			},
		}})
		if !resp.GetSuccess() {
			t.Fatalf("SendMessage #%d: %s", attempts, resp.GetErrorMessage())
		}
		if waitUntil(t, 3*time.Second, func() bool { return hasBody(t, bob, conv, body) }) {
			break
		}
		if time.Now().After(sendDeadline) {
			t.Fatalf("за %d попыток письмо не дошло: приём %+v, база %v",
				attempts, statsOf(t, bob), storedBodies(t, bob, conv))
		}
	}
	if st := statsOf(t, bob); st.rejected != 0 {
		t.Fatalf("на принятом бандле есть отказы раскрытия: %+v", st)
	}
}

// newBundleOf — копия бандла, чтобы тест портил ответ, не портя собственное
// состояние ядер. Поле за полем: protobuf-сообщение копированием значением
// переносит внутренний мьютекс.
func newBundleOf(b *yggproto.PreKeyBundle) *yggproto.PreKeyBundle {
	return &yggproto.PreKeyBundle{
		IdentityEd25519Pubkey: append([]byte(nil), b.GetIdentityEd25519Pubkey()...),
		IdentityX25519Pubkey:  append([]byte(nil), b.GetIdentityX25519Pubkey()...),
		SignedPrekey:          append([]byte(nil), b.GetSignedPrekey()...),
		SignedPrekeySignature: append([]byte(nil), b.GetSignedPrekeySignature()...),
		OneTimePrekeys:        append([][]byte(nil), b.GetOneTimePrekeys()...),
		PrekeyTimestamp:       b.GetPrekeyTimestamp(),
		KeyId:                 b.GetKeyId(),
		NodeEd25519Pubkey:     append([]byte(nil), b.GetNodeEd25519Pubkey()...),
		NodeKeyProof:          append([]byte(nil), b.GetNodeKeyProof()...),
	}
}

// File: core/pkg/rpc/sf_test.go
package rpc

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"ygggram/core/pkg/crypto"
	"ygggram/core/pkg/storage"
	"ygggram/core/pkg/yggnode"
	yggproto "ygggram/core/proto"
)

// Слепой store-and-forward проверен с обеих сторон: как хранитель узел принимает
// чужие конверты, не узнавая о них ничего, и как клиент забирает свои. Отдельно
// проверено то, на чём такие схемы ломаются чаще всего, — что отказ назван
// причиной, а пустой ящик не притворяется отказом.

func sfStatsOf(t *testing.T, e *Engine) sfStats {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sfStats
}

// sfStored — сколько чужих писем лежит на узле прямо сейчас.
func sfStored(t *testing.T, e *Engine) int {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.db == nil {
		t.Fatal("хранилище не открыто")
	}
	return e.db.SFStoredTotal()
}

// sfStatusOf читает S&F-статистику так, как её видит приложение: через
// GetNodeStatus, а не через внутреннюю структуру движка.
func sfStatusOf(t *testing.T, e *Engine) *yggproto.SfStats {
	t.Helper()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetNodeStatus{}})
	if !resp.GetSuccess() {
		t.Fatalf("GetNodeStatus: %s", resp.GetErrorMessage())
	}
	st := resp.GetNodeStatus().GetSf()
	if st == nil {
		t.Fatal("статус ядра без S&F-счётчиков")
	}
	return st
}

// inject — пакет, пришедший с узла from: ровно тот путь, которым живые байты
// попадают в приёмный цикл.
func inject(t *testing.T, e *Engine, from ed25519.PublicKey, frame []byte) {
	t.Helper()
	e.handleInbound(yggnode.Packet{From: from, Payload: frame})
}

// frameOf упаковывает кадр и тут же проверяет, что он читается тем же
// разбором, которым его увидит приёмный цикл.
func frameOf(t *testing.T, frame *yggproto.WireFrame) []byte {
	t.Helper()
	var (
		out []byte
		err error
	)
	switch body := frame.Body.(type) {
	case *yggproto.WireFrame_Chat:
		out, err = chatFrame(body.Chat)
	case *yggproto.WireFrame_SfPut:
		out, err = sfPutFrame(body.SfPut)
	case *yggproto.WireFrame_SfPull:
		out, err = sfPullFrame(body.SfPull)
	case *yggproto.WireFrame_SfReply:
		out, err = sfReplyFrame(body.SfReply)
	case *yggproto.WireFrame_BundleReq:
		out, err = bundleReqFrame(body.BundleReq)
	case *yggproto.WireFrame_BundleResp:
		out, err = bundleRespFrame(body.BundleResp)
	default:
		t.Fatalf("неизвестный вариант кадра: %v", frame.Body)
	}
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := deframe(out)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(parsed, frame) {
		t.Fatalf("кадр разошёлся сам с собой: %v", parsed.Body)
	}
	return out
}

// letterTo — настоящий конверт в ящик получателя: blind_token, работу и ключ
// адресата считает само ядро, а не тест.
func letterTo(t *testing.T, from, to *Engine, messageID string) *yggproto.SealedEnvelope {
	t.Helper()
	bundle := registerContact(t, from, to)
	return wireOf(t, from, to, bundle, &yggproto.ChatMessage{
		MessageId:      messageID,
		ConversationId: "слепой диалог",
		TimestampMs:    1700000000000,
		Content:        textContent("письмо в ящик: " + messageID),
	})
}

func putOf(env *yggproto.SealedEnvelope) *yggproto.WireFrame {
	return &yggproto.WireFrame{Body: &yggproto.WireFrame_SfPut{
		SfPut: &yggproto.SfPut{MailboxId: env.GetBlindToken(), Envelope: env},
	}}
}

// TestStartNodeRejectsMalformedRelayKey — кривой ключ релея не может быть
// молча проигнорирован: «релеи настроены, а почта никуда не кладётся»
// пользователь заметит только по недоставленному сообщению.
func TestStartNodeRejectsMalformedRelayKey(t *testing.T) {
	e := New()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StartNode{
		StartNode: &yggproto.StartNodeReq{Config: &yggproto.NodeConfig{
			DataDir:         t.TempDir() + "/data",
			CacheDir:        t.TempDir() + "/cache",
			MnemonicSeed:    seedBob,
			AntiDpiMode:     yggproto.AntiDpiMode_ANTI_DPI_DISABLED,
			SfRelayNodeKeys: [][]byte{bytes.Repeat([]byte("k"), 31)},
		}},
	}})
	if resp.GetSuccess() {
		t.Fatal("StartNode принял ключ релея неверной длины")
	}
	if msg := resp.GetErrorMessage(); !strings.Contains(msg, "sf_relay_node_keys") {
		t.Fatalf("ошибка не называет поле: %q", msg)
	}
}

// TestSfRelayKeysDeduplicates — дубль одного ключа удвоил бы и кладку, и
// счётчик «передано релеям»: пользователь прочитал бы две копии там, где релей
// один.
func TestSfRelayKeysDeduplicates(t *testing.T) {
	key := bytes.Repeat([]byte("r"), ed25519.PublicKeySize)
	keys, err := sfRelayKeys(&yggproto.NodeConfig{SfRelayNodeKeys: [][]byte{key, key, key}})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("дубликат ключа не схлопнулся: %d релея", len(keys))
	}
	// Правка входных байтов не должна иметь права переписать конфиг ядра.
	key[0] = 'x'
	if keys[0][0] == 'x' {
		t.Fatal("sfRelayKeys отдаёт ссылку на входные байты")
	}
}

// TestRelayStoresBlindPut — годная кладка ложится в ящик, и хранитель не
// узнаёт о её содержимом ничего. Повтор тех же байт не удваивает ни хранение,
// ни счётчик.
func TestRelayStoresBlindPut(t *testing.T) {
	relay := startEngineWith(t, seedCarol, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.SfRelayEnabled = true
	})
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())

	env := letterTo(t, alice, bob, "sf-put-1")
	frame := frameOf(t, putOf(env))

	inject(t, relay, alice.node.PublicKey(), frame)
	st := sfStatsOf(t, relay)
	if st.stored != 1 || st.refused != 0 {
		t.Fatalf("кладка посчитана неверно: %+v", st)
	}
	if got := sfStored(t, relay); got != 1 {
		t.Fatalf("на хранении %d писем, ожидалось 1", got)
	}
	// Хранитель не читает начинку: у него нет ни строк диалога, ни открытого
	// текста.
	if bodies := storedBodies(t, relay, inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())); len(bodies) != 0 {
		t.Fatalf("релей раскрыл чужое сообщение: %v", bodies)
	}

	inject(t, relay, alice.node.PublicKey(), frame)
	if got := sfStatsOf(t, relay); got.stored != 1 || got.refused != 0 {
		t.Fatalf("повторная кладка посчитана новым письмом: %+v", got)
	}
	if got := sfStored(t, relay); got != 1 {
		t.Fatalf("повторная кладка удвоила хранение: %d", got)
	}

	status := sfStatusOf(t, relay)
	if !status.GetRelayEnabled() || status.GetStored() != 1 || status.GetRefused() != 0 {
		t.Fatalf("статус релея расходится со счётчиками: %+v", status)
	}
}

// TestRelayRefusesBadPutShapes — каждый отказ назван причиной, и ни один
// негодный конверт не доходит до хранилища. «Отказ» и «тишина» для релея —
// разные новости.
func TestRelayRefusesBadPutShapes(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	env := letterTo(t, alice, bob, "sf-refuse-1")
	mailbox, ok := mailboxArray(env.GetBlindToken())
	if !ok {
		t.Fatal("blind_token конверта не 32 байта")
	}

	// 1. Узел не вызывался быть релеем.
	off := startEngine(t, seedDave, t.TempDir())
	inject(t, off, alice.node.PublicKey(), frameOf(t, putOf(env)))
	if st := sfStatsOf(t, off); st.stored != 0 || st.refused != 1 || st.lastRefusal == "" {
		t.Fatalf("кладка принята узлом без роли релея: %+v", st)
	}
	if got := sfStored(t, off); got != 0 {
		t.Fatalf("узел без роли хранит чужое: %d", got)
	}

	relay := startEngineWith(t, seedCarol, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.SfRelayEnabled = true
	})

	// 2. Индекс кладки не совпал с blind_token: адресат такое письмо не найдёт.
	nextEpoch := crypto.ComputeMailboxID(bob.master.MasterX25519.PublicKey().Bytes(), sfEpochDay(time.Now())+1)
	inject(t, relay, alice.node.PublicKey(), frameOf(t, &yggproto.WireFrame{Body: &yggproto.WireFrame_SfPut{
		SfPut: &yggproto.SfPut{MailboxId: nextEpoch[:], Envelope: env},
	}}))

	// 3. Работа обнулена — спам-кандидат.
	noPow := proto.Clone(env).(*yggproto.SealedEnvelope)
	noPow.PowNonce = 0
	inject(t, relay, alice.node.PublicKey(), frameOf(t, putOf(noPow)))

	// 4. Неполный конверт: без ключа адресата письмо некому адресовать.
	headless := &yggproto.SealedEnvelope{
		EncryptedPayload: env.GetEncryptedPayload(),
		BlindToken:       env.GetBlindToken(),
		PowNonce:         env.GetPowNonce(),
	}
	inject(t, relay, alice.node.PublicKey(), frameOf(t, putOf(headless)))

	// 5. Ящик больше квоты по байтам: отказ, а не обрезанная запись.
	inject(t, relay, alice.node.PublicKey(), frameOf(t, putOf(&yggproto.SealedEnvelope{
		RecipientX25519Pubkey: env.GetRecipientX25519Pubkey(),
		EncryptedPayload:      bytes.Repeat([]byte("x"), int(sfMaxBytesPerMailbox)+1024),
		BlindToken:            env.GetBlindToken(),
		PowNonce:              crypto.ComputeHashcashPoW(mailbox, powDifficulty),
	})))

	st := sfStatsOf(t, relay)
	if st.stored != 0 {
		t.Fatalf("негодная кладка легла в хранилище: %+v", st)
	}
	if st.refused != 4 {
		t.Fatalf("отказов %d, ожидалось 4: %+v", st.refused, st)
	}
	if st.lastRefusal == "" {
		t.Fatal("отказ без причины")
	}
	if got := sfStored(t, relay); got != 0 {
		t.Fatalf("после отказов на хранении %d писем", got)
	}
}

// TestRelayRefusesPullFromForeignOwner — запрашивающий доказал свою личность
// подписью и связкой ключей, но назвал чужой ящик. Без пересчёта ящика из ключа
// подписанта достаточно было бы знать открытый ключ человека, чтобы забрать его
// почту.
func TestRelayRefusesPullFromForeignOwner(t *testing.T) {
	relay := startEngineWith(t, seedCarol, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.SfRelayEnabled = true
	})
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	env := letterTo(t, alice, bob, "sf-steal-1")
	inject(t, relay, alice.node.PublicKey(), frameOf(t, putOf(env)))
	if got := sfStatsOf(t, relay); got.stored != 1 {
		t.Fatalf("письмо не легло: %+v", got)
	}

	epoch := sfEpochDay(time.Now())
	bobsMailbox := crypto.ComputeMailboxID(bob.master.MasterX25519.PublicKey().Bytes(), epoch)
	ts := time.Now().Unix()

	// Алиса честно подписывает запрос своим мастер-Ed25519 и приносит свою
	// связку — но ящик считает по ключу Боба.
	sig, err := crypto.BuildSFPullSignature(alice.master, bobsMailbox, epoch, ts)
	if err != nil {
		t.Fatal(err)
	}
	inject(t, relay, alice.node.PublicKey(), frameOf(t, &yggproto.WireFrame{Body: &yggproto.WireFrame_SfPull{
		SfPull: &yggproto.SfPull{
			MailboxId: bobsMailbox[:], EpochDay: epoch, TimestampS: ts,
			Signature: sig, Proof: proofToWire(proofOf(t, alice)),
		},
	}}))

	st := sfStatsOf(t, relay)
	if st.handedOut != 0 {
		t.Fatalf("чужая почта выдана: %+v", st)
	}
	if st.refused == 0 || !strings.Contains(st.lastRefusal, "ящик не выведен") {
		t.Fatalf("несоответствие ящика не названо: %+v", st)
	}
	// Отказанному письмо остаётся на хранении: чужая попытка не должна стоить
	// адресату почты.
	if got := sfStored(t, relay); got != 1 {
		t.Fatalf("после чужого запроса на хранении %d писем", got)
	}

	// Тот же запрос просит владелец — и ворота авторизации он проходит: дело
	// доходит до выдачи. Ответ в изолированном тесте уйти не может (маршрута
	// обратно нет), и релей обязан на этом остановиться, не трогая почту:
	// списанное на неудачной попытке письмо адресат не увидел бы никогда.
	// Сквозная выдача проверена e2e-тестом ниже.
	ownSig, err := crypto.BuildSFPullSignature(bob.master, bobsMailbox, epoch, ts)
	if err != nil {
		t.Fatal(err)
	}
	inject(t, relay, bob.node.PublicKey(), frameOf(t, &yggproto.WireFrame{Body: &yggproto.WireFrame_SfPull{
		SfPull: &yggproto.SfPull{
			MailboxId: bobsMailbox[:], EpochDay: epoch, TimestampS: ts,
			Signature: ownSig, Proof: proofToWire(proofOf(t, bob)),
		},
	}}))
	got := sfStatsOf(t, relay)
	if !strings.Contains(got.lastRefusal, "маршрута обратно") {
		t.Fatalf("владелец не допущен к выдаче: %+v", got)
	}
	if got.handedOut != 0 {
		t.Fatalf("неудачная попытка записана как выдача: %+v", got)
	}
	if stored := sfStored(t, relay); stored != 1 {
		t.Fatalf("ответ не ушёл, а почта списана: на хранении %d писем", stored)
	}
}

// TestSfCollectWithoutRelaysIsHonest — команда без релеев возвращает ноль, а не
// правдоподобную имитацию работы, и не выполняется на остановленном ядре.
func TestSfCollectWithoutRelaysIsHonest(t *testing.T) {
	e := startEngine(t, seedBob, t.TempDir())
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SfCollect{
		SfCollect: &yggproto.SfCollectReq{},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("SfCollect без релеев: %s", resp.GetErrorMessage())
	}
	if got := resp.GetSfCollect(); got.GetRequested() != 0 || len(got.GetRelays()) != 0 {
		t.Fatalf("запрос в никуда засчитан как работа: %+v", got)
	}
	stopIsolated(t, e)

	if e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SfCollect{
		SfCollect: &yggproto.SfCollectReq{},
	}}).GetSuccess() {
		t.Fatal("SfCollect на остановленном ядре прошёл")
	}
}

// TestRouteGateIsBypassedOnlyForRelayMail — ворота Шага 5-a сверяют узел
// собеседника с картой и отбрасывают пакет, пришедший с другого адреса. У почты
// с релея адреса отправителя нет в помине: личность доказана связкой внутри
// конверта, поэтому ворота пропускаются — но только для варианта SfReply.
func TestRouteGateIsBypassedOnlyForRelayMail(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	carol := startEngine(t, seedCarol, t.TempDir())
	bobsBundle := registerContact(t, alice, bob)
	registerContact(t, bob, alice)

	env := wireOf(t, alice, bob, bobsBundle, &yggproto.ChatMessage{
		MessageId:      "sf-gate-1",
		ConversationId: "диалог через релей",
		TimestampMs:    1700000000000,
		Content:        textContent("принесено релеем"),
	})
	conv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())

	// 1. Тот же конверт «напрямую» с чужого узла — ворота закрыты.
	inject(t, bob, carol.node.PublicKey(), marshalWire(t, env))
	if st := statsOf(t, bob); st.routeMismatch != 1 || st.applied != 0 {
		t.Fatalf("разъезд адреса не отброшен: %+v", st)
	}
	if got := storedBodies(t, bob, conv); len(got) != 0 {
		t.Fatalf("отклонённый пакет лёг в базу: %v", got)
	}

	// 2. Его принёс релей: письмо обязано открыться.
	inject(t, bob, carol.node.PublicKey(), frameOf(t, &yggproto.WireFrame{Body: &yggproto.WireFrame_SfReply{
		SfReply: &yggproto.SfReply{Envelopes: []*yggproto.SealedEnvelope{env}},
	}}))
	st := statsOf(t, bob)
	if st.applied != 1 || st.routeMismatch != 1 {
		t.Fatalf("почта релея применена неверно: %+v", st)
	}
	if got := storedBodies(t, bob, conv); len(got) != 1 || got[0] != "принесено релеем" {
		t.Fatalf("диалог после почты релея: %v", got)
	}
	if got := sfStatsOf(t, bob); got.pulled != 1 {
		t.Fatalf("забранное с релея не посчитано: %+v", got)
	}
	// Ответить можно только заверенному узлу отправителя из карты; здесь он
	// есть, но маршрута до него нет, поэтому ACK засчитан пропущенным, а не
	// отправленным в никуда.
	if got := statsOf(t, bob); got.ackSent != 0 || got.ackSkipped != 1 {
		t.Fatalf("ACK на почту релея послан некуда: %+v", got)
	}

	// 3. Отказ релея — новость, а не молчание: причина уходит в счётчик.
	client := startEngine(t, seedDave, t.TempDir())
	inject(t, client, carol.node.PublicKey(), frameOf(t, &yggproto.WireFrame{Body: &yggproto.WireFrame_SfReply{
		SfReply: &yggproto.SfReply{Error: "квота ящика исчерпана"},
	}}))
	got := sfStatsOf(t, client)
	if got.pulled != 0 || got.refused != 1 || !strings.Contains(got.lastRefusal, "квота") {
		t.Fatalf("отказ релея проглочен: %+v", got)
	}
}

// TestRelayRedeliveryIsDeduplicatedAtClient — релей пересылает уже выданное
// письмо в окне повторной выдачи: адресат, до которого пакет ответа не дошёл,
// приходит за почтой снова. Тот же шифртекст вторично приёмной цепочке не нужен —
// шаг рэтчета уже сдвинут, и повтор утонул бы в счётчике отказов там, где отказа
// не было. Поэтому клиент опознаёт выдачу по следам конверта до расшифровывания.
func TestRelayRedeliveryIsDeduplicatedAtClient(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	carol := startEngine(t, seedCarol, t.TempDir())
	bobsBundle := registerContact(t, alice, bob)
	registerContact(t, bob, alice)

	env := wireOf(t, alice, bob, bobsBundle, &yggproto.ChatMessage{
		MessageId:      "sf-redelivery-1",
		ConversationId: "пересылка релея",
		TimestampMs:    1700000000000,
		Content:        textContent("отдам ещё раз"),
	})
	conv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	frame := frameOf(t, &yggproto.WireFrame{Body: &yggproto.WireFrame_SfReply{
		SfReply: &yggproto.SfReply{Envelopes: []*yggproto.SealedEnvelope{env}},
	}})

	inject(t, bob, carol.node.PublicKey(), frame)
	if got := sfStatsOf(t, bob); got.pulled != 1 {
		t.Fatalf("первая выдача не засчитана: %+v", got)
	}
	inject(t, bob, carol.node.PublicKey(), frame)
	if got := sfStatsOf(t, bob); got.pulled != 1 {
		t.Fatalf("повтор выдачи посчитан забранным: %+v", got)
	}
	if st := statsOf(t, bob); st.applied != 1 || st.duplicated != 1 || st.rejected != 0 {
		t.Fatalf("повтор выдачи дошёл до рэтчета: %+v", st)
	}
	if got := storedBodies(t, bob, conv); len(got) != 1 || got[0] != "отдам ещё раз" {
		t.Fatalf("пересылка удвоила диалог: %v", got)
	}
}

// TestMailboxGCCountersExpired — просроченное чужое письмо выбрасывается по
// опубликованному TTL и считается отдельно от отказов: потеря по сроку — не
// отказ, и прятать её за одной цифрой нельзя.
func TestMailboxGCCountersExpired(t *testing.T) {
	relay := startEngineWith(t, seedCarol, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.SfRelayEnabled = true
	})
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	env := letterTo(t, alice, bob, "sf-expire-1")
	if err := relay.db.SFPut(&storage.StoredSF{
		MailboxID: hexKey(env.GetBlindToken()), EnvelopeID: "старое", Data: []byte("конверт"),
		PlacedMS: 1, ExpiresMS: time.Now().Add(-time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatal(err)
	}

	relay.sfTick()

	if got := sfStored(t, relay); got != 0 {
		t.Fatalf("просроченное письмо уцелело: %d", got)
	}
	if st := sfStatsOf(t, relay); st.expired != 1 {
		t.Fatalf("сброс по TTL не посчитан: %+v", st)
	}
	if got := sfStatusOf(t, relay).GetExpired(); got != 1 {
		t.Fatalf("статус молчит о потерянном по сроку: %d", got)
	}
	// Релеи не настроены — фоновый опрос не выполняется и ничего не портит.
	if st := sfStatsOf(t, relay); st.pushed != 0 || st.pushFailed != 0 {
		t.Fatalf("пустой список релеев породил трафик: %+v", st)
	}
}

// TestBlindRelayCarriesMailToOfflineRecipient — сквозной смысл Шага 6: адресат
// офлайн, маршрут до него не появится до его возвращения, и всё же письмо
// живёт у третьего узла, а вернувшись, адресат забирает его и открывает диалог.
// Три настоящие ноды поверх оверлея.
func TestBlindRelayCarriesMailToOfflineRecipient(t *testing.T) {
	const body = "оно дождётся адресата"
	port := freeTCPPort(t)

	relay := startEngineWith(t, seedCarol, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(port)
		cfg.SfRelayEnabled = true
	})
	relayKey := ed25519.PublicKey(append([]byte(nil), relay.node.PublicKey()...))

	// Боб существует как идентичность, но его узла в сети нет: он погасен до
	// кладки и вернётся другим процессом с той же базой.
	bobDir := t.TempDir()
	bobOffline := startEngineWith(t, seedBob, bobDir, func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
	})
	bobBundle := bundleOf(t, bobOffline)
	bobNodeKey := ed25519.PublicKey(append([]byte(nil), bobOffline.node.PublicKey()...))
	bobNodeIPv6 := bundleNodeIPv6(bobOffline)
	stopIsolated(t, bobOffline)

	alice := startEngineWith(t, seedAlice, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.BootstrapPeers = []string{fmt.Sprintf("127.0.0.1:%d", port)}
		cfg.SfRelayNodeKeys = [][]byte{relayKey}
	})
	resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			NodeIpv6:      bobNodeIPv6,
			Ed25519Pubkey: bobNodeKey,
			X25519Pubkey:  bobBundle.GetIdentityX25519Pubkey(),
			Bundle:        bobBundle,
		}},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("AddContact: %s", resp.GetErrorMessage())
	}
	waitForRoute(t, alice, relayKey)

	if got := sfStatusOf(t, alice); got.GetRelaysConfigured() != 1 {
		t.Fatal("релей не виден в статусе клиента")
	}

	msg := &yggproto.ChatMessage{
		MessageId:      "sf-e2e-1",
		ConversationId: "офлайновый диалог",
		TimestampMs:    1700000000000,
		Content:        textContent(body),
	}
	sendResp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{
			DestinationX25519Pubkey: bobBundle.GetIdentityX25519Pubkey(),
			Message:                 msg,
		},
	}})
	if !sendResp.GetSuccess() {
		t.Fatalf("SendMessage: %s", sendResp.GetErrorMessage())
	}
	sent := sendResp.GetSendMessage()
	if sent.GetWrittenToOverlay() {
		t.Fatal("сообщение ушло адресату, которого нет в сети")
	}
	if !sent.GetQueuedInOutbox() || !sent.GetRelaysPushed() || sent.GetRelaysHanded() == 0 {
		t.Fatalf("офлайновая отправка не легла релеем: %+v", sent)
	}
	// Честная формулировка: «передано оверлею» не значит «хранится», поэтому
	// хранение подтверждает только счётчик самого хранителя.
	if got := sfStatsOf(t, alice); got.pushed < uint64(sent.GetRelaysHanded()) || got.pushFailed != 0 {
		t.Fatalf("пуш релеям посчитан неверно: %+v", got)
	}

	if !waitUntil(t, 30*time.Second, func() bool {
		return sfStored(t, relay) == 1 && sfStatusOf(t, relay).GetStored() == 1
	}) {
		t.Fatalf("релей не хранит письмо: %+v; у отправителя %+v; счётчики релея %+v / приём %+v",
			sfStatusOf(t, relay), sfStatsOf(t, alice), sfStatsOf(t, relay), statsOf(t, relay))
	}

	bob := startEngineWith(t, seedBob, bobDir, func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.BootstrapPeers = []string{fmt.Sprintf("127.0.0.1:%d", port)}
		cfg.SfRelayNodeKeys = [][]byte{relayKey}
	})
	waitForRoute(t, bob, relayKey)
	waitForRoute(t, relay, bob.node.PublicKey())

	conv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	// Запрос почты повторяется, как и живая отправка в e2e-тестах Шага 3:
	// пакет запроса — первый на ещё не прогретом маршруте, а оверлей такие
	// регулярно теряет. Релей при этом ничего не списывает по неудачной
	// попытке, так что повтор — это штатный путь, а не костыль теста.
	deadline := time.Now().Add(60 * time.Second)
	for attempts := 1; ; attempts++ {
		collect := bob.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SfCollect{
			SfCollect: &yggproto.SfCollectReq{},
		}})
		if !collect.GetSuccess() {
			t.Fatalf("SfCollect (попытка %d): %s", attempts, collect.GetErrorMessage())
		}
		if got := collect.GetSfCollect(); got.GetRequested() == 0 || len(got.GetRelays()) != 1 {
			t.Fatalf("запрос почте не ушёл: %+v", got)
		}
		if waitUntil(t, 3*time.Second, func() bool { return hasBody(t, bob, conv, body) }) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("за %d попыток почта не открылась у адресата: приём %+v, клиент %+v, релей %+v (%+v), база %v",
				attempts, statsOf(t, bob), sfStatsOf(t, bob), sfStatsOf(t, relay),
				sfStatusOf(t, relay), storedBodies(t, bob, conv))
		}
	}
	if st := statsOf(t, bob); st.routeMismatch != 0 {
		t.Fatalf("почта релея прошла через адресные ворота: %+v", st)
	}
	if st := sfStatsOf(t, bob); st.pulled == 0 {
		t.Fatalf("забранное не посчитано: %+v", st)
	}
	if got := sfStatsOf(t, relay); got.handedOut == 0 || got.refused != 0 {
		t.Fatalf("выдача посчитана неверно: %+v", got)
	}

	// Выданное письмо остаётся на хранении до окна повторной выдачи: пакет
	// ответа релей доставляет тем же оверлеем, что и живой конверт, и потеря
	// ответа не должна стоить адресату почты. Дубликаты выдачи гасит клиент
	// (см. TestRelayRedeliveryIsDeduplicatedAtClient).
	if stored := sfStored(t, relay); stored != 1 {
		t.Fatalf("релей списал почту по первой выдаче: %d писем", stored)
	}
	// ...но не вечно: письмо, которое так и не забрали, сбрасывается по TTL, а
	// выданное — по истечении окна (счётчик expired).
	relay.sfTick()
	if got := sfStatsOf(t, relay); got.expired != 0 {
		t.Fatalf("окно пересылки истекло раньше срока: %+v", got)
	}
}

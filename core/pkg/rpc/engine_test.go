package rpc

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"ygggram/core/pkg/crypto"
	"ygggram/core/pkg/storage"
	"ygggram/core/pkg/yggnode"
	yggproto "ygggram/core/proto"
)

func startEngine(t *testing.T, mnemonic, dir string) *Engine {
	t.Helper()
	return startEngineWith(t, mnemonic, dir, nil)
}

// startEngineWith запускает ядро с дополненной конфигурацией — так тесты
// получают управление слушателем и пирами, не переписывая общий путь.
func startEngineWith(t *testing.T, mnemonic, dir string, mut func(*yggproto.NodeConfig)) *Engine {
	t.Helper()
	e := New()
	cfg := &yggproto.NodeConfig{
		DataDir:      dir + "/data",
		CacheDir:     dir + "/cache",
		MnemonicSeed: mnemonic,
		AntiDpiMode:  yggproto.AntiDpiMode_ANTI_DPI_UTLS_CHROME,
	}
	if mut != nil {
		mut(cfg)
	}
	resp := e.Handle(&yggproto.CoreRequest{RequestId: 1, Request: &yggproto.CoreRequest_StartNode{
		StartNode: &yggproto.StartNodeReq{Config: cfg},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("StartNode не прошёл: %s", resp.GetErrorMessage())
	}
	t.Cleanup(func() {
		e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}})
	})
	return e
}

func freeTCPPort(t *testing.T) uint32 {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	return uint32(l.Addr().(*net.TCPAddr).Port)
}

func waitForRoute(t *testing.T, e *Engine, dest ed25519.PublicKey) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if e.node.HasRouteTo(dest) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("маршрут до %x так и не появился (peers: %v)", dest, e.node.Peers())
}

// TestStartNodeRejectsEmptyMnemonic — идентичность обязана выводиться из seed,
// молча подставлять дефолт нельзя.
func TestStartNodeRejectsEmptyMnemonic(t *testing.T) {
	e := New()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StartNode{
		StartNode: &yggproto.StartNodeReq{Config: &yggproto.NodeConfig{DataDir: t.TempDir()}},
	}})
	if resp.GetSuccess() {
		t.Fatal("ожидается ошибка на пустом mnemonic_seed")
	}
}

func TestNodeStatusReflectsRealNode(t *testing.T) {
	e := startEngine(t, seedCrush, t.TempDir())
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetNodeStatus{}})
	st := resp.GetNodeStatus()
	if st == nil || !resp.GetSuccess() {
		t.Fatalf("GetNodeStatus: %s", resp.GetErrorMessage())
	}
	if st.GetMyIpv6() != e.node.NodeIPv6().String() {
		t.Fatalf("ipv6 в ответе %q не совпадает с нодой %q", st.GetMyIpv6(), e.node.NodeIPv6())
	}
	if !bytes.Equal(st.GetMyEd25519Pubkey(), e.node.PublicKey()) {
		t.Fatal("публичный ключ ноды в ответе не настоящий")
	}
	if st.GetUptimeSeconds() > 60 {
		t.Fatalf("подозр uptime: %d", st.GetUptimeSeconds())
	}
}

// TestEndToEndSealedChatMessage — главный смысл Шага 1.5: сообщение реально
// шифруется X3DH+Double Ratchet, упаковывается в Sealed Sender, кладётся в
// зашифрованное хранилище и расшифровывается у получателя.
func TestEndToEndSealedChatMessage(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())

	bundleResp := bob.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetPrekeyBundle{}})
	bundle := bundleResp.GetPrekeyBundle()
	if !bundleResp.GetSuccess() || bundle == nil {
		t.Fatalf("PreKeyBundle: %s", bundleResp.GetErrorMessage())
	}
	if len(bundle.GetSignedPrekey()) != 32 || len(bundle.GetOneTimePrekeys()) == 0 {
		t.Fatal("бандл неполный: нет signed prekey или one-time prekey")
	}
	if !ed25519.Verify(bundle.GetIdentityEd25519Pubkey(), bundle.GetSignedPrekey(), bundle.GetSignedPrekeySignature()) {
		t.Fatal("подпись signed prekey не проходит проверку")
	}

	addResp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			NodeIpv6:      bundleNodeIPv6(bob),
			Ed25519Pubkey: bob.node.PublicKey(),
			X25519Pubkey:  bundle.GetIdentityX25519Pubkey(),
			Bundle:        bundle,
		}},
	}})
	if !addResp.GetSuccess() {
		t.Fatalf("AddContact: %s", addResp.GetErrorMessage())
	}

	msg := &yggproto.ChatMessage{
		MessageId:      "msg-1",
		ConversationId: "conv-1",
		TimestampMs:    1700000000000,
		IsOutbound:     true,
		Content:        &yggproto.ChatMessage_Text{Text: &yggproto.TextContent{Body: "ТСПУ молчит"}},
	}
	sendResp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{
			DestinationIpv6:         msg.GetConversationId(),
			DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
			Message:                 msg,
		},
	}})
	if !sendResp.GetSuccess() {
		t.Fatalf("SendMessage: %s", sendResp.GetErrorMessage())
	}
	sr := sendResp.GetSendMessage()
	if sr.GetMessageId() != "msg-1" {
		t.Fatalf("message_id в ответе: %q", sr.GetMessageId())
	}
	// Маршрута до Боба нет, поэтому честный ответ — локальная исходящая
	// очередь, а не «пакет ушёл в оверлей».
	if sr.GetWrittenToOverlay() {
		t.Fatal("пакет не мог уйти в оверлей: пиров нет")
	}
	if !sr.GetQueuedInOutbox() {
		t.Fatal("ожидается queued_in_outbox=true")
	}

	// Расшифровываем ровно тот конверт, который ядро отдало в сеть. Второй
	// вызов seal здесь означал бы второй шаг отправной цепочки — и Боб бы его
	// не принял.
	wire := sr.GetSealedEnvelope()
	if len(wire) == 0 {
		t.Fatal("SendMessage не вернул sealed_envelope")
	}

	env := chatOf(t, wire)
	plaintext, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(env.GetEncryptedPayload(), plaintext) {
		t.Fatal("полезная нагрузка не зашифрована")
	}

	payload, err := crypto.UnpackSealedEnvelope(bob.master.MasterX25519, envelopeFromWire(env))
	if err != nil {
		t.Fatalf("UnpackSealedEnvelope: %v", err)
	}
	if !bytes.Equal(payload.SenderMasterEd25519, alice.master.MasterEd25519.Public().(ed25519.PublicKey)) {
		t.Fatal("идентичность отправителя внутри конверта не совпадает")
	}

	inner, ekPub, err := DecodeRatchetMessage(payload.InnerRatchetMsg)
	if err != nil {
		t.Fatalf("разбор рэтчет-сообщения: %v", err)
	}
	ek, err := ecdh.X25519().NewPublicKey(ekPub)
	if err != nil {
		t.Fatal(err)
	}
	senderIK, err := ecdh.X25519().NewPublicKey(alice.master.MasterX25519.PublicKey().Bytes())
	if err != nil {
		t.Fatal(err)
	}
	sharedKey, err := crypto.PerformX3DHReceiver(bob.master.MasterX25519, bob.spkPriv, bob.opkPriv[bundle.GetKeyId()], senderIK, ek)
	if err != nil {
		t.Fatalf("X3DH получателя: %v", err)
	}
	receiver := crypto.NewReceiverSession(sharedKey, bob.spkPriv)
	got, err := receiver.Decrypt(*inner, []byte("conv-1"))
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	decoded := &yggproto.ChatMessage{}
	if err := proto.Unmarshal(got, decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetText().GetBody() != "ТСПУ молчит" {
		t.Fatalf("расшифровали не то: %q", decoded.GetText().GetBody())
	}

	// Прочитанное UI должно совпадать с отправленным: тот же диалог, то же
	// количество и тот же текст.
	convs := alice.Handle(&yggproto.CoreRequest{
		Request: &yggproto.CoreRequest_ListConversations{ListConversations: &yggproto.ListConversationsReq{}},
	}).GetConversations().GetConversations()
	if len(convs) != 1 || convs[0].GetConversationId() != "conv-1" || convs[0].GetMessageCount() != 1 {
		t.Fatalf("ListConversations вернул не то: %v", convs)
	}
	stored := alice.Handle(&yggproto.CoreRequest{
		Request: &yggproto.CoreRequest_GetMessages{GetMessages: &yggproto.GetMessagesReq{ConversationId: "conv-1"}},
	}).GetMessages().GetMessages()
	if len(stored) != 1 || stored[0].GetText().GetBody() != "ТСПУ молчит" {
		t.Fatalf("GetMessages вернул не то: %v", stored)
	}
	contacts := alice.Handle(&yggproto.CoreRequest{
		Request: &yggproto.CoreRequest_ListContacts{ListContacts: &yggproto.ListContactsReq{}},
	}).GetContacts().GetContacts()
	if len(contacts) != 1 || contacts[0].GetBundle() == nil {
		t.Fatalf("ListContacts вернул не то: %v", contacts)
	}
}

// TestSendWithoutContactFails — нельзя «отправить» в никуда, только по
// предварительно добавленному контакту с бандлом.
func TestSendWithoutContactFails(t *testing.T) {
	alice := startEngine(t, seedCarol, t.TempDir())
	resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{
			DestinationX25519Pubkey: make([]byte, 32),
			Message:                 &yggproto.ChatMessage{ConversationId: "c", Content: &yggproto.ChatMessage_Text{Text: &yggproto.TextContent{Body: "x"}}},
		},
	}})
	if resp.GetSuccess() {
		t.Fatal("ожидается ошибка: контакт с PreKeyBundle не добавлен")
	}
}

// TestGeneratedIdentityStartsNode — фраза, которую отдаёт ядро, обязана
// запускать ноду: иначе генерация и вывод ключей живут в разных вселенных.
func TestGeneratedIdentityStartsNode(t *testing.T) {
	e := New()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GenerateIdentity{}})
	if !resp.GetSuccess() {
		t.Fatalf("GenerateIdentity: %s", resp.GetErrorMessage())
	}
	mnemonic := resp.GetIdentity().GetMnemonic()
	if len(strings.Fields(mnemonic)) != 12 {
		t.Fatalf("фраза не из 12 слов: %q", mnemonic)
	}

	restored := startEngine(t, mnemonic, t.TempDir())
	again := startEngine(t, mnemonic, t.TempDir())
	if restored.node.NodeIPv6().String() != again.node.NodeIPv6().String() {
		t.Fatal("одна и та же фраза даёт разные адреса — вывод идентичности не детерминирован")
	}
}

func bundleNodeIPv6(e *Engine) string { return e.node.NodeIPv6().String() }

// TestOverlayDeliveryBetweenEngines — смысл Шага 2b на уровне движка: два
// ядра сходятся по настоящему TCP, и SendMessage по Known-маршруту отдаёт
// written_to_overlay=true ровно теми байтами, которые видит оверлей у
// получателя.
func TestOverlayDeliveryBetweenEngines(t *testing.T) {
	port := freeTCPPort(t)
	alice := startEngineWith(t, seedAlice, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(port)
	})
	// Пир задан без схемы: ядро обязано дополнить его tcp, а не tls/wss.
	bob := startEngineWith(t, seedBob, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.BootstrapPeers = []string{fmt.Sprintf("127.0.0.1:%d", port)}
	})

	bundleResp := bob.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetPrekeyBundle{}})
	bundle := bundleResp.GetPrekeyBundle()
	if !bundleResp.GetSuccess() || bundle == nil {
		t.Fatalf("PreKeyBundle: %s", bundleResp.GetErrorMessage())
	}
	addResp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			NodeIpv6:      bundleNodeIPv6(bob),
			Ed25519Pubkey: bob.node.PublicKey(),
			X25519Pubkey:  bundle.GetIdentityX25519Pubkey(),
			Bundle:        bundle,
		}},
	}})
	if !addResp.GetSuccess() {
		t.Fatalf("AddContact: %s", addResp.GetErrorMessage())
	}

	waitForRoute(t, alice, bob.node.PublicKey())
	waitForRoute(t, bob, alice.node.PublicKey())

	peersResp := bob.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetPeers{}})
	peers := peersResp.GetPeerList()
	if !peersResp.GetSuccess() || len(peers.GetPeers()) == 0 {
		t.Fatalf("у Боба нет пиров: %v (%s)", peers, peersResp.GetErrorMessage())
	}
	if got := peers.GetPeers()[0].GetTransport(); got != yggproto.PeerTransportType_TRANSPORT_TCP {
		t.Fatalf("транспорт пира определён неверно: %v", got)
	}

	inbound, cancel := bob.node.Subscribe(64)
	defer cancel()

	msg := &yggproto.ChatMessage{
		ConversationId: "conv-overlay",
		TimestampMs:    1700000000000,
		IsOutbound:     true,
		Content:        &yggproto.ChatMessage_Text{Text: &yggproto.TextContent{Body: "оверлей живой"}},
	}
	deadline := time.Now().Add(30 * time.Second)
	for attempts := 1; ; attempts++ {
		resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
			SendMessage: &yggproto.SendMessageReq{
				DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
				Message:                 msg,
			},
		}})
		if !resp.GetSuccess() {
			t.Fatalf("SendMessage (попытка %d): %s", attempts, resp.GetErrorMessage())
		}
		sr := resp.GetSendMessage()
		if !sr.GetWrittenToOverlay() {
			if time.Now().After(deadline) {
				t.Fatalf("written_to_overlay так и не стал true за %d попыток", attempts)
			}
			continue
		}
		// Раз оверлей принял пакет, те же байты обязаны прийти на сторону
		// получателя: без этого «доставлено» было бы самообманом.
		env := sr.GetSealedEnvelope()
		select {
		case p := <-inbound:
			if !bytes.Equal(p.From, alice.node.PublicKey()) {
				t.Fatalf("пакет пришёл от %x, ожидался ключ Алисы", p.From)
			}
			if !bytes.Equal(p.Payload, env) {
				t.Fatalf("байты у получателя не совпадают с отправленными: %d против %d", len(p.Payload), len(env))
			}
			return
		case <-time.After(2 * time.Second):
			if time.Now().After(deadline) {
				t.Fatalf("оверлей принял пакет, но у Боба ничего не появилось (%d попыток)", attempts)
			}
		}
	}
}

// foreignBundle собирает бандл «внешнего» контакта без целиком узла: идентичность
// выводится из фразы, ключ ноды передаётся тестом, а сертификат подписывает этот
// же мастер-Ed25519. Так проверка связки ключей остаётся настоящей криптографией,
// даже когда проверять надо именно её, а не оверлей.
func foreignBundle(t *testing.T, mnemonic string, nodePub ed25519.PublicKey) *yggproto.PreKeyBundle {
	t.Helper()
	master, err := crypto.MasterIdentityFromMnemonic(mnemonic, "")
	if err != nil {
		t.Fatal(err)
	}
	spk, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	opk, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := crypto.BuildNodeKeyProof(master, nodePub)
	if err != nil {
		t.Fatal(err)
	}
	return &yggproto.PreKeyBundle{
		IdentityEd25519Pubkey: master.MasterEd25519.Public().(ed25519.PublicKey),
		IdentityX25519Pubkey:  master.MasterX25519.PublicKey().Bytes(),
		SignedPrekey:          spk.PublicKey().Bytes(),
		SignedPrekeySignature: ed25519.Sign(master.MasterEd25519, spk.PublicKey().Bytes()),
		OneTimePrekeys:        [][]byte{opk.PublicKey().Bytes()},
		KeyId:                 1,
		NodeEd25519Pubkey:     append([]byte(nil), nodePub...),
		NodeKeyProof:          proof,
	}
}

// cardOf — честная карта под этот бандл: адрес выведен из заверенного ключа ноды,
// identity-ключ в карте и в бандле один.
func cardOf(b *yggproto.PreKeyBundle) *yggproto.ContactCard {
	return &yggproto.ContactCard{
		NodeIpv6:      yggnode.AddressForKey(b.GetNodeEd25519Pubkey()).String(),
		Ed25519Pubkey: b.GetNodeEd25519Pubkey(),
		X25519Pubkey:  b.GetIdentityX25519Pubkey(),
		Bundle:        b,
	}
}

// addContactErr — ответ ядра на AddContact: пустая строка означает принятие.
func addContactErr(t *testing.T, e *Engine, card *yggproto.ContactCard) string {
	t.Helper()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: card},
	}})
	if resp.GetSuccess() {
		return ""
	}
	return resp.GetErrorMessage()
}

// freshNodeKey — настоящий ed25519-ключ узла для карт, которые никто не запускает.
func freshNodeKey(t *testing.T) ed25519.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

// TestAddContactRejectsSpoofedNodeAddress — адрес Yggdrasil выводится из ключа
// ноды, поэтому карту с несовпадающей парой принять нельзя.
func TestAddContactRejectsSpoofedNodeAddress(t *testing.T) {
	e := startEngine(t, seedAlice, t.TempDir())
	nodeKey := freshNodeKey(t)
	other := freshNodeKey(t)
	bundle := foreignBundle(t, seedBob, nodeKey)

	spoofed := cardOf(proto.Clone(bundle).(*yggproto.PreKeyBundle))
	spoofed.NodeIpv6 = yggnode.AddressForKey(other).String()
	if msg := addContactErr(t, e, spoofed); !strings.Contains(msg, "не соответствует ключу ноды") {
		t.Fatalf("карта с чужим node_ipv6: %q", msg)
	}

	if msg := addContactErr(t, e, cardOf(bundle)); msg != "" {
		t.Fatalf("корректная карта отклонена: %s", msg)
	}

	noKey := cardOf(proto.Clone(bundle).(*yggproto.PreKeyBundle))
	noKey.Ed25519Pubkey = nil
	if msg := addContactErr(t, e, noKey); !strings.Contains(msg, "нет ed25519-ключа ноды") {
		t.Fatalf("контакт без ключа ноды принят: %q", msg)
	}
}

// TestAddContactRequiresNodeKeyBinding — смысл Шага 3-c-4: адрес доставки обязан
// быть заверен той же идентичностью, под которую собирается X3DH. Иначе карта
// «чужой identity-ключ + мой узел» увела бы шифротекст честного контакта
// пересыльщику — тот его не расшифрует, но увидит, что диалог существует, и
// сообщение до адресата не дойдёт никогда.
func TestAddContactRequiresNodeKeyBinding(t *testing.T) {
	e := startEngine(t, seedAlice, t.TempDir())
	bobNode := freshNodeKey(t)
	malloryNode := freshNodeKey(t)
	bobBundle := foreignBundle(t, seedBob, bobNode)
	// Сертификат Кароли поверх ключа ноды Боба: подписан чужим мастер-Ed25519 и
	// над чужим мастер-X25519, поэтому пройдёт проверку только у Кароли.
	carolBundle := foreignBundle(t, seedCarol, bobNode)

	fresh := func() *yggproto.ContactCard {
		return cardOf(proto.Clone(bobBundle).(*yggproto.PreKeyBundle))
	}
	if msg := addContactErr(t, e, fresh()); msg != "" {
		t.Fatalf("честная карта отклонена: %s", msg)
	}

	cases := []struct {
		name string
		mut  func(*yggproto.ContactCard)
		want string
	}{
		{
			name: "сертификата нет",
			mut:  func(c *yggproto.ContactCard) { c.Bundle.NodeKeyProof = nil },
			want: "не привязан к идентичности",
		},
		{
			name: "сертификат от другой идентичности",
			mut:  func(c *yggproto.ContactCard) { c.Bundle.NodeKeyProof = carolBundle.GetNodeKeyProof() },
			want: "не привязан к идентичности",
		},
		{
			name: "подпись испорчена",
			mut:  func(c *yggproto.ContactCard) { c.Bundle.NodeKeyProof[0] ^= 0xff },
			want: "не привязан к идентичности",
		},
		{
			name: "ключ ноды подменён",
			mut: func(c *yggproto.ContactCard) {
				c.Ed25519Pubkey = malloryNode
				c.NodeIpv6 = yggnode.AddressForKey(malloryNode).String()
			},
			want: "не тот, что заверен в бандле",
		},
		{
			name: "ключа ноды в бандле нет",
			mut:  func(c *yggproto.ContactCard) { c.Bundle.NodeEd25519Pubkey = nil },
			want: "нет ключа ноды",
		},
		{
			name: "identity-ключ карты и бандла разноятся",
			mut: func(c *yggproto.ContactCard) {
				other := make([]byte, 32)
				if _, err := rand.Read(other); err != nil {
					t.Fatal(err)
				}
				c.X25519Pubkey = other
			},
			want: "не совпадает с identity-ключом бандла",
		},
	}
	for _, tc := range cases {
		card := fresh()
		tc.mut(card)
		msg := addContactErr(t, e, card)
		if !strings.Contains(msg, tc.want) {
			t.Errorf("%s: ожидался отказ %q, получено %q", tc.name, tc.want, msg)
		}
	}

	// Отбракованные карты не должны были вытеснить честную.
	list := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_ListContacts{
		ListContacts: &yggproto.ListContactsReq{},
	}})
	contacts := list.GetContacts().GetContacts()
	if !list.GetSuccess() || len(contacts) != 1 {
		t.Fatalf("в списке %d контактов: %s", len(contacts), list.GetErrorMessage())
	}
	if !bytes.Equal(contacts[0].GetX25519Pubkey(), bobBundle.GetIdentityX25519Pubkey()) {
		t.Fatalf("в списке не тот контакт: %s", contacts[0].GetNodeIpv6())
	}
}

func TestURIsAndTransports(t *testing.T) {
	peerCases := []struct{ in, want string }{
		{"127.0.0.1:8666", "tcp://127.0.0.1:8666"},
		{"host.example", "tcp://host.example:54321"},
		{"tcp://1.2.3.4:2222", "tcp://1.2.3.4:2222"},
		{"tls://1.2.3.4:995", "tls://1.2.3.4:995"},
		{"quic://[fe80::1]:8282", "quic://[fe80::1]:8282"},
		{"tls://[fe80::1]", "tls://[fe80::1]:54321"},
	}
	for _, tc := range peerCases {
		got, err := peerURI(tc.in, yggproto.AntiDpiMode_ANTI_DPI_UTLS_CHROME)
		if err != nil || got != tc.want {
			t.Errorf("peerURI(%q) = %q, %v; ожидается %q", tc.in, got, err, tc.want)
		}
	}

	peerErrCases := []string{"", "  ", "udp://1.2.3.4:8666", "1.2.3.4:-1", "tcp://1.2.3.4:70000", ":8666", "::1:8666", "[]:8666"}
	for _, in := range peerErrCases {
		if got, err := peerURI(in, yggproto.AntiDpiMode_ANTI_DPI_DISABLED); err == nil {
			t.Errorf("peerURI(%q) молча стал %q", in, got)
		}
	}

	listenCases := []struct {
		addr string
		port int32
		mode yggproto.AntiDpiMode
		want string
	}{
		{"127.0.0.1", 8666, yggproto.AntiDpiMode_ANTI_DPI_DISABLED, "tcp://127.0.0.1:8666"},
		{"0.0.0.0", 0, yggproto.AntiDpiMode_ANTI_DPI_UTLS_CHROME, "tls://0.0.0.0:54321"},
		{"[::1]", 9999, yggproto.AntiDpiMode_ANTI_DPI_WEBSOCKET_CDN, "wss://[::1]:9999"},
		{"[::]", 1234, yggproto.AntiDpiMode_ANTI_DPI_UDP_QUIC_OBFS, "quic://[::]:1234"},
	}
	for _, tc := range listenCases {
		cfg := &yggproto.NodeConfig{ListenAddress: tc.addr, ListenPort: tc.port}
		got, err := listenURIs(cfg, tc.mode)
		if err != nil {
			t.Errorf("listenURIs(%q): %v", tc.addr, err)
			continue
		}
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("listenURIs(%q, порт %d) = %v; ожидается [%s]", tc.addr, tc.port, got, tc.want)
		}
	}

	// Без listen_address нода обязана оставаться исключительно исходящей —
	// молча открывать порт нельзя.
	got, err := listenURIs(&yggproto.NodeConfig{ListenPort: 8666}, yggproto.AntiDpiMode_ANTI_DPI_DISABLED)
	if err != nil || len(got) != 0 {
		t.Errorf("listenURIs без адреса = %v, %v; ожидается пустой список", got, err)
	}

	transportCases := map[string]yggproto.PeerTransportType{
		"tcp://1.2.3.4:8666":  yggproto.PeerTransportType_TRANSPORT_TCP,
		"tls://1.2.3.4:8666":  yggproto.PeerTransportType_TRANSPORT_TLS_UTLS,
		"wss://1.2.3.4:8666":  yggproto.PeerTransportType_TRANSPORT_WSS,
		"quic://1.2.3.4:8666": yggproto.PeerTransportType_TRANSPORT_QUIC_OBFS,
	}
	for uri, want := range transportCases {
		if got := transportFromURI(uri); got != want {
			t.Errorf("transportFromURI(%q) = %v, ожидается %v", uri, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Шаг 3-b: приёмный путь (Sealed Sender → X3DH → Double Ratchet → база → UI)
// ---------------------------------------------------------------------------

const (
	// Годные фразы BIP-39: ядро обязано отвергать фразу, не проходящую
	// контрольную сумму, поэтому «условные» наборы слов в фикстурах больше не
	// поднимают ноду. Значения выведены из sha256(«ygggram-test-» + имя) и
	// остаются буквальными, чтобы тест не зависел от генератора.
	seedAlice = "pattern goddess gift rail cabbage habit gap pelican toddler truck mom blood"
	seedBob   = "advice eager step magic velvet sheriff bicycle lake mango weapon rescue brave"
	seedCarol = "glare network dad paper now around safe parent jazz side soft shift"
	seedDave  = "shiver bulk duty buddy thrive boat damage apple quality sphere mean job"
	seedCrush = "pulse trash unit exit swamp world digital grape spice fault aim animal"
)

func textContent(body string) *yggproto.ChatMessage_Text {
	return &yggproto.ChatMessage_Text{Text: &yggproto.TextContent{Body: body}}
}

// bundleOf запрашивает бандл предключей и проверяет, что он настоящий.
func bundleOf(t *testing.T, e *Engine) *yggproto.PreKeyBundle {
	t.Helper()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetPrekeyBundle{}})
	b := resp.GetPrekeyBundle()
	if !resp.GetSuccess() || b == nil {
		t.Fatalf("PreKeyBundle: %s", resp.GetErrorMessage())
	}
	if len(b.GetSignedPrekey()) != 32 || len(b.GetOneTimePrekeys()) == 0 || b.GetKeyId() == 0 {
		t.Fatalf("бандл предключей неполный: %v", b)
	}
	// Бандл обязан нести и сертификат ключа ноды: AddContact без него откажет,
	// поэтому проверять связку надо сразу при выпуске бандла.
	if len(b.GetNodeEd25519Pubkey()) != ed25519.PublicKeySize || len(b.GetNodeKeyProof()) != ed25519.SignatureSize {
		t.Fatalf("бандл без сертификата ключа ноды: %v", b)
	}
	if err := crypto.VerifyNodeKeyProof(b.GetIdentityEd25519Pubkey(), b.GetIdentityX25519Pubkey(),
		b.GetNodeEd25519Pubkey(), b.GetNodeKeyProof()); err != nil {
		t.Fatalf("сертификат ключа ноды не проходит проверку: %v", err)
	}
	return b
}

// registerContact кладёт карту получателя отправителю: без бандла X3DH не собрать.
func registerContact(t *testing.T, from, to *Engine) *yggproto.PreKeyBundle {
	t.Helper()
	b := bundleOf(t, to)
	resp := from.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			NodeIpv6:      to.node.NodeIPv6().String(),
			Ed25519Pubkey: to.node.PublicKey(),
			X25519Pubkey:  b.GetIdentityX25519Pubkey(),
			Bundle:        b,
		}},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("AddContact: %s", resp.GetErrorMessage())
	}
	return b
}

// wireOf прогоняет сообщение через реальный SendMessage и возвращает конверт из
// ровно того кадра, который ядро отдаёт в сеть.
func wireOf(t *testing.T, from, to *Engine, bundle *yggproto.PreKeyBundle, msg *yggproto.ChatMessage) *yggproto.SealedEnvelope {
	t.Helper()
	resp := from.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{
			DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
			Message:                 msg,
		},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("SendMessage: %s", resp.GetErrorMessage())
	}
	return chatOf(t, resp.GetSendMessage().GetSealedEnvelope())
}

// statsOf снимает счётчик приёма под локом: приёмный цикл живёт в своей
// горутине, и без блокировки чтение было бы гонкой.
func statsOf(t *testing.T, e *Engine) inboundStats {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rxStats
}

func proofOf(t *testing.T, e *Engine) *crypto.IdentityProof {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.proof
}

func storedBodies(t *testing.T, e *Engine, conv string) []string {
	t.Helper()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetMessages{
		GetMessages: &yggproto.GetMessagesReq{ConversationId: conv},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("GetMessages: %s", resp.GetErrorMessage())
	}
	out := make([]string, 0, len(resp.GetMessages().GetMessages()))
	for _, m := range resp.GetMessages().GetMessages() {
		out = append(out, m.GetText().GetBody())
	}
	return out
}

func waitUntil(t *testing.T, limit time.Duration, ok func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if ok() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return ok()
}

// deliveredOf читает флаг сквозной доставки исходящего сообщения так, как его
// увидит UI: через GetMessages.
func deliveredOf(t *testing.T, e *Engine, conv, messageID string) bool {
	t.Helper()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetMessages{
		GetMessages: &yggproto.GetMessagesReq{ConversationId: conv},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("GetMessages: %s", resp.GetErrorMessage())
	}
	for _, m := range resp.GetMessages().GetMessages() {
		if m.GetMessageId() == messageID {
			return m.GetDelivered()
		}
	}
	t.Fatalf("сообщения %s нет в диалоге %q", messageID, conv)
	return false
}

// announcementEvent — фоновое объявление, которое ядро кладёт в канал само, по
// изменению вершины: peers_updated (Шаг 9-a) и anti_dpi_status (Шаг 16-d). Ни
// то, ни другое не является ответом на сообщение, поэтому тестовые помощники
// пролистывают оба. Отдельный предикат — не косметика: пока правило жило в
// двух `GetPeersUpdated() == nil` подряд, третье фоновое событие молча
// роняло четыре теста, и падали они не на логике, а на тайминге ноды.
func announcementEvent(ev *yggproto.CoreEvent) bool {
	return ev.GetPeersUpdated() != nil || ev.GetAntiDpiStatus() != nil
}

// nextEventOf ждёт событие с таймаутом: приёмный цикл кладывает его в канал
// до того, как тест успевает прочитать.
//
// Объявления вершины (peers_updated, Шаг 9) перешагиваются: ядро кладёт их в
// тот же канал по своему расписанию, и первое попавшееся событие больше не
// означает «событие о сообщении».
func nextEventOf(t *testing.T, e *Engine, limit time.Duration) *yggproto.CoreEvent {
	t.Helper()
	deadline := time.Now().Add(limit)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatal("за ожидание в канале событий нечего не легло")
			return nil
		}
		select {
		case ev := <-e.Events():
			if !announcementEvent(ev) {
				return ev
			}
		case <-time.After(remaining):
			t.Fatal("за ожидание в канале событий нечего не легло")
			return nil
		}
	}
}

// mustEvent читает следующее событие, не связанное с вершиной пиров. Терпение в
// две секунды нужно ровно потому, что канал общий: объявление peers_updated
// может лежать перед ним.
func mustEvent(t *testing.T, e *Engine) *yggproto.CoreEvent {
	t.Helper()
	return nextEventOf(t, e, 2*time.Second)
}

// assertNoMessageEvent требует, чтобы за окно в канале не пришло ни одного
// события кроме фоновых объявлений вершины. Раньше проверка была немедленной
// («в канале же пустоту видно»), но с Шага 9 пустой канал ничего не обещает:
// ядро вправе объявить пиры или состояние ссылок в любой момент, а лишнее
// событие о сообщении — это сбой.
func assertNoMessageEvent(t *testing.T, e *Engine, why string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-e.Events():
			if announcementEvent(ev) {
				continue
			}
			t.Fatalf("%s: лишнее событие %v", why, ev)
		case <-deadline:
			return
		}
	}
}

// awaitStatus вычитывает события, пока не попадётся MessageStatusEvent нужного
// вида. Позиция в очереди ничего не гарантирует: переотправки кладут свои
// статусы вперёд подтверждения.
func awaitStatus(t *testing.T, e *Engine, limit time.Duration, want func(*yggproto.MessageStatusEvent) bool) *yggproto.MessageStatusEvent {
	t.Helper()
	deadline := time.Now().Add(limit)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatal("подходящего MessageStatusEvent не дождались")
			return nil
		}
		select {
		case ev := <-e.Events():
			if s := ev.GetMessageStatus(); s != nil && want(s) {
				return s
			}
		case <-time.After(remaining):
			t.Fatal("подходящего MessageStatusEvent не дождались")
			return nil
		}
	}
}

// TestInboundPipelineStoresAndAnnounces — смысл Шага 3-b без сети: конверт,
// собранный ядром отправителя, проходит handleInbound получателя и обязан
// появиться в его зашифрованной базе и в канале событий, а повтор того же
// пакета — не создать второго сообщения.
func TestInboundPipelineStoresAndAnnounces(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	bundle := registerContact(t, alice, bob)

	const body = "шаг три: приём открыт"
	msg := &yggproto.ChatMessage{
		MessageId:      "msg-rx-1",
		ConversationId: "диалог у алисы",
		TimestampMs:    1700000000000,
		IsOutbound:     true,
		Content:        textContent(body),
	}
	env := wireOf(t, alice, bob, bundle, msg)
	pkt := yggnode.Packet{From: alice.node.PublicKey(), Payload: marshalWire(t, env)}

	bob.handleInbound(pkt)

	// Диалог у получателя именуется по identity key отправителя: имя, выбранное
	// другой стороной, подставлять нельзя.
	conv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	if conv == "диалог у алисы" {
		t.Fatal("имя диалога у получателя совпало с именем отправителя")
	}
	got := storedBodies(t, bob, conv)
	if len(got) != 1 || got[0] != body {
		t.Fatalf("в базе Боба: %v, ожидалось [%q]", got, body)
	}

	resp := bob.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_GetMessages{
		GetMessages: &yggproto.GetMessagesReq{ConversationId: conv},
	}})
	inbound := resp.GetMessages().GetMessages()[0]
	if inbound.GetIsOutbound() {
		t.Fatal("входящее сообщение помечено как исходящее")
	}
	if inbound.GetSenderNodeIpv6() != alice.node.NodeIPv6().String() {
		t.Fatalf("источник сообщения: %q, ожидался узел Алисы %q", inbound.GetSenderNodeIpv6(), alice.node.NodeIPv6())
	}

	// Событие в UI: тот же текст, подтверждённый Sealed Sender proof.
	ev := mustEvent(t, bob)
	in := ev.GetIncomingMessage()
	if in == nil {
		t.Fatalf("событие не IncomingMessage: %v", ev.GetEvent())
	}
	if in.GetMessage().GetText().GetBody() != body {
		t.Fatalf("событие принесло другой текст: %q", in.GetMessage().GetText().GetBody())
	}
	if !in.GetSealedSenderVerified() {
		t.Fatal("подпись отправителя в событии не подтверждена")
	}
	if ev.GetEventId() == 0 {
		t.Fatal("событие без идентификатора")
	}

	// Буквальный повтор тех же байтов рэтчет обязан отвергнуть: ключ сообщения
	// одноразовый, и второй заход — это replay, а не дубликат доставки.
	bob.handleInbound(pkt)
	if st := statsOf(t, bob); st.rejected != 1 || st.applied != 1 {
		t.Fatalf("replay принят как новое сообщение: %+v", st)
	}

	// А вот переотправка того же сообщения (тот же message_id, но новый шаг
	// отправной цепочки — так ведёт себя отправитель, когда WriteTo не прошёл)
	// раскрывается и гасится дедупликацией по идентификатору.
	repeat := wireOf(t, alice, bob, bundle, msg)
	bob.handleInbound(yggnode.Packet{From: alice.node.PublicKey(), Payload: marshalWire(t, repeat)})
	st := statsOf(t, bob)
	if st.applied != 1 || st.duplicated != 1 {
		t.Fatalf("статистика приёма %+v, ожидалось applied=1 duplicated=1", st)
	}
	if again := storedBodies(t, bob, conv); len(again) != 1 {
		t.Fatalf("после дубликата в базе %d сообщений", len(again))
	}
	assertNoMessageEvent(t, bob, "дубликат объявил новое событие")

	// One-time prekey израсходован: следующий бандл публикует другой ключ.
	next := bundleOf(t, bob)
	if next.GetKeyId() == bundle.GetKeyId() {
		t.Fatalf("one-time prekey не повернулся: ключ %d снова в бандле", next.GetKeyId())
	}
	if bytes.Equal(next.GetOneTimePrekeys()[0], bundle.GetOneTimePrekeys()[0]) {
		t.Fatal("новый one-time prekey совпадает с потраченным")
	}

	// Следом идёт второе сообщение того же диалога: сессия уже известна, X3DH
	// не повторяется, рэтчет шагает — и сообщение обязано долечь до базы.
	second := &yggproto.ChatMessage{
		MessageId:      "msg-rx-2",
		ConversationId: "диалог у алисы",
		TimestampMs:    1700000000500,
		Content:        textContent("продолжение того же рэтчета"),
	}
	bob.handleInbound(yggnode.Packet{From: alice.node.PublicKey(), Payload: marshalWire(t, wireOf(t, alice, bob, bundle, second))})
	if st := statsOf(t, bob); st.applied != 2 {
		t.Fatalf("второе сообщение не применено: %+v", st)
	}
	if got := storedBodies(t, bob, conv); len(got) != 2 || got[0] != "продолжение того же рэтчета" {
		t.Fatalf("после второго сообщения в базе %v", got)
	}
	if ev := mustEvent(t, bob); ev.GetIncomingMessage().GetMessage().GetMessageId() != "msg-rx-2" {
		t.Fatalf("событие о втором сообщении не то: %v", ev)
	}
}

// marshalWire оборачивает конверт в кадр оверлея — именно эти байты с Шага 6
// ложатся в сокет, поэтому и в тестовый пакет кладётся кадр, а не голый
// SealedEnvelope.
func marshalWire(t *testing.T, env *yggproto.SealedEnvelope) []byte {
	t.Helper()
	out, err := chatFrame(env)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// chatOf разворачивает кадр, который ядро отдало в сеть, обратно в разговорный
// конверт. Разбор делает deframe, а не proto.Unmarshal в SealedEnvelope: protobuf
// молча принял бы чужую оболочку, отдав в поле ключа содержимое всего кадра, и
// тест падал бы на «неверной длине эфемерного ключа» вместо честного
// «это не кадр».
func chatOf(t *testing.T, frame []byte) *yggproto.SealedEnvelope {
	t.Helper()
	parsed, err := deframe(frame)
	if err != nil {
		t.Fatal(err)
	}
	env := parsed.GetChat()
	if env == nil {
		t.Fatalf("в кадре нет разговорного конверта: %v", parsed.Body)
	}
	return env
}

// TestInboundRejectsForeignAndTampered — чужой конверт, перевёрнутый байт,
// отсутствие proof-of-work и неизвестный one-time prekey обязаны быть
// отброшены до хранилища: молча принять их нельзя, но и реагировать не на чем.
func TestInboundRejectsForeignAndTampered(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	carol := startEngine(t, seedCarol, t.TempDir())
	bundle := registerContact(t, alice, bob)
	carolBundle := registerContact(t, alice, carol)

	msg := &yggproto.ChatMessage{
		MessageId:      "msg-rx-bad",
		ConversationId: "диалог",
		TimestampMs:    1700000000000,
		Content:        textContent("не должно дожить до базы"),
	}
	good := wireOf(t, alice, bob, bundle, msg)

	// 1. Конверт, адресованный Карол: слепая почтовая коробка для Боба
	//    раскрываться не обязана.
	foreign := wireOf(t, alice, carol, carolBundle, msg)
	bob.handleInbound(yggnode.Packet{From: alice.node.PublicKey(), Payload: marshalWire(t, foreign)})

	// 2. Тот же конверт с перебитым байтом полезной нагрузки.
	corrupt := proto.Clone(good).(*yggproto.SealedEnvelope)
	data := append([]byte(nil), corrupt.GetEncryptedPayload()...)
	data[len(data)/2] ^= 1
	corrupt.EncryptedPayload = data
	bob.handleInbound(yggnode.Packet{From: alice.node.PublicKey(), Payload: marshalWire(t, corrupt)})

	// 3. Proof-of-work обнулён: спам-кандидат расшифровывать незачем.
	noPow := proto.Clone(good).(*yggproto.SealedEnvelope)
	noPow.PowNonce = 0
	bob.handleInbound(yggnode.Packet{From: alice.node.PublicKey(), Payload: marshalWire(t, noPow)})

	// 4. Настоящий proof, но one-time prekey, которого у Боба нет: X3DH
	//    собрать нечем, и сообщение не имеет права лечь в базу.
	fakeInner := encodeRatchetMessage(&crypto.EncryptedRatchetMessage{
		Header:     crypto.MessageHeader{DHPublicKey: alice.master.MasterX25519.PublicKey().Bytes()},
		Nonce:      make([]byte, 12),
		Ciphertext: []byte("проверка неизвестного prekey"),
	}, alice.master.MasterX25519.PublicKey().Bytes())
	recipientX, err := ecdh.X25519().NewPublicKey(bundle.GetIdentityX25519Pubkey())
	if err != nil {
		t.Fatal(err)
	}
	ghost, err := crypto.PackSealedEnvelope(proofOf(t, alice), deviceOf(t, alice), recipientX, fakeInner, "диалог", 999, powDifficulty)
	if err != nil {
		t.Fatalf("PackSealedEnvelope: %v", err)
	}
	bob.handleInbound(yggnode.Packet{
		From:    alice.node.PublicKey(),
		Payload: marshalWire(t, envelopeToWire(ghost)),
	})

	st := statsOf(t, bob)
	if st.received != 4 {
		t.Fatalf("получено %d пакетов, ожидалось 4", st.received)
	}
	if st.applied != 0 || st.duplicated != 0 {
		t.Fatalf("мусор попал в обработку: %+v", st)
	}
	if st.ignored != 2 || st.rejected != 2 {
		t.Fatalf("учёт отказов неточен: %+v", st)
	}
	if len(storedBodies(t, bob, inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes()))) != 0 {
		t.Fatal("отклонённое сообщение оказалось в хранилище")
	}
	assertNoMessageEvent(t, bob, "отклонённый пакет объявил событие")
}

func deviceOf(t *testing.T, e *Engine) *crypto.DeviceIdentity {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.device
}

func envelopeToWire(env *crypto.SealedEnvelope) *yggproto.SealedEnvelope {
	return &yggproto.SealedEnvelope{
		EphemeralX25519Pubkey: env.EphemeralX25519,
		EncryptedPayload:      env.EncryptedData,
		Nonce:                 env.Nonce,
		Timestamp:             uint64(env.Timestamp),
		PowNonce:              env.PoWNonce,
		BlindToken:            env.MailboxID[:],
	}
}

// TestIdentityProofBindsMasterKeys — конверт обязан доказывать, что Ed25519 и
// X25519 принадлежат одному владельцу, а устройство подписано этим же мастером.
// Без этой связки X3DH можно было бы провести от чужой идентичности.
func TestIdentityProofBindsMasterKeys(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	bundle := bundleOf(t, bob)
	recipientX, err := ecdh.X25519().NewPublicKey(bundle.GetIdentityX25519Pubkey())
	if err != nil {
		t.Fatal(err)
	}

	proof := proofOf(t, alice)
	if err := proof.Verify(); err != nil {
		t.Fatalf("настоящий proof не прошёл: %v", err)
	}

	spoofed := *proof
	spoofed.Binding = append([]byte(nil), proof.Binding...)
	spoofed.Binding[0] ^= 1
	if err := spoofed.Verify(); err == nil {
		t.Fatal("подделанная связка мастер-ключей прошла проверку")
	}
	if _, err := crypto.PackSealedEnvelope(&spoofed, deviceOf(t, alice), recipientX, []byte("тело"), "диалог", 0, powDifficulty); err == nil {
		t.Fatal("PackSealedEnvelope принял непроверенный proof")
	}

	// Мастер Алисы + устройство Боба: сертификат подписан другим ключом.
	foreignDevice := *proof
	bobProof := proofOf(t, bob)
	foreignDevice.DeviceID = bobProof.DeviceID
	foreignDevice.DeviceName = bobProof.DeviceName
	foreignDevice.DeviceEd25519 = bobProof.DeviceEd25519
	foreignDevice.DeviceX25519 = bobProof.DeviceX25519
	foreignDevice.DeviceCreatedAt = bobProof.DeviceCreatedAt
	foreignDevice.DeviceCertSig = bobProof.DeviceCertSig
	if err := foreignDevice.Verify(); err == nil {
		t.Fatal("сертификат чужого устройства принят")
	}

	// А вот собранный по-настоящему конверт раскрывается и несёт ровно те
	// данные, по которым получатель строит диалог.
	env, err := crypto.PackSealedEnvelope(proof, deviceOf(t, alice), recipientX, []byte("тело"), "диалог", bundle.GetKeyId(), powDifficulty)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := crypto.UnpackSealedEnvelope(bob.master.MasterX25519, env)
	if err != nil {
		t.Fatalf("UnpackSealedEnvelope: %v", err)
	}
	if payload.ConversationID != "диалог" || payload.OneTimePreKeyID != bundle.GetKeyId() {
		t.Fatalf("в конверте не то имя диалога или не тот prekey: %+v", payload)
	}
	if err := payload.Proof.Verify(); err != nil {
		t.Fatalf("proof внутри конверта не прошёл: %v", err)
	}
}

// TestReceiveOverRealOverlay — «два клиента, одно доставленное сообщение»:
// пакет идёт по настоящему TCP-оверлею, приёмный цикл Боба сам расшифровывает
// его, кладёт в базу и объявляет событие.
func TestReceiveOverRealOverlay(t *testing.T) {
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
	bundle := registerContact(t, alice, bob)

	waitForRoute(t, alice, bob.node.PublicKey())

	const body = "оверлей донёс до приёмника"
	msg := &yggproto.ChatMessage{
		MessageId:      "msg-e2e-1",
		ConversationId: "диалог через сеть",
		TimestampMs:    1700000000000,
		IsOutbound:     true,
		Content:        textContent(body),
	}
	conv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	deadline := time.Now().Add(45 * time.Second)
	for attempts := 1; ; attempts++ {
		resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
			SendMessage: &yggproto.SendMessageReq{
				DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
				Message:                 msg,
			},
		}})
		if !resp.GetSuccess() {
			t.Fatalf("SendMessage (попытка %d): %s", attempts, resp.GetErrorMessage())
		}
		if resp.GetSendMessage().GetWrittenToOverlay() &&
			waitUntil(t, 3*time.Second, func() bool { return len(storedBodies(t, bob, conv)) == 1 }) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("за %d попыток сообщение не дошло до приёмного цикла", attempts)
		}
	}

	if got := storedBodies(t, bob, conv); got[0] != body {
		t.Fatalf("Боб прочитал %q", got[0])
	}
	ev := mustEvent(t, bob)
	in := ev.GetIncomingMessage()
	if in == nil || in.GetMessage().GetText().GetBody() != body {
		t.Fatalf("событие о доставке не то: %v", ev)
	}
	if !in.GetSealedSenderVerified() {
		t.Fatal("доставленное по сети сообщение не подтверждено proof'ом")
	}
	if statsOf(t, bob).rejected != 0 {
		t.Fatalf("приём сети вернул отказы: %+v", statsOf(t, bob))
	}
}

// TestDeliveryAckOverRealOverlay — сквозное подтверждение доставки. Боб шлёт
// ACK тем же сеансом Double Ratchet, что открыл сообщение, и Алиса обязана
// принять его без единого отказа рэтчета: только так «доставлено» перестаёт быть
// догадкой о том, что сделал оверлей.
func TestDeliveryAckOverRealOverlay(t *testing.T) {
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
	bundle := registerContact(t, alice, bob)

	// Маршрут нужен в обе стороны: без обратного ACK не уйдёт, и счётчик
	// ackSkipped это честно покажет.
	waitForRoute(t, alice, bob.node.PublicKey())
	waitForRoute(t, bob, alice.node.PublicKey())

	const body = "подтверждение дошло по сети"
	const conv = "диалог с подтверждением"
	msg := &yggproto.ChatMessage{
		MessageId:      "msg-ack-1",
		ConversationId: conv,
		TimestampMs:    1700000000000,
		IsOutbound:     true,
		Content:        textContent(body),
	}
	resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{
			DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
			Message:                 msg,
		},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("SendMessage: %s", resp.GetErrorMessage())
	}

	// Первое событие на стороне Алисы — пакет ушёл в оверлей. Оно намеренно не
	// называет сообщение доставленным.
	sent := nextEventOf(t, alice, 5*time.Second)
	status := sent.GetMessageStatus()
	if status == nil {
		t.Fatalf("ожидание MessageStatusEvent, пришло: %v", sent.GetEvent())
	}
	if status.GetMessageId() != "msg-ack-1" || !status.GetWrittenToOverlay() || status.GetDelivered() {
		t.Fatalf("статус отправки неверный: %+v", status)
	}
	if deliveredOf(t, alice, conv, "msg-ack-1") {
		t.Fatal("сообщение стало доставленным до подтверждения получателя")
	}

	// Первый пакет оверлей теряет чаще всего (сессия ironwood ещё прогревается),
	// поэтому переотправляем тем же message_id: рэтчет шагнёт, а Боб погасит
	// повтор дедупликацией и не пришлёт второй ACK.
	bobConv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	deadline := time.Now().Add(45 * time.Second)
	for attempts := 1; ; attempts++ {
		if waitUntil(t, 5*time.Second, func() bool { return len(storedBodies(t, bob, bobConv)) == 1 }) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("за %d попыток сообщение не дошло: alice=%+v bob=%+v",
				attempts, statsOf(t, alice), statsOf(t, bob))
		}
		if !alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
			SendMessage: &yggproto.SendMessageReq{
				DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
				Message:                 msg,
			},
		}}).GetSuccess() {
			t.Fatalf("переотправка %d не прошла", attempts)
		}
	}

	// ACK идёт по сети, поэтому ждём, а не читаем сразу.
	if !waitUntil(t, 30*time.Second, func() bool { return deliveredOf(t, alice, conv, "msg-ack-1") }) {
		t.Fatalf("доставка не подтверждена: alice=%+v bob=%+v", statsOf(t, alice), statsOf(t, bob))
	}

	acked := awaitStatus(t, alice, 5*time.Second, func(s *yggproto.MessageStatusEvent) bool {
		return s.GetDelivered()
	})
	if acked.GetMessageId() != "msg-ack-1" || acked.GetConversationId() != conv {
		t.Fatalf("событие о доставке не то: %+v", acked)
	}
	if acked.GetAckTimestampMs() == 0 {
		t.Fatal("ACK без метки времени")
	}

	bobStats := statsOf(t, bob)
	if bobStats.ackSent != 1 || bobStats.ackSkipped != 0 {
		t.Fatalf("Боб отправил подтверждений: %d, пропустил: %d", bobStats.ackSent, bobStats.ackSkipped)
	}
	if bobStats.rejected != 0 {
		t.Fatalf("приём у Боба с отказами: %+v", bobStats)
	}
	// Смысл единой карты сеансов: ACK раскрывается сеансом, который Алиса
	// завела для X3DH. Любая размолвка рэтчета попала бы в rejected.
	aliceStats := statsOf(t, alice)
	if aliceStats.acked != 1 || aliceStats.rejected != 0 || aliceStats.ignored != 0 {
		t.Fatalf("статистики Алисы не чистые: %+v", aliceStats)
	}

	// ACK — служебное сообщение: строкой диалога оно стать не должно.
	if got := storedBodies(t, bob, inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())); len(got) != 1 || got[0] != body {
		t.Fatalf("в диалоге Боба %v, ожидался один текст", got)
	}
	if got := storedBodies(t, alice, conv); len(got) != 1 {
		t.Fatalf("в диалоге Алисы %d сообщений, ожидалось 1", len(got))
	}
}

// ---------------------------------------------------------------------------
// Шаг 3-c-2: исходящая очередь (нет маршрута → конверт ждёт пира)
// ---------------------------------------------------------------------------

func outStatsOf(t *testing.T, e *Engine) outboxStats {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.outStats
}

// queuedOutbound — сколько конвертов всё ещё ждут маршрута в зашифрованной базе.
func queuedOutbound(t *testing.T, e *Engine) int {
	t.Helper()
	return len(queuedSnapshot(t, e))
}

// queuedSnapshot снимает копию очереди под локом: сканер живёт в своей горутине,
// и без блокировки чтение было бы гонкой.
func queuedSnapshot(t *testing.T, e *Engine) []storage.StoredOutbound {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.db == nil {
		return nil
	}
	return e.db.PendingOutbound()
}

// TestOutboxResendsWhenRouteAppears — смысл Шага 3-c-2: сообщение, у которого в
// момент отправки не было маршрута, не теряется и не требует второго
// SendMessage. Байты конверта лежат в зашифрованной очереди, и сканер отправляет
// их, как только оверлей узнает адресата, — ровно теми байтами, что были собраны
// при первой попытке (повторный seal двигал бы отправную цепочку ради пакетов,
// которые никуда не ушли).
func TestOutboxResendsWhenRouteAppears(t *testing.T) {
	port := freeTCPPort(t)
	// Боб слушает, Алиса стартует без единого пира: маршрута до ключа Боба у неё
	// нет, и первое же сообщение обязано лечь в исходящую очередь.
	bob := startEngineWith(t, seedBob, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(port)
	})
	alice := startEngineWith(t, seedAlice, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
	})
	bundle := registerContact(t, alice, bob)

	const body = "дождалось маршрута"
	const conv = "диалог из очереди"
	msg := &yggproto.ChatMessage{
		MessageId:      "msg-outbox-1",
		ConversationId: conv,
		TimestampMs:    1700000000000,
		IsOutbound:     true,
		Content:        textContent(body),
	}
	resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{
			DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
			Message:                 msg,
		},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("SendMessage: %s", resp.GetErrorMessage())
	}
	sent := resp.GetSendMessage()
	if sent.GetWrittenToOverlay() || !sent.GetQueuedInOutbox() {
		t.Fatalf("ответ без маршрута неверный: %+v", sent)
	}
	if n := queuedOutbound(t, alice); n != 1 {
		t.Fatalf("в очереди %d конвертов, ожидался 1", n)
	}
	if st := outStatsOf(t, alice); st.queued != 1 || st.written != 0 {
		t.Fatalf("учёт очереди неверный: %+v", st)
	}
	// У получателя пусто: без маршрута уходить нечему.
	bobConv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	if got := storedBodies(t, bob, bobConv); len(got) != 0 {
		t.Fatalf("сообщение без маршрута дошло до Боба: %v", got)
	}

	// Пир появляется уже после отправки — тот самый случай, ради которого очередь
	// и заведена. Никакого повторного SendMessage: только ConnectPeer, и сканер
	// обязан переотложить байты сам.
	peerResp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_ConnectPeer{
		ConnectPeer: &yggproto.ConnectPeerReq{PeerUri: fmt.Sprintf("127.0.0.1:%d", port)},
	}})
	if !peerResp.GetSuccess() {
		t.Fatalf("ConnectPeer: %s", peerResp.GetErrorMessage())
	}

	if !waitUntil(t, 45*time.Second, func() bool {
		return len(storedBodies(t, bob, bobConv)) == 1
	}) {
		t.Fatalf("очередь не переотправилась: alice=%+v outbox=%+v bob=%+v",
			statsOf(t, alice), outStatsOf(t, alice), statsOf(t, bob))
	}

	// Один дубль допустим (первый пакет на холодной сессии ironwood теряется
	// регулярно), а очередь после него пуста: бесконечно слать те же байты
	// нельзя — получатель раскроет только первую копию.
	if !waitUntil(t, 10*time.Second, func() bool { return queuedOutbound(t, alice) == 0 }) {
		t.Fatalf("ушедшее сообщение осталось в очереди: %d", queuedOutbound(t, alice))
	}
	st := outStatsOf(t, alice)
	if st.queued != 1 || st.written < 1 || st.written > outboxMaxSends || st.invalid != 0 {
		t.Fatalf("учёт переотправки неверный: %+v (лимит %d)", st, outboxMaxSends)
	}
	if got := storedBodies(t, bob, bobConv); len(got) != 1 || got[0] != body {
		t.Fatalf("Боб прочитал %v", got)
	}

	// Статус «ушло в сеть» приходит из сканера, а не из SendMessage: именно это
	// и видит UI, когда пир поднимается после отправки.
	status := nextEventOf(t, alice, 5*time.Second)
	ms := status.GetMessageStatus()
	if ms == nil || ms.GetMessageId() != "msg-outbox-1" || !ms.GetWrittenToOverlay() || ms.GetDelivered() {
		t.Fatalf("сканер не объявил written_to_overlay: %v", status.GetEvent())
	}

	// Дословная переотправка не ломает рэтчет: сообщение раскрыто с первого
	// attempts, ACK вернулся, и «доставлено» поставлено по-настоящему.
	if !waitUntil(t, 30*time.Second, func() bool { return deliveredOf(t, alice, conv, "msg-outbox-1") }) {
		t.Fatalf("доставка не подтверждена: alice=%+v outbox=%+v bob=%+v",
			statsOf(t, alice), outStatsOf(t, alice), statsOf(t, bob))
	}
	if got := statsOf(t, alice); got.acked != 1 || got.rejected != 0 || got.ignored != 0 {
		t.Fatalf("статистика Алисы после переотправки: %+v", got)
	}
	if got := statsOf(t, bob); got.applied != 1 || got.ackSent != 1 || got.ackSkipped != 0 {
		t.Fatalf("статистика Боба после переотправки: %+v", got)
	}
}

// TestOutboxSurvivesRestart — неотправленный конверт обязан пережить остановку
// ядра: без этого перезапуск приложения означал бы молчаливую потерю того, что
// пользователь уже напечатал.
func TestOutboxSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	e := New()
	start := func() *yggproto.CoreResponse {
		return e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StartNode{
			StartNode: &yggproto.StartNodeReq{Config: &yggproto.NodeConfig{
				DataDir:      dir + "/data",
				CacheDir:     dir + "/cache",
				MnemonicSeed: seedAlice,
				// Ни слушателя, ни пира: маршрута до контакта не будет никогда, и
				// сообщение гарантированно остаётся в очереди.
				AntiDpiMode: yggproto.AntiDpiMode_ANTI_DPI_DISABLED,
			}},
		}})
	}
	if resp := start(); !resp.GetSuccess() {
		t.Fatalf("StartNode: %s", resp.GetErrorMessage())
	}
	t.Cleanup(func() {
		e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}})
	})

	// Бандл берём у живой ноды: сборный X3DH инициатора обязан состояться, иначе
	// в очередь ложить нечего.
	peer := startEngine(t, seedBob, t.TempDir())
	bundle := bundleOf(t, peer)
	addResp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			Ed25519Pubkey: peer.node.PublicKey(),
			X25519Pubkey:  bundle.GetIdentityX25519Pubkey(),
			Bundle:        bundle,
		}},
	}})
	if !addResp.GetSuccess() {
		t.Fatalf("AddContact: %s", addResp.GetErrorMessage())
	}

	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{
			DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
			Message: &yggproto.ChatMessage{
				MessageId:      "msg-restart",
				ConversationId: "диалог до перезапуска",
				TimestampMs:    1700000000000,
				Content:        textContent("пережить рестарт"),
			},
		},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("SendMessage: %s", resp.GetErrorMessage())
	}
	sent := resp.GetSendMessage()
	if sent.GetWrittenToOverlay() || !sent.GetQueuedInOutbox() {
		t.Fatalf("изолированная нода отчиталась об отправке: %+v", sent)
	}
	env := append([]byte(nil), sent.GetSealedEnvelope()...)

	if !e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}}).GetSuccess() {
		t.Fatal("StopNode не прошёл")
	}
	if resp := start(); !resp.GetSuccess() {
		t.Fatalf("повторный StartNode: %s", resp.GetErrorMessage())
	}

	pending := queuedSnapshot(t, e)
	if len(pending) != 1 || pending[0].MessageID != "msg-restart" {
		t.Fatalf("после перезапуска очередь %+v", pending)
	}
	if !bytes.Equal(pending[0].Envelope, env) {
		t.Fatal("восстановлен другой конверт: байты изменились")
	}
	if !bytes.Equal(pending[0].NodeKey, peer.node.PublicKey()) {
		t.Fatalf("ключ ноды в очереди потерян: %x", pending[0].NodeKey)
	}
	if pending[0].ConversationID != "диалог до перезапуска" || pending[0].Sends != 0 {
		t.Fatalf("запись очереди восстановлена неполно: %+v", pending[0])
	}
}

// ---------------------------------------------------------------------------
// Шаг 3-c-3: сеансы Double Ratchet переживают рестарт ядра
// ---------------------------------------------------------------------------

func sessStatsOf(t *testing.T, e *Engine) sessionStats {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sessStats
}

// hasBody смотрит, лежит ли строка в диалоге получателя.
func hasBody(t *testing.T, e *Engine, conv, body string) bool {
	t.Helper()
	for _, got := range storedBodies(t, e, conv) {
		if got == body {
			return true
		}
	}
	return false
}

// TestSessionsSurviveRestartAndDialogContinues — сквозной смысл персистентности
// сеансов: оба ядра перезапускаются, а диалог продолжается без единого отказа
// рэтчета. Без записанного состояния второй шаг цепочки не открылся бы никогда:
// собеседник ждал бы номер, который у другой стороны уже сдвинут.
func TestSessionsSurviveRestartAndDialogContinues(t *testing.T) {
	port := freeTCPPort(t)
	aliceDir, bobDir := t.TempDir(), t.TempDir()

	aliceCfg := &yggproto.NodeConfig{
		DataDir:       aliceDir + "/data",
		CacheDir:      aliceDir + "/cache",
		MnemonicSeed:  seedAlice,
		AntiDpiMode:   yggproto.AntiDpiMode_ANTI_DPI_DISABLED,
		ListenAddress: "127.0.0.1",
		ListenPort:    int32(port),
	}
	bobCfg := &yggproto.NodeConfig{
		DataDir:        bobDir + "/data",
		CacheDir:       bobDir + "/cache",
		MnemonicSeed:   seedBob,
		AntiDpiMode:    yggproto.AntiDpiMode_ANTI_DPI_DISABLED,
		BootstrapPeers: []string{fmt.Sprintf("127.0.0.1:%d", port)},
	}

	var launched []*Engine
	start := func(cfg *yggproto.NodeConfig) *Engine {
		t.Helper()
		e := New()
		resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StartNode{
			StartNode: &yggproto.StartNodeReq{Config: cfg},
		}})
		if !resp.GetSuccess() {
			t.Fatalf("StartNode: %s", resp.GetErrorMessage())
		}
		launched = append(launched, e)
		return e
	}
	stop := func(e *Engine) {
		t.Helper()
		if !e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}}).GetSuccess() {
			t.Fatal("StopNode не прошёл")
		}
	}
	t.Cleanup(func() {
		for _, e := range launched {
			e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}})
		}
	})

	// deliver переотправляет сообщение тем же message_id, пока адресат не
	// положит его в диалог: первый пакет на ещё не прогретой сессии ironwood
	// оверлей теряет регулярно, а дедупликация по message_id гасит повторы.
	deliver := func(from *Engine, destX []byte, msg *yggproto.ChatMessage, to *Engine, toConv, body string) {
		t.Helper()
		deadline := time.Now().Add(60 * time.Second)
		for attempts := 1; ; attempts++ {
			if !from.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
				SendMessage: &yggproto.SendMessageReq{DestinationX25519Pubkey: destX, Message: msg},
			}}).GetSuccess() {
				t.Fatalf("SendMessage %s (попытка %d) не прошёл", msg.GetMessageId(), attempts)
			}
			if waitUntil(t, 5*time.Second, func() bool { return hasBody(t, to, toConv, body) }) {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s не дошло за %d попыток: from=%+v to=%+v",
					msg.GetMessageId(), attempts, statsOf(t, from), statsOf(t, to))
			}
		}
	}

	alice, bob := start(aliceCfg), start(bobCfg)
	bundleBob := registerContact(t, alice, bob)
	registerContact(t, bob, alice)
	waitForRoute(t, alice, bob.node.PublicKey())
	waitForRoute(t, bob, alice.node.PublicKey())

	const conv = "диалог через рестарт"
	bobConv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	aliceConv := inboundConversationID(bob.master.MasterX25519.PublicKey().Bytes())

	textMsg := func(id, body string) *yggproto.ChatMessage {
		return &yggproto.ChatMessage{
			MessageId:      id,
			ConversationId: conv,
			TimestampMs:    1700000000000,
			Content:        textContent(body),
		}
	}

	deliver(alice, bundleBob.GetIdentityX25519Pubkey(), textMsg("msg-before", "до перезапуска"),
		bob, bobConv, "до перезапуска")
	if !waitUntil(t, 30*time.Second, func() bool { return deliveredOf(t, alice, conv, "msg-before") }) {
		t.Fatalf("первое сообщение не подтверждено доставкой: alice=%+v bob=%+v",
			statsOf(t, alice), statsOf(t, bob))
	}
	// Диалог живёт, пока живёт сеанс: до рестарта обе стороны успели сдвинуть
	// цепочки, и это состояние теперь должно пережить перезапуск.
	if sessStatsOf(t, alice).saved == 0 || sessStatsOf(t, bob).saved == 0 {
		t.Fatalf("шаги рэтчета не легли в хранилище: alice=%+v bob=%+v",
			sessStatsOf(t, alice), sessStatsOf(t, bob))
	}

	stop(alice)
	stop(bob)
	alice, bob = start(aliceCfg), start(bobCfg)

	// Контакты ядро держит в памяти, поэтому карта возвращается заново — но
	// самих сеансов в ней быть не должно: их поднимают из базы.
	bundleBob = registerContact(t, alice, bob)
	bundleAlice := registerContact(t, bob, alice)
	waitForRoute(t, alice, bob.node.PublicKey())
	waitForRoute(t, bob, alice.node.PublicKey())

	for _, tc := range []struct {
		who string
		e   *Engine
	}{{"Алиса", alice}, {"Боб", bob}} {
		st := sessStatsOf(t, tc.e)
		if st.restored == 0 {
			t.Fatalf("%s: после перезапуска не поднято ни одного сеанса (%+v)", tc.who, st)
		}
		if st.restoreErr != 0 || st.skippedOwner != 0 {
			t.Fatalf("%s: состояние сеансов разобралось с ошибками: %+v", tc.who, st)
		}
	}

	// Сообщения, записанные до перезапуска, на месте — база та же.
	if !hasBody(t, bob, bobConv, "до перезапуска") {
		t.Fatalf("диалог Боба после перезапуска потерял старое сообщение: %v", storedBodies(t, bob, bobConv))
	}

	// Разговор идёт дальше в обе стороны тем же сеансом.
	deliver(alice, bundleBob.GetIdentityX25519Pubkey(), textMsg("msg-after", "после перезапуска"),
		bob, bobConv, "после перезапуска")
	if !waitUntil(t, 30*time.Second, func() bool { return deliveredOf(t, alice, conv, "msg-after") }) {
		t.Fatalf("сообщение после рестарта не подтверждено: alice=%+v bob=%+v",
			statsOf(t, alice), statsOf(t, bob))
	}

	// Ответ Боба уходит в том же имени диалога, что выбрала Алиса: по нему она
	// и находит восстановленный сеанс.
	reply := textMsg("msg-reply", "ответ после перезапуска")
	deliver(bob, bundleAlice.GetIdentityX25519Pubkey(), reply, alice, aliceConv, "ответ после перезапуска")
	if !waitUntil(t, 30*time.Second, func() bool { return deliveredOf(t, bob, conv, "msg-reply") }) {
		t.Fatalf("ответ после рестарта не подтверждён: bob=%+v alice=%+v",
			statsOf(t, bob), statsOf(t, alice))
	}

	// Главный признак целости цепочек: после рестарта ни одна сторона не
	// отвергла ни одного пакета. Рассинхрон рэтчета выглядел бы именно здесь.
	for _, tc := range []struct {
		who string
		e   *Engine
	}{{"Алиса", alice}, {"Боб", bob}} {
		st := statsOf(t, tc.e)
		if st.rejected != 0 || st.ignored != 0 {
			t.Fatalf("%s после рестарта отвергла пакеты: %+v", tc.who, st)
		}
		if st.applied < 1 {
			t.Fatalf("%s после рестарта ничего не приняла: %+v", tc.who, st)
		}
	}
}

// TestSessionsOfAnotherIdentityAreNotRestored — состояние сеанса привязано к
// мастер-ключу этого узла. Смена фразы обычно даже не открывает базу (ключ
// хранилища выводится из seed), но при явном master_key_256 одна и та же база
// доступна разным личностям: старую строку обязано отбросить по владельцу, иначе
// расшифровка пошла бы ключами чужой личности.
func TestSessionsOfAnotherIdentityAreNotRestored(t *testing.T) {
	dir := t.TempDir()
	vaultKey := bytes.Repeat([]byte("k"), 32)
	peer := startEngine(t, seedBob, t.TempDir())
	bundle := bundleOf(t, peer)

	run := func(mnemonic string) *Engine {
		t.Helper()
		e := New()
		resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StartNode{
			StartNode: &yggproto.StartNodeReq{Config: &yggproto.NodeConfig{
				DataDir:       dir + "/data",
				CacheDir:      dir + "/cache",
				MnemonicSeed:  mnemonic,
				MasterKey_256: vaultKey,
				// Ни слушателя, ни пира: маршрут не появится, и сеанс
				// переживает перезапуск только в базе.
				AntiDpiMode: yggproto.AntiDpiMode_ANTI_DPI_DISABLED,
			}},
		}})
		if !resp.GetSuccess() {
			t.Fatalf("StartNode (%s): %s", mnemonic, resp.GetErrorMessage())
		}
		return e
	}

	alice := run(seedAlice)
	defer func() {
		alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}})
	}()
	if resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			Ed25519Pubkey: peer.node.PublicKey(),
			X25519Pubkey:  bundle.GetIdentityX25519Pubkey(),
			Bundle:        bundle,
		}},
	}}); !resp.GetSuccess() {
		t.Fatalf("AddContact: %s", resp.GetErrorMessage())
	}
	if resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{
			DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
			Message: &yggproto.ChatMessage{
				MessageId:      "msg-owner",
				ConversationId: "диалог владельца",
				TimestampMs:    1700000000000,
				Content:        textContent("сеанс создан"),
			},
		},
	}}); !resp.GetSuccess() {
		t.Fatalf("SendMessage: %s", resp.GetErrorMessage())
	}
	if sessStatsOf(t, alice).saved == 0 {
		t.Fatal("сеанс не лёг в хранилище")
	}
	stopResp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}})
	if !stopResp.GetSuccess() {
		t.Fatal("StopNode не прошёл")
	}

	other := run(seedCarol)
	defer func() {
		other.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}})
	}()

	st := sessStatsOf(t, other)
	if st.restored != 0 {
		t.Fatalf("чужой сеанс поднят другой идентичностью: %+v", st)
	}
	if st.skippedOwner == 0 {
		t.Fatalf("строка другой идентичности не замечена: %+v", st)
	}
	if st.restoreErr != 0 {
		t.Fatalf("разбор состояния прошёл с ошибкой: %+v", st)
	}
	// Тот же диалог под новой личностью обязан завести новый сеанс, а не
	// продолжить старый.
	if resp := other.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			Ed25519Pubkey: peer.node.PublicKey(),
			X25519Pubkey:  bundle.GetIdentityX25519Pubkey(),
			Bundle:        bundle,
		}},
	}}); !resp.GetSuccess() {
		t.Fatalf("AddContact второй идентичности: %s", resp.GetErrorMessage())
	}
	if resp := other.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{
			DestinationX25519Pubkey: bundle.GetIdentityX25519Pubkey(),
			Message: &yggproto.ChatMessage{
				MessageId:      "msg-owner-2",
				ConversationId: "диалог владельца",
				TimestampMs:    1700000000001,
				Content:        textContent("сеанс создан заново"),
			},
		},
	}}); !resp.GetSuccess() {
		t.Fatalf("SendMessage второй идентичности: %s", resp.GetErrorMessage())
	}
	if got := sessStatsOf(t, other); got.saved == 0 {
		t.Fatal("новый сеанс второй идентичности не записан")
	}
}

// sessionCount — сколько сеансов живёт в ядре: отказ по несовпадению узла не
// имеет права заводить новый.
func sessionCount(t *testing.T, e *Engine) int {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.sessions)
}

// TestInboundFromKnownContactRequiresCertifiedNodeKey — карта контакта заверяет
// связку «идентичность ↔ узел» не только в момент AddContact: на приёме она
// обязана держаться. Пакет, где подпись знакомой идентичности, а узел другой, —
// это разъезд адреса доставки, и принимать его нельзя. Отказ обязан случиться до
// раскрытия: приёмная цепочка не сдвигается, сообщение не сгорает, сеанс не
// заводится.
func TestInboundFromKnownContactRequiresCertifiedNodeKey(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	bundle := registerContact(t, alice, bob)
	registerContact(t, bob, alice)

	conv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	envOf := func(id, wireConv, body string, ts uint64) *yggproto.SealedEnvelope {
		return wireOf(t, alice, bob, bundle, &yggproto.ChatMessage{
			MessageId:      id,
			ConversationId: wireConv,
			TimestampMs:    ts,
			IsOutbound:     true,
			Content:        textContent(body),
		})
	}

	honest := yggnode.Packet{
		From:    alice.node.PublicKey(),
		Payload: marshalWire(t, envOf("msg-node-bind-1", "у алисы", "с заверенного узла", 1700000000000)),
	}
	bob.handleInbound(honest)
	if got := storedBodies(t, bob, conv); len(got) != 1 || got[0] != "с заверенного узла" {
		t.Fatalf("базовая доставка не работает: %v", got)
	}

	// Второй диалог той же отправительницы, но байты приходят с чужого узла.
	before := sessionCount(t, bob)
	second := marshalWire(t, envOf("msg-node-bind-2", "у алисы второй", "с незаверенного узла", 1700000000001))
	bob.handleInbound(yggnode.Packet{From: freshNodeKey(t), Payload: second})

	if st := statsOf(t, bob); st.routeMismatch != 1 {
		t.Fatalf("разъезд узла не учтён: %+v", st)
	}
	if now := sessionCount(t, bob); now != before {
		t.Fatalf("отказ завёл сеанс: было %d, стало %d", before, now)
	}
	if got := storedBodies(t, bob, conv); len(got) != 1 {
		t.Fatalf("отклонённый пакет что-то оставил в базе: %v", got)
	}

	// Ровно те же байты, пришедшие с заверенного узла, обязаны открыться:
	// отказ не израсходовал ни номер сообщения, ни шаг цепочки.
	bob.handleInbound(yggnode.Packet{From: alice.node.PublicKey(), Payload: second})
	got := storedBodies(t, bob, conv)
	if len(got) != 2 || !hasBody(t, bob, conv, "с незаверенного узла") {
		t.Fatalf("повтор с честного узла не принят: %v", got)
	}
	if now := sessionCount(t, bob); now != before+1 {
		t.Fatalf("сеанс второго диалога не заведён: %d", now)
	}
	if st := statsOf(t, bob); st.routeMismatch != 1 {
		t.Fatalf("честный пакет засчитан как разъезд: %+v", st)
	}
}

// TestInboundFromStrangerKeepsWorking — проверка узла касается только тех, чью
// карту уже приняли. Иначе первое сообщение от незнакомца стало бы недоступным,
// а офлайн-старт X3DH существует именно для него.
func TestInboundFromStrangerKeepsWorking(t *testing.T) {
	alice := startEngine(t, seedAlice, t.TempDir())
	bob := startEngine(t, seedBob, t.TempDir())
	carol := startEngine(t, seedCarol, t.TempDir())
	registerContact(t, bob, carol)
	bundle := registerContact(t, alice, bob)

	env := wireOf(t, alice, bob, bundle, &yggproto.ChatMessage{
		MessageId:      "msg-stranger-1",
		ConversationId: "у алисы",
		TimestampMs:    1700000000000,
		IsOutbound:     true,
		Content:        textContent("первое сообщение от незнакомца"),
	})
	bob.handleInbound(yggnode.Packet{From: alice.node.PublicKey(), Payload: marshalWire(t, env)})

	conv := inboundConversationID(alice.master.MasterX25519.PublicKey().Bytes())
	if got := storedBodies(t, bob, conv); len(got) != 1 {
		t.Fatalf("сообщение от незнакомца отброшено: %v (%+v)", got, statsOf(t, bob))
	}
	if st := statsOf(t, bob); st.routeMismatch != 0 {
		t.Fatalf("незнакомец засчитан как разъезд узла: %+v", st)
	}
}

// ---------------------------------------------------------------------------
// Шаг 5-b: карты контактов переживают рестарт ядра
// ---------------------------------------------------------------------------

// startIsolated поднимает ядро без слушателя и без пира: маршрута до контакта
// тут не будет никогда, поэтому отправленное сообщение гарантированно остаётся
// в очереди, а не уходит в сеть.
func startIsolated(t *testing.T, e *Engine, dir, mnemonic string) {
	t.Helper()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StartNode{
		StartNode: &yggproto.StartNodeReq{Config: &yggproto.NodeConfig{
			DataDir:      dir + "/data",
			CacheDir:     dir + "/cache",
			MnemonicSeed: mnemonic,
			AntiDpiMode:  yggproto.AntiDpiMode_ANTI_DPI_DISABLED,
		}},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("StartNode: %s", resp.GetErrorMessage())
	}
}

func stopIsolated(t *testing.T, e *Engine) {
	t.Helper()
	if !e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}}).GetSuccess() {
		t.Fatal("StopNode не прошёл")
	}
}

// sendQueued — SendMessage изолированным ядром: сообщение обязано честно
// встать в очередь, а не притвориться отправленным.
func sendQueued(t *testing.T, e *Engine, dest []byte, msg *yggproto.ChatMessage) *yggproto.SendMessageResp {
	t.Helper()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_SendMessage{
		SendMessage: &yggproto.SendMessageReq{DestinationX25519Pubkey: dest, Message: msg},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("SendMessage: %s", resp.GetErrorMessage())
	}
	sent := resp.GetSendMessage()
	if sent.GetWrittenToOverlay() || !sent.GetQueuedInOutbox() {
		t.Fatalf("изолированная нода отчиталась об отправке: %+v", sent)
	}
	return sent
}

// contactsOf — список контактов через FFI-команду: тест видит ровно то же, что
// видит приложение, а не внутреннюю карту движка.
func contactsOf(t *testing.T, e *Engine) []*yggproto.ContactCard {
	t.Helper()
	resp := e.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_ListContacts{}})
	if !resp.GetSuccess() {
		t.Fatalf("ListContacts: %s", resp.GetErrorMessage())
	}
	return resp.GetContacts().GetContacts()
}

func contactStatsOf(t *testing.T, e *Engine) contactStats {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.contactStats
}

// TestAddContactRequiresStartedCore — карта кладётся в то же хранилище, что и
// сообщения, а оно открывается вместе с идентичностью в StartNode. До запуска
// непонятно, к кому относится контакт, поэтому его не принимают.
func TestAddContactRequiresStartedCore(t *testing.T) {
	e := New()
	bobNode := freshNodeKey(t)
	msg := addContactErr(t, e, cardOf(foreignBundle(t, seedBob, bobNode)))
	if !strings.Contains(msg, "не запущено") {
		t.Fatalf("AddContact без StartNode принят: %q", msg)
	}
	if got := contactsOf(t, e); len(got) != 0 {
		t.Fatalf("контакт появился без запущенного ядра: %d", len(got))
	}
}

// TestContactsSurviveRestartAndSendStillWorks — смысл Шага 5-b: диалог живёт
// дольше процесса ядра. До этого e.contacts был map в памяти, и после
// перезапуска сообщение падало на «контакта нет», хотя сохранённый диалог
// никуда не делся.
func TestContactsSurviveRestartAndSendStillWorks(t *testing.T) {
	dir := t.TempDir()
	e := New()
	startIsolated(t, e, dir, seedAlice)
	t.Cleanup(func() { stopIsolated(t, e) })

	peer := startEngine(t, seedBob, t.TempDir())
	bundle := bundleOf(t, peer)
	if msg := addContactErr(t, e, cardOf(bundle)); msg != "" {
		t.Fatalf("AddContact: %s", msg)
	}
	if st := contactStatsOf(t, e); st.persisted != 1 {
		t.Fatalf("карта не легла в хранилище: %+v", st)
	}

	sendQueued(t, e, bundle.GetIdentityX25519Pubkey(), &yggproto.ChatMessage{
		MessageId:      "msg-5b-первое",
		ConversationId: "диалог до перезапуска",
		TimestampMs:    1700000000000,
		Content:        textContent("до рестарта"),
	})

	stopIsolated(t, e)
	startIsolated(t, e, dir, seedAlice)

	cards := contactsOf(t, e)
	if len(cards) != 1 {
		t.Fatalf("после перезапуска контактов %d, ожидался 1", len(cards))
	}
	card := cards[0]
	// Карта обязана вернуться дословной: адрес узла — это то, куда пойдут
	// байты, а identity-ключ — с кем собирается сеанс.
	if !bytes.Equal(card.GetX25519Pubkey(), bundle.GetIdentityX25519Pubkey()) {
		t.Fatalf("восстановлен другой контакт: %x", card.GetX25519Pubkey())
	}
	if !bytes.Equal(card.GetEd25519Pubkey(), peer.node.PublicKey()) {
		t.Fatalf("ключ ноды контакта потерян: %x", card.GetEd25519Pubkey())
	}
	if want := yggnode.AddressForKey(peer.node.PublicKey()).String(); card.GetNodeIpv6() != want {
		t.Fatalf("node_ipv6 восстановлена не та: %s, ожидалась %s", card.GetNodeIpv6(), want)
	}
	// Бандл — непрозрачная для хранилища часть карты, и она обязана пережить
	// кодировку целиком: без signed prekey и сертификата X3DH не соберётся.
	restored := card.GetBundle()
	if !bytes.Equal(restored.GetSignedPrekey(), bundle.GetSignedPrekey()) {
		t.Fatal("signed prekey восстановился другим")
	}
	if !bytes.Equal(restored.GetNodeKeyProof(), bundle.GetNodeKeyProof()) {
		t.Fatal("сертификат ключа ноды восстановился другим")
	}
	if !bytes.Equal(restored.GetIdentityEd25519Pubkey(), bundle.GetIdentityEd25519Pubkey()) {
		t.Fatal("identity-ключ бандла восстановился другим")
	}

	st := contactStatsOf(t, e)
	if st.restored != 1 || st.restoreErr != 0 || st.skippedKey != 0 {
		t.Fatalf("восстановление контактов учтено неверно: %+v", st)
	}

	// Дальше — главное: отправка работает без повторного AddContact. Новый
	// диалог значит новый X3DH, то есть восстановленный бандл реально годен,
	// а кривой он был бы отвергнут ещё на сборке сеанса.
	sendQueued(t, e, bundle.GetIdentityX25519Pubkey(), &yggproto.ChatMessage{
		MessageId:      "msg-5b-второе",
		ConversationId: "второй диалог после рестарта",
		TimestampMs:    1700000060000,
		Content:        textContent("после рестарта"),
	})
	// Продолжение прежнего диалога идёт уже по восстановленному сеансу
	// (Шаг 3-c-3) и обязано остаться в той же очереди.
	sendQueued(t, e, bundle.GetIdentityX25519Pubkey(), &yggproto.ChatMessage{
		MessageId:      "msg-5b-третье",
		ConversationId: "диалог до перезапуска",
		TimestampMs:    1700000120000,
		Content:        textContent("продолжение старого диалога"),
	})

	// Адрес доставки в очередных записях взят из восстановленной карты: если бы
	// рестарт вернул пустую или иную пару ключей, сюда попал бы другой узел.
	pending := queuedSnapshot(t, e)
	if len(pending) != 3 {
		t.Fatalf("в очереди %d записей, ожидалось 3", len(pending))
	}
	for _, rec := range pending {
		if !bytes.Equal(rec.NodeKey, peer.node.PublicKey()) {
			t.Fatalf("сообщение %s идёт на другой узел: %x", rec.MessageID, rec.NodeKey)
		}
	}
}

// TestContactsAreNotCarriedIntoAnotherVault — карта в памяти обязана жить
// ровно столько, сколько хранилище, из которого она взята. Рестарт на другом
// томе (откат, другой data_dir, restoration из бэкапа) не имеет права
// дотягивать контакты прошлого тома: они проверялись против другой базы.
func TestContactsAreNotCarriedIntoAnotherVault(t *testing.T) {
	e := New()
	startIsolated(t, e, t.TempDir(), seedAlice)
	t.Cleanup(func() { stopIsolated(t, e) })

	peer := startEngine(t, seedBob, t.TempDir())
	if msg := addContactErr(t, e, cardOf(bundleOf(t, peer))); msg != "" {
		t.Fatalf("AddContact: %s", msg)
	}
	if got := contactsOf(t, e); len(got) != 1 {
		t.Fatalf("контактов %d, ожидался 1", len(got))
	}

	stopIsolated(t, e)
	startIsolated(t, e, t.TempDir(), seedAlice)

	if got := contactsOf(t, e); len(got) != 0 {
		t.Fatalf("чистый том унаследовал контакты прошлого: %+v", got)
	}
	if st := contactStatsOf(t, e); st.restored != 0 {
		t.Fatalf("с пустой базы восстановлено %d контактов", st.restored)
	}
}

// TestRestoredContactsMustPassVerification — сохранённая карта проходит те же
// ворота, что и живая AddContact. Диск правят руками, а доверие ядра к адресу
// доставки держится ровно на этой карте, поэтому связку «ключ строки -> чужая
// карта» и битые снимки поднимать нельзя.
func TestRestoredContactsMustPassVerification(t *testing.T) {
	dir := t.TempDir()
	e := New()
	startIsolated(t, e, dir, seedAlice)
	t.Cleanup(func() { stopIsolated(t, e) })

	bobNode := freshNodeKey(t)
	malloryNode := freshNodeKey(t)
	bobBundle := foreignBundle(t, seedBob, bobNode)
	carolBundle := foreignBundle(t, seedCarol, malloryNode)

	good := cardOf(bobBundle)
	if msg := addContactErr(t, e, good); msg != "" {
		t.Fatalf("честная карта отклонена: %s", msg)
	}

	// Подделки кладём в базу напрямую, минуя AddContact: именно так их увидит
	// отредактированный или повреждённый том. У каждой — свой ключ строки,
	// иначе они перетрут честную запись Боба.
	dave := cardOf(foreignBundle(t, seedDave, freshNodeKey(t)))
	dave.Bundle.NodeKeyProof[0] ^= 0xff
	badBytes, err := proto.Marshal(dave)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.db.SaveContact(hexKey(dave.GetX25519Pubkey()), badBytes); err != nil {
		t.Fatal(err)
	}

	// Годная карта Боба под ключом Каролы: verifyContact её пропускает, но
	// адрес Боба тогда ушёл бы в диалог Каролы.
	carolKey := hexKey(carolBundle.GetIdentityX25519Pubkey())
	goodBytes, err := proto.Marshal(cardOf(bobBundle))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.db.SaveContact(carolKey, goodBytes); err != nil {
		t.Fatal(err)
	}

	if err := e.db.SaveContact("deadbeef", []byte("это не protobuf")); err != nil {
		t.Fatal(err)
	}

	stopIsolated(t, e)
	startIsolated(t, e, dir, seedAlice)

	cards := contactsOf(t, e)
	if len(cards) != 1 {
		t.Fatalf("поднялось %d контактов, ожидался 1 честный: %+v", len(cards), cards)
	}
	if !bytes.Equal(cards[0].GetEd25519Pubkey(), bobNode) {
		t.Fatalf("осталась не та карта: %x", cards[0].GetEd25519Pubkey())
	}

	st := contactStatsOf(t, e)
	// restoreErr — испорченный сертификат и мусор, который не разобрался;
	// skippedKey — честная карта, лежащая под чужим ключом.
	if st.restored != 1 || st.restoreErr != 2 || st.skippedKey != 1 {
		t.Fatalf("учёт восстановления неверный: %+v", st)
	}
	// Разъезда с узлом-подделкой быть не должно: в памяти только честный Боб.
	if got := e.contacts[hexKey(bobBundle.GetIdentityX25519Pubkey())]; got == nil {
		t.Fatal("честный контакт не восстановился")
	}
	if got := e.contacts[carolKey]; got != nil {
		t.Fatalf("под связкой «ключ Каролы -> карта Боба» появился контакт: %+v", got)
	}
}

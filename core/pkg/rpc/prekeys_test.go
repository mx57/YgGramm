// File: core/pkg/rpc/prekeys_test.go
package rpc

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"ygggram/core/pkg/crypto"
	"ygggram/core/pkg/storage"
	"ygggram/core/pkg/yggnode"
	yggproto "ygggram/core/proto"
)

// Prekey-материал — то, чем держатся чужие X3DH-сессии. Ниже проверено, что он
// переживает рестарт, что по прежнему бандлу диалог открывается снова, что битый
// снимок не надевается вместо годного и что набор принадлежит одной личности.

// spkOf — публичная часть текущего signed prekey: её и только её видят
// собеседники в бандле.
func spkOf(t *testing.T, e *Engine) []byte {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.spkPriv == nil {
		t.Fatal("ядро без signed prekey")
	}
	return e.spkPriv.PublicKey().Bytes()
}

func prekeyStatsOf(t *testing.T, e *Engine) prekeyStats {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.prekeyStats
}

// TestPrekeysSurviveRestartAndStaleBundleStillOpens — главный смысл Шага 6-e:
// рестарт ядра не имеет права менять signed prekey. Контакт держит бандл,
// выданный однажды, и если выпустить ключи заново, первое же сообщение этого
// контакта (в том числе почта, оставленная слепым релеем) развалится на X3DH —
// и у отправителя при этом ничего не заболит.
func TestPrekeysSurviveRestartAndStaleBundleStillOpens(t *testing.T) {
	dir := t.TempDir()
	bob := startEngine(t, seedBob, dir)
	alice := startEngine(t, seedAlice, t.TempDir())
	bundle := registerContact(t, alice, bob)

	first := &yggproto.ChatMessage{
		MessageId:      "prekey-first",
		ConversationId: "диалог до рестарта",
		TimestampMs:    1700000000000,
		Content:        textContent("первое по прежнему бандлу"),
	}
	bob.handleInbound(yggnode.Packet{
		From:    alice.node.PublicKey(),
		Payload: marshalWire(t, wireOf(t, alice, bob, bundle, first)),
	})
	if st := statsOf(t, bob); st.applied != 1 {
		t.Fatalf("первое сообщение не открылось: %+v", st)
	}
	if !bytes.Equal(spkOf(t, bob), bundle.GetSignedPrekey()) {
		t.Fatal("бандл разойшёлся с живым signed prekey ещё до рестарта")
	}
	stopIsolated(t, bob)

	bob2 := startEngine(t, seedBob, dir)
	if st := prekeyStatsOf(t, bob2); st.restored != 1 || st.skippedBroken != 0 {
		t.Fatalf("материал не поднят из хранилища: %+v", st)
	}
	if !bytes.Equal(spkOf(t, bob2), bundle.GetSignedPrekey()) {
		t.Fatal("после рестарта signed prekey другой — старые бандлы обречены")
	}

	// Третий узел узнаёт Боба по бандлу, выданному ДО рестарта, и пишет ему.
	// Открытие этого конверта — единственная проверка, что вернулись и SPK, и
	// потраченный one-time prekey.
	carol := startEngine(t, seedCarol, t.TempDir())
	resp := carol.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_AddContact{
		AddContact: &yggproto.AddContactReq{Contact: &yggproto.ContactCard{
			NodeIpv6:      bundleNodeIPv6(bob2),
			Ed25519Pubkey: bob2.node.PublicKey(),
			X25519Pubkey:  bundle.GetIdentityX25519Pubkey(),
			Bundle:        bundle,
		}},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("AddContact по прежнему бандлу: %s", resp.GetErrorMessage())
	}
	second := &yggproto.ChatMessage{
		MessageId:      "prekey-second",
		ConversationId: "диалог после рестарта",
		TimestampMs:    1700000001000,
		Content:        textContent("после рестарта открылось"),
	}
	bob2.handleInbound(yggnode.Packet{
		From:    carol.node.PublicKey(),
		Payload: marshalWire(t, wireOf(t, carol, bob2, bundle, second)),
	})
	if st := statsOf(t, bob2); st.applied != 1 {
		t.Fatalf("конверт по прежнему бандлу не открылся: %+v", st)
	}
	conv := inboundConversationID(carol.master.MasterX25519.PublicKey().Bytes())
	if got := storedBodies(t, bob2, conv); len(got) != 1 || got[0] != "после рестарта открылось" {
		t.Fatalf("диалог после рестарта: %v", got)
	}
}

// TestBrokenPrekeySnapshotIsRegeneratedNotAdopted — набор, который не разбирается
// или неполон, не имеет права лечь в память: ядро выпускает новый и честно
// считает, что прежние бандлы контактов теперь не откроются.
func TestBrokenPrekeySnapshotIsRegeneratedNotAdopted(t *testing.T) {
	dir := t.TempDir()
	vaultKey := bytes.Repeat([]byte("p"), 32)

	run := func() *Engine {
		return startEngineWith(t, seedBob, dir, func(cfg *yggproto.NodeConfig) {
			cfg.MasterKey_256 = vaultKey
		})
	}
	bob := run()
	oldSPK := spkOf(t, bob)
	stopIsolated(t, bob)

	// Битый снимок: SPK разбирается, а опубликованный one-time prekey в складе
	// отсутствует. Для хранилища набор цел, для X3DH — нет.
	master, err := crypto.MasterIdentityFromMnemonic(seedBob, "")
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
	db, err := storage.OpenEncryptedDB(dir+"/data", storageKey(master, vaultKey), "write-by-hand")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SavePrekeys(&storage.StoredPrekeys{
		Owner:   master.MasterX25519.PublicKey().Bytes(),
		SpkPriv: spk.Bytes(),
		OpkID:   77,
		Opk:     []storage.StoredPrekey{{ID: 4, Priv: opk.Bytes()}},
	}); err != nil {
		t.Fatal(err)
	}
	// Рука, писавшая этот снимок, обязана разжать пальцы до старта ядра: у
	// дневника один владелец на каталог, и второй экземпляр не открылся бы.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	after := run()
	st := prekeyStatsOf(t, after)
	if st.skippedBroken != 1 || st.restored != 0 || st.regenerated != 1 {
		t.Fatalf("битый снимок прожит нечестно: %+v", st)
	}
	newSPK := spkOf(t, after)
	if bytes.Equal(newSPK, oldSPK) || bytes.Equal(newSPK, spk.PublicKey().Bytes()) {
		t.Fatal("после отказа от битого снимка ключ не сменился")
	}

	// Выпущенный заново материал обязан тут же лечь на диск: иначе следующий
	// рестарт потеряет его снова. Прежнее ядро при этом должно быть остановлено:
	// живой экземпляр держит каталог, а второй на тот же дневник не открывается.
	stopIsolated(t, after)
	restarted := run()
	if got := prekeyStatsOf(t, restarted); got.restored != 1 || got.skippedBroken != 0 {
		t.Fatalf("новый набор не пережил повторный рестарт: %+v", got)
	}
	if !bytes.Equal(spkOf(t, restarted), newSPK) {
		t.Fatal("после повторного рестарта signed prekey другой")
	}
}

// TestPrekeysAreNotCarriedIntoAnotherIdentity — тот же файл хранилища при явном
// master_key_256 служит нескольким личностям. Prekey-материал — не строка
// диалога, а сам ключ: надеть чужой значит расшифровывать чужие сообщения, а
// перезаписать чужой — потерять их навсегда.
func TestPrekeysAreNotCarriedIntoAnotherIdentity(t *testing.T) {
	dir := t.TempDir()
	vaultKey := bytes.Repeat([]byte("s"), 32)
	withKey := func(cfg *yggproto.NodeConfig) { cfg.MasterKey_256 = vaultKey }

	alice := startEngineWith(t, seedAlice, dir, withKey)
	if st := prekeyStatsOf(t, alice); st.regenerated != 1 || st.restored != 0 {
		t.Fatalf("первый запуск не выпустил материал: %+v", st)
	}
	aliceSPK := spkOf(t, alice)
	stopIsolated(t, alice)

	carol := startEngineWith(t, seedCarol, dir, withKey)
	if st := prekeyStatsOf(t, carol); st.restored != 0 || st.regenerated != 1 {
		t.Fatalf("чужой материал поднят другой личностью: %+v", st)
	}
	if bytes.Equal(spkOf(t, carol), aliceSPK) {
		t.Fatal("вторая идентичность получила signed prekey первой")
	}
	stopIsolated(t, carol)

	// Алиса возвращается в то же хранилище: её набор обязан лежать нетронутым,
	// а не быть перезаписанным соседкой.
	back := startEngineWith(t, seedAlice, dir, withKey)
	if st := prekeyStatsOf(t, back); st.restored != 1 {
		t.Fatalf("свой материал после соседки не поднят: %+v", st)
	}
	if !bytes.Equal(spkOf(t, back), aliceSPK) {
		t.Fatal("чужая идентичность перезаписала чужой signed prekey")
	}
}

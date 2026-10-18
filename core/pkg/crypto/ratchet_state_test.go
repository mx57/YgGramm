// File: core/pkg/crypto/ratchet_state_test.go
package crypto

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"testing"
	"time"
)

// newRatchetPair собирает два конца сеанса напрямую, без X3DH: здесь важна
// связность именно рэтчета, а handshake покрыт тестами pkg/rpc.
func newRatchetPair(t *testing.T) (initiator, receiver *DoubleRatchetSession) {
	t.Helper()

	shared := make([]byte, 32)
	if _, err := rand.Read(shared); err != nil {
		t.Fatal(err)
	}
	receiverDH, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	initiator, err = NewInitiatorSession(shared, receiverDH.PublicKey())
	if err != nil {
		t.Fatalf("сессия инициатора: %v", err)
	}
	return initiator, NewReceiverSession(shared, receiverDH)
}

// reopen — то, что делает ядро после рестарта: снимок состояния обратно в сеанс.
func reopen(t *testing.T, s *DoubleRatchetSession) *DoubleRatchetSession {
	t.Helper()
	restored, err := SessionFromState(s.ExportState())
	if err != nil {
		t.Fatalf("восстановление сеанса: %v", err)
	}
	return restored
}

func send(t *testing.T, from, to *DoubleRatchetSession, body, conv string) {
	t.Helper()

	enc, err := from.Encrypt([]byte(body), []byte(conv))
	if err != nil {
		t.Fatalf("шифрование %q: %v", body, err)
	}
	plain, err := to.Decrypt(*enc, []byte(conv))
	if err != nil {
		t.Fatalf("расшифровка %q: %v", body, err)
	}
	if string(plain) != body {
		t.Fatalf("пришло %q, ожидалось %q", plain, body)
	}
}

// assertStateIntact проверяет, что снимок прожил сериализацию без потерь:
// счётчики, обе цепочки и DH-ключи те же, что были до перезапуска.
func assertStateIntact(t *testing.T, name string, want, got SessionState) {
	t.Helper()

	if !bytes.Equal(want.DHSPrivateKey, got.DHSPrivateKey) {
		t.Fatalf("%s: восстановлен другой локальный DH-ключ", name)
	}
	if !bytes.Equal(want.DHRPublicKey, got.DHRPublicKey) {
		t.Fatalf("%s: восстановлен другой удалённый DH-ключ", name)
	}
	if !bytes.Equal(want.RootKey, got.RootKey) {
		t.Fatalf("%s: ключ корня разъехался", name)
	}
	if !bytes.Equal(want.SendChainKey, got.SendChainKey) {
		t.Fatalf("%s: отправная цепочка разъехалась", name)
	}
	if !bytes.Equal(want.RecvChainKey, got.RecvChainKey) {
		t.Fatalf("%s: приёмная цепочка разъехалась", name)
	}
	if want.SendSequence != got.SendSequence || want.RecvSequence != got.RecvSequence ||
		want.PrevChainLength != got.PrevChainLength {
		t.Fatalf("%s: счётчики сбились: было %d/%d/%d, стало %d/%d/%d",
			name, want.SendSequence, want.RecvSequence, want.PrevChainLength,
			got.SendSequence, got.RecvSequence, got.PrevChainLength)
	}
	if len(want.Skipped) != len(got.Skipped) {
		t.Fatalf("%s: пропущенных ключей было %d, стало %d", name, len(want.Skipped), len(got.Skipped))
	}
}

// TestRestoredSessionContinuesConversation — смысл персистентности: оба конца
// переговоров перезапускаются и диалог идёт дальше, будто его не прерывали.
func TestRestoredSessionContinuesConversation(t *testing.T) {
	alice, bob := newRatchetPair(t)
	const conv = "диалог через рестарт"

	send(t, alice, bob, "до перезапуска", conv)
	// Ответ получателя возможен только потому, что первый приём завёл его
	// отправную цепочку тем же DH-шагом.
	send(t, bob, alice, "ответ до перезапуска", conv)

	aliceState, bobState := alice.ExportState(), bob.ExportState()
	alice, bob = reopen(t, alice), reopen(t, bob)
	assertStateIntact(t, "инициатор", aliceState, alice.ExportState())
	assertStateIntact(t, "получатель", bobState, bob.ExportState())
	if len(bob.ExportState().SendChainKey) == 0 {
		t.Fatal("у восстановленного получателя нет отправной цепочки — ACK было бы шифровать нечем")
	}

	send(t, alice, bob, "после перезапуска", conv)
	send(t, bob, alice, "ответ после перезапуска", conv)
}

// TestRestoredSessionKeepsSkippedKeys — разрыв в приёмной цепочке переживает
// рестарт: иначе припозднившийся пакет стал бы невосстановимой потерей.
func TestRestoredSessionKeepsSkippedKeys(t *testing.T) {
	alice, bob := newRatchetPair(t)
	const conv = "порядок нарушен"

	var packets []EncryptedRatchetMessage
	for i := 0; i < 3; i++ {
		enc, err := alice.Encrypt([]byte(fmt.Sprintf("сообщение %d", i)), []byte(conv))
		if err != nil {
			t.Fatal(err)
		}
		packets = append(packets, *enc)
	}

	// Приходит последнее: два предыдущих ключа откладываются в кэш.
	plain, err := bob.Decrypt(packets[2], []byte(conv))
	if err != nil {
		t.Fatalf("расшифровка последнего пакета: %v", err)
	}
	if string(plain) != "сообщение 2" {
		t.Fatalf("пришло %q", plain)
	}
	bobState := bob.ExportState()
	if skipped := len(bobState.Skipped); skipped != 2 {
		t.Fatalf("пропущенных ключей %d, ожидалось 2", skipped)
	}

	bob = reopen(t, bob)
	assertStateIntact(t, "получатель с разрывом", bobState, bob.ExportState())

	for _, num := range []int{1, 0} {
		plain, err := bob.Decrypt(packets[num], []byte(conv))
		if err != nil {
			t.Fatalf("пакет %d после рестарта не раскрылся: %v", num, err)
		}
		if want := fmt.Sprintf("сообщение %d", num); string(plain) != want {
			t.Fatalf("пришло %q, ожидалось %q", plain, want)
		}
	}
	if left := len(bob.ExportState().Skipped); left != 0 {
		t.Fatalf("израсходованные ключи остались в кэше: %d", left)
	}

	// Цепочка при этом продолжает двигаться вперёд.
	send(t, alice, bob, "после разрыва", conv)
}

// TestSessionFromStateRejectsBrokenState — состояние читается из файла, который
// мог быть испорчен или подправлен. Принять его молча значит завести сеанс,
// который никогда ничего не раскроет.
func TestSessionFromStateRejectsBrokenState(t *testing.T) {
	dhs, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	good := SessionState{
		DHSPrivateKey: dhs.Bytes(),
		DHRPublicKey:  dhs.PublicKey().Bytes(),
		RootKey:       key,
		SendChainKey:  append([]byte(nil), key...),
		RecvChainKey:  append([]byte(nil), key...),
		Skipped: []SkippedKeyState{{
			ID:          "abcd-0",
			MessageKey:  append([]byte(nil), key...),
			CreatedAtMS: time.Now().UnixMilli(),
		}},
	}

	cases := []struct {
		name string
		mut  func(*SessionState)
	}{
		{"короткий DH-приватный ключ", func(s *SessionState) { s.DHSPrivateKey = s.DHSPrivateKey[:31] }},
		{"ключ корня не 32 байта", func(s *SessionState) { s.RootKey = s.RootKey[:16] }},
		{"отправная цепочка не 32 байта", func(s *SessionState) { s.SendChainKey = append(s.SendChainKey, 0) }},
		{"приёмная цепочка не 32 байта", func(s *SessionState) { s.RecvChainKey = s.RecvChainKey[:31] }},
		{"удалённый DH-ключ битый", func(s *SessionState) { s.DHRPublicKey = s.DHRPublicKey[:31] }},
		{"пропущенный ключ без идентификатора", func(s *SessionState) { s.Skipped[0].ID = "" }},
		{"пропущенный ключ не 32 байта", func(s *SessionState) { s.Skipped[0].MessageKey = s.Skipped[0].MessageKey[:4] }},
		{"слишком много пропущенных ключей", func(s *SessionState) {
			s.Skipped = make([]SkippedKeyState, maxSkippedKeys+1)
			for i := range s.Skipped {
				s.Skipped[i] = SkippedKeyState{
					ID:          fmt.Sprintf("k-%d", i),
					MessageKey:  append([]byte(nil), key...),
					CreatedAtMS: time.Now().UnixMilli(),
				}
			}
		}},
	}
	for _, tc := range cases {
		broken := good
		broken.Skipped = append([]SkippedKeyState(nil), good.Skipped...)
		tc.mut(&broken)
		if _, err := SessionFromState(broken); err == nil && tc.name != "пропущенный ключ без метки времени" {
			t.Errorf("%s: битое состояние принято без ошибки", tc.name)
		}
	}

	// Метка времени обязательна для TTL-списания, но её отсутствие не делает
	// сеанс битым: ключ годный, просто считается живым с этого момента.
	untimed := good
	untimed.Skipped = append([]SkippedKeyState(nil), good.Skipped...)
	untimed.Skipped[0].CreatedAtMS = 0
	restored, err := SessionFromState(untimed)
	if err != nil {
		t.Fatalf("ключ без метки отвергнут: %v", err)
	}
	if len(restored.ExportState().Skipped) != 1 {
		t.Fatal("ключ без метки не попал в кэш")
	}

	// Ровно предел кэша — законное состояние: отбраковать его значило бы
	// выбросить рабочий диалог.
	edge := good
	edge.Skipped = make([]SkippedKeyState, maxSkippedKeys)
	for i := range edge.Skipped {
		edge.Skipped[i] = SkippedKeyState{
			ID:          fmt.Sprintf("k-%d", i),
			MessageKey:  append([]byte(nil), key...),
			CreatedAtMS: time.Now().UnixMilli(),
		}
	}
	if _, err := SessionFromState(edge); err != nil {
		t.Fatalf("состояние с пределом кэша отвергнуто: %v", err)
	}
}

// TestExportStateIsIndependentSnapshot — снимок не должен делить байты с живым
// сеансом: иначе записанное в базу состояние уехало бы вперёд само.
func TestExportStateIsIndependentSnapshot(t *testing.T) {
	alice, bob := newRatchetPair(t)

	snapshot := alice.ExportState()
	if snapshot.SendSequence != 0 {
		t.Fatalf("свежий сеанс уже сдвинул счётчик: %d", snapshot.SendSequence)
	}
	root := append([]byte(nil), snapshot.RootKey...)

	send(t, alice, bob, "сдвиг цепочки", "диалог")

	if !bytes.Equal(snapshot.RootKey, root) || snapshot.SendSequence != 0 {
		t.Fatal("снимок поехал вместе с живым сеансом")
	}

	restored, err := SessionFromState(snapshot)
	if err != nil {
		t.Fatalf("снимок не восстановился: %v", err)
	}
	if got := restored.ExportState(); got.SendSequence != 0 || !bytes.Equal(got.RootKey, root) {
		t.Fatal("восстановленный сеанс не соответствует снимку")
	}
}

package storage

import "testing"

func openDB(t *testing.T) *EncryptedDB {
	t.Helper()
	var key [32]byte
	copy(key[:], "device-master-key-for-tests-0000")
	db, err := OpenEncryptedDB(t.TempDir(), key, "test-device")
	if err != nil {
		t.Fatalf("OpenEncryptedDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestSaveMessageIsIdempotentByMessageID — переотправка того же message_id не
// должна удваивать строку диалога: UI читает список через GetMessages, и
// дубликат выглядел бы как второе сообщение.
func TestSaveMessageIsIdempotentByMessageID(t *testing.T) {
	db := openDB(t)

	first := &StoredMessage{MessageID: "m1", ConversationID: "c1", Body: "одно и то же", TimestampMS: 10, IsOutbound: true}
	if err := db.SaveMessage(first); err != nil {
		t.Fatal(err)
	}
	hlc := first.Clock

	second := &StoredMessage{MessageID: "m1", ConversationID: "c1", Body: "одно и то же", TimestampMS: 12, IsOutbound: true, IsRead: true}
	if err := db.SaveMessage(second); err != nil {
		t.Fatal(err)
	}

	msgs := db.GetMessages("c1", 50)
	if len(msgs) != 1 {
		t.Fatalf("в диалоге %d сообщений, ожидалось 1", len(msgs))
	}
	if msgs[0].TimestampMS != 12 {
		t.Fatalf("пересланное сообщение не обновило метку: %d", msgs[0].TimestampMS)
	}
	if msgs[0].Clock != hlc {
		t.Fatalf("пересланное сообщение сдвинулось в CRDT-порядке: %+v вместо %+v", msgs[0].Clock, hlc)
	}
	if !msgs[0].IsRead {
		t.Fatal("пересланное сообщение стало непрочитанным")
	}
	// Индекс остаётся однозначным: записывать то же сообщение в fts второй раз
	// нельзя, иначе поиск начал бы показывать дубликаты.
	if got := len(db.SearchMessages("одно", 10)); got != 1 {
		t.Fatalf("полнотекстовый индекс даёт %d совпадения, ожидалось 1", got)
	}
}

// TestMarkDeliveredOnlyOutbound — «доставлено» подтверждает только исходящее:
// у входящего сообщения доставка на нашей стороне, и ACK ей не пишется.
func TestMarkDeliveredOnlyOutbound(t *testing.T) {
	db := openDB(t)

	if err := db.SaveMessage(&StoredMessage{MessageID: "out", ConversationID: "c1", IsOutbound: true}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveMessage(&StoredMessage{MessageID: "in", ConversationID: "c1"}); err != nil {
		t.Fatal(err)
	}

	if db.IsDelivered("out") {
		t.Fatal("исходящее уже помечено доставленным")
	}
	conv, known, err := db.MarkDelivered("out", 500)
	if err != nil || !known || conv != "c1" {
		t.Fatalf("MarkDelivered: conv=%q known=%v err=%v", conv, known, err)
	}
	if !db.IsDelivered("out") {
		t.Fatal("отметка доставки не проставилась")
	}

	// Повторный ACK не двигает время: доставкой считается первое подтверждение.
	if _, known, err := db.MarkDelivered("out", 900); err != nil || !known {
		t.Fatalf("повторный ACK: known=%v err=%v", known, err)
	}
	if got := db.GetMessages("c1", 50); got[0].DeliveredMS != 500 {
		t.Fatalf("повторный ACK переписал время: %d", got[0].DeliveredMS)
	}

	if _, known, err := db.MarkDelivered("in", 700); err != nil || known {
		t.Fatalf("ACK входящего сообщения принят: known=%v err=%v", known, err)
	}
	if _, known, err := db.MarkDelivered("нет-такого", 700); err != nil || known {
		t.Fatalf("ACK чужого message_id принят: known=%v err=%v", known, err)
	}

	// Перезапуск: отметка живёт в файле базы, а не только в памяти.
	reopened := reopenDB(t, db)
	if !reopened.IsDelivered("out") {
		t.Fatal("отметка доставки не пережила перезагрузку базы")
	}
}

// TestOutboundQueueSurvivesRestart — неотправленный конверт обязан пережить
// перезапуск ядра и сохраниться дословно: пересобирать его заново значит двигать
// отправную цепочку ради пакета, который никуда не ушёл, а выбросить — молча
// потерять сообщение.
func TestOutboundQueueSurvivesRestart(t *testing.T) {
	db := openDB(t)

	nodeKey := []byte("0123456789abcdef0123456789abcdef")
	if err := db.QueueOutbound(&StoredOutbound{
		MessageID: "m-late", ConversationID: "c1", NodeKey: nodeKey,
		Envelope: []byte("байты конверта"), EnqueuedMS: 10,
	}); err != nil {
		t.Fatal(err)
	}
	// Более старое сообщение впереди: порядок очереди = порядку печати.
	if err := db.QueueOutbound(&StoredOutbound{
		MessageID: "m-older", ConversationID: "c1", NodeKey: nodeKey,
		Envelope: []byte("другой конверт"), EnqueuedMS: 5,
	}); err != nil {
		t.Fatal(err)
	}
	pending := db.PendingOutbound()
	if len(pending) != 2 || pending[0].MessageID != "m-older" {
		t.Fatalf("очередь не та: %+v", pending)
	}

	// Повторная отправка того же message_id обновляет запись, а не заводит
	// вторую, и сохраняет время постановки вместе со счётчиком отправок.
	if err := db.QueueOutbound(&StoredOutbound{
		MessageID: "m-late", ConversationID: "c1", NodeKey: nodeKey,
		Envelope: []byte("переложенный конверт"), EnqueuedMS: 99,
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.RecordOutboundSend("m-late", 50); err != nil {
		t.Fatal(err)
	}
	stored := db.outbound["m-late"]
	if stored == nil {
		t.Fatal("записи m-late в очереди нет")
	}
	if stored.EnqueuedMS != 10 {
		t.Fatalf("перепостановка сдвинула время в очереди: %d", stored.EnqueuedMS)
	}
	if stored.Sends != 1 || stored.LastSendMS != 50 {
		t.Fatalf("отправка учтена неверно: %+v", stored)
	}
	if string(stored.Envelope) != "переложенный конверт" {
		t.Fatalf("конверт не обновился: %q", stored.Envelope)
	}
	// Пустой конверт класть в очередь нечего.
	if err := db.QueueOutbound(&StoredOutbound{MessageID: "m-empty"}); err == nil {
		t.Fatal("пустой конверт принят в очередь")
	}

	// Неотправленное переживает перезапуск в том же виде.
	reopened := reopenDB(t, db)
	again := reopened.PendingOutbound()
	if len(again) != 2 || again[0].MessageID != "m-older" {
		t.Fatalf("после перезапуска очередь %+v", again)
	}
	if string(again[1].Envelope) != "переложенный конверт" || again[1].Sends != 1 {
		t.Fatalf("восстановлен не тот конверт: %+v", again[1])
	}

	if err := reopened.CompleteOutbound("m-older"); err != nil {
		t.Fatal(err)
	}
	if got := reopened.PendingOutbound(); len(got) != 1 || got[0].MessageID != "m-late" {
		t.Fatalf("CompleteOutbound не убрал запись: %+v", got)
	}
	// Убрать то, чего нет, — не ошибка: успешная отправка без прошлых неудач
	// встречается чаще всего.
	if err := reopened.CompleteOutbound("m-older"); err != nil {
		t.Fatalf("повторный CompleteOutbound: %v", err)
	}
	if err := reopened.RecordOutboundSend("m-нет-такого", 1); err != nil {
		t.Fatalf("RecordOutboundSend по несуществующей записи: %v", err)
	}
}

// TestStoredSessionsSurviveRestart — состояние сеанса — единственное, что
// держит открытый диалог живым после перезапуска ядра, поэтому оно обязано
// пережить его дословно.
func TestStoredSessionsSurviveRestart(t *testing.T) {
	db := openDB(t)

	if err := db.SaveSession("peer|conv", []byte(`{"ns":1}`)); err != nil {
		t.Fatal(err)
	}
	// Перезапись заменяет состояние, а не заводит вторую строку: сеанс у диалога
	// один, и старый снимок цепочки опаснее отсутствия любого.
	if err := db.SaveSession("peer|conv", []byte(`{"ns":2}`)); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveSession("other|conv", []byte(`{"ns":7}`)); err != nil {
		t.Fatal(err)
	}

	got := db.Sessions()
	if len(got) != 2 {
		t.Fatalf("состояний сеансов %d, ожидалось 2", len(got))
	}
	if string(got["peer|conv"]) != `{"ns":2}` {
		t.Fatalf("состояние не заменилось: %q", got["peer|conv"])
	}

	// Копия независима: правка выданного буфера не должна трогать хранилище.
	got["peer|conv"][0] = 'X'
	if string(db.Sessions()["peer|conv"]) != `{"ns":2}` {
		t.Fatal("Sessions отдаёт ссылку на состояние в памяти")
	}

	if err := db.SaveSession("", []byte("x")); err == nil {
		t.Fatal("принято состояние без имени сеанса")
	}
	if err := db.SaveSession("x", nil); err == nil {
		t.Fatal("принято пустое состояние")
	}

	reopened := reopenDB(t, db)
	again := reopened.Sessions()
	if len(again) != 2 {
		t.Fatalf("после перезапуска состояний %d, ожидалось 2", len(again))
	}
	if string(again["peer|conv"]) != `{"ns":2}` || string(again["other|conv"]) != `{"ns":7}` {
		t.Fatalf("состояния восстановлены не те: %q / %q", again["peer|conv"], again["other|conv"])
	}
}

// TestStoredContactsSurviveRestart — карта контакта — единственное, чем ядро
// знает адрес доставки знакомому, поэтому она обязана пережить перезапуск
// хранилища дословно, как и состояние сеанса.
func TestStoredContactsSurviveRestart(t *testing.T) {
	db := openDB(t)

	if err := db.SaveContact("aa", []byte(`{"v":1}`)); err != nil {
		t.Fatal(err)
	}
	// Перезапись по тому же ключу заменяет карту: две версии одного контакта —
	// это два разных узла, и выбор между ними хранилище подсказать не может.
	if err := db.SaveContact("aa", []byte(`{"v":2}`)); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveContact("bb", []byte(`{"v":7}`)); err != nil {
		t.Fatal(err)
	}

	got := db.Contacts()
	if len(got) != 2 {
		t.Fatalf("карт контакта %d, ожидалось 2", len(got))
	}
	if string(got["aa"]) != `{"v":2}` {
		t.Fatalf("карта не заменилась: %q", got["aa"])
	}

	// Копия независима: правка выданного буфера не должна трогать хранилище.
	got["aa"][0] = 'X'
	if string(db.Contacts()["aa"]) != `{"v":2}` {
		t.Fatal("Contacts отдаёт ссылку на карту в памяти")
	}

	if err := db.SaveContact("", []byte("x")); err == nil {
		t.Fatal("принята карта без ключа контакта")
	}
	if err := db.SaveContact("x", nil); err == nil {
		t.Fatal("принята пустая карта")
	}

	reopened := reopenDB(t, db)
	again := reopened.Contacts()
	if len(again) != 2 {
		t.Fatalf("после перезапуска карт %d, ожидалось 2", len(again))
	}
	if string(again["aa"]) != `{"v":2}` || string(again["bb"]) != `{"v":7}` {
		t.Fatalf("карты восстановлены не те: %q / %q", again["aa"], again["bb"])
	}
}

// TestStoredSFSurvivesRestart — чужая почта S&F-релея обязана пережить
// перезапуск сервиса: иначе рестарт означал бы молчаливое исчезновение писем,
// которые адресат ещё не забрал.
func TestStoredSFSurvivesRestart(t *testing.T) {
	db := openDB(t)

	if err := db.SFPut(&StoredSF{MailboxID: "aa", EnvelopeID: "e1",
		Data: []byte("первое письмо"), PlacedMS: 30, ExpiresMS: 900}); err != nil {
		t.Fatal(err)
	}
	if err := db.SFPut(&StoredSF{MailboxID: "aa", EnvelopeID: "e2",
		Data: []byte("второе письмо длиннее"), PlacedMS: 10, ExpiresMS: 900}); err != nil {
		t.Fatal(err)
	}
	if err := db.SFPut(&StoredSF{MailboxID: "bb", EnvelopeID: "e3",
		Data: []byte("чужой ящик"), PlacedMS: 20, ExpiresMS: 900}); err != nil {
		t.Fatal(err)
	}
	// Письмо без ящика, без идентификатора или без тела — не письмо.
	if err := db.SFPut(&StoredSF{EnvelopeID: "e", Data: []byte("x")}); err == nil {
		t.Fatal("принято письмо без ящика")
	}
	if err := db.SFPut(&StoredSF{MailboxID: "aa", Data: []byte("x")}); err == nil {
		t.Fatal("принято письмо без идентификатора")
	}
	if err := db.SFPut(&StoredSF{MailboxID: "aa", EnvelopeID: "e"}); err == nil {
		t.Fatal("принято письмо без тела")
	}

	cnt, bytes := db.SFMailbox("aa")
	if cnt != 2 {
		t.Fatalf("в ящике %d писем, ожидалось 2", cnt)
	}
	if want := int64(len("первое письмо") + len("второе письмо длиннее")); bytes != want {
		t.Fatalf("квота по байтам мнимая: %d, ожидалось %d", bytes, want)
	}
	if total := db.SFStoredTotal(); total != 3 {
		t.Fatalf("релей держит %d писем, ожидалось 3", total)
	}

	reopened := reopenDB(t, db)
	if total := reopened.SFStoredTotal(); total != 3 {
		t.Fatalf("после перезапуска писем %d, ожидалось 3", total)
	}
	if cnt, _ := reopened.SFMailbox("bb"); cnt != 1 {
		t.Fatalf("ящик bb восстановлен пустым: %d", cnt)
	}

	// Выдача ящика: соседний не задеван, а порядок — по времени кладки, чтобы
	// адресат читал почту в том порядке, в котором она приходила. Выдача письмо
	// не удаляет: релей не знает, дошёл ли ответ, и обязан уметь переслать его
	// ещё раз (окно задаёт pkg/rpc).
	taken, err := reopened.SFHandOut("aa", 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(taken) != 2 || taken[0].EnvelopeID != "e2" || taken[1].EnvelopeID != "e1" {
		t.Fatalf("выдача не та: %+v", taken)
	}
	if string(taken[0].Data) != "второе письмо длиннее" || taken[0].ExpiresMS != 900 {
		t.Fatalf("письмо восстановлено не то: %+v", taken[0])
	}
	if taken[0].HandedMS != 500 || taken[1].HandedMS != 500 {
		t.Fatalf("выдача не помечена: %+v", taken)
	}
	if cnt, _ := reopened.SFMailbox("aa"); cnt != 2 {
		t.Fatalf("выдача удалила почту из ящика: %d писем", cnt)
	}
	if cnt, _ := reopened.SFMailbox("bb"); cnt != 1 {
		t.Fatalf("выдача aa задела чужой ящик: %d писем", cnt)
	}
	// Метка выдачи переживает перезапуск: без неё окно пересылки отсчитывалось
	// бы заново, и письмо жило бы на чужом узле дольше обещанного.
	again := reopenDB(t, reopened)
	if stored := again.sf[sfKey("aa", "e1")]; stored == nil || stored.HandedMS != 500 {
		t.Fatalf("метка выдачи не пережила перезапуск: %+v", stored)
	}
	// Пересылка в окне отдаёт те же письма снова — иначе адресат, до которого
	// пакет ответа не дошёл, остаётся без почты навсегда.
	resend, err := again.SFHandOut("aa", 600)
	if err != nil || len(resend) != 2 {
		t.Fatalf("повторная выдача: %v, %+v", err, resend)
	}
	// Граница окна: до неё — ничего, по ней — списано.
	if dropped := again.SFDropHanded(599); dropped != 0 {
		t.Fatalf("выданное списано раньше окна: %d", dropped)
	}
	if dropped := again.SFDropHanded(600); dropped != 2 {
		t.Fatalf("выданное не списано по оконной границе: %d", dropped)
	}
	if cnt, _ := again.SFMailbox("aa"); cnt != 0 {
		t.Fatalf("после окна в ящике осталось %d писем", cnt)
	}
	if cnt, _ := again.SFMailbox("bb"); cnt != 1 {
		t.Fatalf("списание aa задело чужой ящик: %d писем", cnt)
	}
	// Забрать пустой ящик — не ошибка: релей обязан ответить «почты нет».
	if got, err := again.SFHandOut("aa", 700); err != nil || len(got) != 0 {
		t.Fatalf("выдача пустого ящика: %v, %+v", err, got)
	}
	// Невыданное письмо окном не списывается: у него есть только TTL.
	if dropped := again.SFDropHanded(1 << 40); dropped != 0 {
		t.Fatalf("списано невыданное письмо: %d", dropped)
	}
	// Списанное не возвращается с диска.
	if total := reopenDB(t, again).SFStoredTotal(); total != 1 {
		t.Fatalf("списанное вернулось с диска: %d", total)
	}
}

// TestSFPutKeepsFirstPlacement — повторная кладка тех же байт не продлевает
// и не переставляет письмо: иначе отправитель, крутящий очередь каждые две
// секунды, держал бы чужое письмо на релее вечно.
func TestSFPutKeepsFirstPlacement(t *testing.T) {
	db := openDB(t)

	if err := db.SFPut(&StoredSF{MailboxID: "aa", EnvelopeID: "e1",
		Data: []byte("письмо"), PlacedMS: 100, ExpiresMS: 200}); err != nil {
		t.Fatal(err)
	}
	if err := db.SFPut(&StoredSF{MailboxID: "aa", EnvelopeID: "e1",
		Data: []byte("письмо"), PlacedMS: 150, ExpiresMS: 400}); err != nil {
		t.Fatal(err)
	}
	if total := db.SFStoredTotal(); total != 1 {
		t.Fatalf("повторная кладка удвоила почту: %d", total)
	}
	stored := db.sf[sfKey("aa", "e1")]
	if stored == nil {
		t.Fatal("письмо пропало из хранилища")
	}
	if stored.PlacedMS != 100 || stored.ExpiresMS != 200 {
		t.Fatalf("перекладка сдвинула срок хранения: %+v", stored)
	}
}

// TestSFDropExpired — просроченное письмо выбрасывается по публичному TTL, а
// бессрочное (срок не проставлен) по TTL не трогается.
func TestSFDropExpired(t *testing.T) {
	db := openDB(t)

	if err := db.SFPut(&StoredSF{MailboxID: "aa", EnvelopeID: "e1",
		Data: []byte("срочное"), PlacedMS: 10, ExpiresMS: 100}); err != nil {
		t.Fatal(err)
	}
	if err := db.SFPut(&StoredSF{MailboxID: "aa", EnvelopeID: "e2",
		Data: []byte("без срока"), PlacedMS: 10}); err != nil {
		t.Fatal(err)
	}
	if dropped := db.SFDropExpired(99); dropped != 0 {
		t.Fatalf("сброшено %d писем до срока", dropped)
	}
	// На границе срок истёк: «до» включает сам момент.
	if dropped := db.SFDropExpired(100); dropped != 1 {
		t.Fatalf("сброшено %d писем, ожидалось 1", dropped)
	}
	if total := db.SFStoredTotal(); total != 1 {
		t.Fatalf("после сброса осталось %d писем, ожидалось 1", total)
	}
	if reopened := reopenDB(t, db); reopened.SFStoredTotal() != 1 {
		t.Fatal("сброс не лёг на диск")
	}
}

// TestRecordRelaySendSurvivesRestart — счётчик кладок слепым релеям обязан
// пережить перезапуск: без него сканер очереди после рестарта заново выкладывает
// те же байты, а это пустая трата квоты чужого узла.
func TestRecordRelaySendSurvivesRestart(t *testing.T) {
	db := openDB(t)

	nodeKey := []byte("0123456789abcdef0123456789abcdef")
	if err := db.QueueOutbound(&StoredOutbound{MessageID: "m1", ConversationID: "c1",
		NodeKey: nodeKey, Envelope: []byte("кадр"), EnqueuedMS: 10}); err != nil {
		t.Fatal(err)
	}
	if err := db.RecordRelaySend("m1", 55); err != nil {
		t.Fatal(err)
	}
	got := db.PendingOutbound()
	if len(got) != 1 || got[0].RelayedMS != 55 || got[0].RelaySends != 1 {
		t.Fatalf("отметка о relay посчитана неверно: %+v", got)
	}
	// Перепостановка (SendMessage тем же message_id) обнуляет кладки: на
	// хранителя обязан лечь новый конверт, а не тот, что лежал до рестарта.
	if err := db.QueueOutbound(&StoredOutbound{MessageID: "m1", ConversationID: "c1",
		NodeKey: nodeKey, Envelope: []byte("новый кадр"), EnqueuedMS: 20}); err != nil {
		t.Fatal(err)
	}
	if got := db.PendingOutbound(); got[0].RelayedMS != 0 || got[0].RelaySends != 0 {
		t.Fatalf("перепостановка сохранила старые кладки: %+v", got[0])
	}
	if err := db.RecordRelaySend("m1", 77); err != nil {
		t.Fatal(err)
	}
	if err := db.RecordRelaySend("m1", 99); err != nil {
		t.Fatal(err)
	}

	reopened := reopenDB(t, db)
	if got := reopened.PendingOutbound(); len(got) != 1 || got[0].RelayedMS != 99 || got[0].RelaySends != 2 {
		t.Fatalf("кладка не пережила перезапуск: %+v", got)
	}
	// Отметить то, чего в очереди нет, — не ошибка: гонка с CompleteOutbound.
	if err := reopened.RecordRelaySend("m-нет-такого", 1); err != nil {
		t.Fatalf("RecordRelaySend по несуществующей записи: %v", err)
	}
}

// TestStoredPrekeysSurviveRestart — prekey-материал ядра обязан пережить
// рестарт дословно: контакт держит бандл, выданный однажды, и если SPK/OPK
// станут другими, X3DH этого контакта соберёт не тот корень.
func TestStoredPrekeysSurviveRestart(t *testing.T) {
	db := openDB(t)
	owner := ownerOf("владелец")
	foreign := ownerOf("чужая личность")

	if got := db.Prekeys(owner); got != nil {
		t.Fatalf("на пустом хранилище уже есть материал: %+v", got)
	}
	// Набор без signed prekey или без владельца не сохраняется: и то и другое
	// означало бы ключ, который некому вернуть после перезапуска.
	if err := db.SavePrekeys(&StoredPrekeys{Owner: owner, OpkID: 3}); err == nil {
		t.Fatal("принят материал без SPK")
	}
	if err := db.SavePrekeys(&StoredPrekeys{SpkPriv: []byte("spk-private-key-32-bytes-0000000")}); err == nil {
		t.Fatal("принят материал без владельца")
	}

	pk := &StoredPrekeys{
		Owner:       owner,
		SpkPriv:     []byte("spk-private-key-32-bytes-0000000"),
		OpkID:       9,
		Opk:         []StoredPrekey{{ID: 7, Priv: []byte("opk-7")}, {ID: 8, Priv: []byte("opk-8")}},
		OpkConsumed: []uint32{4, 5},
	}
	if err := db.SavePrekeys(pk); err != nil {
		t.Fatal(err)
	}

	// Копия независима в обе стороны: правка выданного снимка не трогает
	// хранилище, а правка входного — не трогает сохранённое.
	got := db.Prekeys(owner)
	got.Opk[0].Priv[0] = 'X'
	got.OpkConsumed[0] = 99
	pk.Opk[1].Priv[0] = 'Y'
	if again := db.Prekeys(owner); again.Opk[0].Priv[0] != 'o' || again.OpkConsumed[0] != 4 ||
		again.Opk[1].Priv[0] != 'o' {
		t.Fatalf("Prekeys делит массивы с памятью: %+v", again)
	}

	reopened := reopenDB(t, db)
	restored := reopened.Prekeys(owner)
	if restored == nil {
		t.Fatal("материал не пережил перезапуск")
	}
	if string(restored.SpkPriv) != string(pk.SpkPriv) || restored.OpkID != 9 {
		t.Fatalf("восстановлен другой набор: %+v", restored)
	}
	if len(restored.Opk) != 2 || restored.Opk[0].ID != 7 || string(restored.Opk[0].Priv) != "opk-7" {
		t.Fatalf("склад one-time prekeys восстановлен не тот: %+v", restored.Opk)
	}
	if len(restored.OpkConsumed) != 2 || restored.OpkConsumed[1] != 5 {
		t.Fatalf("сожжённые prekey восстановлены не те: %+v", restored.OpkConsumed)
	}

	// Чужой владелец не получает чужой набор: в одном хранилище под
	// master_key_256 могут лежать ключи нескольких идентичностей.
	if got := reopened.Prekeys(foreign); got != nil {
		t.Fatalf("чужой prekey-материал отдан другой личности: %+v", got)
	}

	// Замена полная: выброшенный one-time prekey обязан исчезнуть и на диске,
	// но только у своего владельца.
	if err := reopened.SavePrekeys(&StoredPrekeys{
		Owner:   owner,
		SpkPriv: []byte("spk-private-key-32-bytes-0000000"),
		Opk:     []StoredPrekey{{ID: 7, Priv: []byte("opk-7")}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reopened.SavePrekeys(&StoredPrekeys{
		Owner:   foreign,
		SpkPriv: []byte("chuzhoy-spk-32-bytes-00000000000"),
		Opk:     []StoredPrekey{{ID: 3, Priv: []byte("opk-3")}},
	}); err != nil {
		t.Fatal(err)
	}
	third := reopenDB(t, reopened)
	if got := third.Prekeys(owner); len(got.Opk) != 1 || len(got.OpkConsumed) != 0 {
		t.Fatalf("полная замена не сработала: %+v", got)
	}
	if got := third.Prekeys(foreign); string(got.SpkPriv) != "chuzhoy-spk-32-bytes-00000000000" {
		t.Fatalf("соседний набор пострадал: %+v", got)
	}
}

// ownerOf — 32-байтовый «мастер-X25519» тестовой идентичности.
func ownerOf(name string) []byte {
	out := make([]byte, 32)
	copy(out, name)
	return out
}

func openDBAt(t *testing.T, dir string) *EncryptedDB {
	t.Helper()
	var key [32]byte
	copy(key[:], "device-master-key-for-tests-0000")
	db, err := OpenEncryptedDB(dir, key, "test-device")
	if err != nil {
		t.Fatalf("повторное открытие: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// reopenDB — «перезапуск приложения»: предыдущий экземпляр закрывается, потом
// каталог открывается заново. У дневника один владелец на каталог в процессе, и
// тест обязан это соблюдать: два открытых экземпляра на одном каталоге — ровно тот случай,
// который запрет на второй экземпляр и должен ловить (см.
// TestOpenSameDirRefusedWhileHeld).
func reopenDB(t *testing.T, prev *EncryptedDB) *EncryptedDB {
	t.Helper()
	dir := prev.dataDir
	if err := prev.Close(); err != nil {
		t.Fatalf("закрытие хранилища: %v", err)
	}
	return openDBAt(t, dir)
}

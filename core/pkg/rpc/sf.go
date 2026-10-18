// File: core/pkg/rpc/sf.go
package rpc

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"ygggram/core/pkg/crypto"
	"ygggram/core/pkg/storage"
	"ygggram/core/pkg/yggnode"
	yggproto "ygggram/core/proto"
)

// Слепой store-and-forward: узел-хранитель держит чужие конверты, пока их не
// заберёт адресат, и при этом не узнаёт ни отправителя, ни содержимого. Индекс
// почты — HMAC от identity-ключа получателя и дня-эпохи, а само письмо заперто
// Sealed Sender'ом.
//
// Что это честно даёт: сообщение уходит, когда адресат офлайн и когда
// отправитель больше в сети не появится. Чего оно не даёт: релей не подтверждает
// хранение (письмо потеряно, если хранитель его стёр, а ACK адресату не дошёл),
// а момент выхода адресата из офлайна хранитель всё-таки видит — забирать почту,
// не назвав свою идентичность, невозможно.
const (
	// sfTTL — опубликованный срок хранения. Без него релей превращается в
	// свалку: место кончается, а чужая почта не вытесняется.
	sfTTL = 14 * 24 * time.Hour
	// Квоты на ящик: ящик открывается любому, кто знает открытый ключ
	// получателя, поэтому без лимита один адресат мог бы забить диск чужого
	// узла чужой почтой.
	sfMaxPerMailbox      = 512
	sfMaxBytesPerMailbox = 8 << 20
	// sfRedelivery — окно повторной выдачи: сколько времени релей готов ещё раз
	// отдать уже выданное письмо. Пакет ответа идёт по тому же оверлею, что и
	// живой конверт, и первый пакет на ещё не прогретом обратном маршруте
	// теряется; списывать почту по первой выдаче означало бы терять её вместе с
	// тем пакетом. Дубликаты гасит клиент (см. acceptSFReply).
	sfRedelivery = 10 * time.Minute
	// sfEpochWindow — за сколько суток-эпох опрашивать ящики. Конверт лежит в
	// ящике дня, в который его положили; окно в две эпохи покрывает отправку до
	// полуночи и получение после неё, не превращая опрос в перебор.
	sfEpochWindow = 2
	// sfPollInterval — как часто ядро само заглядывает на релеи. Команда
	// SfCollect остаётся (UI зовёт её при выходе из офлайна), но полагаться
	// только на неё нельзя: приложение могло пережить рестарт, пока адресат
	// был офлайн.
	sfPollInterval = 30 * time.Second
)

// sfStats — учёт обеих ролей: хранить чужое и забирать своё. «Передано
// оверлею» и «хранится у релея» — разные новости, и вторая протоколом не
// подтверждается, поэтому в счётчиках записана ровно первая.
type sfStats struct {
	pushed     uint64 // sf_put-кадров передано оверлею в адрес релея
	pushFailed uint64 // ни до одного релея маршрута нет
	// stored — как хранитель: новых писем легло на хранение.
	stored uint64
	// refused — как хранитель: отказ (режим, форма, PoW, квота, подпись, связка,
	// отсутствие обратного маршрута).
	refused uint64
	// handedOut — как хранитель: сколько раз конверты ушли адресату. Повторная
	// выдача в окне пересылки считается: это настоящий трафик выдачи, а
	// «получил ли адресат» хранителю неизвестно.
	handedOut uint64
	// expired — как хранитель: чужих конвертов сброшено, и по истечении TTL, и
	// по исчерпанию окна повторной выдачи.
	expired uint64
	// pulled — как клиент: конвертов получено с релеев, уже без повторов той же
	// криптосвязки (их гасит дедупликация выдач).
	pulled      uint64
	lastRefusal string // последняя причина отказа (своя или чужого релея)
}

// noteRefusal — единое место, где отказ превращается в новость: счётчик и
// причина. Молча отклонённый запрос к чужой почте неотличим от «ящика нет», а
// это ровно те два случая, которые пользователь путает.
func (s *sfStats) noteRefusal(format string, args ...interface{}) {
	s.refused++
	s.lastRefusal = fmt.Sprintf(format, args...)
}

// sfRelayKeys разбирает конфиг релеев. Негодный ключ — ошибка запуска, а не
// молча проигнорированный пункт: «релеи настроены, а почта никуда не кладётся»
// пользователь заметит только когда сообщение не дойдёт.
func sfRelayKeys(cfg *yggproto.NodeConfig) ([][]byte, error) {
	var out [][]byte
	for i, raw := range cfg.GetSfRelayNodeKeys() {
		if len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("rpc: sf_relay_node_keys[%d]: %d байт, нужно %d (Ed25519-ключ ноды)",
				i, len(raw), ed25519.PublicKeySize)
		}
		if !sfKeyUnique(out, raw) {
			continue
		}
		out = append(out, append([]byte(nil), raw...))
	}
	return out, nil
}

func sfKeyUnique(keys [][]byte, candidate []byte) bool {
	for _, k := range keys {
		if bytes.Equal(k, candidate) {
			return false
		}
	}
	return true
}

// sfMailboxOf — ящик этого узла в указанную эпоху. Ключ — мастер-X25519: ровно
// тот, что собеседник видит в карте контакта и по которому считает ящик сам.
func (e *Engine) sfMailboxOf(epochDay int64) [32]byte {
	return crypto.ComputeMailboxID(e.master.MasterX25519.PublicKey().Bytes(), epochDay)
}

func sfEpochDay(t time.Time) int64 { return t.Unix() / 86400 }

// mailboxArray — ящик из байтов запроса. Длина проверяется до сравнения:
// укороченный идентификатор иначе превратился бы в нуль-дополненный, а
// «совпал с нуль-дополненным» — это не «совпал с ящиком».
func mailboxArray(b []byte) ([32]byte, bool) {
	var out [32]byte
	if len(b) != 32 {
		return out, false
	}
	copy(out[:], b)
	return out, true
}

// -----------------------------------------------------------------------------
// ХРАНИТЕЛЬ
// -----------------------------------------------------------------------------

// acceptSFPut — релей складывает чужой конверт в ящик. Проверки идут в порядке
// «режим -> дешёвая формальность -> работа -> квота», и ни одна из них не читает
// начинку: иначе слепое хранение становится чтением чужой почты.
//
// Отказ здесь без ответа: кладка не просит подтверждения, а отправитель и так
// не узнаёт, что письмо у кого-то лежит — достоверный сигнал только ACK
// адресата. Причина отказа остаётся в счётчике и lastRefusal.
func (e *Engine) acceptSFPut(req *yggproto.SfPut) {
	if !e.sfRelayEnabled {
		e.sfStats.noteRefusal("rpc: узел не работает релеем, кладка отклонена")
		return
	}
	mailbox, ok := mailboxArray(req.GetMailboxId())
	env := req.GetEnvelope()
	if !ok || env == nil {
		e.sfStats.noteRefusal("rpc: кладка без ящика или конверта")
		return
	}
	// Индекс и назначение обязаны совпасть: конверт, положенный в ящик X при
	// blind_token Y, адресат из X не прочитает, а из Y — не найдёт.
	if !bytes.Equal(env.GetBlindToken(), mailbox[:]) {
		e.sfStats.noteRefusal("rpc: ящик кладки не совпал с blind_token конверта")
		return
	}
	if len(env.GetRecipientX25519Pubkey()) != 32 || len(env.GetEncryptedPayload()) == 0 {
		e.sfStats.noteRefusal("rpc: кладка с неполным конвертом")
		return
	}
	// Anti-spam лежит в самом конверте: релей платит место и ничем не может
	// проверить, что кладущий потрудился, кроме как пересчитав hashcash.
	if !crypto.VerifyHashcashPoW(mailbox, env.GetPowNonce(), powDifficulty) {
		e.sfStats.noteRefusal("rpc: кладка без годной hashcash-работы")
		return
	}

	durable, err := proto.Marshal(env)
	if err != nil {
		e.sfStats.noteRefusal("rpc: конверт кладки не сериализуется: %v", err)
		return
	}
	mbHex := hexKey(mailbox[:])
	count, size := e.db.SFMailbox(mbHex)
	if count >= sfMaxPerMailbox || size+int64(len(durable)) > sfMaxBytesPerMailbox {
		e.sfStats.noteRefusal("rpc: квота ящика исчерпана (%d конвертов, %d байт)", count, size)
		return
	}

	now := time.Now()
	id := sha256.Sum256(append(append([]byte(nil), mailbox[:]...), durable...))
	if err := e.db.SFPut(&storage.StoredSF{
		MailboxID:  mbHex,
		EnvelopeID: hexKey(id[:]),
		Data:       durable,
		PlacedMS:   now.UnixMilli(),
		ExpiresMS:  now.Add(sfTTL).UnixMilli(),
	}); err != nil {
		e.sfStats.noteRefusal("rpc: хранилище релея не приняло конверт: %v", err)
		return
	}
	// Счёт «хранится» двигается только по новому письму: хранилище гасит
	// повторную кладку тех же байт, и отчитаться ею как ещё одной единицей
	// чужой почты значило бы завысить то, что узел действительно держит.
	if after, _ := e.db.SFMailbox(mbHex); after > count {
		e.sfStats.stored++
	}
}

// acceptSFPull — выдача по ящику. Подпись сама по себе ничего не доказывает:
// подписать чем угодно умеет всякий, поэтому хранитель обязан увидеть, что
// мастер-Ed25519 подписанта привязан к мастер-X25519 (IdentityProof), и что
// запрошенный ящик выведен именно из него в названной эпохе. Без последнего
// шага достаточно знать открытый ключ человека, чтобы забрать его почту.
func (e *Engine) acceptSFPull(req *yggproto.SfPull, from ed25519.PublicKey) {
	deny := func(format string, args ...interface{}) {
		reason := fmt.Sprintf(format, args...)
		e.sfStats.noteRefusal("%s", reason)
		if err := e.replySF(from, &yggproto.SfReply{Error: reason}); err != nil {
			e.sfStats.lastRefusal = fmt.Sprintf("%s (и отказ не ушёл в сеть: %v)", reason, err)
		}
	}
	if !e.sfRelayEnabled {
		deny("rpc: узел не работает релеем")
		return
	}
	mailbox, ok := mailboxArray(req.GetMailboxId())
	if !ok {
		deny("rpc: ящик запроса не 32 байта")
		return
	}
	proof := proofFromWire(req.GetProof())
	if proof == nil {
		deny("rpc: запрос без связки мастер-ключей")
		return
	}
	if err := proof.Verify(); err != nil {
		deny("rpc: связка мастер-ключей не проходит: %v", err)
		return
	}
	if err := crypto.VerifySFPullSignature(proof.MasterEd25519, mailbox,
		req.GetEpochDay(), req.GetTimestampS(), req.GetSignature(), time.Now().Unix()); err != nil {
		deny("%v", err)
		return
	}
	if err := crypto.MailboxBelongsTo(proof.MasterX25519, req.GetEpochDay(), mailbox); err != nil {
		deny("%v", err)
		return
	}

	// Выдача не списывает письмо: хранилище остаётся до окна повторной выдачи,
	// поэтому перед выдачей проверяем обратный маршрут. Иначе попытка, до
	// адресата которой дороги нет, сожгла бы это окно молча.
	if !e.node.HasRouteTo(from) {
		deny("rpc: маршрута обратно к запрашивающему нет, почта осталась на хранении")
		return
	}

	taken, err := e.db.SFHandOut(hexKey(mailbox[:]), time.Now().UnixMilli())
	if err != nil {
		// Хранилище не отдало почту или не записало метку выдачи: выдавать ничего
		// нельзя, и просившему об этом сообщаем, а не делаем вид, что ящик пуст.
		deny("rpc: хранилище релея не отдало почту: %v", err)
		return
	}
	if len(taken) == 0 {
		// Пустой ящик — не отказ: писем нет, и это честный ответ.
		if err := e.replySF(from, &yggproto.SfReply{}); err != nil {
			e.sfStats.noteRefusal("rpc: пустой ответ релея не ушёл в сеть: %v", err)
		}
		return
	}

	reply := &yggproto.SfReply{}
	for _, rec := range taken {
		env := &yggproto.SealedEnvelope{}
		if err := proto.Unmarshal(rec.Data, env); err != nil {
			// Байты ящика не оказались конвертом: в выдачу чужой мусор не
			// вставляем, а с хранения его уберёт срок жизни.
			continue
		}
		reply.Envelopes = append(reply.Envelopes, env)
	}
	if err := e.replySF(from, reply); err != nil {
		// Ответ не лёг в сокет, а письмо осталось в ящике: записать такую выдачу
		// в выданные нельзя, адресат придёт за ней снова.
		e.sfStats.noteRefusal("rpc: выданное не ушло в сеть: %v", err)
		return
	}
	e.sfStats.handedOut += uint64(len(reply.Envelopes))
}

// replySF отправляет ответ тому, кто спросил. Адрес берём у оверлея (from — ключ
// узла, с которого реально пришли байты), а не из запроса: запрашивающий вправе
// назвать что угодно, а канал, по которому пришёл пакет, принадлежит ему.
//
// Счётчики здесь не двигаются: у отказа уже есть своя причина и свой
// noteRefusal у вызывающего, а удваивать их из-за того, что и сам ответ не
// дошёл, значит сделать статистику нечитаемой.
func (e *Engine) replySF(to ed25519.PublicKey, reply *yggproto.SfReply) error {
	frame, err := sfReplyFrame(reply)
	if err != nil {
		return fmt.Errorf("rpc: сборка ответа релея: %w", err)
	}
	if !e.running || e.node == nil {
		return errors.New("rpc: ответа релея некому отправить, ядро остановлено")
	}
	if len(to) != ed25519.PublicKeySize {
		return fmt.Errorf("rpc: ключ запрашивающего %d байт, нужно %d", len(to), ed25519.PublicKeySize)
	}
	if !e.node.HasRouteTo(to) {
		return errors.New("rpc: маршрута обратно к запрашивающему нет")
	}
	target := yggnode.AddrForPublicKey(to)
	if target == nil {
		return errors.New("rpc: ключ запрашивающего не годится как адрес")
	}
	n, err := e.node.PacketConn().WriteTo(frame, target)
	if err != nil {
		return fmt.Errorf("rpc: ответ релея не лёг в сокет: %w", err)
	}
	if n != len(frame) {
		return fmt.Errorf("rpc: ответ релея ушёл неполностью: %d из %d", n, len(frame))
	}
	return nil
}

// -----------------------------------------------------------------------------
// КЛИЕНТ: отправитель и получатель
// -----------------------------------------------------------------------------

// pushToRelays кладёт конверт на все настроенные релеи, до которых оверлей знает
// маршрут, и возвращает число переданных кадров. Локальная исходящая очередь при
// этом не снимается: единственный надёжный сигнал о том, что письмо дошло, — ACK
// адресата, а релей за хранение не ручается.
func (e *Engine) pushToRelays(env *yggproto.SealedEnvelope) uint64 {
	if !e.running || e.node == nil || len(e.sfRelays) == 0 || env == nil {
		return 0
	}
	frame, err := sfPutFrame(&yggproto.SfPut{MailboxId: env.GetBlindToken(), Envelope: env})
	if err != nil {
		e.sfStats.pushFailed++
		return 0
	}
	var pushed uint64
	for _, key := range e.sfRelays {
		nodeKey := ed25519.PublicKey(append([]byte(nil), key...))
		if !e.node.HasRouteTo(nodeKey) {
			continue
		}
		target := yggnode.AddrForPublicKey(nodeKey)
		if target == nil {
			continue
		}
		// WriteTo в ironwood принимает пакет всегда, поэтому здесь честно
		// считается «кадр передан оверлею», а не «релей его хранит».
		n, err := e.node.PacketConn().WriteTo(frame, target)
		if err != nil || n != len(frame) {
			continue
		}
		pushed++
	}
	if pushed == 0 {
		e.sfStats.pushFailed++
		return 0
	}
	e.sfStats.pushed += pushed
	return pushed
}

// relayMaxSends — сколько раз застрявший конверт кладётся слепым релеям. Лимит
// щедрее живого (outboxMaxSends): первый пакет на ещё не прогретой сессии
// ironwood теряется, и одной кладки мало. Бесконечно повторять незачем — ящик
// дедуплицируется по содержимому конверта, и повтором хранение не удлинить.
const relayMaxSends = 4

// relayPending кладёт на релеи конверт, застрявший в исходящей очереди: пир
// адресата мог появиться позже, а релей — быть доступен уже сейчас. Попытки
// считаются записью очереди, а не одним флагом: ровно одна кладка — это ровно
// один шанс, и ещё не прогретый маршрут способен потерять её так же, как живую
// отправку. Кладка, до релея которой маршрут ещё не созрел, в счётчик не идёт:
// таких попыток трафик не стоит, и они ничего не тратят.
func (e *Engine) relayPending(rec storage.StoredOutbound) {
	if rec.RelaySends >= relayMaxSends {
		return
	}
	frame, err := deframe(rec.Envelope)
	if err != nil {
		e.sfStats.noteRefusal("rpc: кадр из исходящей очереди не разбирается: %v", err)
		return
	}
	chat := frame.GetChat()
	if chat == nil {
		// В очереди лежат только chat-кадры; служебный пакет сюда не кладут.
		e.sfStats.noteRefusal("rpc: в исходящей очереди оказался не разговорный кадр")
		return
	}
	if e.pushToRelays(chat) > 0 {
		if err := e.db.RecordRelaySend(rec.MessageID, time.Now().UnixMilli()); err != nil {
			e.outStats.persistErr++
		}
	}
}

// collectFromRelays шлёт запросы на все релеи: по двум последним эпохам
// собственного ящика. Ответы приходят отдельными пакетами в приёмный цикл,
// поэтому команда возвращается сразу: ждать их, удерживая блокировку ядра,
// нельзя — тот же mutex нужен receiveLoop'у, и ожидание стало бы взаимной
// блокировкой, а не синхронной доставкой.
func (e *Engine) collectFromRelays() (uint32, []string, error) {
	if !e.running || e.master == nil || e.node == nil {
		return 0, nil, errors.New("rpc: ядро не запущено (отправьте StartNode)")
	}
	if len(e.sfRelays) == 0 {
		return 0, nil, nil
	}
	now := time.Now()
	day := sfEpochDay(now)
	var requested uint32
	used := make([]string, 0, len(e.sfRelays))

	for _, key := range e.sfRelays {
		nodeKey := ed25519.PublicKey(append([]byte(nil), key...))
		if !e.node.HasRouteTo(nodeKey) {
			continue
		}
		target := yggnode.AddrForPublicKey(nodeKey)
		if target == nil {
			continue
		}
		used = append(used, hexKey(key))
		for back := 0; back < sfEpochWindow; back++ {
			epoch := day - int64(back)
			mailbox := e.sfMailboxOf(epoch)
			ts := now.Unix()
			sig, err := crypto.BuildSFPullSignature(e.master, mailbox, epoch, ts)
			if err != nil {
				return requested, used, fmt.Errorf("rpc: подпись запроса к релею: %w", err)
			}
			frame, err := sfPullFrame(&yggproto.SfPull{
				MailboxId:  mailbox[:],
				EpochDay:   epoch,
				TimestampS: ts,
				Signature:  sig,
				Proof:      proofToWire(e.proof),
			})
			if err != nil {
				return requested, used, err
			}
			n, err := e.node.PacketConn().WriteTo(frame, target)
			if err != nil || n != len(frame) {
				e.sfStats.pushFailed++
				continue
			}
			requested++
		}
	}
	return requested, used, nil
}

// sfCollect — команда SfCollect: забрать почту прямо сейчас.
func (e *Engine) sfCollect(out *yggproto.SfCollectResp) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	requested, relays, err := e.collectFromRelays()
	out.Requested = requested
	out.Relays = relays
	return err
}

// sfTick — фоновая часть S&F в сканере очереди: выбросить просроченное чужое и
// сходить на релеи за своим. Вызывается без e.mu.
func (e *Engine) sfTick() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running || e.db == nil {
		return
	}
	if dropped := e.db.SFDropExpired(time.Now().UnixMilli()); dropped > 0 {
		e.sfStats.expired += uint64(dropped)
	}
	// Выданное живёт ещё окно повторной выдачи: пакет ответа тонет на оверлее
	// так же, как живой конверт, и адресат придёт за почтой при следующем опросе.
	// За окном письмо всё равно некому отдавать — оно списывается.
	if dropped := e.db.SFDropHanded(time.Now().Add(-sfRedelivery).UnixMilli()); dropped > 0 {
		e.sfStats.expired += uint64(dropped)
	}
	// Незакрытые fetch-запросы бандлов живут здесь же: ретрай того же кадра и
	// итоговая новость «узел не ответил» не должны зависеть от того, настроены
	// ли релеи (у запроса бандла с ними общего нет).
	e.expireBundleFetches(time.Now())
	if len(e.sfRelays) == 0 || e.node == nil {
		return
	}
	now := time.Now()
	if now.Before(e.sfNextPoll) {
		return
	}
	// Интервал ставится до запроса: отказ релея или ошибка подписи не должны
	// превращать опрос в цикл без паузы.
	e.sfNextPoll = now.Add(sfPollInterval)
	_, _, _ = e.collectFromRelays()
}

// acceptSFReply — ответ релея. Конверты отсюда идут в тот же приёмный путь, что
// и живые пакеты, но с пропущенными адресными воротами (см. acceptChat): релей
// не ручается за узел отправителя, он переносит байты, а личность отправителя
// доказывает связка внутри самого конверта.
//
// Учёт отказа идёт через noteRefusal: счётчик и причина пишутся вместе, иначе
// «пусто» и «не пустили» сливаются в одну молчаливую цифру.
func (e *Engine) acceptSFReply(reply *yggproto.SfReply, from ed25519.PublicKey) {
	if reply.GetError() != "" {
		e.sfStats.noteRefusal("релей %s отказал: %s", hexKey(from), reply.GetError())
		return
	}
	for _, env := range reply.GetEnvelopes() {
		id := sfEnvelopeID(env)
		if e.sfTaken[id] {
			// Тот же конверт выдан ещё раз: письмо этим узлом уже разобрано
			// (или отвергнуто), а вторичное раскрытие сдвинуло бы приёмную
			// цепочку и попало в счётчик отказов там, где отказа не было.
			e.rxStats.duplicated++
			continue
		}
		e.rememberSFTaken(id)
		e.sfStats.pulled++
		e.acceptChat(env, from, true)
	}
}

// sfTakenKeep — сколько выдач помнить. Окно пересылки релея — минуты, а не
// сутки, поэтому набор держим короткий: разросшаяся карта памяти ради дублей,
// которые всё равно кончатся сами, не стоит ничего.
const sfTakenKeep = 1024

// sfEnvelopeID — узнаваемый без расшифровывания след конверта: тройка
// (одноразовый ключ, nonce, тело), по которой он заперт. Релей отдаёт её дословно,
// поэтому повтор выдачи опознаётся до шага рэтчета.
func sfEnvelopeID(env *yggproto.SealedEnvelope) string {
	buf := make([]byte, 0, len(env.GetEphemeralX25519Pubkey())+len(env.GetNonce())+len(env.GetEncryptedPayload()))
	buf = append(buf, env.GetEphemeralX25519Pubkey()...)
	buf = append(buf, env.GetNonce()...)
	buf = append(buf, env.GetEncryptedPayload()...)
	sum := sha256.Sum256(buf)
	return hexKey(sum[:])
}

// rememberSFTaken кладёт след выдачи в конец очереди и выбрасывает самый старый,
// когда очередь переполняется.
func (e *Engine) rememberSFTaken(id string) {
	if len(e.sfTakenOrder) >= sfTakenKeep {
		oldest := e.sfTakenOrder[0]
		delete(e.sfTaken, oldest)
		e.sfTakenOrder = e.sfTakenOrder[1:]
	}
	if e.sfTaken == nil {
		e.sfTaken = make(map[string]bool)
	}
	e.sfTaken[id] = true
	e.sfTakenOrder = append(e.sfTakenOrder, id)
}

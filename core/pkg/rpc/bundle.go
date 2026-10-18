// File: core/pkg/rpc/bundle.go
package rpc

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"ygggram/core/pkg/yggnode"

	yggproto "ygggram/core/proto"
)

// Обмен PreKeyBundle поверх оверлея (Шаг 7). До этого контакта можно было
// коснуться только руками: AddContact требовал карту, а карту — откуда-то
// принесённую вне сети. Здесь узел по bundle_req отдаёт свой бандл, а
// спрашивающий превращает честный ответ в контакт.
//
// Два правила, без которых шаг бессмыслен:
//
//   - Ответ берётся в работу только если его принёс тот же узел, у которого
//     спрашивали: адрес в оверлее — это ed25519-ключ, поэтому pkt.From
//     аутентичен, и сверка с ним отсекает подмену карты чужим узлом. Одной
//     криптографии бандла мало: подпись заверяет пару «идентичность ↔ ключ
//     ноды», но не отвечает на вопрос «этот ли узел мне ответил».
//   - Карта проходит ровно те же ворота, что и живая AddContact (verifyContact),
//     и тем же путём сохраняется. Сетевой ответ — не доверенный вход.

const (
	// bundleServeInterval — как часто одному и тому же узлу отдаём бандл. Ответ
	// выпускает новый one-time prekey, поэтому запрос-шторм не должен
	// вычерпывать склад ключей; отказ назван словами, чтобы клиент не ждал
	// вечно.
	bundleServeInterval = 2 * time.Second
	// bundleFetchTimeout — сколько живёт fetch-запрос без ответа. дольше ждать
	// смысла нет: узел либо ответил, либо его нет в сети, а UI обязан получить
	// новость, а не вращать индикатор.
	bundleFetchTimeout = 30 * time.Second
	// bundleRetryInterval и bundleMaxSends — про то, что первый пакет на ещё не
	// прогретой сессии ironwood теряется. Повторяются ровно те же байты запроса:
	// request_id тот же, и два ответа на один запрос гасятся по нему же.
	bundleRetryInterval = 5 * time.Second
	bundleMaxSends      = 3
)

// bundleFetch — незакрытый fetch-запрос. frame хранится, потому что повтор
// обязан быть байт в байт: request_id — единственная ниточка, связывающая
// ответ с запросом.
type bundleFetch struct {
	nodeKey  []byte
	nickname string
	frame    []byte
	sentAt   time.Time
	sends    uint32
}

// bundleStats — учёт раздачи и запроса бандлов. Роль сервера и роль клиента
// считаются раздельно, как в S&F: «мне отдали» и «я отдал» — разные новости.
type bundleStats struct {
	requested uint64 // fetch-запросы, ушедшие в сеть
	serves    uint64 // бандлов отдано
	refused   uint64 // в раздаче отказано (роль выключена, интервал не вышел)
	resolved  uint64 // ответов принято и контактов заведено
	failed    uint64 // ответ не прошёл ворота или пришёл не от того узла
	// expired — запросы без ответа: узел молчит. Терять их молча нельзя,
	// именно поэтому по ним уходит событие с причиной, а не просто счётчик.
	expired uint64
	// unmatched — ответы на запросы, которых уже нет (дубликат после истечения
	// окна, опоздавший из сети).
	unmatched uint64
	// sendFailed — ответ не ушёл в сеть: кадр не собрался или сокет не принял
	// байты. Без этого счётчика «отказали клиенту» и «отказалось сказать»
	// сливаются в одну молчаливую цифру.
	sendFailed uint64
	// persistErr — хранилище не приняло prekey-материал или карту контакта:
	// учёт попыток поехал, а обещать «контакт добавлен» нельзя.
	persistErr uint64
}

// fetchContact — точка команды FetchContact: отправить bundle_req и
// зарегистрировать ожидание. Ответ придёт в приёмный цикл, поэтому здесь мы не
// ждём: ждать, удерживая e.mu, нельзя — тот же mutex нужен handleInbound,
// который этот ответ и разберёт.
func (e *Engine) fetchContact(req *yggproto.FetchContactReq, out *yggproto.FetchContactResp) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	nodeKey := req.GetNodeEd25519Pubkey()
	if len(nodeKey) != ed25519.PublicKeySize {
		return errors.New("rpc: FetchContact без ed25519-ключа узла (32 байта)")
	}
	if bytes.Equal(nodeKey, e.node.PublicKey()) {
		return errors.New("rpc: этот ключ принадлежит самому узлу — себя в контакты добавлять нечего")
	}
	target := yggnode.AddrForPublicKey(nodeKey)
	if target == nil {
		return fmt.Errorf("rpc: ключ узла %s не разбирается как адрес", hexKey(nodeKey))
	}
	if !e.node.HasRouteTo(nodeKey) {
		return fmt.Errorf("rpc: маршрута к узлу %s нет — сначала ConnectPeer", yggnode.AddressForKey(nodeKey).String())
	}

	e.bundleSeq++
	id := e.bundleSeq
	frame, err := bundleReqFrame(&yggproto.BundleReq{RequestId: id})
	if err != nil {
		return err
	}
	n, err := e.node.PacketConn().WriteTo(frame, target)
	if err != nil || n != len(frame) {
		// Ожидание не регистрируется: просроченный запрос без отправленного
		// байта отчитался бы «узел молчит» там, где запрос даже не ушёл.
		return fmt.Errorf("rpc: bundle_req не ушёл в сеть: %w", err)
	}
	e.bundlePending[id] = &bundleFetch{
		nodeKey:  append([]byte(nil), nodeKey...),
		nickname: req.GetNickname(),
		frame:    frame,
		sentAt:   time.Now(),
		sends:    1,
	}
	e.bundleStats.requested++
	out.Requested = true
	out.RequestId = id
	out.NodeIpv6 = yggnode.AddressForKey(nodeKey).String()
	return nil
}

// acceptBundleReq — роль сервера: отдать свой бандл. Проверяются только право
// узла его получить и частота: сама раздача ничего не шифрует и не хранит, но
// каждый ответ выпускает новый one-time prekey.
func (e *Engine) acceptBundleReq(req *yggproto.BundleReq, from ed25519.PublicKey) {
	respond := func(resp *yggproto.BundleResp) {
		frame, err := bundleRespFrame(resp)
		if err != nil {
			e.bundleStats.sendFailed++
			return
		}
		target := yggnode.AddrForPublicKey(from)
		if target == nil {
			e.bundleStats.sendFailed++
			return
		}
		// Пакет ответа — первый на ещё не прогретом обратном маршруте и тонет
		// ровно как в S&F. Честный ответ тут — счётчик: клиент повторит
		// запрос сам, а интервал раздачи ему в этом не стена, потому что отказ
		// уже назван.
		if n, err := e.node.PacketConn().WriteTo(frame, target); err != nil || n != len(frame) {
			e.bundleStats.sendFailed++
		}
	}

	if !e.serveBundles {
		e.bundleStats.refused++
		respond(&yggproto.BundleResp{RequestId: req.GetRequestId(), Error: "rpc: узел не раздаёт PreKeyBundle (serve_prekey_bundles)"})
		return
	}
	key := hexKey(from)
	now := time.Now()
	if last, ok := e.bundleLastServed[key]; ok && now.Sub(last) < bundleServeInterval {
		e.bundleStats.refused++
		respond(&yggproto.BundleResp{RequestId: req.GetRequestId(), Error: "rpc: бандл уже отдавали этому узлу, повторите запрос позже"})
		return
	}
	bundle := &yggproto.PreKeyBundle{}
	if err := e.preKeyBundle(bundle); err != nil {
		// Без годного prekey-материала бандл не отдаётся: выдать ключ, который
		// хранилище не запомнило, значит обречь отправителя на расшифровку,
		// которая не переживёт рестарт.
		e.bundleStats.persistErr++
		respond(&yggproto.BundleResp{RequestId: req.GetRequestId(), Error: fmt.Sprintf("rpc: бандл не собран: %v", err)})
		return
	}
	e.bundleLastServed[key] = now
	e.bundleStats.serves++
	respond(&yggproto.BundleResp{RequestId: req.GetRequestId(), Bundle: bundle})
}

// acceptBundleResp — роль клиента: ответ на свой bundle_req. Ворота идут в
// порядке «дешёвые и точные»: сначала сверка с тем, у кого спрашивали, потом
// общие ворота контакта, и только затем запись в базу.
func (e *Engine) acceptBundleResp(resp *yggproto.BundleResp, from ed25519.PublicKey) {
	pend, ok := e.bundlePending[resp.GetRequestId()]
	if !ok {
		// Дубликат или опоздавший ответ: контакта не касаемся, но счётчик
		// остаётся — иначе «ответов было два» и «ответа не было» неразличимы.
		e.bundleStats.unmatched++
		return
	}
	if msg := resp.GetError(); msg != "" {
		delete(e.bundlePending, resp.GetRequestId())
		e.bundleStats.failed++
		e.emitContactResolved(resp.GetRequestId(), nil, fmt.Sprintf("узел %s отказал: %s", hexKey(from), msg))
		return
	}
	// Адрес в оверлее — это ed25519-ключ, и pkt.From даёт его сам транспорт.
	// Спрашивали узел A, ответил B: подставленный бандл ушёл бы в базу как
	// честный контакт, а почта поехала бы по адресу B.
	if !bytes.Equal(from, pend.nodeKey) {
		delete(e.bundlePending, resp.GetRequestId())
		e.bundleStats.failed++
		e.emitContactResolved(resp.GetRequestId(), nil, fmt.Sprintf(
			"ответ пришёл от узла %s, а спрашивали %s", hexKey(from), hexKey(pend.nodeKey)))
		return
	}
	bundle := resp.GetBundle()
	if bundle == nil {
		delete(e.bundlePending, resp.GetRequestId())
		e.bundleStats.failed++
		e.emitContactResolved(resp.GetRequestId(), nil, "узел прислал пустой PreKeyBundle")
		return
	}
	if !bytes.Equal(bundle.GetNodeEd25519Pubkey(), pend.nodeKey) {
		delete(e.bundlePending, resp.GetRequestId())
		e.bundleStats.failed++
		e.emitContactResolved(resp.GetRequestId(), nil, fmt.Sprintf(
			"в бандле ключ ноды %s, а спрашивали %s", hexKey(bundle.GetNodeEd25519Pubkey()), hexKey(pend.nodeKey)))
		return
	}

	card := &yggproto.ContactCard{
		NodeIpv6:      yggnode.AddressForKey(pend.nodeKey).String(),
		Ed25519Pubkey: append([]byte(nil), pend.nodeKey...),
		X25519Pubkey:  bundle.GetIdentityX25519Pubkey(),
		Nickname:      pend.nickname,
		UpdatedAt:     uint64(time.Now().UnixMilli()),
		// Флаг «проверен» тут остаётся false намеренно: сеть доказывает, что
		// бандл принёс именно тот узел, у которого спрашивали, но не то, что за
		// этим ключом стоит знакомый человек. Это по-прежнему доверие при
		// первой встрече — ключ кто-то должен был передать вне сети.
		Bundle: bundle,
	}
	// Ворота те же, что у живой AddContact: адрес строкой выше берётся из того
	// же ключа, поэтому verifyContact проверяет здесь связку identity-ключей и
	// сертификат ноды, а не пересказ того, что уже проверено.
	if err := verifyContact(card); err != nil {
		delete(e.bundlePending, resp.GetRequestId())
		e.bundleStats.failed++
		e.emitContactResolved(resp.GetRequestId(), nil, err.Error())
		return
	}
	if err := e.addContact(card); err != nil {
		delete(e.bundlePending, resp.GetRequestId())
		e.bundleStats.persistErr++
		e.emitContactResolved(resp.GetRequestId(), nil, err.Error())
		return
	}
	delete(e.bundlePending, resp.GetRequestId())
	e.bundleStats.resolved++
	e.emitContactResolved(resp.GetRequestId(), card, "")
}

// expireBundleFetches — обслуживание незакрытых запросов из фонового цикла.
// Повтор отправляет ровно те же байты: потеря первого пакета на холодном
// маршруте не требует нового request_id, а дубликат ответа гасится таблицей
// незакрытых запросов.
func (e *Engine) expireBundleFetches(now time.Time) {
	if !e.running || len(e.bundlePending) == 0 {
		return
	}
	for id, pend := range e.bundlePending {
		age := now.Sub(pend.sentAt)
		if age >= bundleFetchTimeout {
			delete(e.bundlePending, id)
			e.bundleStats.expired++
			e.emitContactResolved(id, nil, fmt.Sprintf(
				"узел %s не ответил за %s", yggnode.AddressForKey(pend.nodeKey).String(), bundleFetchTimeout))
			continue
		}
		if pend.sends >= bundleMaxSends || age < bundleRetryInterval {
			continue
		}
		nodeKey := ed25519.PublicKey(append([]byte(nil), pend.nodeKey...))
		if !e.node.HasRouteTo(nodeKey) {
			continue
		}
		target := yggnode.AddrForPublicKey(nodeKey)
		if target == nil {
			continue
		}
		if n, err := e.node.PacketConn().WriteTo(pend.frame, target); err != nil || n != len(pend.frame) {
			continue
		}
		pend.sends++
		pend.sentAt = now
		e.bundleStats.requested++
	}
}

// emitContactResolved — итог fetch-запроса в событийный канал. Без него UI
// видел бы только вечное «ожидание»: успех и отказ здесь принципиально
// разные новости, и обе обязан знать пользователь.
func (e *Engine) emitContactResolved(requestID uint64, card *yggproto.ContactCard, reason string) {
	e.emit(&yggproto.CoreEvent{
		EventId:     e.nextEventID(),
		TimestampMs: uint64(time.Now().UnixMilli()),
		Event: &yggproto.CoreEvent_ContactUpdated{ContactUpdated: &yggproto.ContactResolvedEvent{
			RequestId: requestID,
			Contact:   card,
			Error:     reason,
		}},
	})
}

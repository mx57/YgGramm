// File: core/pkg/rpc/peers_test.go
package rpc

import (
	"fmt"
	"strings"
	"testing"
	"time"

	yggproto "ygggram/core/proto"
)

// awaitPeers вычитывает события ядра, пока не попадётся peers_updated, и
// возвращает снятый список. Остальные события (статусы сообщений, контакты)
// пропускаются: порядок в канале ничем не гарантирован, а проверка должна
// зависеть именно от объявления вершины.
func awaitPeers(t *testing.T, e *Engine, limit time.Duration) *yggproto.PeerListUpdatedEvent {
	t.Helper()
	deadline := time.Now().Add(limit)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatal("события peers_updated не дождались")
			return nil
		}
		select {
		case ev := <-e.Events():
			if pl := ev.GetPeersUpdated(); pl != nil {
				return pl
			}
		case <-time.After(remaining):
			t.Fatal("события peers_updated не дождались")
			return nil
		}
	}
}

// awaitPeer ждёт объявления, в котором пир адреса ipv6 проходит проверку want.
// Первая публикация после ConnectPeer может показать линк ещё не поднявшимся
// (AddPeer регистрирует намерение до рукопожатия), поэтому ждём именно нужное
// состояние, а не первое упоминание.
func awaitPeer(t *testing.T, e *Engine, ipv6 string, limit time.Duration,
	want func(*yggproto.PeerInfo) bool) *yggproto.PeerInfo {
	t.Helper()
	deadline := time.Now().Add(limit)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatalf("пира %s в нужном состоянии в событиях peers_updated не дождались", ipv6)
			return nil
		}
		select {
		case ev := <-e.Events():
			pl := ev.GetPeersUpdated()
			if pl == nil {
				continue
			}
			for _, p := range pl.GetPeers() {
				if p.GetYggdrasilIpv6() == ipv6 && want(p) {
					return p
				}
			}
		case <-time.After(remaining):
			t.Fatalf("пира %s в нужном состоянии в событиях peers_updated не дождались", ipv6)
			return nil
		}
	}
}

// noPeerOf объявление вершины, где адреса ipv6 либо вовсе нет, либо линк не
// поднят: так выглядит «пир пропал».
func noPeerOf(pl *yggproto.PeerListUpdatedEvent, ipv6 string) bool {
	for _, p := range pl.GetPeers() {
		if p.GetYggdrasilIpv6() == ipv6 && p.GetIsConnected() {
			return false
		}
	}
	return true
}

// TestPeersUpdatedEventTracksOverlayTopology — смысл Шага 9-a: вершина оверлея
// перестаёт быть тем, что UI обязан выпрашивать через GetPeers. Ядро само
// объявляет peers_updated при структурном изменении набора пиров — и ровно один
// раз на изменение.
//
// Проверяется на настоящем оверлее: две ноды, живой линк, оба конца получают
// своё объявление (исходящее и входящее соединение выглядят по-разному).
func TestPeersUpdatedEventTracksOverlayTopology(t *testing.T) {
	if testing.Short() {
		t.Skip("оверлейный тест: нужен реальный линк между двумя нодами")
	}
	port := freeTCPPort(t)

	bob := startEngineWith(t, seedBob, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(port)
	})
	alice := startEngineWith(t, seedAlice, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
	})

	bobAddr := bob.node.NodeIPv6().String()
	aliceAddr := alice.node.NodeIPv6().String()

	// Первое объявление — пустой список: «нода поднялась, пиров нет» тоже обязано
	// дойти до UI, иначе он показывал бы вершину прежнего запуска.
	if first := awaitPeers(t, alice, 30*time.Second); len(first.GetPeers()) != 0 {
		t.Fatalf("первое объявление вершины непустое: %+v", first.GetPeers())
	}

	peerResp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_ConnectPeer{
		ConnectPeer: &yggproto.ConnectPeerReq{PeerUri: fmt.Sprintf("127.0.0.1:%d", port)},
	}})
	if !peerResp.GetSuccess() {
		t.Fatalf("ConnectPeer: %s", peerResp.GetErrorMessage())
	}

	// Инициирующая сторона: линк поднят, соединение исходящее, tcp.
	up := awaitPeer(t, alice, bobAddr, 60*time.Second, func(p *yggproto.PeerInfo) bool {
		return p.GetIsConnected()
	})
	if up.GetIsInbound() {
		t.Fatalf("исходящее соединение помечено входящим: %+v", up)
	}
	if up.GetTransport() != yggproto.PeerTransportType_TRANSPORT_TCP {
		t.Fatalf("транспорт пира неверный: %v", up.GetTransport())
	}

	// Принимающая сторона видит тот же линк как входящий и знает адрес инициатора.
	awaitPeer(t, bob, aliceAddr, 60*time.Second, func(p *yggproto.PeerInfo) bool {
		return p.GetIsConnected() && p.GetIsInbound()
	})

	// Окно тишины: сканер тикает каждые outboxInterval, и если бы объявление не
	// было диффом, UI получал бы один и тот же список каждую пару секунд.
	quiet := time.Now().Add(3*outboxInterval + time.Second)
	for {
		remaining := time.Until(quiet)
		if remaining <= 0 {
			break
		}
		select {
		case ev := <-alice.Events():
			if pl := ev.GetPeersUpdated(); pl != nil {
				t.Fatalf("вершина объявлена повторно без изменения: %+v", pl.GetPeers())
			}
		case <-time.After(remaining):
		}
	}

	// Настоящее изменение объявляется снова: Боб гасит ноду, и Алиса обязана
	// донести пропавший линк, а не держать его в UI.
	stopResp := bob.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_StopNode{}})
	if !stopResp.GetSuccess() {
		t.Fatalf("StopNode у Боба: %s", stopResp.GetErrorMessage())
	}
	if !waitUntil(t, 30*time.Second, func() bool {
		select {
		case ev := <-alice.Events():
			if pl := ev.GetPeersUpdated(); pl != nil {
				return noPeerOf(pl, bobAddr)
			}
			return false
		case <-time.After(50 * time.Millisecond):
			return false
		}
	}) {
		t.Fatalf("Алиса не объявила пропавший линк: пиры %v", alice.node.Peers())
	}
}

// TestDisconnectPeerDropsLiveLink — disconnect_peer из контракта обязан рвать
// линк, а не только отвечать success: ядро yggdrasil отменяет контекст линка и
// закрывает соединение, поэтому пир пропадает из Peers() и из снимка, который
// несёт ответ команды. За маршрутом spanning-дерева проверка не идёт намеренно
// — см. комментарий в середине теста.
//
// Проверка идёт на настоящем TCP-оверлее и именно с тем написанием URI, с
// каким пир добавлялся (без схемы): команда обязана найти тот же линк, а не
// отвергнуть «пира нет в настройке» из-за другого способа записать адрес.
func TestDisconnectPeerDropsLiveLink(t *testing.T) {
	if testing.Short() {
		t.Skip("оверлейный тест: нужен реальный линк между двумя нодами")
	}
	port := freeTCPPort(t)
	uri := fmt.Sprintf("127.0.0.1:%d", port)

	bob := startEngineWith(t, seedBob, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
		cfg.ListenAddress = "127.0.0.1"
		cfg.ListenPort = int32(port)
	})
	alice := startEngineWith(t, seedAlice, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
	})
	bobAddr := bob.node.NodeIPv6().String()

	if resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_ConnectPeer{
		ConnectPeer: &yggproto.ConnectPeerReq{PeerUri: uri},
	}}); !resp.GetSuccess() {
		t.Fatalf("ConnectPeer: %s", resp.GetErrorMessage())
	}
	awaitPeer(t, alice, bobAddr, 60*time.Second, func(p *yggproto.PeerInfo) bool {
		return p.GetIsConnected()
	})
	waitForRoute(t, alice, bob.node.PublicKey())

	resp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_DisconnectPeer{
		DisconnectPeer: &yggproto.DisconnectPeerReq{PeerUri: uri},
	}})
	if !resp.GetSuccess() {
		t.Fatalf("DisconnectPeer(%s): %s", uri, resp.GetErrorMessage())
	}
	// Ответ несёт свежий снимок вершины ровно тем же сообщением, что GetPeers и
	// connect_peer: три пути к одному списку расходятся молча.
	if resp.GetPeerList() == nil {
		t.Fatalf("DisconnectPeer не вернул список пиров: %+v", resp)
	}
	droppedAt := time.Now()

	// Линк гаснет асинхронно (отмена контекста живёт в горутине оверлея),
	// поэтому ждём факта, а не читаем список сразу после команды.
	if !waitUntil(t, 30*time.Second, func() bool {
		for _, p := range alice.node.Peers() {
			if p.Up && strings.Contains(p.URI, fmt.Sprintf(":%d", port)) {
				return false
			}
		}
		return true
	}) {
		t.Fatalf("линк не разорван: пиры %v", alice.node.Peers())
	}
	t.Logf("линк погас через %s после команды", time.Since(droppedAt).Round(time.Millisecond))
	// Маршрут проверки тут нет намеренно: ironwood снимает запись о ключе не
	// когда линк умер, а когда истечёт routerTimeout — по умолчанию 5 минут
	// (network/config.go, удаление в network/router.go._update). Замер
	// 2026-10-16: через 150 с после disconnect_peer ключ адресата всё ещё
	// сидел в вершине, то есть HasRouteTo врал. Это не свойство команды, а
	// свойство оверлея (ровно то же при обрыве без команды), и именно поэтому
	// «доставлено» подтверждает только ACK, а очередь переотправляет.
	if !waitUntil(t, 30*time.Second, func() bool {
		select {
		case ev := <-alice.Events():
			if pl := ev.GetPeersUpdated(); pl != nil {
				return noPeerOf(pl, bobAddr)
			}
			return false
		case <-time.After(50 * time.Millisecond):
			return false
		}
	}) {
		t.Fatalf("вершина без разорванного линка не объявлена: пиры %v", alice.node.Peers())
	}

	// Повтор того же URI — названный отказ, а не success: иначе «убрали»
	// посчитается дважды выполненным, и UI потеряет границу между «пира нет»
	// и «мы его выключили».
	again := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_DisconnectPeer{
		DisconnectPeer: &yggproto.DisconnectPeerReq{PeerUri: uri},
	}})
	if again.GetSuccess() {
		t.Fatalf("повтор DisconnectPeer прошёл успешно, хотя пира больше нет: %+v", again.GetPeerList().GetPeers())
	}
}

// TestDisconnectPeerRejectsBadInput — граница команды: без поднятой ноды, с
// пустым URI и с неизвестным транспортом отказ приходит до обращения к
// оверлею, и ни один из них не превращается в молчаливый успех.
func TestDisconnectPeerRejectsBadInput(t *testing.T) {
	cold := New()
	resp := cold.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_DisconnectPeer{
		DisconnectPeer: &yggproto.DisconnectPeerReq{PeerUri: "127.0.0.1:1"},
	}})
	if resp.GetSuccess() {
		t.Fatal("disconnect без запущенной ноды прошёл")
	}

	alice := startEngineWith(t, seedAlice, t.TempDir(), func(cfg *yggproto.NodeConfig) {
		cfg.AntiDpiMode = yggproto.AntiDpiMode_ANTI_DPI_DISABLED
	})
	for _, bad := range []string{"", "   ", "udp://127.0.0.1:9009", "127.0.0.1"} {
		got := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_DisconnectPeer{
			DisconnectPeer: &yggproto.DisconnectPeerReq{PeerUri: bad},
		}})
		if got.GetSuccess() {
			t.Errorf("пустой или невозможный URI %q принят", bad)
		}
	}
}

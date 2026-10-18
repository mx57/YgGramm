package rpc

import (
	"fmt"
	"strings"
	"testing"
	"time"

	yggproto "ygggram/core/proto"
)

// TestLinkSNIAndHandshakeNotes — разбор ссылки на схему, хост и SNI. Это та
// часть anti_dpi_status, которую событие считает само, без оверлея: ошибиться
// тут значит соврать в том, что видно посреднику, поэтому таблица покрывает
// ровно те формы URI, которые ядро строит для исходящих ссылок.
func TestLinkSNIAndHandshakeNotes(t *testing.T) {
	cases := []struct {
		uri    string
		scheme string
		host   string
		sni    string
	}{
		// Схема по умолчанию — tcp, как у yggdrasil для «голого» адреса.
		{"127.0.0.1:8666", "tcp", "127.0.0.1", ""},
		{"tcp://relay.example.org:8666", "tcp", "relay.example.org", ""},
		// Имя уходит в SNI только там, где есть TLS.
		{"tls://relay.example.org:8443", "tls", "relay.example.org", "relay.example.org"},
		{"wss://cdn.example.org:443", "wss", "cdn.example.org", "cdn.example.org"},
		// Литеральный адрес: crypto/tls не отправляет SNI, и это замерено на
		// живом линке (Шаг 4-b), а не выведено из spec.
		{"tls://127.0.0.1:8443", "tls", "127.0.0.1", ""},
		{"tls://[::1]:8443", "tls", "::1", ""},
		{"quic://[2001:db8::1]:8443", "quic", "2001:db8::1", ""},
		{"socks://peer.example.org:8666", "socks", "peer.example.org", ""},
	}
	for _, c := range cases {
		scheme, host := splitLink(c.uri)
		if scheme != c.scheme || host != c.host {
			t.Errorf("splitLink(%q) = %q, %q; ждали %q, %q", c.uri, scheme, host, c.scheme, c.host)
		}
		if got := linkSNI(c.uri); got != c.sni {
			t.Errorf("linkSNI(%q) = %q; ждали %q", c.uri, got, c.sni)
		}
	}

	// Схема без TLS не вправе описываться как рукопожатие: первая версия
	// события делала это для всех ссылок подряд и врала ровно на tcp://.
	for _, s := range []string{"tcp", "ws", "socks"} {
		if carriesTLS(s) {
			t.Errorf("carriesTLS(%q) = true", s)
		}
		if n := linkHandshakeNote(s); !strings.Contains(n, "нет") && !strings.Contains(n, "не мы") {
			t.Errorf("заметка схемы %q не говорит об отсутствии нашего рукопожатия: %s", s, n)
		}
	}
	for _, s := range []string{"tls", "wss"} {
		if !carriesTLS(s) {
			t.Errorf("carriesTLS(%q) = false", s)
		}
		if n := linkHandshakeNote(s); !strings.Contains(n, "crypto/tls") {
			t.Errorf("заметка схемы %q не называет настоящего автора рукопожатия: %s", s, n)
		}
	}
}

// awaitDpi ждёт объявления anti_dpi_status, подходящего под want. Другие ветки
// CoreEvent при этом расходуются: канал один, и оставляя их в очереди, тест
// заглушил бы следующий peers_updated.
func awaitDpi(t *testing.T, e *Engine, limit time.Duration, want func(*yggproto.AntiDpiStatusEvent) bool) *yggproto.AntiDpiStatusEvent {
	t.Helper()
	deadline := time.Now().Add(limit)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatal("нужного объявления anti_dpi_status не дождались")
			return nil
		}
		select {
		case ev := <-e.Events():
			if s := ev.GetAntiDpiStatus(); s != nil && want(s) {
				return s
			}
		case <-time.After(remaining):
		}
	}
}

// TestAntiDpiStatusDescribesLiveLink — Шаг 16-d: событие строится из живых
// данных оверлея, а не из намерений. Проверяются ровно те границы честности,
// которые заявлены в комментарии к publishAntiDpi: mode из конфига, SNI из URI
// (литеральный адрес — без SNI), handshake_success по Up линка, входящий линк
// в описание не входит, а fallback не выдумывается.
func TestAntiDpiStatusDescribesLiveLink(t *testing.T) {
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

	// До ConnectPeer сравнивать нечего — и событие обязано сказать это прямо,
	// а не молчать до появления линка.
	empty := awaitDpi(t, alice, 30*time.Second, func(s *yggproto.AntiDpiStatusEvent) bool {
		return s.GetTransportDetails() != ""
	})
	if !strings.Contains(empty.GetTransportDetails(), "исходящих ссылок нет") {
		t.Fatalf("пустая вершина описана нечестно: %s", empty.GetTransportDetails())
	}
	if empty.GetHandshakeSuccess() {
		t.Fatalf("у ноды без исходящих ссылок handshake_success истинно: %+v", empty)
	}
	if empty.GetMode() != yggproto.AntiDpiMode_ANTI_DPI_DISABLED {
		t.Fatalf("mode не совпадает с конфигом: %v", empty.GetMode())
	}

	peerResp := alice.Handle(&yggproto.CoreRequest{Request: &yggproto.CoreRequest_ConnectPeer{
		ConnectPeer: &yggproto.ConnectPeerReq{PeerUri: fmt.Sprintf("127.0.0.1:%d", port)},
	}})
	if !peerResp.GetSuccess() {
		t.Fatalf("ConnectPeer: %s", peerResp.GetErrorMessage())
	}

	up := awaitDpi(t, alice, 90*time.Second, func(s *yggproto.AntiDpiStatusEvent) bool {
		return s.GetHandshakeSuccess()
	})
	// Переключения пути в живом коде нет — значит поле обязано быть ложным.
	if up.GetFallbackTriggered() {
		t.Fatalf("fallback_triggered выдуман при отсутствующем фолбэке: %+v", up)
	}
	// Адрес литеральный — SNI нет, и событие не должно его приписывать.
	if up.GetCurrentSni() != "" {
		t.Fatalf("ссылке на литеральный адрес приписан SNI %q", up.GetCurrentSni())
	}
	details := up.GetTransportDetails()
	for _, want := range []string{"tcp://", "TLS-рукопожатия нет"} {
		if !strings.Contains(details, want) {
			t.Errorf("в описании ссылок нет %q: %s", want, details)
		}
	}
	if strings.Contains(details, "crypto/tls") {
		t.Errorf("tcp-ссылке приписано TLS-рукопожатие: %s", details)
	}

	// Принимающая сторона линк видит, но исходящим путём она не управляет: её
	// описание остаётся «сравнивать нечего». Снимок берётся напрямую — входящий
	// линк не меняет вершину, поэтому второго объявления от дедупа не будет, и
	// ждать его в канале было бы ожиданием несуществующего события.
	awaitPeer(t, bob, alice.node.NodeIPv6().String(), 90*time.Second, func(p *yggproto.PeerInfo) bool {
		return p.GetIsConnected() && p.GetIsInbound()
	})
	bobView, _ := bob.antiDpiStatus()
	if !strings.Contains(bobView.GetTransportDetails(), "исходящих ссылок нет") {
		t.Fatalf("принимающая сторона описала чужой исходящий путь как свой: %s", bobView.GetTransportDetails())
	}
	if bobView.GetHandshakeSuccess() {
		t.Fatalf("у принимающей стороны handshake_success истинно при чужом рукопожатии: %+v", bobView)
	}

	// Окно тишины: вершина не менялась, значит повторного объявления быть не
	// должно. Без дедупа событие стало бы опросом каждые outboxInterval.
	quiet := time.Now().Add(3*outboxInterval + time.Second)
	for {
		remaining := time.Until(quiet)
		if remaining <= 0 {
			break
		}
		select {
		case ev := <-alice.Events():
			if s := ev.GetAntiDpiStatus(); s != nil {
				t.Fatalf("anti_dpi_status объявлен повторно без изменения: %+v", s)
			}
		case <-time.After(remaining):
		}
	}
}

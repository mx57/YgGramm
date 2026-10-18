package yggnode

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"testing"
	"time"
)

func nodeKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return priv
}

// freeTCPPort отдаёт порт, который тест собирается занять под слушателя.
// Гонка возможна: между закрытием listener и стартом ноды порт может
// занять кто-то другой — для loopback-теста это приемлемо.
func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

func waitFor(limit time.Duration, ok func() bool) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if ok() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return ok()
}

// TestTwoNodesExchangePackets — главный смысл Шага 2: два узла в одном
// процессе сходятся по настоящему TCP, строят spanning-дерево и передают
// пакет через userspace-доступ без TUN.
func TestTwoNodesExchangePackets(t *testing.T) {
	port := freeTCPPort(t)
	uri := fmt.Sprintf("tcp://127.0.0.1:%d", port)

	a, err := NewYggNode(NodeOptions{PrivateKey: nodeKey(t), Listen: []string{uri}})
	if err != nil {
		t.Fatalf("сборка узла A: %v", err)
	}
	if err := a.Start(); err != nil {
		t.Fatalf("старт A: %v", err)
	}
	defer func() { _ = a.Stop() }()

	b, err := NewYggNode(NodeOptions{PrivateKey: nodeKey(t), Peers: []string{uri}})
	if err != nil {
		t.Fatalf("сборка узла B: %v", err)
	}
	if err := b.Start(); err != nil {
		t.Fatalf("старт B: %v", err)
	}
	defer func() { _ = b.Stop() }()

	if !waitFor(20*time.Second, func() bool {
		return b.HasRouteTo(a.PublicKey()) && a.HasRouteTo(b.PublicKey())
	}) {
		t.Fatalf("маршрут так и не появился: tree B=%d A=%d, peers B=%v",
			b.RoutingEntries(), a.RoutingEntries(), b.Peers())
	}

	// Адресная модель обязана совпадать с эталонной: IPv6 узла выводится из
	// его Ed25519-ключа.
	if !a.NodeIPv6().Equal(AddressForKey(a.PublicKey())) {
		t.Fatalf("NodeIPv6 %v не совпадает с AddressForKey %v", a.NodeIPv6(), AddressForKey(a.PublicKey()))
	}

	inbound, cancel := a.Subscribe(16)
	defer cancel()

	payload := []byte("проверка маршрута ygggram")
	writer := b.PacketConn()
	dest := AddrForPublicKey(a.PublicKey())
	if dest == nil {
		t.Fatal("AddrForPublicKey вернул nil для валидного ключа")
	}

	// Первая запись лишь заводит сессию, поэтому повторяем до получения.
	deadline := time.Now().Add(20 * time.Second)
	for attempts := 0; ; attempts++ {
		n, err := writer.WriteTo(payload, dest)
		if err != nil {
			t.Fatalf("WriteTo (попытка %d): %v", attempts, err)
		}
		if n != len(payload) {
			t.Fatalf("WriteTo принял %d байт из %d", n, len(payload))
		}
		select {
		case p := <-inbound:
			if !bytes.Equal(p.Payload, payload) {
				continue
			}
			if !bytes.Equal(p.From, b.PublicKey()) {
				t.Fatalf("пакет пришёл от %x, ожидался ключ B", p.From)
			}
			return
		case <-time.After(500 * time.Millisecond):
			if time.Now().After(deadline) {
				t.Fatalf("пакет не дошёл за 20s (%d попыток)", attempts+1)
			}
		}
	}
}

func TestNewYggNodeRejectsBadConfig(t *testing.T) {
	tests := []struct {
		name string
		opts NodeOptions
	}{
		{"ключ не ed25519", NodeOptions{PrivateKey: make([]byte, 16)}},
		{"несогласованная пара ключей", NodeOptions{PrivateKey: mismatchedKey(t)}},
		{"неизвестный транспорт", NodeOptions{PrivateKey: nodeKey(t), Listen: []string{"udp://127.0.0.1:1"}}},
		{"слушатель без порта", NodeOptions{PrivateKey: nodeKey(t), Listen: []string{"tcp://127.0.0.1"}}},
		// Слушатель без хоста легален: tcp://:1234 значит «слушать все
		// интерфейсы на 1234». А вот пир без хоста — ошибка.
		{"пир без хоста", NodeOptions{PrivateKey: nodeKey(t), Peers: []string{"tcp://:1234"}}},
	}
	for _, tc := range tests {
		if _, err := NewYggNode(tc.opts); err == nil {
			t.Errorf("%s: ожидается ошибка, получили работающую ноду", tc.name)
		}
	}
}

// mismatchedKey собирает приватный ключ, публичная часть которого не
// соответствует приватной — NewYggNode обязан такое отклонить.
func mismatchedKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	first := nodeKey(t)
	second := nodeKey(t)
	priv := make([]byte, ed25519.PrivateKeySize)
	copy(priv, first[:ed25519.SeedSize])
	copy(priv[ed25519.SeedSize:], second[:ed25519.PublicKeySize])
	return ed25519.PrivateKey(priv)
}

// TestUnstartedNodeHasNoRoutes — изолированная нода честна: ни пиров, ни
// маршрутов, ни «доставлено».
func TestUnstartedNodeHasNoRoutes(t *testing.T) {
	n, err := NewYggNode(NodeOptions{PrivateKey: nodeKey(t)})
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = n.Stop() }()

	other := nodeKey(t)
	if n.HasRouteTo(other.Public().(ed25519.PublicKey)) {
		t.Fatal("маршрута быть не может: нода ни с кем не спирится")
	}
	if len(n.Peers()) != 0 {
		t.Fatalf("пиры появились из ниоткуда: %v", n.Peers())
	}
	if n.HasRouteTo(make([]byte, 8)) {
		t.Fatal("невалидный ключ не должен давать маршрут")
	}
}

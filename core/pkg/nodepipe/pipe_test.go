// File: core/pkg/nodepipe/pipe_test.go
package nodepipe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	yggproto "ygggram/core/proto"
)

// stub — ядро-марионетка: отвечает на каждый запрос его же request_id и
// считает вызовы. Настоящий rpc.Engine здесь не нужен: рамка проверяется
// рамкой, а не криптографией.
type stub struct {
	events chan *yggproto.CoreEvent
	calls  atomic.Int64
	// hold, если задан, заставляет ответ ждать: так проверяется, что событие
	// может прийти ровно в середине незакрытого вызова.
	hold time.Duration
}

func (s *stub) InvokeBytes(payload []byte) ([]byte, error) {
	req := &yggproto.CoreRequest{}
	if err := proto.Unmarshal(payload, req); err != nil {
		return nil, err
	}
	n := s.calls.Add(1)
	if s.hold > 0 {
		time.Sleep(s.hold)
	}
	return proto.Marshal(&yggproto.CoreResponse{
		RequestId: req.GetRequestId(),
		Success:   true,
		// Номер вызова позволяет проверить, что ответ вернулся именно тому,
		// кто его ждал, а не первому читателю.
		ErrorMessage: fmt.Sprintf("вызов %d", n),
	})
}

func (s *stub) Events() <-chan *yggproto.CoreEvent { return s.events }

// pipePair поднимает Serve над io.Pipe в отдельной горутине и возвращает
// клиента. Закрытие клиентской стороны приводит Serve к концу.
func pipePair(t *testing.T, core Core) (*Conn, context.CancelFunc) {
	t.Helper()

	clientReads, serverWrites := io.Pipe()
	serverReads, clientWrites := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, core, serverReads, serverWrites)
	}()
	t.Cleanup(func() {
		_ = clientWrites.Close()
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Serve завершился с ошибкой: %v", err)
		}
		// Serve не владеет этим потоком и не закрывает его: без закрытия
		// читатель клиента остался бы висящим горутинкой до конца прогона.
		_ = clientReads.Close()
	})

	return NewConn(clientReads, clientWrites), cancel
}

func ping(id uint64) *yggproto.CoreRequest {
	return &yggproto.CoreRequest{RequestId: id, Request: &yggproto.CoreRequest_GetNodeStatus{
		GetNodeStatus: &yggproto.GetNodeStatusReq{},
	}}
}

// TestServeRoundTripAndEvents — базовый смысл канала: запрос доходит, ответ
// возвращается тому же, кто ждёт, а событие приходит отдельно от ответа.
func TestServeRoundTripAndEvents(t *testing.T) {
	core := &stub{events: make(chan *yggproto.CoreEvent, eventQueue), hold: 50 * time.Millisecond}
	conn, _ := pipePair(t, core)

	const concurrent = 8
	var wg sync.WaitGroup
	replies := make([]string, concurrent)
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Идентификатор проставляет сам Call, и заказывать его заранее
			// нельзя: горутины стартуют в свободном порядке. Проверять
			// приходится соответствие «ответ вернулся тому же объекту
			// запроса, который отправлял», а не конкретному числу.
			req := ping(0)
			resp, err := conn.Call(req, 10*time.Second)
			if err != nil {
				t.Errorf("вызов %d: %v", i+1, err)
				return
			}
			if resp.GetRequestId() != req.GetRequestId() {
				t.Errorf("вызов %d (request_id %d) получил ответ с request_id %d",
					i+1, req.GetRequestId(), resp.GetRequestId())
			}
			replies[i] = resp.GetErrorMessage()
		}(i)
	}

	// Событие шлётся, пока все вызовы ещё в воздухе: кадр ответа и кадр
	// события делят один поток, и рамка обязана удержать оба.
	core.events <- &yggproto.CoreEvent{EventId: 42, TimestampMs: 1}
	wg.Wait()

	// Каждый из восьми вызовов обязан получить свой ответ ровно один раз:
	// перепутанные или потерянные ответы дали бы бы «успех» не там, где его
	// ждали.
	seen := make(map[string]int, concurrent)
	for _, r := range replies {
		seen[r]++
	}
	for i := 0; i < concurrent; i++ {
		want := fmt.Sprintf("вызов %d", i+1)
		if seen[want] != 1 {
			t.Fatalf("ответ %s пришёл %d раз, ожидался ровно один: %v", want, seen[want], replies)
		}
	}
	if got := core.calls.Load(); got != concurrent {
		t.Fatalf("ядро приняло %d запросов, ожидалось %d", got, concurrent)
	}

	select {
	case ev := <-conn.Events():
		if ev.GetEventId() != 42 {
			t.Fatalf("пришло событие %d, ожидалось 42", ev.GetEventId())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("событие не дошло до клиента")
	}
}

// TestServeStopsWhenClientHangsUp — закрытый stdin клиента обязан завершить
// Serve без ошибки: это штатный способ остановить процесс ядра.
func TestServeStopsWhenClientHangsUp(t *testing.T) {
	clientReads, serverWrites := io.Pipe()
	serverReads, clientWrites := io.Pipe()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- Serve(context.Background(), &stub{events: make(chan *yggproto.CoreEvent)}, serverReads, serverWrites)
	}()

	if err := clientWrites.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Serve завершился с ошибкой: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve не завершился после закрытия входа")
	}
	_ = clientReads.Close()
	_ = serverWrites.Close()
}

// TestFrameGuards — рамка не должна верить заголовку на слово: неизвестный
// признак и невозможная длина считаются ошибкой разбора, а не «дочитать».
func TestFrameGuards(t *testing.T) {
	if err := writeFrame(io.Discard, KindResponse, make([]byte, maxFrame+1)); err == nil {
		t.Fatal("кадр сверх лимита записался молча")
	}

	// Признак 3 вне списка.
	_, _, err := readFrame(bufReader([]byte{3, 0, 0, 0, 1, 'x'}))
	if err == nil {
		t.Fatal("неизвестный признак кадра принят")
	}

	// Обещанный 1 МиБ, а в потоке один байт: это оборванный поток, а не пустое
	// событие — io.ErrUnexpectedEOF как раз про такое.
	if _, _, err := readFrame(bufReader([]byte{KindEvent, 0, 0, 0x10, 0, 'x'})); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("оборванный кадр: %v", err)
	}
}

func bufReader(b []byte) io.Reader { return &chunkedReader{data: b} }

// chunkedReader отдаёт по два байта: читатель рамки обязан собрать кадр и через
// резку потока, и через склейку — ровно то, что делает живая труба.
type chunkedReader struct {
	data []byte
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data[:min(len(r.data), 2)])
	r.data = r.data[n:]
	return n, nil
}

// TestConnDropsEventsInsteadOfBlocking — клиент, который события не читает, не
// имеет права остановить ядро: переполнение считается потерей.
func TestConnDropsEventsInsteadOfBlocking(t *testing.T) {
	core := &stub{events: make(chan *yggproto.CoreEvent, eventQueue)}
	conn, _ := pipePair(t, core)

	for i := 0; i < eventQueue*3; i++ {
		core.events <- &yggproto.CoreEvent{EventId: uint64(i)}
	}
	// Если бы читатель встал, Call ниже просто не вернулся бы.
	if _, err := conn.Call(ping(1), 10*time.Second); err != nil {
		t.Fatalf("канал встал на unread-событиях: %v", err)
	}
}

// File: core/pkg/nodepipe/conn.go
package nodepipe

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	yggproto "ygggram/core/proto"
)

// Conn — клиентская сторона трубы: один собеседник на другом конце, который
// обязан отвечать на запросы и может сам присылать события.
//
// Читатель на канал один: он раскладывает кадры либо в ожидающий ответ с таким
// же request_id, либо в канал событий. Иначе ответ можно было бы принять за
// событие, а событие — выбросить как «не тот идентификатор».
type Conn struct {
	in  io.Reader
	out io.Writer
	// wrMu держит записи запросов: два вызова из разных горутин не должны
	// перемешать свои кадры.
	wrMu sync.Mutex

	mu      sync.Mutex
	pending map[uint64]chan []byte
	nextID  uint64

	events chan *yggproto.CoreEvent
	closed chan struct{}
	once   sync.Once

	rerrMu sync.Mutex
	rerr   error
}

// eventQueue — сколько событий влезает, пока клиент их не прочитал. Столько же
// держит ядро, так что узкое место не в трубе.
const eventQueue = 64

// NewConn поднимает клиента над парой потоков и запускает читателя.
func NewConn(in io.Reader, out io.Writer) *Conn {
	c := &Conn{
		in:      in,
		out:     out,
		pending: make(map[uint64]chan []byte),
		events:  make(chan *yggproto.CoreEvent, eventQueue),
		closed:  make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// Events — канал событий ядра.
func (c *Conn) Events() <-chan *yggproto.CoreEvent { return c.events }

// Done закрывается, когда читатель остановлен: ядро закрыло трубу, кадр
// оказался мусор или клиент закрылся сам. Нужен родителю процесса: os/exec.Wait
// закрывает трубы, поэтому звать его можно только когда читатель уже отключился,
// иначе гонка съедает последний кадр.
func (c *Conn) Done() <-chan struct{} { return c.closed }

// Close закрывает клиента: читатель останавливается, а ожидающие ответа вызовы
// просыпаются с ошибкой. Возвращает причину, по которой канал умер, если она
// была.
func (c *Conn) Close() error {
	c.fail(nil)
	return c.ReadErr()
}

// ReadErr — ошибка читателя (io.EOF, когда ядро просто закрыло ответную трубу).
func (c *Conn) ReadErr() error {
	c.rerrMu.Lock()
	defer c.rerrMu.Unlock()
	return c.rerr
}

// fail помечает канал мёртвым. err == nil означает «закрыли сами».
func (c *Conn) fail(err error) {
	if err != nil {
		c.rerrMu.Lock()
		if c.rerr == nil {
			c.rerr = err
		}
		c.rerrMu.Unlock()
	}
	c.once.Do(func() { close(c.closed) })

	// Все, кто ждёт ответа, обязаны проснуться: висящий вызов хуже ошибки.
	c.mu.Lock()
	for id, ch := range c.pending {
		delete(c.pending, id)
		close(ch)
	}
	c.mu.Unlock()
}

func (c *Conn) readLoop() {
	for {
		kind, body, err := readFrame(c.in)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				c.fail(io.EOF)
			} else {
				c.fail(err)
			}
			return
		}
		switch kind {
		case KindResponse:
			resp := &yggproto.CoreResponse{}
			if err := proto.Unmarshal(body, resp); err != nil {
				c.fail(fmt.Errorf("nodepipe: ответ не разбирается: %w", err))
				return
			}
			c.deliver(resp.GetRequestId(), body)
		case KindEvent:
			ev := &yggproto.CoreEvent{}
			if err := proto.Unmarshal(body, ev); err != nil {
				c.fail(fmt.Errorf("nodepipe: событие не разбирается: %w", err))
				return
			}
			select {
			case c.events <- ev:
			case <-c.closed:
				return
			default:
				// Клиент события не читает — считаем потерянным и идём
				// дальше: вставать ради одного события канал не вправе.
			}
		default:
			c.fail(fmt.Errorf("nodepipe: от ядра пришёл кадр признака %d", kind))
			return
		}
	}
}

// deliver отдаёт тело ответа тому, кто ждёт этот request_id. Незнакомый
// идентификатор значит, что клиент такой запрос не слал (или отказался ждать),
// и кадр просто выбрасывается.
func (c *Conn) deliver(id uint64, body []byte) {
	c.mu.Lock()
	ch, ok := c.pending[id]
	if ok {
		delete(c.pending, id)
	}
	c.mu.Unlock()
	if !ok {
		return
	}
	ch <- body
	close(ch)
}

// Call шлёт запрос и ждёт ответ ровно с этим request_id. Идентификатор
// проставляется здесь, чтобы вызывающий не считал их руками: по нему же
// читатель и находит, кому вернуть ответ.
func (c *Conn) Call(req *yggproto.CoreRequest, timeout time.Duration) (*yggproto.CoreResponse, error) {
	if err := c.ReadErr(); err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.nextID++
	id := c.nextID
	req.RequestId = id
	ch := make(chan []byte, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	payload, err := proto.Marshal(req)
	if err != nil {
		c.forgetRequest(id)
		return nil, fmt.Errorf("nodepipe: сборка запроса: %w", err)
	}

	c.wrMu.Lock()
	werr := writeFrame(c.out, KindRequest, payload)
	c.wrMu.Unlock()
	if werr != nil {
		c.forgetRequest(id)
		return nil, werr
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case body, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("nodepipe: канал закрылся до ответа: %v", c.ReadErr())
		}
		resp := &yggproto.CoreResponse{}
		if err := proto.Unmarshal(body, resp); err != nil {
			return nil, fmt.Errorf("nodepipe: разбор ответа: %w", err)
		}
		return resp, nil
	case <-timer.C:
		c.forgetRequest(id)
		return nil, fmt.Errorf("nodepipe: ответ на запрос %d не пришёл за %s", id, timeout)
	case <-c.closed:
		// Канал умер, а ожидавший вызов уже снят с pending горютой fail;
		// перепроверяем, не успел ли ответ прийти раньше закрытия.
		select {
		case body, ok := <-ch:
			if ok {
				resp := &yggproto.CoreResponse{}
				if err := proto.Unmarshal(body, resp); err != nil {
					return nil, fmt.Errorf("nodepipe: разбор ответа: %w", err)
				}
				return resp, nil
			}
		default:
		}
		return nil, c.ReadErr()
	}
}

func (c *Conn) forgetRequest(id uint64) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

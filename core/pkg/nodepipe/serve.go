// Package nodepipe держит рамочный протокол «ядро в своём процессе».
//
// Смысл ровно один: rpc.Engine до этого умел отвечать только вызывающему —
// либо cmd/libygggram через C-указатель, либо тест, который держал движок в
// своей памяти. «Два клиента» при этом означало один процесс с двумя движками,
// а не два клиента. Здесь — транспорт, который позволяет запустить ядро
// отдельным процессом и говорить с ним protobuf-кадрами по трубкам.
//
// Кадр: 1 байт признака + 4 байта длины в сетевом порядке + protobuf.
// Признаки: запрос (0) идёт от клиента к ядру, ответ (1) и событие (2) —
// обратно. Рамка нужна потому, что поток байтов границ записей не хранит, а
// CoreEvent может прийти ровно в тот момент, когда клиент ждёт ответ.
package nodepipe

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"

	"google.golang.org/protobuf/proto"

	yggproto "ygggram/core/proto"
)

// Признаки кадра. Три, потому что больше направлений у канала нет: клиент
// умеет только просить, ядро умеет отвечать и сообщать само.
const (
	KindRequest  byte = 0
	KindResponse byte = 1
	KindEvent    byte = 2
)

// maxFrame — потолок одного кадра. Кадр с чужой длиной иначе заставит выделить
// гигабайты; 8 МиБ с запасом покрывает конверт с вложением, а всё, что больше,
// — уже не протокол, а испорченный поток.
const maxFrame = 8 << 20

// headerSize — байт признака плюс четыре байта длины.
const headerSize = 5

// Core — минимум, который нужен трубе от ядра. Интерфейс, а не *rpc.Engine,
// потому что rpc тянет сгенерированный protobuf вместе с этим пакетом: прямой
// импорт собрал бы цикл.
type Core interface {
	InvokeBytes(payload []byte) ([]byte, error)
	Events() <-chan *yggproto.CoreEvent
}

// Serve крутит ядро над парой потоков: читает запросы, пишет ответы и события.
// Возвращается, когда вход закрыт (клиент ушёл), поток сломался или ctx
// отменён. Отдельной «стоп-команды» в протоколе нет намеренно: закрыть stdin
// — и есть остановка, и этот сигнал не потеряется.
func Serve(ctx context.Context, core Core, in io.Reader, out io.Writer) error {
	if core == nil {
		return errors.New("nodepipe: ядро не передано")
	}

	var wrMu sync.Mutex
	write := func(kind byte, payload []byte) error {
		wrMu.Lock()
		defer wrMu.Unlock()
		return writeFrame(out, kind, payload)
	}

	// Насос событий живёт, пока живёт Serve: отменяем его до возврата, чтобы
	// горутина не писала в поток, который вызывающий уже закрыл.
	pumpCtx, stopPump := context.WithCancel(ctx)
	defer stopPump()

	go func() {
		for {
			select {
			case <-pumpCtx.Done():
				return
			case ev := <-core.Events():
				if ev == nil {
					continue
				}
				payload, err := proto.Marshal(ev)
				if err != nil {
					continue
				}
				// Ошибка записи здесь не авария: читатель мог уйти в любой
				// момент, а приёмный цикл ядро вставать не должен.
				if err := write(KindEvent, payload); err != nil {
					return
				}
			}
		}
	}()

	for {
		kind, body, err := readFrame(in)
		if err != nil {
			// Клиент закрыл трубу — это штатный конец жизни процесса-ядра.
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if kind != KindRequest {
			return fmt.Errorf("nodepipe: в ядро пришёл кадр признака %d, а не запрос", kind)
		}
		resp, err := core.InvokeBytes(body)
		if err != nil {
			// Ядро не смогло разобрать байты: ответа нет, и клиенту об этом
			// сообщается закрытием канала, а не пустым кадром.
			return fmt.Errorf("nodepipe: ядро не приняло запрос: %w", err)
		}
		if err := write(KindResponse, resp); err != nil {
			return err
		}
	}
}

// writeFrame кладёт кадр целиком: длина обязана сойтись, иначе читатель
// съест начало следующего кадра как хвост этого.
func writeFrame(w io.Writer, kind byte, payload []byte) error {
	if len(payload) > maxFrame {
		return fmt.Errorf("nodepipe: кадр в %d байт не влезает в лимит %d", len(payload), maxFrame)
	}
	head := make([]byte, headerSize)
	head[0] = kind
	binary.BigEndian.PutUint32(head[1:], uint32(len(payload)))
	buf := append(head, payload...)
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("nodepipe: запись кадра: %w", err)
	}
	return nil
}

// readExactly дочитывает ровно n байт, не залезая в следующий кадр.
func readExactly(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// readFrame снимает один кадр и возвращает его признак и тело.
func readFrame(r io.Reader) (byte, []byte, error) {
	head, err := readExactly(r, headerSize)
	if err != nil {
		return 0, nil, err
	}
	kind := head[0]
	if kind > KindEvent {
		return 0, nil, fmt.Errorf("nodepipe: неизвестный признак кадра %d", kind)
	}
	size := binary.BigEndian.Uint32(head[1:headerSize])
	if size > maxFrame {
		return 0, nil, fmt.Errorf("nodepipe: заявлен кадр в %d байт, лимит %d", size, maxFrame)
	}
	body, err := readExactly(r, int(size))
	if err != nil {
		return 0, nil, err
	}
	return kind, body, nil
}

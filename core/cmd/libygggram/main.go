// File: core/cmd/libygggram/main.go
//
// C-ABI обёртка над rpc.Engine. Единственный формат на границе — protobuf
// (yggproto.CoreRequest / CoreResponse / CoreEvent): раньше здесь парсился
// самописный BigEndian-заголовок, а StartNode/SendMessage меняли только
// локальные счётчики, из-за чего криптография и хранилище ядра были
// недостижимы из приложения.
package main

/*
#include <stdlib.h>
#include <stdint.h>

// Typedef for asynchronous event callback into Dart NativeCallable.listener
typedef void (*EventCallback)(const uint8_t* event_data, int length);

static inline void invoke_event_callback(EventCallback cb, const uint8_t* data, int length) {
    if (cb != NULL) {
        cb(data, length);
    }
}
*/
import "C"

import (
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"google.golang.org/protobuf/proto"

	"ygggram/core/pkg/rpc"
	yggproto "ygggram/core/proto"
)

type coreService struct {
	mu            sync.RWMutex
	engine        *rpc.Engine
	eventCallback C.EventCallback
	stopChan      chan struct{}
	eventSeq      uint64
	loopStarted   bool
}

var (
	coreInstance *coreService
	coreOnce     sync.Once
)

func getCore() *coreService {
	coreOnce.Do(func() {
		coreInstance = &coreService{
			engine:   rpc.New(),
			stopChan: make(chan struct{}),
		}
	})
	return coreInstance
}

//export InitCore
func InitCore(configBytes *C.uchar, length C.int) C.int {
	if length <= 0 || configBytes == nil {
		return -1
	}
	cfg := &yggproto.NodeConfig{}
	if err := proto.Unmarshal(C.GoBytes(unsafe.Pointer(configBytes), length), cfg); err != nil {
		return -2
	}

	// Директории создаём сразу: Dart-сторона видит конкретный код ошибки,
	// а ядру не нужно выдумывать пути по умолчанию.
	for _, dir := range []string{cfg.GetDataDir(), cfg.GetCacheDir()} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return -3
		}
	}

	core := getCore()
	core.mu.Lock()
	if !core.loopStarted {
		core.loopStarted = true
		go core.eventLoop()
	}
	core.mu.Unlock()
	return 0
}

//export RegisterEventCallback
func RegisterEventCallback(cb C.EventCallback) {
	core := getCore()
	core.mu.Lock()
	core.eventCallback = cb
	core.mu.Unlock()
}

//export InvokeCommand
func InvokeCommand(reqBytes *C.uchar, length C.int, respPtr **C.uchar, respLen *C.int) C.int {
	if length <= 0 || reqBytes == nil || respPtr == nil || respLen == nil {
		return -1
	}
	payload := C.GoBytes(unsafe.Pointer(reqBytes), length)

	core := getCore()
	response, err := core.engine.InvokeBytes(payload)
	if err != nil {
		// Ошибка разборки возвращается тем же CoreResponse: у Dart-стороны
		// один путь обработки ответов.
		response, err = proto.Marshal(&yggproto.CoreResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		})
		if err != nil {
			return -2
		}
	}

	cResponse := C.CBytes(response)
	*respPtr = (*C.uchar)(cResponse)
	*respLen = C.int(len(response))
	return 0
}

//export FreeMemory
func FreeMemory(ptr *C.uchar) {
	if ptr != nil {
		C.free(unsafe.Pointer(ptr))
	}
}

// eventLoop пульсирует состоянием реальной ноды и доносит до UI события
// приёмного цикла ядра. События идут из engine.Events(): фиктивных peer'ов
// здесь больше нет, и если пиров нет — в событии приходит пустой список.
func (c *coreService) eventLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	incoming := c.engine.Events()
	for {
		select {
		case <-c.stopChan:
			return
		case ev := <-incoming:
			c.dispatch(marshalEvent(ev))
		case <-ticker.C:
			c.dispatch(c.heartbeat())
		}
	}
}

func marshalEvent(ev *yggproto.CoreEvent) []byte {
	if ev == nil {
		return nil
	}
	out, err := proto.Marshal(ev)
	if err != nil {
		return nil
	}
	return out
}

func (c *coreService) dispatch(payload []byte) {
	c.mu.RLock()
	cb := c.eventCallback
	c.mu.RUnlock()
	if cb == nil || len(payload) == 0 {
		return
	}
	C.invoke_event_callback(cb, (*C.uint8_t)(unsafe.Pointer(&payload[0])), C.int(len(payload)))
}

func (c *coreService) heartbeat() []byte {
	status := c.engine.Handle(&yggproto.CoreRequest{
		Request: &yggproto.CoreRequest_GetNodeStatus{GetNodeStatus: &yggproto.GetNodeStatusReq{}},
	}).GetNodeStatus()
	peers := c.engine.Handle(&yggproto.CoreRequest{
		Request: &yggproto.CoreRequest_GetPeers{GetPeers: &yggproto.GetPeersReq{}},
	}).GetPeerList()

	note := "ядро не запущено"
	if status.GetIsRunning() {
		note = ""
	}
	// Число активных пиров берётся у самой ноды: GetPeers отдаёт и CONNECTING, и
	// уже упавшие соединения, а UI в NetworkStatusEvent ждёт именно поднятые
	// линки. Байты суммируются по всем записям — молча выбросить счётчик
	// прошедшего трафика значило бы соврать в меньшую сторону.
	var rx, tx uint64
	for _, p := range peers.GetPeers() {
		rx += p.GetBytesReceived()
		tx += p.GetBytesSent()
	}
	event := &yggproto.CoreEvent{
		EventId:     atomic.AddUint64(&c.eventSeq, 1),
		TimestampMs: uint64(time.Now().UnixMilli()),
		Event: &yggproto.CoreEvent_NetworkStatus{NetworkStatus: &yggproto.NetworkStatusEvent{
			Online:       status.GetIsRunning(),
			CurrentIpv6:  status.GetMyIpv6(),
			ActivePeers:  status.GetConnectedPeersCount(),
			TotalRxBytes: rx,
			TotalTxBytes: tx,
			StatusNote:   note,
		}},
	}
	return marshalEvent(event)
}

func main() {
	// Required for -buildmode=c-shared compilation targets
}

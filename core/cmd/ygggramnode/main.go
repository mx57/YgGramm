// File: core/cmd/ygggramnode/main.go
//
// ygggramnode — ядро отдельным процессом. До этой команды движок был доступен
// только внутри процесса: cmd/libygggram требует C-указатель на функцию из UI,
// а тесты держали оба движка в одной памяти. Здесь — тот же rpc.Engine, только
// говорящий protobuf-кадрами по stdin/stdout (см. pkg/nodepipe), что и позволяет
// запустить двух настоящих клиентов одновременно.
//
//	./ygggramnode            # слушает stdin, пишет ответы и события в stdout
//
// Флагов нет намеренно: вся конфигурация (data_dir, мнемоника, пиры, режим
// anti-DPI) приходит внутри StartNode, поэтому один и тот же бинарник годится
// и для первой, и для второй ноды, а различаются они только запросом.
//
// Диагностику процесс пишет в stderr: stdout занят кадрами, и примешивать к
// ним журнал означало бы ломать рамку.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ygggram/core/pkg/nodepipe"
	"ygggram/core/pkg/rpc"
)

func main() {
	log.SetOutput(os.Stderr)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := nodepipe.Serve(ctx, rpc.New(), os.Stdin, os.Stdout); err != nil {
		log.Printf("ygggramnode: %v", err)
		os.Exit(1)
	}
}

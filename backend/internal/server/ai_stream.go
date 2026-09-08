package server

import (
	"context"
	"time"

	"curated-backend/internal/contracts"
)

// streamAIEvents owns all response writes on one goroutine. Heartbeats keep
// downstream idle timers alive; the model has a separate generation deadline.
func streamAIEvents(ctx context.Context, interval time.Duration, run func(context.Context, func(contracts.AIChatSSEEvent)) error, send func(contracts.AIChatSSEEvent) error, heartbeat func() error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	events := make(chan contracts.AIChatSSEEvent)
	done := make(chan error, 1)
	// 独立运行生产者；上下文取消后结束，所有响应仍由接收方串行写出。
	go func() {
		// 把事件交给唯一响应写入者；断开连接后停止等待发送。
		done <- run(ctx, func(event contracts.AIChatSSEEvent) {
			select {
			case events <- event:
			case <-ctx.Done():
			}
		})
	}()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		case event := <-events:
			if err := send(event); err != nil {
				return err
			}
		case <-ticker.C:
			if err := heartbeat(); err != nil {
				return err
			}
		}
	}
}

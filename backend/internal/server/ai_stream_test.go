package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"curated-backend/internal/contracts"
)

// TestAIStreamHeartbeatAndSerializedCompletion 验证 AIStream Heartbeat And Serialized Completion 的行为与失败边界，使用隔离测试数据。
func TestAIStreamHeartbeatAndSerializedCompletion(t *testing.T) {
	ready := make(chan struct{})
	heartbeats := 0
	var kinds []string
	// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
	err := streamAIEvents(context.Background(), time.Millisecond, func(ctx context.Context, emit func(contracts.AIChatSSEEvent)) error {
		emit(contracts.AIChatSSEEvent{Type: "message_start"})
		select {
		case <-ready:
		case <-ctx.Done():
			return ctx.Err()
		}
		emit(contracts.AIChatSSEEvent{Type: "message_done"})
		return nil
		// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
	}, func(event contracts.AIChatSSEEvent) error { kinds = append(kinds, event.Type); return nil }, func() error {
		heartbeats++
		if heartbeats == 1 {
			close(ready)
		}
		return nil
	})
	if err != nil || heartbeats < 1 || len(kinds) != 2 || kinds[1] != "message_done" {
		t.Fatal(err, heartbeats, kinds)
	}
}

// TestAIStreamDisconnectCancelsProducer 验证 AIStream Disconnect Cancels Producer 的行为与失败边界，使用隔离测试数据。
func TestAIStreamDisconnectCancelsProducer(t *testing.T) {
	finished := make(chan struct{})
	broken := errors.New("closed response")
	// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
	err := streamAIEvents(context.Background(), time.Millisecond, func(ctx context.Context, _ func(contracts.AIChatSSEEvent)) error {
		defer close(finished)
		<-ctx.Done()
		return ctx.Err()
		// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
	}, func(contracts.AIChatSSEEvent) error { return nil }, func() error { return broken })
	if !errors.Is(err, broken) {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("producer not cancelled")
	}
}

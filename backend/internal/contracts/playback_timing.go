package contracts

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type playbackTimingKey struct{}
type PlaybackTiming struct {
	mu     sync.Mutex
	values []string
}

func WithPlaybackTiming(ctx context.Context) (context.Context, *PlaybackTiming) {
	t := &PlaybackTiming{}
	return context.WithValue(ctx, playbackTimingKey{}, t), t
}

// RecordPlaybackTiming accepts only static stage names from the playback code.
func RecordPlaybackTiming(ctx context.Context, stage string, started time.Time) {
	if t, ok := ctx.Value(playbackTimingKey{}).(*PlaybackTiming); ok {
		t.mu.Lock()
		defer t.mu.Unlock()
		if len(t.values) < 32 {
			t.values = append(t.values, fmt.Sprintf("%s;dur=%.2f", stage, float64(time.Since(started).Microseconds())/1000))
		}
	}
}

func (t *PlaybackTiming) Header() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.Join(t.values, ", ")
}

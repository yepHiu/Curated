package playback

import "sync"

// FFmpeg writes concurrently with startup diagnostics; keep only a bounded tail.
type lockedTailBuffer struct {
	mu   sync.Mutex
	data []byte
}

func (b *lockedTailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	const limit = 64 * 1024
	b.data = append(b.data, p...)
	if len(b.data) > limit {
		b.data = append([]byte(nil), b.data[len(b.data)-limit:]...)
	}
	return len(p), nil
}

func (b *lockedTailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}

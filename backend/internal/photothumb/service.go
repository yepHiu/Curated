// Package photothumb provides bounded, derived previews without storing photo data in comic caches.
package photothumb

import (
	"bytes"
	"container/list"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"sync"

	"curated-backend/internal/photoarchive"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const MaxSide = 420
const maxSourceBytes = 32 << 20
const maxPixels = 24_000_000
const cacheBudget = 32 << 20

type entry struct {
	key  string
	body []byte
}
type Service struct {
	mu      sync.Mutex
	entries map[string]*list.Element
	recent  *list.List
	bytes   int
	gate    chan struct{}
}

func New() *Service {
	return &Service{entries: make(map[string]*list.Element), recent: list.New(), gate: make(chan struct{}, 1)}
}

// Get validates the archive revision before cache lookup. One decode at a time bounds peak memory.
func (s *Service) Get(ctx context.Context, archivePath, entryPath string) ([]byte, string, error) {
	info, err := os.Stat(archivePath)
	if err != nil {
		return nil, "", err
	}
	key := fmt.Sprintf("%s\x00%s\x00%d\x00%d", archivePath, entryPath, info.Size(), info.ModTime().UnixNano())
	hash := sha256.Sum256([]byte(key))
	etag := fmt.Sprintf("\"photo-thumb-v1-%x\"", hash[:12])
	// A cached thumbnail need not wait behind an unrelated expensive decode.
	s.mu.Lock()
	if e := s.entries[key]; e != nil {
		s.recent.MoveToFront(e)
		body := e.Value.(entry).body
		s.mu.Unlock()
		return body, etag, nil
	}
	s.mu.Unlock()
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return nil, "", ctx.Err()
	}
	s.mu.Lock()
	if e := s.entries[key]; e != nil {
		s.recent.MoveToFront(e)
		body := e.Value.(entry).body
		s.mu.Unlock()
		return body, etag, nil
	}
	s.mu.Unlock()
	source, _, err := photoarchive.OpenPage(ctx, archivePath, entryPath)
	if err != nil {
		return nil, "", err
	}
	data, err := io.ReadAll(io.LimitReader(source, maxSourceBytes+1))
	_ = source.Close()
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxSourceBytes {
		return nil, "", fmt.Errorf("photo preview source exceeds limit")
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	body, err := encode(data)
	if err != nil {
		return nil, "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for s.bytes+len(body) > cacheBudget && s.recent.Len() > 0 {
		old := s.recent.Back()
		value := old.Value.(entry)
		delete(s.entries, value.key)
		s.bytes -= len(value.body)
		s.recent.Remove(old)
	}
	s.entries[key] = s.recent.PushFront(entry{key, body})
	s.bytes += len(body)
	return body, etag, nil
}

func encode(data []byte) ([]byte, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxPixels {
		return nil, fmt.Errorf("photo preview dimensions exceed limit")
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	width, height := config.Width, config.Height
	if width > MaxSide || height > MaxSide {
		if width >= height {
			height = max(1, height*MaxSide/width)
			width = MaxSide
		} else {
			width = max(1, width*MaxSide/height)
			height = MaxSide
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	var output bytes.Buffer
	if err := jpeg.Encode(&output, dst, &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

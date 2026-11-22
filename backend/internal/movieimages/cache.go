// Package movieimages maintains bounded, regenerable browsing images.
package movieimages

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

var Widths = [...]int{320, 640, 960}

const maxSourceBytes = 32 << 20
const maxPixels = 24_000_000
const defaultBudget = 1 << 30

type job struct {
	key  string
	open func(context.Context) (*os.File, error)
}
type Cache struct {
	root    string
	ctx     context.Context
	mu      sync.Mutex
	queue   []job
	pending map[string]bool
	failed  map[string]time.Time
	workers int
	wg      sync.WaitGroup
	pruneMu sync.Mutex
	budget  int64
}

func New(ctx context.Context, root string) *Cache {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Cache{root: root, ctx: ctx, pending: make(map[string]bool), failed: make(map[string]time.Time), budget: defaultBudget}
}

func ValidWidth(width int) bool { return width == 320 || width == 640 || width == 960 }
func (c *Cache) path(key string, width int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("movie-image-v1\x00%s\x00%d", key, width)))
	return filepath.Join(c.root, fmt.Sprintf("%x.img", sum))
}

// Ready only advertises completely written files. It never waits for decoding.
func (c *Cache) Ready(key string, width int) (string, bool) {
	if c.root == "" || !ValidWidth(width) {
		return "", false
	}
	path := c.path(key, width)
	st, err := os.Stat(path)
	return path, err == nil && st.Mode().IsRegular() && st.Size() > 0
}

// Ensure queues work without blocking the caller. At most 64 jobs and two
// decoders exist; workers exit when the queue drains. Source openers revalidate
// current asset policy and revision before returning a file.
func (c *Cache) Ensure(key string, open func(context.Context) (*os.File, error)) {
	if c.root == "" || c.ctx.Err() != nil {
		return
	}
	complete := true
	for _, width := range Widths {
		if _, ok := c.Ready(key, width); !ok {
			complete = false
			break
		}
	}
	if complete {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending[key] || len(c.pending) >= 64 || time.Since(c.failed[key]) < time.Minute {
		return
	}
	c.pending[key] = true
	c.queue = append(c.queue, job{key, open})
	c.wg.Add(1)
	if c.workers < 2 {
		c.workers++
		go c.work()
	}
}

func (c *Cache) work() {
	for {
		c.mu.Lock()
		if len(c.queue) == 0 {
			c.workers--
			c.mu.Unlock()
			return
		}
		task := c.queue[0]
		c.queue[0] = job{}
		c.queue = c.queue[1:]
		c.mu.Unlock()
		ctx, cancel := context.WithTimeout(c.ctx, 20*time.Second)
		err := c.generate(ctx, task)
		cancel()
		c.mu.Lock()
		delete(c.pending, task.key)
		if err != nil {
			if len(c.failed) >= 128 {
				clear(c.failed)
			}
			c.failed[task.key] = time.Now()
		} else {
			delete(c.failed, task.key)
		}
		c.mu.Unlock()
		if err == nil {
			c.prune()
		}
		c.wg.Done()
	}
}

func (c *Cache) generate(ctx context.Context, task job) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	source, err := task.open(ctx)
	if err != nil {
		return err
	}
	before, err := source.Stat()
	if err != nil {
		_ = source.Close()
		return err
	}
	data, err := io.ReadAll(io.LimitReader(source, maxSourceBytes+1))
	after, statErr := source.Stat()
	_ = source.Close()
	if err != nil {
		return err
	}
	if statErr != nil {
		return statErr
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return fmt.Errorf("image changed during read")
	}
	if len(data) > maxSourceBytes {
		return fmt.Errorf("image exceeds byte limit")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return fmt.Errorf("image exceeds pixel limit")
	}
	src, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.root, 0700); err != nil {
		return err
	}
	for _, width := range Widths {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, ok := c.Ready(task.key, width); ok {
			continue
		}
		target := src.Bounds().Dx()
		if target > width {
			target = width
		}
		dst := imaging.Fit(src, target, width*2, imaging.Lanczos)
		var encoded bytes.Buffer
		if dst.Opaque() {
			err = jpeg.Encode(&encoded, dst, &jpeg.Options{Quality: 80})
		} else {
			err = png.Encode(&encoded, dst)
		}
		if err != nil {
			return err
		}
		tmp, err := os.CreateTemp(c.root, ".building-*")
		if err != nil {
			return err
		}
		_, writeErr := tmp.Write(encoded.Bytes())
		closeErr := tmp.Close()
		if writeErr == nil {
			writeErr = closeErr
		}
		if writeErr == nil {
			writeErr = os.Rename(tmp.Name(), c.path(task.key, width))
		}
		if writeErr != nil {
			_ = os.Remove(tmp.Name())
			return writeErr
		}
	}
	return nil
}

// Touch makes eviction favor images actually downloaded, not just listed.
func (c *Cache) Touch(path string) {
	if filepath.Dir(path) != filepath.Clean(c.root) {
		return
	}
	now := time.Now()
	_ = os.Chtimes(path, now, now)
}

func (c *Cache) prune() {
	c.pruneMu.Lock()
	defer c.pruneMu.Unlock()
	entries, err := os.ReadDir(c.root)
	if err != nil {
		return
	}
	type candidate struct {
		name     string
		size     int64
		modified time.Time
	}
	var files []candidate
	var size int64
	for _, e := range entries {
		if e.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(e.Name(), ".img") {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		files = append(files, candidate{e.Name(), info.Size(), info.ModTime()})
		size += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modified.Before(files[j].modified) })
	for _, f := range files {
		if size <= c.budget {
			break
		}
		if os.Remove(filepath.Join(c.root, f.name)) == nil {
			size -= f.size
		}
	}
}

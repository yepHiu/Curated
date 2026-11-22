package movieimages

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDerivedImagesBoundSizeDeduplicateAndInvalidate(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 1200, 1800))
	for y := 0; y < 1800; y++ {
		for x := 0; x < 1200; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	f, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	f.Close()
	c := New(context.Background(), filepath.Join(root, "cache"))
	var opens atomic.Int32
	opener := func(context.Context) (*os.File, error) { opens.Add(1); return os.Open(source) }
	var callers sync.WaitGroup
	for i := 0; i < 20; i++ {
		callers.Add(1)
		go func() { defer callers.Done(); c.Ensure("revision-one", opener) }()
	}
	callers.Wait()
	c.wg.Wait()
	if opens.Load() != 1 {
		t.Fatalf("duplicate decodes: %d", opens.Load())
	}
	original, _ := os.Stat(source)
	for _, width := range Widths {
		path, ok := c.Ready("revision-one", width)
		if !ok {
			t.Fatalf("missing %d", width)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		cfg, _, err := image.DecodeConfig(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Width != width || cfg.Height != width*3/2 {
			t.Fatalf("wrong dimensions: %+v", cfg)
		}
		st, _ := os.Stat(path)
		if st.Size() >= original.Size() {
			t.Fatalf("not smaller: %d >= %d", st.Size(), original.Size())
		}
	}
	if _, ok := c.Ready("revision-two", 320); ok {
		t.Fatal("stale revision reused")
	}
	c.Ensure("revision-two", opener)
	c.wg.Wait()
	if opens.Load() != 2 {
		t.Fatal("changed revision not generated")
	}
	c.budget = 1
	c.prune()
	if _, ok := c.Ready("revision-two", 320); ok {
		t.Fatal("quota did not evict derived files")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("original was removed")
	}
}

func TestSmallTransparentImageKeepsAlphaAndDoesNotUpscale(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "alpha.png")
	img := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	f, _ := os.Create(source)
	png.Encode(f, img)
	f.Close()
	c := New(context.Background(), filepath.Join(root, "cache"))
	c.Ensure("a", func(context.Context) (*os.File, error) { return os.Open(source) })
	c.wg.Wait()
	path, ok := c.Ready("a", 320)
	if !ok {
		t.Fatal("not generated")
	}
	f, _ = os.Open(path)
	defer f.Close()
	got, format, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if format != "png" || got.Bounds().Dx() != 20 || got.Bounds().Dy() != 10 {
		t.Fatalf("unexpected %s %+v", format, got.Bounds())
	}
	_, _, _, alpha := got.At(0, 0).RGBA()
	if alpha == 0 || alpha == 65535 {
		t.Fatal("transparency lost")
	}
}

func TestMalformedSourceBacksOffAndCancelledCacheDoesNotOpen(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "broken")
	os.WriteFile(source, []byte("broken"), 0600)
	c := New(context.Background(), filepath.Join(root, "cache"))
	var opens atomic.Int32
	opener := func(context.Context) (*os.File, error) { opens.Add(1); return os.Open(source) }
	c.Ensure("bad", opener)
	c.wg.Wait()
	c.Ensure("bad", opener)
	c.wg.Wait()
	if opens.Load() != 1 {
		t.Fatal("failed source retried without backoff")
	}
	if _, ok := c.Ready("bad", 320); ok {
		t.Fatal("bad source advertised")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stopped := New(ctx, filepath.Join(root, "stop"))
	stopped.Ensure("bad", opener)
	stopped.wg.Wait()
	if opens.Load() != 1 {
		t.Fatal("cancelled cache opened source")
	}
}

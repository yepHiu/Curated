package wishlistassets

import (
	"bytes"
	"image"
	"image/png"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// TestSavePersistentImages 验证持久图片、缩略图与重复内容复用。
func TestSavePersistentImages(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 800, 400)))
	root := t.TempDir()
	f, e := Save(root, "test-id", b.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	again, e := Save(root, "test-id", b.Bytes())
	if e != nil || again != f {
		t.Fatal("not idempotent")
	}
	file, e := os.Open(filepath.Join(root, f.ThumbnailPath))
	if e != nil {
		t.Fatal(e)
	}
	defer file.Close()
	cfg, _, e := image.DecodeConfig(file)
	if e != nil || cfg.Width > 400 || cfg.Height > 600 {
		t.Fatalf("bad thumbnail %+v %v", cfg, e)
	}
	if _, e = Save(root, "../outside", b.Bytes()); e == nil {
		t.Fatal("path traversal")
	}
	if _, e = Save(root, "test-id", []byte("not an image")); e == nil {
		t.Fatal("bad image accepted")
	}
}

// TestPublicImageTargets 验证本机和内网地址不成为刮削图片目标。
func TestPublicImageTargets(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "::1", "10.1.2.3", "192.168.1.2", "100.100.1.2", "169.254.169.254"} {
		if publicIP(net.ParseIP(s)) {
			t.Fatalf("private %s allowed", s)
		}
	}
}

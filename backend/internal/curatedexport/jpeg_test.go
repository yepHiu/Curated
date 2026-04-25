package curatedexport

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func makeEncodeJPEGTestPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: uint8(10 * x), G: uint8(20 * y), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestEncodeImageToJPEGWithCuratedMeta_EmbedsExif(t *testing.T) {
	t.Parallel()

	meta := FrameMetaJSON{
		Title:         "Frame",
		Code:          "ABC-123",
		Actors:        []string{"Airi"},
		PositionSec:   12.5,
		CapturedAt:    "2026-04-26T09:00:00Z",
		FrameID:       "frame-1",
		MovieID:       "movie-1",
		Tags:          []string{"closeup"},
		SchemaVersion: 1,
		ExportedAt:    "2026-04-26T09:00:01Z",
		AppName:       "Curated",
		AppVersion:    "1.0.0",
	}

	out, err := EncodeImageToJPEGWithCuratedMeta(makeEncodeJPEGTestPNG(t), meta, 90)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("jpeg decode failed: %v", err)
	}
	if !bytes.Contains(out, []byte("Exif\x00\x00")) {
		t.Fatal("expected jpeg to contain EXIF header")
	}
	if !bytes.Contains(out, []byte(`"code":"ABC-123"`)) {
		t.Fatal("expected embedded metadata json in EXIF bytes")
	}
}

func TestExportJPGFilename_Collision(t *testing.T) {
	t.Parallel()

	used := make(map[string]struct{})
	a := ExportJPGFilename("Act", "X-1", 5, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", used)
	b := ExportJPGFilename("Act", "X-1", 5, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", used)
	if a == b {
		t.Fatalf("expected different names, got %q", a)
	}
	if !strings.HasSuffix(a, ".jpg") || !strings.HasSuffix(b, ".jpg") {
		t.Fatalf("expected .jpg suffix: %q %q", a, b)
	}
}

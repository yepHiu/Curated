package curatedexport

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"strings"
	"testing"
)

func TestInjectITxtChunk_embedsJSON(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	meta := FrameMetaJSON{
		Title: "T", Code: "ABC-123", Actors: []string{"A"},
		PositionSec: 1.5, CapturedAt: "2020-01-02T03:04:05Z",
		FrameID: "fid", MovieID: "mid",
	}
	out, err := EncodePNGWithCuratedMetaITxt(pngBuf.Bytes(), meta)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("decode after inject: %v", err)
	}
	if !bytes.Contains(out, []byte(`"code":"ABC-123"`)) {
		t.Fatal("expected json substring in png bytes")
	}
	if !containsChunkType(out, "iTXt") {
		t.Fatal("expected iTXt chunk")
	}
}

func containsChunkType(pngData []byte, wantType string) bool {
	if len(pngData) < 8 || string(pngData[:8]) != pngSignature {
		return false
	}
	pos := 8
	for pos+12 <= len(pngData) {
		length := int(binary.BigEndian.Uint32(pngData[pos : pos+4]))
		chunkType := string(pngData[pos+4 : pos+8])
		end := pos + 12 + length
		if end > len(pngData) {
			return false
		}
		if chunkType == wantType {
			return true
		}
		if chunkType == "IEND" {
			return false
		}
		pos = end
	}
	return false
}

func TestExportPNGFilename_collision(t *testing.T) {
	used := make(map[string]struct{})
	a := ExportPNGFilename("Act", "X-1", 5, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", used)
	b := ExportPNGFilename("Act", "X-1", 5, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", used)
	if a == b {
		t.Fatalf("expected different names, got %q", a)
	}
	if !strings.HasSuffix(a, ".png") || !strings.HasSuffix(b, ".png") {
		t.Fatalf("expected .png suffix: %q %q", a, b)
	}
}

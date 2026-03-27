package curatedexport

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"testing"

	"github.com/deepteams/webp/mux"
)

func TestEncodePNGToWebP_EMBEDEXIF(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	meta := FrameMetaJSON{
		Title: "T", Code: "ABC-123", Actors: []string{"A", "B"},
		PositionSec: 12.7, CapturedAt: "2020-01-02T03:04:05Z",
		FrameID: "fid", MovieID: "mid",
	}
	webpBytes, err := EncodePNGToWebP(pngBuf.Bytes(), meta, 80)
	if err != nil {
		t.Fatal(err)
	}
	d, err := mux.NewDemuxer(webpBytes)
	if err != nil {
		t.Fatal(err)
	}
	exifRaw, err := d.GetChunk(mux.FourCCEXIF)
	if err != nil || len(exifRaw) == 0 {
		t.Fatalf("exif chunk: %v len=%d", err, len(exifRaw))
	}
	if len(exifRaw) < 6 || string(exifRaw[0:6]) != "Exif\x00\x00" {
		t.Fatalf("exif header: %q", exifRaw[:min(10, len(exifRaw))])
	}
	// Demuxer returns full EXIF payload including Exif\0\0; find JSON after UserComment ASCII prefix in TIFF — simplest check: meta fields appear in raw bytes
	if !bytes.Contains(exifRaw, []byte(`"code":"ABC-123"`)) {
		t.Fatalf("expected json substring in exif, sample: %q", string(exifRaw[:min(200, len(exifRaw))]))
	}
}

func TestExportWebPFilename_collision(t *testing.T) {
	used := make(map[string]struct{})
	a := ExportWebPFilename("Act", "X-1", 5, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", used)
	b := ExportWebPFilename("Act", "X-1", 5, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", used)
	if a == b {
		t.Fatalf("expected different names, got %q", a)
	}
}

func TestBuildExifUserComment_roundTripJSON(t *testing.T) {
	want := []byte(`{"k":"v"}`)
	blob := BuildExifUserComment(want)
	if len(blob) < 6 || string(blob[0:6]) != "Exif\x00\x00" {
		t.Fatalf("header %q", blob[:6])
	}
	// Rough parse: JSON should appear after TIFF + ASCII prefix inside user comment data
	if !bytes.Contains(blob, want) {
		t.Fatalf("json not in blob")
	}
}

func TestFrameMetaJSON_roundTrip(t *testing.T) {
	m := FrameMetaJSON{Title: "あ", Code: "C", Actors: []string{"x"}, PositionSec: 1, CapturedAt: "t", FrameID: "i", MovieID: "m"}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var out FrameMetaJSON
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Title != m.Title {
		t.Fatal(out.Title)
	}
}

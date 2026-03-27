package curatedexport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"

	"github.com/deepteams/webp"
)

// FrameMetaJSON is embedded in EXIF UserComment as UTF-8 JSON.
type FrameMetaJSON struct {
	Title       string   `json:"title"`
	Code        string   `json:"code"`
	Actors      []string `json:"actors"`
	PositionSec float64  `json:"positionSec"`
	CapturedAt  string   `json:"capturedAt"`
	FrameID     string   `json:"frameId"`
	MovieID     string   `json:"movieId"`
}

// EncodePNGToWebP decodes PNG bytes, re-encodes as lossy WebP, and embeds EXIF UserComment JSON.
func EncodePNGToWebP(pngBytes []byte, meta FrameMetaJSON, quality float32) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("decode png: %w", err)
	}
	j, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal meta: %w", err)
	}
	exif := BuildExifUserComment(j)
	opts := &webp.EncoderOptions{
		Quality: quality,
		Method:  4,
		EXIF:    exif,
	}
	var out bytes.Buffer
	if err := webp.Encode(&out, img, opts); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return out.Bytes(), nil
}

// EncodePNGToWebPReader is like EncodePNGToWebP but reads PNG from r.
func EncodePNGToWebPReader(r io.Reader, meta FrameMetaJSON, quality float32) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return EncodePNGToWebP(data, meta, quality)
}

// DecodePNGConfig returns bounds without full decode (for validation).
func DecodePNGConfig(pngBytes []byte) (image.Config, error) {
	return png.DecodeConfig(bytes.NewReader(pngBytes))
}

package curatedthumb

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
)

const (
	MaxThumbnailWidth  = 320
	MaxThumbnailHeight = 180
)

func fitWithin(width, height, maxWidth, maxHeight int) (int, int) {
	if width <= 0 || height <= 0 {
		return 0, 0
	}
	if width <= maxWidth && height <= maxHeight {
		return width, height
	}
	ratioW := float64(maxWidth) / float64(width)
	ratioH := float64(maxHeight) / float64(height)
	ratio := ratioW
	if ratioH < ratio {
		ratio = ratioH
	}
	outW := int(float64(width) * ratio)
	outH := int(float64(height) * ratio)
	if outW < 1 {
		outW = 1
	}
	if outH < 1 {
		outH = 1
	}
	return outW, outH
}

func PNG(imageBytes []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, fmt.Errorf("decode curated frame image: %w", err)
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	outW, outH := fitWithin(width, height, MaxThumbnailWidth, MaxThumbnailHeight)
	if outW == 0 || outH == 0 {
		return nil, fmt.Errorf("invalid curated frame size %dx%d", width, height)
	}
	if outW == width && outH == height {
		return append([]byte(nil), imageBytes...), nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, fmt.Errorf("encode curated frame thumbnail: %w", err)
	}
	return buf.Bytes(), nil
}

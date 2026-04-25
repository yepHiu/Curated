package curatedexport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
)

// EncodeImageToJPEGWithCuratedMeta decodes the source image bytes, re-encodes as JPEG,
// and injects EXIF UserComment metadata with the curated frame JSON payload.
func EncodeImageToJPEGWithCuratedMeta(imageBytes []byte, meta FrameMetaJSON, quality int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	if quality <= 0 || quality > 100 {
		quality = 90
	}

	j, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal meta: %w", err)
	}

	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return injectJPEGExif(encoded.Bytes(), BuildExifUserComment(j))
}

func injectJPEGExif(jpegBytes []byte, exif []byte) ([]byte, error) {
	if len(jpegBytes) < 2 || jpegBytes[0] != 0xFF || jpegBytes[1] != 0xD8 {
		return nil, fmt.Errorf("invalid jpeg")
	}
	if len(exif)+2 > 0xFFFF {
		return nil, fmt.Errorf("exif payload too large")
	}

	var out bytes.Buffer
	out.Grow(len(jpegBytes) + len(exif) + 4)
	out.Write(jpegBytes[:2])
	out.WriteByte(0xFF)
	out.WriteByte(0xE1)
	segmentLen := len(exif) + 2
	out.WriteByte(byte(segmentLen >> 8))
	out.WriteByte(byte(segmentLen))
	out.Write(exif)
	out.Write(jpegBytes[2:])
	return out.Bytes(), nil
}

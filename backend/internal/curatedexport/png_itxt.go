package curatedexport

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image/png"
)

const pngSignature = "\x89PNG\r\n\x1a\n"

// InjectITxtChunk inserts an uncompressed iTXt chunk immediately before IEND.
// keyword must be 1–79 Latin-1 characters. utf8Text is stored as the text field (UTF-8).
func InjectITxtChunk(pngBytes []byte, keyword, utf8Text string) ([]byte, error) {
	if len(pngBytes) < len(pngSignature) || string(pngBytes[:len(pngSignature)]) != pngSignature {
		return nil, fmt.Errorf("invalid png signature")
	}
	if len(keyword) == 0 || len(keyword) > 79 {
		return nil, fmt.Errorf("iTXt keyword length must be 1-79")
	}
	for i := 0; i < len(keyword); i++ {
		c := keyword[i]
		if c < 0x20 || c > 0x7e {
			return nil, fmt.Errorf("iTXt keyword must be Latin-1 printable")
		}
	}
	if _, err := png.Decode(bytes.NewReader(pngBytes)); err != nil {
		return nil, fmt.Errorf("invalid png: %w", err)
	}

	itxtPayload := buildITxtPayload(keyword, utf8Text)
	itxtChunk := encodePNGChunk("iTXt", itxtPayload)

	var out bytes.Buffer
	out.WriteString(pngSignature)
	pos := len(pngSignature)
	for pos+12 <= len(pngBytes) {
		length := int(binary.BigEndian.Uint32(pngBytes[pos : pos+4]))
		chunkType := string(pngBytes[pos+4 : pos+8])
		end := pos + 12 + length
		if end > len(pngBytes) || length < 0 {
			return nil, fmt.Errorf("invalid png chunk bounds")
		}
		if chunkType == "IEND" {
			if length != 0 {
				return nil, fmt.Errorf("invalid IEND length")
			}
			out.Write(itxtChunk)
			out.Write(pngBytes[pos:end])
			out.Write(pngBytes[end:])
			return out.Bytes(), nil
		}
		out.Write(pngBytes[pos:end])
		pos = end
	}
	return nil, fmt.Errorf("png missing IEND")
}

func buildITxtPayload(keyword, utf8Text string) []byte {
	var b bytes.Buffer
	b.WriteString(keyword)
	b.WriteByte(0) // null after keyword
	b.WriteByte(0) // compression flag: uncompressed
	b.WriteByte(0) // compression method (ignored when uncompressed)
	b.WriteByte(0) // language tag (empty)
	b.WriteByte(0) // translated keyword (empty)
	b.WriteString(utf8Text)
	return b.Bytes()
}

func encodePNGChunk(chunkType string, data []byte) []byte {
	if len(chunkType) != 4 {
		panic("chunk type must be 4 bytes")
	}
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, uint32(len(data)))
	buf.WriteString(chunkType)
	buf.Write(data)
	crc := crc32.ChecksumIEEE(append([]byte(chunkType), data...))
	_ = binary.Write(&buf, binary.BigEndian, crc)
	return buf.Bytes()
}

// EncodePNGWithCuratedMetaITxt embeds FrameMetaJSON as UTF-8 JSON in an iTXt chunk (keyword CuratedMeta).
func EncodePNGWithCuratedMetaITxt(pngBytes []byte, meta FrameMetaJSON) ([]byte, error) {
	j, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal meta: %w", err)
	}
	return InjectITxtChunk(pngBytes, "CuratedMeta", string(j))
}

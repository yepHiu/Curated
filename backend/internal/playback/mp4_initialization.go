package playback

import (
	"encoding/binary"
	"io"
	"os"
)

// MP4InitializationBytes reads only bounded top-level box headers. It never
// downloads/copies mdat or allocates the moov body. Zero means unknown/non-MP4.
func MP4InitializationBytes(path string) int64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return 0
	}
	return mp4InitializationBytes(f, st.Size())
}

func mp4InitializationBytes(r io.ReaderAt, length int64) int64 {
	var header [16]byte
	var offset int64
	for boxes := 0; boxes < 64 && length-offset >= 8; boxes++ {
		if _, err := r.ReadAt(header[:8], offset); err != nil {
			return 0
		}
		kind := string(header[4:8])
		if boxes == 0 && kind != "ftyp" {
			return 0
		}
		size := uint64(binary.BigEndian.Uint32(header[:4]))
		minimum := uint64(8)
		if size == 1 {
			if length-offset < 16 {
				return 0
			}
			if _, err := r.ReadAt(header[8:], offset+8); err != nil {
				return 0
			}
			size = binary.BigEndian.Uint64(header[8:])
			minimum = 16
		} else if size == 0 {
			size = uint64(length - offset)
		}
		if size < minimum || size > uint64(length-offset) {
			return 0
		}
		if kind == "moov" {
			return int64(size)
		}
		offset += int64(size)
	}
	return 0
}

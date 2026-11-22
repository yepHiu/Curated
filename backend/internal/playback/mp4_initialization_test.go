package playback

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func mp4Box(kind string, payload []byte) []byte {
	b := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(b, uint32(len(b)))
	copy(b[4:], kind)
	copy(b[8:], payload)
	return b
}

func TestMP4InitializationBytes(t *testing.T) {
	ftyp := mp4Box("ftyp", []byte("isom"))
	moov := mp4Box("moov", make([]byte, 48))
	for _, tc := range []struct {
		name string
		data []byte
		want int64
	}{
		{"front", append(append([]byte{}, ftyp...), moov...), 56},
		{"tail", append(append(append([]byte{}, ftyp...), mp4Box("mdat", make([]byte, 100))...), moov...), 56},
		{"not mp4", []byte("not an mp4 file"), 0},
		{"truncated", append(append([]byte{}, ftyp...), moov[:12]...), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := mp4InitializationBytes(bytes.NewReader(tc.data), int64(len(tc.data))); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestMP4InitializationRejectsOverflowAndBoundsHeaderReads(t *testing.T) {
	ftyp := mp4Box("ftyp", nil)
	large := make([]byte, 16)
	binary.BigEndian.PutUint32(large, 1)
	copy(large[4:], "moov")
	binary.BigEndian.PutUint64(large[8:], ^uint64(0))
	data := append(ftyp, large...)
	if got := mp4InitializationBytes(bytes.NewReader(data), int64(len(data))); got != 0 {
		t.Fatal(got)
	}
	data = append([]byte{}, ftyp...)
	for i := 0; i < 64; i++ {
		data = append(data, mp4Box("free", nil)...)
	}
	data = append(data, mp4Box("moov", nil)...)
	if got := mp4InitializationBytes(bytes.NewReader(data), int64(len(data))); got != 0 {
		t.Fatal("exceeded box limit")
	}
}

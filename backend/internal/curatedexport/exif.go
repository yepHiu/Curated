package curatedexport

import (
	"encoding/binary"
)

// BuildExifUserComment wraps payload as EXIF UserComment (ASCII charset prefix per EXIF spec)
// and returns a full APP1-style blob: "Exif\0\0" + TIFF, suitable for WebP EXIF chunk.
func BuildExifUserComment(comment []byte) []byte {
	prefix := []byte("ASCII\000\000\000")
	payload := append(append([]byte{}, prefix...), comment...)
	if len(payload)%2 == 1 {
		payload = append(payload, 0)
	}

	const (
		tiffStart  = 6 // after "Exif\0\0"
		ifd0Off    = 8
		exifIFDOff = ifd0Off + 2 + 12 + 4 // 26
		dataOff    = exifIFDOff + 2 + 12 + 4 // 44
	)
	totalTIFF := dataOff + len(payload)
	buf := make([]byte, tiffStart+totalTIFF)
	copy(buf[0:6], []byte("Exif\x00\x00"))

	tiff := buf[tiffStart:]
	tiff[0], tiff[1] = 'I', 'I'
	binary.LittleEndian.PutUint16(tiff[2:4], 42)
	binary.LittleEndian.PutUint32(tiff[4:8], uint32(ifd0Off))

	// IFD0: ExifIFD pointer
	te := tiff[ifd0Off:]
	binary.LittleEndian.PutUint16(te[0:2], 1)
	binary.LittleEndian.PutUint16(te[2:4], 0x8769)
	binary.LittleEndian.PutUint16(te[4:6], 4)
	binary.LittleEndian.PutUint32(te[6:10], 1)
	binary.LittleEndian.PutUint32(te[10:14], uint32(exifIFDOff))
	binary.LittleEndian.PutUint32(te[14:18], 0)

	// Exif sub-IFD: UserComment
	xe := tiff[exifIFDOff:]
	binary.LittleEndian.PutUint16(xe[0:2], 1)
	binary.LittleEndian.PutUint16(xe[2:4], 0x9286)
	binary.LittleEndian.PutUint16(xe[4:6], 7)
	binary.LittleEndian.PutUint32(xe[6:10], uint32(len(payload)))
	binary.LittleEndian.PutUint32(xe[10:14], uint32(dataOff))
	binary.LittleEndian.PutUint32(xe[14:18], 0)

	copy(tiff[dataOff:], payload)
	return buf
}

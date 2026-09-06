package curatedthumb

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func BenchmarkThumbnailFormat(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 180))
	for y := 0; y < 180; y++ {
		for x := 0; x < 320; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x*7 + y*3), uint8(y * 3), uint8(x + y), 255})
		}
	}
	for _, name := range []string{"png", "jpeg85"} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var out bytes.Buffer
				if name == "png" {
					_ = png.Encode(&out, img)
				} else {
					_ = jpeg.Encode(&out, img, &jpeg.Options{Quality: 85})
				}
				b.ReportMetric(float64(out.Len()), "bytes/image")
			}
		})
	}
}

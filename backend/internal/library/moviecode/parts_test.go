package moviecode

import "testing"

// TestExtractPartIndex covers numeric ordering inputs and rejects code digits/quality suffixes.
func TestExtractPartIndex(t *testing.T) {
	cases := []struct {
		name, code string
		want       int
	}{
		{"FC2-1234567-CD1.mp4", "FC2-1234567", 1},
		{"FC2PPV-1234567_2.mp4", "FC2-1234567", 2},
		{"FC2-PPV-1234567_10.mp4", "FC2PPV-1234567", 10},
		{"ABC-123-part10.mkv", "ABC-123", 10},
		{"ABC_123_2.mp4", "ABC-123", 2},
		{"ABC-123.mp4", "ABC-123", 0},
		{"ABC-123-4K.mp4", "ABC-123", 0},
		{"FC2-12345678.mp4", "FC2-1234567", 0},
		{"ABC-123-1080p.mp4", "ABC-123", 0},
		{"invalid.mp4", "[", 0},
		{"ABC-123.mp4", "", 0},
	}
	for _, tc := range cases {
		if got := ExtractPartIndex(tc.name, tc.code); got != tc.want {
			t.Errorf("%s / %s: %d want %d", tc.name, tc.code, got, tc.want)
		}
	}
}

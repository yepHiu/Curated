package moviecode

import "testing"

// TestExtractPartIndex 验证数字及紧接番号的字母分部，不把版本和质量标记当作分部。
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
		{"FC2-PPV-4942041-1.mp4", "FC2-4942041", 1},
		{"FC2-PPV-4942041-2.mp4", "FC2-4942041", 2},
		{"STAR-380A.mp4", "STAR-380", 1},
		{"STAR-380B.mp4", "STAR-380", 2},
		{"STAR-684A-C.mp4", "STAR-684", 1},
		{"STAR-684B-C.mp4", "STAR-684", 2},
		{"star-684b-c.mp4", "STAR-684", 2},
		{"STAR-684C.mp4", "STAR-684", 3},
		{"STAR-684Z.mp4", "STAR-684", 26},
		{"STAR-684AB.mp4", "STAR-684", 0},
		{"STAR-684-第一部.mp4", "STAR-684", 0},
		{"SSIS-562-C.mp4", "SSIS-562", 0},
		{"SSIS-588-C.mp4", "SSIS-588", 0},
		{"STAR-684-UC.mp4", "STAR-684", 0},
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

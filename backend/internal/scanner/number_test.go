package scanner

import "testing"

func TestExtractNumber(t *testing.T) {
	t.Parallel()

	cases := []struct {
		filename string
		expected string
	}{
		// === Real filenames from videos_test/ ===
		{"489155.com@FC2PPV-4854489.mp4", "FC2-4854489"},
		{"489155.com@START-483.mp4", "START-483"},
		{"489155.com@EBWH-287-C.mp4", "EBWH-287"},
		{"489155.com@EBWH-317-C.mp4", "EBWH-317"},
		{"489155.com@IPZZ-788.mp4", "IPZZ-788"},
		{"489155.com@JUR-600-C.mp4", "JUR-600"},
		{"IPZZ-708-C.mp4", "IPZZ-708"},
		{"EBWH-218.mp4", "EBWH-218"},
		{"hhd800.com@EBWH-247.mp4", "EBWH-247"},
		{"FC2PPV-4162750-C.mp4", "FC2-4162750"},

		// === docs/film-scanner/videos_test/ ===
		{"SNOS-106.mp4", "SNOS-106"},
		{"IPZZ-788.mp4", "IPZZ-788"},
		{"DVMM-354.mp4", "DVMM-354"},

		// === Standard patterns ===
		{"ABC-123.mp4", "ABC-123"},
		{"abc123.mkv", "ABC-123"},
		{"ABC_123.avi", "ABC-123"},
		{"ABC 123.mov", "ABC-123"},

		// === FC2 variations ===
		{"FC2-123456.mp4", "FC2-123456"},
		{"fc2ppv123456.mkv", "FC2-123456"},
		{"FC2PPV-123456.mp4", "FC2-123456"},
		{"FC2_PPV_123456.mp4", "FC2-123456"},

		// === HEYZO ===
		{"heyzo_4321.mp4", "HEYZO-4321"},
		{"HEYZO-1234.mp4", "HEYZO-1234"},

		// === Quality suffix stripping ===
		{"ABP-100-UC.mp4", "ABP-100"},
		{"SSNI-500-HD.mkv", "SSNI-500"},
		{"MIDE-300-4K.mp4", "MIDE-300"},
		{"STARS-200-uncensored.mp4", "STARS-200"},

		// === Site domain variations ===
		{"javhd.com@SNIS-500.mp4", "SNIS-500"},
		{"xxx123.net@PRED-200-C.mp4", "PRED-200"},

		// === Edge cases ===
		{"holiday.mp4", ""},
		{"readme.txt", ""},
		{"", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.filename, func(t *testing.T) {
			t.Parallel()
			if actual := ExtractNumber(tc.filename); actual != tc.expected {
				cleaned := CleanFilename(tc.filename)
				t.Fatalf("ExtractNumber(%q) = %q, want %q (cleaned: %q)", tc.filename, actual, tc.expected, cleaned)
			}
		})
	}
}

func TestCleanFilename(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input    string
		expected string
	}{
		{"489155.com@EBWH-287-C.mp4", "EBWH-287"},
		{"hhd800.com@EBWH-247.mp4", "EBWH-247"},
		{"IPZZ-708-C.mp4", "IPZZ-708"},
		{"FC2PPV-4162750-C.mp4", "FC2PPV-4162750"},
		{"STARS-200-uncensored.mp4", "STARS-200"},
		{"ABP-100-UC.mp4", "ABP-100"},
		{"SSNI-500-HD.mkv", "SSNI-500"},
		{"normal-file.mp4", "normal-file"},
		{"no-extension", "no-extension"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if actual := CleanFilename(tc.input); actual != tc.expected {
				t.Fatalf("CleanFilename(%q) = %q, want %q", tc.input, actual, tc.expected)
			}
		})
	}
}

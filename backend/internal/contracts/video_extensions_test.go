package contracts

import "testing"

func TestSupportedVideoExtensionsWhitelist(t *testing.T) {
	want := map[string]bool{
		"mp4": true, "m4v": true, "mkv": true, "avi": true, "mov": true,
		"wmv": true, "webm": true, "ts": true, "m2ts": true, "flv": true,
		"mpeg": true, "mpg": true, "ogv": true, "rmvb": true, "iso": true,
	}
	if len(SupportedVideoExtensions) != len(want) {
		t.Fatalf("extension count = %d, want %d (%v)", len(SupportedVideoExtensions), len(want), SupportedVideoExtensions)
	}
	for _, ext := range SupportedVideoExtensions {
		if !want[ext] {
			t.Fatalf("unexpected extension %q in SupportedVideoExtensions", ext)
		}
	}
}

func TestIsSupportedVideoExtension(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{".mp4", true},
		{"MKV", true},
		{" .Rmvb ", true},
		{".iso", true},
		{".m2ts", true},
		{".txt", false},
		{"", false},
		{".", false},
		{"mp4x", false},
	}
	for _, tc := range cases {
		if got := IsSupportedVideoExtension(tc.in); got != tc.want {
			t.Fatalf("IsSupportedVideoExtension(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

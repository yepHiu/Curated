package playback

import "testing"

func TestTimestampsUnstableDetectsVFRAndUnknownAverage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		info MediaInfo
		want bool
	}{
		{name: "empty rates stay unknown", info: MediaInfo{}, want: false},
		{
			name: "matching NTSC fractions stay stable",
			info: MediaInfo{RFrameRate: "30000/1001", AvgFrameRate: "30000/1001"},
			want: false,
		},
		{
			name: "30000/1001 vs 30/1 stays within 2 percent",
			info: MediaInfo{RFrameRate: "30000/1001", AvgFrameRate: "30/1"},
			want: false,
		},
		{
			name: "60 vs 24 is unstable",
			info: MediaInfo{RFrameRate: "60/1", AvgFrameRate: "24/1"},
			want: true,
		},
		{
			name: "explicit 0/0 average is unstable when r is known",
			info: MediaInfo{RFrameRate: "30/1", AvgFrameRate: "0/0"},
			want: true,
		},
		{
			name: "empty average is not treated as probed-unknown",
			info: MediaInfo{RFrameRate: "30/1", AvgFrameRate: ""},
			want: false,
		},
		{
			name: "decimal fps mismatch",
			info: MediaInfo{RFrameRate: "29.97", AvgFrameRate: "23.976"},
			want: true,
		},
		{
			name: "DVDMS-like matching 29.97 fractions stay stable by rate",
			info: MediaInfo{RFrameRate: "30000/1001", AvgFrameRate: "1283454521/42825027"},
			want: false,
		},
		{
			name: "negative priming PTS is unstable even when rates match",
			info: MediaInfo{
				RFrameRate:          "30000/1001",
				AvgFrameRate:        "1283454521/42825027",
				HasNegativeVideoPTS: true,
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.info.TimestampsUnstable(); got != tc.want {
				t.Fatalf("TimestampsUnstable() = %v, want %v", got, tc.want)
			}
		})
	}
}

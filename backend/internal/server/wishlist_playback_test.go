package server

import (
	"strings"
	"testing"
)

func TestPlaybackPageResult(t *testing.T) {
	tests := []struct {
		name   string
		site   playbackSite
		body   string
		status string
		url    string
	}{
		{"direct match", playbackSites[0], `<title>SSIS-001</title><div class="info-header">SSIS-001</div>`, "available", "https://jable.tv/videos/ssis-001/"},
		{"challenge", playbackSites[0], `<title>Checking your browser</title>`, "unknown", ""},
		{"missav match", playbackSites[1], `<h1>SSIS-001</h1>`, "available", "https://missav.ws/ssis-001/"},
		{"different code", playbackSites[1], `<h1>SSIS-002</h1>`, "missing", ""},
		{"similar code", playbackSites[1], `<h1>SSIS-0011</h1>`, "missing", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageURL := "https://jable.tv/videos/ssis-001/"
			if tt.site.name == "MISSAV" {
				pageURL = "https://missav.ws/ssis-001/"
			}
			got := playbackPageResult(tt.site, "SSIS-001", pageURL, strings.NewReader(tt.body))
			if got.Status != tt.status || got.URL != tt.url {
				t.Fatalf("got %+v, want %s %s", got, tt.status, tt.url)
			}
		})
	}
}

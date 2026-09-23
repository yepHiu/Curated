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
		{"different code", playbackSites[1], `<h1>SSIS-002</h1>`, "missing", ""},
		{"search match", playbackSites[2], `<div class="detail"><a href="/v/one">SSIS-001 watch</a></div>`, "available", "https://123av.com/v/one"},
		{"search wrong code", playbackSites[2], `<div class="detail"><a href="/v/one">SSIS-0011 watch</a></div>`, "missing", ""},
		{"search layout changed", playbackSites[2], `<title>Search</title><main>Results unavailable</main>`, "unknown", ""},
		{"external result", playbackSites[2], `<div class="detail"><a href="https://evil.example/v/one">SSIS-001</a></div>`, "missing", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageURL := "https://jable.tv/videos/ssis-001/"
			if tt.site.search {
				pageURL = "https://123av.com/zh/search?keyword=ssis-001"
			}
			got := playbackPageResult(tt.site, "SSIS-001", pageURL, strings.NewReader(tt.body))
			if got.Status != tt.status || got.URL != tt.url {
				t.Fatalf("got %+v, want %s %s", got, tt.status, tt.url)
			}
		})
	}
}

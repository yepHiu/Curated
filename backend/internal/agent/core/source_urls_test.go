package core

import "testing"

func TestSourceURLStoreRequiresExactHTTPS(t *testing.T) {
	t.Parallel()
	store := NewSourceURLStore()
	store.Remember("ses", []string{"https://www.javbus.com/ABC-123#frag", "http://evil.test/x", "ftp://x"})
	if !store.Known("ses", "https://www.javbus.com/ABC-123") {
		t.Fatal("expected remembered https url")
	}
	if store.Known("ses", "https://www.javbus.com/OTHER") {
		t.Fatal("path not on allowlist")
	}
	if !store.HostKnown("ses", "www.javbus.com") {
		t.Fatal("expected host")
	}
}

func TestExtractSourceURLsFromDetailAndProviderItems(t *testing.T) {
	t.Parallel()
	urls := ExtractSourceURLs(Result{OK: true, Data: map[string]any{
		"source": map[string]any{
			"homepage":      "https://www.javbus.com/ABC-123",
			"externalLinks": []any{"https://example.test/actor"},
			"items": []any{
				map[string]any{"homepage": "https://www.javbus.com/ABC-124", "title": "Other"},
			},
			"summary": "see https://evil.test/ignore",
		},
	}})
	joined := stringsJoin(urls)
	if !containsAll(urls, "https://www.javbus.com/ABC-123", "https://example.test/actor", "https://www.javbus.com/ABC-124") {
		t.Fatalf("urls = %#v", urls)
	}
	if len(urls) != 3 {
		t.Fatalf("should ignore summary https: %s", joined)
	}
}

func stringsJoin(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += ","
		}
		out += value
	}
	return out
}

func containsAll(have []string, want ...string) bool {
	set := map[string]struct{}{}
	for _, item := range have {
		set[item] = struct{}{}
	}
	for _, item := range want {
		if _, ok := set[item]; !ok {
			return false
		}
	}
	return true
}

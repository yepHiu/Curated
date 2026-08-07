package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrapHandlerFrontendCachePolicy(t *testing.T) {
	distDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(distDir, "index.html"), "<!doctype html><title>Curated</title>")
	mustWriteTestFile(t, filepath.Join(distDir, "assets", "index-contenthash.js"), "console.log('curated')")
	mustWriteTestFile(t, filepath.Join(distDir, "Curated-icon.png"), "png")

	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "api-policy")
		w.WriteHeader(http.StatusNoContent)
	})
	handler := wrapHandlerWithDist(apiHandler, distDir)

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantCache    string
		wantPragma   string
		wantExpires  string
		wantBodyText string
	}{
		{
			name:         "root entry is never cached",
			path:         "/",
			wantStatus:   http.StatusOK,
			wantCache:    entryCacheControl,
			wantPragma:   "no-cache",
			wantExpires:  "0",
			wantBodyText: "Curated",
		},
		{
			name:         "direct index request is never cached",
			path:         "/index.html",
			wantStatus:   http.StatusMovedPermanently,
			wantCache:    entryCacheControl,
			wantPragma:   "no-cache",
			wantExpires:  "0",
		},
		{
			name:         "spa fallback is never cached",
			path:         "/library/saved-view",
			wantStatus:   http.StatusOK,
			wantCache:    entryCacheControl,
			wantPragma:   "no-cache",
			wantExpires:  "0",
			wantBodyText: "Curated",
		},
		{
			name:         "hashed assets are immutable",
			path:         "/assets/index-contenthash.js",
			wantStatus:   http.StatusOK,
			wantCache:    assetCacheControl,
			wantBodyText: "curated",
		},
		{
			name:         "non hashed static files revalidate",
			path:         "/Curated-icon.png",
			wantStatus:   http.StatusOK,
			wantCache:    staticCacheControl,
			wantBodyText: "png",
		},
		{
			name:       "api cache policy remains backend owned",
			path:       "/api/health",
			wantStatus: http.StatusNoContent,
			wantCache:  "api-policy",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if got := response.Header().Get("Cache-Control"); got != test.wantCache {
				t.Fatalf("Cache-Control = %q, want %q", got, test.wantCache)
			}
			if got := response.Header().Get("Pragma"); got != test.wantPragma {
				t.Fatalf("Pragma = %q, want %q", got, test.wantPragma)
			}
			if got := response.Header().Get("Expires"); got != test.wantExpires {
				t.Fatalf("Expires = %q, want %q", got, test.wantExpires)
			}
			if test.wantBodyText != "" && !strings.Contains(response.Body.String(), test.wantBodyText) {
				t.Fatalf("body %q does not contain %q", response.Body.String(), test.wantBodyText)
			}
		})
	}
}

func mustWriteTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create test directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

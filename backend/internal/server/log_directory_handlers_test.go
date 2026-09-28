package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

func TestRevealLogDirectoryLocalBoundary(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	previous := openDirectoryFn
	t.Cleanup(func() { openDirectoryFn = previous })
	var opened string
	openDirectoryFn = func(_ context.Context, path string) error { opened = path; return nil }
	routes := NewHandler(Deps{}).Routes()
	for _, tc := range []struct {
		name, peer, host, forward, origin string
		allowed                           bool
	}{
		{"local", "127.0.0.1:1234", "localhost", "", "", true},
		{"IPv6", "[::1]:1234", "[::1]", "", "", true},
		{"remote Desktop", "192.168.1.2:1234", "localhost", "", "", false},
		{"LAN target", "127.0.0.1:1234", "192.168.1.2", "", "", false},
		{"forwarded", "127.0.0.1:1234", "localhost", "192.168.1.2", "", false},
		{"remote origin", "127.0.0.1:1234", "localhost", "", "https://remote.example", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opened = ""
			req := httptest.NewRequest(http.MethodPost, "http://"+tc.host+"/api/settings/logs/reveal", strings.NewReader(`{"path":"untrusted"}`))
			req.RemoteAddr = tc.peer
			req.Header.Set("X-Curated-Client", "desktop")
			if tc.forward != "" {
				req.Header.Set("X-Forwarded-For", tc.forward)
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			res := httptest.NewRecorder()
			routes.ServeHTTP(res, req)
			if !tc.allowed {
				if res.Code != http.StatusForbidden || opened != "" {
					t.Fatalf("remote open = %d %q", res.Code, opened)
				}
				return
			}
			want, _ := filepath.Abs(config.DefaultLogDir())
			if res.Code != http.StatusNoContent || opened != want {
				t.Fatalf("open = %d %q, want %q", res.Code, opened, want)
			}
			if info, err := os.Stat(opened); err != nil || !info.IsDir() {
				t.Fatalf("directory not created: %v", err)
			}
		})
	}
	openDirectoryFn = func(context.Context, string) error { return errors.New("no desktop session") }
	req := httptest.NewRequest(http.MethodPost, "http://localhost/api/settings/logs/reveal", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	res := httptest.NewRecorder()
	routes.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("open failure = %d", res.Code)
	}
}

func TestRemoteLogSettingsCannotMutateServer(t *testing.T) {
	ctl := &stubBackendLogCtl{v: contracts.BackendLogSettingsDTO{LogLevel: "info", LogMaxAgeDays: 7}}
	routes := NewHandler(Deps{BackendLogCtl: ctl}).Routes()
	req := httptest.NewRequest(http.MethodPatch, "http://localhost/api/settings", strings.NewReader(`{"backendLog":{"logLevel":"debug","logMaxAgeDays":30}}`))
	req.RemoteAddr = "192.168.1.2:1234"
	res := httptest.NewRecorder()
	routes.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden || ctl.v.LogLevel != "info" || ctl.v.LogMaxAgeDays != 7 {
		t.Fatalf("remote log settings = %d %+v", res.Code, ctl.v)
	}
}

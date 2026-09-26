package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/tasks"
)

func TestRemoteLibraryPathManagementRejected(t *testing.T) {
	h := NewHandler(Deps{})
	routes := h.Routes()
	for _, path := range []string{"/api/library/paths", "/api/library/comics/paths", "/api/library/photos/paths"} {
		for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
			target := path
			if method != http.MethodPost {
				target += "/existing"
			}
			t.Run(method+target, func(t *testing.T) {
				r := httptest.NewRequest(method, "http://127.0.0.1:8081"+target, strings.NewReader(`{"path":"C:/Media","title":"Changed"}`))
				r.RemoteAddr = "192.168.1.30:1234"
				w := httptest.NewRecorder()
				routes.ServeHTTP(w, r)
				if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "LIBRARY_PATHS_READ_ONLY") {
					t.Fatalf("response = %d %s", w.Code, w.Body.String())
				}
			})
		}
	}
	for _, action := range []string{"reveal", "storage-binding/rebind"} {
		r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8081/api/library/paths/existing/"+action, nil)
		r.RemoteAddr = "192.168.1.30:1234"
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s = %d", action, w.Code)
		}
	}
}

func TestRemoteDefaultLibraryPathPatchRejectsWholeRequest(t *testing.T) {
	ctl := &stubDefaultImportLibraryPathCtl{id: "unchanged"}
	h := NewHandler(Deps{DefaultImportLibraryPathCtl: ctl})
	for _, field := range []string{"defaultImportLibraryPathId", "defaultComicImportLibraryPathId", "defaultPhotoImportLibraryPathId"} {
		r := httptest.NewRequest(http.MethodPatch, "http://127.0.0.1:8081/api/settings", strings.NewReader(`{"`+field+`":"changed","autoLibraryWatch":true}`))
		r.RemoteAddr = "192.168.1.30:1234"
		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, r)
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "SERVER_SETTINGS_READ_ONLY") {
			t.Fatalf("%s = %d %s", field, w.Code, w.Body.String())
		}
		if ctl.id != "unchanged" {
			t.Fatal("default directory mutated")
		}
	}
}

func TestLibraryPathCapabilityAndManagementShareLocalBoundary(t *testing.T) {
	cases := []struct {
		name, peer, host, origin, forward string
		allowed                           bool
	}{
		{"IPv4", "127.0.0.1:1234", "127.0.0.1:8081", "", "", true},
		{"IPv6", "[::1]:1234", "[::1]:8081", "", "", true},
		{"localhost", "127.0.0.1:1234", "localhost:8081", "http://localhost:5173", "", true},
		{"remote", "192.168.1.30:1234", "127.0.0.1:8081", "", "", false},
		{"LAN", "127.0.0.1:1234", "192.168.1.20:8081", "", "", false},
		{"remote origin", "127.0.0.1:1234", "127.0.0.1:8081", "https://nas.example", "", false},
		{"proxy", "127.0.0.1:1234", "localhost:8081", "", "X-Forwarded-For", false},
		{"unknown", "", "localhost:8081", "", "", false},
	}
	h := NewHandler(Deps{})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "http://"+tc.host+"/api/health", nil)
			r.RemoteAddr = tc.peer
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.forward != "" {
				r.Header.Set(tc.forward, "127.0.0.1")
			}
			w := httptest.NewRecorder()
			h.handleHealth(w, r)
			var dto contracts.HealthDTO
			if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
				t.Fatal(err)
			}
			if dto.CanManageLibraryPaths != tc.allowed {
				t.Fatalf("capability = %v", dto.CanManageLibraryPaths)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("capability cached")
			}
			called := false
			wrapped := localLibraryPathManagement(func(http.ResponseWriter, *http.Request) { called = true })
			wrapped(httptest.NewRecorder(), r)
			if called != tc.allowed {
				t.Fatalf("handler called = %v", called)
			}
			called = false
			localServerManagement(func(http.ResponseWriter, *http.Request) { called = true })(httptest.NewRecorder(), r)
			if called != tc.allowed {
				t.Fatalf("server management called = %v", called)
			}
		})
	}
}

func TestRemoteImportStillUsesServerDefaultDirectory(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	dir := filepath.Join(root, "server-library")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddLibraryPath(context.Background(), dir, "Server library")
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{
		Store: store, Logger: zap.NewNop(), Tasks: tasks.NewManager(),
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: path.ID},
	})
	body, contentType := multipartMoviesBody(t, map[string]string{"REMOTE-001.mp4": "remote-upload"})
	r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8081/api/import/movies", body)
	r.RemoteAddr = "192.168.1.30:1234"
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("upload = %d %s", w.Code, w.Body.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, "REMOTE-001.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "remote-upload" {
		t.Fatalf("content = %q", got)
	}
}

func TestRemoteServerAdministrationRejected(t *testing.T) {
	routes := NewHandler(Deps{}).Routes()
	for _, target := range []string{
		"PATCH /api/settings", "POST /api/auth/setup-pin", "POST /api/auth/change-pin",
		"PATCH /api/auth/settings", "GET /api/auth/sessions", "DELETE /api/auth/sessions/other",
		"POST /api/auth/sessions/revoke-others", "GET /api/connected-clients",
		"POST /api/maintenance/backups", "POST /api/maintenance/backups/verify", "POST /api/maintenance/backups/preflight",
		"POST /api/library/health/scan", "POST /api/library/health/repairs", "GET /api/library/health/repairs/repair",
		"POST /api/library/health/actions", "POST /api/library/comics/cache/cleanup",
		"POST /api/providers/ping", "POST /api/providers/ping-all", "POST /api/proxy/ping-javbus", "POST /api/proxy/ping-google",
		"POST /api/ai/provider/test", "PATCH /api/ai/settings", "GET /api/ai/usage", "GET /api/ai/audit", "POST /api/ai/cleanup",
	} {
		t.Run(target, func(t *testing.T) {
			method, path, _ := strings.Cut(target, " ")
			req := httptest.NewRequest(method, "http://localhost:8081"+path, strings.NewReader(`{}`))
			req.RemoteAddr = "192.168.1.30:1234"
			res := httptest.NewRecorder()
			routes.ServeHTTP(res, req)
			if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), contracts.ErrorCodeServerSettingsReadOnly) {
				t.Fatalf("remote administration = %d %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestRemoteSettingsRedactCredentialsAndRejectMutation(t *testing.T) {
	store := newImportTestStore(t, t.TempDir())
	settings := &stubAISettingsController{current: contracts.AIProviderSettingsDTO{
		Kind: "openai-compatible", APIKey: "secret-api-key", BaseURL: "https://user:password@example.com", Model: "model",
	}}
	cfg := config.Default()
	routes := NewHandler(Deps{Store: store, Cfg: cfg, AISettingsCtl: settings, ProxyCtl: &remoteSettingsProxyStub{}}).Routes()
	for _, local := range []bool{true, false} {
		req := httptest.NewRequest(http.MethodGet, "http://localhost/api/settings", nil)
		req.RemoteAddr = "192.168.1.30:1234"
		if local {
			req.RemoteAddr = "127.0.0.1:1234"
		}
		res := httptest.NewRecorder()
		routes.ServeHTTP(res, req)
		var dto contracts.SettingsDTO
		if res.Code != http.StatusOK {
			t.Fatalf("read = %d %s", res.Code, res.Body.String())
		}
		if err := json.Unmarshal(res.Body.Bytes(), &dto); err != nil {
			t.Fatal(err)
		}
		if dto.AIProvider.Model != "model" {
			t.Fatal("remote runtime settings lost")
		}
		if local && (dto.AIProvider.APIKey != "secret-api-key" || dto.Proxy.Password != "proxy-secret") {
			t.Fatal("local settings redacted")
		}
		if !local && (dto.AIProvider.APIKey != "" || dto.AIProvider.BaseURL != "" || dto.Proxy.URL != "" || dto.Proxy.Password != "" || dto.Proxy.Username != "") {
			t.Fatal("remote credentials exposed")
		}
		if res.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("settings cacheable")
		}
	}
	req := httptest.NewRequest(http.MethodPatch, "http://localhost/api/settings", strings.NewReader(`{"aiProvider":{"apiKey":"changed"},"organizeLibrary":true}`))
	req.RemoteAddr = "192.168.1.30:1234"
	res := httptest.NewRecorder()
	routes.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden || settings.current.APIKey != "secret-api-key" {
		t.Fatal("remote patch mutated settings")
	}
}

func TestRemoteSessionLockAndUnlockRemainAvailable(t *testing.T) {
	server, _ := newAuthTestServer(t)
	resp := postAuthJSON(t, server.Client(), server.URL+"/api/auth/setup-pin", map[string]any{"pin": "1234", "confirmPin": "1234"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup = %d", resp.StatusCode)
	}
	// Forwarding metadata makes an otherwise loopback test connection remote.
	var cookies []*http.Cookie
	for _, path := range []string{"unlock", "lock"} {
		req, err := http.NewRequest(http.MethodPost, server.URL+"/api/auth/"+path, strings.NewReader(`{"pin":"1234"}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "192.168.1.30")
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		cookies = res.Cookies()
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("remote %s = %d", path, res.StatusCode)
		}
	}
}

type remoteSettingsProxyStub struct{}

func (*remoteSettingsProxyStub) Proxy() config.ProxyConfig {
	return config.ProxyConfig{Enabled: true, URL: "http://user:embedded-secret@localhost:7890", Username: "proxy-user", Password: "proxy-secret"}
}
func (*remoteSettingsProxyStub) SetProxy(config.ProxyConfig) error { return nil }

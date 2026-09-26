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
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "LIBRARY_PATHS_READ_ONLY") {
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

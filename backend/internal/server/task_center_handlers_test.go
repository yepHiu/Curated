package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/storage"
)

func TestTaskCenterReadEndpoints(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err = store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := &Handler{store: store}
	mux := http.NewServeMux()
	h.registerTaskCenterRoutes(mux)
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/api/task-center/summary", 200},
		{"GET", "/api/task-center/tasks", 200},
		{"GET", "/api/task-center/tasks?state=invalid", 400},
		{"GET", "/api/task-center/tasks?limit=101", 400},
		{"GET", "/api/task-center/tasks?cursor=invalid", 400},
		{"GET", "/api/task-center/tasks/subtitle:missing", 404},
		{"POST", "/api/task-center/tasks", 405},
	} {
		r := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.RemoteAddr = "192.168.1.8:1234"
		mux.ServeHTTP(r, req)
		if r.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.path, r.Code, r.Body)
		}
		if tc.status == 200 && r.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cacheable task response")
		}
		if tc.path == "/api/task-center/tasks" && tc.method == "GET" && !strings.Contains(r.Body.String(), `"items":[]`) {
			t.Fatal(r.Body)
		}
	}
}

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
	"go.uber.org/zap"
)

// TestPhotoRatingHandlersPersistAndRejectInvalid 验证写真评分 HTTP 写入、清除、越界、缺书和 Beta 关闭。
func TestPhotoRatingHandlersPersistAndRejectInvalid(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	book, err := store.UpsertPhotoBook(t.Context(), storage.PhotoBookUpsert{
		Location: filepath.Join(root, "photo.cbz"),
		Title:    "Photo",
	})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{
		Cfg:    config.Config{PhotoLibraryEnabled: true},
		Store:  store,
		Logger: zap.NewNop(),
		Tasks:  tasks.NewManager(),
	})
	url := "/api/library/photos/" + book.ID
	request := func(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
		// 构造同步 HTTP 请求，覆盖评分写入与拒绝路径。
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}

	w := request(h.Routes(), http.MethodPatch, url, `{"ratingSet":true,"rating":4.5}`)
	if w.Code != http.StatusOK {
		t.Fatalf("set rating: %d %s", w.Code, w.Body.String())
	}
	var detail contracts.PhotoBookDetailDTO
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Rating == nil || *detail.Rating != 4.5 {
		t.Fatalf("detail = %#v", detail)
	}

	w = request(h.Routes(), http.MethodPatch, url, `{"ratingSet":true,"ratingClear":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("clear rating: %d %s", w.Code, w.Body.String())
	}
	var cleared contracts.PhotoBookDetailDTO
	if err := json.Unmarshal(w.Body.Bytes(), &cleared); err != nil {
		t.Fatal(err)
	}
	if cleared.Rating != nil {
		t.Fatalf("cleared = %#v body = %s", cleared, w.Body.String())
	}

	if w := request(h.Routes(), http.MethodPatch, url, `{"ratingSet":true,"rating":5.1}`); w.Code != http.StatusBadRequest {
		t.Fatalf("over-range: %d", w.Code)
	}
	if w := request(h.Routes(), http.MethodPatch, "/api/library/photos/missing", `{"ratingSet":true,"rating":4}`); w.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", w.Code)
	}
	disabled := NewHandler(Deps{Cfg: config.Config{}, Store: store, Logger: zap.NewNop(), Tasks: tasks.NewManager()})
	if w := request(disabled.Routes(), http.MethodPatch, url, `{"ratingSet":true,"rating":4}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "PHOTO_LIBRARY_DISABLED") {
		t.Fatal(w.Code, w.Body.String())
	}

	w = request(h.Routes(), http.MethodPatch, url, `{"title":"展示写真"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("set title: %d %s", w.Code, w.Body.String())
	}
	var titled contracts.PhotoBookDetailDTO
	if err := json.Unmarshal(w.Body.Bytes(), &titled); err != nil {
		t.Fatal(err)
	}
	if titled.Title != "展示写真" {
		t.Fatalf("titled = %#v", titled)
	}
	if w := request(h.Routes(), http.MethodPatch, url, `{"title":"  "}`); w.Code != http.StatusBadRequest {
		t.Fatalf("empty title: %d %s", w.Code, w.Body.String())
	}
}

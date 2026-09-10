package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
	"go.uber.org/zap"
)

func TestPhotoTagsPersistValidateAndRespectBeta(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	book, err := store.UpsertPhotoBook(context.Background(), storage.PhotoBookUpsert{Location: filepath.Join(root, "photo.cbz"), Title: "Photo"})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{Cfg: config.Config{PhotoLibraryEnabled: true}, Store: store, Logger: zap.NewNop(), Tasks: tasks.NewManager()})
	url := "http://127.0.0.1/api/library/photos/" + book.ID
	request := func(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}
	for _, body := range []string{`{"tags":[" portrait ","portrait","风景"]}`} {
		w := request(h.Routes(), http.MethodPatch, strings.Replace(url, "/photos/", "/photos/books/", 1)+"/tags", body)
		if w.Code != http.StatusOK {
			t.Fatalf("%d: %s", w.Code, w.Body.String())
		}
	}
	read := request(h.Routes(), http.MethodGet, url, "")
	var detail contracts.PhotoBookDetailDTO
	if err := json.Unmarshal(read.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(detail.Tags, []string{"portrait", "风景"}) {
		t.Fatal(detail.Tags)
	}
	for _, body := range []string{`{}`, `{"tags":null}`, `{"tags":"bad"}`, `{"tags":["` + strings.Repeat("界", 65) + `"]}`, `{"tags":[],"unknown":1}`, `{"tags":[]} {}`} {
		w := request(h.Routes(), http.MethodPatch, strings.Replace(url, "/photos/", "/photos/books/", 1)+"/tags", body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", body, w.Code)
		}
	}
	tooMany, _ := json.Marshal(map[string]any{"tags": make([]string, 65)})
	if w := request(h.Routes(), http.MethodPatch, strings.Replace(url, "/photos/", "/photos/books/", 1)+"/tags", string(tooMany)); w.Code != http.StatusBadRequest {
		t.Fatal(w.Code)
	}
	stored, err := store.GetPhotoBookDetail(context.Background(), book.ID)
	if err != nil || !reflect.DeepEqual(stored.Tags, detail.Tags) {
		t.Fatalf("failed request changed tags: %+v %v", stored.Tags, err)
	}
	if w := request(h.Routes(), http.MethodPatch, "http://127.0.0.1/api/library/photos/books/missing/tags", `{"tags":[]}`); w.Code != http.StatusNotFound {
		t.Fatal(w.Code)
	}
	disabled := NewHandler(Deps{Cfg: config.Config{}, Store: store, Logger: zap.NewNop(), Tasks: tasks.NewManager()})
	if w := request(disabled.Routes(), http.MethodPatch, strings.Replace(url, "/photos/", "/photos/books/", 1)+"/tags", `{"tags":[]}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "PHOTO_LIBRARY_DISABLED") {
		t.Fatal(w.Code, w.Body.String())
	}
	stored, _ = store.GetPhotoBookDetail(context.Background(), book.ID)
	if !reflect.DeepEqual(stored.Tags, detail.Tags) {
		t.Fatal("disabled request changed tags")
	}
	if w := request(h.Routes(), http.MethodPatch, strings.Replace(url, "/photos/", "/photos/books/", 1)+"/tags", `{"tags":[]}`); w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	stored, _ = store.GetPhotoBookDetail(context.Background(), book.ID)
	if len(stored.Tags) != 0 {
		t.Fatal("tags not cleared")
	}
}

package server

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
	"go.uber.org/zap"
)

func TestPhotoThumbnailIsDerivedRevalidatedAndGated(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	var source bytes.Buffer
	if err := png.Encode(&source, image.NewRGBA(image.Rect(0, 0, 1200, 800))); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, "photos.cbz")
	writeServerPhotoZip(t, archivePath, map[string]string{"001.png": source.String()})
	path, err := store.AddPhotoLibraryPath(context.Background(), root, "Photos")
	if err != nil {
		t.Fatal(err)
	}
	book, err := store.UpsertPhotoBook(context.Background(), storage.PhotoBookUpsert{LibraryPathID: path.ID, Location: archivePath, Title: "Photos", PageCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplacePhotoPages(context.Background(), book.ID, []storage.PhotoPageInput{{Index: 0, EntryPath: "001.png", FileName: "001.png", ImageExt: ".png"}}); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{Cfg: config.Config{PhotoLibraryEnabled: true}, Store: store, Logger: zap.NewNop(), Tasks: tasks.NewManager()})
	url := "http://127.0.0.1/api/library/photos/books/" + book.ID + "/pages/0/thumbnail"
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(w.Body.Bytes()))
	if err != nil || imageConfig.Width != 420 {
		t.Fatalf("%+v %v", imageConfig, err)
	}
	request := httptest.NewRequest(http.MethodGet, url, nil)
	request.Header.Set("If-None-Match", w.Header().Get("ETag"))
	cached := httptest.NewRecorder()
	h.Routes().ServeHTTP(cached, request)
	if cached.Code != http.StatusNotModified || cached.Body.Len() != 0 {
		t.Fatal("not revalidated")
	}
	original := httptest.NewRecorder()
	h.Routes().ServeHTTP(original, httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/library/photos/books/"+book.ID+"/pages/0/image", nil))
	if !bytes.Equal(original.Body.Bytes(), source.Bytes()) {
		t.Fatal("original modified")
	}
	disabled := NewHandler(Deps{Cfg: config.Config{PhotoLibraryEnabled: false}, Store: store, Logger: zap.NewNop(), Tasks: tasks.NewManager()})
	denied := httptest.NewRecorder()
	disabled.Routes().ServeHTTP(denied, request)
	if denied.Code == http.StatusOK || denied.Code == http.StatusNotModified {
		t.Fatal("disabled library exposed cached preview")
	}
}

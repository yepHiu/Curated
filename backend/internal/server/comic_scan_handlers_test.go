package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/tasks"
)

func TestHandleStartComicScan_DisabledLibrary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tasks.NewManager(),
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: false},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/api/library/comics/scans", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var appErr contracts.AppError
	if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
		t.Fatal(err)
	}
	if appErr.Code != contracts.ErrorCodeComicLibraryDisabled {
		t.Fatalf("error code = %q, want %q", appErr.Code, contracts.ErrorCodeComicLibraryDisabled)
	}
}

func TestHandleStartComicScan_IndexesSupportedArchive(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeComicZip(filepath.Join(comicRoot, "Book One.cbz"), map[string]string{
		"chapter1/001.jpg": "page 1",
		"chapter1/002.png": "page 2",
	}); err != nil {
		t.Fatal(err)
	}
	comicPath, err := store.AddComicLibraryPath(context.Background(), comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/api/library/comics/scans", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Type != contracts.TaskTypeScanComics {
		t.Fatalf("task type = %q, want %q", task.Type, contracts.TaskTypeScanComics)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		page, err := store.ListComicBooks(context.Background(), contracts.ListComicBooksRequest{Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total == 1 {
			item := page.Items[0]
			if item.Title != "Book One" || item.PageCount != 2 {
				t.Fatalf("indexed comic = %#v", item)
			}
			if item.Location != filepath.Join(comicRoot, "Book One.cbz") {
				t.Fatalf("location = %q", item.Location)
			}
			if item.ID == "" || comicPath.ID == "" {
				t.Fatalf("missing IDs: item=%q path=%q", item.ID, comicPath.ID)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for comic scan to index archive")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestHandleStartComicScan_RestrictsScanToRequestedComicPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	comicRootA := filepath.Join(root, "comics-a")
	comicRootB := filepath.Join(root, "comics-b")
	if err := os.MkdirAll(comicRootA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(comicRootB, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeComicZip(filepath.Join(comicRootA, "Book A.cbz"), map[string]string{
		"001.jpg": "a page",
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeComicZip(filepath.Join(comicRootB, "Book B.cbz"), map[string]string{
		"001.jpg": "b page",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddComicLibraryPath(context.Background(), comicRootA, "Comics A"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddComicLibraryPath(context.Background(), comicRootB, "Comics B"); err != nil {
		t.Fatal(err)
	}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, err := json.Marshal(contracts.StartScanRequest{Paths: []string{comicRootB}})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(srv.URL+"/api/library/comics/scans", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		page, err := store.ListComicBooks(context.Background(), contracts.ListComicBooksRequest{Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total == 1 {
			item := page.Items[0]
			if item.Title != "Book B" || item.Location != filepath.Join(comicRootB, "Book B.cbz") {
				t.Fatalf("indexed comic = %#v", item)
			}
			return
		}
		if page.Total > 1 {
			t.Fatalf("expected only requested comic path to be scanned, indexed %#v", page.Items)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for selected comic path scan")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func writeComicZip(path string, entries map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	for name, content := range entries {
		w, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			return err
		}
		if _, err := io.WriteString(w, content); err != nil {
			_ = writer.Close()
			return err
		}
	}
	return writer.Close()
}

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

type recordingPhotoScanStarter struct {
	paths [][]contracts.PhotoLibraryPathDTO
}

func (s *recordingPhotoScanStarter) StartPhotoScan(ctx context.Context, paths []contracts.PhotoLibraryPathDTO) (contracts.TaskDTO, error) {
	_ = ctx
	s.paths = append(s.paths, append([]contracts.PhotoLibraryPathDTO(nil), paths...))
	return tasks.NewManager().Create(contracts.TaskTypeScanPhotos, nil), nil
}

func TestHandleStartPhotoScan_IndexesSupportedArchive(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	photoPath, err := store.AddPhotoLibraryPath(context.Background(), photoRoot, "Photos")
	if err != nil {
		t.Fatal(err)
	}
	writeServerPhotoZip(t, filepath.Join(photoRoot, "Auto Portrait.cbz"), map[string]string{
		"001.jpg": "one",
		"002.jpg": "two",
	})
	h := NewHandler(Deps{
		Cfg: config.Config{
			PhotoLibraryEnabled: true,
		},
		Logger: zap.NewNop(),
		Store:  store,
		Tasks:  tasks.NewManager(),
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/api/library/photos/scans", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}

	waitForPhotoBookIndexed(t, store, "Auto Portrait")

	listResp, err := http.Get(srv.URL + "/api/library/photos")
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(listResp.Body)
		t.Fatalf("list status = %d body=%s", listResp.StatusCode, string(b))
	}
	var page contracts.PhotoBooksPageDTO
	if err := json.NewDecoder(listResp.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].Title != "Auto Portrait" || page.Items[0].CoverURL == "" {
		t.Fatalf("photo list page = %#v, want indexed item with cover", page)
	}
	_ = photoPath
}

func TestHandleStartPhotoScan_RestrictsScanToRequestedPhotoPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	firstRoot := filepath.Join(root, "photos-a")
	secondRoot := filepath.Join(root, "photos-b")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	firstPath, err := store.AddPhotoLibraryPath(context.Background(), firstRoot, "A")
	if err != nil {
		t.Fatal(err)
	}
	secondPath, err := store.AddPhotoLibraryPath(context.Background(), secondRoot, "B")
	if err != nil {
		t.Fatal(err)
	}
	starter := &recordingPhotoScanStarter{}
	h := NewHandler(Deps{
		Cfg: config.Config{
			PhotoLibraryEnabled: true,
		},
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tasks.NewManager(),
		PhotoScanStarter: starter,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Post(
		srv.URL+"/api/library/photos/scans",
		"application/json",
		bytes.NewBufferString(`{"paths":[`+quoteJSON(secondRoot)+`]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	if len(starter.paths) != 1 || len(starter.paths[0]) != 1 || starter.paths[0][0].ID != secondPath.ID {
		t.Fatalf("scan paths = %#v, want second path %q", starter.paths, secondPath.ID)
	}
	if starter.paths[0][0].ID == firstPath.ID {
		t.Fatal("scan should not include first path")
	}
}

func waitForPhotoBookIndexed(t *testing.T, store interface {
	ListPhotoBooks(context.Context, contracts.ListPhotoBooksRequest) (contracts.PhotoBooksPageDTO, error)
}, title string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		page, err := store.ListPhotoBooks(context.Background(), contracts.ListPhotoBooksRequest{Query: title, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total == 1 && page.Items[0].Title == title {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for photo book %q", title)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func writeServerPhotoZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	for entry, body := range entries {
		w, err := zw.Create(entry)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

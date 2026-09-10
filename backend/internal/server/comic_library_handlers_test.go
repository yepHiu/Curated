package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/comicscanner"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

func TestComicLibraryHandlers_ListDetailPatchDeleteAndReveal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	detail, archivePath := indexComicFixture(t, store, root)

	var revealed string
	prevReveal := revealInFileManagerFn
	revealInFileManagerFn = func(ctx context.Context, path string) error {
		_ = ctx
		revealed = path
		return nil
	}
	t.Cleanup(func() { revealInFileManagerFn = prevReveal })

	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/library/comics?q=Book&limit=10")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", resp.StatusCode)
	}
	var page contracts.ComicBooksPageDTO
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("list page = %#v", page)
	}
	if page.Items[0].CoverURL == "" {
		t.Fatal("expected list item coverUrl")
	}

	resp, err = http.Get(srv.URL + "/api/library/comics/" + detail.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail status = %d", resp.StatusCode)
	}
	var gotDetail contracts.ComicBookDetailDTO
	if err := json.NewDecoder(resp.Body).Decode(&gotDetail); err != nil {
		t.Fatal(err)
	}
	if len(gotDetail.Pages) != 2 || gotDetail.Pages[0].ImageURL == "" || gotDetail.Pages[0].ThumbURL == "" {
		t.Fatalf("detail pages = %#v", gotDetail.Pages)
	}

	patchBody := `{"title":"Renamed Book","tags":["作者:Someone","系列:Demo"],"favorite":true,"ratingSet":true,"rating":4.5}`
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/comics/"+detail.ID, bytes.NewBufferString(patchBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("patch status = %d body=%s", resp.StatusCode, string(b))
	}
	if err := json.NewDecoder(resp.Body).Decode(&gotDetail); err != nil {
		t.Fatal(err)
	}
	if gotDetail.Title != "Renamed Book" || !gotDetail.IsFavorite || gotDetail.Rating == nil || *gotDetail.Rating != 4.5 {
		t.Fatalf("patched detail = %#v", gotDetail)
	}
	if len(gotDetail.Tags) != 2 {
		t.Fatalf("patched tags = %#v", gotDetail.Tags)
	}

	req, err = http.NewRequest(http.MethodPost, srv.URL+"/api/library/comics/books/"+detail.ID+"/reveal", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("reveal status = %d", resp.StatusCode)
	}
	if revealed != archivePath {
		t.Fatalf("revealed = %q, want %q", revealed, archivePath)
	}

	req, err = http.NewRequest(http.MethodDelete, srv.URL+"/api/library/comics/"+detail.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("source archive must remain after deleting index: %v", err)
	}
	if _, err := store.GetComicBookDetail(context.Background(), detail.ID); err == nil {
		t.Fatal("comic index should be removed")
	}
}

func TestComicLibraryHandlers_DisabledLibraryRejectsList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: false},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/library/comics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var appErr contracts.AppError
	if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
		t.Fatal(err)
	}
	if appErr.Code != contracts.ErrorCodeComicLibraryDisabled {
		t.Fatalf("error code = %q", appErr.Code)
	}
}

func indexComicFixture(t *testing.T, store comicScannerStore, root string) (contracts.ComicBookDetailDTO, string) {
	t.Helper()
	ctx := context.Background()
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(comicRoot, "Book One.cbz")
	if err := writeComicZipBytes(archivePath, map[string][]byte{
		"chapter1/001.png": tinyPNG(t),
		"chapter1/002.png": tinyPNG(t),
	}); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddComicLibraryPath(ctx, comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := comicscanner.NewService(store).Scan(ctx, []contracts.ComicLibraryPathDTO{path}); err != nil {
		t.Fatal(err)
	}
	page, err := store.ListComicBooks(ctx, contracts.ListComicBooksRequest{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Fatalf("indexed total = %d, want 1", page.Total)
	}
	detail, err := store.GetComicBookDetail(ctx, page.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return detail, archivePath
}

type comicScannerStore interface {
	AddComicLibraryPath(context.Context, string, string) (contracts.ComicLibraryPathDTO, error)
	ListComicBooks(context.Context, contracts.ListComicBooksRequest) (contracts.ComicBooksPageDTO, error)
	GetComicBookDetail(context.Context, string) (contracts.ComicBookDetailDTO, error)
	comicscanner.Store
}

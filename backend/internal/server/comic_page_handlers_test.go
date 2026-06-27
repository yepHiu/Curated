package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

func TestComicPageProgressPreferencesAndCacheHandlers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	detail, archivePath := indexComicFixture(t, store, root)
	cfg := config.Default()
	cfg.CacheDir = filepath.Join(root, "cache")
	h := NewHandler(Deps{
		Cfg:              cfg,
		Logger:           zap.NewNop(),
		Store:            store,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true, cache: contracts.ComicCacheSettingsDTO{MaxBytes: 2 * 1024 * 1024 * 1024}},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/library/comics/books/" + detail.ID + "/pages")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pages status = %d", resp.StatusCode)
	}
	var pages []contracts.ComicPageDTO
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 || pages[0].ImageURL == "" || pages[0].ThumbURL == "" {
		t.Fatalf("pages = %#v", pages)
	}

	resp, err = http.Get(srv.URL + "/api/library/comics/books/" + detail.ID + "/pages/0/image")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("image status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("image content-type = %q", ct)
	}
	if b, _ := io.ReadAll(resp.Body); len(b) == 0 {
		t.Fatal("image body is empty")
	}

	resp, err = http.Get(srv.URL + "/api/library/comics/books/" + detail.ID + "/pages/0/thumbnail")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("thumbnail status = %d body=%s", resp.StatusCode, string(b))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("thumbnail content-type = %q", ct)
	}

	req, err := http.NewRequest(http.MethodPut, srv.URL+"/api/library/comics/books/"+detail.ID+"/progress", bytes.NewBufferString(`{"pageIndex":1,"completed":false}`))
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
		t.Fatalf("put progress status = %d", resp.StatusCode)
	}
	var progress contracts.ComicReadingProgressDTO
	if err := json.NewDecoder(resp.Body).Decode(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.PageIndex != 1 || progress.Completed {
		t.Fatalf("progress = %#v", progress)
	}

	req, err = http.NewRequest(http.MethodDelete, srv.URL+"/api/library/comics/books/"+detail.ID+"/progress", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete progress status = %d", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodPut, srv.URL+"/api/library/comics/books/"+detail.ID+"/preferences", bytes.NewBufferString(`{"mode":"scroll","fit":"width","direction":"rtl"}`))
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
		t.Fatalf("put preferences status = %d", resp.StatusCode)
	}
	var prefs contracts.ComicReadingPreferencesDTO
	if err := json.NewDecoder(resp.Body).Decode(&prefs); err != nil {
		t.Fatal(err)
	}
	if prefs.Mode != "scroll" || prefs.Fit != "width" || prefs.Direction != "rtl" {
		t.Fatalf("preferences = %#v", prefs)
	}

	resp, err = http.Get(srv.URL + "/api/library/comics/cache/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cache status = %d", resp.StatusCode)
	}
	var status contracts.ComicCacheStatusDTO
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.EntryCount != 1 || status.UsedBytes <= 0 {
		t.Fatalf("cache status = %#v", status)
	}

	resp, err = http.Post(srv.URL+"/api/library/comics/cache/cleanup", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cache cleanup status = %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.EntryCount != 0 || status.UsedBytes != 0 {
		t.Fatalf("cache status after cleanup = %#v", status)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("source archive must remain after cache cleanup: %v", err)
	}
}

func writeComicZipBytes(path string, entries map[string][]byte) error {
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
		if _, err := w.Write(content); err != nil {
			_ = writer.Close()
			return err
		}
	}
	return writer.Close()
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 8), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

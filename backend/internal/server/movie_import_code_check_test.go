package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

func TestHandleCheckImportMovieCodes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	if _, err := store.PersistScanMovie(context.Background(), contracts.ScanFileResultDTO{
		Path:     filepath.Join(root, "ssis-001.mp4"),
		FileName: "SSIS-001.mp4",
		Number:   "SSIS-001",
		Status:   "new",
	}); err != nil {
		t.Fatalf("persist movie: %v", err)
	}

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/movies/code-check", bytes.NewBufferString(
		`{"names":["489155.com@SSIS-001-C.mp4","holiday.mp4"]}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(body))
	}
	var dto contracts.ImportMovieCodeCheckDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.MatchedCount != 1 || len(dto.Items) != 2 {
		t.Fatalf("dto = %+v", dto)
	}
	if dto.Items[0].ExtractedCode != "SSIS-001" || len(dto.Items[0].Matches) != 1 {
		t.Fatalf("first item = %+v", dto.Items[0])
	}
	if dto.Items[0].Matches[0].MovieID != "ssis-001" || dto.Items[0].Matches[0].MatchKind != "exact" {
		t.Fatalf("match = %+v", dto.Items[0].Matches[0])
	}
	if dto.Items[0].Matches[0].Code != "SSIS-001" {
		t.Fatalf("code = %q", dto.Items[0].Matches[0].Code)
	}
	if dto.Items[1].ExtractedCode != "" || len(dto.Items[1].Matches) != 0 {
		t.Fatalf("second item = %+v", dto.Items[1])
	}
}

func TestHandleCheckImportMovieCodesRejectsEmptyBody(t *testing.T) {
	t.Parallel()
	store := newImportTestStore(t, t.TempDir())
	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/movies/code-check", bytes.NewBufferString(`{"names":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

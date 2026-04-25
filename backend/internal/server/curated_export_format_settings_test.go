package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

type stubCuratedExportFormatCtl struct {
	v   string
	err error
}

func (s *stubCuratedExportFormatCtl) CuratedFrameExportFormat() string {
	return s.v
}

func (s *stubCuratedExportFormatCtl) SetCuratedFrameExportFormat(v string) error {
	if s.err != nil {
		return s.err
	}
	s.v = v
	return nil
}

func newSettingsCuratedExportFormatTestServer(t *testing.T, deps Deps) *httptest.Server {
	t.Helper()

	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "settings.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	deps.Store = store
	if deps.Logger == nil {
		deps.Logger = zap.NewNop()
	}
	if deps.Cfg.HttpAddr == "" {
		deps.Cfg = config.Default()
	}

	srv := httptest.NewServer(NewHandler(deps).Routes())
	t.Cleanup(srv.Close)
	return srv
}

func TestHandleGetSettings_CuratedFrameExportFormatFromController(t *testing.T) {
	t.Parallel()

	exportCtl := &stubCuratedExportFormatCtl{v: "png"}
	srv := newSettingsCuratedExportFormatTestServer(t, Deps{
		Cfg:                         config.Default(),
		OrganizeLibraryCtl:          stubOrganizeCtl{},
		AutoLibraryWatchCtl:         stubAutoWatchCtl{},
		MetadataScrapeCtl:           stubMetadataCtl{},
		CuratedFrameExportFormatCtl: exportCtl,
	})

	resp, err := http.Get(srv.URL + "/api/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var dto contracts.SettingsDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if got, want := dto.CuratedFrameExportFormat, "png"; got != want {
		t.Fatalf("CuratedFrameExportFormat = %q, want %q", got, want)
	}
}

func TestHandlePatchSettings_CuratedFrameExportFormat(t *testing.T) {
	t.Parallel()

	exportCtl := &stubCuratedExportFormatCtl{v: "jpg"}
	srv := newSettingsCuratedExportFormatTestServer(t, Deps{
		Cfg:                         config.Default(),
		OrganizeLibraryCtl:          stubOrganizeCtl{},
		AutoLibraryWatchCtl:         stubAutoWatchCtl{},
		MetadataScrapeCtl:           stubMetadataCtl{},
		CuratedFrameExportFormatCtl: exportCtl,
	})

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", strings.NewReader(`{"curatedFrameExportFormat":"webp"}`))
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

	var dto contracts.SettingsDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if got, want := exportCtl.v, "webp"; got != want {
		t.Fatalf("controller value = %q, want %q", got, want)
	}
	if got, want := dto.CuratedFrameExportFormat, "webp"; got != want {
		t.Fatalf("dto CuratedFrameExportFormat = %q, want %q", got, want)
	}
}

func TestHandlePatchSettings_CuratedFrameExportFormatRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	exportCtl := &stubCuratedExportFormatCtl{v: "jpg"}
	srv := newSettingsCuratedExportFormatTestServer(t, Deps{
		Cfg:                         config.Default(),
		OrganizeLibraryCtl:          stubOrganizeCtl{},
		AutoLibraryWatchCtl:         stubAutoWatchCtl{},
		MetadataScrapeCtl:           stubMetadataCtl{},
		CuratedFrameExportFormatCtl: exportCtl,
	})

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", strings.NewReader(`{"curatedFrameExportFormat":"gif"}`))
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
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(body))
	}
	if got, want := exportCtl.v, "jpg"; got != want {
		t.Fatalf("controller value = %q, want %q after invalid patch", got, want)
	}
}

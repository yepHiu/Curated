package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

type stubLibraryPathStorageStatusProvider struct {
	checkIDs []string
}

func (p *stubLibraryPathStorageStatusProvider) ListLibraryPathStorageStatus(ctx context.Context) (contracts.LibraryPathStorageStatusListDTO, error) {
	_ = ctx
	return contracts.LibraryPathStorageStatusListDTO{
		Items: []contracts.LibraryPathStorageStatusDTO{{
			LibraryPathID: "library-1",
			Path:          `E:\Movies`,
			Title:         "External HDD",
			Status:        contracts.LibraryPathStorageStatusOffline,
			Message:       "storage root is not available",
			CheckedAt:     "2026-05-11T00:00:00Z",
			RootPath:      `E:\`,
			DriveType:     "removable",
			CanRescan:     false,
			CanImport:     false,
		}},
	}, nil
}

func (p *stubLibraryPathStorageStatusProvider) CheckLibraryPathStorageStatus(ctx context.Context, ids []string) (contracts.LibraryPathStorageStatusListDTO, error) {
	_ = ctx
	p.checkIDs = append([]string(nil), ids...)
	return contracts.LibraryPathStorageStatusListDTO{
		Items: []contracts.LibraryPathStorageStatusDTO{{
			LibraryPathID: "library-2",
			Path:          `F:\Movies`,
			Title:         "Vault",
			Status:        contracts.LibraryPathStorageStatusOnline,
			Message:       "Storage path is online.",
			CheckedAt:     "2026-05-11T00:00:01Z",
			RootPath:      `F:\`,
			CanRescan:     true,
			CanImport:     true,
		}},
	}, nil
}

func (p *stubLibraryPathStorageStatusProvider) RebindLibraryPathStorage(ctx context.Context, id string) (contracts.LibraryPathStorageStatusDTO, error) {
	_ = ctx
	return contracts.LibraryPathStorageStatusDTO{
		LibraryPathID: id,
		Path:          `F:\Movies`,
		Title:         "Vault",
		Status:        contracts.LibraryPathStorageStatusOnline,
		Message:       "Storage path is online.",
		CheckedAt:     "2026-05-11T00:00:02Z",
		RootPath:      `F:\`,
		CanRescan:     true,
		CanImport:     true,
	}, nil
}

func TestHandleGetLibraryPathStorageStatus(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{
		Cfg:                              config.Default(),
		Logger:                           zap.NewNop(),
		LibraryPathStorageStatusProvider: &stubLibraryPathStorageStatusProvider{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/library/paths/storage-status", http.NoBody)
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var dto contracts.LibraryPathStorageStatusListDTO
	if err := json.NewDecoder(rr.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if len(dto.Items) != 1 || dto.Items[0].Status != contracts.LibraryPathStorageStatusOffline {
		t.Fatalf("dto = %+v", dto)
	}
}

func TestHandleCheckLibraryPathStorageStatus(t *testing.T) {
	t.Parallel()

	provider := &stubLibraryPathStorageStatusProvider{}
	h := NewHandler(Deps{
		Cfg:                              config.Default(),
		Logger:                           zap.NewNop(),
		LibraryPathStorageStatusProvider: provider,
	})

	body := []byte(`{"libraryPathIds":["library-2"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/library/paths/storage-status/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if len(provider.checkIDs) != 1 || provider.checkIDs[0] != "library-2" {
		t.Fatalf("checkIDs = %#v", provider.checkIDs)
	}
}

func TestHandleRebindLibraryPathStorage(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{
		Cfg:                              config.Default(),
		Logger:                           zap.NewNop(),
		LibraryPathStorageStatusProvider: &stubLibraryPathStorageStatusProvider{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/library/paths/library-2/storage-binding/rebind", http.NoBody)
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var dto contracts.LibraryPathStorageStatusDTO
	if err := json.NewDecoder(rr.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.LibraryPathID != "library-2" || dto.Status != contracts.LibraryPathStorageStatusOnline {
		t.Fatalf("dto = %+v", dto)
	}
}

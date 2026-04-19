package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

type stubAppUpdateProvider struct {
	dto         contracts.AppUpdateStatusDTO
	checkCount  int
	statusCount int
}

func (s *stubAppUpdateProvider) GetAppUpdateStatus(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	_ = ctx
	s.statusCount++
	return s.dto, nil
}

func (s *stubAppUpdateProvider) CheckAppUpdateNow(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	_ = ctx
	s.checkCount++
	return s.dto, nil
}

func TestAppUpdateStatusReturnsUnsupportedWhenNotConfigured(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/app-update/status")
	if err != nil {
		t.Fatalf("GET app update status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}

	var dto struct {
		Supported        bool   `json:"supported"`
		Status           string `json:"status"`
		InstalledVersion string `json:"installedVersion"`
		ReleaseURL       string `json:"releaseUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if dto.Supported {
		t.Fatal("expected supported=false when app update checker is not configured")
	}
	if dto.Status != "unsupported" {
		t.Fatalf("status = %q, want unsupported", dto.Status)
	}
	if dto.InstalledVersion != "" {
		t.Fatalf("installedVersion = %q, want empty", dto.InstalledVersion)
	}
	if dto.ReleaseURL == "" {
		t.Fatal("expected fallback releaseUrl in unsupported response")
	}
}

func TestAppUpdateCheckReturnsUnsupportedWhenNotConfigured(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/app-update/check", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST app update check: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}

	var dto struct {
		Supported bool   `json:"supported"`
		Status    string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if dto.Supported {
		t.Fatal("expected supported=false when forced app update check is unavailable")
	}
	if dto.Status != "unsupported" {
		t.Fatalf("status = %q, want unsupported", dto.Status)
	}
}

func TestAppUpdateStatusUsesProviderResult(t *testing.T) {
	t.Parallel()

	provider := &stubAppUpdateProvider{
		dto: contracts.AppUpdateStatusDTO{
			Supported:        true,
			Status:           "update-available",
			InstalledVersion: "1.2.7",
			LatestVersion:    "1.2.8",
			HasUpdate:        true,
			ReleaseURL:       "https://github.com/yepHiu/Curated/releases/tag/v1.2.8",
		},
	}

	h := NewHandler(Deps{
		Cfg:               config.Config{},
		Logger:            zap.NewNop(),
		AppUpdateProvider: provider,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/app-update/status")
	if err != nil {
		t.Fatalf("GET app update status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if provider.statusCount != 1 {
		t.Fatalf("statusCount = %d, want 1", provider.statusCount)
	}
	if provider.checkCount != 0 {
		t.Fatalf("checkCount = %d, want 0", provider.checkCount)
	}

	var dto contracts.AppUpdateStatusDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !dto.HasUpdate || dto.LatestVersion != "1.2.8" {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

func TestAppUpdateCheckUsesForcedProviderPath(t *testing.T) {
	t.Parallel()

	provider := &stubAppUpdateProvider{
		dto: contracts.AppUpdateStatusDTO{
			Supported:        true,
			Status:           "up-to-date",
			InstalledVersion: "1.2.8",
			LatestVersion:    "1.2.8",
		},
	}

	h := NewHandler(Deps{
		Cfg:               config.Config{},
		Logger:            zap.NewNop(),
		AppUpdateProvider: provider,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/app-update/check", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST app update check: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if provider.checkCount != 1 {
		t.Fatalf("checkCount = %d, want 1", provider.checkCount)
	}
	if provider.statusCount != 0 {
		t.Fatalf("statusCount = %d, want 0", provider.statusCount)
	}
}

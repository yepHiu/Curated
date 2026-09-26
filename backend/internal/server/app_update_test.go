package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

type stubAppUpdateProvider struct {
	dto           contracts.AppUpdateStatusDTO
	checkCount    int
	statusCount   int
	downloadCount int
	installCount  int
	clearCount    int
	installMode   string
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

func (s *stubAppUpdateProvider) DownloadAppUpdateInstaller(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	_ = ctx
	s.downloadCount++
	return s.dto, nil
}

func (s *stubAppUpdateProvider) InstallAppUpdate(ctx context.Context, req contracts.AppUpdateInstallRequest) (contracts.AppUpdateStatusDTO, error) {
	_ = ctx
	s.installCount++
	s.installMode = req.Mode
	return s.dto, nil
}

func (s *stubAppUpdateProvider) ClearDownloadedAppUpdateInstaller(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	_ = ctx
	s.clearCount++
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

func TestAppUpdateDownloadUsesProvider(t *testing.T) {
	t.Parallel()

	provider := &stubAppUpdateProvider{
		dto: contracts.AppUpdateStatusDTO{
			Supported:         true,
			Status:            "update-available",
			ArtifactStatus:    "verified",
			DownloadedVersion: "1.4.5",
			InstallReady:      true,
		},
	}

	h := NewHandler(Deps{
		Cfg:               config.Config{},
		Logger:            zap.NewNop(),
		AppUpdateProvider: provider,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/app-update/download", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST app update download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if provider.downloadCount != 1 {
		t.Fatalf("downloadCount = %d, want 1", provider.downloadCount)
	}

	var dto contracts.AppUpdateStatusDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if dto.ArtifactStatus != "verified" || !dto.InstallReady {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

func TestAppUpdateInstallUsesProviderMode(t *testing.T) {
	t.Parallel()

	provider := &stubAppUpdateProvider{
		dto: contracts.AppUpdateStatusDTO{
			Supported:      true,
			Status:         "update-available",
			ArtifactStatus: "install-launched",
		},
	}

	h := NewHandler(Deps{
		Cfg:               config.Config{},
		Logger:            zap.NewNop(),
		AppUpdateProvider: provider,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/app-update/install", strings.NewReader(`{"mode":"silent"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST app update install: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if provider.installCount != 1 {
		t.Fatalf("installCount = %d, want 1", provider.installCount)
	}
	if provider.installMode != "silent" {
		t.Fatalf("installMode = %q, want silent", provider.installMode)
	}
}

func TestAppUpdateClearDownloadedInstallerUsesProvider(t *testing.T) {
	t.Parallel()

	provider := &stubAppUpdateProvider{
		dto: contracts.AppUpdateStatusDTO{
			Supported:      true,
			Status:         "update-available",
			ArtifactStatus: "",
		},
	}

	h := NewHandler(Deps{
		Cfg:               config.Config{},
		Logger:            zap.NewNop(),
		AppUpdateProvider: provider,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/app-update/downloaded-installer", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE app update downloaded installer: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if provider.clearCount != 1 {
		t.Fatalf("clearCount = %d, want 1", provider.clearCount)
	}
}

func TestAppUpdateRemoteRequestsCannotMutateServer(t *testing.T) {
	cases := []struct {
		name, peer, host, origin, header string
	}{
		{"remote peer", "192.168.1.30:4321", "127.0.0.1:8081", "", ""},
		{"LAN target", "127.0.0.1:4321", "192.168.1.20:8081", "", ""},
		{"remote origin", "127.0.0.1:4321", "127.0.0.1:8081", "https://nas.example", ""},
		{"forwarded", "127.0.0.1:4321", "127.0.0.1:8081", "", "Forwarded"},
		{"proxy", "127.0.0.1:4321", "127.0.0.1:8081", "", "X-Forwarded-For"},
		{"real ip", "127.0.0.1:4321", "127.0.0.1:8081", "", "X-Real-IP"},
		{"unknown", "", "localhost:8081", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &stubAppUpdateProvider{dto: contracts.AppUpdateStatusDTO{Supported: true, Status: "update-available", InstallReady: true}}
			h := NewHandler(Deps{AppUpdateProvider: provider})
			for _, operation := range []struct {
				method  string
				handler http.HandlerFunc
			}{
				{http.MethodPost, h.handleDownloadAppUpdateInstaller},
				{http.MethodPost, h.handleInstallAppUpdate},
				{http.MethodDelete, h.handleClearDownloadedAppUpdateInstaller},
			} {
				r := httptest.NewRequest(operation.method, "http://"+tc.host+"/api/app-update/install", nil)
				r.RemoteAddr = tc.peer
				if tc.origin != "" {
					r.Header.Set("Origin", tc.origin)
				}
				if tc.header != "" {
					r.Header.Set(tc.header, "127.0.0.1")
				}
				w := httptest.NewRecorder()
				operation.handler(w, r)
				if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "APP_UPDATE_REMOTE_DISABLED") {
					t.Fatalf("response = %d %s", w.Code, w.Body.String())
				}
			}
			if provider.downloadCount+provider.installCount+provider.clearCount != 0 {
				t.Fatal("remote request reached update provider")
			}
		})
	}
}

func TestAppUpdateCapabilityIsComputedPerRequest(t *testing.T) {
	provider := &stubAppUpdateProvider{dto: contracts.AppUpdateStatusDTO{Supported: true, Status: "up-to-date"}}
	h := NewHandler(Deps{AppUpdateProvider: provider})
	for _, peer := range []string{"127.0.0.1:1234", "192.168.1.30:1234", "[::1]:1234"} {
		r := httptest.NewRequest(http.MethodGet, "http://localhost:8081/api/app-update/status", nil)
		r.RemoteAddr = peer
		w := httptest.NewRecorder()
		h.handleGetAppUpdateStatus(w, r)
		var dto contracts.AppUpdateStatusDTO
		if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
			t.Fatal(err)
		}
		if dto.LocalUpdateAllowed != (peer != "192.168.1.30:1234") {
			t.Fatalf("wrong capability for %s", peer)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("capability must not be cached")
		}
	}
	if provider.dto.LocalUpdateAllowed {
		t.Fatal("request capability leaked into provider state")
	}
}

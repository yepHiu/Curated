package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

// stubLANAccessCtl 在设置测试中模拟局域网访问控制器。
type stubLANAccessCtl struct {
	enabled   bool
	listening bool
	urls      []string
}

// LANEnabled 返回已保存的局域网访问偏好。
func (s *stubLANAccessCtl) LANEnabled() bool { return s.enabled }

// LANListening 返回当前进程是否已绑定非 loopback。
func (s *stubLANAccessCtl) LANListening() bool { return s.listening }

// LANAccessURLs 返回测试注入的局域网 URL 列表。
func (s *stubLANAccessCtl) LANAccessURLs() []string {
	if s.urls == nil {
		return []string{}
	}
	return s.urls
}

// SetLANEnabled 只更新内存偏好，不模拟改绑。
func (s *stubLANAccessCtl) SetLANEnabled(v bool) error {
	s.enabled = v
	return nil
}

// TestHandlePatchSettings_LANEnabledWithoutPIN 确认没有 PIN 时也能打开局域网访问偏好。
func TestHandlePatchSettings_LANEnabledWithoutPIN(t *testing.T) {
	t.Parallel()

	ctl := &stubLANAccessCtl{}
	srv := newSettingsCuratedExportFormatTestServer(t, Deps{
		Cfg:                 config.Default(),
		OrganizeLibraryCtl:  stubOrganizeCtl{},
		AutoLibraryWatchCtl: stubAutoWatchCtl{},
		MetadataScrapeCtl:   stubMetadataCtl{},
		LANAccessCtl:        ctl,
	})

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", bytes.NewBufferString(`{"lanEnabled":true}`))
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
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var dto contracts.SettingsDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if !dto.LANEnabled {
		t.Fatal("expected lanEnabled true")
	}
	if !ctl.enabled {
		t.Fatal("expected controller lanEnabled true")
	}
}

// TestHandleGetSettings_LANAccessFromController 确认 GET settings 使用控制器的偏好与 URL。
func TestHandleGetSettings_LANAccessFromController(t *testing.T) {
	t.Parallel()

	ctl := &stubLANAccessCtl{
		enabled:   true,
		listening: false,
		urls:      []string{"http://192.168.1.8:8081"},
	}
	srv := newSettingsCuratedExportFormatTestServer(t, Deps{
		Cfg:                 config.Default(),
		OrganizeLibraryCtl:  stubOrganizeCtl{},
		AutoLibraryWatchCtl: stubAutoWatchCtl{},
		MetadataScrapeCtl:   stubMetadataCtl{},
		LANAccessCtl:        ctl,
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
	if !dto.LANEnabled || dto.LANListening {
		t.Fatalf("LANEnabled=%v LANListening=%v", dto.LANEnabled, dto.LANListening)
	}
	if len(dto.LANAccessURLs) != 1 || dto.LANAccessURLs[0] != "http://192.168.1.8:8081" {
		t.Fatalf("LANAccessURLs = %#v", dto.LANAccessURLs)
	}
}

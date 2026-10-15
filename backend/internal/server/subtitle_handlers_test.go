package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

type subtitleSettingsStub struct{ cfg config.SubtitleConfig }

func (s *subtitleSettingsStub) SubtitleSettings() contracts.SubtitleSettings {
	return contracts.SubtitleSettings{Enabled: s.cfg.Enabled, APIKeyConfigured: s.cfg.APIKey != "", TranslationModel: s.cfg.TranslationModel}
}
func (s *subtitleSettingsStub) SubtitleConfig() config.SubtitleConfig { return s.cfg }
func (s *subtitleSettingsStub) SetSubtitleSettings(p contracts.SubtitleSettingsPatch) error {
	if p.APIKey != nil {
		s.cfg.APIKey = *p.APIKey
	}
	return nil
}
func (s *subtitleSettingsStub) TestSubtitleProvider(context.Context) error { return nil }
func TestSubtitleSettingsNeverExposeCredentialsAndRemoteCannotWrite(t *testing.T) {
	h := &Handler{subtitleSettingsCtl: &subtitleSettingsStub{cfg: config.SubtitleConfig{Enabled: true, APIKey: "secret-not-for-client", TranslationModel: "qwen-plus"}}}
	mux := http.NewServeMux()
	h.registerSubtitleRoutes(mux)
	for _, remote := range []bool{false, true} {
		req := httptest.NewRequest("GET", "http://127.0.0.1:8080/api/subtitles/settings", nil)
		req.RemoteAddr = "127.0.0.1:1000"
		if remote {
			req.RemoteAddr = "192.168.1.2:1000"
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if strings.Contains(w.Body.String(), "secret-not-for-client") {
			t.Fatal("credential leaked")
		}
		var cfg contracts.SubtitleSettings
		_ = json.Unmarshal(w.Body.Bytes(), &cfg)
		if cfg.CanManage == remote || !cfg.APIKeyConfigured {
			t.Fatalf("settings=%+v remote=%v", cfg, remote)
		}
	}
	reveal := httptest.NewRequest("POST", "http://127.0.0.1:8080/api/subtitles/provider/key", strings.NewReader("{}"))
	reveal.RemoteAddr = "127.0.0.1:1000"
	revealed := httptest.NewRecorder()
	mux.ServeHTTP(revealed, reveal)
	if revealed.Code != 200 || revealed.Header().Get("Cache-Control") != "no-store" || !strings.Contains(revealed.Body.String(), "secret-not-for-client") {
		t.Fatal("explicit local key reveal failed")
	}
	for _, path := range []string{"/api/subtitles/provider/key", "/api/subtitles/provider/test", "/api/library/movies/movie/subtitle-jobs", "/api/subtitle-jobs/job/retry", "/api/subtitle-jobs/job/cancel", "/api/subtitles/asset/reveal"} {
		req := httptest.NewRequest("POST", "http://127.0.0.1:8080"+path, strings.NewReader("{}"))
		req.RemoteAddr = "192.168.1.2:1000"
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("remote %s: %d", path, w.Code)
		}
	}
}

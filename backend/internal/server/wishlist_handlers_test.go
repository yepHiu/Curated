package server

import (
	"bytes"
	"context"
	"curated-backend/internal/config"
	"curated-backend/internal/storage"
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// TestWishlistIntakeScope 验证无凭证提交受开关控制，且不绕过应用其他路由的 PIN。
func TestWishlistIntakeScope(t *testing.T) {
	s, e := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "api.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	ctx := context.Background()
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	ctl := &stubBrowserPluginCtl{}
	if e = s.SetAppPIN(ctx, "123456"); e != nil {
		t.Fatal(e)
	}
	h := NewHandler(Deps{Store: s, Logger: zap.NewNop(), BrowserPluginCtl: ctl}).Routes()
	// request 模拟真实扩展 Origin 请求，无需 Authorization。
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Origin", "chrome-extension://abcdefghijklmnop")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request("POST", wishlistIntakePath, `{"code":"SSIS-001"}`); w.Code != 403 {
		t.Fatalf("anonymous %d %s", w.Code, w.Body)
	}
	ctl.enabled = true
	w := request("POST", wishlistIntakePath, `{"code":"SSIS-001"}`)
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	var first map[string]string
	json.Unmarshal(w.Body.Bytes(), &first)
	w = request("POST", wishlistIntakePath, `{"code":"ssis001"}`)
	if w.Code != 200 {
		t.Fatalf("repeat %d", w.Code)
	}
	var next map[string]string
	json.Unmarshal(w.Body.Bytes(), &next)
	if first["id"] != next["id"] {
		t.Fatal("duplicate")
	}
	if w = request("POST", wishlistIntakePath, `{"code":"SSIS-002","title":"extra"}`); w.Code != 400 {
		t.Fatal("extra field accepted")
	}
	if w = request("GET", "/api/wishlist/items", ""); w.Code == http.StatusOK {
		t.Fatal("token escaped scope")
	}
	r := httptest.NewRequest("GET", "/api/wishlist/items", nil)
	locked := httptest.NewRecorder()
	h.ServeHTTP(locked, r)
	if locked.Code != http.StatusLocked {
		t.Fatalf("PIN bypassed: %d", locked.Code)
	}
	ctl.enabled = false
	if w = request("POST", wishlistIntakePath, `{"code":"SSIS-002"}`); w.Code != 403 {
		t.Fatal("disabled integration accepted request")
	}
	if _, err := s.GetWishlist(ctx, first["id"]); err != nil {
		t.Fatal("disabling removed existing item", err)
	}
}

// stubBrowserPluginCtl exercises runtime changes without reconstructing routes.
type stubBrowserPluginCtl struct{ enabled bool }

func (s *stubBrowserPluginCtl) BrowserPluginEnabled() bool           { return s.enabled }
func (s *stubBrowserPluginCtl) SetBrowserPluginEnabled(v bool) error { s.enabled = v; return nil }

// TestBrowserPluginSettings covers public API gating and the settings contract.
func TestBrowserPluginSettings(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctl := &stubBrowserPluginCtl{}
	h := NewHandler(Deps{Store: store, Cfg: config.Default(), BrowserPluginCtl: ctl, Logger: zap.NewNop()}).Routes()
	request := func(method, path, body, client string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost"+path, bytes.NewBufferString(body))
		r.Header.Set("X-Curated-Client", client)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/api/health", "", "Curated-Plugin"); w.Code != 403 || !bytes.Contains(w.Body.Bytes(), []byte("BROWSER_PLUGIN_DISABLED")) {
		t.Fatalf("plugin not blocked: %d %s", w.Code, w.Body)
	}
	if w := request("GET", "/api/health", "", ""); w.Code != 200 {
		t.Fatalf("ordinary API blocked: %d", w.Code)
	}
	for _, value := range []string{"true", "false", "true"} {
		w := request("PATCH", "/api/settings", `{"browserPluginEnabled":`+value+`}`, "")
		if w.Code != 200 {
			t.Fatalf("patch: %d %s", w.Code, w.Body)
		}
		var dto struct {
			BrowserPluginEnabled bool `json:"browserPluginEnabled"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
			t.Fatal(err)
		}
		if dto.BrowserPluginEnabled != (value == "true") {
			t.Fatal("wrong saved value")
		}
		want := 403
		if value == "true" {
			want = 200
		}
		if w := request("GET", "/api/health", "", "Curated-Plugin"); w.Code != want {
			t.Fatalf("toggle did not apply: %d", w.Code)
		}
	}
}

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"go.uber.org/zap"
)

func TestWishlistStatusIntegration(t *testing.T) {
	s, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "status.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAppPIN(ctx, "123456"); err != nil {
		t.Fatal(err)
	}
	id, _, err := s.AddWishlist(ctx, "SSIS-001", "https://javdb.com/v/private")
	if err != nil {
		t.Fatal(err)
	}
	item, _ := s.GetWishlist(ctx, id)
	completed := true
	if err := s.PatchWishlist(ctx, id, contracts.WishlistPatch{Version: item.Version, Completed: &completed}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.AddWishlist(ctx, "FC2-PPV-123456", ""); err != nil {
		t.Fatal(err)
	}
	ctl := &stubBrowserPluginCtl{}
	h := NewHandler(Deps{Store: s, Logger: zap.NewNop(), BrowserPluginCtl: ctl}).Routes()
	request := func(method, path, origin, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Curated-Client", "Curated-Plugin")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	origin := "chrome-extension://abcdefghijklmnop"
	body := `{"codes":["SSIS-001","ssis001","FC2-123456","SSIS-999"]}`
	if w := request("POST", wishlistStatusPath, origin, body); w.Code != 403 || !bytes.Contains(w.Body.Bytes(), []byte("BROWSER_PLUGIN_DISABLED")) {
		t.Fatalf("disabled: %d %s", w.Code, w.Body)
	}
	ctl.enabled = true
	w := request("POST", wishlistStatusPath, origin, body)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d %s", w.Code, w.Body)
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatal("missing cache/CORS headers")
	}
	var result contracts.WishlistStatusDTO
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"SSIS-001", "ssis001", "FC2-123456"} {
		if !result.StatusMap[code].Added {
			t.Fatalf("missing membership: %s", code)
		}
	}
	if len(result.StatusMap) != 4 || result.StatusMap["SSIS-999"].Added {
		t.Fatal("incorrect membership map")
	}
	for _, private := range []string{"sourceUrl", "private", "metadata", "note", "movieIds", id} {
		if bytes.Contains(w.Body.Bytes(), []byte(private)) {
			t.Fatalf("leaked %s", private)
		}
	}
	if w := request("OPTIONS", wishlistStatusPath, origin, ""); w.Code != 204 {
		t.Fatalf("preflight: %d", w.Code)
	}
	if w := request("POST", wishlistStatusPath, "https://missav.ws", body); w.Code != 403 {
		t.Fatalf("ordinary site allowed: %d", w.Code)
	}
	if w := request("GET", "/api/wishlist/items", "", ""); w.Code != 423 {
		t.Fatalf("application PIN bypass: %d", w.Code)
	}
	if w := request("GET", wishlistStatusPath, origin, ""); w.Code == 200 {
		t.Fatal("wrong method accepted")
	}
	for _, invalid := range []string{`{}`, `{"codes":[]}`, `{"codes":["invalid"]}`, `{"codes":["SSIS-001"],"extra":1}`, `{"codes":["SSIS-001"]} {}`, `{"codes":null}`, `{"codes":[1]}`, `{"codes":["SSIS-001","bad"]}`} {
		if w := request("POST", wishlistStatusPath, origin, invalid); w.Code != 400 {
			t.Fatalf("invalid: %s: %d %s", invalid, w.Code, w.Body)
		}
	}
	codes := make([]string, 101)
	for i := range codes {
		codes[i] = "SSIS-001"
	}
	large, _ := json.Marshal(map[string]any{"codes": codes})
	if w := request("POST", wishlistStatusPath, origin, string(large)); w.Code != 400 {
		t.Fatalf("batch limit: %d", w.Code)
	}
	page, err := s.ListWishlist(ctx, "all", "", "", 100)
	if err != nil || page.Total != 2 {
		t.Fatalf("read created data: %+v %v", page, err)
	}
	if err := s.DeleteWishlist(ctx, id); err != nil {
		t.Fatal(err)
	}
	w = request("POST", wishlistStatusPath, origin, body)
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.StatusMap["SSIS-001"].Added {
		t.Fatalf("deleted membership: %s %v", w.Body, err)
	}
	ctl.enabled = false
	if w := request("POST", wishlistStatusPath, origin, body); w.Code != 403 {
		t.Fatal("live disable ignored")
	}
}

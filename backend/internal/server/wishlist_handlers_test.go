package server

import (
	"bytes"
	"context"
	"curated-backend/internal/storage"
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// TestWishlistIntakeScope 验证锁定应用中的专用凭证只授权番号提交。
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
	token, e := s.CreateWishlistToken(ctx, "plugin", "")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetAppPIN(ctx, "123456"); e != nil {
		t.Fatal(e)
	}
	h := NewHandler(Deps{Store: s, Logger: zap.NewNop()}).Routes()
	// request 模拟真实扩展 Origin 和 bearer 请求。
	request := func(method, path, body, secret string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Authorization", "Bearer "+secret)
		r.Header.Set("Origin", "chrome-extension://abcdefghijklmnop")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request("POST", wishlistIntakePath, `{"code":"SSIS-001"}`, ""); w.Code != 401 {
		t.Fatalf("anonymous %d %s", w.Code, w.Body)
	}
	w := request("POST", wishlistIntakePath, `{"code":"SSIS-001"}`, token.Token)
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	var first map[string]string
	json.Unmarshal(w.Body.Bytes(), &first)
	w = request("POST", wishlistIntakePath, `{"code":"ssis001"}`, token.Token)
	if w.Code != 200 {
		t.Fatalf("repeat %d", w.Code)
	}
	var next map[string]string
	json.Unmarshal(w.Body.Bytes(), &next)
	if first["id"] != next["id"] {
		t.Fatal("duplicate")
	}
	if w = request("POST", wishlistIntakePath, `{"code":"SSIS-002","title":"extra"}`, token.Token); w.Code != 400 {
		t.Fatal("extra field accepted")
	}
	if w = request("GET", "/api/wishlist/items", "", token.Token); w.Code == http.StatusOK {
		t.Fatal("token escaped scope")
	}
	if e = s.DeleteWishlistToken(ctx, token.ID); e != nil {
		t.Fatal(e)
	}
	if w = request("POST", wishlistIntakePath, `{"code":"SSIS-002"}`, token.Token); w.Code != 401 {
		t.Fatal("revoked")
	}
}

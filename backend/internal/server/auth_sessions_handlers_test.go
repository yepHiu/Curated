package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func TestAuthSessionEndpointsListAndRevokeWithoutExposingBearerTokens(t *testing.T) {
	t.Parallel()

	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "auth-sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAppPIN(ctx, "123456"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	current, err := store.CreateAuthSession(ctx, storage.CreateAuthSessionInput{
		ID:             "current-bearer-token",
		ClientKey:      "current-client",
		UserAgent:      "Current Browser",
		IP:             "127.0.0.1",
		TrustedForever: true,
		Now:            now,
	})
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateAuthSession(ctx, storage.CreateAuthSessionInput{
		ID:             "other-bearer-token",
		ClientKey:      "other-client",
		UserAgent:      "Other Browser",
		IP:             "192.168.1.20",
		TrustedForever: true,
		Now:            now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)
	cookie := &http.Cookie{Name: authCookieName, Value: current.ID, Path: "/"}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/sessions", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", resp.StatusCode)
	}
	var listed contracts.AuthSessionsDTO
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if len(listed.Items) != 2 || !listed.Items[0].Current {
		t.Fatalf("listed sessions = %+v", listed.Items)
	}
	encoded, err := json.Marshal(listed)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), current.ID) || strings.Contains(string(encoded), other.ID) {
		t.Fatalf("session response exposed a bearer token: %s", encoded)
	}

	revokeReq, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/auth/sessions/"+other.PublicID, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	revokeReq.AddCookie(cookie)
	revokeResp, err := http.DefaultClient.Do(revokeReq)
	if err != nil {
		t.Fatal(err)
	}
	if revokeResp.StatusCode != http.StatusOK {
		t.Fatalf("revoke status = %d, want 200", revokeResp.StatusCode)
	}
	_ = revokeResp.Body.Close()
	if _, ok, err := store.GetValidAuthSession(ctx, other.ID, now); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("revoked trusted session is still valid")
	}

	third, err := store.CreateAuthSession(ctx, storage.CreateAuthSessionInput{
		ID:             "third-bearer-token",
		ClientKey:      "third-client",
		TrustedForever: true,
		Now:            now,
	})
	if err != nil {
		t.Fatal(err)
	}
	revokeOthers := postAuthJSONWithCookie(t, http.DefaultClient, srv.URL+"/api/auth/sessions/revoke-others", map[string]any{}, cookie)
	if revokeOthers.StatusCode != http.StatusOK {
		t.Fatalf("revoke others status = %d, want 200", revokeOthers.StatusCode)
	}
	_ = revokeOthers.Body.Close()
	if _, ok, err := store.GetValidAuthSession(ctx, current.ID, now); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("current trusted session should remain valid")
	}
	if _, ok, err := store.GetValidAuthSession(ctx, third.ID, now); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("other trusted session should be revoked")
	}

	revokeCurrentReq, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/auth/sessions/"+current.PublicID, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	revokeCurrentReq.AddCookie(cookie)
	revokeCurrentResp, err := http.DefaultClient.Do(revokeCurrentReq)
	if err != nil {
		t.Fatal(err)
	}
	if revokeCurrentResp.StatusCode != http.StatusOK {
		t.Fatalf("revoke current status = %d, want 200", revokeCurrentResp.StatusCode)
	}
	_ = revokeCurrentResp.Body.Close()
	cleared := false
	for _, responseCookie := range revokeCurrentResp.Cookies() {
		if responseCookie.Name == authCookieName && responseCookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("revoking the current trusted session did not clear its auth cookie")
	}
}

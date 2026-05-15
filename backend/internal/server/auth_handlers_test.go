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

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func newAuthTestServer(t *testing.T) (*httptest.Server, *storage.SQLiteStore) {
	t.Helper()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)
	return srv, store
}

func postAuthJSON(t *testing.T, client *http.Client, url string, body any) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do(%s) error = %v", url, err)
	}
	return resp
}

func decodeAuthJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var dto T
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return dto
}

func TestAuthStatusUnlockedWhenPINDisabled(t *testing.T) {
	t.Parallel()
	srv, _ := newAuthTestServer(t)

	resp, err := http.Get(srv.URL + "/api/auth/status")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	dto := decodeAuthJSON[contracts.AuthStatusDTO](t, resp)
	if dto.PINEnabled {
		t.Fatal("PINEnabled = true, want false")
	}
	if !dto.Unlocked {
		t.Fatal("Unlocked = false, want true when PIN is disabled")
	}
	if dto.SessionTTLMinutes != 60 {
		t.Fatalf("SessionTTLMinutes = %d, want 60", dto.SessionTTLMinutes)
	}
}

func TestAuthSetupPINRejectsMismatchedConfirmation(t *testing.T) {
	t.Parallel()
	srv, _ := newAuthTestServer(t)

	resp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/setup-pin", map[string]any{
		"pin":        "123456",
		"confirmPin": "654321",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	appErr := decodeAuthJSON[contracts.AppError](t, resp)
	if appErr.Code != contracts.ErrorCodeBadRequest {
		t.Fatalf("error code = %q, want %q", appErr.Code, contracts.ErrorCodeBadRequest)
	}
}

func TestAuthSetupAndUnlockIssuesHTTPOnlyCookie(t *testing.T) {
	t.Parallel()
	srv, _ := newAuthTestServer(t)

	setupResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/setup-pin", map[string]any{
		"pin":               "1234",
		"confirmPin":        "1234",
		"sessionTtlMinutes": 15,
	})
	if setupResp.StatusCode != http.StatusOK {
		t.Fatalf("setup status = %d, want 200", setupResp.StatusCode)
	}
	setupStatus := decodeAuthJSON[contracts.AuthStatusDTO](t, setupResp)
	if !setupStatus.PINEnabled || !setupStatus.Unlocked {
		t.Fatalf("setup status = %+v, want PIN enabled and unlocked", setupStatus)
	}
	if setupStatus.PINLength != 4 {
		t.Fatalf("setup PINLength = %d, want 4", setupStatus.PINLength)
	}

	lockResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/lock", map[string]any{})
	if lockResp.StatusCode != http.StatusOK {
		t.Fatalf("lock status = %d, want 200", lockResp.StatusCode)
	}
	_ = lockResp.Body.Close()

	unlockResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/unlock", map[string]any{
		"pin": "1234",
	})
	if unlockResp.StatusCode != http.StatusOK {
		t.Fatalf("unlock status = %d, want 200", unlockResp.StatusCode)
	}
	status := decodeAuthJSON[contracts.AuthStatusDTO](t, unlockResp)
	if !status.Unlocked || status.SessionExpiresAt == "" || status.TrustedForever {
		t.Fatalf("unlock status = %+v, want expiring unlocked session", status)
	}
	if status.PINLength != 4 {
		t.Fatalf("unlock PINLength = %d, want 4", status.PINLength)
	}
	cookie := findAuthCookie(unlockResp.Cookies())
	if cookie == nil {
		t.Fatal("expected auth cookie")
	}
	if !cookie.HttpOnly {
		t.Fatal("auth cookie must be HTTP-only")
	}
	if cookie.Value == "" {
		t.Fatal("auth cookie value is empty")
	}
}

func TestAuthUnlockTrustedForeverHasNoExpiry(t *testing.T) {
	t.Parallel()
	srv, _ := newAuthTestServer(t)

	setupResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/setup-pin", map[string]any{
		"pin":        "123456",
		"confirmPin": "123456",
	})
	_ = setupResp.Body.Close()

	unlockResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/unlock", map[string]any{
		"pin":            "123456",
		"trustedForever": true,
	})
	if unlockResp.StatusCode != http.StatusOK {
		t.Fatalf("unlock status = %d, want 200", unlockResp.StatusCode)
	}
	status := decodeAuthJSON[contracts.AuthStatusDTO](t, unlockResp)
	if !status.TrustedForever {
		t.Fatalf("TrustedForever = false in %+v", status)
	}
	if status.SessionExpiresAt != "" {
		t.Fatalf("SessionExpiresAt = %q, want empty for trusted forever", status.SessionExpiresAt)
	}
	cookie := findAuthCookie(unlockResp.Cookies())
	if cookie == nil {
		t.Fatal("expected auth cookie")
	}
	if cookie.MaxAge < 300000000 {
		t.Fatalf("trusted cookie MaxAge = %d, want long-lived persistent cookie", cookie.MaxAge)
	}
}

func TestAuthUnlockRegularCookieIsSessionScopedForIdleLock(t *testing.T) {
	t.Parallel()
	srv, _ := newAuthTestServer(t)

	setupResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/setup-pin", map[string]any{
		"pin":        "123456",
		"confirmPin": "123456",
	})
	_ = setupResp.Body.Close()

	unlockResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/unlock", map[string]any{
		"pin": "123456",
	})
	if unlockResp.StatusCode != http.StatusOK {
		t.Fatalf("unlock status = %d, want 200", unlockResp.StatusCode)
	}
	_ = unlockResp.Body.Close()
	cookie := findAuthCookie(unlockResp.Cookies())
	if cookie == nil {
		t.Fatal("expected auth cookie")
	}
	if cookie.MaxAge != 0 || !cookie.Expires.IsZero() {
		t.Fatalf("regular auth cookie should be browser-session scoped for server idle expiry, got MaxAge=%d Expires=%s", cookie.MaxAge, cookie.Expires)
	}
}

func TestAuthChangePINRequiresCurrentPINAndUpdatesStoredPIN(t *testing.T) {
	t.Parallel()
	srv, store := newAuthTestServer(t)

	setupResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/setup-pin", map[string]any{
		"pin":        "1234",
		"confirmPin": "1234",
	})
	if setupResp.StatusCode != http.StatusOK {
		t.Fatalf("setup status = %d, want 200", setupResp.StatusCode)
	}
	_ = setupResp.Body.Close()
	cookie := findAuthCookie(setupResp.Cookies())
	if cookie == nil {
		t.Fatal("expected auth cookie after setup")
	}

	wrongResp := postAuthJSONWithCookie(t, http.DefaultClient, srv.URL+"/api/auth/change-pin", map[string]any{
		"currentPin": "0000",
		"newPin":     "98765",
		"confirmPin": "98765",
	}, cookie)
	if wrongResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong current PIN status = %d, want 401", wrongResp.StatusCode)
	}
	_ = wrongResp.Body.Close()

	changeResp := postAuthJSONWithCookie(t, http.DefaultClient, srv.URL+"/api/auth/change-pin", map[string]any{
		"currentPin": "1234",
		"newPin":     "98765",
		"confirmPin": "98765",
	}, cookie)
	if changeResp.StatusCode != http.StatusOK {
		t.Fatalf("change PIN status = %d, want 200", changeResp.StatusCode)
	}
	status := decodeAuthJSON[contracts.AuthStatusDTO](t, changeResp)
	if status.PINLength != 5 || !status.Unlocked {
		t.Fatalf("change PIN status = %+v, want unlocked status with PINLength=5", status)
	}
	if ok, err := store.VerifyAppPIN(context.Background(), "1234"); err != nil {
		t.Fatalf("VerifyAppPIN(old) error = %v", err)
	} else if ok {
		t.Fatal("old PIN should not verify after change")
	}
	if ok, err := store.VerifyAppPIN(context.Background(), "98765"); err != nil {
		t.Fatalf("VerifyAppPIN(new) error = %v", err)
	} else if !ok {
		t.Fatal("new PIN should verify after change")
	}
}

func TestAuthMiddlewareLocksSensitiveAPIUntilUnlocked(t *testing.T) {
	t.Parallel()
	srv, store := newAuthTestServer(t)
	if err := store.SetAppPIN(context.Background(), "123456"); err != nil {
		t.Fatalf("SetAppPIN() error = %v", err)
	}

	lockedResp, err := http.Get(srv.URL + "/api/library/movies")
	if err != nil {
		t.Fatal(err)
	}
	if lockedResp.StatusCode != http.StatusLocked {
		t.Fatalf("locked status = %d, want 423", lockedResp.StatusCode)
	}
	appErr := decodeAuthJSON[contracts.AppError](t, lockedResp)
	if appErr.Code != contracts.ErrorCodeAuthLocked {
		t.Fatalf("error code = %q, want %q", appErr.Code, contracts.ErrorCodeAuthLocked)
	}

	unlockResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/unlock", map[string]any{
		"pin": "123456",
	})
	if unlockResp.StatusCode != http.StatusOK {
		t.Fatalf("unlock status = %d, want 200", unlockResp.StatusCode)
	}
	_ = unlockResp.Body.Close()
	cookie := findAuthCookie(unlockResp.Cookies())
	if cookie == nil {
		t.Fatal("expected auth cookie after unlock")
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/library/movies", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(cookie)
	okResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer okResp.Body.Close()
	if okResp.StatusCode == http.StatusLocked {
		t.Fatal("unlocked request still returned 423")
	}
}

func findAuthCookie(cookies []*http.Cookie) *http.Cookie {
	for _, cookie := range cookies {
		if cookie != nil && cookie.Name == "curated_auth" && strings.TrimSpace(cookie.Value) != "" {
			return cookie
		}
	}
	return nil
}

func postAuthJSONWithCookie(t *testing.T, client *http.Client, url string, body any, cookie *http.Cookie) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do(%s) error = %v", url, err)
	}
	return resp
}

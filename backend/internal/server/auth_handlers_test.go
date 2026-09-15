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
	"time"

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

// TestAuthSettingsCannotDisablePINWhileLANModeIsEnabled 确认非 loopback 局域网监听下不能关闭 PIN。
func TestAuthSettingsCannotDisablePINWhileLANModeIsEnabled(t *testing.T) {
	t.Parallel()

	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "lan-auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{
		Cfg: config.Config{
			HttpAddr:   "0.0.0.0:8081",
			LANEnabled: true,
		},
		Logger: zap.NewNop(),
		Store:  store,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	setupResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/setup-pin", map[string]any{
		"pin":        "123456",
		"confirmPin": "123456",
	})
	if setupResp.StatusCode != http.StatusOK {
		t.Fatalf("setup status = %d, want 200", setupResp.StatusCode)
	}
	_ = setupResp.Body.Close()
	cookie := findAuthCookie(setupResp.Cookies())
	if cookie == nil {
		t.Fatal("expected auth cookie after setup")
	}

	payload := bytes.NewBufferString(`{"pinEnabled":false}`)
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/auth/settings", payload)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("disable PIN status = %d, want 400", resp.StatusCode)
	}
	appErr := decodeAuthJSON[contracts.AppError](t, resp)
	if appErr.Code != contracts.ErrorCodeAuthPINRequiredForLAN {
		t.Fatalf("disable PIN error code = %q, want %q", appErr.Code, contracts.ErrorCodeAuthPINRequiredForLAN)
	}

	settings, err := store.GetAppSecuritySettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !settings.PINEnabled {
		t.Fatal("PIN was disabled while LAN mode remained enabled")
	}
}

func TestAuthUnlockRateLimitReturnsRetryAfterAndResetsAfterSuccess(t *testing.T) {
	t.Parallel()

	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "auth-rate-limit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAppPIN(context.Background(), "123456"); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	h.authAttempts.now = func() time.Time { return now }
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	for attempt := 1; attempt <= authFailuresBeforeBackoff; attempt++ {
		resp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/unlock", map[string]any{"pin": "000000"})
		if attempt < authFailuresBeforeBackoff {
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("attempt %d status = %d, want 401", attempt, resp.StatusCode)
			}
			_ = resp.Body.Close()
			continue
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("attempt %d status = %d, want 429", attempt, resp.StatusCode)
		}
		if got := resp.Header.Get("Retry-After"); got != "1" {
			t.Fatalf("Retry-After = %q, want 1", got)
		}
		appErr := decodeAuthJSON[contracts.AppError](t, resp)
		if appErr.Code != contracts.ErrorCodeAuthRateLimited || !appErr.Retryable {
			t.Fatalf("rate limit error = %+v", appErr)
		}
		if got := appErr.Details["retryAfterSeconds"]; got != float64(1) {
			t.Fatalf("retryAfterSeconds = %#v, want 1", got)
		}
	}

	blocked := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/unlock", map[string]any{"pin": "123456"})
	if blocked.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("correct PIN during backoff status = %d, want 429", blocked.StatusCode)
	}
	_ = blocked.Body.Close()

	now = now.Add(time.Second)
	unlocked := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/unlock", map[string]any{"pin": "123456"})
	if unlocked.StatusCode != http.StatusOK {
		t.Fatalf("correct PIN after backoff status = %d, want 200", unlocked.StatusCode)
	}
	_ = unlocked.Body.Close()

	wrongAfterReset := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/unlock", map[string]any{"pin": "000000"})
	if wrongAfterReset.StatusCode != http.StatusUnauthorized {
		t.Fatalf("first failure after reset status = %d, want 401", wrongAfterReset.StatusCode)
	}
	_ = wrongAfterReset.Body.Close()
}

func TestAuthSetupPINRateLimitsInvalidAttempts(t *testing.T) {
	t.Parallel()

	srv, _ := newAuthTestServer(t)
	for attempt := 1; attempt <= authFailuresBeforeBackoff; attempt++ {
		resp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/setup-pin", map[string]any{
			"pin":        "123456",
			"confirmPin": "654321",
		})
		if attempt < authFailuresBeforeBackoff {
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("attempt %d status = %d, want 400", attempt, resp.StatusCode)
			}
			_ = resp.Body.Close()
			continue
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("attempt %d status = %d, want 429", attempt, resp.StatusCode)
		}
		if got := resp.Header.Get("Retry-After"); got != "1" {
			t.Fatalf("Retry-After = %q, want 1", got)
		}
		appErr := decodeAuthJSON[contracts.AppError](t, resp)
		if appErr.Code != contracts.ErrorCodeAuthRateLimited {
			t.Fatalf("rate limit code = %q, want %q", appErr.Code, contracts.ErrorCodeAuthRateLimited)
		}
	}
}

func TestAuthBackoffGrowsExponentiallyAndCaps(t *testing.T) {
	t.Parallel()

	if got := authBackoffForFailures(4); got != 0 {
		t.Fatalf("backoff before threshold = %s, want 0", got)
	}
	if got := authBackoffForFailures(5); got != time.Second {
		t.Fatalf("backoff at threshold = %s, want 1s", got)
	}
	if got := authBackoffForFailures(6); got != 2*time.Second {
		t.Fatalf("second backoff = %s, want 2s", got)
	}
	if got := authBackoffForFailures(100); got != authBackoffMaximum {
		t.Fatalf("capped backoff = %s, want %s", got, authBackoffMaximum)
	}
}

// TestAuthSettingsCanDisablePINOnLoopback 确认默认本机监听下可以关闭 PIN 锁。
func TestAuthSettingsCanDisablePINOnLoopback(t *testing.T) {
	t.Parallel()
	srv, store := newAuthTestServer(t)

	setupResp := postAuthJSON(t, http.DefaultClient, srv.URL+"/api/auth/setup-pin", map[string]any{
		"pin":        "123456",
		"confirmPin": "123456",
	})
	if setupResp.StatusCode != http.StatusOK {
		t.Fatalf("setup status = %d, want 200", setupResp.StatusCode)
	}
	cookie := findAuthCookie(setupResp.Cookies())
	_ = setupResp.Body.Close()
	if cookie == nil {
		t.Fatal("expected auth cookie after setup")
	}

	payload := bytes.NewBufferString(`{"pinEnabled":false}`)
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/auth/settings", payload)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("disable PIN status = %d, want 200", resp.StatusCode)
	}
	dto := decodeAuthJSON[contracts.AuthStatusDTO](t, resp)
	if dto.PINEnabled {
		t.Fatal("PINEnabled = true, want false after disable")
	}
	if !dto.Unlocked || !dto.SetupRequired || dto.PINLength != 0 {
		t.Fatalf("disable status = %+v, want unlocked setup-required with no PIN length", dto)
	}

	settings, err := store.GetAppSecuritySettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if settings.PINEnabled || settings.PINHash != "" || settings.PINLength != 0 {
		t.Fatalf("stored settings = %+v, want disabled with cleared secret", settings)
	}

	statusResp, err := http.Get(srv.URL + "/api/auth/status")
	if err != nil {
		t.Fatal(err)
	}
	status := decodeAuthJSON[contracts.AuthStatusDTO](t, statusResp)
	if status.PINEnabled || !status.Unlocked {
		t.Fatalf("status after disable = %+v, want PIN off and unlocked", status)
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

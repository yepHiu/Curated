package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

func TestRequestSecurityAllowsLoopbackDevelopmentOrigin(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{Cfg: config.Config{HttpAddr: "127.0.0.1:8080"}, Logger: zap.NewNop()})
	req := httptest.NewRequest(http.MethodOptions, "http://127.0.0.1:8080/api/health", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	rr := httptest.NewRecorder()

	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q", got)
	}
}

func TestRequestSecurityRejectsUnlistedBrowserOrigin(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{Cfg: config.Config{HttpAddr: "127.0.0.1:8080"}, Logger: zap.NewNop()})
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/api/health", nil)
	req.Header.Set("Origin", "https://attacker.example")
	rr := httptest.NewRecorder()

	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
	assertAppErrorCode(t, rr, contracts.ErrorCodeForbidden)
}

func TestRequestSecurityAllowsLANSameOrigin(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{Cfg: config.Config{HttpAddr: ":8081", LANEnabled: true}, Logger: zap.NewNop()})
	req := httptest.NewRequest(http.MethodGet, "http://192.168.1.25:8081/api/health", nil)
	req.Header.Set("Origin", "http://192.168.1.25:8081")
	rr := httptest.NewRecorder()

	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://192.168.1.25:8081" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestRequestSecurityAllowsExplicitCrossOrigin(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{Cfg: config.Config{
		HttpAddr:           "127.0.0.1:8080",
		CORSAllowedOrigins: []string{"https://app.example"},
	}, Logger: zap.NewNop()})
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/api/health", nil)
	req.Header.Set("Origin", "https://app.example")
	rr := httptest.NewRecorder()

	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

// TestRequestSecurityRejectsLANHostWhileLoopbackBound 确认仅保存 LAN 偏好、仍绑 loopback 时拒绝私网 Host。
func TestRequestSecurityRejectsLANHostWhileLoopbackBound(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{Cfg: config.Config{HttpAddr: "127.0.0.1:8080", LANEnabled: true}, Logger: zap.NewNop()})
	req := httptest.NewRequest(http.MethodGet, "http://192.168.1.25:8080/api/health", nil)
	rr := httptest.NewRecorder()

	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	assertAppErrorCode(t, rr, contracts.ErrorCodeForbidden)
}

func TestRequestSecurityRejectsDNSRebindingHost(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{Cfg: config.Config{HttpAddr: "127.0.0.1:8080"}, Logger: zap.NewNop()})
	req := httptest.NewRequest(http.MethodGet, "http://attacker.example/api/health", nil)
	rr := httptest.NewRecorder()

	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	assertAppErrorCode(t, rr, contracts.ErrorCodeForbidden)
}

func assertAppErrorCode(t *testing.T, rr *httptest.ResponseRecorder, want string) {
	t.Helper()
	var appErr contracts.AppError
	if err := json.NewDecoder(rr.Result().Body).Decode(&appErr); err != nil {
		t.Fatal(err)
	}
	if appErr.Code != want {
		t.Fatalf("app error code = %q, want %q", appErr.Code, want)
	}
}

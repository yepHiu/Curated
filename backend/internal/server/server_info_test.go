package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"go.uber.org/zap"
)

func TestServerInfoIsPublicAndMinimal(t *testing.T) {
	if !isAuthPublicPath(http.MethodGet, "/api/server-info") || isAuthPublicPath(http.MethodPost, "/api/server-info") {
		t.Fatal("wrong public methods")
	}
	h := NewHandler(Deps{Cfg: config.Config{ServerID: "installation-id", ServerName: "My NAS", DatabasePath: "/private/library.db"}, Logger: zap.NewNop()})
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, httptest.NewRequest("GET", "/api/server-info", nil))
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"serverId":"installation-id"`) || strings.Contains(rr.Body.String(), "private") {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("identity must not be cached")
	}
}

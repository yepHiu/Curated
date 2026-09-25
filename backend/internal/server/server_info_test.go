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

func TestDiscoveryDescriptionRequiresLANAndEscapesName(t *testing.T) {
	enabled, disabled := true, false
	for _, tc := range []struct {
		lan       bool
		addr      string
		discovery *bool
		status    int
	}{
		{false, "127.0.0.1:8081", &enabled, 404}, {true, "127.0.0.1:8081", &enabled, 404},
		{true, "0.0.0.0:8081", &disabled, 404}, {true, "0.0.0.0:8081", &enabled, 200},
	} {
		h := NewHandler(Deps{Cfg: config.Config{ServerID: "id", ServerName: "A & B", LANEnabled: tc.lan, HttpAddr: tc.addr, DiscoveryEnabled: tc.discovery}, Logger: zap.NewNop()})
		rr := httptest.NewRecorder()
		h.Routes().ServeHTTP(rr, httptest.NewRequest("GET", "http://localhost:8081/discovery/description.xml", nil))
		if rr.Code != tc.status {
			t.Fatalf("%+v: %d", tc, rr.Code)
		}
		if rr.Code == 200 && !strings.Contains(rr.Body.String(), "A &amp; B") {
			t.Fatal(rr.Body.String())
		}
	}
}

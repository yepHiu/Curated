package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/version"
	"go.uber.org/zap"
)

func TestHealthSeparatesProductVersionAndBuildStamp(t *testing.T) {
	h := NewHandler(Deps{Logger: zap.NewNop()})
	rr := httptest.NewRecorder()
	h.handleHealth(rr, httptest.NewRequest("GET", "/api/health", nil))
	var health contracts.HealthDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health.Version != version.ProductVersion() || health.BuildStamp != version.Stamp() || health.Channel != version.Channel {
		t.Fatalf("unexpected health identities: %#v", health)
	}
}

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
)

func TestHandleConnectedClientsRecordsRequestsAndOmitsMacAddress(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{Logger: zap.NewNop()})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/health", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "curl/8.7.1")
	if resp, err := http.DefaultClient.Do(req); err != nil {
		t.Fatal(err)
	} else {
		_ = resp.Body.Close()
	}

	resp, err := http.Get(srv.URL + "/api/connected-clients")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var dto contracts.ConnectedClientsDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.Total == 0 || len(dto.Clients) == 0 {
		t.Fatalf("expected at least one tracked client, got total=%d len=%d", dto.Total, len(dto.Clients))
	}
	if dto.SampledAt == "" {
		t.Fatal("expected sampledAt")
	}
	client := dto.Clients[0]
	if client.IP == "" {
		t.Fatal("expected client IP")
	}
	if client.Browser == "" || client.OS == "" || client.DeviceType == "" {
		t.Fatalf("expected parsed client metadata, got browser=%q os=%q device=%q", client.Browser, client.OS, client.DeviceType)
	}

	raw := map[string]any{}
	b, _ := json.Marshal(dto)
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["serverMac"]; ok {
		t.Fatal("connected clients response must not expose serverMac")
	}
}

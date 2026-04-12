package app

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/version"
)

func TestHandleCommand_SystemHealthIncludesInstallerVersion(t *testing.T) {
	t.Parallel()

	prev := version.InstallerVersion
	version.InstallerVersion = "1.1.3"
	t.Cleanup(func() {
		version.InstallerVersion = prev
	})

	a := &App{
		cfg:    config.Config{DatabasePath: "runtime/curated.db"},
		logger: zap.NewNop(),
	}

	var out bytes.Buffer
	err := a.handleCommand(context.Background(), &out, contracts.Command{
		ID:   "health-1",
		Type: contracts.CommandSystemHealth,
	})
	if err != nil {
		t.Fatal(err)
	}

	var resp contracts.Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !resp.OK {
		t.Fatalf("response not ok: %#v", resp)
	}

	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("marshal response data: %v", err)
	}

	var health contracts.HealthDTO
	if err := json.Unmarshal(raw, &health); err != nil {
		t.Fatalf("unmarshal health dto: %v", err)
	}

	if got, want := health.InstallerVersion, "1.1.3"; got != want {
		t.Fatalf("health.InstallerVersion = %q, want %q", got, want)
	}
}

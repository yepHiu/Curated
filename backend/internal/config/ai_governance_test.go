package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAIGovernanceOptionalStepLimit(t *testing.T) {
	if DefaultAIGovernance().StepLimit != 0 {
		t.Fatal("default tool limit must be disabled")
	}
	for _, limit := range []int{0, 1, 15, 30, -1, 31} {
		cfg := DefaultAIGovernance()
		cfg.StepLimit = limit
		if valid := cfg.Validate() == nil; valid != (limit >= 0 && limit <= 30) {
			t.Fatalf("limit %d valid=%v", limit, valid)
		}
	}
}

func TestLibrarySettingsPreserveOptionalStepLimit(t *testing.T) {
	for _, tc := range []struct {
		json string
		want int
	}{{`{"aiGovernance":{}}`, 0}, {`{"aiGovernance":{"stepLimit":0}}`, 0}, {`{"aiGovernance":{"stepLimit":15}}`, 15}} {
		path := filepath.Join(t.TempDir(), "library-config.cfg")
		if err := os.WriteFile(path, []byte(tc.json), 0600); err != nil {
			t.Fatal(err)
		}
		var cfg Config
		if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
			t.Fatal(err)
		}
		if cfg.AIGovernance == nil || cfg.AIGovernance.StepLimit != tc.want {
			t.Fatalf("governance = %+v, want limit %d", cfg.AIGovernance, tc.want)
		}
	}
}

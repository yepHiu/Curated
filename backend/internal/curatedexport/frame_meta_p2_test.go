package curatedexport

import (
	"encoding/json"
	"testing"
)

func TestFrameMetaJSON_P2RoundTrip(t *testing.T) {
	t.Parallel()

	meta := FrameMetaJSON{
		Title:         "round-trip-title",
		Code:          "ABC-123",
		Actors:        []string{"Mina"},
		PositionSec:   12.5,
		CapturedAt:    "2026-04-11T10:00:00Z",
		FrameID:       "frame-1",
		MovieID:       "movie-1",
		Tags:          []string{"closeup", "favorite"},
		SchemaVersion: 1,
		ExportedAt:    "2026-04-11T10:01:00Z",
		AppName:       "Curated",
		AppVersion:    "test-dev",
	}

	raw, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}

	var out FrameMetaJSON
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	if out.SchemaVersion != 1 || out.ExportedAt == "" || out.AppName != "Curated" || out.AppVersion == "" {
		t.Fatalf("export metadata version fields missing after round trip: %+v", out)
	}
	if len(out.Tags) != 2 || out.Tags[0] != "closeup" || out.Tags[1] != "favorite" {
		t.Fatalf("tags = %#v", out.Tags)
	}
}

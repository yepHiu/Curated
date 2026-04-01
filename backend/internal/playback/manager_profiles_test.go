package playback

import (
	"strings"
	"testing"
)

func TestBuildTranscodeProfilesIncludesHardwareCandidatesWhenEnabled(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: true},
		"movie.mkv",
		"segment-%05d.ts",
		"index.m3u8",
		"",
		0,
	)
	if len(profiles) < 2 {
		t.Fatalf("expected hardware profiles plus software fallback, got %d", len(profiles))
	}
	if profiles[len(profiles)-1].Name != "libx264" {
		t.Fatalf("last profile = %q, want libx264 fallback", profiles[len(profiles)-1].Name)
	}
}

func TestBuildTranscodeProfilesSoftwareOnlyWhenHardwareDisabled(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.ts",
		"index.m3u8",
		"",
		0,
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	if profiles[0].Name != "libx264" {
		t.Fatalf("profile name = %q, want libx264", profiles[0].Name)
	}
}

func TestBuildTranscodeProfilesPrefersConfiguredHardwareEncoder(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: true, HardwareEncoder: "amf"},
		"movie.mkv",
		"segment-%05d.ts",
		"index.m3u8",
		"",
		0,
	)
	if len(profiles) < 2 {
		t.Fatalf("expected hardware profiles plus software fallback, got %d", len(profiles))
	}
	if profiles[0].Name != "h264_amf" {
		t.Fatalf("first profile = %q, want h264_amf", profiles[0].Name)
	}
}

func TestBuildTranscodeProfilesCanForceSoftwarePreference(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: true, HardwareEncoder: "software"},
		"movie.mkv",
		"segment-%05d.ts",
		"index.m3u8",
		"",
		0,
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	if profiles[0].Name != "libx264" {
		t.Fatalf("profile name = %q, want libx264", profiles[0].Name)
	}
}

func TestBuildTranscodeProfilesKeepsSessionOutputsRelativeToCmdDir(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		`D:\movie.mkv`,
		"segment-%05d.ts",
		"index.m3u8",
		"",
		0,
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	args := strings.Join(profiles[0].Args, " ")
	if strings.Contains(args, `runtime\cache\playback-sessions`) {
		t.Fatalf("expected HLS output args to stay relative to cmd.Dir, got %q", args)
	}
	if !strings.Contains(args, "segment-%05d.ts") || !strings.Contains(args, "index.m3u8") {
		t.Fatalf("expected relative HLS output names in args, got %q", args)
	}
	if !strings.Contains(args, "-hls_init_time 2") || !strings.Contains(args, "-hls_time 4") {
		t.Fatalf("expected balanced HLS segment timings in args, got %q", args)
	}
	if !strings.Contains(args, "-force_key_frames expr:gte(t,n_forced*4)") {
		t.Fatalf("expected 4-second keyframe cadence in args, got %q", args)
	}
	if !strings.Contains(args, "-pix_fmt yuv420p") {
		t.Fatalf("expected browser-safe pixel format in args, got %q", args)
	}
	if !strings.Contains(args, "-hls_allow_cache 0") || !strings.Contains(args, "-start_number 0") {
		t.Fatalf("expected deterministic HLS playlist flags in args, got %q", args)
	}
}

func TestBuildTranscodeProfilesUsesHybridSeekWindowWhenRequested(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.ts",
		"index.m3u8",
		"",
		3723.5,
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	args := profiles[0].Args
	seekIndices := make([]int, 0, 2)
	inputIndex := -1
	for idx, arg := range args {
		switch arg {
		case "-i":
			inputIndex = idx
		case "-ss":
			seekIndices = append(seekIndices, idx)
		}
	}
	if inputIndex < 0 || len(seekIndices) != 2 {
		t.Fatalf("expected hybrid input/output seek args, got %q", strings.Join(args, " "))
	}
	if seekIndices[0] >= inputIndex {
		t.Fatalf("expected fast seek before input, got %q", strings.Join(args, " "))
	}
	if seekIndices[1] <= inputIndex {
		t.Fatalf("expected precise seek after input, got %q", strings.Join(args, " "))
	}
	if seekIndices[0]+1 >= len(args) || args[seekIndices[0]+1] != "3721.500" {
		t.Fatalf("expected input seek offset 3721.500 in ffmpeg args, got %q", strings.Join(args, " "))
	}
	if seekIndices[1]+1 >= len(args) || args[seekIndices[1]+1] != "2.000" {
		t.Fatalf("expected accurate seek offset 2.000 in ffmpeg args, got %q", strings.Join(args, " "))
	}
}

func TestTakeSessionsForMovieRemovesOnlyMatchingMovieSessions(t *testing.T) {
	t.Parallel()
	manager := New(Config{})
	manager.sessions["sess-a"] = &sessionState{session: Session{ID: "sess-a", MovieID: "movie-a"}}
	manager.sessions["sess-b"] = &sessionState{session: Session{ID: "sess-b", MovieID: "movie-a"}}
	manager.sessions["sess-c"] = &sessionState{session: Session{ID: "sess-c", MovieID: "movie-b"}}

	stale := manager.takeSessionsForMovie("movie-a")
	if len(stale) != 2 {
		t.Fatalf("stale session count = %d, want 2", len(stale))
	}
	if _, ok := manager.sessions["sess-c"]; !ok {
		t.Fatal("expected unrelated movie session to remain registered")
	}
	if _, ok := manager.sessions["sess-a"]; ok {
		t.Fatal("expected sess-a to be removed")
	}
	if _, ok := manager.sessions["sess-b"]; ok {
		t.Fatal("expected sess-b to be removed")
	}
}

func TestManagerCloseClearsAllSessions(t *testing.T) {
	t.Parallel()
	manager := New(Config{})
	manager.sessions["sess-a"] = &sessionState{session: Session{ID: "sess-a", MovieID: "movie-a"}}
	manager.sessions["sess-b"] = &sessionState{session: Session{ID: "sess-b", MovieID: "movie-b"}}

	manager.Close()

	if len(manager.sessions) != 0 {
		t.Fatalf("session count = %d, want 0", len(manager.sessions))
	}
}

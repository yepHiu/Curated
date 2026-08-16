package playback

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBuildTranscodeProfilesFiltersUnavailableHardwareEncoders(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		t.Skip("hardware encoder filtering is only asserted on windows and darwin")
	}

	specs := hardwareEncoderSpecs()
	if len(specs) == 0 {
		t.Fatal("expected platform hardware encoder specs")
	}
	availability := make(map[string]bool, len(specs))
	for idx, spec := range specs {
		availability[spec.Name] = idx == 0
	}

	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: true},
		"movie.mkv",
		"segment-%05d.ts",
		"index.m3u8",
		buildProfileOptions{EncoderAvailability: availability},
	)
	names := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		names = append(names, profile.Name)
	}
	joined := strings.Join(names, ",")
	if strings.Contains(joined, specs[1].Name) {
		t.Fatalf("expected unavailable encoder %q to be filtered, got %q", specs[1].Name, joined)
	}
	if !strings.Contains(joined, specs[0].Name) {
		t.Fatalf("expected available encoder %q to stay in the chain, got %q", specs[0].Name, joined)
	}
	if names[len(names)-1] != "libx264" {
		t.Fatalf("last profile = %q, want libx264 fallback", names[len(names)-1])
	}
}

func TestBuildTranscodeProfilesKeepsAllEncodersWhenCapabilityUnknown(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: true},
		"movie.mkv",
		"segment-%05d.ts",
		"index.m3u8",
		buildProfileOptions{EncoderAvailability: nil},
	)
	if len(profiles) < 2 {
		t.Fatalf("expected hardware candidates plus fallback when capability is unknown, got %d", len(profiles))
	}
}

func TestEncoderAvailabilitySnapshotCachesPerCommand(t *testing.T) {
	if len(hardwareEncoderSpecs()) == 0 {
		t.Skip("no hardware encoder candidates on this platform")
	}

	restore := probeEncoderRuntimeFunc
	defer func() { probeEncoderRuntimeFunc = restore }()

	manager := New(Config{})
	t.Cleanup(manager.Close)

	calls := make(chan string, len(hardwareEncoderSpecs())*2)
	probeEncoderRuntimeFunc = func(cmdName string, encoder string) bool {
		calls <- encoder
		return encoder == hardwareEncoderSpecs()[0].Name
	}

	if snapshot := manager.encoderAvailabilitySnapshot("ffmpeg-one", 0); snapshot != nil {
		t.Fatalf("expected nil snapshot before the probe finishes, got %v", snapshot)
	}

	deadline := time.After(2 * time.Second)
	for {
		if snapshot := manager.encoderAvailabilitySnapshot("ffmpeg-one", 2*time.Second); snapshot != nil {
			if !snapshot[hardwareEncoderSpecs()[0].Name] {
				t.Fatalf("expected first encoder to be available, got %v", snapshot)
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for encoder probe to finish")
		case <-time.After(20 * time.Millisecond):
		}
	}

	// A finished probe must be reused without re-running the test encodes.
	probeEncoderRuntimeFunc = func(cmdName string, encoder string) bool {
		t.Fatalf("unexpected re-probe of %q", encoder)
		return false
	}
	if snapshot := manager.encoderAvailabilitySnapshot("ffmpeg-one", 2*time.Second); snapshot == nil {
		t.Fatal("expected cached snapshot without re-probing")
	}

	// A different ffmpeg command gets its own probe state.
	probeEncoderRuntimeFunc = func(cmdName string, encoder string) bool { return false }
	if snapshot := manager.encoderAvailabilitySnapshot("ffmpeg-two", 0); snapshot != nil {
		t.Fatalf("expected separate in-flight probe for the second command, got %v", snapshot)
	}
}

func TestEncoderListedMatchesExactEncoderField(t *testing.T) {
	sample := []byte(`Encoders:
 V.....D libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (codec h264)
 V.....D h264_nvenc           NVIDIA NVENC H.264 encoder (codec h264)
 V....D  h264_qsv             H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (Intel Quick Sync Video) (codec h264)
 A.....D aac                  AAC (Advanced Audio Coding)
`)
	if !encoderListed(sample, "h264_nvenc") {
		t.Fatal("expected h264_nvenc to be listed")
	}
	if !encoderListed(sample, "libx264") {
		t.Fatal("expected libx264 to be listed")
	}
	if encoderListed(sample, "h264_vaapi") {
		t.Fatal("encoder name must match the name field, not descriptions")
	}
	if encoderListed(sample, "nvenc") {
		t.Fatal("partial names must not match")
	}
}

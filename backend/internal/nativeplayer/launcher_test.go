package nativeplayer

import "testing"

func TestNormalizePresetInfersFromCommand(t *testing.T) {
	t.Parallel()

	if got := NormalizePreset("", `C:\Program Files\DAUM\PotPlayer\PotPlayerMini64.exe`); got != "potplayer" {
		t.Fatalf("NormalizePreset inferred %q, want potplayer", got)
	}
	if got := NormalizePreset("", "custom-launcher.exe"); got != "custom" {
		t.Fatalf("NormalizePreset inferred %q, want custom", got)
	}
}

func TestBuildPresetArgsForPotPlayerUsesSeekFlag(t *testing.T) {
	t.Parallel()

	args := buildPresetArgs("potplayer", 3723.5, "ignored")
	if len(args) != 1 || args[0] != "/seek=01:02:03.500" {
		t.Fatalf("buildPresetArgs returned %v, want PotPlayer seek arg", args)
	}
}

func TestBuildPresetArgsForMPVUsesStartAndTitle(t *testing.T) {
	t.Parallel()

	args := buildPresetArgs("mpv", 95.25, "ABCD-123")
	if len(args) != 2 {
		t.Fatalf("buildPresetArgs returned %d args, want 2", len(args))
	}
	if args[0] != "--start=95.250" {
		t.Fatalf("first arg = %q, want --start=95.250", args[0])
	}
	if args[1] != "--force-media-title=ABCD-123" {
		t.Fatalf("second arg = %q, want media title", args[1])
	}
}

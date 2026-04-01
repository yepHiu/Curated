package app

import (
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func TestChoosePlaybackDurationSecPrefersProbedDuration(t *testing.T) {
	t.Parallel()

	got := choosePlaybackDurationSec(7234.5, 7200, 4)
	if got != 7234.5 {
		t.Fatalf("choosePlaybackDurationSec() = %v, want 7234.5", got)
	}
}

func TestChoosePlaybackDurationSecFallsBackToLargestSaneDuration(t *testing.T) {
	t.Parallel()

	got := choosePlaybackDurationSec(0, 7200, 4)
	if got != 7200 {
		t.Fatalf("choosePlaybackDurationSec() = %v, want 7200", got)
	}
}

func TestBuildDirectPlaybackDescriptorKeepsResolvedDurationAndClampsResume(t *testing.T) {
	t.Parallel()

	dto := buildDirectPlaybackDescriptor(
		"movie-1",
		contracts.MovieDetailDTO{
			MovieListItemDTO: contracts.MovieListItemDTO{
				Location: `D:\media\movie-1.mkv`,
			},
		},
		&storage.PlaybackProgressRow{
			PositionSec: 10800,
			DurationSec: 4,
		},
		7200,
	)

	if dto.DurationSec != 7200 {
		t.Fatalf("durationSec = %v, want 7200", dto.DurationSec)
	}
	if dto.ResumePositionSec != 7200 {
		t.Fatalf("resumePositionSec = %v, want 7200", dto.ResumePositionSec)
	}
}

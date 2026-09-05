package app

import "math"

func normalizePlaybackStart(requested *float64, saved, duration float64) float64 {
	position := saved
	if requested != nil {
		position = *requested
	}
	if math.IsNaN(position) || math.IsInf(position, 0) || position < 0 {
		return 0
	}
	// Match the player's existing replay-near-end rule before launching FFmpeg.
	if duration > 0 && position >= duration*0.95 {
		return 0
	}
	return position
}

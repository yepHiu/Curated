package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const defaultPlaybackWatchTimeDays = 91

func clampPlaybackWatchTimeDays(raw string) int {
	days, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || days <= 0 {
		return defaultPlaybackWatchTimeDays
	}
	if days > defaultPlaybackWatchTimeDays {
		return defaultPlaybackWatchTimeDays
	}
	return days
}

func buildPlaybackWatchTimeDailyDTO(rows []storage.PlaybackWatchTimeDailyRow) contracts.PlaybackWatchTimeDailyListDTO {
	items := make([]contracts.PlaybackWatchTimeDailyItemDTO, 0, len(rows))
	daySet := make(map[string]float64, len(rows))
	dto := contracts.PlaybackWatchTimeDailyListDTO{}

	for _, row := range rows {
		if row.WatchedSec <= 0 {
			continue
		}
		items = append(items, contracts.PlaybackWatchTimeDailyItemDTO{
			DayKey:     row.DayKey,
			WatchedSec: row.WatchedSec,
		})
		daySet[row.DayKey] = row.WatchedSec
		dto.TotalWatchedSec += row.WatchedSec
		if row.WatchedSec > dto.MaxDayWatchedSec {
			dto.MaxDayWatchedSec = row.WatchedSec
		}
	}

	dto.Items = items
	dto.ActiveDays = len(items)
	dto.LongestStreakDays = longestPlaybackWatchTimeStreak(daySet)
	return dto
}

func longestPlaybackWatchTimeStreak(daySet map[string]float64) int {
	if len(daySet) == 0 {
		return 0
	}
	keys := make([]string, 0, len(daySet))
	for key := range daySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	longest := 0
	current := 0
	var prev time.Time
	for _, key := range keys {
		day, err := time.Parse("2006-01-02", key)
		if err != nil {
			current = 0
			continue
		}
		if current == 0 || day.Sub(prev) == 24*time.Hour {
			current++
		} else {
			current = 1
		}
		if current > longest {
			longest = current
		}
		prev = day
	}
	return longest
}

func (h *Handler) handleListPlaybackWatchTimeDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	days := clampPlaybackWatchTimeDays(r.URL.Query().Get("days"))
	rows, err := h.store.ListPlaybackWatchTimeDaily(r.Context(), days)
	if err != nil {
		h.logger.Error("list playback watch time daily", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list watch time")
		return
	}
	writeJSON(w, http.StatusOK, buildPlaybackWatchTimeDailyDTO(rows))
}

func (h *Handler) handlePostPlaybackWatchTimeDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}
	var req contracts.AddPlaybackWatchTimeBody
	if err := json.Unmarshal(body, &req); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	if err := h.store.AddPlaybackWatchTime(
		r.Context(),
		strings.TrimSpace(req.MovieID),
		strings.TrimSpace(req.DayKey),
		req.WatchedSec,
	); err != nil {
		switch {
		case errors.Is(err, storage.ErrPlaybackWatchTimeInvalidDayKey),
			errors.Is(err, storage.ErrPlaybackWatchTimeInvalidSeconds):
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		case errors.Is(err, storage.ErrPlaybackWatchTimeMovieNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
		default:
			h.logger.Error("add playback watch time", zap.Error(err))
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save watch time")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

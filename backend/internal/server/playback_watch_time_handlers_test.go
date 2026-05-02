package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func newWatchTimeTestServer(t *testing.T) (*storage.SQLiteStore, *httptest.Server) {
	t.Helper()
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "watch-time.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
	}).Routes())
	t.Cleanup(srv.Close)
	return store, srv
}

func addWatchTimeHandlerMovieForTest(t *testing.T, store *storage.SQLiteStore, code string) string {
	t.Helper()
	outcome, err := store.PersistScanMovie(context.Background(), contracts.ScanFileResultDTO{
		TaskID:   "watch-time-handler-test",
		Path:     filepath.Join(t.TempDir(), code+".mp4"),
		FileName: code + ".mp4",
		Number:   code,
	})
	if err != nil {
		t.Fatal(err)
	}
	return outcome.MovieID
}

func TestPlaybackWatchTimeDailyHandlers(t *testing.T) {
	t.Parallel()
	store, srv := newWatchTimeTestServer(t)
	movieID := addWatchTimeHandlerMovieForTest(t, store, "WT-HANDLER")
	today := time.Now().UTC()
	day0 := today.Format("2006-01-02")
	day1 := today.AddDate(0, 0, -1).Format("2006-01-02")

	for _, body := range []string{
		`{"movieId":` + strconvQuote(movieID) + `,"dayKey":` + strconvQuote(day1) + `,"watchedSec":120}`,
		`{"movieId":` + strconvQuote(movieID) + `,"dayKey":` + strconvQuote(day1) + `,"watchedSec":30}`,
		`{"movieId":` + strconvQuote(movieID) + `,"dayKey":` + strconvQuote(day0) + `,"watchedSec":240}`,
	} {
		resp, err := http.Post(srv.URL+"/api/playback/watch-time/daily", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("POST status = %d, want 204", resp.StatusCode)
		}
	}

	resp, err := http.Get(srv.URL + "/api/playback/watch-time/daily")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", resp.StatusCode)
	}

	var dto contracts.PlaybackWatchTimeDailyListDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.TotalWatchedSec != 390 {
		t.Fatalf("totalWatchedSec = %v, want 390", dto.TotalWatchedSec)
	}
	if dto.ActiveDays != 2 {
		t.Fatalf("activeDays = %d, want 2", dto.ActiveDays)
	}
	if dto.MaxDayWatchedSec != 240 {
		t.Fatalf("maxDayWatchedSec = %v, want 240", dto.MaxDayWatchedSec)
	}
	if dto.LongestStreakDays != 2 {
		t.Fatalf("longestStreakDays = %d, want 2", dto.LongestStreakDays)
	}
	if len(dto.Items) != 2 || dto.Items[0].DayKey != day0 || dto.Items[0].WatchedSec != 240 {
		t.Fatalf("items = %+v", dto.Items)
	}
}

func TestPlaybackWatchTimeDailyPostRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	_, srv := newWatchTimeTestServer(t)

	resp, err := http.Post(
		srv.URL+"/api/playback/watch-time/daily",
		"application/json",
		bytes.NewBufferString(`{"movieId":"missing","dayKey":"20260430","watchedSec":0}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func strconvQuote(value string) string {
	b, _ := json.Marshal(value)
	return string(b)
}

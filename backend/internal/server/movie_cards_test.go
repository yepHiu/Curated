package server

import (
	"context"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"encoding/json"
	"go.uber.org/zap"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHomepageCardsOptInAndBatchLimit(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "cards.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	movie, err := store.PersistScanMovie(context.Background(), contracts.ScanFileResultDTO{Path: filepath.Join(t.TempDir(), "ABC-123.mp4"), FileName: "ABC-123.mp4", Number: "ABC-123"})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{Cfg: config.Config{}, Store: store, Logger: zap.NewNop(), HomepageRecommendations: stubHomepageRecommendationsProvider{dto: contracts.HomepageDailyRecommendationsDTO{HeroMovieIDs: []string{movie.MovieID}, RecommendationMovieIDs: []string{movie.MovieID, "gone"}}}})
	for _, query := range []string{"", "?include=cards"} {
		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, httptest.NewRequest("GET", "/api/homepage/recommendations"+query, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", query, w.Code, w.Body.String())
		}
		var result map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		_, hasCards := result["cards"]
		if hasCards != (query != "") {
			t.Fatalf("opt in: %s", w.Body.String())
		}
		if hasCards {
			var cards []contracts.MovieCardDTO
			json.Unmarshal(result["cards"], &cards)
			if len(cards) != 1 || cards[0].ID != movie.MovieID {
				t.Fatalf("cards: %s", result["cards"])
			}
			if strings.Contains(string(result["cards"]), "previewImages") || strings.Contains(string(result["cards"]), "location") {
				t.Fatal("detail fields leaked")
			}
		}
	}
	for _, body := range []string{`{"movieIds":[]}`, `{"movieIds":["missing"]} {}`, `{"movieIds":[` + strings.Repeat(`"x",`, 50) + `"x"]}`} {
		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, httptest.NewRequest("POST", "/api/library/movies/summaries", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatalf("expected 400 got %d", w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, httptest.NewRequest("POST", "/api/library/movies/summaries", strings.NewReader(`{"movieIds":["missing"]}`)))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[]`) {
		t.Fatalf("missing response: %d %s", w.Code, w.Body.String())
	}
}

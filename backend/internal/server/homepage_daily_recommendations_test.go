package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

type stubHomepageRecommendationsProvider struct {
	dto         contracts.HomepageDailyRecommendationsDTO
	forcedCount int
}

func (s stubHomepageRecommendationsProvider) GetOrCreateHomepageDailyRecommendations(ctx context.Context, dateUTC string) (contracts.HomepageDailyRecommendationsDTO, error) {
	_ = ctx
	_ = dateUTC
	return s.dto, nil
}

func (s stubHomepageRecommendationsProvider) RegenerateHomepageDailyRecommendations(ctx context.Context, dateUTC string) (contracts.HomepageDailyRecommendationsDTO, error) {
	_ = ctx
	_ = dateUTC
	s.forcedCount++
	return s.dto, nil
}

func TestHomepageRecommendationsReturnsPersistedSnapshot(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		HomepageRecommendations: stubHomepageRecommendationsProvider{
			dto: contracts.HomepageDailyRecommendationsDTO{
				DateUTC:                "2026-04-15",
				GeneratedAt:            "2026-04-15T00:00:00Z",
				GenerationVersion:      "v1",
				HeroMovieIDs:           []string{"m01", "m02", "m03", "m04", "m05", "m06", "m07", "m08"},
				RecommendationMovieIDs: []string{"m09", "m10", "m11", "m12", "m13", "m14"},
			},
		},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/homepage/recommendations")
	if err != nil {
		t.Fatalf("GET homepage recommendations: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}

	var dto struct {
		DateUTC string   `json:"dateUtc"`
		Hero    []string `json:"heroMovieIds"`
		Reco    []string `json:"recommendationMovieIds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if dto.DateUTC != "2026-04-15" {
		t.Fatalf("DateUTC = %q, want %q", dto.DateUTC, "2026-04-15")
	}
	if len(dto.Hero) != 8 || len(dto.Reco) != 6 {
		t.Fatalf("hero=%d reco=%d", len(dto.Hero), len(dto.Reco))
	}
}

func TestHomepageRecommendationsRefreshRegeneratesSnapshot(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		HomepageRecommendations: stubHomepageRecommendationsProvider{
			dto: contracts.HomepageDailyRecommendationsDTO{
				DateUTC:                "2026-04-16",
				GeneratedAt:            "2026-04-16T00:00:00Z",
				GenerationVersion:      "v3",
				HeroMovieIDs:           []string{"m01", "m02", "m03", "m04", "m05", "m06", "m07", "m08"},
				RecommendationMovieIDs: []string{"m09", "m10", "m11", "m12", "m13", "m14"},
			},
		},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/homepage/recommendations/refresh", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST homepage recommendations refresh: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}

	var dto struct {
		DateUTC           string `json:"dateUtc"`
		GenerationVersion string `json:"generationVersion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if dto.DateUTC != "2026-04-16" {
		t.Fatalf("DateUTC = %q, want %q", dto.DateUTC, "2026-04-16")
	}
	if dto.GenerationVersion != "v3" {
		t.Fatalf("GenerationVersion = %q, want %q", dto.GenerationVersion, "v3")
	}
}

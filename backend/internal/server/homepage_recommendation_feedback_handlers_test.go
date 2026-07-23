package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
)

type stubRecommendationFeedbackProvider struct {
	items []contracts.RecommendationFeedbackDTO
}

func (s *stubRecommendationFeedbackProvider) ListHomepageRecommendationFeedback(context.Context) (contracts.RecommendationFeedbackListDTO, error) {
	return contracts.RecommendationFeedbackListDTO{Items: append([]contracts.RecommendationFeedbackDTO(nil), s.items...)}, nil
}

func (s *stubRecommendationFeedbackProvider) CreateHomepageRecommendationFeedback(_ context.Context, body contracts.CreateRecommendationFeedbackBody) (contracts.RecommendationFeedbackDTO, error) {
	if body.TargetValue == "missing" {
		return contracts.RecommendationFeedbackDTO{}, contracts.ErrRecommendationFeedbackTargetNotFound
	}
	item := contracts.RecommendationFeedbackDTO{
		ID:            "feedback_1",
		Action:        body.Action,
		TargetType:    body.TargetType,
		TargetValue:   body.TargetValue,
		SourceMovieID: body.SourceMovieID,
		CreatedAt:     "2026-07-21T00:00:00Z",
		UpdatedAt:     "2026-07-21T00:00:00Z",
	}
	s.items = append(s.items, item)
	return item, nil
}

func (s *stubRecommendationFeedbackProvider) DeleteHomepageRecommendationFeedback(_ context.Context, id string) error {
	if id != "feedback_1" {
		return contracts.ErrRecommendationFeedbackTargetNotFound
	}
	s.items = nil
	return nil
}

func TestHomepageRecommendationFeedbackHandlersLifecycleAndErrors(t *testing.T) {
	t.Parallel()
	provider := &stubRecommendationFeedbackProvider{}
	h := NewHandler(Deps{Logger: zap.NewNop(), HomepageRecommendationFeedback: provider})
	server := httptest.NewServer(h.Routes())
	t.Cleanup(server.Close)

	response, err := http.Post(
		server.URL+"/api/homepage/recommendations/feedback",
		"application/json",
		strings.NewReader(`{"action":"less","targetType":"actor","targetValue":"Actor A","sourceMovieId":"m01"}`),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", response.StatusCode)
	}

	listResponse, err := http.Get(server.URL + "/api/homepage/recommendations/feedback")
	if err != nil {
		t.Fatalf("list request: %v", err)
	}
	defer listResponse.Body.Close()
	var list contracts.RecommendationFeedbackListDTO
	if err := json.NewDecoder(listResponse.Body).Decode(&list); err != nil || len(list.Items) != 1 {
		t.Fatalf("list = %#v, err=%v", list, err)
	}

	request, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/homepage/recommendations/feedback/feedback_1", nil)
	deleteResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	defer deleteResponse.Body.Close()
	if deleteResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", deleteResponse.StatusCode)
	}

	missing, err := http.Post(
		server.URL+"/api/homepage/recommendations/feedback",
		"application/json",
		strings.NewReader(`{"action":"less","targetType":"actor","targetValue":"missing","sourceMovieId":"m01"}`),
	)
	if err != nil {
		t.Fatalf("missing request: %v", err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status = %d", missing.StatusCode)
	}
	var appError contracts.AppError
	if err := json.NewDecoder(missing.Body).Decode(&appError); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if appError.Code != contracts.ErrorCodeRecommendationFeedbackTargetNotFound {
		t.Fatalf("error = %#v", appError)
	}

	strict, err := http.Post(
		server.URL+"/api/homepage/recommendations/feedback",
		"application/json",
		strings.NewReader(`{"action":"less","targetType":"actor","targetValue":"Actor A","sourceMovieId":"m01","unknown":true}`),
	)
	if err != nil {
		t.Fatalf("strict request: %v", err)
	}
	defer strict.Body.Close()
	if strict.StatusCode != http.StatusBadRequest {
		t.Fatalf("strict status = %d", strict.StatusCode)
	}
}

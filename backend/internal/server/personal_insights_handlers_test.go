package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"curated-backend/internal/contracts"
)

type stubPersonalInsightsProvider struct {
	overview           contracts.PersonalInsightsOverviewDTO
	overviewErr        error
	breakdown          contracts.PersonalInsightsBreakdownDTO
	breakdownErr       error
	overviewRange      string
	overviewTimezone   string
	breakdownRange     string
	breakdownTimezone  string
	breakdownDimension string
	breakdownLimit     int
	breakdownCalls     int
}

func (s *stubPersonalInsightsProvider) GetPersonalInsightsOverview(
	_ context.Context,
	rangeValue string,
	timezone string,
) (contracts.PersonalInsightsOverviewDTO, error) {
	s.overviewRange = rangeValue
	s.overviewTimezone = timezone
	return s.overview, s.overviewErr
}

func (s *stubPersonalInsightsProvider) GetPersonalInsightsBreakdown(
	_ context.Context,
	rangeValue string,
	timezone string,
	dimension string,
	limit int,
) (contracts.PersonalInsightsBreakdownDTO, error) {
	s.breakdownCalls++
	s.breakdownRange = rangeValue
	s.breakdownTimezone = timezone
	s.breakdownDimension = dimension
	s.breakdownLimit = limit
	return s.breakdown, s.breakdownErr
}

func TestPersonalInsightsHandlersReturnOverviewAndBreakdown(t *testing.T) {
	provider := &stubPersonalInsightsProvider{
		overview: contracts.PersonalInsightsOverviewDTO{
			Range:          contracts.PersonalInsightsRange30Days,
			Timezone:       "Asia/Shanghai",
			WatchedSeconds: 7200,
			StartedMovies:  4,
		},
		breakdown: contracts.PersonalInsightsBreakdownDTO{
			Range:       contracts.PersonalInsightsRange90Days,
			Dimension:   contracts.PersonalInsightsDimensionActor,
			Timezone:    "UTC",
			Attribution: "full-per-entity",
			Limit:       7,
			Items: []contracts.PersonalInsightsBreakdownItemDTO{
				{Name: "Actor A", WatchedSeconds: 3600, MovieCount: 2, ShareOfTotal: 0.5},
			},
		},
	}
	handler := NewHandler(Deps{PersonalInsightsProvider: provider}).Routes()

	overviewRequest := httptest.NewRequest(http.MethodGet, "/api/insights/overview?range=30d&timezone=Asia%2FShanghai", http.NoBody)
	overviewRecorder := httptest.NewRecorder()
	handler.ServeHTTP(overviewRecorder, overviewRequest)
	if overviewRecorder.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", overviewRecorder.Code, overviewRecorder.Body.String())
	}
	if provider.overviewRange != "30d" || provider.overviewTimezone != "Asia/Shanghai" {
		t.Fatalf("overview query mismatch: range=%q timezone=%q", provider.overviewRange, provider.overviewTimezone)
	}
	var overview contracts.PersonalInsightsOverviewDTO
	if err := json.Unmarshal(overviewRecorder.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if overview.WatchedSeconds != 7200 || overview.StartedMovies != 4 {
		t.Fatalf("unexpected overview response: %+v", overview)
	}

	breakdownRequest := httptest.NewRequest(http.MethodGet, "/api/insights/breakdown?range=90d&timezone=UTC&dimension=actor&limit=7", http.NoBody)
	breakdownRecorder := httptest.NewRecorder()
	handler.ServeHTTP(breakdownRecorder, breakdownRequest)
	if breakdownRecorder.Code != http.StatusOK {
		t.Fatalf("breakdown status=%d body=%s", breakdownRecorder.Code, breakdownRecorder.Body.String())
	}
	if provider.breakdownRange != "90d" || provider.breakdownTimezone != "UTC" || provider.breakdownDimension != "actor" || provider.breakdownLimit != 7 {
		t.Fatalf("breakdown query mismatch: range=%q timezone=%q dimension=%q limit=%d", provider.breakdownRange, provider.breakdownTimezone, provider.breakdownDimension, provider.breakdownLimit)
	}
	var breakdown contracts.PersonalInsightsBreakdownDTO
	if err := json.Unmarshal(breakdownRecorder.Body.Bytes(), &breakdown); err != nil {
		t.Fatal(err)
	}
	if len(breakdown.Items) != 1 || breakdown.Items[0].Name != "Actor A" {
		t.Fatalf("unexpected breakdown response: %+v", breakdown)
	}
}

func TestPersonalInsightsHandlersRejectMalformedLimitBeforeProviderCall(t *testing.T) {
	provider := &stubPersonalInsightsProvider{}
	handler := NewHandler(Deps{PersonalInsightsProvider: provider}).Routes()
	request := httptest.NewRequest(http.MethodGet, "/api/insights/breakdown?range=30d&timezone=UTC&dimension=tag&limit=ten", http.NoBody)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assertPersonalInsightsErrorCode(t, recorder, http.StatusBadRequest, contracts.ErrorCodePersonalInsightsInvalidLimit)
	if provider.breakdownCalls != 0 {
		t.Fatalf("provider called %d times for malformed limit", provider.breakdownCalls)
	}
}

func TestPersonalInsightsHandlersMapStableProviderErrors(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		err       error
		code      string
		breakdown bool
	}{
		{name: "range", path: "/api/insights/overview?range=7d&timezone=UTC", err: contracts.ErrPersonalInsightsInvalidRange, code: contracts.ErrorCodePersonalInsightsInvalidRange},
		{name: "timezone", path: "/api/insights/overview?range=30d&timezone=Mars%2FOlympus", err: contracts.ErrPersonalInsightsInvalidTimezone, code: contracts.ErrorCodePersonalInsightsInvalidTimezone},
		{name: "dimension", path: "/api/insights/breakdown?range=30d&timezone=UTC&dimension=director", err: contracts.ErrPersonalInsightsInvalidDimension, code: contracts.ErrorCodePersonalInsightsInvalidDimension, breakdown: true},
		{name: "limit", path: "/api/insights/breakdown?range=30d&timezone=UTC&dimension=actor&limit=26", err: contracts.ErrPersonalInsightsInvalidLimit, code: contracts.ErrorCodePersonalInsightsInvalidLimit, breakdown: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &stubPersonalInsightsProvider{}
			if test.breakdown {
				provider.breakdownErr = test.err
			} else {
				provider.overviewErr = test.err
			}
			handler := NewHandler(Deps{PersonalInsightsProvider: provider}).Routes()
			request := httptest.NewRequest(http.MethodGet, test.path, http.NoBody)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			assertPersonalInsightsErrorCode(t, recorder, http.StatusBadRequest, test.code)
		})
	}
}

func TestPersonalInsightsHandlersReturnInternalErrorWithoutProvider(t *testing.T) {
	handler := NewHandler(Deps{}).Routes()
	for _, path := range []string{
		"/api/insights/overview?range=30d&timezone=UTC",
		"/api/insights/breakdown?range=30d&timezone=UTC&dimension=actor",
	} {
		request := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		assertPersonalInsightsErrorCode(t, recorder, http.StatusInternalServerError, contracts.ErrorCodeInternal)
	}

	provider := &stubPersonalInsightsProvider{overviewErr: errors.New("database unavailable")}
	configured := NewHandler(Deps{PersonalInsightsProvider: provider}).Routes()
	request := httptest.NewRequest(http.MethodGet, "/api/insights/overview?range=30d&timezone=UTC", http.NoBody)
	recorder := httptest.NewRecorder()
	configured.ServeHTTP(recorder, request)
	assertPersonalInsightsErrorCode(t, recorder, http.StatusInternalServerError, contracts.ErrorCodeInternal)
}

func assertPersonalInsightsErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, status, recorder.Body.String())
	}
	var response contracts.AppError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != code {
		t.Fatalf("error code=%q want=%q body=%s", response.Code, code, recorder.Body.String())
	}
}

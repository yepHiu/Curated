package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

type nativeResolverStub struct {
	stubPlaybackResolver
	starts int
	plans  int
	start  *float64
}

func (s *nativeResolverStub) PlanNativePlayback(context.Context, string) (contracts.NativePlaybackPlanDTO, error) {
	s.plans++
	return contracts.NativePlaybackPlanDTO{FileID: "part-2"}, nil
}
func (s *nativeResolverStub) ResolveNativePlayback(_ context.Context, _ string, _ contracts.NativePlaybackSelection, start *float64) (contracts.PlaybackDescriptorDTO, error) {
	s.starts++
	s.start = start
	return contracts.PlaybackDescriptorDTO{FileID: "part-2", Mode: "direct"}, nil
}
func TestNativePlaybackContract(t *testing.T) {
	token := strings.Repeat("a", 64)
	for _, tc := range []struct {
		body                  string
		status, plans, starts int
	}{
		{`{"mode":"plan"}`, 200, 1, 0},
		{`{"mode":"auto","nativePlayback":{"version":1,"sourceToken":"` + token + `","directPlay":true}}`, 201, 0, 1},
		{`{"mode":"auto"}`, 400, 0, 0},
		{`{"mode":"auto","nativePlayback":{"version":2,"sourceToken":"` + token + `"}}`, 400, 0, 0},
		{`{"mode":"auto","autoStartPositionSec":-1,"nativePlayback":{"version":1,"sourceToken":"` + token + `"}}`, 400, 0, 0},
		{`{"mode":"auto","nativePlayback":{"version":1,"sourceToken":"bad"}}`, 400, 0, 0},
		{`{"mode":"plan"}{}`, 400, 0, 0},
		{`{"mode":"plan","padding":"` + strings.Repeat("x", 17000) + `"}`, 400, 0, 0},
		{`{"nativePlayback":{"version":1}}`, 400, 0, 0},
	} {
		stub := &nativeResolverStub{}
		h := NewHandler(Deps{Cfg: config.Config{}, PlaybackResolver: stub})
		r := httptest.NewRecorder()
		h.Routes().ServeHTTP(r, httptest.NewRequest("POST", "/api/library/movies/movie/playback-session", strings.NewReader(tc.body)))
		if r.Code != tc.status || stub.plans != tc.plans || stub.starts != tc.starts {
			t.Fatalf("%s: status=%d plan=%d start=%d %s", tc.body[:min(len(tc.body), 120)], r.Code, stub.plans, stub.starts, r.Body.String())
		}
		if stub.starts > 0 && stub.start != nil {
			t.Fatal("absent position must preserve resume")
		}
		if r.Code < 300 && r.Header().Get("Server-Timing") == "" {
			t.Fatal("missing timing")
		}
	}
}

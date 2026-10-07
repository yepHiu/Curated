package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMeasuredUsageObservations(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		stream, known bool
		status        int
		code          string
	}{
		{"stream tail", "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":3,\"total_tokens\":15}}\n\ndata: [DONE]\n\n", true, true, 200, ""},
		{"complete", `{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":12,"completion_tokens":3,"total_tokens":15}}`, false, true, 200, ""},
		{"missing", `{"choices":[{"message":{"content":"ok"}}]}`, false, false, 200, ""},
		{"incomplete", `{"choices":[{"message":{"content":"ok"}}],"usage":{"total_tokens":15}}`, false, false, 200, ""},
		{"rate limit", `secret raw body`, false, false, 429, "rate_limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.stream {
					var req chatCompletionRequest
					_ = json.NewDecoder(r.Body).Decode(&req)
					if req.StreamOpts == nil || !req.StreamOpts.IncludeUsage {
						t.Error("usage not requested")
					}
				}
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.payload)
			}))
			defer server.Close()
			c := NewClient(ClientConfig{BaseURL: server.URL, Model: "test"}, server.Client())
			count := 0
			c.Observe = func(o Observation) {
				count++
				if (o.Usage != nil) != tc.known || o.ErrorCode != tc.code {
					t.Errorf("observation %#v", o)
				}
				if tc.known && o.Usage.TotalTokens != 15 {
					t.Error("wrong usage")
				}
				if (o.FirstTextMs != nil) != tc.stream {
					t.Error("first text availability")
				}
			}
			if tc.stream {
				_, _ = c.StreamTurn(context.Background(), TurnRequest{}, nil)
			} else {
				_, _ = c.Complete(context.Background(), nil, 0)
			}
			if count != 1 {
				t.Fatalf("observed %d times", count)
			}
		})
	}
}

func TestInterruptedStreamKeepsTextAndRecordsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
	}))
	defer server.Close()
	c := NewClient(ClientConfig{BaseURL: server.URL, Model: "test"}, server.Client())
	var observed Observation
	c.Observe = func(o Observation) { observed = o }
	turn, err := c.StreamTurn(context.Background(), TurnRequest{}, nil)
	if err == nil || turn.Content != "partial" || observed.ErrorCode != "stream_interrupted" || observed.FirstTextMs == nil {
		t.Fatalf("turn %+v observation %+v err %v", turn, observed, err)
	}
}

func TestProviderCacheUsage(t *testing.T) {
	for _, tc := range []struct {
		name, extra string
		hit, miss   *int64
	}{
		{"deepseek", `,"prompt_cache_hit_tokens":8,"prompt_cache_miss_tokens":4`, cacheCount(8), cacheCount(4)},
		{"cold", `,"prompt_cache_hit_tokens":0,"prompt_cache_miss_tokens":12`, cacheCount(0), cacheCount(12)},
		{"openai", `,"prompt_tokens_details":{"cached_tokens":8}`, cacheCount(8), nil},
		{"unreported", "", nil, nil},
		{"negative", `,"prompt_cache_hit_tokens":-1`, nil, nil},
		{"oversized", `,"prompt_cache_hit_tokens":13`, nil, nil},
		{"inconsistent", `,"prompt_cache_hit_tokens":8,"prompt_cache_miss_tokens":8`, nil, nil},
	} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream=%v", tc.name, stream), func(t *testing.T) {
				usage := `{"prompt_tokens":12,"completion_tokens":3,"total_tokens":15` + tc.extra + `}`
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if stream {
						fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: {\"choices\":[],\"usage\":"+usage+"}\n\ndata: [DONE]\n\n")
					} else {
						fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}],"usage":`+usage+`}`)
					}
				}))
				defer server.Close()
				client := NewClient(ClientConfig{BaseURL: server.URL, Model: "test"}, server.Client())
				var observed Observation
				client.Observe = func(o Observation) { observed = o }
				var err error
				if stream {
					_, err = client.StreamTurn(context.Background(), TurnRequest{}, nil)
				} else {
					_, err = client.Complete(context.Background(), nil, 0)
				}
				if err != nil || observed.Usage == nil || observed.Usage.TotalTokens != 15 {
					t.Fatalf("missing usage: %+v, %v", observed, err)
				}
				for i, got := range []*int64{observed.Usage.CacheHitTokens, observed.Usage.CacheMissTokens} {
					want := []*int64{tc.hit, tc.miss}[i]
					if (got == nil) != (want == nil) || (got != nil && *got != *want) {
						t.Fatalf("cache counts = %+v", observed.Usage)
					}
				}
			})
		}
	}
}

func cacheCount(n int64) *int64 { return &n }

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

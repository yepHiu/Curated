package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompleteOptionalThinkingFlag(t *testing.T) {
	disabled := false
	for _, flag := range []*bool{nil, &disabled} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			v, ok := b["enable_thinking"]
			if (flag != nil) != ok || (ok && v != false) {
				t.Errorf("thinking flag=%v present=%v", v, ok)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
		}))
		c := NewClient(ClientConfig{BaseURL: srv.URL, Model: "model", EnableThinking: flag}, srv.Client())
		if _, e := c.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "test"}}, 32); e != nil {
			t.Fatal(e)
		}
		srv.Close()
	}
}

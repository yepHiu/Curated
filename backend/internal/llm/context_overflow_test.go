package llm

import "testing"

func TestContextOverflowClassification(t *testing.T) {
	for _, tc := range []struct {
		status int
		detail string
		want   bool
	}{
		{400, "context_length_exceeded", true}, {413, "prompt is too long", true}, {422, "maximum context length", true},
		{401, "context window", false}, {429, "too many input tokens", false}, {400, "invalid tool schema", false},
	} {
		if got := IsContextOverflow(&HTTPError{Status: tc.status, Detail: tc.detail}); got != tc.want {
			t.Fatalf("%+v: %v", tc, got)
		}
	}
}

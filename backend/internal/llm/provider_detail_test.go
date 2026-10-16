package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderHTTPDiagnostic(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"code":"DataInspectionFailed","message":"Rejected test-secret Bearer other-secret https://example.com/?token=private"},"request_id":"request-123","api_key":"hidden"}`))
	}))
	defer s.Close()
	_, err := NewClient(ClientConfig{BaseURL: s.URL, APIKey: "test-secret", Model: "test"}, s.Client()).Complete(context.Background(), nil, 10)
	var provider *HTTPError
	if !errors.As(err, &provider) {
		t.Fatalf("expected HTTP error, got %v", err)
	}
	for _, want := range []string{"HTTP 400", "DataInspectionFailed", "request-123"} {
		if !strings.Contains(provider.PublicDetail, want) {
			t.Fatalf("missing %s: %s", want, provider.PublicDetail)
		}
	}
	for _, secret := range []string{"test-secret", "other-secret", "private", "hidden", "https://"} {
		if strings.Contains(provider.PublicDetail, secret) {
			t.Fatalf("leaked %s", secret)
		}
	}
	if ErrorCategory(err) != "provider_http" {
		t.Fatal("category changed")
	}
}

func TestPublicProviderDetailBoundsAndEscapedKey(t *testing.T) {
	if got := publicProviderDetail(`{"message":"a\"b"}`, `a"b`); strings.Contains(got, `a\"b`) {
		t.Fatal("escaped key leaked")
	}
	got := publicProviderDetail(strings.Repeat("错", 5000), "")
	if len([]rune(got)) > 4120 || !strings.HasSuffix(got, "[truncated]") {
		t.Fatal("diagnostic not bounded")
	}
}

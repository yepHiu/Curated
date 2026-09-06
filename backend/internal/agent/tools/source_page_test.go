package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"curated-backend/internal/agent/core"
)

type stubSourcePage struct {
	text string
	err  error
	got  string
}

func (s *stubSourcePage) FetchSourcePage(_ context.Context, pageURL string, _ []string) (SourcePageContent, error) {
	s.got = pageURL
	if s.err != nil {
		return SourcePageContent{}, s.err
	}
	return SourcePageContent{Text: s.text, FinalURL: pageURL}, nil
}

func TestGetSourcePageRejectsUnknownAndNonHTTPS(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	gateway := core.NewGateway(reg, nil, nil, nil)
	pages := &stubSourcePage{text: "review"}
	if err := RegisterProviderTools(reg, stubQuery{}, stubProviderLookup{}, pages, gateway.MovieRefs(), gateway.ActorRefs(), gateway.SourceURLs()); err != nil {
		t.Fatal(err)
	}
	unknown := gateway.Invoke(context.Background(), core.Call{
		Name:      core.GetSourcePageName,
		Args:      json.RawMessage(`{"url":"https://www.javbus.com/ABC-123"}`),
		SessionID: "ses_1",
	})
	if unknown.OK {
		t.Fatalf("unknown url should fail: %+v", unknown)
	}

	gateway.RememberSourceURLs("ses_1", []string{"https://www.javbus.com/ABC-123"})
	httpURL := gateway.Invoke(context.Background(), core.Call{
		Name:      core.GetSourcePageName,
		Args:      json.RawMessage(`{"url":"http://www.javbus.com/ABC-123"}`),
		SessionID: "ses_1",
	})
	if httpURL.OK {
		t.Fatalf("http url should fail: %+v", httpURL)
	}

	ok := gateway.Invoke(context.Background(), core.Call{
		Name:      core.GetSourcePageName,
		Args:      json.RawMessage(`{"url":"https://www.javbus.com/ABC-123"}`),
		SessionID: "ses_1",
	})
	if !ok.OK {
		t.Fatalf("allowlisted url = %+v", ok)
	}
	raw, _ := json.Marshal(ok.Data)
	if !strings.Contains(string(raw), "review") || pages.got != "https://www.javbus.com/ABC-123" {
		t.Fatalf("missing extracted text: %s got=%q", raw, pages.got)
	}
}

func TestValidatePublicHTTPSURLRejectsPrivateHosts(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"http://example.test/x",
		"https://localhost/x",
		"https://127.0.0.1/x",
		"https://10.0.0.1/x",
		"https://192.168.1.1/x",
		"https://[::1]/x",
		"file:///etc/passwd",
	} {
		if err := ValidatePublicHTTPSURL(raw); err == nil {
			t.Fatalf("expected reject %s", raw)
		}
	}
	if err := ValidatePublicHTTPSURL("https://www.javbus.com/ABC-123"); err != nil {
		t.Fatalf("public host: %v", err)
	}
}

func TestExtractVisibleTextDropsScripts(t *testing.T) {
	t.Parallel()
	text, truncated := extractVisibleText(`<html><script>alert(1)</script><p>Score 4.2</p><style>p{}</style></html>`, 1024)
	if truncated || !strings.Contains(text, "Score 4.2") || strings.Contains(text, "alert") {
		t.Fatalf("text = %q truncated=%v", text, truncated)
	}
}

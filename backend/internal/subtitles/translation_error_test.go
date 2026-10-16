package subtitles

import (
	"fmt"
	"strings"
	"testing"

	"curated-backend/internal/llm"
)

func TestTranslationErrorPreservesOnlyPublicDiagnostic(t *testing.T) {
	err := &TranslationError{Category: "provider_http", Cause: &llm.HTTPError{Status: 400, Detail: "private raw body", PublicDetail: "HTTP 400: InvalidParameter"}}
	if strings.Contains(err.Error(), "private") {
		t.Fatal("raw body exposed")
	}
	if got := translationProviderDetail(fmt.Errorf("wrapped: %w", err)); got != "HTTP 400: InvalidParameter" {
		t.Fatalf("lost diagnostic: %s", got)
	}
	if got := translationProviderDetail(fmt.Errorf("timeout")); got != "" {
		t.Fatal("unexpected diagnostic")
	}
}

package subtitles

import (
	"errors"
	"fmt"

	"curated-backend/internal/llm"
)

// TranslationError retains the cause without exposing its raw body in Error().
type TranslationError struct {
	Category string
	Cause    error
}

func (e *TranslationError) Error() string {
	return fmt.Sprintf("subtitle translation failed (%s)", e.Category)
}
func (e *TranslationError) Unwrap() error { return e.Cause }

// IncompleteTranslationError distinguishes a safe subset from corrupt IDs or
// empty translations. Only a validated subset may be checkpointed and repaired.
type IncompleteTranslationError struct {
	Expected int
	Received int
}

func (e *IncompleteTranslationError) Error() string {
	return fmt.Sprintf("translation omitted %d of %d cues (received %d); retry remaining translation", e.Expected-e.Received, e.Expected, e.Received)
}

func translationProviderDetail(err error) string {
	var provider *llm.HTTPError
	if errors.As(err, &provider) {
		return provider.PublicDetail
	}
	return ""
}

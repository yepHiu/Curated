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

func translationProviderDetail(err error) string {
	var provider *llm.HTTPError
	if errors.As(err, &provider) {
		return provider.PublicDetail
	}
	return ""
}

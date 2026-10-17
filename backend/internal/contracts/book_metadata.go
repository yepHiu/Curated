package contracts

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// BookCredit keeps the source spelling and the exact title fragment supporting a role.
type BookCredit struct {
	Name     string `json:"name"`
	Evidence string `json:"evidence"`
}

type BookMetadata struct {
	CleanTitle      string       `json:"cleanTitle,omitempty"`
	TranslatedTitle string       `json:"translatedTitle,omitempty"`
	Authors         []BookCredit `json:"authors,omitempty"`
	Circles         []BookCredit `json:"circles,omitempty"`
	Models          []BookCredit `json:"models,omitempty"`
	Photographers   []BookCredit `json:"photographers,omitempty"`
	Uncertainty     string       `json:"uncertainty,omitempty"`
}

// BookMetadataState binds a preview to both its source and current saved values.
type BookMetadataState struct {
	Title          string       `json:"title"`
	OriginalTitle  string       `json:"originalTitle"`
	SourceFileName string       `json:"sourceFileName"`
	Metadata       BookMetadata `json:"metadata"`
}

func ValidateBookMetadata(kind string, m BookMetadata, source BookMetadataState) error {
	if kind != "comic" && kind != "photo" {
		return fmt.Errorf("invalid book kind")
	}
	for _, title := range []string{m.CleanTitle, m.TranslatedTitle} {
		if strings.TrimSpace(title) == "" || utf8.RuneCountInString(title) > MaxBookTitleRunes {
			return fmt.Errorf("invalid book title")
		}
	}
	if utf8.RuneCountInString(m.Uncertainty) > 1000 {
		return fmt.Errorf("uncertainty too long")
	}
	if kind == "comic" && len(m.Models)+len(m.Photographers) > 0 || kind == "photo" && len(m.Authors)+len(m.Circles) > 0 {
		return fmt.Errorf("roles do not match book kind")
	}
	for _, group := range [][]BookCredit{m.Authors, m.Circles, m.Models, m.Photographers} {
		if len(group) > 12 {
			return fmt.Errorf("too many book credits")
		}
		seen := map[string]bool{}
		for _, credit := range group {
			name, evidence := strings.TrimSpace(credit.Name), strings.TrimSpace(credit.Evidence)
			if name == "" || evidence == "" || name != credit.Name || evidence != credit.Evidence || utf8.RuneCountInString(name) > 120 || utf8.RuneCountInString(evidence) > MaxBookTitleRunes || seen[name] || !strings.Contains(evidence, name) || (!strings.Contains(source.OriginalTitle, evidence) && !strings.Contains(source.SourceFileName, evidence)) {
				return fmt.Errorf("book credit lacks exact source evidence")
			}
			seen[name] = true
		}
	}
	return nil
}

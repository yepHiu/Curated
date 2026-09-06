package importcheck

import (
	"sort"
	"strings"
	"unicode/utf8"

	"curated-backend/internal/library/moviecode"
	"curated-backend/internal/scanner"
)

const (
	MaxNames          = 200
	MaxNameRunes      = 512
	MaxMatchesPerFile = 8
)

// IndexItem is a compact active-library catalog row used for import code checks.
type IndexItem struct {
	ID    string
	Code  string
	Title string
}

// Match is one library movie that matches an incoming filename's catalog code.
type Match struct {
	MovieID   string
	Code      string
	Title     string
	MatchKind string
}

// Item is the check result for one submitted name.
type Item struct {
	Name          string
	ExtractedCode string
	Matches       []Match
}

// Check extracts catalog codes from filenames and matches them against the library index.
func Check(names []string, index []IndexItem) []Item {
	out := make([]Item, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		item := Item{Name: name, Matches: []Match{}}
		extracted := scanner.ExtractNumber(baseName(name))
		item.ExtractedCode = extracted
		if extracted != "" {
			item.Matches = matchIndex(extracted, index)
		}
		out = append(out, item)
	}
	return out
}

func baseName(name string) string {
	normalized := strings.ReplaceAll(name, "\\", "/")
	if idx := strings.LastIndex(normalized, "/"); idx >= 0 {
		return normalized[idx+1:]
	}
	return normalized
}

func matchIndex(extracted string, index []IndexItem) []Match {
	byID := map[string]Match{}
	for _, row := range index {
		kind := moviecode.StrongerKind(
			moviecode.Classify(extracted, row.Code),
			moviecode.Classify(extracted, row.ID),
		)
		if kind == moviecode.MatchNone {
			continue
		}
		id := strings.TrimSpace(row.ID)
		if id == "" {
			id = moviecode.NormalizeForStorageID(row.Code)
		}
		if id == "" {
			continue
		}
		if prev, ok := byID[id]; ok {
			kind = moviecode.StrongerKind(prev.MatchKind, kind)
		}
		code := strings.TrimSpace(row.Code)
		if code == "" {
			code = extracted
		}
		byID[id] = Match{
			MovieID:   id,
			Code:      code,
			Title:     strings.TrimSpace(row.Title),
			MatchKind: kind,
		}
	}
	matches := make([]Match, 0, len(byID))
	for _, match := range byID {
		matches = append(matches, match)
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].MatchKind != matches[j].MatchKind {
			return moviecode.StrongerKind(matches[i].MatchKind, matches[j].MatchKind) == matches[i].MatchKind
		}
		if matches[i].Code != matches[j].Code {
			return matches[i].Code < matches[j].Code
		}
		return matches[i].MovieID < matches[j].MovieID
	})
	if len(matches) > MaxMatchesPerFile {
		matches = matches[:MaxMatchesPerFile]
	}
	return matches
}

// ValidateNames checks request limits before running Check.
func ValidateNames(names []string) string {
	if len(names) > MaxNames {
		return "too many filenames"
	}
	count := 0
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > MaxNameRunes {
			return "filename is too long"
		}
		count++
	}
	if count == 0 {
		return "filenames are required"
	}
	return ""
}

package storage

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"curated-backend/internal/contracts"
)

const (
	maxSavedViewNameRunes = 40
	maxSavedViewTextRunes = 200
)

func NormalizeSavedViewName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", errors.New("saved view name is required")
	}
	if utf8.RuneCountInString(name) > maxSavedViewNameRunes {
		return "", errors.New("saved view name is too long")
	}
	return name, nil
}

func NormalizeSavedViewFilters(input contracts.SavedViewFiltersV1) (contracts.SavedViewFiltersV1, error) {
	if input.SchemaVersion != contracts.SavedViewSchemaVersion {
		return contracts.SavedViewFiltersV1{}, errors.New("unsupported saved view schemaVersion")
	}
	out := contracts.SavedViewFiltersV1{
		SchemaVersion: contracts.SavedViewSchemaVersion,
		Mode:          strings.ToLower(strings.TrimSpace(input.Mode)),
		Query:         strings.TrimSpace(input.Query),
		Tag:           strings.Join(ParseMovieTagFilters(input.Tag), ","),
		Actor:         strings.Join(ParseMovieTagFilters(input.Actor), ","),
		Studio:        strings.Join(ParseMovieTagFilters(input.Studio), ","),
		Tab:           strings.ToLower(strings.TrimSpace(input.Tab)),
		PlayState:     strings.ToLower(strings.TrimSpace(input.PlayState)),
		Resolution:    normalizeSavedViewResolutionValue(input.Resolution),
		Year:          normalizeSavedViewYearValue(input.Year),
		Runtime:       strings.ToLower(strings.TrimSpace(input.Runtime)),
		Catalog:       strings.ToLower(strings.TrimSpace(input.Catalog)),
		Sort:          strings.ToLower(strings.TrimSpace(input.Sort)),
	}
	if out.Mode == "" {
		out.Mode = "library"
	}
	if !savedViewOneOf(out.Mode, "library", "favorites", "recent", "tags", "trash") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view mode")
	}
	if out.Tab == "" {
		out.Tab = "all"
	}
	if !savedViewOneOf(out.Tab, "all", "new", "top-rated") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view tab")
	}
	if out.PlayState == "" {
		out.PlayState = "all"
	}
	if !savedViewOneOf(out.PlayState, "all", "unwatched", "in-progress", "completed") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view playState")
	}
	for _, value := range []string{out.Query, out.Tag, out.Actor, out.Studio} {
		if utf8.RuneCountInString(value) > maxSavedViewTextRunes {
			return contracts.SavedViewFiltersV1{}, errors.New("saved view filter text is too long")
		}
	}
	if utf8.RuneCountInString(out.Resolution) > 40 {
		return contracts.SavedViewFiltersV1{}, errors.New("saved view resolution is too long")
	}
	if input.UserRating != nil {
		value := *input.UserRating
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 5 {
			return contracts.SavedViewFiltersV1{}, errors.New("saved view userRating must be between 0 and 5")
		}
		out.UserRating = &value
	}
	if input.Unrated {
		out.Unrated = true
		out.UserRating = nil
	}
	if out.Year == "" && strings.TrimSpace(input.Year) != "" {
		return contracts.SavedViewFiltersV1{}, errors.New("saved view year must be unknown or a year from 1800 to 3000")
	}
	if out.Runtime != "" && !savedViewOneOf(out.Runtime, "short", "standard", "long") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view runtime")
	}
	if out.Catalog != "" && !savedViewOneOf(out.Catalog, "unscraped", "no-cover") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view catalog")
	}
	if out.Sort != "" && !savedViewOneOf(out.Sort, "added", "release", "rating", "code", "actor", "studio", "year") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view sort")
	}
	if out.Sort == "added" {
		out.Sort = ""
	}
	if input.AddedWithinDays < 0 || input.AddedWithinDays > 3650 {
		return contracts.SavedViewFiltersV1{}, errors.New("saved view addedWithinDays must be between 1 and 3650")
	}
	out.AddedWithinDays = input.AddedWithinDays

	if out.Mode == "trash" {
		return contracts.SavedViewFiltersV1{
			SchemaVersion: contracts.SavedViewSchemaVersion,
			Mode:          "trash",
			Tab:           "all",
			PlayState:     "all",
		}, nil
	}
	return out, nil
}

func normalizeSavedViewYearValue(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	if normalized == "unknown" {
		return "unknown"
	}
	if len(normalized) != 4 {
		return ""
	}
	year, err := strconv.Atoi(normalized)
	if err != nil || year < 1800 || year > 3000 {
		return ""
	}
	return normalized
}

func normalizeSavedViewResolutionValue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "4k", "2160p", "uhd", "3840x2160":
		return "4k"
	case "1080p", "full hd", "fhd":
		return "1080p"
	case "720p", "hd":
		return "720p"
	case "480p", "sd":
		return "480p"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func savedViewOneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

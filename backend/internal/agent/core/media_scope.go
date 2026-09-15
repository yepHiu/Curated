package core

import (
	"curated-backend/internal/contracts"
	"strings"
)

// IsBookLibraryRoute reports whether the Agent page route belongs to the comic or photo Beta.
func IsBookLibraryRoute(route string) bool {
	value := strings.ToLower(strings.TrimSpace(route))
	switch value {
	case "comics", "comic-detail", "comic-reader", "photos", "photo-detail", "photo-viewer":
		return true
	}
	value = strings.TrimPrefix(value, "#")
	for _, root := range []string{"/comics", "/photos"} {
		if value == root || strings.HasPrefix(value, root+"/") || strings.HasPrefix(value, root+"?") {
			return true
		}
	}
	return false
}

// MoviePageContext keeps movie and book page payloads in their own domains.
// Book routes may carry comic/photo ids and filters, but never become movie or actor anchors.
// Movie routes drop book ids so an old client cannot grant movie tools a book identity.
func MoviePageContext(page *contracts.AIChatContext) *contracts.AIChatContext {
	if page == nil {
		return nil
	}
	if IsBookLibraryRoute(page.Route) {
		out := *page
		out.MovieID = ""
		out.ActorName = ""
		out.SelectedMovieIDs = nil
		out.SelectedActors = nil
		if out.ActiveFilters != nil {
			filters := *out.ActiveFilters
			filters.Actor = ""
			filters.PlayState = ""
			filters.Runtime = ""
			if filters.Query == "" && filters.Tag == "" && filters.Favorite == nil && filters.ReadStatus == "" {
				out.ActiveFilters = nil
			} else {
				out.ActiveFilters = &filters
			}
		}
		return &out
	}
	if page.ComicID == "" && page.PhotoID == "" && len(page.SelectedComicIDs) == 0 && len(page.SelectedPhotoIDs) == 0 {
		if page.ActiveFilters == nil || (page.ActiveFilters.Favorite == nil && page.ActiveFilters.ReadStatus == "") {
			return page
		}
	}
	out := *page
	out.ComicID = ""
	out.PhotoID = ""
	out.SelectedComicIDs = nil
	out.SelectedPhotoIDs = nil
	if out.ActiveFilters != nil {
		filters := *out.ActiveFilters
		filters.Favorite = nil
		filters.ReadStatus = ""
		out.ActiveFilters = &filters
	}
	return &out
}

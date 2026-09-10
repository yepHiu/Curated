package core

import (
	"curated-backend/internal/contracts"
	"strings"
)

// MoviePageContext excludes Beta book data before prompt construction or entity seeding.
// This is also enforced server-side for old clients that still send book page context.
func MoviePageContext(page *contracts.AIChatContext) *contracts.AIChatContext {
	if page == nil {
		return nil
	}
	route := strings.ToLower(strings.TrimSpace(page.Route))
	switch route {
	case "comics", "comic-detail", "comic-reader", "photos", "photo-detail", "photo-viewer":
		return nil
	}
	route = strings.TrimPrefix(route, "#")
	for _, root := range []string{"/comics", "/photos"} {
		if route == root || strings.HasPrefix(route, root+"/") || strings.HasPrefix(route, root+"?") {
			return nil
		}
	}
	return page
}

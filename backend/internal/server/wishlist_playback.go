package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"curated-backend/internal/proxyenv"
	"github.com/PuerkitoBio/goquery"
)

// These destinations are fixed in code. A wishlist code never becomes a host or an arbitrary URL.
type playbackSite struct {
	name     string
	address  string
	selector string
}

var playbackSites = []playbackSite{
	{"Jable", "https://jable.tv/videos/%s/", ".info-header"},
	{"MISSAV", "https://missav.ws/%s/", "h1"},
}

type playbackResult struct {
	Site   string `json:"site"`
	Status string `json:"status"` // available, missing, blocked, or unknown
	URL    string `json:"url,omitempty"`
}

var playbackCodePattern = regexp.MustCompile(`(?i)[a-z0-9]{2,12}-\d{2,6}`)

func playbackCodeMatches(text, code string) bool {
	for _, found := range playbackCodePattern.FindAllString(text, -1) {
		if strings.EqualFold(found, code) {
			return true
		}
	}
	return false
}

func playbackPageResult(site playbackSite, code, pageURL string, body io.Reader) playbackResult {
	result := playbackResult{Site: site.name, Status: "missing"}
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		result.Status = "unknown"
		return result
	}
	if doc.Find(site.selector).Length() == 0 {
		result.Status = "unknown"
		return result
	}
	// A 200 response may be a challenge, search, or error page. Require the exact code.
	if playbackCodeMatches(doc.Find("title, h1, .info-header").Text(), code) {
		result.Status, result.URL = "available", pageURL
	}
	return result
}

func probePlayback(ctx context.Context, client *http.Client, site playbackSite, code string) playbackResult {
	result := playbackResult{Site: site.name, Status: "unknown"}
	address := fmt.Sprintf(site.address, url.QueryEscape(strings.ToLower(code)))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return result
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Curated/1.0)")
	response, err := client.Do(request)
	if err != nil {
		return result
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests || response.Header.Get("Cf-Mitigated") == "challenge" {
		result.Status, result.URL = "blocked", address
		return result
	}
	if response.StatusCode == http.StatusNotFound {
		result.Status = "missing"
		return result
	}
	if response.StatusCode != http.StatusOK {
		return result
	}
	if !strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/html") {
		return result
	}
	return playbackPageResult(site, code, address, io.LimitReader(response.Body, 2<<20))
}

func probePlaybackSite(ctx context.Context, client *http.Client, site playbackSite, code string) playbackResult {
	result := probePlayback(ctx, client, site, code)
	if site.name == "Jable" && result.Status == "missing" {
		// The source userscript also checks Jable's subtitle variant.
		variant := site
		variant.address = "https://jable.tv/videos/%s-c/"
		result = probePlayback(ctx, client, variant, code)
	}
	return result
}

func (h *Handler) handleWishlistPlayback(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeAppError(w, 503, "COMMON_UNAVAILABLE", "wishlist unavailable")
		return
	}
	item, err := h.store.GetWishlist(r.Context(), r.PathValue("id"))
	if err != nil {
		wishlistError(w, err)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(item.Code))
	if playbackCodePattern.FindString(code) != code {
		writeAppError(w, 400, "WISHLIST_INVALID_INPUT", "unsupported code")
		return
	}
	proxy := h.cfg.Proxy
	if h.proxyCtl != nil {
		proxy = h.proxyCtl.Proxy()
	}
	client, err := proxyenv.NewHTTPClientForProxy(proxy, 8*time.Second)
	if err != nil {
		writeAppError(w, 503, "COMMON_UNAVAILABLE", "outbound proxy unavailable")
		return
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Do not follow redirects to an arbitrary host or scheme.
		if len(via) >= 3 || req.URL.Scheme != "https" || req.URL.Hostname() != via[0].URL.Hostname() {
			return http.ErrUseLastResponse
		}
		return nil
	}
	results := make([]playbackResult, len(playbackSites))
	done := make(chan int, len(playbackSites))
	for i, site := range playbackSites {
		go func(i int, site playbackSite) {
			results[i] = probePlaybackSite(r.Context(), client, site, code)
			done <- i
		}(i, site)
	}
	for range playbackSites {
		<-done
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

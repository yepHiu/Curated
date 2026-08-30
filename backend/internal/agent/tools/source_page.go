package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
)

var (
	scriptStylePattern = regexp.MustCompile(`(?is)<(script|style|noscript)\b[^>]*>.*?</(script|style|noscript)>`)
	tagPattern         = regexp.MustCompile(`(?s)<[^>]+>`)
)

// SourcePageFetcher loads a previously allowlisted https source page.
type SourcePageFetcher interface {
	FetchSourcePage(ctx context.Context, pageURL string, allowedHosts []string) (SourcePageContent, error)
}

// SourcePageContent is extracted visible text from a source-site page.
type SourcePageContent struct {
	Text      string
	FinalURL  string
	Truncated bool
}

func getSourcePage(pages SourcePageFetcher, urls *core.SourceURLStore) core.ToolDefinition {
	return core.ToolDefinition{
		Name: core.GetSourcePageName,
		Description: "Read one https source-site page already returned this turn (homepage or search_provider_titles homepage). " +
			"Do not pass arbitrary URLs. Use this for longer reviews or bios after search_provider_titles. Not a web search.",
		ParamsSchema: object(map[string]core.Schema{
			"url": strField("Exact https URL from this-turn homepage or provider-title homepage"),
		}, "url"),
		Permission: core.PermissionRead,
		Domain:     core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			rawURL := strArg(decodeArgs(call.Args), "url")
			if urls == nil || !urls.Known(call.SessionID, rawURL) {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_TOOL_INVALID_ARGS",
					Message: "url is not on this turn's source allowlist",
				}}, nil
			}
			if err := ValidatePublicHTTPSURL(rawURL); err != nil {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_TOOL_INVALID_ARGS",
					Message: err.Error(),
				}}, nil
			}
			if pages == nil {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_CHAT_FAILED",
					Message: "source page fetch is unavailable",
				}}, nil
			}
			page, err := pages.FetchSourcePage(ctx, rawURL, urls.Hosts(call.SessionID))
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_CHAT_FAILED",
					Message: err.Error(),
				}}, nil
			}
			return core.Result{OK: true, Data: wrapSource(map[string]any{
				"url":       rawURL,
				"finalUrl":  page.FinalURL,
				"text":      page.Text,
				"truncated": page.Truncated,
			}), Truncated: page.Truncated}, nil
		},
	}
}

// ValidatePublicHTTPSURL rejects non-https, credentials, localhost, and literal private IPs.
func ValidatePublicHTTPSURL(raw string) error {
	parsed, host, err := parseHTTPSURL(raw)
	if err != nil {
		return err
	}
	_ = parsed
	if hostBlocked(host) {
		return fmt.Errorf("url host is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil && ipBlocked(ip) {
		return fmt.Errorf("url host is not allowed")
	}
	return nil
}

func parseHTTPSURL(raw string) (*url.URL, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return nil, "", fmt.Errorf("invalid url")
	}
	if !strings.EqualFold(parsed.Scheme, "https") || parsed.Opaque != "" || parsed.User != nil {
		return nil, "", fmt.Errorf("only https urls are allowed")
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return nil, "", fmt.Errorf("invalid url")
	}
	return parsed, host, nil
}

func hostBlocked(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return true
	}
	return false
}

func ipBlocked(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return true
	}
	return false
}

func SourceIPBlocked(ip net.IP) bool {
	return ipBlocked(ip)
}

func SourceHostAllowed(host string, allowedHosts []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	for _, allowed := range allowedHosts {
		if host == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func ReadSourcePage(ctx context.Context, client *http.Client, pageURL string) (SourcePageContent, error) {
	if client == nil {
		return SourcePageContent{}, fmt.Errorf("http client is unavailable")
	}
	req, err := sourcePageRequest(ctx, pageURL)
	if err != nil {
		return SourcePageContent{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return SourcePageContent{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SourcePageContent{}, fmt.Errorf("source page returned HTTP %d", resp.StatusCode)
	}
	body, truncatedBody, err := readLimitedBody(resp.Body, core.MaxSourcePageBytes)
	if err != nil {
		return SourcePageContent{}, err
	}
	text, truncatedText := extractVisibleText(string(body), core.MaxSourcePageBytes)
	finalURL := pageURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	return SourcePageContent{
		Text:      text,
		FinalURL:  finalURL,
		Truncated: truncatedBody || truncatedText,
	}, nil
}

func extractVisibleText(html string, maxBytes int) (string, bool) {
	if maxBytes <= 0 {
		maxBytes = core.MaxSourcePageBytes
	}
	stripped := scriptStylePattern.ReplaceAllString(html, " ")
	stripped = tagPattern.ReplaceAllString(stripped, " ")
	stripped = strings.Join(strings.Fields(stripped), " ")
	truncated := false
	if len(stripped) > maxBytes {
		stripped = stripped[:maxBytes]
		truncated = true
	}
	if utf8.RuneCountInString(stripped) == 0 {
		return "", truncated
	}
	return stripped, truncated
}

func readLimitedBody(body io.Reader, maxBytes int) ([]byte, bool, error) {
	if maxBytes <= 0 {
		maxBytes = core.MaxSourcePageBytes
	}
	raw, err := io.ReadAll(io.LimitReader(body, int64(maxBytes)+1))
	if err != nil {
		return nil, false, err
	}
	truncated := len(raw) > maxBytes
	if truncated {
		raw = raw[:maxBytes]
	}
	return raw, truncated, nil
}

func sourcePageRequest(ctx context.Context, pageURL string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,text/plain;q=0.8")
	req.Header.Set("User-Agent", "Curated-Agent/1.0")
	return req, nil
}

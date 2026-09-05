package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/tools"
)

func (a *App) FetchSourcePage(ctx context.Context, pageURL string, allowedHosts []string) (tools.SourcePageContent, error) {
	if err := tools.ValidatePublicHTTPSURL(pageURL); err != nil {
		return tools.SourcePageContent{}, err
	}
	_, host, err := parseSourceHost(pageURL)
	if err != nil {
		return tools.SourcePageContent{}, err
	}
	if !tools.SourceHostAllowed(host, allowedHosts) {
		return tools.SourcePageContent{}, fmt.Errorf("url host is not on this turn's allowlist")
	}
	client, err := newAIHTTPClient(a.currentProxyConfig(), core.ReadTimeout)
	if err != nil {
		return tools.SourcePageContent{}, err
	}
	if err := attachSourcePageGuards(client, allowedHosts); err != nil {
		return tools.SourcePageContent{}, err
	}
	return tools.ReadSourcePage(ctx, client, pageURL)
}

func parseSourceHost(raw string) (normalized, host string, err error) {
	normalized, host, ok := core.NormalizeSourceURL(raw)
	if !ok {
		return "", "", fmt.Errorf("only https urls are allowed")
	}
	return normalized, host, nil
}

func attachSourcePageGuards(client *http.Client, allowedHosts []string) error {
	if client == nil {
		return fmt.Errorf("http client is unavailable")
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport == nil {
		transport, ok = http.DefaultTransport.(*http.Transport)
		if !ok {
			return fmt.Errorf("http transport is unavailable")
		}
	}
	client.Transport = &sourcePageTransport{
		base:         transport.Clone(),
		allowedHosts: append([]string(nil), allowedHosts...),
		lookupIP:     net.DefaultResolver.LookupIPAddr,
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return validateSourcePageRequest(req, allowedHosts)
	}
	return nil
}

func validateSourcePageRequest(req *http.Request, allowedHosts []string) error {
	if req == nil || req.URL == nil {
		return fmt.Errorf("invalid source request")
	}
	if err := tools.ValidatePublicHTTPSURL(req.URL.String()); err != nil {
		return err
	}
	if !tools.SourceHostAllowed(strings.ToLower(req.URL.Hostname()), allowedHosts) {
		return fmt.Errorf("url host is not on this turn's allowlist")
	}
	return nil
}

// Validate and pin the origin before Transport chooses its connection route.
// DialContext may connect to a trusted local proxy, whereas CONNECT/SOCKS must
// still target a public origin IP. Pinning also prevents a second DNS lookup
// at the proxy from redirecting an approved hostname to the private network.
type sourcePageTransport struct {
	base         *http.Transport
	allowedHosts []string
	lookupIP     func(context.Context, string) ([]net.IPAddr, error)
}

func (t *sourcePageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := validateSourcePageRequest(req, t.allowedHosts); err != nil {
		return nil, err
	}
	host := req.URL.Hostname()
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := t.lookupIP(req.Context(), host)
		if err != nil {
			return nil, err
		}
		for _, candidate := range ips {
			if !tools.SourceIPBlocked(candidate.IP) {
				ip = candidate.IP
				break
			}
		}
	}
	if ip == nil || tools.SourceIPBlocked(ip) {
		return nil, fmt.Errorf("url host resolved to a private address")
	}

	transport := t.base.Clone()
	// Each request has its own TLS origin and pinned IP (including redirects).
	transport.DisableKeepAlives = true
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.ServerName = host
	if transport.Proxy != nil {
		proxyURL, err := transport.Proxy(req)
		if err != nil {
			return nil, err
		}
		if proxyURL == nil {
			transport.Proxy = nil
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	pinned := req.Clone(req.Context())
	port := req.URL.Port()
	if port == "" {
		port = "443"
	}
	pinned.URL.Host = net.JoinHostPort(ip.String(), port)
	pinned.Host = req.URL.Host
	resp, err := transport.RoundTrip(pinned)
	if resp != nil {
		resp.Request = req
	}
	return resp, err
}

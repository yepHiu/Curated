package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

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
		if dt, ok := http.DefaultTransport.(*http.Transport); ok {
			transport = dt.Clone()
		} else {
			return fmt.Errorf("http transport is unavailable")
		}
	} else {
		transport = transport.Clone()
	}
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if ip := net.ParseIP(host); ip != nil {
			if tools.SourceIPBlocked(ip) {
				return nil, fmt.Errorf("url host is not allowed")
			}
			return dialer.DialContext(ctx, network, address)
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		var first net.IP
		for _, ip := range ips {
			if tools.SourceIPBlocked(ip.IP) {
				continue
			}
			first = ip.IP
			break
		}
		if first == nil {
			return nil, fmt.Errorf("url host resolved to a private address")
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(first.String(), port))
	}
	client.Transport = transport
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		if req == nil || req.URL == nil {
			return fmt.Errorf("invalid redirect")
		}
		if err := tools.ValidatePublicHTTPSURL(req.URL.String()); err != nil {
			return err
		}
		host := strings.ToLower(req.URL.Hostname())
		if !tools.SourceHostAllowed(host, allowedHosts) {
			return fmt.Errorf("redirect host is not on this turn's allowlist")
		}
		return nil
	}
	return nil
}

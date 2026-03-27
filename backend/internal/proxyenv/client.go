package proxyenv

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"jav-shadcn/backend/internal/config"
)

const defaultOutboundTestTimeout = 20 * time.Second

// NewHTTPClientForProxy returns an HTTP client that either uses no proxy (direct) or the given proxy URL.
// It does not read HTTP_PROXY from the environment; use this for one-off tests with draft proxy settings.
func NewHTTPClientForProxy(cfg config.ProxyConfig, timeout time.Duration) (*http.Client, error) {
	if timeout <= 0 {
		timeout = defaultOutboundTestTimeout
	}
	dt, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("http.DefaultTransport is not *http.Transport")
	}
	tr := dt.Clone()
	if !cfg.Enabled || strings.TrimSpace(cfg.URL) == "" {
		tr.Proxy = nil
	} else {
		u, err := buildProxyURL(cfg)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(u)
	}
	return &http.Client{Transport: tr, Timeout: timeout}, nil
}

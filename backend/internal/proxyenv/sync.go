package proxyenv

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"

	"jav-shadcn/backend/internal/config"
)

const (
	envHTTP  = "HTTP_PROXY"
	envHTTPS = "HTTPS_PROXY"
	envAll   = "ALL_PROXY"
)

// Sync sets or clears HTTP_PROXY, HTTPS_PROXY, and ALL_PROXY from library proxy config.
// Metatube (cleanhttp) and net/http DefaultTransport use http.ProxyFromEnvironment,
// which reads these variables per request.
func Sync(cfg config.ProxyConfig, logger *zap.Logger) {
	if !cfg.Enabled || strings.TrimSpace(cfg.URL) == "" {
		clearEnv()
		if logger != nil {
			logger.Info("outbound proxy disabled; cleared HTTP_PROXY/HTTPS_PROXY/ALL_PROXY")
		}
		return
	}

	rawTrim := strings.TrimSpace(cfg.URL)
	proxyURL, err := buildProxyURL(cfg)
	if err != nil {
		if logger != nil {
			logger.Warn("invalid proxy URL; clearing proxy env", zap.Error(err))
		}
		clearEnv()
		return
	}

	if logger != nil && strings.HasPrefix(strings.ToLower(rawTrim), "https://") &&
		strings.HasPrefix(strings.ToLower(proxyURL.String()), "http://") {
		logger.Info(
			"proxy URL: normalized https to http for loopback (local HTTP proxies use CONNECT over cleartext, not TLS to the proxy)",
			zap.String("before", redactProxyRawForLog(rawTrim)),
			zap.String("after", proxyURL.Redacted()),
		)
	}

	s := proxyURL.String()
	_ = os.Setenv(envHTTP, s)
	_ = os.Setenv(envHTTPS, s)
	_ = os.Setenv(envAll, s)
	if logger != nil {
		logger.Info("outbound proxy env applied", zap.String("proxy", proxyURL.Redacted()))
	}
}

func clearEnv() {
	_ = os.Unsetenv(envHTTP)
	_ = os.Unsetenv(envHTTPS)
	_ = os.Unsetenv(envAll)
}

func buildProxyURL(cfg config.ProxyConfig) (*url.URL, error) {
	raw := strings.TrimSpace(cfg.URL)
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("proxy URL must include scheme and host")
	}

	normalizeLocalHTTPProxyScheme(u)

	user := strings.TrimSpace(cfg.Username)
	pass := cfg.Password
	hasUserInURL := u.User != nil && (u.User.Username() != "" || passwordSet(u.User))
	if !hasUserInURL && (user != "" || pass != "") {
		u.User = url.UserPassword(user, pass)
	}

	return u, nil
}

func passwordSet(ui *url.Userinfo) bool {
	if ui == nil {
		return false
	}
	_, ok := ui.Password()
	return ok
}

// normalizeLocalHTTPProxyScheme fixes a common mistake: Clash/sing-box etc. expose an HTTP proxy
// (CONNECT over cleartext). Using https://127.0.0.1:port makes Go speak TLS to the proxy and
// causes errors like "proxyconnect tcp: EOF".
func redactProxyRawForLog(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User == nil {
		return raw
	}
	return u.Redacted()
}

func normalizeLocalHTTPProxyScheme(u *url.URL) {
	if u == nil || !strings.EqualFold(u.Scheme, "https") {
		return
	}
	host := strings.Trim(u.Hostname(), "[]")
	if host == "localhost" || host == "::1" {
		u.Scheme = "http"
		return
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		u.Scheme = "http"
	}
}

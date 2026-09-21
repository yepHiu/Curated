package server

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

const (
	corsAllowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowHeaders = "Content-Type, Authorization, X-Curated-Offset, X-Curated-Chunk-Size, X-Curated-Chunk-SHA256, X-Curated-Client, X-Curated-Client-Version, X-Curated-OS, X-Curated-OS-Version, Sec-CH-UA-Platform, Sec-CH-UA-Platform-Version"
	clientHints      = "Sec-CH-UA-Platform, Sec-CH-UA-Platform-Version"
)

var _, carrierGradeNAT, _ = net.ParseCIDR("100.64.0.0/10")

func (h *Handler) withRequestSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-CH", clientHints)

		if !requestHostAllowed(r.Host, h.cfg) {
			writeAppError(w, http.StatusForbidden, contracts.ErrorCodeForbidden, "request host is not allowed")
			return
		}

		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			w.Header().Add("Vary", "Origin")
			if !browserOriginAllowed(origin, r, h.cfg) && !(r.URL.Path == wishlistIntakePath && (r.Method == http.MethodPost || r.Method == http.MethodOptions) && wishlistExtensionOrigin(origin)) {
				writeAppError(w, http.StatusForbidden, contracts.ErrorCodeForbidden, "browser origin is not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
		w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// browserOriginAllowed 允许同源、loopback 开发 Origin，以及主配置精确列出的跨域入口。
func browserOriginAllowed(origin string, r *http.Request, cfg config.Config) bool {
	parsed, normalized, ok := parseBrowserOrigin(origin)
	if !ok {
		return false
	}
	if requestOrigin(r) == normalized {
		return true
	}
	if originHostIsLoopback(parsed.Hostname()) {
		return true
	}
	for _, candidate := range cfg.CORSAllowedOrigins {
		_, allowed, valid := parseBrowserOrigin(candidate)
		if valid && normalized == allowed {
			return true
		}
	}
	return false
}

// requestHostAllowed 校验请求 Host 是否可以访问当前监听；LAN 偏好未改绑前仍按 loopback 拒绝私网 Host。
func requestHostAllowed(requestHost string, cfg config.Config) bool {
	// Unit handlers without a runtime config retain their existing test behavior. Production
	// always receives a normalized non-empty HttpAddr from config.Load.
	if strings.TrimSpace(cfg.HttpAddr) == "" {
		return true
	}
	host := authorityHostname(requestHost)
	if host == "" {
		return false
	}
	if originHostIsLoopback(host) {
		return true
	}
	// Host 策略跟当前真实监听走：仅保存 lanEnabled 偏好、仍绑 loopback 时不放宽私网 Host。
	if !cfg.LANEnabled || config.HTTPAddrIsLoopback(cfg.HttpAddr) {
		return false
	}

	listenHost := authorityHostname(cfg.HttpAddr)
	if listenHost != "" && !isWildcardHost(listenHost) && strings.EqualFold(host, listenHost) {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && lanAddressAllowed(ip) {
		return true
	}
	if machineName, err := os.Hostname(); err == nil && strings.EqualFold(host, strings.TrimSpace(machineName)) {
		return true
	}
	for _, origin := range cfg.CORSAllowedOrigins {
		parsed, _, ok := parseBrowserOrigin(origin)
		if ok && strings.EqualFold(host, parsed.Hostname()) {
			return true
		}
	}
	return false
}

func parseBrowserOrigin(raw string) (*url.URL, string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.User != nil || parsed.Host == "" {
		return nil, "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, "", false
	}
	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, "", false
	}
	authority := canonicalAuthority(parsed.Host)
	if authority == "" {
		return nil, "", false
	}
	return parsed, scheme + "://" + authority, true
}

func requestOrigin(r *http.Request) string {
	if r == nil {
		return ""
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	authority := canonicalAuthority(r.Host)
	if authority == "" {
		return ""
	}
	return scheme + "://" + authority
}

func canonicalAuthority(raw string) string {
	host := authorityHostname(raw)
	if host == "" {
		return ""
	}
	port := authorityPort(raw)
	if port == "" {
		if strings.Contains(host, ":") {
			return "[" + strings.ToLower(host) + "]"
		}
		return strings.ToLower(host)
	}
	return strings.ToLower(net.JoinHostPort(host, port))
}

func authorityHostname(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		return strings.Trim(strings.TrimSpace(host), "[]")
	}
	return strings.Trim(raw, "[]")
}

func authorityPort(raw string) string {
	_, port, err := net.SplitHostPort(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return port
}

func originHostIsLoopback(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isWildcardHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	return host == "" || host == "0.0.0.0" || host == "::"
}

func lanAddressAllowed(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsMulticast() {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return true
	}
	return carrierGradeNAT != nil && carrierGradeNAT.Contains(ip)
}

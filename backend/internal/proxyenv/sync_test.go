package proxyenv

import (
	"net/url"
	"os"
	"testing"

	"curated-backend/internal/config"
)

func TestSync_disabled_clearsEnv(t *testing.T) {
	t.Setenv(envHTTP, "http://old")
	t.Setenv(envHTTPS, "http://old")
	t.Setenv(envAll, "http://old")

	Sync(config.ProxyConfig{Enabled: false}, nil)

	if os.Getenv(envHTTP) != "" || os.Getenv(envHTTPS) != "" || os.Getenv(envAll) != "" {
		t.Fatalf("expected proxy env cleared, got http=%q https=%q all=%q", os.Getenv(envHTTP), os.Getenv(envHTTPS), os.Getenv(envAll))
	}
}

func TestSync_disabled_whenURLMissing(t *testing.T) {
	t.Setenv(envHTTP, "http://old")

	Sync(config.ProxyConfig{Enabled: true, URL: "   "}, nil)

	if os.Getenv(envHTTP) != "" {
		t.Fatalf("expected HTTP_PROXY cleared when URL empty")
	}
}

func TestSync_enabled_setsEnv(t *testing.T) {
	Sync(config.ProxyConfig{
		Enabled: true,
		URL:     "http://127.0.0.1:7890",
	}, nil)

	want := "http://127.0.0.1:7890"
	if got := os.Getenv(envHTTP); got != want {
		t.Fatalf("HTTP_PROXY: got %q want %q", got, want)
	}
	if got := os.Getenv(envHTTPS); got != want {
		t.Fatalf("HTTPS_PROXY: got %q want %q", got, want)
	}
	if got := os.Getenv(envAll); got != want {
		t.Fatalf("ALL_PROXY: got %q want %q", got, want)
	}
}

func TestSync_mergesUsernamePassword(t *testing.T) {
	Sync(config.ProxyConfig{
		Enabled:  true,
		URL:      "http://127.0.0.1:7890",
		Username: "user",
		Password: "secret",
	}, nil)

	u, err := url.Parse(os.Getenv(envHTTP))
	if err != nil {
		t.Fatal(err)
	}
	if u.User == nil || u.User.Username() != "user" {
		t.Fatalf("username: got %+v", u.User)
	}
	if p, ok := u.User.Password(); !ok || p != "secret" {
		t.Fatalf("password: ok=%v p=%q", ok, p)
	}
}

func TestSync_keepsUserinfoFromURL(t *testing.T) {
	Sync(config.ProxyConfig{
		Enabled:  true,
		URL:      "http://a:b@127.0.0.1:7890",
		Username: "ignored",
		Password: "ignored",
	}, nil)

	u, err := url.Parse(os.Getenv(envHTTP))
	if err != nil {
		t.Fatal(err)
	}
	if u.User == nil || u.User.Username() != "a" {
		t.Fatalf("expected URL userinfo to win, got %+v", u.User)
	}
}

func TestSync_httpsLoopback_normalizedToHTTP(t *testing.T) {
	Sync(config.ProxyConfig{
		Enabled: true,
		URL:     "https://127.0.0.1:7897",
	}, nil)

	got := os.Getenv(envHTTP)
	if want := "http://127.0.0.1:7897"; got != want {
		t.Fatalf("HTTPS scheme on loopback should become HTTP proxy URL: got %q want %q", got, want)
	}
}

func TestSync_invalidURL_clearsEnv(t *testing.T) {
	t.Setenv(envHTTP, "http://stale")

	Sync(config.ProxyConfig{
		Enabled: true,
		URL:     "://bad",
	}, nil)

	if os.Getenv(envHTTP) != "" {
		t.Fatalf("expected env cleared on invalid URL")
	}
}

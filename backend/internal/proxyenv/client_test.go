package proxyenv

import (
	"testing"

	"curated-backend/internal/config"
)

func TestNewHTTPClientForProxy_direct(t *testing.T) {
	c, err := NewHTTPClientForProxy(config.ProxyConfig{Enabled: false}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil || c.Transport == nil {
		t.Fatal("expected non-nil client and transport")
	}
}

func TestNewHTTPClientForProxy_withProxyURL(t *testing.T) {
	c, err := NewHTTPClientForProxy(config.ProxyConfig{
		Enabled: true,
		URL:     "http://127.0.0.1:7890",
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("expected client")
	}
}

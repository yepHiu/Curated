package app

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"curated-backend/internal/config"
)

func sourceTestClient(t *testing.T, proxyURL string) (*http.Client, *sourcePageTransport) {
	t.Helper()
	client, err := newAIHTTPClient(config.ProxyConfig{Enabled: proxyURL != "", URL: proxyURL}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := attachSourcePageGuards(client, []string{"example.com", "private.test"}); err != nil {
		t.Fatal(err)
	}
	guarded := client.Transport.(*sourcePageTransport)
	guarded.lookupIP = func(_ context.Context, host string) ([]net.IPAddr, error) {
		if host == "private.test" {
			return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
		}
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
	}
	return client, guarded
}

func TestSourcePageLocalHTTPProxyPinsPublicOrigin(t *testing.T) {
	target := make(chan string, 1)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target <- r.Method + " " + r.Host
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()
	client, _ := sourceTestClient(t, proxy.URL)
	_, err := client.Get("https://example.com/source")
	if err == nil {
		t.Fatal("expected proxy's 502 response")
	}
	select {
	case got := <-target:
		if got != "CONNECT 93.184.216.34:443" {
			t.Fatalf("proxy target = %s", got)
		}
	default:
		t.Fatalf("configured proxy was not reached: %v", err)
	}
}

func TestSourcePageLocalSOCKSProxyPinsPublicOrigin(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	target := make(chan []byte, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		greeting := make([]byte, 3)
		if _, err := io.ReadFull(conn, greeting); err != nil {
			return
		}
		if _, err := conn.Write([]byte{5, 0}); err != nil {
			return
		}
		request := make([]byte, 10)
		if _, err := io.ReadFull(conn, request); err != nil {
			return
		}
		target <- request
		_, _ = conn.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
	}()
	client, _ := sourceTestClient(t, "socks5://"+listener.Addr().String())
	_, err = client.Get("https://example.com/source")
	if err == nil {
		t.Fatal("expected SOCKS rejection")
	}
	select {
	case got := <-target:
		if got[3] != 1 || !net.IP(got[4:8]).Equal(net.ParseIP("93.184.216.34")) || got[8] != 1 || got[9] != 187 {
			t.Fatalf("SOCKS target = %v", got)
		}
	default:
		t.Fatalf("configured SOCKS proxy was not reached: %v", err)
	}
}

func TestSourcePageDirectPreservesOriginHostAndTLSName(t *testing.T) {
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "example.com" || r.TLS.ServerName != "example.com" {
			t.Errorf("origin host=%s TLS name=%s", r.Host, r.TLS.ServerName)
		}
		_, _ = w.Write([]byte("source"))
	}))
	defer origin.Close()
	client, guarded := sourceTestClient(t, "")
	guarded.base.TLSClientConfig = origin.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	guarded.base.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "93.184.216.34:443" {
			t.Errorf("unpinned target: %s", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, origin.Listener.Addr().String())
	}
	resp, err := client.Get("https://example.com/source")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Request.URL.Host != "example.com" {
		t.Fatal("pinned address leaked into source URL")
	}
}

func TestSourcePageRejectsPrivateTargetsBeforeConnectingToProxy(t *testing.T) {
	var calls atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()
	client, _ := sourceTestClient(t, proxy.URL)
	for _, target := range []string{"https://127.0.0.1/secret", "https://private.test/secret", "https://off-allowlist.test/", "http://example.com/"} {
		if _, err := client.Get(target); err == nil {
			t.Errorf("accepted %s", target)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("unsafe targets reached proxy")
	}
}

func TestSourcePageRedirectChecksHostAndResolvedIP(t *testing.T) {
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://private.test/secret", http.StatusFound)
	}))
	defer origin.Close()
	client, guarded := sourceTestClient(t, "")
	guarded.base.TLSClientConfig = origin.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	guarded.base.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "93.184.216.34:443" {
			return nil, errors.New("unsafe dial")
		}
		return (&net.Dialer{}).DialContext(ctx, network, origin.Listener.Addr().String())
	}
	if _, err := client.Get("https://example.com/source"); err == nil || !strings.Contains(err.Error(), "private address") {
		t.Fatalf("private redirect = %v", err)
	}
	for _, target := range []string{"https://elsewhere.test/", "http://example.com/", "https://127.0.0.1/"} {
		req, _ := http.NewRequest(http.MethodGet, target, nil)
		if err := client.CheckRedirect(req, nil); err == nil {
			t.Errorf("accepted redirect %s", target)
		}
	}
}

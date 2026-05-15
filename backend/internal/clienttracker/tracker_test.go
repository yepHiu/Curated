package clienttracker

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func requestFrom(remoteAddr string, userAgent string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/health", http.NoBody)
	req.RemoteAddr = remoteAddr
	req.Header.Set("User-Agent", userAgent)
	return req
}

func TestTrackerDeduplicatesByIPAndUserAgent(t *testing.T) {
	base := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	now := base
	tracker := New(
		WithNow(func() time.Time { return now }),
		WithLocalIPs([]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.1.10")}),
	)

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36"
	tracker.Record(requestFrom("192.168.1.20:51001", ua))
	now = base.Add(2 * time.Minute)
	tracker.Record(requestFrom("192.168.1.20:51099", ua))

	clients := tracker.Snapshot()
	if len(clients) != 1 {
		t.Fatalf("client count = %d, want 1", len(clients))
	}
	client := clients[0]
	if client.IP != "192.168.1.20" {
		t.Fatalf("IP = %q, want 192.168.1.20", client.IP)
	}
	if client.RequestCount != 2 {
		t.Fatalf("RequestCount = %d, want 2", client.RequestCount)
	}
	if !client.FirstSeen.Equal(base) {
		t.Fatalf("FirstSeen = %s, want %s", client.FirstSeen, base)
	}
	if !client.LastSeen.Equal(base.Add(2 * time.Minute)) {
		t.Fatalf("LastSeen = %s, want %s", client.LastSeen, base.Add(2*time.Minute))
	}
	if client.AccessKind != AccessKindRemote {
		t.Fatalf("AccessKind = %q, want %q", client.AccessKind, AccessKindRemote)
	}
	if client.IsLocalMachine {
		t.Fatal("remote LAN client should not be marked as local machine")
	}
	if client.Browser != "Chrome" || client.OS != "Windows" || client.DeviceType != DeviceTypeDesktop {
		t.Fatalf("parsed client = browser %q os %q device %q", client.Browser, client.OS, client.DeviceType)
	}
}

func TestTrackerRecognizesCuratedDesktopMarker(t *testing.T) {
	base := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	tracker := New(
		WithNow(func() time.Time { return base }),
		WithLocalIPs([]net.IP{net.ParseIP("127.0.0.1")}),
	)

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36"
	req := requestFrom("127.0.0.1:50001", ua)
	req.Header.Set("X-Curated-Client", "desktop-electron")
	req.Header.Set("X-Curated-Client-Version", "0.0.1-test")

	tracker.Record(req)

	clients := tracker.Snapshot()
	if len(clients) != 1 {
		t.Fatalf("client count = %d, want 1", len(clients))
	}
	client := clients[0]
	if client.Browser != "Curated Desktop" {
		t.Fatalf("Browser = %q, want Curated Desktop", client.Browser)
	}
	if client.BrowserVersion != "0.0.1-test" {
		t.Fatalf("BrowserVersion = %q, want 0.0.1-test", client.BrowserVersion)
	}
	if client.OS != "Windows" {
		t.Fatalf("OS = %q, want Windows", client.OS)
	}
	if client.DeviceType != DeviceTypeDesktop {
		t.Fatalf("DeviceType = %q, want desktop", client.DeviceType)
	}
}

func TestTrackerUsesCuratedDesktopOSHeaders(t *testing.T) {
	base := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	tracker := New(
		WithNow(func() time.Time { return base }),
		WithLocalIPs([]net.IP{net.ParseIP("127.0.0.1")}),
	)

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36"
	req := requestFrom("127.0.0.1:50001", ua)
	req.Header.Set("X-Curated-Client", "desktop-electron")
	req.Header.Set("X-Curated-OS", "Windows")
	req.Header.Set("X-Curated-OS-Version", "11")

	tracker.Record(req)

	clients := tracker.Snapshot()
	if len(clients) != 1 {
		t.Fatalf("client count = %d, want 1", len(clients))
	}
	client := clients[0]
	if client.OS != "Windows" || client.OSVersion != "11" {
		t.Fatalf("OS = %q version %q, want Windows 11", client.OS, client.OSVersion)
	}
}

func TestTrackerUsesClientHintsForWindows11(t *testing.T) {
	base := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	tracker := New(WithNow(func() time.Time { return base }))

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36"
	req := requestFrom("192.168.1.20:50001", ua)
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-CH-UA-Platform-Version", `"13.0.0"`)

	tracker.Record(req)

	clients := tracker.Snapshot()
	if len(clients) != 1 {
		t.Fatalf("client count = %d, want 1", len(clients))
	}
	client := clients[0]
	if client.OS != "Windows" || client.OSVersion != "11" {
		t.Fatalf("OS = %q version %q, want Windows 11", client.OS, client.OSVersion)
	}
}

func TestTrackerSeparatesCuratedDesktopFromChromeWithSameUserAgent(t *testing.T) {
	base := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	now := base
	tracker := New(WithNow(func() time.Time { return now }))
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36"

	tracker.Record(requestFrom("127.0.0.1:50001", ua))
	now = base.Add(time.Second)
	desktopReq := requestFrom("127.0.0.1:50002", ua)
	desktopReq.Header.Set("X-Curated-Client", "desktop-electron")
	tracker.Record(desktopReq)

	clients := tracker.Snapshot()
	if len(clients) != 2 {
		t.Fatalf("client count = %d, want 2", len(clients))
	}
	browsers := map[string]bool{}
	for _, client := range clients {
		browsers[client.Browser] = true
	}
	if !browsers["Chrome"] || !browsers["Curated Desktop"] {
		t.Fatalf("browsers = %#v, want Chrome and Curated Desktop", browsers)
	}
}

func TestTrackerSeparatesSameIPDifferentUserAgentAndIdentifiesLocalMachine(t *testing.T) {
	base := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	now := base
	tracker := New(
		WithNow(func() time.Time { return now }),
		WithLocalIPs([]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.1.10")}),
	)

	tracker.Record(requestFrom("127.0.0.1:50001", "curl/8.7.1"))
	now = base.Add(time.Second)
	tracker.Record(requestFrom("127.0.0.1:50002", "python-requests/2.32.0"))

	clients := tracker.Snapshot()
	if len(clients) != 2 {
		t.Fatalf("client count = %d, want 2", len(clients))
	}
	for _, client := range clients {
		if client.AccessKind != AccessKindLocal {
			t.Fatalf("AccessKind = %q, want local", client.AccessKind)
		}
		if !client.IsLocalMachine {
			t.Fatal("loopback clients should be marked as local machine")
		}
		if client.DeviceType != DeviceTypeTool {
			t.Fatalf("DeviceType = %q, want tool", client.DeviceType)
		}
	}
}

func TestTrackerCapsSnapshotsToMostRecentClients(t *testing.T) {
	base := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	now := base
	tracker := New(
		WithMaxEntries(2),
		WithNow(func() time.Time { return now }),
	)

	tracker.Record(requestFrom("192.168.1.21:1", "curl/8.1.0"))
	now = base.Add(time.Second)
	tracker.Record(requestFrom("192.168.1.22:1", "curl/8.2.0"))
	now = base.Add(2 * time.Second)
	tracker.Record(requestFrom("192.168.1.23:1", "curl/8.3.0"))

	clients := tracker.Snapshot()
	if len(clients) != 2 {
		t.Fatalf("client count = %d, want 2", len(clients))
	}
	if clients[0].IP != "192.168.1.23" || clients[1].IP != "192.168.1.22" {
		t.Fatalf("snapshot order/IPs = %q, %q; want newest two clients", clients[0].IP, clients[1].IP)
	}
}

func TestTrackerIgnoresCORSPreflight(t *testing.T) {
	tracker := New()
	req := requestFrom("192.168.1.20:51001", "curl/8.7.1")
	req.Method = http.MethodOptions

	tracker.Record(req)

	if clients := tracker.Snapshot(); len(clients) != 0 {
		t.Fatalf("client count = %d, want 0", len(clients))
	}
}

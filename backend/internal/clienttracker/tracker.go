package clienttracker

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AccessKind string

const (
	AccessKindLocal  AccessKind = "local"
	AccessKindRemote AccessKind = "remote"
)

const defaultMaxEntries = 50

type ClientSnapshot struct {
	Key            string
	RemoteAddr     string
	IP             string
	Port           int
	UserAgent      string
	Browser        string
	BrowserVersion string
	OS             string
	OSVersion      string
	DeviceType     DeviceType
	AccessKind     AccessKind
	IsLocalMachine bool
	Hostname       string
	FirstSeen      time.Time
	LastSeen       time.Time
	RequestCount   int64
}

type clientRecord struct {
	ClientSnapshot
}

type Tracker struct {
	mu         sync.RWMutex
	clients    map[string]*clientRecord
	localIPs   map[string]struct{}
	maxEntries int
	now        func() time.Time
}

type Option func(*Tracker)

func New(options ...Option) *Tracker {
	tracker := &Tracker{
		clients:    make(map[string]*clientRecord),
		localIPs:   discoverLocalIPs(),
		maxEntries: defaultMaxEntries,
		now:        time.Now,
	}
	for _, option := range options {
		option(tracker)
	}
	if tracker.maxEntries <= 0 {
		tracker.maxEntries = defaultMaxEntries
	}
	if tracker.now == nil {
		tracker.now = time.Now
	}
	if tracker.localIPs == nil {
		tracker.localIPs = map[string]struct{}{}
	}
	return tracker
}

func WithNow(now func() time.Time) Option {
	return func(t *Tracker) {
		if now != nil {
			t.now = now
		}
	}
}

func WithLocalIPs(ips []net.IP) Option {
	return func(t *Tracker) {
		t.localIPs = map[string]struct{}{}
		for _, ip := range ips {
			if ip == nil {
				continue
			}
			t.localIPs[ip.String()] = struct{}{}
		}
	}
}

func WithMaxEntries(maxEntries int) Option {
	return func(t *Tracker) {
		t.maxEntries = maxEntries
	}
}

func (t *Tracker) Record(r *http.Request) {
	if t == nil || r == nil || r.Method == http.MethodOptions {
		return
	}

	ip, port, ok := parseRemoteAddr(r.RemoteAddr)
	if !ok {
		return
	}
	userAgent := strings.TrimSpace(r.UserAgent())
	if userAgent == "" {
		userAgent = "Unknown Client"
	}
	key := clientKey(ip, userAgent)
	now := t.now().UTC()
	parsed := ParseUserAgent(userAgent)
	accessKind := AccessKindRemote
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil && parsedIP.IsLoopback() {
		accessKind = AccessKindLocal
	}
	_, knownLocalIP := t.localIPs[ip]
	isLocalMachine := knownLocalIP || accessKind == AccessKindLocal

	t.mu.Lock()
	defer t.mu.Unlock()

	record, exists := t.clients[key]
	if !exists {
		t.clients[key] = &clientRecord{
			ClientSnapshot: ClientSnapshot{
				Key:            key,
				RemoteAddr:     r.RemoteAddr,
				IP:             ip,
				Port:           port,
				UserAgent:      userAgent,
				Browser:        parsed.Browser,
				BrowserVersion: parsed.BrowserVersion,
				OS:             parsed.OS,
				OSVersion:      parsed.OSVersion,
				DeviceType:     parsed.DeviceType,
				AccessKind:     accessKind,
				IsLocalMachine: isLocalMachine,
				FirstSeen:      now,
				LastSeen:       now,
				RequestCount:   1,
			},
		}
		t.trimOldestLocked()
		return
	}

	record.RemoteAddr = r.RemoteAddr
	record.Port = port
	record.Browser = parsed.Browser
	record.BrowserVersion = parsed.BrowserVersion
	record.OS = parsed.OS
	record.OSVersion = parsed.OSVersion
	record.DeviceType = parsed.DeviceType
	record.AccessKind = accessKind
	record.IsLocalMachine = isLocalMachine
	record.LastSeen = now
	record.RequestCount += 1
	t.trimOldestLocked()
}

func (t *Tracker) Snapshot() []ClientSnapshot {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()

	out := make([]ClientSnapshot, 0, len(t.clients))
	for _, record := range t.clients {
		out = append(out, record.ClientSnapshot)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastSeen.After(out[j].LastSeen)
	})
	return out
}

func (t *Tracker) trimOldestLocked() {
	if len(t.clients) <= t.maxEntries {
		return
	}
	records := make([]ClientSnapshot, 0, len(t.clients))
	for _, record := range t.clients {
		records = append(records, record.ClientSnapshot)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].LastSeen.Before(records[j].LastSeen)
	})
	for len(t.clients) > t.maxEntries && len(records) > 0 {
		delete(t.clients, records[0].Key)
		records = records[1:]
	}
}

func parseRemoteAddr(remoteAddr string) (string, int, bool) {
	trimmed := strings.TrimSpace(remoteAddr)
	if trimmed == "" {
		return "", 0, false
	}
	host, portText, err := net.SplitHostPort(trimmed)
	if err != nil {
		host = strings.Trim(trimmed, "[]")
		portText = ""
	}
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "" {
		return "", 0, false
	}
	port := 0
	if portText != "" {
		if parsed, err := strconv.Atoi(portText); err == nil && parsed > 0 {
			port = parsed
		}
	}
	return host, port, true
}

func clientKey(ip string, userAgent string) string {
	sum := sha256.Sum256([]byte(ip + "\x00" + userAgent))
	return hex.EncodeToString(sum[:16])
}

func discoverLocalIPs() map[string]struct{} {
	out := map[string]struct{}{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil {
				out[ip.String()] = struct{}{}
			}
		}
	}
	return out
}

package core

import (
	"encoding/json"
	"net/url"
	"strings"
	"sync"
)

// SourceURLStore remembers https source URLs seen this turn for get_source_page.
type SourceURLStore struct {
	mu     sync.Mutex
	bySess map[string]*sourceURLBucket
}

type sourceURLBucket struct {
	urls  map[string]struct{}
	hosts map[string]struct{}
}

func NewSourceURLStore() *SourceURLStore {
	return &SourceURLStore{bySess: map[string]*sourceURLBucket{}}
}

func (s *SourceURLStore) Reset(sessionID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.bySess, sessionKey(sessionID))
}

func (s *SourceURLStore) Remember(sessionID string, rawURLs []string) {
	if s == nil || len(rawURLs) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionKey(sessionID)
	bucket := s.bySess[key]
	if bucket == nil {
		bucket = &sourceURLBucket{urls: map[string]struct{}{}, hosts: map[string]struct{}{}}
		s.bySess[key] = bucket
	}
	for _, raw := range rawURLs {
		normalized, host, ok := NormalizeSourceURL(raw)
		if !ok {
			continue
		}
		bucket.urls[normalized] = struct{}{}
		bucket.hosts[host] = struct{}{}
	}
}

func (s *SourceURLStore) Known(sessionID, raw string) bool {
	if s == nil {
		return false
	}
	normalized, _, ok := NormalizeSourceURL(raw)
	if !ok {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket := s.bySess[sessionKey(sessionID)]
	if bucket == nil {
		return false
	}
	_, exists := bucket.urls[normalized]
	return exists
}

func (s *SourceURLStore) HostKnown(sessionID, host string) bool {
	if s == nil {
		return false
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket := s.bySess[sessionKey(sessionID)]
	if bucket == nil {
		return false
	}
	_, exists := bucket.hosts[host]
	return exists
}

func (s *SourceURLStore) Hosts(sessionID string) []string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket := s.bySess[sessionKey(sessionID)]
	if bucket == nil {
		return nil
	}
	out := make([]string, 0, len(bucket.hosts))
	for host := range bucket.hosts {
		out = append(out, host)
	}
	return out
}

// NormalizeSourceURL accepts https URLs only and returns a comparable form plus hostname.
func NormalizeSourceURL(raw string) (normalized, host string, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return "", "", false
	}
	if !strings.EqualFold(parsed.Scheme, "https") || parsed.Opaque != "" || parsed.User != nil {
		return "", "", false
	}
	host = strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return "", "", false
	}
	parsed.Scheme = "https"
	parsed.Fragment = ""
	port := parsed.Port()
	if port != "" && port != "443" {
		parsed.Host = host + ":" + port
	} else {
		parsed.Host = host
	}
	return parsed.String(), host, true
}

// ExtractSourceURLs collects homepage and externalLinks https URLs from a tool envelope.
func ExtractSourceURLs(result Result) []string {
	if !result.OK || result.Data == nil {
		return nil
	}
	raw, err := json.Marshal(result.Data)
	if err != nil {
		return nil
	}
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil
	}
	var out []string
	collectSourceURLs(unwrapSource(root), &out)
	return uniqueStrings(out)
}

func collectSourceURLs(value any, out *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		if homepage := stringField(typed, "homepage"); homepage != "" {
			*out = append(*out, homepage)
		}
		*out = append(*out, stringSliceField(typed, "externalLinks")...)
		for _, child := range typed {
			collectSourceURLs(child, out)
		}
	case []any:
		for _, child := range typed {
			collectSourceURLs(child, out)
		}
	}
}

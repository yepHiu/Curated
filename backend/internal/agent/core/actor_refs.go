package core

import (
	"encoding/json"
	"strings"
	"sync"
)

// ActorRefStore remembers actor names returned this turn so provider search
// cannot take an arbitrary query string.
type ActorRefStore struct {
	mu     sync.Mutex
	byName map[string]map[string]struct{}
}

func NewActorRefStore() *ActorRefStore {
	return &ActorRefStore{byName: map[string]map[string]struct{}{}}
}

func (s *ActorRefStore) Reset(sessionID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byName, sessionKey(sessionID))
}

func (s *ActorRefStore) Remember(sessionID string, names []string) {
	if s == nil || len(names) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionKey(sessionID)
	bucket := s.byName[key]
	if bucket == nil {
		bucket = map[string]struct{}{}
		s.byName[key] = bucket
	}
	for _, name := range names {
		normalized := normalizeActorKey(name)
		if normalized == "" {
			continue
		}
		bucket[normalized] = struct{}{}
	}
}

func (s *ActorRefStore) Known(sessionID, name string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket := s.byName[sessionKey(sessionID)]
	if bucket == nil {
		return false
	}
	_, ok := bucket[normalizeActorKey(name)]
	return ok
}

func normalizeActorKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}

// ExtractActorNames collects actor names from query tool envelopes.
func ExtractActorNames(result Result) []string {
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
	source := unwrapSource(root)
	var names []string
	if items := asObjectSlice(source, "items"); len(items) > 0 {
		for _, item := range items {
			names = appendActorNames(names, item)
		}
		return uniqueStrings(names)
	}
	if obj, ok := source.(map[string]any); ok {
		return uniqueStrings(appendActorNames(nil, obj))
	}
	return nil
}

func appendActorNames(dst []string, obj map[string]any) []string {
	if name := stringField(obj, "name"); name != "" {
		dst = append(dst, name)
	}
	dst = append(dst, stringSliceField(obj, "actors")...)
	dst = append(dst, stringSliceField(obj, "aliases")...)
	return dst
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := normalizeActorKey(value)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, strings.TrimSpace(value))
	}
	return out
}

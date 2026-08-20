package core

import (
	"encoding/json"
	"strings"
	"sync"
)

// MovieRef is a display projection of a movie already retrieved in this turn.
type MovieRef struct {
	ID       string
	Title    string
	Code     string
	CoverURL string
	ThumbURL string
	Actors   []string
	Reason   string
}

// MovieRefStore remembers movies returned by query tools so present_movies
// cannot invent IDs. It is in-memory and scoped to one agent turn.
type MovieRefStore struct {
	mu   sync.Mutex
	byID map[string]map[string]MovieRef
}

func NewMovieRefStore() *MovieRefStore {
	return &MovieRefStore{byID: map[string]map[string]MovieRef{}}
}

func (s *MovieRefStore) Reset(sessionID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byID, sessionKey(sessionID))
}

func (s *MovieRefStore) Remember(sessionID string, refs []MovieRef) {
	if s == nil || len(refs) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionKey(sessionID)
	bucket := s.byID[key]
	if bucket == nil {
		bucket = map[string]MovieRef{}
		s.byID[key] = bucket
	}
	for _, ref := range refs {
		id := strings.TrimSpace(ref.ID)
		if id == "" {
			continue
		}
		ref.ID = id
		if prev, ok := bucket[id]; ok {
			ref = mergeMovieRef(prev, ref)
		}
		bucket[id] = ref
	}
}

func (s *MovieRefStore) Lookup(sessionID string, ids []string) (found []MovieRef, missing []string) {
	if s == nil {
		return nil, append([]string{}, ids...)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket := s.byID[sessionKey(sessionID)]
	seen := map[string]bool{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ref, ok := bucket[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		found = append(found, ref)
	}
	return found, missing
}

func sessionKey(sessionID string) string {
	if strings.TrimSpace(sessionID) == "" {
		return "_"
	}
	return sessionID
}

func mergeMovieRef(prev, next MovieRef) MovieRef {
	out := prev
	if next.Title != "" {
		out.Title = next.Title
	}
	if next.Code != "" {
		out.Code = next.Code
	}
	if next.CoverURL != "" {
		out.CoverURL = next.CoverURL
	}
	if next.ThumbURL != "" {
		out.ThumbURL = next.ThumbURL
	}
	if len(next.Actors) > 0 {
		out.Actors = next.Actors
	}
	if next.Reason != "" {
		out.Reason = next.Reason
	}
	return out
}

// ExtractMovieRefs collects movie cards from a tool result envelope.
func ExtractMovieRefs(result Result) []MovieRef {
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
	if items := asObjectSlice(source, "items"); len(items) > 0 {
		out := make([]MovieRef, 0, len(items))
		for _, item := range items {
			if ref, ok := movieRefFromMap(item); ok {
				out = append(out, ref)
			}
		}
		return out
	}
	if obj, ok := source.(map[string]any); ok {
		if ref, ok := movieRefFromMap(obj); ok {
			return []MovieRef{ref}
		}
	}
	return nil
}

func unwrapSource(value any) any {
	obj, ok := value.(map[string]any)
	if !ok {
		return value
	}
	if source, exists := obj["source"]; exists && source != nil {
		return source
	}
	return obj
}

func asObjectSlice(value any, key string) []map[string]any {
	obj, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := obj[key].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		child, ok := item.(map[string]any)
		if ok {
			out = append(out, child)
		}
	}
	return out
}

func movieRefFromMap(obj map[string]any) (MovieRef, bool) {
	id := stringField(obj, "id")
	if id == "" {
		id = stringField(obj, "movieId")
	}
	if id == "" {
		return MovieRef{}, false
	}
	actors := stringSliceField(obj, "actors")
	if len(actors) > 5 {
		actors = actors[:5]
	}
	return MovieRef{
		ID:       id,
		Title:    stringField(obj, "title"),
		Code:     stringField(obj, "code"),
		CoverURL: stringField(obj, "coverUrl"),
		ThumbURL: stringField(obj, "thumbUrl"),
		Actors:   actors,
		Reason:   stringField(obj, "reason"),
	}, true
}

func stringField(obj map[string]any, key string) string {
	value, ok := obj[key]
	if !ok || value == nil {
		return ""
	}
	s, _ := value.(string)
	return strings.TrimSpace(s)
}

func stringSliceField(obj map[string]any, key string) []string {
	raw, ok := obj[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, _ := item.(string)
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

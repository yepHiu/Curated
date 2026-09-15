package core

import (
	"encoding/json"
	"strings"
	"sync"
)

// BookRef is a display projection of a comic or photo book already retrieved in this turn.
type BookRef struct {
	Kind     string
	ID       string
	Title    string
	CoverURL string
	Tags     []string
}

// BookRefStore remembers books returned by query tools so present_comics / present_photos
// cannot invent IDs. It is in-memory and scoped to one agent turn.
type BookRefStore struct {
	mu   sync.Mutex
	byID map[string]map[string]BookRef
}

// NewBookRefStore constructs an empty per-turn book reference store.
func NewBookRefStore() *BookRefStore {
	return &BookRefStore{byID: map[string]map[string]BookRef{}}
}

// bookRefKey 用 kind+id 区分漫画和写真，避免同数字 ID 互相覆盖。
func bookRefKey(kind, id string) string {
	return strings.TrimSpace(kind) + ":" + strings.TrimSpace(id)
}

// Reset forgets every book remembered for one request scope.
func (s *BookRefStore) Reset(sessionID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byID, sessionKey(sessionID))
}

// Remember stores or merges book cards for later present_* lookups.
func (s *BookRefStore) Remember(sessionID string, refs []BookRef) {
	if s == nil || len(refs) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionKey(sessionID)
	bucket := s.byID[key]
	if bucket == nil {
		bucket = map[string]BookRef{}
		s.byID[key] = bucket
	}
	for _, ref := range refs {
		id := strings.TrimSpace(ref.ID)
		kind := strings.TrimSpace(ref.Kind)
		if id == "" || (kind != "comic" && kind != "photo") {
			continue
		}
		ref.ID = id
		ref.Kind = kind
		storeKey := bookRefKey(kind, id)
		if prev, ok := bucket[storeKey]; ok {
			ref = mergeBookRef(prev, ref)
		}
		bucket[storeKey] = ref
	}
}

// Lookup returns remembered books of one kind, preserving request order.
func (s *BookRefStore) Lookup(sessionID, kind string, ids []string) (found []BookRef, missing []string) {
	if s == nil {
		return nil, append([]string{}, ids...)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket := s.byID[sessionKey(sessionID)]
	seen := map[string]bool{}
	kind = strings.TrimSpace(kind)
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ref, ok := bucket[bookRefKey(kind, id)]
		if !ok {
			missing = append(missing, id)
			continue
		}
		found = append(found, ref)
	}
	return found, missing
}

// mergeBookRef 用后一次检索补全标题、封面和标签，不覆盖空值。
func mergeBookRef(prev, next BookRef) BookRef {
	out := prev
	if next.Title != "" {
		out.Title = next.Title
	}
	if next.CoverURL != "" {
		out.CoverURL = next.CoverURL
	}
	if len(next.Tags) > 0 {
		out.Tags = next.Tags
	}
	return out
}

// ExtractBookRefs collects comic/photo cards from a tool result envelope.
// It never treats movieId or a bare id field as a book identity.
func ExtractBookRefs(result Result) []BookRef {
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
		out := make([]BookRef, 0, len(items))
		for _, item := range items {
			if ref, ok := bookRefFromMap(item); ok {
				out = append(out, ref)
			}
		}
		return out
	}
	if obj, ok := source.(map[string]any); ok {
		if ref, ok := bookRefFromMap(obj); ok {
			return []BookRef{ref}
		}
	}
	return nil
}

// bookRefFromMap 只认 comicId/photoId，忽略裸 id 或 movieId。
func bookRefFromMap(obj map[string]any) (BookRef, bool) {
	kind := stringField(obj, "kind")
	comicID := stringField(obj, "comicId")
	photoID := stringField(obj, "photoId")
	id := ""
	switch {
	case kind == "comic" && comicID != "":
		id = comicID
	case kind == "photo" && photoID != "":
		id = photoID
	case comicID != "" && photoID == "":
		kind = "comic"
		id = comicID
	case photoID != "" && comicID == "":
		kind = "photo"
		id = photoID
	default:
		return BookRef{}, false
	}
	tags := stringSliceField(obj, "tags")
	if len(tags) > 8 {
		tags = tags[:8]
	}
	return BookRef{
		Kind:     kind,
		ID:       id,
		Title:    stringField(obj, "title"),
		CoverURL: stringField(obj, "coverUrl"),
		Tags:     tags,
	}, true
}

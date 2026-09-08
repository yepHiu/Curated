package core

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const SubmitAnswerName = "submit_answer"
const maxAnswerRefs = 512

// AnswerRef is an immutable snapshot of ONE successful read, never a merge of
// fields from different sources. It grants no permission to modify a movie.
type AnswerRef struct {
	RefID       string         `json:"refId"`
	MovieID     string         `json:"movieId,omitempty"`
	Source      string         `json:"source"`
	Tool        string         `json:"tool"`
	RetrievedAt string         `json:"retrievedAt"`
	Truncated   bool           `json:"truncated,omitempty"`
	Fields      map[string]any `json:"fields"`
}

// AnswerRefHint identifies a result without exposing fields hidden by minimal
// projection. Model-selected field names are checked against the full snapshot.
type AnswerRefHint struct {
	RefID   string   `json:"refId"`
	MovieID string   `json:"movieId,omitempty"`
	Code    string   `json:"code,omitempty"`
	Source  string   `json:"source"`
	Fields  []string `json:"fields"`
}

type AnswerRefStore struct {
	mu     sync.Mutex
	prefix string
	refs   []AnswerRef
}

type answerContextKey struct{}

func NewAnswerRefStore() *AnswerRefStore {
	return &AnswerRefStore{prefix: "ref_" + rand.Text() + "_"}
}

func WithAnswerRefs(ctx context.Context, refs *AnswerRefStore) context.Context {
	return context.WithValue(ctx, answerContextKey{}, refs)
}

func AnswerRefsFromContext(ctx context.Context) *AnswerRefStore {
	refs, _ := ctx.Value(answerContextKey{}).(*AnswerRefStore)
	return refs
}

// Capture uses an explicit tool/envelope allowlist. Presentation, writes,
// arbitrary source-page text and unresolved candidates are never evidence.
// Call before model privacy projection; only hints are sent to the model.
func (s *AnswerRefStore) Capture(name string, result Result) []AnswerRefHint {
	if s == nil || !result.OK || result.Error != nil || result.Data == nil {
		return nil
	}
	switch name {
	case "search_movies", "get_movie_detail", "get_watch_history", "resolve_entities", SearchProviderTitlesName:
	default:
		return nil
	}
	raw, err := json.Marshal(result.Data)
	if err != nil {
		return nil
	}
	var envelope struct {
		Source map[string]any `json:"source"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Source == nil {
		return nil
	}
	obj := envelope.Source
	var rows []map[string]any
	switch name {
	case "get_movie_detail":
		rows = []map[string]any{obj}
	case "resolve_entities":
		if stringField(obj, "status") != "matched" || stringField(obj, "kind") != "movie" {
			return nil
		}
		rows = asObjectSlice(obj, "candidates")
		if len(rows) != 1 {
			return nil
		}
	default:
		rows = asObjectSlice(obj, "items")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var hints []AnswerRefHint
	for _, row := range rows {
		if len(s.refs) >= maxAnswerRefs {
			break
		}
		movieID := stringField(row, "movieId")
		if movieID == "" && name != SearchProviderTitlesName {
			movieID = stringField(row, "id")
		}
		code := stringField(row, "code")
		source := "local"
		if name == SearchProviderTitlesName {
			source = "provider"
			if code == "" {
				continue
			}
			if inLibrary, _ := row["inLibrary"].(bool); !inLibrary {
				movieID = ""
			}
		} else if movieID == "" || (code == "" && stringField(row, "title") == "") {
			continue
		}
		fields := make(map[string]any)
		for _, key := range []string{"code", "title", "actors", "runtimeMinutes", "tags", "userRating", "metadataRating", "metadataProvider", "isFavorite", "year", "studio", "releaseDate", "coverUrl", "thumbUrl", "homepage", "provider", "score", "inLibrary", "playState"} {
			v, exists := row[key]
			if !exists || v == nil || v == "" {
				continue
			}
			// Legacy DTOs use zero for absent runtime/year/site score/rating.
			if n, ok := v.(float64); ok && n == 0 {
				continue
			}
			if arr, ok := v.([]any); ok && len(arr) == 0 {
				continue
			}
			fields[key] = v
		}
		ref := AnswerRef{RefID: fmt.Sprintf("%s%d", s.prefix, len(s.refs)+1), MovieID: movieID, Source: source, Tool: name, RetrievedAt: time.Now().UTC().Format(time.RFC3339Nano), Truncated: result.Truncated, Fields: fields}
		s.refs = append(s.refs, ref)
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		hints = append(hints, AnswerRefHint{RefID: ref.RefID, MovieID: movieID, Code: code, Source: source, Fields: keys})
	}
	return hints
}

func cloneAnswerRef(ref AnswerRef) AnswerRef {
	raw, _ := json.Marshal(ref)
	var copy AnswerRef
	_ = json.Unmarshal(raw, &copy)
	return copy
}

func (s *AnswerRefStore) Lookup(id string) (AnswerRef, bool) {
	if s == nil {
		return AnswerRef{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ref := range s.refs {
		if ref.RefID == id {
			return cloneAnswerRef(ref), true
		}
	}
	return AnswerRef{}, false
}

// LocalMovie intentionally excludes provider hits, even when reconciled with
// a local ID. A local card requires the local record's own facts.
func (s *AnswerRefStore) LocalMovie(id string) (AnswerRef, bool) {
	if s == nil {
		return AnswerRef{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.refs) - 1; i >= 0; i-- {
		if ref := s.refs[i]; ref.MovieID == strings.TrimSpace(id) && ref.Source == "local" {
			return cloneAnswerRef(ref), true
		}
	}
	return AnswerRef{}, false
}

func (s *AnswerRefStore) All() []AnswerRef {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AnswerRef, len(s.refs))
	for i, ref := range s.refs {
		out[i] = cloneAnswerRef(ref)
	}
	return out
}

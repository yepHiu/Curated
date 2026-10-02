package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/config"
	"curated-backend/internal/llm"
	"curated-backend/internal/scraper"
	"curated-backend/internal/storage"
)

type topicScaleRequest struct {
	Existing   []storage.TopicDefinition `json:"existing"`
	Vocabulary []storage.TopicDefinition `json:"vocabulary"`
	Movies     []storage.TopicMovieInput `json:"movies"`
}

func TestTagOrganizationPagesOversizedExistingVocabulary(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 6)
	ctx := context.Background()
	defs := []storage.TopicDefinition{}
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("Theme-%02d", i)
		if i == 49 {
			name = "Theme"
		}
		defs = append(defs, storage.TopicDefinition{Name: name, Description: strings.Repeat("字", 300)})
	}
	if err := a.store.SaveTopicVocabulary(ctx, defs); err != nil {
		t.Fatal(err)
	}
	seen := map[string]map[string]bool{}
	scaleTopicProvider(t, a, func(vocabulary bool, source topicScaleRequest) (any, int) {
		if vocabulary {
			return map[string]any{"topics": []any{}}, 200
		}
		movies := []topicClassification{}
		for _, input := range source.Movies {
			if seen[input.MovieID] == nil {
				seen[input.MovieID] = map[string]bool{}
			}
			movie := topicClassification{MovieID: input.MovieID, Matches: []topicMatch{}}
			for _, def := range source.Vocabulary {
				if seen[input.MovieID][def.Name] {
					t.Error("vocabulary page replayed")
				}
				seen[input.MovieID][def.Name] = true
				if def.Name == "Theme" {
					movie.Matches = append(movie.Matches, topicMatch{Topic: "Theme", Field: "metadataTags", Quote: "Theme"})
				}
			}
			movies = append(movies, movie)
		}
		return map[string]any{"movies": movies}, 200
	})
	id := createScaleTopicJob(t, a, ids)
	a.runTagOrganization(ctx, id)
	job, err := a.store.GetTagOrganization(ctx, id)
	if err != nil || job.Status != "completed" || job.Succeeded != 6 || !job.VocabularyReady {
		t.Fatalf("vocabulary overflow poisoned task: %+v err=%v", job, err)
	}
	for _, mid := range ids {
		if len(seen[mid]) != 50 {
			t.Fatalf("movie %s only saw %d definitions", mid, len(seen[mid]))
		}
	}
}

// A local HTTP provider exercises the real client, JSON validation and SQLite writes.
func scaleTopicProvider(t *testing.T, a *App, reply func(bool, topicScaleRequest) (any, int)) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []llm.ChatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Messages) != 2 {
			t.Error("invalid completion request")
			http.Error(w, "fixture error", 400)
			return
		}
		var source topicScaleRequest
		raw := strings.TrimSuffix(strings.TrimPrefix(req.Messages[1].Content, "<source>"), "</source>")
		if err := json.Unmarshal([]byte(raw), &source); err != nil {
			t.Error(err)
		}
		if strings.Contains(req.Messages[0].Content, "topic reuse v") {
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": `{"matches":[]}`}}}})
			return
		}
		result, status := reply(strings.Contains(req.Messages[0].Content, "vocabulary v"), source)
		if status != 200 {
			http.Error(w, "synthetic provider failure", status)
			return
		}
		content, _ := json.Marshal(result)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": string(content)}}}})
	}))
	t.Cleanup(server.Close)
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "local-scale-fixture", ContextWindow: 32768}
	a.cfg.AIGovernance.WritePerMinute = 10000
}

func scaleTopicClassification(source topicScaleRequest) any {
	movies := []topicClassification{}
	for _, input := range source.Movies {
		movies = append(movies, topicClassification{MovieID: input.MovieID, Matches: []topicMatch{{Topic: "Theme", Field: "metadataTags", Quote: "Theme"}}})
	}
	return map[string]any{"movies": movies}
}

func createScaleTopicJob(t *testing.T, a *App, ids []string) string {
	t.Helper()
	id, err := a.store.CreateTagOrganization(context.Background(), "scale-job", "scale-request", "manual", "zh-CN", ids)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestTagOrganization1200VocabularyGrowthAndRestart(t *testing.T) {
	testTagOrganizationScaleAndRestart(t, 1200)
}

// The 10,000-movie write/audit run is opt-in; the 1,200-movie regression runs in CI.
func TestTagOrganization10000VocabularyGrowthAndRestart(t *testing.T) {
	if os.Getenv("CURATED_TAG_SCALE_TEST") != "1" {
		t.Skip("set CURATED_TAG_SCALE_TEST=1 for the 10,000-movie end-to-end fixture")
	}
	testTagOrganizationScaleAndRestart(t, 10000)
}

func testTagOrganizationScaleAndRestart(t *testing.T, count int) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, count)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var vocabCalls, classificationCalls atomic.Int64
	scaleTopicProvider(t, a, func(vocabulary bool, source topicScaleRequest) (any, int) {
		if vocabulary {
			vocabCalls.Add(1)
			defs := []storage.TopicDefinition{}
			start := int(vocabCalls.Load()-1) * 12
			for i := start; i < min(start+12, 240); i++ {
				name := fmt.Sprintf("Theme-%03d", i)
				if i == 0 {
					name = "Theme"
				}
				defs = append(defs, storage.TopicDefinition{Name: name, Description: "Synthetic subject", Aliases: []string{}})
			}
			return map[string]any{"topics": defs}, 200
		}
		if classificationCalls.Add(1) == 9 {
			cancel() // Service shutdown while a request is in flight, after 40 committed movies.
		}
		return scaleTopicClassification(source), 200
	})
	id := createScaleTopicJob(t, a, ids)
	a.runTagOrganization(ctx, id)
	job, err := a.store.GetTagOrganization(context.Background(), id)
	if err != nil || job.Status != "running" || job.Succeeded != 40 || !job.VocabularyReady || job.VocabularyProcessed != count {
		t.Fatalf("interrupted job=%+v err=%v", job, err)
	}
	defs, err := a.store.GetTagOrganizationVocabulary(context.Background(), id)
	if err != nil || len(defs) != 240 || vocabCalls.Load() != int64(count/50) {
		t.Fatalf("vocabulary size=%d calls=%d err=%v", len(defs), vocabCalls.Load(), err)
	}
	before := map[string]int64{}
	for _, mid := range ids {
		input, err := a.store.TopicMovieInput(context.Background(), mid)
		if err != nil {
			t.Fatal(err)
		}
		before[mid] = input.Revision
	}
	// A fresh runtime uses only persisted job state, not any earlier in-memory cursor.
	restarted := &App{store: a.store, cfg: a.cfg, librarySettingsPath: a.librarySettingsPath}
	restarted.runTagOrganization(context.Background(), id)
	job, err = restarted.store.GetTagOrganization(context.Background(), id)
	if err != nil || job.Status != "completed" || job.Succeeded != count || job.Failed != 0 {
		t.Fatalf("resumed job=%+v err=%v", job, err)
	}
	if vocabCalls.Load() != int64(count/50) || classificationCalls.Load() != int64(count/5+1) {
		t.Fatalf("repeated checkpoints: vocabulary=%d classification=%d", vocabCalls.Load(), classificationCalls.Load())
	}
	unchanged := 0
	for _, mid := range ids {
		input, err := a.store.TopicMovieInput(context.Background(), mid)
		if err != nil || !reflect.DeepEqual(input.MetadataTags, []string{"Theme"}) {
			t.Fatalf("source tags changed: %+v err=%v", input, err)
		}
		if input.Revision == before[mid] {
			unchanged++
		} else if input.Revision != before[mid]+1 {
			t.Fatal("movie was written more than once")
		}
	}
	if unchanged != 40 {
		t.Fatalf("successful checkpoints were replayed: unchanged=%d", unchanged)
	}
}

func TestTagOrganizationInvalidEvidenceIsolatesMovie(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 11)
	badID := ids[4]
	scaleTopicProvider(t, a, func(vocabulary bool, source topicScaleRequest) (any, int) {
		if vocabulary {
			return map[string]any{"topics": []storage.TopicDefinition{{Name: "Theme"}}}, 200
		}
		result := scaleTopicClassification(source).(map[string]any)
		for i := range result["movies"].([]topicClassification) {
			movie := &result["movies"].([]topicClassification)[i]
			if movie.MovieID == badID {
				movie.Matches[0].Quote = "fabricated evidence"
			}
		}
		return result, 200
	})
	id := createScaleTopicJob(t, a, ids)
	a.runTagOrganization(context.Background(), id)
	job, err := a.store.GetTagOrganization(context.Background(), id)
	if err != nil || job.Status != "partial_failed" || job.Succeeded != 10 || job.Failed != 1 {
		t.Fatalf("bad evidence poisoned other movies: %+v err=%v", job, err)
	}
	input, err := a.store.TopicMovieInput(context.Background(), badID)
	if err != nil || !reflect.DeepEqual(input.UserTags, []string{"Alias", "Personal note"}) {
		t.Fatalf("invalid evidence written: %+v err=%v", input, err)
	}
}

func TestTagOrganizationCheckpointFailureBlocksAndResumes(t *testing.T) {
	for _, status := range []string{"succeeded", "unresolved", "failed"} {
		t.Run(status, func(t *testing.T) {
			ctx := context.Background()
			dbPath := filepath.Join(t.TempDir(), "checkpoint.db")
			store, err := storage.NewSQLiteStore(dbPath)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			if err := store.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			a := &App{store: store, cfg: enabledAITestConfig()}
			ids, _ := seedTopicMovies(t, a, 6)
			faultDB, err := sql.Open("sqlite", dbPath)
			if err != nil {
				t.Fatal(err)
			}
			defer faultDB.Close()
			// The injected error leaves reads/job-state writes available, as with a
			// failed per-movie checkpoint, and must never spin on pending forever.
			_, err = faultDB.Exec(`CREATE TRIGGER reject_checkpoint BEFORE UPDATE OF status ON ai_tag_organization_items
			WHEN NEW.status='` + status + `' BEGIN SELECT RAISE(ABORT, 'synthetic checkpoint failure'); END`)
			if err != nil {
				t.Fatal(err)
			}
			var repaired atomic.Bool
			scaleTopicProvider(t, a, func(vocabulary bool, source topicScaleRequest) (any, int) {
				if vocabulary {
					return map[string]any{"topics": []storage.TopicDefinition{{Name: "Theme"}}}, 200
				}
				if status == "failed" && !repaired.Load() {
					return map[string]any{"movies": []any{}}, 200
				}
				result := scaleTopicClassification(source).(map[string]any)
				if status == "unresolved" && !repaired.Load() {
					for i := range result["movies"].([]topicClassification) {
						result["movies"].([]topicClassification)[i].Matches = nil
					}
				}
				return result, 200
			})
			id := createScaleTopicJob(t, a, ids)
			bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			a.runTagOrganization(bounded, id)
			job, err := store.GetTagOrganization(ctx, id)
			if err != nil || job.Status != "blocked" || job.Error != "CHECKPOINT_FAILED" || job.Processed != 0 {
				t.Fatalf("checkpoint failure stranded job: %+v err=%v", job, err)
			}
			if _, err = faultDB.Exec(`DROP TRIGGER reject_checkpoint`); err != nil {
				t.Fatal(err)
			}
			repaired.Store(true)
			if err = store.RetryTagOrganization(ctx, id); err != nil {
				t.Fatal(err)
			}
			a.runTagOrganization(ctx, id)
			job, err = store.GetTagOrganization(ctx, id)
			if err != nil || job.Status != "completed" || job.Succeeded != 6 {
				t.Fatalf("recovery=%+v err=%v", job, err)
			}
		})
	}
}

func TestTagOrganizationClassificationAdaptsLongInputs(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 11)
	ctx := context.Background()
	// A single source fits the budget, but five full summaries do not.
	for _, mid := range ids {
		if err := a.store.SaveMovieMetadata(ctx, scraper.Metadata{MovieID: mid, Title: "Synthetic", Summary: strings.Repeat("字", 4000), Tags: []string{"Theme"}}); err != nil {
			t.Fatal(err)
		}
	}
	var maxBatch atomic.Int64
	scaleTopicProvider(t, a, func(vocabulary bool, source topicScaleRequest) (any, int) {
		if vocabulary {
			return map[string]any{"topics": []storage.TopicDefinition{{Name: "Theme"}}}, 200
		}
		if int64(len(source.Movies)) > maxBatch.Load() {
			maxBatch.Store(int64(len(source.Movies)))
		}
		for _, input := range source.Movies {
			if input.Summary != strings.Repeat("字", 4000) {
				t.Error("classification evidence was truncated")
			}
		}
		return scaleTopicClassification(source), 200
	})
	id := createScaleTopicJob(t, a, ids)
	a.runTagOrganization(ctx, id)
	job, err := a.store.GetTagOrganization(ctx, id)
	if err != nil || job.Succeeded != 11 || job.Status != "completed" || maxBatch.Load() != 1 {
		t.Fatalf("job=%+v maxBatch=%d err=%v", job, maxBatch.Load(), err)
	}
}

func TestTagOrganizationProviderOutagePreservesPending(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 60)
	var calls atomic.Int64
	var recoverProvider atomic.Bool
	scaleTopicProvider(t, a, func(vocabulary bool, source topicScaleRequest) (any, int) {
		if vocabulary {
			return map[string]any{"topics": []storage.TopicDefinition{{Name: "Theme"}}}, 200
		}
		n := calls.Add(1)
		if n > 1 && !recoverProvider.Load() {
			return nil, 503
		}
		return scaleTopicClassification(source), 200
	})
	ctx := context.Background()
	id := createScaleTopicJob(t, a, ids)
	a.runTagOrganization(ctx, id)
	job, err := a.store.GetTagOrganization(ctx, id)
	if err != nil || job.Status != "blocked" || job.Processed != 5 || job.Failed != 0 || calls.Load() != 4 {
		t.Fatalf("outage consumed remaining movies: %+v calls=%d err=%v", job, calls.Load(), err)
	}
	recoverProvider.Store(true)
	if err := a.store.RetryTagOrganization(ctx, id); err != nil {
		t.Fatal(err)
	}
	a.runTagOrganization(ctx, id)
	job, err = a.store.GetTagOrganization(ctx, id)
	if err != nil || job.Status != "completed" || job.Succeeded != 60 || calls.Load() != 15 {
		t.Fatalf("retry=%+v calls=%d err=%v", job, calls.Load(), err)
	}
}

func TestTopicRequestTransientRetryAndCancellation(t *testing.T) {
	t.Run("429 then 503 then success", func(t *testing.T) {
		a := governanceTestApp(t)
		var calls atomic.Int64
		scaleTopicProvider(t, a, func(bool, topicScaleRequest) (any, int) {
			switch calls.Add(1) {
			case 1:
				return nil, 429
			case 2:
				return nil, 503
			default:
				return map[string]any{"topics": []any{}}, 200
			}
		})
		_, err := a.topicComplete(context.Background(), prompts.TopicVocabularyPrompt(), map[string]any{})
		if err != nil || calls.Load() != 3 {
			t.Fatalf("calls=%d err=%v", calls.Load(), err)
		}
	})
	t.Run("cancel backoff", func(t *testing.T) {
		a := governanceTestApp(t)
		var calls atomic.Int64
		entered := make(chan struct{})
		scaleTopicProvider(t, a, func(bool, topicScaleRequest) (any, int) {
			if calls.Add(1) == 1 {
				close(entered)
			}
			return nil, 429
		})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := a.topicComplete(ctx, prompts.TopicVocabularyPrompt(), map[string]any{})
			done <- err
		}()
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("provider not called")
		}
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) || calls.Load() != 1 {
				t.Fatalf("calls=%d err=%v", calls.Load(), err)
			}
		case <-time.After(time.Second):
			t.Fatal("cancel did not interrupt request/backoff")
		}
	})
}

func TestRetryableTopicRequest(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 408, 413, 422, 429, 500, 502, 503, 504} {
		want := status == 408 || status == 429 || status >= 500
		if got := retryableTopicRequest(&llm.HTTPError{Status: status}); got != want {
			t.Errorf("HTTP %d retry=%v", status, got)
		}
	}
	for _, err := range []error{context.DeadlineExceeded, io.ErrUnexpectedEOF, io.EOF} {
		if !retryableTopicRequest(err) {
			t.Errorf("transient error not retried: %v", err)
		}
	}
	if retryableTopicRequest(context.Canceled) {
		t.Fatal("cancellation retried")
	}
}

func TestTagOrganizationWriteQuotaCanCancel(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 6)
	topicFixtureProvider(t, a, nil)
	a.cfg.AIGovernance.WritePerMinute = 1
	id := createScaleTopicJob(t, a, ids)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.runTagOrganization(ctx, id)
	}()
	defer func() { cancel(); <-done }()
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		job, err := a.store.GetTagOrganization(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if job.Stage == "waiting_quota" {
			if job.Succeeded != 1 || job.Status != "running" {
				t.Fatalf("quota bypassed: %+v", job)
			}
			break
		}
		select {
		case <-deadline:
			t.Fatalf("quota wait never reported: %+v", job)
		case <-ticker.C:
		}
	}
	if err := a.CancelTagOrganization(ctx, id); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("quota wait ignored cancellation")
	}
	job, err := a.store.GetTagOrganization(ctx, id)
	if err != nil || job.Status != "cancelled" || job.Succeeded != 1 || job.Processed != 1 {
		t.Fatalf("late quota write: %+v err=%v", job, err)
	}
}

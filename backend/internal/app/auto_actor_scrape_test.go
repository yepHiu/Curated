package app

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
)

type stubActorAutoScrapeScraper struct {
	mu         sync.Mutex
	actorCalls []string
}

func (s *stubActorAutoScrapeScraper) Scrape(ctx context.Context, movieID string, number string, opts scraper.MovieScrapeOptions) (scraper.Metadata, error) {
	_ = ctx
	_ = movieID
	_ = number
	_ = opts
	return scraper.Metadata{}, errors.New("not used in test")
}

func (s *stubActorAutoScrapeScraper) ScrapeActor(ctx context.Context, displayName string) (scraper.ActorProfile, error) {
	_ = ctx
	s.mu.Lock()
	s.actorCalls = append(s.actorCalls, displayName)
	s.mu.Unlock()
	return scraper.ActorProfile{}, errors.New("test actor scrape failure")
}

func (s *stubActorAutoScrapeScraper) ListProviders() []string {
	return nil
}

func (s *stubActorAutoScrapeScraper) CheckProviderHealth(ctx context.Context, name string) (status string, latencyMs int64, err error) {
	_ = ctx
	_ = name
	return "", 0, errors.New("not used in test")
}

func (s *stubActorAutoScrapeScraper) ActorCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.actorCalls))
	copy(out, s.actorCalls)
	return out
}

func seedMovieWithActors(t *testing.T, store *storage.SQLiteStore, code string, actors []string) {
	t.Helper()
	ctx := context.Background()
	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-" + code,
		Path:     filepath.Join(`D:\Media`, code+".mp4"),
		FileName: code + ".mp4",
		Number:   code,
	})
	if err != nil {
		t.Fatalf("persist scan movie: %v", err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  code,
		Title:   code,
		Actors:  actors,
	}); err != nil {
		t.Fatalf("save movie metadata: %v", err)
	}
}

func waitForActorTaskCount(t *testing.T, tm *tasks.Manager, want int) []contracts.TaskDTO {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		recent := tm.ListRecentFinished(20)
		count := 0
		for _, task := range recent {
			if task.Type == "scrape.actor" {
				count++
			}
		}
		if count == want {
			return recent
		}
		time.Sleep(20 * time.Millisecond)
	}
	recent := tm.ListRecentFinished(20)
	t.Fatalf("timed out waiting for %d scrape.actor tasks, got %+v", want, recent)
	return nil
}

func TestAutoQueueMissingActorProfileScrapes_QueuesOnlyMissingActorsWhenEnabled(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	seedMovieWithActors(t, store, "ABC-100", []string{"Actor Missing", "Actor Ready"})
	if err := store.UpdateActorProfile(ctx, scraper.ActorProfile{
		DisplayName: "Actor Ready",
		Summary:     "already scraped",
	}); err != nil {
		t.Fatalf("update actor profile: %v", err)
	}

	scraperStub := &stubActorAutoScrapeScraper{}
	tm := tasks.NewManager()
	a := &App{
		cfg: config.Config{
			Scraper: config.ScraperConfig{TaskTimeoutSeconds: 1},
		},
		logger:                        zap.NewNop(),
		store:                         store,
		scraper:                       scraperStub,
		tasks:                         tm,
		appCtx:                        ctx,
		scrapeSem:                     make(chan struct{}, 2),
		autoActorProfileScrape:        true,
		autoActorProfileScrapePending: make(map[string]struct{}),
	}

	a.enqueueAutoActorProfileScrapes(ctx, []string{"Actor Missing", "Actor Ready", "Actor Missing"})

	waitForActorTaskCount(t, tm, 1)
	if got := scraperStub.ActorCalls(); len(got) != 1 || got[0] != "Actor Missing" {
		t.Fatalf("actor scrape calls = %#v, want [Actor Missing]", got)
	}
}

func TestAutoQueueMissingActorProfileScrapes_DisabledSkipsAll(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	seedMovieWithActors(t, store, "ABC-101", []string{"Actor Missing"})

	scraperStub := &stubActorAutoScrapeScraper{}
	tm := tasks.NewManager()
	a := &App{
		cfg: config.Config{
			Scraper: config.ScraperConfig{TaskTimeoutSeconds: 1},
		},
		logger:                        zap.NewNop(),
		store:                         store,
		scraper:                       scraperStub,
		tasks:                         tm,
		appCtx:                        ctx,
		scrapeSem:                     make(chan struct{}, 1),
		autoActorProfileScrape:        false,
		autoActorProfileScrapePending: make(map[string]struct{}),
	}

	a.enqueueAutoActorProfileScrapes(ctx, []string{"Actor Missing"})
	time.Sleep(100 * time.Millisecond)

	if got := scraperStub.ActorCalls(); len(got) != 0 {
		t.Fatalf("actor scrape calls = %#v, want none", got)
	}
	if recent := tm.ListRecentFinished(10); len(recent) != 0 {
		t.Fatalf("expected no finished tasks, got %+v", recent)
	}
}

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
	"curated-backend/internal/library"
	"curated-backend/internal/scraper"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
)

type blockingMovieScraper struct {
	mu         sync.Mutex
	started    int
	current    int
	maxCurrent int
	release    chan struct{}
}

func newBlockingMovieScraper() *blockingMovieScraper {
	return &blockingMovieScraper{release: make(chan struct{})}
}

func (s *blockingMovieScraper) Scrape(ctx context.Context, movieID string, number string, opts scraper.MovieScrapeOptions) (scraper.Metadata, error) {
	_ = movieID
	_ = number
	_ = opts
	s.mu.Lock()
	s.started++
	s.current++
	if s.current > s.maxCurrent {
		s.maxCurrent = s.current
	}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.current--
		s.mu.Unlock()
	}()
	select {
	case <-s.release:
		return scraper.Metadata{}, errors.New("test scrape stopped after slot acquired")
	case <-ctx.Done():
		return scraper.Metadata{}, ctx.Err()
	}
}

func (s *blockingMovieScraper) ScrapeActor(ctx context.Context, displayName string) (scraper.ActorProfile, error) {
	_ = ctx
	_ = displayName
	return scraper.ActorProfile{}, nil
}

func (s *blockingMovieScraper) ListProviders() []string { return nil }

func (s *blockingMovieScraper) CheckProviderHealth(ctx context.Context, name string) (string, int64, error) {
	_ = ctx
	_ = name
	return "", 0, nil
}

func (s *blockingMovieScraper) snapshot() (started, current, maxCurrent int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started, s.current, s.maxCurrent
}

func seedMovieForScrapeQueue(t *testing.T, store *storage.SQLiteStore, code string) string {
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
	return outcome.MovieID
}

func newScrapeQueueTestApp(t *testing.T, store *storage.SQLiteStore, scraperService scraper.Service, scrapeConc int) *App {
	t.Helper()
	ctx := context.Background()
	return &App{
		cfg: config.Config{
			Scraper: config.ScraperConfig{TaskTimeoutSeconds: 5},
		},
		logger:              zap.NewNop(),
		store:               store,
		library:             library.NewService(),
		scraper:             scraperService,
		tasks:               tasks.NewManager(),
		appCtx:              ctx,
		scrapeSem:           make(chan struct{}, scrapeConc),
		scrapeMovieInflight: make(map[string]string),
	}
}

func waitForScraperStarted(t *testing.T, scraperStub *blockingMovieScraper, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		started, _, _ := scraperStub.snapshot()
		if started >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	started, current, maxCurrent := scraperStub.snapshot()
	t.Fatalf("timed out waiting for %d scrape starts, started=%d current=%d max=%d", want, started, current, maxCurrent)
}

func TestStartAsyncMovieMetadataScrape_RespectsGlobalConcurrency(t *testing.T) {
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

	idA := seedMovieForScrapeQueue(t, store, "ABC-200")
	idB := seedMovieForScrapeQueue(t, store, "ABC-201")
	scraperStub := newBlockingMovieScraper()
	a := newScrapeQueueTestApp(t, store, scraperStub, 1)
	defer a.Close()

	detailA, err := store.GetMovieDetail(ctx, idA)
	if err != nil {
		t.Fatal(err)
	}
	detailB, err := store.GetMovieDetail(ctx, idB)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := a.startAsyncMovieMetadataScrape(ctx, detailA); err != nil {
		t.Fatal(err)
	}
	waitForScraperStarted(t, scraperStub, 1)
	if _, err := a.startAsyncMovieMetadataScrape(ctx, detailB); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	started, current, maxCurrent := scraperStub.snapshot()
	if started != 1 || current != 1 || maxCurrent != 1 {
		t.Fatalf("concurrency snapshot started=%d current=%d max=%d, want 1/1/1", started, current, maxCurrent)
	}

	close(scraperStub.release)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		started, current, maxCurrent = scraperStub.snapshot()
		if started == 2 && current == 0 && maxCurrent == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected serialized scrapes, started=%d current=%d max=%d", started, current, maxCurrent)
}

func TestStartAsyncMovieMetadataScrape_DedupesInFlightMovie(t *testing.T) {
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

	movieID := seedMovieForScrapeQueue(t, store, "ABC-202")
	scraperStub := newBlockingMovieScraper()
	a := newScrapeQueueTestApp(t, store, scraperStub, 2)
	defer a.Close()

	detail, err := store.GetMovieDetail(ctx, movieID)
	if err != nil {
		t.Fatal(err)
	}

	first, err := a.startAsyncMovieMetadataScrape(ctx, detail)
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.startAsyncMovieMetadataScrape(ctx, detail)
	if err != nil {
		t.Fatal(err)
	}
	if first.TaskID == "" || first.TaskID != second.TaskID {
		t.Fatalf("expected reused task, first=%q second=%q", first.TaskID, second.TaskID)
	}
	waitForScraperStarted(t, scraperStub, 1)
	time.Sleep(50 * time.Millisecond)
	started, _, maxCurrent := scraperStub.snapshot()
	if started != 1 || maxCurrent != 1 {
		t.Fatalf("duplicate scrape started=%d max=%d, want 1/1", started, maxCurrent)
	}
	close(scraperStub.release)
}

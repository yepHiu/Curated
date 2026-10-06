package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
)

// TestSaveEnrichedMetadataAssetSources 验证次来源图片及旧单源图片保存各自的下载上下文。
func TestSaveEnrichedMetadataAssetSources(t *testing.T) {
	t.Parallel()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "assets.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID: "scan", Path: "D:/Media/FC2-3977618.mp4", FileName: "FC2-3977618.mp4", Number: "FC2-3977618",
	})
	if err != nil {
		t.Fatal(err)
	}
	metadata := scraper.Metadata{
		MovieID: outcome.MovieID, Number: "FC2-3977618", Title: "Primary title", Provider: "fc2hub", Homepage: "https://javten.com/video/1/id3977618/title",
		CoverURL: "https://cdn.example/cover.jpg", ThumbURL: "https://javten.com/thumb.jpg", PreviewImages: []string{"https://cdn.example/sample.jpg"},
		AssetSources: map[string]scraper.AssetSource{
			"https://cdn.example/cover.jpg":  {Provider: "PPVDataBank", Homepage: "https://ppvdatabank.com/article/3977618/"},
			"https://cdn.example/sample.jpg": {Provider: "FC2", Homepage: "https://adult.contents.fc2.com/article/3977618/"},
		},
	}
	if err := store.SaveMovieMetadata(ctx, metadata); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ kind, provider, referer string }{
		{"cover", "PPVDataBank", "https://ppvdatabank.com/article/3977618/"},
		{"thumb", "fc2hub", metadata.Homepage},
		{"preview_image", "FC2", "https://adult.contents.fc2.com/article/3977618/"},
	} {
		var provider, referer string
		if err := store.db.QueryRowContext(ctx, `SELECT source_provider, referer_url FROM media_assets WHERE movie_id = ? AND type = ?`, outcome.MovieID, test.kind).Scan(&provider, &referer); err != nil {
			t.Fatal(err)
		}
		if provider != test.provider || referer != test.referer {
			t.Fatalf("%s context=%s %s want=%s %s", test.kind, provider, referer, test.provider, test.referer)
		}
	}
	var primary string
	if err := store.db.QueryRowContext(ctx, `SELECT provider FROM movies WHERE id = ?`, outcome.MovieID).Scan(&primary); err != nil || primary != "fc2hub" {
		t.Fatalf("primary provider changed: %s %v", primary, err)
	}
}

func TestSaveMovieMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate store: %v", err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     "D:/Media/JAV/Main/ABC-123.mp4",
		FileName: "ABC-123.mp4",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatalf("failed to persist scan movie: %v", err)
	}

	err = store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:         outcome.MovieID,
		Number:          "ABC-123",
		Title:           "Example Title",
		Summary:         "Example Summary",
		Provider:        "javbus",
		Homepage:        "https://example.com/movie",
		Director:        "Jane Doe",
		Studio:          "Sample Studio",
		Actors:          []string{"Actor A", "Actor B"},
		Tags:            []string{"Drama", "Sample"},
		RuntimeMinutes:  120,
		Rating:          4.6,
		ReleaseDate:     "2025-03-01",
		CoverURL:        "https://example.com/poster.jpg",
		ThumbURL:        "https://example.com/thumb.jpg",
		PreviewVideoURL: "https://example.com/preview.mp4",
		PreviewImages:   []string{"https://example.com/1.jpg", "https://example.com/2.jpg"},
	})
	if err != nil {
		t.Fatalf("failed to save movie metadata: %v", err)
	}

	var (
		title, studio, provider, coverURL string
		actorCount, tagCount, assetCount  int
	)
	if err := store.db.QueryRowContext(ctx, `SELECT title, studio, provider, cover_url FROM movies WHERE id = ?`, outcome.MovieID).
		Scan(&title, &studio, &provider, &coverURL); err != nil {
		t.Fatalf("failed to query saved movie: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_actors WHERE movie_id = ?`, outcome.MovieID).Scan(&actorCount); err != nil {
		t.Fatalf("failed to count movie actors: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_tags WHERE movie_id = ?`, outcome.MovieID).Scan(&tagCount); err != nil {
		t.Fatalf("failed to count movie tags: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_assets WHERE movie_id = ?`, outcome.MovieID).Scan(&assetCount); err != nil {
		t.Fatalf("failed to count media assets: %v", err)
	}

	if title != "Example Title" || studio != "Sample Studio" || provider != "javbus" || coverURL == "" {
		t.Fatalf("unexpected movie metadata values: title=%q studio=%q provider=%q cover=%q", title, studio, provider, coverURL)
	}
	if actorCount != 2 || tagCount != 2 {
		t.Fatalf("unexpected relation counts: actors=%d tags=%d", actorCount, tagCount)
	}
	if assetCount != 4 {
		t.Fatalf("expected 4 media assets, got %d", assetCount)
	}
}

func TestSaveMovieMetadata_UnknownMovieID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate store: %v", err)
	}

	err = store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: "no-such-movie",
		Number:  "X-1",
		Title:   "T",
	})
	if err == nil {
		t.Fatal("expected error for unknown movie id")
	}
	if !errors.Is(err, ErrMovieNotFoundForMetadata) {
		t.Fatalf("expected ErrMovieNotFoundForMetadata, got %v", err)
	}
}

func TestSaveMovieMetadata_PreservesUserTags(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate store: %v", err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     "D:/Media/JAV/Main/ABC-123.mp4",
		FileName: "ABC-123.mp4",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatalf("failed to persist scan movie: %v", err)
	}

	err = store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  "ABC-123",
		Title:   "First",
		Summary: "S",
		Studio:  "St",
		Tags:    []string{"GenreA", "GenreB"},
	})
	if err != nil {
		t.Fatalf("first save: %v", err)
	}

	if err := store.PatchMovieUserPrefs(ctx, outcome.MovieID, contracts.PatchMovieInput{
		UserTagsSet: true,
		UserTags:    []string{"我的收藏", "待看"},
	}); err != nil {
		t.Fatalf("patch user tags: %v", err)
	}

	err = store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  "ABC-123",
		Title:   "Updated",
		Summary: "S2",
		Studio:  "St",
		Tags:    []string{"OnlyNfo"},
	})
	if err != nil {
		t.Fatalf("second save: %v", err)
	}

	detail, err := store.GetMovieDetail(ctx, outcome.MovieID)
	if err != nil {
		t.Fatalf("get detail: %v", err)
	}
	if len(detail.Tags) != 1 || detail.Tags[0] != "OnlyNfo" {
		t.Fatalf("metadata tags = %#v, want [OnlyNfo]", detail.Tags)
	}
	if len(detail.UserTags) != 2 {
		t.Fatalf("user tags = %#v, want 2 items", detail.UserTags)
	}
}

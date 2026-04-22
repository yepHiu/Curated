package storage

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
)

func TestListActorsAndReplaceActorUserTags(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-a",
		Path:     filepath.Join(root, "MOV-1.mp4"),
		FileName: "MOV-1.mp4",
		Number:   "MOV-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:        outcome.MovieID,
		Number:         "MOV-1",
		Title:          "One",
		Summary:        "s",
		Provider:       "p",
		Homepage:       "",
		Director:       "",
		Studio:         "St",
		Actors:         []string{"Alpha Star", "Beta"},
		Tags:           nil,
		RuntimeMinutes: 1,
		Rating:         0,
		ReleaseDate:    "",
	}); err != nil {
		t.Fatal(err)
	}

	res, err := store.ListActors(ctx, contracts.ListActorsRequest{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Fatalf("total = %d, want 2", res.Total)
	}
	var alpha *contracts.ActorListItemDTO
	for i := range res.Actors {
		if res.Actors[i].Name == "Alpha Star" {
			alpha = &res.Actors[i]
			break
		}
	}
	if alpha == nil || alpha.MovieCount != 1 {
		t.Fatalf("alpha = %+v", alpha)
	}

	if err := store.ReplaceActorUserTagsByName(ctx, "Alpha Star", []string{"  lead ", "lead", "fav"}); err != nil {
		t.Fatal(err)
	}
	res2, err := store.ListActors(ctx, contracts.ListActorsRequest{ActorTag: "fav", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Total != 1 || len(res2.Actors) != 1 || res2.Actors[0].Name != "Alpha Star" {
		t.Fatalf("filtered = %+v", res2)
	}
	if len(res2.Actors[0].UserTags) != 2 {
		t.Fatalf("tags = %#v", res2.Actors[0].UserTags)
	}

	res3, err := store.ListActors(ctx, contracts.ListActorsRequest{Q: "star", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if res3.Total != 1 || res3.Actors[0].Name != "Alpha Star" {
		t.Fatalf("q filter = %+v", res3)
	}

	res4, err := store.ListActors(ctx, contracts.ListActorsRequest{Q: "fav", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if res4.Total != 1 || res4.Actors[0].Name != "Alpha Star" {
		t.Fatalf("q filter by actor user tag = %+v", res4)
	}
}

func TestGetActorProfile_ExternalLinks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "actor-links.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     filepath.Join(root, "MOV-1.mp4"),
		FileName: "MOV-1.mp4",
		Number:   "MOV-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:        outcome.MovieID,
		Number:         "MOV-1",
		Title:          "One",
		Summary:        "s",
		Studio:         "Studio",
		Actors:         []string{"Alpha Star"},
		RuntimeMinutes: 1,
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.ReplaceActorExternalLinksByName(ctx, "Alpha Star", []string{
		" https://example.com/a ",
		"https://example.com/b",
		"https://example.com/a",
	}); err != nil {
		t.Fatal(err)
	}

	profile, err := store.GetActorProfile(ctx, "Alpha Star")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"https://example.com/a", "https://example.com/b"}
	if !reflect.DeepEqual(profile.ExternalLinks, want) {
		t.Fatalf("external links = %#v, want %#v", profile.ExternalLinks, want)
	}
}

func TestReplaceActorExternalLinksByName_RejectsInvalidURL(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "actor-links-invalid.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-2",
		Path:     filepath.Join(root, "MOV-2.mp4"),
		FileName: "MOV-2.mp4",
		Number:   "MOV-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:        outcome.MovieID,
		Number:         "MOV-2",
		Title:          "Two",
		Summary:        "s",
		Studio:         "Studio",
		Actors:         []string{"Beta Star"},
		RuntimeMinutes: 1,
	}); err != nil {
		t.Fatal(err)
	}

	err = store.ReplaceActorExternalLinksByName(ctx, "Beta Star", []string{"javascript:alert(1)"})
	if !errors.Is(err, ErrInvalidActorExternalLinks) {
		t.Fatalf("err = %v, want ErrInvalidActorExternalLinks", err)
	}
}

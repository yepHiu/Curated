package storage

import (
	"context"
	"testing"
)

func TestOrganizationTranslationCheckpoints(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	if _, err := s.CreateTagOrganization(ctx, "translations", "translations", "manual", "en", []string{"a"}, "title", "summary"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTagOrganization(ctx, "translations", "running", "translating_metadata", ""); err != nil {
		t.Fatal(err)
	}
	first, err := s.NextOrganizationTranslation(ctx, "translations")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CompleteOrganizationTranslation(ctx, "translations", first, "Translated summary", "succeeded", ""); err != nil {
		t.Fatal(err)
	}
	next, err := s.NextOrganizationTranslation(ctx, "translations")
	if err != nil || next.Kind == first.Kind {
		t.Fatalf("checkpoint not advanced: %+v %v", next, err)
	}
	if _, err = s.db.Exec(`UPDATE movies SET user_title='Manual edit' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	if err = s.CompleteOrganizationTranslation(ctx, "translations", next, "Translated title", "succeeded", ""); err != nil {
		t.Fatal(err)
	}
	job, err := s.GetTagOrganization(ctx, "translations")
	if err != nil || job.Total != 1 || job.Processed != 1 || job.Failed != 1 || job.TranslationProcessed != 1 {
		t.Fatalf("counts %+v %v", job, err)
	}
	items, err := s.TagOrganizationItems(ctx, "translations", false, 25, 0)
	if err != nil || len(items) != 1 || items[0].Status != "failed" || len(items[0].Translations) != 2 {
		t.Fatalf("items %+v %v", items, err)
	}
	var title, summary, override string
	if err = s.db.QueryRow(`SELECT title,summary,user_title FROM movies WHERE id='a'`).Scan(&title, &summary, &override); err != nil {
		t.Fatal(err)
	}
	if title != "a" || summary != "A clear theme" || override != "Manual edit" {
		t.Fatalf("source or user edit changed: %s %s %s", title, summary, override)
	}
	_ = s.UpdateTagOrganization(ctx, "translations", "partial_failed", "finished", "")
	if err = s.RetryTagOrganization(ctx, "translations"); err != nil {
		t.Fatal(err)
	}
	retry, err := s.NextOrganizationTranslation(ctx, "translations")
	if err != nil || retry.Kind != "title" {
		t.Fatalf("successful field retried: %+v %v", retry, err)
	}
	_ = s.UpdateTagOrganization(ctx, "translations", "cancelled", "stopped", "")
	if err = s.CompleteOrganizationTranslation(ctx, "translations", retry, "late", "succeeded", ""); err == nil {
		t.Fatal("cancelled job wrote data")
	}
}

func TestOrganizationMixedTasksKeepMovieCounts(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	if _, err := s.CreateTagOrganization(ctx, "mixed", "mixed", "manual", "en", []string{"a"}, "tags", "title"); err != nil {
		t.Fatal(err)
	}
	_ = s.UpdateTagOrganization(ctx, "mixed", "running", "translating_metadata", "")
	input, err := s.NextOrganizationTranslation(ctx, "mixed")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CompleteOrganizationTranslation(ctx, "mixed", input, "Translation", "succeeded", ""); err != nil {
		t.Fatal(err)
	}
	job, err := s.GetTagOrganization(ctx, "mixed")
	if err != nil || job.Processed != 0 || job.TranslationProcessed != 1 {
		t.Fatalf("premature completion: %+v %v", job, err)
	}
	if err = s.SetTagOrganizationItem(ctx, "mixed", "a", "succeeded", ""); err != nil {
		t.Fatal(err)
	}
	job, err = s.GetTagOrganization(ctx, "mixed")
	if err != nil || job.Total != 1 || job.Succeeded != 1 || job.Processed != 1 {
		t.Fatalf("counts %+v %v", job, err)
	}
}

func TestOrganizationTranslationKeepsDeletedMovieHistory(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	if _, err := s.CreateTagOrganization(ctx, "deleted", "deleted", "manual", "en", []string{"a"}, "title"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteMovie(ctx, "a", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	input, err := s.NextOrganizationTranslation(ctx, "deleted")
	if err != nil || input.Available {
		t.Fatalf("missing source %+v %v", input, err)
	}
	_ = s.UpdateTagOrganization(ctx, "deleted", "running", "translating_metadata", "")
	if err = s.CompleteOrganizationTranslation(ctx, "deleted", input, "", "failed", "MOVIE_UNAVAILABLE"); err != nil {
		t.Fatal(err)
	}
	job, err := s.GetTagOrganization(ctx, "deleted")
	if err != nil || job.Failed != 1 {
		t.Fatalf("missing history %+v %v", job, err)
	}
}

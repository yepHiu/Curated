package storage

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
)

func TestDeleteTagOrganizationHistoryPreservesMovieState(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	if _, err := s.CreateTagOrganization(ctx, "history", "history", "manual", "en", []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTagOrganization(ctx, "history", "running", "applying", ""); err != nil {
		t.Fatal(err)
	}
	input, err := s.TopicMovieInput(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ApplyMovieTopics(ctx, "history", input, []string{"Theme"}); err != nil {
		t.Fatal(err)
	}
	if err = s.SetTagOrganizationItem(ctx, "history", "b", "failed", "SOURCE_TOO_LONG"); err != nil {
		t.Fatal(err)
	}
	if err = s.UpdateTagOrganization(ctx, "history", "partial_failed", "finished", ""); err != nil {
		t.Fatal(err)
	}
	before, _ := s.TagOrganizationStats(ctx)
	if err = s.DeleteTagOrganization(ctx, "history"); err != nil {
		t.Fatal(err)
	}
	jobs, err := s.ListTagOrganizations(ctx, false)
	if err != nil || len(jobs) != 0 {
		t.Fatalf("history=%+v err=%v", jobs, err)
	}
	if _, err = s.GetTagOrganization(ctx, "history"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted record accessible: %v", err)
	}
	after, err := s.TagOrganizationStats(ctx)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("coverage changed: before=%+v after=%+v err=%v", before, after, err)
	}
	issues, err := s.TagOrganizationIssues(ctx, 25, 0)
	if err != nil || len(issues) != 1 || issues[0].MovieID != "b" {
		t.Fatalf("issues=%+v err=%v", issues, err)
	}
	afterInput, err := s.TopicMovieInput(ctx, "a")
	if err != nil || !reflect.DeepEqual(afterInput.UserTags, []string{"Theme"}) {
		t.Fatalf("tags=%+v err=%v", afterInput, err)
	}
	if err = s.RetryTagOrganization(ctx, "history"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted record retried: %v", err)
	}
	if _, err = s.UndoTopicOrganization(ctx, "history"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted record undone: %v", err)
	}
}

func TestDeleteTagOrganizationStates(t *testing.T) {
	for _, status := range []string{"queued", "running", "completed", "partial_failed", "failed", "blocked", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			s := newTopicTestStore(t)
			ctx := context.Background()
			if _, err := s.CreateTagOrganization(ctx, "job", "request", "manual", "en", []string{"a"}); err != nil {
				t.Fatal(err)
			}
			if status != "queued" {
				if err := s.UpdateTagOrganization(ctx, "job", status, "", ""); err != nil {
					t.Fatal(err)
				}
			}
			err := s.DeleteTagOrganization(ctx, "job")
			if status == "queued" || status == "running" {
				if !errors.Is(err, ErrOrganizationActive) {
					t.Fatalf("active deletion: %v", err)
				}
				if _, err = s.GetTagOrganization(ctx, "job"); err != nil {
					t.Fatal(err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if err = s.DeleteTagOrganization(ctx, "job"); !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("repeat deletion: %v", err)
				}
			}
		})
	}
}

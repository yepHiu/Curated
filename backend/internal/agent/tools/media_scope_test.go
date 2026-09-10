package tools

import (
	"context"
	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"encoding/json"
	"strings"
	"testing"
)

type mediaTaskQuery struct {
	stubQuery
	kind string
}

func (q mediaTaskQuery) GetTask(context.Context, string) (contracts.TaskDTO, bool) {
	return contracts.TaskDTO{TaskID: "task-private", Type: q.kind, Message: "private title and path", Status: contracts.TaskRunning}, true
}

func TestAgentTaskLookupDoesNotExposeBookTasks(t *testing.T) {
	for _, kind := range []string{contracts.TaskTypeScanComics, contracts.TaskTypeScanPhotos, contracts.TaskTypeImportComics, contracts.TaskTypeImportPhotos, contracts.TaskTypeComicCacheCleanup, "photo.cache.cleanup", "future.domain"} {
		result, err := getTaskStatus(mediaTaskQuery{kind: kind}).Handler(context.Background(), core.Call{Args: json.RawMessage(`{"taskId":"task-private"}`)})
		if err != nil || result.OK || result.Data != nil || result.Error == nil || strings.Contains(result.Error.Message, "private") {
			t.Fatalf("leaked %s: %+v %v", kind, result, err)
		}
	}
	for _, kind := range []string{"scan.library", "scrape.movie", contracts.TaskTypeImportMovies} {
		result, err := getTaskStatus(mediaTaskQuery{kind: kind}).Handler(context.Background(), core.Call{Args: json.RawMessage(`{"taskId":"task-private"}`)})
		if err != nil || !result.OK {
			t.Fatalf("movie task rejected: %s %+v", kind, result)
		}
	}
}

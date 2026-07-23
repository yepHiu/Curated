package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func newSavedViewHandlerServer(t *testing.T) *httptest.Server {
	t.Helper()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "saved-view-handler.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.Migrate(t.Context()); err != nil {
		_ = store.Close()
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	server := httptest.NewServer(NewHandler(Deps{Store: store, Logger: zap.NewNop()}).Routes())
	t.Cleanup(server.Close)
	return server
}

func savedViewRequest(t *testing.T, client *http.Client, method, url string, body any) *http.Response {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	request, err := http.NewRequest(method, url, &payload)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func TestSavedViewHandlersCRUDAndCanonicalization(t *testing.T) {
	t.Parallel()
	server := newSavedViewHandlerServer(t)
	client := server.Client()

	create := savedViewRequest(t, client, http.MethodPost, server.URL+"/api/library/saved-views", map[string]any{
		"name": "  My 4K  ",
		"filters": map[string]any{
			"schemaVersion":   1,
			"mode":            "library",
			"tab":             "all",
			"playState":       "unwatched",
			"resolution":      "2160P",
			"addedWithinDays": 30,
		},
	})
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", create.StatusCode)
	}
	var created contracts.SavedViewDTO
	if err := json.NewDecoder(create.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.Name != "My 4K" || created.Filters.Resolution != "4k" || created.Filters.PlayState != "unwatched" {
		t.Fatalf("created view was not canonicalized: %#v", created)
	}

	duplicate := savedViewRequest(t, client, http.MethodPost, server.URL+"/api/library/saved-views", map[string]any{
		"name":    "my 4k",
		"filters": map[string]any{"schemaVersion": 1},
	})
	if duplicate.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate status=%d", duplicate.StatusCode)
	}
	var duplicateError contracts.AppError
	if err := json.NewDecoder(duplicate.Body).Decode(&duplicateError); err != nil {
		t.Fatalf("decode duplicate error: %v", err)
	}
	if duplicateError.Code != contracts.ErrorCodeSavedViewNameConflict {
		t.Fatalf("duplicate code=%q", duplicateError.Code)
	}

	transient := savedViewRequest(t, client, http.MethodPost, server.URL+"/api/library/saved-views", map[string]any{
		"name":    "Invalid",
		"filters": map[string]any{"schemaVersion": 1, "selected": "movie-1"},
	})
	if transient.StatusCode != http.StatusBadRequest {
		t.Fatalf("transient filter status=%d", transient.StatusCode)
	}

	patch := savedViewRequest(t, client, http.MethodPatch, server.URL+"/api/library/saved-views/"+created.ID, map[string]any{
		"name": "Unwatched UHD",
	})
	if patch.StatusCode != http.StatusOK {
		t.Fatalf("patch status=%d", patch.StatusCode)
	}

	list := savedViewRequest(t, client, http.MethodGet, server.URL+"/api/library/saved-views", nil)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", list.StatusCode)
	}
	var listed contracts.SavedViewsDTO
	if err := json.NewDecoder(list.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].Name != "Unwatched UHD" {
		t.Fatalf("unexpected list: %#v", listed)
	}

	badOrder := savedViewRequest(t, client, http.MethodPut, server.URL+"/api/library/saved-views/order", map[string]any{
		"ids": []string{},
	})
	if badOrder.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad order status=%d", badOrder.StatusCode)
	}

	remove := savedViewRequest(t, client, http.MethodDelete, server.URL+"/api/library/saved-views/"+created.ID, nil)
	if remove.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status=%d", remove.StatusCode)
	}
}

func TestListMoviesRejectsInvalidSavedViewFilters(t *testing.T) {
	t.Parallel()
	server := newSavedViewHandlerServer(t)
	for _, path := range []string{
		"/api/library/movies?mode=unknown",
		"/api/library/movies?playState=unknown",
		"/api/library/movies?userRating=6",
		"/api/library/movies?addedAfter=not-a-date",
	} {
		response := savedViewRequest(t, server.Client(), http.MethodGet, server.URL+path, nil)
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s status=%d, want 400", path, response.StatusCode)
		}
	}
}

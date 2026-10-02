package server

import (
	"context"
	"curated-backend/internal/contracts"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// topicHandlerFixture counts mutations to detect method and schema bypasses.
type topicHandlerFixture struct {
	TopicOrganizationProvider
	starts, cancels int
}

func (f *topicHandlerFixture) StartTagOrganization(_ context.Context, _ contracts.TagOrganizationRequest) (contracts.TagOrganizationJobDTO, error) {
	f.starts++
	return contracts.TagOrganizationJobDTO{ID: "fixture", Status: "queued"}, nil
}
func (f *topicHandlerFixture) GetTagOrganization(_ context.Context, id string) (contracts.TagOrganizationJobDTO, error) {
	return contracts.TagOrganizationJobDTO{ID: id, Status: "running"}, nil
}
func (f *topicHandlerFixture) CancelTagOrganization(_ context.Context, _ string) error {
	f.cancels++
	return nil
}

func (f *topicHandlerFixture) TagOrganizationStats(_ context.Context) (contracts.TagOrganizationStatsDTO, error) {
	return contracts.TagOrganizationStatsDTO{Total: 6, Organized: 3, Unorganized: 2, Outdated: 1, Unresolved: 1}, nil
}

func TestTopicCoverageHandlerIsReadOnlyAndNotCached(t *testing.T) {
	f := &topicHandlerFixture{}
	h := &Handler{topicOrganization: f}
	response := httptest.NewRecorder()
	h.handleTagOrganizationStats(response, httptest.NewRequest(http.MethodGet, "/api/ai/tag-organizations/stats", nil))
	if response.Code != 200 || response.Header().Get("Cache-Control") != "no-store" || f.starts != 0 || !strings.Contains(response.Body.String(), `"unorganized":2`) {
		t.Fatalf("stats=%s headers=%v", response.Body, response.Header())
	}
}

// TestTopicHandlersRejectMetadataWrites verifies HTTP callers cannot smuggle NFO fields or use GET mutations.
func TestTopicHandlersRejectMetadataWrites(t *testing.T) {
	f := &topicHandlerFixture{}
	h := &Handler{topicOrganization: f}
	for _, body := range []string{`{"scope":"all","requestId":"x","metadataTags":["bad"]}`, `{"scope":"all","requestId":"x","tagType":"nfo"}`, `{"scope":"all","requestId":"x"}{}`} {
		req := httptest.NewRequest(http.MethodPost, "/api/ai/tag-organizations", strings.NewReader(body))
		response := httptest.NewRecorder()
		h.handleTagOrganizations(response, req)
		if response.Code != 400 || f.starts != 0 {
			t.Fatalf("schema bypass: %d %s", response.Code, response.Body)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/ai/tag-organizations/fixture/cancel", nil)
	req.SetPathValue("jobId", "fixture")
	req.SetPathValue("operation", "cancel")
	response := httptest.NewRecorder()
	h.handleTagOrganization(response, req)
	if response.Code != 405 || f.cancels != 0 {
		t.Fatal("GET mutated task")
	}
	req = httptest.NewRequest(http.MethodPost, "/api/ai/tag-organizations", strings.NewReader(`{"scope":"all","requestId":"x"}`))
	response = httptest.NewRecorder()
	h.handleTagOrganizations(response, req)
	if response.Code != 202 || f.starts != 1 {
		t.Fatalf("explicit start failed: %s", response.Body)
	}
}

// TestTopicRenameSchema rejects metadata fields and ambiguous edits before touching storage.
func TestTopicRenameSchema(t *testing.T) {
	h := &Handler{}
	for _, body := range []string{`{"name":"Travel"}`, `{"name":"Travel","expectedName":"Trips","hidden":false}`, `{"hidden":true,"expectedName":"Trips"}`, `{"name":"Travel","expectedName":"Trips","metadataTags":[]}`} {
		req := httptest.NewRequest(http.MethodPatch, "/api/library/topics/topic", strings.NewReader(body))
		req.SetPathValue("topicId", "topic")
		response := httptest.NewRecorder()
		h.handleGetTopic(response, req)
		if response.Code != 400 {
			t.Fatalf("invalid rename schema: %d", response.Code)
		}
	}
}

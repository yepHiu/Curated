package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

type bookOrganizationStub struct {
	AIChatProvider
	started   int
	reviewErr error
}

func (s *bookOrganizationStub) StartBookOrganization(_ context.Context, req contracts.BookOrganizationRequest) (contracts.BookOrganizationJob, error) {
	s.started++
	return contracts.BookOrganizationJob{ID: "job", Kind: req.Kind, Total: len(req.BookIDs), Status: "queued"}, nil
}
func (*bookOrganizationStub) ControlBookOrganization(context.Context, string, string) error {
	return nil
}
func (s *bookOrganizationStub) ReviewBookOrganization(context.Context, string, contracts.BookOrganizationReview) error {
	return s.reviewErr
}

func TestBookOrganizationSubmissionLocalAndStrict(t *testing.T) {
	stub := &bookOrganizationStub{}
	h := &Handler{aiChatProvider: stub}
	handler := localServerManagement(h.handleBookOrganizations)
	for _, tc := range []struct {
		host, remote, body string
		want               int
	}{
		{"localhost", "127.0.0.1:1234", `{"kind":"comic","bookIds":["a"],"locale":"zh-CN","requestId":"r"}`, 202},
		{"localhost", "127.0.0.1:1234", `{"kind":"comic","unknown":true}`, 400},
		{"localhost", "127.0.0.1:1234", `{} {}`, 400},
		{"192.168.1.2", "192.168.1.3:1234", `{"kind":"comic"}`, 403},
	} {
		req := httptest.NewRequest(http.MethodPost, "http://"+tc.host+"/api/ai/book-organizations", strings.NewReader(tc.body))
		req.RemoteAddr = tc.remote
		response := httptest.NewRecorder()
		handler(response, req)
		if response.Code != tc.want {
			t.Fatalf("%s: %d %s", tc.body, response.Code, response.Body.String())
		}
	}
	if stub.started != 1 {
		t.Fatalf("unauthorized or malformed request reached app: %d", stub.started)
	}
}

func TestBookReviewRateLimitIsRetryableHTTP(t *testing.T) {
	h := &Handler{aiChatProvider: &bookOrganizationStub{reviewErr: &core.ToolError{Code: "AI_RATE_LIMITED", Message: "write rate limit reached"}}}
	req := httptest.NewRequest(http.MethodPost, "/api/ai/book-organizations/job/review", strings.NewReader(`{"bookId":"book","revision":1,"decision":"apply"}`))
	req.SetPathValue("jobId", "job")
	req.SetPathValue("operation", "review")
	response := httptest.NewRecorder()
	h.handleBookOrganizations(response, req)
	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "60" || !strings.Contains(response.Body.String(), "AI_RATE_LIMITED") || !strings.Contains(response.Body.String(), `"retryable":true`) {
		t.Fatalf("response: %d %v %s", response.Code, response.Header(), response.Body.String())
	}
}

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAIGovernanceRoutesRequirePIN(t *testing.T) {
	srv, store := newAuthTestServer(t)
	if err := store.SetAppPIN(context.Background(), "123456"); err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ method, path string }{{"GET", "/api/ai/settings"}, {"PATCH", "/api/ai/settings"}, {"GET", "/api/ai/usage"}, {"GET", "/api/ai/audit"}, {"POST", "/api/ai/cleanup"}} {
		req, _ := http.NewRequest(route.method, srv.URL+route.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusLocked && resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s bypassed PIN: %d", route.method, route.path, resp.StatusCode)
		}
	}
}

func TestAIReportFiltersRejectInvalidQueries(t *testing.T) {
	for _, q := range []string{"days=0", "days=366", "offset=-1", "limit=101", "channel=secret", "status=anything", "days=x"} {
		if _, err := parseAIReportQuery(httptest.NewRequest("GET", "/api/ai/usage?"+q, nil), false); err == nil {
			t.Errorf("accepted %s", q)
		}
	}
	q, err := parseAIReportQuery(httptest.NewRequest("GET", "/api/ai/usage?days=7&channel=chat&status=failed&offset=25", nil), false)
	if err != nil || q.Days != 7 || q.Offset != 25 || q.Limit != 25 {
		t.Fatalf("query %+v %v", q, err)
	}
}

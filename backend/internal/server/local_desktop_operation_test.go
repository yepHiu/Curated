package server

import (
	"net/http/httptest"
	"testing"
)

func TestHostDesktopOperationsRejectRemoteAndForwardedClients(t *testing.T) {
	for _, tc := range []struct {
		remote, forwarded string
		allowed           bool
	}{
		{"127.0.0.1:4500", "", true}, {"[::1]:4500", "", true}, {"192.168.1.2:4500", "", false}, {"127.0.0.1:4500", "192.168.1.2", false},
	} {
		r := httptest.NewRequest("POST", "http://localhost/api/library/movies/a/reveal", nil)
		r.RemoteAddr = tc.remote
		if tc.forwarded != "" {
			r.Header.Set("X-Forwarded-For", tc.forwarded)
		}
		rr := httptest.NewRecorder()
		if got := requireLocalDesktopOperation(rr, r); got != tc.allowed {
			t.Fatalf("%+v: %v", tc, got)
		}
		if !tc.allowed && rr.Code != 403 {
			t.Fatal(rr.Code)
		}
	}
}

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithAccessLog_InfoLevel(t *testing.T) {
	core, obs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	h := WithAccessLog(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/library/movies", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	entries := obs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Message != "http_access" {
		t.Fatalf("unexpected message: %q", entries[0].Message)
	}
	if entries[0].ContextMap()["status"].(int64) != 418 {
		t.Fatalf("expected status 418, got %v", entries[0].ContextMap()["status"])
	}
}

func TestWithAccessLog_HealthIsDebug(t *testing.T) {
	core, obs := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	h := WithAccessLog(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	entries := obs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Level != zap.DebugLevel {
		t.Fatalf("expected debug for /api/health, got %v", entries[0].Level)
	}
}

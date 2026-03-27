package server

import (
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// accessLogQuiet returns true when the request should be logged at Debug (high-volume or noisy routes).
func accessLogQuiet(r *http.Request) bool {
	if r.Method == http.MethodGet && r.URL.Path == "/api/health" {
		return true
	}
	p := r.URL.Path
	if strings.Contains(p, "/api/library/movies/") && strings.HasSuffix(p, "/stream") {
		return true
	}
	if strings.Contains(p, "/api/curated-frames/") && strings.HasSuffix(p, "/image") {
		return true
	}
	return false
}

// WithAccessLog wraps the handler to record one HTTP access line per request (method, path, status, duration, remote).
// Noisy routes (health, video stream, frame image) use Debug; others use Info.
func WithAccessLog(logger *zap.Logger, next http.Handler) http.Handler {
	if logger == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sr, r)
		dur := time.Since(start)
		fields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", sr.status),
			zap.Int64("duration_ms", dur.Milliseconds()),
			zap.String("remote", r.RemoteAddr),
		}
		if accessLogQuiet(r) {
			logger.Debug("http_access", fields...)
		} else {
			logger.Info("http_access", fields...)
		}
	})
}

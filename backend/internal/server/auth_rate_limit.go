package server

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
)

const (
	authFailuresBeforeBackoff = 5
	authFailureResetWindow    = 15 * time.Minute
	authAttemptRetention      = 24 * time.Hour
	authBackoffBase           = time.Second
	authBackoffMaximum        = 5 * time.Minute
)

type authAttemptState struct {
	failures     int
	blockedUntil time.Time
	lastAttempt  time.Time
}

type authAttemptLimiter struct {
	mu       sync.Mutex
	attempts map[string]authAttemptState
	now      func() time.Time
}

type authAttemptDecision struct {
	Failures   int
	RetryAfter time.Duration
}

func newAuthAttemptLimiter() *authAttemptLimiter {
	return &authAttemptLimiter{
		attempts: make(map[string]authAttemptState),
		now:      time.Now,
	}
}

func (l *authAttemptLimiter) check(keys []string) authAttemptDecision {
	if l == nil {
		return authAttemptDecision{}
	}
	now := l.currentTime()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.removeExpired(now)

	decision := authAttemptDecision{}
	for _, key := range keys {
		state, ok := l.attempts[key]
		if !ok {
			continue
		}
		if state.failures > decision.Failures {
			decision.Failures = state.failures
		}
		if delay := state.blockedUntil.Sub(now); delay > decision.RetryAfter {
			decision.RetryAfter = delay
		}
	}
	return decision
}

func (l *authAttemptLimiter) recordFailure(keys []string) authAttemptDecision {
	if l == nil {
		return authAttemptDecision{}
	}
	now := l.currentTime()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.removeExpired(now)

	decision := authAttemptDecision{}
	for _, key := range keys {
		state := l.attempts[key]
		if !state.lastAttempt.IsZero() && now.Sub(state.lastAttempt) >= authFailureResetWindow {
			state = authAttemptState{}
		}
		state.failures++
		state.lastAttempt = now
		if state.failures >= authFailuresBeforeBackoff {
			delay := authBackoffForFailures(state.failures)
			state.blockedUntil = now.Add(delay)
		}
		l.attempts[key] = state
		if state.failures > decision.Failures {
			decision.Failures = state.failures
		}
		if delay := state.blockedUntil.Sub(now); delay > decision.RetryAfter {
			decision.RetryAfter = delay
		}
	}
	return decision
}

func (l *authAttemptLimiter) reset(keys []string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, key := range keys {
		delete(l.attempts, key)
	}
}

func (l *authAttemptLimiter) currentTime() time.Time {
	if l.now == nil {
		return time.Now().UTC()
	}
	return l.now().UTC()
}

func (l *authAttemptLimiter) removeExpired(now time.Time) {
	for key, state := range l.attempts {
		if !state.lastAttempt.IsZero() && now.Sub(state.lastAttempt) >= authAttemptRetention {
			delete(l.attempts, key)
		}
	}
}

func authBackoffForFailures(failures int) time.Duration {
	if failures < authFailuresBeforeBackoff {
		return 0
	}
	shift := failures - authFailuresBeforeBackoff
	if shift > 9 {
		shift = 9
	}
	delay := authBackoffBase * time.Duration(1<<shift)
	if delay > authBackoffMaximum {
		return authBackoffMaximum
	}
	return delay
}

func authAttemptKeys(r *http.Request) (keys []string, clientKey string, ip string) {
	clientKey, ip = authClientKey(r)
	if ip != "" {
		keys = append(keys, "ip:"+ip)
	}
	if clientKey != "" {
		keys = append(keys, "client:"+clientKey)
	}
	return keys, clientKey, ip
}

func (h *Handler) enforceAuthAttemptLimit(w http.ResponseWriter, r *http.Request, operation string) bool {
	keys, clientKey, ip := authAttemptKeys(r)
	decision := h.authAttempts.check(keys)
	if decision.RetryAfter <= 0 {
		return false
	}
	h.logAuthAttempt(operation, "rate_limited", clientKey, ip, decision)
	writeAuthRateLimitError(w, decision)
	return true
}

func (h *Handler) writeAuthAttemptFailure(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	status int,
	code string,
	message string,
) {
	keys, clientKey, ip := authAttemptKeys(r)
	decision := h.authAttempts.recordFailure(keys)
	h.logAuthAttempt(operation, "failed", clientKey, ip, decision)
	if decision.RetryAfter > 0 {
		writeAuthRateLimitError(w, decision)
		return
	}
	writeAppError(w, status, code, message)
}

func (h *Handler) resetAuthAttempts(r *http.Request) {
	keys, _, _ := authAttemptKeys(r)
	h.authAttempts.reset(keys)
}

func (h *Handler) logAuthAttempt(operation, outcome, clientKey, ip string, decision authAttemptDecision) {
	if h == nil || h.logger == nil {
		return
	}
	h.logger.Warn("authentication attempt", zap.String("operation", operation), zap.String("outcome", outcome), zap.String("ip", ip), zap.String("clientKey", clientKey), zap.Int("failures", decision.Failures), zap.Int("retryAfterSeconds", retryAfterSeconds(decision.RetryAfter)))
}

func writeAuthRateLimitError(w http.ResponseWriter, decision authAttemptDecision) {
	seconds := retryAfterSeconds(decision.RetryAfter)
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeJSON(w, http.StatusTooManyRequests, contracts.AppError{
		Code:      contracts.ErrorCodeAuthRateLimited,
		Message:   "Too many PIN attempts. Try again later.",
		Retryable: true,
		Details: map[string]any{
			"retryAfterSeconds": seconds,
		},
	})
}

func retryAfterSeconds(delay time.Duration) int {
	if delay <= 0 {
		return 1
	}
	seconds := int((delay + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

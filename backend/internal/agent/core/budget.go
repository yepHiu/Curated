package core

import (
	"sync"
	"time"
)

type budgetTracker struct {
	mu     sync.Mutex
	steps  map[string]int
	writes map[string][]time.Time
}

func newBudgetTracker() *budgetTracker {
	return &budgetTracker{
		steps:  map[string]int{},
		writes: map[string][]time.Time{},
	}
}

func (b *budgetTracker) resetSteps(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := sessionID
	if key == "" {
		key = "_"
	}
	delete(b.steps, key)
}

func (b *budgetTracker) addStep(sessionID string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := sessionID
	if key == "" {
		key = "_"
	}
	b.steps[key]++
	return b.steps[key]
}

func (b *budgetTracker) writeCount(sessionID string, now time.Time) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := sessionID
	if key == "" {
		key = "_"
	}
	cutoff := now.Add(-time.Minute)
	kept := b.writes[key][:0]
	for _, ts := range b.writes[key] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	b.writes[key] = kept
	return len(kept)
}

func (b *budgetTracker) addWrite(sessionID string, now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := sessionID
	if key == "" {
		key = "_"
	}
	b.writes[key] = append(b.writes[key], now)
}

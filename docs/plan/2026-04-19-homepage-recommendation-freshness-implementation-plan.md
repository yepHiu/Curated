# Homepage Recommendation Freshness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep homepage hero and recommendation rows fresh across a rolling week when the library has enough inventory, while degrading gracefully to shorter exclusion windows when it does not.

**Architecture:** The backend homepage snapshot generator remains daily, but its cross-day de-duplication policy changes from "hard exclude yesterday + soft historical penalty" to a fallback ladder of recent hard-exclusion windows. Candidate ranking, same-day dedupe, and diversity penalties stay in place; only the recent exclusion policy changes.

**Tech Stack:** Go, SQLite-backed storage, existing homepage snapshot generation in `backend/internal/app`.

---

### Task 1: Add failing tests for rolling-window freshness and fallback

**Files:**
- Modify: `backend/internal/app/homepage_daily_recommendations_test.go`

- [ ] **Step 1: Write the failing tests**

Add tests covering:

1. A large library where the current day can fully avoid the combined last-7-day slate.
2. A smaller library where last-7-day avoidance is impossible, but the generator can degrade to an older window and still fill the slate.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/app -run HomepageDailyRecommendations`

Expected:
- FAIL because the current implementation only hard-excludes yesterday, so it will still reuse titles from the last 7 days in the first scenario.
- FAIL because there is no explicit multi-window fallback policy in the second scenario.

- [ ] **Step 3: Implement minimal production changes**

Update homepage recommendation generation so it:

1. Builds recent-exposure exclusion sets for windows `7`, `5`, `3`, `1`, and `0` days.
2. Selects hero and recommendation IDs by trying each exclusion set in order until the slate is full.
3. Preserves same-day dedupe and diversity penalties.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/app -run HomepageDailyRecommendations`

Expected: PASS

### Task 2: Refresh generation metadata and diagnostics

**Files:**
- Modify: `backend/internal/app/homepage_daily_recommendations.go`
- Modify: `docs/plan/2026-04-19-homepage-recommendation-repeat-analysis.md`

- [ ] **Step 1: Bump generation version and update logs**

Adjust:
- generation version constant
- log fields so they report which exclusion window was actually used for hero and recommendation selection

- [ ] **Step 2: Record the implemented policy**

Update the analysis doc to reflect:
- rolling 7-day hard exclusion
- fallback ladder `7 -> 5 -> 3 -> 1 -> 0`
- why daily generation remains the default

- [ ] **Step 3: Run targeted verification again**

Run: `cd backend && go test ./internal/app -run HomepageDailyRecommendations`

Expected: PASS

### Task 3: Full verification

**Files:**
- Modify: `backend/internal/app/homepage_daily_recommendations.go`
- Modify: `backend/internal/app/homepage_daily_recommendations_test.go`

- [ ] **Step 1: Run broader backend tests**

Run: `cd backend && go test ./...`

Expected: PASS

- [ ] **Step 2: Summarize verification scope**

Confirm the final behavior covers:
- no same-day duplicates across hero and recommendation rows
- no repeats from the last 7 days when inventory allows
- graceful fallback to shorter exclusion windows when inventory is insufficient


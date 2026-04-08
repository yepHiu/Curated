# Dev Performance Monitor Bar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a development-only bottom overlay bar that shows frontend/backend integration metrics without affecting page layout.

**Architecture:** Keep the shell integration thin by mounting the bar from `AppShell` only in development. Put request aggregation and frontend sampling into dedicated monitor modules, and expose CPU metrics from a small backend development endpoint backed by a lightweight sampler.

**Tech Stack:** Vue 3, TypeScript, Vite, Vitest, Go HTTP server, Windows CPU sampling via Go platform-specific files.

---

### Task 1: Backend CPU Summary Endpoint

**Files:**
- Create: `backend/internal/devmetrics/cpu_sampler.go`
- Create: `backend/internal/devmetrics/cpu_sampler_windows.go`
- Create: `backend/internal/devmetrics/cpu_sampler_stub.go`
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/server/server_test.go`
- Test: `backend/internal/server/server_test.go`

- [ ] Add failing server tests for `GET /api/dev/performance`
- [ ] Run: `go test ./internal/server -run TestHandleGetDevPerformance`
- [ ] Add DTOs and app/server wiring for the development summary endpoint
- [ ] Add CPU sampler implementation with Windows-specific process/system CPU support and non-Windows fallback
- [ ] Run: `go test ./internal/server ./internal/app`
- [ ] Commit this backend slice

### Task 2: Frontend Request Monitor Core

**Files:**
- Create: `src/lib/dev-performance/request-monitor.ts`
- Create: `src/lib/dev-performance/request-monitor.test.ts`
- Modify: `src/api/http-client.ts`

- [ ] Add failing tests for request window aggregation and summary statistics
- [ ] Run: `pnpm vitest run src/lib/dev-performance/request-monitor.test.ts`
- [ ] Implement request monitor core and wire `http-client` request lifecycle hooks
- [ ] Run: `pnpm vitest run src/lib/dev-performance/request-monitor.test.ts`
- [ ] Commit this request-monitor slice

### Task 3: Frontend Runtime Sampling Core

**Files:**
- Create: `src/lib/dev-performance/frontend-monitor.ts`
- Create: `src/lib/dev-performance/frontend-monitor.test.ts`
- Modify: `src/main.ts`

- [ ] Add failing tests for summary formatting / monitor state transitions that can be tested without DOM-heavy component mounts
- [ ] Run: `pnpm vitest run src/lib/dev-performance/frontend-monitor.test.ts`
- [ ] Implement frontend sampler for FPS, long tasks, memory, route timing, and lightweight player-video quality probing
- [ ] Run: `pnpm vitest run src/lib/dev-performance/frontend-monitor.test.ts`
- [ ] Commit this sampler slice

### Task 4: Dev Performance Bar UI Integration

**Files:**
- Create: `src/composables/use-dev-performance-monitor.ts`
- Create: `src/components/dev/DevPerformanceBar.vue`
- Modify: `src/layouts/AppShell.vue`
- Modify: `src/api/endpoints.ts`
- Modify: `src/api/types.ts`

- [ ] Add failing tests for monitor summary formatting helpers if extracted, or for the composable’s derived summary state
- [ ] Run: `pnpm vitest run src/lib/dev-performance/request-monitor.test.ts src/lib/dev-performance/frontend-monitor.test.ts`
- [ ] Implement the development-only Teleport bar with collapsed and expanded states plus minimal controls
- [ ] Wire backend summary polling and frontend monitors into the composable
- [ ] Run: `pnpm typecheck`
- [ ] Run: `pnpm lint`
- [ ] Commit this UI slice

### Task 5: Verification and Docs Sync

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `docs/architecture-and-implementation.html`
- Modify: `docs/plan/dev-performance-monitor-bar-plan.md`

- [ ] Sync docs for the new development diagnostics endpoint and overlay behavior
- [ ] Run: `pnpm test`
- [ ] Run: `pnpm build`
- [ ] Run: `cd backend && go test ./...`
- [ ] Commit the docs/verification slice

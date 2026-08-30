---
name: curated-regression-selector
description: Select and report the smallest evidence-based Curated test and build set for a change, based on affected paths and runtime boundaries.
---

# Curated Regression Selector

Choose verification from changed behavior, not habit. Read `docs/ops/2026-04-08-agent-build-and-test.md`, the diff, and nearby tests before recommending commands.

## Mapping

| Changed area | Baseline evidence |
| --- | --- |
| Vue component, composable, frontend service, locale | focused Vitest; `pnpm typecheck`; add `pnpm lint` when code changed |
| Vite/runtime bootstrap or browser-only flow | focused Playwright, then `pnpm test:e2e` when the flow is material |
| Electron main, preload, desktop shell | focused Electron tests; `pnpm test:electron`; build Electron main when packaging/runtime integration changes |
| Go app/service/handler/contract/storage | focused package test; `cd backend && go test ./...`; `go vet ./...` for non-trivial Go changes |
| SQLite migration/backup/maintenance | focused upgrade/integrity tests plus backend suite; use `curated-sqlite-migration` or `curated-maintenance-safety` |
| release scripts/packaging | relevant Python unit tests and `curated-release-readiness`; do not package merely to test ordinary code |
| shared styles, shell, grids, dialogs, typography | relevant UI tests/E2E and display-scaling checklist; `pnpm test:display` only with explicit approval |

`pnpm build` is appropriate for frontend release/build-impacting changes; it includes typecheck and bundle budgets. Do not run broad suites merely because they exist, and do not claim coverage a test does not exercise.

Report commands run, behavior each one demonstrates, and high-value checks intentionally skipped with reason. Respect the required working directories: frontend commands at repository root and Go commands inside `backend/`; never place Go caches in the repository.

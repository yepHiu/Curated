---
name: curated-e2e-scenario
description: Add or refine Curated Playwright runtime scenarios for user flows and regressions, using stable semantic assertions and isolated Mock/Web API behavior.
---

# Curated E2E Scenario

Use this Skill when a browser-level behavior, bootstrap flow, navigation flow, or bug regression cannot be established by focused unit tests alone.

## Choose the right boundary

Read `playwright.e2e.config.ts`, `tests/e2e/`, and the closest scenario. `pnpm test:e2e` starts isolated Mock and Web API stub Vite servers; it must not depend on a developer's 5173/8080 processes. Test Mock behavior without backend requests where that is the contract, and stub Web API requests explicitly.

Describe the user outcome and the smallest flow that proves it. Prefer semantic locators, accessible roles/names, and stable test IDs only where semantics are not sufficient. Do not assert incidental Tailwind classes, timing, or internal implementation structure.

## Isolation and coverage

- Make unexpected API requests fail loudly rather than silently reaching a local backend.
- Cover a meaningful failure or unavailable state when the change can fail, not only a happy path.
- Keep test data minimal and independent; avoid depending on execution order or localStorage left by another test.
- Reserve `tests/display-scaling/` and `pnpm test:display` for explicitly approved display work; ordinary E2E belongs under `tests/e2e/`.

Run the focused scenario and, when reasonable, `pnpm test:e2e`. Report coverage gained and deliberate omissions.

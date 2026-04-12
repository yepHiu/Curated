# Curated Backend Runtime Skill Plan

Date: 2026-04-12

## Goal

Create a repo-local skill that tells agents exactly how to build and run the Curated Go backend in:

- development mode
- production / release mode

The skill must reduce recurring mistakes around:

- using `go run` when the request is actually to compile a dev binary
- compiling a dev Windows backend binary to `curated.exe`
- mixing up dev and release ports / identities / binary names

## Decision

Create a dedicated skill under `.cursor/skills/curated-backend-runtime/` instead of extending the packaging skill.

Reason:

- packaging requests and backend runtime requests are related but not the same workflow
- backend compile/run requests are frequent day-to-day operations, not just release actions
- the naming conflict rule is important enough to deserve a focused skill

## Required Coverage

The skill should explicitly document:

- dev run: `cd backend && go run ./cmd/curated`
- dev build: `pnpm backend:build:dev`
- dev Windows binary output: `backend/runtime/curated-dev.exe`
- release backend build: `pnpm release:backend`
- full production packaging: `pnpm release:publish`
- release Windows binary output: `release/backend/curated.exe`
- dev default port / identity: `:8080`, `curated-dev`
- release default port / identity: `:8081`, `curated`

## Hard Rules

- Windows dev backend binaries must use `curated-dev.exe`
- `curated.exe` is reserved for release / packaged builds
- do not leave a dev Windows backend binary named `curated.exe` in the workspace

## Integration

Add the skill as a repo-local skill and mention it from the repo context so future agents can discover it quickly.

## Validation

Validate at least:

- skill folder structure is correct
- `SKILL.md` matches actual repo commands and paths
- `agents/openai.yaml` exists and reflects the skill purpose

---
name: curated-dev-start
description: Use when working in the Curated project and the user asks to start, restart, run, launch, or verify the local development environment, frontend, backend, Vite, Go API, Web API mode, ports 5173 or 8080.
---

# Curated Dev Start

## Overview

Start the Curated development environment with one project-local script. Prefer this skill over manually retyping `go run ./cmd/curated` and `pnpm dev`.

## Quick Start

Run from the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File .agents\skills\curated-dev-start\scripts\start-curated-dev.ps1
```

The script:

- Ensures `.env` contains `VITE_USE_WEB_API=true`.
- Reuses an existing healthy backend on `http://127.0.0.1:8080`.
- Starts the Go backend from `backend/` only when needed.
- Reuses an existing Vite frontend on `http://127.0.0.1:5173`.
- Starts `pnpm dev -- --host 127.0.0.1` only when needed.
- Writes logs to `.workspace/dev-logs/`.
- Verifies backend health and frontend HTTP status before reporting success.

## Commands

| Task | Command |
| --- | --- |
| Start or verify both services | `powershell -ExecutionPolicy Bypass -File .agents\skills\curated-dev-start\scripts\start-curated-dev.ps1` |
| Restart both services | `powershell -ExecutionPolicy Bypass -File .agents\skills\curated-dev-start\scripts\start-curated-dev.ps1 -Restart` |
| Use another frontend port | `powershell -ExecutionPolicy Bypass -File .agents\skills\curated-dev-start\scripts\start-curated-dev.ps1 -FrontendPort 5174` |
| Use another backend port for checks | `powershell -ExecutionPolicy Bypass -File .agents\skills\curated-dev-start\scripts\start-curated-dev.ps1 -BackendPort 8080` |

## Rules

- Run the backend command from `backend/`, never from the repository root.
- Keep Go caches in default user locations; do not point `GOCACHE`, `GOMODCACHE`, or `GOTMPDIR` into the repo.
- Do not kill unrelated processes unless `-Restart` is used and the port owner is identified as the expected Curated/Vite process.
- If a port is occupied by an unrelated process, stop and report the owner PID and command line.
- Preserve logs under `.workspace/dev-logs/`; do not write ad-hoc logs in the repo root.

## Expected Result

Backend:

```text
http://127.0.0.1:8080/api/health
```

Frontend:

```text
http://127.0.0.1:5173/
```

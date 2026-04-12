---
name: curated-backend-runtime
description: Use when the user asks to build, compile, run, restart, or inspect the Curated Go backend in this repository, especially when choosing between dev go run, dev curated-dev.exe, release curated.exe, release ports, or production packaging.
---

# Curated Backend Runtime

## Overview

Use this repo-local skill for Curated backend build and runtime choices. The goal is to keep development and production backend binaries, ports, and identities separated.

## Decision Table

| Request | Use | Output / Runtime |
| --- | --- | --- |
| Run the dev backend now, without needing an exe | `cd backend && go run ./cmd/curated` | dev backend on `:8080`, health name `curated-dev` |
| Compile the dev backend on Windows | `pnpm backend:build:dev` | `backend/runtime/curated-dev.exe` |
| Run the compiled dev backend | `.\backend\runtime\curated-dev.exe` | dev backend on `:8080`, health name `curated-dev` |
| Build a release-shaped backend binary only | `pnpm release:backend` | `release/backend/curated.exe` with package script version |
| Build the production package | `pnpm release:publish` | portable zip, installer, manifest, and release `curated.exe` |

## Hard Rules

- Development Windows backend binaries must be named `curated-dev.exe`.
- Do not compile or leave a development Windows backend binary named `curated.exe`.
- Reserve `curated.exe` for release / packaged builds only.
- Do not run Go commands from the repository root when the package path is `./cmd/curated`; run from `backend/`.
- Do not use `go run` for production-package validation. Use release scripts.

## Development Workflows

Run dev backend directly:

```powershell
cd backend
go run ./cmd/curated
```

Compile dev backend:

```powershell
pnpm backend:build:dev
```

Run compiled dev backend from the repo root:

```powershell
.\backend\runtime\curated-dev.exe
```

Expected dev identity:

- default HTTP port: `:8080`
- health name: `curated-dev`
- default file log directory: `backend/runtime/logs`

## Release Workflows

Build only the release backend binary:

```powershell
pnpm release:backend
```

Use this only when the request is specifically about the backend release binary. For a real distributable production build, prefer:

```powershell
pnpm release:publish
```

Expected release identity:

- default HTTP port: `:8081`
- health name: `curated`
- release backend binary: `release/backend/curated.exe`
- default file log directory: `LOCALAPPDATA\Curated\logs`

## Common Mistakes

| Mistake | Correct action |
| --- | --- |
| User asks to run dev backend and agent compiles an exe first | Use `cd backend && go run ./cmd/curated` unless a binary is requested |
| User asks to compile dev backend and agent runs `go run` | Use `pnpm backend:build:dev` |
| Agent builds dev backend as `curated.exe` | Stop and rebuild as `backend/runtime/curated-dev.exe` |
| Agent runs `go run ./cmd/curated` from repo root | Change directory to `backend` first |
| Agent uses `pnpm release:backend` for full production package | Use `pnpm release:publish` |

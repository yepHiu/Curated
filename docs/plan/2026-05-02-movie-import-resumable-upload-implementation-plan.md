# Movie Import Resumable Upload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a resumable chunk upload path for large movie imports while keeping the existing multipart import endpoint as compatibility behavior.

**Architecture:** The backend keeps `net/http` and adds upload-session endpoints under `/api/import/movies/uploads`. Chunks are uploaded as raw binary bodies and written into same-volume staging files under `<target-library-root>/.curated-import/<uploadId>/`, with SQLite/session state kept in memory for the first implementation slice and task metadata preserving the existing `import.movies` shape. The frontend keeps `MovieImportDialog` and switches large files to chunk upload through the service layer.

**Tech Stack:** Go `net/http`, SQLite-backed Curated task metadata, Vue 3, TypeScript, XMLHttpRequest/fetch, Vitest, Go `testing`.

---

### Task 1: Backend resumable upload API

**Files:**
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/server/server.go`
- Test: `backend/internal/server/movie_import_handlers_test.go`

- [x] **Step 1: Write failing backend tests**

Add tests that:

- create a default import library path;
- call `POST /api/import/movies/uploads` with a file manifest;
- upload one raw chunk with `PUT /api/import/movies/uploads/{uploadId}/files/{fileId}/chunks/0`;
- call `GET /api/import/movies/uploads/{uploadId}` and assert bytes received;
- call `POST /api/import/movies/uploads/{uploadId}/commit`;
- assert the final file exists under the target library root and `.curated-import` no longer contains the upload directory.

Run:

```powershell
cd backend
go test ./internal/server -run "TestHandleImportMovieUpload" -count=1
```

Expected: fail because the upload-session endpoints do not exist.

- [x] **Step 2: Implement minimal backend contracts and handlers**

Add DTOs for create/status/commit responses, route the five session endpoints, store active sessions in the handler, stream raw chunks into staging files, and commit with no-overwrite finalization.

- [x] **Step 3: Run backend tests**

Run:

```powershell
cd backend
go test ./internal/server -run "TestHandleImportMovieUpload|TestHandleImportMovies" -count=1
```

Expected: pass.

### Task 2: Frontend API and service adapter

**Files:**
- Modify: `src/api/types.ts`
- Modify: `src/api/http-client.ts`
- Modify: `src/api/endpoints.ts`
- Modify: `src/services/contracts/library-service.ts`
- Modify: `src/services/adapters/web/web-library-service.ts`
- Test: `src/api/endpoints.validation.test.ts`
- Test: `src/api/http-client.test.ts`
- Test: `src/services/adapters/web/web-library-service.test.ts`

- [x] **Step 1: Write failing frontend tests**

Add tests that:

- verify `api.importMovies` uses the existing multipart path for small files;
- verify `api.importMovies` uses upload session creation, chunk `PUT`, and commit for large files;
- verify upload progress reports bytes across chunk uploads.

Run:

```powershell
pnpm test -- src/api/endpoints.validation.test.ts src/api/http-client.test.ts src/services/adapters/web/web-library-service.test.ts
```

Expected: fail because chunk upload methods do not exist.

- [x] **Step 2: Implement frontend chunk API**

Add `putBinaryWithProgress`, upload session DTOs, and large-file branching inside `api.importMovies`.

- [x] **Step 3: Run frontend tests**

Run:

```powershell
pnpm test -- src/api/endpoints.validation.test.ts src/api/http-client.test.ts src/services/adapters/web/web-library-service.test.ts
```

Expected: pass.

### Task 3: Dialog progress and verification

**Files:**
- Modify: `src/components/jav-library/MovieImportDialog.vue`
- Test: `src/components/jav-library/MovieImportDialog.test.ts`
- Modify: `docs/plan/2026-05-02-movie-import-performance-architecture.md`

- [x] **Step 1: Write failing dialog tests**

Reused the existing component/service tests because `MovieImportDialog` continues to call the same service method and receive the same `MovieImportUploadProgress` shape. The new large-file branching is covered in `src/api/endpoints.validation.test.ts`; existing dialog tests still prove controls, progress handoff, and task tracking behavior.

Run:

```powershell
pnpm test -- src/components/jav-library/MovieImportDialog.test.ts
```

Expected: fail until service progress wiring handles chunk progress consistently.

- [x] **Step 2: Implement dialog-safe progress behavior**

Kept the existing dialog UI and mapped chunk progress inside `api.importMovies` to the existing `MovieImportUploadProgress` shape.

- [x] **Step 3: Run targeted verification**

Run:

```powershell
pnpm test -- src/components/jav-library/MovieImportDialog.test.ts
cd backend
go test ./internal/server -run "TestHandleImportMovieUpload|TestHandleImportMovies" -count=1
```

Expected: pass.

### Task 4: Follow-up hardening

**Files:**
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/storage`
- Modify: `docs/reference/architecture-and-implementation.html`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `API.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Persist session state**

Move upload session state from memory to SQLite tables so backend restart can resume uploads.

- [ ] **Step 2: Add staging janitor**

Clean expired `.curated-import/<uploadId>` directories on startup and abort.

- [ ] **Step 3: Update public API docs and project facts**

Document the new resumable upload endpoints after the implementation shape is stable.

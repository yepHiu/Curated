# Movie Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans or follow this plan inline with TDD. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Add a top-bar movie import flow that uploads selected local video files to the configured default library path, tracks copy progress, handles failures, and triggers the existing library scan pipeline.

**Architecture:** The MVP uses browser file selection/drag-drop and uploads file streams to the local Go backend with multipart form data. The backend writes each file under the default library path, reports task progress through the existing `TaskDTO` system, and starts a restricted scan after successful copies. Frontend state stays in the service/API layer and the UI reuses shadcn-vue Dialog, Button, Select, Progress, Badge, and the existing toast/Dock patterns.

**Tech Stack:** Vue 3 + TypeScript + Vite, shadcn-vue primitives, vue-sonner, Go `net/http`, SQLite settings/config file, existing in-memory task manager.

---

## Files

- Modify: `backend/internal/contracts/contracts.go`
  - Add import request/response DTOs, settings fields, task metadata conventions, and error codes.
- Modify: `backend/internal/config/config.go`
  - Add default import library path setting to runtime config.
- Modify: `backend/internal/config/library_settings.go`
  - Persist and merge `defaultImportLibraryPathId`.
- Modify: `backend/internal/app/app.go`
  - Add getter/setter for default import path, import task orchestration, and scan handoff.
- Modify: `backend/internal/server/server.go`
  - Add `POST /api/import/movies`, include setting in GET/PATCH settings, and expose task errors.
- Modify: `backend/internal/tasks/manager.go`
  - Add a partial-fail helper so multi-file import can finish as `partial_failed`.
- Add/modify backend tests in `backend/internal/config`, `backend/internal/tasks`, `backend/internal/server`.
- Modify: `src/api/types.ts`
  - Add settings field, import task types, and `TaskDTO.metadata` typing helpers where useful.
- Modify: `src/api/endpoints.ts`
  - Add multipart import API with upload progress support.
- Modify: `src/services/contracts/library-service.ts`
  - Add default import path and import methods.
- Modify: `src/services/adapters/web/web-library-service.ts`
  - Wire settings and import API.
- Modify: `src/services/adapters/mock/mock-library-service.ts`
  - Add mock import behavior and default import path state.
- Modify: `src/composables/use-scan-task-tracker.ts`
  - Generalize terminal task toast/reload behavior for `import.movies`.
- Modify: `src/components/jav-library/ScanProgressDock.vue`
  - Render import-specific metadata: bytes, current file, file counts, and failures.
- Add: `src/components/jav-library/MovieImportDialog.vue`
  - Main import UI.
- Modify: `src/layouts/AppShell.vue`
  - Add top-bar "Add movies" trigger beside theme toggle.
- Modify: `src/components/jav-library/settings/SettingsLibraryPathsSection.vue`
  - Add default import path selection in storage card.
- Modify: `src/components/jav-library/SettingsPage.vue`
  - Wire default import path setting.
- Modify: `src/locales/zh-CN.json`, `src/locales/en.json`, `src/locales/ja.json`
  - Add UI strings.
- Sync docs after implementation: `.cursor/rules/project-facts.mdc`, `README.md`, `CLAUDE.md`, `docs/reference/architecture-and-implementation.html`.

## Tasks

### Task 1: Backend Settings And Task Metadata

- [x] Write failing tests for `defaultImportLibraryPathId` merge/persist in `backend/internal/config/library_settings_test.go`.
- [x] Implement config field merge/persist.
- [x] Write failing tests for task partial failure helper in `backend/internal/tasks/manager_test.go`.
- [x] Implement task partial failure helper.
- [x] Run targeted Go tests.

### Task 2: Backend Import API

- [x] Write failing server tests for:
  - `GET /api/settings` returns `defaultImportLibraryPathId`.
  - `PATCH /api/settings` persists a valid library path id.
  - `POST /api/import/movies` rejects missing/default target.
  - `POST /api/import/movies` writes uploaded files under the target root and returns a task.
  - one failing file yields `partial_failed` metadata while successful files remain copied.
- [x] Implement handler and app import orchestration with multipart streaming.
- [x] Trigger restricted scan on copied files' target root after successful copy.
- [x] Run targeted backend server tests.

### Task 3: Frontend Service And Progress Plumbing

- [x] Write failing Vitest tests for API multipart upload progress and service default import path state.
- [x] Implement API/service methods.
- [x] Extend task tracker/Dock tests for import task metadata and terminal notifications.
- [x] Implement import-specific tracker/Dock rendering.
- [x] Run targeted frontend tests.

### Task 4: Frontend UI

- [x] Write component tests for `MovieImportDialog`:
  - opens from trigger.
  - accepts file selection.
  - disables submit without default path or files.
  - calls import service and starts task tracking.
  - shows upload/task errors.
- [x] Implement `MovieImportDialog`.
- [x] Add top-bar trigger in `AppShell.vue`.
- [x] Add default import path select to settings storage card and wire through `SettingsPage.vue`.
- [x] Run targeted component tests.

### Task 5: Documentation And Full Verification

- [x] Sync public/project docs for new settings field and API endpoint.
- [x] Run `pnpm typecheck`.
- [x] Run targeted Vitest files and relevant `pnpm test -- ...` checks.
- [x] Run targeted Go tests, then `cd backend && go test ./...` if targeted tests pass.
- [x] Report any skipped broad verification explicitly.

## Implementation Notes

- The frontend shows immediate upload progress in `MovieImportDialog` and task-level copy/scan progress in `ScanProgressDock`.
- Backend import failure handling is per-file: conflicts and copy errors are recorded in task metadata; a mixed result finishes as `partial_failed`, while a total failure finishes as `failed`.
- The MVP copies files into the default library root and never deletes or overwrites source files.

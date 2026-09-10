# Comic Library Auto Watch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or execute the checklist step-by-step with TDD. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an independent fsnotify-based comic library auto watch flow that scans newly added `.zip` / `.cbz` archives under configured comic storage paths.

**Architecture:** Keep movie and comic domains separate. Add `autoComicLibraryWatch` as a comic-specific persisted setting, create `backend/internal/comicwatch`, route watcher events to App-owned `scan.comics` tasks, and expose a comic settings toggle plus comic watch toasts.

**Tech Stack:** Go backend, SQLite repositories, `fsnotify`, Vue 3, TypeScript, shadcn-vue, Vitest.

---

### Task 1: Persist `autoComicLibraryWatch`

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/library_settings.go`
- Modify: `backend/internal/config/comic_settings_test.go`
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `src/api/types.ts`

- [x] Add failing Go tests that default config returns `AutoComicLibraryWatch=true` and `MergeLibrarySettingsFile` reads `"autoComicLibraryWatch": false`.
- [x] Add `AutoComicLibraryWatch bool` to backend config and contracts.
- [x] Add `autoComicLibraryWatch` to frontend settings DTO and patch request types.
- [x] Run `go test ./internal/config ./internal/contracts` from `backend/`.

### Task 2: App-Level Comic Watch Settings and Scan Starter

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/comic_settings_test.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/server/comic_scan_handlers.go`
- Modify: `backend/internal/server/comic_paths_handlers_test.go`
- Modify: `backend/internal/server/comic_scan_handlers_test.go`

- [x] Add failing App tests that `SetAutoComicLibraryWatch(false)` persists the setting and that App can create `scan.comics` tasks with extra metadata.
- [x] Add App methods `AutoComicLibraryWatch`, `SetAutoComicLibraryWatch`, `StartComicScan`, and internal `startComicScan`.
- [x] Change HTTP comic scan/import paths to use `ComicScanStarter` backed by App in normal runtime.
- [x] Extend settings handler/controller interfaces for `autoComicLibraryWatch`.
- [x] Run focused backend tests for `internal/app` and `internal/server`.

### Task 3: Independent Comic Watcher

**Files:**
- Create: `backend/internal/comicwatch/watcher.go`
- Create: `backend/internal/comicwatch/watcher_test.go`
- Modify: `backend/internal/storage/comic_paths_repository.go`
- Modify: `backend/internal/storage/comic_paths_repository_test.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/cmd/curated/main.go`

- [x] Add failing watcher tests for `.zip` / `.cbz` create/write events and ignored `.tmp`, `.part`, hidden, and non-archive files.
- [x] Add `ListComicLibraryPathStrings(ctx)` to storage.
- [x] Implement `comicwatch.Watcher` with comic-specific interfaces and archive filtering.
- [x] Add App methods `EnsureComicLibraryWatchRunning`, `StopComicLibraryWatchLoop`, `ReloadComicLibraryWatches`, and `EnqueueComicLibraryWatchScanRoots`.
- [x] Start comic watcher at boot only when yaml watch is on, comic library is enabled, and `autoComicLibraryWatch` is true.
- [x] Run `go test ./internal/comicwatch ./internal/storage ./internal/app`.

### Task 4: Frontend Settings Toggle

**Files:**
- Modify: `src/services/contracts/comic-library-service.ts`
- Modify: `src/services/adapters/web/web-comic-library-service.ts`
- Modify: `src/services/adapters/mock/mock-comic-library-service.ts`
- Modify: `src/components/jav-library/settings/SettingsComicLibrarySection.vue`
- Modify: `src/components/jav-library/settings/SettingsComicLibrarySection.test.ts`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`

- [x] Add failing frontend tests for the comic auto watch toggle rendering and patching `autoComicLibraryWatch`.
- [x] Add service state and setter `setAutoComicLibraryWatch`.
- [x] Add a comic settings row styled like existing settings controls.
- [x] Run `pnpm vitest run src/components/jav-library/settings/SettingsComicLibrarySection.test.ts src/services/adapters/web/web-comic-library-service.test.ts`.

### Task 5: Comic Watch Toasts

**Files:**
- Modify: `src/composables/use-library-watch-toasts.ts`
- Modify: `src/composables/use-library-watch-toasts.test.ts`
- Modify: `src/composables/use-scan-task-tracker.ts`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`

- [x] Add failing tests for terminal `scan.comics` with `metadata.trigger="fsnotify"`.
- [x] Show comic-specific auto scan messages and reload comics, not movies.
- [x] Keep manual comic scan toast behavior in `use-scan-task-tracker`.
- [x] Run focused composable tests.

### Task 6: Docs and Verification

**Files:**
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `docs/reference/architecture-and-implementation.html`
- Modify: `docs/reference/2026-03-21-library-organize.md` if `library-config.cfg` behavior summary changes.

- [x] Update docs to remove comic auto watch from MVP exclusions and list `autoComicLibraryWatch`.
- [x] Run `pnpm typecheck`, `pnpm lint`, `pnpm test`, and focused Go tests.
- [x] Validate Settings -> Comics in browser: toggle renders, long comic import filename stays inside the dialog, and no relevant console errors appear.

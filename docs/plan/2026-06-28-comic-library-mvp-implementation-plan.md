# Comic Library MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Curated's optional, strongly independent comic library MVP for `.zip` / `.cbz` comics, including setup, scanning, import, library browsing, details, reader, progress, preferences, tags, ratings, and cache governance.

**Architecture:** Implement comics as a separate media domain beside movies: independent SQLite tables, settings keys, repositories, scanner, import routes, API DTOs, frontend service contract, routes, views, components, and mock adapter. Reuse only generic infrastructure such as tasks, HTTP helpers, storage-health probing patterns, settings UI conventions, notifications, and virtualized grid ideas.

**Tech Stack:** Go 1.25 backend, SQLite migrations/repositories, Go `archive/zip`, image thumbnail generation using existing image dependencies, Vue 3 + TypeScript + Vite, shadcn-vue/Tailwind v4, vue-router, vue-virtual-scroller, Vitest, Go test.

---

## Implementation Ground Rules

- Create an isolated git worktree before code changes. The current workspace may contain unrelated user or agent changes.
- Do not mix comic data with movie tables or movie service types.
- Do not extend `Movie`, `LibraryService`, `/api/library/movies`, or movie import routes to carry comics.
- Keep comic tags fully isolated from movie tags.
- Keep comic cache separate from movie assets/cache.
- Never delete source `.zip` / `.cbz` files in MVP behavior.
- First implementation should be split into the tasks below, with one atomic commit per task or per tightly coupled test/implementation pair.
- Do not run `pnpm test:display` unless the user explicitly approves it.
- Use repository test commands from `docs/ops/2026-04-08-agent-build-and-test.md`:
  - Frontend root: `pnpm typecheck`, `pnpm lint`, `pnpm test -- <file>`, `pnpm test`
  - Backend: `cd backend && go test ./...`

## File Map

### Backend Core

- Create `backend/internal/storage/migrations/0026_comic_library.sql`: comic tables and indexes.
- Modify `backend/internal/storage/sqlite.go`: include migration 0026.
- Create `backend/internal/contracts/comic_contracts.go`: comic DTOs, settings DTOs, request DTOs, task/error constants.
- Modify `backend/internal/contracts/contracts.go`: add comic fields to the existing `SettingsDTO` and `PatchSettingsRequest`.
- Create `backend/internal/config/comic_settings.go`: normalize comic reader/cache defaults and parse/write library-config keys.
- Modify `backend/internal/config/config.go`: add comic config fields with defaults.
- Modify `backend/internal/config/library_settings.go`: merge/write `comicLibraryEnabled`, `defaultComicImportLibraryPathId`, `comicReader`, `comicCache`.

### Backend Storage

- Create `backend/internal/storage/comic_paths_repository.go`
- Create `backend/internal/storage/comic_books_repository.go`
- Create `backend/internal/storage/comic_pages_repository.go`
- Create `backend/internal/storage/comic_tags_repository.go`
- Create `backend/internal/storage/comic_progress_repository.go`
- Create `backend/internal/storage/comic_preferences_repository.go`
- Create `backend/internal/storage/comic_cache_repository.go`
- Add matching `*_test.go` files for each repository slice.

### Backend Domain Services

- Create `backend/internal/comicarchive/service.go`: zip/cbz open, page filtering, natural sort, page extraction.
- Create `backend/internal/comicarchive/service_test.go`
- Create `backend/internal/comicscanner/service.go`: scan configured comic paths and upsert comic books/pages.
- Create `backend/internal/comicscanner/service_test.go`
- Create `backend/internal/comiccache/service.go`: thumbnail lookup/generation/eviction.
- Create `backend/internal/comiccache/service_test.go`

### Backend HTTP

- Modify `backend/internal/server/server.go`: route registration and settings DTO/PATCH handling.
- Create `backend/internal/server/comic_paths_handlers.go`
- Create `backend/internal/server/comic_library_handlers.go`
- Create `backend/internal/server/comic_page_handlers.go`
- Create `backend/internal/server/comic_progress_handlers.go`
- Create `backend/internal/server/comic_import_handlers.go`
- Create `backend/internal/server/comic_cache_handlers.go`
- Add matching handler tests.

### Frontend Types And Services

- Create `src/domain/comic/types.ts`
- Modify `src/api/types.ts`: add comic DTOs and request/response shapes.
- Create `src/api/comic-endpoints.ts`: comic API calls used by the web comic service adapter.
- Create `src/services/contracts/comic-library-service.ts`
- Create `src/services/comic-library-service.ts`
- Create `src/services/adapters/web/web-comic-library-service.ts`
- Create `src/services/adapters/mock/mock-comic-library-service.ts`
- Add focused tests for service boundary and mappers.

### Frontend Routes And Shell

- Modify `src/domain/library/types.ts`: add comic app page ids only where route typing requires it.
- Modify `src/router/index.ts`: add `/comics`, `/comics/:id`, `/comics/:id/read`, and guard disabled comic library.
- Modify `src/components/jav-library/AppSidebar.vue`: conditional Browse entry.
- Modify `src/layouts/AppShell.vue`: comic search and import menu integration.
- Create `src/components/jav-library/ImportMenu.vue`
- Keep `MovieImportDialog.vue` movie-specific.
- Create `src/components/jav-library/ComicImportDialog.vue`

### Frontend Comic UI

- Create `src/views/ComicsView.vue`
- Create `src/views/ComicDetailView.vue`
- Create `src/views/ComicReaderView.vue`
- Create `src/components/jav-library/comics/ComicLibraryPage.vue`
- Create `src/components/jav-library/comics/VirtualComicGrid.vue`
- Create `src/components/jav-library/comics/ComicCard.vue`
- Create `src/components/jav-library/comics/ComicDetailPanel.vue`
- Create `src/components/jav-library/comics/ComicPagePreviewGrid.vue`
- Create `src/components/jav-library/comics/ComicReader.vue`
- Create `src/components/jav-library/comics/ComicReaderChrome.vue`
- Create `src/components/jav-library/comics/ComicReaderSettingsMenu.vue`

### Settings And Local State

- Modify `src/lib/settings-nav.ts`: add `comics` settings slug.
- Modify `src/components/jav-library/SettingsPage.vue`: render comic section and initialization flow.
- Create `src/components/jav-library/settings/SettingsComicLibrarySection.vue`
- Create `src/components/jav-library/settings/SettingsComicLibraryPathsSection.vue`
- Create `src/components/jav-library/settings/SettingsComicReaderSection.vue`
- Create `src/components/jav-library/settings/SettingsComicCacheSection.vue`
- Create `src/lib/comic-reader-route.ts`
- Create `src/lib/comic-search.ts`
- Create `src/lib/comic-reader-controls.ts`
- Create `src/lib/mock-comic-prefs-storage.ts`

### Localization And Docs

- Modify `src/locales/en.json`
- Modify `src/locales/zh-CN.json`
- Modify `src/locales/ja.json`
- After implementation lands, update:
  - `.cursor/rules/project-facts.mdc`
  - `README.md`
  - `API.md`
  - `CLAUDE.md`
  - `docs/reference/architecture-and-implementation.html`
  - `docs/reference/2026-03-21-library-organize.md` if `library-config.cfg` docs need the new comic keys.

## Task 0: Prepare Isolated Worktree

**Files:**
- No source files.

- [ ] **Step 1: Confirm current status in the main worktree**

Run:

```powershell
git status --short
```

Expected: working tree may contain unrelated existing changes. Do not revert them.

- [ ] **Step 2: Create a worktree for implementation**

Run from the repo root:

```powershell
git worktree add ..\jav-shadcn-comic-library -b feature/comic-library-mvp
```

Expected: a new clean worktree exists at `..\jav-shadcn-comic-library`.

- [ ] **Step 3: Switch to the implementation worktree**

Run:

```powershell
Set-Location ..\jav-shadcn-comic-library
git status --short
```

Expected: clean worktree or only worktree-local generated files.

## Task 1: Add Backend Comic Schema And Settings Defaults

**Files:**
- Create: `backend/internal/storage/migrations/0026_comic_library.sql`
- Modify: `backend/internal/storage/sqlite.go`
- Create: `backend/internal/contracts/comic_contracts.go`
- Create: `backend/internal/config/comic_settings.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/library_settings.go`
- Test: `backend/internal/config/comic_settings_test.go`
- Test: `backend/internal/storage/comic_migrations_test.go`

- [ ] **Step 1: Write migration tests**

Create `backend/internal/storage/comic_migrations_test.go` with tests that open a temp SQLite DB through the existing migration runner and assert these tables exist:

```go
func TestComicLibraryMigrationCreatesTables(t *testing.T) {
	store := openTestSQLiteStore(t)
	for _, table := range []string{
		"comic_library_paths",
		"comic_books",
		"comic_pages",
		"comic_tags",
		"comic_book_tags",
		"comic_reading_progress",
		"comic_reading_preferences",
		"comic_cache_entries",
	} {
		assertTableExists(t, store.DB(), table)
	}
}
```

Create local helpers in the test file that open an in-memory SQLite store, apply all migrations through `storage.Open`, and assert table existence through `sqlite_master`.

- [ ] **Step 2: Run migration test and verify failure**

Run:

```powershell
cd backend
go test ./internal/storage -run TestComicLibraryMigrationCreatesTables -count=1
```

Expected: FAIL because migration 0026 and/or tables do not exist.

- [ ] **Step 3: Add migration**

Create `backend/internal/storage/migrations/0026_comic_library.sql` with independent comic tables. Include indexes for:

- `comic_library_paths(path)`
- `comic_books(location)`
- `comic_books(added_at DESC, id ASC)`
- `comic_books(is_favorite)`
- `comic_books(read_status)`
- `comic_pages(comic_id, page_index)`
- `comic_book_tags(comic_id)`
- `comic_cache_entries(last_accessed_at)`

The migration must not alter movie tables.

- [ ] **Step 4: Wire migration**

Modify `backend/internal/storage/sqlite.go` so migration `0026_comic_library.sql` is included in the migration list.

- [ ] **Step 5: Add comic contract types**

Create `backend/internal/contracts/comic_contracts.go` containing:

- `ComicLibraryPathDTO`
- `ComicBookListItemDTO`
- `ComicBookDetailDTO`
- `ComicPageDTO`
- `ComicReaderSettingsDTO`
- `ComicCacheSettingsDTO`
- `ComicCacheStatusDTO`
- `PatchComicBookRequest`
- `PutComicProgressRequest`
- `PutComicReadingPreferencesRequest`
- task constants `TaskTypeScanComics`, `TaskTypeImportComics`, `TaskTypeComicCacheCleanup`
- error constants listed in the design doc.

- [ ] **Step 6: Add config defaults and parsing tests**

Create `backend/internal/config/comic_settings_test.go` covering:

- default comic library disabled.
- default reader mode `page`.
- default fit mode `contain`.
- default reading direction `ltr`.
- default cache max bytes 2GB.
- parsing `comicLibraryEnabled`, `defaultComicImportLibraryPathId`, `comicReader`, and `comicCache` from `library-config.cfg`.

- [ ] **Step 7: Implement config defaults and library settings merge**

Modify config files so `MergeLibrarySettingsFile` reads comic keys and `WriteLibrarySettingsMerge` callers can persist them. Unknown keys must remain preserved.

- [ ] **Step 8: Run backend tests for schema and config**

Run:

```powershell
cd backend
go test ./internal/storage ./internal/config -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit**

Run:

```powershell
git add backend/internal/storage/migrations/0026_comic_library.sql backend/internal/storage/sqlite.go backend/internal/contracts/comic_contracts.go backend/internal/config/comic_settings.go backend/internal/config/config.go backend/internal/config/library_settings.go backend/internal/config/comic_settings_test.go backend/internal/storage/comic_migrations_test.go
git commit -m "feat(backend): add comic schema and settings"
```

## Task 2: Add Comic Repositories

**Files:**
- Create: `backend/internal/storage/comic_paths_repository.go`
- Create: `backend/internal/storage/comic_books_repository.go`
- Create: `backend/internal/storage/comic_pages_repository.go`
- Create: `backend/internal/storage/comic_tags_repository.go`
- Create: `backend/internal/storage/comic_progress_repository.go`
- Create: `backend/internal/storage/comic_preferences_repository.go`
- Create: `backend/internal/storage/comic_cache_repository.go`
- Test: matching `*_test.go` files.

- [ ] **Step 1: Write repository tests**

Add tests covering:

- create/list/update/delete comic paths.
- upsert comic book by location.
- replace pages for one comic.
- list paginated comics by `q`, `tag`, favorite, read status.
- patch title/tags/favorite/rating.
- save/read/delete progress.
- save/read per-book reader preferences.
- create/list/touch/delete cache entries.

Use small table-driven tests with temp DBs.

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
cd backend
go test ./internal/storage -run "Comic" -count=1
```

Expected: FAIL because repository methods are not implemented.

- [ ] **Step 3: Implement repositories**

Implement focused repository methods. Keep path repositories separate from book/page/tag/progress/cache repositories. Do not touch movie repositories.

- [ ] **Step 4: Run repository tests**

Run:

```powershell
cd backend
go test ./internal/storage -run "Comic" -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```powershell
git add backend/internal/storage/comic_*_repository.go backend/internal/storage/comic_*_repository_test.go
git commit -m "feat(backend): add comic repositories"
```

## Task 3: Implement Comic Archive Reader

**Files:**
- Create: `backend/internal/comicarchive/service.go`
- Create: `backend/internal/comicarchive/service_test.go`

- [ ] **Step 1: Write archive tests**

Create tests that build temporary zip files and assert:

- `.zip` and `.cbz` are accepted.
- supported image extensions are included.
- unsupported files are ignored.
- nested zip/pdf/video files are ignored.
- directory hierarchy and natural file names sort correctly.
- empty archive returns `COMIC_ARCHIVE_EMPTY`.
- image bytes for a page can be read by page entry path.

Include an explicit case:

```go
entries := []string{
	"chapter2/001.jpg",
	"chapter1/010.jpg",
	"chapter1/002.jpg",
	"chapter1/001.jpg",
}
want := []string{
	"chapter1/001.jpg",
	"chapter1/002.jpg",
	"chapter1/010.jpg",
	"chapter2/001.jpg",
}
```

- [ ] **Step 2: Run archive tests and verify failure**

Run:

```powershell
cd backend
go test ./internal/comicarchive -count=1
```

Expected: FAIL because package does not exist.

- [ ] **Step 3: Implement archive service**

Implement:

- `IsSupportedArchivePath(path string) bool`
- `IsSupportedImageEntry(name string) bool`
- `ListPages(ctx context.Context, archivePath string) ([]PageEntry, error)`
- `OpenPage(ctx context.Context, archivePath string, entryPath string) (io.ReadCloser, PageEntry, error)`

Use `archive/zip`. Never extract zip entries to arbitrary disk locations.

- [ ] **Step 4: Run archive tests**

Run:

```powershell
cd backend
go test ./internal/comicarchive -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```powershell
git add backend/internal/comicarchive/service.go backend/internal/comicarchive/service_test.go
git commit -m "feat(backend): add comic archive reader"
```

## Task 4: Implement Comic Scanner

**Files:**
- Create: `backend/internal/comicscanner/service.go`
- Create: `backend/internal/comicscanner/service_test.go`
- Modify: `backend/internal/app/app.go` if application wiring owns scan starters.
- Modify: `backend/internal/server/server.go` only when route dependencies need scanner injection.

- [ ] **Step 1: Write scanner tests**

Tests should create temp comic library roots and zip files, then assert:

- scanner discovers `.zip` and `.cbz`.
- scanner ignores unsupported archive extensions.
- scanner creates or updates `comic_books`.
- scanner replaces page index when source file changes.
- title defaults to filename without extension.
- empty archive is skipped with task metadata error item.

- [ ] **Step 2: Run scanner tests and verify failure**

Run:

```powershell
cd backend
go test ./internal/comicscanner -count=1
```

Expected: FAIL because scanner package does not exist.

- [ ] **Step 3: Implement scanner service**

Implement scanner service that accepts configured comic roots and storage repositories. It should return a summary with discovered/imported/updated/skipped counts and per-file errors.

- [ ] **Step 4: Run scanner tests**

Run:

```powershell
cd backend
go test ./internal/comicscanner ./internal/storage -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```powershell
git add backend/internal/comicscanner backend/internal/app/app.go backend/internal/server/server.go
git commit -m "feat(backend): add comic scanner"
```

## Task 5: Add Comic Settings, Path, And Scan HTTP APIs

**Files:**
- Modify: `backend/internal/server/server.go`
- Create: `backend/internal/server/comic_paths_handlers.go`
- Create: `backend/internal/server/comic_scan_handlers.go`
- Test: `backend/internal/server/comic_paths_handlers_test.go`
- Test: `backend/internal/server/comic_scan_handlers_test.go`
- Modify: `src/api/types.ts`
- Create: `src/api/comic-endpoints.ts`

- [ ] **Step 1: Write backend handler tests**

Cover:

- `GET /api/settings` returns comic fields.
- `PATCH /api/settings` validates and persists comic enabled state, reader defaults, cache settings, and default import path.
- `POST /api/library/comics/paths` adds a comic path.
- enabling without a valid path is rejected.
- `POST /api/library/comics/scans` starts `scan.comics`.
- disabled comic library returns `COMIC_LIBRARY_DISABLED` for scan.

- [ ] **Step 2: Run handler tests and verify failure**

Run:

```powershell
cd backend
go test ./internal/server -run "Comic|Settings" -count=1
```

Expected: FAIL because routes are missing.

- [ ] **Step 3: Implement handlers and routes**

Add route registration in `server.go` and focused handlers in new files. Reuse existing JSON/error helpers. Keep comic path storage-status route names separate from movie path routes.

- [ ] **Step 4: Update frontend API types and comic endpoints**

Add DTOs to `src/api/types.ts`:

- `ComicLibraryPathDTO`
- `ComicReaderSettingsDTO`
- `ComicCacheSettingsDTO`
- `ComicBookListItemDTO`
- `ComicBookDetailDTO`
- `ComicPageDTO`
- `ComicReadingProgressDTO`
- `ComicReadingPreferencesDTO`

Create `src/api/comic-endpoints.ts` with typed wrappers for comic settings/path/scan routes introduced in this task. Later tasks extend this file with list/detail/page/import/cache calls.

- [ ] **Step 5: Run tests**

Run:

```powershell
cd backend
go test ./internal/server -run "Comic|Settings" -count=1
cd ..
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```powershell
git add backend/internal/server/server.go backend/internal/server/comic_paths_handlers.go backend/internal/server/comic_scan_handlers.go backend/internal/server/comic_paths_handlers_test.go backend/internal/server/comic_scan_handlers_test.go src/api/types.ts src/api/comic-endpoints.ts
git commit -m "feat: expose comic settings and scan APIs"
```

## Task 6: Add Comic Library, Page, Progress, Preference, And Cache APIs

**Files:**
- Create: `backend/internal/server/comic_library_handlers.go`
- Create: `backend/internal/server/comic_page_handlers.go`
- Create: `backend/internal/server/comic_progress_handlers.go`
- Create: `backend/internal/server/comic_cache_handlers.go`
- Modify: `backend/internal/server/server.go`
- Create: `backend/internal/comiccache/service.go`
- Create: `backend/internal/comiccache/service_test.go`
- Test: matching handler tests.

- [ ] **Step 1: Write handler and cache tests**

Cover:

- list comics with filters.
- get comic detail with first preview URLs.
- patch title/tags/favorite/rating.
- remove index without deleting source file.
- reveal source file validates file exists.
- get pages list.
- get page image from zip.
- get thumbnail creates cache entry.
- save/reset progress.
- save/get reading preferences.
- cache status reports size/max bytes.
- cleanup deletes only cache files and rows.

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
cd backend
go test ./internal/server ./internal/comiccache -run "Comic" -count=1
```

Expected: FAIL because handlers/cache service are missing.

- [ ] **Step 3: Implement comic cache service**

Implement thumbnail generation, cache lookup, touch, status, and eviction. Use a comic-specific cache directory. The cleanup function must accept only paths registered in `comic_cache_entries`.

- [ ] **Step 4: Implement library/page/progress handlers**

Register routes under `/api/library/comics`. Keep image responses same-origin and avoid exposing zip entry paths directly.

- [ ] **Step 5: Run backend tests**

Run:

```powershell
cd backend
go test ./internal/server ./internal/comiccache ./internal/storage -run "Comic" -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```powershell
git add backend/internal/server/comic_library_handlers.go backend/internal/server/comic_page_handlers.go backend/internal/server/comic_progress_handlers.go backend/internal/server/comic_cache_handlers.go backend/internal/server/server.go backend/internal/comiccache backend/internal/storage/comic_cache_repository.go
git commit -m "feat(backend): add comic library APIs"
```

## Task 7: Add Frontend Comic Service Boundary

**Files:**
- Create: `src/domain/comic/types.ts`
- Create: `src/services/contracts/comic-library-service.ts`
- Create: `src/services/comic-library-service.ts`
- Create: `src/services/adapters/web/web-comic-library-service.ts`
- Create: `src/services/adapters/mock/mock-comic-library-service.ts`
- Create: `src/lib/mock-comic-prefs-storage.ts`
- Test: `src/services/comic-library-service-boundary.test.ts`
- Test: `src/services/adapters/web/web-comic-library-service.test.ts`
- Test: `src/services/adapters/mock/mock-comic-library-service.test.ts`

- [ ] **Step 1: Write frontend service tests**

Cover:

- `useComicLibraryService()` returns web adapter when `VITE_USE_WEB_API=true`.
- mock adapter returns sample comics.
- mock stores favorite/rating/progress/preferences in localStorage.
- web adapter calls comic endpoints, not movie endpoints.

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
pnpm test -- src/services/comic-library-service-boundary.test.ts src/services/adapters/web/web-comic-library-service.test.ts src/services/adapters/mock/mock-comic-library-service.test.ts
```

Expected: FAIL because files do not exist.

- [ ] **Step 3: Implement domain types and service contract**

Define comic-only types and service methods. Do not import `Movie` or `LibraryService`.

- [ ] **Step 4: Implement web and mock adapters**

The web adapter should call comic API routes. The mock adapter should provide UI demo data only and reject real import/scan with clear mock errors.

- [ ] **Step 5: Run frontend service tests**

Run:

```powershell
pnpm test -- src/services/comic-library-service-boundary.test.ts src/services/adapters/web/web-comic-library-service.test.ts src/services/adapters/mock/mock-comic-library-service.test.ts
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```powershell
git add src/domain/comic src/services/contracts/comic-library-service.ts src/services/comic-library-service.ts src/services/adapters/web/web-comic-library-service.ts src/services/adapters/mock/mock-comic-library-service.ts src/lib/mock-comic-prefs-storage.ts src/services/*comic*.test.ts src/services/adapters/**/*comic*.test.ts
git commit -m "feat(frontend): add comic service boundary"
```

## Task 8: Add Settings Comic Section And Enable Flow

**Files:**
- Modify: `src/lib/settings-nav.ts`
- Modify: `src/components/jav-library/SettingsPage.vue`
- Create: `src/components/jav-library/settings/SettingsComicLibrarySection.vue`
- Create: `src/components/jav-library/settings/SettingsComicLibraryPathsSection.vue`
- Create: `src/components/jav-library/settings/SettingsComicReaderSection.vue`
- Create: `src/components/jav-library/settings/SettingsComicCacheSection.vue`
- Test: matching component tests.
- Modify: `src/locales/en.json`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/ja.json`

- [ ] **Step 1: Write settings tests**

Cover:

- Comics nav item appears in settings nav.
- disabled state shows enable action.
- enable requires at least one valid comic path.
- enabled state shows path/default import/reader/cache sections.
- disabling hides app entry but does not request data deletion.
- cache size presets include 1GB/2GB/5GB/10GB/unlimited.

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
pnpm test -- src/components/jav-library/SettingsPage.test.ts src/components/jav-library/settings/SettingsComicLibrarySection.test.ts
```

Expected: FAIL because comic components/nav do not exist.

- [ ] **Step 3: Implement settings components**

Follow Settings card/nested block UI rules from `.cursor/rules/ui-component-spec.mdc`. Use `useComicLibraryService()`.

- [ ] **Step 4: Add localized copy**

Add concise English, Chinese, and Japanese labels for comic settings, errors, and cache controls.

- [ ] **Step 5: Run tests and typecheck**

Run:

```powershell
pnpm test -- src/components/jav-library/SettingsPage.test.ts src/components/jav-library/settings/SettingsComicLibrarySection.test.ts
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```powershell
git add src/lib/settings-nav.ts src/components/jav-library/SettingsPage.vue src/components/jav-library/settings/SettingsComic*.vue src/components/jav-library/settings/SettingsComic*.test.ts src/locales/en.json src/locales/zh-CN.json src/locales/ja.json
git commit -m "feat(frontend): add comic settings"
```

## Task 9: Add Routes, Sidebar Entry, And Import Menu Shell

**Files:**
- Modify: `src/router/index.ts`
- Modify: `src/domain/library/types.ts`
- Modify: `src/components/jav-library/AppSidebar.vue`
- Modify: `src/layouts/AppShell.vue`
- Create: `src/components/jav-library/ImportMenu.vue`
- Create: `src/components/jav-library/ComicImportDialog.vue`
- Test: `src/router/comic-library-route.test.ts`
- Test: `src/components/jav-library/AppSidebar.test.ts`
- Test: `src/components/jav-library/ImportMenu.test.ts`

- [ ] **Step 1: Write routing and shell tests**

Cover:

- `/comics` is guarded when comic library disabled.
- sidebar shows comic entry only when enabled.
- import menu contains movie only when comics disabled.
- import menu contains movie and comic when comics enabled.
- comic import dialog blocks submit without default comic import path.

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
pnpm test -- src/router/comic-library-route.test.ts src/components/jav-library/ImportMenu.test.ts
```

Expected: FAIL because route/menu are missing.

- [ ] **Step 3: Implement routes and guard**

Add `comics`, `comic-detail`, and `comic-reader` routes. Guard disabled comics by redirecting to `settings?section=comics`.

- [ ] **Step 4: Implement sidebar and import menu**

Move current top-bar import button behavior into `ImportMenu`. Keep `MovieImportDialog` movie-specific and add a separate `ComicImportDialog`.

- [ ] **Step 5: Run tests and typecheck**

Run:

```powershell
pnpm test -- src/router/comic-library-route.test.ts src/components/jav-library/ImportMenu.test.ts
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```powershell
git add src/router/index.ts src/domain/library/types.ts src/components/jav-library/AppSidebar.vue src/layouts/AppShell.vue src/components/jav-library/ImportMenu.vue src/components/jav-library/ComicImportDialog.vue src/router/comic-library-route.test.ts src/components/jav-library/ImportMenu.test.ts
git commit -m "feat(frontend): add comic navigation shell"
```

## Task 10: Add Comic Library Grid And Detail Page

**Files:**
- Create: `src/views/ComicsView.vue`
- Create: `src/views/ComicDetailView.vue`
- Create: `src/components/jav-library/comics/ComicLibraryPage.vue`
- Create: `src/components/jav-library/comics/VirtualComicGrid.vue`
- Create: `src/components/jav-library/comics/ComicCard.vue`
- Create: `src/components/jav-library/comics/ComicDetailPanel.vue`
- Create: `src/components/jav-library/comics/ComicPagePreviewGrid.vue`
- Create: `src/lib/comic-search.ts`
- Test: matching component/lib tests.

- [ ] **Step 1: Write comic library UI tests**

Cover:

- grid renders sample comics.
- filters all/favorite/unread/read.
- search matches title, tags, filename, path.
- card shows title, page count, rating, favorite, read progress.
- detail page edits title/tags/rating/favorite.
- tag input suggests `作者:`, `系列:`, `卷:`, `社团:`.
- preview grid defaults to first 12 pages and opens reader at selected page.

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
pnpm test -- src/components/jav-library/comics/ComicCard.test.ts src/lib/comic-search.test.ts
```

Expected: FAIL because components/libs do not exist.

- [ ] **Step 3: Implement comic grid and search**

Use virtualized rendering for large lists. Do not import `MovieCard`.

- [ ] **Step 4: Implement detail page and preview grid**

Use comic service methods. Preview thumbnails should call comic thumbnail URLs, not movie asset URLs.

- [ ] **Step 5: Run tests and typecheck**

Run:

```powershell
pnpm test -- src/components/jav-library/comics src/lib/comic-search.test.ts
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```powershell
git add src/views/ComicsView.vue src/views/ComicDetailView.vue src/components/jav-library/comics src/lib/comic-search.ts src/lib/comic-search.test.ts
git commit -m "feat(frontend): add comic library and detail pages"
```

## Task 11: Add Comic Reader

**Files:**
- Create: `src/views/ComicReaderView.vue`
- Create: `src/components/jav-library/comics/ComicReader.vue`
- Create: `src/components/jav-library/comics/ComicReaderChrome.vue`
- Create: `src/components/jav-library/comics/ComicReaderSettingsMenu.vue`
- Create: `src/lib/comic-reader-route.ts`
- Create: `src/lib/comic-reader-controls.ts`
- Test: matching tests.

- [ ] **Step 1: Write reader behavior tests**

Cover:

- default preference fallback uses global defaults.
- per-book preference overrides global defaults.
- LTR right arrow advances and left arrow goes back.
- RTL right arrow goes back and left arrow advances.
- `Space` advances according to reading direction.
- temporary stitch current+next displays lower page on left in LTR and higher page on left in RTL.
- temporary stitch is cleared when component unmounts.
- progress save is throttled and sends current page.
- completed is set at final page.

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
pnpm test -- src/components/jav-library/comics/ComicReader.test.ts src/lib/comic-reader-controls.test.ts
```

Expected: FAIL because reader files do not exist.

- [ ] **Step 3: Implement reader state helpers**

Implement pure helpers in `comic-reader-controls.ts` for:

- next/previous action by direction.
- stitch pair display order by direction.
- visible page progression when a stitched pair is active.

- [ ] **Step 4: Implement reader components**

Implement immersive reader UI with hover/activity chrome, page mode, scroll mode, fit contain/width, LTR/RTL, keyboard controls, and temporary stitch controls.

- [ ] **Step 5: Run reader tests and typecheck**

Run:

```powershell
pnpm test -- src/components/jav-library/comics/ComicReader.test.ts src/lib/comic-reader-controls.test.ts
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```powershell
git add src/views/ComicReaderView.vue src/components/jav-library/comics/ComicReader*.vue src/lib/comic-reader-route.ts src/lib/comic-reader-controls.ts src/components/jav-library/comics/ComicReader*.test.ts src/lib/comic-reader-controls.test.ts
git commit -m "feat(frontend): add comic reader"
```

## Task 12: Implement Comic Import End To End

**Files:**
- Create: `backend/internal/server/comic_import_handlers.go`
- Create: `backend/internal/server/comic_import_upload_handlers.go`
- Test: `backend/internal/server/comic_import_handlers_test.go`
- Modify: `src/api/comic-endpoints.ts`
- Modify: `src/services/adapters/web/web-comic-library-service.ts`
- Modify: `src/components/jav-library/ComicImportDialog.vue`
- Test: `src/components/jav-library/ComicImportDialog.test.ts`

- [ ] **Step 1: Write backend import tests**

Cover:

- import rejects when default comic import path missing.
- import accepts `.zip` and `.cbz`.
- import rejects unsupported file types.
- target path conflict skips file and does not overwrite.
- successful import starts `scan.comics`.
- scan error is recorded in task metadata without deleting copied file.

- [ ] **Step 2: Write frontend import tests**

Cover:

- dialog accepts zip/cbz files.
- unsupported files are skipped.
- submit disabled without default path.
- storage unavailable blocks submit.
- upload progress is shown.
- started import task is tracked.

- [ ] **Step 3: Run tests and verify failure**

Run:

```powershell
cd backend
go test ./internal/server -run "ComicImport" -count=1
cd ..
pnpm test -- src/components/jav-library/ComicImportDialog.test.ts
```

Expected: FAIL because import implementation is incomplete.

- [ ] **Step 4: Implement backend comic import**

Mirror movie import infrastructure where useful, but use comic task types, comic default path, and comic supported file validation.

- [ ] **Step 5: Implement frontend comic import**

Wire `ComicImportDialog` to `useComicLibraryService().importComics()` and `useScanTaskTracker()` for `import.comics`.

- [ ] **Step 6: Run import tests**

Run:

```powershell
cd backend
go test ./internal/server -run "ComicImport" -count=1
cd ..
pnpm test -- src/components/jav-library/ComicImportDialog.test.ts
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```powershell
git add backend/internal/server/comic_import*.go backend/internal/server/comic_import*_test.go src/api/comic-endpoints.ts src/services/adapters/web/web-comic-library-service.ts src/components/jav-library/ComicImportDialog.vue src/components/jav-library/ComicImportDialog.test.ts
git commit -m "feat: add comic import flow"
```

## Task 13: Wire Task Tracking, Toasts, And Recent Task Copy

**Files:**
- Modify: `src/composables/use-scan-task-tracker.ts`
- Modify: `src/components/jav-library/ScanProgressDock.vue`
- Modify: `src/composables/use-library-watch-toasts.ts` only if it owns terminal task copy.
- Modify: `src/locales/en.json`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/ja.json`
- Test: existing task tracker/dock tests plus comic cases.

- [ ] **Step 1: Write task UI tests**

Cover:

- `scan.comics` terminal success refreshes comics, not movies.
- `import.comics` progress metadata renders comic copy.
- partial failure toast uses comic-specific wording.
- movie import/scan behavior remains unchanged.

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
pnpm test -- src/composables/use-scan-task-tracker.test.ts src/components/jav-library/ScanProgressDock.test.ts
```

Expected: FAIL for comic task handling.

- [ ] **Step 3: Implement comic task handling**

Extend task tracker to route comic tasks through `useComicLibraryService()` refresh methods. Keep movie refresh behavior untouched.

- [ ] **Step 4: Run tests**

Run:

```powershell
pnpm test -- src/composables/use-scan-task-tracker.test.ts src/components/jav-library/ScanProgressDock.test.ts
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```powershell
git add src/composables/use-scan-task-tracker.ts src/composables/use-scan-task-tracker.test.ts src/components/jav-library/ScanProgressDock.vue src/components/jav-library/ScanProgressDock.test.ts src/locales/en.json src/locales/zh-CN.json src/locales/ja.json
git commit -m "feat(frontend): track comic tasks"
```

## Task 14: Browser Verification With Real Comic Zip

**Files:**
- Create local untracked fixture: `.workspace/comic-fixtures/sample.cbz`

- [ ] **Step 1: Create a small local comic fixture outside tracked source**

Create a zip in `.workspace/comic-fixtures/sample.cbz` with five tiny images:

```text
chapter1/001.jpg
chapter1/002.jpg
chapter1/010.jpg
chapter2/001.jpg
chapter2/002.jpg
```

Do not commit `.workspace/`.

- [ ] **Step 2: Run backend and frontend**

Terminal 1:

```powershell
cd backend
go run ./cmd/curated
```

Terminal 2:

```powershell
$env:VITE_USE_WEB_API='true'
pnpm dev
```

Expected: backend on `http://127.0.0.1:8080`, frontend on Vite port.

- [ ] **Step 3: Verify enabled flow in browser**

Use the browser to:

- open Settings -> 漫画库.
- enable comic library with a temp comic root.
- set default comic import path.
- import the sample cbz.
- confirm scan task completes.

- [ ] **Step 4: Verify library and reader**

Use the browser to confirm:

- comic appears in comic grid.
- detail page shows first 12 preview area or fewer if page count is smaller.
- clicking page 2 opens reader at page 2.
- LTR arrow behavior works.
- RTL arrow behavior flips navigation.
- temporary stitch current+next changes display and clears after leaving reader.
- cache cleanup does not delete source cbz.

- [ ] **Step 5: Capture issues as follow-up tasks**

If browser verification finds defects, create focused fix commits before moving to docs sync.

## Task 15: Full Test Pass

**Files:**
- No code files unless tests reveal fixes.

- [ ] **Step 1: Run focused backend tests**

Run:

```powershell
cd backend
go test ./internal/storage ./internal/comicarchive ./internal/comicscanner ./internal/comiccache ./internal/server -run "Comic" -count=1
```

Expected: PASS.

- [ ] **Step 2: Run backend full tests**

Run:

```powershell
cd backend
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend focused tests**

Run:

```powershell
pnpm test -- src/services/comic-library-service-boundary.test.ts src/components/jav-library/comics src/components/jav-library/ComicImportDialog.test.ts
```

Expected: PASS.

- [ ] **Step 4: Run frontend baseline checks**

Run:

```powershell
pnpm typecheck
pnpm lint
pnpm test
```

Expected: PASS.

- [ ] **Step 5: Commit any verification fixes**

If a fix was required:

```powershell
git add <fixed-files>
git commit -m "fix: stabilize comic library MVP"
```

If no fix was required, do not create an empty commit.

## Task 16: Documentation Sync

**Files:**
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `README.md`
- Modify: `API.md`
- Modify: `CLAUDE.md`
- Modify: `docs/reference/architecture-and-implementation.html`
- Modify: `docs/reference/2026-03-21-library-organize.md`
- Review only: `.cursor/rules/workspace-quick-reference.mdc` stays unchanged because startup commands, ports, and proxy configuration do not change.

- [ ] **Step 1: Update project facts**

Document:

- comic library optional module.
- independent comic tables/API/service.
- new settings keys.
- route names.
- import and scan task types.
- cache behavior.

- [ ] **Step 2: Update public API docs**

Add comic endpoints, DTO summaries, task types, and error codes to `API.md` and `CLAUDE.md`.

- [ ] **Step 3: Update README**

Add a concise feature summary and note that comics are optional and support `.zip/.cbz`.

- [ ] **Step 4: Update architecture reference**

Add a comic feature row and implementation notes to `docs/reference/architecture-and-implementation.html`.

- [ ] **Step 5: Update library-config documentation**

Add `comicLibraryEnabled`, `defaultComicImportLibraryPathId`, `comicReader`, and `comicCache` to `docs/reference/2026-03-21-library-organize.md`.

- [ ] **Step 6: Run docs-adjacent checks**

Run:

```powershell
pnpm typecheck
```

Expected: PASS.

- [ ] **Step 7: Commit docs**

Run:

```powershell
git add .cursor/rules/project-facts.mdc README.md API.md CLAUDE.md docs/reference/architecture-and-implementation.html docs/reference/2026-03-21-library-organize.md .cursor/rules/workspace-quick-reference.mdc
git commit -m "docs: document comic library MVP"
```

## Task 17: Final Review Checklist

**Files:**
- No code files unless review finds fixes.

- [ ] **Step 1: Check independent-domain boundary**

Run:

```powershell
rg -n "Comic|comic" src backend/internal | rg "Movie|movie|LibraryService|movies"
```

Expected: only intentional references such as import menu co-location, docs, and task tracker branching. No comic data should be modeled as `Movie`.

- [ ] **Step 2: Check no source deletion behavior**

Run:

```powershell
rg -n "Remove\\(|Delete|delete|os\\.Remove|RemoveAll" backend/internal | rg "comic"
```

Expected: deletion is limited to comic indexes/cache files. No handler should delete source zip/cbz.

- [ ] **Step 3: Check API docs match route registration**

Run:

```powershell
rg -n "/api/library/comics|/api/import/comics" backend/internal/server API.md CLAUDE.md
```

Expected: registered routes are documented.

- [ ] **Step 4: Final test command set**

Run:

```powershell
pnpm typecheck
pnpm lint
pnpm test
cd backend
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Prepare handoff summary**

Summarize:

- implemented phases.
- tests run.
- any known limitations retained from the MVP scope.
- whether display scaling testing was skipped due to requiring user approval.

## Self-Review Notes

- Spec coverage: This plan covers independent comic settings, schema, repositories, scanner, archive reader, APIs, cache, frontend service boundary, settings UI, navigation, import, library grid, detail, reader, task tracking, tests, browser verification, and docs sync.
- Known intentional exclusions from MVP: metadata scraping, OCR, rar/cbr/7z, source file deletion, auto watch, persistent double-page layout, gamepad support, cross-device sync.
- Type consistency: Use `Comic*` names for comic domain types and `comic*` API/settings keys. Do not introduce generic `MediaItem` or extend `Movie`.

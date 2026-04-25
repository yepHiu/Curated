# Curated Frame Export Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist curated-frame export format in backend settings, add JPG export support with JPG as the default, and collapse detail/context/batch export UI to one button per entry point while preserving batch export.

**Architecture:** The backend remains the source of truth for export format through `GET/PATCH /api/settings`, and the frontend reads that setting through the existing library-service settings pipeline. Export UI no longer chooses a format at the action site; it resolves the persisted format once and continues to call the existing `/api/curated-frames/export` endpoint. JPG support is added as a new curated-export encoder path with EXIF metadata embedding, aligned with current WebP metadata behavior.

**Tech Stack:** Go HTTP backend, SQLite-backed settings merge via `library-config.cfg`, Vue 3 + TypeScript frontend, Vitest, Go `testing`, existing curated export helpers.

---

## File Map

- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/library_settings.go`
- Modify: `backend/internal/config/library_settings_test.go`
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/server/server_test.go`
- Modify: `backend/internal/server/curated_export_handler.go`
- Add: `backend/internal/curatedexport/jpeg.go`
- Add: `backend/internal/curatedexport/jpeg_test.go`
- Modify: `backend/internal/curatedexport/filename.go`
- Modify: `src/api/types.ts`
- Modify: `src/services/contracts/library-service.ts`
- Modify: `src/services/adapters/web/web-library-service.ts`
- Modify: `src/services/adapters/mock/mock-library-service.ts`
- Modify: `src/components/jav-library/SettingsPage.vue`
- Modify: `src/components/jav-library/CuratedFrameContextMenu.vue`
- Modify: `src/components/jav-library/CuratedFrameContextMenu.test.ts`
- Modify: `src/components/jav-library/CuratedFramesBatchActionBar.vue`
- Modify: `src/components/jav-library/CuratedFramesLibrary.vue`
- Modify: `src/views/CuratedFramesView.vue`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`
- Modify: `src/locales/zh-CN.json`
- Modify: `API.md`
- Modify: `CLAUDE.md`
- Modify: `.cursor/rules/project-facts.mdc`

### Task 1: Backend Settings Contract For Export Format

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/library_settings.go`
- Modify: `backend/internal/config/library_settings_test.go`
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/server/server_test.go`

- [ ] **Step 1: Write the failing Go tests for default and persisted export format**

Add focused tests covering:
- `config.Default()` returns `CuratedFrameExportFormat == "jpg"`
- `MergeLibrarySettingsFile` loads `"curatedFrameExportFormat": "png"`
- invalid config value is ignored or normalized to default `"jpg"` according to chosen implementation
- `GET /api/settings` returns the field
- `PATCH /api/settings` accepts `jpg`, `webp`, `png` and rejects unsupported values

Target commands:

```bash
cd backend
go test ./internal/config -run CuratedFrameExportFormat -count=1
go test ./internal/server -run CuratedFrameExportFormat -count=1
```

- [ ] **Step 2: Run the tests to verify RED**

Expected:
- config/server tests fail because the field does not exist yet
- failures mention missing struct fields or missing JSON handling

- [ ] **Step 3: Implement the minimal backend settings support**

Add a new setting field named `CuratedFrameExportFormat` across:
- `config.Config`
- `contracts.SettingsDTO`
- `contracts.PatchSettingsRequest`
- `App` getter/setter methods persisted via `WriteLibrarySettingsMerge`
- `server.buildSettingsDTO`
- `server.handlePatchSettings`

Behavior:
- allowed values: `jpg | webp | png`
- default: `jpg`
- empty or invalid runtime fallback: `jpg`

- [ ] **Step 4: Run the focused tests to verify GREEN**

Run:

```bash
cd backend
go test ./internal/config -run CuratedFrameExportFormat -count=1
go test ./internal/server -run CuratedFrameExportFormat -count=1
```

Expected:
- targeted tests pass cleanly

### Task 2: Backend JPG Export Support

**Files:**
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/server/curated_export_handler.go`
- Modify: `backend/internal/curatedexport/filename.go`
- Add: `backend/internal/curatedexport/jpeg.go`
- Add: `backend/internal/curatedexport/jpeg_test.go`
- Modify: `backend/internal/server/server_test.go` or add focused curated export handler tests there if that is where export endpoint tests already live

- [ ] **Step 1: Write the failing Go tests for JPG export behavior**

Cover:
- JPEG helper writes decodable JPEG bytes and includes EXIF header bytes
- export endpoint accepts `format: "jpg"`
- single export response returns `Content-Type: image/jpeg`
- single export filename ends with `.jpg`
- empty format defaults to JPG

Target commands:

```bash
cd backend
go test ./internal/curatedexport -run JPEG -count=1
go test ./internal/server -run CuratedFramesExport.*JPG -count=1
```

- [ ] **Step 2: Run the tests to verify RED**

Expected:
- compile or assertion failures because JPEG encoder path and JPG handler support are missing

- [ ] **Step 3: Implement the minimal JPG export path**

Implementation outline:
- add `EncodeImageToJPEGWithCuratedMeta` or equivalent helper in `backend/internal/curatedexport/jpeg.go`
- decode source image using `image.Decode`
- encode JPEG with `go:jpeg`
- inject EXIF APP1 segment built from `BuildExifUserComment`
- add `ExportJPGFilename(...).jpg`
- update export handler to:
  - accept `jpg` and optionally `jpeg`
  - default to `jpg`
  - return `image/jpeg`
  - choose `.jpg` fallback filename
- update export contract comments/types from `webp | png` to `jpg | webp | png`

- [ ] **Step 4: Run the focused tests to verify GREEN**

Run:

```bash
cd backend
go test ./internal/curatedexport -run JPEG -count=1
go test ./internal/server -run CuratedFramesExport.*JPG -count=1
```

Expected:
- targeted JPG tests pass

### Task 3: Frontend Settings Wiring For Export Format

**Files:**
- Modify: `src/api/types.ts`
- Modify: `src/services/contracts/library-service.ts`
- Modify: `src/services/adapters/web/web-library-service.ts`
- Modify: `src/services/adapters/mock/mock-library-service.ts`
- Modify: `src/components/jav-library/SettingsPage.vue`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`
- Modify: `src/locales/zh-CN.json`

- [ ] **Step 1: Write the failing frontend test for settings state if a narrow test exists nearby**

If there is an existing mock-service/settings test surface, add focused assertions for:
- default mock export format is `jpg`
- changing the setting updates the service state

If no narrow settings test exists, skip creating a new broad component test and rely on targeted component export-action tests plus type-safe wiring.

Suggested command:

```bash
pnpm test -- src/services/adapters/mock/mock-library-service.test.ts
```

- [ ] **Step 2: Run the test to verify RED**

Expected:
- missing field or missing setter/getter failures

- [ ] **Step 3: Implement the minimal settings wiring**

Add the export format field through:
- `SettingsDTO`
- `PatchSettingsBody`
- web adapter load/apply flow
- mock adapter in-memory state
- Settings page curated section control

UI behavior:
- dropdown select aligned with the existing logging-level control pattern, with `JPG`, `WebP`, `PNG`
- initial value reflects settings state
- saving uses existing `patchSettings` pipeline
- settings copy should be upgraded from a minimal `Export` label to a fuller section title plus helper text that explains the scope of the setting

- [ ] **Step 4: Run the focused frontend test to verify GREEN**

Run the same command as step 1, or typecheck if no narrow test was added:

```bash
pnpm test -- src/services/adapters/mock/mock-library-service.test.ts
pnpm typecheck
```

### Task 4: Collapse Export Entry Points To One Action

**Files:**
- Modify: `src/components/jav-library/CuratedFrameContextMenu.vue`
- Modify: `src/components/jav-library/CuratedFrameContextMenu.test.ts`
- Modify: `src/components/jav-library/CuratedFramesBatchActionBar.vue`
- Modify: `src/components/jav-library/CuratedFramesLibrary.vue`
- Modify: `src/views/CuratedFramesView.vue`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`
- Modify: `src/locales/zh-CN.json`

- [ ] **Step 1: Write the failing frontend tests for single export actions**

Cover:
- context menu shows only one export action and emits `export`
- batch toolbar shows only one export action
- detail dialog shows only one export button
- library export logic reads persisted settings-derived format instead of accepting per-button format

Suggested commands:

```bash
pnpm test -- src/components/jav-library/CuratedFrameContextMenu.test.ts
```

If there is no existing `CuratedFramesBatchActionBar` or `CuratedFramesLibrary` test file, add a focused test file only if needed; otherwise keep verification through existing component tests plus a final typecheck.

- [ ] **Step 2: Run the tests to verify RED**

Expected:
- current tests fail because the UI still renders WebP/PNG-specific actions

- [ ] **Step 3: Implement the minimal UI consolidation**

Refactor:
- context menu emits `export`
- batch bar emits `export`
- `CuratedFramesLibrary` exposes `exportSelected`
- single-frame dialog uses one export button
- export format resolves from settings state with fallback `jpg`

Do not remove batch export capability; only remove duplicated format-specific entry points.

- [ ] **Step 4: Run focused frontend verification**

Run:

```bash
pnpm test -- src/components/jav-library/CuratedFrameContextMenu.test.ts
pnpm typecheck
```

If component coverage expands, include those test files explicitly.

### Task 5: Documentation And Final Verification

**Files:**
- Modify: `API.md`
- Modify: `CLAUDE.md`
- Modify: `.cursor/rules/project-facts.mdc`

- [ ] **Step 1: Update docs to reflect the new settings field and JPG default**

Document:
- settings payload includes `curatedFrameExportFormat`
- export endpoint supports `jpg | webp | png`
- default export format is `jpg`
- export UI is single-action in detail/context/batch and format is controlled from Settings

- [ ] **Step 2: Run focused verification**

Run:

```bash
pnpm test -- src/components/jav-library/CuratedFrameContextMenu.test.ts
pnpm typecheck
cd backend && go test ./internal/config ./internal/curatedexport ./internal/server
```

- [ ] **Step 3: Perform a final regression pass on touched flows**

Check:
- `GET /api/settings` payload shape remains valid
- `PATCH /api/settings` still rolls back on later failure
- curated export still works for PNG/WebP
- batch export still returns ZIP for multi-select

## Self-Review

- Spec coverage:
  - backend-persisted export format: Task 1
  - JPG support and JPG default: Tasks 1 and 2
  - single export action in detail/context/batch: Task 4
  - settings control: Task 3
  - docs sync: Task 5
- Placeholder scan: no TBD/TODO placeholders remain
- Type consistency:
  - canonical payload value is `jpg`
  - allowed setting values remain `jpg | webp | png`

# Curated Frame Export Entry And Format Design

Date: 2026-04-26
Status: Proposed (backend persistence + JPG support confirmed)

## Goal

Simplify curated-frame export UX so users do not choose image format at each export entry point.

Confirmed requirements:

- Keep curated-frame export available in three places only:
  - single-frame detail dialog
  - single-frame right-click context menu
  - batch action bar
- Each place should expose only one export action instead of separate format-specific actions.
- The actual export format should follow a single curated-frame setting chosen in Settings.
- Batch export remains supported.
- Add JPG export support.
- Default export format should be JPG.
- The existing backend export API shape can remain similar for this phase; the frontend still passes `format` to `/api/curated-frames/export`, but the allowed values and backend encoder paths must expand.

## Existing Context

- `CuratedFramesLibrary.vue` currently exposes separate `exportSelectedWebp` / `exportSelectedPng` methods for batch export.
- The single-frame detail dialog currently renders two buttons: `exportSingleFromDialog('webp')` and `exportSingleFromDialog('png')`.
- `CuratedFrameContextMenu.vue` currently renders two actions and emits `exportWebp` / `exportPng`.
- Curated-frame preferences in `src/lib/curated-frames/settings-storage.ts` are browser-local and already store:
  - save mode
  - capture shortcut
- There is not yet a stored curated-frame export-format preference.
- Backend export currently only accepts:
  - `webp`
  - `png`
- Backend `curatedexport` currently has WebP and PNG metadata embedding paths, but not JPEG/JPG export yet.

## Approaches

### Approach A: Frontend-only export-format preference in local storage

Add a new curated-frame export-format preference in `settings-storage.ts` and reuse it across the settings page, detail dialog, context menu, and batch toolbar.

Pros:

- Smallest change surface.
- Fits existing curated-frame preference storage model.
- No backend contract or migration work.
- Keeps Web API and Mock behavior aligned at the UI state layer.

Cons:

- Export format is per-browser, not per-machine or per-account.

This remains the lowest-cost option, but is no longer the default recommendation now that persistent configuration is being considered.

### Approach B: Persist export format in backend settings

Extend backend `SettingsDTO` / `PATCH /api/settings` to store curated export format centrally, and expand curated export capability to support JPG.

Pros:

- One preference shared across browsers for the same backend.
- Behaves like a real product setting instead of a browser-local preference.
- Better fit if the user expects export behavior to survive browser changes or Web/Desktop switching.

Cons:

- Larger backend/frontend scope for a UI simplification.
- Current curated-frame preferences are already browser-local, so this would introduce a mixed persistence model.

Recommendation if persistence is required: use this approach.

### Approach C: Keep per-entry format pick behind a secondary menu

Replace visible PNG/WebP buttons with a single primary export button that expands to a format picker.

Pros:

- Preserves per-export flexibility.

Cons:

- Does not satisfy the desired simplification.
- Still forces users to think about format during export.

Not recommended.

## Recommended Design

### 1. Export-format source of truth

Confirmed decisions:

- export format must be a real Curated setting, not browser-local state
- JPG must be supported as an export format
- JPG is the default export format

Add a curated-frame export-format preference to backend settings:

- type: `"jpg" | "webp" | "png"`
- default: `"jpg"`
- read path: `GET /api/settings`
- write path: `PATCH /api/settings`
- persistence: same backend settings file / app configuration pipeline as other Curated settings

Frontend behavior:

- Web API mode reads and writes this preference through the existing settings flow.
- Mock mode can keep a local in-memory/browser fallback only for demo parity, but the product source of truth is backend settings.

### 2. Settings page behavior

In the curated-frame settings card inside `SettingsPage.vue`, add a new export-format control:

- section title: curated-frame export image settings
- field label: export image format
- options:
  - JPG
  - WebP
  - PNG

Behavior:

- Changing the control sends a settings update through the existing settings pipeline.
- The control reflects the persisted backend value on load.
- The UI copy should make it clear that this controls export output format for detail, context-menu, and batch export.
- The control should follow the existing logging-level setting pattern and use the same dropdown select / combobox-style interaction rather than segmented buttons.
- Add a short helper description under the title, explaining that single export and batch export both follow this format setting.

### 3. Single-frame export entry points

#### Detail dialog

Replace the two format-specific buttons with one export button.

Behavior:

- Button label: `Export`
- Secondary hint or inline caption can show the active format, e.g. `Current format: JPG`
- Clicking export reads the persisted settings value and calls the existing export path with that format

#### Right-click context menu

Replace:

- `Export WebP`
- `Export PNG`

with:

- `Export`

Behavior:

- The menu emits one `export` event.
- Parent code reads the persisted settings value and dispatches the existing export flow.
- The menu remains disabled when Web API is unavailable, matching existing behavior.

### 4. Batch export entry point

Keep batch export in `CuratedFramesBatchActionBar.vue`, but collapse the two buttons into one:

- `Export`

Behavior:

- Clicking export reads the persisted settings value and reuses the current selected-frame export path.
- Actor-group naming rules remain unchanged.
- Existing selection, delete, clear-selection, and exit actions remain unchanged.

### 5. Internal wiring changes

Refactor the curated-frame export code so the format decision is centralized:

- add a helper/composable to read the preferred export format from settings state
- keep `runExport(ids, actorName, format, errorTarget)` unchanged or nearly unchanged
- replace format-specific public methods and emits with generic export methods and emits:
  - batch: `exportSelected`
  - dialog: `exportSingleFromDialog`
  - context menu: `export`

This keeps the backend request body unchanged while simplifying the UI seams.

### 6. Error handling and UX

- Export errors continue to surface in the same location as today:
  - detail dialog error area
  - batch toolbar error area
- No new confirmation dialog is needed.
- If the preferred format is invalid or missing in settings, fall back to `jpg`.

### 7. JPG export backend behavior

Add JPEG/JPG export support to curated export.

Recommended behavior:

- Settings and frontend use the user-facing label `JPG`
- API payload value should be `jpg`
- Backend should accept `jpg` and optionally tolerate `jpeg` as an alias for compatibility

Implementation direction:

- Add a JPEG encoder path in `backend/internal/curatedexport`
- Reuse the existing `FrameMetaJSON`
- Embed metadata in JPEG EXIF `UserComment`, matching the existing WebP EXIF strategy as closely as practical
- Add filename helper for `.jpg`

Why this is the recommended JPG path:

- It keeps JPG semantically aligned with WebP as an EXIF-backed export format
- It avoids treating JPG as a metadata-poor fallback while PNG/WebP remain rich exports
- It preserves future import/reconciliation options better than a metadata-free JPEG

## Files Expected To Change

- `src/api/types.ts`
- `src/services/contracts/library-service.ts`
- `src/services/adapters/web/web-library-service.ts`
- `src/services/adapters/mock/mock-library-service.ts`
- `src/components/jav-library/SettingsPage.vue`
- `src/components/jav-library/CuratedFrameContextMenu.vue`
- `src/components/jav-library/CuratedFrameContextMenu.test.ts`
- `src/components/jav-library/CuratedFramesBatchActionBar.vue`
- `src/views/CuratedFramesView.vue`
- `src/components/jav-library/CuratedFramesLibrary.vue`
- `src/locales/zh-CN.json`
- `src/locales/en.json`
- `src/locales/ja.json`
- `backend/internal/contracts/contracts.go`
- `backend/internal/curatedexport/exif.go`
- `backend/internal/curatedexport/filename.go`
- `backend/internal/curatedexport/jpeg.go`
- `backend/internal/curatedexport/jpeg_test.go`
- `backend/internal/config/library_settings.go`
- `backend/internal/config/library_settings_test.go`
- `backend/internal/app/app.go`
- `backend/internal/server/server.go`
- `backend/internal/server/server_test.go`
- `.cursor/rules/project-facts.mdc`
- `API.md`
- `CLAUDE.md`

## Testing

Add or update frontend tests for:

- export-format storage default and persistence
- settings control updates export-format preference
- context menu renders one export action instead of two
- batch toolbar renders one export action instead of two
- detail dialog renders one export action instead of two
- export code uses stored format for:
  - detail export
  - context-menu export
  - batch export

Add or update backend tests for:

- settings default value is `jpg`
- settings patch accepts `jpg`, `webp`, `png` and rejects unsupported values
- curated export handler accepts `format=jpg`
- single-file JPG export returns `image/jpeg`
- generated JPG filenames end with `.jpg`
- JPG export contains expected metadata bytes or passes the chosen metadata embedding assertions

## Out Of Scope

- Per-export temporary format override
- Changing backend export metadata or ZIP behavior

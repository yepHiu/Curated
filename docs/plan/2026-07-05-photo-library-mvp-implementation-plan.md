# Photo Library MVP Implementation Plan

> 历史实验方案保留。2026-09-10 主线 Beta 的批准范围、当前限制及验收以 [整合记录](2026-09-10-comic-photo-beta-integration.md) 和 REQ-0046 / REQ-0047 为准。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an independent optional 写真库 that sits beside 影片 and 漫画, starting from a small service/sidebar/settings spine and expanding toward zip/cbz 写真集 import, browsing, detail, and viewer flows.

**Architecture:** 写真库 is a separate media domain with independent frontend service, routes, settings, backend contracts, tables, API handlers, watcher, and cache. It may share neutral low-level image-archive code with 漫画, but it must not store 写真集 in comic tables or expose them through comic APIs.

**Tech Stack:** Vue 3 + TypeScript + Vue Router + Vitest + shadcn-vue frontend; Go backend with SQLite migrations, HTTP handlers, config persistence, and task queue patterns already used by the comic MVP.

---

## File Structure Map

Frontend domain and service:

- Create `src/domain/photo/types.ts`: 写真集 domain types such as `PhotoBook`, `PhotoPage`, `PhotoLibrarySetting`, `PhotoViewerSettings`, and patch/list params.
- Create `src/services/contracts/photo-library-service.ts`: `PhotoLibraryService` interface mirroring comic service boundaries with photo naming.
- Create `src/services/photo-library-service.ts`: environment switch between web and mock implementations.
- Create `src/services/adapters/mock/mock-photo-library-service.ts`: mock service state for UI tests and local development.
- Create `src/services/adapters/web/web-photo-library-service.ts`: web API adapter around `/api/library/photos` and `/api/settings`.
- Modify `src/api/types.ts`: add photo settings, paths, book/page/progress/preference/cache DTOs.
- Create `src/api/photo-endpoints.ts`: central frontend URL builders for photo API routes.

Frontend shell and UI:

- Modify `src/components/jav-library/AppSidebar.vue`: add optional `写真` nav item when photo library is enabled.
- Modify `src/components/jav-library/ImportMenu.vue`: add `PhotoImportDialog` when photo library is enabled.
- Create `src/components/jav-library/PhotoImportDialog.vue`: zip/cbz import dialog for photo books.
- Create `src/views/PhotosView.vue`: photo wall route.
- Create `src/views/PhotoDetailView.vue`: photo book detail route.
- Create `src/views/PhotoViewerView.vue`: immersive photo viewer route.
- Create `src/components/jav-library/photos/PhotoLibraryPage.vue`: toolbar, sort controls, batch shell, grid host.
- Create `src/components/jav-library/photos/PhotoCard.vue`: photo book card aligned with movie/comic cards.
- Create `src/components/jav-library/photos/VirtualPhotoGrid.vue`: grid layout using movie grid variables.
- Create `src/components/jav-library/photos/PhotoDetailPanel.vue`: title/tags/rating/cover/preview/actions.
- Create `src/components/jav-library/photos/PhotoPagePreviewGrid.vue`: image-only page previews.
- Create `src/components/jav-library/photos/PhotoViewer.vue`: photo-named wrapper around the shared comic viewer behavior.

Frontend settings:

- Create `src/components/jav-library/settings/SettingsPhotoLibrarySection.vue`.
- Create `src/components/jav-library/settings/SettingsPhotoLibraryPathsSection.vue`.
- Create `src/components/jav-library/settings/SettingsPhotoLibraryPathActions.vue`.
- Create `src/components/jav-library/settings/SettingsPhotoViewerSection.vue`.
- Create `src/components/jav-library/settings/SettingsPhotoCacheSection.vue`.
- Modify `src/components/jav-library/SettingsPage.vue`: render photo settings section.

Router and navigation:

- Modify `src/router/index.ts`: add `/photos`, `/photos/:id`, `/photos/:id/view/:pageIndex?`.
- Create `src/router/photo-library-route.test.ts`.
- Modify `src/layouts/AppShell.vue`: treat `photos` as primary/flush route and show shell search.
- Modify `src/lib/navigation-intent.ts`: add photo viewer return behavior if needed.

Backend contracts/config/storage:

- Modify `backend/internal/config/config.go`: add photo settings defaults.
- Modify `backend/internal/config/library_settings.go`: merge photo settings from `library-config.cfg`.
- Create `backend/internal/config/photo_settings.go`: parse photo viewer/cache config.
- Modify `backend/internal/contracts/contracts.go`: add photo fields to settings DTO and patch DTO.
- Create `backend/internal/contracts/photo_contracts.go`: path/book/page/progress/cache DTOs.
- Modify `backend/internal/app/app.go`: photo settings getters/setters and later watcher/scanner integration.
- Create `backend/internal/storage/photo_paths_repository.go`.
- Create `backend/internal/storage/photo_books_repository.go`.
- Create `backend/internal/storage/photo_pages_repository.go`.
- Create `backend/internal/storage/photo_tags_repository.go`.
- Create `backend/internal/storage/photo_progress_repository.go`.
- Create `backend/internal/storage/photo_preferences_repository.go`.
- Create `backend/internal/storage/photo_cache_repository.go`.
- Create `backend/internal/storage/migrations/0027_photo_library.sql`.

Backend archive/scan/API:

- Create or refactor `backend/internal/imagearchive` from `comicarchive` behavior.
- Create `backend/internal/photoscanner`.
- Create `backend/internal/photocache`.
- Create `backend/internal/photowatch`.
- Create `backend/internal/server/photo_paths_handlers.go`.
- Create `backend/internal/server/photo_library_handlers.go`.
- Create `backend/internal/server/photo_page_handlers.go`.
- Create `backend/internal/server/photo_progress_handlers.go`.
- Create `backend/internal/server/photo_import_handlers.go`.
- Create `backend/internal/server/photo_scan_handlers.go`.
- Create `backend/internal/server/photo_cache_handlers.go`.
- Modify `backend/internal/server/server.go`: register photo routes under `/api/library/photos` and `/api/import/photos`.

Docs:

- Update `.cursor/rules/project-facts.mdc`, `README.md`, `CLAUDE.md`, and `docs/reference/architecture-and-implementation.html` after meaningful endpoints/settings exist.
- Update `docs/reference/2026-03-21-library-organize.md` if photo settings are persisted in `library-config.cfg`.

---

## Task 1: Frontend Photo Service Spine and Sidebar Entry

**Files:**

- Create: `src/domain/photo/types.ts`
- Create: `src/services/contracts/photo-library-service.ts`
- Create: `src/services/adapters/mock/mock-photo-library-service.ts`
- Create: `src/services/adapters/web/web-photo-library-service.ts`
- Create: `src/services/photo-library-service.ts`
- Modify: `src/components/jav-library/AppSidebar.vue`
- Modify: `src/components/jav-library/AppSidebar.test.ts`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`

- [ ] **Step 1: Write failing sidebar tests**

Add to `src/components/jav-library/AppSidebar.test.ts`:

```ts
const photoLibraryEnabled = ref(false)
const refreshPhotoSettings = vi.fn()

vi.mock("@/services/photo-library-service", () => ({
  usePhotoLibraryService: () => ({
    photoLibraryEnabled: computed(() => photoLibraryEnabled.value),
    refreshSettings: refreshPhotoSettings,
  }),
}))
```

Update `beforeEach`:

```ts
photoLibraryEnabled.value = false
refreshPhotoSettings.mockReset()
```

Add tests:

```ts
it("hides the photo entry when the photo library is disabled", async () => {
  photoLibraryEnabled.value = false

  const wrapper = mount(AppSidebar, { props: { compact: false } })
  await flushPromises()

  expect(wrapper.text()).not.toContain("nav.photos")
})

it("shows the photo entry when the photo library is enabled", async () => {
  photoLibraryEnabled.value = true

  const wrapper = mount(AppSidebar, { props: { compact: false } })
  await flushPromises()

  const photoLink = wrapper
    .findAll("[data-sidebar-nav-link]")
    .find((link) => link.text().includes("nav.photos"))

  expect(photoLink?.attributes("data-to")).toContain('"name":"photos"')
})
```

- [ ] **Step 2: Run tests to verify RED**

Run:

```powershell
pnpm vitest run src/components/jav-library/AppSidebar.test.ts --reporter=verbose
```

Expected: FAIL because `@/services/photo-library-service` does not exist and/or `AppSidebar` does not render `nav.photos`.

- [ ] **Step 3: Create photo domain and service contracts**

Create `src/domain/photo/types.ts` with minimal settings-only types:

```ts
export type PhotoViewerMode = "page" | "scroll"
export type PhotoFitMode = "contain" | "width"
export type PhotoViewingDirection = "ltr" | "rtl"

export interface PhotoLibrarySetting {
  id: string
  path: string
  title: string
  firstLibraryScanPending?: boolean
}

export interface PhotoViewerSettings {
  mode: PhotoViewerMode
  fit: PhotoFitMode
  direction: PhotoViewingDirection
}

export interface PhotoCacheSettings {
  maxBytes: number
}
```

Create `src/services/contracts/photo-library-service.ts`:

```ts
import type { ComputedRef } from "vue"
import type {
  PhotoCacheSettings,
  PhotoLibrarySetting,
  PhotoViewerSettings,
} from "@/domain/photo/types"

export interface PhotoLibraryService {
  photoLibraryEnabled: ComputedRef<boolean>
  autoPhotoLibraryWatch: ComputedRef<boolean>
  photoLibraryPaths: ComputedRef<readonly PhotoLibrarySetting[]>
  defaultPhotoImportLibraryPathId: ComputedRef<string>
  photoViewer: ComputedRef<PhotoViewerSettings>
  photoCache: ComputedRef<PhotoCacheSettings>
  refreshSettings(): Promise<void>
}
```

Create mock and web services with disabled defaults:

```ts
import { computed, ref } from "vue"
import type { PhotoLibraryService } from "@/services/contracts/photo-library-service"

const enabled = ref(false)

export const mockPhotoLibraryService: PhotoLibraryService = {
  photoLibraryEnabled: computed(() => enabled.value),
  autoPhotoLibraryWatch: computed(() => true),
  photoLibraryPaths: computed(() => []),
  defaultPhotoImportLibraryPathId: computed(() => ""),
  photoViewer: computed(() => ({ mode: "page", fit: "contain", direction: "ltr" })),
  photoCache: computed(() => ({ maxBytes: 5 * 1024 * 1024 * 1024 })),
  async refreshSettings() {},
}
```

`web-photo-library-service.ts` can initially re-export the mock shape with disabled defaults, then be replaced when backend settings land:

```ts
export { mockPhotoLibraryService as webPhotoLibraryService } from "@/services/adapters/mock/mock-photo-library-service"
```

Create `src/services/photo-library-service.ts`:

```ts
import { mockPhotoLibraryService } from "@/services/adapters/mock/mock-photo-library-service"
import { webPhotoLibraryService } from "@/services/adapters/web/web-photo-library-service"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

export const usePhotoLibraryService = () =>
  USE_WEB ? webPhotoLibraryService : mockPhotoLibraryService
```

- [ ] **Step 4: Wire AppSidebar**

In `src/components/jav-library/AppSidebar.vue`, import photo service and add the optional nav item after comics:

```ts
import { Image } from "lucide-vue-next"
import { usePhotoLibraryService } from "@/services/photo-library-service"

const photoService = usePhotoLibraryService()
const photoLibraryEnabled = computed(() => photoService.photoLibraryEnabled.value)
```

Add:

```ts
if (photoLibraryEnabled.value) {
  browse.push({ label: t("nav.photos"), page: "photos", icon: Image })
}
```

Update `isActive`:

```ts
if (page === "photos") {
  return ["photos", "photo-detail", "photo-viewer"].includes(String(route.name ?? ""))
}
```

- [ ] **Step 5: Add locale labels**

Add:

```json
"photos": "写真"
```

to `src/locales/zh-CN.json` under `nav`.

Add English/Japanese nav labels:

```json
"photos": "Photo Books"
```

```json
"photos": "写真"
```

- [ ] **Step 6: Run tests to verify GREEN**

Run:

```powershell
pnpm vitest run src/components/jav-library/AppSidebar.test.ts --reporter=verbose
pnpm typecheck
pnpm lint
```

Expected: PASS.

---

## Task 2: Photo Routes and Flush Shell Search Skeleton

**Files:**

- Create: `src/router/photo-library-route.test.ts`
- Create: `src/views/PhotosView.vue`
- Create: `src/views/PhotoDetailView.vue`
- Create: `src/views/PhotoViewerView.vue`
- Modify: `src/router/index.ts`
- Modify: `src/layouts/AppShell.vue`
- Modify: `src/layouts/AppShell.test.ts`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`

- [ ] **Step 1: Write failing router and shell tests**

Add tests asserting:

- `/photos` resolves to route name `photos`.
- `/photos/photo-1` resolves to `photo-detail`.
- `/photos/photo-1/view/3` resolves to `photo-viewer`.
- `photos` is a primary route and does not show the global back-to-library link.
- `photos` uses the shell search input with `photos.searchPlaceholder`.
- `photos` route frame has no global page padding.

- [ ] **Step 2: Run tests to verify RED**

Run:

```powershell
pnpm vitest run src/router/photo-library-route.test.ts src/layouts/AppShell.test.ts --reporter=verbose
```

Expected: FAIL because routes and shell behavior are missing.

- [ ] **Step 3: Add placeholder views**

`PhotosView.vue` should render a full-height flush content frame with `data-photos-view-content`.

`PhotoDetailView.vue` and `PhotoViewerView.vue` can render minimal placeholder content with route params while backend/UI tasks are pending.

- [ ] **Step 4: Add routes**

Add route records:

```ts
{
  path: "/photos",
  name: "photos",
  component: () => import("@/views/PhotosView.vue"),
},
{
  path: "/photos/:id",
  name: "photo-detail",
  component: () => import("@/views/PhotoDetailView.vue"),
},
{
  path: "/photos/:id/view/:pageIndex?",
  name: "photo-viewer",
  component: () => import("@/views/PhotoViewerView.vue"),
},
```

- [ ] **Step 5: Wire shell**

Treat `photos` like `comics`:

- `isPhotosRoute`
- primary route detection
- flush frame route list
- shell search placeholder `photos.searchPlaceholder`
- no movie search suggestions while on photos

- [ ] **Step 6: Run tests**

Run:

```powershell
pnpm vitest run src/router/photo-library-route.test.ts src/layouts/AppShell.test.ts --reporter=verbose
pnpm typecheck
pnpm lint
```

Expected: PASS.

---

## Task 3: Backend Photo Settings Contract and Persistence

**Files:**

- Create: `backend/internal/config/photo_settings.go`
- Create: `backend/internal/config/photo_settings_test.go`
- Create: `backend/internal/app/photo_settings_test.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/library_settings.go`
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/server/settings_handlers.go` or current settings handler owner
- Modify: `src/api/types.ts`

- [ ] **Step 1: Write failing config tests**

Tests must assert defaults:

- `PhotoLibraryEnabled == false`
- `AutoPhotoLibraryWatch == true`
- `PhotoViewer.Mode == "page"`
- `PhotoViewer.Fit == "contain"`
- `PhotoViewer.Direction == "ltr"`
- `PhotoCache.MaxBytes == 5 GiB`

Tests must assert `library-config.cfg` merges:

```json
{
  "photoLibraryEnabled": true,
  "autoPhotoLibraryWatch": false,
  "defaultPhotoImportLibraryPathId": "photo-lib-main",
  "photoViewer": { "mode": "scroll", "fit": "width", "direction": "ltr" },
  "photoCache": { "maxBytes": 2147483648 }
}
```

- [ ] **Step 2: Run Go tests to verify RED**

Run:

```powershell
go test ./internal/config ./internal/app
```

from `backend`.

Expected: FAIL because photo settings fields do not exist.

- [ ] **Step 3: Implement config and contracts**

Mirror comic settings with photo names:

- `PhotoLibraryEnabled`
- `AutoPhotoLibraryWatch`
- `DefaultPhotoImportLibraryPathID`
- `PhotoViewer`
- `PhotoCache`

Add DTO fields to settings responses and patches.

- [ ] **Step 4: Implement app getters/setters**

Add:

- `PhotoLibraryEnabled()`
- `AutoPhotoLibraryWatch()`
- `DefaultPhotoImportLibraryPathID()`
- `PhotoViewerSettings()`
- `PhotoCacheSettings()`
- `SetPhotoLibraryEnabled(bool)`
- `SetAutoPhotoLibraryWatch(bool)`
- `SetDefaultPhotoImportLibraryPathID(string)`
- `SetPhotoViewerSettings(...)`
- `SetPhotoCacheSettings(...)`

Persist all fields through `library-config.cfg`.

- [ ] **Step 5: Run tests**

Run:

```powershell
go test ./internal/config ./internal/app
pnpm typecheck
```

Expected: PASS.

---

## Task 4: Frontend Web Photo Settings Adapter and Settings Page

**Files:**

- Modify: `src/api/types.ts`
- Modify: `src/services/adapters/web/web-photo-library-service.ts`
- Create: `src/services/adapters/web/web-photo-library-service.test.ts`
- Create: `src/components/jav-library/settings/SettingsPhotoLibrarySection.vue`
- Create: `src/components/jav-library/settings/SettingsPhotoLibrarySection.test.ts`
- Modify: `src/components/jav-library/SettingsPage.vue`
- Modify: `src/components/jav-library/SettingsPage.test.ts`

- [ ] **Step 1: Write failing adapter tests**

Assert `refreshSettings()` maps backend settings into:

- `photoLibraryEnabled`
- `autoPhotoLibraryWatch`
- `photoLibraryPaths`
- `defaultPhotoImportLibraryPathId`
- `photoViewer`
- `photoCache`

Assert setting toggles PATCH only photo fields.

- [ ] **Step 2: Run RED**

Run:

```powershell
pnpm vitest run src/services/adapters/web/web-photo-library-service.test.ts --reporter=verbose
```

Expected: FAIL.

- [ ] **Step 3: Implement web adapter**

Use the same settings fetch/patch helpers as the comic adapter, with photo field names.

- [ ] **Step 4: Implement settings section**

First slice includes:

- Enable/disable photo library.
- Auto scan toggle.
- Empty path state.

Path CRUD can be added in Task 5 after backend path repository exists.

- [ ] **Step 5: Run tests**

Run:

```powershell
pnpm vitest run src/services/adapters/web/web-photo-library-service.test.ts src/components/jav-library/settings/SettingsPhotoLibrarySection.test.ts src/components/jav-library/SettingsPage.test.ts --reporter=verbose
pnpm typecheck
pnpm lint
```

Expected: PASS.

---

## Task 5: Backend Photo Path Storage and API

**Files:**

- Create: `backend/internal/storage/migrations/0027_photo_library.sql`
- Create: `backend/internal/storage/photo_paths_repository.go`
- Create: `backend/internal/storage/photo_paths_repository_test.go`
- Create: `backend/internal/server/photo_paths_handlers.go`
- Create: `backend/internal/server/photo_paths_handlers_test.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/contracts/photo_contracts.go`

- [ ] **Step 1: Write failing storage tests**

Assert:

- Migration creates `photo_library_paths`.
- Absolute path can be added.
- Duplicate normalized path is rejected.
- Non-absolute path is rejected.
- Paths can be listed, renamed, and removed.

- [ ] **Step 2: Run RED**

Run:

```powershell
go test ./internal/storage
```

Expected: FAIL.

- [ ] **Step 3: Implement migration and repository**

Create `photo_library_paths` matching comic path behavior but independent table.

- [ ] **Step 4: Write and implement handler tests**

Add handlers for:

- `POST /api/library/photos/paths`
- `PATCH /api/library/photos/paths/{id}`
- `DELETE /api/library/photos/paths/{id}`
- `POST /api/library/photos/paths/{id}/scan`

The scan endpoint can return a queued task only after Task 8; until then, it should return a clear `PHOTO_SCAN_NOT_READY` only if the UI does not call it. Prefer implementing actual scan in Task 8 before exposing scan in UI.

- [ ] **Step 5: Run tests**

Run:

```powershell
go test ./internal/storage ./internal/server
```

Expected: PASS.

---

## Task 6: Photo Book Tables, Mock Library Page, and Grid UI

**Files:**

- Create: `src/lib/photo-sort.ts`
- Create: `src/lib/photo-search.ts`
- Create: `src/components/jav-library/photos/PhotoCard.vue`
- Create: `src/components/jav-library/photos/VirtualPhotoGrid.vue`
- Create: `src/components/jav-library/photos/PhotoLibraryPage.vue`
- Modify: `src/views/PhotosView.vue`
- Modify: `src/services/adapters/mock/mock-photo-library-service.ts`
- Create tests beside each file.

- [ ] **Step 1: Write failing frontend tests**

Assert:

- Photo library toolbar has sort controls and no inline search input.
- Photo grid uses movie grid CSS variables.
- Photo card does not show source file path.
- PhotosView filters by `q` and sorts by query `sort`.

- [ ] **Step 2: Run RED**

Run:

```powershell
pnpm vitest run src/components/jav-library/photos/PhotoLibraryPage.test.ts src/components/jav-library/photos/VirtualPhotoGrid.test.ts src/views/PhotosView.test.ts --reporter=verbose
```

Expected: FAIL.

- [ ] **Step 3: Implement mock photo books and UI**

Use two or three mock `PhotoBook` entries with cover URLs and page counts.

- [ ] **Step 4: Run tests**

Run:

```powershell
pnpm vitest run src/components/jav-library/photos/PhotoLibraryPage.test.ts src/components/jav-library/photos/VirtualPhotoGrid.test.ts src/views/PhotosView.test.ts --reporter=verbose
pnpm typecheck
pnpm lint
```

Expected: PASS.

---

## Task 7: Photo Detail and Viewer Frontend

**Files:**

- Create: `src/components/jav-library/photos/PhotoDetailPanel.vue`
- Create: `src/components/jav-library/photos/PhotoPagePreviewGrid.vue`
- Create: `src/components/jav-library/photos/PhotoViewer.vue`
- Modify: `src/views/PhotoDetailView.vue`
- Modify: `src/views/PhotoViewerView.vue`
- Create tests beside each file.

- [ ] **Step 1: Write failing tests**

Assert:

- Detail shows title, tags, rating, cover, preview images, and `浏览`.
- More menu contains edit/delete/reveal actions.
- Preview cards show only images, not page labels.
- Viewer uses `浏览` language and hides comic stitch buttons.

- [ ] **Step 2: Run RED**

Run:

```powershell
pnpm vitest run src/views/PhotoDetailView.test.ts src/components/jav-library/photos/PhotoViewer.test.ts --reporter=verbose
```

Expected: FAIL.

- [ ] **Step 3: Implement detail and viewer using photo naming**

Reuse comic visual patterns, but keep photo component names, photo route names, and photo copy.

- [ ] **Step 4: Run tests**

Run:

```powershell
pnpm vitest run src/views/PhotoDetailView.test.ts src/components/jav-library/photos/PhotoViewer.test.ts --reporter=verbose
pnpm typecheck
pnpm lint
```

Expected: PASS.

---

## Task 8: Backend Image Archive Refactor and Photo Scanner

**Files:**

- Create: `backend/internal/imagearchive/service.go`
- Create: `backend/internal/imagearchive/service_test.go`
- Create: `backend/internal/photoscanner/service.go`
- Create: `backend/internal/photoscanner/service_test.go`
- Modify: `backend/internal/comicarchive/service.go` only if replaced by wrapper.
- Modify: `backend/internal/comicscanner/service.go` only if it uses neutral archive service.
- Create photo book/page repositories if not already complete.

- [ ] **Step 1: Write image archive tests**

Assert:

- `.zip` and `.cbz` supported.
- Image entries are naturally sorted.
- Non-image entries ignored.
- Empty archive returns a typed empty error.
- Reading by page index returns bytes and content type.

- [ ] **Step 2: Run RED**

Run:

```powershell
go test ./internal/imagearchive
```

Expected: FAIL.

- [ ] **Step 3: Implement `imagearchive`**

Port logic from `comicarchive` without comic naming.

- [ ] **Step 4: Write photo scanner tests**

Assert scanning a photo path creates `photo_books` and `photo_pages`, with first page cover and source type `archive`.

- [ ] **Step 5: Run tests**

Run:

```powershell
go test ./internal/imagearchive ./internal/photoscanner ./internal/comicscanner
```

Expected: PASS.

---

## Task 9: Photo Library Backend Handlers, Import, Progress, and Cache

**Files:**

- Create server handlers listed in the file structure map.
- Create repository tests for books/pages/tags/progress/preferences/cache.
- Modify `backend/internal/app/app.go` to queue photo scans/imports.
- Modify `backend/internal/server/server.go` route registration.

- [ ] **Step 1: Write endpoint tests**

Cover:

- list/get/patch/delete photo books.
- list pages.
- image/thumbnail/cover resources.
- get/put/delete progress.
- get/put preferences.
- import photos.
- cache status/cleanup.

- [ ] **Step 2: Run RED**

Run:

```powershell
go test ./internal/server ./internal/storage ./internal/app
```

Expected: FAIL.

- [ ] **Step 3: Implement handlers and app methods**

Use comic handlers as reference, replacing all table/API/task names with photo names.

- [ ] **Step 4: Run tests**

Run:

```powershell
go test ./internal/server ./internal/storage ./internal/app
```

Expected: PASS.

---

## Task 10: Browser QA and Documentation Update

**Files:**

- Modify `.cursor/rules/project-facts.mdc`
- Modify `README.md`
- Modify `CLAUDE.md`
- Modify `docs/reference/architecture-and-implementation.html`
- Modify `docs/reference/2026-03-21-library-organize.md` if photo settings are persisted in `library-config.cfg`

- [ ] **Step 1: Run full verification**

Run:

```powershell
pnpm vitest run --reporter=verbose
pnpm typecheck
pnpm lint
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Browser QA**

Verify:

- `写真` hidden when disabled.
- `写真` visible when enabled.
- `/photos` page uses shell search and has no global page padding.
- Photo grid card sizing aligns with movie/comic grid.
- Photo detail loads.
- Photo viewer shows images and no comic stitch controls.
- Console has no relevant error/warn.

- [ ] **Step 3: Update docs**

Update docs listed above with photo settings, routes, and API endpoints.

- [ ] **Step 4: Final verification**

Run:

```powershell
pnpm typecheck
pnpm lint
go test ./...
```

Expected: PASS.

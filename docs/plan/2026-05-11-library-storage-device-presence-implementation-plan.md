# Library Storage Device Presence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Windows-first storage presence check for configured Curated library paths, with backend API, settings-page status, and startup notification-center alerts.

**Architecture:** Add a backend `storagehealth` domain module that resolves each configured library path to a platform probe result, compares it with a persisted volume binding, and returns a normalized status. Expose the result through `/api/library/paths/storage-status` endpoints, then surface it through the frontend service layer, Settings -> Library & storage, and the existing app toast / notification center.

**Tech Stack:** Go 1.25 backend, SQLite migrations, Windows API via `golang.org/x/sys/windows`, Vue 3 + TypeScript + Vite, Vitest, Go test.

---

## File Map

- Create `backend/internal/storagehealth/`: storage-device status classification, platform probes, service tests.
- Modify `backend/internal/contracts/contracts.go`: add storage status DTOs and request DTOs.
- Modify `backend/internal/storage/`: add migration and repository methods for library-path storage bindings.
- Modify `backend/internal/app/app.go`: construct the storage-health service and expose methods used by HTTP handlers.
- Modify `backend/internal/server/server.go`: add routes and handlers for status query, check, and rebind.
- Modify `src/api/types.ts` and `src/api/endpoints.ts`: add frontend DTOs and API calls.
- Modify `src/services/contracts/library-service.ts`, `src/services/adapters/web/web-library-service.ts`, and `src/services/adapters/mock/mock-library-service.ts`: expose status state and operations.
- Modify `src/components/jav-library/settings/SettingsLibraryPathList.vue` and parent props: display status badge and add manual recheck.
- Create `src/composables/use-library-storage-status-alerts.ts`: startup check + notification-center integration.
- Modify `src/layouts/AppShell.vue`: mount the startup alert composable.
- Modify locale files: add storage-status copy in `en`, `zh-CN`, `ja`.
- Update docs: `project-facts.mdc`, `README*`, `docs/reference/architecture-and-implementation.html` after implementation behavior is stable.

## Task 1: Backend Storage-Health Classifier

**Files:**
- Create: `backend/internal/storagehealth/types.go`
- Create: `backend/internal/storagehealth/checker.go`
- Create: `backend/internal/storagehealth/checker_test.go`

- [ ] **Step 1: Write failing classifier tests**

Create tests covering:

```go
func TestCheckerClassifiesOfflineRoot(t *testing.T)
func TestCheckerClassifiesVolumeMismatch(t *testing.T)
func TestCheckerBindsOnlinePathWhenMissingBinding(t *testing.T)
func TestCheckerClassifiesPathMissingAfterVolumeMatch(t *testing.T)
func TestCheckerClassifiesPermissionDeniedAfterVolumeMatch(t *testing.T)
```

Run:

```powershell
cd backend
go test ./internal/storagehealth
```

Expected: fail because the package does not exist.

- [ ] **Step 2: Implement minimal classifier**

Implement:

```go
type Status string
const (
  StatusOnline Status = "online"
  StatusOffline Status = "offline"
  StatusVolumeMismatch Status = "volume_mismatch"
  StatusPathMissing Status = "path_missing"
  StatusPermissionDenied Status = "permission_denied"
  StatusUnknown Status = "unknown"
)

type ProbeResult struct {
  RootPath string
  RootAvailable bool
  PathExists bool
  PathIsDir bool
  PathReadable bool
  PermissionDenied bool
  VolumeID string
  VolumeLabel string
  FileSystem string
  DriveType string
  IdentityConfidence string
  ErrorMessage string
}
```

Add a checker that compares `ProbeResult` with expected binding fields and returns a DTO-shaped result.

- [ ] **Step 3: Verify backend classifier**

Run:

```powershell
cd backend
go test ./internal/storagehealth
```

Expected: pass.

## Task 2: Windows-First Platform Probe

**Files:**
- Create: `backend/internal/storagehealth/probe_windows.go`
- Create: `backend/internal/storagehealth/probe_other.go`
- Create: `backend/internal/storagehealth/probe_windows_test.go`

- [ ] **Step 1: Write failing probe tests for path root parsing**

Test Windows-style paths:

```go
func TestWindowsRootFromPath(t *testing.T) {
  got := windowsRootFromPath(`E:\Movies`)
  if got != `E:\` { t.Fatalf("root = %q", got) }
}
```

Run on Windows:

```powershell
cd backend
go test ./internal/storagehealth
```

Expected: fail because `windowsRootFromPath` is missing.

- [ ] **Step 2: Implement Windows probe**

Use `golang.org/x/sys/windows`:

- `windows.UTF16PtrFromString`
- `windows.GetDriveType`
- `windows.GetVolumeInformation`

The probe should:

- Resolve `E:\Movies` to `E:\`.
- Mark missing root as `RootAvailable=false`.
- Use `os.Stat` on the configured path only; never scan children.
- Return `PermissionDenied=true` on permission errors.

Non-Windows fallback should use `filepath.VolumeName`, `os.Stat`, and `unix`-agnostic best effort.

- [ ] **Step 3: Verify probe package**

Run:

```powershell
cd backend
go test ./internal/storagehealth
```

Expected: pass on Windows.

## Task 3: Persistence and API

**Files:**
- Create: `backend/internal/storage/migrations/0022_library_path_storage_bindings.sql`
- Create: `backend/internal/storage/library_path_storage_bindings.go`
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/server/server.go`
- Test: `backend/internal/server/library_path_storage_status_test.go`

- [ ] **Step 1: Write failing HTTP handler tests**

Create tests asserting:

- `GET /api/library/paths/storage-status` returns `items`.
- `POST /api/library/paths/storage-status/check` accepts optional `libraryPathIds`.
- `POST /api/library/paths/{id}/storage-binding/rebind` returns an online binding result for a configured path.

Run:

```powershell
cd backend
go test ./internal/server -run LibraryPathStorage
```

Expected: fail because routes and contracts do not exist.

- [ ] **Step 2: Add contracts and storage repository**

Add DTOs:

```go
type LibraryPathStorageStatus string
type LibraryPathStorageStatusDTO struct { ... }
type LibraryPathStorageStatusListDTO struct { Items []LibraryPathStorageStatusDTO `json:"items"` }
type CheckLibraryPathStorageStatusRequest struct { LibraryPathIDs []string `json:"libraryPathIds,omitempty"` }
```

Add SQLite table:

```sql
CREATE TABLE IF NOT EXISTS library_path_storage_bindings (
  library_path_id TEXT PRIMARY KEY,
  root_path TEXT NOT NULL,
  volume_id TEXT,
  volume_label TEXT,
  file_system TEXT,
  drive_type TEXT,
  identity_confidence TEXT NOT NULL DEFAULT 'unknown',
  bound_at TEXT,
  last_seen_at TEXT,
  last_checked_at TEXT,
  last_status TEXT,
  last_error TEXT,
  updated_at TEXT NOT NULL
);
```

- [ ] **Step 3: Wire app and server**

Add `LibraryPathStorageStatusProvider` interface to server dependencies, implement it on `app.App`, register routes:

```go
mux.HandleFunc("GET /api/library/paths/storage-status", h.handleGetLibraryPathStorageStatus)
mux.HandleFunc("POST /api/library/paths/storage-status/check", h.handleCheckLibraryPathStorageStatus)
mux.HandleFunc("POST /api/library/paths/{id}/storage-binding/rebind", h.handleRebindLibraryPathStorage)
```

- [ ] **Step 4: Verify backend API**

Run:

```powershell
cd backend
go test ./internal/storage ./internal/storagehealth ./internal/server ./internal/app
```

Expected: pass.

## Task 4: Frontend Service and Settings UI

**Files:**
- Modify: `src/api/types.ts`
- Modify: `src/api/endpoints.ts`
- Modify: `src/services/contracts/library-service.ts`
- Modify: `src/services/adapters/web/web-library-service.ts`
- Modify: `src/services/adapters/mock/mock-library-service.ts`
- Modify: `src/components/jav-library/settings/SettingsLibraryPathsSection.vue`
- Modify: `src/components/jav-library/settings/SettingsLibraryPathList.vue`
- Test: `src/components/jav-library/settings/SettingsLibraryPathList.test.ts`

- [ ] **Step 1: Write failing frontend component tests**

Assert that:

- Offline status displays a warning label.
- Online status displays an online label.
- Recheck event is emitted from the storage section.

Run:

```powershell
pnpm test -- src/components/jav-library/settings/SettingsLibraryPathList.test.ts
```

Expected: fail because status props do not exist.

- [ ] **Step 2: Implement DTOs and adapter state**

Add `libraryPathStorageStatuses` computed state and methods:

```ts
checkLibraryPathStorageStatus(libraryPathIds?: string[]): Promise<void>
rebindLibraryPathStorage(id: string): Promise<void>
```

- [ ] **Step 3: Implement status badge UI**

Render status copy beside each path title and expose a “Recheck storage” action in the section toolbar.

- [ ] **Step 4: Verify frontend UI**

Run:

```powershell
pnpm test -- src/components/jav-library/settings/SettingsLibraryPathList.test.ts
pnpm typecheck
```

Expected: pass.

## Task 5: Startup Toast and Notification Center

**Files:**
- Create: `src/composables/use-library-storage-status-alerts.ts`
- Modify: `src/composables/use-notification-center.ts`
- Modify: `src/layouts/AppShell.vue`
- Modify: `src/locales/en.json`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/ja.json`
- Test: `src/composables/use-library-storage-status-alerts.test.ts`

- [ ] **Step 1: Write failing notification tests**

Assert that:

- One offline path sends `pushAppToast` with notification type `storage`.
- Multiple abnormal paths are aggregated into one toast.
- Online-only results do not toast.

Run:

```powershell
pnpm test -- src/composables/use-library-storage-status-alerts.test.ts
```

Expected: fail because composable does not exist.

- [ ] **Step 2: Implement startup alert composable**

Use `libraryService.checkLibraryPathStorageStatus()` on mount in Web API mode, filter statuses not equal to `online`, push toast with `source.route = "/settings?section=library"`, and session-dedupe abnormal status keys.

- [ ] **Step 3: Verify startup alert**

Run:

```powershell
pnpm test -- src/composables/use-library-storage-status-alerts.test.ts
pnpm typecheck
pnpm lint
```

Expected: pass.

## Task 6: Flow Guards and Docs

**Files:**
- Modify: `src/components/jav-library/SettingsPage.vue`
- Modify: `src/components/jav-library/MovieImportDialog.vue`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `README.ja-JP.md`
- Modify: `docs/reference/architecture-and-implementation.html`

- [ ] **Step 1: Add guard tests where practical**

Settings path re-scan should not call scan API for known offline / mismatch path. Import should block when default import path is offline.

- [ ] **Step 2: Implement guards**

Use storage status from service state:

- `offline`, `volume_mismatch`, `path_missing`, `permission_denied` block scan/import.
- `unknown` allows action but displays a warning toast if check failed.

- [ ] **Step 3: Update docs**

Document Windows-first storage presence detection and future macOS/Linux adaptation.

- [ ] **Step 4: Final verification**

Run:

```powershell
pnpm test -- src/composables/use-library-storage-status-alerts.test.ts src/components/jav-library/settings/SettingsLibraryPathList.test.ts
pnpm typecheck
pnpm lint
cd backend
go test ./internal/storage ./internal/storagehealth ./internal/server ./internal/app
```

Expected: all commands pass.

# App Update Download Install Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend Curated app update from "check and browser download" to in-app installer download, SHA256 verification, and user-confirmed installer launch.

**Architecture:** Keep GitHub Releases as the update source. Extend `internal/appupdate.Service` with an installer artifact state machine and expose it through existing app update DTOs and `/api/app-update/*` endpoints. The frontend continues to use `useAppUpdate()`, but adds download/install actions and renders artifact progress in Settings -> About.

**Tech Stack:** Go HTTP backend, SQLite migrations, in-memory task manager, Vue 3 + TypeScript + shadcn-vue UI, Vitest.

---

## File Structure

- Modify `backend/internal/contracts/contracts.go`
  - Extend `AppUpdateStatusDTO` with artifact fields.
  - Add `AppUpdateInstallRequest`.
  - Add app update task/error constants.
- Modify `backend/internal/storage/app_update_status.go`
  - Persist downloaded artifact state in the existing singleton snapshot row.
- Create `backend/internal/storage/migrations/0023_app_update_artifact_status.sql`
  - Add artifact status, downloaded installer metadata, checksum, and install-attempt columns.
- Modify `backend/internal/appupdate/service.go`
  - Parse release asset digest / manifest checksum.
  - Download installer to the update cache.
  - Verify SHA256.
  - Track progress in `tasks.Manager`.
  - Launch installer with explicit install mode.
- Modify `backend/internal/app/app.go`
  - Wire the task manager and cache directory into app update service.
  - Expose download/install methods through `AppUpdateProvider`.
- Modify `backend/internal/server/app_update_handlers.go`
  - Add handlers for download, install, and cleanup.
- Modify `backend/internal/server/server.go`
  - Register new `/api/app-update/*` routes and provider methods.
- Modify `src/api/types.ts`
  - Add update artifact fields and install request type.
- Modify `src/api/endpoints.ts`
  - Add `downloadAppUpdateInstaller`, `installAppUpdate`, and `clearDownloadedAppUpdateInstaller`.
- Modify `src/composables/use-app-update.ts`
  - Add download/install action state and keep status synchronized.
- Modify `src/components/jav-library/settings/SettingsAppUpdateSection.vue`
  - Replace raw browser installer link primary action with in-app download/install actions.
  - Keep Release page fallback.
- Update docs:
  - `API.md`
  - `.cursor/rules/project-facts.mdc`
  - `README.md`
  - `CLAUDE.md` if the API list exists in this workspace.

## Task 1: Backend Contract And Storage

**Files:**
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/storage/app_update_status.go`
- Create: `backend/internal/storage/migrations/0023_app_update_artifact_status.sql`
- Test: `backend/internal/storage/app_update_status_test.go`

- [ ] **Step 1: Write the failing storage test**

Create `backend/internal/storage/app_update_status_test.go` with:

```go
package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestAppUpdateStatusSnapshotPersistsArtifactState(t *testing.T) {
	t.Parallel()

	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "app-update-artifact.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	want := AppUpdateStatusSnapshot{
		InstalledVersion:     "1.4.4",
		LatestVersion:        "1.4.5",
		Status:               "update-available",
		CheckedAt:            "2026-05-12T12:00:00Z",
		PublishedAt:          "2026-05-12T10:00:00Z",
		ReleaseName:          "Curated v1.4.5",
		ReleaseURL:           "https://github.com/yepHiu/Curated/releases/tag/v1.4.5",
		InstallerDownloadURL: "https://github.com/yepHiu/Curated/releases/download/v1.4.5/Curated-Setup-1.4.5.exe",
		InstallerSHA256:      "ABCDEF",
		ArtifactStatus:       "verified",
		DownloadedVersion:    "1.4.5",
		DownloadedFileName:   "Curated-Setup-1.4.5.exe",
		DownloadedFilePath:   filepath.Join(t.TempDir(), "Curated-Setup-1.4.5.exe"),
		DownloadedBytes:      123,
		TotalBytes:           123,
		SignatureStatus:      "not_checked",
		InstallReady:         true,
		LastInstallAttemptAt: "2026-05-12T12:05:00Z",
		LastInstallError:     "previous failure",
		Source:               "github-releases",
	}

	if err := store.UpsertAppUpdateStatusSnapshot(context.Background(), want); err != nil {
		t.Fatalf("UpsertAppUpdateStatusSnapshot() error = %v", err)
	}

	got, ok, err := store.GetAppUpdateStatusSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetAppUpdateStatusSnapshot() error = %v", err)
	}
	if !ok {
		t.Fatal("expected snapshot")
	}
	if got.ArtifactStatus != want.ArtifactStatus || got.DownloadedVersion != want.DownloadedVersion || !got.InstallReady {
		t.Fatalf("artifact state = %+v, want %+v", got, want)
	}
	if got.InstallerSHA256 != want.InstallerSHA256 || got.DownloadedBytes != want.DownloadedBytes || got.TotalBytes != want.TotalBytes {
		t.Fatalf("artifact metadata = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 2: Run the storage test and verify it fails**

Run: `cd backend && go test ./internal/storage -run TestAppUpdateStatusSnapshotPersistsArtifactState -count=1`

Expected: FAIL because `AppUpdateStatusSnapshot` does not have artifact fields yet.

- [ ] **Step 3: Implement contract and storage fields**

Add DTO fields to `contracts.AppUpdateStatusDTO` and matching fields to `storage.AppUpdateStatusSnapshot`. Add migration `0023_app_update_artifact_status.sql` with `ALTER TABLE app_update_status ADD COLUMN ...` statements for artifact state.

- [ ] **Step 4: Re-run the storage test**

Run: `cd backend && go test ./internal/storage -run TestAppUpdateStatusSnapshotPersistsArtifactState -count=1`

Expected: PASS.

## Task 2: Backend Download And Verification Service

**Files:**
- Modify: `backend/internal/appupdate/service.go`
- Test: `backend/internal/appupdate/service_test.go`

- [ ] **Step 1: Write the failing download test**

Add a test that serves a fake latest release with an `.exe` asset and `digest: "sha256:<hash>"`, calls `DownloadInstaller`, and expects:

```go
if dto.ArtifactStatus != "verified" { t.Fatalf(...) }
if dto.InstallReady != true { t.Fatalf(...) }
if dto.DownloadedBytes != int64(len(installerBytes)) { t.Fatalf(...) }
```

- [ ] **Step 2: Run the service test and verify it fails**

Run: `cd backend && go test ./internal/appupdate -run TestDownloadInstallerVerifiesSHA256Digest -count=1`

Expected: FAIL because `DownloadInstaller` is not implemented.

- [ ] **Step 3: Implement download, SHA256 verification, and progress state**

Implement:

```go
func (s *Service) DownloadInstaller(ctx context.Context) (contracts.AppUpdateStatusDTO, error)
```

Behavior:
- Require `status=update-available`.
- Require HTTPS or test server HTTP URL when configured in tests.
- Require installer URL and SHA256.
- Download to `<cacheDir>/updates/<version>/<assetName>.part`.
- Rename to final `.exe` after hash match.
- Persist `artifact_status=verified`, bytes, total, path, file name, version, and `install_ready=true`.

- [ ] **Step 4: Re-run appupdate tests**

Run: `cd backend && go test ./internal/appupdate -count=1`

Expected: PASS.

## Task 3: Backend HTTP Endpoints

**Files:**
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/server/app_update_handlers.go`
- Test: `backend/internal/server/app_update_test.go`

- [ ] **Step 1: Write failing handler tests**

Add tests for:
- `POST /api/app-update/download` calls provider `DownloadAppUpdateInstaller`.
- `POST /api/app-update/install` decodes `{ "mode": "silent" }` and calls provider `InstallAppUpdate`.
- `DELETE /api/app-update/downloaded-installer` calls provider cleanup.

- [ ] **Step 2: Run handler tests and verify failure**

Run: `cd backend && go test ./internal/server -run AppUpdate -count=1`

Expected: FAIL because routes/provider methods are missing.

- [ ] **Step 3: Implement provider methods and routes**

Extend `AppUpdateProvider`, route registration, and handler methods. Unsupported provider should return the existing unsupported DTO for download and install.

- [ ] **Step 4: Re-run handler tests**

Run: `cd backend && go test ./internal/server -run AppUpdate -count=1`

Expected: PASS.

## Task 4: Frontend API And Composable

**Files:**
- Modify: `src/api/types.ts`
- Modify: `src/api/endpoints.ts`
- Modify: `src/composables/use-app-update.ts`
- Test: `src/composables/use-app-update.test.ts`

- [ ] **Step 1: Write failing composable tests**

Add tests that mock:

```ts
api.downloadAppUpdateInstaller.mockResolvedValueOnce({
  supported: true,
  status: "update-available",
  artifactStatus: "verified",
  installReady: true,
})
```

Assert `downloadInstaller()` calls the API and updates `summary.value.artifactStatus`.

- [ ] **Step 2: Run the composable test and verify failure**

Run: `pnpm test -- src/composables/use-app-update.test.ts`

Expected: FAIL because the composable has no download/install methods.

- [ ] **Step 3: Implement frontend API wrappers and composable methods**

Add methods:
- `downloadInstaller()`
- `installUpdate(mode?: "interactive" | "silent")`
- `clearDownloadedInstaller()`

- [ ] **Step 4: Re-run composable tests**

Run: `pnpm test -- src/composables/use-app-update.test.ts`

Expected: PASS.

## Task 5: Settings About UI

**Files:**
- Modify: `src/components/jav-library/settings/SettingsAppUpdateSection.vue`
- Test: `src/components/jav-library/settings/SettingsAppUpdateSection.test.ts`

- [ ] **Step 1: Write failing UI test**

Add tests for:
- update available + not downloaded shows `下载并安装`.
- verified installer shows `立即安装`.
- browser release link remains visible as fallback.

- [ ] **Step 2: Run UI test and verify failure**

Run: `pnpm test -- src/components/jav-library/settings/SettingsAppUpdateSection.test.ts`

Expected: FAIL because UI still only links to the installer URL.

- [ ] **Step 3: Implement UI actions**

Wire buttons to `downloadInstaller()` and `installUpdate("interactive")`. Keep Release page as secondary action when available.

- [ ] **Step 4: Re-run UI test**

Run: `pnpm test -- src/components/jav-library/settings/SettingsAppUpdateSection.test.ts`

Expected: PASS.

## Task 6: Docs And Verification

**Files:**
- Modify: `API.md`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `README.md`

- [ ] **Step 1: Document new API fields and routes**

Update app update sections with `artifactStatus`, download/install endpoints, and unsigned build behavior.

- [ ] **Step 2: Run focused verification**

Run:

```powershell
pnpm test -- src/composables/use-app-update.test.ts src/components/jav-library/settings/SettingsAppUpdateSection.test.ts
cd backend
go test ./internal/storage ./internal/appupdate ./internal/server
```

Expected: PASS.

- [ ] **Step 3: Run broad verification if focused tests pass**

Run:

```powershell
pnpm typecheck
pnpm lint
cd backend
go test ./...
```

Expected: PASS.

## Self Review

- Spec coverage: The plan covers automatic discovery reuse, in-app download, SHA256 verification, explicit install, UI state, and docs.
- Scope intentionally excludes default silent auto-install and real Authenticode signing because no production signing certificate is present.
- No placeholders: all tasks have concrete files, commands, and expected outcomes.

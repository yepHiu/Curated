# Windows Autostart Silent Start Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a default-off Settings -> General switch that persists Windows login autostart and launches Curated silently in tray mode when invoked by Windows autostart.

**Architecture:** Reuse the existing `GET/PATCH /api/settings` -> `config/library-config.cfg` settings pipeline. Add a Windows desktop helper that manages a current-user `Run` registry entry pointing to `curated.exe -mode tray -autostart`, and add an `-autostart` flag so tray mode suppresses only the initial browser launch for OS login startup.

**Tech Stack:** Go backend, Windows registry via `golang.org/x/sys/windows/registry`, Vue 3 settings UI, TypeScript service adapters, Vitest/Go tests, pnpm

---

### Task 1: Backend tests first

**Files:**
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/library_settings.go`
- Modify: `backend/internal/config/library_settings_test.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/server/server_test.go`
- Add: `backend/internal/desktop/autostart_windows.go`
- Add: `backend/internal/desktop/autostart_stub.go`
- Add: `backend/internal/desktop/autostart_windows_test.go`
- Modify: `backend/cmd/curated/main.go`
- Add: `backend/cmd/curated/main_test.go`

- [ ] Add failing config tests proving `launchAtLogin` defaults false and is read from `library-config.cfg`.
- [ ] Add failing server tests proving `GET /api/settings` returns `launchAtLogin` and `PATCH /api/settings` calls the launch-at-login controller.
- [ ] Add failing desktop tests proving the Windows Run command is quoted and includes `-mode tray -autostart`.
- [ ] Add failing command-line test proving autostart tray launches suppress the initial browser open.

### Task 2: Backend implementation

**Files:**
- Same as Task 1

- [ ] Add `LaunchAtLogin bool` to config, settings DTO, and patch request.
- [ ] Add a `LaunchAtLoginController` server interface and wire it through handler deps.
- [ ] Add `App.LaunchAtLogin()` and `App.SetLaunchAtLogin(bool)` that persist the value and sync the OS autostart entry.
- [ ] Add Windows-only desktop helper methods for `LaunchAtLoginSupported`, `SetLaunchAtLogin`, `RemoveLaunchAtLogin`, and command construction.
- [ ] Add non-Windows stubs that report unsupported.
- [ ] Add `-autostart` to `cmd/curated` and pass a `silentInitialBrowser` option into tray startup.

### Task 3: Frontend settings UI

**Files:**
- Modify: `src/api/types.ts`
- Modify: `src/services/contracts/library-service.ts`
- Modify: `src/services/adapters/web/web-library-service.ts`
- Modify: `src/services/adapters/mock/mock-library-service.ts`
- Modify: `src/services/adapters/mock/mock-library-service.test.ts`
- Modify: `src/components/jav-library/SettingsPage.vue`
- Modify: `src/locales/en.json`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/ja.json`

- [ ] Add `launchAtLogin` and `launchAtLoginSupported` to API types.
- [ ] Add `launchAtLogin` and `setLaunchAtLogin()` to the library service contract and adapters.
- [ ] In Settings -> General, add a default-off switch card explaining that autostart is silent and only starts the tray service.
- [ ] Disable the switch with a clear hint in mock/non-supported contexts.
- [ ] Add locale strings for English, Simplified Chinese, and Japanese.

### Task 4: Docs and verification

**Files:**
- Modify: `.cursor/rules/workspace-quick-reference.mdc`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `docs/2026-03-21-library-organize.md`
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `README.ja-JP.md`
- Modify: `CLAUDE.md`

- [ ] Document that `launchAtLogin` is persisted in `config/library-config.cfg`.
- [ ] Document that Windows autostart uses a silent tray launch and does not open the browser on login.
- [ ] Run targeted backend tests for config/server/desktop/cmd packages.
- [ ] Run `pnpm test` for touched frontend service tests.
- [ ] Run `pnpm build`.
- [ ] Run `go test ./...` from `backend/`, using a temp `GOCACHE` if the default local Go cache is permission-blocked.

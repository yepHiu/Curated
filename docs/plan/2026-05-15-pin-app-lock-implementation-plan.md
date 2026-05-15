# PIN App Lock Implementation Plan

日期：2026-05-15

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 Curated 第一版 PIN App Lock：开启后所有敏感 API 与前端页面必须先通过 PIN 解锁，并允许用户在解锁时选择“信任此设备，之后不再要求认证”。

**Architecture:** 采用方案 B：后端保存 PIN hash 与服务端会话，浏览器只持有 HTTP-only session cookie。前端通过 `GET /api/auth/status` 判断锁定状态；未解锁时显示 shadcn-vue 风格锁屏。敏感 `/api` 路径由后端中间件统一拦截，避免只锁前端路由造成绕过。

**Tech Stack:** Go 1.25 + SQLite + Argon2id，Vue 3 + TypeScript + vue-router + shadcn-vue + Tailwind v4。

---

## Status Update - 2026-05-15

- Accepted implementation path remains Plan B: backend-owned PIN verification, SQLite-stored Argon2id hash/salt/KDF, server-side sessions, and an HTTP-only `curated_auth` browser cookie.
- Latest product decision is included: after one successful PIN unlock, a user may choose `trustedForever` for the current device. That session has no `expires_at`, receives a long-lived cookie, and survives restart-lock cleanup until the user locks that device or the session is later revoked.
- Lock screen UX has been revised to match Curated/shadcn-vue settings surfaces: no background photo, no library artwork, no numeric keypad, only PIN cells plus keyboard input, submit, forgotten-PIN help, and the trusted-device checkbox.
- Settings -> Security is the MVP configuration entry for initial PIN setup, PIN change, idle lock delay, LAN PIN policy, backend restart lock policy, and immediate lock. Initial setup and PIN change are entered through buttons that open shadcn-vue dialogs, keeping the settings page itself compact.
- Current MVP includes changing an existing PIN after verifying the current PIN. Disabling PIN, automated PIN recovery key, failed-attempt rate limiting, and connected-client session revocation remain follow-up slices.

## Decisions

- PIN 第一版只接受 4-8 位数字；后端仍按 secret 字符串处理，便于后续扩展 passcode。
- PIN 不存明文。SQLite 保存 `pin_hash`、`pin_salt`、`pin_kdf`。
- 解锁接口支持 `trustedForever: true`。这类会话不设置 `expires_at`，但仍可通过“锁定当前设备”或后续会话管理撤销。
- 非永久会话使用 `session_ttl_minutes` 作为“无操作后锁定”的 idle timeout，默认 60 分钟；用户活动会触发前端刷新 `GET /api/auth/status`，后端验证有效会话时滑动更新 `last_seen_at` 和 `expires_at`，避免用户正在使用 Curated 时被固定倒计时锁住。
- 非永久会话的 `curated_auth` cookie 保持浏览器会话级，不写入固定 `Max-Age` / `Expires`；真正的 idle 截止时间由后端会话表控制。`trustedForever` 会话仍使用长期 cookie 且不设置 `expires_at`。
- `GET /api/auth/status`、`POST /api/auth/setup-pin`、`POST /api/auth/unlock` 对锁定状态开放；其他敏感 API 未解锁时返回 HTTP `423 Locked` + `AUTH_LOCKED`。
- `GET /api/health` 暂时保持开放，后续再单独收敛未解锁时的健康信息字段。
- Mock 模式默认 `pinEnabled=false`，不打断 UI 原型与日常前端演示。
- 忘记 PIN 的 MVP 恢复方式先在 UI 文案说明为“需要在本机重置安全设置”；不在本轮实现自动恢复密钥。

## Backend File Map

- Create `backend/internal/storage/migrations/0024_pin_app_lock.sql`: 建表 `app_security_settings` 与 `auth_sessions`。
- Create `backend/internal/storage/security_repository.go`: 读取/更新 PIN 设置，创建/读取/撤销会话。
- Create `backend/internal/storage/security_repository_test.go`: 覆盖默认设置、PIN hash 不明文、永久信任会话。
- Create `backend/internal/server/auth_handlers.go`: auth DTO handlers 与 cookie 写入/清除。
- Create `backend/internal/server/auth_middleware.go`: 判断敏感路径是否需要解锁。
- Create `backend/internal/server/auth_handlers_test.go`: 覆盖 setup、unlock、lock、middleware、trusted forever。
- Modify `backend/internal/contracts/contracts.go`: 增加 auth DTO 与错误码。
- Modify `backend/internal/server/server.go`: 注册 auth 路由并包裹 auth middleware。

## Frontend File Map

- Modify `src/api/types.ts`: 增加 `AuthStatusDTO`、`SetupPinBody`、`UnlockPinBody`、`ChangePinBody`、`PatchAuthSettingsBody`。
- Modify `src/api/endpoints.ts`: 增加 `authStatus()`、`setupPin()`、`unlockPin()`、`changePin()`、`lockApp()`、`patchAuthSettings()`。
- Create `src/services/auth-lock-service.ts`: 管理 auth 状态、unlock、lock、设置 PIN、修改 PIN。
- Create `src/services/auth-idle-lock-service.ts`: 监听用户活动并按 idle lock delay 刷新 auth 状态；后端返回锁定时跳转 `/lock`。
- Create `src/views/LockView.vue`: shadcn 风格锁屏，只显示 PIN 位、键盘输入、错误和“信任此设备”选项，不显示背景照片和影片信息。
- Modify `src/router/index.ts`: 新增 `/lock` 路由与 Web API auth guard。
- Create `src/components/jav-library/settings/SettingsSecuritySection.vue`: 设置页 Security 分区，包含启用 PIN、修改 PIN、无操作后锁定时长、LAN PIN 开关、立即锁定；启用/修改 PIN 通过入口按钮打开 Dialog 表单。
- Modify `src/components/jav-library/SettingsPage.vue` and `src/lib/settings-nav.ts`: 接入 Security 分区。
- Modify `src/locales/en.json`, `src/locales/zh-CN.json`, `src/locales/ja.json`: 添加锁屏与 Security 设置文案。

## Tasks

### Task 1: Backend Storage

- [x] Write failing tests in `backend/internal/storage/security_repository_test.go`:
  - default settings are disabled with `sessionTtlMinutes=60`
  - `SetPIN` stores a non-empty hash/salt and never stores the raw PIN
  - `CreateAuthSession(... trustedForever=true)` persists a valid session without `expires_at`
  - regular auth sessions extend `expires_at` on activity instead of expiring by a fixed countdown while the user is active
- [x] Run `cd backend && go test ./internal/storage -run "TestSecurity"`.
- [x] Add migration `0024_pin_app_lock.sql` and repository methods:
  - `GetAppSecuritySettings(ctx)`
  - `SetAppPIN(ctx, pin string)`
  - `PatchAppSecuritySettings(ctx, patch)`
  - `CreateAuthSession(ctx, input)`
  - `GetAuthSession(ctx, id)`
  - `RevokeAuthSession(ctx, id)`
- [x] Re-run the storage test command and keep it green.
- [x] Add migration `0025_pin_app_lock_length.sql` so `app_security_settings.pin_length` stores only non-secret PIN length metadata for the lock screen cell count.

### Task 2: Backend Auth API And Middleware

- [x] Write failing tests in `backend/internal/server/auth_handlers_test.go`:
  - `GET /api/auth/status` is unlocked when PIN is disabled
  - `POST /api/auth/setup-pin` rejects mismatched confirmation
  - setup + unlock returns an HTTP-only `curated_auth` cookie
  - regular unlock uses a browser-session-scoped cookie while the server owns the idle deadline
  - unlock with `trustedForever:true` returns `trustedForever=true` and no expiry
  - `POST /api/auth/change-pin` requires an unlocked session, verifies `currentPin`, and updates the stored PIN hash/length
  - `GET /api/library/movies` returns `423 AUTH_LOCKED` when PIN is enabled and no session is present
  - the same request succeeds after unlock
- [x] Run `cd backend && go test ./internal/server -run "TestAuth"`.
- [x] Implement auth handlers, cookie helpers, and middleware.
- [x] Re-run the server auth tests and selected storage tests.

### Task 3: Frontend Auth Service And Lock Route

- [x] Write failing tests for endpoint paths and auth service behavior:
  - `api.authStatus()` calls `/auth/status`
  - `api.unlockPin({ pin, trustedForever:true })` posts the flag
  - `api.changePin({ currentPin, newPin, confirmPin })` posts to `/auth/change-pin`
  - auth guard redirects locked Web API pages to `/lock?redirect=...`
- [x] Run targeted Vitest files for those tests.
- [x] Implement auth DTOs, endpoint methods, `auth-lock-service`, `/lock` route, and route guard.
- [x] Implement `auth-idle-lock-service` so activity refreshes the backend session and idle expiry redirects to `/lock`.
- [x] Re-run targeted tests.

### Task 4: Lock Screen UI

- [x] Write a component test for `LockView.vue`:
  - renders PIN cells but no numeric keypad
  - contains no background image class or `<img>`
  - toggles “trust this device” and submits keyboard-entered PIN
- [x] Implement `LockView.vue` with shadcn-vue primitives and semantic tokens.
- [x] Render lock-screen PIN cells from auth status `pinLength` instead of a hardcoded six-cell layout.
- [x] Re-run the LockView test.

### Task 5: Settings Security Section

- [x] Write a component test for `SettingsSecuritySection.vue`:
  - follows settings `CardHeader` and nested block layout
  - shows enable PIN and change PIN entry buttons without rendering PIN inputs inline
  - opens shadcn-vue Dialog forms for initial PIN setup and current/new PIN change
  - shows idle lock delay and “LAN requires PIN”
  - exposes “Lock Curated now”
- [x] Implement the section and wire it into `SettingsPage.vue`.
- [x] Update Web service and Mock service state as needed.
- [x] Re-run the settings section test.

### Task 6: Docs And Verification

- [x] Update `.cursor/rules/project-facts.mdc`, `README.md`, `API.md`, and `CLAUDE.md` with auth endpoints and behavior.
- [x] Update `docs/plan/2026-05-15-pin-lock-and-lan-access-control.md` with the accepted decisions and implementation status.
- [x] Run targeted backend verification:
  - `cd backend && go test ./internal/storage ./internal/server`
- [x] Run targeted frontend verification:
  - `pnpm test -- src/api/endpoints.validation.test.ts src/router/auth-lock.test.ts src/views/LockView.test.ts src/components/jav-library/settings/SettingsSecuritySection.test.ts`
- [x] If frontend surface changed enough, start or reuse the dev server and visually verify `/lock` and Settings -> Security in the browser.

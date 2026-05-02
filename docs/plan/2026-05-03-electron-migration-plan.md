# Curated Electron Desktop Migration Plan

## 1. Document Purpose

This document defines the migration path from the current **Web phase** (Vue SPA + Go HTTP backend) to the **desktop phase** (Electron shell wrapping the Vue renderer + Go backend sidecar). It covers architecture design, migration phases, work breakdown, and outstanding decisions.

## 2. Current Architecture (Baseline)

### 2.1 What exists today

```
Browser ──HTTP── Go Backend (:8080/:8081) ── SQLite
   │                    │
   └── Vue SPA          ├── REST API (/api/*)
       (Vite dev)       ├── Stdio JSONL mode (unused by any consumer)
                        ├── Native Win32 tray (release builds)
                        ├── Single-instance mutex
                        └── Windows login autostart
```

- **Frontend** uses service contract + adapter pattern. Two adapters exist: `web` (HTTP) and `mock` (in-memory). Selection is build-time via `VITE_USE_WEB_API`.
- **Backend** has four modes: `http`, `stdio`, `both`, `tray`. Stdio mode reads JSONL commands from stdin, writes JSONL responses to stdout — designed for a future host process but has zero consumers today.
- **Desktop features** (tray icon, single-instance, autostart) are implemented in Go using native Win32 APIs, not Electron.
- **Zero Electron code** exists in the repository. No `electron`, `electron-builder`, preload scripts, or IPC bridge code.

### 2.2 What carries forward

| Asset | Fate in Electron |
|-------|-----------------|
| Vue SPA (all views, components, composables) | Reused as-is in Electron renderer |
| Service contract + adapter pattern | Add `electron` adapter; keep `web` for fallback |
| Hash router (`createWebHashHistory`) | Already `file://` compatible — zero changes |
| Go backend (all services, SQLite, scraper) | Runs as child-process sidecar |
| Stdio JSONL protocol | Becomes the Electron main↔backend IPC channel |
| i18n (vue-i18n, en/ja/zh-CN) | Reused as-is |
| Tailwind CSS + shadcn-vue | Reused as-is |

### 2.3 What gets replaced or retired

| Current | Replacement |
|---------|-------------|
| Native Win32 tray (`desktop/tray_windows.go`) | Electron `Tray` API |
| Native Win32 single-instance mutex | Electron `app.requestSingleInstanceLock()` |
| Windows registry autostart | Electron `app.setLoginItemSettings()` |
| Browser opening (`shellopen.OpenURL`) | Electron `BrowserWindow` or `shell.openExternal()` |
| Browser `/api` proxy (Vite dev) | Direct `http://localhost:$PORT/api` from renderer |
| CDN-loaded hls.js | Bundled as dependency or local asset |
| Google Fonts CDN | Bundled locally |
| Python release scripts (partial) | `electron-builder` packaging pipeline |

## 3. Target Architecture

### 3.1 Process model

```
┌─────────────────────────────────────────────────────┐
│                  Electron Main Process               │
│                                                      │
│  ┌──────────┐  ┌────────────┐  ┌─────────────────┐  │
│  │ Tray     │  │ Window mgr │  │ Backend lifecycle│  │
│  │ (Electron│  │ (BrowserWin│  │ (child_process   │  │
│  │  Tray)   │  │  state)    │  │  spawn/kill)     │  │
│  └──────────┘  └────────────┘  └────────┬────────┘  │
│                                          │           │
│  ┌──────────────────────────────────┐    │           │
│  │ IPC Bridge (ipcMain handlers)    │    │ stdin     │
│  │ - Settings / file dialogs        │    │ stdout    │
│  │ - Native player launch           │    │           │
│  │ - Window controls                │    │           │
│  │ - App lifecycle                  │    │           │
│  └──────────┬───────────────────────┘    │           │
│             │ contextBridge              │           │
├─────────────┼────────────────────────────┼───────────┤
│             │    Renderer Process        │           │
│             │                            │           │
│  ┌──────────▼──────────────────────┐     │           │
│  │ Preload (contextBridge)          │     │           │
│  │ - window.curated.* API surface   │     │           │
│  └──────────┬───────────────────────┘     │           │
│             │                              │           │
│  ┌──────────▼──────────────────────┐      │           │
│  │ Vue SPA (existing code)         │      │           │
│  │ - LibraryService (electron adap│      │           │
│  │   calls window.curated.* or     │      │           │
│  │   direct HTTP to localhost)     │      │           │
│  │ - All existing views/components │      │           │
│  └─────────────────────────────────┘      │           │
│                                            │           │
└────────────────────────────────────────────┼───────────┘
                                             │
                    ┌────────────────────────▼──────────┐
                    │  Go Backend (child process)        │
                    │  Mode: "both"                      │
                    │  - HTTP on dynamic port (127.0.0.1)│
                    │  - Stdio JSONL to Electron main    │
                    │  - SQLite, scanner, scraper, etc.  │
                    └───────────────────────────────────┘
```

### 3.2 Communication channels

Two channels between Electron main and Go backend:

1. **HTTP** (port-based, `127.0.0.1:$DYNAMIC_PORT`): The primary data channel. The Vue renderer calls the Go REST API directly via `fetch()`. This preserves 100% of the existing API surface without rewriting ~40 endpoints as IPC bridges.

2. **Stdio JSONL** (stdin/stdout pipe): A control/event channel. Used for:
   - Backend lifecycle (health check, graceful shutdown)
   - Async event push (task progress, scan status, scraper events) — replacing the current polling model
   - File system operations that need main-process mediation (open file dialog, reveal in explorer)
   - Dynamic port negotiation (backend writes chosen port to stdout on startup)

Why keep HTTP as the primary data channel:
- The REST API has ~40 endpoints with full DTO coverage. Rewriting all of them as IPC handlers would be high-effort, high-risk, and offer no user-visible benefit.
- Localhost HTTP from renderer to a sidecar on `127.0.0.1` is a well-established Electron pattern (used by VS Code, Postman, Obsidian, etc.).
- The existing `web` adapter and `src/api/` layer work unchanged.
- Stdio supplements HTTP for events and privileged operations, not as a replacement.

### 3.3 Preload API surface

The preload script exposes a minimal, curated API via `contextBridge`:

```typescript
// window.curated — the only Electron-specific API the renderer sees
interface CuratedBridge {
  // Backend lifecycle
  getBackendPort(): Promise<number>
  waitForBackend(): Promise<boolean>

  // File dialogs (privileged, must go through main process)
  pickDirectory(): Promise<{ path: string } | null>

  // Native actions
  revealInExplorer(filePath: string): Promise<void>
  launchNativePlayer(command: string, args: string[]): Promise<void>

  // App lifecycle
  getAppVersion(): Promise<string>
  onBackendEvent(callback: (event: BackendEvent) => void): () => void  // returns unsubscribe

  // Window controls
  setWindowTitle(title: string): void
}
```

This is intentionally small. The bulk of functionality stays on the HTTP channel. The preload bridge only handles operations that require main-process privileges (file system, child processes, window management).

## 4. Migration Phases

### Phase 0: Foundation (Electron shell with sidecar backend)

**Goal:** A working Electron app that launches the Go backend as a child process and displays the existing Vue SPA in a BrowserWindow, with no feature regressions.

**Key work items:**

1. **Scaffold Electron project**
   - Add `electron` to devDependencies
   - Create `electron/` directory with main process entry (`electron/main.ts`)
   - Create preload script (`electron/preload.ts`)
   - Configure build tooling (electron-vite or manual vite + tsc)
   - Add `"main": "electron/main.js"` to package.json

2. **Main process: backend lifecycle**
   - Spawn Go backend as child process (`curated-dev.exe -mode both`)
   - Read backend port from stdout (add a startup message to stdio mode: `{"kind":"event","type":"server.listening","payload":{"port":8080}}`)
   - Health-check polling until backend is ready
   - Graceful shutdown: send SIGTERM / stdin close on app quit
   - Handle backend crash: show error dialog, offer restart
   - Dynamic port allocation to avoid conflicts

3. **Main process: window management**
   - Create `BrowserWindow` with security-hardened webPreferences
   - Load the built Vue SPA (`file://` protocol from `dist/`)
   - Remember window position/size (electron-store or similar)
   - Handle window-all-closed → quit (or hide to tray, see Phase 3)

4. **Renderer: minimal changes**
   - Introduce `VITE_APP_MODE` env var: `web` | `electron` | `mock`
   - When `electron`: wait for `window.curated.waitForBackend()` before mounting
   - Resolve API base URL dynamically from port
   - Bundle hls.js locally instead of CDN
   - Bundle Google Fonts locally (or use system fonts)
   - Adjust CSP for Electron's `file://` origin

5. **Build pipeline**
   - `pnpm build:electron` — builds frontend + compiles main/preload TypeScript
   - Keep existing `pnpm build` for web-only builds
   - Development workflow: `pnpm dev:electron` (vite dev server + electron)

**Completion criteria:**
- App launches, backend starts, UI renders in Electron window
- Library browsing, movie detail, playback all work
- Settings, scan, scrape all work
- App can be closed cleanly (backend exits, no orphan processes)

### Phase 1: Native Integration (preload bridge)

**Goal:** Replace browser-specific behaviors with Electron-native equivalents where they improve UX.

**Key work items:**

1. **Preload bridge implementation**
   - Implement all `window.curated.*` methods
   - Register `ipcMain` handlers for each bridge method
   - Add TypeScript type declarations for `window.curated`

2. **File dialog integration**
   - Replace browser `<input webkitdirectory>` with Electron `dialog.showOpenDialog()`
   - Wire `window.curated.pickDirectory()` through preload → main → dialog
   - Used by settings page (add library path) and movie import

3. **Native file reveal**
   - Implement `window.curated.revealInExplorer()` via Electron `shell.showItemInFolder()`
   - Replace backend-side `POST /api/library/movies/{id}/reveal` with frontend-native call

4. **Renderer adaptation**
   - Create `electronLibraryService` or extend `webLibraryService` with bridge calls
   - Update `src/lib/pick-directory.ts` to prefer `window.curated.pickDirectory()`
   - Keep web fallback for all bridge-dependent features

**Completion criteria:**
- File picker uses native OS dialog
- "Reveal in Explorer" uses native shell
- All existing features still work in web mode

### Phase 2: Tray & Desktop UX

**Goal:** Replicate and improve upon the current native Win32 tray behavior using Electron APIs.

**Key work items:**

1. **Electron Tray**
   - Implement tray icon using Electron `Tray` + `nativeImage`
   - Context menu: Open, Settings, Logs, Quit
   - Left-click toggles window show/hide
   - Close button minimizes to tray (configurable)
   - Cross-platform: tray works on Windows, macOS, Linux

2. **Single instance lock**
   - `app.requestSingleInstanceLock()` instead of Windows named mutex
   - `second-instance` event: focus existing window
   - Cross-platform by default

3. **Login autostart**
   - `app.setLoginItemSettings()` instead of Windows registry
   - Cross-platform by default

4. **Window enhancements**
   - Frameless window option with custom titlebar
   - Minimize to tray on close
   - Remember maximized/fullscreen state
   - Portable mode: store data next to executable (existing config behavior)

**Completion criteria:**
- Tray behavior matches or exceeds current Win32 tray
- Single-instance works cross-platform
- Login autostart works cross-platform
- Retire `backend/internal/desktop/tray_windows.go`, `single_instance_windows.go`, `autostart_windows.go`

### Phase 3: Player Integration

**Goal:** Improve video playback by leveraging Electron's native capabilities.

**Key work items:**

1. **mpv integration (optional, high-value)**
   - Research feasibility of embedding mpv via `node-mpv` or custom addon
   - Alternative: spawn mpv as child process with IPC control
   - Fallback: continue using Chromium `<video>` (already works)

2. **External player launch (native)**
   - Replace browser protocol handler (`potplayer:{url}`) with `child_process.spawn()`
   - Support mpv, PotPlayer, VLC, IINA presets
   - Configurable command templates

3. **HLS improvements**
   - Bundle hls.js locally (remove CDN dependency)
   - Consider leveraging Electron's Chromium for HLS where native support exists

4. **Gamepad enhancement (future)**
   - Move from Web Gamepad API to `node-hid` for depth features
   - Adaptive triggers, touchpad, LED control (DualSense)
   - This is documented as future scope, not Phase 3 MVP

**Completion criteria:**
- External player launch is native (no browser protocol workaround)
- No CDN dependencies remain in the player
- hls.js bundled locally

### Phase 4: Packaging & Distribution

**Goal:** Produce signed, auto-updating installers for Windows, macOS, and Linux.

**Key work items:**

1. **electron-builder configuration**
   - Windows: NSIS installer + portable zip
   - macOS: DMG + code signing
   - Linux: AppImage + deb
   - Auto-update via `electron-updater` (GitHub Releases)

2. **Build pipeline integration**
   - `pnpm release:electron` — builds frontend, backend, packages all platforms
   - Integrate with existing Python release scripts or replace them
   - Version stamping: build stamp in Go binary + package.json version

3. **Portable mode**
   - Detect `curated-portable` marker file next to executable → store all data locally
   - Respect existing config file locations
   - No registry/AppData contamination in portable mode

4. **Code signing**
   - Windows: EV code signing certificate
   - macOS: Apple Developer notarization
   - CI/CD integration for automated signing

**Completion criteria:**
- Installers available for all three platforms
- Auto-update functional
- Portable mode works

### Phase 5: Polish & Parity

**Goal:** Ensure the Electron app is at least as good as the current tray/web experience in every dimension.

**Key work items:**

1. **First-run experience**
   - Welcome/onboarding flow
   - Library path selection on first launch
   - Default settings sensible for desktop

2. **Performance**
   - Cold start time optimization (backend startup, window paint)
   - Memory baseline monitoring
   - Prevent backend from being a startup bottleneck

3. **Error handling**
   - Backend crash recovery with user-visible diagnostics
   - Database corruption recovery
   - Port conflict resolution

4. **Cross-platform testing**
   - Windows 10/11 (primary)
   - macOS (secondary)
   - Linux (tertiary)
   - Display scaling (HiDPI, fractional scaling)

**Completion criteria:**
- App is shippable as a 1.0 desktop release
- All existing tests pass in Electron context
- No known regressions vs. web mode

## 5. Technical Decisions (Resolved)

| Decision | Rationale |
|----------|-----------|
| Keep HTTP as primary data channel | Preserves ~40 existing API endpoints, avoids months of IPC rewriting, well-established pattern |
| Use stdio JSONL for events/control | Backend already implements it; natural fit for child-process stdin/stdout; avoids SSE/WebSocket complexity |
| Dynamic port allocation | Prevents conflicts with other Curated instances or other software |
| Hash router stays | Already `file://` compatible |
| Electron adapter extends web adapter | Web adapter handles 95% of operations; electron adapter adds bridge-only features (dialogs, reveals, native launch) |
| Bundle hls.js locally | Removes CDN dependency for offline/desktop use |
| Retire native Win32 Go code | Tray, single-instance, autostart all move to Electron APIs for cross-platform support |

## 6. Technical Decisions (Pending)

These need evaluation during implementation:

| Decision | Options | Recommendation |
|----------|---------|---------------|
| Build tooling | (A) electron-vite, (B) Vite for renderer + tsc for main, (C) electron-forge | (A) for integrated DX; (B) if electron-vite is too opinionated |
| Frameless vs native titlebar | Custom titlebar (more design control) vs native (free accessibility) | Start with native; add frameless as enhancement |
| Backend binary location in packaged app | Bundled in `resources/` (ASAR-adjacent) vs `extraResources/` | `extraResources/` to keep Go binary outside ASAR |
| Window state persistence | electron-store, electron-util, or manual JSON file | electron-store (battle-tested, encrypted option) |
| Auto-update provider | electron-updater (GitHub Releases) vs custom | electron-updater (free, well-maintained) |

## 7. Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| Backend crash takes down UI | Medium | Health-check loop in main process; restart backend; show error UI in renderer |
| Port conflict on some machines | Low | Dynamic port (port 0); communicate chosen port via stdout |
| `file://` CSP restrictions break something | Medium | Test early; Electron allows CSP customization; can use custom protocol (`app://`) if needed |
| Cross-platform path differences | Medium | Go backend already uses `filepath`; test on macOS/Linux early |
| Binary size (Go + Electron + Chromium) | Medium | Baseline 150-200MB; acceptable for desktop app; UPX compress Go binary |
| hls.js bundled but license compliance | Low | hls.js is Apache 2.0; include in NOTICE |
| Renderer process can `fetch()` anywhere | Medium | CSP restrict to `127.0.0.1:$PORT` and known CDN domains |

## 8. Estimated Effort

Rough sizing for a single developer familiar with the codebase:

| Phase | Description | Est. Weeks |
|-------|-------------|-----------|
| Phase 0 | Foundation (Electron shell + sidecar) | 2-3 |
| Phase 1 | Native integration (preload bridge) | 1-2 |
| Phase 2 | Tray & desktop UX | 1-2 |
| Phase 3 | Player integration | 1-2 |
| Phase 4 | Packaging & distribution | 2-3 |
| Phase 5 | Polish & parity | 2-3 |
| **Total** | | **9-15 weeks** |

Phase 0 is the critical path; phases 1-3 can be parallelized to some degree.

## 9. Backend Changes Required

Summary of Go backend changes needed:

1. **Stdio protocol extensions** (Phase 0):
   - Emit `server.listening` event with port on startup (for dynamic port)
   - Add graceful shutdown command (`system.shutdown`)
   - Possibly add a few more command types for main-process operations

2. **New mode or mode refinement** (Phase 0):
   - Consider an `electron` mode that is `both` + electron-friendly defaults
   - Auto-choose dynamic port (port 0) unless explicitly configured
   - Suppress console window on Windows (build flag: `-ldflags -H=windowsgui`)

3. **Code to retire** (Phase 2):
   - `backend/internal/desktop/tray_windows.go` → replaced by Electron Tray
   - `backend/internal/desktop/single_instance_windows.go` → replaced by Electron
   - `backend/internal/desktop/autostart_windows.go` → replaced by Electron
   - Keep `backend/internal/desktop/message_windows.go` for fatal error dialogs (useful if Electron fails to start)

4. **Build tags** (Phase 4):
   - New `//go:build electron` tag for Electron-specific behaviors
   - Or detect Electron mode at runtime via `-mode electron`

## 10. Frontend Changes Required

1. **New env var** (Phase 0): `VITE_APP_MODE` = `web` | `electron` | `mock`
2. **New directory** (Phase 0): `electron/` with main, preload, and type declarations
3. **Minimal renderer changes** (Phase 0): dynamic API base URL, bundled fonts, bundled hls.js
4. **New adapter** (Phase 1): `electronLibraryService` extending web adapter with bridge calls
5. **No changes to**: views, components, composables, router, i18n, Tailwind config — all work as-is

## 11. Compatibility Matrix

| Feature | Web (current) | Electron (target) |
|---------|---------------|-------------------|
| Library browsing | Full | Full (same code) |
| Movie detail | Full | Full (same code) |
| Video playback (HTML5) | Full | Full (same code) |
| HLS playback | CDN hls.js | Bundled hls.js |
| External player (PotPlayer/mpv) | Browser protocol | Native spawn |
| File dialogs | Browser `<input>` | Native dialog |
| Reveal in explorer | Backend API | Native shell |
| Tray icon | Go Win32 | Electron Tray |
| Single instance | Go Win32 mutex | Electron API |
| Login autostart | Go Win32 registry | Electron API |
| Settings | Full | Full (same code) |
| Scan/Scrape | Full | Full (same code) |
| Curated frames | Full | Full (same code) |
| i18n | Full | Full (same code) |
| Gamepad (basic) | Web Gamepad API | Web Gamepad API (same) |
| Gamepad (advanced) | N/A | node-hid (future) |

## 12. References

- [Frontend-backend integration gap plan](./2026-03-21-frontend-backend-integration-gap-plan.md) — original Phase 1/Phase 2 strategy
- [Windows tray layer plan](./2026-03-31-windows-tray-layer-plan.md) — current native tray implementation
- [Production packaging strategy](./2026-03-31-production-packaging-and-config-strategy.md) — current release pipeline
- [Player pipeline evolution](./2026-04-01-player-pipeline-evolution-plan.md) — playback architecture
- [PS5 controller integration feasibility](./2026-05-01-ps5-controller-integration-feasibility.md) — gamepad depth features
- [Architecture boundaries](../../.cursor/rules/architecture-boundaries.mdc) — current vs target
- [Project memory](../reference/2026-03-20-project-memory.md) — implementation facts

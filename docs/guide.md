# Curated Handbook

This is the detailed project handbook. Root `README.md` stays a short GitHub entry; this file holds the operational detail that used to live in the README, and it is the index for deeper articles under `docs/`.

Public HTTP API remains in root [`API.md`](../API.md). Folder policy for new documents remains in [`docs/README.md`](README.md).

If this handbook and current code disagree, treat the code as the source of truth.

---

## 1. Product

Curated is a local-first media library: a Vue 3 SPA, a Go + SQLite HTTP backend, and an Electron desktop shell. The product name is **Curated**. The repository folder and npm package may still use `jav-shadcn`. The Go module is `curated-backend`; the server entrypoint is `backend/cmd/curated`.

Current architecture is **web-first plus a minimal desktop shell**. Business APIs stay on HTTP. Electron starts or reuses the Go backend, loads the existing Web UI, hides to tray on window close, and exposes only `window.javLibrary.pickDirectory()`. Deeper IPC, mpv, and broad native bridges remain target-direction work.

Longer product and architecture writing:

- [Product design (current vs target)](product/2026-03-20-jav-libary.md)
- [Feature inventory](features/2026-05-03-feature-inventory.md)
- [Architecture and implementation](reference/architecture-and-implementation.html)
- [Project memory](reference/2026-03-20-project-memory.md)
- [Actor library design](product/2026-03-24-actor-libary.md)

### AI settings and governance

Enable AI in Settings → AI and configure an OpenAI-compatible provider. The backend global enable switch defaults off, including after upgrading from the browser-only experimental switch. Persistent chats and edits require Web API mode and the Go backend. The current reliability implementation and remaining acceptance work are tracked in [the Agent milestone plan, section 14](plan/2026-08-19-agent-milestone-plan.md). The connectivity probe permits up to 1024 output tokens within 30 seconds, so reasoning providers have room to produce a final answer.

Use the Agent button in the top bar to open the right panel. At desktop widths (1024 CSS pixels and above), it shares the content area with the current page; drag the left divider to adjust its width, or focus the divider and use Left/Right (Shift for larger steps), Home, or End. The panel remembers its preferred width, is capped at half the content area, and scrolls independently. On narrower screens it fills the content area; closing it restores the mounted page. Closing also cancels an active response. History, page context, and mentions remain available.

Chat continuation sends the current question. The backend retrieves 80 user/assistant candidates, then retains up to 24 messages within a 24 KiB history budget. Before every model call, JSON UTF-8 byte counts provide a conservative request estimate capped at 65536; these are not measured provider tokens. The newest input is never silently cut. Exceeding the budget asks the user to narrow the request or start a new chat, while keeping completed tool results. Automatic summaries are not implemented.

Reopening a chat shows its latest 80 stored rows. Use Load earlier messages to read older pages without losing newly generated replies. New replies preserve tool evidence, failures, entity choices, and completion outcomes. Older records cannot recover evidence that was never stored. Historical editing previews are read-only; committed receipts mark known successful previews as applied, while other previews require fresh generation before confirmation.

Interrupted replies retain partial text and restore the request to the composer. Chat requests time out after 90 seconds without data; polish, translation, and narrative actions have a two-minute deadline and a Cancel control. Closing the edit dialog, changing its target, or leaving the page cancels generation. Empty terminal answers no longer leave a spinner; tool evidence remains visible. If a note, title, or summary changed after its preview, confirmation returns a conflict and leaves current data intact; regenerate the preview to continue.

Successful confirmed writes and their receipts commit together in SQLite. Retrying an identical confirmation retrieves the original result without repeating the edit, including after a backend restart. Editors reload current data when the result is replayed. The app does not automatically retry writes; unused tickets still expire. Transient confirmation failures retain Retry; expired tickets or write conflicts require a fresh preview. Chat deletion removes that chat's receipts but does not undo its edits. Standalone Action receipts now expire under the configured retention policy. Mock mode simulates previews; durable receipt recovery requires Web API mode.

Tool calls have no default count limit (`aiGovernance.stepLimit: 0`). In Settings → AI, enable **Limit tool calls per turn** to choose 1–30, or turn it off to remove a saved limit. Existing positive settings are preserved on upgrade; changing an old limit to unlimited requires switching it off. Reaching an enabled limit stops before the next model request and displays **Partially completed**, with the specific limit reason and completed evidence retained. Network deadlines, context capacity, cancellation and write confirmation remain independent of this setting.

A write preview ends generation as **Awaiting confirmation**, not Completed. Successful confirmation changes its feedback to **Saved**; discarding it states that no changes were saved. Failed confirmation keeps the confirmation controls available for retry. When reopening history, durable receipts determine whether the changes were saved; an unconfirmed historical preview requires a new request. Older previews previously recorded as Completed are displayed using the same receipt rules. Usage records retain the original generation outcome, so Awaiting confirmation describes where generation stopped rather than the current receipt state. See [the investigation and implemented behavior](plan/2026-09-09-agent-early-completion-analysis.md).

The AI panel provides read-only mode, standard/minimal tool-result redaction, an optional per-turn tool limit (disabled by default; 1–30 when enabled), a global per-minute confirmation limit (1–60), and retention days (7–365, default 30). Changes are enforced by the backend; saving policy stops active generation. Read-only mode blocks write previews and applies, while ordinary queries and Insights remain available. Disabling AI still permits explicit provider connection tests and record inspection. Existing unused confirmation tickets are not restored or granted extra authority. The UI refreshes policy on window focus.

AI policy and model-provider fields save automatically. Switches and policy selections submit immediately; number and text fields save after a 550 ms typing pause or on blur. Each card shows saving, saved, validation or retry feedback. Writes for each group are serialized, failed drafts remain editable, and switching settings sections flushes pending valid edits. Effective AI policy changes only after the server confirms the save. Connection tests and expired-record cleanup remain explicit actions; editing settings does not trigger either. Saving fields does not reload usage or audit pages.

Usage includes chat, inline actions and connection tests, with time/channel/outcome filters and paginated recent requests and tool audit. Each request records provider kind, model, prompt version, duration, first visible text latency, model/tool calls, usage coverage and failure category. Non-streaming first-text latency is unknown. Only provider-reported token counts are summed; absent counts are unknown and incomplete coverage is labelled partially known. This panel does not estimate money or retroactively reconstruct old token counts. Invalid requests rejected before entering the App runtime are not recorded as model runs.

New statistics and tool audit store metadata without raw prompts, arguments, response text, keys, URLs or provider error bodies. Chat content still persists separately. Standard redaction hides paths in remote model tool results; minimal mode also hides titles, summaries and notes from those results. User messages, history context and explicitly submitted translation/polish text still go to the configured model. API keys remain in local configuration, not an OS credential vault.

Expired statistics, tool audit and standalone Action receipts are cleaned when records are queried or a run finishes, or by Clean expired records. Chat messages and chat-linked receipts remain until explicit session deletion. Cleanup never reverses library edits. This is access-triggered cleanup, not a scheduled background sweep; in-progress run statistics are not crash-durable. The new governance endpoints and contracts are documented in [API.md](../API.md#415b-experimental-agent实验性).

To opt into the synthetic real-provider suite, run from `backend/` in PowerShell: set `$env:CURATED_AI_EVAL_SETTINGS` to the absolute path of your configured `library-config.cfg`, then run `go test ./internal/app -run TestAILiveSynthetic -count=1 -v -timeout 20m`. This sends paid requests to that configured provider, using only temporary synthetic library data. It does not read the user's library database or register metadata/source-page tools. Unset the variable afterward with `Remove-Item Env:CURATED_AI_EVAL_SETTINGS`; normal tests skip the live suite. Do not commit configuration files or credentials. See section 14 of the milestone plan for evidence and coverage limits.

---

## 2. Quick start

Requirements: Node.js current LTS compatible with Vite 8, **pnpm**, Go `1.25.4+`.

Agent/build command conventions live in [docs/ops/2026-04-08-agent-build-and-test.md](ops/2026-04-08-agent-build-and-test.md).

### Backend

```bash
cd backend
go run ./cmd/curated
```

Development defaults: `127.0.0.1:8080` (loopback only), health name `curated-dev`.

Windows development helper:

```bash
pnpm backend:build:dev
```

This produces `backend/runtime/curated-dev.exe`. Do not name a development Windows backend `curated.exe`; that name is reserved for release builds.

### Frontend

```bash
pnpm install
pnpm dev
```

Vite usually serves `http://localhost:5173`.

### Real API vs Mock

- Root `.env` with `VITE_USE_WEB_API=true` uses the real backend.
- Any other value keeps Mock mode.
- Local loopback Web API development talks to `http://127.0.0.1:8080` directly; the Vite `/api` proxy remains as fallback.
- Optional `VITE_API_BASE_URL` overrides the API base. Optional `VITE_LOG_LEVEL` sets the default browser log level.

### Electron

```powershell
pnpm dev:electron
```

This builds `backend/runtime/curated-dev.exe` and `electron-dist/`, starts or reuses the Go backend and Vite, then opens the Curated window. Packaged releases install `Curated.exe` as the Electron shell and run the bundled Go backend from `resources/app/curated.exe` on `http://127.0.0.1:8081`.

---

## 3. Backup, restore, and path migration

In Web API mode, Settings → Maintenance can create and verify a backup package, verify an existing package, and run restore preflight. Enter an absolute directory on the backend machine; Curated generates a UTC-timestamped filename. After a successful create, only the directory is remembered as `backupDirectory`. There is no in-app restore button.

Run maintenance from `backend/`. Add `-config path/to/config.json` when the database path comes from a custom main config.

```powershell
go run ./cmd/curated -maintenance backup-create -backup-path C:\Backups\curated.curated-backup
go run ./cmd/curated -maintenance backup-verify -backup-path C:\Backups\curated.curated-backup
go run ./cmd/curated -maintenance backup-preflight -backup-path C:\Backups\curated.curated-backup
```

Restore is offline: fully quit Curated, review a successful preflight, then confirm:

```powershell
go run ./cmd/curated -maintenance backup-restore -backup-path C:\Backups\curated.curated-backup -confirm-restore
```

The package contains a consistent SQLite snapshot and, when present, `library-config.cfg`. It does not include media or user asset files. Verification runs SHA-256 checks plus SQLite `quick_check` / `foreign_key_check`. Restore rejects future migrations and insufficient disk space, replaces files atomically, and keeps `.pre-restore-*` rollback copies.

When a drive letter or library root changes, plan first:

```powershell
go run ./cmd/curated -maintenance path-migrate-plan -path-from D:\Media -path-to E:\Media
```

Apply requires a new backup destination and explicit confirmation:

```powershell
go run ./cmd/curated -maintenance path-migrate-apply -path-from D:\Media -path-to E:\Media -backup-path D:\Backups\before-path-migration.curated-backup -confirm-path-migration
```

Matching is path-segment aware. The whitelist is `library_paths.path`, `movies.location`, `scan_items.path`, `media_assets.local_path`, `actors.avatar_local_path`, `library_path_storage_bindings.root_path`, and `app_update_status.downloaded_file_path`. Conflicts, wrong target types, and missing targets block apply unless `-allow-missing-paths` is used as an explicit override.

---

## 4. Configuration

Runtime config is split between frontend environment variables and backend JSON.

Library-level settings live in `config/library-config.cfg` and are merged on startup. `PATCH /api/settings` writes them atomically. Release packaging copies only the tracked `config/library-config.example.cfg`; the machine-specific file is never shipped.

Common keys:

- `organizeLibrary`
- `metadataMovieProvider` / `metadataMovieStrategy`
- `defaultImportLibraryPathId`
- `backupDirectory` (directory only)
- `autoLibraryWatch`
- `autoActorProfileScrape`
- `autoDownloadUpdates`
- `launchAtLogin`
- `curatedFrameExportFormat` (`jpg` / `webp` / `png`)
- `curatedFrameExportMode` (`raw` / `watermarked`)
- `proxy`
- backend log directory, retention, and level

Empty `logDir` means “use the default log directory”, not “disable file logging”: release uses `LOCALAPPDATA\Curated\logs`, development uses `backend/runtime/logs`.

Development and release builds default to loopback `127.0.0.1:8080` and `127.0.0.1:8081`. A non-loopback `httpAddr` also requires `"lanEnabled": true` and an initialized application PIN. CORS allows same-origin, loopback development origins, and exact `corsAllowedOrigins`.

Library organization details: [docs/reference/2026-03-21-library-organize.md](reference/2026-03-21-library-organize.md).

---

## 5. Features (summary)

The complete shipped/target catalog is [docs/features/2026-05-03-feature-inventory.md](features/2026-05-03-feature-inventory.md). Short groups:

| Area | What is in the current app |
| --- | --- |
| Library | Virtualized poster grid, search, multi-select tags (user + INFO, AND), multi-select actors (AND) and studios (OR), Saved Views, favorites, trash/restore, comments, multi-root paths, fsnotify auto-scan |
| Scan / metadata | Manual and watched scans, metatube scraping, provider strategies and health, auto actor-profile scrape, Library Health and bounded repairs |
| Import | Drag/drop and folder import, resumable chunked upload, restart recovery, storage-presence checks |
| Playback | HTML5 Range streaming, resume, HLS remux/transcode sessions, bundled `hls.js/light`, optional native-player handoff, daily watch-time |
| Actors | Browse, profile, tags, links, avatar cache, scrape, canonical aliases, audited merge |
| Curated frames | Queued captures with preview/retry/undo, source-file frames, cursor browsing, visual similarity review, JPG/WebP/PNG/ZIP export and GIF/MP4/WebM motion |
| Homepage / insights | Daily recommendations with reason codes and local feedback; Personal Insights ranges and actor/studio/tag breakdowns |
| Security | Optional PIN App Lock, HTTP-only sessions, trusted-forever devices, idle lock |
| Desktop | Electron tray shell, Windows installer/portable, FFmpeg bundle, GitHub update check |

In the web player, **D** steps backward and **F** steps forward while pausing playback. Use the fullscreen button to toggle fullscreen. Frame duration comes from stream metadata or media-timestamp measurements; when neither is available, the player uses a 30fps estimate.

### Curated capture and inspection

Use the capture shortcut or the visible Capture button to save the current displayed frame. A small preview shows the captured media time, saving state and final receipt; successive captures share a bounded queue (four retained jobs, 128 MiB admission budget, one upload at a time). A failed item remains available for retry or original download for up to three minutes; successful receipts expire after twelve seconds. Undo removes the library entry, not an already downloaded external file. Directory export failure is reported separately and can be retried.

4K PNG encoding uses a Worker when supported, with a frozen-canvas fallback. Original images remain PNG by default. If an image exceeds the 12 MiB upload limit, explicitly choose a JPEG copy or download the original. Source frame is available in Web API mode and reads the original file at the absolute media time; HLS/browser color conversion and variable frame rates may produce a different image from the displayed stream. It uses the same save receipt after extraction.

Hold the shortcut to record a clip, or use the visible GIF/MP4/WebM action and press it again to finish. Choose the format in the adjacent menu. Media time determines duration; pause ends recording and seek cancels it. Clips are 0.4–6 seconds; the processing panel supports cancellation. GIF uses a palette, while MP4/WebM offer smaller playable files. Cancelling a clip preserves its static frame.

The frame library loads originals only for the current and adjacent detail slides. Use 100% to inspect pixels and Fit image to return. Visual similarity reviews up to the first 200 loaded frames, at most 50 pairs, with two image reads at a time; it does not scan the entire library or delete automatically. Time-nearby badges remain distinct from image similarity. Offscreen card rows unload while retaining layout placeholders and keyboard entry points.

Web frames remain in SQLite; migration 0045 adds stable ordering indexes. Mock IndexedDB upgrades to version 2 and moves full image blobs into a separate store transactionally; frame metadata and directory handles remain available. Measurements and implementation decisions are in [the curated-frame review](plan/2026-04-11-curated-frames-review.md#12-2026-09-06-实施记录与验证).

### Playback recovery

Opening a curated frame passes its requested position into the first playback request. Each HLS playback session is independent, so opening the same movie on another device keeps the first device's session intact. A failed replacement can return to the previous stream until the replacement's first frame data arrives.

HLS network and media failures get bounded recovery attempts before reporting an error; use the play button to retry after recovery is exhausted. An expired session is recreated at the current position. Short seeks wait at most four seconds for encoding to catch up, with the decision based on encoder speed. Startup waits for contiguous downloaded data ahead of the current position, accounting for playback speed.

FFmpeg is paced and suspended/resumed when it runs far ahead of requested segments. Windows controls the actual FFmpeg child (including resolving Scoop shims); supported Unix builds use stop/continue signals. Unsupported process control falls back to paced input. Long event playlists still retain generated segments until session cleanup; per-byte cache quotas and on-demand VOD remain future work. Review and implementation evidence: [playback audit](plan/2026-08-16-playback-pipeline-capability-and-performance-audit.md).

---

## 6. API

Curated exposes a Go HTTP API under `/api`. The full public reference is [`API.md`](../API.md).

Contract and task design:

- [Backend contract constraints](reference/2026-03-21-backend-contract-constraints.md)
- [Go coding standards](reference/2026-03-21-backend-go-standards.md)
- [Task constraints](reference/2026-03-21-backend-task-constraints.md)

Frontend types live in `src/api/types.ts`; HTTP wrappers in `src/api/endpoints.ts`. Views should go through `useLibraryService()`, not concrete Web/Mock adapters.

---

## 7. Repository layout

```text
.
├── src/                    # Vue SPA
├── backend/                # Go module curated-backend
│   ├── cmd/curated/
│   └── internal/
├── config/                 # Library-level runtime config (stays at repo root)
├── docs/                   # This handbook, plus reference / product / ops / plan / prd
├── electron/               # Desktop shell MVP
├── icon/                   # Brand source assets (wordmark / appicon / mark)
└── package.json
```

Root-directory policy:

- `videos_test/` stays at the repo root as local test fixtures.
- `config/` stays at the repo root; do not merge it into `backend/internal/config`.
- `backend/runtime/` is the allowed development runtime output.
- Prefer `.workspace/` for new local scratch.
- Do not create Go build caches inside the repository.

---

## 8. Release and packaging

Official Windows installer and portable packages are published on [GitHub Releases](https://github.com/yepHiu/Curated/releases). Prefer the [latest release](https://github.com/yepHiu/Curated/releases/latest): `Curated-Setup-<version>.exe` for normal installs, `Curated-<version>-windows-x64.zip` for a portable copy. Installed apps can check and download a newer installer from Settings → About.

Recommended packaging entry:

```powershell
pnpm release:publish
```

Production versioning is owned by `scripts/release/version.json`. `pnpm release:*` is orchestrated by `python scripts/release/release_cli.py`. The installed `Curated.exe` is the Electron shell; the Go backend is `resources/app/curated.exe`.

Deeper packaging writing:

- [Production packaging and config strategy](plan/2026-03-31-production-packaging-and-config-strategy.md)
- [Package build history](ops/2026-04-02-package-build-history.md)
- [Package build ledger CSV](ops/package-build-history.csv)
- [Release notes](release-notes/README.md)

---

## 9. Documentation index

Use this table as the citation hub. Dated `docs/plan/*.md` files are working papers; their status rules are in [docs/plan/README.md](plan/README.md). Current requirement status lives in [docs/prd/requirements.csv](prd/requirements.csv).

### Start here

| Document | Use it for |
| --- | --- |
| [README.md](../README.md) | Short public entry |
| [docs/guide.md](guide.md) | This handbook and index |
| [docs/README.md](README.md) | Where new docs belong |
| [API.md](../API.md) | Public HTTP API |
| [Feature inventory](features/2026-05-03-feature-inventory.md) | Shipped vs target feature catalog |
| [Agent build and test](ops/2026-04-08-agent-build-and-test.md) | Install, run, test, and cache rules |

### Product and UX

| Document | Use it for |
| --- | --- |
| [Product design](product/2026-03-20-jav-libary.md) | Domain model, current vs target desktop product |
| [Actor library design](product/2026-03-24-actor-libary.md) | Actor browsing and profile |
| [Frontend UI spec](reference/2026-03-24-frontend-ui-spec.md) | Tokens, surfaces, component rules |
| [Display scaling checklist](reference/frontend-display-scaling-checklist.md) | Desktop density / scaling |

### Architecture and current facts

| Document | Use it for |
| --- | --- |
| [Project memory](reference/2026-03-20-project-memory.md) | Durable implementation facts |
| [Architecture HTML](reference/architecture-and-implementation.html) | Implementation map and feature table |
| [Library organize](reference/2026-03-21-library-organize.md) | `library-config.cfg` and folder organization |
| [Backend Go standards](reference/2026-03-21-backend-go-standards.md) | Backend layout and Go conventions |
| [Backend contracts](reference/2026-03-21-backend-contract-constraints.md) | DTO / error-code / event constraints |
| [Backend tasks](reference/2026-03-21-backend-task-constraints.md) | Async task model |

### Operations

| Document | Use it for |
| --- | --- |
| [Backend notes](ops/2026-03-26-backend.md) | Historical backend notes |
| [Backend improvement](ops/2026-03-26-backend-improvement.md) | Historical improvement notes |
| [Settings page UI notes](ops/2026-03-31-settings-page-uiux-improvements.md) | Settings UX history |
| [Package build history](ops/2026-04-02-package-build-history.md) | How packaging history is recorded |
| [Release notes index](release-notes/README.md) | Published release bodies |

### Requirements and active plans

| Document | Use it for |
| --- | --- |
| [PRD ledger](prd/requirements.csv) | Stable `REQ-xxxx` status |
| [Project overview dashboard](../project-overview-dashboard.html) | Static snapshot of current delivery, active requirements, Git branches, and worktrees |
| [PRD workflow](prd/README.md) | How to add or update requirements |
| [Plan status rules](plan/README.md) | How to read `docs/plan/` |
| [September code review and fixes](plan/2026-09-05-project-code-review.md) | Installer lifetime, Agent proxy/session isolation, frame stepping, and regression evidence |
| [Agent charter](plan/2026-08-18-agent-charter.md) | Constitutional rules for all Agent/AI features (principles, architecture, tool registry, roadmap) |
| [Agent user-facing PRD](plan/2026-08-19-agent-user-prd.md) | Initial user-side Agent requirements (REQ-0029 through REQ-0043) |
| [Agent milestone plan](plan/2026-08-19-agent-milestone-plan.md) | Execution milestones E1-E4 with AI governance and the docked Agent panel |
| [Library filter strengthening](plan/2026-08-14-library-filter-strengthening.md) | Current library filter work (REQ-0026) |
| [Scrape governance](plan/2026-08-14-scrape-governance-implementation-plan.md) | Current scrape-governance work (REQ-0023) |
| [Actor missing-profile auto-scrape](plan/2026-08-16-actor-missing-profile-auto-scrape.md) | Actor backfill scrape (REQ-0027) |
| [README short entry and guide](plan/2026-08-16-readme-short-entry-and-guide.md) | Current public-docs layout (short README + this handbook) |

Do not treat an undated or status-less plan file as approved work. Prefer the PRD ledger and `docs/plan/README.md` when choosing what is current.

---

## 10. Notes

- `docs/film-scanner/` is reference/experimental material, not the production module tree.
- `docs/review/` holds one-off audits; they are snapshots, not living specs.
- Cursor/agent rules in `.cursor/rules/` are operational memory for agents; this handbook is the human-readable map of the same project.

# Curated Handbook

This is the detailed project handbook. Root `README.md` stays a short GitHub entry; this file holds the operational detail that used to live in the README, and it is the index for deeper articles under `docs/`.

Public HTTP API remains in root [`API.md`](../API.md). Folder policy for new documents remains in [`docs/README.md`](README.md).

If this handbook and current code disagree, treat the code as the source of truth.

## Build size monitoring

Builds allow reasonable feature growth. `bundle-policy.json` defines warning levels and generous absolute failure limits for initial JavaScript, all JavaScript, CSS, and the complete frontend output. A spike relative to the latest successful master frontend-quality job or cumulative growth since the reviewed `bundle-baseline.json` produces a warning, not a failure. Only an absolute severe limit blocks a build; named chunks no longer have individual hard caps.

Read `dist/bundle-report.md` after `pnpm build`, or the GitHub Actions job Summary and uploaded bundle-size artifact (retained for 90 days). The JSON report includes the largest files and modules for investigation. CI saves comparison snapshots only from successful master frontend-quality jobs; missing or incompatible snapshots are reported and never disable absolute limits. Limits do not increase automatically. After reviewing a fresh production Web API build, maintainers can explicitly run `pnpm bundle:baseline` and commit the reviewed baseline. This monitors frontend output, including fonts and public assets; Electron, Go, FFmpeg and installer sizes are outside its scope. See [thresholds and comparison rules](ops/2026-04-08-agent-build-and-test.md#4-前端类型检查--lint--测试--构建).

## Repository tracking policy

Git tracks application source, tests, dependency locks, CI/release scripts, required fonts and licenses, shared project rules, and **all documentation and prototypes**. Build output, runtime data, machine-specific configuration and optional local tools stay on the developer's machine. In particular, `backend/frontend-dist/`, `.claude/settings.local.json` and `.agents/` are not part of a fresh checkout. Build the frontend from source when needed; release packaging assembles its own frontend output.

`.gitignore` also excludes local frontend/FFmpeg staging, coverage and Playwright reports, and scratch exports under `output/`. Existing release installers and portable archives remain local and must not be deleted during repository cleanup. CI rejects files that are both tracked and ignored; check with `git ls-files -ci --exclude-standard` (expected: no output). To stop tracking an existing local artifact, use `git rm --cached -- <file>` (or `-r` for a directory); this preserves the current local copy.

This policy changes future commits, not Git history or existing GitHub uploads. Historical blobs remain until a separately agreed history cleanup. The [repository cleanup record](plan/2026-04-24-repository-structure-improvement-plan.zh-CN.md#2026-09-25github-跟踪范围精简) records the applied scope.

---

## 1. Product

Curated is a local-first media library: a Vue 3 SPA, a Go + SQLite HTTP backend, and an Electron desktop shell. The product name is **Curated**. The repository folder and npm package may still use `jav-shadcn`. The Go module is `curated-backend`; the server entrypoint is `backend/cmd/curated`.

Current architecture is **web-first plus a minimal desktop shell**. Business APIs stay on HTTP. Electron starts or reuses the Go backend, loads the existing Web UI, hides to tray on window close, and exposes `window.javLibrary.pickDirectory()` plus a read-only `windowChrome` styling capability. Deeper IPC, mpv, and broad native bridges remain target-direction work.

Longer product and architecture writing:

- [Desktop / Server version sources and naming rules (split packaging pending)](plan/2026-09-25-desktop-server-connection-and-ssdp.md#11-2026-09-26拆分发行命名与版本规范)
- [Product design (current vs target)](product/2026-03-20-jav-libary.md)
- [Feature inventory](features/2026-05-03-feature-inventory.md)
- [Architecture and implementation](reference/architecture-and-implementation.html)
- [Project memory](reference/2026-03-20-project-memory.md)
- [Actor library design](product/2026-03-24-actor-libary.md)

### AI settings and governance

Enable AI in Settings → AI and configure an OpenAI-compatible provider. The backend global enable switch defaults off, including after upgrading from the browser-only experimental switch. Persistent chats and edits require Web API mode and the Go backend. The current reliability implementation and remaining acceptance work are tracked in [the Agent milestone plan, section 14](plan/2026-08-19-agent-milestone-plan.md). The connectivity probe permits up to 1024 output tokens within 30 seconds, so reasoning providers have room to produce a final answer.

In **Configure model**, set **Context window (tokens)** to the total capacity offered by your provider. The default is **65,536**, with whole numbers from **32,768 to 2,097,152** supported. **Reference model preset** fills only this capacity; model ID, API URL and credentials remain independently editable. Presets include DeepSeek, MiniMax, GLM, GPT and Claude, with official source links. A relay or local deployment may expose a smaller capacity, so you can override the number. Valid changes save automatically and apply to the next chat request; Mock stores them in localStorage. The setting does not change provider protocol support.

The chat budget reserves `min(32768, floor(window / 4))` output tokens and `floor(window / 20)` safety units. The remainder is the input limit, charged conservatively at one serialized UTF-8 byte per unit until tokenizer/usage calibration is available. Recent history uses half that input budget; the message cap scales from 24 at 65,536 to at most 200. Summary input is separately bounded, and chat/summary requests send the output reserve as `max_tokens`. This policy covers Agent chat and checkpoints; standalone translation/polish actions and connectivity probes retain their existing limits. Presets and exact formulas are documented in [the context plan](plan/2026-09-21-agent-context-continuity.md#模型容量配置2026-09-21).

Use the Agent button in the top bar to open the right panel. At desktop widths (1024 CSS pixels and above), it shares the content area with the current page; drag the left divider to adjust its width, or focus the divider and use Left/Right (Shift for larger steps), Home, or End. The panel remembers its preferred width, is capped at half the content area, and scrolls independently. On narrower screens it fills the content area; closing it restores the mounted page. Closing also cancels an active response. History, page context, and mentions remain available.

Specific movie answers use references returned by the current request's queries. The server fills catalog codes, titles, actors and requested facts from the same source record, marks local/provider records, and keeps off-library results separate. Free-form movie identifiers and links are checked before publication; an invalid draft gets at most one correction, then ends with a partial result. A selected page ID needs a record read before a card can be displayed. Older `present_movies` free-text reasons are no longer shown. Sources may themselves contain incorrect metadata; this verifies record consistency, not every claim in arbitrary prose or standalone translation/polish actions.

While the model prepares an answer, the panel shows server progress instead of raw model thinking. Checked text appears once ready; cancellation or provider failure discards the unverified draft. SSE sends a heartbeat every 15 seconds, while each model call has a separate two-minute deadline. Published answers can carry source snapshots and a record-count note; reopening history preserves them without granting future tool authority. The chat first-text timing now measures the first published text. See [the implementation and validation record, section 16.6](plan/2026-08-19-agent-milestone-plan.md).

Chat continuation sends the current question. The backend keeps a recent window scaled to the configured capacity (default: up to 24 messages / approximately 22.4 KiB) plus a persistent checkpoint of older goals, constraints and progress. Older conversations are summarized forward from the beginning in bounded batches, including messages outside the latest 200 candidates. Original chat messages remain available in history; deleting a session also removes its checkpoint. Same-session requests are serialized and waiting requests can be cancelled.

During a long task, the Agent automatically organizes working context at 80% of the input budget derived from the configured context window, then continues with the latest input and a complete recent tool batch. The panel shows **Organizing conversation memory**, followed by completion or a partial-memory notice. Recognized provider context-overflow errors get at most one recovery retry. If the current input still cannot fit, shorten that input and continue in the same chat. JSON UTF-8 bytes are a conservative estimate, not measured provider tokens. Summaries are lossy historical clues; exact facts still require current tool reads, and summaries never authorize writes. Summary requests use the configured provider, contribute to measured usage when reported, and may add latency. Persistent checkpoints require Web API mode and migration 0052, applied on normal backend startup. Mock does not generate persistent model memory. See [the source research, implementation and next stages](plan/2026-09-21-agent-context-continuity.md).

Reopening a chat shows its latest 80 stored rows. Use Load earlier messages to read older pages without losing newly generated replies. New replies preserve tool evidence, failures, entity choices, and completion outcomes. Older records cannot recover evidence that was never stored. Historical editing previews are read-only; committed receipts mark known successful previews as applied, while other previews require fresh generation before confirmation.

Interrupted replies retain already-published text and restore the request to the composer; buffered drafts are discarded. Chat requests time out after 90 seconds without data; polish, translation, and narrative actions have a two-minute deadline and a Cancel control. Closing the edit dialog, changing its target, or leaving the page cancels generation. Empty terminal answers no longer leave a spinner; tool evidence remains visible. If a note, title, or summary changed after its preview, confirmation returns a conflict and leaves current data intact; regenerate the preview to continue.

On a movie detail page, the small AI icons beside the title and synopsis translate each field into the current interface language and show the result in place. Click the icon again to restore the original text. These detail-page translations are temporary previews; saving a translated display title or synopsis still uses the edit dialog's confirmation flow. The icons follow AI write-preview availability and are hidden for read-only wishlist details.

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

On macOS, native window buttons sit in a 40px strip above the sidebar brand, with the sidebar background continuing behind them. There is no divider below the brand. In wide windows the content toolbar keeps its original top position and height; narrow windows reserve space for the window buttons. Drag the empty header area to move the window; search and toolbar controls remain interactive. Collapsed navigation reserves space for the system buttons. See [window chrome notes](plan/2026-09-26-macos-integrated-titlebar.md).

```powershell
pnpm dev:electron
```

This builds `backend/runtime/curated-dev.exe` and `electron-dist/`, starts or reuses the Go backend and Vite, then opens the Curated window. Packaged releases install `Curated.exe` as the Electron shell and run the bundled Go backend from `resources/app/curated.exe` on `http://127.0.0.1:8081`.

---

## 3. Backup, restore, and path migration

In Web API mode, Settings → Maintenance & backup can create and verify a backup package, verify an existing package, and run restore preflight. Enter an absolute directory on the backend machine; Curated generates a UTC-timestamped filename. After a successful create, only the directory is remembered as `backupDirectory`. There is no in-app restore button.

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
- `lanEnabled` (Settings → Network & devices; PIN optional; full quit to rebind)
- `curatedFrameExportFormat` (`jpg` / `webp` / `png`)
- `curatedFrameExportMode` (`raw` / `watermarked`)
- `proxy`
- backend log directory, retention, and level

Empty `logDir` means “use the default log directory”, not “disable file logging”: release uses `LOCALAPPDATA\Curated\logs`, development uses `backend/runtime/logs`.

Development and release builds default to loopback `127.0.0.1:8080` and `127.0.0.1:8081`. Settings → Network & devices can persist `"lanEnabled": true` in `library-config.cfg`; the next full restart binds `0.0.0.0` on the same port. A non-loopback `httpAddr` in the main runtime JSON still requires `"lanEnabled": true`. PIN lock is independent of LAN access. CORS allows same-origin, loopback development origins, and exact `corsAllowedOrigins`.

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
| Curated frames | Queued captures with preview/retry/undo, source-file frames, cursor browsing, JPG/WebP/PNG/ZIP export and GIF/MP4/WebM motion |
| Homepage / insights | Daily recommendations with reason codes and local feedback; Personal Insights ranges and actor/studio/tag breakdowns |
| Security | Optional PIN App Lock, HTTP-only sessions, trusted-forever devices, idle lock, Settings LAN access toggle |
| Desktop | Electron tray shell, Windows installer/portable, FFmpeg bundle, GitHub update check |

On the homepage, scroll to the end of the recent, recommendation, and continue-watching sections, then keep scrolling down to open the Movies library. The library slides up over the homepage and starts at the top; the sidebar selection follows the Movies route. The “Continue to Movies” button at the bottom provides the same action for touch and keyboard use. The former Taste Radar section is no longer shown.

In the web player, **D** steps backward and **F** steps forward while pausing playback. Use the fullscreen button to toggle fullscreen. Frame duration comes from stream metadata or media-timestamp measurements; when neither is available, the player uses a 30fps estimate.

### Curated capture and inspection

Use the capture shortcut or the visible Capture button to save the current displayed frame. A small preview shows the captured media time, saving state and final receipt; successive captures share a bounded queue (four retained jobs, 128 MiB admission budget, one upload at a time). A failed item remains available for retry or original download for up to three minutes; successful receipts expire after twelve seconds. Undo removes the library entry, not an already downloaded external file. Directory export failure is reported separately and can be retried.

4K PNG encoding uses a Worker when supported, with a frozen-canvas fallback. Original images remain PNG by default. If an image exceeds the 12 MiB upload limit, explicitly choose a JPEG copy or download the original. Source frame is available in Web API mode and reads the original file at the absolute media time; HLS/browser color conversion and variable frame rates may produce a different image from the displayed stream. It uses the same save receipt after extraction.

Hold the shortcut to record a clip, or use the visible GIF/MP4/WebM action and press it again to finish. Choose the format in the adjacent menu. Media time determines duration; pause ends recording and seek cancels it. Clips are 0.4–6 seconds; the processing panel supports cancellation. If task status cannot be retrieved or processing times out, the panel shows an error and closes automatically; check Curated Frames later because the server task may still finish. GIF uses a palette, while MP4/WebM offer smaller playable files. Cancelling a clip preserves its static frame.

The frame library loads originals only for the current and adjacent detail slides. Use 100% to inspect pixels and Fit image to return. Time-nearby badges can still help locate captures from the same movie for manual review. Offscreen card rows unload while retaining layout placeholders and keyboard entry points.

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

Official Windows installer and portable packages are published on [GitHub Releases](https://github.com/yepHiu/Curated/releases). Prefer the [latest release](https://github.com/yepHiu/Curated/releases/latest): `Curated-Setup-<version>.exe` for normal installs, `Curated-<version>-windows-x64.zip` for a portable copy. Installed apps can check and download a newer installer from Settings → About & updates.

Recommended packaging entry:

```powershell
pnpm release:publish
```

Legacy all-in-one production versioning is owned by `scripts/release/version.json`. Independent component targets live in `scripts/release/versions/{desktop,full}.json` and `backend/internal/version/server.json`; split installers are not yet built by the legacy publish command. `pnpm release:*` is orchestrated by `python scripts/release/release_cli.py`. The installed `Curated.exe` is the Electron shell; the Go backend is `resources/app/curated.exe`.

The packaged frontend includes local HarmonyOS Sans SC, Noto Sans, and Noto Sans JP assets. The full `dist` directory must be shipped so Chinese, English, and Japanese typography remains available offline. Settings → About & updates → Open-source project licenses links to the app's MIT license, all three font licenses, and a local notice file for selected frontend, backend, and desktop components. The [license inventory](plan/2026-09-25-open-source-license-inventory.md) records its scope and FFmpeg build-specific terms.

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
| [Settings information architecture](plan/2026-09-25-settings-information-architecture.md) | Current setting groups, item locations and legacy links |
| [Package build history](ops/2026-04-02-package-build-history.md) | How packaging history is recorded |
| [Release notes index](release-notes/README.md) | Published release bodies |

### Requirements and active plans

| Document | Use it for |
| --- | --- |
| [PRD ledger](prd/requirements.csv) | Stable `REQ-xxxx` status |
| [Project overview dashboard](../project-overview-dashboard.html) | Static snapshot of current delivery, active requirements, Git branches, and worktrees |
| [PRD workflow](prd/README.md) | How to add or update requirements |
| [Plan status rules](plan/README.md) | How to read `docs/plan/` |
| [Wishlist design and implementation plan](plan/2026-09-22-wishlist.md) | Implemented code-only intake, server enrichment, persistent images, import reuse and backup; includes local verification evidence and remaining live-site acceptance |
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

## Comic and photo library Beta

设置页导航现按「使用与功能、资料库、访问与连接、系统」分组；窄屏使用分组选择器。旧的 `?section=experimental` 链接会跳到漫画库，`?section=logging` 跳到维护与备份。已连接客户端位于「网络与设备」；自动下载更新位于「关于与更新」；日志与开发诊断位于「维护与备份」。

漫画库与写真库现以 Beta 合入主线。真实试用时设置 `VITE_USE_WEB_API=true`，运行新版后端及前端，进入 **设置 → 资料库 → 漫画库 / 写真库**：

1. 开启「漫画库 Beta」或「写真库 Beta」。两个开关独立，初始默认关闭。
2. 开启后才显示该库的配置。添加绝对存储路径，通过路径「更多操作 → 扫描漫画 / 扫描写真」扫描 `.zip` / `.cbz`。
3. 从侧栏「漫画 / 写真」打开详情与阅读器。漫画支持导入、阅读进度/偏好、收藏评分标签及独立缓存清理；写真支持 ZIP/CBZ 导入、扫描和浏览。
4. 关闭开关会隐藏该库的内容入口与配置、停止目录监听，保留源包、索引和已保存设置。设置中的 Beta 入口仍可用于再次开启。

开关及路径在 Web 模式下刷新或重启后保留；Mock 开关仅当前会话有效。写真暂未实现单册进度/偏好 API 或独立缓存清理，缩略图使用最长边 420px 的 JPEG，进程内缓存最多 32 MiB；设置中的磁盘缓存上限仍为预留项。支持 ZIP/CBZ，不支持 RAR/CBR/7z。

旧实验 worktree 和数据保留，合并不会搬移其运行数据库或真实媒体。迁移使用 `0046_comic_library.sql`、`0047_photo_library.sql`、`0048_photo_books.sql`、`0049_comic_photo_comments.sql`。详情见 [Beta 整合与验收记录](plan/2026-09-10-comic-photo-beta-integration.md)、[配置说明](reference/2026-03-21-library-organize.md) 与 [API](../API.md#comic-and-photo-library-beta)。

### AI Agent scope for media Beta

漫画库或写真库启用后，Agent 可以查询、展示该库内容，并预览/润色个人笔记。未启用的库仍在范围外。个人洞察、Saved Views、源站检索、萃取帧和评分写工具仍只服务影片，不能把影片统计说成全部媒体。在漫画/写真页面打开 Agent 会附带该页的图册与筛选上下文。详见 [漫画 / 写真接入 Agent](plan/2026-09-12-comic-photo-agent.md)。

## 添加媒体（2026-09-11）

点击顶栏「添加媒体」，在同一个弹窗内选择「添加影片 / 添加漫画 / 添加写真」。影片 Tab 始终显示；漫画、写真 Tab 仅在「设置 → 资料库 → 漫画库 / 写真库」开启对应 Beta 后显示。首次打开默认选择影片。

在各库配置中设置默认导入目录，再选择文件并开始导入。影片保留断点续传与番号重复提示；漫画和写真支持 ZIP/CBZ，复制后自动扫描各自的库。原始文件保留，重名文件不会覆盖。切换 Tab 保留已选文件；上传中暂时无法切换或关闭弹窗。写真部分失败会保留结果提示，不会显示成功并关闭。

## 漫画与写真浏览（2026-09-11）

写真详情页的标签区提供「添加」：输入后点击添加或按 Enter 保存，Esc 取消；保存失败会保留草稿。标签去除首尾空白并去重，最多 64 个，每个最多 64 个 Unicode 字符。Web 模式保存到独立写真 SQLite 表，刷新后仍保留；Mock 模式保存在 `curated-mock-photo-tags` localStorage。保存后同步更新写真库缓存，便于按标签搜索；Agent 没有写真标签写工具。

库页工具栏显示册数，排序通过右侧菜单选择；搜索条件可直接清除，加载失败可以重新加载。漫画墙和写真墙用与影片墙相同的分块虚拟滚动，封面按焦点块决定 eager/lazy。Web 模式会把列表按 500 本一批拉完，因此写真不会停在默认的 100 本。再次进入已加载的漫画/写真页不会重复拉列表；扫描、导入或目录监听完成后仍会刷新。详情封面可直接打开第一页；主按钮显示阅读或从上次页码继续，主区保留页数和添加日期。漫画和写真均可通过右上角三点菜单的“媒体信息”查看源文件名、完整存储位置、文件格式、页数、添加时间及更新时间，并在同一弹窗里修改展示标题；Agent 开启时可预览翻译。路径与页数保持只读。扫描会更新来源文件名标题，但不会清掉已保存的展示标题。漫画保留编辑、打开文件位置、单册删除操作；写真详情仍不显示未接通的单册操作。写真库页提供批量管理：选择可见写真集（最多 100 本），可批量收藏、取消收藏、追加标签或删除索引。删除前需确认，仅移除索引及其附属数据，不删除源 ZIP/CBZ；失败项保留选择以便重试。

图片预览参考影片详情，以图片实际比例决定卡片宽度并自动换行，卡片不再固定为 2:3。每批数量根据实际容器宽度调整，窄屏通常 4 张，桌面自动增加，最多 20 张。使用「上一批 / 下一批」及数字分页查看全书；数字代表预览批次，当前批次高亮，批次较多时显示省略号并保留首末批入口。点击缩略图进入对应页阅读；不再显示页码输入框、定位或打开此页按钮。横图保留完整画面。每批替换旧节点，只加载接近可见区域的图片，失败时点击重试。

写真预览与封面现在使用缩略图，阅读器仍读取原图。过大的或不可解码的图片可能无法生成预览，但不会自动改为下载大原图；可通过主阅读按钮进入阅读器查看原图。漫画封面缓存会按设置的 `maxBytes` 淘汰最久未访问的缩略图；阅读器和写真查看器会预取当前页前后各一页原图。

## 漫画与写真个人评论（2026-09-12）

漫画详情和写真详情在页面预览下方提供与影片页相同的「我的评论」卡片：自动保存，最多 10000 个字符。Web 模式分别写入 `comic_book_comments` / `photo_book_comments`；Mock 模式分别保存在 `curated-mock-comic-comments-v1` 与 `curated-mock-photo-comments-v1`。删除漫画时备注一并清除。AI Agent 仍不能读取、修改或润色这两类备注。

## 漫画与写真本地评分（2026-09-12）

影片、漫画、写真详情都把评分卡放在信息列标签下方，固定 250px 宽。漫画和写真复用同一张本地评分卡：半星步进、综合分、清除；封面比例不固定，所以不放在封面列。只有本地用户分，没有站点分。漫画走已有 `PATCH /api/library/comics/{id}`；写真走 `PATCH /api/library/photos/{id}`。Mock 写真评分保存在 `curated-mock-photo-ratings-v1`。AI Agent 不能读写这两类评分。

## Wishlist

The browser plugin sends a catalog code and, when added from a supported movie page or card, its source page URL. Curated saves the request immediately, then uses the configured metadata provider and proxy to fetch metadata and images in the background. No local video or library path is required.

1. Restart Curated with the updated backend and frontend. Development requires `VITE_USE_WEB_API=true`; Mock mode does not support plugin intake.
2. On the computer running Curated, open **Settings → Network & devices → Browser plugin integration** and turn it on. It defaults to off, persists across restarts, and applies immediately. No token or pairing is required.
3. Reload the extension from `C:/Users/wujiahui/code/curated-plugin/dist` in Chrome. In its settings, set the Curated server address. Development commonly uses port 8080; packaged builds commonly use 8081. Use the actual running address.
4. Click **加入愿望单** on a supported JAVDB or jable page. If extraction is unavailable, enter the code in the extension popup. When integration is off, Curated returns `403 BROWSER_PLUGIN_DISABLED` for wishlist intake and identified plugin API requests; the plugin prompts you to enable it. Existing wishlist entries remain intact.
5. Open **Wishlist** in the sidebar. The default view shows pending imports. Details show a site-name link such as **JAVDB** or **Jable** after the metadata provider; clicking opens that movie’s original page in a new tab. The full URL is not displayed. Older entries and manually entered codes without a source remain unchanged; add an older entry again from its source page to fill in the missing link. Repeated submissions preserve the first saved source and completion status, and metadata refreshes do not replace the source.

Later imports are linked by a unique normalized catalog identity, disappear from the pending view and remain in **In library**. Saved metadata and images are reused before automatic scraping. A periodic background reconciliation recovers delayed changes; unavailable sources or ambiguous matches stay visible for correction. Manual library-link confirmation/exclusion is available through the API; a selection UI is not yet included.

The wishlist uses the movie library's poster grid and shared `DetailPage`/`DetailPanel`, including the existing poster viewer and preview gallery. The sidebar label has no count. Wishlist details display metadata without the old wishlist-specific forms or action blocks; local playback, rating, editing and movie comments are hidden. The shared header returns to the original wishlist filters. The plugin indicator shows a gray dot and **Integration off** when disabled, and stops polling. When enabled, it shows a compact red/green dot with **Offline / Online** text and refreshes through the connected-clients API every 15 seconds while visible. Online means a `Curated-Plugin` request within five minutes, not a persistent socket. Failed checks and Mock mode display offline. Restart the updated backend once to load this implementation; subsequent switch changes apply immediately.

On a wishlist detail page, click **查询** to check Jable and MISSAV for the saved code. The request runs through `GET /api/wishlist/items/{id}/playback` with the normal application session. A confirmed match displays a watch link; a missing match displays **未找到**. Sites that reject automated requests (403/429 or a Cloudflare challenge) display **站点限制自动查询** with a **手动查看** link to that code's page, without claiming the title is available. Timeouts and changed page layouts display **暂时无法确认**. Results are queried on demand and are not saved to the wishlist. Mock mode requires a connected Curated backend for this feature.

Wishlist images live at `<actual database parent>/assets/wishlist`, outside disposable cache. With the default Windows release layout this is `%LOCALAPPDATA%/Curated/data/assets/wishlist`; development defaults to `backend/runtime/assets/wishlist`. Database references are relative. Move this directory together with the database when manually relocating data. New format-v2 backups include wishlist images and accept older format-v1 packages; video files and other user assets still require separate backup. Restore uses the existing offline restore workflow.

The implementation and local test evidence are recorded in the [wishlist plan](plan/2026-09-22-wishlist.md#14-实施结果2026-09-22). Actual live-site extension integration remains to be verified; synthetic fixtures are not live-site evidence.

### Browser plugin developer setting

The plugin has one name (**Curated Plugin**) and one output directory (**dist**). Run `npm run dev` to watch source changes or `npm run build` for a single optimized build from `C:/Users/wujiahui/code/curated-plugin`. Both commands use the same runtime settings.

In plugin settings, enable **开发者模式**, enter **默认服务端地址** such as `http://127.0.0.1:8080`, and save. Disabling the switch uses local port 8081 while retaining the custom address for next time. Existing custom addresses remain valid. No wishlist credential field or saved-token lookup is needed.

Load `dist` in Chrome and reload the extension and source website after code changes. If a previous Dev extension was loaded from `dist-dev`, disable that old entry and use the unified directory. Watch logs are under `.workspace/dev-logs/plugin.out.log` and `plugin.err.log`.


### Desktop version and component release planning

About & Updates shows the local Desktop component version (currently `0.1.0`) with a separate development badge and UTC build timestamp, via the trusted main-process bridge. Desktop and Server appear side by side; both headings use their product names. The Desktop timestamp is captured during `build:electron:main` and persisted in local build metadata, so restarting the app does not change it. Browsers omit this section. One shared Check updates button checks the local legacy installation package and Desktop in parallel only when local updates are allowed. Remote or unverified Desktop connections check Desktop only; Server version information stays visible. `pnpm build:electron:main` generates `electron-dist/desktop-release.json` from the component source and `electron/release-config.json`; restart Electron after changing it. The old npm/app package version remains the legacy installation identity, not the displayed Desktop component version.

Preview a future package name without building, publishing or incrementing versions:

```sh
python3 scripts/release/release_cli.py plan-component --component desktop --platform macos --arch arm64 --format dmg
python3 scripts/release/release_cli.py plan-component --component full --platform windows --arch x64 --format exe
```

Plans carry `status: planned`; Full includes exact component versions. Unsupported combinations and prerelease version suffixes are rejected. Existing `release:publish` remains on the legacy naming, identity and feed.

Desktop checks run in the main process. Development builds return a development status; legacy bundles update through their existing installer. The checked-in config is `distribution: legacy, updateFeed: null`. Once standalone installation identity and migration are implemented, a separately deployed HTTPS feed can use this schema:

```json
{
  "schema": 1,
  "artifacts": [{
    "component": "desktop", "variant": "standalone", "channel": "stable",
    "version": "0.2.0", "platform": "macos", "arch": "arm64", "format": "dmg",
    "url": "https://downloads.example.com/Curated-Desktop-0.2.0-macos-arm64.dmg",
    "sha256": "<64 hexadecimal characters from the actual artifact>"
  }]
}
```

This is a format example, not an active feed. The checker rejects redirects, limits the feed to 1 MiB with a 15-second timeout, and matches component, variant, channel, platform, architecture and installer format before numerical version comparison. No matching artifact, no configured feed, and network errors have distinct statuses. Available updates offer a manual installer link; automatic download, file hash verification, installation and restart are not implemented. Keep split feeds isolated from the legacy GitHub latest endpoint.


### Update target and remote connections

About & Updates allows the legacy Server installer only for a confirmed direct local connection. The actual API target and page must be loopback; Desktop additionally supplies its actual `serverOrigin` through `getDesktopInfo()`, which must match the API origin, and must be a legacy distribution. Standalone Desktop always uses its own update channel. Missing fields in older bridges/servers, bridge errors, LAN addresses and ambiguous proxy setups do not grant local update access. To update Server locally, open its loopback address on that computer with the updated Server and frontend.

All HTTP app-update responses include a request-specific `localUpdateAllowed` boolean and `Cache-Control: no-store`. Server checks the socket peer, Host, optional Origin and proxy headers; download, install and downloaded-installer deletion reject nonlocal requests with HTTP 403 `APP_UPDATE_REMOTE_DISABLED`. Reverse proxies must preserve the external Host or forwarding headers; a proxy that erases all client evidence cannot be distinguished from a direct local process. Do not expose these mutation routes through such a proxy. No remote Server update feature is added.

Remote Desktop hides Server installer actions, the Server auto-download preference, legacy release links/notes and Server update badges. Client-triggered automatic installer downloads use the same restriction, and each mutation rechecks permission. A matching independent Desktop artifact appears as **Download Desktop installer**, with manual installation instructions; it does not close or update Server. Current legacy distributions explicitly state that independent Desktop updates are unavailable. No Full/legacy installer fallback, production Desktop feed or automatic Desktop installer was added.

Restart Server and Desktop after updating their code to enable the new capability fields. The frontend treats older versions as unverified and keeps Server update actions unavailable.

### Server version and build timestamp

`GET /api/health` and stdio `system.health` now return `version: "1.5.7"` (independent Server SemVer), `buildStamp: "YYYYMMDD.HHMMSS"` (UTC), and `channel` separately. About shows the numeric Server version, a separate development badge, and the timestamp below it. The sidebar uses the product version; logs and export metadata retain both identities. `installerVersion` remains the legacy installer version used by its updater.

The Server version source is `backend/internal/version/server.json`, moved from `scripts/release/versions/server.json` so Go can embed the same source during ordinary `go run` / `go build`. The component release planner reads this same file; there is no generated duplicate. Desktop remains `scripts/release/versions/desktop.json` at `0.1.0`. Bump each component only for its own releases.

Release tooling continues injecting `-X curated-backend/internal/version.BuildStamp=<UTC timestamp>`. For direct Go builds without that flag, the existing fallback is Git commit time, then revision / unknown; that fallback is not the actual compilation time. Older clients still receive a string in `version`; consumers that used it as a timestamp must switch to `buildStamp`.

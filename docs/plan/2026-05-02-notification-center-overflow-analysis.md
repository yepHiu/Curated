# Notification Center Overflow Analysis

## Context

The notification center could grow too tall during long-running sessions. When the popover was already open, new task/watch notifications still entered the unread list, so the visible unread area kept accumulating rows and could push through the intended panel boundary.

## Root Cause

1. `use-notification-center.ts` stored notifications newest-first, but `cleanup()` used `slice(-MAX_NOTIFICATIONS)`. That keeps the oldest records when the array is newest-first. It also meant the long-term cap did not preserve the newest notification history.
2. `flush()` only persisted `cleanup(notifications.value)` and did not make the in-memory ref equal to the cleaned queue. During a long session, the UI could keep rendering more than the retention cap until the next reload.
3. `NotificationCenter.vue` called `markAllRead()` only when the popover opened. Notifications arriving while the popover stayed open were still created as unread, so the unread section grew under the user's eyes.
4. The popover content had a fixed width but no max-height or overflow boundary. The main view also reused the 20-item history slice for the compact read preview, so expanding the read section could add too many rows to the same panel.

## Implemented Fix

1. Normalize notification cleanup by sorting newest-first and keeping `slice(0, MAX_NOTIFICATIONS)`.
2. Make `flush()` write the cleaned list back to the in-memory ref before persisting, keeping runtime UI and `localStorage` consistent.
3. Add notification-center open state to the composable. Opening the center marks existing unread items as read, and notifications created while the center is open are inserted as already read.
4. Bound the popover with a max viewport-relative height and `overflow-hidden`.
5. Give the unread/history scroll areas explicit viewport-relative heights.
6. Limit the compact read preview to 5 items while keeping the history view at 20 recent read items.
7. Add regression tests for queue trimming, open-center read behavior, bounded popover classes, and capped read preview rendering.

## Verification

Targeted tests:

```bash
pnpm test -- src/composables/use-notification-center.test.ts
pnpm test -- src/components/notification-center/NotificationCenter.test.ts
```

Both targeted test files pass after the fix.

## Further Optimization Ideas

1. Deduplicate noisy task notifications by `source.taskId` or a `(type, title, source)` signature so repeated polling cannot produce multiple records for the same terminal event.
2. Group bursty filesystem-watch notifications into a single summary row, for example "3 scans completed, 1 scrape completed", instead of writing every low-value event as a row.
3. Add per-notification actions from `source`: jump to task detail, movie detail, update page, or relevant settings section.
4. Separate severity filtering in the UI: all, errors/warnings, scans/scrapes, updates/system.
5. Add an explicit "unread since opened" affordance if product behavior should distinguish "seen because panel is open" from "manually acknowledged".
6. Move notification persistence behind a small storage adapter so future backend/Web API persistence can reuse the same composable contract.
7. Add storage migration/version handling for malformed or old localStorage entries, including strict validation of `type`, `severity`, `read`, `title`, and `message`.
8. Improve accessibility with live-region behavior for critical errors while keeping normal success notifications quiet.
9. Add a compact badge count cap such as `99+` if the unread badge later displays a number instead of a dot.
10. Consider virtualizing the full history list if the retention cap is raised beyond the current 200 records.

## 2026-05-02 Optimization Pass

Implemented after the initial overflow fix:

1. Notification deduplication now updates an existing row when a new notification has the same `source.taskId`. For source-backed non-task notifications, deduplication uses `(type, title, movieId, route)`. For source-less notifications, exact `(type, severity, title, message)` matches collapse into one row.
2. Persisted notification loading now strictly validates `type`, `severity`, `read`, `title`, `message`, and finite `timestamp`. Malformed rows are dropped instead of being rendered into the UI.
3. Notification `source` is sanitized to string-only `taskId`, `movieId`, and `route` fields so old or corrupted localStorage entries cannot leak unexpected shapes into the app.
4. The bell badge now shows a bounded numeric count with `99+` above 99 unread notifications.
5. The popover now includes lightweight filters: all notifications, items needing attention, task notifications, and system/update notifications.
6. Locale keys were added for the filter labels, filtered empty state, and unread count aria label.
7. Regression tests cover deduplication, persisted-data validation, badge capping, and attention filtering.

Still deferred:

1. Burst grouping for filesystem-watch scan/scrape events. The current deduplication lowers repeated-task noise, but multi-task grouping should be designed together with the task/event taxonomy.
2. Per-notification navigation actions. This needs a small UX decision about whether clicking a row dismisses it, opens a route, or exposes explicit row actions.
3. Backend/Web API notification persistence. The local composable remains the current boundary until notifications need cross-device or backend-owned history.
4. Virtualized full history. The retention cap remains 200, so plain rendering is still sufficient.

## 2026-05-10 Current Notification System Review

### Current Implementation Snapshot

- Runtime notification entrypoint is `pushAppToast()` in `src/composables/use-app-toast.ts`. It always shows a `vue-sonner` toast, and only writes a notification-center record when callers pass `options.notification`.
- Notification-center state is local-only in `src/composables/use-notification-center.ts`, persisted under `curated-notification-center-v1`, capped at 200 rows and 7 days.
- The top-bar UI lives in `src/components/notification-center/NotificationCenter.vue` and is mounted by `src/layouts/AppShell.vue`.
- Current notification-center types are narrow: `scan`, `scrape`, `update`, `error`, and `system`.
- Current source metadata is also narrow: `taskId`, `movieId`, and `route`.
- A repository search on 2026-05-10 found about 71 `pushAppToast()` call sites, but only 4 pass `notification`. Those 4 are:
  - manual library scan completion/failure in `use-scan-task-tracker.ts`
  - fsnotify library scan completion in `use-library-watch-toasts.ts`
  - fsnotify-linked movie scrape completion in `use-library-watch-toasts.ts`
  - manual update check when an update is available in `SettingsAppUpdateSection.vue`

Targeted baseline verification:

```bash
pnpm test -- src/composables/use-notification-center.test.ts src/components/notification-center/NotificationCenter.test.ts
```

Result on 2026-05-10: 2 files / 7 tests passed.

### Gaps Worth Fixing

1. **Toast and notification-center coverage are inconsistent.** Most high-value user-visible toasts disappear after a few seconds and are not recoverable from the notification center. The largest missed areas are movie import completion/failure, manual movie/actor scrape results, batch library operations, curated-frame export/delete/tag failures, settings/network results, and playback fallback/errors.

2. **Import notification titles exist but are not wired.** Locale files already include `notificationCenter.titles.importDone` and `importFailed`, but `use-scan-task-tracker.ts` currently shows import terminal toasts without `notification`, and `NotificationType` has no `import` type. This should be either a first-class `import` type or a `system` notification with a stable `route` to Settings -> Library/storage or the library page.

3. **Rows are not actionable.** `NotificationSource.route` exists, but `NotificationCenter.vue` does not navigate. Clicking an unread row currently dismisses it. That makes the stored `route` field underused and makes notification-center history less useful than toast text.

4. **Dismiss and open actions are overloaded.** Opening the popover marks every notification read, and clicking an unread row deletes it. This is simple, but it prevents a user from opening the center to triage errors without acknowledging everything. For low-value success notifications this is acceptable; for warnings/errors, consider explicit acknowledgement or "mark read" separate from "dismiss".

5. **Notification taxonomy is too coarse for filtering.** The current "Tasks" filter only knows `scan` and `scrape`; import, export, playback, network, settings, and library mutations are forced into `system` or omitted. As more events are connected, the center will need either richer `type` values or a separate `category`/`domain` field.

6. **Burst grouping is still missing.** Fsnotify scan and scrape tasks can still generate multiple rows during a folder drop or mass import. Deduplication prevents repeated rows for the same task, but does not combine related terminal events into one user-readable summary.

7. **Auto update checks do not create a notification.** The app can silently discover `update-available` through the scheduled status check and show a sidebar badge, but the notification center only records update availability when the user manually checks from Settings.

8. **No notification preference / suppression layer exists.** Some events should stay toast-only or inline-only. Without a small policy layer, future contributors may either over-notify or continue to skip the center.

9. **Persistence is browser-local only.** This is fine for the current Web SPA boundary, but not enough for backend-owned events that happen while the UI is closed, cross-browser sessions, or future tray/native notification behavior.

10. **Accessibility can improve for critical alerts.** Normal success notifications should stay quiet, but error/warning notifications should be reviewed for screen-reader live-region behavior and keyboard flow through the notification center.

### Recommended Notification Policy

Use three levels instead of "all toast equals notification":

| Level | Behavior | Examples |
|---|---|---|
| `history` | Toast + notification-center row | Completed/failed background work, update available, import terminal result, network/provider verification failure, export failure |
| `toast-only` | Toast, no history row | Copy-to-clipboard, reveal-in-file-manager success, transient player handoff feedback, selection cap warnings |
| `inline-only` | Page-local message, no toast/history | Field validation, disabled/mock-mode hints already visible next to the control |

This keeps the notification center useful as a recoverable activity/error log instead of a full transcript of every UI hint.

### Candidate Notifications To Connect

High priority:

1. **Movie import terminal results.** Add notification-center rows for completed, partial-failed, and failed `import.movies` tasks. Use existing import title locale keys. Include `source.taskId` and a route to the library or Settings -> Library/storage.
2. **Manual movie scrape terminal results.** `LibraryView.vue` currently handles failed scrape tasks with a destructive toast, but successful scrape completion is mostly reflected through cache refresh. Add notification rows for completed/failed `scrape.movie`, with `source.taskId`, `source.movieId`, and a detail route.
3. **Actor scrape terminal results.** `ActorProfileCard.vue` auto-scrape start/done/fail toasts are not persisted. Record done/fail; keep "start" toast-only unless a long-running task view is added.
4. **App update auto-check.** When scheduled update status resolves to `update-available`, add a deduped notification per latest version, routed to Settings -> About.
5. **Provider/proxy verification failures.** Settings network verification results can be useful after a user leaves the page. Persist failed/unverified results; keep successful saves toast-only or low-priority system notifications.

Medium priority:

6. **Curated-frame export failures and batch export results.** Persist failed export attempts and successful batch exports when multiple files are generated. Include route to `curated-frames`.
7. **Curated-frame tag save failures.** Autosave failures currently toast when explicitly requested; persist errors because they may explain why a frame still has old tags.
8. **Batch library operations.** Persist batch scrape/favorite/tag/trash/restore/permanent-delete summaries only when `fail > 0`, or when a long batch finishes after the user may have navigated away.
9. **Playback fallback/errors.** HLS fallback to direct playback, session creation failure, and repeated native-player handoff failure are worth keeping as warning/error rows. Routine "native launch requested" should stay toast-only.
10. **Backend health transitions.** In Web API mode, notify only when the backend transitions from online to offline after initial load, and when it recovers. Avoid notifying on first-load offline unless the current page depends on the API.

Low priority / likely toast-only:

11. Reveal-in-file-manager success.
12. Copy dev performance summary success/failure.
13. Batch selection cap warnings.
14. Successful simple settings saves.
15. Player native-launch requested/still-here hints unless they become repeated failures.

### Suggested Implementation Shape

1. Add a small helper layer around `pushAppToast()` for common notification sources, for example `notifyTaskTerminal()`, `notifySystemResult()`, and `notifyUpdateAvailable()`. Keep the lower-level `pushAppToast()` escape hatch for toast-only cases.
2. Extend notification metadata before broad rollout:
   - either add types: `import`, `export`, `playback`, `network`, `library`
   - or add `domain` while keeping the current `type` set for compatibility
   - add optional source fields such as `actorName`, `frameId`, `version`, and `actionRoute`
3. Make notification rows actionable:
   - row click should navigate when `source.route` exists
   - a separate small close button should dismiss
   - warnings/errors should not be destroyed by accidental row clicks
4. Add dedupe keys for non-task events:
   - update: latest version
   - provider/proxy verification: provider name or proxy target
   - playback fallback: `movieId + mode + reasonCode`
5. Add focused tests as each source is connected:
   - `use-scan-task-tracker.test.ts` for import notification writes
   - `ActorProfileCard.test.ts` for actor scrape done/fail notification options
   - `SettingsAppUpdateSection.test.ts` / `use-app-update.test.ts` for update dedupe behavior
   - `NotificationCenter.test.ts` for row navigation vs dismiss behavior

### Recommended Order

1. Make rows actionable and separate dismiss from navigation. This unlocks value from `source.route` before adding more notification volume.
2. Connect import terminal notifications, because the i18n titles already exist and import tasks are long-running/high-value.
3. Connect manual movie scrape and actor scrape terminal notifications.
4. Connect update auto-check notification with per-version dedupe.
5. Add notification policy helpers and then migrate batch/export/network/playback sources selectively.

## 2026-05-10 Implementation Update

Implemented from the high-priority list:

1. Notification rows are now actionable when `source.route` exists, with a separate dismiss button so opening a source does not delete the row.
2. Movie import terminal tasks now write notification-center records for completed, partial-failed, and failed states. Rows route to Settings -> Library/storage.
3. Automatic and manual update-available checks now write update notifications routed to Settings -> About. Automatic status checks dedupe by latest version or release URL.
4. Actor profile auto-scrape done/fail toasts now persist `scrape` notifications routed to the Actors page search for that actor. The start toast remains toast-only.
5. Single movie metadata refresh tasks are now opt-in notification sources via `useScanTaskTracker().start(taskId, { notifyMovieScrape: true })`. Detail-page refresh and library context-menu refresh opt in; batch refresh stays toast-summary-only to avoid per-movie notification bursts.

Focused verification added or updated:

```bash
pnpm test -- src/components/notification-center/NotificationCenter.test.ts src/composables/use-scan-task-tracker.test.ts
pnpm test -- src/composables/use-app-update.test.ts
pnpm test -- src/components/jav-library/settings/SettingsAppUpdateSection.test.ts
pnpm test -- src/components/jav-library/ActorProfileCard.test.ts
pnpm test -- src/views/DetailView.test.ts
pnpm test -- src/views/LibraryView.test.ts
```

Still deferred after this pass:

1. Provider/proxy verification failures.
2. Curated-frame export/tag-save failures.
3. Batch library operation summaries when failures occur.
4. Playback fallback/error notifications.
5. Backend health transition notifications.
6. A small notification policy/helper layer to keep future call sites consistent.

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

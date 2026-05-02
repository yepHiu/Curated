# Library Watch Notification Copy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace raw backend scan summaries in library-watch toast notifications with localized, user-readable result text.

**Architecture:** Keep the backend task contract unchanged. The frontend library-watch toast layer will derive scan counts from `TaskDTO.metadata` and pass localized count fields into i18n strings, with a fallback for older tasks that only have `message`.

**Tech Stack:** Vue 3 composable, TypeScript, vue-i18n JSON locales, Vitest.

---

### Task 1: Add Regression Test For Fsnotify Scan Toast Copy

**Files:**
- Create: `src/composables/use-library-watch-toasts.test.ts`
- Modify: none

- [ ] **Step 1: Write the failing test**

Create `src/composables/use-library-watch-toasts.test.ts` with a mounted harness that enables `VITE_USE_WEB_API`, returns a completed `scan.library` task with `metadata.trigger = "fsnotify"`, `scanTotal = 24`, `scanImported = 0`, `scanUpdated = 0`, and `scanSkipped = 24`, then asserts:

```ts
expect(mocks.pushAppToast).toHaveBeenCalledWith(
  "toasts.libraryWatchScanDoneNoChanges",
  expect.objectContaining({ variant: "success" }),
)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/composables/use-library-watch-toasts.test.ts`

Expected: FAIL because the current code still calls `toasts.libraryWatchScanDone`.

### Task 2: Implement Localized Scan Summary Selection

**Files:**
- Modify: `src/composables/use-library-watch-toasts.ts`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`

- [ ] **Step 1: Add metadata parsing helpers**

In `src/composables/use-library-watch-toasts.ts`, add a small numeric metadata reader:

```ts
function taskMetaNumber(task: TaskDTO, key: string): number {
  const value = task.metadata?.[key]
  if (typeof value === "number" && Number.isFinite(value)) return value
  if (typeof value === "string") {
    const n = Number(value)
    return Number.isFinite(n) ? n : 0
  }
  return 0
}
```

- [ ] **Step 2: Add localized scan message helper**

Add a helper inside `useLibraryWatchToasts()` that returns `libraryWatchScanDoneNoChanges` when `imported + updated === 0`, otherwise `libraryWatchScanDoneWithChanges`. Use `scanTotal`, `scanImported`, `scanUpdated`, and `scanSkipped` from metadata. If all metadata counters are missing and the backend only provides `task.message`, keep the old generic fallback key.

- [ ] **Step 3: Replace raw message toast call**

Replace:

```ts
const msg = task.message ?? ""
pushAppToast(t("toasts.libraryWatchScanDone", { message: msg }), ...)
```

with the new localized helper result so raw text such as `Scan finished: 24 discovered...` is not shown for modern scan tasks.

- [ ] **Step 4: Add locale keys**

Add these keys to all three locale files:

```json
"libraryWatchScanDoneWithChanges": "...",
"libraryWatchScanDoneNoChanges": "..."
```

Keep `libraryWatchScanDone` as a fallback for older tasks.

- [ ] **Step 5: Run focused test**

Run: `pnpm test -- src/composables/use-library-watch-toasts.test.ts`

Expected: PASS.

### Task 3: Verify Adjacent Locale And Existing Composable Tests

**Files:**
- Test only

- [ ] **Step 1: Run locale parity test**

Run: `pnpm test -- src/i18n/locales.test.ts`

Expected: PASS.

- [ ] **Step 2: Run adjacent tracker test**

Run: `pnpm test -- src/composables/use-scan-task-tracker.test.ts`

Expected: PASS.

- [ ] **Step 3: Review diff**

Run: `git diff -- src/composables/use-library-watch-toasts.ts src/composables/use-library-watch-toasts.test.ts src/locales/zh-CN.json src/locales/en.json src/locales/ja.json docs/plan/2026-05-02-library-watch-notification-copy-plan.md`

Expected: Only the notification-copy plan, regression test, localized helper, and locale messages changed.

# Package History Change Column Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `变更内容` column to the package build history and auto-fill it from raw `git log --oneline` entries between the previous history record and the current build commit.

**Architecture:** Keep the package-history diff logic in a small Node helper under `scripts/release/` so it is easy to test with Vitest and simple for PowerShell release scripts to call. Let `release-common.ps1` stay responsible for writing Markdown rows, while delegating previous-record parsing and git-range summary generation to the helper.

**Tech Stack:** PowerShell release scripts, Node ESM helper scripts, Vitest, Markdown table history file.

---

### Task 1: Add a tested package-history helper

**Files:**
- Create: `scripts/release/package-history.mjs`
- Create: `scripts/release/package-history.d.mts`
- Create: `src/lib/package-history.test.ts`

- [ ] **Step 1: Write the failing test**

Create tests for:
- parsing the previous commit from the latest history row
- returning `首条打包记录，无上一包可比对` when no prior row exists
- returning `无代码差异（同一提交重复打包）` when previous and current commits match
- formatting multiple raw `git log --oneline` lines with `<br>`

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/lib/package-history.test.ts`
Expected: FAIL because `scripts/release/package-history.mjs` does not exist yet.

- [ ] **Step 3: Write minimal implementation**

Implement helper functions to:
- parse the last Markdown history row
- extract the short SHA from the `提交 / 分支` cell
- run or accept raw `git log --oneline` output
- return one of the agreed summary strings or a `<br>`-joined commit list

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/lib/package-history.test.ts`
Expected: PASS

### Task 2: Wire the helper into release history writing

**Files:**
- Modify: `scripts/release/release-common.ps1`

- [ ] **Step 1: Update PowerShell history-writing flow**

Before appending a new row, call the new Node helper to generate the `变更内容` string from:
- repo root
- history file path
- current git SHA

- [ ] **Step 2: Extend the Markdown row layout**

Update `Add-PackageBuildHistoryEntry` so it writes:
- `日期`
- `版本`
- `提交 / 分支`
- `打包类型`
- `产物路径`
- `状态`
- `操作人`
- `变更内容`
- `备注`

- [ ] **Step 3: Verify with targeted tests and static checks**

Run:
- `pnpm test -- src/lib/package-history.test.ts`
- `pnpm typecheck`

Expected: PASS

### Task 3: Migrate the existing history document

**Files:**
- Modify: `docs/2026-04-02-package-build-history.md`

- [ ] **Step 1: Update the table header**

Insert the new `变更内容` column before `备注`.

- [ ] **Step 2: Backfill existing rows**

Set every historical row’s `变更内容` to:

```text
历史记录补齐前未采集
```

- [ ] **Step 3: Verify final formatting**

Run:
- `pnpm lint`
- `pnpm build`

Expected: PASS and the Markdown table remains column-consistent.

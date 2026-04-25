# Python Release Scripts And CSV History Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current PowerShell + Node release tooling with Python-based release tooling, and migrate the package build ledger from Markdown to CSV without changing release behavior.

**Architecture:** Keep `scripts/release/version.json` as the single package-version source and keep Inno Setup as the installer backend. Introduce a Python CLI plus focused Python library modules for versioning, git history, CSV ledger IO, build orchestration, and Markdown-to-CSV migration. Switch `package.json` release commands to Python only after the Python path is verified.

**Tech Stack:** Python 3.14 standard library, pnpm, Go toolchain, Inno Setup (`ISCC.exe`), existing release assets under `scripts/release/windows/`.

---

## File Map

**Create**
- `scripts/release/release_cli.py`
- `scripts/release/release_lib/__init__.py`
- `scripts/release/release_lib/models.py`
- `scripts/release/release_lib/paths.py`
- `scripts/release/release_lib/versioning.py`
- `scripts/release/release_lib/git_utils.py`
- `scripts/release/release_lib/history.py`
- `scripts/release/release_lib/build_steps.py`
- `scripts/release/tests/__init__.py`
- `scripts/release/tests/test_versioning.py`
- `scripts/release/tests/test_history.py`
- `docs/ops/package-build-history.csv`

**Modify**
- `package.json`
- `README.md`
- `README.zh-CN.md`
- `README.ja-JP.md`
- `.cursor/rules/workspace-quick-reference.mdc`
- `.cursor/rules/project-facts.mdc`
- `docs/ops/2026-04-08-agent-build-and-test.md`
- `docs/ops/2026-04-02-package-build-history.md`

**Keep But Stop Using As Primary Entrypoints**
- `scripts/release/*.ps1`
- `scripts/release/*.mjs`

## Task 1: Add failing tests for Python versioning and CSV history behavior

**Files:**
- Create: `scripts/release/tests/test_versioning.py`
- Create: `scripts/release/tests/test_history.py`

- [ ] Write failing tests for:
  - reading `version.json`
  - allocating the next patch version
  - setting major/minor base and resetting patch to `0`
  - extracting the previous commit from CSV history
  - building `change_summary` from a previous and current commit range
  - migrating Markdown history rows into CSV records
- [ ] Run `python -m unittest discover scripts/release/tests -v`
- [ ] Confirm the new tests fail because the Python modules do not exist yet

## Task 2: Implement Python release core utilities

**Files:**
- Create: `scripts/release/release_lib/models.py`
- Create: `scripts/release/release_lib/paths.py`
- Create: `scripts/release/release_lib/versioning.py`
- Create: `scripts/release/release_lib/git_utils.py`
- Create: `scripts/release/release_lib/history.py`
- Create: `scripts/release/release_lib/__init__.py`

- [ ] Implement typed Python data containers for release results and history records
- [ ] Implement repo-root and relative-path helpers matching current PowerShell behavior
- [ ] Implement `version.json` read / allocate / set-base behavior identical to current Node tool
- [ ] Implement git helpers for `rev-parse`, branch lookup, and `git log previous..current`
- [ ] Implement CSV history read/write with UTF-8 BOM and newline-safe `change_summary`
- [ ] Implement one-time Markdown table migration into CSV records
- [ ] Re-run `python -m unittest discover scripts/release/tests -v`
- [ ] Confirm all Python utility tests pass

## Task 3: Implement Python build and packaging steps

**Files:**
- Create: `scripts/release/release_lib/build_steps.py`
- Create: `scripts/release/release_cli.py`

- [ ] Implement frontend build step matching current `build-frontend.ps1`
- [ ] Implement backend release build step matching current `build-backend.ps1`
- [ ] Implement release assembly step matching current `assemble-release.ps1`
- [ ] Implement portable zip packaging with optional history skip
- [ ] Implement installer packaging by rendering `Curated.iss.tpl` and invoking `ISCC.exe`
- [ ] Implement publish orchestration:
  - allocate version once
  - reuse it for frontend, backend, portable, installer, and manifest
  - append exactly one CSV release ledger row
- [ ] Implement CLI subcommands:
  - `show-version`
  - `set-version-base`
  - `build-frontend`
  - `build-backend`
  - `assemble-release`
  - `package-portable`
  - `package-installer`
  - `publish`
  - `migrate-history`

## Task 4: Migrate release history data and switch command entrypoints

**Files:**
- Modify: `package.json`
- Create: `docs/ops/package-build-history.csv`
- Modify: `docs/ops/2026-04-02-package-build-history.md`

- [ ] Run the Python migration command to convert the existing Markdown ledger into CSV
- [ ] Verify the CSV contains all current history rows and headers:
  - `date`
  - `version`
  - `commit`
  - `branch`
  - `build_type`
  - `artifact_paths`
  - `status`
  - `operator`
  - `change_summary`
  - `notes`
- [ ] Update `package.json` release scripts so `pnpm release:*` calls Python instead of PowerShell / Node
- [ ] Reduce the Markdown history file to a pointer document explaining that CSV is now the source of truth

## Task 5: Update docs and project memory

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `README.ja-JP.md`
- Modify: `.cursor/rules/workspace-quick-reference.mdc`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `docs/ops/2026-04-08-agent-build-and-test.md`

- [ ] Replace references to PowerShell / Node release entrypoints with Python release entrypoints
- [ ] Replace references to `docs/ops/2026-04-02-package-build-history.md` as the active ledger with `docs/ops/package-build-history.csv`
- [ ] Keep references to `scripts/release/version.json` unchanged as the single version source
- [ ] Document that the installer still uses Inno Setup under Python orchestration

## Task 6: Verify the full Python release workflow end-to-end

**Files:**
- Modify as needed based on verification findings only

- [ ] Run `python -m unittest discover scripts/release/tests -v`
- [ ] Run `pnpm release:version:show`
- [ ] Run `pnpm release:portable`
- [ ] Run `pnpm release:installer`
- [ ] Run `pnpm release:publish`
- [ ] Verify:
  - the expected artifacts exist under `release/portable`, `release/installer`, and `release/manifest`
  - `scripts/release/version.json` advanced correctly
  - `docs/ops/package-build-history.csv` appended new rows with valid `change_summary`
  - the Markdown ledger file was not appended anymore
- [ ] Run `git diff --stat` and inspect for accidental unrelated changes before closeout

# PRD CSV Bootstrap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap a repo-local PRD workflow with a CSV ledger, operating README, lint script, and `prd-csv` skill so future requirement capture and status tracking can be done consistently in git.

**Architecture:** Keep the PRD source of truth under `docs/prd/` so the product ledger lives with the repository. Add a small Python validator under `scripts/prd/` to lint the CSV contract, and a repo-local `.cursor/skills/prd-csv/` skill that tells Codex how to read, append, refine, and update requirement rows without rewriting the whole file.

**Tech Stack:** Markdown, CSV, Python 3 standard library (`csv`, `argparse`, `pathlib`, `unittest`), repo-local Cursor skills.

---

### Task 1: Create the PRD document skeleton

**Files:**
- Create: `docs/prd/README.md`
- Create: `docs/prd/requirements.csv`
- Modify: `docs/plan/2026-04-21-prd-csv-skill-proposal.md` only if implementation decisions materially differ

- [ ] Define the first-version CSV header with stable product fields: `id,title,type,area,priority,status,progress,source,problem,proposal,acceptance_criteria,dependencies,owner,target_version,implementation_refs,test_refs,detail_doc,updated_at,notes`.
- [ ] Seed `docs/prd/requirements.csv` with one bootstrap row for the PRD workflow itself so the ledger is immediately non-empty and demonstrative.
- [ ] Write `docs/prd/README.md` to document:
  - field meanings
  - allowed status values
  - progress expectations
  - when to create `detail_doc`
  - how Codex should update rows with minimal diffs
  - how to run the lint command

### Task 2: Add a failing test for the PRD lint contract

**Files:**
- Create: `scripts/prd/tests/test_prd_lint.py`
- Test: `python -m unittest scripts.prd.tests.test_prd_lint -v`

- [ ] Write tests that describe the required validator behavior before implementation:
  - accept the seeded `docs/prd/requirements.csv`
  - reject duplicate `id`
  - reject unknown `status`
  - reject `implemented` rows with empty `implementation_refs`
  - reject `verified` rows with empty `test_refs`
  - reject `specified` and later rows with empty `acceptance_criteria`
- [ ] Run `python -m unittest scripts.prd.tests.test_prd_lint -v` and confirm it fails because the lint module does not exist yet.

### Task 3: Implement the minimal PRD lint script

**Files:**
- Create: `scripts/prd/__init__.py`
- Create: `scripts/prd/prd_lint.py`
- Modify: `scripts/prd/tests/test_prd_lint.py`

- [ ] Implement `scripts/prd/prd_lint.py` with a small public surface:
  - `lint_csv(path: Path) -> list[str]`
  - CLI entrypoint returning exit code `0` on success and `1` on validation errors
- [ ] Enforce only the checks covered by the tests; avoid speculative rules in v1.
- [ ] Re-run `python -m unittest scripts.prd.tests.test_prd_lint -v` and confirm green.

### Task 4: Create the repo-local `prd-csv` skill

**Files:**
- Create: `.cursor/skills/prd-csv/SKILL.md`
- Create: `.cursor/skills/prd-csv/agents/openai.yaml`

- [ ] Create a skill that triggers when the user asks to add, refine, modify, triage, or update PRD requirements in this repository.
- [ ] Instruct the skill to:
  - treat `docs/prd/requirements.csv` as the source of truth
  - read `docs/prd/README.md` before editing
  - preserve stable `REQ-xxxx` IDs
  - prefer targeted row/field edits over whole-file rewrites
  - search for likely duplicates before adding a row
  - update `implementation_refs` and `test_refs` as development progresses
  - run the lint command after edits when feasible
- [ ] Keep the skill body concise and repo-specific.

### Task 5: Verify the bootstrap workflow

**Files:**
- Verify: `docs/prd/README.md`
- Verify: `docs/prd/requirements.csv`
- Verify: `scripts/prd/prd_lint.py`
- Verify: `.cursor/skills/prd-csv/SKILL.md`

- [ ] Run `python -m unittest scripts.prd.tests.test_prd_lint -v`.
- [ ] Run `python scripts/prd/prd_lint.py docs/prd/requirements.csv`.
- [ ] If skill validation tooling is available, run `python C:/Users/wujiahui/.codex/skills/.system/skill-creator/scripts/quick_validate.py .cursor/skills/prd-csv`.
- [ ] Check `git status --short` and isolate only the new PRD bootstrap files from unrelated repository changes.


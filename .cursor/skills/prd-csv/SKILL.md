---
name: prd-csv
description: Use when the user asks to add, refine, triage, update, or track product requirements in this repository through `docs/prd/requirements.csv`, including turning loose ideas into structured PRD rows, editing an existing `REQ-xxxx`, updating requirement status or progress, or linking implementation and test evidence back to the PRD ledger.
---

# PRD CSV

Use this repo-local skill to manage Curated requirements in git-tracked CSV form.

## Workflow

1. Read `docs/prd/README.md` before editing.
2. Read `docs/prd/requirements.csv` and identify whether the user request is:
   - a new requirement
   - a refinement of an existing requirement
   - a status or progress update
   - an implementation or verification evidence update
3. Search the existing rows for likely duplicates before creating a new `REQ-xxxx`.
4. Prefer minimal row and field edits. Do not rewrite, sort, or normalize the whole CSV unless the user asks.
5. After editing, run:

```powershell
python scripts/prd/prd_lint.py docs/prd/requirements.csv
```

6. In the response, summarize:
   - which `REQ-xxxx` was added or changed
   - which fields changed
   - any missing information that still blocks moving the row forward

## Row Rules

- Treat `docs/prd/requirements.csv` as the source of truth.
- Preserve stable `REQ-xxxx` IDs. Never reuse or renumber them.
- Keep `title` short and searchable.
- Keep `problem`, `proposal`, and `acceptance_criteria` concise enough to remain scanable in CSV.
- Use `detail_doc` only when the requirement needs long-form context.
- Do not silently remove `acceptance_criteria`, `implementation_refs`, or `test_refs`.

## Status Rules

- `idea` and `triaged` can be lightweight.
- `specified` and later should include `acceptance_criteria`.
- `implemented` should include `implementation_refs`.
- `verified` should include both `implementation_refs` and `test_refs`.
- Use `blocked`, `deferred`, `rejected`, and `superseded` explicitly instead of overloading normal states.

## Common Requests

New requirement:

```text
把这个需求加入 PRD：设置页显示当前安装包版本，并提示是否有新版。
```

Existing requirement edit:

```text
把 REQ-0007 的验收标准改得更清楚一点，补上失败态提示。
```

Implementation progress update:

```text
把 REQ-0007 标成 implemented，并把 docs/plan/2026-04-19-app-update-check-prd.md 写到 implementation_refs。
```

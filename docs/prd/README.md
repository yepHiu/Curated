# PRD CSV Workflow

`docs/prd/requirements.csv` is the repository PRD ledger for Curated. Each row is a stable requirement record that can be refined over time, linked to implementation work, and tracked through release.

## Source Of Truth

- Treat `docs/prd/requirements.csv` as the source of truth for requirement metadata and status.
- Keep one requirement per row.
- Preserve the `id` once a row exists. Titles can evolve; IDs cannot.
- Prefer targeted row and field edits over rewriting or re-sorting the whole file.

## CSV Fields

| Field | Meaning |
| --- | --- |
| `id` | Stable requirement ID such as `REQ-0001` |
| `title` | Searchable one-line summary |
| `type` | `feature`, `bug`, `ux`, `refactor`, `docs`, `ops` |
| `area` | Module or product area |
| `priority` | `P0`, `P1`, `P2`, `P3` |
| `status` | Requirement lifecycle state |
| `progress` | Numeric progress from `0` to `100` |
| `source` | Where the requirement came from |
| `problem` | User or product problem statement |
| `proposal` | Current solution summary |
| `acceptance_criteria` | Semicolon-separated acceptance statements |
| `dependencies` | Related `REQ-xxxx`, docs, or external blockers |
| `owner` | Current owner or driver |
| `target_version` | Planned milestone or release |
| `implementation_refs` | Related plan docs, files, commits, or PR references |
| `test_refs` | Related verification commands, test files, or notes |
| `detail_doc` | Optional long-form requirement doc path |
| `updated_at` | Last update date in `YYYY-MM-DD` |
| `notes` | Short residual notes or state context |

## Status Values

Allowed normal states:

- `idea`
- `triaged`
- `specified`
- `planned`
- `in_progress`
- `implemented`
- `verified`
- `released`

Allowed exceptional states:

- `blocked`
- `deferred`
- `rejected`
- `superseded`

## Status Expectations

- `idea` and `triaged` can be lightweight.
- `specified` and later should have `acceptance_criteria`.
- `implemented` must have `implementation_refs`.
- `verified` must have both `implementation_refs` and `test_refs`.
- `released` should normally keep the evidence already added in earlier states.
- `superseded` should explain the replacement in `notes` or `dependencies`.

## Detail Docs

Create a `detail_doc` only when a row no longer fits comfortably in CSV cells, for example:

- the proposal needs structured subsections
- the acceptance criteria are long enough to hurt scanability
- the requirement needs design tradeoffs or rollout notes

Use `docs/prd/details/REQ-xxxx.md` for new long-form requirement docs unless there is already an existing plan or design document that should remain the canonical detail reference.

## Codex Editing Contract

When Codex updates the PRD:

- read this file first
- read the existing CSV before editing
- search for similar rows before adding a new one
- keep the existing order unless there is a strong reason to change it
- update only the necessary row and fields
- never reuse an existing `REQ-xxxx`
- do not silently remove `acceptance_criteria`, `implementation_refs`, or `test_refs`

## Lint Command

Run the validator from the repository root:

```powershell
python scripts/prd/prd_lint.py docs/prd/requirements.csv
```

For unit tests:

```powershell
python -m unittest discover scripts/prd/tests -v
```

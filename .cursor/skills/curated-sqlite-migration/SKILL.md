---
name: curated-sqlite-migration
description: Add or alter Curated SQLite schema/data migrations safely, including repository changes, integrity checks, and upgrade-path tests.
---

# Curated SQLite Migration

Curated's SQLite database is long-lived user data. Treat every migration as an upgrade compatibility change, not an implementation detail.

## Before editing

Read `backend/internal/storage/sqlite.go`, the existing migration sequence, nearby repository tests, and the affected domain contract. Check migration filenames and numeric prefixes for collisions; the executor applies embedded files by name order. Define the data invariant, compatibility expectation, and whether existing rows need a backfill.

Prefer a focused migration plus repository/service changes. Do not use ad-hoc startup writes or silently repair unrelated historical data. New constraints and foreign keys must tolerate the real upgrade sequence, not only an empty database.

## Required safeguards

- Keep schema and data changes transactional where SQLite permits.
- Preserve `PRAGMA foreign_keys=ON`; validate meaningful changes with `foreign_key_check` and, where appropriate, `quick_check`.
- Test fresh initialization and an upgrade fixture/state that represents pre-migration data, including failure or conflicting data when relevant.
- Do not hand-edit a user's runtime database to test a feature. Do not direct Go caches into the repository.
- If the migration changes backup, restore, maintenance, or path behavior, invoke `curated-maintenance-safety` and update its affected contracts/docs.

## Verification and reporting

Run targeted storage tests and `cd backend && go test ./...`; add `go vet ./...` for non-trivial Go changes. State the invariant, migration filename/order, upgrade coverage, integrity checks, and rollback/recovery story. Synchronize durable schema facts only when the feature surface requires it.

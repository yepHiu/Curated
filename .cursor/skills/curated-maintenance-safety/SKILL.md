---
name: curated-maintenance-safety
description: Safely inspect or implement Curated backup, restore, path migration, Library Health repair, or cleanup workflows involving persistent user data.
---

# Curated Maintenance Safety

Use this Skill whenever an operation can alter, delete, restore, or relocate persistent library data.

## Safety order

1. Start with a read-only status, scan, verify, preflight, or plan operation.
2. Present the resolved targets, affected count/sample, conflicts, and blocking condition before any material mutation.
3. Require the existing explicit confirmation mechanism for mutation; do not invent implicit confirmation from conversational context.
4. Preserve the runtime lock and backup requirements. Never bypass a lock by deleting `.runtime.lock`.
5. Revalidate at mutation time, limit the action to an allowlist, write/retain audit evidence, and report the outcome.

## Curated-specific invariants

- Backup destinations must be new; never overwrite a requested target.
- Restore and path migration run only after Curated is fully stopped and the maintenance command obtains the database lock.
- Path migration must run plan before apply; missing targets require the existing explicit override and conflicts/wrong target types remain blockers.
- Library Health actions only handle freshly revalidated, whitelisted findings. Final movie files are never cleanup candidates.
- Maintain integrity checks and transaction/rollback behavior for all data-changing flows.

Read the corresponding implementation, `workspace-quick-reference.mdc`, and `docs/ops/2026-04-08-agent-build-and-test.md` before use. Run the narrowest safe verification first. Do not execute a destructive operation merely to demonstrate the workflow; stop for user direction when the target, confirmation, or impact is unclear.

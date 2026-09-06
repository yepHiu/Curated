---
name: curated-project-dashboard
description: Refresh Curated's static project overview dashboard from the PRD, implementation facts, Git history, branches, and worktrees. Use when updating the recurring project-status HTML, not for implementing product features.
---

# Curated Project Dashboard

Use this Skill to refresh the root-level `project-overview-dashboard.html` when someone asks for an updated project overview, delivery status, Git/branch summary, or worktree inventory.

The dashboard is a **static, timestamped snapshot**. It must say when it was collected and link back to its live sources; it is never a replacement for the PRD, source code, or Git.

## Gather authoritative inputs first

1. Read `docs/plan/README.md`, `docs/prd/requirements.csv`, `.cursor/rules/project-facts.mdc`, and `.cursor/rules/architecture-boundaries.mdc`.
2. Run the read-only collector from the repository root:

   ```powershell
   powershell -ExecutionPolicy Bypass -File .cursor/skills/curated-project-dashboard/scripts/collect-dashboard-snapshot.ps1
   ```

3. Treat `requirements.csv` as the only source for requirement status and progress. A plan file is historical unless the plan status rules and PRD support calling it active.
4. Treat the current code and `project-facts.mdc` as implementation truth. Keep implemented, target, and deliberately deferred capabilities separate.
5. Read the current dashboard before editing so its existing presentation structure, sources, and any user-authored changes are preserved.

## Update the snapshot deliberately

- Update every affected count, percentage, requirement row, branch/HEAD value, version, worktree row, dirty-state description, recent commit, and timestamp as one consistent snapshot. Do not update a headline while leaving stale detail below it.
- Keep the existing dashboard sections: requirements, current work, delivered capabilities, not-now boundaries, Git/worktree snapshot, and sources. Use clear Chinese copy and retain the accessible, responsive, tokenized single-file HTML style.
- Report unavailable information explicitly (for example, an unreadable worktree or missing upstream) instead of guessing.
- If the dashboard's source snapshot disagrees with a live command, prefer the live command for Git/worktree facts. If a PRD row and an implementation note disagree, report the distinction rather than silently promoting a feature to delivered.
- Do not turn an exploratory worktree into an approved roadmap item. State its actual branch, dirtiness, and relationship to the PRD execution stream.
- Preserve the statement that the dashboard is static. If an edit or its own commit changes the live Git state after collection, word the page as a capture-time snapshot rather than claiming live status.

## Safety and completion

- Begin with `git status --short --branch`. Never reset, clean, stash, switch branches, or alter unrelated worktree files.
- If the dashboard itself has conflicting uncommitted edits, stop and ask for direction rather than overwriting them. Otherwise use `apply_patch` for the HTML and only update `docs/guide.md` when its dashboard link changes.
- Validate with `git diff --check -- project-overview-dashboard.html docs/guide.md README.md` and confirm the rendered content contains all six dashboard sections. No frontend build is required for a documentation-only snapshot.
- When the repository commit workflow applies, stage only the dashboard and any intentionally changed index file, inspect the staged diff, then create one documentation-only commit. Never push unless the user explicitly asks.
- Finish by linking the dashboard, giving its snapshot timestamp, identifying the sources refreshed, and disclosing any unavailable or stale-after-commit Git information.

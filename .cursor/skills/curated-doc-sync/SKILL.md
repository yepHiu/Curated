---
name: curated-doc-sync
description: Synchronize Curated implementation changes with the correct repository documentation, API reference, architecture facts, plans, and release notes.
---

# Curated Documentation Sync

Use this Skill after changing a material endpoint, architecture fact, configuration behavior, public workflow, or user-visible capability.

## Let impact choose documents

Read `.cursor/rules/docs-sync.mdc`, `project-facts.mdc`, and the changed code. Code is authoritative when docs disagree. Update only documents whose claims change:

- public API: `API.md`, `CLAUDE.md`, and contract/project facts;
- architecture or important endpoints: `.cursor/rules/project-facts.mdc`, `docs/reference/architecture-and-implementation.html`, root README short entry, and `docs/guide.md` as applicable;
- configuration / `library-config.cfg`: `docs/reference/2026-03-21-library-organize.md`, guide, and examples as applicable;
- UI system: UI spec and UI rule; plans retain rationale, not duplicated implementation facts;
- packaging: use `curated-packaging` and write successful release notes under `docs/release-notes/`.

Keep root README short. Put plans in `docs/plan/` with dated names and explicitly separate implemented facts, target design, and unresolved decisions. Preserve translations when public README content changes.

Finish with a document impact list, including consciously unchanged documents and why. Do not create ceremonial docs for a code-only change with no durable user or operational effect.

---
name: curated-api-contract-change
description: Change Curated HTTP APIs, DTOs, stable errors, or task contracts while keeping Go services, frontend contracts, Web/Mock adapters, and public docs aligned.
---

# Curated API Contract Change

Use this Skill for public REST paths, request/response DTOs, error codes, task states, and transport-facing contract changes.

## Contract first

Read `.cursor/rules/backend-api-contracts.mdc`, `project-facts.mdc`, `API.md`, `CLAUDE.md`, and the closest contract. Define frontend-consumable DTOs and stable error codes before handler wiring. Keep contracts transport-agnostic: HTTP is the current adapter, not permission to couple renderers to storage or filesystems.

For asynchronous work, return a task-oriented result and expose status/progress rather than holding a synchronous request open. Validate inputs at the boundary and avoid returning raw database rows or provider payloads.

## Change matrix

Trace each changed field and behavior through:

- `backend/internal/contracts`, service/app, handler, storage as applicable, and Go tests;
- frontend types and `LibraryService` contract;
- Web adapter and Mock adapter behavior;
- callers, loading/error states, and user copy;
- `API.md`, `CLAUDE.md`, `.cursor/rules/project-facts.mdc`, and other docs required by the repository's documentation rule.

Maintain backwards compatibility when an existing client may still send an old supported form. If a breaking change is necessary, identify migration/rollout consequences before editing and do not silently alter wire semantics.

## Verification

Exercise valid, invalid, authorization/lock, and task/error paths relevant to the contract. Run focused frontend tests plus `cd backend && go test ./...` for backend changes; use `curated-regression-selector` for the remaining checks. Summarize the contract delta and synced consumers.

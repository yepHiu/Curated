---
name: curated-feature-slice
description: Implement a user-visible Curated feature across contracts, adapters, UI, tests, and documentation. Use for new or materially extended product behavior, not isolated internal refactors.
---

# Curated Feature Slice

Deliver a vertical product slice rather than a partial backend or UI-only change.

## Map the impact before editing

Read `workspace-quick-reference.mdc`, `project-facts.mdc`, the relevant frontend/backend rules, and the closest existing feature. List only applicable layers:

`contracts → app/service → HTTP → frontend service contract → Web adapter → Mock adapter → composable/UI → i18n/messages → tests → docs`.

Define contracts, stable error codes, and task/progress behavior before wiring transport or a view. Long-running work must not block a request. Views and components use `useLibraryService()`; they never import concrete adapters to bypass the service layer.

## Preserve runtime parity deliberately

For every user action, decide whether Web API and Mock both support it. Keep equivalent behavior where feasible. If Mock cannot honestly emulate an external/runtime capability, retain a clear unavailable state instead of fabricating a successful result. Preserve the documented localStorage persistence for mock favorites and ratings.

Keep user-visible copy localizable. For new user messages, follow `curated-localization-message`; for material UI design work, use `curated-ui-governance` before implementation.

## Verify and close

Use `curated-regression-selector` to choose proportionate checks. Cover changed business logic and at least the meaningful success/failure boundary; add browser E2E only for an end-user flow or regression that unit tests cannot establish.

Report the completed layer matrix, intentional omissions, verification run, and documentation updated. Important endpoints, architecture, configuration, or public behavior require the appropriate documentation synchronization.

# About Installer Version Display

Date: 2026-04-12

## Goal

Show both of these values in Settings -> About:

- application build version
- installer/package version

The user-visible requirement is that the production package can show the installer version number explicitly in the About section.

## Current state

- About currently shows only backend health `version` plus `channel`.
- Backend health `version` is the build stamp (`YYYYMMDD.HHMMSS`) or fallback build metadata.
- Release packaging already has a single authoritative package version source: `scripts/release/version.json`.
- Backend release builds currently do not embed that package version into the binary.

## Decision

Use a two-value model in About:

1. `version` remains the application build identifier.
2. Add a new health field for installer/package version, sourced from the release packaging version injected into the backend binary at build time.

## Design

### Backend

- Add a new version variable in `backend/internal/version`.
- Release build script injects the package version with `-X`.
- Health DTO adds a new optional field, exposed by both:
  - HTTP `GET /api/health`
  - stdio system health command

### Frontend

- Extend `HealthDTO` in `src/api/types.ts`.
- Extract About version formatting into a small helper for testability.
- Settings -> About shows:
  - App version
  - Installer version
- In dev / mock / missing-value scenarios:
  - keep existing app-version fallback behavior
  - hide or degrade installer-version display cleanly instead of showing bogus values

## Naming

- API field: `installerVersion`
- UI label: installer/package version wording per locale

## Test plan

- Backend test:
  - system health response includes `installerVersion` when embedded
- Frontend test:
  - About formatting distinguishes app version and installer version
  - missing installer version degrades cleanly

## Scope limits

- Do not redesign the About page layout.
- Do not change package version allocation rules.
- Do not derive version separately in frontend from release files.

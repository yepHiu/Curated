---
name: curated-release-readiness
description: Perform a read-only Curated release-readiness gate before production packaging, checking versioning, validation, documentation, and runtime prerequisites.
---

# Curated Release Readiness

Use this Skill before a real portable/installer publish request. It is a gate before—not a replacement for—`curated-packaging`.

## Read-only readiness review

Read `scripts/release/version.json`, `docs/ops/package-build-history.csv`, recent release notes, `package.json` scripts, and the current working-tree/git status. Determine the requested release scope and whether the change set has release-note material. Do not delete or mutate versioned artifacts while inspecting readiness.

Confirm the planned package flow uses the repository release CLI and its automatic version policy. `scripts/release/version.json` is authoritative; package.json is not. Major/minor base changes require the existing explicit release command. Existing installer and portable artifacts must remain untouched.

## Gate checklist

- appropriate frontend, Electron, Go, release-script, and E2E checks have passed or their risk is explicitly recorded;
- release notes will be created in `docs/release-notes/` after successful packaging, following the repository format;
- sanitized `config/library-config.example.cfg`, Electron runtime layout, and a real non-shim FFmpeg runtime can be assembled by the existing release flow;
- package/history implications and any version bump are understood;
- signing/checksum/manifest inspection requirements are identified for the target environment.

Report ready/not-ready with evidence and exact blockers. Only after the user authorizes packaging and the readiness result is clear, hand off to `curated-packaging`, which previews before execution.

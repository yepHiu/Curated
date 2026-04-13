---
name: curated-packaging
description: Use when the user asks in natural language to preview or run Curated packaging workflows in this repository, including production, installer, portable, or version-base changes before packaging.
license: MIT
metadata:
  repo: Curated
  version: "0.1.0"
  previewScript: .cursor/skills/curated-packaging/scripts/preview-package.ps1
  executeScript: .cursor/skills/curated-packaging/scripts/execute-package.ps1
  packagingCommands:
    - pnpm release:portable
    - pnpm release:installer
    - pnpm release:publish
    - pnpm release:version:set-base
---

This skill orchestrates the repo-local packaging workflow for Curated.

Use this skill only inside this repository. Do not treat it as a global skill or publish it outside the repo.

## Request Handling

- Map natural-language packaging requests to one of: `publish`, `installer`, `portable`, `preview`, `set-base`.
- Respect the repository release rules:
  - `scripts/release/version.json` is the only automatic production version source.
  - `publish` allocates one version once and reuses it for all generated artifacts.
  - `major` and `minor` changes are manual base changes; `patch` is the auto-bump part.
- Always preview before execution.
- If the request includes a base change before packaging, preview the base change first and then the resulting packaging version.

## Preview

- Use `scripts/preview-package.ps1` for the preview output.
- Preview must include:
  - detected mode
  - current base version
  - base version after any requested change
  - predicted version
  - whether patch will bump

## Execution

- Execute only after the preview has been shown to the user.
- Use `scripts/execute-package.ps1` for execution.
- Reuse the repository packaging commands instead of duplicating release logic:
  - `pnpm release:publish`
  - `pnpm release:installer`
  - `pnpm release:portable`
  - `pnpm release:version:set-base`

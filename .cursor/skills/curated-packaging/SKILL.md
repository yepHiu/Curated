---
name: curated-packaging
description: Use when the user asks in natural language to preview or run Curated packaging workflows in this repository, including production, installer, portable, or version-base changes before packaging.
license: MIT
metadata:
  repo: Curated
  version: "0.3.0"
  previewScript: .cursor/skills/curated-packaging/scripts/preview-package.py
  executeScript: .cursor/skills/curated-packaging/scripts/execute-package.py
  packagingCommands:
    - python scripts/release/release_cli.py show-version
    - python scripts/release/release_cli.py package-portable
    - python scripts/release/release_cli.py package-installer
    - python scripts/release/release_cli.py publish
    - python scripts/release/release_cli.py set-version-base
    - python scripts/release/release_cli.py migrate-history
---

This skill orchestrates the repo-local packaging workflow for Curated.

Use this skill only inside this repository. Do not treat it as a global skill or publish it outside the repo.

## Request Handling

- Map natural-language packaging requests to one of: `publish`, `installer`, `portable`, `preview`, `set-base`.
- Respect the repository release rules:
  - `scripts/release/version.json` is the only automatic production version source.
  - `docs/package-build-history.csv` is the active package/version history ledger.
  - `publish` allocates one version once and reuses it for all generated artifacts.
  - `major` and `minor` changes are manual base changes; `patch` is the auto-bump part.
- Treat package history management as part of the packaging workflow:
  - packaging commands append new rows to `docs/package-build-history.csv`
  - `change_summary` comes from the git range between the previous ledger row and the current commit
  - the legacy Markdown history doc is only an entry point, not the active write target
- Always preview before execution.
- If the request includes a base change before packaging, preview the base change first and then the resulting packaging version.

## Preview

- Use `scripts/preview-package.py` for the preview output.
- Preview must include:
  - detected mode
  - current base version
  - base version after any requested change
  - predicted version
  - whether patch will bump
  - active history ledger path
  - latest history row summary when the CSV ledger already exists

## Execution

- Execute only after the preview has been shown to the user.
- Use `scripts/execute-package.py` for execution.
- Reuse the repository packaging commands instead of duplicating release logic:
  - `python scripts/release/release_cli.py show-version`
  - `python scripts/release/release_cli.py publish`
  - `python scripts/release/release_cli.py package-installer`
  - `python scripts/release/release_cli.py package-portable`
  - `python scripts/release/release_cli.py set-version-base`
  - `python scripts/release/release_cli.py migrate-history`
- When the user asks about package/version history, use `docs/package-build-history.csv` as the source of truth and prefer showing the latest row plus its `change_summary`.

# Release Notes

This directory stores packaged release notes for Curated builds.

Convention:

- Each production packaging run should produce one release note file here.
- File naming should use the release date and version, for example:
  - `2026-04-19-release-1.2.7-notes.md`
- The release note should include:
  - release version
  - short summary of changes
  - artifact names
  - checksums when available
  - a `## GitHub Release Body` section containing the final publish-ready body for the repository GitHub Release
- Do not label the GitHub Release body as a draft or suggested draft. The release note should contain the actual release description to publish.

Public descriptions are written in English and follow the structure used by v1.5.8: opening summary, What's Changed, Upgrade Notes, Downloads, and Full Changelog. Component releases use `<component>-vX.Y.Z.md`; each file contains a `## GitHub Release Body` section. Keep internal implementation and build diagnostics in project documentation.

Published component descriptions can be updated from the corresponding note files through the Release notes workflow. It updates the display title and body, preserves the original source marker, and checks that assets, tags, publication state and Latest remain unchanged.

GitHub Release titles use `Curated vX.Y.Z`. Component prefixes belong in tags and asset filenames, not the display title.

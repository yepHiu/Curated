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

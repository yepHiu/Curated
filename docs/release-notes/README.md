# Release Notes

This directory stores packaged release notes for Curated builds.

## Release batches (2026-10-01)

Use one `docs/release-notes/<tag>.md` per prepared `scripts/release/batches/<tag>.json`. Titles are **Curated YYYYMMDD**, **Curated YYYYMMDD-2**, **-3**, etc.; date means Beijing date. The manifest fixes the date and sequence. Do not derive a fresh date on rerun or Notes synchronization.

Run `pnpm release:prepare` to inspect and `pnpm release:prepare --write` to prepare local versions, manifest and a notes draft. Review generated commit summaries before committing and tagging. Ordinary tags are `release-YYYYMMDD[-N]`; the initial Server bridge uses `server-v<version>` with the same dated title. Full is retired. One Release may contain either or both updated components; unchanged modules keep their versions and link to earlier downloads.

Every GitHub body includes exactly these two module rows; status and versions must match the immutable manifest. Example:

```markdown
## GitHub Release Body

This release improves Server scanning and Desktop connections.

### Module updates

| Module | Status | Before | After |
| --- | --- | --- | --- |
| Server | Updated | 1.7.7 | 1.7.8 |
| Desktop | Updated | 0.2.2 | 0.2.3 |

### What's Changed

- Server: describe actual changes.
- Desktop: describe actual changes.

### Upgrade Notes

Describe installation, compatibility and migration requirements.

### Downloads

Link each new package; link unchanged components to their previous packages.

### Full Changelog

Link each updated component's published baseline commit to this tag.
```

For an unchanged module, use `Unchanged` and identical Before/After. Both updated means one batch, one Notes file, one Release. Component changes are compared against each component's last published code. See [guide §8](../guide.md#8-release-and-packaging) for draft/publish, bridge channels and recovery.

Historical convention:

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

Public descriptions are written in English and follow the structure used by v1.5.8: opening summary, What's Changed, Upgrade Notes, Downloads, and Full Changelog. Historical component releases used `<component>-vX.Y.Z.md`; current batches use `<batch-tag>.md`. Each file contains a `## GitHub Release Body` section. Keep internal implementation and build diagnostics in project documentation.

Published component descriptions can be updated from the corresponding note files through the Release notes workflow. It updates the display title and body, preserves the original source marker, and checks that assets, tags, publication state and Latest remain unchanged.

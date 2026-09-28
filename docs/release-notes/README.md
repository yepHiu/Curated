# Release Notes

This directory stores packaged release notes for Curated builds.

## Standalone release policy (2026-09-28)

Future releases contain **Desktop or Server only**. Full is retired from new releases. Advance a module version only when that module has actual shipped changes; an unchanged module keeps its version, packages and channel. Server owns its hosted Web UI; Desktop owns the Electron shell and bundled connection UI. Shared changes require a version increase only for the affected deliverables. Documentation/test/CI-only changes do not automatically increase product versions.

Use `server-vX.Y.Z.md` or `desktop-vX.Y.Z.md`. Every publish-ready body must include both module rows in this exact four-column format. Example for a Server-only update (illustrative, not a release request):

```markdown
## GitHub Release Body

This release improves Server library scanning.

### Module updates

| Module | Status | Before | After |
| --- | --- | --- | --- |
| Desktop | Unchanged | 0.2.1 | 0.2.1 |
| Server | Updated | 1.7.3 | 1.7.4 |

Desktop has no changes in this release; keep the existing installation.

### What's Changed

- Server: describe the actual user-visible scanning improvement.

### Upgrade Notes

Only Server needs updating. Desktop remains compatible at 0.2.1.

### Downloads

List the Server installer and ZIP for this tag, with verified asset links.

### Full Changelog

Link the comparison with the previous Server release source.
```

CD requires exactly one row per module. The tagged module must be `Updated`, its After must equal the tag, and its version must increase. The other must be `Unchanged` with identical Before/After. The updated module’s Before must match its currently published channel (`0.0.0` if never published). The unchanged row is a snapshot: its version must not exceed the published channel, but a later independent update of that other module does not invalidate these notes. Keep the table in the GitHub body, not only internal notes. Include concrete changes for the updated module; review which deliverables actually changed before bumping any version source.

When both modules change, publish two tags and two notes. Each table describes only its own Release; cross-link the companion Release in prose. If the draft's own module channel has since changed, prepare reviewed notes and a new immutable tag. Existing historical descriptions are not rewritten by this policy.

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

Public descriptions are written in English and follow the structure used by v1.5.8: opening summary, What's Changed, Upgrade Notes, Downloads, and Full Changelog. Component releases use `<component>-vX.Y.Z.md`; each file contains a `## GitHub Release Body` section. Keep internal implementation and build diagnostics in project documentation.

Published component descriptions can be updated from the corresponding note files through the Release notes workflow. It updates the display title and body, preserves the original source marker, and checks that assets, tags, publication state and Latest remain unchanged.

New GitHub Release titles use `Curated - Server X.Y.Z + Desktop A.B.C - Server update` (or `Desktop update`), with ordinary hyphens. Publication and note synchronization share the same title generator and read the validated Module updates After snapshots, never unpublished source targets. Historical Full titles remain unchanged. The combination is display-only; tags, artifacts and update comparisons keep independent component versions.

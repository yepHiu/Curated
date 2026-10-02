# Curated 20261002

## GitHub Release Body

This batch adds homepage topic browsing, recoverable AI tag organization, and continuous picture-in-picture playback to Server, and improves Desktop window recovery on macOS. Both components receive independent packages and update channels.

### Module updates

| Module | Status | Before | After |
| --- | --- | --- | --- |
| Server | Updated | 1.7.6 | 1.7.7 |
| Desktop | Updated | 0.2.2 | 0.2.3 |

### What's Changed

- **Server — Homepage and navigation:** Browse your library by subject, move between the homepage and movie library with scrolling, and return reliably to the top of virtual movie lists. Homepage sections, posters, and page transitions have cleaner spacing and presentation.
- **Server — AI topics and tags:** Organize user tags into persistent, reversible topics with recoverable progress and requests bounded by the configured model capacity. Topic labels follow the task language and can be renamed while preserving their identity and undo history. Existing site tags remain separate.
- **Server — Playback:** Keep picture-in-picture playback running across pages, use the unified sidebar playback controls, and return to normal playback reliably. Hardware HLS playback starts with shorter buffering.
- **Server — Actors and captured frames:** Merge actor profiles with clearer selection and profile preservation. Browse captured frames on movie and actor pages through a shared detail viewer, with simpler image controls and improved page scrolling.
- **Server — Updates and page recovery:** Recover from stale remote page scripts after an upgrade. This release introduces the bridge to the new Server update feed while retaining compatibility with previously installed Servers.
- **Desktop — macOS window recovery:** Clicking the Dock icon restores a hidden Desktop window.
- **Server and Desktop — Windows installation:** Restore standard wizard button sizes and prevent the finish-page launch checkbox from being clipped at different display scaling settings.

### Upgrade Notes

- Server 1.7.7 and Desktop 0.2.3 are separate installations. Update both components to receive all changes; exiting Desktop leaves Server running. Existing library data and Desktop connections are retained.
- Back up the Server library before upgrading. Server applies the new topic-storage migrations when it starts; AI tag organization requires a configured provider.
- The macOS Desktop package supports Apple Silicon and is ad-hoc signed; it is not Apple notarized.
- Old all-in-one installations require the [Full 1.7.3 migration](https://github.com/yepHiu/Curated/releases/tag/full-v1.7.3) first.
- This Server release bridges older installations to the new update channel. Later updates remain available after installing it.

### Downloads

- [Curated-Server-Setup-1.7.7-windows-x64.exe](https://github.com/yepHiu/Curated/releases/download/server-v1.7.7/Curated-Server-Setup-1.7.7-windows-x64.exe)
- [Curated-Server-1.7.7-windows-x64.zip](https://github.com/yepHiu/Curated/releases/download/server-v1.7.7/Curated-Server-1.7.7-windows-x64.zip)
- [Curated-Desktop-Setup-0.2.3-windows-x64.exe](https://github.com/yepHiu/Curated/releases/download/server-v1.7.7/Curated-Desktop-Setup-0.2.3-windows-x64.exe)
- [Curated-Desktop-0.2.3-windows-x64.zip](https://github.com/yepHiu/Curated/releases/download/server-v1.7.7/Curated-Desktop-0.2.3-windows-x64.zip)
- [Curated-Desktop-0.2.3-macos-arm64.dmg](https://github.com/yepHiu/Curated/releases/download/server-v1.7.7/Curated-Desktop-0.2.3-macos-arm64.dmg)
- [Curated-Desktop-0.2.3-macos-arm64.zip](https://github.com/yepHiu/Curated/releases/download/server-v1.7.7/Curated-Desktop-0.2.3-macos-arm64.zip)

### Full Changelog

- [Server changes](https://github.com/yepHiu/Curated/compare/c692d98ba3e2b6a3c5b5539591e600476ac56108...server-v1.7.7)
- [Desktop changes](https://github.com/yepHiu/Curated/compare/c692d98ba3e2b6a3c5b5539591e600476ac56108...server-v1.7.7)

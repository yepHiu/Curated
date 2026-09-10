# Comic Library Auto Watch Design

> Status: confirmed by the user on 2026-06-28 and implemented with an independent comic watcher.

## Background

Curated already has a movie library directory watch flow based on `fsnotify`: watched roots emit file events, events are debounced, and the backend queues `scan.library` with `metadata.trigger = "fsnotify"`.

The comic library is intentionally independent from the movie library. The auto watch feature must follow the movie library's behavior pattern without mixing tables, repositories, services, routes, or frontend service contracts.

## Product Behavior

- The comic library remains hidden unless `comicLibraryEnabled=true`.
- When the comic library is enabled and comic auto watch is enabled, configured comic storage paths are watched for new or changed archive files.
- Supported watched archive extensions are `.zip` and `.cbz`.
- Watch-triggered import means "scan and index archives already placed under configured comic storage paths"; it does not copy source files.
- Browser/manual comic import remains separate: `POST /api/import/comics` copies selected archives into `defaultComicImportLibraryPathId` and then starts a comic scan.
- Comic auto watch ignores temp-like files such as hidden files, `*.tmp`, `*.part`, and editor backup files.
- Missing or offline watched roots are skipped by the watcher. Manual scan/import availability remains governed by existing storage checks.

## Architecture

Add an independent persisted setting:

- `autoComicLibraryWatch`, default `true`.
- It is active only when `comicLibraryEnabled=true` and the global yaml `libraryWatchEnabled` gate allows fsnotify.
- It is returned by `GET /api/settings`, accepted by `PATCH /api/settings`, and persisted in `config/library-config.cfg`.
- The setting is exposed only in the comic settings section.

Add a dedicated comic watcher:

- New package: `backend/internal/comicwatch`.
- It mirrors the reliable parts of `internal/librarywatch` but uses comic-specific interfaces and archive extension filtering.
- It reads roots from `comic_library_paths`.
- It queues root paths through an App-level method such as `EnqueueComicLibraryWatchScanRoots`.
- It reloads when comic library paths are added or removed.

Centralize comic scan startup in `App`:

- Add an App-owned comic scan starter used by manual scan, comic import follow-up scan, and watch-triggered scan.
- Watch-triggered scans add task metadata such as `trigger: "fsnotify"` and `paths`.
- Add single-flight/pending queue behavior for comic scans so import-triggered and watcher-triggered scans do not race each other or duplicate work excessively.
- If a scan is already running, watch-triggered roots are retained and drained after the active scan finishes.

Frontend updates:

- Extend `SettingsDTO`, `PatchSettingsRequest`, and comic service contracts/adapters with `autoComicLibraryWatch`.
- Add a toggle in `SettingsComicLibrarySection.vue` using the same row/card style as existing settings controls.
- Extend watch toasts to recognize terminal `scan.comics` tasks with `metadata.trigger = "fsnotify"`, show comic-specific copy, and reload the comic list.

## Alternatives Considered

1. Independent comic watcher and independent setting. This is the selected approach because it preserves the user's requirement that movies and comics have strong domain independence.
2. Reuse the existing movie `autoLibraryWatch` setting and watcher with comic callbacks. This reduces code but makes the setting semantics muddy and couples two domains the product wants to keep separate.
3. Poll comic roots periodically. This avoids filesystem watcher edge cases but is less immediate, more expensive, and less aligned with the existing movie library behavior.

## Implementation Notes

- Do not model comics as movies.
- Do not route comic auto watch through `/api/library/movies` or `useLibraryService()`.
- Do not write comic files or metadata into movie tables.
- Keep `.cursor/rules/project-facts.mdc` aligned with the implemented comic watcher behavior.

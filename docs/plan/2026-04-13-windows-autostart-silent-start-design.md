# Windows Autostart Silent Start Design

Status: draft, pending approval

## Goal

Add a Settings -> General switch for Windows login autostart. It must default to off, persist through the existing settings pipeline, and when Windows launches Curated from autostart it should start silently in tray mode without opening the browser.

## Recommended approach

Use the current-user Windows `Run` registry entry as the OS integration surface.

- Persist a new boolean setting through `GET/PATCH /api/settings` into `config/library-config.cfg`.
- When enabled on Windows, write/update a `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\Curated` value pointing to the packaged executable with autostart markers such as `-mode tray -autostart`.
- When disabled, remove that registry value.
- Extend tray startup so `-autostart` suppresses the initial browser launch while keeping the tray icon and local HTTP server running.
- Surface the toggle in Settings -> General alongside the existing backend log controls, following the current auto-save behavior.

## Alternatives considered

### Option A: HKCU Run registry entry

Pros:

- Matches the current-user tray app model.
- No admin permission required.
- Easy to add/remove atomically from a settings toggle.
- Straightforward to make silent by appending an autostart flag.

Cons:

- Windows-only implementation path.
- Requires careful quoting of the executable path.

### Option B: Startup folder shortcut

Pros:

- Visible to users in the Startup folder.
- Familiar Windows behavior.

Cons:

- Requires `.lnk` creation/update code or scripting glue.
- More moving parts than a registry string value.
- Harder to keep arguments synchronized.

### Option C: Scheduled task

Pros:

- More control over delayed startup and conditions.

Cons:

- Overkill for a tray app toggle.
- More fragile UX and more cleanup cases.

## Proposed UX and persistence

- New toggle label under Settings -> General, default off.
- In Web API mode, changing the toggle uses the same debounced auto-save pattern as other persisted general settings.
- In mock mode or non-Windows environments, keep the switch disabled or show a clear unsupported hint instead of pretending the OS side is active.
- Persist the boolean in `library-config.cfg` so the UI reflects the last saved value.

## Proposed runtime behavior

- Manual launch:
  - Tray mode still opens the browser as it does today.
- Windows autostart launch:
  - The executable starts in tray mode.
  - Local HTTP server still starts.
  - Tray icon still appears.
  - Initial browser launch is suppressed.
- Secondary launch while an autostarted instance is already running:
  - Existing single-instance behavior can continue reopening the browser, so explicit user launches still reveal the UI.

## Expected code areas

- Backend contracts and settings DTO/patch types
- Backend config merge/write helpers for the new persisted boolean
- Windows desktop integration helper for add/remove Run entry
- `cmd/curated` tray startup path for `-autostart` silent launch behavior
- Settings service adapter and Settings -> General UI
- Docs/rules updates for the new persisted setting

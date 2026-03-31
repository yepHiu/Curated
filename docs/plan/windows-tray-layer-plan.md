# Curated Windows Tray Layer Plan

## Goal

Replace the current `systray`-based Windows tray layer with a self-managed native Win32 implementation that is stable under repeated right-click use, while preserving the current product behavior:

- Windows release builds default to tray mode
- the local HTTP server still hosts the UI and API
- the browser remains the primary UI shell
- single-instance behavior stays in place
- installer and portable zip continue to work from the same release pipeline

## Why We Are Replacing `systray`

The current `github.com/getlantern/systray` integration still shows an unstable context menu on Windows after repeated right-clicks, even after compatibility fixes. That makes the tray unreliable for production use.

Because the tray is now a core desktop entry point for Curated, we should own the Windows implementation directly instead of continuing to patch a third-party abstraction.

## Target User Experience

After launching `curated.exe` on Windows:

1. Curated starts the local HTTP server.
2. Curated places an icon in the Windows notification area.
3. Curated opens the browser to the local UI.
4. Repeated right-clicks on the tray icon always show the context menu.
5. Left-click may optionally reopen the browser, but right-click must remain the primary menu interaction.
6. The menu includes:
   - Open Curated
   - Open Settings
   - Open Logs
   - Quit
7. Closing the browser does not terminate Curated.
8. Choosing `Quit` shuts down the tray and backend cleanly.

## Implementation Strategy

### 1. Own the Win32 tray runtime

Implement the tray directly with Win32 APIs:

- `RegisterClassExW`
- `CreateWindowExW`
- `Shell_NotifyIconW`
- `CreatePopupMenu`
- `AppendMenuW` or `InsertMenuItemW`
- `TrackPopupMenu`
- `PostMessageW`
- `DestroyWindow`

This lets us control:

- notification icon registration
- tray callback messages
- left-click and right-click behavior
- popup menu lifetime
- menu focus reset after close
- explorer restart recovery

### 2. Keep the current app lifecycle

The existing tray mode flow remains structurally the same:

- bootstrap config, logging, storage, and server
- enforce single instance
- start HTTP server
- wait for health readiness
- start tray runtime
- open browser
- keep process alive until tray quit or shutdown signal

### 3. Reuse existing helpers

We should continue using:

- `internal/shellopen` for URL and directory opening
- existing single-instance mutex code
- existing version display and health endpoint data
- current release scripts and installer flow

## Code Structure

Recommended ownership inside `backend/internal/desktop/`:

- `tray_windows.go`
  - public `RunTray(...)`
  - tray lifecycle orchestration
- `tray_native_windows.go`
  - Win32 structs, constants, callbacks, menu wiring
- `tray_stub.go`
  - unchanged non-Windows stub
- `message_windows.go`
  - keep error dialogs and optionally add info dialogs later
- `single_instance_windows.go`
  - unchanged

## Native Tray Requirements

The native tray implementation should support:

- loading `backend/internal/assets/curated.ico`
- setting tooltip text to include version information
- stable repeated popup display
- menu item command dispatch
- quit behavior that removes the tray icon before exit
- handling `TaskbarCreated` so the icon is restored after explorer restarts

## Menu Commands

Initial command IDs:

- `1001` Open Curated
- `1002` Open Settings
- `1003` Open Logs
- `1099` Quit

This is enough for the minimum stable production tray.

## Release Integration

The release pipeline should continue to:

- embed the tray icon into the backend binary
- copy `curated.ico` into the assembled release directory
- use the same icon in the installer shortcuts and uninstall metadata
- build a GUI subsystem binary for Windows release mode

No change is needed to the overall installer vs zip distribution strategy.

## Validation Checklist

For each release candidate:

1. Launch installed Curated.
2. Confirm tray icon is visible.
3. Right-click the tray icon at least 10 times.
4. Confirm the menu appears every time.
5. Click `Open Curated` and confirm the browser opens.
6. Click `Open Settings` and confirm `/#/settings` opens.
7. Click `Open Logs` and confirm the log directory opens.
8. Click `Quit` and confirm:
   - tray icon disappears
   - backend process exits
   - relaunch works normally

## Risks

- Win32 tray code is more verbose and lower-level than `systray`.
- Window procedure bugs can cause subtle UI issues if message handling is incomplete.
- Explorer restart behavior must be handled explicitly.

Even with those risks, this approach is still the right one because it removes a flaky abstraction from a critical desktop path.

## Decision

Move Curated to a self-managed native Windows tray implementation and stop using `github.com/getlantern/systray` for production tray behavior.

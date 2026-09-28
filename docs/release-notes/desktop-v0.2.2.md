# Curated Desktop 0.2.2

Preparation status: source and release description prepared locally; this document does not indicate that a tag, production package or public Release already exists. CD must validate the complete distribution before publication.

## GitHub Release Body

Curated Desktop 0.2.2 makes Windows installation and upgrades clearer, with an explicit destination page, version confirmation and a separate step for closing a running application.

### Module updates

| Module | Status | Before | After |
| --- | --- | --- | --- |
| Desktop | Updated | 0.2.1 | 0.2.2 |
| Server | Unchanged | 1.7.4 | 1.7.4 |

This Release updates **Desktop only**. Server is unchanged by this package. The companion Server improvements are available separately in [Server 1.7.4](https://github.com/yepHiu/Curated/releases/tag/server-v1.7.4); each module has its own Release and installation. No new Full package is produced.

### What's Changed

- A dedicated destination page always shows the program location. Fresh installations provide a default folder and allow a custom empty local folder; upgrades show and retain the original location.
- Setup shows the installed and target versions, asks to overwrite and upgrade, supports same-version reinstallation and blocks downgrades.
- A running-application page appears only when needed. Choose to request a normal exit, or close the application yourself and check again. Failed or refused shutdown prevents file replacement and allows retry or cancellation.
- Shutdown is limited to the selected installation under the current user and session. Server keeps running during a Desktop update.
- The completion page offers to launch Desktop; an upgrade of a previously stopped client leaves this choice unchecked.
- The new prompts support Simplified Chinese and English.

### Upgrade Notes

- Existing independent Desktop installations upgrade in place and preserve saved Server connections and client settings. Changing an existing installation directory is not supported by this upgrade flow.
- Finish active work before allowing setup to close Desktop. If automatic shutdown fails, exit it from the tray and retry.
- Unattended installation requires `/CLOSECURATED=1` to permit closing a running instance. Silent setup does not launch the application afterward. Interactive setup offers an explicit launch choice.
- Users of the old all-in-one package should first use the already published [Full 1.7.3 migration package](https://github.com/yepHiu/Curated/releases/tag/full-v1.7.3), then update the independent components.
- This release changes the Windows installer. The macOS Desktop delivery keeps its existing drag-to-Applications flow and shares the Desktop version.

### Downloads

Select the matching asset attached to this Release:

- [Curated-Desktop-Setup-0.2.2-windows-x64.exe](https://github.com/yepHiu/Curated/releases/download/desktop-v0.2.2/Curated-Desktop-Setup-0.2.2-windows-x64.exe)
- [Curated-Desktop-0.2.2-windows-x64.zip](https://github.com/yepHiu/Curated/releases/download/desktop-v0.2.2/Curated-Desktop-0.2.2-windows-x64.zip)
- [Curated-Desktop-0.2.2-macos-arm64.dmg](https://github.com/yepHiu/Curated/releases/download/desktop-v0.2.2/Curated-Desktop-0.2.2-macos-arm64.dmg)
- [Curated-Desktop-0.2.2-macos-arm64.zip](https://github.com/yepHiu/Curated/releases/download/desktop-v0.2.2/Curated-Desktop-0.2.2-macos-arm64.zip)

Use the accompanying `SHA256SUMS` for artifact integrity verification.

### Full Changelog

[Compare full-v1.7.3...desktop-v0.2.2](https://github.com/yepHiu/Curated/compare/full-v1.7.3...desktop-v0.2.2)

# Curated Server 1.7.5

Preparation status: source and release description prepared locally; this document does not indicate that a tag, production package or public Release already exists. CD must validate the complete distribution before publication.

## GitHub Release Body

Curated Server 1.7.5 improves backup and maintenance workflows, simplifies library and log settings, and makes Windows installation and upgrades clearer.

### Module updates

| Module | Status | Before | After |
| --- | --- | --- | --- |
| Desktop | Unchanged | 0.2.1 | 0.2.1 |
| Server | Updated | 1.7.3 | 1.7.5 |

This Release updates **Server only**. Desktop is unchanged by this package. The companion Windows installer improvement is prepared separately as [Desktop 0.2.2](https://github.com/yepHiu/Curated/releases/tag/desktop-v0.2.2); each module has its own Release and installation. No new Full package is produced.

### What's Changed

- Backup tools let you select a backup file before validation, inspect its details and find recent packages. Imported files remain available for later use; the default backup destination follows the Server data directory.
- Backup descriptions clarify the included data and assets, including static preview frames; maintenance and library health actions use a simpler layout.
- Movie storage directories use a compact list, clearly identify the default destination and keep actions beside each directory.
- Server logs use the default application log directory. Local Desktop connections can open it; log retention and level controls are restricted to local management.
- The hosted interface distinguishes Web and Desktop branding, and wishlist catalog badges remain visible above cover images.

- A dedicated destination page always shows the program location. Fresh installations provide a default folder and allow a custom empty local folder; upgrades show and retain the original location.
- Setup shows the installed and target versions, asks to overwrite and upgrade, supports same-version reinstallation and blocks downgrades.
- A running-application page appears only when needed. Choose to request a normal exit, or close the application yourself and check again. Failed or refused shutdown prevents file replacement and allows retry or cancellation.
- Shutdown is limited to the selected installation under the current user and session. Other clients may temporarily disconnect while Server is stopped.
- The first launch opens Server management after the service is ready; upgrades can restore Server in the background.
- The new prompts support Simplified Chinese and English.

### Upgrade Notes

- Existing independent Server installations upgrade in place and preserve library, configuration and media. Changing an existing installation directory is not supported by this upgrade flow.
- Finish active work before allowing setup to close Server. If automatic shutdown fails, exit it from the tray and retry.
- Unattended installation requires `/CLOSECURATED=1` to permit closing a running instance. Silent setup does not launch the application afterward. Interactive setup offers an explicit launch choice.
- Users of the old all-in-one package should first use the already published [Full 1.7.3 migration package](https://github.com/yepHiu/Curated/releases/tag/full-v1.7.3), then update the independent components.

### Downloads

Select the matching asset attached to this Release:

- [Curated-Server-Setup-1.7.5-windows-x64.exe](https://github.com/yepHiu/Curated/releases/download/server-v1.7.5/Curated-Server-Setup-1.7.5-windows-x64.exe)
- [Curated-Server-1.7.5-windows-x64.zip](https://github.com/yepHiu/Curated/releases/download/server-v1.7.5/Curated-Server-1.7.5-windows-x64.zip)

Use the accompanying `SHA256SUMS` for artifact integrity verification.

### Full Changelog

[Compare full-v1.7.3...server-v1.7.5](https://github.com/yepHiu/Curated/compare/full-v1.7.3...server-v1.7.5)

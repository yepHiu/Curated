# Curated Full 1.7.1

## GitHub Release Body

Curated Full 1.7.1 makes finding and connecting to your Server easier, adds local Desktop network preferences, and preserves saved connections and sessions when upgrading the split apps.

### What's Changed

**Desktop connections and device settings**

- Use the new local connection page to enter a Server address, reconnect to recent servers, or discover available servers on your local network.
- Choose a Desktop theme, use the system proxy or a direct/manual connection, and configure login startup from this device's settings. These preferences do not change Server settings; proxy changes apply after restarting Desktop.
- Keep the current Server page available when a new connection fails. Quitting Desktop still leaves the independent Server running.

**Server discovery and connection compatibility**

- Servers now have a persistent identity. First-time identity binding preserves existing sessions; a subsequent identity change requires confirmation and starts a separate session.
- Enable local-network discovery alongside LAN access, or turn discovery off from the Server computer and restart Server. Manual address entry remains available when discovery is blocked by a firewall or VLAN.
- Connect to older Servers, including PIN-locked 1.5.x and 1.6.x versions. PIN unlock remains required, and unsupported future connection protocols are rejected explicitly.
- Preserve saved server names, IDs, the last connection, cookies and local storage when upgrading Desktop 0.1.0. Development connection profiles are validated completely before import, and their original files are retained.

### Upgrade Notes

- Full 1.7.1 includes Server 1.7.0 and Desktop 0.2.0. Their version numbers and update channels are independent.
- Existing split installations with Server 1.6.0 and Desktop 0.1.0 support in-place upgrades using the same installation identities and locations. Windows installation and database checks, and macOS Desktop profile upgrades, have passed. This release adds no SQLite schema migration. Back up and verify your data, then quit the old apps before upgrading.
- All-in-one installations at 1.5.8 or earlier still require manual migration. Back up, disable the old login startup entry, fully quit Curated, uninstall the old program while keeping your library, and install Full under the same user account. For custom `CURATED_DATA_DIR` or `-config`, use `/NOLAUNCH=1` and start Server with the original configuration. Cross-account data, custom configuration and old shared sessions are not migrated automatically.
- Keep `<databasePath>.server-id` when moving the same Server. A corrupt identity file stops startup rather than silently creating a new identity.
- Server-wide settings and native operations remain restricted to the Server computer. Split apps use their own update feeds; GitHub Latest remains on the compatible all-in-one release for old update clients.
- Windows packages remain unsigned; Server includes FFmpeg. macOS Desktop supports Apple Silicon only, is ad-hoc signed, and is not notarized by Apple. Verify downloads with `SHA256SUMS.txt`.

See the [upgrade and migration guide](https://github.com/yepHiu/Curated/blob/full-v1.7.1/docs/guide.md) for configuration and recovery details.

### Downloads

- [Windows Full installer](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/Curated-Full-Setup-1.7.1-windows-x64.exe)
- [Windows Full offline ZIP](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/Curated-Full-1.7.1-windows-x64.zip)
- [Windows Server installer](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/Curated-Server-Setup-1.7.0-windows-x64.exe)
- [Windows Server portable ZIP](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/Curated-Server-1.7.0-windows-x64.zip)
- [Windows Desktop installer](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/Curated-Desktop-Setup-0.2.0-windows-x64.exe)
- [Windows Desktop portable ZIP](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/Curated-Desktop-0.2.0-windows-x64.zip)
- [macOS Desktop DMG (Apple Silicon)](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/Curated-Desktop-0.2.0-macos-arm64.dmg)
- [macOS Desktop ZIP (Apple Silicon)](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/Curated-Desktop-0.2.0-macos-arm64.zip)
- [Release manifest](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/release.json)
- [SHA-256 checksums](https://github.com/yepHiu/Curated/releases/download/full-v1.7.1/SHA256SUMS.txt)

### Full Changelog

[full-v1.6.0...full-v1.7.1](https://github.com/yepHiu/Curated/compare/full-v1.6.0...full-v1.7.1)

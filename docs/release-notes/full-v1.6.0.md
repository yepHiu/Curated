# Curated Full 1.6.0

## GitHub Release Body

Curated Full 1.6.0 introduces independently installed Server and Desktop apps, adds an Apple Silicon Desktop client, and lets the Server keep running when you close Desktop.

### What's Changed

**Independent Server and Desktop apps**

- Install Server to host your library, Desktop to connect to an existing Server, or Full to install both on Windows.
- Full includes Server 1.6.0 and Desktop 0.1.0. It uses the same installers as the standalone downloads and reuses components already installed at the same or a newer version.
- Keep Server running independently in the Windows tray. Quitting Desktop no longer stops your library service.

**Downloads and updates**

- Windows x64 now has separate Full, Server, and Desktop installers and ZIP downloads. The Full ZIP contains the offline component installers.
- Desktop 0.1.0 is available for Apple Silicon Macs as a DMG or ZIP. It connects to an existing Server and does not include a local Server.
- Server and Desktop use separate update channels. Desktop can check for its own updates from the connection window before connecting to a Server.

### Upgrade Notes

- This is the first split distribution. Existing all-in-one installations at 1.5.8 or earlier require manual migration; in-app or direct in-place upgrades to Full are not supported.
- Create and verify a backup, record your data directory, configuration and port, disable the old login startup entry, and fully quit Curated. Uninstall the old program while keeping your library, then install Full under the same user account. The default data directory remains `%LOCALAPPDATA%\Curated`.
- If you use `CURATED_DATA_DIR` or a custom `-config`, install with `/NOLAUNCH=1`, start Server with your original configuration, then connect Desktop. Cross-account data and old authentication sessions are not migrated automatically.
- The installer blocks overlapping legacy installations. GitHub Latest remains on the compatible all-in-one release for old update clients; split components use their own update feeds.
- Windows packages remain unsigned; Server includes FFmpeg. macOS Desktop supports Apple Silicon only, is ad-hoc signed, and is not notarized by Apple. Verify downloads with `SHA256SUMS.txt`.

See the [upgrade and migration guide](https://github.com/yepHiu/Curated/blob/full-v1.6.0/docs/guide.md) for configuration and recovery details.

### Downloads

- [Windows Full installer](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/Curated-Full-Setup-1.6.0-windows-x64.exe)
- [Windows Full offline ZIP](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/Curated-Full-1.6.0-windows-x64.zip)
- [Windows Server installer](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/Curated-Server-Setup-1.6.0-windows-x64.exe)
- [Windows Server portable ZIP](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/Curated-Server-1.6.0-windows-x64.zip)
- [Windows Desktop installer](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/Curated-Desktop-Setup-0.1.0-windows-x64.exe)
- [Windows Desktop portable ZIP](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/Curated-Desktop-0.1.0-windows-x64.zip)
- [macOS Desktop DMG (Apple Silicon)](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/Curated-Desktop-0.1.0-macos-arm64.dmg)
- [macOS Desktop ZIP (Apple Silicon)](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/Curated-Desktop-0.1.0-macos-arm64.zip)
- [Release manifest](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/release.json)
- [SHA-256 checksums](https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/SHA256SUMS.txt)

### Full Changelog

[v1.5.8...full-v1.6.0](https://github.com/yepHiu/Curated/compare/v1.5.8...full-v1.6.0)

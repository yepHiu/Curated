# Curated Full 1.7.2

## GitHub Release Body

Curated Full 1.7.2 adds a Windows upgrade flow for existing all-in-one installations. Run Full once to back up your library, replace the old program and install the independent Server and Desktop apps.

### What's Changed

**Upgrade without manually uninstalling first**

- Full detects registered older all-in-one installations and asks for the original data directory and optional custom Server configuration.
- Before removing the old program, Full locks the database and creates a verified backup. Your library, favorites, ratings and external media remain in their original locations.
- The installer handles removal of the old program and installation of both components. If a component fails, keep the migration records and run Full again to retry.

**Keep your original Server configuration**

- Normal Server shortcuts and login startup retain the migrated data directory and saved custom configuration, including the original port.
- Missing migrated files produce an error instead of silently opening a new empty library.
- Full replaces the old Server startup entry when needed and preserves saved Desktop connections when the new profile has no connection file.

### Upgrade Notes

- Full 1.7.2 includes Server 1.7.2 and Desktop 0.2.0. Existing split installations retain their installation identities; newer installed components are not downgraded.
- Fully quit Curated from its tray, then run Full as the same Windows user that owns the library. Windows may ask for permission to remove the old machine-wide program. No separate manual uninstall is required for supported layouts.
- Automatic migration requires an existing registered all-in-one installation with data and media outside its program directory. Custom filesystem paths must be absolute. Program-local data, portable installations, conflicting installations and cross-account migration need the explicit procedure in the migration guide.
- Migration backups and configuration snapshots remain under `%LOCALAPPDATA%\Curated\installer-migrations`. Keep this directory and `server-startup.json`; the startup profile may reference a saved custom configuration there.
- Old authentication sessions are not merged. Unlock again if prompted. After a newer Server has migrated the database, use the verified pre-upgrade backup before returning to an older Server.
- Old in-app update clients continue to use the legacy update feed. Download and run the Full installer explicitly for this migration.
- Windows packages remain unsigned. macOS Desktop remains Apple Silicon only, ad-hoc signed and not notarized; its version is unchanged. Verify downloads with `SHA256SUMS.txt`.

See the [upgrade and migration guide](https://github.com/yepHiu/Curated/blob/full-v1.7.2/docs/guide.md) for custom configuration and recovery details.

### Downloads

- [Windows Full installer](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/Curated-Full-Setup-1.7.2-windows-x64.exe)
- [Windows Full offline ZIP](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/Curated-Full-1.7.2-windows-x64.zip)
- [Windows Server installer](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/Curated-Server-Setup-1.7.2-windows-x64.exe)
- [Windows Server portable ZIP](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/Curated-Server-1.7.2-windows-x64.zip)
- [Windows Desktop installer](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/Curated-Desktop-Setup-0.2.0-windows-x64.exe)
- [Windows Desktop portable ZIP](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/Curated-Desktop-0.2.0-windows-x64.zip)
- [macOS Desktop DMG (Apple Silicon)](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/Curated-Desktop-0.2.0-macos-arm64.dmg)
- [macOS Desktop ZIP (Apple Silicon)](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/Curated-Desktop-0.2.0-macos-arm64.zip)
- [Release manifest](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/release.json)
- [SHA-256 checksums](https://github.com/yepHiu/Curated/releases/download/full-v1.7.2/SHA256SUMS.txt)

### Full Changelog

[full-v1.7.1...full-v1.7.2](https://github.com/yepHiu/Curated/compare/full-v1.7.1...full-v1.7.2)

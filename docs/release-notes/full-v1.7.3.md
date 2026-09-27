# Curated Full 1.7.3

Curated Full 1.7.3 includes Server 1.7.3 and Desktop 0.2.1.

- Windows setup now offers 简体中文 and English, follows the Windows UI language, and translates migration pages, launch options and upgrade errors. Full passes the selected language to both component installers.
- Setup requests that the target installation exit and waits before replacing program files. It checks executable paths, Windows user identity and session; unrelated installations are not closed. An application that refuses shutdown blocks the upgrade with an error.
- For supported registered 1.0–1.5 all-in-one installations, Full closes the old program, verifies an offline backup, runs the registered uninstaller and installs the independent components. Library data and external media remain in place. Split components retain their existing installation identity and upgrade in place.
- Desktop's connection window now participates in Windows session shutdown. Server keeps running when Windows cancels a shutdown request.
- The Apple Silicon Desktop DMG adds a branded Retina drag-to-Applications window with Chinese/English instructions. Packaging verifies the mounted Finder layout and application signature.

Run Full as the original Windows user. Confirm the original data directory and optional `-config` file. Machine-wide old uninstallers may request UAC. If automatic shutdown fails, quit from the tray and retry. Keep migration backups and `server-startup.json`; portable installations, program-local data and conflicting installations still need the documented manual procedure.

Publication requires the automated quality checks and Windows/Mac packaging and upgrade gates to pass. Existing public artifacts are not replaced.

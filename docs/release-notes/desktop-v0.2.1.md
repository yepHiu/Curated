# Curated Desktop 0.2.1

- Windows installer supports 简体中文 and English, including launch and upgrade prompts.
- Setup requests a graceful exit of Desktop in the selected installation directory before replacing files. Other installations are not closed; an application that refuses shutdown blocks installation.
- Both the content window and the server connection window handle Windows session shutdown, including installer updates.
- The Apple Silicon DMG includes a branded Retina drag-to-Applications window with Chinese/English instructions and build-time verification of its mounted Finder layout and application signature.

Existing Desktop installations upgrade in place and retain saved connections. Server is independent. Windows installer and Mac package acceptance remain CD gates before publication.

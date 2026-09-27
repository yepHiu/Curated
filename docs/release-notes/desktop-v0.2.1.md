# Curated Desktop 0.2.1

- Windows installer supports 简体中文 and English, including launch and upgrade prompts.
- Setup requests a graceful exit of Desktop in the selected installation directory before replacing files. Other installations are not closed; an application that refuses shutdown blocks installation.
- Both the content window and the server connection window handle Windows session shutdown, including installer updates.

Existing Desktop installations upgrade in place and retain saved connections. Server is independent. The shared Desktop version also advances the Apple Silicon package; this release's behavior changes are Windows-specific. Windows installer acceptance remains a CD gate before publication.

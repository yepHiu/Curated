# Curated Server 1.7.3

- Windows installer supports 简体中文 and English, including upgrade and launch prompts.
- Before replacing files, setup requests a graceful shutdown of Server in the selected installation directory and waits for exit. Other users/sessions are not terminated; a refused shutdown stops installation with an error.
- A cancelled Windows shutdown no longer stops the Server tray.

Existing split Server installations upgrade in place and preserve data. Use Curated Full for migration from an old all-in-one installation. New Windows installer acceptance must pass CD before publication.

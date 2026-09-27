# Curated DMG artwork and layout

The installation window uses the existing Curated wordmark, macOS system fonts,
real Finder icons and an Applications symlink. The 680 × 440 artwork is static
light artwork; the 680 × 488 window leaves space for native window chrome.
Do not render mock application/folder icons into the background. Do not apply
`SetFile`/`hide_extensions` to the signed app: FinderInfo metadata causes strict
code-signature verification to fail.

## Local packaging

From the repository root on Apple Silicon macOS:

```sh
python3 -m venv .workspace/dmg-tools
.workspace/dmg-tools/bin/python -m pip install -r scripts/release/macos/requirements.txt
.workspace/dmg-tools/bin/python scripts/release/release_cli.py package-macos-desktop --output-dir release/macos-desktop
```

Use a new output directory for a second validation build; existing packages are
never overwritten. `pnpm release:macos-desktop` also works when the active Python
has these dependencies (activate the virtual environment first). CD installs the
same pinned requirements. Desktop versions are shared by Windows and macOS;
allocate a new Desktop version when publishing changed artifacts.

## Regenerate artwork

```sh
swift scripts/release/macos/render-background.swift "$PWD" "$PWD/scripts/release/macos"
```

Commit both PNGs after visually checking them. AppKit uses the bitmap's logical
size to scale the 2× context; adding another scale transform crops the artwork.
`dmgbuild` combines the PNGs into a Retina TIFF and writes Finder metadata without
opening Finder or requesting Automation permissions. Layout settings live in
`release_lib/macos_desktop.py`; keep native icon/filename space clear.

Each build verifies the final image checksum, mounts it read-only and checks the
actual layout, application signature, background and Applications link. After an
artwork/layout change, also open that DMG in Finder and inspect the real icons,
labels and instructions at Retina resolution. Automatic checks cannot replace
visual inspection. No Developer ID signing or Apple notarization is added here.

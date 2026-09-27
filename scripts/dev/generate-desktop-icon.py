"""Derive off-white Desktop assets from the existing artwork (requires Pillow)."""
from pathlib import Path

from PIL import Image

DESKTOP_BACKGROUND = (245, 246, 248, 255)  # #F5F6F8, a soft cool off-white.


def generate(root: Path) -> None:
    server = Image.open(root / 'icon/curated-appicon.png').convert('RGBA')
    mark = Image.open(root / 'icon/curated-mark.png').convert('RGBA')
    if server.size != mark.size:
        raise ValueError('App icon and mark must use the same canvas and alignment')
    # Keep the original rounded silhouette and composite the unchanged transparent
    # mark on off-white so its antialiased edges do not retain the dark background.
    desktop = Image.new('RGBA', server.size, DESKTOP_BACKGROUND)
    desktop.putalpha(server.getchannel('A'))
    desktop.alpha_composite(mark)
    desktop.save(root / 'icon/curated-desktop-appicon.png')
    desktop.save(root / 'public/Curated-desktop-icon.png')
    desktop.save(root / 'icon/curated-desktop.ico', sizes=[
        (size, size) for size in (16, 20, 24, 32, 40, 48, 64, 128, 256)
    ])

    canvas_size = 2 * round(max(desktop.size) / 0.84 / 2)
    canvas = Image.new('RGBA', (canvas_size, canvas_size))
    canvas.paste(desktop, ((canvas_size - desktop.width) // 2,
                          (canvas_size - desktop.height) // 2))
    canvas.save(root / 'public/Curated-desktop-icon-macos.png')


if __name__ == '__main__':
    generate(Path(__file__).resolve().parents[2])

"""Remove the dark matte baked into the historical Curated wordmark edge."""
from pathlib import Path

from PIL import Image


PINK = (254, 98, 142)
SERVER_MATTE = (20, 24, 38)


def clean(root: Path) -> Image.Image:
    source = Image.open(root / 'icon/brand/source/curated-wordmark-original.png').convert('RGBA')
    difference = tuple(pink - matte for pink, matte in zip(PINK, SERVER_MATTE))
    denominator = sum(channel * channel for channel in difference)
    pixels = []
    for red, green, blue, alpha in source.get_flattened_data():
        if alpha == 0:
            pixels.append((0, 0, 0, 0))
            continue
        # The original edge is pink pre-blended with #141826. Project its
        # RGB value onto that line to recover pink coverage, then combine it
        # with the file's existing alpha. Full pink pixels stay byte-identical.
        coverage = sum((channel - matte) * direction for channel, matte, direction
                       in zip((red, green, blue), SERVER_MATTE, difference)) / denominator
        coverage = max(0.0, min(1.0, coverage))
        pixels.append((*PINK, round(alpha * coverage)))
    image = Image.new('RGBA', source.size)
    image.putdata(pixels)
    for target in ('icon/curated-wordmark.png', 'src/icon/curated-wordmark.png'):
        image.save(root / target)
    return image


if __name__ == '__main__':
    clean(Path(__file__).resolve().parents[2])

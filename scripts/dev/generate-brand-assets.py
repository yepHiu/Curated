"""Generate brand lockups, an offline preview and a contact sheet.

Requires Pillow, fonttools and brotli. Uses the locked local Outfit package.
The existing Curated master wordmark and mark are retained without redrawing.
"""
from base64 import b64encode
from io import BytesIO
import json
from pathlib import Path
from shutil import copyfile

from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont
from fontTools.pens.svgPathPen import SVGPathPen
from PIL import Image, ImageDraw, ImageFont


def generate(root: Path) -> None:
    output = root / 'icon/brand'
    spec = json.loads((output / 'brand.json').read_text(encoding='utf-8'))
    source = root / 'node_modules/@fontsource-variable/outfit'
    fonts = output / 'fonts'
    fonts.mkdir(exist_ok=True)
    copyfile(source / 'files/outfit-latin-wght-normal.woff2', fonts / 'outfit-latin-wght-normal.woff2')
    copyfile(source / 'LICENSE', fonts / 'Outfit-OFL.txt')
    variable = TTFont(fonts / 'outfit-latin-wght-normal.woff2')
    font = instantiateVariableFont(variable, {'wght': spec['typography']['brandWeight']})
    font.flavor = None
    buffer = BytesIO()
    font.save(buffer)
    buffer.seek(0)
    size = 154
    pillow_font = ImageFont.truetype(buffer, size)
    glyphs = font.getGlyphSet()
    cmap = font.getBestCmap()
    scale = size / font['head'].unitsPerEm
    master_path = root / 'icon/curated-wordmark.png'
    master = Image.open(master_path).convert('RGBA')
    master_data = b64encode(master_path.read_bytes()).decode('ascii')
    for product, original in (('server', 'curated-appicon.png'),
                              ('desktop', 'curated-desktop-appicon.png'),
                              ('app', 'curated-desktop-appicon.png'),
                              ('web', 'curated-appicon.png')):
        copyfile(root / 'icon' / original, output / f'curated-{product}-appicon.png')
    # Mobile platforms apply their own masks: export opaque square artwork,
    # not the rounded Desktop silhouette. Android's foreground stays separate.
    mark = Image.open(root / 'icon/curated-mark.png').convert('RGBA')
    mobile = Image.new('RGBA', (1024, 1024), spec['colors']['clientBackground'])
    mobile.alpha_composite(mark.resize((1024, 1024), Image.Resampling.LANCZOS))
    mobile.convert('RGB').save(output / 'curated-app-icon-1024.png')
    mark.resize((432, 432), Image.Resampling.LANCZOS).save(output / 'curated-app-android-foreground.png')
    # Existing master ink ends at x=960; a 56px gap separates the product suffix.
    suffix_x = 1016
    cards = []
    for product in spec['products']:
        suffix = product['suffix']
        width = round(suffix_x + pillow_font.getlength(suffix) + 126) if suffix else master.width
        lockup = Image.new('RGBA', (width, master.height))
        lockup.paste(master, (0, 0))
        paths = []
        if suffix:
            baseline = 218
            ImageDraw.Draw(lockup).text((suffix_x, baseline), suffix, font=pillow_font,
                                       fill=spec['colors']['pink'], anchor='ls')
            for index, char in enumerate(suffix):
                glyph = glyphs[cmap[ord(char)]]
                pen = SVGPathPen(glyphs)
                glyph.draw(pen)
                x = suffix_x + pillow_font.getlength(suffix[:index])
                paths.append(f'<path transform="translate({x:.3f} {baseline}) scale({scale:.6f} {-scale:.6f})" d="{pen.getCommands()}"/>')
        name = 'curated' if product['id'] == 'curated' else f"curated-{product['id']}"
        png = output / f'{name}-wordmark.png'
        if suffix:
            lockup.save(png)
        else:
            copyfile(master_path, png)
        svg = (f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="322" viewBox="0 0 {width} 322" role="img" aria-label="{product["name"]}">'
               f'<title>{product["name"]}</title><image width="1085" height="322" href="data:image/png;base64,{master_data}"/>'
               f'<g fill="{spec["colors"]["pink"]}">{"".join(paths)}</g></svg>')
        (output / f'{name}-wordmark.svg').write_text(svg, encoding='utf-8')
        cards.append(f'<section><h2>{product["name"]}</h2><div class="samples"><div class="light"><img src="{png.name}" alt="{product["name"]}"></div><div class="dark"><img src="{png.name}" alt="{product["name"]}"></div></div><p><a href="{png.name}">PNG</a> · <a href="{name}-wordmark.svg">SVG</a> · <a href="{product["icon"]}">图标</a></p></section>')

    html = '''<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Curated 品牌资源</title>
<style>@font-face{font-family:Outfit;src:url(fonts/outfit-latin-wght-normal.woff2);font-weight:100 900}*{box-sizing:border-box}body{margin:0;background:#F5F6F8;color:#0F1219;font-family:system-ui,sans-serif}main{max-width:1120px;margin:auto;padding:48px 24px}h1,h2{font-family:Outfit,system-ui,sans-serif}h1{font-size:44px;margin:0}h2{font-size:24px}p{line-height:1.7;color:#5A6378}section{margin-top:36px}.samples{display:grid;grid-template-columns:1fr 1fr;gap:16px}.samples>div{border-radius:20px;padding:16px;min-width:0;display:flex;align-items:center;min-height:150px}.light{background:#fff;border:1px solid #e3e5ec}.dark{background:#141826}.samples img{width:100%;height:auto}a{color:#5A6378}header img{width:64px;height:64px}.icons{display:flex;gap:24px;flex-wrap:wrap}.icons figure{margin:0}.icons img{width:112px}.colors{display:flex;flex-wrap:wrap;gap:16px}.swatch{width:120px;height:60px;border-radius:12px;border:1px solid #e3e5ec}footer{margin-top:48px}@media(max-width:640px){main{padding:28px 16px}.samples{grid-template-columns:1fr}h1{font-size:32px}}</style>
<main><header><h1>Curated · Brand resources</h1><p>沿用现有核心标志、粉色与 Outfit 字体。Server / Web 深色底；Desktop / App 浅灰白底。Curated App 为 Android / iOS 移动客户端。</p></header>
<div class="icons"><figure><img src="../curated-appicon.png" alt="Server 深色底图标"><figcaption>Server · #141826</figcaption></figure><figure><img src="../curated-desktop-appicon.png" alt="Desktop 浅灰白底图标"><figcaption>Desktop · #F5F6F8</figcaption></figure><figure><img src="../curated-mark.png" alt="透明核心标志"><figcaption>通用核心标志</figcaption></figure></div>
'''
    html += ''.join(cards)
    html += '<section><h2>Palette</h2><div class="colors">'
    for label, color in spec['colors'].items():
        html += f'<div><div class="swatch" style="background:{color}"></div><p>{label}<br>{color}</p></div>'
    html += '</div></section><footer><p>品牌字体：Outfit 600 · 正文：Noto Sans / HarmonyOS Sans SC / Noto Sans JP。SVG 保留原 PNG 主字标并将后缀转为路径；不依赖外部字体，不是纯矢量重绘。</p><a href="../../docs/reference/curated-brand-guidelines.md">完整品牌规范</a></footer></main></html>'
    (output / 'index.html').write_text(html, encoding='utf-8')
    sheet = Image.new('RGB', (1440, 1220), spec['colors']['clientBackground'])
    draw = ImageDraw.Draw(sheet)
    heading_buffer = BytesIO(buffer.getvalue())
    heading_font = ImageFont.truetype(heading_buffer, 34)
    draw.text((56, 35), 'Curated / Brand family', font=heading_font, fill=spec['colors']['lightText'])
    for row, product in enumerate(spec['products']):
        y = 112 + row * 210
        name = 'curated' if product['id'] == 'curated' else f"curated-{product['id']}"
        draw.text((56, y), product['name'], font=heading_font, fill=spec['colors']['lightText'])
        image = Image.open(output / f'{name}-wordmark.png').convert('RGBA')
        image.thumbnail((600, 140), Image.Resampling.LANCZOS)
        for x, color in ((56, '#FFFFFF'), (752, spec['colors']['serverBackground'])):
            draw.rounded_rectangle((x, y + 52, x + 632, y + 182), 18, fill=color)
            sheet.paste(image, (x + 16, y + 52 + (130 - image.height) // 2), image)
    sheet.save(output / 'brand-overview.png')


if __name__ == '__main__':
    generate(Path(__file__).resolve().parents[2])

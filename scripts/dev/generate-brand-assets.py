"""Generate brand lockups, an offline preview and a contact sheet.

Requires Pillow, fonttools and brotli. Uses the locked local Outfit package.
The Curated master geometry and mark are retained without redrawing.
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
    transparent = output / 'transparent'
    transparent.mkdir(exist_ok=True)
    # The legacy transparent mark has small RGB variations in its antialiasing.
    # Keep its original alpha/geometry and make every visible pixel brand pink.
    original_mark = Image.open(root / 'icon/curated-mark.png').convert('RGBA')
    clean_mark = Image.new('RGBA', original_mark.size)
    pink = tuple(bytes.fromhex(spec['colors']['pink'].lstrip('#')))
    clean_mark.putdata([(*pink, alpha) if alpha else (0, 0, 0, 0)
                        for alpha in original_mark.getchannel('A').get_flattened_data()])
    clean_mark.save(transparent / 'curated-mark.png')
    mark_data = b64encode((transparent / 'curated-mark.png').read_bytes()).decode('ascii')
    (transparent / 'curated-mark.svg').write_text(
        '<svg xmlns="http://www.w3.org/2000/svg" width="322" height="322" viewBox="0 0 322 322" role="img" aria-label="Curated mark">'
        f'<title>Curated mark</title><image width="322" height="322" href="data:image/png;base64,{mark_data}"/></svg>',
        encoding='utf-8',
    )
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
        copyfile(png, transparent / f'{name}-wordmark.png')
        copyfile(output / f'{name}-wordmark.svg', transparent / f'{name}-wordmark.svg')
        if name != 'curated':
            copyfile(transparent / 'curated-mark.png', transparent / f'{name}-mark.png')
        cards.append(f'<section><h2>{product["name"]}</h2><div class="samples"><div class="light"><img src="{png.name}" alt="{product["name"]}"></div><div class="dark"><img src="{png.name}" alt="{product["name"]}"></div></div><p><a href="{png.name}">PNG</a> · <a href="{name}-wordmark.svg">SVG</a> · <a href="{product["icon"]}">图标</a></p></section>')

    transparent_cards = []
    for product in spec['products']:
        name = 'curated' if product['id'] == 'curated' else f"curated-{product['id']}"
        transparent_cards.append(
            f'<section><h2>{product["name"]}</h2><div class="sample"><img src="{name}-wordmark.png" alt="{product["name"]} 透明字标"></div>'
            f'<p><a href="{name}-wordmark.png">透明 PNG 字标</a> · <a href="{name}-wordmark.svg">透明 SVG 字标</a> · '
            f'<a href="{name}-mark.png">透明核心图案</a></p></section>'
        )
    transparent_html = '''<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Curated 透明背景品牌资源</title>
<style>@font-face{font-family:Outfit;src:url(../fonts/outfit-latin-wght-normal.woff2);font-weight:100 900}*{box-sizing:border-box}body{margin:0;background:#F5F6F8;color:#0F1219;font-family:system-ui,sans-serif}main{max-width:1000px;margin:auto;padding:42px 24px}h1,h2{font-family:Outfit,system-ui,sans-serif}h1{font-size:36px}section{margin:28px 0}.sample{border-radius:16px;border:1px solid #ccd1da;padding:20px;min-height:156px;display:flex;align-items:center;background-color:#fff;background-image:linear-gradient(45deg,#dce0e8 25%,transparent 25%),linear-gradient(-45deg,#dce0e8 25%,transparent 25%),linear-gradient(45deg,transparent 75%,#dce0e8 75%),linear-gradient(-45deg,transparent 75%,#dce0e8 75%);background-size:28px 28px;background-position:0 0,0 14px,14px -14px,-14px 0}.sample img{max-width:100%;max-height:116px}p{color:#5A6378;line-height:1.6}a{color:#5A6378}</style>
<main><h1>Curated · 透明背景资源</h1><p>棋盘格仅用于预览；下载的 PNG / SVG 没有背景。字标与核心图案均使用品牌粉色 #FE628E，边缘保留透明度。</p>'''
    transparent_html += ''.join(transparent_cards)
    transparent_html += '<p><a href="curated-mark.png">通用透明核心图案 PNG</a> · <a href="curated-mark.svg">通用透明核心图案 SVG</a> · <a href="../index.html">返回全部品牌资源</a></p></main></html>'
    (transparent / 'index.html').write_text(transparent_html, encoding='utf-8')
    # This contact sheet intentionally contains a checkerboard. The linked
    # files above keep genuine alpha; this image is only a visual index.
    preview = Image.new('RGB', (1200, 1170), '#F5F6F8')
    preview_draw = ImageDraw.Draw(preview)
    preview_font = ImageFont.truetype(BytesIO(buffer.getvalue()), 31)
    preview_draw.text((40, 25), 'Curated / Transparent assets', font=preview_font, fill='#0F1219')
    for row, product in enumerate(spec['products']):
        name = 'curated' if product['id'] == 'curated' else f"curated-{product['id']}"
        top = 93 + row * 210
        preview_draw.text((40, top), product['name'], font=preview_font, fill='#0F1219')
        left, top_box, right, bottom = 40, top + 46, 1160, top + 182
        for y in range(top_box, bottom, 24):
            for x in range(left, right, 24):
                preview_draw.rectangle((x, y, min(x + 23, right), min(y + 23, bottom)),
                                       fill='#DCE0E8' if ((x - left) // 24 + (y - top_box) // 24) % 2 else '#FFFFFF')
        wordmark = Image.open(transparent / f'{name}-wordmark.png').convert('RGBA')
        wordmark.thumbnail((870, 120), Image.Resampling.LANCZOS)
        preview.paste(wordmark, (left + 16, top_box + (136 - wordmark.height) // 2), wordmark)
        mini_mark = clean_mark.resize((126, 126), Image.Resampling.LANCZOS)
        preview.paste(mini_mark, (right - 152, top_box + 5), mini_mark)
    preview.save(transparent / 'preview.png')

    html = '''<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Curated 品牌资源</title>
<style>@font-face{font-family:Outfit;src:url(fonts/outfit-latin-wght-normal.woff2);font-weight:100 900}*{box-sizing:border-box}body{margin:0;background:#F5F6F8;color:#0F1219;font-family:system-ui,sans-serif}main{max-width:1120px;margin:auto;padding:48px 24px}h1,h2{font-family:Outfit,system-ui,sans-serif}h1{font-size:44px;margin:0}h2{font-size:24px}p{line-height:1.7;color:#5A6378}section{margin-top:36px}.samples{display:grid;grid-template-columns:1fr 1fr;gap:16px}.samples>div{border-radius:20px;padding:16px;min-width:0;display:flex;align-items:center;min-height:150px}.light{background:#fff;border:1px solid #e3e5ec}.dark{background:#141826}.samples img{width:100%;height:auto}a{color:#5A6378}header img{width:64px;height:64px}.icons{display:flex;gap:24px;flex-wrap:wrap}.icons figure{margin:0}.icons img{width:112px}.colors{display:flex;flex-wrap:wrap;gap:16px}.swatch{width:120px;height:60px;border-radius:12px;border:1px solid #e3e5ec}footer{margin-top:48px}@media(max-width:640px){main{padding:28px 16px}.samples{grid-template-columns:1fr}h1{font-size:32px}}</style>
<main><header><h1>Curated · Brand resources</h1><p>沿用现有核心标志、粉色与 Outfit 字体。Server / Web 深色底；Desktop / App 浅灰白底。Curated App 为 Android / iOS 移动客户端。</p></header>
<div class="icons"><figure><img src="../curated-appicon.png" alt="Server 深色底图标"><figcaption>Server · #141826</figcaption></figure><figure><img src="../curated-desktop-appicon.png" alt="Desktop 浅灰白底图标"><figcaption>Desktop · #F5F6F8</figcaption></figure><figure><img src="../curated-mark.png" alt="透明核心标志"><figcaption>通用核心标志</figcaption></figure></div>
'''
    html += ''.join(cards)
    html += '<section><h2>Palette</h2><div class="colors">'
    for label, color in spec['colors'].items():
        html += f'<div><div class="swatch" style="background:{color}"></div><p>{label}<br>{color}</p></div>'
    html += '</div></section><footer><p>品牌字体：Outfit 600 · 正文：Noto Sans / HarmonyOS Sans SC / Noto Sans JP。SVG 保留原 PNG 主字标并将后缀转为路径；不依赖外部字体，不是纯矢量重绘。</p><a href="transparent/index.html">透明背景资源（棋盘格预览）</a> · <a href="../../docs/reference/curated-brand-guidelines.md">完整品牌规范</a></footer></main></html>'
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

# Curated 品牌源图

`icon/` 只放设计源文件。文件名全部小写 kebab-case：`curated-<用途>.png`。口头提及时用中间那个词即可。

| 怎么叫 | 文件 | 是什么 |
| --- | --- | --- |
| **wordmark** | `curated-wordmark.png` | 标志 + “Curated” 文字，透明底。README 顶部用这个。 |
| **appicon** | `curated-appicon.png` | 圆角深色底 + 标志。托盘、安装包、Web favicon 的源图。 |
| **mark** | `curated-mark.png` | 只有核心标志（四角星 + 加号 + 圆点），透明底。 |

前端运行时副本在 `src/icon/`，文件名与上表相同。

改 `appicon` 之后，还要同步派生：

- `src/icon/curated-appicon.png`
- `public/Curated-icon.png`
- `dist/Curated-icon.png`、`backend/frontend-dist/Curated-icon.png`（若本地构建目录存在）
- `backend/internal/assets/curated.ico`

应用图标统一使用当前 322 × 322 的居中源图。粉色图案的上下留白均为 79 px，左右留白为 78 / 79 px；奇偶像素带来的半像素中心差允许保留，避免重复重采样使边缘模糊。PNG 副本应与源图字节一致，ICO 从源图生成 16、20、24、32、40、48、64、128、256 px 共 9 个尺寸。

macOS Dock 单独使用 `public/Curated-icon-macos.png`：将 322 × 322 源图原尺寸居中放进 384 × 384 透明画布，四边各留 31 px，主体占约 84%。此派生图不做重采样，避免 Dock 中深色底铺满画布而显得过大；Web、Windows 和托盘继续使用原资源。修改 `appicon` 后也需要重新生成此文件（Python + Pillow）：

```python
from PIL import Image

source = Image.open("icon/curated-appicon.png").convert("RGBA")
canvas_size = 2 * round(max(source.size) / 0.84 / 2)
canvas = Image.new("RGBA", (canvas_size, canvas_size))
canvas.alpha_composite(source, ((canvas_size - source.width) // 2, (canvas_size - source.height) // 2))
canvas.save("public/Curated-icon-macos.png")
```

改 `mark` 后同步 `src/icon/curated-mark.png`。`wordmark` 是横向字标，保留标志与文字的组合排版。浏览器图标更新时同步更新根 `index.html` 中图标 URL 的版本参数，避免旧缓存；不再保留未使用的模板 `public/favicon.svg`。

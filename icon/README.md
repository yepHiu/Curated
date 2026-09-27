# Curated 品牌源图

完整的 Curated / Server / Desktop / App（Android / iOS）/ Web 品牌家族规范见 [品牌规范](../docs/reference/curated-brand-guidelines.md)。组合字标、明确产品命名的图标副本、移动图标源、字体许可与离线预览放在 [`brand/`](brand/index.html)，生成入口为 `python scripts/dev/generate-brand-assets.py`（Pillow、fonttools、brotli）。

`icon/` 放品牌源文件与 Desktop 的 Windows ICO 派生资源。文件名全部小写 kebab-case。

| 怎么叫 | 文件 | 是什么 |
| --- | --- | --- |
| **wordmark** | `curated-wordmark.png` | 标志 + “Curated” 文字，透明底。README 顶部用这个。 |
| **Server appicon** | `curated-appicon.png` | 圆角深色底（`#141826`）+ 标志。Server 托盘、安装包、Web favicon 的源图。 |
| **Desktop appicon** | `curated-desktop-appicon.png` | 圆角浅灰白底（`#F5F6F8`）+ 同一标志。Desktop 窗口、托盘、安装包与快捷方式使用。 |
| **mark** | `curated-mark.png` | 只有核心标志（四角星 + 加号 + 圆点），透明底。 |

Server 与透明标志的前端副本在 `src/icon/`。Desktop 独立使用 `public/Curated-desktop-icon.png` 与 `icon/curated-desktop.ico`，不回退到 Server 的深色图标。

改 `appicon` 之后，还要同步派生：

- `src/icon/curated-appicon.png`
- `public/Curated-icon.png`
- `dist/Curated-icon.png`、`backend/frontend-dist/Curated-icon.png`（若本地构建目录存在）
- `backend/internal/assets/curated.ico`

应用图标统一使用当前 322 × 322 的居中源图。粉色图案的上下留白均为 79 px，左右留白为 78 / 79 px；奇偶像素带来的半像素中心差允许保留，避免重复重采样使边缘模糊。PNG 副本应与源图字节一致，ICO 从源图生成 16、20、24、32、40、48、64、128、256 px 共 9 个尺寸。

### Desktop 图标生成

使用现有 `curated-appicon.png` 的透明度保留圆角轮廓，在冷调浅灰白底上叠加未移动、未缩放的 `curated-mark.png`。核心图案形状、位置与粉色保持不变；边缘透明度在白底重新合成，避免深色描边。修改任一源图后运行（需要 Python + Pillow）：

```powershell
python scripts/dev/generate-desktop-icon.py
```

生成 `icon/curated-desktop-appicon.png`、`public/Curated-desktop-icon.png`、`icon/curated-desktop.ico`（上述 9 个尺寸）及 `public/Curated-desktop-icon-macos.png`。

macOS Dock 与安装包 ICNS 使用后者：322 × 322 白底图原尺寸居中放进 384 × 384 透明画布，四边各留 31 px，主体占约 84%，不做重采样。完全退出并重新启动 Desktop 后生效；已安装版本的快捷方式与包图标在更新安装包后生效。

Windows Desktop 打包将白底 ICO 放到安装目录的 `curated.ico`（安装器 / 卸载 / 快捷方式）和 `resources/app/curated-desktop.ico`（Electron 窗口 / 托盘）。Server 仍打包 `backend/internal/assets/curated.ico`。旧一体包的 Electron 壳同样使用白底；其中 Go Server 的内嵌托盘图标仍是深色。

改 `mark` 后同步 `src/icon/curated-mark.png`。`wordmark` 是横向字标，保留标志与文字的组合排版。浏览器图标更新时同步更新根 `index.html` 中图标 URL 的版本参数，避免旧缓存；不再保留未使用的模板 `public/favicon.svg`。

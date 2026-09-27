# Curated 品牌资源规范

更新：2026-09-27。依据现有 `icon/` 源图、前端 UI 规范、字体实施记录、Server / Desktop / Web 命名记录，以及本次用户确认的 App 定位整理。

## 1. 品牌与产品名称

| 正式名称 | 定位 | 图标背景 | 使用场景 |
|---|---|---|---|
| Curated | 产品家族 / 总品牌 | 通用透明核心标志；历史应用图标保留深色 | README、品牌介绍、资料库内部导航、水印 |
| Curated Server | 资料库服务端 | `#141826` 深色 | 服务端发行、安装包、托盘、部署说明 |
| Curated Desktop | 桌面客户端 | `#F5F6F8` 浅灰白 | Windows / macOS 客户端、连接页、安装包、快捷方式 |
| Curated App | Android / iOS 移动客户端 | `#F5F6F8` 浅灰白 | 移动客户端、启动图、商店素材、移动产品说明 |
| Curated Web | 随 Server 交付的浏览器入口 | 沿用 Server 深色 favicon | 浏览器入口、访问说明；不据此新增独立发行组件 |

正式拼写保留大小写和一个空格：`Curated Server`、`Curated Desktop`、`Curated App`、`Curated Web`。App 与 Web 是不同入口。Server 使用单数；Servers 仅表示服务器列表。安装组合 Full 是发行组合，不另造一套品牌标志。

产品身份由名称和后缀补充表达，不给同一核心标志增加设备、云、齿轮或字母，也不为不同平台另配品牌主色。App / Desktop 共用客户端图标是有意的家族关系；并列介绍时用完整名称区分。

## 2. 字体

| 用途 | 字体 / 字重 | 来源与约定 |
|---|---|---|
| Curated 主字标 | 保留 `icon/curated-wordmark.png` 原始形状 | 已去除旧资源边缘预混入的深色；原图归档在 `icon/brand/source/` |
| 新增 Server / Desktop / App / Web 后缀 | Outfit 600 | 锁定的 `@fontsource-variable/outfit`，随资源包附 OFL |
| 界面可编辑品牌标题 | Outfit 600 / `font-curated` | 沿用侧栏、关于页的既有字体；使用 `text-primary` |
| 英文与数字正文 | Noto Sans | 沿用本地可变字体 |
| 中文正文 | HarmonyOS Sans SC | 原字体与许可不修改、不裁剪 |
| 日文正文 | Noto Sans JP | 按界面语言回退字形 |

品牌名称保持英文，不把品牌字体替换成正文 CJK 字体。产品后缀使用相同 Outfit，不用全大写、不用斜体、不人工拉宽压扁。界面品牌标题延续现有 `font-semibold tracking-wide`；导出的组合字标使用字体自然间距，避免重复对图片加字距。

现有主字标是栅格设计源，不能由界面 `font-curated` 的字号 / 字重推断其精确原始字体参数。新增后缀参照 Outfit，并保留主字标原始像素，避免借此次整理改变原有品牌。

## 3. 配色

| 名称 | 值 | 用途 |
|---|---|---|
| 品牌粉色 | `#FE628E` | 核心图案、字标、UI `primary`；明暗主题保持一致 |
| Server 深色 | `#141826` | Server 图标底色、深色品牌展示底 |
| 客户端浅灰白 | `#F5F6F8` | Desktop / App 图标底色；用户已确认替代纯白 |
| 浅色正文 | `#0F1219` | 浅色展示面的正文参考；UI 用 `foreground` token |
| 深色正文 | `#F8F7FB` | 深色展示面的正文参考；UI 用 `foreground` token |

图标身份色与 UI 主题是两种用途。Desktop 在深色界面仍使用浅灰白应用图标；Server 在浅色界面仍使用深色图标。普通页面继续遵守 `src/style.css` 的语义主题 token，不把图标底色强制套到整个应用背景。

粉色字标用于品牌展示。小号正文、帮助文字及需要阅读的产品说明用正常正文色，不把粉色当正文色或状态色。没有新增产品专属渐变、光晕、描边或阴影。

## 4. 图案与组合字标

- 核心图案沿用 `icon/curated-mark.png`：四角星、加号、圆点保持形状、相对位置和粉色。
- Server / Desktop 图标保持 322 × 322 画布、现有圆角轮廓和原始中央图案居中关系；不能移动、放大其中某个元素。
- 字标采用透明背景，图案在左、Curated 在右、产品后缀接在其后；统一水平排列，共用原字标基线。
- 当前导出主字标画布为 1085 × 322，原文字墨迹右缘约 x=960；后缀起点 x=1016，视觉间隔 56 源像素。后缀使用 Outfit 154 源像素、600 字重、baseline y=218；整体按比例缩放。
- 保护区至少为实际图案高度的 1/4；不把文字、边框或图片放入保护区。导出已有外边距，嵌入时不要随意裁掉。
- 推荐透明组合字标显示高度至少 48 CSS px；紧凑导航使用现有可编辑 Outfit 品牌标题 + 核心图案，小图标不塞入产品后缀。
- 不旋转、倾斜、变形、重新描边、改粉色，或把 App / Desktop 背景按主题自动反转。

## 5. 资源格式与入口

资源包：[`icon/brand/`](../../icon/brand/)。可视预览：[`index.html`](../../icon/brand/index.html)；总览图：[`brand-overview.png`](../../icon/brand/brand-overview.png)。机器可读名称、字体、颜色与产品映射在 `brand.json`。

| 资源 | 内容 |
|---|---|
| `curated-wordmark.{png,svg}` | 总品牌字标 |
| `curated-{server,desktop,app,web}-wordmark.{png,svg}` | 四个产品 / 入口组合字标 |
| `curated-{server,desktop,app,web}-appicon.png` | 对应现有图标的明确命名副本 |
| `curated-app-icon-1024.png` | 1024 px、RGB 无透明、无预制圆角的移动图标源 |
| `curated-app-android-foreground.png` | 432 px 透明前景；背景颜色使用 `#F5F6F8` |
| `fonts/outfit-latin-wght-normal.woff2`、`fonts/Outfit-OFL.txt` | 离线预览字体与原许可 |

PNG 是现有栅格品牌的交付资源。SVG 包含内嵌的去暗边主字标 PNG 与转为路径的后缀，离线使用且无需安装字体；它是混合 SVG，**不是纯矢量重绘**，不能宣称无限分辨率。原有核心图案的真正矢量设计源尚未找到；大幅印刷应先补足原始设计源。

历史主字标 `icon/brand/source/curated-wordmark-original.png` 的边缘 RGB 预混入 `#141826`，部分暗边甚至是完全不透明像素。放在白底上会出现黑色轮廓。当前交付文件根据旧图的粉色与深色底恢复边缘覆盖率，仅使用 `#FE628E` 加透明度绘制，保留原有画布、墨迹边界和完整粉色像素。`src/icon/curated-wordmark.png` 与此源同步，避免水印导出保留旧黑边。

移动端图标源单独导出为不透明方图，由系统 / 商店施加圆角或自适应蒙版，不能直接提交带透明圆角的 Desktop PNG。Android 前景保留中心图案比例，另用纯色背景层；实际 mipmap、adaptive icon XML、Android 单色主题图标与 iOS Asset Catalog 由各移动仓库集成时按平台要求导出。本资源包不表示移动端已接入或已发布。

## 6. 生成与维护

在仓库根目录，安装 Python Pillow、fonttools、brotli 并安装锁定的 pnpm 依赖后运行：

```powershell
python scripts/dev/clean-brand-wordmark.py
python scripts/dev/generate-desktop-icon.py
python scripts/dev/generate-brand-assets.py
```

第一个命令清理原主字标的深色预混边缘并同步前端水印资源；第二个同步当前 Desktop PNG、Windows 9 尺寸 ICO 和 macOS Dock 图；第三个生成品牌家族字标、平台命名副本、移动图标源与离线预览。Outfit 源来自锁定的本地 Fontsource 包，不在线下载另一版本。保留其许可及署名。

Web favicon / Go Server / Electron / 安装包现有链路见 [`icon/README.md`](../../icon/README.md)。资源更新后完整退出再启动 Desktop；已安装包的快捷方式、Dock bundle 图标须后续重建 / 更新安装包。对其它仓库进行 Android / iOS 或插件引用更新时，再分别检查它们的规则与构建入口。

## 7. 参考记录

- [前端 UI 规范](2026-03-24-frontend-ui-spec.md)：语义颜色、Outfit 品牌标题与 CJK 正文字体。
- [字体实施记录](../plan/2026-09-25-font-rollout.md)：字体来源、原始字体分发与许可。
- [组件命名记录](../plan/2026-09-25-desktop-server-connection-and-ssdp.md)：Server / Desktop / Web 的职责及发行关系。
- [图标同步记录](../plan/2026-04-13-desktop-icon-sync-plan.md)：现有源图、居中关系、Server / Desktop 图标底色。

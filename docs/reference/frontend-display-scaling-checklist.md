# Curated 前端显示缩放检查清单

本文用于检查 Curated 前端在多端屏幕、不同分辨率、不同 DPR、浏览器缩放和系统显示缩放下的 UI 稳定性。凡是修改壳层布局、网格、卡片、播放器 HUD、设置页、对话框、全局字体或全局 spacing，都应按本清单做回归。

## 1. 必查浏览器

| 平台 | 浏览器 | 检查重点 |
| --- | --- | --- |
| macOS | Safari 最新版 | WebKit 字体度量、Retina 渲染、overlay scrollbar、`dvh`、`backdrop-filter` |
| macOS | Chrome 最新版 | Chromium 在 Retina / HiDPI 下的字体、滚动、transform 表现 |
| Windows | Chrome 或 Edge | 普通屏、保留滚动条、系统缩放 100% / 125% / 150% |
| Playwright | Chrome | Windows 自动化主路径回归，使用本机 Chrome channel |
| Playwright | Firefox | Windows 自动化跨浏览器回归 |

## 2. 必查 viewport

以下尺寸均指 CSS viewport，不是物理像素。

| 类型 | Viewport | 说明 |
| --- | --- | --- |
| 小屏手机 | `320 x 568` | iPhone SE 类下限 |
| 手机 | `375 x 812` | 常见 iPhone 竖屏 |
| 大屏手机 | `430 x 932` | 大屏手机竖屏 |
| 平板竖屏 | `768 x 1024` | iPad 类 |
| 小桌面 / 平板横屏 | `1024 x 768` | 断点边界 |
| MacBook Air 逻辑尺寸 | `1280 x 832` | 常见 macOS 缩放工作区 |
| 桌面基准 | `1440 x 900` | 常见开发尺寸 |
| MacBook Pro 逻辑尺寸 | `1512 x 982` | 常见 Retina 逻辑工作区 |
| 外接显示器 | `1920 x 1080` | 普通桌面 |
| 高分外接显示器 | `2560 x 1440` | 大屏高分 |

## 3. 必查 DPR

| DPR | 目的 |
| --- | --- |
| `1` | 普通外接显示器 / Windows 基准 |
| `1.25` | 系统缩放近似 |
| `1.5` | Windows 常见缩放近似 |
| `2` | macOS Retina |
| `3` | 高密移动设备 |

## 4. 必查浏览器缩放

真实浏览器缩放不能完全由 Playwright `deviceScaleFactor` 代替。涉及视觉或布局的改动至少手动检查：

- `90%`
- `100%`
- `125%`
- `150%`

Mac 端优先检查 Safari 与 Chrome；Windows 端优先检查 Chrome 或 Edge。

## 5. 必查页面

- Home / 首页聚合区。
- Library / 资料库网格。
- Detail / 详情页。
- Player / 播放器页。
- Actors / 演员页。
- History / 历史记录。
- Curated Frames / 精选帧。
- Settings / 设置页。
- 图片预览、确认弹窗、编辑弹窗等主要对话框。

## 6. 页面级验收项

每个被检查页面都必须满足：

- 无非预期横向 document scroll。
- 主文本不被截断。
- 按钮、输入框、标签、图标不重叠。
- 固定 header、sticky nav、sidebar、播放器控制条不遮挡内容。
- 内部滚动区域可用鼠标、触控板和键盘访问。
- 表单控件边界清晰，文字可读。
- 对话框不超出 viewport；内容超出时有明确内部滚动。
- poster、封面、预览图保持稳定比例，不因亚像素舍入出现半列或错位。
- light theme 与 dark theme 都保持足够视觉对比。

## 7. 组件级验收项

### 壳层与导航

- `AppShell` 主内容区使用 `minmax(0, 1fr)`、`min-w-0` 或等价约束，避免被子内容撑破。
- desktop sidebar、collapsed sidebar、mobile drawer 使用统一尺寸 token 或同源约束。
- 顶部搜索、导航按钮、折叠按钮在 125% / 150% 缩放下仍可用。

### 设置页与表单

- 设置项 label 不因固定列宽被截断。
- 路径、provider、版本号等长文本要么可换行，要么有明确截断策略。
- 横向 action row 在窄容器下允许换行。
- 表单控件不依赖极浅边框或透明底来表达状态。

### 播放器 HUD

- 核心 icon button 保持稳定点击区域。
- icon-only button 必须有可访问名称。
- 进度条、时间、状态提示在 125% / 150% 缩放下不互相挤压。
- 带文字的元素避免直接放在明显 transform scale 的层中。
- 浏览器前端不假设 Electron、mpv 或原生播放器能力一定存在，相关能力判断仍通过服务层。

### 网格、卡片与标签

- 海报网格不出现半列、错位或明显宽度抖动。
- 媒体区域优先使用 `aspect-ratio`，避免固定高度和动态内容冲突。
- 卡片标题、演员名、标签行在缩放后不和操作按钮重叠。
- chip/tag 不应使用硬固定高度承载可变长度文本。

### 对话框与 overlay

- 对话框最大高度使用 viewport 约束，移动端和 Safari 可见场景优先考虑 `dvh`。
- 标题、关闭按钮、主要操作按钮始终可见或可滚动到。
- 图片预览主图不被缩略图 rail、header、footer 遮挡。

## 8. 自动化检查建议

Windows 本地自动化显示缩放检查只覆盖 Chrome 与 Firefox，不在 Windows 下使用 WebKit 作为 Safari 近似。Safari/WebKit 的最终判断保留在 macOS 手动检查中。

自动化冒烟测试应优先检查：

- `document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1`。
- `data-scaling-critical` 元素无明显 `scrollWidth > clientWidth` 或 `scrollHeight > clientHeight`。
- 每个核心路由至少存在可见按钮或主要可交互控件。
- 失败时保存当前 viewport、DPR、浏览器和截图。

## 9. 人工记录要求

每轮显示适配修复后，至少记录：

- 检查日期。
- 检查平台与浏览器。
- viewport / DPR / 浏览器缩放。
- 失败页面和截图位置。
- 已修复项与剩余 follow-up。

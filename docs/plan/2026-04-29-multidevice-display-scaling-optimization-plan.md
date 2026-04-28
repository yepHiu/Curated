# Curated 多端屏幕与显示缩放优化文档

日期：2026-04-29

## 执行跟踪

| 阶段 | 状态 | 说明 |
| --- | --- | --- |
| 阶段 1：沉淀显示缩放检查清单 | 已完成 | 已新增 `docs/reference/frontend-display-scaling-checklist.md`，并在 UI 规范与 Cursor UI 规则中引用 |
| 阶段 2：增加自动化显示缩放冒烟测试 | 待开始 | 后续新增 Playwright 显示缩放测试 |
| 阶段 3：增加共享显示适配 tokens | 待开始 | 后续更新 `src/style.css` 与可选测试工具 |
| 阶段 4：AppShell 布局稳定化 | 待开始 | 后续处理壳层尺寸 token 与关键区域标记 |
| 阶段 5：设置页适配 | 待开始 | 后续处理设置页固定列宽、换行和关键区域标记 |
| 阶段 6：播放器 HUD 适配 | 待开始 | 后续处理播放器控制区、字号和能力判断展示 |
| 阶段 7：网格、卡片、标签适配 | 待开始 | 后续处理 poster grid、卡片标题和 chip 高度 |
| 阶段 8：对话框和 overlay 适配 | 待开始 | 后续处理 dialog、图片预览和 viewport 高度 |

## 1. 需求理解

目前在 Mac 端浏览器中出现的 UI 渲染“不好看”，不应简单理解为某一个组件的样式缺陷。更准确地说，这是 Curated 前端在不同设备、分辨率、浏览器缩放、系统显示缩放和浏览器渲染引擎下，还没有形成系统化适配与回归验证的问题。

这类问题常见触发条件包括：

- macOS Retina / HiDPI 屏幕，常见 `devicePixelRatio = 2`。
- 外接显示器的系统缩放，例如 100%、125%、150%。
- 浏览器缩放，例如 90%、100%、125%、150%。
- Safari/WebKit 与 Chrome/Chromium 在字体度量、行高、滚动条、`backdrop-filter`、亚像素渲染上的差异。
- 固定宽高、极小字号、任意 Tailwind 值、transform 缩放、嵌套滚动容器在缩放后产生的文本截断、控件重叠或模糊。
- macOS overlay scrollbar 与 Windows 保留滚动条宽度的差异。

因此，本次优化的目标不是“给 Mac 单独写一套样式”，而是建立一套跨屏幕、跨 DPR、跨缩放比例、跨浏览器的前端显示适配标准，并按高风险页面逐步落地。

## 2. 优化目标

本优化面向 Curated 当前的 Vue 3 + TypeScript + Vite + Tailwind v4 + shadcn-vue 前端，不改变现有业务分层，不引入大规模状态管理重构。

目标如下：

- 在 macOS Safari、macOS Chrome、Windows Chrome/Edge 中保持一致可读、稳定、无明显错位的 UI。
- 在 Retina、普通屏、外接高分屏和常见系统缩放比例下，页面布局不出现横向溢出、文本截断、控件重叠。
- 在浏览器缩放 90%、100%、125%、150% 下，核心页面仍可操作。
- 建立显示缩放测试矩阵和自动化冒烟测试，避免后续 UI 改动再次引入同类问题。
- 将适配能力沉淀为设计规范、CSS tokens、组件约束和测试工具，而不是依赖临时页面补丁。

## 3. 非目标

以下内容不作为本轮文档建议的直接目标：

- 不做全站视觉风格重设计。
- 不一次性替换所有任意 Tailwind class。
- 不为每一种物理分辨率写独立样式。
- 不把浏览器缩放或 DPR 当作业务状态接入 Pinia/Vuex。
- 不为了适配问题引入新的全局状态管理库。后续如需引入 Pinia，也应从小服务、小模块逐步试点，不作为本轮显示适配的前置条件。
- 不把 Electron、mpv 或原生播放器能力作为浏览器前端可用性的默认前提。

## 4. 当前代码风险观察

基于现有项目结构和 UI 规范，优先关注以下区域。

### 4.1 `src/style.css`

当前 `html`、`body`、`#app` 采用 `height: 100%` 和 `overflow: hidden` 的应用壳模型。这有利于桌面应用式布局，但也意味着滚动问题会集中在内部容器中。

风险：

- 内部滚动容器高度计算错误时，内容可能被固定 header、sidebar 或底部控件遮挡。
- Safari 动态视口变化、移动端浏览器地址栏变化、浏览器缩放会放大这些问题。
- 全局暂未形成显示缩放相关的 CSS token，例如 header 高度、sidebar 宽度、内容 gutter、最小触控尺寸等。

### 4.2 `src/layouts/AppShell.vue`

当前 shell 使用固定 sidebar 宽度、固定 collapsed 宽度和固定 header 高度。

风险：

- 125% 或 150% 浏览器缩放下，搜索框、导航项、折叠按钮、用户区域可能发生拥挤。
- 小尺寸桌面窗口下，sidebar 与主内容区可能互相挤压。
- mobile drawer 使用固定宽度表达，需要统一到 token，避免后续多个地方产生不一致。

### 4.3 `src/components/jav-library/SettingsPage.vue`

设置页是高风险页面：控件密集、表单行多、说明文字多、存在 sticky 导航和嵌套滚动。

风险：

- 固定 grid column 在缩放后截断 label。
- 横向排列的按钮和说明文字在窄宽度下重叠。
- sticky 区域和内部滚动容器在 Safari 下可能产生遮挡。

### 4.4 `src/components/jav-library/PlayerPage.vue`

播放器页包含沉浸式 HUD、进度条、图标按钮、状态提示、播放器控制区。

风险：

- `text-[10px]`、`text-[11px]` 或过小控件在 Retina/Safari 下可读性差。
- 带 transform 的文本在 Retina 上可能变软或模糊。
- 缩放后控制区按钮可能互相挤压，进度条和时间信息可能被截断。

### 4.5 `HomeHeroCarousel`、卡片网格与图片预览

相关文件：

- `src/components/jav-library/HomeHeroCarousel.vue`
- `src/components/jav-library/VirtualMovieMasonry.vue`
- `src/components/jav-library/ActorLibraryCard.vue`
- `src/components/jav-library/ActorProfileCard.vue`
- `src/components/jav-library/PreviewImageViewer.vue`
- `src/components/jav-library/PreviewImageViewerInner.vue`

风险：

- hero carousel 使用 transform、scale、clamp 高度和卡片宽度变量，容易出现亚像素模糊、裁切或溢出。
- poster grid 在不同 CSS 像素宽度和 DPR 下可能出现半列、错位或卡片宽度抖动。
- 图片预览依赖 viewport 高度，移动 Safari 或缩放时容易出现关闭按钮不可达、缩略图条遮挡、主图裁切。

## 5. 显示适配原则

### 5.1 以 CSS 像素为布局基准

普通布局不要根据 DPR 写多套逻辑。浏览器布局以 CSS pixel 为基准，DPR 只是物理像素和 CSS 像素之间的映射。

DPR 专项逻辑只适合：

- canvas 渲染。
- 图片导出。
- 高分辨率媒体资源选择。
- 自动化截图校验。
- 极细 border 的视觉微调。

不建议为普通布局写 `if DPR = 2` 这种分支。

### 5.2 组件优先响应容器，而不是只响应 viewport

Curated 是桌面应用式 shell，页面实际可用宽度受 sidebar、drawer、header、内部面板影响。组件应该更多根据父容器空间自适应，而不是只看全局 viewport。

推荐：

- 使用 `minmax(0, 1fr)` 避免 grid/flex 子项撑破容器。
- 对可收缩文本容器加 `min-w-0`。
- 对媒体和卡片使用 `aspect-ratio`。
- 对工具栏使用 wrap 或分组折叠，而不是强行单行固定宽度。
- 后续可对高复用卡片和设置项引入 CSS container queries。

### 5.3 避免依赖极小字号维持密度

Mac Safari 的字体度量和抗锯齿表现与 Windows Chrome 不同，过小字号会更明显地暴露可读性问题。

建议：

- 正文、设置项、表单说明默认不低于 `text-sm`。
- `text-xs` 只用于元信息、badge、辅助信息或 HUD 次要信息。
- `text-[10px]`、`text-[11px]` 仅用于非核心状态或调试性信息，并需要在 125%/150% 缩放下检查。
- 控件内文字不要依赖固定高度强行压缩。

### 5.4 固定宽高必须有明确理由

允许固定尺寸的场景：

- icon button。
- 媒体比例盒。
- 虚拟列表的测量估算。
- 播放器控制按钮。
- 内容已经明确 clamp 的卡片。

应避免固定尺寸的场景：

- 带本地化文本的设置行。
- 表单项。
- 可换行标签。
- 对话框。
- 动态内容卡片。

推荐替代：

- `min-h-*` 替代硬固定 `h-*`。
- `max-h` + 内部滚动用于对话框。
- `clamp()` 用于大块视觉区域。
- `aspect-ratio` 用于图片、poster、视频预览。

### 5.5 不把业务逻辑放入 UI 适配层

显示适配可以新增 CSS tokens、测试工具函数和 Composables，但不应让页面组件直接承担业务逻辑。

约束：

- UI 组件只处理展示、交互状态和布局适配。
- 业务数据、播放器能力、文件扫描、收藏评分等仍通过服务层和现有 Composables 进入 UI。
- 显示缩放检测如果需要抽象，应放在 `src/lib` 或独立 Composable 中，避免散落在页面组件里。
- 当前阶段不因为适配问题引入 Pinia/Vuex。Pinia 可在后续功能中从小模块试点引入，而不是一次性大改。

## 6. 测试矩阵

### 6.1 浏览器矩阵

| 平台 | 浏览器 | 目的 |
| --- | --- | --- |
| macOS | Safari 最新版 | WebKit、Retina、overlay scrollbar、字体度量 |
| macOS | Chrome 最新版 | Chromium 在 Retina 下的表现 |
| Windows | Chrome 或 Edge | 普通屏和系统缩放基准 |
| Playwright | WebKit | 自动化近似 Safari 回归 |
| Playwright | Chromium | 自动化主路径回归 |

### 6.2 viewport 矩阵

| 类型 | CSS viewport | 说明 |
| --- | --- | --- |
| 小屏手机 | `320 x 568` | iPhone SE 类下限 |
| 手机 | `375 x 812` | 常见 iPhone 竖屏 |
| 大屏手机 | `430 x 932` | 大屏手机竖屏 |
| 平板竖屏 | `768 x 1024` | iPad 类 |
| 小桌面 / 平板横屏 | `1024 x 768` | breakpoint 边界 |
| MacBook Air 逻辑尺寸 | `1280 x 832` | 常见 macOS 缩放工作区 |
| MacBook Pro 逻辑尺寸 | `1512 x 982` | 常见 Retina 逻辑工作区 |
| 桌面基准 | `1440 x 900` | 常见开发尺寸 |
| 外接显示器 | `1920 x 1080` | 普通桌面 |
| 高分外接显示器 | `2560 x 1440` | 大屏高分 |

### 6.3 DPR 矩阵

| DPR | 用途 |
| --- | --- |
| `1` | 普通外接显示器 / Windows 基准 |
| `1.25` | 系统缩放近似 |
| `1.5` | Windows 常见缩放近似 |
| `2` | macOS Retina |
| `3` | 高密移动设备 |

### 6.4 浏览器缩放矩阵

手动检查至少覆盖：

| 缩放 | 目的 |
| --- | --- |
| `90%` | 用户压缩视图偏好 |
| `100%` | 基准 |
| `125%` | 常见缩放 |
| `150%` | 可访问性和大字号压力测试 |

Playwright 的 `deviceScaleFactor` 不能完全等同真实浏览器缩放，所以最终判断仍需要 macOS Safari/Chrome 手动检查。

## 7. 验收标准

核心页面在测试矩阵下应满足：

- 无非预期横向滚动。
- 主文本不被截断。
- 按钮、标签、输入框、图标不重叠。
- 固定 header、sticky nav、sidebar 不遮挡内容。
- 内部滚动区域可用鼠标、触控板和键盘访问。
- icon-only button 保持稳定方形点击区域，并有可访问名称。
- 表单控件边界清晰，内容可读。
- 对话框不超出 viewport，内容超出时使用内部滚动。
- poster grid 不因亚像素舍入出现半列或错位。
- 播放器控制区在 125% 和 150% 缩放下仍可操作。
- 设置页侧边导航和内容区域在平板、小桌面宽度下不互相挤压。
- light/dark theme 均保持足够视觉对比。

## 8. 实施计划

建议按小步提交推进，每个阶段都能独立验证和回滚。

### 阶段 1：沉淀显示缩放检查清单

文件：

- 新增 `docs/reference/frontend-display-scaling-checklist.md`
- 更新 `docs/reference/2026-03-24-frontend-ui-spec.md`
- 更新 `.cursor/rules/ui-component-spec.mdc`

内容：

- 写入 viewport、DPR、浏览器缩放、浏览器矩阵。
- 明确无横向滚动、无截断、无重叠、对话框可滚动、点击区域尺寸等检查项。
- 在 UI 规范和 Cursor 规则中引用该清单。

建议提交：

```powershell
git add docs/reference/frontend-display-scaling-checklist.md docs/reference/2026-03-24-frontend-ui-spec.md .cursor/rules/ui-component-spec.mdc
git commit -m "docs(ui): add display scaling checklist"
```

### 阶段 2：增加自动化显示缩放冒烟测试

文件：

- 新增 `tests/display-scaling/display-scaling.spec.ts`
- 新增 `tests/display-scaling/routes.ts`
- 更新 `package.json`
- 如新增依赖则更新 `pnpm-lock.yaml`

建议测试内容：

- 使用 Playwright Chromium + WebKit。
- 遍历核心路由：Home、Library、Actors、History、Curated Frames、Settings、Player 或可直接访问的播放器状态页。
- 覆盖桌面、平板、手机 viewport。
- 覆盖 DPR 1、1.5、2、3。
- 检查 `documentElement.scrollWidth > clientWidth`。
- 对 `data-scaling-critical` 元素检查文本溢出。
- 为失败页面保存截图到 `test-results/display-scaling/`。

建议脚本：

```json
{
  "scripts": {
    "test:display": "playwright test tests/display-scaling/display-scaling.spec.ts"
  }
}
```

启动开发环境：

```powershell
powershell -ExecutionPolicy Bypass -File .agents\skills\curated-dev-start\scripts\start-curated-dev.ps1
```

执行：

```powershell
pnpm test:display
```

建议提交：

```powershell
git add package.json pnpm-lock.yaml tests/display-scaling
git commit -m "test(ui): add display scaling smoke tests"
```

### 阶段 3：增加共享显示适配 tokens

文件：

- 更新 `src/style.css`
- 可选新增 `src/lib/display-scaling.ts`
- 可选新增 `src/lib/display-scaling.test.ts`

建议在全局样式中沉淀：

```css
:root {
  --app-header-min-height: 4.5rem;
  --app-sidebar-width: 19rem;
  --app-sidebar-collapsed-width: 4.75rem;
  --app-content-gutter-sm: 1rem;
  --app-content-gutter-md: 1.25rem;
  --app-content-gutter-lg: 1.5rem;
  --app-readable-measure: 72ch;
  --app-min-touch-target: 2.75rem;
}
```

建议增加动态 viewport 工具类：

```css
@supports (height: 100dvh) {
  .app-dvh {
    min-height: 100dvh;
  }
}
```

可选工具函数：

```ts
export function hasUnintendedHorizontalOverflow(root: Pick<Element, "scrollWidth" | "clientWidth">): boolean {
  return root.scrollWidth > root.clientWidth + 1
}

export function isLikelyClipped(
  root: Pick<HTMLElement, "scrollWidth" | "clientWidth" | "scrollHeight" | "clientHeight">,
): boolean {
  return root.scrollWidth > root.clientWidth + 2 || root.scrollHeight > root.clientHeight + 2
}
```

验证：

```powershell
pnpm test -- src/lib/display-scaling.test.ts
pnpm typecheck
```

建议提交：

```powershell
git add src/style.css src/lib/display-scaling.ts src/lib/display-scaling.test.ts
git commit -m "feat(ui): add display scaling tokens"
```

### 阶段 4：AppShell 布局稳定化

文件：

- 更新 `src/layouts/AppShell.vue`

建议：

- 将 sidebar 宽度和 collapsed 宽度替换为 CSS token。
- mobile drawer 使用 `min(var(--app-sidebar-width), 88vw)`。
- header、搜索区域、sidebar 折叠按钮、mobile menu button、主内容容器标记 `data-scaling-critical`。
- 检查 `min-w-0`、`overflow-hidden`、`minmax(0, 1fr)` 是否覆盖主内容区域。

验证：

```powershell
pnpm typecheck
pnpm test:display
```

建议提交：

```powershell
git add src/layouts/AppShell.vue
git commit -m "fix(ui): stabilize shell layout scaling"
```

### 阶段 5：设置页适配

文件：

- 更新 `src/components/jav-library/SettingsPage.vue`
- 按需更新 `src/components/jav-library/settings/*.vue`

排查命令：

```powershell
rg -n "min-w-\[|w-\[|text-\[|text-xs|grid-cols-\[" src/components/jav-library/SettingsPage.vue src/components/jav-library/settings
```

建议：

- 固定 grid column 改为 `minmax()`。
- 横向设置行在小宽度下允许换行。
- 长 label、路径、provider 名称、版本信息允许截断并提供 tooltip 或换行策略。
- 为关键设置行增加 `data-scaling-critical`。

验证：

```powershell
pnpm typecheck
pnpm test:display
```

建议提交：

```powershell
git add src/components/jav-library/SettingsPage.vue src/components/jav-library/settings
git commit -m "fix/ui: improve settings scaling"
```

### 阶段 6：播放器 HUD 适配

文件：

- 更新 `src/components/jav-library/PlayerPage.vue`

排查命令：

```powershell
rg -n "text-\[10px\]|text-\[11px\]|min-w-\[|h-9|size-9|grid-cols-\[" src/components/jav-library/PlayerPage.vue
```

建议：

- 核心控制按钮保持稳定尺寸，例如 `size-9` 或 `size-10`，并确保 `shrink-0`。
- icon-only button 必须有 `aria-label`。
- 避免在带文字的元素上直接使用 transform scale。
- 时间、进度条、状态提示在 125%/150% 下不截断。
- 原生播放器入口不能假设 Electron/mpv 一定存在，仍通过服务层能力判断展示状态。

验证：

```powershell
pnpm typecheck
pnpm test:display
```

建议提交：

```powershell
git add src/components/jav-library/PlayerPage.vue
git commit -m "fix(ui): improve player hud scaling"
```

### 阶段 7：网格、卡片、标签适配

文件：

- 更新 `src/lib/movie-grid-template.ts`
- 更新 `src/components/jav-library/VirtualMovieMasonry.vue`
- 更新 `src/components/jav-library/MovieCard.vue`
- 更新 `src/components/jav-library/ActorLibraryCard.vue`
- 按需更新其他卡片组件

建议：

- poster grid 使用稳定的 `minmax(min(100%, ...), 1fr)` 模式。
- 媒体内容使用 `aspect-ratio`，避免固定高度和内容尺寸互相冲突。
- 标签/chip 避免 `h-[29px] min-h-[29px] max-h-[29px]` 这类三重固定高度；可使用 `min-h-8 px-2.5 py-1 leading-tight`。
- 标题、元信息、标签行标记 `data-scaling-critical`。

验证：

```powershell
pnpm test -- src/lib/movie-grid-template.test.ts
pnpm typecheck
pnpm test:display
```

建议提交：

```powershell
git add src/lib/movie-grid-template.ts src/lib/movie-grid-template.test.ts src/components/jav-library/VirtualMovieMasonry.vue src/components/jav-library/MovieCard.vue src/components/jav-library/ActorLibraryCard.vue
git commit -m "fix(ui): improve card and grid scaling"
```

### 阶段 8：对话框和 overlay 适配

文件：

- 更新 `src/components/ui/dialog/DialogContent.vue`
- 更新 `src/components/jav-library/PreviewImageViewer.vue`
- 更新 `src/components/jav-library/PreviewImageViewerInner.vue`
- 按需更新其他高风险 dialog

排查命令：

```powershell
rg -n "DialogContent|max-h-\[|90vh|90dvh|fixed top-\[50%\]" src/components
```

建议：

- 移动端和 Safari 可见对话框优先使用 `dvh`。
- 对话框内容超出 viewport 时使用内部滚动。
- 关闭按钮、标题、主要操作按钮必须始终可见或可滚动到。
- 图片预览检查缩略图 rail 是否挤压主图。

验证：

```powershell
pnpm typecheck
pnpm test:display
```

建议提交：

```powershell
git add src/components/ui/dialog/DialogContent.vue src/components/jav-library/PreviewImageViewer.vue src/components/jav-library/PreviewImageViewerInner.vue
git commit -m "fix(ui): improve dialog viewport scaling"
```

## 9. 手动 QA 流程

启动开发环境：

```powershell
powershell -ExecutionPolicy Bypass -File .agents\skills\curated-dev-start\scripts\start-curated-dev.ps1
```

macOS Safari 检查：

1. 打开 `http://127.0.0.1:5173/`。
2. 分别测试浏览器缩放 `100%`、`125%`、`150%`。
3. 视窗尺寸近似调整为 `1280 x 832`、`1440 x 900`、`1512 x 982`。
4. 访问 Home、Library、Detail、Player、Actors、Curated Frames、Settings。
5. 检查文字截断、横向滚动、模糊文本、过小控件、内部滚动陷阱、固定元素遮挡、对话框可达性。

macOS Chrome 检查：

1. 重复 Safari 的页面与缩放流程。
2. 额外关注 Chrome 与 Safari 的字体行高差异、滚动区域表现、transform 动画清晰度。

Windows / 外接显示器检查：

1. 如条件允许，分别设置系统缩放 `100%`、`125%`、`150%`。
2. Chrome/Edge 浏览器缩放测试 `100%` 和 `125%`。
3. 重点检查滚动条宽度变化是否导致横向溢出。

## 10. 推进顺序

推荐顺序：

1. 先补检查清单和 UI 规范引用。
2. 再补自动化 display scaling smoke test。
3. 沉淀全局 CSS tokens。
4. 修 AppShell。
5. 修 Settings。
6. 修 Player HUD。
7. 修 grid/card/chip。
8. 修 dialog/overlay。

这样可以避免一次性大范围视觉改动，也能保证每一步都有明确验证方式。

## 11. 风险与取舍

- Playwright 的 DPR 测试不等于真实浏览器缩放，Safari/Chrome 手动检查仍必须保留。
- 全量扫描所有文字节点可能产生大量误报，应优先用 `data-scaling-critical` 标记关键区域。
- 一味放大字号和间距会降低媒体浏览密度，应优先使用可换行、可收缩、容器响应和合理截断。
- 虚拟滚动依赖测量估算，不能盲目移除所有固定高度。
- Safari 的 blur、shadow、transform 差异可能需要 token 级调整，避免写大量局部 hack。

## 12. 完成定义

该优化完成时应满足：

- `docs/reference/frontend-display-scaling-checklist.md` 已创建，并被 UI 规范和 Cursor 规则引用。
- `pnpm test:display` 已存在，覆盖 Chromium + WebKit 的核心页面冒烟测试。
- AppShell、Settings、Player、Grid/Card、Dialog/Overlay 已通过自动化检查。
- macOS Safari 和 Chrome 在 `100%`、`125%`、`150%` 下有人工检查记录或截图。
- 没有引入新的大规模状态管理重构。
- 没有让浏览器前端假设 Electron/mpv 一定存在。
- 剩余问题被记录为具体 route/component 的后续任务，而不是笼统描述为“Mac 显示不好”。

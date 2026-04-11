# Curated 前端 UI 规范

本文记录当前仓库已经落地的前端 UI 事实与约束，供实现、评审与后续迭代对照使用。若文档与代码不一致，以代码为准，并应同步回写本文件。

## 1. 总体原则

- 产品正式名称为 `Curated`。
- 前端主要是桌面浏览体验，核心流程是浏览资料库、查看详情、播放、设置。
- 业务组件放在 `src/components/jav-library`，通用基础组件放在 `src/components/ui`。
- 新增能力优先复用现有主题变量、卡片结构和交互模式，避免随意引入新的视觉体系。

## 2. 主题与设计令牌

- 全局主题令牌定义在 [`src/style.css`](../src/style.css)。
- 业务界面应优先使用语义化颜色与表面类名，例如 `bg-background`、`text-foreground`、`bg-card`、`border-border`、`text-muted-foreground`。
- 避免在业务组件中直接硬编码主背景色、正文色和交互色，除非是非常局部的装饰性图形。
- 品牌字重与标题表现继续沿用 `font-curated` 等现有约定。

## 3. 目录职责

| 路径 | 职责 |
|------|------|
| `src/components/ui` | 基础 UI 原语与可复用通用组件 |
| `src/components/jav-library` | Curated 业务组件与页面片段 |
| `src/layouts/AppShell.vue` | 主应用壳层布局 |
| `src/views/*.vue` | 路由级页面装配 |

## 4. 页面与壳层

- 常规业务页面继续运行在 `AppShell` 内。
- 资料库、收藏、最近、标签等页面保持同一套浏览模型与壳层节奏。
- `AppShell` 桌面端采用 split shell：左侧 `AppSidebar` 作为持久导航面，右侧内容区作为同层工作区；不再用一个共同的大圆角卡片容器包住侧栏和内容区。
- 设置页仍然是常规业务页面，不承担大型实验性展示画布职责。

## 5. 业务组件约束

- `MovieCard`、`ActorLibraryCard`、`PlaybackHistoryCard`、`DetailPanel` 等业务组件优先保持产品语义，不应为临时展示环境或内部实验区引入专用 props。
- 组件中的交互、表单、菜单、焦点态应尽量保留原有结构与视觉层级。

## 6. 可访问性与交互

- 可点击区域优先使用真实按钮或可聚焦控件，避免裸 `div` 点击。
- 保留清晰的 `focus-visible` 状态，不要只去掉 outline 而不提供替代焦点样式。
- 装饰性层应使用 `aria-hidden="true"`，图片与封面应提供有意义的 `alt`。
- 在展示型页面中，如需避免误操作，可在 showcase adapter 层抑制交互，而不是修改业务组件本体。

## 7. 维护要求

- 修改全局主题令牌、基础组件默认样式或关键业务组件视觉结构后，应同步更新本文件。
- 若架构事实发生变化，还应同步检查 `AGENTS.md`、`.cursor/rules/*.mdc` 与 `docs/plan/*` 中的相关说明。

## 8. 颜色治理约束

- 常规业务页面优先使用语义化 token：`background`、`foreground`、`card`、`surface`、`muted`、`accent`、`border`、`primary`。
- 状态表达统一收敛到四类语义色：`success`、`warning`、`danger`、`info`。状态点、状态徽标、状态文字、状态提示面板应优先复用统一承载方式，而不是每个页面手写一套颜色。
- 常规业务组件中不再直接引入 `amber-*`、`emerald-*`、`red-*`、`blue-*`、`sky-*` 等 Tailwind 原生状态色阶来表达业务状态；若必须使用，需要先说明其属于明确例外区域。
- 允许保留专属配色的例外区域包括：播放器沉浸式 HUD、开发环境水印与性能监视条、媒体内容覆盖层。例外颜色只能留在各自区域内部，不能反向扩散到普通业务页面。
- 新增或重构状态 UI 时，优先补足通用承载能力，再替换页面中的 raw color；不要为了治理颜色而同步重做整页布局或品牌主色。

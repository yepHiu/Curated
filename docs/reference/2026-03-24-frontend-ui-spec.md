# Curated 前端 UI 规范

本文记录当前仓库已经落地的前端 UI 事实与约束，供实现、评审与后续迭代对照使用。若文档与代码不一致，以代码为准，并应同步回写本文件。

## 1. 总体原则

- 产品正式名称为 `Curated`。
- 前端主要是桌面浏览体验，核心流程是浏览资料库、查看详情、播放、设置。
- 业务组件放在 `src/components/jav-library`，通用基础组件放在 `src/components/ui`。
- 新增能力优先复用现有主题变量、卡片结构和交互模式，避免随意引入新的视觉体系。

### 1.1 UI 治理与设计宪法

- 对新页面、重要面板、壳层/布局、共享组件、设计令牌或重复交互范式的变更，应先明确：所属产品面与用户任务、主要行动、信息层级、已有参考实现、适用状态（加载/空/错误/锁定或不可用/确认/窄屏）以及是否会影响设计系统。
- 设计优先服从产品面：浏览保持海报优先和紧凑密度；详情突出媒体元数据；播放保持视频优先、低干扰；设置保持居中、纵向卡片；壳层继续由 `AppShell` 管理内部滚动与持久导航。
- 一页或一面板应有明确的主任务和视觉起点；不让次级操作与媒体、元数据或主要行动争夺注意力。
- 重复出现的颜色、间距、卡片结构或交互行为属于系统决策：优先复用现有 token / 基元；确有新增必要时，连同语义、使用范围和例外一并写入本规范与对应规则。一次性例外必须局部化，不得无说明地扩散。
- UI 质量包括全部状态而非仅成功态：语义交互元素、可见焦点、可读输入框、图片替代文本、装饰层隐藏、移动端主要动作 44px 触控目标都属于设计完成条件。
- 项目级前置治理入口是 `.cursor/skills/curated-ui-governance/SKILL.md`；它不替代本规范，而是要求实施前形成设计框架、实施后将已稳定的系统决策回写到本规范。

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

### 4.1 设置页：Tab 大卡片与内嵌区块（当前范式）

适用于 `SettingsPage` 等以 **`Card` 分 Tab / 分区** 的界面（**元数据** 大区以 `SettingsMetadataSection.vue` 为参考实现）。

**信息架构**

- **卡片级 `CardTitle`**：只写**短标题**（分区名），不写长说明。
- **`CardDescription`**：**可选**；若下方各 nested 块已自带说明，**勿**再加大段顶栏描述，以免重复。

**卡片标题行（`CardHeader`）**

- 布局：`grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5`；左 **`size-9`** 图标区（弱 primary 表面），右 **`CardTitle`**：`text-lg tracking-tight` + `min-w-0`；装饰图标 `aria-hidden="true"`。

**垂直节奏**

- 单张业务卡片可收紧标题与正文首块间距：对 **`Card`** 使用 **`gap-2`**，**`CardHeader`** **`pb-0`**，**`CardContent`** **`pt-0`**（与 `SettingsMetadataSection` 一致）。
- **`CardContent`** 内并列子区块栈间距：**`gap-3`**（与同页其它 settings 一致），除非全 Tab 统一改用其它 spacing 令牌。

**nested 浅色块（MUST）**

- 容器：`rounded-* border border-border/… bg-muted/…`，统一 **`p-4`**，各块标题与说明**左缘对齐**；勿在仅标题一行加 **`px-*`** 破坏对齐。
- **块标题**：`text-sm font-semibold text-foreground`。
- **块说明**：`text-xs leading-relaxed text-muted-foreground sm:text-sm`（长文可加 `text-pretty`）。
- **块内标题与说明间距**：默认 **`flex flex-col gap-3`**；紧凑块可用 **`gap-2`**（如源连通性）。

**选项与横向操作**

- 互斥策略：**`fieldset`** + **`legend.sr-only`**；**`label` + `radio`** 行用 **`items-center`**。
- 选项主文：`text-sm font-medium`；选项下说明：与「块说明」同款 token。
- 文案 + 按钮/开关：**`justify-between`**；**`sm:flex-row`** 时整块 **`items-center`**，避免操作控件相对多行文案贴顶。
- `SettingsPage` 默认将常规按钮、选择器触发器和侧栏 Tab 收紧为 32px。对于手机端需要舒适触控的主要维护动作，可在按钮上增加 **`data-settings-comfortable-control`** 显式退出该密度规则，并使用 **`h-auto min-h-11`** 保证实际交互框至少 44px；这是一项局部可访问性例外，不应机械扩散到全部设置控件。

**索引**：`.cursor/rules/ui-component-spec.mdc`「设置页 Tab 大卡片与内嵌区块」；实现目录 **`src/components/jav-library/settings/SettingsMetadata*.vue`**。

维护页沿用上述结构：健康检查、备份与恢复检查、手动扫描为同卡片内的三个区块；输入与操作不再包重复的内层卡片。目录选择紧邻路径输入，创建/验证/预检靠右按内容宽度排列；主要维护操作手机端保留 44px 高度，`sm` 起恢复 32px 紧凑尺寸。健康报告沿用语义状态色，未扫描时显示简短状态。仅用于介绍未来架构的“配置模型”占位区不再显示。

## 5. 业务组件约束

- `MovieCard`、`ActorLibraryCard`、`PlaybackHistoryCard`、`DetailPanel` 等业务组件优先保持产品语义，不应为临时展示环境或内部实验区引入专用 props。
- 组件中的交互、表单、菜单、焦点态应尽量保留原有结构与视觉层级。

## 6. 可访问性与交互

萃取帧（2026-09-06）：截图回执属于播放面局部反馈，使用 `CaptureReceipt` 缩略图、时间与状态；控件事件阻止传播至视频，触屏主要动作至少 44px。正常成功回执 4.5 秒后退出，失败候选最多保留 3 分钟供用户重试。大图使用 `FrameImageViewer`，支持尺寸、加载失败重试、100% 与适应窗口。全屏播放器的 Dialog / DropdownMenu 可传 `portalTo` 定位到全屏 surface；其它页面默认行为不变。图库离屏卡片行保留布局高度和可聚焦加载入口，当前焦点所在行不卸载。

- 可点击区域优先使用真实按钮或可聚焦控件，避免裸 `div` 点击。
- 保留清晰的 `focus-visible` 状态，不要只去掉 outline 而不提供替代焦点样式。
- 装饰性层应使用 `aria-hidden="true"`，图片与封面应提供有意义的 `alt`。
- 在展示型页面中，如需避免误操作，可在 showcase adapter 层抑制交互，而不是修改业务组件本体。

## 7. 显示缩放与多端检查

- 涉及 macOS Retina、外接屏、浏览器缩放、系统显示缩放、窄 viewport 的 UI 检查，统一使用 [`frontend-display-scaling-checklist.md`](frontend-display-scaling-checklist.md)。
- 修改壳层布局、海报网格、播放器 HUD、设置页、对话框、全局 typography 或 spacing 后，应按显示缩放检查清单做回归。
- `pnpm test:display` 和其他 display scaling 冒烟测试耗时较长，代理不得在没有用户明确同意的情况下主动运行。
- 普通布局以 CSS 像素为基准，不为每一种物理分辨率或 DPR 单独写布局分支。
- 桌面 Retina 紧凑密度通过 `src/style.css` 的全局变量实现，只在 `(hover: hover) and (pointer: fine) and (min-width: 1024px) and (min-resolution: 2dppx)` 下覆盖侧栏、壳层 padding、海报网格和卡片 spacing；DPR `1.5` 外接屏继续使用默认密度变量。
- 显示适配不应把业务逻辑放入页面组件；业务能力判断仍通过服务层与 Composables。
- 当前阶段不因为显示适配引入全局状态管理重构。Pinia 如需引入，应后续从小模块逐步试点。

### 7.1 资料库手机端基线

- 375px 宽度下，资料库工具栏只保留筛选 / 排序 / 书签 / 批量管理，不再用「入库时间 / 发售日期 / 评分」三列 Tabs；排序改走独立排序按钮。书签按钮不显示数量。
- 移动端壳层菜单、导入、通知、主题切换、资料库筛选/排序/批量入口和影片勾选区的触控目标至少为 44×44 CSS px；`lg` 桌面端可恢复紧凑尺寸。
- 影片卡片标签文字允许在窄卡片内收缩与省略，但 `+N` overflow badge 固定保留，不得被卡片 `overflow-hidden` 裁切。
- 资料库路由保留一个语义 `h1`，即使视觉上下文主要由壳层搜索和筛选条承担，也不能让页面从无一级标题直接进入卡片标题。
- 轻量回归使用 `pnpm test:e2e` 的 375×812 Chromium 用例验证横向溢出、上述触控尺寸和 overflow badge 边界；它不替代需明确同意才能运行的完整 `pnpm test:display`。

### 7.2 嵌套下拉与二级菜单

下拉菜单再展开另一级面板时，按两个独立浮层处理，不要贴死在触发行旁边。

- 一级菜单相对触发器：`sideOffset: 4`（`DropdownMenuContent` / `PopoverContent` 默认）。
- 二级菜单相对父菜单：`sideOffset: 8`（`DropdownMenuSubContent` 默认），避免两个 `rounded-2xl` 面板重叠。
- 与父菜单并列的伴随面板（例如书签命名）必须与父菜单外边缘保持 8px 间距并顶边对齐。定位锚点是菜单项而非父面板：父菜单 `p-1 border` 时，书签命名使用 `side-offset: 13`（8 + 4 + 1）、`align-offset: -5`，并关闭 `align-flip`；不得只检查 prop 数值而忽略实际面板边界。
- 指向单个菜单项的操作子菜单仍对齐该行，但同样保持 8px 水平间距。
- 二级短表单的主按钮为右对齐胶囊按钮（`rounded-full`），按文案宽度收缩，不要做成全宽。
- 书签命名表单沿用 `p-4`、`FieldGroup` / `Field` / `FieldLabel` 与有边界的 `Input`；桌面保留紧凑按钮。低于 640px 时关闭父菜单，使用独立 `Dialog`，保存与关闭按钮至少 44px，避免两个固定宽度浮层互相覆盖。
- 输入聚焦后，Tab 可到保存按钮、Enter 提交、Esc 关闭并回到书签入口；保存失败保留输入，进行中禁用重复提交。不能用表单的 `keydown.stop` 吞掉 Esc 而不提供退出行为。

当前参考实现：`src/components/jav-library/LibrarySavedViewsControls.vue`。

### 7.3 阅读器底栏 HUD 菜单

漫画 / 写真阅读器底栏的设置菜单是播放/阅读面的局部例外：必须停在整条工具栏上方，与工具栏外缘保持 8px 间距，不得覆盖底栏。定位锚点是底栏卡片（`data-book-reader-chrome`），而不是设置按钮本身；`side="top"`、`sideOffset: 8`，并关闭 `sideFlip`，避免空间不足时翻到工具栏下方。普通页面一级菜单仍使用默认 `sideOffset: 4`。

## 8. 维护要求

- 修改全局主题令牌、基础组件默认样式或关键业务组件视觉结构后，应同步更新本文件。
- 若架构事实发生变化，还应同步检查 `AGENTS.md`、`.cursor/rules/*.mdc` 与 `docs/plan/*` 中的相关说明。

## 9. 颜色治理约束

- 常规业务页面优先使用语义化 token：`background`、`foreground`、`card`、`surface`、`muted`、`accent`、`border`、`primary`。
- 状态表达统一收敛到四类语义色：`success`、`warning`、`danger`、`info`。状态点、状态徽标、状态文字、状态提示面板应优先复用统一承载方式，而不是每个页面手写一套颜色。
- 常规业务组件中不再直接引入 `amber-*`、`emerald-*`、`red-*`、`blue-*`、`sky-*` 等 Tailwind 原生状态色阶来表达业务状态；若必须使用，需要先说明其属于明确例外区域。
- 允许保留专属配色的例外区域包括：播放器沉浸式 HUD、开发环境水印与性能监视条、媒体内容覆盖层。例外颜色只能留在各自区域内部，不能反向扩散到普通业务页面。
- 新增或重构状态 UI 时，优先补足通用承载能力，再替换页面中的 raw color；不要为了治理颜色而同步重做整页布局或品牌主色。

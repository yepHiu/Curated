# Curated 前端 UI 规范（Web 原型）

本文描述**当前仓库已实现**的前端 UI 约定，供实现与代码评审对照。产品级页面蓝图与远期桌面能力见 [`jav-libary.md`](jav-libary.md) **§9 UI 页面设计**：该节偏「目标设计」，**代码级事实与强制约定以本文为准**；若与实现不一致，以代码与 [`src/style.css`](../src/style.css) 为准，并应在文档中标注差距（见 [`.cursor/rules/docs-sync.mdc`](../.cursor/rules/docs-sync.mdc)）。

Cursor / Agent 速查规则：[`.cursor/rules/ui-component-spec.mdc`](../.cursor/rules/ui-component-spec.mdc)。表单与 `Input` / `dark:` 细节：[`.cursor/rules/vue-frontend-standards.mdc`](../.cursor/rules/vue-frontend-standards.mdc)。业务体验要点：[`.cursor/rules/jav-library-frontend-patterns.mdc`](../.cursor/rules/jav-library-frontend-patterns.mdc)。

---

## 1. 设计原则（与产品规则对齐）

- **桌面向心智**：浏览 → 查看详情 → 播放 → 设置；不依赖尚未实现的 Electron / mpv。
- **浏览连续性**：`library`、`favorites`、`recent`、`tags` 共用同一套浏览模型（路由与 query），体验上像同一应用的不同切片，而非四个独立产品。
- **海报优先**：资料库以封面网格为主，保持紧凑、可扫读；避免默认做成宽表或后台仪表盘式布局。
- **可复用边界**：领域展示放在 `jav-library`，通用控件放在 `ui`，数据经服务层与 API，避免页面内硬编码假数据路径。

---

## 2. 设计令牌与 `src/style.css`

### 2.1 事实源

- **2026-04 令牌扩展**：在不改变 Curated 主视觉基调的前提下，`src/style.css` 额外定义了 `--surface` / `--surface-elevated` / `--surface-muted`、`--success`、`--warning`、`--danger`、`--info`、`--border-strong` 等语义变量，用于 Design Lab 和后续组件规范收敛。
- **Design Lab（开发态）**：`/design-lab` 是仅开发环境开放的内部 UI Playground；入口位于 `Settings > About`，用于展示设计令牌、组件状态、交互原型、动效和可访问性示例，避免把实验性界面直接堆进设置页主体。

- **语义色与圆角**：`:root` 与 `.dark` 中定义 `--background`、`--foreground`、`--card`、`--primary`、`--border`、`--muted` 等；Tailwind v4 在 `@theme inline` 中映射为 `--color-*`，供 `bg-background`、`text-foreground` 等使用。
- **默认外观**：当前 **`:root` 即为深色主界面**（`html` 上未必有 `class="dark"`）。`body` 使用 `bg-background text-foreground`，并带有轻微背景渐变。
- **深色变体**：`@custom-variant dark (&:is(.dark *))` 表示 **`dark:` 前缀类仅在 `.dark` 祖先内生效**。主界面不要依赖「全局 `dark:`」来撑表单对比度。
- **品牌字体**：`--font-curated`（**Outfit**；`index.html` 经 Google Fonts 加载）用于侧栏 Curated 字标等局部，Tailwind 类名 **`font-curated`**（见 `AppSidebar.vue`）。
- **滚动条**：`--scrollbar-track` / `--scrollbar-thumb` / `--scrollbar-thumb-hover` 在 `@layer base` 中统一应用。

### 2.2 实现时优先使用的类名（示例）

| 用途 | 推荐 |
|------|------|
| 页面底/主背景 | `bg-background` |
| 正文 | `text-foreground` |
| 次级说明 | `text-muted-foreground` |
| 卡片表面 | `bg-card`、`border-border` |
| 弱对比区块 | `bg-muted/30`、`bg-muted/40` |
| 主操作 | `primary`、`text-primary-foreground` |
| 危险操作 | `destructive` |
| 聚焦环 | `ring`、`focus-visible:ring-*` |

**避免**：在业务组件里随意新增硬编码 `#` / 任意 `rgb()` 作为主背景或主文本色（难以与主题同步）。**已约定例外**：侧栏 Curated 字标使用主题 **`text-primary`** 与 **`font-curated`**（Outfit），见 `AppSidebar.vue`。

---

## 3. 组件目录与职责

| 路径 | 职责 |
|------|------|
| `src/components/ui` | shadcn-vue 基元与通用块（Button、Card、Dialog、Input、Tabs…） |
| `src/components/jav-library` | Curated 业务组件（影片卡、详情、播放器、设置聚合等） |
| `src/components/design-lab` | 开发态内部设计系统工作台（Design Lab）与 Playground 专用展示组件 |
| `src/layouts/AppShell.vue` | 侧栏、顶栏、主内容区布局与壳层导航行为 |
| `src/views/*.vue` | 路由页面：组装上述组件，保持精简 |

新增能力时：**先判断**是否仅为库领域专用；若是，放 `jav-library` 并复用 `ui`；若可跨项目复用且符合 shadcn 模式，再考虑放 `ui`。

---

## 4. 各产品面布局要点

- **资料库 / 收藏 / 最近 / 标签**：虚拟化海报列表（如 `VirtualMovieMasonry`、`MovieGrid`）；卡片整卡可点进入详情；避免在卡脚上堆大量主操作。
- **详情**：左海报 + 右元数据层次；相关推荐与主列表视觉权重协调。
- **播放**：视频区域优先，控件克制；进度与路由续播逻辑见项目记忆与 `PlayerPage`。
- **观看历史**：独立路由 `history`；卡片形态与进度展示与 `PlaybackHistoryCard` 等保持一致性。
- **设置**：居中内容带 + 纵向卡片/分组；与「宽屏管理后台」区分。
- **壳层滚动**：滚动应主要发生在 `AppShell` 内容区，避免整页壳无必要地一起滚动。

---

## 5. 表单、对话框与搜索框

- 使用 **`<Input>`** 时必须在 `<script setup>` 中 **显式** `import { Input } from '@/components/ui/input'`。
- 在 **Dialog / Card** 等表面上，输入框需有清晰边界（边框 + 弱填充），与 **`vue-frontend-standards.mdc`** 中 `Input` 默认一致；同一表单内 **`<textarea>`** 应与 `Input` 视觉语言一致。
- 顶栏搜索等可在传入 `class` 做圆角、高度覆盖；**不要**在覆盖时去掉可读边框/背景，除非父容器提供同等可读性。

---

## 6. 空态、加载与反馈

- 全页或区块空状态可复用现有模式（如 `NotFoundState.vue`）；保持文案与 `Card` / `muted` 层次一致。
- 长任务（扫描、刮削）：使用既有任务轮询与 `ScanProgressDock` / Toast 等模式，避免每页重复发明轮询 UI。

---

## 7. 交互与可访问性

- 海报/卡片主交互区：优先 `button type="button"` 或等价可聚焦控件；禁用裸 `div` 点击且无键盘支持。
- 需要可见聚焦环时保留 `focus-visible`；避免仅 `outline-none` 而无替代样式。
- 装饰性叠层使用 `aria-hidden="true"`；封面/缩略图 `alt` 宜有意义（如番号）。

---

## 8. 维护与变更流程

1. 修改 **全局 CSS 变量** 或 **`@theme inline` 映射**：更新本文 **§2**，并检查 [`.cursor/rules/ui-component-spec.mdc`](../.cursor/rules/ui-component-spec.mdc) 是否需同步。
2. 修改 **`components/ui` 基元默认样式**（如 `Input`）：更新本文 **§5**，并核对 **`vue-frontend-standards.mdc`**。
3. 新增或调整 **跨页面体验**（如统一卡高）：更新 **`jav-library-frontend-patterns.mdc`** 与本文 **§4**。
4. 与 [`docs/project-memory.md`](project-memory.md) 中的架构事实冲突时，以代码为准并修正记忆文档。

---

## 9. 参考索引

- 项目结构事实：[`docs/project-memory.md`](project-memory.md)
- 产品蓝图（含 §9）：[`docs/jav-libary.md`](jav-libary.md)
- 组件示例：`src/components/jav-library/MovieCard.vue`（Card + 令牌 + `MediaStill`）

# 项目记忆：`jav-shadcn`（产品名 **Curated**）

## 1. 产品定位

- 当前仓库是可运行、可打包的 **Curated** 本地优先媒体资料库，不再只是前端高保真原型。
- `docs/product/2026-03-20-jav-libary.md` 同时描述当前实现与目标桌面蓝图；其中 mpv、深度业务 IPC 和广泛原生桥接仍是目标，不应与已落地 Electron 壳混淆。
- 当前仓库包含 **Vue 前端** 与 **`Go + SQLite` 后端**；开发模式下可通过 **`VITE_USE_WEB_API=true`** 联通真实 HTTP API，本地 loopback 默认直连开发后端 **`127.0.0.1:8080`**，Vite 代理 **`/api` → `127.0.0.1:8080`** 仍作为 fallback；release **`127.0.0.1:8081`** 静态托管继续使用同源 **`/api`**（详见 `README.md`）。非 loopback 监听需要主配置显式设置 **`lanEnabled: true`** 且已初始化 PIN。关闭该开关时仍可使用内存 **Mock** 适配器。
- 当前阶段采用 `Web 优先 + 最小桌面壳层` 策略：核心业务仍是 `Vue Web App -> HTTP API -> Go Backend`，`electron/` 负责启动或复用 Go HTTP 后端、开发态启动或复用 Vite 前端、用带 Curated 图标的 BrowserWindow 加载 Web UI、关闭窗口时退到托盘，并仅通过 preload 暴露 `window.javLibrary.pickDirectory()` 这一类窄原生能力；深度 IPC 桥接仍是后续目标。

## 2. 当前代码事实

### 技术栈

- 框架：`Vue 3`
- 语言：`TypeScript`
- 构建：`Vite`
- UI：`shadcn-vue`
- 样式：`Tailwind CSS v4`
- 虚拟列表：`vue-virtual-scroller`
- 包管理器：`pnpm`

### 已落地结构

- 应用入口：`src/main.ts`
- 根组件：`src/App.vue`
- 路由：`src/router/index.ts`
- 布局壳：`src/layouts/AppShell.vue`
- 产品组件：`src/components/jav-library`
- UI 原子组件：`src/components/ui`
- 主题样式：`src/style.css`
- 品牌资源：`icon/curated-title-nobg.png` 为 README 顶部带字标志；`icon/curated-icon-rg-dark-pink.png` 为应用无字图标源图，已同步到 `public/Curated-icon.png`、`backend/frontend-dist/Curated-icon.png` 与 `backend/internal/assets/curated.ico`
- 公开文档：`README.md` 为英文主版，`README.zh-CN.md` 与 `README.ja-JP.md` 为完整翻译版；根目录 `API.md` 为唯一公开 API 参考文档
- **UI 设计规范（代码级）**：[`2026-03-24-frontend-ui-spec.md`](2026-03-24-frontend-ui-spec.md)；Cursor 速查 [`.cursor/rules/ui-component-spec.mdc`](../../.cursor/rules/ui-component-spec.mdc)
- 原型数据与类型（Mock 模式）：`src/lib/jav-library.ts`
- 播放进度（Web API 为 SQLite、Mock 为 localStorage）：`src/lib/playback-progress-storage.ts`、`src/services/protected-web-state-bootstrap.ts`、`src/lib/player-route.ts`、`src/lib/playback-history-groups.ts`
- 已播计数（Web API 为 SQLite、Mock 为 localStorage）：`src/lib/played-movies-storage.ts`

### 当前交付状态

- 应用已经是路由驱动的 SPA，而不是最初的单页按钮演示。
- `App.vue` 仅承载 `RouterView`，页面切换统一由 `vue-router` 管理。
- `AppShell` 已实现侧边栏、顶部搜索区和主内容区的稳定壳层。
- **数据源**：环境变量 **`VITE_USE_WEB_API`** 为 `true` 时，页面通过 **`src/services/adapters/web`** 与 **`src/api/*`** 消费后端 HTTP API；否则使用 **`mock`** 适配器与 `jav-library` 假数据。
- 已具备：首页推荐、虚拟化资料库、详情、演员详情、HTML5/HLS 播放、观看历史、萃取帧、导入与分片上传、设置、PIN 锁、可信会话、存储在线检测、应用更新，以及扫描/刮削任务的 SSE + 轮询补偿；部分能力在 Mock 模式下为本地降级表现。

### 当前前后端互联事实

- 已落地 **library-service 契约**（`src/services/contracts/library-service.ts`）与 **Web / Mock 双适配器**。
- **Go Backend** 提供 `/api` 下健康检查、认证与可信会话、影片/演员/萃取帧、导入、播放与 HLS session、库路径与存储健康、设置、扫描/刮削任务、SSE `events`、connected clients 与应用更新等（公开摘要见 `API.md` 与 `README.md`）。
- **库行为 JSON**：`config/library-config.cfg` 与 **`PATCH /api/settings`** 同步；**`autoLibraryWatch`**（默认开）控制是否在主配置允许时启用 **fsnotify** 监听并在新文件事件后排队防抖扫描，**不**关闭手动或周期全库扫描。
- 已有 **Electron MVP**：`electron/main.ts` / `electron/backend-process.ts` / `electron/frontend-process.ts` 启动或复用 Go HTTP 后端，等待 `/api/health` 后在开发态启动或复用 `http://127.0.0.1:5173` 的 Vite 前端并加载 Web UI；打包态仍加载 `http://127.0.0.1:8081` 上由后端托管的静态 UI。窗口使用 Curated 图标，关闭窗口会隐藏到托盘，托盘菜单可恢复窗口、在浏览器打开 Web 端、打开 Settings 或真正退出；`preload` 仅暴露 `window.javLibrary.pickDirectory()` 供现有目录选择入口调用原生目录对话框，不承载业务 API。仍无 **mpv** 命名管道；Web 阶段播放由浏览器 `<video>` 解码。
- **观看进度与已播放状态**在 Web API 模式写入 SQLite，在 Mock 模式写入 `localStorage`。Web 模式启动会先读取权威认证状态，只有解锁后才 hydrate，两类状态都只执行一次。

## 3. 当前产品信息架构

### 顶层页面

- `library`
- `favorites`
- `recent`（redirect）
- `tags`
- `trash`
- `actors`（演员库：[`ActorsView.vue`](../../src/views/ActorsView.vue)；API **`GET/PATCH /api/library/actors…`**；路由 query **`actorsQ`** / **`actorTag`**）
- `actors/:actorName`
- `history`（观看历史：按本地日期分组，数据见下）
- `curated-frames`
- `detail/:id`
- `player/:id?`
- `settings`
- `lock`、home 与应用内 404

### 观看进度与历史（前端事实）

- **续播进度**（当前时间、时长、`updatedAt`）在 Web API 模式经 `GET/PUT/DELETE /api/playback/progress` 与 SQLite 同步；Mock 模式保存在 **`localStorage`** 键 **`jav-library-playback-progress-v1`**，清除该浏览器站点数据会丢失 Mock 进度。
- **路由**：`history` → [`src/views/HistoryView.vue`](../../src/views/HistoryView.vue)；按本地日历日分组（今天/昨天/日期）；卡片 [`PlaybackHistoryCard.vue`](../../src/components/jav-library/PlaybackHistoryCard.vue)（左文案、右海报 **`coverUrl` 优先**、`object-cover` 裁切、底栏进度条）。
- **续播**：`PlayerPage` 在 `loadedmetadata` 后根据 **`?t=`**（优先）或本地存储 seek；`timeupdate` 节流写入；pause/ended/隐藏页签/unmount 补写。
- 从资料库/详情进入播放器：[`buildPlayerRouteFromBrowse`](../../src/lib/player-route.ts) 可在有效进度下附带 **`?t=`**；从历史进入附带 **`?from=history`**，`AppShell` 顶栏返回历史而非详情。

### 当前体验重点

- 影片库页以封面浏览、搜索、标签/演员筛选和选中态为主。
- 详情页强调单片信息、预览与相关推荐；支持用户标签、评分等与 API 联动（Web 模式）。
- 播放器页为 video-first；Web API 模式下主片源来自后端 **stream**（浏览器解码）。
- **观看历史**侧栏入口 + 按日分组瀑布列布局；设置页围绕目录、扫描、刮削、**整理入库**与**监听触发的自动扫描**（`autoLibraryWatch`）等组织。

## 4. 已确认的产品实现方式

- `library / favorites / recent / tags` 共享同一套浏览页面模型，通过 route name 和 query 参数切换上下文；**`actors` 与 `history` 为独立路由**，不混入 `LibraryMode` 查询拼装。
- 搜索词 `q`、标签页 `tab`、当前选中影片 `selected` 已作为 URL 状态存在。
- 影片浏览体验已经偏向“媒体库 / 海报墙”而不是“后台表格管理”。
- 库页已引入虚拟滚动能力，说明大规模海报浏览是明确方向。
- 设置页已经联通后端持久化，并覆盖通用、影片存储、元数据、网络、播放、萃取帧、安全、关于与维护等领域。

## 5. 当前尚未实现的能力

- 尚未接入深度 `Electron` preload / IPC 桌面桥接；当前 Electron 仍是最小浏览器壳层，只补了托盘生命周期和原生目录选择这一类窄 preload 能力。
- 尚未实现 **mpv** 与命名管道 / IPC 的桌面播放闭环（Web 阶段为 `<video>` + HTTP Range）。
- 尚未实现账号级多用户、跨资料库远程同步与冲突解决；这与当前同一 SQLite 资料库内共享播放进度是不同范围。
- 尚未完成备份/恢复预检、路径迁移 CLI、上传 session 重启恢复、Library Health 和元数据修复队列。

## 6. 产品与架构边界

- 前端页面可以提前表达未来桌面产品形态，但不能把未来能力当成当前事实。
- Renderer 层应依赖前端服务接口和 typed contracts，而不是直接耦合 IPC、文件系统或数据库细节。
- 文档必须持续区分三类信息：`当前原型事实`、`目标桌面架构`、`待决策事项`。
- 若代码与文档冲突，应优先以代码现状修正文档记忆。

### 前端互联要求

- 前端已具备 service contract 与 Web/Mock adapter；UI 组件和 `views` 继续不得直接依赖具体 adapter、Electron IPC 或数据库实现。
- 新能力应先扩展现有 typed DTO、错误码、事件和 service contract，再接 UI 调用链。
- 只有 active adapter 可以启动；Mock 模式不得因为模块导入而请求后端，Web 模式不得在锁定状态 hydrate 受保护数据。
- 若未来新增 desktop adapter，必须证明 HTTP service seam 无法满足该窄原生能力，不得复制整套业务接口。
- 前端不应同时感知 `HTTP`、`IPC`、命名管道等多套底层协议，只消费统一的前端服务抽象。

## 7. 当前主要风险

- 最大风险仍然是把“产品蓝图”和“当前实现”混为一谈。
- **Mock 与 Web API 双模式**并存：未读 `.env` / 文档时易误判“是否已接后端”。
- **数据恢复能力不足**：SQLite 已成为长期用户状态来源，但备份验证、恢复预检和路径迁移尚未形成产品闭环。
- 上传 session 仍需跨后端重启持久化；资料库异常也缺少统一健康入口。
- 文案层面当前界面偏英文，设计文档偏中文，后续需要决定正式的产品语言策略。

## 8. 推荐的下一步产品优先级

1. 完成可验证的备份、恢复预检与路径迁移。
2. 将上传 session 持久化到 SQLite，并在启动时清理过期 session 与孤立 staging 文件。
3. 建立 Library Health 与元数据修复队列的统一入口。
4. 在数据可靠性基线后增加 Saved Views、推荐解释/负反馈、演员别名归并与 Insights。
5. 战略扩展只选择一个主分支，避免同时铺开 Linux/fnOS、WebDAV、漫画库和 Android。

## 9. 表单与文本输入（项目要求）

以下为 **Curated 前端** 对常用输入控件的**持久约定**（与 `src/components/ui/input/Input.vue`、弹窗表单实现保持一致）。

### 组件与导入

- 使用 shadcn-vue 封装的 **`Input`** 时，在对应 SFC 的 `<script setup>` 中**必须显式**编写：`import { Input } from '@/components/ui/input'`，不要依赖隐式解析。

### 深色界面与 `dark:` 工具类

- 主题色主要写在 **`src/style.css` 的 `:root`**（及 `.dark` 中与 `:root` 同步的一套变量），页面根节点**不一定**挂 `class="dark"`。
- Tailwind 自定义变体 **`dark:`** 仅在 **`.dark` 祖先**下生效（`@custom-variant dark (&:is(.dark *))`）。因此**不能**单靠 `dark:bg-*` / `dark:border-*` 保证输入框在弹窗、卡片上的可见性。
- 表单控件应优先使用**不依赖 `dark:`** 的 token：`border-border/60`、`bg-muted/40`、`text-foreground`、`placeholder:text-muted-foreground` 等，保证在默认深色背景下边框与填充仍清晰。

### 可读性与尺寸

- 单行输入需有足够**最小高度与内边距**（例如 `min-h-10`、`py-2` 量级），避免在对话框里显得过扁、难点击。
- 弹窗内同一表单中，**`<Input>` 与 `<textarea>`** 应采用**一致的边框 + 浅底**语言，避免一个清晰可见、另一个与背景融在一起。

### 局部覆盖

- 顶栏搜索等场景可对 `Input` 追加 `class`（如 `h-10`、`rounded-2xl`、`bg-background/70`）；合并类名时**不要**在无意中去掉对比度所依赖的边框或背景，除非外层已提供同等清晰的容器样式。

### 规则索引

- Cursor 细则：**`.cursor/rules/vue-frontend-standards.mdc`**（章节 *Form controls*）。
- 业务页面习惯说明：**`.cursor/rules/jav-library-frontend-patterns.mdc`**（Dialog / form fields）。

## 10. 维护约定

- 本文记录“稳定事实、产品判断、阶段边界”，不记录短期实现细节。
- 当路由结构、页面骨架、数据来源方式或桌面集成状态发生变化时，应优先更新本文。
- 若 `docs/product/2026-03-20-jav-libary.md` 继续扩展，需同步标注哪些是愿景，哪些已经在当前仓库落地。
- 打整机安装包或执行正式发布时，版本号必须跟随 `docs/ops/package-build-history.csv` 的发布历史延续；先查最近一条有效记录，再确定本次发布版本，并让安装包、发布清单与历史台账保持同一版本号。
- 调整全局 `Input` 默认样式或主题变量时，同步检查 **§9 表单与文本输入** 与 **`vue-frontend-standards.mdc`** 是否仍一致。
- 调整品牌资源时，优先以 `icon/` 为设计源：README 使用 `icon/curated-title-nobg.png`，应用图标统一由 `icon/curated-icon-rg-dark-pink.png` 派生；至少同步检查 `public/Curated-icon.png`、`backend/frontend-dist/Curated-icon.png` 与 `backend/internal/assets/curated.ico`。
- 调整公开接口时，根目录 `API.md` 是唯一对外 API 参考文档；README 三语版只保留 API 概要和链接，不再维护完整接口表。
- 调整用户可见说明、命令入口或公开功能描述时，默认同步检查 `README.md`、`README.zh-CN.md`、`README.ja-JP.md` 三份文档，避免多语言 README 长期漂移。

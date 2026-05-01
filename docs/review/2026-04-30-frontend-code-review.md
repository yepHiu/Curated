# 前端代码 Review 报告

**日期:** 2026-04-30  
**范围:** `src/` 全部代码（115 `.vue`、107 `.ts`、68 `.test.ts`）  
**维度:** TypeScript 类型安全、组件架构、Composable 正确性、Service/Adapter 层、错误处理、性能模式、i18n、安全性、测试质量

---

## Critical 级

### 1. 生产环境适配器零测试

**文件:** `src/services/adapters/web/web-library-service.ts`

所有电影 CRUD、扫描、设置、播放进度、curated frames 的生产后端路径均无测试。Mock 适配器有测试，但真实适配器没有。

### 2. PlayerView / PlayerPage 零测试

最复杂的视图之一（PlayerView + PlayerPage 合计约 2900 行），包含 HLS 播放、进度追踪、键盘快捷键、策展帧截图等，完全无测试。

### 3. 影片列表加载失败静默吞没

- `src/services/adapters/web/web-library-service.ts:188` — `ensureLoaded` 失败仅 `console.error`，用户永远不知道加载失败
- `src/services/adapters/web/web-library-service.ts:175` — `reloadMoviesFromApiImmediate` 同样模式

### 4. `playback-progress-storage.ts` 95% 未测试

**文件:** `src/lib/playback-progress-storage.ts`（163 行，仅 1 个测试覆盖 `parseResumeSecondsFromQuery`）

未测试的核心路径：
- `hydratePlaybackProgress()` — Web API 请求 + 错误处理
- `saveProgress()` — localStorage / API 双路径 + position clamping
- `removeProgress()` — API + 本地双路径删除
- `getResumeSecondsForOpenPlayer()` — 5 秒/95% 续播阈值逻辑
- `listSortedByUpdatedDesc()` — 排序/过滤
- `loadStore()`/`saveStore()` — localStorage parse/stringify + 错误恢复

### 5. 超大组件（3 个超 1000 行）

| 文件 | 行数 | 核心问题 |
|------|------|---------|
| `SettingsPage.vue` | 3727 | 混合设置 CRUD、库路径管理、代理管理、健康检查、更新检测、capture 快捷方式 |
| `PlayerPage.vue` | 2775 | 混合 video 元素生命周期、HLS 流、性能统计、PiP、全屏、键盘快捷键、策展帧截图 |
| `CuratedFramesLibrary.vue` | 1828 | 混合 paginated list、分组视图、tag 过滤、tag 编辑、批量导出、批量删除、近重复检测 |

---

## High 级

### 6. 无全局错误处理边界

- `src/main.ts:36` — 未设置 `app.config.errorHandler`
- `src/App.vue` — 根组件无 `onErrorCaptured`
- 后果：任何未捕获的 Vue 渲染错误或 Promise rejection 会静默崩溃到空白页

### 7. HTTP 请求无超时机制

**`src/api/http-client.ts:61`** — 所有 `fetch()` 调用均无 `AbortController.signal` 和 timeout。网络卡住时请求永远不会 resolve/reject，对应 UI 将无限等待。

### 8. 用户操作错误被吞没（无 toast）

| 位置 | 操作 | 后果 |
|------|------|------|
| `LibraryView.vue:722` | `toggleFavorite` | 仅 `console.error`，UI 不一致 |
| `LibraryView.vue:264` | `patchMovieDisplayForLibraryEdit` | 仅 `console.error` |
| `SettingsPage.vue:1476` | `confirmRemoveLibraryPath` | 仅 `console.error` |
| `web-library-service.ts:859` | `loadMovieDetail` | 仅 `console.error`，返回 `undefined` |

### 9. `JSON.parse()` 无运行时验证的类型断言

| 文件 | 行号 | 风险 |
|------|------|------|
| `api/http-client.ts` | 87, 108 | 每个 API 响应体都经过 `as T` 强转 |
| `api/endpoints.ts` | 99, 107, 221, 237 | 类型参数 widen 到 `Record<string, string \| number \| undefined>` |
| `lib/movie-comment-local-storage.ts` | 13 | localStorage 数据无结构校验 |
| `lib/playback-progress-storage.ts` | 35 | 同上 |
| `lib/mock-movie-prefs-storage.ts` | 27 | 同上 |
| `lib/curated-frames/db.ts` | 117, 159, 190, 274 | IndexedDB 数据无运行时校验 |

### 10. i18n locale key 不一致

- `zh-CN` 有 8 个 `curated.*` key（`tagFilterTitle`, `tagFilterAll`, `tagFilterEmpty`, `tagFilterNoMatches`, `tagFilterShowMore`, `tagFilterShowLess`, `ariaFilterFrameTag`, `ariaClearFrameTagFilter`）在 `en.json` 和 `ja.json` 中缺失
- 英文/日文用户在策展帧 tag 过滤 UI 中会看到回退的中文文本
- `settings.curatedExportFormatSaving` 仅在 `en.json` 中存在

### 11. 多处硬编码中文（非 locale 文本）

| 文件 | 内容 |
|------|------|
| `PlayerView.vue:67` | `正在加载播放目标…` |
| `PreviewImageViewerInner.vue:151-265` | 对话框标题 `预览图`、描述、所有 aria-label |
| `ScanProgressDock.vue:32-93` | 扫描状态、统计标签 |
| `PlayerPage.vue:2725` | `隐藏详细统计信息` / `详细统计信息` |
| `MovieRatingStars.vue:33-51` | 所有 aria-label |
| `ExpandableText.vue:15-16` | 展开/收起默认 prop 值 |
| `pick-directory.ts:61,96` | 用户提示信息 |
| `use-scan-task-tracker.ts:83` | 错误提示 |

### 12. `portalModel` computed 在播放期间频繁重算

**`src/views/HomeView.vue:25`** — `portalModel` 依赖 `playbackProgressRevision`，该信号在播放时每秒多次更新。每次更新都触发整个 portal model 重建（遍历所有 movie + playback entry + hero carousel 候选排序）。

### 13. 多个 composable 和关键 view 零测试

| 模块 | 影响 |
|------|------|
| `use-backend-health.ts` | 无测试 |
| `use-scan-task-tracker.ts` | 无测试 |
| `use-app-update.ts` | 无测试 |
| `curated-frames/db.ts` | IndexedDB 核心操作无测试 |
| `curated-frames/capture.ts` | Video 帧截图无测试 |
| `DetailView.test.ts` | 仅 1 个测试（Escape 键） |
| `HistoryView.test.ts` | 仅 1 个测试（toolbar CSS class） |
| `DetailPanel.test.ts` | 仅 1 个测试（11 个 mock） |

---

## Medium 级

### 14. 6 个组件绕过 service 层直接调 `api`

| 组件 | 调用的 api 方法 |
|------|----------------|
| `SettingsPage.vue` | `api.pingProxyGoogle()`, `api.pingProxyJavbus()`, `api.pingAllProviders()`, `api.pingProvider()`, `api.health()` |
| `PlayerPage.vue` | `api.deletePlaybackSession()` |
| `CuratedFramesLibrary.vue` | `api.postCuratedFramesExport()` |
| `ActorProfileCard.vue` | `api.getActorProfile()`, `api.scrapeActorProfile()`, `api.patchActorExternalLinks()`, `api.getTaskStatus()` |
| `MovieCommentSection.vue` | `api.getMovieComment()`, `api.putMovieComment()` |
| `SettingsHomepageDevTools.vue` | `api.refreshHomepageDailyRecommendations()` |

### 15. `use-scan-task-tracker.ts` 无 onUnmounted 清理

模块级 `setInterval` 和 `setTimeout`，若组件在任务非终态时卸载，轮询将永远继续，无法停止。

### 16. `httpClient.delete` 错误处理不一致

`delete` 方法重复实现了 `handleResponse` 的错误逻辑，但使用 `response.json()` 而非 `response.text()` + `JSON.parse()`。空 body 的 204 响应会导致 JSON 解析异常。

### 17. Mock 适配器与 Web 适配器行为差异

| 差异点 | Mock | Web |
|--------|------|-----|
| `getMovieById` | 不搜索 trash | 搜索 active + trash |
| `loadMovieDetail` | 同步查找数组 | 异步 API 调用 |
| 错误类型 | `new Error()` | `HttpClientError` |

### 18. 后端错误码几乎未被使用

`TaskDTO.errorCode` / `TaskDTO.errorCategory` 在整个前端代码中从未被读取。唯一一次错误码检查是 `ActorProfileCard.vue:408` 的 `COMMON_NOT_FOUND`。其余全部依赖 `err.message` 字符串匹配。

### 19. 缺少 v-once / v-memo 优化

整个代码库零使用 `v-memo` 和 `v-once`。Dialog 静态内容、v-for 不变列表、大型表单布局均有优化空间。

### 20. shallowRef 缺失

| 位置 | 问题 |
|------|------|
| `web-library-service.ts` | `moviesState`/`trashedMoviesState` 使用 `ref`（数组整体替换，深层响应无意义） |
| `HistoryView.vue:46` | `batchSelectedIds` 使用 `ref<Set<string>>` 而非 `shallowRef`（与 LibraryView 不一致） |
| `ActorsPage.vue:37` | `actors` 使用 `ref` 而非 `shallowRef` |

### 21. Native player URL 验证不完整

`looksLikeBrowserProtocolLaunchTarget` regex 未显式阻止 `javascript:` 协议。虽由 localStorage 配置（用户可控自身风险），但建议加强校验。

### 22. 超大组件详情

**ActorProfileCard.vue**（859 行）：混合数据获取、tag CRUD、外部链接编辑、actor scrape 轮询、UI 渲染。

---

## Low 级

### 孤儿 timer / 无界增长

- `use-library-scroll-preserve.ts` — `restoreScrollSequence` 中 4 个 `setTimeout` + `rAF` 不可取消；`Map` 无 size 限制
- `use-home-scroll-preserve.ts` — 不可取消的 60ms `setTimeout`
- `use-app-update.ts` — `scheduleAutoCheck` 中 `setTimeout` 不可取消
- `wait-tracked-task.ts` — task 永不终止时 Promise 永不 resolve
- `use-theme.ts` — `matchMedia` 监听器在 HMR 下可累积

### Broad `unknown` 参数（~12 处）

Vue template ref 回调和事件处理器使用 `unknown` 正确但可收窄。例如 `SettingsPage.vue:144` 的 `setThemeFromSelect(v: unknown)`。

### 批量操作内部 catch 仅计数（8 处）

`LibraryView.vue` 和 `HistoryView.vue` 中批量操作的内部 `catch { fail++ }` 丢失了具体错误信息。

---

## 亮点 & 值得肯定的方面

- **TypeScript 严格模式全开**（`strict: true`, `noUnusedLocals`, `noUnusedParameters`）
- **零 `any`、零 `@ts-ignore`、零 `@ts-expect-error`** — 罕见的 TypeScript 纪律
- **100% `<script setup lang="ts">`** — 全项目一致
- **Props/Emits 100% 类型化** — 全部使用 `defineProps<>()` / `defineEmits<>()`
- **零 `v-html`、零 `innerHTML`** — XSS 防护良好
- **虚拟滚动实现质量高** — 正确处理了 `DynamicScroller` 的 padding 测量、poster load policy、size-dependencies 等已知陷阱
- **computed vs watch 使用正确** — 未发现滥用
- **纯逻辑 lib 测试质量优秀** — `player-playback-clock`, `player-immersive-chrome`, `homepage-portal`, `library-query`, `curated-frames/*` 等
- **`use-dev-performance-monitor.ts` 清理纪律** — 所有 subscription/timer/guard 在 `onUnmounted` 中完整拆卸

---

## 统计数据

| 类别 | 严重 | 数量 |
|------|------|------|
| Critical | 影片加载静默失败、关键模块零测试、超大组件 | 5 |
| High | 无错误边界、无 timeout、错误吞没、类型断言、i18n 缺失、零测试模块 | 8 |
| Medium | 绕过 service 层、清理缺失、Mock/Web 差异、shallowRef 缺失 | 9 |
| Low | 孤儿 timer、unknown 参数、批量错误丢失 | 3+ |
| Positive | TypeScript 纪律、组件一致性、XSS 防护、虚拟滚动 | - |

# 前端优化方案

**日期:** 2026-04-30  
**基于:** [2026-04-30-frontend-code-review.md](../review/2026-04-30-frontend-code-review.md)

---

## 总览

| Phase | 目标 | 文件数 | 优先级 |
|-------|------|--------|--------|
| P1 | 稳定性修复（错误边界、timeout、错误吞没） | ~12 | 立即 |
| P2 | 组件拆分（SettingsPage、PlayerPage、CuratedFramesLibrary） | ~15 | 本周 |
| P3 | 测试补全（关键模块、composable、view） | ~12 | 本周 |
| P4 | 架构改进（service 层收口、响应校验、shallowRef） | ~20 | 本月 |
| P5 | i18n 与打磨（hardcoded 文本迁移、性能微优化） | ~30 | 本月 |

---

## 执行进度（2026-05-01）

### 已完成切片

| 对应计划项 | 状态 | 已落地内容 | 主要文件 |
|-----------|------|------------|----------|
| 1.1 添加全局错误边界 | 已完成 | `App.vue` 使用 `onErrorCaptured` 捕获路由子树渲染异常并渲染故障态；`main.ts` 注册 `app.config.errorHandler` 作为全局兜底日志；补齐三语 `app.*` 故障态文案，并新增根错误边界回归测试 | `src/App.vue`, `src/main.ts`, `src/App.test.ts`, `src/locales/en.json`, `src/locales/ja.json`, `src/locales/zh-CN.json` |
| 1.2 HTTP Client 添加超时机制 | 已完成 | 所有 `httpClient` 请求统一使用 30s `AbortController` 超时；超时转换为 `HttpClientError(0, COMMON_TIMEOUT)`；`DELETE` 改为复用共享 `handleResponse<void>()` | `src/api/http-client.ts`, `src/api/http-client.test.ts` |
| 1.3 修复关键操作的错误吞没 | 已完成 | `LibraryView` 收藏/编辑失败改为 destructive toast；资料库加载失败通过 `LibraryService.loadError` 展示 banner；`SettingsPage` 移除库根失败改为 destructive toast；Web adapter 在列表/详情加载失败时写入可消费的 `loadError` | `src/views/LibraryView.vue`, `src/views/LibraryView.test.ts`, `src/components/jav-library/SettingsPage.vue`, `src/services/contracts/library-service.ts`, `src/services/adapters/web/web-library-service.ts`, `src/services/adapters/web/web-library-service.test.ts`, `src/services/adapters/mock/mock-library-service.ts` |
| 1.4 补全 i18n locale key 缺口 | 已完成 | 补齐策展帧 tag filter 的英文/日文 key；补齐 `settings.curatedExportFormatSaving` 的中文/日文 key；新增 locale parity 测试防回归 | `src/locales/en.json`, `src/locales/ja.json`, `src/locales/zh-CN.json`, `src/i18n/locales.test.ts` |
| 3.1 web-library-service 测试 | 部分完成 | 新增 Web adapter 测试骨架，覆盖初始列表加载失败写入 `loadError`、详情加载失败写入 `loadError`、初始分页加载、多并发详情请求合并与缓存写入、`toggleFavorite` 成功/失败缓存行为、`reloadMoviesFromApi` debounce 合并刷新 | `src/services/adapters/web/web-library-service.test.ts` |
| 3.2 playback-progress-storage 测试 | 部分完成 | 已覆盖 route query 解析、localStorage 坏数据恢复、保存 position clamp、续播阈值、排序、删除、Web API hydrate 失败保留缓存、Web API 写入/删除；localStorage quota/private mode 仍可后续单独补 | `src/lib/playback-progress-storage.ts`, `src/lib/playback-progress-storage.test.ts` |
| 3.3 PlayerView 基础测试 | 部分完成 | 新增 `PlayerView` 入口测试，覆盖缓存命中渲染 `PlayerPage`、autoplay 路由参数、未找到状态、Web API hydrate loading 与播放记录写入；播放器内部 resume 行为后续在 `PlayerPage`/播放目标测试中继续补 | `src/views/PlayerView.test.ts` |
| 3.4 Composable 测试补全 | 部分完成 | 新增 `use-scan-task-tracker` 卸载清理测试；新增 `use-backend-health` mock/Web 成功失败、轮询、卸载清理、手动 recheck spinner 测试；新增 `use-app-update` disabled、按需加载、手动失败、自动检查去重测试 | `src/composables/use-scan-task-tracker.ts`, `src/composables/use-scan-task-tracker.test.ts`, `src/composables/use-backend-health.test.ts`, `src/composables/use-app-update.test.ts` |
| 4.3 统一 `httpClient.delete` 错误处理 | 已完成 | `delete()` 不再自行 `response.json()`，统一走共享响应解析，支持 204 空 body | `src/api/http-client.ts` |
| 4.4 修复 `use-scan-task-tracker` 清理 | 已完成 | 使用消费者计数，最后一个消费者卸载后清理轮询、dismiss timer 和模块级状态，避免页面卸载后孤儿轮询 | `src/composables/use-scan-task-tracker.ts` |
| 4.5 shallowRef 优化 | 已完成 | 将 Web adapter 的影片列表/回收站列表、观看历史批量选择 Set、演员列表切换为 `shallowRef`，保留原有整体替换触发模式，减少大列表深层响应追踪 | `src/services/adapters/web/web-library-service.ts`, `src/views/HistoryView.vue`, `src/components/jav-library/ActorsPage.vue` |
| 4.7 Native player URL 安全加固 | 已完成 | `looksLikeBrowserProtocolLaunchTarget` 显式拒绝 `javascript:` / `data:` / `vbscript:`，保留 `potplayer:` 等外部播放器协议 | `src/lib/native-player-launch.ts`, `src/lib/native-player-launch.test.ts` |
| 5.1 硬编码中文迁移到 locale 文件 | 部分完成 | `PlayerView` 的 loading / not found 可见文本已迁移到 `player.loadingTarget`、`player.notFoundTitle`、`player.notFoundDesc`，并由 `PlayerView.test.ts` 与 locale parity 测试覆盖 | `src/views/PlayerView.vue`, `src/views/PlayerView.test.ts`, `src/locales/en.json`, `src/locales/ja.json`, `src/locales/zh-CN.json` |
| Lint 本地工作区排除 | 已完成（计划外支撑项） | `eslint .` 排除 `.workspace/**` 与 `.local/**`，避免扫描本地 Go/cache 临时目录导致 EPERM，符合仓库本地临时目录政策 | `eslint.config.js` |

### 验证记录

- `pnpm test -- src/App.test.ts src/i18n/locales.test.ts`：2 files / 4 tests passed
- `pnpm test -- src/views/LibraryView.test.ts src/services/adapters/web/web-library-service.test.ts src/i18n/locales.test.ts`：3 files / 7 tests passed
- `pnpm test -- src/services/adapters/web/web-library-service.test.ts`：1 file / 7 tests passed
- `pnpm test -- src/services/adapters/web/web-library-service.test.ts src/views/HistoryView.test.ts`：2 files / 8 tests passed
- `pnpm test -- src/views/PlayerView.test.ts`：1 file / 3 tests passed
- `pnpm test -- src/composables/use-backend-health.test.ts`：1 file / 5 tests passed
- `pnpm test -- src/composables/use-app-update.test.ts`：1 file / 4 tests passed
- `pnpm test -- src/views/PlayerView.test.ts src/i18n/locales.test.ts`：2 files / 6 tests passed
- `pnpm test -- src/api/http-client.test.ts src/composables/use-scan-task-tracker.test.ts src/lib/playback-progress-storage.test.ts src/i18n/locales.test.ts src/lib/native-player-launch.test.ts`：5 files / 15 tests passed
- `pnpm typecheck`：passed
- `pnpm lint`：passed
- `pnpm test`：70 files / 234 tests passed
- `pnpm build`：passed（包含 `pnpm typecheck && vite build`）
- `git diff --check`：exit 0（仅 Windows CRLF 提示）

### 下一批优先继续

1. **3.1 web-library-service 测试**：继续扩展 `patchMovie` 乐观更新与回滚、`loadMovieDetail` 404/HTTP 错误路径、`getMovies`/筛选分页相关路径。
2. **3.3 PlayerView / PlayerPage 基础测试**：继续覆盖 resume 参数、播放目标解析和 PlayerPage 关键加载态。
3. **5.1 硬编码文本迁移**：继续处理 preview / scan dock / rating / pick-directory 等可见文本。

---

## Phase 1: 稳定性修复

### 1.1 添加全局错误边界

**状态（2026-05-01）:** 已完成。`src/App.vue` 已渲染根级故障态，`src/main.ts` 已注册全局错误处理；回归测试见 `src/App.test.ts`，三语文案见 `app.faultTitle` / `app.faultDescription` / `app.reload`。

**目标:** 防止未捕获错误导致白屏

**文件:** `src/main.ts`, `src/App.vue`

**方案:**
```
// src/main.ts — 注册全局 errorHandler
app.config.errorHandler = (err, instance, info) => {
  console.error("[global error handler]", err, info)
  // 降级：尝试渲染一个全局 FaultBarrier 或至少不白屏
}

// src/App.vue — 添加 onErrorCaptured 作为第二层防线
onErrorCaptured((err, instance, info) => {
  // 阻止错误继续向上传播
  // 如果当前路由是 player/detail，展示轻量提示，不中断整个应用
  return false
})
```

**预估:** 2 文件，~30 行改动

---

### 1.2 HTTP Client 添加超时机制

**状态（2026-05-01）:** 已完成。实现见 `src/api/http-client.ts`；回归测试见 `src/api/http-client.test.ts`。

**目标:** 防止网络卡死导致 UI 无限等待

**文件:** `src/api/http-client.ts`

**方案:**
```
// 为每个请求创建 AbortController，默认超时 30s
const DEFAULT_TIMEOUT_MS = 30_000

async function request<T>(path, options) {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS)
  try {
    const response = await fetch(url, { ...options, signal: controller.signal })
    return handleResponse<T>(response)
  } catch (err) {
    if (err.name === "AbortError") {
      throw new HttpClientError(0, { code: "COMMON_TIMEOUT", message: "Request timed out" })
    }
    throw err
  } finally {
    clearTimeout(timeoutId)
  }
}
```

**预估:** 1 文件，~25 行改动

---

### 1.3 修复关键操作的错误吞没

**状态（2026-05-01）:** 已完成。`LibraryService` 合约新增 `loadError`；Web adapter 在列表/详情加载失败时写入错误；`LibraryView` 展示加载错误 banner，并对收藏/编辑失败弹出 destructive toast；`SettingsPage` 移除库根失败弹出 destructive toast。回归测试见 `src/views/LibraryView.test.ts` 与 `src/services/adapters/web/web-library-service.test.ts`。

**目标:** 用户操作失败必须有 toast 反馈

**文件:**
- `src/views/LibraryView.vue` — `toggleFavorite`、`patchMovieDisplayForLibraryEdit`
- `src/components/jav-library/SettingsPage.vue` — `confirmRemoveLibraryPath`
- `src/services/adapters/web/web-library-service.ts` — `ensureLoaded`、`reloadMoviesFromApiImmediate`、`loadMovieDetail`

**方案:**

`LibraryView.vue` — `toggleFavorite`:
```
} catch (err) {
  console.error("[LibraryView] toggle favorite failed", err)
  pushAppToast("destructive", t("library.favoriteToggleFailed"))
}
```

`web-library-service.ts` — `ensureLoaded`:
```
// 当前只 console.error；改为：
// 1. 设置一个模块级 error ref，供 LibraryView 消费
// 2. 或者在 views 层检测 moviesLoaded 长时间为 false 时展示错误态
```

推荐方案：在 `LibraryService` 合约中新增 `loadError: Ref<string | null>` 字段，views 层消费此字段展示错误 banner。

**预估:** 5 文件，~40 行改动

---

### 1.4 补全 i18n locale key 缺口

**状态（2026-05-01）:** 已完成。已补齐 `en` / `ja` / `zh-CN` 相关 key，并新增 `src/i18n/locales.test.ts` 校验。

**目标:** 英文/日文用户不看到回退的中文

**文件:** `src/locales/en.json`, `src/locales/ja.json`

**缺失 key（需翻译）:**
| Key | 中文原文 | en | ja |
|-----|---------|-----|-----|
| `curated.tagFilterTitle` | 标签筛选 | Tag Filter | タグフィルター |
| `curated.tagFilterAll` | 全部 | All | すべて |
| `curated.tagFilterEmpty` | 无标签 | No Tags | タグなし |
| `curated.tagFilterNoMatches` | 无匹配标签 | No Matching Tags | 一致するタグなし |
| `curated.tagFilterShowMore` | 显示更多 | Show More | もっと見る |
| `curated.tagFilterShowLess` | 收起 | Show Less | 折りたたむ |
| `curated.ariaFilterFrameTag` | 按标签筛选 | Filter by Tag | タグでフィルター |
| `curated.ariaClearFrameTagFilter` | 清除标签筛选 | Clear Tag Filter | タグフィルターをクリア |

`zh-CN` 需补：`settings.curatedExportFormatSaving`（从 en 同步）

**预估:** 3 文件，~20 行改动

---

## Phase 2: 组件拆分

### 2.1 SettingsPage 拆分（3727 行）

**目标:** 每个 section 独立组件，共享 settings composable

**方案:**
```
SettingsPage.vue (保留为路由入口，~300 行)
├── settings/
│   ├── SettingsGeneralSection.vue       (~200 行，已有部分)
│   ├── SettingsLibraryPathsSection.vue   (~350 行)
│   ├── SettingsProxySection.vue          (~250 行)
│   ├── SettingsMetadataSection.vue       (~250 行)
│   ├── SettingsPlaybackSection.vue       (~600 行，已有)
│   ├── SettingsCuratedSection.vue        (~200 行)
│   ├── SettingsScrapingSection.vue       (~150 行)
│   ├── SettingsLoggingSection.vue        (~200 行，已有)
│   ├── SettingsAppUpdateSection.vue      (~250 行，已有)
│   ├── SettingsHealthSection.vue         (~200 行)
│   └── SettingsHomepageDevTools.vue      (~100 行，已有)
```

新建 composable `use-settings-form.ts` 集中管理：
- 乐观更新 + seq 号模式
- 表单草稿态
- PATCH 失败后的 recovery refresh

**预估:** 7 新文件 + 1 composable，~500 行迁移

---

### 2.2 PlayerPage 拆分（2775 行）

**目标:** 核心逻辑抽取为 composable，PlayerPage 保留为 orchestration 层

**方案:**
```
新增 composables:
├── use-hls-playback.ts          — HLS 引擎初始化、bitrate tracking、fallback 逻辑
├── use-player-playback-core.ts  — video 元素生命周期（play/pause/seek/volume）
├── use-player-progress-tracker.ts — 播放进度上报、防抖、localStorage/API 同步
├── use-player-stats.ts          — FPS 统计、bitrate、性能数据聚合
├── use-player-keyboard.ts       — 键盘快捷键注册（已有 lib/player-shortcuts）
├── use-player-immersive-chrome.ts — 沉浸模式定时器（已有 lib/player-immersive-chrome）
├── use-curated-capture.ts       — 截图热键、frame 保存
├── use-native-player-launch.ts  — 外部播放器启动流程
```

PlayerPage.vue 保留为：
- 组合所有 composable 的入口
- 播放器 UI 布局（video 容器、controls overlay、settings menu）
- context menu 和 PiP 管理

**预估:** 8 新 composable 文件，PlayerPage 缩减至 ~600 行

---

### 2.3 CuratedFramesLibrary 拆分（1828 行）

**目标:** 每个 tab 独立组件，共享数据层

**方案:**
```
CuratedFramesLibrary.vue (保留为入口 ~200 行)
├── CuratedFramesTimelineTab.vue   — 时间线视图
├── CuratedFramesActorsTab.vue     — 按演员分组
├── CuratedFramesMoviesTab.vue     — 按影片分组
├── CuratedFrameEditDialog.vue     — tag 编辑 dialog
├── CuratedFramesExportDialog.vue  — 导出 dialog
```

新建 composable `use-curated-frames-list.ts`：
- paginated fetch（IndexedDB / API 切换）
- tag facet 计算
- 批量选择状态

**预估:** 5 新组件 + 1 composable，~400 行迁移

---

## Phase 3: 测试补全

### 3.1 web-library-service 测试

**状态（2026-05-01）:** 部分完成。已新增 `src/services/adapters/web/web-library-service.test.ts`，覆盖初始列表失败、详情失败、分页加载、并发详情请求合并与缓存写入、`toggleFavorite` 成功/失败缓存行为、`reloadMoviesFromApi` debounce 合并刷新；后续继续补 `patchMovie` 乐观更新与回滚、404/HTTP 错误路径、筛选分页相关路径。

**目标:** 覆盖生产适配器的核心路径

**文件:** `src/services/adapters/web/web-library-service.test.ts`（新建）

**测试点:**
| 方法 | 场景 |
|------|------|
| `ensureLoaded` | 正常加载、空库、HTTP 错误、后续调用复用缓存 |
| `getMovies` | 排序、过滤（mode/actor/tag）、分页 |
| `toggleFavorite` | 成功、HTTP 错误、已 trash 的 movie |
| `patchMovie` | 乐观更新、seq 号竞态、失败回滚 |
| `reloadMoviesFromApi` | debounce coalesce、加载后缓存替换 |
| `loadMovieDetail` | 正常、404、网络错误 |

**预估:** 1 文件，~300 行测试

---

### 3.2 playback-progress-storage 测试

**状态（2026-05-01）:** 部分完成。核心导出函数和 Web/localStorage 主路径已覆盖；localStorage quota/private mode 可后续继续补充。

**目标:** 覆盖所有导出函数

**文件:** `src/lib/playback-progress-storage.test.ts`（补充）

**新增测试点:**
| 函数 | 场景 |
|------|------|
| `hydratePlaybackProgress` | API 返回数据、空、网络错误 |
| `saveProgress` | localStorage 路径、API 路径（mock 模式）、position 边界 |
| `removeProgress` | localStorage 删除、API 删除 |
| `getResumeSecondsForOpenPlayer` | <5s 从头开始、>5s 续播、>95% 从头开始、0 进度 |
| `listSortedByUpdatedDesc` | 排序正确性、空列表 |
| `loadStore`/`saveStore` | 正常、损坏 JSON、localStorage 满 |

**预估:** 1 文件修改，~250 行新增测试

---

### 3.3 PlayerView 测试

**状态（2026-05-01）:** 部分完成。已新增 `src/views/PlayerView.test.ts`，覆盖缓存命中、未找到、Web API hydrate loading、autoplay 路由开关和播放记录写入；后续 resume 参数建议放在 `PlayerPage` 或 `playback-targets` 层继续补。

**目标:** 最基本的播放器加载路径

**文件:** `src/views/PlayerView.test.ts`（新建）

**测试点:**
- 正常加载（movie 已缓存）
- movie 未找到（NotFoundState）
- loading skeleton 展示
- resume 参数传递

**预估:** 1 新文件，~150 行测试

---

### 3.4 Composable 测试补全

**状态（2026-05-01）:** 部分完成。已补 `use-scan-task-tracker.test.ts` 的卸载清理回归；`use-backend-health`、`use-app-update` 尚未补。

**新建文件及测试点:**

`use-backend-health.test.ts`:
- **状态（2026-05-01）:** 已完成。已覆盖初始 mock 状态、首次成功、首次失败、轮询、组件卸载停止轮询、手动 recheck 最短 spinner 时长。

`use-scan-task-tracker.test.ts`:
- 开始追踪、进度更新、终态 toast、dismiss timer、组件卸载清理

`use-app-update.test.ts`:
- **状态（2026-05-01）:** 已完成。已覆盖 Web API disabled 初始状态、按需加载成功、手动 check 失败、多个消费者只调度一次 silent auto check。

**预估:** 3 新文件，~200 行测试

---

### 3.5 现有测试扩展

| 文件 | 当前 | 目标 |
|------|------|------|
| `DetailView.test.ts` | 1 test | 6 tests（加载、404、favorite、rating、delete、metadata refresh） |
| `HistoryView.test.ts` | 1 test | 4 tests（渲染、空态、删除确认、批量操作流程） |
| `DetailPanel.test.ts` | 1 test | 4 tests（tag 编辑、rating 交互、favorite 切换、delete 按钮） |

---

## Phase 4: 架构改进

### 4.1 API 响应校验（Zod / 轻量替代）

**目标:** 消除 `JSON.parse(text) as T` 的类型不安全

**方案选项:**

**选项 A: 轻量 guards（零依赖）**
```typescript
// src/api/guards.ts
function isMoviesPageDTO(v: unknown): v is MoviesPageDTO {
  return (
    typeof v === "object" && v !== null &&
    "movies" in v && Array.isArray(v.movies) &&
    typeof (v as any).total === "number"
  )
}

// http-client.ts
const parsed = JSON.parse(text)
if (!isMoviesPageDTO(parsed)) throw new Error("Invalid API response shape")
return parsed
```

**选项 B: Zod（类型 + 校验一体）**
```typescript
import { z } from "zod"

const MoviesPageDTOSchema = z.object({
  movies: z.array(MovieDTOSchema),
  total: z.number(),
  limit: z.number(),
  offset: z.number(),
})
```

推荐选项 A（轻量、零依赖、渐进式）。从 API 入口点 `parseJsonBody` 开始，为最关键的 DTO（`MoviesPageDTO`, `MovieDetailDTO`, `HealthDTO`）添加 guard，逐步覆盖。

**预估:** `src/api/guards.ts` 新文件 + `http-client.ts` 修改，~150 行

---

### 4.2 Service 层收口

**目标:** 所有组件通过 service 层调用，不直接 import `api`

**方案:**

**Step 1: 补齐 service contract 方法**
```
// contracts/library-service.ts 新增:
getActorProfile(name: string): Promise<ActorProfileDTO | undefined>
scrapeActor(name: string): Promise<TaskDTO>
updateActorExternalLinks(name: string, links: ExternalLink[]): Promise<void>
getTaskStatus(taskId: string): Promise<TaskDTO>

// 对应 curated frames:
exportCuratedFrames(ids: string[], format: ExportFormat): Promise<Blob>
deletePlaybackSession(sessionId: string): Promise<void>
refreshHomepageDailyRecommendations(): Promise<void>
```

**Step 2: Web adapter 实现新方法**

**Step 3: 组件改用 service 调用**
| 组件 | 改动 |
|------|------|
| `ActorProfileCard.vue` | 改调 `libraryService.getActorProfile()` 等 |
| `MovieCommentSection.vue` | 改调 `libraryService.getMovieComment()` 等 |
| `PlayerPage.vue` | 改调 `libraryService.deletePlaybackSession()` |
| `CuratedFramesLibrary.vue` | 改调 `libraryService.exportCuratedFrames()` |
| `SettingsPage.vue` | 改调 `libraryService.pingProxy()` 等 |
| `SettingsHomepageDevTools.vue` | 改调 `libraryService.refreshHomepageDailyRecommendations()` |

**预估:** contract 接口 + web adapter + mock adapter + 6 组件修改，~200 行新增 / ~80 行修改

---

### 4.3 统一 httpClient.delete 错误处理

**状态（2026-05-01）:** 已完成。`delete()` 已改用共享 `handleResponse<void>()`，并由 `src/api/http-client.test.ts` 覆盖请求超时行为。

**文件:** `src/api/http-client.ts`

**方案:** 删除 `delete` 方法中的重复错误处理逻辑，改为调用共享的 `handleResponse`。

```typescript
async delete(path: string): Promise<void> {
  const url = buildUrl(path)
  const response = await fetch(url, { method: "DELETE" })
  await handleResponse<void>(response) // 复用共享的错误处理
}
```

如果 204 空 body 场景有问题，在 `handleResponse` 中增加空 body 处理。

**预估:** 1 文件，~10 行改动

---

### 4.4 修复 use-scan-task-tracker 清理

**状态（2026-05-01）:** 已完成。使用消费者计数，最后一个消费者卸载后清理轮询与 dismiss timer。

**文件:** `src/composables/use-scan-task-tracker.ts`

**方案:**
```
export function useScanTaskTracker() {
  // ...

  onUnmounted(() => {
    stopPolling()
    clearDismissTimer()
  })

  // ...
}
```

同时在 `start()` 中新增 guard：启动新追踪前先停止旧的。

**预估:** 1 文件，~10 行改动

---

### 4.5 shallowRef 优化

**状态（2026-05-01）:** 已完成。`web-library-service.ts` 的 `moviesState` / `trashedMoviesState`、`HistoryView.vue` 的 `batchSelectedIds`、`ActorsPage.vue` 的 `actors` 已切换为 `shallowRef`；这些状态均通过整体赋值触发响应更新。

**目标:** 减少深层响应追踪开销

| 文件 | 改动 |
|------|------|
| `web-library-service.ts` | `moviesState`/`trashedMoviesState` 改为 `shallowRef` |
| `HistoryView.vue:46` | `batchSelectedIds` 改为 `shallowRef<Set<string>>` |
| `ActorsPage.vue:37` | `actors` 改为 `shallowRef` |

**注意:** 切换 `shallowRef` 后必须通过赋值触发响应（`.value = newVal`），不能用 `.value.push()`。当前这些位置已经是整体替换模式，切换安全。

**预估:** 3 文件，~6 行改动

---

### 4.6 修复 Mock / Web 适配器行为差异

**文件:** `src/services/adapters/mock/mock-library-service.ts`

**方案:**
1. `getMovieById` — 同步搜索 `moviesState` 和 `trashedMoviesState` 两个数组
2. `loadMovieDetail` — 使用微延迟 `Promise.resolve().then(...)` 模拟异步
3. 错误统一抛出 `HttpClientError` 子类（或至少定义 Mock 专用的 error 类型）

**预估:** 1 文件，~30 行改动

---

### 4.7 Native player URL 安全加固

**状态（2026-05-01）:** 已完成。已显式拒绝 `javascript:` / `data:` / `vbscript:`，并补充测试。

**文件:** `src/lib/native-player-launch.ts`

**方案:**
`looksLikeBrowserProtocolLaunchTarget` regex 显式阻止 `javascript:` 和 `data:` 协议：
```
const BLOCKED_PROTOCOLS = /^(javascript|data|vbscript):/i

function looksLikeBrowserProtocolLaunchTarget(url: string): boolean {
  if (BLOCKED_PROTOCOLS.test(url)) return false
  return /^[a-z][a-z0-9+-.]+:/i.test(url)
}
```

**预估:** 1 文件，~5 行改动

---

## Phase 5: i18n 与打磨

### 5.1 硬编码中文迁移到 locale 文件

**状态（2026-05-01）:** 部分完成。`PlayerView.vue` 的 `player.loadingTarget`、`player.notFoundTitle`、`player.notFoundDesc` 已迁移并补齐三语文案；其余 preview、scan、rating、pick-directory 等文本仍需继续。

**目标:** 所有用户可见文本通过 `$t()` 获取

**需要新增的 locale key 及翻译:**

| Key | zh-CN | en | ja |
|-----|-------|-----|-----|
| `player.loadingTarget` | 正在加载播放目标… | Loading playback target… | 再生ターゲットを読み込み中… |
| `player.notFoundTitle` | 播放目标未找到 | Player target not found | 再生ターゲットが見つかりません |
| `player.notFoundDesc` | 此播放路由指向的影片 ID 在当前库中不可用 | This player route points to a movie id that is not available in the current library | このプレイヤールートが参照する動画IDは現在のライブラリで利用できません |
| `preview.title` | {code} 预览图 | {code} Preview | {code} プレビュー |
| `preview.instructions` | 使用左右方向键或两侧按钮切换图片，Esc 关闭；点击下方缩略图跳转 | Use arrow keys or buttons to navigate, Esc to close; click thumbnails to jump | 矢印キーまたはボタンで画像を切り替え、Escで閉じる; サムネイルをクリックしてジャンプ |
| `preview.close` | 关闭 | Close | 閉じる |
| `preview.previous` | 上一张 | Previous | 前へ |
| `preview.next` | 下一张 | Next | 次へ |
| `preview.imageOf` | 第 {i} 张 | Image {i} | {i}枚目 |
| `scan.statusLabel` | 扫描状态 | Scan Status | スキャン状態 |
| `scan.completed` | 扫描完成 | Scan Completed | スキャン完了 |
| `scan.finished` | 扫描结束 | Scan Finished | スキャン終了 |
| `scan.scanning` | 正在扫描库 | Scanning Library | ライブラリをスキャン中 |
| `scan.close` | 关闭 | Close | 閉じる |
| `scan.processed` | 已处理 | Processed | 処理済み |
| `scan.newItems` | 新入库 | New | 新規 |
| `scan.updated` | 更新 | Updated | 更新 |
| `scan.skipped` | 跳过 | Skipped | スキップ |
| `movie.expandSummary` | 展开简介 | Expand Summary | 概要を展開 |
| `movie.collapseSummary` | 收起简介 | Collapse Summary | 概要を折りたたむ |
| `player.hideStats` | 隐藏详细统计信息 | Hide Detailed Stats | 詳細統計を非表示 |
| `player.showStats` | 详细统计信息 | Detailed Stats | 詳細統計 |
| `rating.ariaLabel` | 我的评分，半星步进 | My rating, half-star steps | 自分の評価、半星ステップ |
| `rating.score` | 打 {s} 分 | Rate {s} | {s}点を付ける |
| `pickDir.unsupported` | 当前环境无法自动读取磁盘绝对路径。请在本机资源管理器中打开该文件夹 | Cannot read absolute disk path in current environment. Please open the folder in your local file manager | 現在の環境ではディスクの絶対パスを自動的に読み取れません。ローカルファイルマネージャーでフォルダを開いてください |
| `pickDir.selected` | 已选择文件夹「{name}」。网页出于安全限制无法读取本机绝对路径 | Selected folder "{name}". Web cannot read the absolute path due to security restrictions | フォルダ「{name}」を選択しました。Webはセキュリティ制限により絶対パスを読み取れません |
| `scanTask.fetchFailed` | 无法获取扫描任务状态 | Failed to fetch scan task status | スキャンタスクの状態を取得できません |

受影响文件:
- `src/views/PlayerView.vue` — 2 处
- `src/components/jav-library/PreviewImageViewerInner.vue` — 7 处
- `src/components/jav-library/ScanProgressDock.vue` — 10 处
- `src/components/jav-library/ExpandableText.vue` — 2 处
- `src/components/jav-library/PlayerPage.vue` — 2 处
- `src/components/jav-library/MovieRatingStars.vue` — 12 处
- `src/lib/pick-directory.ts` — 2 处
- `src/composables/use-scan-task-tracker.ts` — 1 处

**预估:** 3 locale 文件 + 8 源文件，~60 行新增 / ~50 行修改

---

### 5.2 性能微优化

**5.2.1 PortalModel 播放期间防抖**

**文件:** `src/views/HomeView.vue`

**方案:** PortalModel 不需要在播放期间实时更新。可以在 `.watch` 中对 `playbackProgressRevision` 加 5s 防抖，或在 HomeView 不可见时暂停重算。

```
// 仅当 HomeView 可见或距上次重算超过 10s 时才触发
const shouldRecalcPortal = computed(() => {
  void playbackProgressRevision.value // consume
  void lastPortalCalcAt.value          // private timestamp
  return {} // 不返回实际数据，仅建立依赖
})
```

**预估:** 1 文件，~15 行改动

**5.2.2 关键位置添加 v-once**

| 文件 | 位置 |
|------|------|
| `ScanProgressDock.vue` | 统计标签（不随数据变化） |
| `DetailPage.vue` | 骨架 placeholder |
| `LibraryBatchActionBar.vue` | 确认 dialog 文本 |

**预估:** 3 文件，~3 行改动

---

### 5.3 同步 Mock / Web 适配器差异

| 差异点 | 方案 |
|--------|------|
| `getMovieById` 不搜索 trash | Mock 适配器中补充 trash 数组搜索 |
| `loadMovieDetail` 同步 vs 异步 | Mock 使用 `setTimeout(0)` 或 `Promise.resolve()` 模拟异步 |
| Error 类型不一致 | Mock 抛出兼容 `HttpClientError` 的错误（或定义 `MockHttpClientError`） |

---

## 实施时间线建议

| 周次 | Phase | 关键产出 |
|------|-------|---------|
| W1 | P1 全部 + P2 开始 | 错误边界、HTTP timeout、locale key 补齐、SettingsPage 拆分开始 |
| W2 | P2 完成 + P3 开始 | PlayerPage composable 拆分、web-library-service 测试 |
| W3 | P3 完成 + P4 开始 | 测试补全、service 层收口、API 校验 guards |
| W4 | P4 完成 + P5 | shallowRef、mock 差异修复、i18n 硬编码迁移 |
| W5 | P5 完成 | v-once 微调、最终验收 |

---

## 风险与注意事项

- **组件拆分**: 拆分 SettingsPage 时，共享的乐观更新 seq 号逻辑必须保持一致，建议先写 `use-settings-form` composable 再拆分组件
- **shallowRef 切换**: 确保所有修改走整体替换（`.value = newVal`）而非原地修改（`.value.push()`）
- **API 校验**: guards 应该是 additive（添加校验 + log warning），不是 breaking（拒绝合法响应），建议先以 warn 模式上线再收紧
- **Mock 适配器改动**: 改动后需确保 mock 模式下所有现有功能正常，建议先用 mock 模式跑一遍 smoke test

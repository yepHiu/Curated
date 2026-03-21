# JAV-Library 前后端未打通清单与改造顺序

## 1. 文档目的

本文用于基于当前仓库代码现状，给出一份可执行的前后端对接清单，明确：

- 当前哪些链路尚未打通
- 前端应新增哪些 adapter
- 前端应补哪些 TypeScript DTO 与契约
- Web 端第一阶段应暴露哪些 HTTP API
- 后端应优先接入哪些命令与事件
- 建议按什么顺序完成改造

当前方案采用 `Web 优先` 的阶段策略：

```text
Phase 1: Vue Web App -> HTTP API -> Go Backend
Phase 2: Vue Renderer -> preload -> Electron Main -> Go Backend
```

其中：

- 第一阶段先以 Web 端完成影片库、设置、扫描任务等主要链路。
- 第二阶段再将同一套服务契约复用到 Electron 桥接层。
- 因此前端当前优先建设 `web adapter`，而不是 `desktop adapter`。

---

## 2. 当前未打通清单

### 2.1 传输链路未打通

- 前端 `src/services/library-service.ts` 仍固定返回 mock 实现。
- 当前不存在 `web adapter`，前端无法通过 HTTP 访问真实后端。
- 当前 Go 后端已存在 `stdio-jsonl` 命令协议，但尚未提供供 Web 前端直接消费的 HTTP API。
- 当前不存在 Electron `preload` 和 `main` 层桥接代码，但它们不再是第一阶段阻塞项。

### 2.2 数据契约未打通

- 前端 `LibraryService` 仍是同步 mock 风格接口，和后端 `Command / Response / Event / AppError` 模型不一致。
- 前端缺少对后端 `MoviesPageDTO`、`MovieDetailDTO`、`SettingsDTO`、`TaskDTO` 的 TypeScript 映射类型。
- 前端缺少统一的错误模型、任务状态模型、事件模型。
- 当前前端 `Movie` 额外包含 `tone`、`coverClass` 等原型展示字段，后端 DTO 暂未覆盖，需要单独映射策略。

### 2.3 功能入口未打通

- 影片库列表页仍读取 mock 列表，而不是真实 `library.list`。
- 影片详情页仍读取 mock 详情，而不是真实 `library.detail`。
- 设置页按钮未触发 `settings.get`、`scan.start`、`scan.status`。
- 播放器页：已通过 **`GET /api/library/movies/{id}/stream` + `<video>`** 接入主文件播放（Web）；文档中的 **`player.play` / `player.command`（mpv IPC）** 仍为桌面阶段，尚未实现。

### 2.4 事件链路未打通

- 后端已存在 `task.*`、`scan.*`、`scraper.*`、`asset.*` 事件常量，但前端没有订阅入口。
- 前端没有任务中心、任务状态条、扫描进度展示或错误反馈区域。
- 后端异步能力和前端 UI 之间目前没有任何事件消费链路。

### 2.5 命令集合未打通

当前后端代码已实现但前端未接入的命令：

- `system.health`
- `library.list`
- `library.detail`
- `settings.get`
- `scan.start`
- `scan.status`

文档中已规划但后端当前代码尚未实现完整闭环的命令：

- `library.update`
- `settings.update`
- `player.play`
- `player.command`

---

## 3. 改造目标

本轮改造目标不是一次性把所有页面都改完，而是先建立一条最小可用的真实链路：

1. 前端可通过统一 `web adapter` 调真实后端
2. 前端可用真实后端驱动影片库和详情页
3. 设置页可读取真实设置并触发扫描
4. 前端可订阅扫描任务与进度事件
5. 后续播放器和写操作在同一套 service contract/DTO/事件体系中继续扩展

---

## 4. 建议新增的前端文件

以下为建议新增的最小文件集，路径为推荐路径。

### 4.1 Web API 基础层

```text
src/api/
  http-client.ts
  endpoints.ts
  types.ts
```

职责：

- `src/api/types.ts`
  - 定义 Web API 请求/响应的基础壳类型
  - 与后端 DTO 稳定对齐
- `src/api/http-client.ts`
  - 封装 `fetch`
  - 统一处理 base URL、JSON、超时、错误转换
- `src/api/endpoints.ts`
  - 管理后端 HTTP 路径和请求构造

### 4.1.1 后续 Electron 兼容位

桌面化阶段再新增：

```text
src/bridge/
  desktop-api.ts
  desktop-events.ts
```

但这些文件不属于当前 Web 阶段的首批必需项。

### 4.2 Service Adapter 层

```text
src/services/adapters/web/
  web-library-service.ts
  web-settings-service.ts
  web-task-service.ts
  mappers.ts
```

职责：

- `web-library-service.ts`
  - 调库列表、详情、更新相关 HTTP API
- `web-settings-service.ts`
  - 调设置读取、设置更新相关 HTTP API
  - 提供触发扫描入口
- `web-task-service.ts`
  - 调扫描状态、任务查询相关 HTTP API
  - 第一阶段可先用轮询消费任务状态
- `mappers.ts`
  - 将后端 DTO 映射为前端领域模型
  - 处理 `tone`、`coverClass` 等纯前端展示字段的补全逻辑

### 4.3 Service Contract 层

建议把现有 `LibraryService` 拆细，避免所有能力混在一个 service 里。

```text
src/services/contracts/
  library-service.ts
  settings-service.ts
  task-service.ts
  player-service.ts
```

建议职责：

- `library-service.ts`
  - 影片列表、详情、收藏、标签等
- `settings-service.ts`
  - 目录配置、扫描频率、播放器偏好
- `task-service.ts`
  - 任务查询、任务订阅、任务事件
- `player-service.ts`
  - 先定义接口，后续再接实现

### 4.4 DTO 与事件类型层

```text
src/types/backend/
  command.ts
  response.ts
  event.ts
  errors.ts
  library.ts
  settings.ts
  tasks.ts
  player.ts
```

职责：

- `command.ts`
  - 前端命令名常量
  - 前端请求壳类型
- `response.ts`
  - 标准响应壳类型
- `event.ts`
  - 事件名常量
  - 通用事件壳类型
- `errors.ts`
  - `AppErrorDto`
  - `ErrorCode` 字面量联合
- `library.ts`
  - `ListMoviesRequestDto`
  - `MoviesPageDto`
  - `MovieListItemDto`
  - `MovieDetailDto`
- `settings.ts`
  - `SettingsDto`
  - `LibraryPathDto`
  - `PlayerSettingsDto`
- `tasks.ts`
  - `TaskDto`
  - `TaskEventDto`
  - `ScanSummaryDto`
  - `ScanFileResultDto`
  - `TaskStatus`
- `player.ts`
  - 先定义 `PlayerStateDto`
  - 预留 `PlayerCommandDto`

### 4.5 Composable 层

```text
src/composables/
  use-library-data.ts
  use-settings-data.ts
  use-scan-task.ts
```

职责：

- `use-library-data.ts`
  - 负责页面与 service 的异步读取、加载态、错误态
- `use-settings-data.ts`
  - 负责设置读取、刷新、表单初始化
- `use-scan-task.ts`
  - 负责触发扫描、订阅任务事件、同步任务状态

---

## 5. 建议新增的 TypeScript DTO

以下 DTO 建议优先新增，并作为前端接后端的第一批契约。

### 5.1 命令与响应基础 DTO

- `BackendCommandDto`
- `BackendResponseDto<T>`
- `BackendEventDto<T>`
- `AppErrorDto`

建议字段：

```ts
interface BackendCommandDto<TPayload = unknown> {
  id: string
  type: string
  payload?: TPayload
}

interface BackendResponseDto<TData = unknown> {
  kind: "response"
  id?: string
  ok: boolean
  data?: TData
  error?: AppErrorDto
  timestamp: string
}

interface BackendEventDto<TPayload = unknown> {
  kind: "event"
  type: string
  payload?: TPayload
  timestamp: string
}

interface AppErrorDto {
  code: string
  message: string
  retryable: boolean
  details?: Record<string, unknown>
}
```

### 5.2 影片库 DTO

- `ListMoviesRequestDto`
- `MovieListItemDto`
- `MovieDetailDto`
- `MoviesPageDto`
- `GetMovieDetailRequestDto`

备注：

- `MovieListItemDto` 对齐后端返回字段
- `MovieDetailDto` 扩展 `summary`
- 前端领域模型 `Movie` 继续保留 `tone`、`coverClass`，但通过 mapper 补齐，不强加给后端

### 5.3 设置 DTO

- `LibraryPathDto`
- `PlayerSettingsDto`
- `SettingsDto`
- `UpdateSettingsRequestDto`

说明：

- 当前后端只实现了 `settings.get`
- `UpdateSettingsRequestDto` 先在 TS 中定义，供未来 `settings.update` 使用

### 5.4 任务与扫描 DTO

- `TaskStatus`
- `TaskDto`
- `TaskEventDto`
- `StartScanRequestDto`
- `GetTaskStatusRequestDto`
- `ScanSummaryDto`
- `ScanFileResultDto`

建议 `TaskStatus` 直接与后端一致：

- `pending`
- `running`
- `completed`
- `partial_failed`
- `failed`
- `cancelled`

### 5.5 播放器 DTO

即使播放器暂未实现，也建议先定义：

- `PlayerStateDto`
- `PlayMovieRequestDto`
- `PlayerCommandRequestDto`
- `PlayerEventDto`

第一版最少字段：

- `movieId`
- `status`
- `positionSeconds`
- `durationSeconds`
- `paused`
- `volume`

---

## 6. 建议新增的 Web API

当前第一阶段应先把 Go 后端暴露为 Web 可调用的 HTTP API，再由前端 `web adapter` 消费。

### 6.1 第一批必须新增的 API

- `GET /api/health`
- `GET /api/library/movies`
- `GET /api/library/movies/:movieId`
- `GET /api/settings`
- `POST /api/scans`
- `GET /api/tasks/:taskId`

### 6.2 第一批 API 与现有命令的映射关系

- `GET /api/health` -> `system.health`
- `GET /api/library/movies` -> `library.list`
- `GET /api/library/movies/:movieId` -> `library.detail`
- `GET /api/settings` -> `settings.get`
- `POST /api/scans` -> `scan.start`
- `GET /api/tasks/:taskId` -> `scan.status`

### 6.3 推荐响应风格

Web 层不必强行暴露 stdio 命令壳，但建议保持统一语义：

- 成功响应返回 DTO 本体或分页对象
- 失败响应统一映射为 `AppErrorDto`
- 长任务创建返回 `TaskDto`
- 列表查询返回 `MoviesPageDto`

### 6.4 第一阶段任务状态同步方式

第一阶段建议优先用轮询，而不是先上 WebSocket 或 SSE：

- 前端调用 `POST /api/scans` 启动扫描
- 后端返回 `TaskDto`
- 前端用 `GET /api/tasks/:taskId` 轮询状态
- 当状态进入结束态时停止轮询

等 Web 端主要业务稳定后，再考虑：

- `GET /api/events` 的 SSE 方案
- 或 WebSocket 推送
- 或 Electron 阶段的桌面事件桥接

---

## 7. 命令接入优先级

应按“先只读、后异步、再写操作、最后播放器”顺序接入。

### 第一优先级：立刻接入

这些命令能最快把前端从 mock 切到真实数据：

1. `system.health`
2. `library.list`
3. `library.detail`
4. `settings.get`

目标：

- 首页可以展示真实影片列表
- 详情页可以展示真实影片详情
- 设置页可以读取真实路径与扫描间隔
- 应用启动时可探测后端健康状态

### 第二优先级：任务链路接入

1. `scan.start`
2. `scan.status`

同时必须接入事件：

- `task.started`
- `task.progress`
- `task.completed`
- `task.failed`
- `scan.started`
- `scan.progress`
- `scan.file_skipped`
- `scan.file_imported`
- `scan.file_updated`
- `scan.completed`

目标：

- 设置页“Trigger full scan”按钮接真实扫描
- UI 可以看到扫描任务状态、进度和结果摘要

### 第三优先级：写操作命令

当前后端代码未完整实现，但建议优先排期：

1. `library.update`
2. `settings.update`

`library.update` 最少应支持：

- `toggleFavorite`
- `rateMovie`
- `addUserTag`
- `removeUserTag`

`settings.update` 最少应支持：

- 更新 `libraryPaths`
- 更新 `scanIntervalSeconds`
- 更新 `hardwareDecode`

目标：

- 收藏不再只改前端内存
- 设置开关不再只是本地 `ref`

### 第四优先级：播放器命令

建议最后接：

1. `player.play`
2. `player.command`

建议第一批播放器事件：

- `player.state_changed`
- `player.time_changed`
- `player.duration_changed`
- `player.ended`
- `player.failed`

目标：

- 播放器页从占位态切为真实状态驱动 UI

---

## 8. 推荐改造顺序

### 阶段一：建立 Web API 与 DTO 基础

新增：

- `src/types/backend/*`
- `src/api/*`
- `src/services/adapters/web/mappers.ts`

完成标准：

- 前端可以请求 `GET /api/health`
- 前端有统一 `http-client`
- 前端具备标准错误类型与响应类型

### 阶段二：接通只读数据链路

新增：

- `src/services/adapters/web/web-library-service.ts`
- `src/services/adapters/web/web-settings-service.ts`
- `src/composables/use-library-data.ts`
- `src/composables/use-settings-data.ts`

改造：

- `src/services/library-service.ts` 改为环境可切换或注入式选择 mock/web
- `LibraryView` 和 `DetailView` 从同步读数据改为使用 composable
- `SettingsPage` 从 mock stat/path 改为真实 settings 数据

完成标准：

- 页面可基于真实后端展示库和详情
- 页面具备基础 `loading / error / empty` 状态

### 阶段三：接通扫描任务链路

新增：

- `src/services/contracts/task-service.ts`
- `src/services/adapters/web/web-task-service.ts`
- `src/composables/use-scan-task.ts`

改造：

- 设置页扫描按钮接 `scan.start`
- 页面通过 `scan.status` 轮询任务状态
- 增加任务反馈区域或状态卡片

完成标准：

- 用户可从前端触发扫描并看到实时结果

### 阶段四：接通写操作

改造：

- 扩展 `LibraryService`
- 扩展 `SettingsService`
- 前端收藏、设置写入改为真实命令

完成标准：

- 收藏、设置修改进入真实后端
- 错误码和失败提示可在 UI 层展示

### 阶段五：接通播放器

新增：

- `src/services/contracts/player-service.ts`
- `src/services/adapters/web/web-player-service.ts`
- `src/composables/use-player-state.ts`

改造：

- `PlayerPage.vue` 从占位改为真实状态驱动
- Web 阶段先定义播放器服务契约与页面状态
- 若后端尚未提供 Web 播放接口，则该阶段可后置到 Electron 阶段

完成标准：

- 播放器页至少具备明确的契约与状态模型，不阻塞后续桌面化

---

## 9. 具体文件改造建议

### 第一批必须改的现有文件

- `src/services/library-service.ts`
  - 不再固定返回 mock
  - 改为 `resolveLibraryService()` 或 `createLibraryService()`

- `src/views/LibraryView.vue`
  - 不再直接依赖同步 `movies`
  - 改为从 composable 读取异步状态

- `src/views/DetailView.vue`
  - 不再假设详情总能同步拿到
  - 需要显式错误态和空态

- `src/components/jav-library/SettingsPage.vue`
  - `scanInterval`、`hardwareDecode`、`autoScrape` 不再只保存在本地 `ref`
  - “Run” 按钮改为真实命令触发

- `src/components/jav-library/PlayerPage.vue`
  - 当前可暂不立即实现，但需避免继续硬编码为纯占位

### 第一批建议保持不动的文件

- `src/domain/movie/types.ts`
- `src/domain/library/types.ts`

原因：

- 当前前端领域模型仍有价值
- 后续通过 mapper 将后端 DTO 映射为领域对象即可

---

## 10. 最小可执行里程碑

建议先追求以下 MVP 联通结果：

### 里程碑 1

- 前端应用启动后调用 `GET /api/health`
- 若后端不可用，给出明确错误提示

### 里程碑 2

- 影片库列表从 `library.list` 获取
- 详情页从 `library.detail` 获取

### 里程碑 3

- 设置页从 `settings.get` 初始化
- 点击扫描按钮触发 `scan.start`
- 页面展示 `scan.status` 和 `task.progress`

### 里程碑 4

- 收藏和设置变更走真实更新命令

### 里程碑 5

- 播放器页完成 Web 阶段可复用的服务契约设计，真实播放能力可延后到 Electron 阶段

---

## 11. 一句话结论

当前项目的核心问题不是前后端都没写，而是**前端 mock service 与真实后端之间缺少一层可直接落地在 Web 端的 HTTP 对接链路**。最优策略是：

1. 先补 `TS DTO + HTTP API + web adapter`
2. 先接 `health + library.list + library.detail + settings.get`
3. 再接 `scan.start + scan.status`，先用轮询跑通任务链
4. 然后补 `library.update + settings.update`
5. 最后再根据需要决定播放器是先走 Web 方案还是直接留给 Electron 阶段

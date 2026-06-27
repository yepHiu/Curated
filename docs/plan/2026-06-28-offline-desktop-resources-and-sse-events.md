# Curated 完全离线桌面资源与 SSE 事件流实施设计

日期：2026-06-28  
状态：已完成（2026-06-28）

## 目标

本轮同时完成两个可独立验证的需求：

1. **完全离线桌面资源**：生产桌面包不再依赖 Google Fonts 或 CDN `hls.js`，断网时仍能显示既定字体风格并启动 HLS 播放路径。
2. **SSE 后端事件流**：新增后端事件通道，先覆盖任务生命周期事件，让扫描、导入、刮削等任务可以实时推送到前端，同时保留现有轮询作为 fallback。

## 范围边界

### 本轮包含

- 移除 `index.html` 中 Google Fonts 的网络 `<link>`。
- 将 Outfit 字体作为本地依赖或本地资源进入前端 bundle。
- 将 `src/lib/hls-player.ts` 从 CDN script 注入改为本地动态 import。
- 新增 `GET /api/events` SSE 端点。
- 后端任务管理器发布 `task.updated` 事件，覆盖 `Create`、`Start`、`Progress`、`ProgressWithMetadata`、`Complete`、`PartialFail`、`Fail`。
- 前端新增事件订阅封装，接入现有任务追踪与 library watch toast 场景。
- 断线、浏览器不支持或后端不支持时，保留现有 `/tasks/{id}` 与 `/tasks/recent` 轮询。
- 更新 `API.md`、`README.md`、`CLAUDE.md`、`.cursor/rules/project-facts.mdc` 和本计划/需求清单的状态说明。

### 本轮不包含

- WebSocket。
- 后端持久通知中心。
- 用户可配置事件订阅过滤器。
- 用 SSE 替换所有轮询。
- 大文件导入 session 持久化。
- Android 客户端事件消费。

## 推荐方案

### 方案 A：任务事件 MVP（推荐）

后端在 `tasks.Manager` 内部增加轻量订阅机制。每次任务快照变化后发布 `task.updated` 事件；`GET /api/events` 负责把事件编码成 SSE：

```text
event: task.updated
data: {"type":"task.updated","task":{...TaskDTO}}
```

前端新增 `backend-events` 模块，用 `EventSource` 订阅 `/api/events`。`useScanTaskTracker` 优先消费正在跟踪的 taskId 的 `task.updated` 事件，轮询作为 fallback 和重连保护。`useLibraryWatchToasts` 可消费 terminal task 事件来减少 `/tasks/recent` 轮询压力，但仍保留 periodic recent poll，避免连接丢失时漏掉后台任务。

优点：改动集中，覆盖所有任务来源，不需要每个业务 handler 单独发事件。  
缺点：第一版只解决任务类实时性，更新/存储健康等事件后续再扩展。

### 方案 B：业务层事件总线

在 server/app 层增加独立事件 hub，由扫描、导入、更新、存储健康等业务调用点主动发布业务事件。

优点：事件语义更贴近业务，可以直接推 `storage.changed`、`update.available` 等事件。  
缺点：第一版需要改很多调用点，漏发风险高，不适合作为本轮 MVP。

### 方案 C：只做前端 EventSource 包装，不接业务

只新增 `/api/events` 心跳和前端连接状态，不改任务流。

优点：风险最低。  
缺点：不能实际完成“减少任务轮询碎片”的需求，用户价值不足。

## 离线资源设计

- 使用 npm 包 `hls.js`，`loadHlsLibrary()` 改为 `await import("hls.js")`，保持当前 `HlsCtor` 适配层和现有 playback 逻辑。
- 使用 `@fontsource/outfit` 或等价本地字体包，在 `src/main.ts` 或 `src/style.css` 引入 `400/500/600/700` 字重。
- 删除 `index.html` 中 `fonts.googleapis.com` / `fonts.gstatic.com` 的 preconnect 和 stylesheet。
- 构建产物应只引用本地 assets，不再包含 `cdn.jsdelivr.net`、`fonts.googleapis.com`、`fonts.gstatic.com`。

## SSE 后端设计

- 在 `backend/internal/tasks` 中新增：
  - `TaskEvent`：包含 `Type string` 与 `Task contracts.TaskDTO`。
  - `Subscribe(buffer int) (id string, ch <-chan TaskEvent, unsubscribe func())`。
  - 发布策略：非阻塞发送；订阅者 channel 满时丢弃该事件，避免慢客户端阻塞任务更新。
- `GET /api/events`：
  - 需要通过现有 PIN Auth middleware；锁定时返回 `423 AUTH_LOCKED`。
  - 响应头：`Content-Type: text/event-stream`、`Cache-Control: no-cache`、`Connection: keep-alive`。
  - 连接成功先发送 `event: hello`。
  - 任务事件发送 `event: task.updated`。
  - 周期性发送 comment heartbeat，防止代理空闲断开。
  - 客户端断开时自动 unsubscribe。

## 前端事件设计

- 新增 `src/lib/backend-events.ts`：
  - 负责构造 `/api/events` URL，兼容 `VITE_API_BASE_URL` 与 dev loopback 直连。
  - 暴露 `subscribeBackendEvents({ onTaskUpdated, onOpen, onError })`。
  - `EventSource` 使用 `withCredentials: true`，复用 PIN cookie。
- `useScanTaskTracker`：
  - `start(taskId)` 后同时启动 SSE 订阅和低频轮询 fallback。
  - 收到匹配 taskId 的 `task.updated` 后复用现有 terminal 处理逻辑。
  - 若 SSE 不可用，现有 500ms 轮询继续工作。
- `useLibraryWatchToasts`：
  - 从 `task.updated` terminal event 处理 fsnotify scan、linked scrape、asset download。
  - 保留 `/tasks/recent` 轮询作为补偿，轮询间隔可维持现状或放宽到较低频率。

## 测试策略

- 后端：
  - `backend/internal/tasks/manager_test.go`：验证 Create/Start/Progress/Complete 发布事件，unsubscribe 后不再收到事件，慢订阅者不阻塞。
  - `backend/internal/server/server_test.go` 或新 handler test：验证 `/api/events` 的 content type、hello 事件、task.updated 格式、断开取消订阅。
- 前端：
  - `src/lib/hls-player.test.ts`：验证 `loadHlsLibrary()` 通过动态 import 返回本地模块，不创建 CDN script。
  - 新增 backend events 单测：验证 task.updated JSON 解析、withCredentials、关闭连接。
  - 更新 `use-scan-task-tracker.test.ts`：验证匹配 task SSE 可更新 activeTask 和触发 terminal toast；轮询仍可 fallback。
  - 更新 `use-library-watch-toasts.test.ts`：验证 terminal task event 可触发现有 toast/reload 流。
- 构建/静态检查：
  - `pnpm test` 目标测试。
  - `pnpm typecheck`。
  - `cd backend && go test ./...`。
  - `pnpm build` 后搜索 `dist/` 中不包含 `cdn.jsdelivr.net`、`fonts.googleapis.com`、`fonts.gstatic.com`。

## 文档同步

实现完成后已同步：

- `README.md`：离线资源与 SSE 事件流说明。
- `API.md`：新增 `GET /api/events`。
- `CLAUDE.md`：API 列表与架构事实。
- `.cursor/rules/project-facts.mdc`：当前实现事实。
- `docs/plan/2026-05-15-next-requirement-ideas.md`：把两项候选标记为已落地。

## 实施结果

- 前端 HTML shell 已移除 Google Fonts 外链，Outfit 改为 `@fontsource/outfit` 本地资源。
- HLS fallback loader 已从 CDN script 注入改为 npm `hls.js` 动态 import。
- 后端新增 `tasks.Manager` 订阅/发布机制和 `GET /api/events` SSE 端点，当前事件为 `task.updated`。
- 前端新增 `src/lib/backend-events.ts`，并接入 `useScanTaskTracker` 与 `useLibraryWatchToasts`；原有 `/tasks/{taskId}` 与 `/tasks/recent` 轮询保留为 fallback。

## 建议执行顺序

1. 本地化 `hls.js` 与 Outfit 字体，先用前端测试和构建搜索验证离线资源。
2. 给 `tasks.Manager` 加事件订阅和后端 SSE 端点，先完成后端测试。
3. 前端新增 backend event client。
4. 接入 `useScanTaskTracker`。
5. 接入 `useLibraryWatchToasts`。
6. 更新文档并跑完整验证。

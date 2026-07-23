# Curated 后端 API 使用指南

本文档是 Curated 仓库的公开 HTTP API 指南，用于当前 Web 前端、后续 Android App、局域网客户端以及其他衍生项目对接同一个 Go 后端。

本文只描述当前 Go HTTP 后端已经实现的接口，不引入新 API 行为。

实现入口：

- 路由注册：`backend/internal/server/server.go`
- 分散 handler：`backend/internal/server/*.go`
- 后端 DTO / 错误码：`backend/internal/contracts/contracts.go`
- 当前前端调用封装：`src/api/endpoints.ts`
- 当前前端类型：`src/api/types.ts`

最后核对日期：2026-06-07。

## 1. 快速接入

### 1.1 Base URL

所有 HTTP API 都挂在 `/api` 前缀下。

常见地址：

| 场景 | API Base URL |
| --- | --- |
| Go 后端开发模式 | `http://127.0.0.1:8080/api` |
| Vite 前端代理 | 前端同源 `/api`，由 Vite 转发到 `127.0.0.1:8080` |
| 打包后的本机后端 | 通常为 `http://127.0.0.1:8081/api`，以发布配置为准 |
| Android / 局域网客户端 | `http://<运行后端的电脑IP>:<端口>/api`，例如 `http://192.168.1.23:8080/api` |

衍生客户端建议把 Base URL 做成可配置项，并在保存前统一去掉末尾 `/`。

### 1.2 最小连通性检查

```bash
curl http://127.0.0.1:8080/api/health
```

典型返回：

```json
{
  "name": "curated-dev",
  "version": "git.abcdef0",
  "channel": "dev",
  "installerVersion": "0.0.0",
  "transport": "http",
  "databasePath": "C:\\Users\\...\\curated.db"
}
```

### 1.3 Android / 非浏览器客户端建议

Android App 使用 OkHttp、Retrofit、Ktor Client 或其他 HTTP 客户端时，需要注意：

- Base URL 指向服务端电脑 IP：`http://<server-ip>:8080/api`。
- 开发期如果使用明文 HTTP，需要在 Android 网络安全配置中允许对应 IP 的 cleartext traffic。
- PIN 解锁依赖 `Set-Cookie: curated_auth=...; HttpOnly; SameSite=Lax`。非浏览器客户端不需要读取 cookie 内容，但必须用 CookieJar 保存并自动回传 `Cookie`。
- DTO 中的媒体 URL 可能是相对路径，例如 `/api/library/movies/{id}/stream`。客户端必须用后端 origin 解析成绝对 URL：`http://<server-ip>:8080/api/...`。
- 路径参数必须 URL encode，尤其是演员名、影片 ID、精选帧 ID、HLS 文件名。
- 大文件上传、视频播放、图片资源、导出下载不要按 JSON 解析。

## 2. 通用协议约定

### 2.1 成功响应不是统一 envelope

HTTP API 成功时直接返回 DTO 本体，不包 `{ "ok": true, "data": ... }`。

示例：

```json
{
  "items": [],
  "total": 0,
  "limit": 50,
  "offset": 0
}
```

空成功响应使用 `204 No Content`，例如删除、恢复、记录播放、更新播放进度等。

### 2.2 错误响应

失败响应统一返回 `AppError` JSON：

```json
{
  "code": "COMMON_BAD_REQUEST",
  "message": "invalid json body",
  "retryable": false,
  "details": {
    "field": "example"
  }
}
```

字段含义：

| 字段 | 含义 |
| --- | --- |
| `code` | 稳定机器可读错误码 |
| `message` | 面向调用方的简短说明 |
| `retryable` | 后端按状态码推导；`5xx` 通常为 `true` |
| `details` | 可选结构化调试信息 |

常见 HTTP 状态：

| 状态 | 场景 |
| --- | --- |
| `200 OK` | 普通查询 / 更新后返回 DTO |
| `201 Created` | 创建上传会话、添加库路径、创建播放会话 |
| `202 Accepted` | 扫描、导入、刮削等异步任务已接受 |
| `204 No Content` | 操作成功但无响应体 |
| `400 Bad Request` | 参数、JSON、文件、业务前置条件不合法 |
| `401 Unauthorized` | PIN 错误 |
| `404 Not Found` | 资源不存在 |
| `409 Conflict` | 状态冲突，例如扫描进行中、目标文件已存在、回收站状态冲突 |
| `423 Locked` | PIN App Lock 已启用且当前请求没有有效解锁会话 |
| `500 Internal Server Error` | 后端内部错误或运行时能力未配置 |

核心错误码：

| 错误码 | 常见含义 |
| --- | --- |
| `COMMON_BAD_REQUEST` | 请求参数、JSON、文件或字段值不合法 |
| `COMMON_FORBIDDEN` | 请求的浏览器 Origin 或 Host 不在服务端允许范围内 |
| `COMMON_NOT_FOUND` | 资源不存在 |
| `COMMON_INTERNAL` | 后端内部错误 |
| `COMMON_CONFLICT` | 当前状态不允许该操作 |
| `AUTH_LOCKED` | 应用已锁定，需要先解锁 |
| `AUTH_INVALID_PIN` | PIN 校验失败 |
| `IMPORT_TARGET_NOT_CONFIGURED` | 未配置默认导入库路径 |
| `IMPORT_TARGET_UNAVAILABLE` | 默认导入库路径不可用 |
| `IMPORT_CONFLICT` | 导入目标文件已存在 |
| `IMPORT_NOT_ENOUGH_SPACE` | 磁盘空间不足 |
| `IMPORT_COPY_FAILED` | 导入复制失败 |
| `IMPORT_SCAN_FAILED` | 文件已导入但后续扫描启动失败 |
| `IMPORT_UPLOAD_PERSIST_FAILED` | 分片已经写盘，但上传范围或会话状态未能持久化；调用方可安全重传同一范围 |
| `IMPORT_UPLOAD_UNRECOVERABLE` | 上传暂存文件或提交结果无法与 SQLite 会话记录安全协调 |
| `IMPORT_UPLOAD_EXPIRED` | 上传会话超过滑动有效期，已进入过期清理流程 |
| `APP_UPDATE_DOWNLOAD_FAILED` | 更新安装包下载失败 |
| `APP_UPDATE_INSTALL_FAILED` | 更新安装启动失败 |
| `CURATED_EXPORT_ACTOR_MISMATCH` | 精选帧导出时 `actorName` 不属于某一帧 |
| `PROVIDER_NOT_FOUND` | 元数据 provider 名称不存在 |
| `PROVIDER_PING_FAILED` | provider 连通性测试失败 |

### 2.3 认证与 Cookie

Curated 的 PIN App Lock 默认可关闭。开启后，除公开端点外，所有 `/api/*` 都需要有效 `curated_auth` cookie。

公开端点：

| Method | Path |
| --- | --- |
| `GET` | `/api/health` |
| `GET` | `/api/auth/status` |
| `POST` | `/api/auth/setup-pin` |
| `POST` | `/api/auth/unlock` |
| `POST` | `/api/auth/lock` |
| `OPTIONS` | 任意路径，用于 CORS preflight |

Cookie：

- 名称：`curated_auth`
- `HttpOnly`
- `Path=/`
- `SameSite=Lax`
- 普通会话使用浏览器 session cookie，服务端保存 idle deadline。
- `trustedForever=true` 会设置约 10 年 `Max-Age` 和 `Expires`，直到当前设备显式 lock 或未来会话管理能力撤销。

客户端处理建议：

- Web：`fetch` 必须带 `credentials: "include"`。
- Android：OkHttp 必须配置 CookieJar；Retrofit 只负责接口声明，cookie 仍由底层 client 管理。
- 收到 `423 AUTH_LOCKED` 时跳转到解锁页或弹出解锁流程，成功后重试原请求。

### 2.4 CORS 与客户端识别 Header

后端 CORS 行为：

- 浏览器 Origin 只有三类会被允许：请求同源、`localhost` / loopback 开发 Origin、主运行时 JSON 中 `corsAllowedOrigins` 配置的精确 Origin。
- 允许的 Origin 返回同值 `Access-Control-Allow-Origin` 与 `Access-Control-Allow-Credentials: true`；不会再对任意 Origin 回显，也不会返回通配符 `*`。
- 未列入允许范围的 Origin 返回 `403 COMMON_FORBIDDEN`，不能读取 API response。
- 非浏览器客户端可以不发送 `Origin`；Host 仍必须符合当前 loopback / LAN 监听策略，避免 DNS rebinding。
- 允许方法：`GET, POST, PUT, PATCH, DELETE, OPTIONS`

允许的请求头：

```text
Content-Type
Authorization
X-Curated-Offset
X-Curated-Chunk-Size
X-Curated-Chunk-SHA256
X-Curated-Client
X-Curated-Client-Version
X-Curated-OS
X-Curated-OS-Version
Sec-CH-UA-Platform
Sec-CH-UA-Platform-Version
```

衍生客户端可发送：

```text
X-Curated-Client: android
X-Curated-Client-Version: 0.1.0
X-Curated-OS: Android
X-Curated-OS-Version: 15
```

这些字段主要用于 `/api/connected-clients` 识别客户端类型，不参与鉴权。

### 2.5 URL、时间、分页

路径参数：

- `movieId`、`sessionId`、`uploadId`、`fileId`、`id`、`name`、`file` 都必须 URL encode。
- 演员名如果包含空格、斜杠、日文、中文，必须 encode。

时间：

- 后端大多数时间字段为 RFC3339 或 RFC3339Nano 字符串。
- `dayKey` 使用本地日历日 `YYYY-MM-DD`。
- 首页推荐使用 UTC 日期 `dateUtc`。

分页：

| 字段 | 含义 |
| --- | --- |
| `limit` | 页大小；不同资源默认值不同 |
| `offset` | 从 0 开始的偏移 |
| `total` | 当前过滤条件下总数 |

列表接口默认：

| 接口 | 默认 `limit` | 上限 |
| --- | --- | --- |
| `GET /library/movies` | 50 | 当前 HTTP handler 不强制上限 |
| `GET /library/actors` | 50 | 当前 handler 不强制上限 |
| `GET /library/actors/merge-audits` | 50 | 100 |
| `GET /insights/breakdown` | 10 | 25 |
| `GET /curated-frames` | 50 | 200 |
| `GET /tasks/recent` | 30 | 当前 handler 不强制上限 |
| `GET /playback/sessions/recent` | 20 | 当前 handler 不强制上限 |
| `GET /playback/watch-time/daily` | 91 天 | 91 天 |

### 2.6 媒体、Blob 与 Range

这些端点不是 JSON：

| 端点 | 内容类型 |
| --- | --- |
| `GET /api/library/movies/{movieId}/stream` | 视频文件，`http.ServeContent`，支持 Range / 206 |
| `GET /api/library/movies/{movieId}/asset/{kind}` | 本地封面 / 缩略图 |
| `GET /api/library/movies/{movieId}/asset/preview/{index}` | 预览图，可能从本地缓存或远端代理返回 |
| `GET /api/library/actors/{name}/asset/avatar` | 演员头像 |
| `GET /api/playback/sessions/{sessionId}/hls/{file}` | HLS playlist / segment |
| `GET /api/events` | Server-Sent Events (`text/event-stream`) |
| `GET /api/curated-frames/{id}/image` | 精选帧原图 |
| `GET /api/curated-frames/{id}/thumbnail` | 精选帧缩略图 |
| `POST /api/curated-frames/export` | 单图 `image/jpeg` / `image/png` / `image/webp`，多图 `application/zip` |

播放客户端建议：

- 优先调用 `/library/movies/{movieId}/playback` 获取播放描述，不直接拼 stream URL。
- `mode=direct` 时使用 `url` 播放；若为相对路径，按后端 origin 解析。
- `mode=hls` 时使用 `url` 指向的 `.m3u8`，并让播放器继续请求同一 session 下的 segment。
- Android 推荐使用 Media3 / ExoPlayer；直放 URL 支持 Range，HLS URL 需要按标准 HLS 播放。

### 2.7 异步任务

扫描、导入、元数据刮削、部分更新下载流程使用 `TaskDTO`。

通用流程：

1. 调用触发接口，收到 `202 Accepted` 和 `TaskDTO`。
2. 用 `taskId` 轮询 `GET /api/tasks/{taskId}`。
3. 如果只关心最近完成任务，调用 `GET /api/tasks/recent?limit=30`。

实时更新：Web API 客户端可以同时订阅 `GET /api/events`。当前 SSE 流会发送 `hello` 连接确认、`task.updated` 任务快照事件，以及 comment heartbeat；轮询端点仍保留为断线或不支持 SSE 时的 fallback。

任务状态：

| 状态 | 含义 |
| --- | --- |
| `pending` | 已创建但未开始 |
| `running` | 运行中 |
| `completed` | 成功完成 |
| `partial_failed` | 部分失败，例如导入成功但扫描失败 |
| `failed` | 失败 |
| `cancelled` | 已取消 |

`TaskDTO.metadata` 是面向场景的扩展字段，导入任务会包含拷贝进度、目标路径、失败文件等。

## 3. 推荐客户端工作流

### 3.1 启动连接与认证

1. `GET /health` 判断服务端是否可达，并读取版本。
2. `GET /auth/status` 判断是否需要 PIN。
3. 如果 `pinEnabled=true` 且 `unlocked=false`，调用 `POST /auth/unlock`。
4. 后续所有请求都自动带 cookie。
5. 任意 protected 请求返回 `423 AUTH_LOCKED` 时，重新执行解锁流程。

示例：

```bash
curl -i http://127.0.0.1:8080/api/auth/status

curl -i \
  -H "Content-Type: application/json" \
  -d "{\"pin\":\"1234\",\"trustedForever\":true}" \
  http://127.0.0.1:8080/api/auth/unlock
```

### 3.2 浏览影片

1. `GET /library/movies?limit=50&offset=0` 获取影片页。
2. 需要搜索时加 `q`，需要筛选演员或片商时加 `actor` / `studio`。
3. 详情页调用 `GET /library/movies/{movieId}`。
4. 封面、缩略图、预览图直接使用 DTO 中的 URL；相对 URL 解析为后端绝对 URL。
5. 收藏、评分、标签、展示字段覆盖使用 `PATCH /library/movies/{movieId}`。

示例：

```bash
curl "http://127.0.0.1:8080/api/library/movies?q=ABC&limit=24&offset=0"
```

### 3.3 播放影片

1. 调用 `GET /library/movies/{movieId}/playback`。
2. 如果返回 `mode=direct`，将 `url` 传给播放器。
3. 如果返回 `mode=hls`，将 `url` 传给 HLS 播放器。
4. 播放中周期性调用 `PUT /playback/progress/{movieId}`，建议 5 到 15 秒一次或在 pause / background 时保存。
5. 为统计热力图调用 `POST /playback/watch-time/daily`，单次 `watchedSec` 必须 `0 < watchedSec <= 300`。
6. 播放到达业务意义上的“已看”阈值时调用 `POST /library/played-movies/{movieId}`。

示例：

```bash
curl http://127.0.0.1:8080/api/library/movies/MOVIE_ID/playback
```

### 3.4 导入影片

普通导入适合较小文件：

1. `GET /settings` 获取 `defaultImportLibraryPathId`。
2. 如果未配置，通过 `PATCH /settings` 设置默认导入库路径。
3. `POST /import/movies` 上传 `multipart/form-data`。
4. 收到 `TaskDTO` 后轮询任务。

大文件导入推荐分片：

1. `POST /import/movies/uploads` 创建上传会话。
2. 按返回 `chunkSize` 切分文件。
3. 每片调用 `PUT /import/movies/uploads/{uploadId}/files/{fileId}/chunks/{chunkIndex}`。
4. 全部完成后 `POST /import/movies/uploads/{uploadId}/commit`。
5. 用户取消时 `DELETE /import/movies/uploads/{uploadId}`。

### 3.5 扫描与元数据刷新

库扫描：

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d "{\"paths\":[\"D:\\\\Library\"]}" \
  http://127.0.0.1:8080/api/scans
```

单片元数据刷新：

```bash
curl -X POST http://127.0.0.1:8080/api/library/movies/MOVIE_ID/scrape
```

按库路径批量刷新：

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d "{\"paths\":[\"D:\\\\Library\"]}" \
  http://127.0.0.1:8080/api/library/metadata-scrape
```

### 3.6 精选帧

当前精选帧流程：

1. 播放器截帧后调用 `POST /curated-frames` 保存图片和元数据。
2. 列表页调用 `GET /curated-frames`。
3. 缩略图调用 `GET /curated-frames/{id}/thumbnail`。
4. 原图调用 `GET /curated-frames/{id}/image`。
5. 标签编辑调用 `PATCH /curated-frames/{id}/tags`。
6. 导出调用 `POST /curated-frames/export`，按响应 `Content-Type` 保存文件。

## 4. Endpoint Reference

除 2.3 中列出的公开端点外，本章所有端点都受 PIN App Lock 保护。

### 4.1 Health / Runtime

#### `GET /api/health`

用途：检查后端进程、版本、通道、数据库路径。

认证：公开。

成功：`200 HealthDTO`

字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `name` | string | `curated-dev` 或 `curated` |
| `version` | string | 构建戳、git 标识或 `unknown` |
| `channel` | string | `dev` 或 `release` |
| `installerVersion` | string | 安装包版本；开发态常为 `0.0.0` |
| `transport` | string | 当前为 `http` |
| `databasePath` | string | SQLite 数据库路径 |

#### `GET /api/dev/performance`

用途：开发态性能监控摘要。

成功：`200 DevPerformanceSummaryDTO`

```json
{
  "supported": true,
  "sampledAt": "2026-06-07T12:00:00Z",
  "systemCpuPercent": 12.5,
  "backendCpuPercent": 1.2
}
```

如果运行时没有配置 provider，会返回 `{ "supported": false }`。

### 4.2 Auth / PIN App Lock

#### `GET /api/auth/status`

用途：读取 PIN App Lock 状态和当前请求是否已解锁。

认证：公开。

成功：`200 AuthStatusDTO`

```json
{
  "pinEnabled": true,
  "unlocked": false,
  "setupRequired": false,
  "pinLength": 4,
  "trustedForever": false,
  "sessionTtlMinutes": 60,
  "lanRequiresPin": true,
  "lockOnRestart": true
}
```

`lanRequiresPin` 是旧客户端兼容字段，当前固定为 `true`；Curated 已统一采用全局 PIN 锁，不再提供“本机与 LAN 分叉”的可写策略。

#### `POST /api/auth/setup-pin`

用途：设置初始 PIN，或在已解锁状态下重设 PIN，并创建当前客户端会话。

认证：公开；如果已经启用 PIN，则请求本身必须已解锁。

Body：

```json
{
  "pin": "1234",
  "confirmPin": "1234",
  "sessionTtlMinutes": 60,
  "lockOnRestart": true,
  "trustedForever": false
}
```

约束：

- `pin` 必须是 4 到 8 位数字。
- `confirmPin` 必须一致。
- 成功时设置 `curated_auth` cookie。

成功：`200 AuthStatusDTO`

常见错误：

- `400 COMMON_BAD_REQUEST`：PIN 格式不合法或确认不一致。
- `429 AUTH_RATE_LIMITED`：同一来源连续提交无效设置请求；响应含 `Retry-After` header 与 `details.retryAfterSeconds`。
- `423 AUTH_LOCKED`：已经设置 PIN 且当前请求未解锁。

#### `POST /api/auth/unlock`

用途：校验 PIN 并创建解锁会话。

认证：公开。

Body：

```json
{
  "pin": "1234",
  "trustedForever": true
}
```

成功：`200 AuthStatusDTO`，并设置 `curated_auth` cookie。

错误：

- `401 AUTH_INVALID_PIN`
- `429 AUTH_RATE_LIMITED`：连续 PIN 失败触发指数退避；响应含 `Retry-After` header，body 的 `retryable=true` 且 `details.retryAfterSeconds` 给出建议等待秒数。成功解锁后清除当前 IP 与客户端指纹的失败计数。

#### `POST /api/auth/change-pin`

用途：已解锁状态下修改 PIN。

Body：

```json
{
  "currentPin": "1234",
  "newPin": "5678",
  "confirmPin": "5678"
}
```

成功：`200 AuthStatusDTO`

错误：

- `401 AUTH_INVALID_PIN`：当前 PIN 错误。
- `400 COMMON_BAD_REQUEST`：新 PIN 格式不合法或确认不一致。

#### `POST /api/auth/lock`

用途：撤销当前会话并清除 cookie。

认证：公开。

成功：`200 AuthStatusDTO`

说明：会把当前请求携带的 `curated_auth` 会话从服务端撤销，包括 trusted-forever 会话。

#### `PATCH /api/auth/settings`

用途：更新非秘密安全设置。

Body：

```json
{
  "pinEnabled": true,
  "sessionTtlMinutes": 60,
  "lockOnRestart": true
}
```

所有字段可选；只发送要修改的字段。

成功：`200 AuthStatusDTO`

#### `GET /api/auth/sessions`

用途：列出仍有效的 trusted-forever 会话。返回安全的 `publicId`、IP、User-Agent、创建/最近活动时间和 `current` 标识；不会返回可用作 cookie 的真实 session token。

认证：需要已解锁会话。

成功：`200 AuthSessionsDTO`

```json
{
  "items": [
    {
      "publicId": "safe-public-reference",
      "userAgent": "Mozilla/5.0 ...",
      "ip": "192.168.1.20",
      "createdAt": "2026-07-19T12:00:00Z",
      "lastSeenAt": "2026-07-19T12:30:00Z",
      "trustedForever": true,
      "current": false
    }
  ]
}
```

#### `DELETE /api/auth/sessions/{publicId}`

用途：按非秘密 `publicId` 撤销一个 trusted-forever 会话。撤销当前会话时同时清除 `curated_auth` cookie。

认证：需要已解锁会话。

成功：`200 AuthSessionsDTO`（撤销后的列表）。

错误：`404 COMMON_NOT_FOUND` 表示会话已不存在或已撤销。

#### `POST /api/auth/sessions/revoke-others`

用途：撤销除当前会话之外的所有 trusted-forever 会话；当前是普通短会话时会撤销全部 trusted-forever 会话。

认证：需要已解锁会话。

成功：`200 AuthSessionsDTO`（撤销后的列表）。

### 4.3 Connected Clients

#### `GET /api/connected-clients`

用途：列出当前后端进程生命周期内访问过 API 的客户端。

成功：`200 ConnectedClientsDTO`

```json
{
  "clients": [
    {
      "key": "client-key",
      "ip": "192.168.1.50",
      "port": 53122,
      "hostname": "phone.local",
      "userAgent": "CuratedAndroid/0.1",
      "browser": "Curated Android",
      "os": "Android",
      "osVersion": "15",
      "deviceType": "mobile",
      "accessKind": "remote",
      "isLocalMachine": false,
      "firstSeen": "2026-06-07T12:00:00.000000000Z",
      "lastSeen": "2026-06-07T12:01:00.000000000Z",
      "requestCount": 12
    }
  ],
  "total": 1,
  "localCount": 0,
  "remoteCount": 1,
  "sampledAt": "2026-06-07T12:01:00Z"
}
```

说明：

- 数据只保存在内存中，重启后端会清空。
- 当前最多保留 50 个最近客户端。
- 不采集 MAC 地址。

### 4.4 App Update

这些接口用于桌面打包应用的更新检查。Android 或其他衍生客户端通常只需要忽略或用于展示服务端桌面版本状态。

#### `GET /api/app-update/status`

用途：读取缓存的更新状态。

成功：`200 AppUpdateStatusDTO`

#### `POST /api/app-update/check`

用途：强制向 GitHub Releases 检查最新版本。

成功：`200 AppUpdateStatusDTO`

#### `POST /api/app-update/download`

用途：下载并校验最新 Windows 安装包。

成功：`200 AppUpdateStatusDTO`

错误：

- `409 APP_UPDATE_DOWNLOAD_FAILED`

#### `POST /api/app-update/install`

用途：启动已下载并验证的安装包。

Body：

```json
{
  "mode": "interactive"
}
```

`mode` 可选：`interactive`、`silent`、`verysilent`。

成功：`200 AppUpdateStatusDTO`

错误：

- `400 COMMON_BAD_REQUEST`：body 不合法。
- `409 APP_UPDATE_INSTALL_FAILED`

#### `DELETE /api/app-update/downloaded-installer`

用途：删除已下载安装包并清理状态。

成功：`200 AppUpdateStatusDTO`

`AppUpdateStatusDTO` 关键字段：

| 字段 | 说明 |
| --- | --- |
| `supported` | 当前运行时是否支持更新 |
| `status` | `unsupported`、`up-to-date`、`update-available`、`error` |
| `installedVersion` / `latestVersion` | 本地和远端版本 |
| `hasUpdate` | 是否存在更新 |
| `installerDownloadUrl` / `installerSha256` | 安装包下载和校验信息 |
| `artifactStatus` | `downloading`、`downloaded`、`verified`、`failed`、`installing`、`install-launched` |
| `downloadProgress` | 下载进度百分比 |
| `installReady` | 是否可安装 |
| `releaseNotesSnippet` | GitHub Release 文本摘要 |

### 4.5 Homepage Recommendations

#### `GET /api/homepage/recommendations`

用途：获取 UTC 当日首页推荐快照。

成功：`200 HomepageDailyRecommendationsDTO`

```json
{
  "dateUtc": "2026-06-07",
  "generatedAt": "2026-06-07T00:00:00Z",
  "generationVersion": "v8",
  "heroMovieIds": ["movie-a", "movie-b"],
  "recommendationMovieIds": ["movie-c", "movie-d"],
  "recommendations": [
    {
      "movieId": "movie-c",
      "reasons": [{ "code": "recently_added" }, { "code": "well_rated" }],
      "feedbackEffects": []
    },
    {
      "movieId": "movie-d",
      "reasons": [{ "code": "rediscovery" }],
      "feedbackEffects": [
        {
          "feedbackId": "feedback_0123",
          "targetType": "actor",
          "targetValue": "Actor A",
          "effect": "weight_reduced"
        }
      ]
    }
  ]
}
```

说明：

- 后端按 UTC 日期生成并持久化。
- 同一天重复请求会复用快照，除非算法版本变化或手动刷新。
- `recommendationMovieIds` 保留为兼容、有序 ID 列表；`recommendations` 与其逐项同序且长度相同，提供真实生成理由和命中的降权反馈。
- 每项至少一个 `reasons`。当前 code：`high_user_rating`、`favorite`、`recently_added`、`well_rated`、`rediscovery`、`catalog_discovery`；Mock 的本地偏好算法还可返回 `shared_actor`、`shared_studio`、`shared_tag` 并携带 `entityType/entityValue`。前端只翻译 code，不自行伪造原因。
- `feedbackEffects` 只记录仍保留探索资格的 `less` 降权；影片级 `not_interested` / 活跃 `snooze` 会直接排除候选，因此不会出现在已选条目里。

#### `POST /api/homepage/recommendations/refresh`

用途：强制刷新 UTC 当日推荐。

Body 可选：

```json
{
  "preserveHeroMovieIds": ["movie-a", "movie-b"],
  "excludeRecommendationMovieIds": ["movie-c", "movie-d"]
}
```

成功：`200 HomepageDailyRecommendationsDTO`

刷新只重新生成快照，不会隐式创建任何反馈。

#### `GET /api/homepage/recommendations/feedback`

用途：列出当前有效的显式推荐反馈，按创建时间倒序。过期 `snooze` 不返回。

成功：`200 RecommendationFeedbackListDTO`

```json
{
  "items": [
    {
      "id": "feedback_0123",
      "action": "less",
      "targetType": "actor",
      "targetValue": "Actor A",
      "sourceMovieId": "movie-c",
      "createdAt": "2026-07-21T01:00:00Z",
      "updatedAt": "2026-07-21T01:00:00Z"
    }
  ]
}
```

#### `POST /api/homepage/recommendations/feedback`

用途：创建显式反馈。同 `action + targetType + 规范化 target` 重复提交幂等并返回原记录；最多保留 500 条有效反馈。

请求组合：

| action | targetType | 额外字段 | 语义 |
|---|---|---|---|
| `not_interested` | `movie` | 无 | 永久排除，直到删除反馈 |
| `snooze` | `movie` | `durationDays` 1～365 | 到期前排除 |
| `less` | `actor` / `studio` / `tag` | 无 | 降低权重但保留至少 10% 探索因子 |

所有请求必须带 `sourceMovieId`；影片必须仍在活动库，演员/片商/标签必须真实属于该影片。任意字符串、回收站影片和错误 action/target 组合都会被拒绝。

```json
{
  "action": "snooze",
  "targetType": "movie",
  "targetValue": "movie-c",
  "sourceMovieId": "movie-c",
  "durationDays": 7
}
```

成功：`201 RecommendationFeedbackDTO`。

错误码：`RECOMMENDATION_FEEDBACK_INVALID`、`RECOMMENDATION_FEEDBACK_TARGET_NOT_FOUND`、`RECOMMENDATION_FEEDBACK_LIMIT_REACHED`。

#### `DELETE /api/homepage/recommendations/feedback/{feedbackId}`

用途：撤销一条反馈。成功 `204`；不存在返回 `404 RECOMMENDATION_FEEDBACK_TARGET_NOT_FOUND`。删除只影响推荐生成，不删除影片、演员、标签、评分或观看记录。

### 4.6 Movies

#### `GET /api/library/movies`

用途：分页列出影片。

Query：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `mode` | string | 可用值：空 / `library` / `favorites` / `recent` / `tags` / `trash`。空与 `library` 返回活动库；`favorites` 只返回收藏；`recent` 只返回最近 30 天入库；`tags` 返回与活动库相同的影片集合，标签聚合与排序由前端布局完成；`trash` 返回回收站。其他值返回 `400 COMMON_BAD_REQUEST` |
| `q` | string | 子串搜索标题、番号、片商、简介，不区分大小写 |
| `tag` | string | 精确匹配元数据或用户标签 |
| `actor` | string | 精确匹配演员名 |
| `studio` | string | 精确匹配有效片商名 |
| `playState` | string | `all` / `unwatched` / `in-progress` / `completed`；播放进度达到 95% 视为完成 |
| `userRating` | number | 精确匹配用户本地评分 0～5；不使用刮削评分替代 |
| `resolution` | string | 精确分辨率；`4k` 同时匹配 `4K` / `2160p` / `UHD` / `3840x2160` |
| `addedAfter` | string | RFC3339 或 `YYYY-MM-DD` 入库时间下界 |
| `limit` | number | 默认 50 |
| `offset` | number | 默认 0 |

成功：`200 MoviesPageDTO`

```json
{
  "items": [
    {
      "id": "movie-1",
      "title": "Example Title",
      "code": "ABC-001",
      "studio": "Studio",
      "actors": ["Actor A"],
      "tags": ["metadata"],
      "userTags": ["favorite"],
      "runtimeMinutes": 120,
      "rating": 4.5,
      "userRating": 5,
      "isFavorite": true,
      "addedAt": "2026-06-07T12:00:00Z",
      "location": "D:\\Library\\ABC-001.mp4",
      "resolution": "1080p",
      "year": 2026,
      "releaseDate": "2026-01-01",
      "coverUrl": "/api/library/movies/movie-1/asset/cover?v=...",
      "thumbUrl": "/api/library/movies/movie-1/asset/thumb?v=..."
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

说明：

- 普通列表默认排除回收站；`mode=trash` 只返回回收站。
- 普通列表按 `addedAt DESC, id ASC`；回收站按 `trashedAt DESC, id ASC`。
- `rating` 是有效评分：用户评分优先，否则使用元数据评分。
- `userRating` 仅在存在用户本地评分时返回，供 Saved Views 等功能区分用户信号与刮削评分。
- `tags` 是元数据 / NFO 标签；`userTags` 是本地用户标签。
- `coverUrl`、`thumbUrl`、`previewImages` 可能为相对 API URL。

#### `GET /api/library/movies/{movieId}`

用途：获取影片详情。

成功：`200 MovieDetailDTO`

`MovieDetailDTO` 在列表字段基础上增加：

| 字段 | 说明 |
| --- | --- |
| `summary` | 简介 |
| `previewImages` | 预览图 URL 数组 |
| `previewVideoUrl` | 预览视频 URL |
| `metadataRating` | 元数据评分 |
| `userRating` | 用户评分；无覆盖时省略或为 null |
| `actorAvatarUrls` | 演员名到头像 URL 的映射 |

错误：

- `404 COMMON_NOT_FOUND`

#### `PATCH /api/library/movies/{movieId}`

用途：更新影片本地用户态和展示字段覆盖。

Body，所有字段可选，但至少要有一个字段：

```json
{
  "isFavorite": true,
  "rating": 4.25,
  "userTags": ["tag-a", "tag-b"],
  "metadataTags": ["nfo-a"],
  "userTitle": "自定义标题",
  "userStudio": "自定义片商",
  "userSummary": "自定义简介",
  "userReleaseDate": "2026-01-01",
  "userRuntimeMinutes": 118
}
```

清除覆盖：

```json
{
  "rating": null,
  "userTitle": null,
  "userStudio": "",
  "userSummary": null,
  "userReleaseDate": null,
  "userRuntimeMinutes": null
}
```

约束：

- `rating` 范围 `0..5`；`null` 表示清除用户评分。
- `userTags` 和 `metadataTags` 出现时是整表替换；空数组表示清空。
- 用户标签最多 64 个，每个最多 64 个 Unicode 字符，后端会 trim 和去重。
- `userReleaseDate` 必须是 `YYYY-MM-DD`。
- `userRuntimeMinutes` 必须是 `0..99999` 的整数。
- `userSummary` 最多约 120000 字符。
- 回收站影片不能 patch，返回 `409 COMMON_CONFLICT`。

成功：`200 MovieDetailDTO`

#### `DELETE /api/library/movies/{movieId}`

用途：把影片移入回收站。

成功：`204 No Content`

错误：

- `404 COMMON_NOT_FOUND`

#### `DELETE /api/library/movies/{movieId}?permanent=true`

用途：永久删除影片数据库记录和相关磁盘文件。

前置条件：影片必须已经在回收站。

成功：`204 No Content`

错误：

- `400 COMMON_BAD_REQUEST`：影片不在回收站。
- `404 COMMON_NOT_FOUND`

#### `POST /api/library/movies/{movieId}/restore`

用途：从回收站恢复影片。

成功：`204 No Content`

错误：

- `400 COMMON_BAD_REQUEST`：影片不在回收站。
- `404 COMMON_NOT_FOUND`

#### `POST /api/library/movies/{movieId}/reveal`

用途：在服务端机器文件管理器中定位影片文件。

成功：`204 No Content`

说明：这是桌面 / 本机能力。Android 或远程客户端调用只会让服务端电脑打开文件管理器。

#### `GET /api/library/movies/{movieId}/comment`

用途：读取影片备注。

成功：`200 MovieCommentDTO`

```json
{
  "body": "备注正文",
  "updatedAt": "2026-06-07T12:00:00Z"
}
```

无备注时返回：

```json
{
  "body": "",
  "updatedAt": ""
}
```

#### `PUT /api/library/movies/{movieId}/comment`

用途：新增或替换影片备注。

Body：

```json
{
  "body": "备注正文"
}
```

约束：

- 后端会 trim。
- 最多 10000 个 Unicode 字符。
- 回收站影片不能写备注，返回 `409 COMMON_CONFLICT`。

成功：`200 MovieCommentDTO`

#### `GET /api/library/movies/{movieId}/stream`

用途：直放主视频文件。

成功：`200 OK` 或 `206 Partial Content`

说明：

- 支持 `GET` 和 `HEAD`。
- 支持 Range，由 `http.ServeContent` 处理。
- 客户端通常不应直接拼接该 URL，而是先调用 `/playback` 获取 descriptor。

#### `GET /api/library/movies/{movieId}/asset/{kind}`

用途：获取影片封面或缩略图。

Path：

| 参数 | 可用值 |
| --- | --- |
| `kind` | `cover`、`thumb` |

成功：图片 bytes。

说明：

- 支持 `GET` 和 `HEAD`。
- `Cache-Control: private, max-age=604800, immutable`

#### `GET /api/library/movies/{movieId}/asset/preview/{index}`

用途：获取第 `index` 张预览图。

Path：

| 参数 | 说明 |
| --- | --- |
| `index` | 从 1 开始 |

成功：图片 bytes。

说明：

- 本地缓存存在时使用本地缓存。
- 本地不存在时，后端可能代理远端预览图源。
- 代理远端时 `Cache-Control: private, no-cache`。

#### `POST /api/library/movies/{movieId}/scrape`

用途：触发单片元数据刷新。

成功：`202 TaskDTO`

错误：

- `400 COMMON_BAD_REQUEST`：影片没有番号或没有视频路径。
- `404 COMMON_NOT_FOUND`
- `409 COMMON_CONFLICT`：影片在回收站。

#### `POST /api/library/metadata-scrape`

用途：按配置库路径批量刷新元数据。

Body：

```json
{
  "paths": ["D:\\Library"]
}
```

成功：`202 MetadataRefreshQueuedDTO`

```json
{
  "queued": 10,
  "skipped": 2,
  "invalidPaths": []
}
```

约束：

- `paths` 至少包含一个路径。
- 路径应匹配已配置的库根。

### 4.7 Played Movies

#### `GET /api/library/played-movies`

用途：获取已播放影片 ID 列表。

成功：`200 PlayedMoviesListDTO`

```json
{
  "movieIds": ["movie-1", "movie-2"]
}
```

#### `POST /api/library/played-movies/{movieId}`

用途：记录影片已播放。

成功：`204 No Content`

错误：

- `404 COMMON_NOT_FOUND`

### 4.8 Actors

#### `GET /api/library/actors`

用途：分页列出演员。

Query：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `q` | string | canonical 演员名、历史 alias 或演员用户标签子串，不区分大小写 |
| `actorTag` | string | 精确匹配演员用户标签 |
| `sort` | string | `name` 默认；`movieCount` 按参演数量降序 |
| `limit` | number | 默认 50 |
| `offset` | number | 默认 0 |

成功：`200 ActorsListDTO`

```json
{
  "total": 1,
  "actors": [
    {
      "name": "Actor A",
      "avatarUrl": "/api/library/actors/Actor%20A/asset/avatar?v=...",
      "avatarRemoteUrl": "https://...",
      "avatarLocalUrl": "/api/library/actors/Actor%20A/asset/avatar?v=...",
      "hasLocalAvatar": true,
      "movieCount": 12,
      "userTags": ["favorite"]
    }
  ]
}
```

说明：只列出至少有一部 active 影片的 canonical 演员；alias 搜索命中时仍返回 canonical 演员行。

#### `GET /api/library/actors/profile?name={name}`

用途：获取演员资料。

Query：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `name` | 是 | canonical 演员名或已归并 alias |

成功：`200 ActorProfileDTO`

```json
{
  "name": "Actor A",
  "avatarUrl": "/api/library/actors/Actor%20A/asset/avatar?v=...",
  "avatarRemoteUrl": "https://...",
  "avatarLocalUrl": "/api/library/actors/Actor%20A/asset/avatar?v=...",
  "hasLocalAvatar": true,
  "summary": "Profile summary",
  "homepage": "https://...",
  "provider": "metatube",
  "providerActorId": "123",
  "height": 160,
  "birthday": "2000-01-01",
  "profileUpdatedAt": "2026-06-07T12:00:00Z",
  "userTags": ["favorite"],
  "externalLinks": ["https://example.com"],
  "aliases": ["Old Actor Name"]
}
```

alias 请求会解析到 canonical profile；响应中的 `name` 始终是 canonical 展示名。旧演员详情 URL、资料库 `actor=` 筛选、头像、标签、外链与刮削入口使用相同解析规则。

#### `GET /api/library/actors/{name}/asset/avatar`

用途：获取本地缓存头像。

成功：图片 bytes。

说明：

- 支持 `GET` 和 `HEAD`。
- `Cache-Control: private, max-age=604800, immutable`

#### `POST /api/library/actors/scrape?name={name}`

用途：触发演员资料刮削。

成功：`202 TaskDTO`

错误：

- `400 COMMON_BAD_REQUEST`：缺少 `name`。
- `404 COMMON_NOT_FOUND`

#### `PATCH /api/library/actors/tags?name={name}`

用途：替换演员用户标签。

Body：

```json
{
  "userTags": ["tag-a", "tag-b"]
}
```

成功：`200 ActorListItemDTO`

约束：同用户标签规则，最多 64 个，每个最多 64 字符；后端 trim、去重。

#### `PATCH /api/library/actors/external-links?name={name}`

用途：替换演员外部链接列表。

Body：

```json
{
  "externalLinks": ["https://example.com/profile"]
}
```

成功：`200 ActorProfileDTO`

约束：

- 最多 16 个链接。
- 每个链接最多 2048 字符。
- 必须是合法 `http` 或 `https` URL。
- 后端 trim、去重。

#### `POST /api/library/actors/merge-preview`

用途：只读预览把一个仍存在的 canonical source 演员归并到现有 target 演员。target 可以使用 canonical 名或已存在 alias；已经归并的 alias 不能再次作为 source。

Body：

```json
{
  "sourceName": "Old Actor Name",
  "targetName": "Canonical Actor Name"
}
```

成功：`200 ActorMergePreviewDTO`

```json
{
  "previewToken": "8d62...",
  "source": {
    "id": 10,
    "name": "Old Actor Name",
    "aliases": ["Older Name"]
  },
  "target": {
    "id": 20,
    "name": "Canonical Actor Name",
    "aliases": []
  },
  "movies": {
    "sourceCount": 8,
    "targetCount": 12,
    "duplicateCount": 2,
    "resultCount": 18
  },
  "userTags": {
    "source": ["source-tag"],
    "target": ["target-tag"],
    "result": ["target-tag", "source-tag"]
  },
  "externalLinks": {
    "source": ["https://source.example"],
    "target": ["https://target.example"],
    "result": ["https://target.example", "https://source.example"]
  },
  "recommendationFeedback": {
    "sourceCount": 1,
    "targetCount": 1,
    "duplicateCount": 1,
    "resultCount": 1
  },
  "curatedFramesAffected": 3,
  "aliasesToMove": ["Old Actor Name", "Older Name"],
  "profileFields": [
    {
      "field": "summary",
      "sourceValue": "Source summary",
      "targetValue": "Target summary",
      "defaultSelection": "target",
      "conflict": true
    }
  ],
  "canApply": true,
  "blockingReasons": [],
  "requiredDecisions": ["summary"]
}
```

预览完全只读。`previewToken` 是覆盖完整预览状态的 opaque SHA-256 token，包括 source/target profile 内部状态、影片 ID、用户标签、外链、alias、演员推荐反馈与会受影响的萃取帧演员 JSON；客户端不得自行生成或解释它。外链合并后超过 16 项、alias/name 冲突等情况返回 `canApply:false` 和 `blockingReasons`，而不是写入任何数据。

#### `POST /api/library/actors/merge`

用途：明确确认并事务化应用一次刚刚预览过的演员归并。

Body：

```json
{
  "sourceName": "Old Actor Name",
  "targetName": "Canonical Actor Name",
  "previewToken": "8d62...",
  "confirm": true,
  "profileDecisions": {
    "summary": "target",
    "providerActorId": "source"
  }
}
```

约束与行为：

- `confirm` 必须为 `true`，并且必须携带最近 preview 返回的 token。
- apply 在同一个 SQLite 事务内重新生成完整 preview；任一相关内容变化都会以 stale preview 拒绝。
- `profileDecisions` 只接受 preview 中列出的字段，值只能是 `source` 或 `target`；两个不同非空值的字段必须显式选择。
- `movie_actors`、演员用户标签、外链、profile、头像本地状态、aliases、演员推荐反馈与萃取帧演员列表按 preview 规则迁移并去重，然后才删除 source actor。
- 任何步骤失败都会回滚，零部分写入；成功后写入持久化 merge audit。
- 已经作为 target 的 canonical actor 后续仍可继续归并到另一个 canonical actor；历史 audit 保存当时的数字 ID、名称和摘要快照，不钉死活跃演员行。

成功：`200 ActorMergeAuditDTO`

```json
{
  "id": "amrg_0123456789abcdef",
  "sourceActorId": 10,
  "targetActorId": 20,
  "sourceName": "Old Actor Name",
  "targetName": "Canonical Actor Name",
  "previewToken": "8d62...",
  "summary": {
    "movies": {
      "sourceCount": 8,
      "targetCount": 12,
      "duplicateCount": 2,
      "resultCount": 18
    },
    "userTags": ["target-tag", "source-tag"],
    "externalLinks": ["https://target.example", "https://source.example"],
    "aliases": ["Old Actor Name", "Older Name"],
    "recommendationFeedback": {
      "sourceCount": 1,
      "targetCount": 1,
      "duplicateCount": 1,
      "resultCount": 1
    },
    "curatedFramesAffected": 3,
    "profileDecisions": {
      "summary": "target",
      "providerActorId": "source"
    }
  },
  "appliedAt": "2026-07-21T04:00:00Z"
}
```

#### `GET /api/library/actors/merge-audits`

用途：分页读取不可变的演员归并审计。

Query：

| 参数 | 说明 |
| --- | --- |
| `limit` | 默认 50，最大 100 |
| `offset` | 默认 0；负数归零 |

成功：`200 ActorMergeAuditListDTO`，结构为 `{ items, total, limit, offset }`。

演员归并错误码：

| HTTP | Code | 含义 |
| --- | --- | --- |
| `400` | `ACTOR_MERGE_INVALID` | body、确认、字段或 profile decision 非法 |
| `404` | `ACTOR_MERGE_NOT_FOUND` | source 或 target 不存在 |
| `409` | `ACTOR_MERGE_SELF` | source 与 target 解析为同一演员 |
| `409` | `ACTOR_MERGE_SOURCE_IS_ALIAS` | 已归并 alias 被再次作为 source |
| `409` | `ACTOR_MERGE_CONFLICT` | alias/name 或 profile decision 冲突 |
| `409` | `ACTOR_MERGE_STALE_PREVIEW` | preview 后完整相关状态已变化 |
| `409` | `ACTOR_MERGE_LINK_LIMIT` | 合并后外链超过 16 项 |

### 4.9 Playback

#### `GET /api/library/movies/{movieId}/playback`

用途：获取播放描述，客户端应以此作为播放入口。

成功：`200 PlaybackDescriptorDTO`

```json
{
  "movieId": "movie-1",
  "mode": "direct",
  "url": "/api/library/movies/movie-1/stream",
  "mimeType": "video/mp4",
  "fileName": "ABC-001.mp4",
  "durationSec": 7200,
  "resumePositionSec": 120.5,
  "canDirectPlay": true,
  "audioTracks": [],
  "subtitleTracks": []
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `mode` | `direct`、`hls`、`native` |
| `sessionId` | HLS / 推流会话 ID |
| `sessionKind` | 会话类型诊断字段 |
| `url` | 播放 URL，可能是相对路径 |
| `mimeType` | 媒体类型 |
| `transcodeProfile` | 转码档位 |
| `startPositionSec` | 请求创建会话时的起播点 |
| `resumePositionSec` | 已保存续播点 |
| `canDirectPlay` | 是否支持直放 |
| `reasonCode` / `reasonMessage` | 模式选择诊断 |
| `audioTracks` / `subtitleTracks` | 音轨 / 字幕轨信息 |

#### `POST /api/library/movies/{movieId}/playback-session`

用途：显式创建播放会话，例如 HLS 推流。

Body：

```json
{
  "mode": "hls",
  "startPositionSec": 120.5
}
```

`mode` 省略时默认为 `direct`。

成功：`201 PlaybackDescriptorDTO`

错误：

- `400 COMMON_BAD_REQUEST`：请求模式或运行时前置条件不满足。
- `404 COMMON_NOT_FOUND`

#### `POST /api/library/movies/{movieId}/native-play`

用途：让服务端机器启动外部本地播放器。

Body 可选：

```json
{
  "startPositionSec": 120.5
}
```

成功：`200 NativePlaybackLaunchDTO`

说明：远程 / Android 客户端调用时，播放器会在服务端电脑上启动，不会在手机上播放。

#### `GET /api/playback/sessions/recent`

用途：列出活跃和最近归档的播放会话。

Query：

| 参数 | 说明 |
| --- | --- |
| `limit` | 默认 20 |

成功：`200 PlaybackSessionListDTO`

#### `GET /api/playback/sessions/{sessionId}`

用途：读取播放会话状态。

成功：`200 PlaybackSessionStatusDTO`

```json
{
  "sessionId": "session-1",
  "movieId": "movie-1",
  "sessionKind": "hls",
  "transcodeProfile": "default",
  "startPositionSec": 120.5,
  "startedAt": "2026-06-07T12:00:00Z",
  "lastAccessedAt": "2026-06-07T12:01:00Z",
  "expiresAt": "2026-06-07T13:00:00Z",
  "state": "running"
}
```

#### `GET /api/playback/sessions/{sessionId}/hls/{file}`

用途：读取 HLS playlist 或 segment。

成功：

- `.m3u8`：`application/vnd.apple.mpegurl`
- `.ts`：`video/mp2t`
- 其他：`http.ServeFile` 自动推断

说明：

- 支持 `GET` 和 `HEAD`。
- 设置 `Cache-Control: no-store, no-cache, must-revalidate`。

#### `DELETE /api/playback/sessions/{sessionId}`

用途：结束 / 删除播放会话。

成功：`204 No Content`

#### `GET /api/playback/progress`

用途：列出已保存播放进度。

成功：`200 PlaybackProgressListDTO`

```json
{
  "items": [
    {
      "movieId": "movie-1",
      "positionSec": 120.5,
      "durationSec": 7200,
      "updatedAt": "2026-06-07T12:00:00Z"
    }
  ]
}
```

#### `PUT /api/playback/progress/{movieId}`

用途：保存单片续播进度。

Body：

```json
{
  "positionSec": 120.5,
  "durationSec": 7200
}
```

成功：`204 No Content`

错误：

- `404 COMMON_NOT_FOUND`

#### `DELETE /api/playback/progress/{movieId}`

用途：清除单片续播进度。

成功：`204 No Content`

#### `GET /api/playback/watch-time/daily`

用途：读取每日观看时长统计。

Query：

| 参数 | 说明 |
| --- | --- |
| `days` | 默认 91，最大 91 |

成功：`200 PlaybackWatchTimeDailyListDTO`

```json
{
  "items": [
    {
      "dayKey": "2026-06-07",
      "watchedSec": 1800
    }
  ],
  "totalWatchedSec": 1800,
  "activeDays": 1,
  "maxDayWatchedSec": 1800,
  "longestStreakDays": 1
}
```

#### `POST /api/playback/watch-time/daily`

用途：追加一段观看时长。

Body：

```json
{
  "movieId": "movie-1",
  "dayKey": "2026-06-07",
  "watchedSec": 30
}
```

约束：

- `dayKey` 必须是合法 `YYYY-MM-DD`。
- `watchedSec` 必须 `> 0` 且 `<= 300`。
- 同一天同影片会累加。

成功：`204 No Content`

### 4.10 Settings

#### `GET /api/settings`

用途：读取当前设置。

成功：`200 SettingsDTO`

关键字段：

| 字段 | 说明 |
| --- | --- |
| `libraryPaths` | 已配置库根 |
| `defaultImportLibraryPathId` | 默认导入目标库根 ID |
| `player` | 播放器设置 |
| `organizeLibrary` | 扫描后是否整理目录 |
| `autoLibraryWatch` | 是否启用目录监听扫描 |
| `autoActorProfileScrape` | 影片刮削后是否自动补演员资料 |
| `autoDownloadUpdates` | 启动更新检查后是否自动下载 |
| `launchAtLogin` | 桌面端是否登录自启 |
| `launchAtLoginSupported` | 当前运行时是否支持登录自启 |
| `curatedFrameExportFormat` | `jpg`、`webp`、`png` |
| `metadataMovieProvider` | 单一影片元数据 provider |
| `metadataMovieProviders` | 当前可用 provider 列表 |
| `metadataMovieProviderChain` | 链式 provider 顺序 |
| `metadataMovieScrapeMode` | `auto`、`specified`、`chain` |
| `metadataMovieStrategy` | `auto-global`、`auto-cn-friendly`、`custom-chain`、`specified` |
| `proxy` | 出站代理设置 |
| `backendLog` | 后端日志设置 |

#### `PATCH /api/settings`

用途：部分更新设置。

Body 示例：

```json
{
  "defaultImportLibraryPathId": "library-path-id",
  "curatedFrameExportFormat": "jpg",
  "autoLibraryWatch": true,
  "player": {
    "hardwareDecode": true,
    "hardwareEncoder": "auto",
    "nativePlayerEnabled": false,
    "streamPushEnabled": true,
    "forceStreamPush": false,
    "ffmpegCommand": "ffmpeg",
    "preferNativePlayer": false,
    "seekForwardStepSec": 10,
    "seekBackwardStepSec": 10
  },
  "metadataMovieScrapeMode": "chain",
  "metadataMovieProviderChain": ["provider-a", "provider-b"],
  "proxy": {
    "enabled": true,
    "url": "http://127.0.0.1:7890",
    "username": "",
    "password": ""
  },
  "backendLog": {
    "logDir": "D:\\CuratedLogs",
    "logFilePrefix": "curated",
    "logMaxAgeDays": 14,
    "logLevel": "info"
  }
}
```

成功：`200 SettingsDTO`

约束：

- 至少发送一个支持字段，否则 `400 COMMON_BAD_REQUEST`。
- `curatedFrameExportFormat` 必须是 `jpg`、`webp`、`png`。
- `defaultImportLibraryPathId` 非空时必须存在。
- `metadataMovieProvider` 和 `metadataMovieProviderChain` 中的 provider 必须在 `metadataMovieProviders` 中。
- `metadataMovieScrapeMode` 必须是 `auto`、`specified`、`chain`。
- `metadataMovieStrategy` 必须是 `auto-global`、`auto-cn-friendly`、`custom-chain`、`specified`。
- `proxy.enabled=true` 时 `proxy.url` 不能为空。
- 后端日志 sink 的部分变更需要重启后端才完全生效。

### 4.11 Library Paths / Storage

#### `POST /api/library/paths`

用途：添加库根路径，并尽量启动首次扫描。

Body：

```json
{
  "path": "D:\\Library",
  "title": "主库"
}
```

成功：`201 AddLibraryPathResponse`

```json
{
  "id": "library-path-id",
  "path": "D:\\Library",
  "title": "主库",
  "firstLibraryScanPending": true,
  "scanTask": {
    "taskId": "task-1",
    "type": "scan",
    "status": "running",
    "createdAt": "2026-06-07T12:00:00Z",
    "progress": 0
  }
}
```

错误：

- `400 COMMON_BAD_REQUEST`：路径为空或不是绝对路径。
- `409 COMMON_CONFLICT`：库路径重复。

#### `PATCH /api/library/paths/{id}`

用途：更新库路径展示标题。

Body：

```json
{
  "title": "新标题"
}
```

成功：`200 LibraryPathDTO`

#### `DELETE /api/library/paths/{id}`

用途：删除库根配置，并清理不再属于任何库根的影片记录。

成功：`204 No Content`

说明：不会删除磁盘上的库目录本身。

#### `POST /api/library/paths/{id}/reveal`

用途：在服务端机器文件管理器中打开库目录。

成功：`204 No Content`

#### `GET /api/library/paths/storage-status`

用途：读取库根存储状态快照。

成功：`200 LibraryPathStorageStatusListDTO`

```json
{
  "items": [
    {
      "libraryPathId": "library-path-id",
      "path": "D:\\Library",
      "title": "主库",
      "status": "online",
      "message": "storage is online",
      "checkedAt": "2026-06-07T12:00:00Z",
      "rootPath": "D:\\",
      "driveType": "fixed",
      "volumeLabel": "Data",
      "fileSystem": "NTFS",
      "identityConfidence": "high",
      "expectedVolumeId": "vol-a",
      "currentVolumeId": "vol-a",
      "canRescan": true,
      "canImport": true
    }
  ]
}
```

状态值：

| 值 | 含义 |
| --- | --- |
| `online` | 目录可达且卷身份匹配 |
| `offline` | 存储根不可用 |
| `volume_mismatch` | 路径解析到不同卷 |
| `path_missing` | 存储存在但库目录缺失 |
| `permission_denied` | 权限不足 |
| `unknown` | 无法分类 |

#### `POST /api/library/paths/storage-status/check`

用途：执行一次新的存储状态检测。

Body 可选：

```json
{
  "libraryPathIds": ["library-path-id"]
}
```

省略或空数组表示检测全部。

成功：`200 LibraryPathStorageStatusListDTO`

#### `POST /api/library/paths/{id}/storage-binding/rebind`

用途：把库根绑定到当前检测到的卷身份，用于恢复 `volume_mismatch`。

成功：`200 LibraryPathStorageStatusDTO`

说明：只有当前路径被识别为 `online` 时才会持久化新绑定。

### 4.12 Movie Imports

#### `POST /api/import/movies`

用途：普通 multipart 导入影片文件。

Content-Type：`multipart/form-data`

Form fields：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `files` | file，重复 | 影片文件 |
| `relativePath` | string，重复 | 可在每个文件前传入，用于保留目录相对路径 |
| `totalBytes` | string / number | 可选，总字节数，用于进度 |

成功：`202 TaskDTO`

约束与行为：

- 必须先配置 `defaultImportLibraryPathId`。
- 只接受视频扩展名：`.mp4`, `.m4v`, `.mkv`, `.avi`, `.mov`, `.wmv`, `.webm`, `.ts`, `.m2ts`, `.flv`, `.mpeg`, `.mpg`, `.ogv`, `.rmvb`, `.iso`。
- 后端复制文件到默认库根，不移动或删除客户端源文件。
- 目标文件已存在则该文件失败，不覆盖。
- 至少成功复制一个文件时会尝试启动受限扫描。
- 成功或部分失败都可能返回 `202 TaskDTO`，客户端要看 `status` 和 `metadata.errorItems`。

导入任务 `metadata` 常见字段：

| 字段 | 说明 |
| --- | --- |
| `targetLibraryPathId` | 目标库根 ID |
| `targetPath` | 目标库根路径 |
| `stage` | `copying`、`completed`、`partial_failed`、`failed` |
| `totalFiles` / `completedFiles` / `failedFiles` | 文件计数 |
| `copiedBytes` / `totalBytes` | 字节进度 |
| `currentFileName` | 当前文件名 |
| `errorItems` | 失败文件数组，含 `fileName`、`code`、`message` |
| `scanTaskId` | 后续扫描任务 ID |
| `scanError` | 扫描启动错误 |

#### `POST /api/import/movies/uploads`

用途：创建可续传 / 分片导入会话。

Body：

```json
{
  "files": [
    {
      "relativePath": "Folder/ABC-001.mp4",
      "size": 1234567890,
      "lastModified": 1710000000000
    }
  ]
}
```

成功：`201 MovieImportUploadDTO`

```json
{
  "uploadId": "upload_0123456789abcdef",
  "targetPath": "D:\\Library",
  "chunkSize": 33554432,
  "bytesReceived": 0,
  "totalBytes": 1234567890,
  "state": "uploading",
  "expiresAt": "2026-07-21T12:00:00Z",
  "recoveryStatus": "ready",
  "files": [
    {
      "fileId": "file_0123456789abcdef",
      "relativePath": "Folder\\ABC-001.mp4",
      "size": 1234567890,
      "bytesReceived": 0,
      "complete": false,
      "state": "pending"
    }
  ],
  "task": {
    "taskId": "task-1",
    "type": "import.movies",
    "status": "running",
    "createdAt": "2026-06-07T12:00:00Z",
    "progress": 0
  }
}
```

说明：

- 默认 chunk size：32 MiB。
- manifest 请求体上限为 2 MiB，最多 10,000 个文件；未知字段、尾随 JSON、重复目标路径、空文件和总大小溢出都会返回 `400 COMMON_BAD_REQUEST`。
- migration `0029_movie_import_upload_sessions.sql` 把 session、文件、已接收 chunk 范围和清理审计持久化到 SQLite。活跃会话使用 24 小时滑动有效期，每个成功分片会续期。
- staging 目录：`<target-library-root>/.curated-import/<uploadId>/`。
- 后端启动时会从 SQLite 恢复 `uploading` / `committing` 会话及原 task ID，并以 chunk 范围账本重新推导字节计数；不会把预分配文件长度当成已上传字节。
- `recoveryStatus` 为 `ready`、`unavailable` 或 `unrecoverable`。目标盘暂时离线时返回 `unavailable` 并保留会话；盘符恢复后，后续 GET、PUT 或 commit 会重新协调。`recoveryError` 仅在需要诊断时出现。
- janitor 每 15 分钟检查终态、确实过期的活跃会话，以及超过 24 小时且严格匹配 `.curated-import/upload_<16 lowercase hex>` 的孤立目录。每次清理都先写 `movie_import_upload_cleanup_audits`；它不会删除最终目标文件，也不会扫描或删除任意隐藏目录。

#### `GET /api/import/movies/uploads/{uploadId}`

用途：读取上传会话状态。

成功：`200 MovieImportUploadDTO`

#### `PUT /api/import/movies/uploads/{uploadId}/files/{fileId}/chunks/{chunkIndex}`

用途：上传一个文件分片。

Content-Type：`application/octet-stream`

Headers：

| Header | 必填 | 说明 |
| --- | --- | --- |
| `X-Curated-Offset` | 是 | 本分片写入文件的起始字节 offset |
| `X-Curated-Chunk-Size` | 否 | 本分片字节数；如果提供，body 实际大小必须一致 |

成功：`200 MovieImportUploadDTO`

行为：

- `chunkIndex` 必须是非负整数。
- 重复上传同一 `chunkIndex` 且 offset/size 一致时幂等返回当前状态。
- 重复上传同一 `chunkIndex` 但范围不同，返回 `409 COMMON_CONFLICT`。
- 不同 chunk index 的范围不得重叠。
- offset 越界或分片超过文件大小，返回 `400 COMMON_BAD_REQUEST`。
- 非空分片先写入指定范围并完成 `Sync` / `Close`，之后才在 SQLite 事务中记录 chunk 并更新文件与会话计数；如果进程在写盘后、记录前退出，客户端可重传同一范围。

#### `POST /api/import/movies/uploads/{uploadId}/commit`

用途：验证所有文件完整，提交 staging 文件到库根，并启动扫描。

提交前会对全部目标做不覆盖预检，并先把会话持久化为 `committing`。每个成功移动的文件都有 SQLite committed marker；后端重启后可继续未完成提交，也可识别“文件已移动但 marker 尚未写入”的中断窗口。无法安全协调的会话会标记为 `unrecoverable`，不会覆盖冲突文件。

成功：`202 TaskDTO`

错误：

- `400 COMMON_BAD_REQUEST`：上传不完整。
- `409 COMMON_CONFLICT`：会话状态不允许提交或目标文件已存在。

#### `DELETE /api/import/movies/uploads/{uploadId}`

用途：取消分片上传并删除 staging 目录。

成功：`204 No Content`

### 4.13 Scans / Tasks

#### `POST /api/scans`

用途：启动库扫描。

Body 可选：

```json
{
  "paths": ["D:\\Library"]
}
```

说明：

- `paths` 省略或空数组通常表示扫描全部配置库根。
- 如果已有扫描在运行，返回 `409 COMMON_CONFLICT`。

成功：`202 TaskDTO`

#### `GET /api/tasks/recent`

用途：列出最近结束的任务。

Query：

| 参数 | 说明 |
| --- | --- |
| `limit` | 默认 30 |

成功：`200 RecentTasksDTO`

```json
{
  "tasks": [
    {
      "taskId": "task-1",
      "type": "import.movies",
      "status": "completed",
      "createdAt": "2026-06-07T12:00:00Z",
      "startedAt": "2026-06-07T12:00:01Z",
      "finishedAt": "2026-06-07T12:01:00Z",
      "progress": 100,
      "message": "Movie import completed"
    }
  ]
}
```

#### `GET /api/tasks/{taskId}`

用途：读取单个任务状态。

成功：`200 TaskDTO`

错误：

- `404 COMMON_NOT_FOUND`

#### `GET /api/events`

用途：订阅后端事件流。当前事件流用于推送任务生命周期快照，前端用它驱动扫描、导入、刮削和目录监听 toast，同时保留 `/tasks/{taskId}` 与 `/tasks/recent` 轮询作为 fallback。

认证：受 PIN App Lock 保护，未解锁时返回 `423 AUTH_LOCKED`。

响应头：

| Header | 值 |
| --- | --- |
| `Content-Type` | `text/event-stream` |
| `Cache-Control` | `no-cache` |
| `Connection` | `keep-alive` |
| `X-Accel-Buffering` | `no` |

连接成功后先发送：

```text
event: hello
data: {"type":"hello"}
```

任务更新事件：

```text
event: task.updated
data: {"type":"task.updated","task":{"taskId":"task-1","type":"scan.library","status":"running","createdAt":"2026-06-28T12:00:00Z","progress":0.4}}
```

服务端还会周期性发送 comment heartbeat。客户端断开后服务端会自动取消订阅；事件投递是非阻塞的，慢客户端可能丢失中间快照，因此客户端仍应保留轮询补偿。

### 4.14 Curated Frames

#### `GET /api/curated-frames`

用途：分页查询精选帧元数据。

Query：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `q` | string | 搜索标题、番号、movieId、演员 JSON、标签 JSON、捕获时间、秒数 |
| `actor` | string | 精确匹配演员名 |
| `movieId` | string | 精确匹配影片 ID |
| `tag` | string | 精确匹配标签 |
| `limit` | number | 默认 50，最大 200 |
| `offset` | number | 默认 0 |

成功：`200 CuratedFramesListDTO`

```json
{
  "items": [
    {
      "id": "frame-1",
      "movieId": "movie-1",
      "title": "Example",
      "code": "ABC-001",
      "actors": ["Actor A"],
      "positionSec": 42.5,
      "capturedAt": "2026-06-07T12:00:00Z",
      "tags": ["favorite"]
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

#### `GET /api/curated-frames/stats`

用途：精选帧总数。

成功：

```json
{
  "total": 123
}
```

#### `GET /api/curated-frames/tags`

用途：精选帧标签 facet。

成功：`200 CuratedFrameFacetListDTO`

#### `GET /api/curated-frames/actors`

用途：精选帧演员 facet。

成功：`200 CuratedFrameFacetListDTO`

facet 返回：

```json
{
  "items": [
    {
      "name": "Actor A",
      "count": 12
    }
  ]
}
```

#### `POST /api/curated-frames`

用途：保存精选帧。

支持两种请求格式。

JSON Body：

```json
{
  "id": "frame-1",
  "movieId": "movie-1",
  "title": "Example",
  "code": "ABC-001",
  "actors": ["Actor A"],
  "positionSec": 42.5,
  "capturedAt": "2026-06-07T12:00:00Z",
  "tags": ["favorite"],
  "imageBase64": "iVBORw0KGgo..."
}
```

Multipart：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `metadata` | string | 上述 JSON 去掉 `imageBase64` 后序列化 |
| `image` | file | 图片 bytes |

约束：

- `id` 和 `movieId` 必填。
- `image` 不能为空，最大 12 MiB。
- `imageBase64` 是标准 base64，不带 `data:image/...;base64,` 前缀。
- `movieId` 必须存在。
- `id` 重复返回 `409 COMMON_CONFLICT`。

成功：`204 No Content`

#### `GET /api/curated-frames/{id}/image`

用途：获取精选帧原图。

成功：图片 bytes，`Cache-Control: private, max-age=3600`

#### `GET /api/curated-frames/{id}/thumbnail`

用途：获取精选帧缩略图。

成功：图片 bytes，`Cache-Control: private, max-age=3600`

#### `PATCH /api/curated-frames/{id}/tags`

用途：替换精选帧标签。

Body：

```json
{
  "tags": ["tag-a", "tag-b"]
}
```

`tags: null` 等价于空数组。

成功：`204 No Content`

#### `DELETE /api/curated-frames/{id}`

用途：删除精选帧。

成功：`204 No Content`

#### `POST /api/curated-frames/export`

用途：导出 1 到 20 张精选帧。

Body：

```json
{
  "ids": ["frame-1", "frame-2"],
  "actorName": "Actor A",
  "format": "jpg"
}
```

字段：

| 字段 | 说明 |
| --- | --- |
| `ids` | 必填；会 trim、去重；最多 20 |
| `actorName` | 可选；用于导出文件名中的演员上下文，必须属于每一帧演员列表 |
| `format` | `jpg` 默认；也支持 `jpeg` alias、`webp`、`png` |

成功响应：

| 数量 | Content-Type | 说明 |
| --- | --- | --- |
| 1 张 | `image/jpeg` / `image/png` / `image/webp` | 单图下载 |
| 多张 | `application/zip` | ZIP 包 |

响应会设置 `Content-Disposition`，客户端应从 header 解析文件名。

错误：

- `400 COMMON_BAD_REQUEST`：缺少 ids、超过 20、format 不支持。
- `400 CURATED_EXPORT_ACTOR_MISMATCH`：`actorName` 不属于某一帧。
- `404 COMMON_NOT_FOUND`：帧不存在或无图。

导出图片会写入 Curated 元数据，包括：

- `title`
- `code`
- `actors`
- `positionSec`
- `capturedAt`
- `frameId`
- `movieId`
- `tags`
- `schemaVersion`
- `exportedAt`
- `appName`
- `appVersion`

### 4.15 Providers / Proxy

#### `POST /api/providers/ping`

用途：检测单个元数据 provider 健康状态。

Body：

```json
{
  "name": "provider-name"
}
```

成功：`200 ProviderHealthDTO`

```json
{
  "name": "provider-name",
  "status": "ok",
  "latencyMs": 123,
  "message": "",
  "errorCategory": "",
  "cooldownUntil": "",
  "consecutiveFailures": 0,
  "avgLatencyMs": 100
}
```

错误：

- `400 COMMON_BAD_REQUEST`：body 不合法或 name 为空。
- `404 PROVIDER_NOT_FOUND`

`status` 可用值：`ok`、`degraded`、`fail`。

`errorCategory` 常见值：

| 值 | 含义 |
| --- | --- |
| `dns_failure` | DNS 失败 |
| `connect_timeout` | 连接超时 |
| `tls_failure` | TLS 问题 |
| `region_restricted` | 地区限制 |
| `hotlink_denied` | 防盗链 / Referer 问题 |
| `provider_empty_result` | provider 无结果 |
| `provider_invalid_content` | 内容不符合预期 |

#### `POST /api/providers/ping-all`

用途：检测所有 provider。

成功：`200 PingAllProvidersResponse`

```json
{
  "providers": [],
  "total": 0,
  "ok": 0,
  "fail": 0
}
```

#### `POST /api/proxy/ping-javbus`

用途：用草稿代理配置或已保存代理配置请求 `https://www.javbus.com/`。

Body 可选：

```json
{
  "proxy": {
    "enabled": true,
    "url": "http://127.0.0.1:7890",
    "username": "",
    "password": ""
  }
}
```

成功：`200 ProxyJavBusPingResponse`

```json
{
  "ok": true,
  "latencyMs": 300,
  "httpStatus": 200,
  "message": ""
}
```

说明：

- 如果省略 `proxy`，使用当前持久化代理设置。
- 出站请求超时约 5 秒。
- 远端连接失败通常仍返回 `200`，但 body 中 `ok=false`。
- `proxy.enabled=true` 且 URL 为空时返回 `400 COMMON_BAD_REQUEST`。

#### `POST /api/proxy/ping-google`

用途：同上，但目标为 `https://www.google.com/`。

成功：`200 ProxyJavBusPingResponse`

### 4.16 Maintenance Backups

以下三个端点都需要已解锁的 PIN 会话，只接受运行 Curated 后端那台机器上的绝对路径。请求 body 上限为 64 KiB，未知字段、尾随 JSON、相对路径和空路径都会以 `400 BACKUP_INVALID_REQUEST` 拒绝。它们不会替换正在使用的数据库，也没有在线 restore 端点。

#### `POST /api/maintenance/backups`

用途：从运行中的 SQLite 创建一致 `.curated-backup` 包。目标文件已存在时绝不覆盖。

Body：

```json
{
  "destinationPath": "D:\\Backups\\curated-20260720.curated-backup"
}
```

成功：`201 BackupManifestDTO`

```json
{
  "format": "curated-backup",
  "formatVersion": 1,
  "createdAt": "2026-07-20T03:00:00Z",
  "appVersion": "1.4.11",
  "appChannel": "release",
  "scope": {
    "databaseIncluded": true,
    "libraryConfigIncluded": true,
    "userAssetsIncluded": false,
    "mediaFilesIncluded": false
  },
  "schemaMigrations": ["0001_init.sql"],
  "files": [
    {
      "kind": "database",
      "path": "database/curated.db",
      "sizeBytes": 1048576,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    }
  ]
}
```

错误：

- `409 BACKUP_CONFLICT`：目标包已经存在。
- `500 BACKUP_CREATE_FAILED`：快照、完整性检查、打包或落盘失败。
- `503 BACKUP_CREATE_FAILED`：备份服务未装配。

#### `POST /api/maintenance/backups/verify`

用途：验证已有包的 manifest、声明文件、大小、SHA-256、SQLite `quick_check`、`foreign_key_check` 与 migration 一致性。

Body：

```json
{
  "backupPath": "D:\\Backups\\curated-20260720.curated-backup"
}
```

成功：`200 BackupVerificationDTO`。包内容无效但仍可读取时返回 `200` 和 `valid=false`，并在 `errors` 中列出完整诊断；文件无法打开或执行验证时返回 `400 BACKUP_VERIFY_FAILED`。

```json
{
  "valid": true,
  "checkedAt": "2026-07-20T03:01:00Z",
  "manifest": {
    "format": "curated-backup",
    "formatVersion": 1,
    "createdAt": "2026-07-20T03:00:00Z",
    "appVersion": "1.4.11",
    "appChannel": "release",
    "scope": {
      "databaseIncluded": true,
      "libraryConfigIncluded": true,
      "userAssetsIncluded": false,
      "mediaFilesIncluded": false
    },
    "schemaMigrations": ["0001_init.sql"],
    "files": [
      {
        "kind": "database",
        "path": "database/curated.db",
        "sizeBytes": 1048576,
        "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      }
    ]
  },
  "databaseIntegrity": {
    "quickCheck": "ok",
    "foreignKeyViolations": 0
  },
  "errors": [],
  "warnings": []
}
```

#### `POST /api/maintenance/backups/preflight`

用途：在不写目标数据库的前提下，验证备份并检查当前目标数据库/配置、未来未知 migration、所需空间和可用空间。

Body 与 verify 相同。

成功：`200 BackupRestorePreflightDTO`。不满足恢复条件时仍返回结构化 `canRestore=false`、`errors` 和 `warnings`；无法读取或执行预检时返回 `400 BACKUP_PREFLIGHT_FAILED`。

```json
{
  "canRestore": true,
  "checkedAt": "2026-07-20T03:02:00Z",
  "verification": {
    "valid": true,
    "checkedAt": "2026-07-20T03:02:00Z",
    "manifest": {
      "format": "curated-backup",
      "formatVersion": 1,
      "createdAt": "2026-07-20T03:00:00Z",
      "appVersion": "1.4.11",
      "appChannel": "release",
      "scope": {
        "databaseIncluded": true,
        "libraryConfigIncluded": true,
        "userAssetsIncluded": false,
        "mediaFilesIncluded": false
      },
      "schemaMigrations": ["0001_init.sql"],
      "files": [
        {
          "kind": "database",
          "path": "database/curated.db",
          "sizeBytes": 1048576,
          "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        }
      ]
    },
    "databaseIntegrity": {
      "quickCheck": "ok",
      "foreignKeyViolations": 0
    },
    "errors": [],
    "warnings": []
  },
  "targetDatabase": "D:\\Curated\\curated.db",
  "targetDatabaseExists": true,
  "targetConfig": "D:\\Curated\\library-config.cfg",
  "targetConfigExists": true,
  "requiredBytes": 2097152,
  "availableBytes": 10737418240,
  "availableBytesKnown": true,
  "unsupportedMigrations": [],
  "errors": [],
  "warnings": ["backup does not include media source files"]
}
```

真正恢复必须完全退出 Curated 后，通过 `curated -maintenance backup-restore ... -confirm-restore` 离线执行；成功恢复会保留 `.pre-restore-*` 数据库和配置回滚副本。

### 4.17 Library Health and Repair

以下端点都受现有 PIN 中间件保护。健康扫描默认只读；扫描与修复严格分离，不会在诊断阶段删除、覆盖、重新刮削或重扫。Settings → Maintenance 在 Web API 模式下提供扫描、JSON 导出、元数据修复确认和限定清理确认；Mock 模式明确禁用这些后端能力。

#### `POST /api/library/health/scan`

用途：生成当前时点的资料库健康快照。可选 query `findingLimit=1..5000` 控制返回的明细条数；完整分类计数不会因明细截断而丢失。

成功：`200 LibraryHealthReportDTO`

```json
{
  "scannedAt": "2026-07-20T05:45:00Z",
  "status": "attention",
  "database": {
    "quickCheckOk": true,
    "quickCheckMessages": ["ok"],
    "foreignKeyOk": true,
    "foreignKeyCount": 0
  },
  "storageStatuses": [],
  "summary": {
    "totalFindings": 1,
    "criticalFindings": 0,
    "warningFindings": 1,
    "infoFindings": 0,
    "skippedOfflineFiles": 0,
    "categoryCounts": { "metadata_failed": 1 }
  },
  "findings": [
    {
      "id": "health_0123456789abcdef01234567",
      "category": "metadata_failed",
      "severity": "warning",
      "entityType": "movie",
      "entityId": "abc-001",
      "label": "ABC-001",
      "path": "D:\\Media\\ABC-001.mp4",
      "message": "metadata provider timed out",
      "repairActions": ["rescrape_metadata", "export_diagnostics"]
    }
  ],
  "truncated": false
}
```

稳定类别包括 `database_integrity`、`foreign_key_violation`、`storage_unavailable`、`source_missing`、`source_unreadable`、`source_empty`、`asset_missing`、`asset_download_failed`、`duplicate_code`、`duplicate_source`、`orphan_user_state`、`metadata_missing`、`metadata_failed`、`movie_poster_missing`、`actor_avatar_missing` 与 `import_staging_residue`。finding ID 由类别、实体和路径稳定生成。

离线、卷不匹配、路径缺失或权限异常的库根只报告根级问题，并跳过其下影片、资源和演员头像的逐文件检查；`skippedOfflineFiles` 记录跳过量，避免把断开的外置盘误报为文件已删除。`.curated-import` 只检查第一层；只有未登记、非 symlink 且严格命名为 `upload_<16 lowercase hex>` 的目录才会得到 `cleanup_import_staging` 建议。

#### `POST /api/library/health/repairs`

用途：对当前仍存在的 `metadata_missing` / `metadata_failed` finding 启动显式确认、持久化且有界的元数据修复队列。后端每次最多 100 项，Settings 每批最多 25 项。

Body：

```json
{
  "action": "rescrape_metadata",
  "categories": ["metadata_failed"],
  "findingIds": ["health_0123456789abcdef01234567"],
  "limit": 25,
  "confirm": true
}
```

成功：`202 LibraryHealthRepairDTO`。父任务类型为 `library.health.repair`；每个影片使用独立 `scrape.movie` 子任务，`GET /api/library/health/repairs/{repairId}` 可持续读取持久化的逐项状态、child task ID、错误和时间戳。

```json
{
  "repairId": "repair-1",
  "taskId": "library.health.repair-1",
  "action": "rescrape_metadata",
  "categories": ["metadata_failed"],
  "status": "running",
  "totalItems": 1,
  "completedItems": 0,
  "succeededItems": 0,
  "failedItems": 0,
  "createdAt": "2026-07-20T05:46:00Z",
  "items": [
    {
      "ordinal": 0,
      "findingId": "health_0123456789abcdef01234567",
      "category": "metadata_failed",
      "movieId": "abc-001",
      "label": "ABC-001",
      "status": "pending"
    }
  ]
}
```

后端在入队前重新扫描并只接受仍可执行的精确 finding；健康影片不能通过任意 movie ID 加入队列。后端重启时，未完成项会被明确持久化为 `cancelled` / `HEALTH_REPAIR_INTERRUPTED`，不会永久停在 `running`。最新 scrape attempt 以影片为键，旧任务的晚到结果不能覆盖更新任务的结果。

#### `GET /api/library/health/repairs/{repairId}`

用途：读取一个持久化修复队列及其全部逐项结果。

成功：`200 LibraryHealthRepairDTO`。不存在时返回 `404 COMMON_NOT_FOUND`。

#### `POST /api/library/health/actions`

用途：对当前健康报告中的精确 finding 执行确认式清理。目前只支持 `cleanup_orphan_state` 和 `cleanup_import_staging`。

Body：

```json
{
  "action": "cleanup_import_staging",
  "findingIds": ["health_abcdef0123456789abcdef01"],
  "confirm": true
}
```

成功：`202 TaskDTO`，任务类型为 `library.health.cleanup`，逐项结果写入 task metadata。执行前会重新扫描；任一 finding 已过期、类别不匹配或不再可执行时，整个请求以 `409 HEALTH_REPAIR_NO_FINDINGS` 拒绝。

孤儿状态清理只允许 `playback_progress`、`library_movie_comments`、`library_played_movies`、`playback_daily_watch_time` 白名单，并在父影片仍不存在时于同一 SQLite 事务写入 `library_health_cleanup_audits`。暂存清理要求候选绝对路径精确位于在线配置根的 `.curated-import/upload_<16 lowercase hex>`，根与候选都不得是 symlink，且 SQLite 中仍无 upload session；删除前写入 `movie_import_upload_cleanup_audits`。最终影片文件永远不会成为候选。

确认式端点的通用错误：

- `409 HEALTH_REPAIR_CONFIRMATION_REQUIRED`：`confirm` 不是 `true`。
- `409 HEALTH_REPAIR_NO_FINDINGS`：没有仍然匹配的当前 finding，或 cleanup finding 已过期/不可执行。
- `500 HEALTH_REPAIR_PERSIST_FAILED`：修复队列、任务或审计状态无法持久化。
- `HEALTH_CLEANUP_FAILED`：清理 task 的终态错误码（`failed` / `partial_failed`）；逐项结果保留在 task metadata，而不是丢失审计上下文。

### 4.18 Saved Views

Saved Views 使用版本化白名单筛选快照。Web API 模式写入当前 SQLite 的 `library_saved_views`；Mock 模式写入独立的 `localStorage` 键 `curated-library-saved-views-v1`。任何端点都不会保存或接受 `selected`、`from`、`browse`、`back`、`autoplay`、`t` 等瞬态导航字段。

#### `GET /api/library/saved-views`

用途：按 `sortOrder ASC` 返回全部保存视图。空列表稳定返回 `{"items":[]}`。

#### `POST /api/library/saved-views`

Body：

```json
{
  "name": "未看 4K",
  "filters": {
    "schemaVersion": 1,
    "mode": "library",
    "tab": "all",
    "playState": "unwatched",
    "userRating": 5,
    "resolution": "4k",
    "addedWithinDays": 365
  }
}
```

成功：`201 SavedViewDTO`。名称 trim 后为 1～40 个 Unicode 字符，忽略大小写及首尾空白后必须唯一；每个库最多 50 个视图。筛选文本最长 200 字符，`addedWithinDays` 为 1～3650。服务端重新规范化所有字段，`2160p` / `UHD` 等保存为 `4k`。

#### `PATCH /api/library/saved-views/{savedViewId}`

可部分更新 `name` 和/或完整 `filters`；成功返回 `200 SavedViewDTO`。空 patch、未知字段、未知 schema、无效枚举或越界值返回 `400 SAVED_VIEW_INVALID`。

#### `DELETE /api/library/saved-views/{savedViewId}`

成功返回 `204` 并压紧剩余 `sortOrder`。只删除视图定义，不删除或修改影片、播放记录、评分、收藏或标签。

#### `PUT /api/library/saved-views/order`

Body：`{"ids":["view_b","view_a"]}`。必须把当前全部 ID 各提交一次；缺失、重复或未知 ID 返回 `400 SAVED_VIEW_INVALID`。成功返回重排后的 `200 SavedViewsDTO`，更新在一个 SQLite 事务内完成。

通用错误：

- `409 SAVED_VIEW_NAME_CONFLICT`：规范化名称重复。
- `409 SAVED_VIEW_LIMIT_REACHED`：已达到 50 个视图。
- `404 COMMON_NOT_FOUND`：目标视图不存在。

应用视图时前端从空 query 重建 canonical route；相对时间窗口在每次应用/计算时重新解释，不把一次性的绝对“当前时间”写入定义。

### 4.19 Personal Insights

Personal Insights 只返回有界聚合，不返回逐日或逐次完整播放历史。Web API 模式由 SQLite 在后端聚合；Mock 模式使用相同指标口径在服务适配器内部读取版本化本地观看时长和播放进度，原始行不会通过 `LibraryService` 暴露给页面。

时间范围固定为 `30d`、`90d`、`365d`、`all`。客户端应传 IANA `timezone`（例如 `Asia/Shanghai`）；省略时后端使用 `UTC`。响应总是回显实际包含首尾日的 `from`、`to`、`timezone`、生成时间 `generatedAt`，并在存在历史数据时返回 `dataSince`。后端二进制内嵌 IANA tzdata，避免 Windows 打包态依赖系统时区数据库。

#### `GET /api/insights/overview`

Query：

| 参数 | 说明 |
| --- | --- |
| `range` | 必填：`30d`、`90d`、`365d` 或 `all` |
| `timezone` | IANA 时区；省略使用 `UTC`，最长 128 字符 |

成功：`200 PersonalInsightsOverviewDTO`

```json
{
  "range": "30d",
  "from": "2026-06-23",
  "to": "2026-07-22",
  "timezone": "Asia/Shanghai",
  "generatedAt": "2026-07-21T16:30:00Z",
  "dataSince": "2026-04-01",
  "watchedSeconds": 18240,
  "startedMovies": 12,
  "completedMovies": 7,
  "completionRate": 0.5833333333,
  "completionThreshold": 0.9,
  "ratedMovies": 5,
  "averageUserRating": 4.2
}
```

指标口径：

- `watchedSeconds`：范围内 `playback_daily_watch_time.watched_sec` 按影片和日期累加后的总和。
- `startedMovies`：范围内具有正观看时长、且当前仍有影片行的不同 `movieId` 数。
- `completedMovies`：上述 started 影片中，**当前保存进度**满足 `durationSec > 0` 且 `positionSec >= durationSec × 0.9` 的数量。当前 schema 没有历史 completion event，因此该字段不声称“完成动作发生在所选范围内”。
- `completionRate`：`completedMovies / startedMovies`；分母为 0 时必须为 JSON `null`，不能返回误导性的 `0`。
- `ratedMovies` / `averageUserRating`：只统计范围内 started 影片当前存在的本地用户评分；不声称评分动作发生在该范围内。无评分分母时平均分为 `null`。

#### `GET /api/insights/breakdown`

Query：

| 参数 | 说明 |
| --- | --- |
| `range` | 与 overview 相同 |
| `timezone` | 与 overview 相同 |
| `dimension` | 必填：`actor`、`studio` 或 `tag` |
| `limit` | 可选，默认 10，范围 1～25 |

成功：`200 PersonalInsightsBreakdownDTO`

```json
{
  "range": "30d",
  "dimension": "actor",
  "from": "2026-06-23",
  "to": "2026-07-22",
  "timezone": "Asia/Shanghai",
  "generatedAt": "2026-07-21T16:30:00Z",
  "dataSince": "2026-04-01",
  "totalWatchedSeconds": 18240,
  "attribution": "full-per-entity",
  "items": [
    {
      "name": "Actor A",
      "watchedSeconds": 7200,
      "movieCount": 4,
      "shareOfTotal": 0.3947368421
    }
  ],
  "limit": 10
}
```

归因固定为 `full-per-entity`：多演员、多标签影片会把该影片完整观看时长分别计入每个关联实体；同一影片的同名 metadata/user tag 先去重。单项 `shareOfTotal = item.watchedSeconds / totalWatchedSeconds`，跨实体合计可能超过 100%，不能解释为互斥饼图。排序固定为观看时长降序、影片数降序、名称忽略大小写升序、名称升序；actor 使用 canonical actor，旧 alias 不产生独立排行。

稳定参数错误均返回 `400`：

| Code | 含义 |
| --- | --- |
| `INSIGHTS_INVALID_RANGE` | 未知或缺失的范围 |
| `INSIGHTS_INVALID_TIMEZONE` | 时区为空白规则之外、过长或无法加载 |
| `INSIGHTS_INVALID_DIMENSION` | 未知或缺失的维度 |
| `INSIGHTS_INVALID_LIMIT` | `limit` 不是整数或不在 1～25 |

### 4.20 Offline Path Migration CLI

路径迁移不是 HTTP API，也不会在运行中的 Settings 页面直接执行。盘符、挂载点或库根变化时，必须完全退出 Curated，并从 `backend/` 运行维护 CLI；自定义数据库配置需同时传 `-config <path>`。plan 获取与正常运行时相同的 `<databasePath>.runtime.lock`，读取数据库但不执行 schema migration 或数据写入：

```powershell
go run ./cmd/curated `
  -maintenance path-migrate-plan `
  -path-from D:\Media `
  -path-to E:\Media
```

apply 要求 plan 可应用、一个尚不存在的 `.curated-backup` 目标和显式确认：

```powershell
go run ./cmd/curated `
  -maintenance path-migrate-apply `
  -path-from D:\Media `
  -path-to E:\Media `
  -backup-path D:\Backups\before-path-migration.curated-backup `
  -confirm-path-migration
```

行为与安全边界：

- 源/目标必须是绝对 Windows drive、UNC 或 Unix 路径；拒绝相对路径、`..`、等价前缀以及嵌套在源前缀下的目标。
- Windows 比较忽略盘符/路径段大小写并统一 `/`、`\`；Unix 保持大小写敏感；匹配按路径段边界进行，`D:\Media` 不匹配 `D:\Media2`。
- 支持 Windows→Unix 等跨平台映射。当前操作系统无法检查目标时默认为 `unchecked` 并阻止 apply；只有显式 `-allow-missing-paths` 才放行 `missing` / `unchecked`。`wrong-type`、I/O error 和唯一性冲突始终阻止 apply。
- 只处理白名单 `library_paths.path`、`movies.location`、`scan_items.path`、`media_assets.local_path`、`actors.avatar_local_path`、`library_path_storage_bindings.root_path`、`app_update_status.downloaded_file_path`；不扫描或替换自由文本、URL 与既有审计 JSON。
- `library_path_storage_bindings` 命中时删除旧 binding，不把旧卷身份迁到新路径；Curated 下次启动时重新探测并绑定。
- apply 先通过 `VACUUM INTO` 创建且验证迁移前备份，再在一个 SQLite 事务中重新 plan、更新白名单字段、执行 `quick_check` / `foreign_key_check` 并写入 `path_migration_audits`。任何更新、完整性检查或审计写入失败都会回滚整个事务。

plan 只要成功读取数据库，就会在 stdout 写出结构化 JSON；`plan.columns` 按白名单列返回 `affectedRows`、`emptyRows`、`outsidePrefixRows`、`invalidStoredPaths`、`missingTargets`、`uncheckedTargets`、`targetErrors`、`conflicts` 和最多 5 个样例，顶层还返回汇总、`errors`、`warnings` 与 `canApply`。`canApply=false` 时 CLI 写出 plan 后以非零状态退出。apply 的事务内重新 plan 若被阻止，也会输出该结构化结果和已创建的备份路径；参数无效、未确认、锁冲突或 I/O 失败等无法形成有效 plan 的错误写到 stderr。apply 成功额外返回经过验证的备份 manifest/verification、`appliedRows`、`auditId`、`appliedAt` 和完整 apply plan。

## 5. DTO 速查

本节列出衍生客户端最常用 DTO。完整字段以 `backend/internal/contracts/contracts.go` 和 `src/api/types.ts` 为准。

### 5.1 `AppError`

```ts
interface AppError {
  code: string
  message: string
  retryable: boolean
  details?: Record<string, unknown>
}
```

### 5.2 `MovieListItemDTO`

```ts
interface MovieListItemDTO {
  id: string
  title: string
  code: string
  studio: string
  actors: string[]
  tags: string[]
  userTags?: string[]
  runtimeMinutes: number
  rating: number
  isFavorite: boolean
  addedAt: string
  location: string
  resolution: string
  year: number
  releaseDate?: string
  coverUrl?: string
  thumbUrl?: string
  trashedAt?: string
}
```

### 5.3 `MovieDetailDTO`

```ts
interface MovieDetailDTO extends MovieListItemDTO {
  summary: string
  previewImages?: string[]
  previewVideoUrl?: string
  metadataRating: number
  userRating?: number | null
  actorAvatarUrls?: Record<string, string>
}
```

### 5.4 `SettingsDTO`

```ts
interface SettingsDTO {
  libraryPaths: LibraryPathDTO[]
  defaultImportLibraryPathId?: string
  player: PlayerSettingsDTO
  organizeLibrary: boolean
  autoLibraryWatch: boolean
  autoActorProfileScrape: boolean
  autoDownloadUpdates: boolean
  launchAtLogin: boolean
  launchAtLoginSupported: boolean
  curatedFrameExportFormat: "jpg" | "webp" | "png"
  metadataMovieProvider: string
  metadataMovieProviders: string[]
  metadataMovieProviderChain: string[]
  metadataMovieScrapeMode?: "auto" | "specified" | "chain"
  metadataMovieStrategy?: "auto-global" | "auto-cn-friendly" | "custom-chain" | "specified"
  proxy: ProxySettingsDTO
  backendLog: BackendLogSettingsDTO
}
```

### 5.5 `PlaybackDescriptorDTO`

```ts
type PlaybackMode = "direct" | "hls" | "native"

interface PlaybackDescriptorDTO {
  movieId: string
  mode: PlaybackMode
  sessionId?: string
  sessionKind?: string
  url: string
  mimeType?: string
  fileName?: string
  transcodeProfile?: string
  durationSec?: number
  startPositionSec?: number
  resumePositionSec?: number
  canDirectPlay: boolean
  reason?: string
  reasonCode?: string
  reasonMessage?: string
  sourceContainer?: string
  sourceVideoCodec?: string
  sourceAudioCodec?: string
  audioTracks?: { id: string; label: string; default: boolean }[]
  subtitleTracks?: { id: string; label: string; kind?: string; default: boolean }[]
}
```

### 5.6 `TaskDTO`

```ts
interface TaskDTO {
  taskId: string
  type: string
  status: "pending" | "running" | "completed" | "partial_failed" | "failed" | "cancelled"
  createdAt: string
  startedAt?: string
  finishedAt?: string
  progress: number
  message?: string
  errorCode?: string
  errorCategory?: string
  errorMessage?: string
  provider?: string
  metadata?: Record<string, unknown>
}
```

### 5.7 `TaskEventDTO`

```ts
interface TaskEventDTO {
  type: "task.updated"
  task: TaskDTO
}
```

### 5.8 `MovieImportUploadDTO`

```ts
interface MovieImportUploadDTO {
  uploadId: string
  targetPath: string
  chunkSize: number
  bytesReceived: number
  totalBytes: number
  state: "uploading" | "committed" | "aborted" | string
  files: Array<{
    fileId: string
    relativePath: string
    size: number
    bytesReceived: number
    complete: boolean
  }>
  task: TaskDTO
}
```

### 5.9 `CuratedFrameItemDTO`

```ts
interface CuratedFrameItemDTO {
  id: string
  movieId: string
  title: string
  code: string
  actors: string[]
  positionSec: number
  capturedAt: string
  tags: string[]
}
```

### 5.10 `ActorProfileDTO`

```ts
interface ActorProfileDTO {
  name: string
  avatarUrl?: string
  avatarRemoteUrl?: string
  avatarLocalUrl?: string
  hasLocalAvatar?: boolean
  summary?: string
  homepage?: string
  provider?: string
  providerActorId?: string
  height?: number
  birthday?: string
  profileUpdatedAt?: string
  userTags?: string[]
  externalLinks?: string[]
  aliases?: string[]
}
```

### 5.11 Actor merge DTOs

```ts
interface ActorMergePreviewDTO {
  previewToken: string
  source: { id: number; name: string; aliases: string[] }
  target: { id: number; name: string; aliases: string[] }
  movies: ActorMergeAssociationSummaryDTO
  userTags: ActorMergeValuesSummaryDTO
  externalLinks: ActorMergeValuesSummaryDTO
  recommendationFeedback: ActorMergeAssociationSummaryDTO
  curatedFramesAffected: number
  aliasesToMove: string[]
  profileFields: Array<{
    field: string
    sourceValue: string
    targetValue: string
    defaultSelection: "source" | "target"
    conflict: boolean
  }>
  canApply: boolean
  blockingReasons: Array<{ code: string; message: string }>
  requiredDecisions: string[]
}

interface ActorMergeAssociationSummaryDTO {
  sourceCount: number
  targetCount: number
  duplicateCount: number
  resultCount: number
}

interface ActorMergeValuesSummaryDTO {
  source: string[]
  target: string[]
  result: string[]
}
```

## 6. 全路由清单

本清单与 `backend/internal/server/server.go` 的 `Routes()` 对齐。

| Method | Path | 响应 |
| --- | --- | --- |
| `GET` | `/api/health` | `HealthDTO` |
| `GET` | `/api/auth/status` | `AuthStatusDTO` |
| `POST` | `/api/auth/setup-pin` | `AuthStatusDTO` |
| `POST` | `/api/auth/unlock` | `AuthStatusDTO` |
| `POST` | `/api/auth/change-pin` | `AuthStatusDTO` |
| `POST` | `/api/auth/lock` | `AuthStatusDTO` |
| `PATCH` | `/api/auth/settings` | `AuthStatusDTO` |
| `GET` | `/api/connected-clients` | `ConnectedClientsDTO` |
| `GET` | `/api/dev/performance` | `DevPerformanceSummaryDTO` |
| `GET` | `/api/app-update/status` | `AppUpdateStatusDTO` |
| `POST` | `/api/app-update/check` | `AppUpdateStatusDTO` |
| `POST` | `/api/app-update/download` | `AppUpdateStatusDTO` |
| `POST` | `/api/app-update/install` | `AppUpdateStatusDTO` |
| `DELETE` | `/api/app-update/downloaded-installer` | `AppUpdateStatusDTO` |
| `POST` | `/api/maintenance/backups` | `BackupManifestDTO` |
| `POST` | `/api/maintenance/backups/verify` | `BackupVerificationDTO` |
| `POST` | `/api/maintenance/backups/preflight` | `BackupRestorePreflightDTO` |
| `POST` | `/api/library/health/scan` | `LibraryHealthReportDTO` |
| `POST` | `/api/library/health/repairs` | `LibraryHealthRepairDTO` |
| `GET` | `/api/library/health/repairs/{repairId}` | `LibraryHealthRepairDTO` |
| `POST` | `/api/library/health/actions` | `TaskDTO` |
| `GET` | `/api/homepage/recommendations` | `HomepageDailyRecommendationsDTO` |
| `POST` | `/api/homepage/recommendations/refresh` | `HomepageDailyRecommendationsDTO` |
| `GET` | `/api/homepage/recommendations/feedback` | `RecommendationFeedbackListDTO` |
| `POST` | `/api/homepage/recommendations/feedback` | `RecommendationFeedbackDTO` |
| `DELETE` | `/api/homepage/recommendations/feedback/{feedbackId}` | `204` |
| `GET` | `/api/library/played-movies` | `PlayedMoviesListDTO` |
| `POST` | `/api/library/played-movies/{movieId}` | `204` |
| `GET` | `/api/library/saved-views` | `SavedViewsDTO` |
| `POST` | `/api/library/saved-views` | `SavedViewDTO` |
| `PUT` | `/api/library/saved-views/order` | `SavedViewsDTO` |
| `PATCH` | `/api/library/saved-views/{savedViewId}` | `SavedViewDTO` |
| `DELETE` | `/api/library/saved-views/{savedViewId}` | `204` |
| `GET` | `/api/library/movies` | `MoviesPageDTO` |
| `GET` | `/api/library/actors` | `ActorsListDTO` |
| `GET` | `/api/library/actors/profile` | `ActorProfileDTO` |
| `GET` | `/api/library/actors/{name}/asset/avatar` | image |
| `POST` | `/api/library/actors/scrape` | `TaskDTO` |
| `PATCH` | `/api/library/actors/tags` | `ActorListItemDTO` |
| `PATCH` | `/api/library/actors/external-links` | `ActorProfileDTO` |
| `POST` | `/api/library/actors/merge-preview` | `ActorMergePreviewDTO` |
| `POST` | `/api/library/actors/merge` | `ActorMergeAuditDTO` |
| `GET` | `/api/library/actors/merge-audits` | `ActorMergeAuditListDTO` |
| `GET` | `/api/insights/overview` | `PersonalInsightsOverviewDTO` |
| `GET` | `/api/insights/breakdown` | `PersonalInsightsBreakdownDTO` |
| `GET` | `/api/library/movies/{movieId}/asset/preview/{index}` | image |
| `GET` | `/api/library/movies/{movieId}/asset/{kind}` | image |
| `GET` | `/api/library/movies/{movieId}/playback` | `PlaybackDescriptorDTO` |
| `POST` | `/api/library/movies/{movieId}/playback-session` | `PlaybackDescriptorDTO` |
| `POST` | `/api/library/movies/{movieId}/native-play` | `NativePlaybackLaunchDTO` |
| `GET` | `/api/library/movies/{movieId}/stream` | video |
| `GET` | `/api/playback/sessions/recent` | `PlaybackSessionListDTO` |
| `GET` | `/api/playback/sessions/{sessionId}` | `PlaybackSessionStatusDTO` |
| `GET` | `/api/playback/sessions/{sessionId}/hls/{file}` | HLS file |
| `DELETE` | `/api/playback/sessions/{sessionId}` | `204` |
| `POST` | `/api/library/movies/{movieId}/reveal` | `204` |
| `GET` | `/api/library/movies/{movieId}/comment` | `MovieCommentDTO` |
| `PUT` | `/api/library/movies/{movieId}/comment` | `MovieCommentDTO` |
| `GET` | `/api/library/movies/{movieId}` | `MovieDetailDTO` |
| `PATCH` | `/api/library/movies/{movieId}` | `MovieDetailDTO` |
| `POST` | `/api/library/movies/{movieId}/restore` | `204` |
| `POST` | `/api/library/movies/{movieId}/scrape` | `TaskDTO` |
| `POST` | `/api/library/metadata-scrape` | `MetadataRefreshQueuedDTO` |
| `DELETE` | `/api/library/movies/{movieId}` | `204` |
| `GET` | `/api/settings` | `SettingsDTO` |
| `PATCH` | `/api/settings` | `SettingsDTO` |
| `POST` | `/api/import/movies` | `TaskDTO` |
| `POST` | `/api/import/movies/uploads` | `MovieImportUploadDTO` |
| `GET` | `/api/import/movies/uploads/{uploadId}` | `MovieImportUploadDTO` |
| `DELETE` | `/api/import/movies/uploads/{uploadId}` | `204` |
| `PUT` | `/api/import/movies/uploads/{uploadId}/files/{fileId}/chunks/{chunkIndex}` | `MovieImportUploadDTO` |
| `POST` | `/api/import/movies/uploads/{uploadId}/commit` | `TaskDTO` |
| `POST` | `/api/library/paths` | `AddLibraryPathResponse` |
| `GET` | `/api/library/paths/storage-status` | `LibraryPathStorageStatusListDTO` |
| `POST` | `/api/library/paths/storage-status/check` | `LibraryPathStorageStatusListDTO` |
| `POST` | `/api/library/paths/{id}/reveal` | `204` |
| `POST` | `/api/library/paths/{id}/storage-binding/rebind` | `LibraryPathStorageStatusDTO` |
| `PATCH` | `/api/library/paths/{id}` | `LibraryPathDTO` |
| `DELETE` | `/api/library/paths/{id}` | `204` |
| `POST` | `/api/scans` | `TaskDTO` |
| `GET` | `/api/events` | SSE `TaskEventDTO` stream |
| `GET` | `/api/tasks/recent` | `RecentTasksDTO` |
| `GET` | `/api/tasks/{taskId}` | `TaskDTO` |
| `GET` | `/api/playback/progress` | `PlaybackProgressListDTO` |
| `PUT` | `/api/playback/progress/{movieId}` | `204` |
| `DELETE` | `/api/playback/progress/{movieId}` | `204` |
| `GET` | `/api/playback/watch-time/daily` | `PlaybackWatchTimeDailyListDTO` |
| `POST` | `/api/playback/watch-time/daily` | `204` |
| `GET` | `/api/curated-frames` | `CuratedFramesListDTO` |
| `GET` | `/api/curated-frames/stats` | `CuratedFrameStatsDTO` |
| `GET` | `/api/curated-frames/tags` | `CuratedFrameFacetListDTO` |
| `GET` | `/api/curated-frames/actors` | `CuratedFrameFacetListDTO` |
| `POST` | `/api/curated-frames` | `204` |
| `GET` | `/api/curated-frames/{id}/thumbnail` | image |
| `GET` | `/api/curated-frames/{id}/image` | image |
| `PATCH` | `/api/curated-frames/{id}/tags` | `204` |
| `DELETE` | `/api/curated-frames/{id}` | `204` |
| `POST` | `/api/curated-frames/export` | image / zip |
| `POST` | `/api/providers/ping` | `ProviderHealthDTO` |
| `POST` | `/api/providers/ping-all` | `PingAllProvidersResponse` |
| `POST` | `/api/proxy/ping-javbus` | `ProxyJavBusPingResponse` |
| `POST` | `/api/proxy/ping-google` | `ProxyJavBusPingResponse` |

## 7. 维护规则

后续维护或新增 API 时请同步更新：

1. `backend/internal/server/server.go` 或对应 handler 文件。
2. `backend/internal/contracts/contracts.go` 中的 DTO、错误码和注释。
3. `src/api/types.ts` 和 `src/api/endpoints.ts`。
4. 本文件 `API.md`，包括 Endpoint Reference、DTO 速查、全路由清单。
5. 如果新增端点或改变公开行为，还要按仓库规则同步 `.cursor/rules/project-facts.mdc`、`README.md`、`CLAUDE.md` 及相关 reference 文档。

兼容性建议：

- 新字段优先做可选字段，旧客户端可忽略。
- 变更枚举值前先确认前端、Android、桌面壳和脚本是否都已支持。
- 媒体端点不要改成 JSON envelope。
- `204 No Content` 接口不要新增 JSON body，除非同步所有客户端。
- 需要长耗时的操作优先返回 `202 TaskDTO`，避免客户端持有长连接。
- 新增错误场景时优先复用现有错误码；需要稳定区分时再新增错误码。
- 新增 query/body 字段时在 handler、Go DTO、TS types、本文档中同时落地。

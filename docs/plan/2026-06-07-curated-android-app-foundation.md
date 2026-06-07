# Curated Android App 基底需求与架构设计

日期：2026-06-07
状态：Foundation Draft
目标读者：后续搭建 Curated Android 客户端的产品、安卓、后端协作者
基于后端：当前仓库 Go HTTP 后端，API 前缀 `/api`，开发默认 `:8080`，release 默认 `:8081`

## 1. 背景与目标

Curated 当前已经是“本机或局域网可访问的 Go HTTP 服务 + Web/Electron 客户端 + SQLite 媒体库”。后端默认监听 `:8080`（开发态）或 `:8081`（release），该写法会绑定所有网络接口，因此安卓设备在同一局域网内可以通过 `http://<主机IP>:8080` 或 `http://<主机IP>:8081` 访问同一个 Curated 后端。

本需求希望新增一个类似 Findroid 使用体验的原生 Android App：用户在手机或平板上输入 Curated 后端 IP 地址，连接现有 Go 服务端，浏览 Curated 媒体库、查看详情、搜索筛选、播放视频、同步观看进度，并复用当前后端已有的 PIN App Lock、播放进度、收藏、评分、演员资料、封面/预览图、HLS/直连播放能力。

这里的“类似 Findroid”指产品体验：原生移动端媒体库浏览与播放，不表示复用 Jellyfin 协议，也不引入 Jellyfin 服务端兼容层。Curated Android 客户端应直接消费 Curated 的现有 REST API。

## 2. 设计原则

1. **服务端不重写**：第一版继续使用当前 Go 后端，不在 Android 端内嵌后端，不新增独立媒体扫描服务。
2. **API 优先于文件访问**：Android 端不直接访问 Windows 共享目录、数据库或媒体文件路径；所有浏览、资产、播放、进度都走 HTTP API。
3. **播放入口优先使用 descriptor**：Android 播放器先请求 `GET /api/library/movies/{id}/playback`，再根据 `mode` 播放 `url`。`/stream` 是实现细节，不作为唯一播放入口。
4. **局域网手动连接先行**：MVP 先支持手动输入 `http://<ip>:<port>`。自动发现、二维码配对、HTTPS、外网访问作为后续增强。
5. **后端会话为准**：如果后端启用 PIN App Lock，Android 端必须通过后端 auth API 解锁并持久化 `curated_auth` Cookie；不能用本地状态假装已解锁。
6. **移动端缓存只做体验优化**：Room/本地缓存用于离线展示最近元数据、减少列表闪烁和支持待同步进度，不作为媒体库真源。
7. **低后端改动 MVP**：第一阶段尽量只新增 Android 客户端；后端增强只记录为后续事项，除非发现当前 API 阻塞播放或鉴权。

## 3. 范围

### 3.1 MVP 必须包含

- 手动添加 Curated 服务器：输入 `baseUrl`，例如 `http://192.168.1.100:8081`。
- 连接校验：调用 `GET /api/health` 展示后端名称、版本、channel。
- PIN 解锁：调用 `GET /api/auth/status`、`POST /api/auth/unlock`，持久化并发送 `curated_auth` Cookie。
- 媒体库列表：调用 `GET /api/library/movies`，支持分页、搜索 `q`、演员 `actor`、厂商 `studio` 等基础筛选。
- 影片详情：调用 `GET /api/library/movies/{id}`，展示封面、预览图、标题、番号、演员、标签、简介、评分、时长、发行日期等。
- 同源图片加载：加载 `coverUrl`、`thumbUrl`、`previewImages`，并能处理相对 URL。
- 播放：使用 Android 原生播放器播放 `PlaybackDescriptorDTO.url`，支持 direct 与 HLS。
- 进度同步：启动时读取 `GET /api/playback/progress` 或使用 descriptor 的 `resumePositionSec`；播放中写入 `PUT /api/playback/progress/{movieId}`。
- 已播放标记：播放到阈值后调用 `POST /api/library/played-movies/{movieId}`。
- 观看时长统计：播放中周期性调用 `POST /api/playback/watch-time/daily`，保持与 Web 端统计一致。
- 收藏与评分：调用 `PATCH /api/library/movies/{id}` 更新 `isFavorite` 与 `rating`。
- 错误处理：统一处理 `AUTH_LOCKED`、`AUTH_INVALID_PIN`、`COMMON_NOT_FOUND`、网络不可达、视频不支持、HLS 会话失败。

### 3.2 MVP 可选但建议包含

- 演员列表与演员详情：`GET /api/library/actors`、`GET /api/library/actors/profile?name=`。
- 用户标签：影片 `userTags`、演员 `userTags` 的展示与修改。
- 影片评论：`GET/PUT /api/library/movies/{id}/comment`。
- 首页推荐：`GET /api/homepage/recommendations`，通过返回的 movie ids 拉取详情组成首页。
- 连接设备可见性：`GET /api/connected-clients`，用于调试“手机是否连到当前 Curated 后端”。
- 播放设置只读展示：`GET /api/settings`，读取 `player.streamPushEnabled`、快进快退步长等。

### 3.3 明确不做

- 不做 Jellyfin 协议兼容。
- 不做 Android 端扫描本地视频入库。
- 不做 Android 端刮削、批量导入、删除磁盘文件等高风险管理操作。
- 不做外网穿透、云账号、多人权限系统。
- 不做 Windows Service / Server Mode 改造。
- 不把当前 REST API 改成 WebSocket、gRPC 或 IPC。
- 不在第一版实现离线下载整片；这需要新的下载策略、存储配额、版权/隐私提示和断点续传设计。

## 4. 用户故事

| 编号 | 用户故事 | 验收点 |
|---|---|---|
| A-01 | 作为用户，我可以输入电脑上的 Curated 地址并连接 | `GET /api/health` 成功，保存 server profile |
| A-02 | 作为用户，Curated 开启 PIN 时我可以在安卓端解锁 | `POST /api/auth/unlock` 成功后后续 API 不再返回 `423 AUTH_LOCKED` |
| A-03 | 作为用户，我可以像媒体库 App 一样浏览影片海报流 | 分页加载，不一次性拉全量；封面/缩略图可显示 |
| A-04 | 作为用户，我可以搜索番号、标题、演员、厂商 | `GET /api/library/movies` 的 `q/actor/studio` 生效 |
| A-05 | 作为用户，我可以打开详情页查看简介、预览图、演员和本地评分 | 详情字段完整展示，图片 URL 正确归一化 |
| A-06 | 作为用户，我可以从详情页或列表直接播放视频 | 优先 descriptor；direct/HLS 都能进入播放器 |
| A-07 | 作为用户，我下次打开影片时可以继续上次进度 | 播放进度写回后 Web/Electron/Android 读取一致 |
| A-08 | 作为用户，我可以收藏、评分、标记已播放 | PATCH/POST 后刷新列表与详情状态 |
| A-09 | 作为用户，网络断开或服务端退出时 App 能清楚提示 | 不崩溃；显示重试、修改服务器地址、返回缓存 |
| A-10 | 作为用户，我能知道当前连接的是哪台 Curated 后端 | 设置页展示 base URL、health、版本和最后连接时间 |

## 5. Android 推荐技术基线

> 具体依赖版本在创建 Android 工程时再锁定。这里先确定方向，避免后续架构反复。

| 领域 | 推荐 | 用途 |
|---|---|---|
| 语言 | Kotlin | Android 主开发语言 |
| UI | Jetpack Compose | 原生声明式 UI、适配手机/平板 |
| 架构 | MVVM + Repository + UseCase | 与 Android 官方架构建议一致，便于测试 |
| 网络 | OkHttp + Retrofit 或 Ktor Client | REST API、Cookie、Range/HLS 网络请求 |
| 播放器 | AndroidX Media3 / ExoPlayer | direct stream、HLS、后台音视频控制基础 |
| 图片 | Coil | 加载同源封面、缩略图、演员头像 |
| 本地配置 | DataStore | server profile、偏好、非敏感配置 |
| 本地缓存 | Room | 最近影片、演员、进度 outbox、列表快照 |
| 后台任务 | WorkManager | 连接健康检查、待同步进度重试 |
| DI | Hilt 或 Koin | API client、repository、player service 注入 |
| 安全存储 | Android Keystore-backed storage | 持久化 `curated_auth` Cookie 或敏感 server token |

Android 官方参考：

- Jetpack Compose：https://developer.android.com/compose
- Media3：https://developer.android.com/media/media3
- App Architecture：https://developer.android.com/topic/architecture
- DataStore：https://developer.android.com/topic/libraries/architecture/datastore
- Room：https://developer.android.com/training/data-storage/room
- WorkManager：https://developer.android.com/develop/background-work/background-tasks/persistent
- Network Security Config：https://developer.android.com/privacy-and-security/security-config
- Cleartext communications：https://developer.android.com/privacy-and-security/risks/cleartext-communications

## 6. 总体架构

```text
Android App
  ├─ app shell / navigation
  ├─ feature-server          服务器添加、连接状态、健康检查
  ├─ feature-auth            PIN 解锁、锁定状态、Cookie 会话
  ├─ feature-home            首页推荐、继续观看
  ├─ feature-library         影片列表、搜索、筛选、分页
  ├─ feature-movie-detail    详情、预览图、收藏、评分、评论
  ├─ feature-player          Media3 播放器、进度同步、HLS session 清理
  ├─ feature-actors          演员列表、演员资料
  ├─ feature-settings        server profile、播放设置只读、诊断
  ├─ core-network            Retrofit/OkHttp、CookieJar、error mapper
  ├─ core-model              domain model、DTO mapper
  ├─ core-database           Room cache、pending sync queue
  └─ core-datastore          user preferences、server profile

Curated Go Backend
  ├─ /api/health
  ├─ /api/auth/*
  ├─ /api/library/movies*
  ├─ /api/library/actors*
  ├─ /api/library/movies/{id}/playback
  ├─ /api/library/movies/{id}/stream
  ├─ /api/playback/*
  └─ /api/settings, /api/connected-clients
```

### 6.1 网络层

`core-network` 负责：

- 保存和切换当前 `baseUrl`，要求以 `http://` 或 `https://` 开头，末尾不带 `/api`。
- 所有 API path 统一拼接为 `${baseUrl}/api/...`。
- 将后端返回的相对 URL 归一化为绝对 URL，例如 `/api/library/movies/abc/stream` 转为 `http://192.168.1.100:8081/api/library/movies/abc/stream`。
- OkHttp CookieJar 持久化 `curated_auth`。后端 Cookie 是 `HttpOnly`、`SameSite=Lax`，这只限制浏览器 JS；原生 HTTP client 仍应按 Cookie 规则保存并发送。
- 请求头建议：
  - `User-Agent: CuratedAndroid/<appVersion> (Android <sdk>; <device>)`
  - 可选后续增强：`X-Curated-Client: android-native`、`X-Curated-Client-Version`、`X-Curated-OS: Android`、`X-Curated-OS-Version: <version>`。当前后端的 CORS 列表已允许这些 header，但 connected-client 识别逻辑对 Android marker 的特殊展示还需要后续后端增强；MVP 可先依赖 User-Agent。
- 统一错误映射：
  - HTTP `423` + `AUTH_LOCKED`：切到 Lock flow。
  - HTTP `401` + `AUTH_INVALID_PIN`：PIN 错误。
  - HTTP `404` + `COMMON_NOT_FOUND`：影片、图片或 HLS session 不存在。
  - HTTP `409` + `COMMON_CONFLICT`：扫描等任务冲突。
  - HTTP `5xx`：显示可重试服务端错误。

### 6.2 清晰的 URL 规则

Android 端必须区分三类 URL：

| 类型 | 示例 | 处理 |
|---|---|---|
| 服务器 base URL | `http://192.168.1.100:8081` | 用户输入，保存到 server profile |
| API path | `/api/library/movies` | 网络层拼接，不暴露给 UI |
| 后端返回的资源 URL | `/api/library/movies/{id}/asset/cover` | 若为相对 URL，按当前 server profile 转绝对 URL |

禁止把 Web 前端的 Vite 地址 `http://localhost:5173` 当成 Android API 地址。Android 端应直连 Go 后端端口。

### 6.3 Cleartext HTTP

当前 Curated 后端默认是 HTTP，不是 HTTPS。Android 9+ 对 cleartext HTTP 有默认限制，因此 Android 工程需要 Network Security Config 明确允许用户配置的局域网 HTTP，或者仅在 debug 构建允许 cleartext。MVP 可先支持局域网 HTTP，但产品文案必须说明：

- 这适用于可信家庭/个人局域网。
- 不建议把端口暴露到公网。
- 长期方向是 HTTPS 反向代理、配对 token 或服务端内置 TLS。

## 7. 数据模型

### 7.1 ServerProfile

```text
ServerProfile
  id: String
  displayName: String
  baseUrl: String
  lastHealthName: String
  lastVersion: String
  lastChannel: String
  lastConnectedAt: Instant
  trustedForever: Boolean
```

`curated_auth` Cookie 不建议直接放在普通 DataStore 明文里。实现时应使用 Android Keystore-backed storage 或受保护的 Cookie store。

### 7.2 Domain Models

Android domain model 不直接暴露后端 DTO，建议通过 mapper 转换：

- `MovieListItem`
- `MovieDetail`
- `ActorListItem`
- `ActorProfile`
- `PlaybackDescriptor`
- `PlaybackProgress`
- `AuthStatus`
- `AppError`

后端字段名以 `src/api/types.ts` 和 `backend/internal/contracts/contracts.go` 为准。

### 7.3 Room Cache

MVP 推荐缓存：

- 最近一次成功加载的分页影片列表快照。
- 影片详情快照。
- 演员列表/资料快照。
- 播放进度 outbox：网络失败时只保留每个 movieId 最新进度，恢复连接后覆盖写回。
- server health 快照。

不建议缓存视频字节；视频交给 Media3 通过 HTTP Range/HLS 读取。

## 8. 核心流程设计

### 8.1 首次连接

```text
用户输入 baseUrl
  -> GET {baseUrl}/api/health
  -> 成功：保存 ServerProfile
  -> GET {baseUrl}/api/auth/status
      -> pinEnabled=false：进入首页/媒体库
      -> pinEnabled=true && unlocked=false：进入 PIN 解锁
      -> setupRequired=true：显示“服务端需要设置 PIN”的引导
  -> 失败：显示不可达、端口、防火墙、HTTP cleartext 提示
```

### 8.2 PIN 解锁

```text
GET /api/auth/status
  -> pinEnabled=true, unlocked=false
用户输入 PIN
  -> POST /api/auth/unlock { pin, trustedForever }
  -> 后端 Set-Cookie: curated_auth=...
  -> CookieJar 持久化 cookie
  -> 重新请求原目标 API
```

任意 API 返回 `423 AUTH_LOCKED` 时，都应清除内存中的 unlocked 状态，进入 PIN flow。是否删除本地 Cookie 取决于 CookieJar 实现；如果后端判断该 Cookie 无效，应允许用户重新解锁覆盖。

### 8.3 影片列表

```text
LibraryScreen
  -> GET /api/library/movies?limit=40&offset=0
  -> 下滑加载更多 offset += limit
  -> 搜索 q：debounce 后重新从 offset=0 加载
  -> actor/studio 筛选：使用精确字段
  -> 图片 URL 归一化后交给 Coil
```

分页不能一次性拉全量。后端响应为 `{ items, total, limit, offset }`。

### 8.4 影片详情

```text
GET /api/library/movies/{id}
  -> 展示 MovieDetailDTO
  -> previewImages 逐个归一化 URL
  -> 演员头像来自 actorAvatarUrls 或演员 profile/avatar endpoint
  -> 并行 GET /api/library/movies/{id}/comment（可选）
```

可编辑状态：

- 收藏：`PATCH /api/library/movies/{id} { "isFavorite": true|false }`
- 评分：`PATCH /api/library/movies/{id} { "rating": number|null }`
- 用户标签：`PATCH /api/library/movies/{id} { "userTags": [...] }`
- 评论：`PUT /api/library/movies/{id}/comment { "body": "..." }`

### 8.5 播放

```text
用户点击播放
  -> GET /api/library/movies/{id}/playback
  -> descriptor.mode == "direct"
       -> Media3 播放 absolute(descriptor.url)
  -> descriptor.mode == "hls"
       -> Media3 HLS 播放 absolute(descriptor.url)
  -> descriptor.resumePositionSec > 0
       -> seekTo(resumePositionSec)
  -> 播放中周期性 PUT /api/playback/progress/{movieId}
  -> 播放结束或达到阈值 POST /api/library/played-movies/{movieId}
  -> 若 descriptor.sessionId 存在，退出播放器时 DELETE /api/playback/sessions/{sessionId}
```

直连视频 endpoint `GET /api/library/movies/{id}/stream` 使用 `http.ServeContent`，支持 HTTP Range。Media3/OkHttp 应让播放器按需请求，不要手动一次性下载完整视频。

HLS playlist/segment endpoint：

- `GET /api/playback/sessions/{sessionId}/hls/index.m3u8`
- `GET /api/playback/sessions/{sessionId}/hls/{segment}.ts`
- `DELETE /api/playback/sessions/{sessionId}`

如果 direct 播放失败且后端 `settings.player.streamPushEnabled=true`，Android 端可以尝试：

```http
POST /api/library/movies/{id}/playback-session
Content-Type: application/json

{ "mode": "hls", "startPositionSec": 123.4 }
```

如果该请求失败，应提示用户在 Curated Settings -> Playback 中启用 Stream Push / FFmpeg，或回退为不可播放错误。

### 8.6 进度与观看时长

播放进度建议策略：

- 进入播放器前读取 descriptor 的 `resumePositionSec`，必要时也可读取 `GET /api/playback/progress` 本地合并。
- 播放时每 5-10 秒写入一次最新进度，且只在 position 变化明显时写入。
- 暂停、退出播放器、App 进入后台时立即写入一次。
- duration 优先使用播放器真实 duration；不可用时使用 descriptor `durationSec`。
- 网络失败时把每个 movieId 的最新进度写入本地 outbox，恢复后只提交最新值。

观看时长建议策略：

- 每 60 秒统计一次有效播放增量，调用 `POST /api/playback/watch-time/daily`。
- `dayKey` 使用 Android 设备本地日期 `YYYY-MM-DD`，与 Web 端当前实现保持一致。
- 快进、暂停、缓冲、后台不应累计为观看时长。

## 9. 后端 API 使用清单

### 9.1 连接与鉴权

| 能力 | API | MVP |
|---|---|---|
| 健康检查 | `GET /api/health` | 必须 |
| Auth 状态 | `GET /api/auth/status` | 必须 |
| 初次设置 PIN | `POST /api/auth/setup-pin` | 可选；也可提示用户回桌面端设置 |
| 解锁 | `POST /api/auth/unlock` | 必须 |
| 锁定当前会话 | `POST /api/auth/lock` | 建议 |
| 修改 PIN | `POST /api/auth/change-pin` | 后续 |
| 修改锁设置 | `PATCH /api/auth/settings` | 后续 |

### 9.2 媒体库

| 能力 | API | MVP |
|---|---|---|
| 影片分页列表 | `GET /api/library/movies?q=&actor=&studio=&limit=&offset=` | 必须 |
| 影片详情 | `GET /api/library/movies/{movieId}` | 必须 |
| 更新收藏/评分/标签 | `PATCH /api/library/movies/{movieId}` | 必须 |
| 删除影片 | `DELETE /api/library/movies/{movieId}` | 不做 |
| 恢复影片 | `POST /api/library/movies/{movieId}/restore` | 不做 |
| 封面/缩略图 | `GET /api/library/movies/{movieId}/asset/{cover|thumb}` | 必须 |
| 预览图 | `GET /api/library/movies/{movieId}/asset/preview/{index}` | 建议 |
| 评论读取 | `GET /api/library/movies/{movieId}/comment` | 建议 |
| 评论写入 | `PUT /api/library/movies/{movieId}/comment` | 建议 |

### 9.3 播放

| 能力 | API | MVP |
|---|---|---|
| 播放 descriptor | `GET /api/library/movies/{movieId}/playback` | 必须 |
| 直连视频流 | `GET /api/library/movies/{movieId}/stream` | descriptor 使用 |
| 创建 HLS session | `POST /api/library/movies/{movieId}/playback-session` | 建议 |
| HLS playlist/segments | `GET /api/playback/sessions/{sessionId}/hls/{file}` | 建议 |
| 删除 HLS session | `DELETE /api/playback/sessions/{sessionId}` | 建议 |
| session 状态 | `GET /api/playback/sessions/{sessionId}` | 调试 |
| 最近 session | `GET /api/playback/sessions/recent` | 调试 |
| 后端 native play | `POST /api/library/movies/{movieId}/native-play` | Android 不使用 |

### 9.4 进度与历史

| 能力 | API | MVP |
|---|---|---|
| 进度列表 | `GET /api/playback/progress` | 必须 |
| 写入进度 | `PUT /api/playback/progress/{movieId}` | 必须 |
| 清除进度 | `DELETE /api/playback/progress/{movieId}` | 可选 |
| 已播放列表 | `GET /api/library/played-movies` | 建议 |
| 标记已播放 | `POST /api/library/played-movies/{movieId}` | 必须 |
| 每日观看时长 | `GET /api/playback/watch-time/daily` | 可选 |
| 增加观看时长 | `POST /api/playback/watch-time/daily` | 必须 |

### 9.5 演员

| 能力 | API | MVP |
|---|---|---|
| 演员列表 | `GET /api/library/actors?q=&actorTag=&sort=&limit=&offset=` | 建议 |
| 演员资料 | `GET /api/library/actors/profile?name=` | 建议 |
| 演员头像 | `GET /api/library/actors/{name}/asset/avatar` | 建议 |
| 演员标签 | `PATCH /api/library/actors/tags?name=` | 后续 |
| 演员外链 | `PATCH /api/library/actors/external-links?name=` | 后续 |
| 刮削演员资料 | `POST /api/library/actors/scrape?name=` | 后续 |

### 9.6 首页、设置、诊断

| 能力 | API | MVP |
|---|---|---|
| 首页推荐 IDs | `GET /api/homepage/recommendations` | 可选 |
| 设置读取 | `GET /api/settings` | 建议 |
| 设置写入 | `PATCH /api/settings` | 后续 |
| 已连接客户端 | `GET /api/connected-clients` | 调试 |
| 任务状态 | `GET /api/tasks/{taskId}` | 后续 |
| 最近任务 | `GET /api/tasks/recent` | 后续 |
| 扫描 | `POST /api/scans` | 不做 |
| 导入影片 | `POST /api/import/movies` 与 resumable upload | 不做 |

## 10. UI 信息架构

### 10.1 导航结构

```text
ServerGate
  ├─ AddServerScreen
  ├─ AuthLockScreen
  └─ MainShell
       ├─ HomeScreen
       ├─ LibraryScreen
       ├─ MovieDetailScreen
       ├─ PlayerScreen
       ├─ ActorsScreen
       └─ SettingsScreen
```

### 10.2 屏幕职责

| 屏幕 | 职责 |
|---|---|
| AddServerScreen | 输入 IP/端口、测试连接、保存 server profile |
| AuthLockScreen | PIN 输入、错误提示、trust this device 选项 |
| HomeScreen | 继续观看、每日推荐、最近添加 |
| LibraryScreen | 瀑布流/网格、搜索、分页、筛选 |
| MovieDetailScreen | 影片信息、预览、演员、收藏、评分、播放入口 |
| PlayerScreen | 横屏优先、Media3 PlayerView/Compose 包装、进度同步 |
| ActorsScreen | 演员列表、头像、影片数量、跳转筛选 |
| SettingsScreen | 当前服务器、连接测试、Cookie/锁定、播放诊断 |

### 10.3 移动端体验要点

- 影片卡片触摸目标不小于 Android 常规点击区域。
- 列表用分页 lazy grid，不在 UI 层持有全量媒体库。
- 播放页默认沉浸式，竖屏显示简洁控制，横屏进入低干扰全屏。
- 搜索输入 debounce，避免每个字符都请求后端。
- 所有图片加载失败都有占位，不暴露本地文件路径。
- 连接错误要给出明确排查：电脑是否启动 Curated、端口是否正确、防火墙是否放行、手机是否同一局域网、是否被 PIN 锁住。

## 11. 安全与隐私

1. **PIN 仍由后端验证**：Android 端不保存 PIN 明文，不保存 PIN hash。
2. **Cookie 属于敏感凭据**：`curated_auth` 持久化必须使用受保护存储；退出登录/锁定时调用 `POST /api/auth/lock` 并清理本地 Cookie。
3. **局域网 HTTP 有风险**：MVP 可以允许 cleartext HTTP，但 App 应明确这是局域网信任模型。不要鼓励公网暴露。
4. **日志脱敏**：日志不得记录 PIN、Cookie、完整媒体路径、完整 query 中的敏感内容。
5. **截图隐私**：PIN 页面和播放器页面建议设置 `FLAG_SECURE` 选项开关，至少 PIN 页面默认禁止系统截图/任务快照。
6. **备份策略**：Cookie/敏感 server profile 不应进入 Android 自动备份。
7. **设备识别**：当前后端 connected clients 不采集 MAC 地址，Android 端也不应尝试获取或上传 MAC。

## 12. 需要关注的后端现状

- `GET /api/library/movies/{id}/stream` 支持 Range，适合 Media3 按需读取。
- `GET /api/library/movies/{id}/playback` 已经是播放规划入口，包含 `mode`、`url`、`mimeType`、`resumePositionSec`、`sessionId`、`reasonCode` 等字段。
- HLS session 依赖后端 `player.streamPushEnabled` 与 FFmpeg；如果未启用或启动失败，后端可能回退 direct，Android 端应容错。
- PIN App Lock 已保护敏感 `/api/*`，未解锁返回 `423 AUTH_LOCKED`。
- `GET /api/health` 和 auth endpoints 是公开路径。
- 当前 connected-client tracker 可通过 User-Agent 识别 Android/mobile；若要显示 “Curated Android” 这种专门名称，需要后端后续识别 `X-Curated-Client: android-native`。
- 后端默认 HTTP 绑定所有网络接口，Windows 防火墙仍可能拦截局域网设备访问。

## 13. 后续后端增强建议

这些不是 MVP 阻塞项，但会显著改善 Android 体验：

1. **Android client marker**：后端识别 `X-Curated-Client: android-native`，在 `GET /api/connected-clients` 中显示 `browser: "Curated Android"`。
2. **服务器配对二维码**：Web/Electron Settings 展示 `baseUrl` + 可选一次性配对 token 的 QR，Android 扫码添加服务器。
3. **LAN access policy**：在现有 PIN 基础上补“仅本机 / 允许 LAN / LAN 必须 PIN / 拒绝未知设备”等策略。
4. **HTTPS 或 pairing token**：降低局域网明文凭据风险。
5. **Playback client capabilities**：Android 请求 playback descriptor 时可提交支持的 container/codec，后端按 Android 能力决策 direct vs HLS，而不是只按 Web 浏览器能力。
6. **字幕/音轨接口完善**：当前 descriptor 有 `audioTracks`、`subtitleTracks` 字段，但实际能力仍需后续补齐。
7. **播放 session 心跳/显式 keepalive**：如果 HLS session 清理策略后续变严，Android 端需要心跳 endpoint 或访问 playlist/segments 自动续期的明确契约。
8. **离线下载 API**：若要做本地缓存整片，需新增受控下载/转码/存储配额设计，而不是直接复用 `/stream` 偷跑下载。

## 14. 分阶段落地建议

### Phase 0：技术验证

- 创建最小 Android 工程。
- 手动配置 `baseUrl`。
- `GET /api/health`、`GET /api/auth/status`、`POST /api/auth/unlock` 跑通。
- 列表加载 20 条影片并显示封面。
- 播放一个 direct descriptor，验证 Range 请求、seek、退出后继续播放。

### Phase 1：MVP

- 完整 server profile 管理。
- PIN flow 与 Cookie 持久化。
- 影片列表、详情、搜索、筛选、收藏、评分。
- Media3 direct/HLS 播放。
- 进度、已播放、观看时长同步。
- 基础设置/诊断页。
- 单元测试覆盖 API mapper、error mapper、URL 归一化、Cookie store。

### Phase 2：媒体库体验完善

- 首页推荐、继续观看、最近添加。
- 演员库与演员详情。
- 评论、用户标签。
- HLS fallback 策略更细。
- 平板布局、横屏播放器、Android TV 方向评估。

### Phase 3：配对、安全、远程能力

- Web/Electron 端显示二维码配对。
- Android client marker + connected clients 显示。
- LAN policy、token 或 HTTPS。
- 可选离线下载设计。

## 15. MVP 验收标准

- 在 Windows 主机启动 `cd backend && go run ./cmd/curated` 后，安卓真机可通过 `http://<主机IP>:8080` 连接。
- 如果使用 release 包，安卓真机可通过 `http://<主机IP>:8081` 连接。
- 后端启用 PIN 时，未解锁请求媒体库返回 `423 AUTH_LOCKED`，Android 端显示 PIN 页面；解锁成功后媒体库可访问。
- 影片列表分页正常，滑动到底部不重复、不乱序、不一次性拉全量。
- 封面、缩略图、预览图能从同一后端加载。
- 直连播放可开始、暂停、拖动进度、退出再继续。
- HLS descriptor 可播放；退出播放器会删除显式创建的 session。
- 播放进度在 Android、Web、Electron 之间一致。
- 收藏/评分更新后刷新详情和列表可见。
- 后端关闭、IP 错误、防火墙阻断时有清晰错误和重试入口。

## 16. 验证环境建议

- 后端开发态：`cd backend && go run ./cmd/curated`。
- 物理安卓设备：连接同一 Wi-Fi，使用 Windows 主机局域网 IP。
- Android Emulator：如果访问宿主机，可先验证 `http://10.0.2.2:8080`；如果要模拟局域网真机，仍建议物理设备。
- Windows 防火墙：需要允许后端端口入站连接。
- Android cleartext：debug 构建先允许局域网 HTTP；release 构建必须有用户可理解的安全提示或改用 HTTPS/token。
- 后端 API 契约核对：`API.md`、`backend/internal/server/server.go`、`backend/internal/contracts/contracts.go`、`src/api/types.ts`。

## 17. 当前结论

Curated Android App 第一版可以在不改造 Go 后端的前提下启动。关键路径不是“搭新服务端”，而是把 Android 端做成一个正确消费 Curated REST API 的原生客户端：

- 手动连接局域网 Curated 后端。
- 正确处理 PIN Cookie 会话。
- 使用 playback descriptor 驱动 Media3。
- 用 `/stream` 和 HLS session 完成播放。
- 把进度、已播放、评分、收藏写回现有 SQLite 后端。

后续真正需要后端配合的是配对、安全、Android 专属 connected-client 标识、客户端能力驱动播放决策，以及离线下载等增强能力。

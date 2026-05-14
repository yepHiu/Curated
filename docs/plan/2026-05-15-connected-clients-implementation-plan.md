# Connected Clients 详细实施计划

日期：2026-05-15

基于：`docs/features/2026-05-03-connected-clients.md`

本文把 Connected Clients 从概念设计拆成可执行切片。用户确认的 UI 位置为：**Settings -> Overview 中，观看时间统计展示下面**。

静态 UI 原型：`docs/plan/2026-05-15-connected-clients-prototype.html`

## 目标

在 Settings -> Overview 增加 **Connected Clients / 连接的客户端** 卡片，展示当前后端进程生命周期内访问过 Curated 的客户端设备，帮助用户理解“谁正在访问我的 Curated”。

第一版只做可见性，不做 PIN、账号、token 鉴权和设备授权。PIN App Lock / LAN policy 后续可以基于同一套 client tracker 扩展。

## 非目标

- 不收集、不展示 MAC 地址。
- 不做历史持久化；后端重启后客户端列表清空。
- 不做多用户、角色、权限系统。
- 不在第一版阻断 LAN 访问。
- 不把 connected clients 写进前端 localStorage。
- 不在组件内直接调用 `src/api/endpoints.ts`；仍通过 `LibraryService` contract。

## 关键产品决策

| 决策点 | 结论 |
|---|---|
| UI 位置 | Settings -> Overview，观看时间统计卡片之后 |
| 记录生命周期 | 后端进程内存，重启清空 |
| 去重键 | `remoteIP + userAgent`，不包含端口 |
| 最大条目数 | 50 个最近活跃客户端 |
| 排序 | `lastSeen` 倒序 |
| 轮询 | Overview tab 激活时每 60 秒刷新；离开 Overview 停止 |
| Hostname | 第一版可选；推荐实现 best-effort reverse DNS，失败留空 |
| User-Agent 解析 | 第一版使用小型内置 parser/heuristics；后续再评估第三方库 |
| Health 增强 | 第一版可加 `connectedClients` 计数，但不是 UI 必须依赖 |
| 安全扩展 | 为 PIN/授权预留 DTO 字段的后续版本，不在 MVP 返回解锁状态 |

## 数据模型

### Backend internal snapshot

```go
type ClientSnapshot struct {
    Key          string
    RemoteAddr   string
    IP           string
    Port         int
    UserAgent    string
    Browser      string
    BrowserVersion string
    OS           string
    OSVersion    string
    DeviceType   string // desktop | laptop | mobile | tablet | tool | unknown
    AccessKind   string // local | remote
    IsLocalMachine bool
    Hostname     string
    FirstSeen    time.Time
    LastSeen     time.Time
    RequestCount int64
}
```

### API DTO

```typescript
export interface ConnectedClientDTO {
  key: string
  ip: string
  port?: number
  hostname?: string
  userAgent?: string
  browser: string
  browserVersion?: string
  os: string
  osVersion?: string
  deviceType: "desktop" | "laptop" | "mobile" | "tablet" | "tool" | "unknown"
  accessKind: "local" | "remote"
  isLocalMachine: boolean
  firstSeen: string
  lastSeen: string
  requestCount: number
}

export interface ConnectedClientsDTO {
  clients: ConnectedClientDTO[]
  total: number
  localCount: number
  remoteCount: number
  sampledAt: string
}
```

未来 PIN / 设备授权可以扩展：

```typescript
// future
isUnlocked?: boolean
sessionExpiresAt?: string
lastUnlockAt?: string
canRevokeSession?: boolean
```

## 后端实施

### 1. 新增 client tracker package

新增目录：

```text
backend/internal/clienttracker/
  tracker.go
  user_agent.go
  hostname.go
  tracker_test.go
  user_agent_test.go
```

职责：

- `Record(r *http.Request)`：记录非空 IP + UA 的请求。
- `Snapshot() []ClientSnapshot`：返回 `lastSeen desc` 的副本。
- `RefreshLocalInterfaces()`：启动时枚举本机接口 IP。
- `ParseUserAgent(ua string)`：识别常见浏览器、OS、设备类型。
- `ResolveHostname(ip string)`：仅对 private LAN IP 做 best-effort reverse DNS，超时 500ms。

内部约束：

- 使用 `sync.RWMutex` 保护 map。
- key 不存明文 UA 全量作为 map key；用 hash 后 key 保存。
- 条目超过 50 时，删除最久未活跃的条目。
- `OPTIONS` 预检请求不计入 requestCount。
- 允许记录 `/api/health`，因为它也代表客户端在线。

### 2. User-Agent 解析策略

MVP 不引入第三方依赖，先用简单 heuristics：

| UA 特征 | 输出 |
|---|---|
| `Edg/` | Edge |
| `Chrome/` 且非 Edge | Chrome |
| `Firefox/` | Firefox |
| `Version/... Safari/` 且非 Chrome | Safari |
| `curl/` | Tool / Script |
| `Windows NT 10.0` | Windows |
| `Mac OS X` | macOS |
| `iPhone` | iOS + mobile |
| `iPad` | iPadOS + tablet |
| `Android` + `Mobile` | Android + mobile |
| `Android` 非 Mobile | Android + tablet |

原因：设备可见性只需要“足够好”的展示，不值得第一版为 UA parser 增加依赖和后续维护成本。

### 3. 新增中间件

在 `backend/internal/server/server.go` 增加 client tracking middleware：

```go
func withClientTracking(next http.Handler, tracker *clienttracker.Tracker) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tracker.Record(r)
        next.ServeHTTP(w, r)
    })
}
```

注册位置：

- 在 access logging 之后或附近。
- 在实际 mux handler 之前。
- 不改变已有 CORS、静态托管、API 路由行为。

### 4. 新增 endpoint

新增：

```text
GET /api/connected-clients
```

响应：

```json
{
  "clients": [],
  "total": 0,
  "localCount": 0,
  "remoteCount": 0,
  "sampledAt": "2026-05-15T10:00:00Z"
}
```

实现位置可先放在 `server.go`，但建议如果 handler 继续增长，后续拆成：

```text
backend/internal/server/connected_clients_handlers.go
```

### 5. 可选 health 增强

`GET /api/health` 可返回：

```json
{
  "connectedClients": 3
}
```

这不是 UI 必需字段。若担心 health 在未来锁屏前泄露信息，可以暂不加。

### 6. 后端测试

新增/扩展测试：

- `clienttracker.Tracker`：
  - 同 IP + UA 去重。
  - 同 IP + 不同 UA 生成不同客户端。
  - requestCount、firstSeen、lastSeen 更新。
  - 超过 50 条裁剪最旧客户端。
  - loopback 识别为 local。
  - private LAN 识别为 remote。
- `ParseUserAgent`：
  - Chrome/Edge/Firefox/Safari。
  - Windows/macOS/iOS/Android。
  - curl/python requests -> tool。
- Server handler：
  - 访问 `/api/health` 后 `GET /api/connected-clients` 返回至少一个客户端。
  - response shape 包含 total/localCount/remoteCount/sampledAt。
  - 不返回 MAC 字段。

## 前端实施

### 1. API types

修改：

```text
src/api/types.ts
```

新增：

- `ConnectedClientDTO`
- `ConnectedClientsDTO`

### 2. API endpoint

修改：

```text
src/api/endpoints.ts
```

新增：

```typescript
listConnectedClients(): Promise<ConnectedClientsDTO>
```

如项目继续扩展 API guards，可新增轻量 guard：

```text
isConnectedClientsDTO(value: unknown): value is ConnectedClientsDTO
```

### 3. Service contract

修改：

```text
src/services/contracts/library-service.ts
src/services/adapters/web/web-library-service.ts
src/services/adapters/mock/mock-library-service.ts
```

新增方法：

```typescript
listConnectedClients(): Promise<ConnectedClientsDTO>
```

Mock adapter 返回示例客户端，方便 mock 模式下确认 UI：

- Local Chrome / Windows
- Remote Safari / iPhone
- Remote Firefox / macOS

### 4. Settings state

修改：

```text
src/components/jav-library/SettingsPage.vue
```

新增状态：

```typescript
const connectedClients = ref<ConnectedClientDTO[]>([])
const connectedClientsLoading = ref(false)
const connectedClientsError = ref("")
const connectedClientsSampledAt = ref("")
let connectedClientsPollTimer: number | null = null
```

行为：

- 进入 `overview` 时立即刷新一次。
- 停留在 `overview` 时每 60 秒刷新。
- 离开 `overview` 时停止轮询。
- 手动点击 Refresh 时立即刷新。
- 请求失败不清空上一次成功数据，只显示错误提示。

可以把轮询逻辑拆成 composable：

```text
src/composables/use-connected-clients.ts
```

但第一版若只被 Settings 使用，也可以先放 SettingsPage；若 SettingsPage 继续变大，建议直接拆 composable。

### 5. 新增 UI component

新增：

```text
src/components/jav-library/settings/SettingsConnectedClientsSection.vue
src/components/jav-library/settings/SettingsConnectedClientsSection.test.ts
```

Props：

```typescript
defineProps<{
  clients: readonly ConnectedClientDTO[]
  loading?: boolean
  error?: string
  sampledAt?: string
}>()
```

Emits：

```typescript
defineEmits<{
  refresh: []
}>()
```

组件结构：

- `Card` 大卡片。
- `CardHeader` 使用 settings 标准图标标题行。
- `CardDescription` 简短说明：当前进程内访问过 Curated 的客户端；不收集 MAC。
- Summary row：
  - total clients
  - remote clients
  - local clients
  - last refresh
- Client list：
  - 每行 icon + 设备名/浏览器系统
  - IP + hostname
  - Local/Remote badge
  - This device badge
  - last seen
  - request count
- Empty state：
  - 没有远程设备时仍显示本机，若完全无数据显示“暂无客户端记录”。
- Error state：
  - 保留旧数据，同时显示 inline warning。

### 6. Overview placement

修改：

```text
src/components/jav-library/settings/SettingsOverviewSection.vue
```

当前顺序：

```vue
<StatsCard />
<SettingsWatchTimeHeatmap />
```

目标顺序：

```vue
<StatsCard />
<SettingsWatchTimeHeatmap />
<SettingsConnectedClientsSection />
```

这满足用户要求：**连接的客户端展示在时间统计展示下面**。

### 7. i18n keys

修改：

```text
src/locales/en.json
src/locales/zh-CN.json
src/locales/ja.json
```

建议 key：

```json
{
  "settings": {
    "connectedClientsTitle": "Connected clients",
    "connectedClientsDesc": "Devices and browsers that have accessed Curated during this backend session.",
    "connectedClientsPrivacy": "MAC addresses are not collected or shown.",
    "connectedClientsRefresh": "Refresh",
    "connectedClientsTotal": "Clients",
    "connectedClientsRemote": "Remote",
    "connectedClientsLocal": "Local",
    "connectedClientsLastSeen": "Last seen {time}",
    "connectedClientsRequests": "{count} requests",
    "connectedClientsThisDevice": "This device",
    "connectedClientsLocalBadge": "Local",
    "connectedClientsRemoteBadge": "Remote",
    "connectedClientsEmpty": "No connected clients recorded yet.",
    "connectedClientsUnknownDevice": "Unknown device",
    "connectedClientsTool": "Tool / Script"
  }
}
```

## UI 细节

### 布局

- 大卡片宽度与 Settings Overview 其他卡片一致。
- 使用 `gap-2 rounded-xl border border-border bg-card shadow-sm`。
- Summary metrics 采用与 Watch Time 一致的 nested metric block。
- Client row 使用 `rounded-lg border border-border/50 bg-muted/5`，避免再嵌套 card。
- Desktop：列表为纵向 rows，不做表格，便于移动端折行。
- Mobile：每行自然换行，badge 到下一行，不产生水平滚动。

### 信息层级

首屏信息优先级：

1. 当前总连接数 / 远程连接数。
2. 哪个设备是本机。
3. 远程设备最后访问时间。
4. IP / hostname。
5. 请求次数。
6. raw User-Agent 只在未来详情展开里展示，不在 MVP 直接铺开。

### 状态表达

- Local：secondary / muted badge。
- Remote：warning/info tone，但不要使用 raw Tailwind 状态色，优先通过现有 status tone helper 或语义 token。
- This device：primary-outline badge。
- Tool / Script：muted badge。

## 验收标准

### Backend

- 访问 Curated 任意页面/API 后，`GET /api/connected-clients` 返回当前客户端。
- 同一 IP + UA 多次请求只产生一条 client，requestCount 增加。
- 不展示 MAC。
- loopback 请求显示 `accessKind=local`。
- LAN 请求显示 `accessKind=remote`。
- 最多保留 50 个最近活跃客户端。

### Frontend

- Settings -> Overview 中顺序为：统计卡片 -> 观看时间统计 -> 连接的客户端。
- Overview 激活时刷新 connected clients，离开后停止 60s 轮询。
- 手动 Refresh 可刷新。
- 加载、错误、空态都有可见反馈。
- Mock 模式下可看到示例数据，便于 UI 验收。
- 组件不直接 import `@/api/endpoints`。

### Tests

建议运行：

```bash
pnpm test -- src/components/jav-library/settings/SettingsConnectedClientsSection.test.ts src/components/jav-library/settings/SettingsOverviewSection.test.ts src/services/adapters/web/web-library-service.test.ts src/services/adapters/mock/mock-library-service.test.ts
```

后端：

```bash
cd backend
go test ./internal/clienttracker ./internal/server
```

最终全量：

```bash
pnpm typecheck
pnpm test
cd backend && go test ./...
```

## 实施顺序

1. 后端 clienttracker 纯逻辑 + 单测。
2. 后端 middleware + endpoint + handler 测试。
3. 前端 DTO + endpoint + service contract。
4. Mock adapter 示例数据。
5. `SettingsConnectedClientsSection.vue` + 单测。
6. `SettingsOverviewSection.vue` 放置到 watch time 下方。
7. `SettingsPage.vue` 接入刷新/轮询状态。
8. i18n 三语文案。
9. 文档同步：`docs/features/2026-05-03-connected-clients.md` 状态从 Draft 更新为 Implemented/MVP；必要时更新 `project-facts.mdc`、`API.md`、`README.md`。

## 后续扩展

- 与 PIN App Lock 结合，显示设备是否已解锁、会话何时过期。
- 增加 “Revoke device session”。
- Settings -> Security 中增加 LAN policy：Localhost only / LAN allowed / LAN requires PIN。
- 增加 SSE 推送客户端上下线事件，减少 60s 轮询。

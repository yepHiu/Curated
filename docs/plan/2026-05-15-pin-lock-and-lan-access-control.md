# PIN 锁与 LAN 访问控制设想

日期：2026-05-15

## Implementation Status - 2026-05-15

- Accepted direction: implement **方案 B** first. The backend owns PIN verification, stores only Argon2id hash/salt/KDF in SQLite, and protects sensitive `/api/*` routes with auth middleware.
- MVP device trust decision: after one successful PIN unlock, the user may select "trust this device forever". This creates a trusted-forever server-side session with no expiry and a long-lived HTTP-only `curated_auth` cookie; it remains valid until explicitly locked or revoked.
- Lock-screen UX decision: no background photo, no media poster/title, no numeric keypad. The screen shows PIN cells based on the configured PIN length and relies on keyboard input.
- First implemented settings entry: Settings -> Security. It covers initial PIN setup, PIN change, idle lock delay, LAN PIN policy, restart-lock policy, immediate lock, and explanatory copy for forgotten PIN and trusted devices. Initial setup and PIN change are opened from entry buttons into shadcn-vue dialogs instead of rendering PIN fields inline.
- Deferred from this slice: disabling PIN from UI, automatic PIN recovery, failed-attempt cooldown, connected-client session status, and per-device revocation from Settings.

本文承接 `docs/features/2026-05-03-connected-clients.md` 与 `docs/plan/2026-05-03-electron-migration-plan.md` 里的 LAN 设备可见性、安全边界和 Server Mode 讨论。目标不是一开始做复杂多用户系统，而是先满足单用户本地媒体库的隐私需求：即使别人拿到已解锁电脑或打开同一局域网地址，也不能直接进入影片库。

## 背景判断

- Curated 当前是本地 HTTP 服务 + Vue SPA + SQLite，Electron 桌面壳会让后端常驻更自然。
- 局域网访问已经具备基础条件，但旧文档已指出当前没有完整鉴权，LAN 中任何人拿到地址都可能访问。
- 当前用户模型很可能是“一个人使用自己的库”，因此不必先做账号、角色、邀请、多人权限。
- 真正的需求更接近 **App Lock**：打开 Curated 时先输入 PIN，解锁后在一段时间内免输入。
- PIN 锁不是磁盘加密，也不能防拥有系统文件权限的人直接读取数据库或篡改配置；它主要防“临时使用这台电脑的人直接打开 Curated 页面”以及“LAN 上有人随手访问地址”。

## 产品原则

1. **先单用户，后权限系统**
   第一版只有 Owner PIN，不做用户列表和角色。未来如果需要再扩展 viewer/editor/admin。

2. **默认不破坏当前本机体验**
   新用户默认可不开启 PIN。开启后，所有敏感页面进入锁屏；普通会话按“无操作后锁定”滑动续期，持续使用 Curated 时不会因为固定倒计时到期而突然锁屏。

3. **PIN 不等于高强度密码**
   UI 可以叫 PIN，但实现上应允许 4-8 位数字或更长 passcode。短 PIN 必须配合失败限速、短时间锁定和安全提示。

4. **本机和 LAN 统一会话模型**
   不要前端只用 localStorage 假装锁住。解锁状态应由后端会话判断，API 对未解锁请求返回 `401/423`，否则直接调用 API 仍能绕过。

5. **设备可见性服务于授权管理**
   Connected Clients 不只是“看谁在线”，还可以显示设备是否已解锁、会话剩余时间，并允许撤销某个设备会话。

## 推荐方案

### 方案 A：纯前端 PIN 锁

- PIN hash 和解锁状态放 localStorage。
- 路由守卫挡住页面。
- 优点：实现最快，不改后端。
- 缺点：API 完全没保护；懂一点浏览器的人可以绕过；LAN 访问仍可直接请求 `/api`。

结论：只适合原型，不建议作为 Curated 的正式方向。

### 方案 B：后端 PIN + 会话 cookie

- 后端持久化 PIN hash、salt、锁定设置。
- `POST /api/auth/unlock` 校验 PIN，成功后签发后端会话。
- 前端通过 `GET /api/auth/status` 判断是否解锁。
- 敏感 API 和静态 SPA 入口都通过中间件保护。
- 会话存在服务端，浏览器只持有 HTTP-only cookie；普通会话用 `last_seen_at` + `expires_at` 表示 idle deadline。
- Settings 中可设置“无操作后锁定”时长，例如 15 分钟、1 小时、4 小时；持续键盘、鼠标、滚轮、触摸或窗口聚焦活动会刷新后端会话，真正空闲超过该时长后才进入锁屏。

结论：推荐作为第一版。它能防 casual access，也能给 LAN 场景打底，复杂度仍可控。

### 方案 C：完整账号 / 角色 / Token 权限系统

- 支持多账号、多角色、API token、设备授权、审计日志。
- 优点：面向长期 Server Mode 和多用户共享。
- 缺点：需求面大，马上引入会拖慢当前单用户产品。

结论：不作为第一阶段。等 PIN 锁、连接设备、LAN 开关稳定后再评估。

## MVP 行为

### 开启 PIN

入口放在 Settings -> General 或 Settings -> Security：

- 开启“Require PIN to open Curated”。
- 首次设置需要输入 PIN + 二次确认。
- 推荐默认 PIN 长度 6 位，允许 4-8 位数字；也可以保留“使用更长 passcode”的后续扩展。
- 设置“无操作后锁定”：
  - 15 分钟
  - 1 小时
  - 4 小时
- 解锁时可选择“永远信任此设备”（可选，风险更高，默认不推荐）；该设备只需一次认证，直到用户主动锁定或后续会话管理撤销。
- Settings -> Security 页面只保留“启用 PIN”或“修改 PIN”入口按钮；点击后在 Dialog 中输入 PIN 表单。
- 已开启 PIN 后，修改 PIN Dialog 需要输入当前 PIN、新 PIN 和确认 PIN。
- 设置“重新打开应用时要求 PIN”：默认开启。
- 设置“LAN 访问总是要求 PIN”：默认开启。

### 锁屏体验

用户打开 Curated 时：

- 如果未开启 PIN：保持当前体验。
- 如果已开启 PIN 且当前会话未解锁：显示全屏锁屏页。
- 锁屏页只显示 Curated 标识、PIN 输入框、错误提示和基础帮助，不显示影片海报、标题、最近播放等敏感信息。
- 输入成功后回到原目标路由。
- 输入失败：
  - 立即提示“PIN 不正确”。
  - 连续失败 5 次后锁定 1 分钟。
  - 后续失败可指数退避或固定 5 分钟冷却。
- 锁定时仍允许退出应用、打开帮助，但不允许进入库。

### 自动锁定

第一版可以支持三种触发：

- 普通会话在用户无键盘、鼠标、滚轮、触摸或窗口聚焦活动超过 idle lock delay 后重新进入锁屏；用户正在观看或操作 Curated 时不会按固定倒计时强制锁屏。
- 用户从头像/菜单/托盘选择“Lock Curated”立即锁定。
- Electron 窗口隐藏到托盘后是否锁定：作为设置项，默认可选“超过 N 分钟后锁定”。

### API 保护边界

第一版不应该只锁路由。后端需要保护至少这些敏感路径：

- `/api/library/*`
- `/api/curated-frames*`
- `/api/playback/*`
- `/api/import/*`
- `/api/scans*`
- `/api/tasks/*` 中会泄露影片名或路径的任务详情
- `/api/settings` 中敏感配置，尤其库路径、代理、日志目录、数据库路径

可以保持少量开放路径：

- `GET /api/health`：返回最小健康信息，不泄露 databasePath 或连接设备细节，除非已解锁。
- `GET /api/auth/status`
- `POST /api/auth/unlock`
- 静态资源：允许加载锁屏所需 JS/CSS/icon，但 SPA 渲染后由 auth guard 决定显示锁屏。

### PIN 存储

- 不存明文 PIN。
- 存 `pinHash`、`salt`、`kdf`、`pinLength`、`createdAt`、`updatedAt`；`pinLength` 只用于锁屏显示正确数量的输入位，不存明文 PIN。
- Go 侧建议使用 `argon2id` 或 `bcrypt` 这类带成本参数的 KDF；短 PIN 仍需要失败限速，因为 4-6 位 PIN 熵很低。
- 设置写入位置可以先放 `config/library-config.cfg`，但长期更适合 SQLite 的 `app_security_settings` 表，便于会话、失败计数和设备撤销一起管理。

### 会话模型

推荐后端维护会话：

```text
auth_session_id
client_key
created_at
last_seen_at
expires_at
trusted_until
revoked_at
```

`expires_at` 对普通会话表示 idle deadline，并在有效会话被后端验证时按 `session_ttl_minutes` 滑动延长。`trustedForever` 会话不设置 `expires_at`，使用长期 HTTP-only cookie。前端拿不到原始会话密钥；浏览器通过 HTTP-only cookie 自动携带。若需要兼容非浏览器 API，可以未来增加显式 Bearer token，但第一版不必做。

## 与 Connected Clients 的结合

Connected Clients 的字段可以扩展为：

| 字段 | 含义 |
|---|---|
| `isUnlocked` | 该设备当前是否有有效解锁会话 |
| `sessionExpiresAt` | 普通会话的 idle deadline，用户活动会滑动延长 |
| `lastUnlockAt` | 最近一次解锁时间 |
| `failedUnlockCount` | 最近失败次数，谨慎展示 |
| `canRevokeSession` | 是否可从 Settings 撤销该设备 |

Settings -> Overview 或 Settings -> Security 中可以显示：

- 本机 Chrome：Unlocked，idle locks after 54 min
- iPhone Safari：Locked，last seen 12 min ago
- Curated Desktop：Unlocked，trusted forever

并提供操作：

- Revoke this device
- Revoke all other devices
- Require PIN now

这会让“设备可见性”从观察工具变成安全管理入口。

## LAN 访问控制的自然扩展

PIN 锁可以和 LAN 开关分阶段组合：

1. **App Lock**：所有客户端访问 Curated 都需要 PIN 解锁。
2. **LAN visibility**：Settings 能看到哪些设备访问过。
3. **LAN access policy**：
   - Localhost only
   - LAN allowed, PIN required
   - LAN allowed, trusted devices can stay unlocked
4. **Device trust**：某个设备可设置为信任 N 天；支持撤销。
5. **Server Mode**：只有在上述基础上，再考虑 Windows Service 或长期 LAN 服务。

## 实施切片建议

### Slice 1：PIN 需求与锁屏原型

- 新增设计文档和 i18n 文案。
- 新建 Lock Screen 组件和路由守卫原型。
- 不保护 API，只验证 UX。
- 风险：不能作为安全完成态，只能作为交互探索。

### Slice 2：后端 auth 状态与 PIN 设置

- 新增安全设置 DTO：
  - `pinEnabled`
  - `pinLength`
  - `sessionTtlMinutes`
  - `lockOnRestart`
  - `lockOnWindowClose`
  - `lanRequiresPin`
- 新增接口：
  - `GET /api/auth/status`
  - `POST /api/auth/setup-pin`
  - `POST /api/auth/unlock`
  - `POST /api/auth/lock`
  - `POST /api/auth/change-pin`
- Settings -> Security 接入设置 PIN、修改 PIN、无操作后锁定时长；设置 PIN 和修改 PIN 均通过入口按钮打开 Dialog。关闭 PIN 作为后续切片。

### Slice 3：敏感 API 中间件保护

- 为 `/api/library/*`、播放、导入、扫描、设置等敏感 API 加中间件。
- 未解锁返回统一错误：
  - HTTP `401 Unauthorized` 或 `423 Locked`
  - error code `AUTH_LOCKED`
- 前端 http-client 捕获后跳转锁屏，并保留原目标路由。

### Slice 4：Connected Clients + 会话管理

- 在 connected clients tracker 中关联 auth session。
- 展示锁定/解锁状态、过期时间。
- 支持撤销当前设备以外的会话。

### Slice 5：LAN policy

- Settings 增加监听范围与 LAN policy。
- 后端区分 loopback 和 non-loopback 请求。
- 对 LAN 请求强制 PIN 或拒绝。
- README/API/feature inventory 同步。

## 需要提前决定的问题

1. PIN 是只允许数字，还是允许 passcode？
   推荐第一版 UI 主打数字 PIN，但后端把它当 secret string，不强依赖纯数字。

2. 未解锁时 `GET /api/health` 是否隐藏 `databasePath`？
   推荐隐藏或拆成 public health / private health，避免锁屏前泄露本地路径。

3. 设置页是否需要解锁后才能进入？
   推荐需要。否则别人可以关掉 PIN 或看库路径。

4. 忘记 PIN 怎么办？
   MVP 可以明确提示：需要本机删除安全设置并重启，相当于管理员级恢复。后续可做 recovery key。

5. 是否默认开启 PIN？
   推荐默认关闭，但首次启动或 Settings Security 里明显提示“保护本地库”。

6. 会话保存在哪里？
   推荐后端 SQLite + HTTP-only cookie；不要只用 localStorage。

## 当前推荐结论

可以开始引入权限管理，但第一阶段不要叫“权限管理系统”，而是叫 **PIN App Lock** 或 **Privacy Lock**。它的目标是保护单用户本地媒体库入口，而不是马上支持多人账号。

推荐顺序：

1. PIN App Lock 设计与锁屏 UX。
2. 后端 PIN hash + session + API lock middleware。
3. Connected Clients 显示设备解锁状态。
4. LAN policy：本机 only / LAN allowed / LAN requires PIN。
5. 再评估多用户、角色、Server Mode。

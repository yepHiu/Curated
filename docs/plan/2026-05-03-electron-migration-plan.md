# Curated Electron Desktop Migration Plan

## 1. 核心思路：Electron 就是一个浏览器壳

当前 Go 后端已经同时承载了两件事：

- **REST API** (`/api/*`) — 所有业务逻辑
- **前端 SPA** (静态文件服务, `webui.WrapHandler`) — Vue 编译产物

后端默认监听 `:8080`(dev) / `:8081`(release)，Go 的 `:PORT` 语法绑定所有网络接口(`0.0.0.0`)，这意味着**远程访问能力已经存在**——局域网内其他设备通过 `http://<主机IP>:8080` 就能打开 Curated。

所以 Electron 不需要"迁移"任何东西。它只需要做一个**带桌面特性的浏览器壳**：

```
┌── Electron Main Process ──────────────────────┐
│                                                │
│  1. 启动 Go 后端子进程 (child_process.spawn)     │
│  2. 等待后端就绪 (poll /api/health)              │
│  3. 打开 BrowserWindow 加载 http://127.0.0.1:8080│
│  4. 提供桌面特性: tray / 单实例 / 自启动 / 窗口管理 │
│                                                │
│  BrowserWindow 里跑的就是现在的 Vue SPA，         │
│  代码一行不改，和浏览器里完全一样。                 │
└────────────────────────────────────────────────┘

┌── 其他设备 (手机/平板/另一台电脑) ───────────────┐
│                                                │
│  浏览器打开 http://192.168.1.x:8080              │
│  同一个后端，同一个前端，同一个体验。              │
└────────────────────────────────────────────────┘
```

### 为什么不需要 IPC 桥接层

之前文档里设想的 preload + IPC bridge 方案，本质上是想用 Electron IPC 替代 HTTP 通信。但当前后端已经有完整的 REST API（约 40 个端点），把所有接口重写为 IPC handler 工作量大、风险高、用户感知不到收益。

两种方案对比：

| | IPC 桥接方案 | 浏览器壳方案 |
|---|---|---|
| 前端改动 | 新建 electron adapter, 重写所有 API 调用 | **零改动** |
| 后端改动 | 扩展 stdio 命令集覆盖全部 40+ API | **零改动**（或仅加少量 stdio 事件） |
| preload 复杂度 | 需要暴露大量 API 到 `window.curated.*` | **不需要 preload 脚本** |
| 远程访问 | 需要单独维护 web 模式 | **天然支持**, 同一个后端 |
| 维护成本 | 三套 adapter (web/electron/mock) | 两套 (web/mock), electron 就是 web |

Electron 里 `BrowserWindow` 加载 `http://localhost:8080`，fetch 请求同源 `/api/*`，和 Chrome 浏览器里跑完全一致。

### 那什么时候需要 IPC

只在 Electron 主进程需要执行**浏览器做不到的事**时，才需要 preload + IPC：

- 原生文件选择对话框 (替代 `<input webkitdirectory>`)
- 原生"在资源管理器里显示"
- 原生播放器进程启动 (`child_process.spawn` 替代 `potplayer://` 协议)
- 窗口标题控制、全屏、最小化到托盘

这些是**锦上添花的增强**，不是 Electron 能跑起来的前提。可以逐步加。

---

## 2. 目标架构

```
┌──────────────────────────────────────────────────┐
│              Electron Main Process                │
│                                                   │
│  后端生命周期管理     Tray (Electron API)            │
│  单实例锁 (Electron)  自启动 (Electron)              │
│  窗口状态记忆         可选的原生交互 (IPC)            │
│                                                   │
│  ┌─────────────────────────────────────────────┐  │
│  │  BrowserWindow                               │  │
│  │  loadURL("http://127.0.0.1:8080")             │  │
│  │                                              │  │
│  │  现有的 Vue SPA — 零改动                       │  │
│  │  fetch("/api/library/movies") 同源请求        │  │
│  │  <video src="/api/library/movies/.../stream"> │  │
│  └─────────────────────────────────────────────┘  │
│                                                   │
│              │ spawn stdio                         │
│              ▼                                     │
│  ┌──────────────────────────────────────┐         │
│  │  Go Backend (child_process)           │         │
│  │  -mode http (监听 :8080)              │         │
│  │  - 服务 REST API + 前端 SPA           │         │
│  │  - SQLite, scanner, scraper, ...      │         │
│  └──────────────────────────────────────┘         │
└──────────────────────────────────────────────────┘

         ┌── 远程访问 ──────────────────┐
         │  浏览器 → http://IP:8080      │
         │  同一个后端，同一个前端        │
         └──────────────────────────────┘
```

---

## 3. 需要做的事情

### 3.1 必须做的（让 Electron 能跑起来）

#### A. 后端侧：微调

**新增 stdio 启动就绪信号**（用于 Electron 知道后端何时准备好）：

后端在 HTTP server 就绪后往 stdout 写一条 JSON：

```json
{"kind":"event","type":"server.listening","payload":{"addr":"127.0.0.1:8080"}}
```

当前代码 `cmd/curated/main.go` 的 `runHTTP()` 启动后没有通知机制。Electron 需要轮询 `/api/health` 或等待这条 stdout 消息。建议两种都支持：Electron 优先读 stdout 的 `server.listening` 事件，fallback 到 HTTP health check 轮询。

改动量：后端约 20 行。

**Windows 下不显示控制台窗口**（Electron 托管运行时）：

Go 后端编译时加 `-ldflags -H=windowsgui` 阻止控制台窗口弹出。开发阶段可以保留控制台便于调试。

改动量：编译参数，无代码改动。

**动态端口支持**（可选，推荐）：

当前端口配置在 JSON config 中的 `httpAddr`。可以让后端在 `httpAddr` 为空或设为 `0` 时随机选端口（`127.0.0.1:0`），然后通过 stdout 告诉 Electron。

改动量：约 30 行。

#### B. Electron 侧：新建文件

新增 `electron/` 目录：

```
electron/
  main.ts          # 主进程入口
  preload.cjs       # 预加载脚本（只放浏览器做不到的窄桌面能力）
  tray.ts           # 托盘管理
```

**`main.ts` 核心逻辑：**

```
1. app.whenReady()
2. 检查单实例 → requestSingleInstanceLock()
3. spawn Go 后端 (child_process.spawn, -mode http)
4. 监听 stdout, 等 "server.listening" 事件拿到端口
5. 或轮询 http://127.0.0.1:{port}/api/health 直到 200
6. new BrowserWindow({ ... })
7. win.loadURL("http://127.0.0.1:{port}")
8. 注册 app.on("before-quit") → 杀后端子进程
```

#### C. 前端侧：几乎不改

当前前端从后端加载（`http://127.0.0.1:8080`），所有 API 调用同源 `/api`，无需改动。

唯一可能需要处理的：Google Fonts (`Outfit`) 和 hls.js 是从 CDN 加载的，离线环境会失败。建议：
- 字体打包到 `dist/` 中, CSS 引用本地路径
- hls.js 改为 npm 依赖 + 动态 import

改动量：前端约 10 行（`index.html` 改字体引用，`PlayerPage.vue` 改 hls.js 加载方式）。

#### D. 构建侧：

- `package.json` 加 `electron` 和 `electron-builder` 为 devDependencies
- 新增 `pnpm dev:electron` 脚本（编译 main.ts + 启动 electron）
- 新增 `pnpm build:electron` 脚本（全量打包）

### 3.2 锦上添花的（Phase 2+）

以下都是可选的增强，不影响 Electron 的基本运行：

| 增强项 | 说明 | 优先级 |
|--------|------|--------|
| Tray 托盘 | 用 Electron `Tray` API 替代 Go Win32 tray，跨平台 | 高 |
| 原生文件对话框 | `dialog.showOpenDialog()` 替代 `<input webkitdirectory>` | 中 |
| 原生资源管理器打开 | `shell.showItemInFolder()` 替代 `POST /api/.../reveal` | 中 |
| 原生播放器启动 | `child_process.spawn(mpv/PotPlayer)` 替代 `potplayer://` 协议 | 中 |
| 关闭到托盘 | 点×最小化到托盘而不是退出 | 中 |
| 窗口状态记忆 | 记住上次窗口位置/大小/最大化状态 | 低 |
| 自启动 | `app.setLoginItemSettings()` 跨平台 | 低 |
| 自动更新 | `electron-updater` + GitHub Releases | 低 |
| 本地字体 & hls.js | 去 CDN 化，完全离线可用 | 低 |
| 自定义标题栏 | frameless window + 自绘标题栏 | 低 |

### 3.3 可以退休的 Go 代码（Phase 2 完成后）

当 Electron 的 tray / 单实例 / 自启动都就绪后，以下 Go 原生 Win32 实现可以删除：

| 文件 | 替代方案 |
|------|---------|
| `backend/internal/desktop/tray_windows.go` | Electron `Tray` |
| `backend/internal/desktop/single_instance_windows.go` | `app.requestSingleInstanceLock()` |
| `backend/internal/desktop/autostart_windows.go` | `app.setLoginItemSettings()` |
| `backend/internal/desktop/tray_stub.go` | 不再需要 |

---

## 4. 远程访问方案

当前后端监听 `:8080`/`:8081`（所有网络接口），前端使用同源 `/api` 路径。远程访问已经可以工作：

```
主机 (192.168.1.100):
  curated.exe -mode http   # 监听 :8080

局域网其他设备:
  浏览器打开 http://192.168.1.100:8080
  → Go 后端返回 SPA (index.html + JS/CSS)
  → SPA 里 fetch("/api/library/movies")
  → 同源请求 http://192.168.1.100:8080/api/library/movies
  → 一切正常工作
```

### 需要确认的点

- **防火墙**：需要允许入站 TCP 连接到后端端口
- **CORS**：同源请求不涉及 CORS，无需额外配置
- **安全**：当前后端没有鉴权机制，局域网内任何人可以访问。如果需要限制，后续可以加 token 认证或 IP 白名单
- **视频流**：`GET /api/library/movies/{id}/stream` 返回 Range 请求流，同样同源，远程播放可用

### 与 Electron 桌面程序的关系

Electron 桌面程序和远程 Web 访问**共享同一个后端实例**：

- Electron 打开时，后端启动，桌面程序可用，远程也可用
- 用户关闭 Electron 窗口时，可以选"退到托盘"（后端继续运行，远程继续可用）或"完全退出"（后端停止）
- 也可以让后端作为 Windows Service / 守护进程独立运行，Electron 只是其中一个客户端

---

## 5. 分阶段实施计划

### Phase 0：最小可用 Electron（1-2 周）

**目标**：Electron 窗口打开，里面跑着和浏览器里一模一样的 Curated。

```
具体步骤:
1. 装 electron 依赖
2. 写 electron/main.ts: spawn 后端, 等 health check, 开窗口, loadURL
3. 写 electron/preload.cjs: 初期可为空壳；当前已补原生目录选择
4. 后端加 server.listening stdout 事件
5. pnpm dev:electron 启动脚本
6. 验证: 库浏览、详情、播放、设置、扫描全部正常
```

代码改动量估算：
- 新增 `electron/` 目录，约 150 行 TypeScript
- 后端 `server.go` / `main.go` 少量修改，约 30 行 Go
- `package.json` 加依赖和脚本
- 前端：零改动

### Phase 1：桌面增强（2-3 周）

```
1. Tray 托盘 (Electron Tray API)
2. 单实例锁 (app.requestSingleInstanceLock)
3. 关闭到托盘
4. 自启动 (app.setLoginItemSettings)
5. 原生文件对话框
6. 原生"在资源管理器显示"
```

### Phase 2：播放器增强 + 去 CDN（1-2 周）

```
1. hls.js 本地化 (npm 依赖)
2. 字体本地化
3. 原生播放器启动 (child_process.spawn)
4. 离线完全可用
```

### Phase 3：打包发布（2-3 周）

```
1. electron-builder 配置 (Windows/macOS/Linux)
2. 代码签名
3. 自动更新 (electron-updater)
4. 便携版 + 安装版
```

---

## 6. 需要提前决策的事项

| 决策 | 选项 | 建议 |
|------|------|------|
| 后端启动方式 | (A) Electron spawn, (B) 用户手动启动, (C) Windows Service | **(A) 为主, 支持 (B)** — Electron 默认管理后端生命周期，但也允许用户先手动启动后端再用 Electron 连接 |
| 端口策略 | (A) 固定 :8080, (B) 动态随机端口 | **(B) 动态端口** — 避免多实例端口冲突 |
| Tray 行为 | (A) 关窗口=退托盘, (B) 关窗口=退出 | **(A) 默认退托盘**, 托盘菜单里可以真正退出 |
| 前端加载方式 | (A) `loadURL("http://127.0.0.1:PORT")`, (B) `loadFile("dist/index.html")` | **(A)** — 这样前端代码零改动，fetch 同源 `/api` 天然可用 |
| Windows 控制台窗口 | (A) 隐藏, (B) 显示 (调试用) | 发布版 **(A)**, 开发版 **(B)** 可选 |

---

## 7. 与当前 Go Win32 Tray 的关系

**Phase 0-1 并行策略**：

- 当前的 Go Win32 tray (`desktop/tray_windows.go`) 继续在非 Electron 场景使用
- 用户在 `curated.exe` 不带 Electron 启动时，仍然走原生 Win32 tray
- Electron 模式下，tray 由 Electron 接管
- Go 后端不知道也不关心自己是被浏览器打开还是被 Electron 打开——它就是一个 HTTP 服务器

两种部署模式可以共存：
```
模式 A: curated.exe -mode tray          → Go Win32 tray + 系统默认浏览器
模式 B: curated.exe -mode http          → Electron 连接 + Electron tray
模式 C: curated.exe -mode http          → 局域网其他设备浏览器访问
```

---

## 8. 总结

原计划把 Electron 当作一个需要深度 IPC 集成的"平台迁移"。但实际上 Electron 就是一个**带 Chromium 的壳**，当前架构已经非常适合这个模型：

- **Go 后端 = HTTP 服务器 + 静态文件服务** (已经完全实现)
- **Electron = 浏览器窗口 + 桌面特性** (需要新增)
- **远程访问 = 浏览器打开 `http://IP:PORT`** (已经可以)
- **前端代码 = 零改动** (最大优势)

核心工作量集中在写 Electron 主进程的启动逻辑（约 150 行 TS），而不是改造现有系统。

---

## 9. 2026-05-13 实施修订：当前仓库内的最小桌面壳层

本轮决策确认：Electron 桌面壳层直接落在当前仓库，而不是新建独立仓库。原因是 Electron 发行形态必须共享同一份 Vue 前端、Go 后端、品牌图标、版本号、FFmpeg runtime、release 台账和更新链路；新仓库会立刻引入跨仓产物同步与发布编排成本。

### 已实施的 MVP 边界

- 新增 `electron/`，包含 Electron main process、窄 preload、后端子进程生命周期管理与单实例窗口聚焦。
- 新增 `pnpm dev:electron`、`pnpm build:electron:main`、`pnpm build:electron` 与 `pnpm test:electron`。
- Electron 默认加载本机 Web UI：开发态会先启动或复用 Go 后端 `http://127.0.0.1:8080`，再启动或复用 Vite 前端 `http://127.0.0.1:5173` 并让 BrowserWindow 加载 `5173`；打包态加载后端静态托管 `http://127.0.0.1:8081`。可用 `CURATED_ELECTRON_BACKEND_URL` 覆盖后端地址，用 `CURATED_ELECTRON_FRONTEND_URL` 覆盖开发态前端地址。
- Electron 启动 Go 后端时使用 `-mode http`，并设置 `CURATED_HOSTED_BY=electron`，避免 Electron MVP 与 Go tray owner 混淆。
- Go HTTP server 改为先 `net.Listen` 成功后再报告 ready，避免日志显示 listening 但端口实际 bind 失败的竞态。
- `electron-dist/` 是 TypeScript 编译产物，已作为本地构建输出忽略，不纳入源码跟踪。

### 仍然坚持不做的事

- 不把 REST API 搬到 IPC；preload 只放浏览器做不到的桌面小能力。
- 不新增 Electron 专用业务 adapter。
- 不让 Vue 业务组件直接依赖 Electron。
- 不在 MVP 中替换现有 Go tray / autostart / release installer 链路。
- 不把动态端口作为唯一策略；局域网共享仍继续依赖稳定的后端配置端口。

### 后续优化优先级

1. 窗口状态记忆。
2. 资源管理器 reveal、原生播放器启动等小范围 preload API。
3. 字体与 hls.js 完全本地化，确保离线桌面体验。
4. 将 Electron 打包接入现有 `scripts/release/release_cli.py`，并与现有 installer/update 台账统一。

---

## 10. 2026-05-14 实施补充：图标与原生目录选择

本轮补齐了桌面壳层中最早应该使用 Electron 原生能力的两个点：

- **应用图标**：Electron 主窗口优先使用 `backend/internal/assets/curated.ico`，回退到 `public/Curated-icon.png` / `icon/curated-icon-rg-dark-pink.png`；Windows 下设置 `app.setAppUserModelId("com.curated.desktop")`，避免任务栏标识继续表现为默认 Electron。
- **原生目录选择**：新增 `electron/preload.cjs`，只暴露 `window.javLibrary.pickDirectory()`；主进程通过 `dialog.showOpenDialog({ properties: ["openDirectory"] })` 处理 `curated:pick-directory` IPC，并返回 `{ path } | null`。
- **前端接入方式**：不改设置页组件。现有 `src/lib/pick-directory.ts` 已经优先调用 `window.javLibrary.pickDirectory()`，因此「影片存储」添加路径与日志目录选择会在 Electron 中自动走原生目录对话框；普通浏览器仍按 `showDirectoryPicker()` / `webkitdirectory` 降级。
- **边界仍保持不变**：业务 API 继续走同源 REST，不引入 Electron 专用业务 adapter；preload 不暴露任意文件系统、数据库或后端业务能力。
- **构建细节**：sandbox preload 不拆本地模块，也不使用 ESM import；`pnpm build:electron:main` 会编译 TypeScript main process，并把独立的 `electron/preload.cjs` 复制到 `electron-dist/preload.cjs`。

对应测试：

- `electron/desktop-shell.test.ts`：覆盖图标解析、目录对话框结果归一化与 IPC channel 命名。
- `electron/preload.test.ts`：用 VM 和伪 `electron` 模块验证 preload 只暴露 `javLibrary.pickDirectory()`，且调用 `curated:pick-directory`。
- `src/lib/pick-directory.test.ts`：补充桌面 bridge 成功/取消场景，确认 Electron bridge 优先于浏览器目录 picker。

---

## 11. 2026-05-14 架构建议：托盘常驻优先，Windows Service 作为后续 Server Mode

### 结论

建议下一步不要直接把生产桌面端默认改成 Windows Service。更稳妥的顺序是：

1. **先做 Electron 托盘常驻**：用户点窗口关闭按钮时隐藏窗口，不退出 Electron app，也不停止 Electron 管理的 Go 后端；托盘菜单提供“打开 Curated”“在浏览器打开 Web 端”“打开设置”“打开日志目录”“退出 Curated”。
2. **保留现有 attach 能力**：如果 `http://127.0.0.1:8081/api/health` 已经是 Curated 后端，Electron 只连接它，不拥有它的生命周期。当前 `electron/backend-process.ts` 已具备这个基础：健康检查通过时返回 `attachedToExisting: true`，`stop()` 是 no-op。
3. **把 Windows Service 做成可选 Server Mode**：只有当用户明确需要“不开桌面 UI、开机即服务、甚至用户未登录也继续提供 Web/LAN 访问”时，再安装服务。它不应成为默认桌面体验。

### 为什么不建议默认 Windows Service

Windows Service 的价值是“机器级常驻”，但它会把桌面媒体库的复杂度拉高：

- **权限与用户身份**：服务通常运行在 LocalSystem、NetworkService 或指定服务账号下，不等同于当前登录用户。用户目录、外接盘、网络映射盘、SMB 凭据、`LOCALAPPDATA`、注册表 `HKCU` 行为都可能和桌面进程不同。
- **Session 0 隔离**：服务不能可靠地和用户桌面交互。当前后端已有“打开资源管理器”“打开浏览器”“打开日志目录”等 shell 行为，放进服务后可能打开失败、打开到不可见 session，或需要改成由 Electron 代执行。
- **安装和更新成本**：安装/卸载服务通常需要管理员权限；更新时要 stop service、替换二进制、迁移配置、start service，并处理文件锁和失败回滚。
- **多用户模型**：一台 Windows 机器上多个用户登录时，是一套机器级库、每用户一套库，还是每用户一个服务，必须提前定义。否则数据库、配置、端口和资料库路径会互相踩。
- **安全边界**：服务长期监听端口，尤其继续支持局域网访问时，必须补 token/访问控制、防火墙规则和“只监听本机 / 允许 LAN”的明确设置。当前后端没有完整鉴权，不适合默认常驻暴露。
- **路径可见性**：Windows 映射盘符是 per-user/session 的，服务账号通常看不到用户在资源管理器里映射的盘符。媒体库很容易踩到这个问题。

### 推荐的默认桌面行为

默认产品行为应更接近常见桌面应用：

```
启动 Curated Desktop
  -> Electron 启动或复用 Go HTTP 后端
  -> 打开 BrowserWindow
  -> 用户点 X
       -> 隐藏窗口，Electron 留在托盘，后端继续运行
       -> http://127.0.0.1:8081/ 和 LAN Web 入口继续可用
  -> 用户从托盘点“打开 Curated”
       -> 重新 show/focus BrowserWindow
  -> 用户从托盘点“退出 Curated”
       -> 退出 Electron
       -> 如果后端是 Electron 本次 spawn 的，则停止后端
       -> 如果后端是已有进程/服务 attach 的，则不停止后端
```

这能满足“关窗口不等于退出应用”和“桌面窗口关了 Web 端仍可访问”的主要诉求，而且不引入服务权限、安装器和安全模型的复杂度。

### 实施计划

#### Phase A：Electron tray + 关闭到托盘

目标：生产桌面端关闭窗口后仍常驻，Web 端继续可访问。

改动点：

- `electron/main.ts`
  - 增加 `Tray`、`Menu`。
  - 增加 `isQuitting` 标志。
  - 拦截 `mainWindow.on("close")`：当 `isQuitting=false` 时 `event.preventDefault()` + `window.hide()`。
  - 删除或调整当前 `window-all-closed -> app.quit()` 行为，避免隐藏窗口后退出 app。
  - 托盘菜单：
    - `Open Curated`：show/focus 当前窗口；不存在则重新 `createMainWindow()`。
    - `Open in Browser`：`shell.openExternal(baseUrl)`。
    - `Open Settings`：窗口或浏览器打开 `/#/settings`。
    - `Open Logs`：后续可由 Electron `shell.openPath(logDir)` 或继续调用后端 reveal/log endpoint。
    - `Quit Curated`：设置 `isQuitting=true`，destroy window，退出 app。
- `electron/desktop-shell.ts`
  - 纯函数化 tray 菜单 label、窗口关闭策略、owned/attached 后端退出策略，便于单测。
- 测试
  - `electron/*.test.ts` 覆盖：
    - 普通 close 时应 hide 而不是 quit。
    - tray Open 应 show/focus。
    - Quit 时只有 `attachedToExisting=false` 的后端会 stop。

验收：

- `pnpm test:electron`
- `pnpm build:electron:main`
- 手动启动 `pnpm dev:electron`：开发态会同时提供 `http://127.0.0.1:5173/` 前端与 `http://127.0.0.1:8080/api/health` 后端；点 X 后窗口消失但 Electron 进程、后端和 Vite 仍可用；托盘可恢复窗口；托盘退出后 Electron 管理的后端和 Vite 停止。

#### Phase B：明确“连接已有后端”的 UX

目标：Electron 既能作为默认桌面入口，也能连接用户先启动的后端。

改动点：

- 保持 `CURATED_ELECTRON_BACKEND_URL` / `CURATED_BACKEND_URL` 覆盖逻辑。
- 启动时如果目标后端健康：
  - UI/日志标记为 `attachedToExisting=true`。
  - Quit Electron 时不停止该后端。
- 启动时如果目标后端不健康：
  - 默认策略：尝试 spawn bundled backend。
  - 如果用户显式配置了 `CURATED_ELECTRON_BACKEND_URL`，可以选择不 fallback spawn，而是显示“无法连接后端”的错误页/对话框，避免误连另一套数据。

验收：

- 先手动启动 `curated.exe -mode http`，再启动 Electron：Electron attach；退出 Electron 后 HTTP 后端仍可访问。
- 不启动后端直接启动 Electron：Electron spawn；托盘退出后后端停止。

#### Phase C：可选 Windows Service / Server Mode

目标：给“希望 Curated 像家用媒体服务器一样常驻”的用户一个明确安装选项，而不是默认行为。

产品形态：

- 安装器提供可选项：`Install Curated Server Service`。
- 设置页或单独管理工具提供：
  - 服务状态：Running / Stopped / Not installed。
  - Start / Stop / Restart。
  - 监听范围：本机 only / LAN。
  - 服务账号说明：当前用户账号 / LocalSystem 的差异。
  - 安全提示：LAN 模式需要访问 token 或至少防火墙确认。

后端要求：

- 服务模式必须禁用直接桌面交互：不由服务打开浏览器、资源管理器、文件夹对话框。
- 需要新增“桌面代理能力”边界：Electron 负责原生 UI/shell 能力，Go service 只做 HTTP/API/扫描/任务。
- 配置和数据库路径要固定到服务可读写的位置，例如 `ProgramData/Curated` 或明确的用户目录，不能含糊依赖当前工作目录。
- 必须补访问控制后再默认允许 LAN。

验收：

- 未登录桌面也能启动服务并响应 `/api/health`。
- 普通浏览器可访问服务 Web UI。
- Electron 能 attach 到 service，并在退出时不停止 service。
- 服务账号能访问用户配置的库路径；无法访问时设置页给出明确诊断，而不是扫描失败后只报 generic error。

---

## 12. 2026-05-14 实施补充：Phase A 托盘常驻

本轮实现了上节建议中的 Phase A：

- **Electron 托盘**：`electron/main.ts` 使用 Curated 图标创建 `Tray`，tooltip 为 `Curated - local media library`。
- **关闭到托盘**：用户点击窗口关闭按钮时，默认 `event.preventDefault()` 并隐藏窗口；Electron app、其管理的 Go HTTP 后端和开发态 Vite 前端继续运行，所以开发态 `http://127.0.0.1:5173/` / 后端 `http://127.0.0.1:8080/api/health` / 打包态 `http://127.0.0.1:8081/` 仍可访问。
- **托盘菜单**：菜单由 `electron/desktop-shell.ts` 的 `buildTrayMenuModel()` 生成，再在 main process 映射到 Electron `Menu`。当前菜单结构：
  - `Curated`（禁用标题）
  - `Desktop service running` 或 `Connected to existing backend`（禁用状态）
  - `Open Curated`
  - `Open Web App in Browser`
  - `Open Settings`
  - `Quit Curated`
- **真正退出**：只有托盘 `Quit Curated` 或系统级 quit 才退出应用。退出时通过 `shouldStopBackendOnQuit()` / `shouldStopFrontendOnQuit()` 判断进程归属：Electron 本次 spawn 的后端和开发态 Vite 会停止；attach 到已有后端或已有前端时不停止它们。
- **二次启动**：`second-instance` 现在统一调用窗口恢复逻辑，隐藏到托盘的窗口会重新 show/focus。

对应测试：

- `electron/desktop-shell.test.ts` 覆盖托盘菜单模型、已有后端状态文案、窗口关闭隐藏策略、退出时后端 stop 策略。
- `electron/frontend-process.test.ts` 覆盖开发态 Vite URL、启动命令、`VITE_USE_WEB_API=true` / `VITE_API_BASE_URL` 注入，以及退出时前端 stop 策略。
- `pnpm test:electron` 与 `pnpm build:electron:main` 是本阶段最小验证命令。

后续仍未做：

- 窗口大小/位置/最大化状态记忆。
- 托盘菜单里的“打开日志目录”（需要确定 Electron 端日志目录来源，或通过后端设置/接口暴露）。
- Windows Service / Server Mode；仍建议作为后续高级模式，而不是默认桌面行为。

---

## 13. 2026-05-14 实施补充：Electron 接入 release 打包链

本轮把默认生产桌面入口从 Go tray 壳推进到 Electron 壳：

- `pnpm release:publish` 现在按顺序构建 Vue 前端、release Go 后端、Electron main process，再组装发布目录、便携包、安装器和 manifest。
- 新增 release CLI 子命令 `build-electron-main`，对应包脚本 `pnpm release:electron-main`，用于单独生成 `electron-dist/`。
- `assemble_release()` 会复制 `node_modules/electron/dist` 到 `release/Curated`，把 `electron.exe` 重命名为顶层 `Curated.exe`，并把应用入口写入 `resources/app/package.json`。
- Go release 后端不再作为安装包顶层入口，而是打包为 `resources/app/curated.exe`。Electron 启动它时仍使用 `-mode http` 和 `CURATED_HOSTED_BY=electron`，因此后端生命周期默认由 Electron 持有。
- 生产前端产物放在 `resources/app/frontend-dist`，Electron 加载 `http://127.0.0.1:8081`；因为后端工作目录是 `resources/app`，现有静态前端查找逻辑可以直接发现 `frontend-dist`。
- FFmpeg runtime 放在 `resources/app/third_party/ffmpeg/bin`，与后端工作目录保持一致，避免打包后 HLS/remux 流程找不到工具。
- Curated 图标同时放在顶层 `curated.ico` 和 `resources/app/curated.ico`。安装器图标、快捷方式图标、Electron 主窗口与托盘都使用 Curated 品牌图标。
- Inno Setup 模板现在把开始菜单/桌面快捷方式、安装完成后的启动入口都指向 `{app}\Curated.exe`，即 Electron 桌面壳。

生产态启动/关闭关系更新为：

```text
安装后启动 Curated.exe
  -> Electron main process 启动
  -> 检查 http://127.0.0.1:8081/api/health
       -> 若已有 Curated 后端健康，则 attach，不拥有其生命周期
       -> 否则启动 resources/app/curated.exe -mode http
  -> Electron BrowserWindow 加载 http://127.0.0.1:8081
  -> 用户关闭窗口
       -> 隐藏到 Electron tray，Electron 进程和其 spawn 的后端继续运行
       -> Web 端 http://127.0.0.1:8081 继续可访问
  -> 用户从 tray 选择 Quit Curated
       -> 退出 Electron
       -> 若后端是本次 Electron spawn 的，则停止该后端
       -> 若后端是 attach 到已有进程，则不停止该后端
```

这意味着默认桌面体验已经可以交给 Electron 壳管理；Go Win32 tray 仍保留给直接运行 `curated.exe -mode tray` 或未来非 Electron 场景，不再是安装包的默认用户入口。

对应测试：

- `scripts/release/tests/test_build_steps.py` 覆盖 Electron runtime staging、`Curated.exe` 入口、`resources/app/curated.exe` 后端、`resources/app/frontend-dist`、`resources/app/electron-dist`、`resources/app/package.json`、FFmpeg runtime 位置，以及 `publish_release()` 的构建顺序。
- `electron/backend-process.test.ts` 覆盖 packaged app path 下优先启动 `resources/app/curated.exe`。
- `electron/desktop-shell.test.ts` 覆盖 packaged app path 下优先使用 `resources/app/curated.ico`。

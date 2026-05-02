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
  preload.ts        # 预加载脚本（初始可为空壳）
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
3. 写 electron/preload.ts: 空壳
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

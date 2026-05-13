<p align="center">
  <img src="icon/curated-title-nobg.png" alt="Curated" width="520" />
</p>

<p align="center">
  <a href="README.md">English</a> | 简体中文 | <a href="README.ja-JP.md">日本語</a>
</p>

<p align="center">
  <img alt="Vue 3" src="https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vuedotjs&logoColor=white">
  <img alt="TypeScript 5.x" src="https://img.shields.io/badge/TypeScript-5.x-3178c6?style=flat-square&logo=typescript&logoColor=white">
  <img alt="Vite 8.x" src="https://img.shields.io/badge/Vite-8.x-646cff?style=flat-square&logo=vite&logoColor=white">
  <img alt="Go 1.25+" src="https://img.shields.io/badge/Go-1.25+-00add8?style=flat-square&logo=go&logoColor=white">
  <img alt="SQLite modernc" src="https://img.shields.io/badge/SQLite-modernc-003b57?style=flat-square&logo=sqlite&logoColor=white">
  <img alt="Tailwind CSS v4" src="https://img.shields.io/badge/Tailwind_CSS-v4-06b6d4?style=flat-square&logo=tailwindcss&logoColor=white">
  <img alt="shadcn-vue" src="https://img.shields.io/badge/shadcn--vue-ui-111111?style=flat-square">
  <img alt="Windows tray ready" src="https://img.shields.io/badge/Windows-tray_ready-0078d4?style=flat-square&logo=windows&logoColor=white">
</p>

# Curated

Curated 是一个本地优先的媒体资料库应用，采用 Vue 3 前端与 Go + SQLite 后端。当前仓库已经具备以 Web 为先的架构、面向 Windows 的发布打包流程、托盘模式运行、Electron 桌面壳层 MVP、元数据刮削、播放链路、萃取帧管理、手柄控制以及完整的设置系统。

完整功能清单请参见 [docs/features/2026-05-03-feature-inventory.md](docs/features/2026-05-03-feature-inventory.md)。

产品正式名称是 **Curated**。仓库目录和 npm 包名仍可能使用 **`jav-shadcn`**。Go 模块名为 **`curated-backend`**，服务端入口位于 **`backend/cmd/curated`**。

## 项目亮点

- **本地优先** — Vue 3 SPA 前端 + Go HTTP API 后端 + SQLite 持久化。
- **双模式开发** — 真实 API 模式（全链路后端）与 Mock 模式（快速 UI 迭代）共用同一服务层。
- **完善的资料库管理** — 虚拟化海报网格、收藏、评分、标签、演员资料、回收站/恢复、影片笔记，以及支持 fsnotify 自动扫描的多目录资料库。
- **影片导入** — 支持拖拽、文件选择或文件夹选择导入，带进度跟踪与大文件断点续传。
- **存储在线检测** — 优先支持 Windows 外置硬盘场景，针对已配置库路径检测硬盘离线或卷身份变化，并通过启动提醒、通知中心、扫描/导入阻断与手动重绑降低误操作风险。
- **元数据刮削** — 多数据源支持，可配置策略（自动全局 / 国内友好 / 自定义链路 / 指定源），数据源健康检查，机器可读的故障分类。
- **播放能力** — HTML5 视频播放（Range 流）、续播进度、每日观看统计、HLS 会话（remux/转码管线）、外部播放器接力、播放会话诊断。
- **首页每日推荐** — 基于 UTC 日的 hero 轮播与推荐栏，通过 SQLite 持久化保证跨设备一致，含加权采样、冷却窗口与演员/厂牌均衡。
- **萃取帧** — 帧截图、浏览、打标、筛选与多格式导出（JPG/WebP/PNG），导出的图片包含嵌入式元数据。
- **演员管理** — 演员浏览、资料详情、用户标签、外部链接、同源头像缓存与异步元数据刮削。
- **手柄控制** — 基于 Web Gamepad API 的标准手柄支持（含 DualSense）：全局焦点导航、资料库网格选择、播放器控制。
- **Windows 发布打包** — 以 Electron 桌面程序作为安装入口，包含 Inno Setup 安装器、便携包、FFmpeg 集成、发布清单、开机自启，以及基于 GitHub Releases 的更新检查与安装器直接下载。
- **Electron 桌面壳层** — 当前仓库内的 Electron 主进程会启动或复用 Go HTTP 后端，并在开发态启动或复用 Vite 前端；使用 Curated 图标与托盘加载现有 Web UI；关闭窗口会隐藏到托盘，并只暴露原生目录选择这一类窄 preload 能力，不把业务 REST API 搬到 IPC。生产安装包中的 `Curated.exe` 是 Electron 壳，Go 后端位于 `resources/app/curated.exe`。
- **设置与配置** — 完整的设置界面（概览、常规、影片存储、元数据、网络、萃取帧、关于、维护），支持资料库级配置持久化、代理与日志控制。

## 快速开始

### 环境要求

- **Node.js**：与 Vite 8 兼容的当前 LTS 版本
- **pnpm**：仓库使用 `pnpm-lock.yaml`
- **Go**：`1.25.4+`

### 启动后端

```bash
cd backend
go run ./cmd/curated
```

开发默认值：

- HTTP 地址：`:8080`
- 健康名：`curated-dev`

Windows 开发辅助命令：

```bash
pnpm backend:build:dev
```

该命令会生成 `backend/runtime/curated-dev.exe`。

### 启动前端

```bash
pnpm install
pnpm dev
```

Vite 开发服务器通常运行在 `http://localhost:5173`。

### 启动 Electron 壳层 MVP

```powershell
pnpm dev:electron
```

该命令会构建 `backend/runtime/curated-dev.exe` 与 `electron-dist/`，用 `-mode http` 启动或复用 Go 后端，等待 `/api/health` 后再启动或复用 `http://127.0.0.1:5173` 的 Vite 前端，并在带 Curated 图标的 BrowserWindow 中打开该前端地址。Electron 启动的 Vite 会带上 `VITE_USE_WEB_API=true`，并把 API 指向 Electron 管理的后端；打包态安装后的 `Curated.exe` 是 Electron 壳，内置 Go 后端位于 `resources/app/curated.exe`，Electron 会加载 `http://127.0.0.1:8081` 上由后端托管的静态 UI。关闭窗口会隐藏到托盘，后端与 Web 入口继续运行；托盘菜单可重新打开 Curated、在浏览器打开 Web 端、打开 Settings 或真正退出应用。业务接口仍走 HTTP；preload 仅暴露 `window.javLibrary.pickDirectory()`，让设置页等现有目录选择流程调用 Electron 原生目录对话框。

### 真实 API 与 Mock 模式

- 在仓库根目录 `.env` 中设置 `VITE_USE_WEB_API=true`，即可启用真实后端 API。
- 其他值会保持为 Mock 模式。
- 本机 loopback 的 Web API 开发态会直连 `http://127.0.0.1:8080`；Vite 的 `/api` 代理仍作为 fallback 和非 loopback 开发路径保留。

## 功能

### 资料库

- 面向大规模资料库的虚拟化海报网格浏览，选中状态基于 URL。
- 虚拟化海报网格支持标准手柄导航。
- 收藏、评分（0-5）、用户标签与元数据标签。
- 多目录资料库管理：添加、编辑、删除、在文件管理器中打开。
- 入库整理（`organizeLibrary` 设置）与回收站/恢复流程。
- 影片笔记/备注持久化。
- 按演员浏览时显示演员资料卡片。
- 支持按关键词、演员或标签搜索。

### 扫描与元数据

- 手动与自动扫描，后台任务跟踪。
- 基于 fsnotify 的目录监听与防抖自动扫描（`autoLibraryWatch`）。
- 通过 metatube-sdk-go 刮削影片元数据，异步任务执行。
- 多数据源支持，可配置策略：`auto-global`、`auto-cn-friendly`、`custom-chain`、`specified`。
- 数据源健康检查（单个/全部），带故障分类。
- 影片刮削成功后自动补刮演员资料（`autoActorProfileScrape`）。

### 导入

- 顶栏影片导入：支持拖拽、文件选择或文件夹选择。
- 进度跟踪，含文件级状态与失败通知。
- 大文件分块断点续传，支持提交/取消生命周期。
- 冲突检测（不覆盖已存在的目标文件）。
- 可配置默认导入目标资料库路径。
- 当默认导入目标硬盘离线或不再匹配绑定卷时，导入会被阻断并显示存储提醒。

### 播放

- HTML5 视频播放，支持 HTTP Range 流。
- 续播进度持久化（Web API 模式存 SQLite，Mock 模式存 localStorage）。
- 播放描述符层：统一直播、remux 与转码路径。
- HLS 会话支持，含会话诊断与最近会话列表。
- 外部播放器接力，基于可配置的浏览器协议模板（PotPlayer 预设）。
- 每日观看统计（设置 → 概览，91 天窗口）。
- 播放统计叠加层、时间轴缩略图预览与萃取帧截图。
- 路由导航上下文：时间戳（`?t=`）与返回路径（`?from=history`）。
- 播放中侧栏返回。

### 演员

- 演员浏览：支持搜索、标签筛选、排序与分页。
- 演员资料详情与元数据展示。
- 用户标签编辑与外部链接管理。
- 同源头像交付（后端缓存）。
- 演员元数据异步刮削。

### 萃取帧

- 播放中截取帧。
- 浏览：分页、文本搜索、按标签/演员/影片筛选。
- 标签编辑与帧删除。
- 统计概览、标签分类与演员分类。
- 导出为 JPG（EXIF）、WebP（EXIF）、PNG（iTXt）或 ZIP，包含嵌入式元数据（tags、schemaVersion、exportedAt、appName、appVersion）。
- 可配置导出格式偏好（`curatedFrameExportFormat`）。

### 首页与推荐

- 基于 UTC 日的每日推荐快照，SQLite 持久化。
- Hero 轮播与推荐栏，跨设备一致。
- 无放回加权采样，含冷却窗口与推荐次数衰减。
- 演员与厂牌多样性均衡。
- 强制刷新，支持保留 hero 与排除当前推荐。

### 设置与配置

- 完整设置界面：概览、常规、影片存储、元数据、网络、萃取帧、关于、维护。
- 资料库级配置持久化到 `config/library-config.cfg`，原子写入。
- 代理配置，含 JavBus 与 Google 连通性测试。
- 后端日志：可配置目录、保留天数与级别。
- 基于 GitHub Releases 的应用更新检查，含侧栏角标与安装器直接下载。
- Windows 开机自启（`launchAtLogin`）。

### 手柄控制

- 基于 Web Gamepad API 的标准手柄支持，含 DualSense。
- 全局焦点导航、资料库网格选择、播放器控制。
- 大步进退、萃取帧截图与统计/控制层切换。
- 浏览器本地设置开关（localStorage 持久化）。

### 打包发布

- Windows 发布流程：`pnpm release:publish`（Python CLI 编排）。
- 安装后的生产入口：`Curated.exe` 是 Electron 桌面壳；release Go 后端打包为 `resources/app/curated.exe`，由 Electron 拥有生命周期时以 `-mode http` 启动。
- 托盘常驻运行，本地前端托管于 `:8081`。
- Inno Setup 安装器与便携 zip 分发。
- FFmpeg 集成与发布清单生成。
- 打包历史台账（`docs/ops/package-build-history.csv`）。

### 开发者体验

- 双模式开发：真实 API 模式与 Mock 模式快速迭代。
- 前端：Vue 3 + TypeScript + Vite 8 + Tailwind CSS v4 + shadcn-vue。
- 后端：Go 1.25+ + SQLite (modernc) + Zap 日志 + 整洁架构。
- 国际化：English、简体中文、日本語（vue-i18n）。
- 开发版性能监控栏（仅 dev 构建）。
- 错误边界与客户端请求超时。
- 全领域结构化错误码。

## 配置

运行时配置分为前端环境变量和后端设置两部分。

### 前端

- `VITE_USE_WEB_API=true`：启用真实后端
- `VITE_API_BASE_URL`：覆盖 API 基地址；未设置时，本机 loopback 的 Web API 开发态会直连开发后端 `:8080`，避免大文件上传经过 Vite 代理；release `:8081` 静态托管和其他模式仍使用同源 `/api`
- `VITE_LOG_LEVEL`：浏览器日志级别默认值

### 后端

后端会从 JSON 读取主运行配置，并合并以下资料库级配置文件：

- `config/library-config.cfg`

常见资料库级配置包括：

- `organizeLibrary`
- `metadataMovieProvider`
- `metadataMovieStrategy`
- `defaultImportLibraryPathId`
- `autoLibraryWatch`
- `autoActorProfileScrape`
- `launchAtLogin`
- `curatedFrameExportFormat`（默认 `jpg`；可选：`jpg`、`webp`、`png`）
- `proxy`
- 后端日志目录与保留设置

  空 `logDir` 表示”使用默认日志目录”，而不是关闭文件日志：
  正式包写入 `LOCALAPPDATA\\Curated\\logs`，开发态写入 `backend/runtime/logs`。

发布构建默认使用端口 `:8081`，除非被配置覆盖。

## API

Curated 提供基于 Go 的 HTTP API，用于资料库、播放、演员、设置、存储在线检测和萃取帧相关能力。

完整接口参考请见 [API.md](API.md)。

资料库存储在线检测使用 `/api/library/paths/storage-status` 相关接口识别离线或卷身份不匹配的库路径。当前实现优先覆盖 Windows；macOS 与 Linux 目前使用基础路径探测，并作为后续适配项。

## 仓库结构

```text
.
├── src/                    # Vue SPA：页面、UI、业务组件、API 客户端、适配器
├── backend/
│   ├── cmd/curated/        # 后端入口
│   └── internal/           # app、config、storage、server、scanner、scraper、tasks、desktop
├── config/                 # 资料库级运行配置
├── docs/                   # 见 docs/README.md：reference、product、ops、plan 等
├── icon/                   # 品牌设计源文件
└── package.json            # pnpm 脚本与依赖
```

根目录政策补充：

- `videos_test/` 固定保留在仓库根目录，作为本地测试素材目录。
- `config/` 继续保留在仓库根目录，承载资料库级运行配置；不要并入 `backend/internal/config`。
- `backend/runtime/` 是允许的开发态运行产物目录。
- 新的本地临时状态优先放到 `.workspace/`。
- Go 构建缓存不要创建在仓库内；release 脚本已经改为使用系统临时目录承载后端构建缓存。

## 发布与打包

推荐发布入口：

```powershell
pnpm release:publish
```

关键说明：

- 生产包版本号统一由 `scripts/release/version.json` 管理。
- 当前版本基线为 `1.4.6`。
- `pnpm release:*` 现在统一由 `python scripts/release/release_cli.py` 编排。
- `pnpm release:publish` 会先构建 Vue 前端、release Go 后端和 Electron main process，再组装产物。
- 发布流程会生成 Windows Electron 发布目录、便携包、安装器可执行文件和发布清单。
- 组装目录会复制 Electron runtime 到 `release/Curated`，把 `electron.exe` 重命名为 `Curated.exe`，写入 `resources/app/package.json`，并把 `electron-dist/`、`frontend-dist/` 和 Go 后端 `curated.exe` 放入 `resources/app/`。
- 打包历史台账已经迁移到 `docs/ops/package-build-history.csv`，文件采用 UTF-8 with BOM，便于 Excel / WPS 直接打开。
- 安装包仍然继续使用 Inno Setup，只是由 Python 负责渲染 `.iss` 模板并调用 `ISCC.exe`；安装后快捷方式和安装完成后的启动入口都指向 `{app}\Curated.exe`。
- 设置页可以为当前用户持久化 Windows 开机自启动；这类登录触发的启动会静默进入托盘，不会自动打开浏览器页面。

更多发布资料：

- [docs/plan/2026-03-31-production-packaging-and-config-strategy.md](docs/plan/2026-03-31-production-packaging-and-config-strategy.md)
- [docs/ops/package-build-history.csv](docs/ops/package-build-history.csv)
- [docs/ops/2026-04-02-package-build-history.md](docs/ops/2026-04-02-package-build-history.md)

## 文档

- [API.md](API.md)：公开 HTTP API 参考
- [docs/features/2026-05-03-feature-inventory.md](docs/features/2026-05-03-feature-inventory.md)：完整功能清单（所有已实现特性）
- [docs/product/2026-03-20-jav-libary.md](docs/product/2026-03-20-jav-libary.md)：产品设计与目标架构
- [docs/reference/2026-03-20-project-memory.md](docs/reference/2026-03-20-project-memory.md)：实现事实与稳定项目记忆
- [docs/reference/architecture-and-implementation.html](docs/reference/architecture-and-implementation.html)：架构总览
- [docs/reference/2026-03-21-library-organize.md](docs/reference/2026-03-21-library-organize.md)：资料库整理说明
- [docs/reference/2026-03-24-frontend-ui-spec.md](docs/reference/2026-03-24-frontend-ui-spec.md)：前端 UI 规范

## 说明

- 当前仓库仍处于 **Web-first** 实现阶段。
- Electron 当前已作为最小桌面壳层落在 `electron/`，并具备托盘生命周期管理与窄范围原生目录选择 preload 桥；更深的 IPC 桥、mpv/进程控制、更广泛的原生文件桥和手柄硬件集成仍是后续目标方向。
- `docs/film-scanner/` 主要保存参考资料和夹具，而不是生产模块布局。

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

Curated 是一个本地优先的媒体资料库应用，采用 Vue 3 前端与 Go + SQLite 后端。当前仓库已经具备以 Web 为先的架构、面向 Windows 的发布打包流程、托盘模式运行、元数据刮削、播放链路以及萃取帧管理能力。

产品正式名称是 **Curated**。仓库目录和 npm 包名仍可能使用 **`jav-shadcn`**。Go 模块名为 **`curated-backend`**，服务端入口位于 **`backend/cmd/curated`**。

## 项目亮点

- 本地优先架构，前端为 Vue SPA，后端为 Go HTTP API。
- 同时支持真实 API 模式与 Mock 模式，便于快速迭代界面。
- 使用 SQLite 持久化资料库数据、播放进度、评论、评分与萃取帧。
- Windows 发布流程已支持托盘模式启动、本地托管前端和安装包构建。
- 当前 Web 阶段已经接入演员元数据、萃取帧导出和播放会话诊断能力。

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

### 真实 API 与 Mock 模式

- 在仓库根目录 `.env` 中设置 `VITE_USE_WEB_API=true`，即可启用真实后端 API。
- 其他值会保持为 Mock 模式。
- Vite 开发服务器会把 `/api` 代理到 `http://localhost:8080`。

## 功能

### 资料库

- 面向大规模资料库的虚拟化海报网格浏览。
- 收藏、评分、标签和入库整理能力。
- 同一套前端服务层同时支持真实后端和 Mock 适配器。

### 播放

- Web API 模式下支持持久化续播进度。
- 当前播放链路已支持浏览器播放、外部播放器接力和 HLS 会话。
- 提供更丰富的播放决策诊断信息，用于解释直播、remux 与转码路径。

### 演员

- 支持演员库浏览、资料加载与用户标签编辑。
- 演员头像通过后端缓存后以同源方式提供。

### 萃取帧

- 支持萃取帧截图、浏览、打标、筛选与导出。
- 支持带元数据的 WebP / PNG 导出。

### 打包发布

- 提供面向 Windows 的发布流程。
- 发布态可使用托盘模式启动，并在可执行文件旁直接托管构建后的前端资源。

## 配置

运行时配置分为前端环境变量和后端设置两部分。

### 前端

- `VITE_USE_WEB_API=true`：启用真实后端
- `VITE_API_BASE_URL`：覆盖 API 基地址
- `VITE_LOG_LEVEL`：浏览器日志级别默认值

### 后端

后端会从 JSON 读取主运行配置，并合并以下资料库级配置文件：

- `config/library-config.cfg`

常见资料库级配置包括：

- `organizeLibrary`
- `metadataMovieProvider`
- `metadataMovieStrategy`
- `autoLibraryWatch`
- `proxy`
- 后端日志目录与保留设置

发布构建默认使用端口 `:8081`，除非被配置覆盖。

## API

Curated 提供基于 Go 的 HTTP API，用于资料库、播放、演员、设置和萃取帧相关能力。

完整接口参考请见 [API.md](API.md)。

## 仓库结构

```text
.
├── src/                    # Vue SPA：页面、UI、业务组件、API 客户端、适配器
├── backend/
│   ├── cmd/curated/        # 后端入口
│   └── internal/           # app、config、storage、server、scanner、scraper、tasks、desktop
├── config/                 # 资料库级运行配置
├── docs/                   # 产品说明、计划文档、UI 规范、架构文档
├── icon/                   # 品牌设计源文件
└── package.json            # pnpm 脚本与依赖
```

## 发布与打包

推荐发布入口：

```powershell
pnpm release:publish
```

关键说明：

- 生产包版本号统一由 `release/version.json` 管理。
- 当前版本基线为 `1.1.0`。
- 发布流程会生成 Windows 发布目录、便携包、安装器脚本和发布清单。
- Windows 发布构建默认以托盘模式运行，并在 `frontend-dist/` 与可执行文件同目录时直接托管前端。

更多发布资料：

- [docs/plan/2026-03-31-production-packaging-and-config-strategy.md](docs/plan/2026-03-31-production-packaging-and-config-strategy.md)
- [docs/2026-04-02-package-build-history.md](docs/2026-04-02-package-build-history.md)

## 文档

- [API.md](API.md)：公开 HTTP API 参考
- [docs/2026-03-20-jav-libary.md](docs/2026-03-20-jav-libary.md)：产品设计与目标架构
- [docs/2026-03-20-project-memory.md](docs/2026-03-20-project-memory.md)：实现事实与稳定项目记忆
- [docs/architecture-and-implementation.html](docs/architecture-and-implementation.html)：架构总览
- [docs/2026-03-21-library-organize.md](docs/2026-03-21-library-organize.md)：资料库整理说明
- [docs/2026-03-24-frontend-ui-spec.md](docs/2026-03-24-frontend-ui-spec.md)：前端 UI 规范

## 说明

- 当前仓库仍处于 **Web-first** 实现阶段。
- Electron 与 mpv 仍然是目标方向，不是当前仓库已交付能力。
- `docs/film-scanner/` 主要保存参考资料和夹具，而不是生产模块布局。

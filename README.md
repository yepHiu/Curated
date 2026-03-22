# Curated

**Curated** 为正式产品名称。代码仓库目录名为 `jav-shadcn`（Go module 等仍为 `jav-shadcn`）。

本地优先的影片资料库：Vue 3 前端 + Go 后端（SQLite、目录扫描、Metatube 刮削、资源缓存）。当前以 **Web 应用 + HTTP API** 为主；长期可演进为 Electron 桌面壳。

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | Vue 3、TypeScript、Vite 8、Tailwind CSS v4、shadcn-vue、vue-router、vue-virtual-scroller |
| 后端 | Go（`backend/go.mod`）、SQLite（modernc）、zap、metatube-sdk-go |
| 联调 | 开发时 Vite 将 `/api` 代理到 `http://localhost:8080`（见 `vite.config.ts`） |

## 仓库结构（摘要）

```text
.
├── src/                    # 前端源码（页面、组件、API、library-service 契约与 Web/Mock 适配器）
├── backend/
│   ├── cmd/javd/           # 守护进程入口（HTTP / stdio / both）
│   └── internal/           # 配置、存储、扫描、刮削、任务、HTTP 路由等
├── config/
│   └── library-config.cfg  # 库行为 JSON（如 organizeLibrary），与主配置合并
├── docs/                   # 产品设计、整理规则等文档
└── package.json            # 前端脚本与依赖（包管理：pnpm）
```

## 环境要求

- **Node.js**：建议 LTS（与 Vite 8 兼容）
- **pnpm**：本仓库以 `pnpm-lock.yaml` 为准
- **Go**：`1.25.4+`（与 `backend/go.mod` 一致）

## 快速开始

### 1. 启动后端（默认 `:8080`）

在仓库根目录或 `backend` 目录下均可；未指定 `-config` 时使用内置默认配置（含默认媒体目录、数据库与缓存路径）。

```bash
cd backend
go run ./cmd/javd
```

常用参数：

- `-config <path>`：JSON 配置文件路径（字段见 `backend/internal/config/config.go` 中 `Config`）
- `-mode http`：仅 HTTP（默认）
- `-mode stdio`：仅标准输入输出协议
- `-mode both`：同时起 HTTP 与 stdio

健康检查：`GET http://localhost:8080/api/health`

### 2. 启动前端

```bash
pnpm install
pnpm dev
```

浏览器打开终端提示的本地地址（通常为 `http://localhost:5173`）。

### 3. 数据源：Mock 与真实 API

- **连接后端 API**：设置环境变量 `VITE_USE_WEB_API=true`（仓库根目录 `.env` 已默认开启）。
- **纯前端 Mock**：`VITE_USE_WEB_API` 不为 `true` 时使用内存假数据，无需后端。

可选：`VITE_API_BASE_URL` 覆盖 API 根路径（默认与开发代理一致为 `/api`）。

## 后端配置说明（摘要）

未传 `-config` 时，默认行为包括：

- **HTTP 地址**：`:8080`
- **数据库**：`backend/runtime/jav-library.db`（相对仓库根；若在 `backend` 目录运行则为 `runtime/jav-library.db`）
- **缓存目录**：`backend/runtime/cache`
- **默认扫描目录**：`videos_test`、`docs/film-scanner/videos_test`（可按需在 JSON 配置里改 `libraryPaths`）

主配置 JSON 可包含例如：`logLevel`、`httpAddr`、`databasePath`、`cacheDir`、`libraryPaths`、`autoScanIntervalSeconds`、`organizeLibrary`、刮削/资源/任务超时等。

`config/library-config.cfg` 为额外 JSON，用于合并 **库级开关**（如 `organizeLibrary`），启动时会与主配置合并。

## HTTP API（摘要）

路由定义见 `backend/internal/server/server.go`，主要包括：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/library/movies` | 影片列表 |
| GET | `/api/library/movies/{id}` | 影片详情 |
| PATCH | `/api/library/movies/{id}` | 更新收藏、用户评分、`userTags`、`metadataTags`（分别整表替换用户标签与 NFO/元数据标签，见 `docs/backend-contract-constraints.md`） |
| DELETE | `/api/library/movies/{id}` | 删除影片记录 |
| GET | `/api/library/movies/{id}/stream` | 视频流 |
| POST | `/api/library/movies/{id}/scrape` | 单部重新刮削（异步任务） |
| GET/PATCH | `/api/settings` | 读取/更新设置 |
| POST/PATCH/DELETE | `/api/library/paths` … | 媒体库路径管理 |
| POST | `/api/scans` | 触发扫描 |
| GET | `/api/tasks/{taskId}` | 任务状态 |

前后端 DTO 与错误约定可参考 `backend/internal/contracts/contracts.go` 与 `src/api/types.ts`。

## 观看历史与续播（前端）

- 侧栏 **History** → 路由 `history`（[`src/views/HistoryView.vue`](src/views/HistoryView.vue)），按本地日期分组展示曾播放条目。
- **续播进度**保存在浏览器 **`localStorage`**（`jav-library-playback-progress-v1`），**不写数据库**；详情/资料库进播放器可带 `?t=`，从历史进播放器带 `?from=history` 以便顶栏返回。
- 实现概要见 [`docs/project-memory.md`](docs/project-memory.md) 中「观看进度与历史」。

## 前端脚本

```bash
pnpm dev        # 开发服务器
pnpm build      # 类型检查 + 生产构建
pnpm preview    # 预览构建产物
pnpm typecheck  # 仅 TypeScript 检查
pnpm lint       # ESLint
pnpm test       # Vitest
```

## 后端测试

```bash
cd backend
go test ./...
```

## 文档

- [`docs/jav-libary.md`](docs/jav-libary.md)：产品设计、目标架构与边界
- [`docs/project-memory.md`](docs/project-memory.md)：当前实现事实与阶段判断（含 Mock/Web 双模式与观看进度说明）
- [`docs/architecture-and-implementation.html`](docs/architecture-and-implementation.html)：架构与数据流说明（浏览器可直接打开）
- [`docs/library-organize.md`](docs/library-organize.md)：媒体库整理相关说明

## 说明

- **Electron / mpv**：见产品设计文档中的目标规划；本仓库当前以 HTTP 服务与 Web UI 为主。
- **`docs/film-scanner/`**：参考与测试素材为主，不等同于运行时依赖的模块布局（详见 `.cursor/rules/backend-go-standards.mdc`）。

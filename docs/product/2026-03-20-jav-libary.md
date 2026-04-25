# Curated 产品设计文档

---

## 目录

0. [文档定位](#0-文档定位)
1. [当前项目状态](#1-当前项目状态)
2. [产品目标与核心体验](#2-产品目标与核心体验)
3. [目标系统架构](#3-目标系统架构)
4. [前端边界与服务契约](#4-前端边界与服务契约)
5. [核心模块设计](#5-核心模块设计)
6. [领域模型与数据结构](#6-领域模型与数据结构)
7. [文件、缓存与配置模型](#7-文件缓存与配置模型)
8. [播放器设计](#8-播放器设计)
9. [UI 页面设计](#9-ui-页面设计)
10. [任务、事件与错误模型](#10-任务事件与错误模型)
11. [性能与日志](#11-性能与日志)
12. [阶段路线图与待决策事项](#12-阶段路线图与待决策事项)

---

## 0. 文档定位

本文用于定义 **Curated**（正式产品名；代码仓库目录多为 `jav-shadcn`）的产品蓝图、目标桌面架构和后续实现边界。

为避免误解，本文中的信息分为三类：

- `当前状态`：代码仓库中已经存在、可被验证的事实。
- `目标设计`：产品计划中的目标能力、推荐架构和理想交互。
- `待决策`：尚未拍板、需要在后续研发中确认的关键问题。

### 当前边界

- 当前仓库是 `Vue 3 + TypeScript + Vite + shadcn-vue` 的前端工程，并包含 **`Go + SQLite` 后端**；开发时可通过 **`VITE_USE_WEB_API`** + Vite 代理使用真实 **HTTP API**，亦可选 **Mock** 适配器（无后端）。
- **已接通时**：页面经 `services` 契约与 `src/api` 消费后端；**未接通时**：列表/详情等仍可由 mock 数据驱动，但 **观看进度与历史** 始终依赖浏览器 **`localStorage`**（与后端无关）。
- 当前仓库尚未在根工程中落地 `Electron`、`preload`、`mpv` 和完整桌面运行时。
- 因此，本文中涉及桌面桥接、**mpv** 控制链和最终桌面编排的内容，均默认属于 `目标设计`，除非明确标注为 `当前状态`。

### 文档维护原则

- 若代码与文档冲突，以代码现状为准，并在文档中补充差距说明。
- Renderer 层只应依赖统一的前端服务接口和 typed contracts。
- 文档更新时应优先说明边界，不应把未来规划写成已实现事实。

---

## 1. 当前项目状态

### 1.1 当前状态

- 已有路由驱动的前端 SPA，而不是单页 demo。
- `src/App.vue` 只承载 `RouterView`。
- `src/layouts/AppShell.vue` 已实现稳定的应用壳层。
- 已落地页面包括：
  - `library`
  - `favorites`
  - `recent`
  - `tags`
  - **`history`（观看历史，按本地日期分组；进度存 `localStorage`）**
  - `detail/:id`
  - `player/:id?`
  - `settings`
- 当前产品组件集中在 `src/components/jav-library`。
- **双数据源**：开启 Web API 时经 **`src/services/adapters/web`** 与后端交互；Mock 模式下类型与假数据仍可参考 `src/lib/jav-library.ts`。
- 当前库页已具备搜索、标签页切换、选中影片状态、海报墙浏览和虚拟滚动能力。
- **Web 模式下**：播放器可通过 **`GET .../stream`** 用 `<video>` 播放；详情 PATCH、扫描/刮削任务等与后端联动。**Mock 模式**下播放流等能力降级。
- **观看进度**：前端已实现续播与历史列表，**未**对应服务端 `play_history` 表（见 §6.5）。

### 1.2 当前问题

- Mock 与真实 API 并存，若未读环境配置易误判“是否已接后端”。
- 部分能力（如续播）仅在前端持久化，与长期「账号/多设备」愿景需后续对齐。
- 文档需持续区分「仅前端」「仅后端」「全链路已通」三类状态。

### 1.3 现阶段定位

- 当前阶段的核心目标不是实现桌面 runtime，而是先把产品信息架构、交互模型和前端契约稳定下来。

---

## 2. 产品目标与核心体验

### 2.1 产品目标

**Curated** 的目标是成为一款本地优先的桌面媒体库应用，用于管理、浏览、搜刮、播放和维护个人影片资料库。

### 2.2 核心用户任务

产品需要围绕以下四条主路径构建：

1. `Browse`：快速浏览大量影片海报并筛选目标内容。
2. `Inspect`：打开单片详情查看元数据、演员、标签和预览内容。
3. `Play`：低干扰地进入播放器并保持流畅控制。
4. `Configure`：管理目录、扫描规则、搜刮策略和播放偏好。

### 2.3 产品设计原则

- 媒体库优先：优先服务“看图找片”和“快速定位”。
- 桌面优先：UI 可以提前体现桌面产品心智，但不能依赖尚不存在的桌面能力。
- 结构先行：先确定页面结构、状态模型和服务契约，再接真实后端。
- 可替换性优先：页面层不应与 mock 数据、IPC 或播放器协议深度耦合。
- 任务可见：扫描、搜刮、缓存、播放状态都应能被用户感知和追踪。

---

## 3. 目标系统架构

### 3.1 目标设计

推荐的目标架构如下：

```txt
┌────────────────────────────────────┐
│            Electron Host           │
│                                    │
│  ┌──────────────────────────────┐  │
│  │ Vue Renderer                 │  │
│  │ - views / components         │  │
│  │ - stores / services          │  │
│  │ - typed contracts            │  │
│  └───────────────┬──────────────┘  │
│                  │ preload bridge   │
│  ┌───────────────▼──────────────┐  │
│  │ Electron Main                │  │
│  │ - 权限边界                    │  │
│  │ - 子进程生命周期              │  │
│  │ - 安全白名单                  │  │
│  └───────────────┬──────────────┘  │
└──────────────────│─────────────────┘
                   │ stdio / command bus
                   ▼
┌────────────────────────────────────┐
│             Go Backend             │
│                                    │
│  - Library Manager                 │
│  - Scanner Service                 │
│  - Metadata Scraper                │
│  - Task/Event Hub                  │
│  - Database Layer (SQLite)         │
│  - Player Controller               │
└───────────────┬────────────────────┘
                │
      ┌─────────┼─────────┐
      ▼         ▼         ▼
   SQLite    FileSystem   mpv
```

### 3.2 当前状态

- 当前仅存在 `Vue Renderer` 侧的原型。
- 当前没有 `preload bridge`、`Electron Main`、`Go Backend` 和命令总线。

### 3.3 模块职责

| 模块 | 职责 |
| --- | --- |
| Renderer | 页面展示、用户交互、前端状态管理、服务契约消费 |
| preload | 收敛桥接 API，暴露白名单能力给 Renderer |
| Electron Main | 管理窗口、权限、系统级能力和 Go 子进程 |
| Go Backend | 业务逻辑、数据库访问、扫描、搜刮、播放器、任务系统 |
| SQLite | 本地持久化 |
| mpv | 原生播放能力与播放状态事件来源 |

### 3.4 架构约束

- Renderer 不直接接触数据库、文件系统、命名管道或 mpv。
- 前端不同时依赖多套底层协议，只依赖一套前端服务接口。
- 真实桥接之前，必须先定义命令、事件、DTO、错误码和任务状态。

---

## 4. 前端边界与服务契约

### 4.1 当前状态

- 当前页面直接从 `src/lib/jav-library.ts` 获取类型和 mock 数据。
- 当前尚未抽象 `services` 层，也没有 adapter 切换机制。

### 4.2 目标设计

前端应新增统一的服务层，例如：

```txt
src/
  services/
    library/
    scan/
    scraper/
    player/
    settings/
```

页面依赖方向应为：

```txt
views / components
        ↓
front-end services
        ↓
mock adapter | desktop adapter
```

### 4.3 推荐的前端服务接口

```ts
interface LibraryService {
  listMovies(input: ListMoviesInput): Promise<ListMoviesResult>
  getMovieDetail(movieId: string): Promise<MovieDetail>
  updateMovieMeta(movieId: string, patch: UpdateMoviePatch): Promise<MovieDetail>
}

interface ScanService {
  startScan(input: StartScanInput): Promise<AsyncTaskReceipt>
  getScanStatus(taskId: string): Promise<TaskSnapshot>
}

interface ScraperService {
  scrapeMovie(input: ScrapeMovieInput): Promise<AsyncTaskReceipt>
  rebuildPosterCache(input: RebuildPosterCacheInput): Promise<AsyncTaskReceipt>
}

interface PlayerService {
  open(input: OpenPlayerInput): Promise<PlayerSession>
  command(input: PlayerCommandInput): Promise<void>
  getState(sessionId: string): Promise<PlayerState>
}

interface SettingsService {
  getSettings(): Promise<AppSettings>
  updateSettings(patch: UpdateSettingsInput): Promise<AppSettings>
}
```

### 4.4 待决策

- 是否在前端引入 `pinia` 作为长期状态层。
- 服务接口返回值是否统一包装为 `Result<T, AppError>` 结构。
- 事件订阅采用主动轮询、事件总线，还是基于 Electron 的推送桥接。

---

## 5. 核心模块设计

### 5.1 影片库管理（Library Manager）

#### 当前状态

- 当前已在前端原型中实现库页、收藏页、最近页、标签页和详情页的基本结构。
- 当前库页的上下文依赖 route name、query 参数和 mock 数据计算。

#### 目标设计

影片库管理负责：

- 获取影片列表，支持分页、筛选、排序、搜索。
- 获取单片详情。
- 更新用户侧元数据，例如收藏、评分、用户标签、已看状态。
- 支撑海报浏览、详情联动和相关推荐。

#### 推荐能力

- 列表视图支持：
  - 搜索关键词
  - 标签过滤
  - 演员过滤
  - 评分排序
  - 最近添加排序
- 详情视图支持：
  - 基础元数据
  - 图片资源
  - 标签编辑
  - 收藏与评分
  - 相关推荐

#### 待决策

- 相关推荐基于演员、标签还是系列关系。
- 用户标签与搜刮标签的冲突解决策略。

### 5.2 扫描服务（Scanner Service）

#### 当前状态

- **已实现（代码为准）**：后端 **`POST /api/scans`**、周期 **`autoScanIntervalSeconds`**、**fsnotify** 库根监听（主配置 **`libraryWatchEnabled`** + 持久化 **`autoLibraryWatch`**，见 **`config/library-config.cfg`**）；设置页可触发路径/全库扫描并与任务轮询联动。监听触发的扫描在任务元数据中常带 **`trigger: fsnotify`**。
- **仍为占位或简化的 UI**：例如部分「维护」按钮、硬件解码等仍可能未接后端或未完整实现；以具体页面与 `README.md` / `project-facts` 为准。

#### 目标设计

扫描流程建议如下：

```txt
扫描配置目录
    │
    ▼
识别视频文件
    │
    ▼
提取番号
    │
    ▼
判断是否已入库
    │
 ┌──┴──┐
 ▼     ▼
更新路径  新建影片记录
         │
         ▼
触发搜刮任务
         │
         ▼
刷新图片缓存
         │
         ▼
更新任务结果
```

#### 支持格式

- `mp4`
- `mkv`
- `avi`
- `mov`
- 可扩展：`ts`

#### 番号识别规则

以下文件名应统一解析为 `ABC-123`：

```txt
ABC-123.mp4
abc123.mp4
ABC_123.mp4
ABC123.mp4
```

建议基础匹配规则：

```txt
([A-Za-z]{2,5})[-_ ]?(\d{2,5})
```

标准化流程：

```txt
原始文件名
   │
   ▼
正则提取字母段 + 数字段
   │
   ▼
统一转为大写
   │
   ▼
输出标准番号 ABC-123
```

#### 待决策

- 是否支持更复杂的番号格式和无码识别。
- 无法识别番号的文件是直接跳过，还是进入人工待处理队列。
- 增量扫描与全量扫描是否共用一套任务模型。

### 5.3 元数据搜刮（Metadata Scraper）

#### 当前状态

- 当前仅在产品文档和设置页原型中体现搜刮能力，尚无真实实现。

#### 目标设计

核心职责：

- 基于番号拉取元数据。
- 下载封面、预览图和演员头像。
- 处理多来源搜刮和失败回退。
- 为前端提供搜刮信息和任务进度。

建议流程：

```txt
接收番号
   │
   ▼
选择搜刮源
   │
   ▼
拉取 metadata
   │
   ▼
下载 poster / thumb / preview
   │
   ▼
写入数据库与缓存
   │
   ▼
发送任务完成事件
```

#### 搜刮器抽象

```go
type Scraper interface {
    Search(code string) (*Metadata, error)
    DownloadCover(url string) ([]byte, error)
    DownloadPreview(url string) ([]byte, error)
}
```

推荐来源优先级：

1. `metatube-sdk-go`
2. 未来可扩展 `JAVBus`
3. 未来可扩展 `JAVDB`
4. 其他可插拔来源

#### 待决策

- 单源失败时是否自动切换备用源。
- 图片缓存的失效与刷新策略。
- 搜刮结果是否保留来源字段和版本字段。

### 5.4 播放器控制（Player Controller）

#### 当前状态

- **Web 路径**：前端 `PlayerPage` 使用 **`<video>` + HTTP stream**，具备真实播放与进度 UI；**非** Go `Player Controller` / mpv IPC。
- **桌面路径**：下文「目标设计」中的统一控制模块与 mpv 事件仍未在仓库落地。

#### 目标设计

播放器控制模块负责：

- 打开指定影片。
- 控制播放、暂停、跳转、音量和全屏。
- 获取时长、播放进度、暂停状态和结束事件。
- 将 mpv 事件转成前端可消费的统一播放器状态。

建议公开的控制能力：

```txt
open(file)
pause()
resume()
seek(seconds)
setVolume(value)
toggleFullscreen()
stop()
```

#### 待决策

- 是否由 Go 独占管理 mpv，还是 Electron Main 直接参与部分窗口控制。
- 播放器会话是单实例还是允许多窗口播放。

---

## 6. 领域模型与数据结构

### 6.1 当前状态

- 当前前端已有 `Movie`、统计卡片、设置项和若干展示字段，但这只是原型层模型。

### 6.2 目标领域对象

建议至少包含以下核心实体：

- `Movie`
- `Actor`
- `Tag`
- `LibraryPath`
- `ScanTask`
- `ScrapeTask`
- `PlayerSession`
- `AppSettings`

### 6.3 推荐前端领域模型

```ts
interface Movie {
  id: string
  code: string
  title: string
  studio?: string
  actors: ActorRef[]
  tags: TagRef[]
  runtimeSeconds?: number
  rating?: number
  favorite: boolean
  summary?: string
  releaseDate?: string
  year?: number
  filePath?: string
  posterUrl?: string
  thumbUrls?: string[]
  createdAt: string
  updatedAt: string
}
```

### 6.4 推荐数据库结构

#### Movies

```sql
movies
----------------------------
id                INTEGER PRIMARY KEY
code              TEXT UNIQUE
title             TEXT
file_path         TEXT
runtime_seconds   INTEGER
rating            REAL
favorite          BOOLEAN
summary           TEXT
release_date      DATETIME
source            TEXT
scan_status       TEXT
scrape_status     TEXT
poster_path       TEXT
created_at        DATETIME
updated_at        DATETIME
```

#### Actors

```sql
actors
----------------------------
id                INTEGER PRIMARY KEY
name              TEXT
avatar_path       TEXT
created_at        DATETIME
updated_at        DATETIME
```

#### MovieActors

```sql
movie_actors
----------------------------
movie_id          INTEGER
actor_id          INTEGER
sort_order        INTEGER
```

#### Tags

```sql
tags
----------------------------
id                INTEGER PRIMARY KEY
name              TEXT
type              TEXT
created_at        DATETIME
updated_at        DATETIME
```

#### MovieTags

```sql
movie_tags
----------------------------
movie_id          INTEGER
tag_id            INTEGER
source            TEXT
```

#### LibraryPaths

```sql
library_paths
----------------------------
id                INTEGER PRIMARY KEY
path              TEXT UNIQUE
title             TEXT
enabled           BOOLEAN
last_scan_at      DATETIME
created_at        DATETIME
updated_at        DATETIME
```

### 6.5 待决策

- 是否需要单独的 `assets` 表记录 poster、thumb、preview 的派生资源。
- 是否需要 **`play_history`（或服务端等价模型）** 支撑断点续播与最近播放的**跨设备/可备份**同步。**`当前状态`（前端）**：续播与观看历史已由 **`localStorage`** 实现（见 `docs/reference/2026-03-20-project-memory.md`），**不写入**当前 SQLite 库。
- 是否要为电影文件和逻辑影片分离建模，以支持多文件版本。

---

## 7. 文件、缓存与配置模型

### 7.1 影片目录结构

以番号 `ABC-123` 为例：

```txt
ABC-123/
 ├─ ABC-123.mp4
 ├─ ABC-123.nfo
 ├─ poster.jpg
 ├─ thumb.jpg
 └─ preview/
     ├─ 01.jpg
     ├─ 02.jpg
     └─ ...
```

### 7.2 应用缓存目录

```txt
Library/
 └─ cache/
     ├─ poster/
     │   ├─ ABC-123_small.jpg
     │   └─ ABC-123_large.jpg
     ├─ preview/
     └─ actor/
```

### 7.3 配置模型

建议持久化配置如下：

```json
{
  "libraryPaths": [
    "D:/Movies",
    "E:/AV"
  ],
  "scanInterval": 3600,
  "autoScrape": true,
  "player": {
    "hardwareDecode": true
  }
}
```

### 7.4 字段说明

| 字段 | 说明 |
| --- | --- |
| `libraryPaths` | 影片存储目录列表，支持多个 |
| `scanInterval` | 自动扫描间隔，单位秒 |
| `autoScrape` | 发现新影片后是否自动搜刮 |
| `player.hardwareDecode` | 是否启用硬件解码 |

### 7.5 待决策

- 配置文件由 Electron 管理还是 Go 管理。
- 是否允许每个媒体目录拥有独立扫描策略。

---

## 8. 播放器设计

### 8.1 目标设计

推荐使用 `mpv + FFmpeg`。

播放器结构建议为：

```txt
Player
 ├─ Video Surface
 ├─ Subtitle Layer
 ├─ Control Overlay
 └─ Gesture Layer
```

### 8.2 视频渲染思路

可由 mpv 直接渲染到 Electron 原生窗口句柄：

```bash
mpv --wid=<window_id> --input-ipc-server=\\.\pipe\mpv-pipe movie.mkv
```

Electron 获取窗口句柄示意：

```js
win.getNativeWindowHandle()
```

### 8.3 控制链路

```txt
UI 操作
  -> Renderer Service
  -> preload
  -> Electron Main
  -> Go Player Controller
  -> mpv JSON IPC
```

### 8.4 常用命令示意

```json
{ "command": ["cycle", "pause"] }
{ "command": ["set_property", "pause", true] }
{ "command": ["seek", 10] }
{ "command": ["set_property", "volume", 80] }
```

### 8.5 关键事件

| 事件 | 用途 |
| --- | --- |
| `time-pos` | 更新进度条 |
| `duration` | 更新总时长 |
| `pause` | 更新暂停状态 |
| `end-file` | 更新播放结束状态 |
| `file-loaded` | 首帧就绪和播放器初始化完成 |

### 8.6 当前状态

- 当前 `PlayerPage` 为 video-first 布局；在 **Web 阶段（`VITE_USE_WEB_API`）** 下，主视频通过 **`GET /api/library/movies/{movieId}/stream`** 输出字节流（支持 `Range`），前端使用 **`<video>`** 播放；解码由 **浏览器**完成，不兼容格式需在 UI 提示或依赖后续桌面 `mpv`。
- 流路径会校验影片 `location` 是否落在已配置的 **library path** 之下，避免任意文件读取。
- **断点续播（前端）**：进度写入浏览器 **`localStorage`**；支持路由 **`?t=秒`** 与从 **观看历史** 进入；详情见 `project-memory` 与 `README`。**非**服务端会话状态。
- **目标设计**中的 `mpv + Electron` 控制链（§8.2–8.3）仍为桌面阶段方案，与上述 Web 播放可并存（后期可双模式）。

---

## 9. UI 页面设计

### 9.1 当前信息架构

- 侧边导航：
  - 全部影片
  - 收藏
  - 最近添加
  - 标签
  - **观看历史（`history`）**
  - 设置
- 主路径：
  - 库页浏览
  - 详情页查看
  - 播放器页播放
  - **观看历史（按日分组、海报卡片 + 进度条）**
  - 设置页维护

### 9.2 影片库页面

#### 当前状态

- 已有海报优先的卡片浏览。
- 已有搜索和 tab 切换。
- 已有虚拟滚动支撑大规模浏览。

#### 目标设计

- 海报优先、紧凑、可快速扫读。
- 支持按标签、演员、评分、添加时间筛选与排序。
- 支持选中态、详情跳转、播放器跳转。
- 避免演变成后台管理表格。

### 9.3 详情页

#### 当前状态

- 已有左侧海报、右侧详情信息、预览占位区和相关推荐。

#### 目标设计

- 聚焦一部影片的完整信息。
- 明确区分系统元数据与用户侧元数据。
- 保持清晰信息层级，不做 marketing 式大 banner。

### 9.4 播放器页

#### 当前状态

- 低干扰、video-first；Web API 模式下为真实 **`<video>`** 流（见 §8.6）。
- 支持键盘快捷键、全屏、音量与进度条；**本地续播**与 **`?t=`** 入口。

#### 目标设计

- 保持低 chrome。
- 控件只服务播放，不喧宾夺主。
- 桌面阶段状态更新需和 **mpv** 等真实播放器会话同步。

### 9.4.1 观看历史页（`history`）

#### 当前状态

- 侧栏 **History** 进入；[`HistoryView`](src/views/HistoryView.vue) 按 **本地日历日**分组（如今天、昨天、中文日期）。
- 卡片 [`PlaybackHistoryCard`](src/components/jav-library/PlaybackHistoryCard.vue)：左侧标题/演员/番号，右侧海报（**`coverUrl` 优先**于 `thumbUrl`，宽比例容器 + `object-cover`），底部细进度条。
- 数据与 **SQLite 无关**；条目与当前库中影片合并展示（已删库条目不显示）。

#### 目标设计

- 可选与服务端 **`play_history`** 或分析事件合并，支持跨设备与备份恢复。

### 9.5 设置页

#### 当前状态

- 已有目录管理、扫描间隔、硬件解码、手动任务入口等原型结构。

#### 目标设计

- 设置页应表达真实系统操作，而不是抽象偏好面板。
- 支持多目录管理、扫描策略、搜刮策略、播放配置和维护动作。

### 9.6 产品语言策略

#### 当前状态

- 设计文档以中文为主，当前前端原型文案偏英文。

#### 待决策

- 最终产品是中文优先、英文优先，还是双语。
- 内部字段命名和对外显示文案是否分离维护。

---

## 10. 任务、事件与错误模型

### 10.1 为什么需要这一层

扫描、搜刮、缓存重建和播放状态同步都属于异步任务。

如果没有统一任务模型，将出现以下问题：

- 页面无法展示“进行中 / 成功 / 失败”状态。
- 任务之间无法串联。
- 错误难以追踪和复现。

### 10.2 推荐任务模型

```ts
type TaskStatus = "queued" | "running" | "success" | "failed" | "cancelled"

interface AsyncTaskReceipt {
  taskId: string
  taskType: "scan" | "scrape" | "poster-cache" | "player-open"
}

interface TaskSnapshot {
  taskId: string
  taskType: string
  status: TaskStatus
  progress?: number
  message?: string
  startedAt?: string
  finishedAt?: string
}
```

### 10.3 推荐事件类型

```ts
type AppEvent =
  | { type: "scan.progress"; taskId: string; progress: number }
  | { type: "scan.finished"; taskId: string }
  | { type: "scrape.finished"; taskId: string; movieId: string }
  | { type: "player.state.changed"; sessionId: string; state: PlayerState }
```

### 10.4 推荐错误码

| 错误码 | 含义 |
| --- | --- |
| `LIBRARY_PATH_NOT_FOUND` | 目录不存在 |
| `SCAN_TASK_FAILED` | 扫描任务失败 |
| `CODE_PARSE_FAILED` | 番号识别失败 |
| `SCRAPER_UNAVAILABLE` | 搜刮源不可用 |
| `PLAYER_START_FAILED` | 播放器启动失败 |
| `CONFIG_SAVE_FAILED` | 配置保存失败 |

### 10.5 当前状态

- 当前还没有真实任务系统、事件流和错误码体系。
- 但设置页和播放器页已经足以反向约束这套模型的设计。

---

## 11. 性能与日志

### 11.1 性能优化

#### 封面缩略图

影片库不应直接消费原始大图，建议统一生成：

```txt
poster_small.jpg
poster_large.jpg
```

#### 虚拟滚动

库页已明确采用大规模海报浏览模式，虚拟滚动是必要能力。

推荐继续沿用：

```txt
vue-virtual-scroller
```

#### 图片缓存策略

- 库页优先使用缩略图。
- 详情页使用中大图。
- 预览图懒加载。
- 图片生成与清理走后台任务。

### 11.2 日志系统

目标后端建议使用 `zap` 记录日志：

```txt
logs/
 └─ app.log
```

建议至少覆盖：

| 模块 | 记录内容 |
| --- | --- |
| 扫描服务 | 扫描开始、结束、识别文件、跳过原因 |
| 元数据搜刮 | 搜刮成功、失败、来源、耗时 |
| 播放器 | 打开、停止、事件异常、命令失败 |
| 配置 | 设置变更、保存失败 |
| 错误 | 异常栈、任务上下文、关联 ID |

### 11.3 待决策

- 日志是否分级展示给用户。
- 是否需要单独的任务审计视图。

---

## 12. 阶段路线图与待决策事项

### 12.1 建议阶段路线图

#### 阶段 1：稳定前端原型

- 收敛页面结构和交互路径。
- 统一文案策略。
- 从 `src/lib/jav-library.ts` 提炼服务契约和 DTO。

#### 阶段 2：建立前端服务层

- 增加 `services` 与 adapter 结构。
- 让 `views` 不再直接依赖 mock 数据模块。
- 定义任务模型、事件模型和错误码。

#### 阶段 3：Web 后端联通

- 为 `Go Backend` 提供 Web 可调用的 HTTP API。
- 建立 `web adapter`，让前端页面可切到真实后端。
- 跑通影片库、详情、设置和扫描的 Web 链路。

#### 阶段 4：任务与写操作增强

- 补齐设置更新、影片更新等写操作。
- 完善扫描任务状态流和前端任务反馈。
- 视需要引入 SSE / WebSocket，或继续使用轮询方案。

#### 阶段 5：桌面桥接与播放器闭环

- 接入 `Electron`。
- 设计 `preload` 白名单 API。
- 复用既有服务契约切换到桌面桥接。
- 接入 `mpv`。
- 建立播放器会话模型。
- 实现进度、暂停、结束等状态同步。

### 12.2 当前最重要的待决策事项

1. 前端长期状态层是否引入 `pinia`。
2. 前端服务层接口的命名、返回值和错误包装标准。
3. Web 阶段任务反馈采用轮询、SSE 还是 WebSocket。
4. 配置持久化长期归属 `Go Backend` 还是未来的 `Electron` 宿主层。
5. 搜刮任务、缓存任务和扫描任务的编排关系。
6. 产品最终显示语言是否统一为中文，还是保留英文界面。

### 12.3 文档结论

这份文档的核心作用不是描述“仓库现在已经做完了什么”，而是把当前前端原型与未来桌面产品之间的桥梁设计清楚：

- 当前已经有较完整的产品原型。
- 当前还没有真实桌面能力。
- 接下来的关键不是盲目接技术，而是先把契约、边界和任务模型定义好。

# Curated 后端实现计划

## 1. 文档目的

本文用于沉淀 **Curated** 的后端落地计划，指导后续 `Go Backend` 的实现顺序、模块边界和阶段性目标。

需要明确的是：

- 当前仓库仍然是 `Vue 3 + TypeScript + Vite` 的前端脚手架。
- `docs/2026-03-20-jav-libary.md` 描述的是目标产品蓝图，不代表这些后端能力已经存在。
- 本文讨论的是未来后端的设计与实施计划，不应被当作当前实现事实。

## 2. 后端目标

目标后端面向桌面端本地媒体库场景，承担以下职责：

- 影片库管理
- 目录扫描与入库
- 元数据搜刮与资源缓存
- 本地数据库持久化
- 播放器进程控制
- 后台任务、日志与错误管理

结合现有设计文档，目标技术方向如下：

- 后端语言：`Go`
- 数据库：`SQLite`
- 元数据搜刮：`metatube-sdk-go`
- 播放控制：`mpv + JSON IPC`
- 日志系统：`zap`

## 3. 设计原则

后端建设建议遵循以下原则：

1. 先定义边界，再实现业务。
2. 先保证 `Library Manager` 可用，再建设扫描、搜刮和播放器。
3. 前端只依赖类型化服务接口，不直接感知 `SQLite`、文件系统和 `mpv` 细节。
4. 扫描、搜刮、图片缓存都视为后台任务，而不是一次性脚本。
5. 第三方数据源必须经过统一映射层，不能直接把外部字段写入业务模型。
6. 文档、协议和错误码要先于复杂实现落地。

## 4. 系统边界建议

建议采用如下调用链路：

```text
Vue Renderer
    ->
preload
    ->
Electron Main
    ->
Go Backend
    ->
SQLite / File System / mpv / Scraper
```

边界约定如下：

- `Renderer` 只调用前端服务接口。
- `preload` 是 Renderer 与桌面能力之间的唯一桥接入口。
- `Electron Main` 负责权限边界、子进程生命周期和桌面编排。
- `Go Backend` 负责业务逻辑、持久化、任务系统、播放器控制。

`Electron Main` 与 `Go Backend` 的通信，建议优先采用 `子进程 + stdio + JSON 命令协议`。相比本地 HTTP 端口，该方案更适合桌面应用，权限边界更清晰，也更便于进程生命周期管理。

## 4.1 当前前端对接事实

在后端真正落地前，需要明确当前前端的对接起点：

- 当前仓库前端已经具备 `library`、`detail`、`player`、`settings` 等页面壳和路由结构。
- 当前页面数据来自 `src/lib/jav-library.ts` 的 typed mock 数据，不是独立服务层返回结果。
- 当前还没有 `services`、`desktop adapter`、`preload bridge` 或真实命令协议实现。

因此，后端的首轮协议设计必须以“可替换 mock 数据层”为目标，而不是假设前端已经具备桌面桥接代码。

## 4.2 与前端互联的接口要求

为确保前后端可以平滑接通，后端实现阶段应遵循以下约束：

- 后端领域边界应与前端服务域保持一致，优先覆盖 `library`、`scan`、`scraper`、`player`、`settings`。
- 所有可被 Renderer 感知的能力，都应先落为类型化命令、事件、DTO、错误码和任务状态，再落具体业务实现。
- Renderer 不应直接调用后端 HTTP 接口或底层进程协议，而应始终通过 `preload -> Electron Main -> Go Backend` 这条主链路访问。
- 后端返回结构应优先服务前端列表页、详情页、播放器页、设置页的既有信息架构，避免协议字段与前端模型严重错位。
- 扫描、搜刮、资源缓存、播放器状态等异步能力，必须通过统一事件模型向前端推送，不能只提供一次性同步命令。

## 5. 后端模块规划

### 5.1 Library Manager

职责：

- 获取影片列表
- 获取影片详情
- 更新收藏、评分、用户标签
- 删除影片记录
- 处理筛选、分页、排序

MVP 接口建议：

- `listMovies`
- `getMovieDetail`
- `toggleFavorite`
- `rateMovie`
- `addUserTag`
- `removeUserTag`
- `deleteMovie`

### 5.2 Scanner Service

职责：

- 遍历配置目录
- 识别视频文件
- 解析番号
- 去重判断
- 创建或更新影片记录
- 投递搜刮任务
- 记录扫描结果与日志

扫描流程建议：

```text
读取影片目录
    ->
识别视频文件
    ->
解析番号
    ->
检查是否已存在
    ->
创建或更新影片记录
    ->
投递搜刮任务
    ->
记录结果与进度
```

参考实现说明：

- 后端扫描与番号识别流程可参考 `docs/film-scanner`。
- 其中可重点参考的能力包括：递归查找视频文件、按扩展名过滤、从文件名提取番号、调用搜刮引擎查询影片信息。
- `film-scanner` 更接近单机 CLI 工具，本项目落地时应提炼其可复用逻辑，而不是直接照搬为最终架构。

### 5.3 Metadata Scraper

职责：

- 根据番号查询影片元数据
- 下载封面和预览图
- 统一映射第三方数据
- 写入数据库和资源缓存

实现建议：

- 定义 `Scraper` 抽象接口
- 第一阶段实现 `metatube` 适配器
- 后续允许扩展 `javbus`、`javdb` 等数据源
- 元数据搜刮实现可参考 `docs/film-scanner` 中基于 `metatube-sdk-go` 的调用方式
- 可借鉴其 `searchMovie`、封面下载、NFO 生成等流程设计，但需要重构为适合桌面应用和任务系统的服务层实现

建议接口：

```go
type Scraper interface {
    Search(num string) (*Metadata, error)
    DownloadCover(url string) ([]byte, error)
}
```

参考边界说明：

- `film-scanner` 已验证 `metatube-sdk-go` 在本地扫描场景中的可行性，可作为本项目搜刮模块的技术参考样本。
- 但本项目后端仍应在此基础上补齐任务状态、数据库写入、资源缓存、错误码和事件推送等桌面应用能力。

### 5.4 Asset Service

建议把资源处理从搜刮逻辑中拆分出来，形成独立服务，职责包括：

- 下载原始封面
- 生成缩略图
- 管理本地缓存路径
- 检查资源是否缺失
- 处理重试和清理

这样后续无论资源来自搜刮器、NFO 还是手动导入，都可以复用。

### 5.5 Player Controller

职责：

- 启动 `mpv`
- 通过 `JSON IPC` 发送控制命令
- 监听播放状态事件
- 向桌面层或前端转发播放器状态

第一阶段控制能力建议只覆盖：

- `play`
- `pause`
- `seek`
- `setVolume`
- `stop`
- `toggleFullscreen`

第一阶段事件建议只覆盖：

- `time-pos`
- `duration`
- `pause`
- `end-file`

## 6. 数据模型规划

`docs/2026-03-20-jav-libary.md` 中已经定义了 `movies`、`actors`、`movie_actors`、`tags`、`movie_tags`。这些结构可作为 MVP 起点，但不足以支撑完整后台任务和资源管理。

建议在首版数据库设计中补充以下模型：

- `library_paths`：影片目录配置
- `scan_jobs`：扫描任务
- `scan_items`：单个文件处理结果
- `media_assets`：封面、缩略图、预览图、本地缓存路径
- `app_settings`：应用设置
- `play_history`：播放历史，可先预留

关键字段建议：

- 所有核心表增加 `created_at`、`updated_at`
- 任务表增加 `status`、`started_at`、`finished_at`、`error_message`
- 资源表增加 `type`、`source_url`、`local_path`
- `movies` 增加 `scan_status`、`metadata_status`

关键约束建议：

- `movies.num` 唯一
- `movies.path` 唯一或半唯一策略
- 关联表增加联合唯一索引
- 扫描项按文件路径去重

## 7. 协议与类型先行

在编写复杂业务逻辑之前，建议先定义以下内容：

- 命令协议
- 事件协议
- DTO
- 错误码
- 任务状态枚举
- 播放器状态模型

推荐最先定义的命令：

- `library.list`
- `library.detail`
- `library.update`
- `scan.start`
- `scan.status`
- `settings.get`
- `settings.update`
- `player.play`
- `player.command`

推荐最先定义的事件：

- `scan.started`
- `scan.progress`
- `scan.file_skipped`
- `scan.file_imported`
- `scan.completed`
- `scan.failed`
- `player.state_changed`
- `player.time_changed`

## 8. 任务系统规划

扫描、搜刮、资源下载本质上都属于后台任务，建议在 Go 后端中建立统一任务模型。

任务状态建议：

- `pending`
- `running`
- `completed`
- `partial_failed`
- `failed`
- `cancelled`

任务系统至少应支持：

- 创建任务
- 查询任务状态
- 推送任务进度
- 记录失败原因
- 支持后续重试

第一阶段可先采用进程内任务队列，不需要一开始引入复杂调度框架。

## 9. 推荐目录结构

建议未来后端工程采用如下结构：

```text
backend/
  cmd/
    curated/
  internal/
    app/
    config/
    logging/
    contracts/
    events/
    library/
    scanner/
    scraper/
    assets/
    player/
    storage/
    tasks/
```

目录职责建议：

- `cmd/curated`：程序入口
- `internal/app`：应用装配和启动流程
- `internal/contracts`：命令、事件、DTO、错误码
- `internal/storage`：数据库访问与 migration
- `internal/tasks`：后台任务模型与调度
- `internal/library`：影片库业务
- `internal/scanner`：扫描业务
- `internal/scraper`：搜刮器抽象与适配器
- `internal/assets`：资源下载和缓存
- `internal/player`：播放器控制

## 10. 分阶段实施计划

### 阶段一：协议与基础设施

目标：让后端工程具备最基本的可运行能力。

交付内容：

- `Go` 工程骨架
- 配置加载
- `zap` 日志初始化
- `SQLite` 连接和 migration
- 命令协议和事件协议草案
- DTO、错误码、任务状态枚举

完成标准：

- 后端可以启动
- 可以响应基础探活或简单命令
- 可以初始化数据库并输出结构化日志

### 阶段二：影片库管理 MVP

目标：支持影片库的读写操作，优先满足前端列表和详情页。

交付内容：

- 影片列表查询
- 影片详情查询
- 收藏/评分更新
- 用户标签增删
- 基础分页、筛选、排序

完成标准：

- 前端可以基于真实后端展示影片库页面
- 基础交互不依赖扫描服务即可演示

### 阶段三：扫描与搜刮 MVP

目标：实现目录扫描、番号识别、自动入库和元数据搜刮闭环。

交付内容：

- 多目录扫描
- 番号解析
- 去重判断
- 扫描任务状态记录
- `metatube` 搜刮接入
- 封面下载与本地缓存

完成标准：

- 指定目录可完成从扫描到入库的完整流程
- 扫描进度和失败原因可查询

### 阶段四：资源与稳定性增强

目标：补齐缩略图、资源缺失处理和任务稳定性。

交付内容：

- `poster_small` / `poster_large` 生成
- 资源存在性校验
- 任务失败重试
- 更完整的错误日志和诊断信息

完成标准：

- 前端影片库和详情页都能稳定使用本地缓存资源
- 资源缺失和任务失败可追踪

### 阶段五：播放器集成

目标：接入 `mpv` 并打通播放控制闭环。

交付内容：

- `mpv` 进程启动与销毁
- JSON IPC 通信
- 基础播放控制
- 播放状态事件监听与转发

完成标准：

- 可以从前端触发播放
- 可以同步暂停、进度、结束等状态

## 11. 测试与验证建议

后端开发过程中建议优先补齐以下测试：

- 番号解析单元测试
- 扫描去重逻辑单元测试
- 搜刮结果映射单元测试
- 数据库 repository 集成测试
- 任务状态流转测试
- 播放器控制层 mock 测试

建议优先自动化验证的高风险点：

- Windows 路径处理
- 大量文件扫描性能
- 重复入库去重
- 搜刮失败后的回退逻辑
- `mpv` 进程异常退出恢复

## 12. 当前最优先事项

按照当前项目阶段，建议优先顺序如下：

1. 先编写后端协议文档，包括命令、事件、DTO 和错误码。
2. 先补一层与当前 mock 数据可对齐的前端服务契约，再开始写真实后端实现。
3. 建立 `backend` 工程骨架和数据库 migration。
4. 优先实现 `Library Manager`，让前端先接入真实影片库接口。
5. 再实现 `Scanner Service` 与 `Metadata Scraper`。
6. 最后实现 `Player Controller`。

## 13. 不应提前假设的事项

在后端尚未落地前，不应默认以下能力已经存在：

- 已具备 `Electron` 主进程和 `preload` 桥接
- 已具备 `Go Backend` 运行时
- 已具备 `SQLite` schema 和 migration
- 已具备真实文件扫描、搜刮、播放器控制能力
- Renderer 可以绕过 `preload` 或前端服务层，直接访问底层后端协议

## 14. 配套约束文档

为避免后续后端实现偏离边界，建议与本文配套使用以下文档：

- `docs/2026-03-21-backend-contract-constraints.md`：约束命令、事件、DTO、错误码和对外协议边界
- `docs/2026-03-21-backend-go-standards.md`：约束 `Go` 目录结构、分层方式、错误处理、日志和测试习惯
- `docs/2026-03-21-backend-task-constraints.md`：约束扫描、搜刮、资源缓存等后台任务的统一模型

后续若项目真正进入实现阶段，应优先更新本文和 `docs/2026-03-20-project-memory.md`，确保“当前事实”和“未来目标”保持清晰分离。

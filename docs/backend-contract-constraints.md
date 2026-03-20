# JAV-Library 后端接口契约约束

## 1. 目的

本文定义 `Go Backend` 与前端层、桌面层对接时的契约约束，确保后续命令、事件、DTO 和错误码保持一致，不因实现先后顺序而失控。

## 2. 主链路约束

当前采用双阶段主链路：

```text
Phase 1: Vue Web App -> HTTP API -> Go Backend
Phase 2: Renderer -> preload -> Electron Main -> Go Backend
```

约束如下：

- Web 阶段前端只通过 HTTP API 访问后端，不直接访问数据库、文件系统或 `mpv`
- Electron 阶段 `preload` 仍是 Renderer 的唯一桥接入口
- Electron 阶段 `Electron Main` 负责命令转发、事件订阅、进程生命周期管理
- `Go Backend` 提供稳定命令与事件模型，不直接为前端页面结构硬编码
- HTTP API 与未来桌面桥接应复用同一套 DTO、错误码与任务状态定义

## 3. 命令设计约束

无论通过 HTTP 还是桌面桥接暴露，内部命令命名仍建议使用“领域.动作”格式，例如：

- `library.list`
- `library.detail`
- `library.update`
- `scan.start`
- `scan.status`
- `settings.get`
- `settings.update`
- `player.play`
- `player.command`

命令设计要求：

- 一个命令只做一类明确动作
- 同步命令返回结果，异步命令返回任务信息
- 长耗时操作不阻塞调用链，应进入任务系统
- 命令参数使用显式 DTO，不直接暴露数据库模型
- 不在契约层泄露 `SQLite`、命名管道、文件路径拼接等实现细节

对应到 Web 阶段时：

- HTTP 路由可以不是命令名本身，但应稳定映射到同一批领域动作
- 例如 `GET /api/library/movies` 对应 `library.list`
- 例如 `POST /api/scans` 对应 `scan.start`
- 例如 `GET /api/tasks/:taskId` 对应 `scan.status`

## 4. 事件设计约束

事件用于承载异步状态变化，命名建议使用“领域.状态”格式，例如：

- `scan.started`
- `scan.progress`
- `scan.file_skipped`
- `scan.file_imported`
- `scan.completed`
- `scan.failed`
- `player.state_changed`
- `player.time_changed`

事件设计要求：

- 事件必须包含稳定的 `type` 和 `timestamp`
- 任务相关事件必须带 `taskId`
- 与影片有关的事件应尽量带 `movieId` 或 `num`
- 事件负载要面向 UI 可消费，不要求前端拼接底层状态
- 同一类事件的字段应稳定，避免不同阶段随意变化

Web 阶段约束补充：

- 第一阶段可优先通过轮询 `task status` 替代实时事件推送
- 若 Web 端后续需要实时事件，可在保持事件 DTO 不变的前提下增加 SSE 或 WebSocket
- Electron 阶段再复用相同事件模型接入桌面桥接

## 5. DTO 设计约束

DTO 应按前端消费场景划分，而不是按数据库表一比一暴露。

建议优先定义：

- `MovieListItemDto`
- `MovieDetailDto`
- `ScanTaskDto`
- `ScanProgressEventDto`
- `SettingsDto`
- `PlayerStateDto`
- `AppErrorDto`

DTO 设计要求：

- 列表 DTO 与详情 DTO 分离
- 聚合展示字段允许在后端拼装
- 时间字段统一约定格式
- 状态字段统一使用枚举字符串
- 可选字段必须明确语义，避免“有时为空但未定义原因”

## 6. 错误码约束

错误码必须稳定、可枚举、可被前端消费，不允许只返回自由文本。

建议先定义如下错误域：

- `COMMON_*`
- `LIBRARY_*`
- `SCAN_*`
- `SCRAPER_*`
- `ASSET_*`
- `PLAYER_*`
- `SETTINGS_*`

错误返回至少包含：

- `code`
- `message`
- `retryable`
- `details`

## 7. 版本与演进约束

- 契约文档先于具体实现更新
- 如需破坏性变更，必须先更新文档并记录迁移影响
- 优先新增字段，避免随意重命名已有字段
- 在前后端尚未稳定前，仍应保持命令和事件命名一致性

## 8. 与 `film-scanner` 的关系

`docs/film-scanner` 可以作为元数据搜刮流程的实现参考，但不属于本项目正式契约的一部分。

约束如下：

- 可以参考其番号提取、搜刮调用、封面下载和 NFO 生成流程
- 不直接沿用其 CLI 输入输出作为本项目契约
- 所有对外能力仍以本项目命令、事件、DTO 和错误码为准

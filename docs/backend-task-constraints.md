# JAV-Library 后端任务系统约束

## 1. 目的

本文定义未来后端任务系统的统一约束，避免扫描、搜刮、资源缓存和播放器状态处理各自实现一套不兼容机制。

## 2. 适用范围

以下能力默认按后台任务模型处理：

- 目录扫描
- 元数据搜刮
- 封面与缩略图下载/生成
- 缓存刷新
- 可选的批量修复或重建任务

播放器控制命令本身可以是同步调用，但播放器状态变化仍应进入统一事件模型。

## 3. 任务状态约束

统一任务状态建议如下：

- `pending`
- `running`
- `completed`
- `partial_failed`
- `failed`
- `cancelled`

约束如下：

- 状态名必须全局统一
- 不同模块不得各自发明相近含义状态
- 任务一旦结束，不应再回到运行态
- 任务失败必须记录可诊断原因

## 4. 任务模型约束

建议任务至少具备以下字段：

- `taskId`
- `type`
- `status`
- `createdAt`
- `startedAt`
- `finishedAt`
- `progress`
- `message`
- `errorCode`
- `errorMessage`

建议任务类型至少包括：

- `scan.library`
- `scrape.movie`
- `asset.download`
- `asset.generate_thumbnail`

## 5. 幂等与去重约束

- 同一路径的重复扫描应具备去重策略
- 同一影片的重复搜刮应具备覆盖或跳过策略
- 同一资源的重复下载应具备缓存判断
- 同一批量任务的重试应避免产生脏数据和重复关联关系

## 6. 事件推送约束

任务系统必须提供统一事件：

- `task.started`
- `task.progress`
- `task.completed`
- `task.failed`

领域任务可再补充细粒度事件，例如：

- `scan.file_skipped`
- `scan.file_imported`
- `scraper.metadata_saved`
- `asset.poster_downloaded`

约束如下：

- 通用任务事件字段保持一致
- 进度字段语义稳定
- 前端可仅依赖通用任务事件做基础 UI
- 领域事件用于增强展示和调试

## 7. 日志与诊断约束

- 任务日志必须可通过 `taskId` 聚合
- 失败任务必须保留关键输入和失败阶段
- 扫描任务需记录跳过原因
- 搜刮任务需记录数据源与影片编号
- 资源任务需记录目标路径和资源类型

## 8. 实现建议

- 第一阶段使用进程内任务队列即可
- 状态持久化优先落数据库
- 事件通过统一事件总线转发
- 重试机制先做手动重试，再考虑自动策略

## 9. 与 `film-scanner` 的关系

`docs/film-scanner` 更偏一次性 CLI 流程，而本项目应将其相关逻辑拆入任务系统：

- 文件扫描逻辑进入 `scanner`
- 元数据搜刮逻辑进入 `scraper`
- 封面下载与缩略图进入 `assets`
- 所有过程都通过任务状态和事件推送对外暴露

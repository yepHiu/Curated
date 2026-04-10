# Playback Pipeline Phase 3 PR 说明

## 建议标题

`feat(playback): add hls session governance diagnostics`

## 背景

本分支前两个阶段已经补齐了播放决策层与 remux-first HLS 启动策略，但 HLS 会话仍然缺少最基本的治理能力：

- 会话闲置后没有统一回收机制
- recent/status 诊断接口缺失
- 会话被显式删除后，链路诊断信息会立刻丢失

这会导致播放器问题难以定位，尤其是 ffmpeg 启动失败、会话被清理、前端切换播放模式之后，很难快速解释“刚才到底发生了什么”。

## 这次改动

- 为 `playback.Manager` 增加 idle timeout 与 janitor 循环，定期回收闲置 HLS 会话
- 为会话状态补齐运行时元数据：
  - `lastAccessedAt`
  - `finishedAt`
  - `lastError`
- 新增会话快照模型与应用层 DTO 映射：
  - `PlaybackSessionStatusDTO`
  - `PlaybackSessionListDTO`
- 新增播放链路诊断接口：
  - `GET /api/playback/sessions/recent`
  - `GET /api/playback/sessions/{id}`
- recent 会话列表不再只看“当前仍活着的会话”，而是保留一份有上限的最近会话归档快照
  - 显式 `DELETE` 会话后，recent/status 仍可查看最近状态
  - janitor 回收超时会话后，recent/status 仍可查看最近状态
- 为新的会话治理能力补齐后端单元测试与 HTTP handler 测试
- 同步仓库文档：
  - `README.md`
  - `CLAUDE.md`
  - `.cursor/rules/project-facts.mdc`
  - `docs/architecture-and-implementation.html`
  - `docs/plan/2026-04-01-player-pipeline-evolution-plan.md`

## API 变化

### `GET /api/playback/sessions/recent`

返回最近的播放会话快照列表，既包含当前 active session，也包含最近被停止或过期回收的会话。

关键字段：

- `sessionId`
- `movieId`
- `sessionKind`
- `transcodeProfile`
- `state`
- `startedAt`
- `lastAccessedAt`
- `expiresAt`
- `finishedAt`
- `lastError`

### `GET /api/playback/sessions/{id}`

返回单个播放会话的状态快照。优先命中当前活动会话；若会话已被显式删除或 janitor 回收，则回落到最近归档快照。

## 验证

建议在合并前保留以下验证结果：

```bash
pnpm lint
pnpm typecheck
pnpm test
pnpm build
cd backend && go test ./...
```

## 风险与边界

- recent 会话归档目前是进程内有上限缓存，不写入 SQLite
- 这次仍未覆盖 ffmpeg stderr 落盘与 profile 命中统计
- 前端尚未新增独立的播放诊断页面，当前仍以播放页统计面板与后端诊断接口为主

## 下一阶段建议

- 增加 ffmpeg stderr 归档与结构化错误分类
- 在前端增加播放链路诊断入口，直接展示 recent/status API
- 把 `hls.js` 从 CDN 动态加载改为本地打包懒加载

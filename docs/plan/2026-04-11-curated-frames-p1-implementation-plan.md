# 萃取帧 P1 改进实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将萃取帧从全量小图库访问改为可分页、可统计、可缩略图展示、可二进制上传的 P1 版本。

**Architecture:** 后端继续以 SQLite `curated_frames` 为事实源，新增查询 DTO、统计/聚合接口、缩略图 BLOB 与 multipart 上传解析；前端继续通过 `src/lib/curated-frames/db.ts` 屏蔽 Web API / Mock 差异，并把 Web API 统计从“全量列表长度”改为专用 stats 接口。设置页只做语义修正，不引入新的持久化配置。

**Tech Stack:** Go `net/http` + SQLite；Vue 3 + TypeScript + Vite；Vitest；pnpm；Go test。

---

## Scope

- [ ] 新增后端 `GET /api/curated-frames` 查询参数：`q`、`actor`、`movieId`、`tag`、`limit`、`offset`，响应追加 `total/limit/offset`，保持 `items` 兼容。
- [ ] 新增后端 `GET /api/curated-frames/stats`，返回总数。
- [ ] 新增后端 `GET /api/curated-frames/tags` 与 `GET /api/curated-frames/actors`，返回本萃取帧库内聚合候选。
- [ ] 新增后端 `GET /api/curated-frames/{id}/thumbnail`，入库时为 Web API 模式生成缩略图；历史数据缺缩略图时回退原图。
- [ ] 新增后端 multipart 创建路径，保留旧 JSON base64 路径以兼容旧前端。
- [ ] 前端 `api` / `types` 接入新接口，`save-capture` 改用 `FormData` 上传。
- [ ] 前端萃取帧页 Web API 模式使用后端查询与缩略图；Mock 模式保留 IndexedDB 本地过滤。
- [ ] 前端侧边栏与设置页统计改用 stats 接口，避免全量列表计数。
- [ ] 设置页文案改为“应用库 + 附加输出”的语义，降低保存位置误解。
- [ ] 同步新增接口到 `.cursor/rules/project-facts.mdc`、`.cursor/rules/backend-api-contracts.mdc`、`README.md`、`CLAUDE.md`、`docs/architecture-and-implementation.html`。

## Verification

- [ ] `cd backend && go test ./internal/storage ./internal/server ./internal/curatedexport`
- [ ] `pnpm test -- --run src/lib/curated-frames/save-capture.test.ts src/lib/player-route.test.ts`
- [ ] `pnpm build`

## Notes

- P1 中“缩略图模型”先落为同表 `thumb_blob`，避免本轮引入文件资产迁移；后续可再迁移到统一资产服务。
- P1 中“保存方式语义”先改文案与设置页说明，不拆新增配置字段，避免扩大范围。

# 2026-04-11 萃取帧 P2 优化实施计划

## 范围

本轮只实现评审文档里的 P2 项，不扩展到 P3 素材库组织能力。

1. 标签编辑恢复为自动保存体验：详情弹窗展示保存状态，保存失败时保留当前草稿并提供重试；关闭、按标签筛选、从帧播放前都走同一套提交逻辑。
2. 导出元数据补齐：WebP EXIF `UserComment` 与 PNG `CuratedMeta` iTXt 统一增加 `tags`、`schemaVersion`、`exportedAt`、`appName`、`appVersion`，为后续导入和兼容升级保留版本语义。
3. 近重复帧治理：截图保存链路不再拦截同片近重复帧；萃取帧库页面对同一 `movieId` 且 `positionSec` 在 3 秒阈值内的帧分组标记，提醒用户复核并删除多余项。

## 契约

- `POST /api/curated-frames` 继续兼容旧版 JSON `{ ...metadata, imageBase64 }` 与新版 `multipart/form-data`（`metadata` JSON + `image` 文件），但不再返回 `409 CURATED_FRAME_DUPLICATE` 阻断近重复保存。
- Web API 与 Mock/IndexedDB 保存路径都应允许同片 3 秒阈值内的近重复帧写入，避免在截图动作中打断用户。
- 萃取帧库页基于当前已加载行进行近重复分组和标记；同一 `movieId` 且相邻 `positionSec` 差值小于等于 3 秒的记录归入同一组。
- 导出元数据 schema version 固定为 `1`，应用名固定为 `Curated`，应用版本使用后端 `version.Display()`。
- 标签保存失败不关闭详情弹窗，不丢弃 `dialogTags` 草稿；重试成功后再允许关闭、筛选或跳转。

## 测试计划

- 后端 `server`：覆盖同片近重复萃取帧保存仍返回成功、列表中保留两条记录。
- 后端 `curatedexport`：扩展元数据 JSON round-trip，覆盖 `tags`、`schemaVersion`、`exportedAt`、`appName`、`appVersion`。
- 前端 `vitest`：覆盖保存链路不调用近重复预检、近重复分组 helper、标签自动保存 helper。
- 验证命令：`go test ./internal/storage ./internal/server ./internal/curatedexport`、`pnpm test`、`pnpm build`。

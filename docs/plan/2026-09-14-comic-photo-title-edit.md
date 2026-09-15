# 漫画 / 写真媒体信息标题可编辑与翻译

日期：2026-09-14  
需求：REQ-0053

## 当前事实

共享弹窗 `BookMediaInfoDialog` 可编辑展示标题，源文件名、路径、页数只读。漫画/写真独立 `user_title`（迁移 `0050`）；列表与详情 `title` 优先 overlay。扫描继续更新来源文件名标题，不覆盖 overlay。漫画 PATCH `title` 与写真新增 PATCH `title` 写入 overlay；空标题拒绝，上限 500 Unicode 标量。Agent `translate_title` 三选一 `movieId` XOR `comicId` XOR `photoId`，确认走 `update_comic_title` / `update_photo_title`。提示词为 `agent-system-v6`。

## 目标

在详情检查面的「媒体信息」里编辑展示标题，并在 Agent 开启时提供与影片同款的标题翻译预览。源文件名、路径、页数保持只读。扫描继续更新来源 `title`（文件名），不覆盖用户展示标题。

## 设计框

| 项 | 决定 |
| --- | --- |
| 产品面 | 详情检查面。主任务仍是查看源文件信息；标题编辑和翻译是次级操作。 |
| 先例 | 影片 `MovieEditDialog` 标题框 + `translate_title` 预览确认。不把媒体信息做成第二套完整编辑器。 |
| 层级 | 弹窗标题「媒体信息」→ 可编辑展示标题（可选 AI 翻译）→ 只读源文件事实。路径与页数不参与编辑。 |
| 状态 | 空标题拒绝保存；保存中禁用；保存失败就地提示；AI 未配置 / 失败 / 无需翻译 / 预览确认 / 放弃；窄屏沿用现有 Dialog。 |
| 系统影响 | 漫画、写真独立 `user_title` 列与 PATCH；Agent 增加书库标题写工具。不写入影片表，不经 `useLibraryService()`。 |

## 合约

- 迁移 `0050_comic_photo_user_title.sql`：`comic_books.user_title`、`photo_books.user_title`
- 列表/详情 `title` = `COALESCE(NULLIF(TRIM(user_title), ''), title)`
- 扫描继续更新来源 `title`，不改 `user_title`
- 漫画现有 PATCH `title` 改为写入 `user_title`（空标题拒绝）；写真 PATCH 增加同样的 `title`
- 上限 500 个 Unicode 标量
- Agent：`update_comic_title` / `update_photo_title`（preview→confirm）；`translate_title` 三选一 `movieId` XOR `comicId` XOR `photoId`；`translate_summary` 仍只服务影片

## 明确不做

- 不给书库做简介/摘要翻译
- 不把书 ID 写入 `update_movie_display_overrides` 或 `present_movies`
- 不在漫画「编辑」弹窗重复做翻译按钮（同一展示标题仍可在该处手工改）
- 不提供一键清除覆盖的独立按钮；空标题拒绝保存

## 验证

- 仓储：写入覆盖、重扫保留、搜索命中展示标题与来源标题
- HTTP：漫画/写真 PATCH 标题、空标题、越长、缺书、Beta 关闭
- Agent：书库标题预览不落库；确认写入；`translate_title` 接受 comicId/photoId
- 前端：媒体信息可保存标题；AI 开时翻译预览；Mock 写真标题跨刷新保留

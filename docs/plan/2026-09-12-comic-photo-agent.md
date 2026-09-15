# 漫画 / 写真接入 Agent

日期：2026-09-12  
需求：REQ-0052

## 当前事实

漫画库或写真库启用后，Agent 已接入该库：页面上下文、`@` 提及、`search_*` / `get_*_detail` / `present_*`、SSE `book_cards`、`save_*_comment` 预览确认、详情页 `polish_comment`。提示词当时为 `agent-system-v5`。未启用的库返回 `COMIC_LIBRARY_DISABLED` / `PHOTO_LIBRARY_DISABLED`。Insights、源站、Saved Views、萃取帧、评分写工具仍只服务影片。展示标题编辑与 `translate_title` 的书库接入见 [2026-09-14-comic-photo-title-edit.md](2026-09-14-comic-photo-title-edit.md)（`agent-system-v6`）。

## 目标

漫画库或写真库 **启用后**，Agent 按影片库同款能力接入该库：当前页上下文、`@` 引用、搜索 / 详情、聊天卡片、个人笔记预览写入、详情页 AI 润色。未启用的库保持关闭，工具返回 `COMIC_LIBRARY_DISABLED` / `PHOTO_LIBRARY_DISABLED`。

## 设计框

| 项 | 决定 |
| --- | --- |
| 产品面 | Agent Window + 详情检查面。主任务仍是浏览/阅读；Agent 是并列助手。 |
| 先例 | 镜像影片：`search_*` / `get_*_detail` / `present_*` / `save_*_comment` / `polish_comment`。 |
| 层级 | 卡片点击进对应详情。书卡不进封面列。影片工具与书工具并存，互不顶替。 |
| 状态 | Beta 关闭时工具失败并说明范围；混合问题处理已启用的库；未启用部分简短说明。 |
| 系统影响 | 独立 `BookRefStore` 与书卡 DTO；不把书 ID 写入 `MovieRefStore`。读写仍走漫画/写真服务，不经 `useLibraryService()`。 |
| Agent | 提示词当时升到 `agent-system-v5`。源站检索、洞察、萃取帧、Saved Views 仍只服务影片。展示标题翻译已改由 REQ-0053 接入书库。 |

## 范围内

- 页面上下文：`comics` / `comic-detail` / `comic-reader` / `photos` / `photo-detail` / `photo-viewer`，以及 `q` / `tag` / 漫画 `favorite` / `readStatus`
- `@` 提及：已加载的漫画 / 写真标题与标签（该库已启用时）
- 只读：`search_comics`、`get_comic_detail`、`search_photos`、`get_photo_detail`；`get_library_overview` 在启用时附带册数；`get_task_status` 可读对应扫描 / 导入任务
- 呈现：`present_comics` / `present_photos`，SSE `book_cards`
- 写入预览：`save_comic_comment` / `save_photo_comment`；Action `polish_comment` 接受 `comicId` 或 `photoId`

## 明确不做

- 书库刮削、源站页、演员实体、Personal Insights、Saved Views、萃取帧
- 写真收藏 / 阅读进度（产品写路径不完整）
- 把影片统计说成全媒体统计
- 把书 ID 授信给 `present_movies` / 影片 answerRefs

## 验证

- 后端：启用 / 关闭 Beta、搜索投影不含路径、卡片需本轮读取、评论预览冲突、任务白名单
- 前端：书库路由上下文、`@` 提及、卡片导航、润色预览；Mock 漫画/写真请求走书工具而不是拒答

# 漫画 / 写真详情页个人评论

日期：2026-09-12  
需求：REQ-0050

## 当前事实

影片详情页有一块个人备注：中文标题「我的评论」，英文为 My notes。每部影片一条，自动保存，最多 10000 个 Unicode 字符。Web 走 `GET/PUT /api/library/movies/{id}/comment`，Mock 走 `jav-library-movie-comment-v1`。影片备注还带可选的 Agent 润色。

漫画和写真详情页目前没有对等能力。两个库与影片域隔离，Agent 仍只处理影片。

## 目标

在漫画详情和写真详情复用同一块检查面：标题、输入框、字数、自动保存状态与影片页一致。不新增第二块「笔记」卡片。

## 设计框

| 项 | 决定 |
| --- | --- |
| 产品面 | 详情检查面；主任务仍是打开阅读/浏览，备注是次要记录。 |
| 先例 | `MovieCommentSection` 的卡片、textarea、800ms 防抖自动保存。 |
| 层级 | 元数据面板、页面预览之后。 |
| 状态 | 加载、空正文、过长、保存中、已自动保存、加载/保存失败。漫画/写真没有回收站只读态。 |
| 系统影响 | `books/BookCommentSection` 共享展示；读写仍走各自服务契约。 |
| Agent | 不提供润色、不增加工具、不采集漫画/写真备注。 |

## 合约

- `GET/PUT /api/library/comics/books/{comicId}/comment`
- `GET/PUT /api/library/photos/books/{photoId}/comment`
- DTO：`{ body, updatedAt }`；空备注返回空 `body` 与空 `updatedAt`
- 正文 trim 后最多 10000 个 Unicode 标量
- Beta 关闭：`400 COMIC_LIBRARY_DISABLED` / `PHOTO_LIBRARY_DISABLED`
- 书不存在：`404 COMIC_BOOK_NOT_FOUND` / `PHOTO_BOOK_NOT_FOUND`
- 过长：`400 COMMON_BAD_REQUEST`

独立表：

- `comic_book_comments`（`comic_id` PK，删除漫画时级联）
- `photo_book_comments`（`photo_id` PK，删除写真时级联）

Mock 使用 `curated-mock-comic-comments-v1` 与 `curated-mock-photo-comments-v1`。

## 明确不做

- 不把漫画/写真备注写入 `library_movie_comments`
- 不给 Agent 读写或润色这两类备注
- 不实现写真删除/回收站只读备注

## 验证

已完成：

- 仓储：空值、覆盖、过长、缺失书、删除漫画后备注消失
- HTTP：成功读写、禁用 Beta、缺失书、过长
- 前端：组件自动保存；详情页挂载评论区；Web/Mock 适配器
- `go test ./...`、`go vet ./...`、聚焦 Vitest、`pnpm lint` 通过

未完成：本机 PIN App Lock 已锁定，未做已解锁浏览器端到端点击。

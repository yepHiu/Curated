# 漫画 / 写真详情页本地评分

日期：2026-09-12  
需求：REQ-0051

## 当前事实

影片、漫画、写真详情都把评分卡放在信息列标签下方，固定 250px 宽（窄屏 `max-w-full`），不再拉满封面列或信息列。漫画继续使用 `comic_books.user_rating` 与 `PATCH /api/library/comics/{id}`。写真使用已有 `photo_books.user_rating`，并通过 `PATCH /api/library/photos/{id}` 写入或清除。书籍共用 `BookRatingCard`。

## 目标

漫画和写真详情复用影片评分卡的样式。评分跟在标签后面，卡片固定 250px 宽；封面大小和比例不固定，所以仍不进封面列。只有本地用户分，没有站点分。

## 设计框

| 项 | 决定 |
| --- | --- |
| 产品面 | 详情检查面；主任务仍是开始阅读/浏览。 |
| 先例 | 评分卡视觉复用 `DetailPanel` + `MovieRatingStars`。影片、漫画、写真共用「标签下方、250px 宽」的放置。 |
| 层级 | 封面仍是左列媒体；标题 → 主操作 → 事实/标签 → 评分卡。评分是次要检查控件，不进封面列，也不拉满信息列。 |
| 状态 | 未评、已评、忙碌禁用、保存失败（页面级错误）。窄屏先封面后信息列，评分仍跟在标签后面。 |
| 系统影响 | 共享 `books/BookRatingCard`；读写仍走各自服务契约。不提升为全局详情布局规则。 |
| Agent | 不增加漫画/写真评分工具。 |

## 合约

- 漫画：继续 `PATCH /api/library/comics/{comicId}`，`ratingSet` / `ratingClear` / `rating`
- 写真新增：`PATCH /api/library/photos/{photoId}`，同样的评分字段
- 范围 0–5；`null` 清除为未评
- Beta 关闭：`400 COMIC_LIBRARY_DISABLED` / `PHOTO_LIBRARY_DISABLED`
- 书不存在：`404 COMIC_BOOK_NOT_FOUND` / `PHOTO_BOOK_NOT_FOUND`

## 明确不做

- 不引入站点/元数据评分双轨
- 不把漫画/写真评分写入 `movies.user_rating`
- 不给 Agent 读写这两类评分
- 不把评分卡改回封面列或拉满信息列宽度

## 验证

- 仓储与 HTTP：设置、清除、越界、缺书、Beta 关闭
- 前端：标签下方、250px 宽评分卡；半星写入；清除；Web/Mock 适配器
- 已跑：聚焦 Vitest、`pnpm typecheck`、`pnpm lint`、`go test ./...`、`go vet ./...`

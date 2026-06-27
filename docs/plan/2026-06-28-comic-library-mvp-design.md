# Curated 漫画库 MVP 设计方案

日期：2026-06-28  
状态：需求讨论已确认，待用户审阅文档后再进入实施计划

## 1. 背景与目标

Curated 当前核心媒体库面向影片。新需求是在 Curated 中加入一个可选的漫画库模块，用于管理以 `.zip` / `.cbz` 形式保存的漫画压缩包。压缩包内部通常包含 `.jpg`、`.jpeg`、`.png`、`.webp`、`.gif` 等图片页；排序后的第一张有效图片作为默认封面。

漫画库不是影片库的扩展类型，而是 Curated 中的独立媒体域。第一版目标不是原型，而是完整可用 MVP：用户可以启用漫画库、配置漫画路径、导入或扫描漫画 zip/cbz、浏览漫画墙、查看详情与页预览、进入阅读器阅读、保存阅读进度、维护标签/评分/收藏，并管理漫画缩略图缓存。

## 2. 核心原则

- 漫画库是可选模块，默认未启用。
- 未启用时，不显示任何漫画入口：侧栏、顶栏导入菜单、漫画路由页面、设置外显入口都应保持隐藏或被守卫。
- 启用漫画库必须完成初始化：至少配置一个漫画存储路径后才算启用成功。
- 漫画和影片保持强独立：不混表、不混库、不把漫画 zip/cbz 伪装成 movie，不复用影片扫描器、影片标签、影片播放进度、影片回收站或影片刮削逻辑。
- 可以复用底层基础设施：任务系统、HTTP 上传/分片上传、存储在线检测、toast/通知、设置页 UI 范式、虚拟网格思路、缓存清理框架。
- 业务类型、服务契约、API、数据表和前端路由应保持漫画域独立。

## 3. MVP 范围

第一版需要包含：

- Settings 中独立的“漫画库”配置区。
- 漫画库启用初始化流程。
- 独立漫画存储路径和默认漫画导入路径。
- `.zip` / `.cbz` 扫描。
- zip 内图片页索引持久化。
- 漫画墙。
- 漫画详情页。
- 默认前 12 页预览，以及完整页缩略图视图。
- 沉浸式漫画阅读器。
- 左右翻页与上下滚动。
- 自适应与适宽图片适配。
- 左到右 / 右到左阅读方向。
- 临时跨页拼接工具。
- 阅读进度与已读状态。
- 收藏与用户评分。
- 漫画标签，且标签与影片标签完全隔离。
- 标签前缀建议，例如 `作者:`、`系列:`、`卷:`、`社团:`。
- 顶栏导入菜单中的“导入漫画”。
- 漫画缩略图缓存，自动按大小上限清理并支持手动清理。
- Mock 模式下的漫画 UI 示例和本地偏好演示。

第一版不做：

- 外部元数据抓取。
- OCR 或图片内容识别。
- `.rar` / `.cbr` / `.7z`。
- 删除源 zip/cbz。
- 漫画回收站。
- 自动监听漫画路径。
- 独立作者表、系列表、演员式实体库。
- 双页常驻阅读模式。第一版只做临时跨页拼接。
- 手柄控制。
- 跨设备同步。

## 4. 产品信息架构

### 4.1 侧栏

启用漫画库后，侧栏 Browse 分组新增“漫画库”入口。未启用时不渲染该入口。

第一版不新增“漫画收藏”“漫画历史”“漫画标签”等侧栏入口。收藏、已读/未读、标签筛选都收在漫画库页面内部。

### 4.2 顶栏导入

未启用漫画库时，顶栏可以保持接近当前体验，继续展示“导入影片”。

启用漫画库后，顶栏导入入口升级为“导入”菜单：

- 导入影片。
- 导入漫画。

“导入漫画”打开独立 `ComicImportDialog`，只接受 `.zip` / `.cbz`，目标路径来自 `defaultComicImportLibraryPathId`。

### 4.3 设置页

Settings 新增独立 section：漫画库。漫画设置不放入现有影片存储，也不放入 Playback。

漫画库 section 包含：

- 启用/禁用漫画库。
- 漫画存储路径。
- 默认漫画导入路径。
- 手动扫描漫画库。
- 阅读器默认偏好。
- 漫画缓存状态、上限和清理操作。

## 5. 后端数据模型

漫画数据使用独立表族，与 `movies`、`movie_tags`、`playback_progress` 等影片表完全隔离。

### 5.1 `comic_library_paths`

保存漫画存储路径：

- `id`
- `path`
- `title`
- `first_scan_pending`
- `created_at`
- `updated_at`

`defaultComicImportLibraryPathId` 指向这张表中的路径，不复用影片 `defaultImportLibraryPathId`。

### 5.2 `comic_books`

每个 `.zip` / `.cbz` 一个漫画条目：

- `id`
- `title`
- `source_file_name`
- `location`
- `file_size`
- `file_modified_at`
- `file_fingerprint`
- `page_count`
- `cover_page_index`
- `is_favorite`
- `user_rating`
- `read_status`
- `added_at`
- `updated_at`

标题默认来自文件名去扩展名，用户可编辑。第一版不暴露作者、系列、简介、发行日期等独立字段；作者和系列先用标签前缀承载。

### 5.3 `comic_pages`

扫描阶段持久化页索引：

- `id`
- `comic_id`
- `page_index`
- `entry_path`
- `normalized_dir`
- `file_name`
- `file_size`
- `crc32`
- `modified_at`
- `image_format`
- `created_at`

建议唯一约束：

- `(comic_id, page_index)`
- `(comic_id, entry_path)`

页序按目录层级 + 文件名自然排序。封面页是排序后的第一张有效图片。

### 5.4 `comic_tags` / `comic_book_tags`

漫画标签独立于影片标签。底层第一版不区分标签类型；前端输入提供结构化前缀建议。搜索和筛选按普通标签匹配。

### 5.5 `comic_reading_progress`

记录阅读状态：

- `comic_id`
- `current_page_index`
- `page_count`
- `completed`
- `completed_at`
- `updated_at`

阅读器翻页、滚动到新页和退出时节流保存进度。读到最后一页时标记已读。用户可以显式重置进度或标记未读。

### 5.6 `comic_reading_preferences`

保存单本漫画覆盖全局默认的阅读偏好：

- `comic_id`
- `reading_mode`：`page` / `scroll`
- `fit_mode`：`contain` / `width`
- `reading_direction`：`ltr` / `rtl`
- `updated_at`

如果单本偏好不存在，使用 Settings 中的全局默认。

### 5.7 `comic_cache_entries`

记录缩略图缓存元数据：

- `id`
- `comic_id`
- `page_index`
- `kind`：`cover` / `page_thumb`
- `width`
- `height`
- `cache_path`
- `source_fingerprint`
- `file_size`
- `last_accessed_at`
- `created_at`

缓存独立于影片资产缓存。

## 6. 配置持久化

配置继续写入 `config/library-config.cfg`，使用漫画专属 key：

- `comicLibraryEnabled`
- `defaultComicImportLibraryPathId`
- `comicReader`
  - `readingMode`
  - `fitMode`
  - `readingDirection`
- `comicCache`
  - `maxBytes`
  - `unlimited`

漫画缓存默认上限 2GB，设置页提供：

- 1GB
- 2GB
- 5GB
- 10GB
- 不限制

禁用漫画库只设置 `comicLibraryEnabled=false`，不删除路径、索引、页索引、标签、评分、收藏、阅读进度、单本偏好或缓存。

## 7. 扫描流程

漫画扫描器独立实现，只扫描 `comic_library_paths`。

候选文件：

- `.zip`
- `.cbz`

zip 内有效图片：

- `.jpg`
- `.jpeg`
- `.png`
- `.webp`
- `.gif`

忽略：

- 目录项。
- 系统文件。
- 文本、URL、数据库文件。
- 嵌套压缩包。
- PDF。
- 视频。

扫描单个 zip/cbz 时：

1. 读取 zip central directory。
2. 过滤有效图片页。
3. 按目录层级 + 文件名自然排序。
4. 持久化 `comic_books` 和 `comic_pages`。
5. 设置封面页为排序后的第一张有效图片。
6. 不在扫描阶段全量生成缩略图。

如果 zip 无有效图片，扫描结果跳过或失败，并在任务结果里给出可读错误。

第一版不做自动监听。扫描来源只有：

- Settings -> 漫画库手动扫描。
- 导入漫画完成后自动触发扫描。

## 8. 导入流程

漫画导入使用独立 API 和任务类型，不复用影片导入语义。

建议 API：

- `POST /api/import/comics`
- 大文件可使用独立分片路由：
  - `POST /api/import/comics/uploads`
  - `PUT /api/import/comics/uploads/{uploadId}/files/{fileId}/chunks/{chunkIndex}`
  - `GET /api/import/comics/uploads/{uploadId}`
  - `POST /api/import/comics/uploads/{uploadId}/commit`
  - `DELETE /api/import/comics/uploads/{uploadId}`

导入规则：

- 目标路径为 `defaultComicImportLibraryPathId` 指向的漫画路径。
- 仅接受 `.zip` / `.cbz`。
- 保留用户选择的相对路径/文件名。
- 如果目标路径已有同名文件，跳过该文件。
- 不覆盖、不自动重命名、不计算内容 hash 阻塞导入。
- 导入复制成功后自动触发漫画扫描。
- 扫描失败不回滚已复制文件，只在任务结果记录 `scanError`。

## 9. 页读取与缓存

前端不直接访问 zip 内 entry path。所有页图片通过后端按漫画 id + 页码读取。

建议 API：

- `GET /api/library/comics/{comicId}/pages`
- `GET /api/library/comics/{comicId}/pages/{pageIndex}/image`
- `GET /api/library/comics/{comicId}/pages/{pageIndex}/thumbnail`
- `GET /api/library/comics/{comicId}/asset/cover`

读取前校验：

- 漫画源文件仍在漫画库路径下。
- 页索引属于该漫画。
- 请求页码存在。

缩略图请求优先命中 `comic_cache_entries`；未命中则从 zip 读取原图生成缩略图，写入缓存并登记缓存项。完整原图通常按需从 zip 读取，不把全部原图展开到缓存目录。

缓存清理：

- 每次缩略图命中或生成时更新 `last_accessed_at`。
- 超过用户设置上限时，按最久未访问清理缓存。
- 手动清理漫画缓存只删除缩略图缓存。
- 清理缓存不删除源 zip/cbz、不删除索引、不删除标签、评分、收藏或进度。

## 10. API 设计

### 10.1 设置与路径

- `GET /api/settings` 增加漫画字段：
  - `comicLibraryEnabled`
  - `comicLibraryPaths`
  - `defaultComicImportLibraryPathId`
  - `comicReader`
  - `comicCache`
- `PATCH /api/settings` 支持更新漫画启用状态、默认导入路径、阅读器默认偏好、缓存上限。
- `POST /api/library/comics/paths`
- `PATCH /api/library/comics/paths/{id}`
- `DELETE /api/library/comics/paths/{id}`
- `POST /api/library/comics/paths/{id}/reveal`
- `GET /api/library/comics/paths/storage-status`
- `POST /api/library/comics/paths/storage-status/check`
- `POST /api/library/comics/paths/{id}/storage-binding/rebind`

### 10.2 列表、详情与编辑

- `GET /api/library/comics`
  - 参数：`q`、`tag`、`favorite`、`readStatus`、`limit`、`offset`
  - 搜索覆盖标题、标签、源文件名、路径。
- `GET /api/library/comics/{comicId}`
- `PATCH /api/library/comics/{comicId}`
  - 支持 `title`、`tags`、`isFavorite`、`rating`。
- `DELETE /api/library/comics/{comicId}`
  - 只移除索引，不删除源 zip/cbz。
- `POST /api/library/comics/{comicId}/reveal`

### 10.3 页与阅读

- `GET /api/library/comics/{comicId}/pages`
- `GET /api/library/comics/{comicId}/pages/{pageIndex}/image`
- `GET /api/library/comics/{comicId}/pages/{pageIndex}/thumbnail`
- `GET /api/library/comics/{comicId}/asset/cover`
- `GET /api/library/comics/{comicId}/progress`
- `PUT /api/library/comics/{comicId}/progress`
- `DELETE /api/library/comics/{comicId}/progress`
- `GET /api/library/comics/{comicId}/reading-preferences`
- `PUT /api/library/comics/{comicId}/reading-preferences`

### 10.4 扫描、导入与缓存

- `POST /api/library/comics/scans`
- `POST /api/import/comics`
- `POST /api/import/comics/uploads`
- `GET /api/library/comics/cache/status`
- `POST /api/library/comics/cache/cleanup`

## 11. 任务与错误码

任务类型：

- `scan.comics`
- `import.comics`
- `comic.cache.cleanup`

漫画错误码独立命名：

- `COMIC_LIBRARY_DISABLED`
- `COMIC_PATH_NOT_CONFIGURED`
- `COMIC_PATH_UNAVAILABLE`
- `COMIC_IMPORT_TARGET_NOT_CONFIGURED`
- `COMIC_IMPORT_UNSUPPORTED_FILE`
- `COMIC_IMPORT_CONFLICT`
- `COMIC_IMPORT_COPY_FAILED`
- `COMIC_SCAN_FAILED`
- `COMIC_ARCHIVE_OPEN_FAILED`
- `COMIC_ARCHIVE_EMPTY`
- `COMIC_PAGE_NOT_FOUND`
- `COMIC_SOURCE_FILE_MISSING`
- `COMIC_CACHE_GENERATION_FAILED`

现有 PIN/Auth 中间件继续保护 `/api/*`。漫画 API 不新增用户系统或权限模型。

## 12. 前端架构

前端新增漫画专属 domain、服务契约和 adapter，不把漫画塞进现有 `LibraryService` 的 movie 方法。

建议文件：

- `src/domain/comic/types.ts`
- `src/services/contracts/comic-library-service.ts`
- `src/services/comic-library-service.ts`
- `src/services/adapters/web/web-comic-library-service.ts`
- `src/services/adapters/mock/mock-comic-library-service.ts`

建议核心类型：

- `ComicBook`
- `ComicPage`
- `ComicReadingProgress`
- `ComicReaderPreferences`
- `ComicReaderMode`
- `ComicFitMode`
- `ComicReadingDirection`

新增路由：

- `/comics` -> `ComicsView.vue`
- `/comics/:id` -> `ComicDetailView.vue`
- `/comics/:id/read` -> `ComicReaderView.vue`

如果 `comicLibraryEnabled=false`，漫画路由由路由守卫重定向到 Settings -> 漫画库启用区。

## 13. 漫画库页面

漫画库页面使用独立组件：

- `ComicsView.vue`
- `ComicLibraryPage.vue`
- `VirtualComicGrid.vue` 或 `VirtualComicMasonry.vue`
- `ComicCard.vue`

漫画卡片展示：

- 封面缩略图。
- 标题。
- 标签摘要。
- 用户评分。
- 收藏状态。
- 阅读进度条。
- 已读标记。
- 页数。

页面内部筛选：

- 全部。
- 收藏。
- 未读。
- 已读。
- 标签筛选。

顶栏搜索在漫画库路由下切换为漫画搜索，只使用漫画数据源。

## 14. 漫画详情页

详情页遵循现有媒体详情模式：左封面，右信息，不做大型 hero。

主要操作：

- 开始阅读。
- 继续阅读。
- 从第一页重新阅读。
- 在文件管理器中显示。
- 从漫画库移除索引。

可编辑字段：

- 标题。
- 标签。
- 用户评分。
- 收藏。

标签输入提供前缀建议：

- `作者:`
- `系列:`
- `卷:`
- `社团:`

信息展示：

- 页数。
- 文件名。
- 文件大小。
- 入库时间。
- 源路径。
- 阅读进度。
- 已读状态。

预览：

- 默认展示前 12 页缩略图。
- 点击某页从该页进入阅读器。
- “查看全部页”打开完整页缩略图视图。
- 完整页视图使用虚拟滚动，按需请求缩略图。

## 15. 漫画阅读器

阅读器以图片为主，控件采用沉浸式悬浮。

阅读模式：

- `page`：左右翻页，默认。
- `scroll`：上下连续滚动。

图片适配：

- `contain`：自适应窗口，完整显示当前页。
- `width`：适宽。

阅读方向：

- `ltr`：左到右。
- `rtl`：右到左。

右到左阅读不反转 zip 页索引。页码始终按扫描顺序保存和显示，只改变阅读动作语义与临时拼页的左右摆放。

左右翻页模式：

- `ltr`
  - 右箭头 / 点击右侧热区：下一页。
  - 左箭头 / 点击左侧热区：上一页。
- `rtl`
  - 右箭头 / 点击右侧热区：上一页。
  - 左箭头 / 点击左侧热区：下一页。
- `Space`：按当前方向推进。
- `Home` / `End`：第一页 / 最后一页。
- `Esc`：返回详情页或进入阅读器前的来源。

上下滚动模式：

- 页序仍从第 1 页向后自然排列。
- 保留浏览器自然滚动。
- `Space` 可向下推进一屏。
- 进度根据当前可见页或滚动位置估算并节流保存。

临时跨页拼接：

- 只在当前阅读会话生效。
- 不写数据库。
- 不修改 zip/cbz。
- 不修改页索引。
- 退出阅读器后清空。
- 用户可以选择：
  - 当前页 + 上一页。
  - 当前页 + 下一页。
  - 取消当前拼页。
- 拼接后翻页在当前会话中跳过被合并的页，避免重复显示。
- `ltr`：较小页码在左，较大页码在右。
- `rtl`：较大页码在左，较小页码在右。

阅读偏好：

- 阅读模式、适配方式、阅读方向支持全局默认。
- 阅读器内切换后保存为单本偏好。
- 临时拼页不保存。

## 16. Mock 模式

Mock 模式只提供漫画 UI 示例，不模拟完整 zip 读取。

Mock 下可以：

- 显示漫画库入口。
- 展示漫画墙。
- 展示漫画详情页。
- 打开阅读器占位页。
- 使用 localStorage 保存收藏、评分、阅读进度和阅读偏好。

Mock 下不做：

- zip 解析。
- 真实文件导入。
- 真实扫描。
- 真实缩略图缓存统计。

## 17. 测试策略

### 17.1 Go 单测

- zip 页过滤与自然排序。
- 空 zip、无图片 zip、损坏 zip 错误。
- comic migrations。
- comic repository CRUD。
- comic settings merge/write。
- comic path storage status。
- scan task metadata。
- import conflict / unsupported file / target unavailable。
- page image and thumbnail handlers。
- cache eviction。

### 17.2 前端单测

- comic service boundary。
- route guard enabled/disabled。
- sidebar conditional entry。
- import menu enabled/disabled。
- comic search query。
- comic card progress/read status。
- detail edit tags/rating/title。
- reader LTR/RTL keyboard mapping。
- temporary stitch behavior。
- reader preference fallback and override。

### 17.3 集成与浏览器验证

- 启用流程。
- 扫描真实小 zip。
- 漫画墙显示。
- 详情预览。
- 阅读器翻页、滚动、RTL、临时拼页。
- 导入 zip 后自动扫描。
- 缓存清理不删除源文件。

## 18. 实施阶段

虽然 MVP 目标完整，实施必须拆阶段验收。

### 阶段 1：基础开关、设置与数据地基

包含 SQLite migrations、配置 key、Settings -> 漫画库、漫画路径 CRUD、启用初始化、禁用隐藏入口、漫画服务契约骨架、Mock 示例状态。

验收点：

- 未启用时无漫画入口。
- 启用时必须配置路径。
- 设置能保存并重启后恢复。

### 阶段 2：扫描与页索引

实现独立漫画扫描器，识别 `.zip/.cbz`，过滤图片页，自然排序并持久化 `comic_books` / `comic_pages`。

验收点：

- zip 页数、封面页、标题、页索引正确。
- 空 zip、无图片 zip、损坏 zip 有明确错误。

### 阶段 3：漫画墙与详情

实现漫画列表 API、详情 API、封面/缩略图接口、漫画墙、详情页、标题/标签/评分/收藏编辑、阅读进度展示。

验收点：

- 大列表虚拟滚动不退化。
- 漫画标签和影片标签隔离。
- 搜索覆盖标题、标签、文件名、路径。

### 阶段 4：阅读器

实现阅读器原图页读取、左右翻页、上下滚动、自适应/适宽、LTR/RTL、键盘、沉浸式控件、阅读进度保存、已读状态、单本阅读偏好、临时跨页拼接。

验收点：

- RTL 不反转页索引，只影响动作和拼页摆放。
- 拼页退出后不保存。
- 进度节流可靠。

### 阶段 5：导入与缓存治理

实现顶栏导入菜单、`ComicImportDialog`、`POST /api/import/comics`、可选分片上传、导入后自动扫描、漫画缓存记录、自动按大小上限清理、手动清理。

验收点：

- 目标路径冲突不覆盖。
- 缓存只清缩略图。
- 源 zip/cbz 永不被清理或删除。

### 阶段 6：打磨与文档同步

补齐文案、多语言、错误态、空态、设置说明、README/API/项目记忆同步，并处理性能和显示缩放检查。

## 19. 主要风险

- zip 文件非常大或图片非常多，扫描和缩略图生成必须避免阻塞 UI。
- WebP/GIF 缩略图生成依赖 Go 图像解码能力，需要确认支持范围。
- Windows 路径、zip 内路径和非法字符处理容易出边界问题。
- RTL + 临时拼页需要明确测试，否则很容易出现视觉正确但键盘方向错误。
- 缓存清理必须只触碰漫画缓存目录，不能误删源 zip/cbz。
- 前端不能为了复用把漫画混进 `Movie` 或 `LibraryService`，这应作为 review 检查项。

## 20. 文档同步要求

架构/API 落地后需要同步：

- `.cursor/rules/project-facts.mdc`
- `.cursor/rules/workspace-quick-reference.mdc`，仅在启动、配置或开发操作关键变化时更新。
- `README.md`
- `API.md`
- `CLAUDE.md` API 列表
- `docs/reference/architecture-and-implementation.html`
- 若涉及 `library-config.cfg`，同步 `docs/reference/2026-03-21-library-organize.md`

## 21. 已确认决策摘要

- 存储路径：漫画使用独立漫画库路径。
- 启用方式：启用时必须先配置至少一个漫画路径。
- zip 处理：保留 zip/cbz 作为漫画本体，不导入时解压。
- 标题：第一版只从文件名生成默认标题。
- 元数据：不做抓取，不做 OCR。
- 作者/系列：第一版不做独立字段，用标签前缀承载。
- 标签：漫画标签与影片标签完全隔离。
- 阅读器默认：左右翻页、沉浸式悬浮控件。
- 阅读器模式：左右翻页 / 上下滚动。
- 图片适配：自适应 / 适宽。
- 阅读方向：左到右 / 右到左；不反转页序。
- 跨页：只做当前会话临时拼接，不持久化。
- 进度：保存页码和已读状态。
- 收藏/评分：支持，且独立于影片。
- 格式：支持 `.zip` / `.cbz`。
- 页序：目录层级 + 文件名自然排序。
- 页面预览：默认前 12 页，可打开完整页缩略图视图。
- 删除：只移除索引，不删除源文件。
- Mock：只做 UI 示例，不模拟 zip。
- 自动监听：第一版不做。
- 缓存：默认 2GB，可调整，自动清理 + 手动清理。

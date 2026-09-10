# Curated 写真库 MVP 设计

> 历史实验方案保留。2026-09-10 主线 Beta 的批准范围、当前限制及验收以 [整合记录](2026-09-10-comic-photo-beta-integration.md) 和 REQ-0046 / REQ-0047 为准。

日期：2026-07-05
状态：需求讨论已确认，尚未进入实现计划

## 1. 背景

当前漫画库 MVP 已经形成一套比较完整的“压缩包图片序列库”能力：导入 `.zip` / `.cbz`，解析压缩包内图片，自然排序，使用第一张图片作为封面，生成缩略图，展示媒体墙、详情页、页面预览和沉浸式阅读器。

用户确认写真库可以复用这类底层能力，但产品上希望写真库与漫画库并列，而不是作为漫画库的分类或子入口。

因此写真库应作为 Curated 的独立媒体域，与影片库、漫画库并列存在。写真库第一版面向 `.zip` / `.cbz` 形式的写真集或图集压缩包；未来可以扩展到“图片文件夹图集”，但 MVP 暂不实现文件夹图集导入。

## 2. 产品定位

写真库是一个独立的本地图片集媒体库，用于管理、浏览和维护以压缩包形式保存的写真集。

写真库第一版目标：

- 与影片、漫画并列显示为独立入口。
- 支持启用 / 禁用写真库。
- 支持独立写真存储路径。
- 支持独立导入、扫描和自动监听。
- 支持写真集媒体墙。
- 支持写真集详情页。
- 支持写真集图片预览。
- 支持沉浸式写真集浏览器。
- 支持标题、标签、评分、收藏、浏览进度。
- 保持写真库与漫画库、影片库数据边界独立。

## 3. 命名约定

用户界面：

- 侧边栏入口：`写真`
- 设置页：`写真库`
- 顶栏导入入口：`添加写真`
- 单个条目：`写真集`
- 导入弹窗标题：`导入写真集`
- 详情页：`写真集详情`
- 浏览页：`写真集浏览器`
- 主操作按钮：`浏览` / `继续浏览`
- 进度文案：`浏览进度`

代码建议命名：

- 领域模型：`PhotoBook`
- 页面图片：`PhotoPage`
- 前端路由：
  - `/photos`
  - `/photos/:id`
  - `/photos/:id/view/:pageIndex?`
- 前端视图：
  - `PhotosView`
  - `PhotoDetailView`
  - `PhotoViewerView`
- 前端组件：
  - `PhotoLibraryPage`
  - `PhotoCard`
  - `PhotoPagePreviewGrid`
  - `PhotoViewer`
  - `PhotoImportDialog`
- 服务：
  - `photo-library-service`
  - `web-photo-library-service`
  - `mock-photo-library-service`

## 4. 输入格式

MVP 支持：

- `.zip`
- `.cbz`

压缩包内图片格式与漫画库保持一致，建议包括：

- `.jpg`
- `.jpeg`
- `.png`
- `.webp`
- `.gif`（如漫画库已支持，则写真库保持一致）

MVP 不支持：

- 直接导入散图。
- 直接导入文件夹。
- 把已有普通图片目录自动识别为写真集。

设计预留：

- 未来可以新增 `sourceType` 或等价字段，支持 `archive` 与 `folder` 两类来源。
- 第一版实现只写入 `archive` 类型，避免后续扩展时需要重建核心模型。

## 5. 独立性边界

写真库必须与影片库、漫画库保持强独立。

写真库独立拥有：

- 独立启用开关：`photoLibraryEnabled`
- 独立自动扫描开关：`autoPhotoLibraryWatch`
- 独立默认导入路径：`defaultPhotoImportLibraryPathId`
- 独立阅读 / 浏览偏好：`photoViewer`
- 独立缓存设置：`photoCache`
- 独立路径列表：`photoLibraryPaths`
- 独立数据表族。
- 独立 API。
- 独立 watcher。
- 独立标签、评分、收藏、浏览进度。
- 独立前端路由和服务合约。

写真库不应：

- 复用漫画表存储写真集。
- 复用漫画 API 暴露写真集。
- 复用漫画 watcher 扫描写真路径。
- 把写真压缩包导入漫画库。
- 把写真标签并入漫画标签或影片标签。
- 把写真浏览进度并入漫画阅读进度或影片播放进度。

## 6. 可复用能力

写真库可以复用漫画库已验证过的底层“压缩包图片序列”能力：

- `.zip` / `.cbz` 格式判断。
- 压缩包图片页过滤。
- 图片页自然排序。
- 第一张有效图片作为封面。
- 原图按需读取。
- 缩略图生成。
- 缓存登记与清理。
- 扫描任务流水线。
- 导入任务流水线。
- 沉浸式图片浏览器基础交互。

建议将底层能力逐步抽成中性模块，例如：

- 后端：`imagearchive` 或 `archiveimages`
- 前端：后续可抽 `ArchiveImageViewer`、`ArchiveImagePreviewGrid` 等中性组件

但写真库业务域本身不应直接暴露漫画语义。即使底层暂时从漫画库迁移而来，最终对外命名也应保持写真域独立。

## 7. 数据模型建议

建议新增独立表族：

- `photo_library_paths`
- `photo_books`
- `photo_pages`
- `photo_tags`
- `photo_book_tags`
- `photo_viewing_progress`
- `photo_viewer_preferences`
- `photo_cache_entries`

`photo_books` 建议字段：

- `id`
- `title`
- `source_file_name`
- `location`
- `library_path_id`
- `source_type`：MVP 固定为 `archive`，预留未来 `folder`
- `page_count`
- `cover_page_index`
- `rating`
- `favorite`
- `added_at`
- `updated_at`

`photo_pages` 建议字段：

- `id`
- `photo_book_id`
- `page_index`
- `entry_path`
- `width`
- `height`
- `size_bytes`
- `created_at`

`photo_viewing_progress` 建议字段：

- `photo_book_id`
- `page_index`
- `updated_at`

## 8. 后端 API 建议

设置与路径：

- `GET /api/settings` 返回写真库设置字段。
- `PATCH /api/settings` 更新写真库设置字段。
- `POST /api/library/photos/paths`
- `PATCH /api/library/photos/paths/{id}`
- `DELETE /api/library/photos/paths/{id}`
- `POST /api/library/photos/paths/{id}/reveal`
- `POST /api/library/photos/paths/{id}/scan`
- `GET /api/library/photos/paths/storage-status`
- `POST /api/library/photos/paths/storage-status/check`

写真集：

- `GET /api/library/photos`
- `GET /api/library/photos/{photoBookId}`
- `PATCH /api/library/photos/{photoBookId}`
- `DELETE /api/library/photos/{photoBookId}`
- `POST /api/library/photos/{photoBookId}/reveal`

页面与资源：

- `GET /api/library/photos/{photoBookId}/pages`
- `GET /api/library/photos/{photoBookId}/pages/{pageIndex}/image`
- `GET /api/library/photos/{photoBookId}/pages/{pageIndex}/thumbnail`
- `GET /api/library/photos/{photoBookId}/asset/cover`

进度与偏好：

- `GET /api/library/photos/{photoBookId}/progress`
- `PUT /api/library/photos/{photoBookId}/progress`
- `DELETE /api/library/photos/{photoBookId}/progress`
- `GET /api/library/photos/{photoBookId}/viewer-preferences`
- `PUT /api/library/photos/{photoBookId}/viewer-preferences`

导入与扫描：

- `POST /api/import/photos`
- `POST /api/library/photos/scans`

缓存：

- `GET /api/library/photos/cache/status`
- `POST /api/library/photos/cache/cleanup`

## 9. 前端体验

### 9.1 侧边栏

当写真库未启用时，不显示 `写真` 入口。

当写真库启用时，侧边栏在 `影片`、`漫画` 同级位置显示 `写真`。

### 9.2 顶栏导入

当写真库启用时，导入菜单提供 `添加写真`。

`添加写真` 打开 `PhotoImportDialog`，第一版只接受 `.zip` / `.cbz`。

### 9.3 写真库列表页

列表页参考漫画库与影片库：

- 不显示大标题和说明。
- 搜索框放在 shell 顶栏。
- 内容区展示排序控制。
- 支持按导入时间、文件名、收藏排序。
- 支持批量管理。
- 网格卡片尺寸、间距和影片 / 漫画卡保持一致。
- 卡片展示封面、标题、页数、浏览进度。
- 不展示源文件路径。
- 不展示冗余源文件名徽标。

### 9.4 写真集详情页

详情页第一版与漫画详情页保持一致：

- 标题
- 标签
- 评分
- 封面
- 页面预览图
- `浏览` / `继续浏览` 按钮
- 更多菜单

更多菜单建议包含：

- 编辑写真集信息
- 删除写真集
- 在文件管理器中显示

第一版不做结构化字段：

- 模特
- 摄影师
- 发行方
- 系列
- 日期

这些信息先通过标签表达，例如：

- `模特:xxx`
- `摄影:xxx`
- `系列:xxx`
- `来源:xxx`
- `泳装`
- `杂志扫图`

### 9.5 写真集浏览器

写真集浏览器第一版沿用漫画阅读器机制，但改为写真语义：

- 主操作叫 `浏览`，不叫 `阅读`。
- 进度叫 `浏览进度`。
- 默认左右翻页。
- 默认左到右。
- 单页大图展示。
- 图片尽可能占满可视区域。
- 工具栏可隐藏。
- 点击中心或空白区域呼出工具栏。
- 保存浏览进度。
- 支持图片适配配置。

第一版不突出拼页能力。写真浏览器可以先不显示“与上一页拼接 / 与下一页拼接”按钮，避免漫画语义泄漏到写真场景。

## 10. 设置页

新增独立 `写真库` 设置区，参考漫画库设置：

- 启用写真库。
- 默认写真导入路径。
- 写真存储路径列表。
- 添加写真路径。
- 移除写真路径。
- 手动扫描写真。
- 自动扫描新增写真开关。
- 写真浏览器默认设置。
- 写真缓存设置。

路径添加方式参考影片存储路径和漫画存储路径，不进入影片路径或漫画路径。

## 11. Watcher 与扫描

写真库 watcher 独立于漫画库 watcher：

- 只监听写真存储路径。
- 只识别写真库支持的 `.zip` / `.cbz`。
- 新增或变更压缩包时触发写真扫描。
- 不扫描影片路径。
- 不扫描漫画路径。
- 不写入漫画表。

任务类型建议：

- `scan.photos`
- `import.photos`
- `photo.cache.cleanup`

## 12. 错误处理

应覆盖以下情况：

- 未启用写真库时访问写真路由，跳转到设置页写真库区。
- 未配置写真路径时阻止启用或提示添加路径。
- 导入非 `.zip` / `.cbz` 文件时提示格式不支持。
- 压缩包内没有有效图片时提示无法导入。
- 路径离线时阻止扫描与导入。
- 原图读取失败时在浏览器中展示错误状态。
- 缩略图生成失败时允许回退占位图，不阻断整个列表。

## 13. 测试策略

后端测试：

- 写真设置持久化。
- 写真路径 CRUD。
- 写真扫描器识别 `.zip` / `.cbz`。
- 写真扫描器忽略无图片压缩包。
- 写真页面自然排序。
- 写真页面图片读取。
- 写真缓存生成与清理。
- 写真 API contract。

前端测试：

- 写真库启用后侧边栏显示 `写真`。
- 未启用时隐藏 `写真` 入口。
- 写真库列表页搜索使用 shell 顶栏。
- 写真卡片网格对齐影片 / 漫画卡布局。
- 写真详情页展示标题、标签、评分、预览图。
- 写真浏览器保存浏览进度。
- 写真批量管理。
- 设置页写真路径添加、移除、扫描按钮。

浏览器验证：

- 写真库页面不是空白页。
- 无框架错误 overlay。
- 控制台无相关 error / warn。
- 写真卡片布局与影片 / 漫画网格一致。
- 浏览器图片高度 / 宽度自适应。
- 工具栏隐藏 / 呼出符合沉浸式体验。

## 14. 实施建议

不建议直接整套复制漫画库并全局替换名称作为最终架构。推荐分阶段实施：

1. 稳定漫画库 MVP 当前能力。
2. 抽出或中性化底层压缩包图片序列能力。
3. 新增写真库独立设置、路由、服务和表族。
4. 参考漫画库实现写真库 UI。
5. 在写真和漫画都稳定后，再考虑把重复 UI 抽成中性 `ArchiveImage*` 组件。

第一版可以允许局部复制漫画库前端结构以加快落地，但必须保持写真域命名、API、服务、设置、数据表和 watcher 独立。

## 15. 明确不做

MVP 不做：

- 文件夹图集导入。
- 散图导入。
- 模特 / 摄影师 / 发行方 / 系列 / 日期等结构化字段。
- OCR。
- 外部元数据抓取。
- 幻灯片模式。
- 写真专属缩略图胶片栏。
- 与漫画库合并展示。
- 与影片库合并展示。

## 16. 已确认决策

- 写真库作为侧边栏并列独立入口。
- 第一版只支持 `.zip` / `.cbz`，但模型预留未来文件夹图集。
- 入口叫 `写真`。
- 单个条目叫 `写真集`。
- 浏览页第一版沿用漫画阅读器机制。
- 详情页字段第一版与漫画详情页保持一致，只做标题、标签、评分等基础信息。

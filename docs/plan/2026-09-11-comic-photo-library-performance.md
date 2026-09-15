# 漫画库 / 写真库性能优化方案

日期：2026-09-11

状态：verified（A–D 已落地；浏览器解锁回归仍待用户 PIN 后手动确认）

关联：评估结论见本文「现状」；详情预览批次见 `docs/plan/2026-09-11-book-library-ui-preview.md`；影片墙虚拟化见 `docs/plan/2026-04-12-library-scroll-production-performance.md`。漫画 MVP 曾要求 `vue-virtual-scroller`，代码未兑现。

## 目标与非目标

**目标**

- 漫画墙、写真墙在几百本规模下，滚动和内存接近影片墙，而不是全量 DOM。
- 写真列表不再静默截断为默认 100 本。
- 在影片、漫画、写真之间来回切换时，不因进页必 reload 打出封面请求风暴。
- 封面缩略图冷启动少做重复 ZIP 枚举和逐条 tags 查询。
- **不改影片页布局与影片墙实现**。漫画/写真复用已有辅助函数和同一套网格 CSS 变量，不把 `VirtualMovieMasonry` 抽成共享组件作为前置条件。

**非目标（本方案不做）**

- 服务端排序 / 筛选 / cursor 分页（影片自己也还是前端全量缓存）。
- 阅读器改成全书长卷虚拟列表。
- 详情页页元数据分页 API（预览 UI 已有界）。
- 写真持久磁盘 cache 落地（`photoCache.maxBytes` 仍为预留设置）。
- 扫描 `WalkDir` 分片调度。
- Agent 触及漫画/写真。

## 现状（已核实）

| 层 | 影片 | 漫画 | 写真 |
| --- | --- | --- | --- |
| 列表 DOM | `DynamicScroller` 分块窗口 | 全量 `v-for` | 全量 `v-for` |
| 列表拉取 | 500 循环到 `total` | 同左 | **只拉一页，默认 100** |
| 进页 | `ensureLoaded` | 每次 mount 全量 reload | 同漫画 |
| 封面 | `MediaStill` + 焦点块 loading | `<img loading="lazy">` | 同漫画 |
| 缩略图 | 本地 asset | 磁盘 JPEG，无写入淘汰 | 32 MiB 内存 LRU + 源闸门 |
| 预览/阅读 | 播放器会话治理 | 预览批次有界；阅读器 1–2 页，无预取 | 预览批次有界；查看器 1 页，无预取 |

详情预览、封面走 thumbnail、阅读器可见页 Dom 已经够克制。瓶颈在列表层。

## 设计决策

1. **先正确性，再窗口化。** 写真分页循环是静默丢数据，必须先于虚拟化。
2. **虚拟化抄影片墙的块模型，不改影片文件。** 复用 `src/lib/library-virtual-scroll.ts` 的行高估算与海报 loading 策略；`ROWS_PER_CHUNK=4`、`BUFFER_CHUNKS=8`、`BUFFER_PX=1400` 与当前影片墙对齐。漫画/写真封面比例已是 `358/537`，可直接用现有估算。
3. **进页短路只跳过「已有完整列表」。** 扫描、导入、目录监听终态继续强制 `reload*FromApi`（`use-scan-task-tracker` / `use-library-watch-toasts` 已有调用点）。
4. **后端列表 tags 一次查回。** 不改 HTTP 契约，只改 repository。
5. **`OpenPage` 按条目打开 ZIP，不再先 `ListPages`。** `ListPages` 仍给扫描用。条目路径继续 `path.Clean` 后精确匹配。
6. **漫画 cache 淘汰用已有 `last_accessed_at`。** 无需新 migration；写入后若总字节超过 `maxBytes`，按最久未访问删文件+行，直到回到上限。`maxBytes < 0` 保持不限制。解码加上限：源字节 32 MiB、像素 24M，与写真 thumb 同档。
7. **阅读器预取只做当前页 ±1 的原图。** 使用隐藏 `Image()` / 预解码，离开路由取消；不把全书页挂进 DOM。

```
进库页
  ├─ settings 已在内存？→ 跳过 refreshSettings（或仅首次）
  ├─ comics/photosLoaded？→ 跳过列表
  └─ 扫描/导入/监听终态 → reload 全量

列表墙
  └─ DynamicScroller 块（4 行/块）
        焦点块 eager+high
        相邻块 eager+auto
        其余 lazy+low

封面请求
  └─ /pages/0/thumbnail
        命中 cache → ServeFile / LRU
        未命中 → OpenPage(entry) → 解码 → 写入 → 必要时淘汰
```

## 分阶段落地

### A. 列表完整性与进页短路（1 个切片）

**做什么**

- `web-photo-library-service.ts`：抽出与漫画相同的 `fetchPagedPhotos`，`LIST_BATCH_SIZE = 500`，未显式传 `limit` 时循环到 `offset >= total`。调用方传了 `limit` 则只拉一页（保持测试里 `{ limit: 50 }` 行为）。
- Mock 写真服务保持内存全量，不引入假截断。
- `ComicLibraryService` / `PhotoLibraryService` 增加 `ensureComicsLoaded` / `ensurePhotosLoaded`（或内部私有、view 改为调用现有 reload 前判断 `*Loaded`）。推荐显式方法，避免 view 里散落 if。
- `ComicsView` / `PhotosView`：`onMounted` 走 ensure；`refreshSettings` 仅在尚未拿到该库 settings 时调用，或 settings 与列表分两次、列表可短路。
- 扫描/导入/监听路径继续直接 `reload*FromApi`。

**主要文件**

- `src/services/adapters/web/web-photo-library-service.ts` 及 test
- `src/services/adapters/web/web-comic-library-service.ts` 及 test
- `src/services/contracts/{comic,photo}-library-service.ts`
- Mock 适配器补空实现
- `src/views/ComicsView.vue` / `PhotosView.vue` 及现有 view test

**验收**

- 写真 `total=600` 时前端缓存 600 条，至少 2 次 `listPhotos`（500+余数）。`total=250` 在 500 一批下只需 1 次请求，仍缓存全部 250 条。
- 已 loaded 时二次进入 `/photos` 或 `/comics` 不再打列表。
- 扫描 `scan.photos` / `scan.comics` 完成仍会 reload。
- 不改 `GET /api/library/photos` 契约。

### B. 列表真虚拟化（1 个切片）

**做什么**

- `VirtualComicGrid.vue` / `VirtualPhotoGrid.vue` 改为与 `VirtualMovieMasonry` 同构的 `DynamicScroller`：按列数×4 行分块、测量真实列数、块内继续用 `buildMovieGridChunkStyle`。
- 保留当前页面壳层：`min-h-0 flex-1` + 内部 `h-full overflow-y-auto pr-2`（与刚对齐的影片槽一致）。
- `ComicCard` / `PhotoCard` 增加可选 `posterLoading` / `posterFetchPriority`，由块索引调用 `resolveVirtualMoviePosterLoadPolicy`。默认仍 lazy，避免其它调用方回归。
- 不引入 `MediaStill`（那是影片海报版本号/骨架专用）；书封面继续 `<img>`，补上 `fetchpriority`。
- 滚动位置：漫画/写真暂不接 `use-library-scroll-preserve`，除非本切片顺手、且不扩大范围。默认进页回到顶部，与当前行为一致。

**主要文件**

- `src/components/jav-library/comics/VirtualComicGrid.vue` + test
- `src/components/jav-library/photos/VirtualPhotoGrid.vue` + test
- `src/components/jav-library/comics/ComicCard.vue` / `photos/PhotoCard.vue`
- 如需抽块结构，新增 `src/lib/book-virtual-scroll.ts` 只放漫画/写真分块，避免改影片文件

**验收**

- 300 本合成列表：活跃 DOM 卡片数远小于 300（与影片同数量级窗口，而不是 O(N)）。
- 网格 CSS 变量、卡片 max-width、滚动 `pr-2` 与影片墙一致。
- 批量选择（漫画）在虚拟化后仍能勾选、全选可见项。
- 窄屏 375px 无横向溢出；工具栏仍右对齐 `sm:min-h-8`。
- 不改 `VirtualMovieMasonry.vue`。

### C. 后端列表与 ZIP 打开（1 个切片）

**做什么**

- `ListComicBooks` / `ListPhotoBooks`：用 `WHERE comic_id IN (...)` / `photo_id IN (...)` 一次取出本页 tags，再按 id 填回。空页不查。
- `comicarchive.OpenPage` / `photoarchive.OpenPage`：`zip.OpenReader` 后按 cleaned 条目名打开，找不到再返回 `ErrPageNotFound`。不再为打开单页调用 `ListPages`。
- `ListPages` 行为与排序保持不变，扫描继续走它。
- 可选（同切片若测试成本低）：按 `path+size+mtime` 缓存 `ListPages` 结果给扫描热路径；不是封面墙的前置。

**主要文件**

- `backend/internal/storage/comic_books_repository.go`
- `backend/internal/storage/photo_books_repository.go`
- `backend/internal/storage/comic_tags_repository.go` / `photo_tags_repository.go`（批量方法）
- `backend/internal/comicarchive/service.go` + test
- `backend/internal/photoarchive/service.go` + test

**验收**

- 列表 50 条不再产生 50 次 tags 查询（repository test 用计数或 SQL trace / 一次 IN 查询断言）。
- `OpenPage` 对合法条目成功、对缺失条目仍 `ErrPageNotFound`；损坏/不支持归档错误不变。
- 封面 thumbnail 与阅读器原图仍能打开正确页。
- 无新 HTTP 路径，无 migration。

### D. 漫画 cache 治理与翻页预取（可后置）

**漫画 cache**

- `GetOrCreateThumbnail` 写入成功后：若 `maxBytes >= 0` 且当前合计超过上限，按 `last_accessed_at ASC` 删最旧条目（文件 + SQLite 行），直到 `sum(size_bytes) <= maxBytes`。正在返回的那条不删。
- 解码前：`LimitReader` 32 MiB+1；像素超过 24M 拒绝，与 `photothumb` 对齐。过大返回现有/新增稳定错误，HTTP 映射为 422，不回退原图。
- Settings 里「清理缓存」仍是全清，语义不变。
- Status 已有 `MaxBytes`；可附带实际占用（若 DTO 已有则用，没有就不为展示新加字段）。

**阅读器 / 查看器**

- `ComicReader.vue` / `PhotoViewer.vue`：对 `current±1` 的 `imageUrl` 预加载；翻页后更新窗口；`onUnmounted` 丢弃。
- 不预取缩略图以外的全书；失败静默，不影响当前页。
- 漫画 stitch 两页时，预取窗口以可见页为中心。

**验收**

- cache 超过 `maxBytes` 后旧文件消失、最新封面仍命中。
- 超大源图不把进程打满（单测用小上限或 mock reader）。
- 阅读器前进一页时，网络/解码日志显示相邻页已在请求（或 Image complete），当前页行为不变。

## 明确不做的交互

- 不为虚拟化改影片工具栏、Saved Views、批量条。
- 不把漫画/写真搜素搬回页面内（继续走壳层搜索）。
- 不把 Agent、Insights、推荐接到这两个库。

## 风险

| 风险 | 缓解 |
| --- | --- |
| 虚拟化块高度估低导致快速滚动空白 | 复用影片已放大的 buffer；块间距放进被测量的 grid 内（影片已踩过的坑） |
| 写真分页循环改变「只测一页」的旧测试 | 显式 `limit` 保持单页；无 limit 才循环 |
| `OpenPage` 不再校验条目是否在自然排序列表中 | 仍要求路径精确匹配且为支持的图片扩展；扫描写入的 `entry_path` 已是 ListPages 结果 |
| cache 淘汰删掉正在浏览的封面 | 跳过本次写入的 cacheKey；淘汰后下次请求再生成 |
| ensureLoaded 让扫描后的新书不出现 | 终态任务路径必须走 reload，并补回归测试 |

## 验证集（按切片，不默认跑全库）

- A：`web-photo-library-service.test.ts`、`web-comic-library-service.test.ts`、`ComicsView.test.ts`、`PhotosView.test.ts`、`use-scan-task-tracker.test.ts`
- B：两套 `Virtual*Grid.test.ts`、卡片 test、`ComicLibraryPage.test.ts` / `PhotoLibraryPage.test.ts`；`pnpm typecheck`；相关 ESLint
- C：对应 storage / archive 包 `go test` + `go vet`
- D：`comiccache` 包测试、阅读器组件 test
- 浏览器：解锁后影片 ↔ 漫画 ↔ 写真切换、300+ 本墙滚动、写真超过 100 本可见、导入/扫描后列表更新
- **不跑** `pnpm test:display`，除非用户明确要求

## 建议提交切分

1. `fix(photos): page through the full photo library list`
2. `perf(comics,photos): skip reloading a warm library on mount`
3. `perf(comics,photos): virtualize book library grids`
4. `perf(library): batch book list tags and open zip entries directly`
5. `perf(comics): evict comic thumbnail cache by maxBytes`
6. `perf(reader): prefetch adjacent original pages`

文档与规则随行为变更已改 `project-facts.mdc` / `docs/guide.md`。本文件状态为 `verified`（A–D 已实现并有聚焦测试）。

### 切片勾选

- [x] A 列表完整性与进页短路
- [x] B 列表真虚拟化
- [x] C 后端列表 tags 批量 + OpenPage 直接开 ZIP
- [x] D 漫画 cache maxBytes 淘汰 + 阅读器相邻页预取

## 未决（不阻塞 A–C）

- 漫画/写真墙要不要接影片同款滚动位置恢复。默认不做。
- 写真是否在 D 之后才做磁盘 cache。默认不做。
- `OpenPage` 是否要额外的 ListPages 内存缓存。默认不做，直接按条目打开足够。

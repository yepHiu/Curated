# 同一番号多文件与 FC2 独立展示

日期：2026-10-04
状态：verified（源码及本地验证完成；未发布生产包）
关联需求：REQ-0056、REQ-0057

## 用户确认与范围

用户确认分片指「同一番号的多个文件」。一条作品记录承载共同资料，下挂多个视频文件；普通影片与 FC2 都适用。不同番号仍为不同作品，不新增系列关系。

保留「影片」入口，新增「FC2」入口。分类按番号识别，兼容 FC2、FC2-PPV 和连接符变体；用户标签不改变分类。两类共用 Server、SQLite、存储池、标签词表、演员、收藏评分、刮削与备份，不要求搬迁物理文件。

合规化继续由独立项目推进，本仓库保持暂缓，见 [此前决定](2026-07-18-curated-compliance-and-metadata-source-decoupling.md#当前决定本仓库暂缓2026-10-04)。

## 已实现的浏览行为

| 入口 | 展示范围 |
| --- | --- |
| 影片、普通收藏及最近添加、首页栏目 | 非 FC2 作品 |
| FC2 | FC2 作品；可用收藏按钮限定本类收藏 |
| 观看历史 | 普通影片与 FC2 统一按观看时间展示，不提供类别切换 |
| 回收站、存储及标签管理 | 共用管理资源；回收站包含两类作品 |

列表、搜索、筛选、分页计数、相关作品及播放列表保持分类；从 FC2 进入详情、演员作品列表和播放器后返回仍保留 FC2 来源。保存筛选支持 `mode=fc2` 和 `favoriteOnly`。无 mode 的列表 API 保留全库能力供共享缓存和管理资源使用，显式浏览 mode 在计数与分页前应用分类。

2026-10-04 观看历史范围修订（已验证）：用户要求历史不区分普通影片与 FC2。沿用 HistoryView 的时间倒序、日期分组、PlaybackHistoryCard、续播和单条/批量删除，移除类别选择器与分类过滤；旧 `category=fc2` 历史链接也显示全部记录。点击历史记录继续恢复该作品最后播放的实际分部和时间，播放器的作品队列仍按该作品类别构造，返回目标统一为观看历史。空状态继续使用 MediaEmptyState 与资料库入口；无新设计令牌、API 或数据库变更。HistoryView、navigation-intent、player-route 和 locales 四文件 49 项测试、修改文件 ESLint、全量 typecheck 通过。Chromium 合成资料确认旧类别链接与播放器返回后的历史页均显示两类记录，无类别选择器；FC2 续播带 `fileId=part-2&t=240`，返回链接为 `/history`。浏览器验证仅覆盖显示和导航，未使用真实媒体；未运行完整显示缩放套件，本次未修改卡片、网格、壳层或设计令牌。

## 文件识别与存储

迁移 `0064_movie_files.sql` 建立 `movie_files` 和 `movie_file_progress`。文件有稳定 ID、作品 ID、路径和分片序号。列表增加 `fileCount`，详情增加有序 `files`；旧 `movies.location` 保留为兼容投影，不再替代文件全集。

扫描同番号的新路径追加文件，重复路径幂等；FC2 PPV 别名与连接符统一身份，保留已有作品 ID。取消同目录同番号只保留一文件的扫描限制。

支持 CD、PART、PT、DISC、DISK 数字标识，以及番号后的 `_1`、`-2` 等数字后缀；按数字排序，例如 1、2、10。普通番号后紧接的一位 A-Z 字母也作为分部序号，A=1、B=2，大小写不敏感；字母后可带连接符分隔的版本或质量标记。带连接符的 `-C` / `-UC` 不作为分部。没有序号的文件保留原文件名，先按文件名排序，详情可通过原名区分。所有同番号文件均保留；没有额外的版本判定、人工改序或文件缺失状态 UI。

自动整理对有序号文件使用 `{番号}/{番号}-CDn.ext`。目标名冲突时不覆盖文件，而是登记源文件原路径；其它整理失败沿用跳过与任务记录。已登记文件改名同步路径并保留 ID 和进度。导入查重区分同番号、新增分片、分片已登记；分片状态基于序号，最终磁盘写入仍执行文件冲突保护。

### 用户目录样本兼容补充 — 2026-10-04

用户提供 31 个 MP4 文件和一个说明 TXT，作为实际命名样本。修正此前 `STAR-380A/B`、`STAR-684A-C/B-C` 无法提取番号的缺口，并同步 Go 扫描和 TypeScript Mock 导入预览的解析规则。

| 样本 | 共同番号 | 分部 |
| --- | --- | --- |
| `FC2-PPV-4942041-1.mp4` / `-2.mp4` | `FC2-4942041` | 1、2 |
| `FC2-PPV-4944497-1.mp4` / `-2.mp4` | `FC2-4944497` | 1、2 |
| `STAR-380A.mp4` / `STAR-380B.mp4` | `STAR-380` | 1、2 |
| `STAR-684A-C.mp4` / `STAR-684B-C.mp4` | `STAR-684` | 1、2 |
| `SSIS-562-C.mp4` / `SSIS-588-C.mp4` | 各自的 `SSIS-562` / `SSIS-588` | 未编号；`-C` 为版本标记 |
| `FC2-PPV-551638 description.txt` | 不进入影片扫描 | 非视频 |

不要求手工改名。通过「添加媒体 → 添加影片」追加新分部，或把文件放入已配置存储目录后选择「⋮ → 重新扫描」。原影片无需删除或重建；分部状态按统一序号判断，例如 A 与 CD1 都是第 1 部。开启自动整理时，A/B 也整理为 CD1/CD2。

完整目录样本通过临时 SQLite 和磁盘集成验证：31 个视频形成 27 部影片（15 部普通影片、12 部 FC2），其中 4 部各有两个分片。分别验证关闭/开启整理时的番号、文件数量、分部顺序、整理后的文件内容和重扫稳定文件 ID。样本只用文件名构造小型测试文件，未读取或移动用户的真实视频。回归入口：`TestIntegration_RunScan_FilenameSamples`、`TestCheckLetterParts`、`TestExtractNumber`、`TestExtractPartIndex`、`src/lib/movie-number.test.ts`。

本次补充验证：`go test ./...`、`go vet ./...` 通过；前端番号解析、导入弹窗及 Mock service 三文件共 51 项通过，`pnpm lint`、`pnpm typecheck` 通过。

## 播放、续播与萃取

作品只显示一张海报，多文件作品显示文件数量。详情与播放器提供共用文件选择器，展示序号和文件名。切换分片保留浏览来源，清除上一片时间参数，恢复当前分片自己的进度；宿主释放上一片播放实例和 HLS 会话，并拒绝旧实例迟到状态事件。

2026-10-04 卡片位置调整：按用户要求把「2 个文件」等数量 Badge 从底部元数据区移到海报左上角番号正下方，左对齐、4px 间距。复用 MovieCard 的 outline Badge 与弱背景，适用于普通影片及 FC2；单文件不显示数量。现有 MovieCard 9 项测试、组件 ESLint 和全量 typecheck 通过；Playwright 合成卡片在 1280px / 375px 下验证同左缘、位于番号下方且在海报内、无横向溢出，并验证普通影片及单文件隐藏状态。

2026-10-04 分部选择条调整：按用户要求将共用 MoviePartSelect 的 SelectTrigger 改为 `rounded-full px-4` 胶囊形，详情页和播放器统一；保留既有下拉、文件名省略、禁用、分部切换和全屏 portal。组件 ESLint、全量 typecheck 通过；Playwright 合成资料在详情/播放器 1280px、375px 下检查真实圆角及尺寸，手机 44px、桌面 36px、无横向溢出。详情选第 2 部后播放路由带 `fileId=part-2`，播放器键盘选第 10 部后路由带 `fileId=part-10`；此次仅验证 UI 与选择导航，不使用真实媒体。

2026-10-04 详情默认分部调整：进入详情时优先选择 `partIndex=1` 的文件，缺少第 1 部时选择现有列表第一项；异步加载分片后同样初始化。当前作品资料刷新时保留有效的手动选择，切换作品或选中的分片移除后重新应用默认值。播放按钮显式传递选中 fileId。此选择仅属于详情面板，播放器与未指定文件的历史/续播入口仍沿用既有恢复逻辑，无 API 或数据库变更。DetailPanel 22 项测试（新增 4 项行为回归）、组件 ESLint 和全量 typecheck 通过；Chromium 合成资料验证进入详情显示第 1 部、直接播放路由带 `fileId=part-1`，重新进入详情仍默认第 1 部，手动选第 2 部后播放路由带 `fileId=part-2`。浏览器仅验证分部显示与导航，未使用真实媒体。

播放 descriptor、直连、HLS 建立与重定位、模式切换、故障回退、原生/协议播放器、高清帧和片段请求传递选定文件。服务端校验作品归属、允许根目录和实际文件。未指定文件时选择上次播放文件，否则首个文件。

每片分别保存进度，作品级历史仅保存最后播放片的一行。启用自动连播时优先进入同番号下一片，最后一片才继续作品队列；后台小窗结束不自动导航。截图及萃取帧保存 fileId，时间轴标记与近重复判断按来源文件隔离；绑定 GIF/MP4/WebM 到静态帧前也检查相同来源。图片库导出的 JPG/PNG/WebP 元数据也保留 fileId。

## 旧库与管理兼容

升级为旧库当前 location 建立首个文件记录，保持作品 ID、用户标签、评分、收藏、备注。旧进度和萃取帧绑定该实际原文件，原路径可能是第 2、3、4 片，不猜成第 1 片。

旧扫描已经覆盖、未保留的路径不能靠迁移恢复；文件仍在磁盘时，升级后重新扫描即可追加。迁移随新 Server 启动自动运行，当前开发没有操作用户运行中的数据库，也没有生产发布。

永久删除收集所有子文件；路径迁移更新子表与主路径；删除一个存储根只移除该根中不再受其它根覆盖的文件，其它根还有分片时保留作品；健康检查逐文件验证；资料写入根枚举所有文件。新表进入既有 SQLite 备份/恢复体系。无 library-config 新字段。

## 验证证据与边界

- 前端全量：Mock 模式、`NODE_OPTIONS=--no-experimental-webstorage` 下 `pnpm test`，308 文件 / 1653 项通过。最后小修复补跑播放器标记、加载、Web adapter、保存筛选，4 文件 / 91 项通过。
- 运行时 `pnpm test:e2e`，6 项通过；最终导航类别及选择器小修复补跑 4 文件 / 81 项通过。
- 前端 `pnpm lint`、`pnpm typecheck` 与 Web API 模式 `pnpm build` 通过；构建体积监管正常完成。
- 后端 `go test ./...` 通过；导出来源 metadata 追加后 server / curatedexport 回归通过；追加来源校验及整理处理后补跑 server/storage/app，并执行 `go vet ./...` 通过。
- 后端测试覆盖同番号追加、FC2 别名、1/2/10 顺序、重扫、独立进度、原文件迁移、根删除、文件改名、真实 HTTP Range 读取第二文件、外部文件拒绝、帧归属及跨片 clip 绑定拒绝。
- Chromium 使用独立 4194 端口和合成 API 资料检查普通/FC2 分离、详情选第 2 片播放、播放器切第 10 片的 fileId 路由；375px 视口 scrollWidth=clientWidth=375。媒体接口未提供真实视频，此验证不代表真实媒体 HLS/外部播放器的端到端验收。

实现入口：`src/lib/movie-category.ts`、`src/components/jav-library/MoviePartSelect.vue`、`backend/internal/storage/movie_files.go`、`backend/internal/library/moviecode/parts.go`。关键回归：`backend/internal/storage/movie_files_test.go`、`backend/internal/server/movie_files_test.go`、`backend/internal/app/runscan_integration_test.go`、`src/components/jav-library/ActivePlaybackHost.test.ts`、`src/lib/playback-progress-storage.test.ts`、`src/lib/library-query.test.ts`。

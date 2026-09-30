# Curated Personalization 实施计划

日期：2026-07-20  
状态：in-progress  
关联需求：REQ-0019～REQ-0022  
上位计划：[2026-07-19-project-feature-quality-audit.md](2026-07-19-project-feature-quality-audit.md)

## 执行快照

| 切片 | 状态 | 当前结论 / 下一步 |
|---|---|---|
| D1 Saved Views | implemented / QA pending | 版本化筛选、SQLite / Mock 持久化、五个 API、资料库管理 UI、三语文案及自动测试已实现；全量代码门禁通过，真实 Chromium QA 因当前受限环境 `spawn EPERM` 待补 |
| D2 推荐解释与负反馈 | implemented / QA pending | v8 快照理由、SQLite / Mock 显式反馈、真实排序影响、撤销/集中管理、三语 UI 已实现且非浏览器全量门禁通过；v8 复用 D3 的 canonical actor normalization；真实 Chromium 桌面/375px QA 因 `spawn EPERM` 待补 |
| D3 演员 alias/canonical merge | implemented / QA pending | Unicode canonical identity、alias-aware lookup、只读 preview、完整状态 token、确认式事务归并、链式归并安全审计、Web/Mock 与三语 UI 已实现；177/730 Vitest、4/28 Electron、Go test/vet、双构建、PRD 与 diff 全通过，真实 Chromium 桌面/375px QA 因 Browser 无实例和 `spawn EPERM` 待补 |
| D4 Insights | implemented / QA pending | Go/SQLite 有界 overview 与 actor/studio/tag breakdown、IANA timezone、null 分母、Web/Mock 契约、懒加载 `/insights`、侧栏入口、三语与 stale-response 保护已实现；180/745 Vitest、4/28 Electron、Go test/vet、双构建和 Bundle hard budget 通过；Browser 无实例且 Playwright `spawn EPERM`，真实 Chromium 桌面/375px QA 待补 |

Milestone D 只有在四项需求都达到 `verified`、全量门禁通过且事实文档同步后才结束。创建计划或完成 UI 不等于完成需求。

## 1. 目标与产品边界

Milestone D 的目标是让 Curated 从“能管理媒体”提升为“能适应用户习惯并解释结果”。实施顺序固定为：

1. Saved Views：先建立稳定、可复用的用户筛选表达。
2. 推荐解释与负反馈：让推荐理由可读，且用户反馈确实改变后续结果。
3. 演员 alias/canonical merge：消除同一演员因不同写法造成的画像碎片。
4. Insights：在筛选、推荐和实体身份稳定后提供可信的个人洞察。

本阶段不并行启动漫画库、Android、WebDAV、Linux / fnOS Server 或公共 metadata extension。这些方向继续作为 Milestone E 候选，完成 D 后只选择一个主分支。

隐私边界：所有个性化数据默认保留在用户本机。Web API 模式写入当前 SQLite；Mock 模式使用带 schema version 的 `localStorage`。本阶段不引入账号、云同步、遥测或第三方行为上传。

## 2. 当前实现基础与缺口

### 2.1 Saved Views

`src/lib/library-query.ts` 现已支持 `mode`、`q`、`tag`、`actor`、`studio`、`tab`、`playState`、`userRating`、`resolution` 与 `addedWithinDays`，并由同一 codec 构建版本化筛选和 canonical route。Web API 列表同时支持 `tag`、`playState`、精确用户评分、规范化分辨率与 `addedAfter`；`mode=recent` 的后端语义固定为最近 30 天，未知 mode 会返回 `400`。

Saved View 不能简单保存当前 URL。`selected`、`from`、`browse`、`back`、`autoplay`、`t` 等字段属于瞬态导航状态，必须排除；可持久化筛选必须经过白名单、规范化和版本化。

当前实现已遵守该边界：应用视图时从空 query 重建路由；资料库内部可以默认选中第一部影片，但不会再把该临时选择写回没有 `selected` 的 URL。

### 2.2 推荐

后端已有 `homepage_daily_recommendations` 与 `homepage_recommendation_states`，状态包含最近推荐时间、推荐次数和 `skip_until`；已有每日快照生成及刷新 API。现状缺少面向用户的推荐原因，也没有明确的“不感兴趣”“稍后再说”或实体级“少推荐”反馈。刷新推荐不能被隐式解释成负反馈。

### 2.3 演员实体

当前已在现有 `actors`、`movie_actors`、`actor_user_tags` 和 `actor_external_links` 上落地 canonical identity。migration `0035` 增加 `actors.normalized_name`、`actor_aliases` 和 `actor_merge_audits`，migration `0036` 解除 audit 对活跃 target actor 生命周期的外键钉死；名称比较统一使用 Unicode NFKC、case folding、trim 与连续空白折叠。profile、搜索、资料库精确筛选、头像、标签、外链、刮削与 metadata ingestion 均解析 alias，旧演员 route 会规范到 canonical actor。

### 2.4 Insights

当前已有 `playback_progress`、`library_played_movies`、`playback_daily_watch_time`、影片评分、演员、标签与厂牌关系。数据足以构建本地洞察，但必须先定义时间范围、观看时长、started/completed 和完成率的分母，且不能让前端拉取全部原始历史自行聚合。

## 3. 共通技术原则

- 先定义 DTO、错误码、filter schema 与 service contract，再实现 Web / Mock adapter 和 UI。
- 所有新增 Web API 继续受现有 PIN / Host / Origin 安全边界保护。
- SQLite migration 只前进、不重写历史 migration；新增唯一约束和索引必须有迁移测试。
- Web API 模式以 SQLite 为权威；Mock 模式保持刷新后可恢复，并做损坏数据回退与 schema version 迁移。
- 列表、反馈、归并和洞察都必须有稳定排序，避免同数据在刷新后跳动。
- 后端限制集合大小、字符串长度、时间范围和批量数量；前端限制只用于即时反馈，不能替代服务端校验。
- 新页面或重型工作台使用路由级或组件级懒加载，并继续满足现有 Bundle hard budget。
- 三语文案同步；主流程支持 375px、键盘操作、可见焦点、语义标题、44px 主要触控目标和无横向溢出。

## 4. D1：Saved Views（REQ-0019）

### 4.1 规范化筛选模型

定义版本化 `SavedViewFiltersV1`，只允许以下字段：

| 字段 | 语义 |
|---|---|
| `mode` | `library` / `favorites` / `recent` / `tags` / `trash` |
| `q` | 规范化后的自由搜索文本 |
| `tag` / `actor` / `studio` | 精确实体筛选，可交集组合 |
| `tab` | `all` / `new` / `top-rated` |
| `playState` | `all` / `unwatched` / `in-progress` / `completed` |
| `userRating` | 精确用户评分 0～5；不使用 scraper/site rating 冒充用户评分 |
| `resolution` | 规范化分辨率档位，例如 `4K`、`1080p` |
| `addedWithinDays` | 最近导入窗口；使用有限枚举或有界天数，不保存一次性的绝对当前时间 |

应用 Saved View 时由同一个 codec 生成 canonical route query。保存、读取、应用和 URL 直达必须共用 codec；未知字段丢弃或拒绝，不能透传为未来行为。

后端影片列表契约需要补齐 `playState`、`userRating`、`resolution`、`addedAfter` 等查询能力，并为常用组合检查索引与查询计划。Mock 使用同一筛选语义和测试向量，避免 Web / Mock 结果漂移。

### 4.2 持久化与 API

SQLite 新增有序 Saved Views 表，至少包含：稳定 ID、名称、规范化名称、filter schema version、filters JSON、排序值、创建时间和更新时间。

约束：

- 每个库最多 50 个 Saved Views；
- 名称 trim 后 1～40 个 Unicode 字符；
- 规范化名称唯一，拒绝只因大小写或首尾空白不同的重复名；
- filters JSON 必须由服务端解码、验证后再以 canonical 形式保存；
- 删除只删除视图定义，不删除影片、评分、标签或观看状态；
- 重排是单事务更新，拒绝缺失、重复或越权 ID。

建议 API：

- `GET /api/library/saved-views`
- `POST /api/library/saved-views`
- `PATCH /api/library/saved-views/{id}`
- `DELETE /api/library/saved-views/{id}`
- `PUT /api/library/saved-views/order`

Mock adapter 使用独立、版本化 localStorage key；不能复用或破坏 `jav-library-movie-prefs`。

### 4.3 UI 与验收

资料库页提供“保存当前视图”入口和 Saved Views 列表；支持应用、重命名、删除和排序。保存对话框必须展示将被保存的筛选摘要，并明确不保存当前选中影片或播放位置。

验收至少覆盖：

- 创建“未看”“五星”“某演员”“某标签”“最近导入”“4K”并得到正确结果；
- 刷新页面和重启后顺序、名称、筛选仍存在；
- Web / Mock 使用相同规范化测试向量；
- 应用视图后 URL 可分享/返回，且不携带 `selected` 等瞬态字段；
- 重名、空名、超长名、未知 schema、损坏 localStorage 和超限均安全失败；
- 删除视图不影响任何媒体或用户偏好数据；
- 375px、键盘、确认对话框和空状态通过真实浏览器 QA。

### 4.4 当前实施证据（2026-07-21）

实现落点：

- Go / SQLite：migration `0032_library_saved_views.sql`、`storage/saved_views.go`、`server/saved_views_handlers.go`，以及列表筛选契约和 `userRating` 列表字段；
- API：`GET/POST /api/library/saved-views`、`PATCH/DELETE /api/library/saved-views/{savedViewId}`、`PUT /api/library/saved-views/order`；
- Web / Mock：service contract、Web adapter 缓存、Mock adapter 与独立 `localStorage` 键 `curated-library-saved-views-v1`；
- 前端：`library-query.ts`、`saved-view-model.ts`、`library-saved-view-filter.ts`、`saved-views-local-storage.ts`、`LibrarySavedViewsControls.vue` 与 `LibraryView.vue` canonical route 修复；
- UI：高级筛选、保存、应用、更新、重命名、上下移动和确认式删除，中英日三语，移动端主工具栏按钮保持 44px 触控高度。

已通过：

- `pnpm typecheck`、`pnpm lint`；
- `pnpm test`：171 files / 707 tests；
- `pnpm test:electron`：4 files / 28 tests；
- `pnpm build`：initial JS 336598 raw / 121461 gzip，total JS 2248692 raw / 739325 gzip；
- `pnpm build:electron:main`；
- `go test ./...`、`go vet ./...`，以及 Saved Views storage / server 专项包测试。

尚未完成：

- `pnpm test:e2e` 与人工真实浏览器 QA 在当前受限 Windows 沙箱中均因 Chromium `spawn EPERM` 无法启动；因此 REQ-0019 保持 `implemented / 90`，不能标记为 `verified`；
- 浏览器权限恢复后需在 1280×900 与 375×812 逐项验证创建、应用、更新、重排、重命名、删除、刷新持久化、URL 无瞬态字段、偏好键不受影响、控制台无错误、无横向溢出和 44px 触控目标。

## 5. D2：推荐解释与负反馈（REQ-0020）

### 5.1 推荐解释

推荐响应从纯 movie ID 扩展为稳定条目，包含 `movieId`、有序 reason codes 与可展示实体引用。首版原因只使用当前本地可验证信号，例如：

- 与近期观看或高用户评分影片共享演员；
- 与近期观看或高用户评分影片共享厂牌或标签；
- 新入库且尚未观看；
- 很久未推荐，作为多样性补充。

前端只翻译 reason code，不自行反推算法。理由必须真实对应本次生成所使用的信号；无可靠理由时使用诚实的通用原因，不伪造“因为你喜欢”。

### 5.2 反馈模型

支持：

- 对影片“不感兴趣”；
- “稍后再说”，设置有界 `skip_until`；
- “少推荐这个演员 / 厂牌 / 标签”；
- 撤销最近反馈；
- 在集中管理页查看并移除活跃反馈。

反馈必须持久化为显式 action 和 target，而不是覆盖推荐快照。生成器在候选过滤和打分阶段读取反馈，并在响应中保留可测试的影响原因。`refresh` 只表示重新生成，绝不自动写负反馈。

建议 API：

- `POST /api/homepage/recommendations/feedback`
- `GET /api/homepage/recommendations/feedback`
- `DELETE /api/homepage/recommendations/feedback/{id}`

### 5.3 安全与验收

- 反馈目标必须属于当前库中的影片或该影片的演员、厂牌、标签；拒绝任意字符串污染画像。
- 同类型同目标重复反馈幂等；并发提交不会重复累计。
- “少推荐”降低权重但保留有界探索能力；“不感兴趣”按明确期限或永久规则排除，产品文案与算法一致。
- 提交反馈后，当前卡片即时给出可撤销状态；下一次生成结果有数据库与算法测试证明发生预期变化。
- 删除/撤销反馈后恢复候选资格；刷新推荐本身不产生反馈。
- 每个推荐条目可看到至少一个真实理由；中英日文案、空状态和 375px 交互通过 QA。

### 5.4 当前实施证据（2026-07-21）

实现落点：

- SQLite / Go：migrations `0033_homepage_recommendation_feedback.sql`、`0034_homepage_recommendation_explanations.sql`，`storage/homepage_recommendation_feedback.go`、`app/homepage_recommendation_feedback.go`、`server/homepage_recommendation_feedback_handlers.go`；D2 首次落地时 generation version 升为 `v7`，D3 统一 canonical actor normalization 后当前为 `v8`；快照同时保留 ID 列表和持久化解释条目；
- API：`GET/POST /api/homepage/recommendations/feedback` 与 `DELETE /api/homepage/recommendations/feedback/{feedbackId}`；响应条目包含真实 reason codes 和可选 `feedbackEffects`；
- Web / Mock：service contract、response guards、Web adapter 与 Mock adapter 已对齐；Mock 使用独立 `localStorage` 键 `curated-homepage-recommendation-feedback-v1`；
- UI：`HomeRecommendationCard.vue` 每卡最多显示两个理由，提供“不感兴趣”、7/30 天稍后推荐、演员/片商/标签降权和当前卡撤销；`HomepagePortal.vue` 提供集中管理 Dialog；中英日文案齐全；
- 算法：`not_interested` 和活跃 `snooze` 排除影片；`less` 每个命中乘以 `0.35` 且保留 `0.1` 探索下限；刷新本身不写反馈；
- Bundle：HLS fallback 改用官方 `hls.js/light`，没有放宽硬预算；最近构建为 initial JS `339739 / 122352`、total JS `2089550 / 691804`，相对 2250000 raw hard budget 保留约 160450 bytes。

已通过的自动化证据：

- `go test ./internal/app ./internal/server ./internal/storage`；
- 推荐 API validation、localStorage、Mock/Web adapters、homepage portal、HomeView、HomeRecommendationCard、i18n 与 HLS helper 共 9 files / 101 tests；
- `hls-player` 5/5 tests；
- `pnpm typecheck`、`pnpm lint`；
- `pnpm test`：173 files / 715 tests；
- `pnpm test:electron`：4 files / 28 tests；
- `pnpm build` 通过未放宽的 Bundle hard budget；
- `pnpm build:electron:main`；
- `go test ./...`、`go vet ./...`；
- PRD lint、PRD Python tests 6/6、`git diff --check`。

尚未完成：

- 应用内 Browser 当前返回无可用浏览器；仓库 `pnpm test:e2e` 启动 Chromium 报 `spawn EPERM`；
- 1280×900、375×812 真实 Chromium QA 仍需补做，所以 REQ-0020 保持 `implemented / 90`，不能标记为 `verified`。

## 6. D3：演员 alias / canonical merge（REQ-0021）

### 6.1 数据模型

新增 actor aliases 与 merge audit：

- alias 保存原始显示值和 normalized value，并唯一指向一个 canonical actor；
- normalized alias 全局唯一，且不能与另一个 canonical actor 的 normalized name 冲突；
- canonical actor 仍使用现有 `actors.id`，避免无必要的大范围外键重构；
- lookup 先解析 canonical name，再解析 alias，最终返回 canonical profile；
- 禁止 self-merge、alias 环和已经归并实体作为新的独立目标。

### 6.2 dry-run 与确认式 apply

归并分为两步：

1. dry-run 返回 source、target、影片关联数量、重复关联、标签/外链/profile 字段冲突、将创建的 aliases 和不可执行原因；不写数据库。
2. apply 必须携带显式 `confirm:true` 与 dry-run token / version，在一个事务中重新核对后执行，并写完整 audit。

建议 API：

- `POST /api/library/actors/merge-preview`
- `POST /api/library/actors/merge`
- `GET /api/library/actors/merge-audits`

### 6.3 字段保留规则

- `movie_actors` 合并到 canonical 后去重；任何影片最多保留一条 canonical 关联。
- `actor_user_tags` 做集合并集并去重。
- `actor_external_links` 保持稳定顺序去重，超过上限时 dry-run 阻止并要求用户处理。
- profile 字段默认“目标非空优先、目标为空时取源”；存在两个不同非空值时必须在 preview 显示，不能静默覆盖。
- source name 与既有 aliases 全部解析到 target；源 actor 行只有在所有引用安全迁移后才删除或标记为 merged。
- scraper 使用 alias 命中时更新 canonical actor，不重新创建同名碎片。

### 6.4 验收

- dry-run 完全只读；apply 事务失败后零部分写入；并发或陈旧 preview 被拒绝。
- 影片、用户标签、外链、头像/简介/provider 字段按规则保留且无重复。
- 原名称 URL、搜索、详情、推荐和 Insights 都解析到 canonical actor。
- self-merge、环、alias/name 冲突、超限外链和不存在实体有稳定错误码。
- migration、repository、handler、service adapter、Mock 和真实浏览器确认流程均有测试。

### 6.5 当前实施证据（2026-07-21）

已实现：

- SQLite / Go：`0035_actor_canonical_merges.sql` 增加 `normalized_name`、`actor_aliases`、`actor_merge_audits`；`0036_actor_merge_audits_detach_actor_lifecycle.sql` 保留不可变审计快照但允许 A→B→C 连续 canonical 归并；启动 migration 后回填 actor normalized names，并规范化/去重旧 D2 actor feedback；
- identity：`NormalizeActorIdentity()` 统一 NFKC、Unicode case folding、首尾空白与连续空白折叠；前端 `actor-identity.ts` 为 Mock、localStorage 与推荐反馈提供一致比较语义，覆盖全角字符、`ß`/`SS` 与希腊 sigma；推荐 generation version 因演员身份语义变化升为 `v8`；
- API：`POST /api/library/actors/merge-preview`、`POST /api/library/actors/merge`、`GET /api/library/actors/merge-audits`；稳定错误码覆盖 invalid、not found、self、source alias、conflict、stale preview 与 link limit；
- preview：完全只读；opaque SHA-256 token 覆盖 source/target 内部 profile、影片 ID、用户标签、外链、aliases、推荐反馈与会变化的萃取帧 actor JSON；外链超过 16 项和 alias/name 冲突会阻断；
- apply：必须 `confirm:true`，在同一 SQLite 事务中重新 preview 并 constant-time 比较 token；冲突 profile 字段必须逐项选择 source/target；影片关系、用户标签、外链、profile/头像状态、aliases、演员推荐反馈和萃取帧 actor JSON 迁移/去重后才删除 source 并写 audit，任何失败零部分写入；
- alias-aware 行为：profile、演员搜索、Library actor 精确筛选、头像、标签、外链、scraper profile update 和 metadata ingestion 全部返回/复用 canonical actor；旧 alias route 自动导航到 canonical actor；
- 前端：`ActorMergeDialog.vue` 在演员详情页提供 target 输入、只读预览、影响摘要、blocking reasons、profile 冲突选择与最终 destructive 确认；Dialog 有 title/description、语义色、`dvh` 限高、按钮 icon `data-icon` 和 375px `min-h-11` 主触控目标；未新增或覆盖 shadcn-vue 基元；
- Mock：`curated-actor-merges-v1` 持久化 display alias 与 audit，preview token、stale 检查、影片演员规范化、标签/外链并集、推荐反馈改写、旧 alias lookup 与刷新后 movie canonicalization 均有覆盖；
- 最新专项门禁：`pnpm typecheck`、`pnpm lint` 通过；D3 前端 9 files / 75 tests 通过；`go test ./internal/storage ./internal/app ./internal/server -count=1` 通过；连续 A→B→C 归并、事务回滚、完整 stale preview、alias collision、link limit、旧反馈 backfill 与 handler error mapping 均有直接测试；
- 完整非浏览器门禁：`pnpm typecheck`、`pnpm lint`、全量 Vitest `177 files / 730 tests`、Electron `4 files / 28 tests`、`pnpm build`、`pnpm build:electron:main`、`go test ./...`、`go vet ./...`、PRD lint、PRD Python tests `6/6` 与 `git diff --check` 全部通过；Go 命令仍打印工作区外 telemetry upload token 权限提示，但退出码为 0；
- 最新生产构建通过未放宽的 Bundle hard budget：initial JS `342915 raw / 123382 gzip`，total JS `2107543 / 697307`，余量分别为 `142457 raw / 52693 gzip`；`ActorDetailView` `12420 / 3942`、`LibraryView` `54233 / 13270`、`HomeView` `29374 / 8685`、`hls-player` `332962 / 103885`。

尚未完成：

- 应用内 Browser 初始化后报告无可用浏览器实例（`[]`），仓库 `pnpm test:e2e` 启动 Chromium 报 `spawn EPERM`；
- 1280×900、375×812 真实 Chromium QA 仍待补做，因此 REQ-0021 保持 `implemented / 90`，不能标记为 `verified`。

### 6.6 演员归并重新梳理（2026-10-01，讨论稿）

本节响应“归并演员功能需要重新捋一捋”。以下现状来自代码静态核对；建议尚未实施，也不代表用户已确定范围。本轮仅更新文档，未操作资料库、未重跑历史测试或浏览器 QA。§6.5 的测试结果保留为历史记录。

#### 产品目标与边界

把已经确认属于同一个人的多条演员记录整理成一个演员：保留一个主要显示名称，其他名称作为别名，作品及相关资料集中到同一详情页。

需要分别定义三个操作，避免混成一个入口：

- 合并重复演员：库中已有 A、B 两条记录，确认同人后合成一条。
- 管理显示名与别名：修改名称或补充曾用名，不必存在另一条演员记录。现有归并弹框不提供这项独立能力。
- 发现疑似重复：根据名字及资料提出候选；相似名字不能直接证明同人。REQ-0035 的 AI 建议仍属规划。

#### 当前流程与缺口

| 环节 | 代码现状 | 建议调整 |
|------|----------|----------|
| 入口 | 演员详情页打开弹框，当前演员固定为来源 | 明确“与另一位演员合并”，让用户看见并选择最终保留的演员 |
| 选人 | 手动输入目标名称，没有候选列表 | 搜索并选择现有演员，展示头像、名称、影片数及可用的身份依据；支持别名命中 |
| 方向 | A → B，保留 B 的 ID 和名称，删除 A 记录 | 用“合并后显示为 B；A 保留为别名”表达；允许交换方向并重新预览 |
| 预览 | 已返回双方数量、结果、别名和资料字段；界面多展示总数，仅冲突字段展示双方值 | 展示双方演员卡片、具体别名和最终资料；说明作品并集及重叠数量，避免把关系去重理解成删除影片文件 |
| 冲突 | 非空不同值逐字段选择；空值自动补齐 | 普通资料延续此规则；头像按可视图片选择，关联字段成组处理 |
| 文案 | 出现 canonical、alias、“事务归并”、本地头像路径和来源 ID | 面向用户使用“显示名称”“别名”“合并演员”；技术字段置于必要的详情中 |
| 完成 | 服务刷新影片后，页面跳到目标演员 | 展示结果并提供合并记录入口；区分合并成功与页面刷新失败 |
| 历史与恢复 | 有审计查询 API，无业务界面调用；当前没有撤销合并接口 | 先提供可查历史；是否支持撤销须明确恢复模型，不能把审计等同于备份 |

#### 必须先明确的数据规则

1. **身份与名称**：保留目标演员 ID，原名及已有别名指向该演员。大小写、全半角及空白规范化只解决文字比较；中文名、日文名、罗马字、改名及同名不同人仍需要身份判断。现有全局名称/别名解析模型不能自然表达同名不同人，需要单独评估需求，不能以归并掩盖。
2. **作品及关联**：保留作品记录，演员关联做并集去重；用户标签做并集，外链稳定去重。当前外链合并超过 16 项会阻断，应提供明确的处理指引。
3. **资料保留**：目标为空时补来源值；两个不同非空值展示双方并选择。自动补齐结果也应可见，而非只展示冲突。
4. **关联字段一致性**：当前 `provider` 与 `providerActorId` 独立决策，存在选择后不匹配的风险；建议作为来源身份组合选择，并由后端校验。头像远程 URL、本地缓存路径与抓取状态也应保持一致，由用户选图、系统维护配套字段。此项是代码路径识别的风险，尚未做运行时复现。
5. **旧名和后续导入**：合并后旧名搜索、详情链接、扫描/刮削应继续命中保留演员，覆盖 A→B→C 连续合并。当前归并调用链仅写 SQLite，不写回 NFO 或重命名媒体文件；后续若扩展文件写回，需要独立定义范围。
6. **反馈与萃取帧**：保留现有推荐反馈及萃取帧演员归属迁移，预览中作为次要影响信息展开，不能因界面精简而漏迁。
7. **执行一致性**：复用只读预览、过期校验、单事务迁移和审计。更换演员或方向必须清除旧预览及冲突选择；过期后重新预览。后端已提交而影片刷新失败时，应提示“已合并，刷新失败”，不能引导重复执行。当前 Web adapter 将两步放在同一个 await 调用链，需要专项验证和修正。
8. **误合并恢复**：当前 audit 保存数量、结果集合、决策及身份快照，缺少完整的合并前资料和逐条关系归属，无法据此可靠拆回。若撤销进入范围，应补合并前快照、后续修改检测及链式合并恢复规则；实现前明确说明暂不支持单次撤销。

#### 建议的单次操作流程

选择另一个演员 → 对比双方并选择保留谁 → 预览合并结果并处理资料冲突 → 确认合并 → 查看合并后的演员及记录。

例如 A 有 12 部、B 有 8 部，其中 3 部共同关联，预览应显示“合并后 17 部作品，3 部重复关联已去重；显示为 B，A 成为别名”。最终确认前展示实际名称及最终头像；无需用户理解 canonical、事务或缓存路径。

建议先收口单对演员合并：选人、方向、可读预览、关联字段一致性、过期和刷新失败处理。随后完善别名展示/管理、合并历史及恢复能力。疑似重复发现和批量处理复用成熟的单次流程；当前单对 API 不应直接包装成没有逐项结果与失败恢复的批量操作。

#### 后续实施验收与待定范围

- 选人支持搜索、空结果、别名命中及同一演员排除；能识别最终保留方向。
- 作品数量符合并集计算，旧名可达；标签、外链、反馈、萃取帧及链式归并无丢失。
- 头像及来源身份成组一致；空值补齐和冲突选择均能在最终预览看见。
- 外链超限、名称冲突、预览过期均有可操作提示；取消不写入，事务失败无部分写入。
- 合并成功但刷新失败不被显示为合并失败；关闭/重开期间的迟到预览响应不能覆盖新选择。
- 历史只声称可查询，不声称已支持撤销；若实现撤销，另验收后续修改及链式合并场景。
- 需要继续确定：当前首要诉求是操作流程、身份/资料规则与恢复，还是发现重复及批量整理；独立改名/别名管理、同名不同人和撤销是否进入本轮实施范围。

核对入口：`src/components/jav-library/ActorMergeDialog.vue`、`ActorDetailPage.vue`、`ActorProfileCard.vue`、`src/services/adapters/web/web-library-service.ts`、`src/api/types.ts`、`backend/internal/app/actor_merge.go`、`backend/internal/storage/actor_merge.go`、`actor_identity.go`、`backend/internal/server/actor_merge_handlers.go`，以及对应归并测试文件。

### 6.7 单次演员合并实施（2026-10-01）

用户已要求开始实施。首个切片包含搜索选人、双演员对比和保留方向、具体结果预览、头像与来源身份成组决策、异步请求隔离、提交后刷新失败处理；独立改名/别名编辑、合并历史界面、撤销及 AI/批量发现留在后续切片。

设计框架：演员详情的维护弹框，唯一主任务是确认一次同人合并；沿用 `ActorMergeDialog`、`ActorLibraryCard` 的头像和语义表面、现有 Dialog/Input/Button/Avatar。先选人，再看保留方向和最终资料，关联统计为次级信息。完整覆盖搜索加载/空结果/失败、未选择、预览加载/失败/过期/阻断、冲突未选、提交中和刷新失败。窄屏单列、主要按钮至少 44px、弹框限高内部滚动；保持项目既有字体/主题，不新增基元或全局令牌。执行项目 `curated-ui-governance` 与 `ui-ux-pro-max` 的表单可访问性检查；通用技能给出的营销型配色与 hero 建议不适用于本维护面，继续服从项目规范。

已实现：

- 搜索候选经既有 `listActors`，250ms 防抖，最多 20 项，排除当前演员；搜索/预览均隔离迟到响应，关闭重开和换方向清空旧选择。
- 双演员单选卡、具体别名和作品并集、用户标签、可展开的外链/反馈/萃取帧结果；空值补齐资料直接显示。中文、英文、日文同步。
- 头像 URL/缓存、来源名称/ID 按整组决定默认值与冲突，后端及 Mock 拒绝混组；头像状态随所选组迁移，无新端点或数据库迁移。旧客户端若提交拆组选择会收到既有冲突错误。
- Web adapter 区分已提交与后续刷新失败；客户端返回 `refreshFailed` 标记并提示刷新，无需重复合并。提交中阻止关闭/重复提交，apply 失败清掉可执行预览。
- 保留当前演员时也重载资料卡。浏览器检查发现资料卡在 Mock 下跳过名称解析，已改为仍读取 profile 并发出 resolvedName，只在 Web 下自动刮削；旧 alias URL 现可正确跳到保留演员。

验证（2026-10-01，macOS）：

- `pnpm typecheck` 及修改文件 ESLint 通过。
- 前端专项共 7 文件 / 82 项通过：ActorMergeDialog、ActorDetailPage、ActorProfileCard、Web/Mock adapter、identity 和 localStorage。Node 25 本机遵循构建手册使用 `NODE_OPTIONS=--no-experimental-webstorage`，避免原生 localStorage 与 jsdom 冲突。
- `backend/` 下 `go test ./internal/storage ./internal/app ./internal/server` 与相同包 `go vet` 通过；含混组拒绝零写入、所选头像抓取状态和空 provider ID 保留测试。
- `pnpm build` 通过，体积监管无提醒/超限。
- Playwright CLI 的独立 Mock 浏览器完成选人→预览→合并→跳转→刷新持久化；30+30、重复关联30、结果30正确，audit 恰好一条。回访 Mina Kaze 旧链接跳转到 Rin Asuka。取消第二次预览后 audit 数量保持1。
- 桌面 1280×720 与手机 375×812 的深色界面已查看截图；手机弹框宽343、高714.55，内部无横向溢出，确认按钮44px，Escape 可关闭。浏览器检查时无 warning/error。截图在 `.workspace/output/playwright/actor-merge-desktop.png`、`actor-merge-mobile.png`（本机产物，不提交）。
- 未运行完整 `pnpm test:display`；真实 Web API 浏览器闭环、Safari/Firefox、其它缩放和主题的完整矩阵仍待专门 QA。本次未动真实资料库，也未将历史 REQ-0021 自动提升为 verified。

后续范围仍为独立改名/别名编辑、合并历史界面、可恢复快照/撤销及 AI/批量候选。

## 7. D4：Personal Insights（REQ-0022）

### 7.1 指标契约

首版固定提供 `30d`、`90d`、`365d`、`all` 四个范围，并在响应中返回实际 `from`、`to` 与时区/日界线定义。所有卡片使用同一范围。

指标定义：

- `watchedSeconds`：范围内 `playback_daily_watch_time.watched_sec` 的去重日聚合总和；
- `startedMovies`：范围内存在有效观看信号的不同 movie 数；
- `completedMovies`：达到统一完成阈值的不同 movie 数；阈值必须在后端常量与 API 文档中明确；
- `completionRate`：`completedMovies / startedMovies`，started 为 0 时返回 `null` 而不是伪造 0%；
- `ratedMovies` 与 `averageUserRating`：只使用用户本地评分，分母为范围内有用户评分且有观看信号的不同影片；
- 演员、厂牌、标签排行：返回观看时长、影片数及占总观看时长比例；多演员/多标签影片必须采用文档化归因规则，避免百分比被误读为互斥分布。

### 7.2 API 与 UI

建议后端聚合 API：

- `GET /api/insights/overview?range=30d|90d|365d|all`
- `GET /api/insights/breakdown?range=...&dimension=actor|studio|tag&limit=...`

新增懒加载 `/insights` route，并从主导航或首页提供明确入口。页面包含范围切换、总观看时长、started/completed/完成率、用户评分摘要以及演员/厂牌/标签排行。图表不是必需条件；当表格、条形列表更准确易读时优先使用更简单的表达。

### 7.3 验收

- 使用固定时区、跨日、重复上报、零数据、多演员/多标签和缺失时长 fixtures 验证聚合。
- 响应为有界聚合，不返回完整播放历史；`limit` 有服务端上限和稳定 tie-break。
- 范围切换不会混用缓存；空状态解释数据从何时开始积累。
- 所有百分比都能在 UI 看到分母或口径说明；无数据时不出现 NaN、Infinity 或误导性 0%。
- 375px 无横向溢出；键盘、焦点、语义标题、颜色对比和中英日文案通过 QA。

### 7.4 当前实施证据（2026-07-21）

- 后端：`contracts.go` 定义四档范围、三个维度、overview/breakdown DTO 与四个稳定参数错误码；`storage/personal_insights.go` 以范围内正观看时长为有界 CTE，聚合 started/current-completed/current-rating，并为 canonical actor、片商和逐片去重标签返回稳定 top-N；
- 口径：请求携带 IANA timezone，Go 内嵌 `time/tzdata`；范围为包含首尾日的本地日历窗口。completed 是“范围内看过且当前保存进度达到 90%”，评分是这些 started 影片的当前本地用户评分，两者都不伪装成范围内发生事件；无 started/rated 分母时返回 `null`；
- 归因：breakdown 默认 10、最大 25，固定 `full-per-entity`；单项 share 以总观看时长为分母，多演员/多标签跨实体合计可超过 100%，API 与 UI 均明确说明；原始每日/逐片历史不通过服务契约返回页面；
- API：`GET /api/insights/overview` 与 `GET /api/insights/breakdown`；handler 覆盖参数透传、畸形 limit、四类稳定 400、provider 缺失及内部错误；前端 DTO guard 拒绝无分母却返回 0%、错误 attribution、非有限值与越界结构；
- Mock：`playback-watch-time-storage.ts` 只向 adapter 提供 detached、规范化的按影片/日快照；`personal-insights.ts` 在服务边界内实现与后端一致的窗口、当前状态分母、去重、stable sort、limit 与 full attribution，页面不接触原始行；
- UI：`/insights` 是独立动态 route，`PersonalInsightsPage.vue` 并行加载 overview 与三个 breakdown，切换范围以请求序号拒绝旧响应；一个语义 `h1`、原生 fieldset/radio、六项摘要、三类排行、loading/error/retry/empty、null 显示 `—`、375px 2×2 范围控件及中英日文案均有组件/路由/侧栏测试；
- Chromium e2e 已新增 `tests/e2e/personal-insights.spec.ts`：预置 Mock 观看时长/进度，检查唯一 `h1`、六项摘要、三类排行、范围切换、无 NaN/Infinity、零后端请求、桌面与 375×812 无横向溢出及 44px 范围控件；`playwright --list` 成功发现 2 个文件中的 5 个测试，但实际执行仍受 `spawn EPERM` 阻断，不能把“可发现”记为浏览器通过；
- 全量非浏览器门禁：`pnpm typecheck`、`pnpm lint`、180 files / 745 Vitest、4 files / 28 Electron、`pnpm build`、`pnpm build:electron:main`、`go test ./...`、`go vet ./...` 均通过；Go 仍打印工作区外 telemetry token 权限提示但命令退出码为 0；storage 与 Mock 额外以等时长/不同影片数/名称大小写 fixture 验证 movieCount、名称 tie-break 和 limit 的执行顺序；
- Bundle：首屏静态 JS `346882 raw / 124701 gzip`，全量 JS `2127607 / 704237`，均低于未放宽的 `500000/165000` 与 `2250000/750000` hard budget；`InsightsView` 为独立 dynamic entry `9767 / 3277`；
- `REQ-0022` 当前保持 `implemented / 90`。应用内 Browser 初始化后实例列表为空，仓库 `pnpm test:e2e` 启动 Chromium 报 `spawn EPERM`；真实 Chromium 1280×900、375×812、console、无横向溢出和交互 QA 完成前不能标记 `verified`。

### 7.5 个人洞察 UI 评估与优化方向（2026-09-24）

**设计框架**：这是 AppShell 内的个人数据阅读面。主要任务是快速理解本期观看情况与偏好，时间范围是主控件，AI 解读是次级动作。继续使用现有 Card、Progress、语义色和页面滚动结构；本轮建议均局限于 `PersonalInsightsPage`，不引入新的全局令牌或图表库。保留现有加载、空、错误、无分母、AI 不可用与窄屏状态。

**现场依据**：在运行中的 Web API 页面、约 1280px 浏览器宽度下，内容区六张摘要卡被 `xl:grid-cols-6` 排成一行，每张卡的解释句折成多行；三类排行各显示最多 10 项，长标签和大量低时长同值条目占据较长滚动区域。代码中范围控件始终横跨整行；切换范围时清空 overview 与三个 breakdown，并等待四个请求全部完成后才显示数据。当前数值口径及归因说明是准确性要求，优化时须保留。

| 优先级 | 调整 | 验收重点 |
| --- | --- | --- |
| P1 | 摘要区改为更稳定的 3×2 桌面网格；将「观看时长、开始观看、完成率」放在第一视觉层，完成数、评分数、平均分作为第二层。卡面只保留短标签、数值及必要的分母，详细定义收进统一的口径说明。 | 在约 1280px 窗口下数值与单位完整、卡片高度不被长说明拉大；375px 仍为 2 列且无横向溢出。 |
| P1 | 时间范围改成按内容收缩的紧凑控件，并把实际日期、数据起始日放在控件附近。AI 解读与其状态就近显示，但保持次级视觉权重。 | 桌面首屏能同时看到时间范围、核心指标和排行入口；键盘焦点、44px 手机触控目标保持可用。 |
| P1 | 三类排行默认展示前 5 项，提供各类「展开全部／收起」；继续用条形列表和明确百分比，不用饼图暗示互斥分布。将 `full-per-entity` 解释保留在排行标题附近并压缩文案。 | 长排行可按需查看，截断名称可读完整文本；跨实体合计可超过 100% 的事实清楚可见。 |
| P2 | 范围切换时尽量保留旧数据并标识更新中；如四请求部分失败，分别处理摘要与排行，避免一个排行失败使整页变为错误卡。 | 旧数据不会被误认为新范围；失败维度可单独重试；请求序号仍阻止过期响应覆盖新范围。 |
| P2 | 在数据量足够时增加一句基于已有数值的事实摘要，例如「观看时长主要集中于前几项」，不推断趋势。趋势或环比需先有可比较时间窗的后端契约。 | 文案由真实聚合数据驱动；零数据与样本很少时不输出误导性结论。 |

**实施边界**：先做页面局部信息层级和渐进展开，不改变现有 API 或统计口径；分请求错误处理只影响页面状态，不修改服务契约。趋势或环比能力仍需单独评估后端契约。验证桌面约 1280px、375px、中英日三语、深浅主题，以及加载/空/错误/无分母/AI 关闭状态。

可交互的静态预览见 [个人洞察 HTML 原型](2026-09-24-personal-insights-ui-prototype.html)。原型使用示例数据，保留为时间范围、摘要层级、排行渐进展开和 AI 解读卡片位置的视觉参考；正式页面使用真实服务数据。

**实施记录（2026-09-24）**：已将 3×2 摘要、紧凑范围控件、统计口径折叠、排行前五项与展开、旧范围标注、独立排行加载/重试，以及满足样本门槛时的事实摘要落到正式页面。未改变两个 Insights API、归因规则或全局令牌。组件测试覆盖范围切换、过期响应、局部失败与重试；Chromium E2E 覆盖桌面/375px 网格、无横向溢出和 44px 手机范围控件。HTML 原型继续保留为视觉参考，数值仍是示例。

**页面文案调整（2026-09-24）**：按用户反馈移除页首介绍、统计口径折叠区、偏好排行归因段落及「非互斥归因」徽标。保留指标卡片的简短提示与实际统计口径；上文为当时评估与实施过程记录，不作为当前页面文案规范。

## 8. 实施切片与最小提交顺序

每个编号都是可独立验证的行为单元；测试与实现可同提交，跨功能文档不混入代码提交。

1. `docs(prd): specify personalization milestone`：计划与 REQ-0019～REQ-0022。
2. `feat(library): add canonical saved-view filters`：影片列表筛选契约、Web / Mock 一致性与索引。
3. `feat(library): persist saved views`：migration、repository、API、service adapter。
4. `feat(library): expose saved-view controls`：资料库 UI、i18n、无障碍与浏览器 QA。
5. `feat(recommendations): persist explicit feedback`：反馈模型、API、幂等与管理。
6. `feat(recommendations): explain and apply feedback`：reason codes、生成器影响、首页 UI。
7. `feat(actors): preview canonical merges`：alias schema、解析和只读 preview。
8. `feat(actors): apply audited canonical merges`：确认、事务 apply、冲突规则和 UI。
9. `feat(insights): aggregate personal metrics`：后端指标口径、范围和 fixtures。
10. `feat(insights): expose personal insights`：懒加载页面、入口、i18n、响应式 QA。
11. `docs: verify personalization workflows`：PRD evidence、项目事实、API、README、CLAUDE、Feature Inventory 与架构 HTML。

未经用户明确要求不执行 `git push`。工作区中的其他未跟踪规划文档和 `docs/plan/2026-04-01-network-link-analysis.md` 用户修改不得加入这些提交。

## 9. 验证矩阵

每个切片先运行专项测试；每个 D1～D4 子需求验收时执行与风险相称的全量门禁：

```powershell
npx -y pnpm@11.0.0 typecheck
npx -y pnpm@11.0.0 lint
npx -y pnpm@11.0.0 test
npx -y pnpm@11.0.0 test:electron
npx -y pnpm@11.0.0 test:e2e
npx -y pnpm@11.0.0 build
npx -y pnpm@11.0.0 build:electron:main
```

后端从 `backend/` 执行：

```powershell
go test ./...
go vet ./...
```

PRD 与文档：

```powershell
python scripts/prd/prd_lint.py docs/prd/requirements.csv
python -m unittest discover scripts/prd/tests -v
git diff --check
```

真实浏览器 QA 至少覆盖桌面 1280×900 和移动端 375×812、console error/warning、无 overlay、无横向溢出、键盘路径、确认对话框、三语切换和 Web API 持久化。除非用户明确授权，不运行 `pnpm test:display`。本机未满足 CGO 时不声称 `go test -race` 已通过。

## 10. 文档与验收闭环

新增重要端点、migration 或架构边界后，同步：

- `.cursor/rules/project-facts.mdc`
- `API.md`
- `README.md`、`README.zh-CN.md`、`README.ja-JP.md`
- `CLAUDE.md`
- `docs/features/2026-05-03-feature-inventory.md`
- `docs/reference/architecture-and-implementation.html`
- `docs/prd/requirements.csv`
- 本实施计划与上位总计划

REQ-0019～REQ-0022 必须分别记录 implementation refs、test refs 和逐条 acceptance evidence。只有四项都达到 `verified`，Milestone D 才能标记为 `verified` 并进入 Milestone E。

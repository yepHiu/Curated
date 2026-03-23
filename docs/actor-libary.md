

---

**实现状态（当前仓库）**：已落地 **`actor_user_tags`** 表、**`GET /api/library/actors`**、**`PATCH /api/library/actors/tags?name=`**、前端路由 **`/actors`**（侧栏「演员」）、**`ActorsPage` / `ActorLibraryCard`**；演员库筛选 query 为 **`actorsQ`**、**`actorTag`**。下列「现状摘要」中未划线的历史描述仍可用于理解影片侧标签与 `actor=` 片单；评估表可作对照，不必再当作阻塞项。

---

现状摘要

数据层

- actors 表：name（唯一）、avatar、简介/主页/身高/生日等（见 0008_actor_profile.sql）。
- movie_actors：影片与演员的关联。
- 影片标签：元数据标签走 tags（type 含 nfo）+ movie_tags；用户标签是另一套（storage 里 lookupTagsGrouped 区分 metadata / user）。**演员用户标签**：表 **`actor_user_tags`**（迁移 `0009_actor_user_tags.sql`），与 `movie_tags` / `tags` 隔离。
  
API

- GET /api/library/actors/profile?name=：单名资料。
- POST /api/library/actors/scrape?name=：异步刮削资料。
- **GET /api/library/actors**：列表分页，query `q`、`actorTag`、`sort`、`limit`、`offset`。
- **PATCH /api/library/actors/tags?name=**：`{ "userTags": [] }` 整表替换演员用户标签。
  
前端浏览行为

- 资料库 URL 支持 actor=（精确名），与 q / tag 等取交集（LibraryView.vue + library-query.ts）。
- 详情页里点演员名 → browseByActor → 回到来源模式（library / favorites / …）并带上 actor=，清空 tag / q（DetailView.vue）。
- 有 actor=（或搜索框整段等于某演员名）时，列表会筛 movie.actors 包含该名；VITE_USE_WEB_API=true 时顶部有 ActorProfileCard（LibraryPage.vue）。
  
也就是说：「点演员 → 只看 TA 演的片」在详情 → 资料库这条链路上已经成立；**演员库入口与演员标签**已按上文的实现状态补齐。


---

需求对照评估

需求
现状
评估
演员库
无独立路由/页面；演员只能从全库影片里间接出现
需新增导航 + 页面（虚拟列表/分页）+ 建议 GET /api/library/actors（或等价：名称、头像、参演数量、可选排序）
演员标签系统
不存在
需新模型：例如在 actors 上 user_tags_json，或 actor_tags + 独立 tag 表（键与 movie_tags/tags 分离，避免和影片元数据标签混用）
与影片标签隔离
影片侧已是 metadata / user 两套；演员侧尚无标签
设计上新表或新字段 + 专用 API（如 PATCH /api/library/actors/{name}/tags）即可自然隔离；前端不要用影片的 tag= 去表示演员标签
点击演员卡 → 该片单
详情里点演员已实现；资料库内若要做「演员卡」需接同一套路由
新页面里卡片 router.push 到 library（或 favorites 等）并带 actor=精确名 即可复用现有筛选与（Web 下）资料卡


---

实现时建议提前定的点

1. 演员是否必有 actors 行：列表若只扫 movie_actors，可能与「仅出现在 JSON 但未关联」的边缘情况一致性问题——需与当前扫库写入逻辑对齐（一般以 actors + movie_actors 为准）。
2. actor= 精确匹配：与库里 actors.name / 列表里的 movie.actors 字符串必须一致；改名若未来支持，要同步关联。
3. 演员库内按「演员标签」筛选：需要新 query（例如 actorTag=），并在列表 API 或前端二次过滤中实现，不要复用影片的 tag=，以免和影片标签语义混淆。
4. Mock 模式：今日演员资料卡偏 Web API；演员库若依赖新 API，Mock 适配器要补列表与标签，否则行为分叉。
  

---

结论

- 「点演员卡只看待演影片」：核心交互已具备（详情 → actor=）；演员库页面主要是新入口 + 卡片导航到同一 query 约定。
- 「演员库 + 演员标签 + 与影片标签隔离」：全新能力，涉及 DB 迁移、合约、HTTP、前端列表与编辑；与现有影片标签系统并列扩展即可，不必改影片标签表结构。
  
若你后续要落地实现，在 Agent 模式下可从 actors 列表 API + 路由 actors + 标签 PATCH 拆任务最顺。
# Curated 后端优化计划

> 面向本仓库当前后端实现的阶段性优化计划。本文档只聚焦已经在仓库中落地的 Go 后端，不讨论尚未实现的 Electron / IPC / mpv 主进程架构。

## 执行状态（2026-04-25）

当前轮次已完成以下修复：

- 已完成：`PATCH /api/settings` 的失败回滚与回归测试补齐
- 已完成：删除 `library path` 时的数据库事务边界修复
- 已完成：Curated Frames 图片 / 缩略图接口的 MIME 类型纠正
- 已完成：Playback session 状态访问加锁与状态快照回归测试
- 未开始：P2 性能优化项（首页推荐、facet 聚合、搜索索引）

本轮对删除 `library path` 的语义再次明确如下：

- 删除 `library path` 只影响数据库中的“归属关系”和统计口径
- 对于位于该 `library path` 下、且不再被其他剩余 library root 覆盖的影片，只删除数据库记录，用于实现“解绑”
- 严禁删除该 `library path` 对应的本机实际目录
- 严禁删除该路径下的本地媒体文件、NFO、封面或其他磁盘文件
- 如删除过程中的数据库清理失败，必须整体回滚，不能出现“路径已删但影片只解绑一半”的中间态

## 本次已优化内容清单（用于反哺/复盘）

本次已经实际落地的优化项如下：

- 已优化：`PATCH /api/settings`
  - 增加失败回滚逻辑，避免多字段提交时出现“前面已生效、后面报错”的部分成功状态
  - 补充了对应回归测试
- 已优化：删除 `library path`
  - 删除流程收敛到单个数据库事务
  - 明确并保持“只解绑数据库，不删除本机目录和媒体文件”的语义
  - 补充了删除过程中失败即整体回滚的测试
- 已优化：Curated Frames 图片响应
  - `/image` 与 `/thumbnail` 改为按真实字节内容返回 `Content-Type`
  - 补充了 JPEG 场景回归测试
- 已优化：Playback session 状态并发安全
  - 为 `sessionState` 增加受控读写入口与锁保护
  - 补充了状态快照相关回归测试

本次尚未落地、仍保留在后续计划中的优化项如下：

- 未优化：首页推荐生成链路的候选集与排序性能
- 未优化：Curated Frames facet 聚合性能
- 未优化：资料库搜索查询与索引策略

## 1. 计划目标

本轮优化计划的目标不是“重写后端”，而是用最小必要改动提升当前后端在以下四个维度的质量：

1. 一致性
2. 并发安全
3. 响应正确性
4. 中期性能可扩展性

对应原则：

- 先修高风险行为问题，再做性能优化
- 先补事务边界和接口语义，再考虑重构
- 优先沿用现有分层，不做无关的大规模目录迁移
- 每一阶段都必须有可验证的完成标准

## 2. 输入依据

本计划基于以下材料整理：

- 后端代码审查文档：
  [2026-04-25-backend-code-review-bilingual.md](/C:/Users/wujiahui/code/jav-lib/jav-shadcn/docs/plan/2026-04-25-backend-code-review-bilingual.md)
- 当前后端实现目录：
  - `backend/internal/server`
  - `backend/internal/app`
  - `backend/internal/storage`
  - `backend/internal/playback`
  - `backend/internal/appupdate`

## 3. 优先级结论

建议按以下顺序推进：

### P0：立即修复（本次已完成）

- `PATCH /api/settings` 的部分成功问题
- 删除资料库路径时的非事务性问题
- Curated Frames 图片 MIME 类型错误问题

### P1：短期修复（本次已完成）

- Playback session 的并发访问同步
- 为高风险链路补足回归测试

### P2：中期优化（本次未开始）

- 首页推荐生成链路的候选集与排序成本
- Curated Frames facet 聚合性能
- 资料库搜索查询能力与索引策略

## 4. 分阶段计划

## 阶段一：修复设置更新的原子性问题（状态：已完成）

### 4.1 目标

确保 `PATCH /api/settings` 对调用方表现为“单次请求内的原子更新”，至少做到：

- 请求返回失败时，不留下隐蔽的部分成功状态
- 多字段提交时，前端能可靠地推断最终生效结果

### 4.2 涉及文件

- `backend/internal/server/server.go`
- `backend/internal/contracts/contracts.go`
- `backend/internal/app/app.go`
- `backend/internal/server/server_test.go`

### 4.3 计划动作

1. 先把 `handlePatchSettings` 拆成两个阶段：
   - 阶段 A：完成所有输入校验
   - 阶段 B：统一执行变更
2. 对纯配置项做“先组装 patch，再统一写入”的收敛处理，避免逐字段立即写盘。
3. 对带副作用的设置项单独定义处理策略：
   - `launchAtLogin`
   - `proxy`
   - `player`
   - `backendLog`
4. 明确失败语义：
   - 如果中途失败，是否允许补偿回滚
   - 如果不能完全回滚，是否要把接口改成显式 partial result
5. 当前建议优先采用“全部先校验，再尽可能统一提交”的方案，而不是把接口改成 partial apply。

### 4.4 完成标准

- 构造一个多字段 PATCH，请求中包含：
  - 一个合法字段
  - 一个非法字段
- 接口返回错误后，合法字段不应偷偷落盘或更新内存态
- `GET /api/settings` 返回值与失败前保持一致

### 4.5 风险说明

- 该 handler 目前承担的设置项太多，后续可能需要再做一次小型拆分
- 如果 `launchAtLogin` 等系统副作用无法完全纳入统一事务，需要单独设计补偿回滚

## 阶段二：修复资料库路径删除的事务边界（状态：已完成）

### 5.1 目标

确保删除 library path 时，数据库不会进入“路径已删、电影只删一半”的中间态。

### 5.2 涉及文件

- `backend/internal/storage/library_paths_repository.go`
- `backend/internal/storage/movie_delete_repository.go`
- `backend/internal/storage/library_paths_repository_test.go`
- 如有必要补充：
  - `backend/internal/server/server_test.go`

### 5.3 计划动作

1. 重构 `DeleteLibraryPathAndPruneOrphanMovies`，把以下操作纳入同一个事务边界：
   - 读取待删除路径
   - 删除 `library_paths` 记录
   - 读取 remaining library roots
   - 计算 orphan movie 列表
   - 删除 orphan movie 关联记录
2. 优化删除方式：
   - 尽量避免循环内逐条开启新的数据库语义分支
   - 可以先在事务内得到待删 movie IDs，再统一调用内部删除逻辑
3. 如果现有 `DeleteMovieRecordsOnly` 不适合事务复用，应抽出一个“事务内删除 movie records”的内部函数。

### 5.4 完成标准

- 注入一个中途失败场景
- 校验失败后：
  - `library_paths` 记录仍存在
  - 相关 movie 记录没有被部分删除
- 正常路径下现有行为不变：
  - overlapping root / nested root 逻辑仍正确

### 5.5 风险说明

- 当前删除逻辑与 movie 相关表较多，重构时要警惕遗漏 join 表
- 要避免把“只删数据库记录，不删磁盘文件”的现有语义误改掉

## 阶段三：修复 Curated Frames 图片响应类型错误（状态：已完成）

### 6.1 目标

确保 `/api/curated-frames/{id}/image` 与 `/api/curated-frames/{id}/thumbnail` 返回的 `Content-Type` 与真实字节类型一致。

### 6.2 涉及文件

- `backend/internal/server/playback_curated_handlers.go`
- `backend/internal/storage/playback_curated.go`
- `backend/internal/contracts/contracts.go`（若需要扩展 DTO 或存储字段）
- `backend/internal/server/curated_frames_p1_test.go`
- 如有必要新增测试文件

### 6.3 计划动作

建议分两步做：

#### 方案 A：最小改动修复

1. 在响应时对 blob 使用 `http.DetectContentType`
2. 用检测结果设置 `Content-Type`
3. 保持数据库 schema 不变

这是本阶段推荐方案，因为改动最小、收益直接。

#### 方案 B：更完整的结构化修复

1. 在存储层新增 image / thumbnail 的 MIME 类型字段
2. 写入时记录类型
3. 响应时直接使用持久化类型

这个方案更干净，但改动更大，适合后续迭代，而不是第一优先级。

### 6.4 完成标准

- PNG 上传时两个接口正确返回 `image/png`
- JPEG 上传时两个接口正确返回 `image/jpeg`
- 缩略图生成失败回退原图时，`thumbnail` 接口仍返回正确类型

### 6.5 风险说明

- 现有测试写死了 `image/png` 断言，修复时要同步更新测试预期
- 如果未来支持 WebP，需要确认检测结果与前端兼容性

## 阶段四：加固 Playback Session 的并发安全（状态：已完成）

### 7.1 目标

消除 `playback.Manager` 对 `sessionState` 的无锁并发读写，避免 race 和偶发状态不一致。

### 7.2 涉及文件

- `backend/internal/playback/manager.go`
- `backend/internal/playback/manager_sessions_test.go`
- `backend/internal/server/playback_sessions_test.go`
- 若新增并发测试，可补充新的 `*_test.go`

### 7.3 计划动作

1. 给 `sessionState` 加入局部锁，或把所有状态字段统一纳入 `Manager.mu` 保护。
2. 至少统一以下字段的读写同步策略：
   - `lastAccessedAt`
   - `finishedAt`
   - `lastError`
3. 明确快照构建入口只能通过一个受控函数读取这些字段，避免散落读取。
4. 在支持 CGO 的环境中执行：
   - `go test -race ./internal/playback`

### 7.4 完成标准

- `touchSession`、`markSessionFinished`、`buildSessionSnapshot` 不再存在裸写共享字段
- 在具备 `-race` 条件的环境中不报竞态
- 最近会话列表和单会话状态接口结果稳定

### 7.5 风险说明

- 不要为了修 race 把锁粒度放得过大，避免影响 HLS 文件访问热路径
- 如果改成 `sessionState` 自带锁，要避免与 `Manager.mu` 形成锁顺序问题

## 阶段五：首页推荐链路性能优化

### 8.1 目标

降低首页每日推荐生成时的全量数据装载和应用层排序成本。

### 8.2 涉及文件

- `backend/internal/app/homepage_daily_recommendations.go`
- `backend/internal/storage/library_repository.go`
- 如有需要新增候选查询：
  - `backend/internal/storage/...`

### 8.3 计划动作

1. 不直接复用前端展示型 `ListMovies` 作为推荐候选输入。
2. 增加更轻量的候选查询结构，只返回推荐逻辑所需字段，例如：
   - `id`
   - `rating`
   - `isFavorite`
   - `addedAt`
   - `actors`
   - `studio`
   - `cover/thumb available`
3. 把可前置过滤的条件尽量下推到 SQL 层：
   - 排除 trashed
   - 排除无基础素材的项目
4. 保留当前应用层的多因素重排能力，不建议一次性把完整策略下推到 SQL。

### 8.4 完成标准

- 推荐生成不再依赖“前端展示 DTO 全量列表”
- 推荐链路的输入数据量与字段量明显下降
- 现有推荐行为不出现明显倒退

### 8.5 风险说明

- 推荐算法目前已经包含曝光惩罚、演员多样性、片商多样性，优化时不能把行为悄悄改掉
- 建议在改动前后保留固定样本的推荐结果对比

## 阶段六：Curated Frames facet 聚合优化

### 9.1 目标

避免每次查询标签/演员 facet 时都全表扫描并逐行 JSON 反序列化。

### 9.2 涉及文件

- `backend/internal/storage/playback_curated.go`
- `backend/internal/storage/migrations/*.sql`

### 9.3 计划动作

推荐分两种演进路径：

#### 路径 A：SQL 聚合优化

- 如果当前 SQLite 运行时支持 JSON1，则优先评估 `json_each`
- 用 SQL 直接展开并聚合 `actors_json` / `tags_json`

#### 路径 B：写时维护索引表

- 新增 curated frame facets 辅助表
- 在插入和更新 tags / actors 时同步维护

当前更建议先验证路径 A，因为迁移成本更低。

### 9.4 完成标准

- 标签和演员 facet 查询不再依赖 Go 层全量 JSON 解码
- 大量 curated frame 数据下响应时间更稳定

### 9.5 风险说明

- 如果 SQLite JSON1 在目标环境中不可用，则需要回退到辅助表方案
- 辅助表方案要额外处理一致性维护成本

## 阶段七：资料库搜索能力优化

### 10.1 目标

提升资料库列表搜索��大库场景下的可扩展性，减少全表扫描压力。

### 10.2 涉及文件

- `backend/internal/storage/library_repository.go`
- `backend/internal/storage/migrations/*.sql`

### 10.3 计划动作

1. 先区分三类查询：
   - 精确过滤：`actor`、`studio`、`code`
   - 弱搜索：标题、摘要、用户覆盖字段
   - 组合过滤：多条件叠加
2. 对精确过滤优先补索引或归一化列。
3. 对自由文本搜索评估 FTS5。
4. 减少 `COALESCE / TRIM / NULLIF / LOWER` 直接出现在主查询谓词中的频率，必要时改用预归一化列。

### 10.4 完成标准

- 常见精确过滤不再主要依赖全表扫描
- 自由文本搜索路径有明确的后续方案：
  - 要么引入 FTS5
  - 要么接受现状并限定库规模

### 10.5 风险说明

- 这是中期优化，不建议与 P0 问题混在同一个提交中完成
- 引入 FTS5 会带来 migration、重建索引、同步策略等配套工作

## 5. 执行策略建议

建议按如下节奏推进：

### 第一周

- 完成阶段一
- 完成阶段二
- 完成阶段三

目标：

- 先把最危险的用户可见一致性问题和接口正确性问题消掉

### 第二周

- 完成阶段四
- 补测试与回归验证

目标：

- 把 playback 的并发隐患收口

### 第三周及以后

- 视库规模和性能反馈推进阶段五到阶段七

目标：

- 做中期性能建设，而不是抢在功能正确性前面做“表面优化”

## 6. 测试与验收计划

每个阶段完成后，至少执行以下验证：

### 后端基础回归

在 `backend/` 目录执行：

```bash
go test ./...
```

### 前后端联调影响验证

重点检查这些页面或功能是否受影响：

- Settings 页面
- Library Paths 增删改
- Curated Frames 浏览与图片展示
- Player 页面与播放会话状态
- 首页推荐展示

### 针对性补测建议

- `PATCH /api/settings` 多字段失败回滚测试
- `DeleteLibraryPathAndPruneOrphanMovies` 故障注入测试
- Curated Frames 非 PNG 输入测试
- Playback session race / 并发快照测试

## 7. 不建议现在做的事情

当前不建议把下面这些内容混入本轮优化：

- 大规模重写 `backend/internal/server` 路由结构
- 把当前后端改造成全新的架构层级
- 提前引入 Electron / IPC 抽象
- 把性能优化和高风险修复放进同一个巨型提交

原因很简单：

- 当前后端的主要问题不是“架构完全错误”
- 而是几个关键路径上的事务边界、并发控制和响应正确性还不够稳

## 8. 最终建议

如果只选一条最务实的路线，建议这样做：

1. 先修 `PATCH /api/settings`
2. 再修 library path 删除事务
3. 再修 Curated Frames MIME 问题
4. 然后处理 playback 并发同步
5. 最后把推荐、facet、搜索纳入性能优化排期

这是当前投入产出比最高、风险最低、最符合现有仓库阶段的一条优化路线。

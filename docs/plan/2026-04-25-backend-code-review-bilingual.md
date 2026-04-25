# 2026-04-25 Backend Code Review / 后端代码审查

## Scope / 范围

### 中文

本轮审查聚焦当前仓库中已经落地的后端实现，主要覆盖以下目录与链路：

- `backend/cmd/curated`
- `backend/internal/server`
- `backend/internal/app`
- `backend/internal/storage`
- `backend/internal/playback`
- `backend/internal/appupdate`

审查目标：

- 发现已经存在的行为级 bug、数据一致性风险、并发风险
- 识别中短期值得投入的性能和可维护性优化点
- 给出可落地的修复方向，而不是泛泛的风格建议

### English

This review focuses on the backend implementation that currently ships in the repository, with primary attention on:

- `backend/cmd/curated`
- `backend/internal/server`
- `backend/internal/app`
- `backend/internal/storage`
- `backend/internal/playback`
- `backend/internal/appupdate`

Review goals:

- identify real behavior bugs, data consistency risks, and concurrency risks
- surface practical performance and maintainability opportunities
- provide actionable remediation guidance rather than style-only comments

## Validation Performed / 本次验证方式

### 中文

- 读取了后端入口、HTTP 路由、设置持久化、播放会话、资料库路径删除、Curated Frames、首页推荐等关键实现。
- 执行了 `cd backend && go test ./...`。结果：所有后端包测试通过。
- 测试结束后出现环境级告警：`go: failed to trim cache ... Access is denied.`。这不属于仓库代码回归，但说明当前机器的 Go cache 清理权限受限。
- 尝试执行 `go test -race ./internal/playback`，但当前环境返回 `-race requires cgo; enable cgo by setting CGO_ENABLED=1`，因此并发问题结论来自代码路径分析，而不是 race detector 直接证据。

### English

- Read the backend entrypoint, HTTP routing, settings persistence, playback sessions, library-path deletion, Curated Frames, and homepage recommendation flows.
- Ran `cd backend && go test ./...`. Result: all backend package tests passed.
- The run ended with an environment-level warning: `go: failed to trim cache ... Access is denied.`. This is not a repo regression, but it does indicate local Go cache trim permissions are restricted on this machine.
- Attempted `go test -race ./internal/playback`, but the environment reported `-race requires cgo; enable cgo by setting CGO_ENABLED=1`. The concurrency finding below is therefore based on code-path analysis rather than direct race-detector output.

## Remediation Status / 修复状态

### 中文

以下问题在 2026-04-25 已完成修复或加固：

- 已修复：`PATCH /api/settings` 的部分成功问题
- 已修复：删除 `library path` 时的非事务性问题
- 已修复：Curated Frames 图片 / 缩略图 `Content-Type` 错误
- 已加固：Playback session 状态访问的并发安全

以下项目仍属于后续优化范围，尚未在本轮落地：

- 未开始：首页推荐生成链路的性能优化
- 未开始：Curated Frames facet 聚合性能优化
- 未开始：资料库搜索查询与索引策略优化

### English

The following items were fixed or hardened on April 25, 2026:

- Fixed: partial-apply behavior in `PATCH /api/settings`
- Fixed: non-transactional library-path deletion flow
- Fixed: incorrect `Content-Type` handling for Curated Frames image / thumbnail endpoints
- Hardened: concurrent access to playback session state

The following items are still planned follow-up optimizations and were not implemented in this round:

- Not started: homepage recommendation pipeline performance work
- Not started: Curated Frames facet aggregation performance work
- Not started: library search and indexing improvements

## Findings / 发现

### 1. `PATCH /api/settings` can partially apply changes and still return an error

Files:

- `backend/internal/server/server.go:1569-1749`

Severity:

- High

### 中文

`handlePatchSettings` 逐个字段执行持久化和副作用更新。前面的字段一旦成功写入，后面的字段如果校验失败或落盘失败，接口会直接返回 `400/500`，但之前的修改不会回滚。

这会导致“请求失败但部分配置已经生效”的状态。例如：

- 请求同时提交 `autoActorProfileScrape=true` 和非法的 `metadataMovieStrategy`
- `autoActorProfileScrape` 已经写入配置文件并更新内存
- 后续 `metadataMovieStrategy` 校验失败
- 前端收到错误，以为本次变更整体失败，但实际系统状态已经被部分修改

这类行为对设置页尤其危险，因为用户通常把一次提交视为一个原子操作。

建议：

- 在 handler 层先完成全部校验，再统一应用变更
- 对纯配置项优先构造一个合并后的 patch，一次性写入
- 对包含副作用的设置（例如 `launchAtLogin`）引入显式的事务语义或补偿回滚逻辑
- 为“多字段 PATCH 中途失败”补回归测试

### English

`handlePatchSettings` persists and applies each field one by one. Once an earlier field succeeds, any later validation or persistence failure causes the endpoint to return `400/500` without rolling back already-applied changes.

That creates a "request failed, but some settings are already live" state. Example:

- the request sends `autoActorProfileScrape=true` together with an invalid `metadataMovieStrategy`
- `autoActorProfileScrape` is already persisted and applied in memory
- validation for `metadataMovieStrategy` fails later
- the frontend sees an error and reasonably assumes the whole patch failed, but the backend state has already changed

This is especially risky for a settings surface, where users typically expect one submission to behave atomically.

Recommendations:

- validate all fields up front before applying any mutation
- merge pure config changes into one combined write
- introduce explicit transactional or compensating behavior for settings with side effects such as `launchAtLogin`
- add a regression test for a multi-field PATCH that fails mid-flight

### 2. Library-path deletion is not transactional and can leave a half-pruned state

Files:

- `backend/internal/storage/library_paths_repository.go:169-256`

Severity:

- High

### 中文

`DeleteLibraryPathAndPruneOrphanMovies` 先删除 `library_paths` 表中的路径，再查询剩余 root，然后逐条删除不再被任何 root 覆盖的电影记录。整个过程没有放在一个数据库事务中。

结果是，只要中途任意一步失败，就可能留下部分完成状态：

- 资料库路径已经删除
- 一部分 orphan movie 已经删掉
- 另一部分 orphan movie 还留着
- 方法最终返回 error

这会让上层拿到失败响应，但数据库已经进入中间态。后续再重试时，系统面对的也不再是原始状态。

建议：

- 用一个事务包住“删除 library path + 计算待删 movie + 删除关联 movie 记录”整个流程
- 如果担心长事务，可先在事务内收集待删 ID，再在同一事务里批量删除
- 至少补一条故障注入测试，验证中途失败时不会把 `library_paths` 先删掉

### English

`DeleteLibraryPathAndPruneOrphanMovies` first deletes the row from `library_paths`, then discovers remaining roots, and finally deletes uncovered movie records one by one. The full workflow is not wrapped in a single database transaction.

As a result, any failure in the middle can leave a partially completed state:

- the library path row is already gone
- some orphan movies were already removed
- some orphan movies are still present
- the method returns an error

That means the caller gets a failure response while the database has already entered an intermediate state. Any retry then starts from a different state than the original one.

Recommendations:

- wrap "delete library path + determine affected movies + delete related movie records" in one transaction
- if long transactions are a concern, compute the candidate IDs first and then delete in bulk within the same transaction
- add a failure-injection test that proves the path row is not removed if later pruning fails

### 3. Curated frame image endpoints hard-code `image/png` even when the stored bytes are not PNG

Files:

- `backend/internal/server/playback_curated_handlers.go:222-225`
- `backend/internal/server/playback_curated_handlers.go:248-251`
- `backend/internal/server/playback_curated_handlers.go:352-356`
- `backend/internal/storage/playback_curated.go:105-118`
- `backend/internal/storage/playback_curated.go:328-340`

Severity:

- Medium

### 中文

`handleGetCuratedFrameImage` 和 `handleGetCuratedFrameThumbnail` 都固定返回 `Content-Type: image/png`。但当前写入链路并没有保存 MIME 类型，且上传入口允许原始 `PNG/JPEG` 字节进入库。

更关键的是，缩略图生成失败时，代码会把原始 `raw` 直接当成 `thumbBlob` 回退保存：

- 如果原图是 JPEG
- `curatedthumb.PNG(raw)` 失败
- `thumbBlob = bytes.Clone(raw)`
- `/thumbnail` 仍然返回 `image/png`

这样会出现“HTTP 头声明 PNG，实际字节是 JPEG”的不一致，可能导致：

- 浏览器或图片库解码失败
- CDN / 客户端缓存错误类型
- 导出或二次处理链路误判格式

建议：

- 存储图片时一并记录 MIME 或文件格式
- 响应时用记录值或 `http.DetectContentType` 推断真实类型
- 给非 PNG 输入补测试，覆盖 image 和 thumbnail 两个接口

### English

Both `handleGetCuratedFrameImage` and `handleGetCuratedFrameThumbnail` always respond with `Content-Type: image/png`. However, the persistence layer does not store MIME type, and the upload path already allows raw `PNG/JPEG` bytes to be stored.

More importantly, when thumbnail generation fails, the code falls back to storing the original `raw` bytes as `thumbBlob`:

- if the source image is JPEG
- `curatedthumb.PNG(raw)` fails
- `thumbBlob = bytes.Clone(raw)`
- `/thumbnail` still responds as `image/png`

That creates a clear mismatch: the HTTP header says PNG while the bytes may actually be JPEG. This can cause:

- browser or image-decoder failures
- wrong cache metadata in clients or intermediaries
- incorrect downstream assumptions in export or post-processing flows

Recommendations:

- persist MIME type or image format together with the blobs
- use the stored format, or at least `http.DetectContentType`, when writing the response
- add tests that cover non-PNG input for both image and thumbnail endpoints

### 4. Playback session state is mutated from multiple goroutines without synchronization

Files:

- `backend/internal/playback/manager.go:728-737`
- `backend/internal/playback/manager.go:820-824`
- `backend/internal/playback/manager.go:827-858`

Severity:

- Medium

### 中文

`playback.Manager` 会在多个 goroutine 中读写同一个 `sessionState`：

- 请求读文件时通过 `touchSession` 更新 `lastAccessedAt`
- ffmpeg 退出等待 goroutine 通过 `markSessionFinished` 更新 `finishedAt` / `lastError`
- janitor 和状态读取路径又会读取这些字段生成快照

这些字段没有锁，也不是原子类型，因此存在数据竞争风险。即使暂时没有直接观察到崩溃，它也可能带来：

- `expiresAt` / `finishedAt` 读到不一致值
- 诊断接口偶发返回旧状态
- 在启用 `-race` 的环境下直接报 race

当前环境无法直接跑 race detector，因为 `-race` 需要 `CGO_ENABLED=1`，但从代码结构看，这个问题是明确存在的。

建议：

- 给 `sessionState` 增加互斥锁，或把快照字段集中纳入 `Manager.mu` 保护
- 保证 `touchSession` / `markSessionFinished` / `buildSessionSnapshot` 对共享字段采用统一同步策略
- 在支持 CGO 的环境里补一轮 `go test -race ./internal/playback`

### English

`playback.Manager` reads and writes the same `sessionState` from multiple goroutines:

- request paths call `touchSession` to update `lastAccessedAt`
- the ffmpeg wait goroutine calls `markSessionFinished` to update `finishedAt` and `lastError`
- the janitor and snapshot paths read those fields to build diagnostics

Those fields are neither mutex-protected nor atomic, so there is a real data-race risk. Even if it is not crashing today, it can still lead to:

- inconsistent `expiresAt` / `finishedAt` values
- stale or flickering data in diagnostics endpoints
- direct race reports once the package is tested with `-race`

I could not run the race detector in the current environment because `-race` requires `CGO_ENABLED=1`, but the code path itself is clearly unsynchronized.

Recommendations:

- add a mutex to `sessionState`, or keep all snapshot field access under `Manager.mu`
- make `touchSession`, `markSessionFinished`, and `buildSessionSnapshot` follow one consistent synchronization strategy
- run `go test -race ./internal/playback` in a CGO-enabled environment

## Optimization Opportunities / 优化机会

### A. Homepage recommendation generation scales by loading and sorting the full movie list in memory

Files:

- `backend/internal/app/homepage_daily_recommendations.go:102-125`

### 中文

当前实现通过 `ListMovies(... Limit: 10000)` 拉取最多一万条电影 DTO，再在应用层做打分、去重、曝光惩罚和排序。对于当前规模这可能够用，但库规模继续增长后会出现：

- 生成推荐时一次性搬运大量 DTO
- 不必要地构造前端展示字段
- 排序和评分成本完全落在应用层

建议：

- 为推荐场景提供更轻量的候选查询 DTO
- 把明显可下推的过滤条件在 SQL 层完成
- 如果推荐逻辑持续增长，可以考虑“候选集查询 + 应用层重排”的两阶段模型

### English

The current implementation fetches up to 10,000 movie DTOs via `ListMovies(... Limit: 10000)` and then applies scoring, dedupe, exposure penalties, and ranking in application code. That is acceptable for the current scale, but it will become more expensive as the library grows:

- a large DTO set is loaded at once
- frontend-facing fields are built even though recommendation generation only needs a subset
- all ranking cost is paid in the application layer

Recommendations:

- introduce a lighter candidate-query DTO for recommendation generation
- push obvious filtering down into SQL
- consider a two-stage "candidate query + application rerank" model if the recommendation logic keeps growing

### B. Curated frame facet aggregation parses every JSON blob in Go

Files:

- `backend/internal/storage/playback_curated.go:261-314`

### 中文

`ListCuratedFrameActors` / `ListCuratedFrameTags` 当前会读取整张 `curated_frames` 表，对每行 JSON 做反序列化，再在 Go 内存里计数。这在数据量上来后会成为明显热点。

建议：

- 若 SQLite JSON1 可用，优先考虑用 `json_each` 在 SQL 层做聚合
- 或者在写入 / 更新 tags、actors 时维护一张单独的 facet 索引表

### English

`ListCuratedFrameActors` and `ListCuratedFrameTags` currently scan the full `curated_frames` table, unmarshal JSON row by row, and count in Go memory. That will become a clear hotspot as the dataset grows.

Recommendations:

- if SQLite JSON1 is available, move the aggregation into SQL with `json_each`
- alternatively, maintain a dedicated facet index table on insert/update

### C. Library search is likely to full-scan large portions of `movies`

Files:

- `backend/internal/storage/library_repository.go:295-316`
- index inventory checked from `backend/internal/storage/migrations/*.sql`

### 中文

当前搜索条件依赖多段 `LOWER(... LIKE ?)`，还叠加了 `COALESCE / TRIM / NULLIF` 包装。这类表达式通常无法利用普通索引。现有 migration 里有关系表和若干功能表索引，但没有看到针对电影主搜索字段的专门搜索索引。

这意味着库继续增大后，列表搜索会越来越依赖全表扫描。

建议：

- 若继续使用 SQLite，可评估 FTS5
- 或至少为 `code`、常用精确过滤字段建立更明确的辅助索引 / 归一化列
- 把复杂的展示级 `COALESCE` 和搜索级字段拆开，减少每次查询的表达式负担

### English

The current search clause relies on multiple `LOWER(... LIKE ?)` expressions wrapped with `COALESCE / TRIM / NULLIF`. Those patterns typically bypass normal indexes. The current migration set includes indexes for relation tables and feature-specific tables, but not dedicated search indexes for the main movie search fields.

As the library grows, search will increasingly depend on full scans.

Recommendations:

- evaluate SQLite FTS5 if search remains an important surface
- at minimum, add clearer helper indexes or normalized columns for `code` and other exact-match filters
- separate display-time `COALESCE` logic from search-time indexed fields

## Suggested Next Steps / 建议的后续动作

### 中文

建议优先级如下：

1. 先修 `PATCH /api/settings` 的原子性问题
2. 再修 `DeleteLibraryPathAndPruneOrphanMovies` 的事务边界
3. 补 Curated Frames 的真实 MIME 处理
4. 在支持 `-race` 的环境里验证并修复 playback session 的并发访问
5. 再安排首页推荐、facet 聚合、库搜索这三项性能优化

### English

Recommended order of execution:

1. fix the atomicity issue in `PATCH /api/settings`
2. then fix the transaction boundary in `DeleteLibraryPathAndPruneOrphanMovies`
3. correct MIME handling for Curated Frame image responses
4. validate and fix playback-session concurrency in a `-race` capable environment
5. schedule the homepage recommendation, facet aggregation, and library-search performance work afterward

## Bottom Line / 结论

### 中文

当前后端的整体结构是清晰的，模块边界也已经比较成熟；这次没有发现“架构彻底失控”这类问题。主要风险集中在少数几个关键路径上的一致性和并发细节，而不是大面积设计失误。

换句话说，这个后端更像是“已经成型，但需要补齐事务边界、并发安全和数据格式一致性”的阶段。

### English

The backend structure is generally coherent, and the module boundaries are already in reasonable shape. I did not find signs of architecture-wide collapse. The main risks are concentrated in a small number of high-impact paths around consistency, concurrency, and response correctness rather than broad design failure.

In other words, the backend looks more like "structurally solid, but still needs transaction boundaries, concurrency hardening, and content-format correctness" than "architecturally broken."

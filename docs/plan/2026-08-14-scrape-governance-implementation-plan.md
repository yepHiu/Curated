# 刮削治理实施计划

日期：2026-08-14  
状态：in-progress  
关联需求：REQ-0023  
上位调研：[2026-08-14-curated-core-scrape-decoupling-and-optimization.md](2026-08-14-curated-core-scrape-decoupling-and-optimization.md)、[2026-04-01-network-link-analysis.md](2026-04-01-network-link-analysis.md) 第 16 节

## 范围决策

本仓库只治理现有 Curated 的刮削调度，让批量刷新、扫描和演员补全不再把本机网络打满。

明确不做：

- 不把 Metatube 抽成独立 Core / sidecar
- 不把当前项目改造成 SFW 通用媒体库；SFW 另建新项目
- 不改用户可见的 provider 列表文案，除非调度行为必须反映到 Settings 健康状态

## 2026-08-14 UI 调研：自动刮削 vs 单部刮削

后端流水线已经共用 `scrape.movie`，但前端按入口接了四套互不相通的反馈。用户觉得「自动」和「单部」显示不一样，主要是这一层，不是刮削算法不同。

| 入口 | 后端 | 进行中反馈 | 结束反馈 | 进度语义 |
|---|---|---|---|---|
| 详情页 / 资料库右键「刷新元数据」 | `POST /movies/{id}/scrape` → 立即返回单个 task | 右下角 `ScanProgressDock`；按钮转圈只持续到 HTTP 返回 | toast「元数据刷新完成/失败」+ 通知中心；详情页另有行内错误 | Dock 标题和计数仍按**扫描**文案渲染，刮削任务的 `scanProcessed` 等字段为空，显示 `-` |
| 资料库批量刷新 | 前端逐部调用同一个单片 API，并等待终态 | 底部批量条 `current/total`；同时仍占用 ScanProgressDock | 只有汇总 toast，没有每部刮削 toast | 有明确「第几部 / 共几部」 |
| 设置页「触发元数据刮削」/ 按目录刷新 | `POST /library/metadata-scrape` 只返回 `queued/skipped` | 按钮「正在排队…」，**不**跟踪任何 task | 行内「已为 N 部排队…后台执行」；没有 dock、没有逐片 toast | 排队即结束，后台刮削对 UI 不可见 |
| 扫描后自动刮削 | `scan.library` 子任务，带 `parentScanTaskId` | Dock 只跟踪父扫描（设置页全库扫描还隐藏 dock） | 手动扫描 toast 只报扫描结果；子刮削默认静默。仅目录监听的子刮削走「库目录监听：元数据刮削」 | 扫描计数 ≠ 刮削进度；扫描 dock 关掉后刮削仍可能在跑 |

另外两点会放大「看起来不像一回事」：

1. **任务创建时机不同。** 单片刷新先建 `running` 任务再等全局槽位，Dock 会立刻出现 `Scraping metadata for …`，实际可能还在排队。扫描自动刮削是拿到槽位后才建 task，所以最近任务里也会更晚出现。
2. **文案语言不一致。** 前端 toast 已翻译；task `message` 仍是后端英文。单部路径把这句英文直接塞进扫描 dock，自动扫描路径则显示中文扫描计数。

### UI 切片（进行中）：单部刮削改为 toast-only

已决定：单部 / 右键刮削与自动刮削同一套反馈，只用右上角 toast + 通知中心，不再占用右下角 `ScanProgressDock`。

- `scrape.movie` 即使被 tracker 跟踪，也不进入 `progressTask`
- 详情页 / 右键刷新：`notifyMovieScrape: true` + `hideProgressDock: true`
- 确认刷新后立刻弹出右上角 loading toast（圆弧 `Loader2`），HTTP 返回后同一条 toast 跟踪任务，终态 success/warning/error 原地替换，不再出现空白等待
- 详情页副标题在年份、影片格式后展示 `metadataProvider`（例如 `2026 · mp4 · JavBus`）；未刮削则省略
- 资料库批量刷新：仍用底部批量条的 `current/total`，但不再弹出扫描进度卡；完成仍走汇总 toast
- 扫描 / 导入进度卡暂时保留，不在本切片拆除

## 当前问题

| 入口 | 现状 | 风险 |
|---|---|---|
| 扫描后 `enqueueScrape` | 有 `scrapeSem`，但获取时不听 context | 关闭应用可能卡住；扫描路径相对安全 |
| 单片 / 批量 `startAsyncMovieMetadataScrape` | 立即 `go`，不走 `scrapeSem`，也不进 `scrapeWg` | 批量刷新 N 部影片瞬时 N 条 pipeline；关闭时可能访问已关 DB |
| `auto-global` | `SearchMovieAll` 对约 35 个 provider 扇出 | 与上面相乘，峰值接近 `N × 35` |
| chain | 搜索成功即记健康，详情失败整任务失败 | 明明后面还有源，却直接失败 |
| 冷却 / 延迟 | 已记录，调度不用 | Settings 看起来有熔断，实际没有 |
| `asset.download` | 每任务 3 路，任务之间无全局上限 | 批量成功后图片并发约 `N × 3` |
| 动态代理 | 改环境变量，既有 transport 可能不刷新 | ping 成功、刮削仍超时 |

## 切片

### G1 统一刮削并发入口（本轮，进行中）

所有 `scrape.movie` / `scrape.actor` 进入同一有界槽位：

- 获取槽位必须 `select` 监听 context，取消时不能永久堵在 `scrapeSem <-`
- 手动刷新和批量刷新也要 `scrapeWg.Add`，`Close()` 才能排空
- 同一 `movieId` 已有 in-flight 刮削时，后续刷新返回已有 task，不再开第二条 pipeline
- 默认 `scraper.maxConcurrent` 仍为 4，语义改为「全应用同时进行的刮削 pipeline 数」，不再只约束扫描路径

验收：`MaxConcurrent=1` 时两部不同影片不能并行进入 `Scrape`；同一影片连续刷新只产生一次 `Scrape`。

### G2 全局资源下载上限

`assets.maxConcurrentDownloads` 改为进程级上限，而不是每个 `asset.download` 任务各自 3 路。封面优先于预览图。关闭应用时 drain 资源 goroutine。

### G3 健康状态参与 chain，详情失败继续

一次 provider attempt = 搜索 + 有效结果 + 详情 + 最低字段质量。任一步失败才记失败并尝试下一个源。冷却中的源在自动 / chain 模式默认跳过。

### G4 用有限竞速替换默认 `SearchMovieAll`

`auto-global` 先打健康最好或用户偏好的有限个源，命中高质量精确结果后停止。全源扇出不再是默认。

### G5 代理注入到 scraper transport

设置页保存代理后，刮削 client 使用显式 `ProxyURL` 或协调队列后重建 Metatube service，不依赖 `HTTP_PROXY` 的 `sync.Once`。

### G6 同编号去重与短缓存

`singleflight`、成功短 TTL、空结果负缓存、封面/缩略图 URL 去重。演员自动刮削保持现有 pending map，并加成功 TTL。

## G1 实现要点

现有三个入口改走同一 helper：

```text
enqueueScrape                     扫描
startAsyncMovieMetadataScrape     单片 / 批量 / Library Health 修复
enqueueActorProfileScrapeTask     手动与自动演员刮削
```

`startAsyncMovieMetadataScrape` 今天在创建 task 之后无限制 `go runMovieScrapeBody`。G1 改为：先登记 in-flight，再在 goroutine 里获取槽位，最后跑 `runMovieScrapeBody`。

不在 G1 改 provider 选择、不改 DTO、不改前端。

# 刮削优化与 Curated-Core 拆分可行性

日期：2026-08-14  
状态：research  
2026-08-14 决策：本仓库下一步只做刮削治理；SFW 不在当前项目改造，另建新项目。实施入口见 [2026-08-14-scrape-governance-implementation-plan.md](2026-08-14-scrape-governance-implementation-plan.md)，需求 `REQ-0023`。  
关联文档：
- [2026-07-18-curated-compliance-and-metadata-source-decoupling.md](2026-07-18-curated-compliance-and-metadata-source-decoupling.md)
- [2026-04-01-network-link-analysis.md](2026-04-01-network-link-analysis.md) 第 16 节
- [2026-07-19-project-feature-quality-audit.md](2026-07-19-project-feature-quality-audit.md) Milestone E

本文回答三个问题：刮削现在还能怎么优化；把刮削单独拎成可替换、可独立更新的模块是否可行；如果拆，后续产品和仓库该怎么分层。它不是已批准实施计划。

## 1. 结论先行

可行，而且和 7 月合规方案是同一条拆分轴，只是命名方向相反。

用户现在的直觉是：

```text
Curated（资料库 / 播放 / 设置）
  └── Curated-Core（刮削，可独立更新，也可换成 SFW 版本）
```

7 月方案的命名是：

```text
Curated Core（中性本地资料库）
  └── metadata extension（用户自行安装的第三方元数据源）
```

技术上应采用后者：公开、可审计、可合规的那一半必须是「资料库应用」，不能是「刮削引擎」。如果把成人站点实现命名为 Core，后续公开分发、商店上架、SFW 替换都会把最敏感的部分放在产品中心位置。

推荐产品名：

| 部件 | 建议名称 | 职责 |
|---|---|---|
| 应用本体 | Curated | 扫描、资料库、播放、PIN、导入、备份、Insights、UI |
| 元数据宿主 | Metadata Host（内置于 Curated） | 队列、缓存、权限、超时、协议、任务进度 |
| 第三方源包 | Metadata Pack / Extension | 具体站点解析；NSFW Metatube 或 SFW TMDB/NFO 都是同一协议的不同包 |

「刮削包可独立更新、Curated 不用发版」只有 sidecar / 独立进程这一档做得到。把 `metatube-sdk-go` 抽成 Go module 仍然要重编 `curated.exe`，不能满足这个目标。

不建议现在立刻拆仓库。当前最高收益是先把调度器修好，再冻结 `curated.metadata.v1` 协议，最后才把 Metatube 迁出二进制。

## 2. 当前刮削事实

当前不是「刮削散落全仓库」，而是「适配层已经收口，调度和站点实现仍编译在同一个后端里」。

```text
扫描 / 手动刷新 / 详情页刷新 / 演员自动补全
        │
        ▼
  app.go 任务编排
  - enqueueScrape：扫描路径有 scrapeSem（默认 4）
  - startAsyncMovieMetadataScrape：批量刷新无全局 semaphore
  - auto actor scrape：成功刮影片后可能再排队
        │
        ▼
  scraper.Service 接口
  - Scrape / ScrapeActor / ListProviders / CheckProviderHealth
        │
        ▼
  scraper/metatube  （唯一实现）
  - 静态编译 metatube-sdk-go v1.3.2
  - engine/register.go 空白导入约 35 个成人 provider
  - 空 provider / auto-global = SearchMovieAll（全源并发）
        │
        ▼
  assets.Service 下载封面/预览/头像
  storage.SaveMovieMetadata 写入 SQLite
```

已经有利于拆分的边界：

- 业务层主要依赖 `backend/internal/scraper` 的 `Service` 接口和 `Metadata` / `ActorProfile` DTO。
- HTTP / SQLite / 播放器不直接 import SDK。
- 资源下载已经按 URL 工作，不需要知道站点 HTML。
- 7 月以后，release 不再复制本机 `library-config.cfg`，默认 HTTP 也收口到 loopback。

仍然绑死在应用里的部分：

- `backend/go.mod` 直接依赖 `metatube-sdk-go`，二进制必然携带全部站点实现和域名。
- `app.go` 对 `*metatube.Service` 做 type assert，用来算 `auto-cn-friendly` chain 和健康检查。
- `server.go` 同样 type assert provider health。
- `scanner/number.go` 内置 FC2 / Heyzo / Tokyo-Hot / 1pondo / Caribbeancom 和标准番号规则。
- Settings 仍有站点级文案和健康探测；空 chain 语义仍是「自动全源」，不是「关闭刮削」。
- `metadataSourcePolicy=disabled` 尚未落地。

因此：抽接口已经完成约 40%；抽进程、抽二进制、抽合规边界都还没做。

## 3. 先优化什么：不必拆仓也能做

最高收益不在换 SDK，也不在重写 DTO，而在调度。7 月网络复审仍然成立。

### 3.1 P0：所有刮削入口共用有界队列

扫描路径有 `scrapeSem`，批量刷新没有。一次刷新 N 部影片会立刻 `go` N 个 pipeline；若策略是 `auto-global`，每个 pipeline 再对约 35 个 provider 扇出。理论峰值接近 `N × 35` 次搜索，外加每片 3 路图片下载。

应变成：

```text
所有入口（扫描 / 单片 / 批量 / 演员）
  -> ScrapeCoordinator 有界队列
  -> worker = scraper.maxConcurrent
  -> 以 movieId + 规范化编号去重
  -> 元数据成功后再进入全局 AssetCoordinator
```

演员补全必须分队列或低优先级，避免占满影片 worker。

### 3.2 P0：健康状态真正参与调度

`ProviderRuntimeHealth` 已记录失败次数、延迟、10 分钟冷却，但 `scrapeWithChain` / `SearchMovieAll` 不读取它。冷却目前主要是 Settings 展示。

调度应跳过冷却源，按成功率 / EWMA 延迟微调顺序，并把「一次 attempt」定义成 search + 精确匹配 + details + 最低字段质量。现在是搜索成功就记成功，详情失败整任务失败且不回写健康。

### 3.3 P0：不要默认 `SearchMovieAll`

`auto-global` 应改成有限竞速：先打用户偏好或健康最好的 2～4 个源，命中高质量精确结果后取消其余请求。全源扇出只作为显式兜底，而不是默认。

### 3.4 P0：动态代理要注入 transport

`SetProxy` 改环境变量后，Go `ProxyFromEnvironment` 的 `sync.Once` 可能让已初始化的 Metatube client 继续走旧代理。设置页 ping 成功、实际刮削仍超时，就是这个缝。应显式 `http.ProxyURL` 或协调队列后重建 scraper service。

### 3.5 P1：缓存与去重

缺四件套：

- 同编号 `singleflight`
- 成功结果短 TTL 正缓存
- 空结果 / 稳定失败负缓存
- 封面与缩略图 URL 去重，避免同一张图下两次

演员自动刮削也要 TTL：头像和简介都在且未过期就跳过。

### 3.6 优化与拆分的关系

这些调度能力应留在 Curated 应用侧，不要放进站点包。原因：

- 队列、缓存、超时、去重与「打哪个成人站」无关。
- 换成 SFW pack 后，同样需要这套治理。
- 站点包只应回答 `search` / `details` / `health`，不负责把整台电脑打满。

建议顺序：先做 3.1～3.4，再冻结协议。否则会把现在的并发爆炸一起搬进 sidecar。

## 4. 拆分可行性：三种深度

### 方案 A：Go module / build tag（编译期拆）

把 `internal/scraper/metatube` 迁到 `github.com/.../curated-metadata-nsfw`，主应用用 build tag 或可替换实现。

| 能满足 | 不能满足 |
|---|---|
| 代码仓库分离、SFW 构建不链 SDK | 站点规则变了仍要重编/重发 Curated |
| 实现成本最低 | 公开 Core 仓库 git 历史和私有依赖仍可能暴露 |
| 适合内部双 SKU | 用户不能只更新刮削包 |

结论：可作为过渡，不能作为「Core 更新、Curated 不更新」的最终形态。

### 方案 B：sidecar 扩展进程（运行期拆，推荐）

Curated 内置 Metadata Host，通过 stdio JSON-RPC 调用户目录里的 `provider.exe`。

```text
%LOCALAPPDATA%\Curated\extensions\metadata-providers\
  nsfw.metatube\     或     sfw.tmdb\
    manifest.json
    provider.exe
```

协议最小集：

```text
provider.initialize
provider.capabilities
movie.search
movie.details
person.search
person.details
provider.health
provider.shutdown
```

这能同时满足三个目标：

1. 站点 HTML 变了，只替换扩展包。
2. 换 SFW 包后，应用本体不变。
3. 干净安装的 `curated.exe` 不再包含 `metatube-sdk-go`。

成本：Windows 进程隔离、超时、崩溃恢复、hash/签名、代理注入、升级向导。估计 2～4 周，且必须先有桥接版本，不能直接从当前 1.5.1 跳到「二进制里没有 Metatube」。

### 方案 C：WASM

隔离更强，长期更漂亮，但宿主网络 API、文件系统权限和现有 Go SDK 迁移成本明显高于 sidecar。不建议作为第一刀。

## 5. 和现有应用怎么集成

推荐保持 Curated 为唯一用户可见产品，刮削包不是第二个 App。

```text
┌─────────────────────────────────────────────┐
│ Curated Desktop / Web UI                    │
│  library / player / settings / insights     │
└──────────────────┬──────────────────────────┘
                   │ HTTP /api 不变
┌──────────────────▼──────────────────────────┐
│ curated.exe                                 │
│  SQLite 资料库、扫描、播放、导入、PIN         │
│  Metadata Host：队列 / 缓存 / 权限 / 任务     │
│  内置 local/NFO provider（无网络）            │
└──────────────────┬──────────────────────────┘
                   │ curated.metadata.v1
         ┌─────────┴─────────┐
         ▼                   ▼
   NSFW pack            SFW pack
   Metatube 站点        TMDB / 本地 NFO
```

对现有 API 的影响应尽量小：

- `POST /api/library/movies/{id}/scrape`、`scrape.actor`、任务 SSE 继续由 Curated 拥有。
- Settings 从「选择 JavBus/JavDB」改为「已安装扩展 + 其内部 provider chain」。
- 未安装扩展时返回 `METADATA_DISABLED` / `METADATA_EXTENSION_NOT_FOUND`，本地浏览和已落库元数据仍可用。
- 废弃 `POST /api/proxy/ping-javbus`，改成扩展自己的 health。

升级体验必须是向导，不是让用户手改 JSON：

1. 覆盖安装后资料库、封面、评分、PIN 全部保留。
2. 首次启动二选一：继续仅本地资料库，或导入一个扩展包并确认网络权限。
3. 旧 `metadataMovieProviderChain` 由扩展 manifest 的 `legacyProviderAliases` 映射，核心不硬编码站点名。

这点 7 月方案第 11 节已经写过，仍然适用。不能静默把旧 Metatube 从 `curated.exe` 里「抽」成插件：当前它不是可校验的独立包格式。

## 6. 项目后续可以怎么分割

不要按「前端 / 后端」切，按「用户是否必须拥有、是否必须随应用发版、是否敏感」切。

```text
层 0  永远在 Curated 本体
      扫描编排、SQLite、播放、导入、备份、PIN、UI

层 1  可替换策略，但仍由本体执行
      文件名身份规则、NFO 读写、整理目录命名
      第一刀可继续留在本体；合规公开时把成人编号规则移入 NSFW pack

层 2  独立版本、独立发布的 Metadata Pack
      站点 HTML/API、演员资料源、provider 健康探针关键词

层 3  更远的产品面，共用层 0 + 层 2 协议
      Android / Server / 漫画库只消费同一 metadata 协议
      不要为每个客户端再嵌一套 Metatube
```

建议的仓库策略：

| 阶段 | 仓库 | 说明 |
|---|---|---|
| 现在 | 继续单仓 | 先做调度优化和 Host 接口，避免双仓同步税 |
| 桥接版 | 单仓 + `extensions/` 构建产物 | 仍编译 Metatube，但走 Host，验证协议 |
| 纯核心后 | `curated` 公开或中性仓 + 私有 `curated-metadata-nsfw` | 应用更新走 GitHub Releases；NSFW pack 不走同一发布页 |
| 可选 SFW pack | 第三个小仓或同一扩展仓的 `sfw` 包 | TMDB / 本地媒体信息，给合规安装使用 |

不建议再拆：

- Vue 和 Go 现在是一个桌面产品，为刮削拆前后端仓库没有收益。
- Electron 继续只做壳，不要让 scraper 走到 IPC。
- 不要把 SQLite schema 放到扩展里。影片、演员、标签是用户资料，不是站点实现。

## 7. SFW / 合规替换的真实边界

换成 SFW pack 只能去掉「内置成人元数据源」，不能自动把整个产品变成审核无感应用。

即使刮削迁出，本体里仍有：

- `scanner/number.go` 的成人站点编号
- UI「番号 / 片商」、路径示例 `D:\Media\JAV`
- `src/components/jav-library`、`window.javLibrary`、部分 `jav-*` localStorage key
- Mock / 测试夹具里的真实向样例
- 仓库名 `jav-shadcn`、公开 README / API 历史

所以合规有两档：

1. **分发隔离（推荐先做）**：安装包和 `curated.exe` 不含 SDK 与站点名；用户自己放 NSFW pack。
2. **源码中性化（公开仓库才需要）**：目录、文案、扫描规则、文档分层迁移。工作量明显大于 sidecar 本身。

插件化本身也不等于免责。如果官方更新通道自动下载、推荐或托管成人 pack，拆分在审计上仍然很弱。NSFW pack 必须与 Curated 官方 Release 分离。

## 8. 推荐路线，避免一次做两件大事

```text
切片 1  刮削治理（仍在当前仓库）
        全局队列、全局图片并发、冷却生效、chain 详情失败继续、代理注入
        去掉默认 SearchMovieAll

切片 2  协议与 Host（仍带 Metatube 适配器）
        curated.metadata.v1
        内置 local/NFO
        metadataSourcePolicy=disabled
        Settings 空状态改为「未安装元数据扩展」

切片 3  桥接发布
        升级向导、扩展导入、旧 chain 映射
        现有用户可继续刮，但需一次确认

切片 4  迁出 SDK
        私有 NSFW pack 独立版本
        核心 go.mod 删除 metatube-sdk-go
        可选 SFW pack

切片 5  仅当目标是公开源码时
        文案、扫描规则、目录名、文档中性化
```

切片 1 可以立刻改善现有 1.5.x 用户体验，且失败成本低。切片 4 才真正兑现「刮削更新不必发 Curated」。切片 5 才真正兑现「换 SFW 后仓库也像通用媒体库」。

## 9. 不建议的做法

- 把成人刮削仓库命名为 Curated-Core，再让应用去「依赖 Core」。公开叙事和依赖方向都会反了。
- 只清空 `metadataMovieProviderChain` 就宣称已解耦。当前空值等于全源搜索。
- 用 JSON/CSS 选择器规则替代站点代码。反爬、鉴权和 JS 页面覆盖不住，还把解析逻辑变相留在核心。
- 为了合规改写 git 历史或从公开页抹掉许可证归属。
- 在调度器未收口前就把 Metatube 挪到 sidecar，把 `N × 35` 的扇出一起搬出去。

## 10. 2026-08-14 决策

已选定：

1. **当前仓库先做刮削治理**（原切片 1），对应 `REQ-0023`。
2. **SFW / 合规产品不改造本仓库。** 把现有 Curated 中性化的性价比低于直接新建一个通用媒体库项目；本仓库继续服务现有资料库用户。
3. 因此本文第 8 节的切片 2～5（Metadata Host、sidecar、公开核心中性化）对本仓库暂停，不进入近期排期。

最小下一步已经转到 [刮削治理实施计划](2026-08-14-scrape-governance-implementation-plan.md)，第一刀是统一刮削并发入口，而不是新仓库。

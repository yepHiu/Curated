# Curated 项目网络链路梳理

本文聚焦当前仓库里与“刮削”直接相关的网络链路，重点回答：

1. 刮削时使用到哪些网络链路。
2. 刮削演员信息、刮削预览图时分别走什么链路。
3. 这些链路的关键差异、代理生效位置、稳定性风险点是什么。

本文基于当前实现代码，而不是理想架构文档。核心参考目录：

- `src/api`
- `src/components/jav-library`
- `backend/internal/app`
- `backend/internal/server`
- `backend/internal/scraper`
- `backend/internal/assets`
- `backend/internal/proxyenv`
- `backend/internal/storage`
- `docs/film-scanner/metatube-sdk-go`

## 1. 总览：项目里实际上有两层网络

当前项目的网络链路，至少要分成两层看：

- 第一层：前端浏览器 -> Curated 本地后端
  - 开发环境下，Vite 把 `/api` 代理到 `http://localhost:8080`
  - 生产构建默认把 API 指向 `http://127.0.0.1:8081/api`
  - 对应代码：
    - `src/api/http-client.ts`
    - `vite.config.ts`

- 第二层：Curated 本地后端 -> 外部站点 / 元数据提供方 / 图片资源站
  - 影片元数据、演员资料、封面、预览图，实际都在这一层向外发请求
  - 主要通过 `metatube-sdk-go` 和 `net/http` 实现
  - 对应代码：
    - `backend/internal/scraper/metatube/service.go`
    - `backend/internal/assets/service.go`
    - `docs/film-scanner/metatube-sdk-go/...`

这两层的区别非常重要：

- 前端到本地后端，是项目内部控制的同源链路。
- 本地后端到外站，是“真正的不稳定公网链路”，容易受代理、站点封禁、反爬、热链、地域限制影响。

## 2. 刮削主链路：从扫描到外站再回写本地库

### 2.1 触发入口

影片刮削有三种主要入口：

- 扫描导入后自动触发
  - `backend/internal/app/app.go`
  - `runScan -> enqueueScrape -> runScrape`

- 单片手动重刮
  - HTTP 接口：`POST /api/library/movies/{movieId}/scrape`
  - `backend/internal/server/server.go`
  - `handleRefreshMovieMetadata -> StartMovieMetadataRefresh`

- 按库路径批量重刮
  - HTTP 接口：`POST /api/library/metadata-scrape`
  - `backend/internal/server/server.go`
  - `handleMetadataScrapeByPaths -> StartMetadataRefreshForLibraryPaths`

### 2.2 核心执行流水线

后端真正执行影片刮削的核心函数是：

- `backend/internal/app/app.go`
  - `runMovieScrapeBody`

它的主流程是：

1. 根据设置生成 provider 选择策略
   - `movieScrapeOptionsForRun`
   - 支持 `auto`、`specified`、`chain`

2. 调用刮削适配层
   - `a.scraper.Scrape(ctx, result.MovieID, result.Number, scrapeOpts)`

3. 刮削适配层进入 Metatube
   - `backend/internal/scraper/metatube/service.go`
   - 自动 provider 模式：`SearchMovieAll`
   - 指定 provider 模式：`SearchMovie`
   - provider chain 模式：依次 `SearchMovie`

4. 从选中的 provider 拉详情
   - `GetMovieInfoByProviderID`

5. 将元数据写入 Curated 自己的 SQLite
   - `backend/internal/storage/metadata_repository.go`
   - `SaveMovieMetadata`

6. 异步下载资源文件
   - `backend/internal/app/app.go`
   - `runAssetDownload`
   - 内部调用 `backend/internal/assets/service.go`

### 2.3 影片刮削时到底会访问哪些外站

从 Curated 自己的代码看，真正的公网调用入口是 Metatube 的 provider 集合。当前仓库里参考源码 `docs/film-scanner/metatube-sdk-go/engine/register.go` 注册了这些 provider：

- `10musume`
- `1pondo`
- `aventertainments`
- `c0930`
- `caribbeancom`
- `caribbeancompr`
- `dahlia`
- `duga`
- `faleno`
- `fanza`
- `fc2`
- `fc2hub`
- `fc2ppvdb`
- `gcolle`
- `getchu`
- `gfriends`
- `h0930`
- `h4610`
- `heydouga`
- `heyzo`
- `jav321`
- `javbus`
- `javfree`
- `kin8tengoku`
- `mgstage`
- `muramura`
- `mywife`
- `pacopacomama`
- `pcolle`
- `sod`
- `tokyo-hot`

这意味着“自动刮削”不是只打一个站，而是可能并发打很多 provider，然后再按 Metatube 的优先级和相似度选结果。

几个最典型的真实站点例子：

- `javbus`
  - `https://www.javbus.com/`
  - 搜索与详情都直接打它

- `fanza`
  - `https://www.dmm.co.jp/`
  - `https://video.dmm.co.jp/`
  - 既有 HTML 页面抓取，也有 GraphQL 风格的数据获取

- `fc2`
  - `https://adult.contents.fc2.com/`
  - FC2 在 Curated 中还是单独 special case 过的

- `gfriends`
  - `https://github.com/gfriends/gfriends`
  - `https://raw.githubusercontent.com/gfriends/gfriends/...`
  - 主要用于演员图像补强

## 3. Provider 选择策略

### 3.1 普通影片

普通影片默认走：

- `SearchMovieAll(number, false)`

这会并发向所有注册 movie provider 搜索，然后按：

- 号码相似度
- provider 优先级

做排序，再选出最优结果。

这条链路的特点是：

- 命中率高
- 对公网依赖广
- 慢 provider 会拖整体搜索时间
- 某些 provider 偶发失败通常不会致命，因为还有别的 provider 兜底

### 3.2 指定 provider / provider chain

设置页可切换：

- `specified`
  - 只打一个 provider

- `chain`
  - 按用户配置顺序一个一个尝试

这条链路的意义是：

- 降低“全网并发搜索”的噪音
- 提高结果一致性
- 方便固定某个你更信任的数据源

### 3.3 FC2 特殊分支

Curated 的 Metatube 适配层里对 FC2 做了特殊分流：

- `backend/internal/scraper/metatube/service.go`
  - `isFC2 := mtnum.IsFC2(...)`
  - `searchMovieFC2Providers`

当前逻辑会把 FC2 限制到专门 provider：

- `FC2`
- `fc2hub`

这意味着：

- FC2 不会像普通影片那样全 provider 广撒网
- 这样可以减少误命中
- 也说明 FC2 成功率高度依赖少数 provider 的可用性

## 4. 刮削时资源下载链路

影片元数据拿回来后，还会进入资源下载链路：

- `backend/internal/app/app.go`
  - `runAssetDownload`

- `backend/internal/assets/service.go`
  - `DownloadAllTo`
  - `downloadOne`

当前下载的资源类型包括：

- `cover`
- `thumb`
- `preview_image`

其执行特征：

- 使用 `http.Client`
- `GET sourceURL`
- 带 Chrome 风格请求头
  - `User-Agent`
  - `Accept`
- 并发数受 `maxConcurrentDownloads` 控制
- 响应体大小受 `maxResponseBodyMB` 控制
- 下载完成后写入本地缓存目录或影片目录
- 再把 `local_path` 回写到 `media_assets`

这条链路本质上和“元数据刮削”不同：

- 元数据刮削主要是在 provider 页面和 API 之间跳转
- 资源下载则是直接请求 provider 给出的图片 URL

所以一部片通常至少包含两段不同的公网行为：

- “搜详情”的页面/API 请求
- “下图片”的静态资源请求

## 5. 代理是如何生效的

### 5.1 后端统一代理入口

代理配置通过：

- `PATCH /api/settings`
- `backend/internal/app/app.go`
  - `SetProxy`

最终调用：

- `backend/internal/proxyenv/sync.go`
  - `Sync`

这个函数会把代理写到环境变量：

- `HTTP_PROXY`
- `HTTPS_PROXY`
- `ALL_PROXY`

### 5.2 为什么 Metatube 会吃到这个代理

Metatube 底层使用：

- `docs/film-scanner/metatube-sdk-go/common/fetch/fetch.go`
  - `cleanhttp.DefaultPooledClient()`

而注释和项目规则也已经明确说明：

- Metatube 与 Go 的默认 `net/http` transport 都会读取环境代理

所以：

- 影片元数据刮削
- 演员资料刮削
- 资源图片下载

理论上都能吃到这层环境代理。

### 5.3 代理探测接口只是诊断，不是刮削必经路径

项目里还有两个出站检测接口：

- `POST /api/proxy/ping-javbus`
- `POST /api/proxy/ping-google`

它们只是为了验证代理是否能连通：

- `https://www.javbus.com/`
- `https://www.google.com/`

这不是刮削流程本身，只是一个“你当前代理大概率可用吗”的探针。

## 6. 演员信息刮削链路

### 6.1 入口

前端演员资料卡会先读：

- `GET /api/library/actors/profile?name=...`

如果资料缺失，并且在 Web API 模式下，会进一步触发：

- `POST /api/library/actors/scrape?name=...`

对应前端代码：

- `src/components/jav-library/ActorProfileCard.vue`

对应后端入口：

- `backend/internal/server/server.go`
  - `handleScrapeActorProfile`

再进入：

- `backend/internal/app/app.go`
  - `StartActorProfileScrape`
  - `runActorScrapeBody`

### 6.2 核心执行流程

演员刮削的核心流程是：

1. 精确演员名存在于 `actors` 表
2. 创建任务 `scrape.actor`
3. 调用 `a.scraper.ScrapeActor(ctx, actorName)`
4. 进入 Metatube actor 流程
5. `UpdateActorProfile` 回写本地 SQLite 的 `actors` 行

### 6.3 Metatube actor 搜索实际怎么走

Curated 的 actor 刮削适配器在：

- `backend/internal/scraper/metatube/service.go`
  - `ScrapeActor`

它会：

1. 生成多个搜索关键词
   - 原始名
   - 折叠空格后的名字
   - 去掉空格后的名字

2. 对每个关键词执行：
   - `engine.SearchActorAll(kw, true)`

3. 从结果里用相似度挑最佳命中
   - `pickBestActorSearchResult`

4. 再调用：
   - `GetActorInfoByProviderID(pid, false)`

这里的 `false` 很关键：

- 它强制尽量走 provider 实时抓取
- 而不是只吃 Metatube 本地缓存

### 6.4 演员头像的特殊逻辑：会注入 Gfriends

Metatube 的 actor 流程里有一段很关键的逻辑：

- `docs/film-scanner/metatube-sdk-go/engine/actor.go`

当演员 provider 属于日文系 provider 时，会尝试用 `Gfriends` 补充图片：

- `GetActorInfoByID(info.Name)`
- 数据来自：
  - `https://raw.githubusercontent.com/gfriends/gfriends/...`

这意味着演员资料链路，经常不是只打一站，而是两段：

1. 原始 actor provider 搜索与详情
2. Gfriends 图像补全

### 6.5 当前演员链路的一个关键现状

演员资料回写到本地后，前端 `ActorProfileCard` 里的头像展示仍然是：

- `AvatarImage :src="profile.avatarUrl"`

也就是说：

- 演员头像当前并不会像影片预览图那样再做一轮本地缓存和同源代理
- 浏览器通常会直接请求外部头像 URL

这会带来几个后果：

- 头像是否显示，取决于浏览器能否直接访问外站
- 后端代理对这一步未必有帮助
- 如果外站做热链限制，头像可能在前端直接挂掉
- 这条链路比影片预览图更“裸”

## 7. 预览图刮削与展示链路

预览图其实分成“拿到 URL”和“显示图片”两段。

### 7.1 第一段：刮削阶段拿到预览图 URL

在影片元数据刮削阶段，provider 会返回：

- `PreviewImages []string`

Curated 将它存到：

- `movies.previewImages` 语义层
- `media_assets` 表中对应的 `preview_image` 行
  - `source_url`
  - 初始 `local_path` 为空

对应代码：

- `backend/internal/scraper/metatube/service.go`
  - `PreviewImages: cleanStrings(info.PreviewImages)`

- `backend/internal/storage/metadata_repository.go`
  - `replaceMediaAssets`

所以刮削完成的一瞬间，数据库里通常先只有“外链 URL”，没有本地图片文件。

### 7.2 第二段：异步把预览图下载到本地

随后 `runAssetDownload` 会异步下载这些预览图：

- `backend/internal/assets/service.go`
  - `DownloadAllTo`
  - `downloadOne`

下载完成后，`media_assets.local_path` 被更新。

这里有一个重要含义：

- 预览图本地化不是“刮削详情”的同步阶段
- 而是“刮削完成后另起一条 asset.download 任务”

所以前端在某些时刻可能看到的是：

- 先返回一批外链预览图 URL
- 稍后才逐步变成本地同源 URL

### 7.3 详情接口如何决定返回外链还是本地 URL

当详情页请求：

- `GET /api/library/movies/{id}`

后端会调用：

- `backend/internal/server/movie_poster_local.go`
  - `enrichMovieDetailLocalPosters`

如果发现本地预览图文件已经下载成功，并且路径合法，就会把：

- `previewImages[i]`

从外链 URL 改写成：

- `/api/library/movies/{movieId}/asset/preview/{index}`

真正服务这个同源地址的是：

- `handleGetMoviePreviewAsset`

### 7.4 前端展示阶段的关键区别

详情页组件：

- `src/components/jav-library/DetailPage.vue`

只认 `movie.previewImages` 里的 URL，然后交给：

- `MediaStill`

去加载。

所以前端并不关心来源是：

- 外站 URL
- 还是 `/api/library/movies/.../asset/preview/...`

它只是“照着 URL 直接加载”。

这就是当前预览图链路最值得注意的地方：

- 如果 `local_path` 已可用，预览图会变成同源图片，稳定性更高
- 如果还没下载好，或者下载失败，就仍然是外链直连

### 7.5 这条链路的风险点

预览图链路天然有两种失败模式：

- 元数据 provider 已返回 URL，但图片站热链限制，前端直接显示失败
- 后端图片下载失败，导致永远无法升级为同源本地 URL

因此预览图表现比演员头像更复杂：

- 它有一条“降级路径”
  - 外链直连
- 也有一条“理想路径”
  - 后端下载后转同源

## 8. 当前项目里演员链路和预览图链路的核心差异

### 8.1 演员资料链路

特点：

- 入口明确：`/api/library/actors/scrape`
- Metatube 会做 actor provider 搜索
- 常带 Gfriends 图像补全
- 结果回写 `actors` 表
- 前端头像通常还是直接请求外站 URL

一句话总结：

- “演员资料是后端抓、数据库存、前端外链显示”

### 8.2 预览图链路

特点：

- 它最初只是 provider 返回的一组图片 URL
- 先落 `media_assets.source_url`
- 后台再异步下载图片
- 下载成功后才会被详情接口改写成同源 `/asset/preview/{index}`

一句话总结：

- “预览图先是外链，后端下载成功后才能升级成同源资源”

### 8.3 两者最大的架构差异

演员头像当前没有形成“本地缓存闭环”：

- 没有 actor avatar 的 `media_assets`
- 没有 actor avatar 的下载任务
- 没有 actor avatar 的同源图片接口

而影片预览图已经有完整闭环：

- source_url
- 本地下载
- local_path
- 同源转发接口

这就是为什么预览图比演员头像更有机会“稳定显示”。

## 9. 网络层面的深一步分析

### 9.1 最重的公网链路其实不是前端，而是后端

真正复杂的外网访问都发生在后端：

- Metatube provider 搜索
- provider 详情页/API
- 封面下载
- 预览图下载
- actor 图像补全

前端多数情况下只是：

- 访问本地 `/api`
- 或者在少数资源字段上直接访问外链图片 URL

### 9.2 “自动 provider 搜索”意味着高命中，也意味着高不确定性

`SearchMovieAll` / `SearchActorAll` 的优势是命中率高，但副作用也明显：

- 同一操作实际会打多个外站
- 慢站、死站、被墙站、被限流站都可能混在里面
- 某个结果为什么选中，有时不直观

对排障来说，这意味着：

- “刮削慢”不一定是主站慢，可能是某个低优先级 provider 在拖
- “刮削失败”不一定是完全失败，也可能是最优站失败、兜底站也失败

### 9.3 代理只覆盖后端出站，不覆盖浏览器直接外链

当前代理最可靠覆盖的是：

- 后端 Metatube 请求
- 后端资源下载
- 后端 connectivity ping

但不天然覆盖：

- 浏览器直接加载的演员头像
- 浏览器直接加载的未本地化预览图

这意味着一个很典型的现象：

- 后端刮削成功了
- 数据库里也有头像 URL / 预览图 URL
- 但前端图片还是显示不出来

原因通常不是刮削失败，而是：

- 浏览器到外站的直连失败
- 或被热链策略拦截

### 9.4 预览图是否“变成本地”，决定了稳定性上限

对预览图来说，真正决定稳定性的不是“有没有刮到 URL”，而是：

- `asset.download` 是否成功
- `media_assets.local_path` 是否回写成功
- 详情接口是否把 URL 改写为同源本地接口

如果这一步没形成闭环，预览图就还是脆弱的外链。

### 9.5 演员链路目前比预览图链路更薄弱

从稳定性角度看，演员头像当前比预览图更脆弱，因为它缺少：

- 本地缓存
- 本地同源接口
- 显式下载任务
- 热链规避层

所以如果你接下来要优先治理网络稳定性，演员头像链路通常比预览图更值得优先改。

## 10. 建议你后续重点观察的点

如果你想继续深挖，这几个观察点最有价值：

- 看日志中 `scrape.movie`、`scrape.actor`、`asset.download` 的耗时拆分
- 区分“provider 搜索成功但图片外链前端失败”与“后端资源下载失败”
- 检查你的代理是否只让后端通了，而浏览器本身并没有通外网
- 关注 FC2 是否因为 provider 数量少而比普通番号更脆弱
- 关注演员头像是否经常来自 Gfriends，而不是原始 provider

## 11. 一句话结论

当前 Curated 的刮削网络链路，本质上是“前端请求本地后端，本地后端再通过 Metatube 和资源下载器访问多个外站”。其中：

- 影片元数据链路最完整，支持 provider 搜索、详情抓取、资源缓存回写。
- 演员资料链路能抓到资料并回写数据库，但头像展示仍高度依赖浏览器直连外站。
- 预览图链路最复杂，既可能是外链直显，也可能在异步下载后升级为同源本地资源。

因此，如果你的目标是提升网络稳定性、可控性、抗热链能力，那么最应该重点关注的不是“能否刮到 URL”，而是“这些 URL 最后有没有被收敛成 Curated 自己可控的同源资源”。

## 12. 面向中国大陆网络环境的优化建议

这一节专门从中国大陆使用场景出发看。当前日期为 2026-04-01，本分析仍然以仓库现有实现为准。

中国大陆网络环境下，真正影响体验的通常不是单纯“慢”，而是下面几类混合问题：

- 某些 provider 域名被 DNS 污染或连接不稳定
- 某些站点虽可访问，但 TLS 握手或首包很慢
- GitHub Raw、Google、部分海外图床可用性不稳定
- 图片热链限制使“浏览器直连外链”比“后端代理下载”更脆弱
- 自动并发访问过多 provider，导致一次刮削被最慢或最差的 provider 拖住

### 12.1 当前最值得优先优化的方向

如果只按收益排序，我会建议优先做这几件事：

1. 让“浏览器直连外链图片”尽量消失，统一改成后端同源输出。
2. 把 provider 从“默认全量并发”改成“大陆友好 provider 链优先”。
3. 给 provider 建立健康缓存和失败熔断，不要每次都试死链路。
4. 把资源下载和元数据搜索的超时、重试、并发拆开调优。
5. 给代理能力加“分域名 / 分 provider 验证”和更明确的降级路径。

下面我展开说。

### 12.2 优化点一：彻底收敛图片链路，避免浏览器直接跨境拉图

这是我认为当前最重要的一点。

现在项目里最脆弱的地方之一是：

- 演员头像大概率仍由前端直接请求外链
- 预览图在资源下载成功前，也可能由前端直接请求外链

这在中国大陆尤其容易出问题，因为：

- 浏览器直连图片域名时不会自动复用后端代理
- 某些图床或 GitHub Raw 在大陆可用性波动很大
- 热链限制通常对浏览器更敏感

最直接的优化方向是：

- 为演员头像建立和预览图同样的“source_url -> local_path -> 同源接口”闭环
- 预览图即便下载失败，也可以考虑增加一个“后端图片转发代理接口”作为兜底

建议优先级：

- P0：演员头像本地缓存化
  - 新增 actor asset 存储模型，至少支持 avatar
  - 后端统一下载并提供 `/api/library/actors/{name}/avatar` 之类的同源接口

- P1：预览图增加“按需代理转发”兜底
  - 当本地文件不存在且 source_url 存在时，由后端代取并回流
  - 可以选择只在开启代理时生效

这件事的价值非常高，因为它把“不稳定的跨境图片请求”从浏览器移到后端统一治理。

### 12.3 优化点二：默认不要全量并发所有 provider，应该提供“中国大陆优先链”

当前 `SearchMovieAll` / `SearchActorAll` 的优点是全，但大陆环境下副作用也最大：

- 会同时打很多海外站
- 某些 provider 很慢但并不常用
- 某些 provider 在大陆环境下几乎长期不可达

因此更合理的策略不是“永远 auto 全量”，而是：

- 保留全量模式
- 但默认给出一套“大陆友好 provider chain”

例如可以把模式设计为：

- `auto-global`
  - 维持现有全量 provider 搜索

- `auto-cn-friendly`
  - 只尝试一组更稳的 provider 链

- `custom-chain`
  - 用户手工排序

为什么这比简单调 timeout 更有效：

- timeout 只是减少等待
- provider 策略本身才决定你是不是在打大量已知劣质链路

你完全可以把当前设置页里的 `chain` 能力进一步产品化，预置几组模板，例如：

- “大陆优先”
- “FC2 优先”
- “无码站优先”
- “全源搜索”

### 12.4 优化点三：给 provider 做健康缓存、熔断和冷却时间

现在项目已经有：

- `POST /api/providers/ping`
- `POST /api/providers/ping-all`

这是个很好起点，但目前更偏手动诊断。

如果要提升真实刮削稳定性，建议把它前移到执行策略里：

- 某 provider 连续失败 3 次，进入 10-30 分钟冷却
- 冷却期内自动跳过，不参与本轮 `auto` 搜索
- 定期或按需再做健康探测

尤其在中国大陆网络下，这个收益很大，因为某些 provider 的“不可达”是长期态，不是偶发现象。

可以考虑维护一份运行时状态：

- last_ok_at
- last_fail_at
- consecutive_failures
- avg_latency_ms
- cooldown_until

然后在 provider 选择前做过滤。

这样做之后，一次刮削就不会反复撞同一堵墙。

### 12.5 优化点四：把“搜索超时”和“图片下载超时”分开调，不要一刀切

当前后端虽然已经区分了 scraper 和 assets 的 timeout，但从中国大陆网络特征来看，建议再更细一点。

原因是：

- provider 搜索通常是 HTML/API 首包问题
- 图片下载通常是静态资源传输问题
- 两者失败模式不同

更合理的建议是：

- provider 搜索
  - 首包超时更短
  - 总超时中等
  - 失败后尽快切到下一个 provider

- provider 详情页抓取
  - 比搜索略长
  - 但仍不应无限等

- 图片下载
  - 允许稍长总时长
  - 但首包也要限制

特别是预览图下载，不要因为某一张图卡住整个 `asset.download` 任务太久。可以考虑：

- 单图失败不拖垮整批
- 允许 partial success
- 先返回已有图，再后台补剩余图

### 12.6 优化点五：降低“最慢 provider 拖住整体体验”的影响

当前自动搜索的体验问题之一是：

- 用户感觉“这一部片刮削很慢”
- 实际上可能只是某些 provider 在尾部拖时间

可以考虑几种优化方向：

1. 首个可信结果优先返回
   - 如果高优先级 provider 已返回足够相似结果，就提前结束

2. 分阶段搜索
   - 第一阶段只搜高优先级少数 provider
   - 第一阶段无结果才进入扩展 provider

3. 并发分组
   - 快速组
   - 扩展组
   - 海外高风险组

这在中国大陆很实用，因为：

- 大部分时候不是缺“第 12 个 provider”
- 而是前 2 到 4 个 provider 能否尽快给出结果

### 12.7 优化点六：把 GitHub Raw / Gfriends 这类依赖视为高风险链路

演员图像补强目前有一条很特别的链路：

- `Gfriends`
- `raw.githubusercontent.com`

这在中国大陆是明显高风险点。

如果演员头像体验很重要，我建议至少做其中一个：

- 给 Gfriends 结果做强缓存，本地长期保留
- 提供可替换镜像源
- 在大陆模式下允许禁用 Gfriends 注入
- 或者把 actor avatar 的下载和缓存做成完全异步，不阻塞主 actor profile 保存

否则会出现一种很常见的问题：

- 演员资料本来已经能拿到
- 结果因为 Gfriends 这条图像补强链路慢或失败，整体体验看起来像“演员刮削不稳定”

### 12.8 优化点七：代理不应只验证 JavBus 和 Google，最好验证真实 provider 族群

当前项目的代理测试是：

- JavBus
- Google

这对“代理通不通”有帮助，但对“当前刮削会不会稳”还不够。

更贴近真实场景的方案是：

- 允许按 provider 分组探测
- 例如：
  - movie providers
  - actor providers
  - image hosts
  - github raw

甚至可以直接利用已有 provider health 体系，在设置页展示：

- 当前代理下推荐 provider 链
- 哪些 provider 在最近 10 分钟稳定
- 哪些 provider 连续失败，建议跳过

这会比“只测 JavBus/Google”更有运营价值。

### 12.9 优化点八：预览图下载应支持更强的局部成功语义

中国大陆环境下，预览图最常见的现实情况不是“全成功”或“全失败”，而是：

- 前几张能下
- 后几张卡住
- 某几张 403
- 某几张域名解析失败

所以建议把 `asset.download` 从“全有全无”心智调整成“局部可用即可上线”：

- 先写回已下载成功的 `local_path`
- 失败项单独记录
- 详情页优先展示已经同源化的图
- 剩余图继续后台重试

这样用户至少能先看到部分预览图，而不是整个任务一直挂着。

### 12.10 优化点九：给不同失败类型做更清晰的分类

如果后面真的要做稳定性优化，日志和任务状态里最好区分这些失败类型：

- DNS 失败
- TCP/TLS 失败
- 连接超时
- 首包超时
- 403 / 404 / 5xx
- 热链拦截
- 内容解析失败
- provider 返回空结果

原因是中国大陆网络问题里，“网络失败”和“站点逻辑失败”需要完全不同的处理：

- 网络失败更适合切换代理 / provider / 降级
- 解析失败更适合修 selector / provider 适配器

如果现在所有失败都只是 `scraper run failed`，后续很难做精细优化。

### 12.11 优化点十：给大陆环境做“本地优先缓存”的产品策略

从产品策略看，中国大陆环境尤其适合：

- 一次成功后，尽量长期缓存
- 避免重复访问相同外站

具体可以考虑：

- 影片元数据成功后，延长本地缓存生命周期
- 演员头像一旦下载成功，除非手动刷新，否则尽量不重新取
- 预览图已落地则不重复探测 source_url
- provider 搜索结果可短期缓存，避免短时间重复搜同一番号

这会显著减少跨境请求频次。

## 13. 我建议的落地优先级

如果让我按“实现成本 / 收益 / 中国大陆适配价值”来排，我会这样排：

### 第一优先级

- 演员头像本地缓存化并通过同源 API 提供
- 预览图失败时提供后端代理兜底
- 默认 provider 策略增加“大陆友好 chain”

### 第二优先级

- provider 健康缓存 + 熔断
- provider 分阶段搜索，而不是全量并发
- 资源下载改成 partial success 语义

### 第三优先级

- 代理检测从 `JavBus/Google` 扩展为 provider 级别
- 更细的错误分类与观测
- Gfriends 链路单独优化或可配置禁用

## 14. 最后结论

如果只用一句话总结我对当前项目的判断：

- 这个项目的主要稳定性短板，不在“有没有代理”，而在“太多关键资源仍然暴露为浏览器直接跨境访问”以及“auto 模式对 provider 过于理想化”。

站在中国大陆环境看，最有效的优化路线不是继续堆更多 provider，而是：

- 尽可能把资源请求收口到后端
- 尽可能减少对不稳定 provider 的盲打
- 尽可能把一次成功沉淀为长期本地缓存

这样做，性能会更稳，用户感知也会明显更好。

## 15. 还需要重点注意的反爬与干扰机制

除了网络慢、被墙、代理不稳之外，元数据搜刮还有另一类风险：站点并不是单纯“连不上”，而是在“能连上”的情况下故意给你脏数据、假内容、限流页、跳转页或热链拦截。这类问题在日志里经常看起来像“解析失败”或“结果不稳定”，但本质上是反爬。

对 Curated 这种“本地后端 + 多 provider 搜索 + 资源下载”的架构来说，反爬问题至少要分成五层看。

### 15.1 请求识别层：站点识别出你不是正常用户

最常见的识别手段包括：

- 固定或异常的 `User-Agent`
- 缺失 `Referer`
- Header 组合不像真实浏览器
- Cookie 缺失
- 请求顺序不符合正常浏览路径
- 同 IP 短时间大量搜索

从当前仓库看，项目已经做了一些基础处理：

- 后端资源下载会加 Chrome 风格 `User-Agent` 和 `Accept`
- Metatube 的部分 provider 会带 `Referer`
- 部分 provider 会主动注入 cookie
  - 例如 JavBus 的 `existmag=all`
  - 例如 FANZA 的 `age_check_done=1`

但这还只是“基础伪装”，不等于真正稳定。要注意的是：

- provider 搜索链路里，不同站点的反爬要求不一样
- 能搜到页面，不代表详情页和图片链接也能稳定拿到

建议：

- 保持 provider 级别的 header/cookie 配置能力，不要只靠全局默认值
- 对已知要求 `Referer` 的图片域名，在下载器里支持按 source provider 注入对应 `Referer`
- 如果后续新增 provider，不要只验证“能打开首页”，要验证“搜索 -> 详情 -> 图片”三段是否都需要额外 header

### 15.2 频率与节奏层：并发过高、请求模式过于机械

很多站点不一定立刻封你，而是根据访问节奏逐渐限流或返回脏内容。大陆环境下如果又叠加代理出口共享，就更容易触发。

当前项目里有两个容易触发问题的点：

- `SearchMovieAll` / `SearchActorAll` 会并发打多个 provider
- 扫描导入时可能短时间连续触发大量 `scrape.movie`

这在目标站眼里很像：

- 同一出口 IP 在短时间内做了大量相似查询
- 且查询间隔非常规则

建议：

- 给 provider 请求增加轻微随机抖动，而不是每次都完全同时发出
- provider 级限速，而不是只有全局 `scrapeSem`
- 区分“搜索请求”和“详情请求”的频率上限
- 扫描大批量导入时，把 scrape 队列做成真正的调度器，不要只是 goroutine + semaphore

更具体一点：

- 每个 provider 可维护独立的最小请求间隔
- 同一 movie 的多 provider 搜索可以分批进行
- 批量扫库时优先做本地入库，再缓慢补刮元数据

这类改造对稳定性帮助通常比单纯提高 timeout 更大。

### 15.3 内容污染层：返回的不是“错误”，而是误导性的 HTML / 文本

这类反爬最麻烦，因为它看起来像“请求成功”：

- 返回 200，但正文是验证页
- 返回 200，但是搜索空页
- 返回 200，但插入干扰文本
- 返回 200，但图片 URL 是降质或占位图

你在仓库里已经能看到一些 provider 对这类污染做了兼容：

- FC2 的标题清洗逻辑，会尝试移除 spam-like span
- FANZA / JavBus 等 provider 对图片 URL 会做规则化和 fallback

这说明一个现实：

- 站点返回的内容未必是干净内容
- 反爬不一定表现为 403/429，很多时候是“给你能解析但错误的数据”

建议：

- 在 provider 适配层增加“页面真实性校验”
  - 是否包含目标字段骨架
  - 是否命中已知风控关键词
  - 是否进入登录页 / 区域限制页 / 验证页

- 对关键字段做合理性校验
  - `title` 是否过短或重复番号
  - `previewImages` 是否全部为空且页面明明应该有图
  - `coverURL` 是否落到异常占位图域名
  - `actors` 是否被污染成无意义文本

- 对搜索结果和详情结果分别设置信任分数
  - 不要只看“有没有结果”
  - 要看“结果像不像真的”

### 15.4 图片热链与资源反爬层：元数据抓到了，但图拿不到

这层在当前项目尤其重要，因为图片链路和文本链路是分开的。

常见机制包括：

- 要求 `Referer`
- 要求特定 cookie
- 限制跨站浏览器访问
- 返回 200 但给占位图
- 同一图片链接短时间后失效

这会导致一种非常典型的现象：

- `previewImages` 已经刮到了
- `coverURL` 也有
- 但前端看不到图，或后端下载失败

建议：

- 资源下载器不要只拿 URL 就下载，最好带上“来源上下文”
  - source provider
  - referer page
  - headers strategy

- `media_assets` 除了 `source_url` 和 `local_path`，建议未来保留更多字段
  - `source_provider`
  - `referer_url`
  - `last_http_status`
  - `last_error`
  - `etag` / `last_modified`（如果有）

- 图片下载失败不应只记成通用错误，至少应区分：
  - 403 热链拒绝
  - 404 源已失效
  - 内容类型异常
  - 返回占位图

特别是演员头像，如果后续做本地缓存，这一层就必须设计进去，否则只是把“浏览器失败”换成“后端失败”。

### 15.5 搜索干扰层：搜索结果页故意混淆、降质或错排

某些站点不会直接阻止搜索，但会让搜索结果变差，比如：

- 只返回部分结果
- 返回相似但不完全匹配的结果
- 优先推荐广告或别的内容
- 番号格式被改写，导致相似度算法失真

当前项目依赖 Metatube 的：

- provider priority
- number similarity

去做排序，这在正常情况下有效，但在被干扰时可能会选错。

建议：

- 对搜索结果和详情结果之间增加交叉验证
  - 详情页返回的番号是否与请求番号一致
  - 标题与番号是否匹配
  - 搜索结果 provider 与详情页 provider 是否一致

- 对高风险 provider 降低“单次命中即可信”的权重
  - 某些 provider 更适合做补充源，不适合做第一命中源

- 对 FC2、无码、特殊番号维持专门分流
  - 这一点当前已经有了，建议继续保持

### 15.6 区域限制与“软封禁”层

中国大陆场景下，有些站点的问题不是传统反爬，而是区域性软封禁：

- 首页能开
- 搜索能开
- 详情页跳地区限制
- 样本视频不可用
- 图像 CDN 走另一域名，单独被限

FANZA 这类 provider 的代码里已经显式处理过 region error，这说明：

- 区域限制不是理论问题，而是现有 provider 已经碰到过的现实问题

建议：

- 任务错误里显式标注 `region_restricted`
- provider 健康探针不要只测 search，还要考虑 detail / asset host
- 同一个 provider 最好拆成：
  - search host
  - detail host
  - image host
  - preview video host

因为它们可能不是同一个网络风险级别。

### 15.7 动态页面与脚本依赖层

一些站点的真实数据并不直接在 HTML 里，而是在：

- 内嵌 JSON
- JS 变量
- GraphQL / XHR
- iframe 页面

项目里的 FANZA provider 就明显有：

- GraphQL
- script 解析
- iframe / ajax movie 跳转

这类站点的反爬风险在于：

- 前端脚本结构稍改，parser 就会失效
- 某些返回值在无 cookie 或异常 header 下会变形

建议：

- 对这类 provider 单独做回归测试样本
- 尽量提炼“结构探针”
  - 比如先检查 JSON schema 是否还存在
  - 再做详细解析

- 把“站点改版导致解析失效”和“网络失败”分开上报

否则你会很难判断应该修 provider 还是修代理。

### 15.8 污染本地缓存的风险

还有一个经常被忽略的问题：

- 一旦反爬页、错误图、脏数据被当成“正常数据”保存进本地 SQLite 或缓存目录，后续就会持续污染体验

例如：

- 把风控页里的标题存成影片标题
- 把占位图存成本地 cover
- 把空演员列表覆盖掉原有演员信息

当前项目里 Metatube 有一些 `IsValid()` 校验，但对“半真半假”的结果仍然不够。

建议：

- 写库前增加更严格的最小真实性判定
- 对已存在的高质量数据，避免被低质量结果覆盖
- 图片下载后可做一些轻量验证
  - MIME 类型
  - 最小尺寸
  - 最小字节数
  - 是否命中已知占位图 hash

尤其是 cover 和 actor avatar，这一点非常值得做。

### 15.9 我最建议增加的防干扰能力

如果从工程投入与收益比看，我最建议补的防反爬能力是这几项：

1. provider 级请求策略
   - 不同 provider 自己管理 header、cookie、referer、速率

2. provider 级熔断与冷却
   - 连续失败后短时间自动跳过

3. 结果真实性校验
   - 不把风控页、占位图、异常空数据直接写库

4. 图片链路上下文化
   - 下载图片时携带来源 provider 和 referer

5. 同源化资源输出
   - 尽量不要让浏览器直接碰外站图片

6. 错误分类
   - 网络失败、区域限制、热链失败、解析失败、脏页面分别统计

### 15.10 一句话结论

从中国大陆场景看，真正棘手的不是“网站封不封你”，而是：

- 站点在不完全阻断的前提下，给你慢、脏、假的内容

所以 Curated 后续如果要把搜刮稳定性做上去，不能只优化代理和 timeout，还要把以下三件事一起做：

- 控制请求行为像正常用户
- 识别返回内容是不是真的
- 不让脏结果轻易污染本地缓存

这三件事补上后，系统在大陆环境下的可用性会明显上一个台阶。

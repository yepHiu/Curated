# FC2 元数据获取可靠性

日期：2026-10-06

状态：verified（源码与本地/在线只读验证；未生产发布）

关联：现有刮削治理 G3/G5；FC2-3977618 的失败排查

## 已确认的问题

- FC2 官方的 HTTP 200 可能是商品不存在页面，不能当作有效元数据。
- Javten 搜索返回 Cloudflare 403 时，Metatube v1.3.2 的适配器返回空结果且没有错误。
- 旧 Javten 搜索只识别 Location，不能读取正常搜索页面中的作品链接。
- FC2 只选一个来源，官方缺少演员时不会使用其它来源补全。
- 通用来源链在搜索成功、详情失败后立即退出，不尝试后续来源。

## 本轮范围

参照 MDCz v0.18.0（295f828c76158c6411b52caf6fba9e7b0ee945ba）的行为，独立实现 Go 适配器，不引入 Electron 或外部刮削服务。

1. 为 FC2 官方、Javten、PPVDataBank 增加支持 context 和浏览器请求特征的客户端；复用现有出站代理，代理保存后下一次请求生效。
2. 自动 FC2 查询支持上述来源和 JavDB，有界查询；指定来源和自定义来源链继续尊重用户设置。
3. 精确核对 FC2 数字身份；搜索支持重定向、canonical、og:url 和结果链接；详情支持 HTML、JSON-LD 及简介演员信息。
4. 将成功来源的缺失字段补齐，保留主来源及其详情地址；每张补全图片须保留其原始来源与 Referer。
5. 403/验证页、商品不存在、解析失败分别记录，不再把访问受阻转换成“无结果”。
6. 修复普通影片来源链的详情失败接续与健康记录。

## 验证与边界

- 用本地 HTTP fixture 覆盖搜索链接、直接命中、JSON-LD、演员补全、错番号、403/验证页、官方商品不存在、取消和来源切换。
- 验证源链在详情失败后继续、指定源不扩大范围，以及来源错误保留。
- 从 backend/ 运行相关单包测试，随后 go test ./... 与 go vet ./...。
- 在线只做只读获取对照，不自动改写正在运行的生产资料库。FC2-3977618 的实际网络结果另行记录，浏览器请求特征不保证所有 Cloudflare 验证都能通过。
- 不新增 HTTP API、library-config 字段或数据库迁移。源码验证完成后记录提交与重启/升级生效条件。

## 实施记录

### 来源与调度

- 新增 `internal/scraper/fc2`，独立获取与解析官方、Javten、PPVDataBank、JavDB 的 FC2 详情，不依赖 SDK 对 403 的空结果处理。
- 自动字段优先级：`fc2hub → FC2 → PPVDataBank → JavDB`，最多两路并发、45 秒总预算；一个来源失败不影响其它成功来源。优先来源提供标题和主详情地址，缺失字段由后续来源补齐，较长简介可替换简短简介。
- 现有 `auto-cn-friendly` / `auto-global` 使用完整 FC2 链；指定单源保持单源，自定义链只查询 FC2 兼容来源，普通链无兼容来源时使用自动 FC2 链。没有新增配置键。
- PPVDataBank/JavDB 出现在来源设置与健康探测中。SDK v1.3.2 未注册 JavDB；本轮独立 JavDB 适配器只支持 FC2，普通影片自动链不加入它，指定它获取普通影片会说明当前只支持 FC2。
- 影片刷新、愿望清单与 Agent FC2 标题检索复用适配器。愿望清单身份冲突保持 `WISHLIST_IDENTITY_MISMATCH` 供人工复核。

### 请求、解析与下载

- 新依赖 `github.com/imroc/req/v3 v3.61.0` 提供 Chrome TLS/HTTP2/请求头特征，其当前 `ImpersonateChrome` 使用 Chrome120；MDCz 使用 impit Chrome142。两者不是同一网络实现，也不保证通过 Cloudflare。
- 每次请求读取当前出站代理，避免标准库环境代理首次读取缓存；请求上限 20 秒、响应上限 4 MiB、同站最多五次重定向，支持 context 取消。
- Javten 支持跳转、canonical/og:url、HTML 搜索链接、JSON-LD 数组与 `@graph`，以及 `■出演` / `名前` 演员说明。
- FC2、Javten、PPVDataBank 校验最终详情地址与已提供的作品标识；PPVDataBank 的旧式 `/article_search.php?id=...` 社交地址与 affiliate `aid` 一并核对。JavDB 只核对番号栏目，不接受相关作品文字或数字前缀近似匹配。
- HTTP 403 和 HTTP 200 验证页保留访问阻断错误；自动查询进入五分钟冷却，显式单源可重试。官方 HTTP 200 商品不存在页属于明确缺失，来源仍视为可访问。全部失败时返回逐源错误，分类优先保留访问阻断。
- `Metadata.AssetSources` 记录图片原来源/详情地址，保存到已有 `media_assets.source_provider/referer_url`；wishlist 下载也携带原详情 Referer。没有数据库迁移或新 HTTP API。
- 普通影片在搜索成功但详情失败/空详情时尝试后续来源，详情候选失败时继续后续有效候选；只有完整成功才记录健康，来源链全部失败时保留各来源原因。

### 本地验证

从 `backend/` 使用默认 Go 缓存运行：

- `go test ./internal/scraper/... ./internal/proxyenv/...`：通过。
- 图片 Referer 与 SQLite 图片来源上下文回归：通过。
- `go test ./...`：全量通过。
- `go vet ./...`：通过。
- `go build -o ../.workspace/fc2-server-check.exe ./cmd/curated`：通过。
- 拆分普通影片测试后 `go test ./internal/scraper/metatube`：通过，新增空详情回退与逐源错误保留覆盖。
- 最后补充官网商品 ID 校验和相邻元信息日期解析后，`go test ./internal/scraper/...` 与 `go vet ./internal/scraper/...`：通过。

HTTP fixture 覆盖精确搜索链接、跳转/元信息直达、JSON-LD、演员补全、四来源字段、错误 canonical/跳转/番号栏目、403/HTTP 200 challenge、官方商品不存在和请求取消。服务回归覆盖最多两路并发、部分来源失败、单源与自动策略、自定义链、冷却与显式重试、wishlist 错号、跨来源图片下载上下文、普通影片详情失败接续与健康记录。SQLite 回归检查实际落库的图片来源/Referer，同时主影片来源不变。

### 2026-10-06 在线只读结果

读取已安装 Server 的 `library-config.cfg` 代理设置，未输出代理凭据；用独立内存引擎和当前普通影片链 `AVBASE → JavBus → JAV321` 调用新 `Service.Scrape`。没有连接生产数据库或替换/重启生产 Server。

| 来源 | FC2-3977618 的实际结果 |
|---|---|
| Javten / `fc2hub` | HTTP 403，访问验证 |
| FC2 官方 / `FC2` | HTTP 200 商品不存在，明确无结果 |
| PPVDataBank | 成功：标题、85 分钟、封面、一张预览图；未返回演员 |
| JavDB | HTTP 200 验证页，明确访问错误 |

聚合调用约 1.95 秒，返回 `FC2-3977618` 与 PPVDataBank 详情地址，无错误。封面携带 PPVDataBank 详情 Referer 获取到 HTTP 200，WebP、356×356 可解码；只在内存中检查，未下载到生产缓存。此结果说明独立补充来源解决了该番号在当前网络下的获取失败，不代表 Javten/JavDB 验证已通过或未来所有影片均可取得演员等字段。

### 生效与交付

源码完成后按普通影片回退、FC2 多源获取、文档记录拆分本地提交。没有 push、发布或生产数据写入。安装环境需要包含本次改动的新 Server，重启/升级后在影片详情执行「刷新元数据」；现有失败任务不会因修改源码自动重新执行。

实现提交：`abb2ffd5`（普通影片详情失败回退）、`f6950253`（FC2 多源获取）。

## MDCz 参考

- [FC2 official adapter](https://github.com/ShotHeadman/mdcz/blob/295f828c76158c6411b52caf6fba9e7b0ee945ba/packages/runtime/src/crawler/sites/fc2.ts)
- [Javten/fc2hub adapter](https://github.com/ShotHeadman/mdcz/blob/295f828c76158c6411b52caf6fba9e7b0ee945ba/packages/runtime/src/crawler/sites/fc2hub.ts)
- [PPVDataBank adapter](https://github.com/ShotHeadman/mdcz/blob/295f828c76158c6411b52caf6fba9e7b0ee945ba/packages/runtime/src/crawler/sites/ppvdatabank.ts)
- [Field aggregation](https://github.com/ShotHeadman/mdcz/blob/295f828c76158c6411b52caf6fba9e7b0ee945ba/packages/runtime/src/scrape/fieldAggregation.ts)

参照其行为在 Go 适配层实现；不引入 MDCz 应用、Electron 浏览器自动验证流程或外部刮削服务。

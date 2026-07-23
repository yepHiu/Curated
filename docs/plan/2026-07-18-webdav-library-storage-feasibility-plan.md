# Curated 媒体库 WebDAV 存储可行性研究与实施计划

日期：2026-07-18  
状态：Feasibility review complete；等待产品范围确认，尚未开始业务实现  
范围：媒体库存储路径、扫描、播放、目录监听、导入、整理、资源文件与存储健康检测

## 1. 结论摘要

当前实现需要区分两种完全不同的“支持 WebDAV”含义：

| 场景 | 当前结论 | 说明 |
| --- | --- | --- |
| WebDAV 先由操作系统或第三方工具挂载为盘符、UNC 或 POSIX 挂载目录 | **有条件可用，值得先做兼容性验证** | Curated 看到的仍是普通绝对文件路径；现有扫描和播放链路理论上可以工作，但目录监听、在线检测、写入语义和大文件 seek 性能存在风险 |
| 在 Curated 中直接填写 `https://host/dav/...` | **当前不支持** | 前后端路径校验和几乎全部媒体 I/O 都建立在 `filepath`、`os.File`、`fsnotify` 与本地文件路径之上；URL 会在入库阶段被拒绝，后续链路也无法工作 |
| Curated 原生连接 WebDAV，第一版只读扫描和播放 | **技术上可行，但属于中等偏大的存储层改造** | 需要新增存储类型、凭据管理、远端列举、稳定媒体标识、HTTP Range 转发、轮询同步和能力降级，不应按“放宽路径校验”实现 |
| Curated 原生连接 WebDAV，并支持导入、整理、NFO/海报写回和删除 | **可行但风险与工作量显著更高，建议后置** | 依赖服务器对 `PUT`、`MKCOL`、`MOVE`、条件请求、覆盖规则和大文件上传的具体实现，必须在只读 MVP 稳定后单独决策 |

建议采用两级路线：

1. 先验证“操作系统挂载的 WebDAV 路径”是否已经满足真实使用场景；这是成本最低、对现有代码侵入最小的路径。
2. 只有在用户必须直接配置 WebDAV URL、不能接受系统挂载时，才建设原生 WebDAV 只读 MVP；不要一开始承诺远端整理和写入。

## 2. 本计划要解决的问题

本次不是先写 WebDAV 客户端，而是先回答以下决策问题：

1. Curated 当前所谓的“媒体库存储路径”究竟是一个字符串配置，还是贯穿业务的本地文件系统身份？
2. 使用系统挂载后的 WebDAV 时，扫描、浏览器播放、拖动 seek、元数据刷新、封面缓存、目录监听、导入和整理分别能否可靠工作？
3. 如果必须支持原生 WebDAV URL，最小可交付范围是什么，哪些能力应明确降级或禁用？
4. 如何避免把 WebDAV 密码放进 URL、SQLite 明文字段、日志或 API 响应？
5. 网络中断、认证失效、远端限流或目录暂时不可达时，如何确保 Curated 不误删数据库记录、不把半次扫描当作完整结果？
6. 什么样的验证结果可以判定“可行”，什么结果意味着必须引入本地缓存、放弃直接播放，或继续使用系统挂载？

## 3. 当前实现证据

### 3.1 路径模型只接受操作系统绝对路径

- 前端 `src/lib/path-validation.ts` 只认可 Windows 盘符、UNC 和 Unix 绝对路径。
- 后端 `backend/internal/storage/library_paths_repository.go` 在 `AddLibraryPath` 中调用 `filepath.Clean` 和 `filepath.IsAbs`；`https://...` 不是当前平台的绝对文件路径，会返回 `library path must be an absolute path`。
- `library_paths` 当前只保存 `id`、`path`、`title` 等本地路径语义，没有 `storageType`、远端 endpoint、远端根目录、凭据引用或能力字段。
- 影片的 `movies.location` 和资源的 `media_assets.local_path` 同样被当作本机路径使用，而不是可解析的媒体 locator。

因此，直接允许 URL 通过校验不仅无效，还会让 `filepath.Clean`、`filepath.Base`、`filepath.Rel` 等调用产生错误语义。

### 3.2 扫描与去重建立在本地目录遍历之上

- `backend/internal/scanner/service.go` 使用 `filepath.Walk` 递归遍历目录，并从 `os.FileInfo` 判断目录和普通文件。
- 扫描结果保存完整本地路径；文件名、父目录、扩展名和重复根均使用 `filepath` 计算。
- 当前扫描器遇到某个根的遍历错误会结束本次扫描，而不是把网络根错误标记为单根 partial failure 后继续其他库。

系统挂载 WebDAV 时，遍历可能工作，但其性能、超时和错误类型由挂载驱动决定。原生 WebDAV 则需要用 `PROPFIND Depth: 1` 分层列举，不能复用 `filepath.Walk`。

### 3.3 播放依赖 `*os.File`、seek 和本地路径

- `backend/internal/storage/movie_stream.go` 先把影片位置解析为绝对路径，校验它位于某个库根下，再执行 `os.Stat` 和 `os.Open`。
- `backend/internal/server/server.go` 的流接口把 `*os.File` 交给 `http.ServeContent`，浏览器 Range/seek 最终依赖底层文件实现 `Seek`。
- HLS/转码和媒体信息探测会把本地路径交给 `ffmpeg` / `ffprobe` 子进程。
- “在文件管理器中显示”、原生播放器启动同样需要后端所在机器能解析该文件路径。

系统挂载路径可能兼容这些行为，但原生 WebDAV 不能返回 `*os.File`。原生实现至少需要一个面向 Range 的远端读取接口；涉及 ffmpeg、ffprobe 或原生播放器时，还要选择“传远端 URL及认证参数”或“先落本地临时缓存”，后者更安全、兼容性更好但占用空间并增加等待时间。

### 3.4 自动监听不能视为 WebDAV 可靠能力

- `backend/internal/librarywatch/watcher.go` 明确说明网络文件系统的通知可能不可靠。
- 当前实现通过 `fsnotify` 为根目录下每个目录注册 watcher；这依赖操作系统挂载驱动能正确传播远端变化。
- 原生 WebDAV 没有与 `fsnotify` 等价的通用实时变更事件。

因此第一版应把 WebDAV 自动监听定义为“不保证支持”：系统挂载模式先实测，原生模式使用定时增量同步或手动扫描，不能复用本地 watcher 的产品承诺。

### 3.5 整理、导入和资料写回都假设本地可写文件系统

- `backend/internal/library/organize.go` 使用 `MkdirAll`、`Rename`，失败时执行 copy + remove。
- 开启 `organizeLibrary` 后，`movie.nfo`、封面和预览图会写入影片所在目录；关闭时资源写入本地 `cacheDir`。
- 普通导入与可续传导入在目标库根下创建目录、临时文件和 `.curated-import` staging，再以本地文件方式 rename/commit。
- 删除相关逻辑会对本地媒体路径和缓存路径调用文件删除。

这些动作无法通过一个通用 `os.File` 替代自动映射到 WebDAV。原生写入需要分别设计 `MKCOL`、`PUT`、`MOVE`、冲突检测、失败回滚与幂等性。

### 3.6 当前存储在线检测偏向本地卷身份

- `backend/internal/storagehealth/probe_windows.go` 通过 `GetDriveType` 和 `GetVolumeInformation` 探测盘符或 UNC 根，并用 `os.Stat` / `os.Open` 检查目录。
- `backend/internal/storagehealth/checker.go` 的核心状态是 `online`、`offline`、`volume_mismatch`、`path_missing` 和 `permission_denied`，并基于卷序列号做绑定。
- 当前检查按路径串行执行，没有为单个远程路径建立独立、强制的 I/O 超时；网络挂载卡顿可能拖慢整个检测请求。

对原生 WebDAV，更合适的状态是 endpoint 可达、TLS 错误、认证失败、根 collection 不存在、无列举权限、Range 不支持、限流或服务器错误，而不是卷序列号是否匹配。

## 4. 推荐产品边界

无论选择哪条路线，以下数据应继续留在本机：

- Curated 的 SQLite 数据库；
- `config/library-config.cfg` 以及其他运行配置；
- 日志；
- WebDAV 原生模式下的封面、预览图、演员头像和转码/探测临时缓存。

本计划只讨论“媒体文件可以位于 WebDAV”。不建议把 SQLite 数据库或运行时目录放到 WebDAV、网络盘或同步盘中。

### 4.1 系统挂载模式的建议边界

第一轮验证时建议：

- `organizeLibrary=false`，避免扫描后立刻对远端媒体执行移动、NFO 和海报写回；
- `autoLibraryWatch=false`，先以手动扫描验证正确性和性能；
- WebDAV 只作为媒体源，元数据资源继续落本地 `cacheDir`；
- 先验证扫描和直接播放，再验证 ffmpeg/HLS、导入、整理和删除；
- 明确要求 Curated 后端进程与 WebDAV 挂载处于同一个用户会话，并能访问相同盘符或 UNC。

如果这组能力已经满足需求，可以把最终交付收敛为“正式支持已挂载网络路径”，只补兼容性测试、错误分类、超时和文档。

### 4.2 原生 WebDAV MVP 的建议边界

第一版仅承诺：

- 配置一个 WebDAV collection；
- 验证连接与认证；
- 手动或定时只读列举影片；
- 保存稳定的库 id + 相对路径身份；
- 通过 Curated 后端代理远端 Range 请求供浏览器播放；
- 海报、预览图和刮削结果继续缓存到本地；
- 网络失败时保留既有影片记录，并把同步任务标记为 partial/failed。

第一版明确不承诺：

- `fsnotify` 实时监听；
- WebDAV 目录内自动整理；
- NFO、海报和预览图写回远端；
- 浏览器导入到 WebDAV；
- 远端文件删除、重命名或移动；
- “在资源管理器中显示”；
- 原生播放器直接打开带认证的 WebDAV URL；
- 对不支持 HTTP Range 的服务器进行无缓存即时 seek。

## 5. 建议架构（仅在需要原生 URL 时）

### 5.1 不把 URL 伪装成本地路径

建议给库路径增加显式存储类型，而不是让 `path` 同时承载文件路径和 URL：

```text
LibraryRoot
  id
  kind              local | webdav
  displayName
  localPath?        仅 local
  endpoint?         仅 webdav，不含密码
  remoteRoot?       WebDAV collection 内路径
  credentialRef?    指向安全凭据存储
  capabilities      list/read/range/write/move/delete/watch/reveal
```

影片物理位置建议逐步迁移为：

```text
libraryPathId + relativePath
```

而不是把用户名密码 URL 或平台相关绝对路径直接写进 `movies.location`。现有本地库可继续兼容 `location`，迁移应是增量式的。

### 5.2 引入领域化存储适配器

建议在后端新增独立包，例如 `backend/internal/librarysource`，定义 Curated 真正需要的领域能力：

- `ProbeRoot`：连接、认证、根目录与能力检查；
- `Walk` / `ListChildren`：可取消、可分页或可增量的目录列举；
- `Stat`：文件大小、修改时间、ETag、内容类型；
- `OpenRange`：按字节范围读取媒体；
- 可选写能力：`CreateCollection`、`Put`、`Move`、`Delete`。

本地适配器封装现有 `filepath/os` 行为，WebDAV 适配器封装 `PROPFIND/GET`。不要只实现一个表面上像 `os.File` 的大接口，因为写入、Range、watch 和远端条件请求并不具有相同语义。

### 5.3 远端流媒体代理

浏览器继续请求 Curated 的同源 `/stream` 接口，后端负责：

1. 校验 movie 与 library root 的归属关系；
2. 转发浏览器的单 Range 请求到 WebDAV；
3. 严格校验远端 `206`、`Content-Range`、长度和内容类型；
4. 以流式方式回传，不把整部影片读入内存；
5. 对取消、超时、认证过期、上游重置和重试返回稳定错误码；
6. 不把认证 header、密码或签名 URL暴露给浏览器和日志。

如果服务器忽略 Range 并总是返回 `200` 全文件，直接播放 MVP 应判为不通过；此时需要单独评估整文件本地缓存，而不是静默下载整部影片。

### 5.4 轮询同步代替 watcher

原生 WebDAV 使用后台同步任务：

- 对 collection 做分层 `PROPFIND Depth: 1`；
- 优先利用 ETag、Last-Modified、content length 和相对路径识别变化；
- 支持取消、超时、退避和每根独立失败；
- 完成整个根的一致性扫描前，不把“未看到”解释为“远端已删除”；
- 扫描摘要记录访问过的目录数、文件数、跳过数、失败目录和重试信息。

如果目标服务器支持更高效的变更同步扩展，可以作为后续服务器特定优化，不作为通用 WebDAV MVP 前提。

### 5.5 凭据与安全

- URL 中禁止携带 `user:password@host`。
- API DTO、任务元数据、通知和结构化日志不得返回密码、Authorization header 或可复用签名。
- 桌面版优先使用 Windows Credential Manager 等系统凭据存储；若需要跨平台，使用独立 `credentialRef` 与平台安全存储适配器。
- 明确 TLS 证书策略；默认不允许静默忽略证书错误。
- 限制重定向时的认证 header 传播，避免凭据被转发到不同 origin。
- 对远端 URL、转义、Unicode、`..`、重复斜杠和 percent-encoding 做统一规范化，防止跨库路径逃逸。

## 6. 分阶段执行计划

### Phase 0：确认真正需要的支持形态

目标：决定“系统挂载即可”还是“必须原生 URL”。

要确认的使用条件：

- 目标 WebDAV 服务类型和版本，例如 NAS、自建服务器、Nextcloud 类服务或 WebDAV 网关；
- Curated 主要运行平台；
- 用户是否可以接受先在操作系统中挂载；
- 是否只需要读取/播放，还是必须导入、整理和删除；
- 是否必须由原生播放器直接播放；
- 典型库规模、单文件大小、网络类型和外网访问需求。

决策门：若系统挂载可接受，进入 Phase 1；若必须直接填 URL，Phase 1 仍作为基线，然后进入 Phase 2。

### Phase 1：已挂载 WebDAV 兼容性 Spike

目标：不改存储模型，实测现有链路是否够用。

步骤：

1. 准备一个可控测试库，包含 MP4、MKV、中文/日文/空格/特殊字符目录、超大文件和嵌套目录。
2. 在目标平台把同一个 WebDAV collection 分别挂载为盘符/UNC 或 POSIX 挂载目录。
3. 以 `organizeLibrary=false`、`autoLibraryWatch=false` 登记路径并执行初次扫描。
4. 验证列表、详情、封面本地缓存、单片元数据刷新、浏览器直接播放和多位置 seek。
5. 验证 ffprobe/ffmpeg/HLS 路径；记录首帧耗时、seek 延迟、上游读取量和失败形式。
6. 验证重启、断网、认证过期、重新连接、盘符变化和服务端 5xx；确认数据库记录不会因暂时离线消失。
7. 单独试验 `fsnotify`，但无论结果如何都不把它作为通用 WebDAV 保证。
8. 最后才试验导入、整理、NFO 写回和删除，并使用可丢弃测试文件。

预期可能需要的最小修正：

- 为 UNC/映射盘补后端与前端自动化测试；
- 给网络路径的存储探测和扫描增加每根超时与更准确的 offline/auth/timeout 分类；
- 全库扫描按根隔离错误，避免一个网络根失败导致其他本地根也失败；
- 对 `driveType=remote` 默认关闭或提示关闭自动监听和自动整理；
- 在设置页明确展示“已挂载网络路径”，而不是声称原生 WebDAV。

Phase 1 通过标准：扫描结果正确；播放 Range/seek 不触发整文件重复下载；中断可恢复；错误不会误删资料；性能对目标网络可接受。

### Phase 2：原生 WebDAV 只读 PoC

目标：用最小代码证明原生 URL 的核心技术路径，而不是直接建设完整 UI。

步骤：

1. 建立 fake WebDAV 测试服务器，覆盖 `OPTIONS`、`PROPFIND`、`GET Range`、401/403/404、429、5xx、超时和断流。
2. 实现只读 WebDAV adapter：连接检测、Depth 1 列举、stat 和 range read。
3. 定义 `libraryPathId + relativePath` 的稳定媒体身份和路径规范化规则。
4. 用一个隔离 PoC 接通文件名识别与扫描结果，不先改整理、导入和删除。
5. 把一个影片的浏览器 Range 请求通过 Curated 转发到 WebDAV，验证起播和多次 seek。
6. 验证大文件过程中内存保持有界，并确认取消浏览器请求会及时取消上游请求。
7. 验证密码不进入数据库 URL、HTTP 响应、任务元数据和日志。

Phase 2 通过标准：目录列举正确；Range 语义正确；不下载整片也能 seek；认证和网络错误可分类；媒体身份可稳定重建；无凭据泄露。

若目标服务器不支持可靠 Range，决策应转为“整片本地缓存模式是否可接受”，而不是继续宣称直接播放可行。

### Phase 3：原生只读 MVP 产品化

目标：把 PoC 纳入正式存储模型和 UI。

步骤：

1. 扩展 contracts、SQLite migration 和 settings DTO，增加 `storageKind`、安全 endpoint 信息、`credentialRef` 与 capability DTO。
2. 引入本地和 WebDAV 两个 library source adapter，逐步把 scanner、stream 和路径归属判断迁移到新的 locator。
3. 增加 WebDAV 库配置、连接测试、状态显示和明确的只读能力提示。
4. 以同步任务替代 WebDAV watcher，并提供进度、取消、partial failure 和重试。
5. 资源文件固定走本地缓存；禁用远端整理、写回、导入、删除、reveal 和不安全的原生播放器交接。
6. 为多库并存、本地库回归和旧数据库迁移补齐测试。
7. 更新项目事实、API 摘要、配置文档和架构实现对照表。

### Phase 4：可靠性与性能加固

- 并发列举限流、每主机连接池、指数退避和 `Retry-After`；
- ETag/Last-Modified 增量同步与扫描 checkpoint；
- 大库分页式持久化，避免把所有远端条目同时保存在内存；
- Range 合并/预读策略，但不得破坏取消和内存上限；
- 认证续期、证书错误、DNS、连接、读取超时和服务器错误的稳定错误码；
- 离线库仍可浏览已索引元数据，但播放入口显示存储不可用；
- 避免并行扫描、播放和写操作把 NAS 或远端服务压垮。

### Phase 5：可选写能力，单独立项

按风险从低到高逐项评估：

1. 上传新文件：临时远端名 `PUT`，校验长度/ETag 后 `MOVE` 到最终名；
2. NFO/海报写回：条件写、冲突与覆盖策略；
3. 远端整理：`MKCOL + MOVE`，不支持原子 MOVE 时不得静默 copy + delete；
4. 远端删除：回收站/软删除优先，必须二次确认并可审计；
5. 原生播放器：优先受控本地缓存，不向命令行暴露长期凭据。

每项都需要服务器能力探测和兼容性矩阵，不能仅依据 WebDAV 标准声明推定所有服务器行为一致。

## 7. 测试矩阵

### 7.1 路径与协议

- Windows 映射盘符；
- Windows UNC/WebDAV redirector 路径；
- macOS/Linux 挂载目录；
- 原生 HTTPS WebDAV collection；
- 带空格、中文、日文、`#`、`%`、括号和组合 Unicode 的路径；
- 同名不同大小写文件，以及服务端大小写敏感/不敏感差异。

### 7.2 网络与认证

- 正确 Basic/Digest/Bearer 或目标服务器实际认证方式；
- 401、403、认证过期；
- TLS 证书过期、主机名不匹配、自签名策略；
- DNS 失败、连接超时、慢响应、读到一半断开；
- 429 + `Retry-After`、502/503；
- 重定向到同 origin 和不同 origin。

### 7.3 扫描一致性

- 初次全量扫描；
- 新增、改名、移动、删除；
- 扫描中网络中断；
- 多库中只有一个远端根失败；
- 同一影片在本地和远端根重复；
- 扫描取消后重启；
- ETag 缺失或不稳定。

### 7.4 播放

- MP4/WebM 浏览器直接播放；
- MKV 或需要 HLS/转码的影片；
- 文件开头、中间和尾部 seek；
- 多次快速拖动进度条；
- 浏览器取消请求；
- 服务端支持 206、忽略 Range、返回错误 Content-Range 三种情况；
- 大文件不整片下载、后端内存有界。

### 7.5 写操作（仅 Phase 5）

- 重名冲突；
- 上传中断与恢复；
- MOVE 不支持或跨 collection 失败；
- 配额不足；
- 写完但响应丢失时的幂等恢复；
- 删除失败与回收策略。

## 8. 决策门槛

在进入正式实现前，建议按以下门槛做 Go/No-Go：

| 决策门 | Go 条件 | No-Go / 改道条件 |
| --- | --- | --- |
| 系统挂载是否足够 | 目标平台可稳定挂载，扫描与播放体验满足需求 | 挂载配置不可接受、会话不可见或目标环境无法稳定维护挂载 |
| 原生目录列举 | 目标服务器的 PROPFIND 结果稳定，特殊字符和大库性能可接受 | 列举结果不一致、无合理超时/分页策略、服务器限流严重 |
| 原生直接播放 | 正确支持 Range/206，多次 seek 不触发整片下载 | 忽略 Range、Content-Range 错误或 seek 成本不可接受；改评估本地缓存 |
| 媒体身份 | `libraryPathId + relativePath` 能稳定映射且迁移不破坏本地库 | 仍依赖把 URL 塞进 `movies.location` 并让 `filepath` 处理 |
| 凭据安全 | 密码不落 URL/API/日志，可由安全凭据存储恢复 | 只能以明文 URL 或命令行参数长期暴露凭据 |
| 写能力 | 目标服务器实测支持所需 PUT/MOVE/冲突语义，失败可恢复 | 仅凭“支持 WebDAV”推定写入行为，或需要危险的 copy + delete 模拟原子移动 |

## 9. 工作量判断

这不是精确工期估算，而是相对规模判断：

| 方案 | 相对规模 | 主要工作 |
| --- | --- | --- |
| 只验证并正式支持已挂载 WebDAV/网络路径 | 小到中 | 兼容性测试、超时、错误隔离、状态提示、文档 |
| 原生 WebDAV 只读 PoC | 中 | WebDAV adapter、fake server、远端列举、Range proxy、locator 设计 |
| 原生 WebDAV 只读 MVP | 中到大 | 数据迁移、双存储适配、同步任务、UI、凭据、完整回归 |
| 原生 WebDAV 可写与整理 | 大 | 大文件上传、MOVE/冲突/回滚、写回、删除安全、服务器矩阵 |

最大的不确定性不是“Go 能否发 WebDAV 请求”，而是当前业务把文件路径当成统一身份使用。原生支持的核心成本在于把“影片属于哪个库、相对位置是什么、当前存储具有什么能力”从本地路径语义中拆出来。

## 10. 推荐下一步

优先执行 Phase 0 + Phase 1，并用真实目标 WebDAV 服务做一次可重复的兼容性 Spike。完成后输出一份结果表：

- 扫描是否正确及耗时；
- 播放起播与 seek 是否可靠；
- ffprobe/ffmpeg 是否可直接读取挂载路径；
- 断网和恢复后的行为；
- 自动监听是否可用但不保证；
- 写操作是否应永久禁用；
- 是否仍有必须原生 URL 的业务理由。

如果系统挂载已经满足需求，就以最小改动补齐正式支持。若不能满足，再按 Phase 2 做原生只读 PoC；PoC 通过 Range、身份与凭据三个关键门槛后，才进入产品化。

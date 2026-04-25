# 后端代码审查与改进建议

## 1. 审查范围

本次审查只针对当前仓库中可验证的后端相关代码，不把产品蓝图或未来目标架构当作已实现事实。

当前可审查对象主要包括：

- `docs/film-scanner/main.go`
- `docs/film-scanner/go.mod`
- `docs/film-scanner/metatube-sdk-go/` 下的 Go 代码

需要明确的是：

- 仓库根目录当前没有正式的 `backend/` 工程。
- `docs/film-scanner/` 更像本地扫描工具与参考实现，而不是已与前端原型打通的正式后端。
- 因此，本文结论更适合作为“现有后端参考代码的质量评估与演进建议”。

## 2. 总体结论

当前后端相关代码能够体现一定的工程基础，例如：

- 已有 Go 工具入口与模块化 SDK；
- 具备 provider 抽象、图片抓取、电影搜索、HTTP 服务入口等能力；
- 有一定的数据库、抓取、路由、鉴权与中间件组织。

但从正式产品后端的标准来看，这套代码仍存在较明显的稳定性与工程化缺口，尤其集中在以下几个方面：

- 资源释放不严谨；
- 错误处理存在 panic 或错误吞掉的情况；
- 配置入口不统一；
- 超时与取消机制不完整；
- 批量任务结果统计不可靠；
- 当前 CLI 与 SDK server 的行为模型割裂。

如果后续计划把这部分代码作为正式后端能力的基础，建议先修复高风险运行时问题，再统一配置模型、超时模型和任务结果模型，最后再考虑和前端产品契约对接。

## 3. 主要问题

### 3.1 高优先级问题

#### 3.1.1 预览图下载循环中延迟关闭响应体，存在资源泄漏风险

文件：`docs/film-scanner/main.go`

`downloadPreviews()` 在循环中对每次 `eng.Fetch()` 返回的 `resp.Body` 使用 `defer resp.Body.Close()`。由于 `defer` 会在整个函数返回时才执行，若预览图较多，会导致多个响应体长时间不释放。

风险：

- HTTP 连接和句柄积压；
- 长时间运行时可能触发资源耗尽；
- 批量扫描时稳定性明显下降。

建议：

- 在每轮循环内立即关闭 `resp.Body`；
- 或将单次下载逻辑提取到独立函数中，在函数内使用 `defer`。

#### 3.1.2 数据库初始化错误被直接丢弃

文件：`docs/film-scanner/metatube-sdk-go/engine/engine.go`

`engine.Default()` 中 `database.Open(...)` 的返回错误没有处理，数据库初始化失败时，问题不会在入口处暴露，而会在后续更深层的数据库操作中以更难排查的方式出现。

风险：

- 初始化失败不可见；
- 错误延后暴露；
- 增加定位和恢复成本。

建议：

- `Default()` 不应吞掉数据库错误；
- 若保留默认工厂函数，应返回 `(*Engine, error)` 或在入口层明确 fail fast。

#### 3.1.3 对外部返回值使用 `MustParse`，会导致 CLI 直接崩溃

文件：`docs/film-scanner/main.go`

`searchMovie()` 在拼接 provider 与 ID 后使用 `providerid.MustParse(...)`。一旦 provider 返回异常格式，整个扫描 CLI 会 panic，而不是记录错误并跳过当前文件。

风险：

- 单个异常结果导致整批任务中断；
- 对第三方 provider 返回质量过度信任；
- 不利于批量扫描场景的鲁棒性。

建议：

- 改用显式 `Parse` 并处理错误；
- 对单文件失败做局部降级，不影响整体扫描。

### 3.2 中优先级问题

#### 3.2.1 成功统计与真实执行结果不一致

文件：`docs/film-scanner/main.go`

当前逻辑只要影片信息查询成功，就会执行 `successCount++`，即使后面的 NFO 写入、封面下载、预览图下载失败，也仍然被记为成功。

风险：

- 汇总结果失真；
- 无法准确评估扫描成功率；
- 后续接入任务系统后难以定义可靠状态。

建议：

- 拆分结果维度，例如：
  - 元数据获取成功
  - NFO 写入成功
  - 封面下载成功
  - 预览图部分成功 / 全部成功
- 使用结构化结果对象替代单一计数器。

#### 3.2.2 番号提取失败时回退到整个文件名，容易放大无效搜索

文件：`docs/film-scanner/main.go`

`extractNumber()` 在正则无法匹配时，会把清理后的整个文件名直接当作番号返回。这样会导致 `SearchMovieAll()` 对所有 provider 发起搜索，带来额外请求、超时和限流风险。

风险：

- 误请求显著增加；
- provider 被动扇出查询；
- 对大目录扫描时整体耗时不可控。

建议：

- 提取失败时返回空字符串并标记为“待人工处理”；
- 不要把整个文件名作为宽松兜底策略；
- 可考虑输出失败原因日志，便于后续命名规范治理。

#### 3.2.3 日期序列化依赖字符串拼接与字符串包含判断，不稳定

文件：`docs/film-scanner/main.go`

`generateNFO()` 使用 `fmt.Sprintf("%s", movie.ReleaseDate)`，并用 `strings.Contains(releaseDate, "time.Location")` 判断日期是否合法。这种写法过度依赖类型的字符串表示形式，不是稳定的数据契约。

风险：

- NFO 中 `releasedate` 字段可能错误；
- 行为依赖上游类型实现细节；
- 后续升级依赖或结构调整时容易失效。

建议：

- 明确 `ReleaseDate` 的真实类型；
- 使用稳定格式化逻辑，例如 `time.Time.Format("2006-01-02")`；
- 若值无效，应显式跳过而不是字符串猜测。

#### 3.2.4 CLI 与 SDK server 的配置入口不一致

文件：

- `docs/film-scanner/main.go`
- `docs/film-scanner/metatube-sdk-go/cmd/cmd.go`

扫描 CLI 直接调用 `engine.Default()`，而 SDK server 使用 `ff.Parse`、`envconfig`、`request-timeout`、provider config 等一整套配置注入方式。这导致两套入口能力不一致。

风险：

- 文档配置与 CLI 行为不一致；
- 代理、provider 配置、超时配置无法统一；
- 测试、部署和调试成本上升。

建议：

- 为 CLI 与 server 统一引擎构建逻辑；
- 将 `timeout`、provider 配置、代理、数据库配置纳入同一套配置模型；
- 尽量避免一套代码库出现两套能力明显不同的启动路径。

#### 3.2.5 全 provider 并发搜索没有整体截止时间

文件：`docs/film-scanner/metatube-sdk-go/engine/movie.go`

`searchMovieAll()` 会为所有 provider 启动 goroutine 并发搜索，但没有统一的 deadline 或 cancel 机制。某些 provider 过慢时，整个搜索流程只能被动等待。

风险：

- 单次搜索耗时不可控；
- 批量扫描总时长放大；
- 难以和未来任务系统、用户取消、超时重试策略集成。

建议：

- 引入 `context.Context`；
- 为单 provider 和整体搜索分别定义超时；
- 支持提前取消和部分结果返回策略。

#### 3.2.6 HTTP 请求未显式绑定上下文

文件：`docs/film-scanner/metatube-sdk-go/common/fetch/fetch.go`

当前请求通过 `http.NewRequest(...)` 构造，没有显式绑定 `context`，取消能力主要依赖 `http.Client.Timeout`，协作式取消能力偏弱。

建议：

- 使用 `http.NewRequestWithContext(...)`；
- 将上下文从任务层传递到抓取层；
- 为未来的任务取消、应用退出、超时中断保留能力。

#### 3.2.7 HTTP 服务入口缺少基础超时保护

文件：`docs/film-scanner/metatube-sdk-go/cmd/server/main.go`

当前服务直接调用 `http.ListenAndServe(...)`，没有设置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` 等保护参数。

风险：

- 慢连接占用资源；
- 对外暴露时服务抗压能力不足；
- 服务行为不利于产品化。

建议：

- 使用 `http.Server` 显式配置超时；
- 至少补上 `ReadHeaderTimeout` 和 `IdleTimeout`。

### 3.3 低优先级问题

#### 3.3.1 CORS 默认策略偏宽松

文件：`docs/film-scanner/metatube-sdk-go/route/route.go`

`cors.Default()` 对开发环境很方便，但如果服务被当作正式 API 对外暴露，默认策略通常过宽。

建议：

- 在正式场景中收紧允许来源、方法和头部；
- 明确区分本地开发策略与生产策略。

#### 3.3.2 Go 版本要求偏新，可能影响可移植性

文件：`docs/film-scanner/go.mod`

当前声明 `go 1.25.5`。这本身不是错误，但会提高环境统一和 CI 维护成本。

建议：

- 明确团队与 CI 的 Go 版本基线；
- 若无必要，避免使用过于前沿而没有明确收益的版本要求。

#### 3.3.3 热路径中重复编译正则

文件：`docs/film-scanner/main.go`

`extractNumber()` 每次调用都会对每个模式执行 `regexp.MustCompile(...)`，在大目录扫描时会产生不必要的 CPU 开销。

建议：

- 将正则预编译为全局变量；
- 将提取逻辑与模式配置拆开。

## 4. 风险判断

从当前实现看，这套代码更适合作为：

- 本地实验工具；
- 搜刮能力验证原型；
- 后端领域知识与 provider 抽象的参考来源。

但它还不适合作为：

- 正式产品后端的稳定基础；
- 面向前端产品链路的统一服务层；
- 有明确任务系统、错误码、日志规范和配置规范的生产实现。

当前最大风险不是“功能不存在”，而是“代码已经足够复杂，看起来像后端产品，但缺少正式产品后端所需的稳定性边界”。

## 5. 改进建议

### 5.1 第一阶段：先修运行时风险

优先级最高，建议立刻处理：

1. 修复 `downloadPreviews()` 中的响应体关闭方式。
2. 移除 `MustParse()` 这类 panic 型路径。
3. 处理 `database.Open(...)` 的初始化错误。
4. 修正成功统计逻辑，避免误导性结果。

### 5.2 第二阶段：统一配置与超时模型

建议尽快推进：

1. 统一 CLI 与 server 的引擎初始化逻辑。
2. 将超时配置从入口显式传递到搜索和抓取层。
3. 引入 `context.Context`，支持整体取消与任务级超时。
4. 明确 provider 配置、代理配置、数据库配置的统一注入方式。

### 5.3 第三阶段：为正式后端演进做结构准备

如果后续要将其演进为正式后端，建议：

1. 不再把正式实现继续放在 `docs/` 下。
2. 新建独立 `backend/` 工程或独立仓库。
3. 先定义稳定的：
   - DTO
   - 错误码
   - 任务状态
   - 事件模型
   - 日志字段规范
4. 将“扫描结果”从打印信息改造成结构化任务结果。
5. 让 CLI、服务端、未来桌面桥接都共用同一套领域服务，而不是各自维护一套入口逻辑。

## 6. 建议的演进方向

若以后要服务当前前端原型，后端应优先围绕以下能力做产品化收敛：

1. `listMovies`
2. `getMovieDetail`
3. `toggleFavorite`
4. `scanLibrary`
5. `getTaskStatus`
6. `getSettings`
7. `updateSettings`

而不是先继续扩展 provider 细节或临时工具能力。后端首先需要的是稳定边界、统一契约和可靠错误模型，其次才是更多的抓取功能。

## 7. 结论

当前这套 Go 代码能作为“后端方向参考”和“领域实现素材”，但距离正式产品后端还有明显差距。最应该优先解决的是运行时稳定性和工程一致性，而不是继续追加新能力。

如果后续要把它接入当前前端原型，推荐路线是：

1. 先修高风险问题；
2. 再统一配置、超时和任务模型；
3. 最后抽离成独立正式后端工程，与前端通过稳定契约对接。

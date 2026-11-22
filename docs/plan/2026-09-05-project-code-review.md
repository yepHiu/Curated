# Curated 项目关键功能代码审查（2026-09-05）

## 代码质量复审（2026-11-22，已修复）

初始审查基线为 HEAD `04284d3d` 加当时工作区。用户随后要求修复下述五项问题，已按独立修改单元提交；下方问题描述、行号及“本轮验证”保留修复前现场，2026-09-05 的章节仍为历史记录。

### 修复状态与验证（2026-11-22）

| 问题 | 实现与验证 | 提交 |
| --- | --- | --- |
| 正文读取期间取消失效 | `monitoredRequest` 包含正文消费；错误页仅忽略 JSON 语法错误，保留取消/网络错误；覆盖 JSON、字幕、Blob、错误响应、无期限主动取消及完成后回收 | `cc600138` |
| 原生播放不识别 423 | 共用 `isPlaybackAuthStatus` 覆盖 API、媒体代理及字幕；字幕列表锁定异常不再吞掉；验证启动竞争、字幕列表/内容、真实 loopback 媒体代理、截图及片段路径 | `bbbd6d36` |
| 测试受开发环境影响 | 禁用 `.env` 加载并固定 Mock/空 API base；保留 `vi.stubEnv` 的 Web 分支覆盖；默认最多 4 workers；外部传入 Web/无效 API 地址的 5 文件 20 用例通过 | `ff1b45d8` |
| 字体配置 XML 非法 | 转义动态路径的 XML 文本，真实 XML 解析器验证含 `&`、尖括号、中文、引号的路径往返 | `b47b3f00` |
| 快捷键反馈图标缺失 | 补齐四个 lucide 图标显式导入；复用播放器 loading/i18n 的 22 项测试验证 | `ee43d64b` |

字体配置及其测试此前为未跟踪文件，本次作为独立字体配置单元纳入版本控制；其他原有字幕资产、选轨和桌面接入改动仍留在工作区。协调器文件采用仅暂存本次修复的补丁，未混入这些已有修改。

修复后验证：前端类型检查、仓库 ESLint、Electron `tsc --noEmit` 均通过；Electron 全套 169 项通过、5 项平台集成测试跳过。直接执行默认 `pnpm test`，无需覆盖环境变量或 worker 参数，336 文件、1847 用例全部通过（192.73 秒）。不涉及 Go 代码修改，未重复已在审查阶段通过的 Go 全套测试。Windows 上的 XML 解析与 Electron 模拟测试不替代 macOS 真机字体验收。

轻量 e2e 使用 `pnpm test:e2e --workers=1`：首次 8 通过、1 失败；锁定启动用例在一次页面启动中观察到两次 `/api/auth/status` 而非一次。仅复跑 `-g 'locked startup'` 后通过（22.3 秒），重复请求未复现，尚未确定首次失败根因；没有将首次全套描述为通过，也未据此扩大修改启动逻辑。日志保存在系统临时目录 `curated-fixes-e2e.log` / `curated-fixes-e2e-lock.log`。

原始复现探针仍保留用于审查溯源，其中断言的是修复前错误行为；修复后以已纳入正式测试集的回归用例为准。

### 原始审查范围与发现

覆盖当前 Electron 字幕改动、原生播放器与 Server 的鉴权调用链、前端 HTTP 客户端、播放器模板、字幕任务/存储约束，以及仓库既有质量检查。不是 976 个生产源码文件的逐行审计，也未运行真实 macOS 播放、生产打包、浏览器 e2e 或付费供应商调用。

### 结论

分层、输入校验、数据库约束和自动化测试基础较完整，但跨层错误合约、异步请求生命周期和测试环境隔离仍有实际缺口。本轮确认 4 项 P2、1 项 P3，其中字体 XML 问题位于当前未提交新增文件，其余是现有代码。没有在已审范围内确认 P0/P1。

### 1. [P2] 收到响应头后，HTTP 超时和外部取消失效

位置：`src/api/http-client.ts:150–152`，正文读取发生于 `156–157` 及各调用方。

`fetch()` 在响应头到达时返回，`monitoredFetch` 随即清除 timeout 并移除外部 AbortSignal 监听；`response.text()` / `response.blob()` 尚未完成。如果 Server 或网络在发送响应头后阻塞正文，页面加载可无限等待；切换影片或页面后取消请求，也无法中止这部分传输。

隔离复现调用真实 `httpClient.get`，让 fetch 返回响应头而正文保持 pending，再取消调用方 signal 并推进 31 秒：调用方 `aborted=true`，实际 fetch signal 仍 `aborted=false`，请求也未完成。已有测试只覆盖响应头到达前的超时与取消，未覆盖此阶段。

修复方向：让超时、取消转发及错误转换持续到正文读取完成；统一封装响应消费，避免每个调用方各自处理生命周期。

### 2. [P2] 原生播放未识别 Server 实际使用的 HTTP 423 锁定响应

位置：`electron/playback-coordinator.ts:143`，同类分支还有 `188`、`489`；媒体代理对应 `electron/native-media-proxy.ts` 的错误状态映射。

Server 的 `auth_middleware.go` 对锁定请求返回 `http.StatusLocked`（423），原生播放层仅识别 401/403。会话在 `/api/auth/status` 检查后被撤销时，后续详情/字幕/截图/媒体请求会走普通 HTTP 或媒体错误，而非 `SERVER_LOCKED` 收尾和解锁跳转。启动失败可因此误走 Web fallback；播放中的常规状态轮询有机会补救，但不能代替对已收到锁定响应的正确处理。

隔离复现让公开 status 返回 unlocked、随后受保护详情返回 423，`open()` 实际抛出 `SERVER_HTTP_423`。前端 `desktop-playback-service.ts` 只有识别 `SERVER_LOCKED` 才进入锁屏分支。现有 Electron 鉴权测试多使用 401/403，未覆盖真实 Server 合约。

修复方向：统一锁定状态/错误码判断，将 423 纳入 API、字幕内容和媒体代理路径，补充真实状态码的跨层测试。

### 3. [P2] 默认前端测试依赖开发者的 VITE_USE_WEB_API 配置

位置：`vitest.config.ts:16–19`；典型受影响用例为 `SettingsSecuritySection.test.ts`。

单元测试默认加载本机 Vite 环境，却没有固定服务模式。多个设置组件用例假定本机管理权限立即可用，而真实 `useServerLocalAccess` 在 Web 模式会请求 health，再决定权限；相关用例没有 mock 这条边界。开发机常用的 `VITE_USE_WEB_API=true` 因此会隐藏测试预期的控件，制造与产品逻辑无关的失败，并允许单元测试发出真实 health 请求。

单独控制变量复现：保留相同 Node 参数及 `--maxWorkers=2`，显式 Web 模式下 `SettingsSecuritySection` 6 失败/1 通过；Mock 模式下该文件全部通过。首次全量默认运行为 27 失败/1810 通过，其中还包含播放器超时；不能将全部 27 项都归因为环境变量。固定 Mock、使用 CI 的 `NODE_OPTIONS=--no-experimental-webstorage` 并限制 2 workers 后，6 个失败文件共 53 项全部通过。

修复方向：测试配置显式固定默认服务模式，需要 Web 行为的测试单独声明环境并 mock health/桌面权限边界；不要依赖开发者删除本机 `.env`。并发导致的超时应与确定性失败分开诊断。

### 4. [P2] macOS 字体配置直接插入未转义路径

位置：当前新增 `electron/mac-subtitle-fonts.ts:48–53`。

`cacheDirectory` 来自 userData 路径，合法路径中的 `&`、`<` 等字符被直接写入 XML 文本。例如 `/Users/A&B/Library/Application Support/Curated/fontconfig` 会生成无法解析的 `fonts.conf`。随后该文件仍被设置为 `FONTCONFIG_FILE`，会破坏新加入的字体初始化路径。

隔离验证使用真实 `buildMacFontConfigBody` 生成上述配置，标准 XML 解析器返回 parsererror。现有测试仅检查字符串包含字体/目录，没有验证 XML 合法性。没有在 Windows 上声称完成 macOS libmpv 字体验收。

修复方向：用 XML 文本转义或 XML writer 输出动态路径，并覆盖特殊字符路径。

### 5. [P3] Web 播放器快捷键反馈的四个图标未导入

位置：`src/components/jav-library/PlayerPage.vue:3135–3160`，关联顶部 lucide import。

模板使用 `Rewind`、`FastForward`、`VolumeX`、`Volume2`，但 `<script setup>` 未导入这些组件，应用也没有全局注册或自动导入插件。隐藏控制栏后快进/后退、静音或调音量的反馈图标不会正常渲染。实际组件测试输出四个 `Failed to resolve component` 警告；当前 typecheck/lint 均未拦住这类模板错误。

修复方向：补齐显式导入，并让相关行为测试验证图标解析或针对未解析组件告警失败。

### 可维护性观察与处理顺序

按 `src/`、`electron/`、`backend/internal/` 下 `.ts/.vue/.go` 文件统计，排除 `.test.*` 和 `_test.go`，共有 976 个源码文件，其中 14 个超过 1000 行。`server.go` 4128 行、`app.go` 3932 行、`PlayerPage.vue` 3597 行、`SettingsPage.vue` 2365 行。行数本身不构成缺陷，但播放器已同时承载网络、媒体状态、快捷键、字幕、截图及展示，审查与定位成本较高。

先修请求生命周期和锁定合约，再稳定测试环境、补齐字体与图标边界。模块拆分应结合后续功能修改，按独立职责逐步进行，不建议仅为缩短文件进行全仓重写。每个缺陷应有独立提交和对应行为验证。

排除项：曾怀疑字幕检查点续试允许同文件多活动任务；使用临时数据库复现时，`subtitle_active_file` 唯一索引正确拒绝第二个活动任务，因此未将其列为缺陷。临时 Go 探针已移除。

### 本轮验证

| 检查 | 结果 |
| --- | --- |
| `pnpm typecheck`、`pnpm lint` | 通过 |
| `pnpm exec tsc -p electron/tsconfig.json --noEmit` | 通过 |
| `pnpm test:electron` | 22 文件/156 用例通过；3 文件/5 用例跳过 |
| `backend/` 的 `go test ./...`、`go vet ./...` | 通过；部分 Go 包使用缓存 |
| 默认 `pnpm test` | 329 文件通过、6 失败；1810 用例通过、27 失败 |
| 固定 Mock、CI Node 参数、2 workers，复跑上述 6 文件 | 53 用例全部通过 |
| 同环境全量前端复跑 | 335 文件、1837 用例全部通过，耗时 338.54 秒 |
| 3 个隔离行为探针 | HTTP 正文取消失效、423 错误分类、非法字体 XML 均复现 |

复现文件保留于被 Git 忽略的 `.workspace/review-quality.config.ts`、`.workspace/review-quality.test.ts`；这些用例断言缺陷实际表现，因此“通过”表示复现成功，不表示业务正确。命令：`pnpm exec vitest run --config .workspace/review-quality.config.ts --configLoader native --pool threads --reporter verbose`。

前端可重复执行环境（PowerShell，仅影响当前 shell）：

```powershell
$env:VITE_USE_WEB_API = 'false'
$env:NODE_OPTIONS = '--no-experimental-webstorage'
pnpm test --maxWorkers=2
```

---

## 修复状态（2026-09-05）

用户随后要求修复全部问题，以下四项已实现并分别提交。后续各节保留初始审查现场与复现记录，其中行号对应修复前基线。

| 问题 | 修复结果 | 提交 |
| --- | --- | --- |
| 安装器被请求取消终止 | 启动前检查取消状态，启动成功后使用独立进程生命周期，后台 Wait 回收；真实测试子进程验证取消后仍完成 | `454d12c9` |
| 本地代理被源站防护拦截 | 在请求层校验源站 allowlist 和公网 IP；对直连、HTTP CONNECT、SOCKS 连接固定已校验的目标 IP，保留原始 Host 与 TLS ServerName；重定向重新校验 | `9978d100` |
| 会话切换后旧流污染当前状态 | 切换/新建/删除时废弃旧流；每次流使用自身 controller 和序号；历史与新建请求使用加载序号，加载期间暂停发送 | `8e527c44` |
| 逐帧步长受显示 fps 影响 | 源帧时长独立维护，优先 HLS manifest 帧率，缺失时根据媒体时间与帧计数估算；暂停/seek 清除采样边界，片源切换清空旧估计 | `ec444955` |

逐帧提交包含审查开始时已有的 D/F 功能与对应文案、保留键设置，在其基础上修正帧率来源。未推送远程。

正式回归测试位于：

- `backend/internal/appupdate/installer_process_test.go`
- `backend/internal/app/agent_source_page_transport_test.go`
- `src/components/agent-window/AgentWindow.session-races.test.ts`
- `src/lib/player-frame-step.test.ts`
- `src/components/jav-library/PlayerPage.loading.test.ts`

实现边界：没有可用帧率元数据或采样时仍按 30fps 回退；浏览器 seek 不提供所有视频格式上的精确源帧导航保证。代理目标使用本机解析后的公网 IP，只有远端代理能够解析的域名会明确失败，不会为此退回未经校验的远端 DNS。

### 修复后验证

| 检查 | 结果 |
| --- | --- |
| `pnpm typecheck` / `pnpm lint` | 通过 |
| `pnpm test --maxWorkers=2` | **211 个文件、967 个用例全部通过** |
| `pnpm test:electron` | 4 个文件、30 个用例通过 |
| `backend/` 下 `go test ./...` / `go vet ./...` | 通过 |
| `pnpm test:e2e` | 2 通过、3 失败；两个入口超时用例单 worker 复跑后 Mock 导航通过，Personal Insights 页面等待仍超时 |
| 移动端 e2e | 等待已不存在于 src 的 `[data-mobile-theme-toggle]` 标记而失败；该用例与对应页面非本轮修改 |
| `pnpm build` | 类型检查与模块转换完成，包体积硬门禁失败：总 JS raw **2365.48 kB > 2250 kB**，gzip **777.31 kB > 750 kB** |

修复前 `26441dce` 的隔离 Vite 构建复现同一包体积门禁失败（日志保留于 `.workspace/review-2026-09-05/baseline-build.log`），因此这不是本轮新引入的门禁失败。未调整预算，也未将失败表述为构建通过。Personal Insights e2e 的页面加载超时原因尚未确定，不能将整个浏览器套件标记为通过。

没有运行真实安装器、display scaling 套件或真实视频解码验证。本轮仅修复四项已确认问题；额外检查揭示的既有门禁与浏览器测试问题单独记录。

## 范围与结论

- 审查基线：HEAD `26441dce`，以及审查开始时已有的播放器 D/F 逐帧控制未提交修改。
- 用户指定范围：整个项目的关键功能与风险。
- 采用按风险抽查与调用链追踪：播放器定位/帧率、PIN 与会话鉴权、Host/Origin 防护、Electron 导航与进程管理、更新安装、上传提交、文件整理、备份校验/恢复、Agent 工具确认/源站读取/前端会话切换。
- 确认 4 个功能缺陷：1 个 P1、3 个 P2。每项都有隔离复现证据。只有逐帧问题位于本次未提交变更中，其余属于现有代码。
- 初始审查阶段没有修复业务代码或提交用户已有修改；随后修复状态见文档开头。该审查并非全仓逐行审计；未列出问题的模块不等于已证明不存在缺陷。

## 1. [P1] 更新安装进程随 HTTP 请求结束被终止

位置：`backend/internal/appupdate/service.go:41–43`。

`handleInstallAppUpdate` 将 `r.Context()` 经 `App.InstallAppUpdate` 传入 `Service.Install`，最后使用 `exec.CommandContext(ctx, ...)` 启动安装器。`cmd.Start()` 后直接返回；HTTP handler 完成会取消请求 context，Go 随之终止仍在运行的安装器。状态已被写成 `install-launched`，用户却可能看到安装窗口闪退，或静默安装没有完成。

隔离验证使用测试二进制作为无副作用的安装器替身：先写 ready 文件，延迟后写 done 文件。确认 ready 已存在后取消 context，done 不再生成，复现了“已启动但随请求取消而终止”。没有启动真实安装器。

修复方向：让成功启动后的安装器生命周期独立于单个 HTTP 请求，并安排子进程退出回收。回归测试必须覆盖 handler 返回/请求取消后的存活，不能仅 mock 启动函数并断言参数。

## 2. [P2] Agent 源站读取将用户配置的本地代理作为私网目标拦截

位置：`backend/internal/app/agent_source_page.go:64–68`。

`newAIHTTPClient` 保留配置的 HTTP 代理，`attachSourcePageGuards` 随后替换 Transport 的 `DialContext`。HTTPS 经 HTTP 代理时，底层拨号地址是代理地址，而非源站地址。因此使用 `http://127.0.0.1:<port>` 等常见本地代理时，`SourceIPBlocked` 在发出 CONNECT 前就拒绝连接，合法且已允许的公网源站也无法读取。

隔离验证用 `httptest.NewServer` 充当本地代理，并请求允许列表中的 `https://example.com/source`。结果是代理收到 0 次 CONNECT，请求返回 `proxyconnect tcp: url host is not allowed`；测试不访问实际公网。

修复方向：分清显式配置的代理端点与不可信源站目标，在允许访问配置代理的同时保留源站与重定向防护。补充直连、本地 HTTP/SOCKS 代理及私网目标拒绝测试，不能直接删除私网过滤。

## 3. [P2] Agent 切换会话后旧响应会改回当前会话 ID

位置：`src/components/agent-window/AgentWindow.vue:211–215`；关联回调 `458–460`。

在会话 A 正在生成时切换到 B，`selectSession` 只加载 B，不中止 A 的流，也不让 `streamSeq` 失效。后端的各类 SSE 事件携带 A 的 `sessionId`，Web 适配器据此调用 `onSession`，于是下一条 A 的事件会将当前会话 ID 从 B 改回 A，而页面已经显示 B 的历史。旧的确认卡片和 outcome 也可能进入当前列表；之后发送消息可能使用错误的会话 ID。

隔离组件测试保持 A 的流未结束，点击 B 并确认 activeId 为 B，再投递 A 的 onSession 事件。期望仍为 B，实际得到 A。测试使用真实 AgentWindow 与侧栏，替换 composer/thread 展示组件以隔离无关渲染。

修复方向：切换会话时中止并废弃旧请求；所有事件与结束清理都绑定请求自己的 controller/代次。为 `loadSession` 增加请求代次判断，避免连续切换时较早的详情请求覆盖当前会话。

## 4. [P2] 逐帧功能将显示帧率当成源视频帧率

位置：`src/components/jav-library/PlayerPage.vue:1721–1727`；关联帧率采样 `2363–2383`。

新加入的 `stepFrame` 使用 `1 / playbackStats.fps` 作为媒体时间步长。但 fps 来自显示帧回调次数除以实际经过时间，会受倍速、掉帧和暂停时间影响，并非稳定的源视频帧率。30fps 视频以 2 倍速显示约 60fps 时，单次 F 只移动约 1/60 秒，可能留在同一源帧；低倍速或暂停后的采样也可能导致跳过多帧。

隔离组件测试模拟 30fps 视频在 2 倍速下的帧回调，从 10 秒按 F：期望 `10.033333333333333`，实际 `10.01639344262295`。现有新增用例只覆盖 fps 未知时的默认 30fps，没有覆盖实际采样路径。

修复方向：源帧时长与性能面板显示 fps 分开维护。优先使用可信的流帧率，或根据媒体时间戳增量推导；暂停/seek 回调不应污染源帧率估算。覆盖倍速、暂停后逐帧和缺失帧率等情况。

## 初始审查验证记录

| 检查 | 结果 |
| --- | --- |
| 根目录 `pnpm typecheck` | 通过 |
| 根目录 `pnpm lint` | 通过 |
| 根目录 `pnpm test:electron` | 4 个文件、30 个测试通过 |
| `backend/` 下 `go test ./...` | 通过；部分包使用 Go 测试缓存 |
| `backend/` 下 `go vet ./...` | 通过 |
| 根目录 `pnpm test` | 210 个文件中 202 个通过，8 个失败；957 个测试中 948 个通过，9 个失败 |
| 相关前端用例降低并发复跑 | 10 个文件、46 个测试全部通过 |
| 四项审查专用复现测试 | 四项缺陷均复现，针对期望正确行为的断言失败 |
| `git diff --check` | 通过（只有已有 LF/CRLF 提示） |

全量前端首次运行的 9 个失败中，8 个是 5 秒超时，另 1 个是同文件前序超时后的调用次数断言。失败文件及逐帧相关测试以 `--maxWorkers=2` 复跑全部通过，因此没有把这些初次失败列为产品缺陷，也没有将首次全量运行描述为通过。

本次没有运行生产构建、浏览器 e2e、display scaling、真实安装或真实媒体转码。播放器问题通过组件事件模拟确认，未宣称已做真实视频逐帧解码验证。

### 复跑命令

以下本地复现材料位于 Git 忽略的 `.workspace/review-2026-09-05/`，Go 使用 overlay 注入测试，不修改业务包文件。

```powershell
# 仓库根目录：Agent 与播放器问题复现
pnpm exec vitest run --config .workspace/review-2026-09-05/vitest.review.config.ts --configLoader native --pool threads

# backend/：源站代理与安装器生命周期问题复现
go test -overlay ../.workspace/review-2026-09-05/go-overlay.json ./internal/app ./internal/appupdate -run '^TestReview' -count=1 -v

# 仓库根目录：首次失败文件及逐帧相关现有用例
pnpm test --maxWorkers=2 src/router/auth-lock.test.ts src/components/notification-center/NotificationCenter.test.ts src/components/jav-library/ActorDetailPage.test.ts src/components/jav-library/PlayerPage.frame-markers.test.ts src/components/jav-library/PlayerPage.loading.test.ts src/components/jav-library/PlayerPage.progress-hover.test.ts src/components/jav-library/PreviewImageViewerInner.test.ts src/components/jav-library/ScanProgressDock.test.ts src/lib/player-frame-step.test.ts src/lib/player-shortcuts.test.ts
```

## 初始处理顺序（现已完成）

优先修复安装器生命周期，再处理 Agent 会话串线与代理连接问题；逐帧问题应在当前未提交功能合入前处理。每项按独立缺陷形成最小修改单元，并把本地复现转成正式回归测试。

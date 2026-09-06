# Curated 项目关键功能代码审查（2026-09-05）

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

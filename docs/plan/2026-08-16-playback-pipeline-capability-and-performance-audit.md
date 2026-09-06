# 播放链路与播放器能力 / 性能审计（2026-08-16）

## 2026-09-06 修复落地与验收

本轮按 9 月 5 日复审的 F1–F10 缺陷完成修复。下方复审描述保留为问题证据，不再表示当前仍未修复。真实影片画质、LAN/GPU 基准和“进一步优化”表中的产品扩展未纳入缺陷修复范围。

| 问题 | 已落地行为 | 主要验证 |
| --- | --- | --- |
| F1 实际转码节流 | Windows NtSuspendProcess/NtResumeProcess，支持的 Unix 使用 SIGSTOP/SIGCONT；控制本次 FFmpeg 子进程，Scoop shim 解析到真实可执行文件；硬编/remux 2.5×、软编 1.5× readrate 回退；消费位置用真实 EXTINF；启动失败等待进程退出并限制 stderr 尾部内存 | 本机 FFmpeg 8.0.1 合成进程暂停时输出保持 0.50s，恢复后增至 0.80s；长 GOP 清单时间计算测试 |
| F2 会话隔离与交接 | 不按 movieId 抢占；新流 loadeddata 前保留旧会话，解码失败恢复旧流，清理集合覆盖连续切换 | 真实 FFmpeg 同片双会话同时可读，删除一个不影响另一个；组件换流失败回退测试 |
| F3 显式 direct | 后端直接构造原文件 descriptor，不启动 forced HLS；能力判断遵循探测结果；前端拒绝响应模式不一致 | 临时 SQLite + ForceStreamPush=true 的 direct 集成测试 |
| F4 HLS 有界恢复 | 网络重连、媒体恢复各最多一次，最多再重建一次；恢复耗尽显示错误，点击播放重新尝试；诊断识别 expired/failed；仅确认源可直放时才允许初始化 fallback | 恢复预算/codec 判断测试，组件 fatal network 不退回 MKV 源 |
| F5 seek 追赶 | 按 encoderSpeed 与最多 4 秒预算判断，60/30 秒仅作距离上限；后发 seek 取消前一个请求，迟到 session 被释放 | 1×/20× 同目标决策；连续快捷键最新意图测试 |
| F6 重复预热 | 删除媒体 fetch 预热和假进度，仅保留 HLS 模块预加载；原 no-store 策略不变 | HLS 模块/播放器测试；移除唯一媒体预热实现和调用 |
| F7 起播缓冲 | 使用包含 currentTime 的连续 buffered 区间，按倍速换算；保留最多 4 秒等待 | duration=7200 但仅少量 buffer 时不会假就绪，区间空洞/2× 测试 |
| F8 首次起点 | GET 新增可选 startPositionSec，起点优先级与 near-end 规则在启动 FFmpeg 前处理；预取 key 含起点与 codecs | HTTP 参数验证、0/1200/near-end 测试、预取不同起点隔离 |
| F9 启动并发 | 默认同时准备 2 个、保留最多 8 个，含旧流；启动队列/探测/profile 共用 24 秒可取消期限；Close 中止排队并等待启动退出 | 取消排队、关闭、失败释放容量测试 |
| F10 退出回收 | AbortSignal 贯穿 HTTP/服务/播放器；卸载使 load/attach generation 失效；预取响应完成后 TTL=10 秒，lease 可释放，消费后转移取消权 | 卸载后迟到响应释放且不 reseek、慢预取不双启动、消费与取消测试 |

提交按最小行为单元拆分：`89f30e19` 会话与队列、`36845da4` 起点与请求生命周期、`0497b72d` 删除预热、`9634d964` seek/缓冲、`e05ba547` 实际进程节流、`ceb6bfaa` 有界恢复与回退、`8dc5de10` 真实会话和连续 seek 补充测试。

验收记录：

- `pnpm typecheck`、`pnpm lint` 通过。
- `pnpm test --maxWorkers=2`：215 文件 / 990 项通过；随后新增连续 seek 用例，单独运行 PlayerPage.loading 的 16 项全部通过。
- `cd backend; go test ./...` 全量通过；随后增加真实双会话测试，`go test ./internal/playback -run TestFFmpeg -v -count=1` 两项通过（暂停/恢复约 1.53s，同片独立 HLS 约 7.88s）。
- `go vet ./internal/playback/... ./internal/app/... ./internal/server/...` 通过。
- `pnpm build` 类型检查与打包转换完成，但现有 total JS 门禁失败：当前 2374.43 kB raw / 780.54 kB gzip，限额 2250 / 750。使用隔离 worktree 检查修复前 `867e7419`，同步相同本地构建 env 后同样失败：2371.88 / 779.66。未提高预算或跳过门禁；本轮约增加 2.55 kB raw / 0.88 kB gzip，不是既有约 122 kB 超限的来源。
- 浏览器 smoke 未全绿：标准双 worker 首次 2 通过 / 3 失败；最后独占串行 `pnpm test:e2e --workers=1` 为 3 通过 / 2 失败。修复前 `867e7419` 同环境串行为 4 通过 / 1 失败：375px 用例所查 `data-mobile-theme-toggle` 在前后版本的源码中均不存在，确认是既有失败。当前另一个失败为个人洞察首次加载超时；trace 显示 `/src/main.ts` 请求耗时约 26.8 秒，未进入应用页面。清理基线 worktree 后默认 30 秒仍超时；仅以 CLI `--timeout=90000` 做诊断，个人洞察单项通过（用例 29.6 秒，总运行 33.1 秒），支持冷启动耗时接近超时门槛的判断，但不将其算作默认门禁通过，亦未改测试超时。日志位于本地忽略目录 `.workspace/playback-e2e-*.log`。播放链路的组件事件测试与真实 FFmpeg 测试已通过，不将其冒充完整浏览器媒体播放验收。
- `go test -race ./internal/playback/...` 受本机工具链限制：默认 CGO disabled，临时设置 CGO_ENABLED=1 后 runtime/cgo 编译器仍 exit status 2，未获得 race 通过结果。

运行边界：标准发行 FFmpeg 需作为直接可执行文件启动；已适配 Scoop shim，任意自定义多层 wrapper 不保证可暂停其真正编码子进程。其他未支持平台保留 readrate 回退。event HLS 历史分片仍保留到会话清理，未实现字节配额、按需 VOD、音频单独转码、多码率或 storyboard；这些是后续独立优化，不宣称本轮已完成。

## 2026-09-05 复审：当前结论

审查基线：HEAD `867e7419`。本节优先于下方 8 月 16 日历史审计；历史部分的 P0 标记、TS 封装、无探测缓存、续播不能 remux 等描述不能再作为当前实现事实。本轮只更新审查文档，未修改业务代码。

审查范围：PlayerView → Web service / descriptor → app 播放决策 → FFmpeg 会话管理 → HLS HTTP 响应 → hls.js / video → seek、模式切换、退出与续播。结论来自源码调用链、已有单测和一次合成媒体实验，未进行真实影片、LAN 带宽或 GPU 性能基准测试。

### 已有优化及主要判断

当前已有媒体探测内存与 SQLite 缓存、浏览器视频 codec 协商、单片详情与 descriptor 并行加载、fMP4 / temp_file、关键帧 seek、硬编能力探测与成功档位记忆、直连媒体 URL、Range / ETag、部分窗口内 seek 复用。应保留这些能力。

最大的进一步收益在会话生命周期、无效预热、seek 等待策略与故障恢复。当前不需要先替换播放器框架；应先修复下面可定位的问题，再用真实媒体测量决定是否投入按需分段 VOD。

### F1 · P1：节流写入成功不代表 FFmpeg 已暂停

- 位置：`backend/internal/playback/manager.go:1062-1082`；`backend/internal/playback/throttle.go:8-37`。
- 当编码领先请求位置超过 90 秒时，`maybeThrottle` 向 stdin 写 `p`；写入成功即设 `throttlePaused=true`，没有确认运行时支持或检查进度是否停止。硬编 profile 不带 readrate，event playlist 保留全部分片。
- 实验：本机 PATH FFmpeg `8.0.1-full_build-www.gyan.dev`，64×64 / 10fps lavfi 合成源，`-re -progress pipe:1 -f null -`。发送 `p` 前输出时间 1.5s，发送后等待 2s，输出时间为 3.5s；再发送 `u`、等待 1s，输出时间为 4.6s。进程正常退出，stderr 为空。此运行时的 `p` 没有停止媒体处理。
- 影响：暂停观看后不能依赖这套逻辑降低编码开销；硬编可能继续生成大量未观看分片，长 event 会话也没有磁盘配额。不能把现有阈值单测通过当作实际反压已生效。
- 建议：给运行时控制增加能力验证与实际暂停确认；不支持时走明确的有界生成策略或经过验证的节流回退。readrate 只能限制速度，不能承诺暂停后停止处理。增加跨平台合成进程集成测试，验证暂停期间 out_time / 分片数稳定、恢复后继续增长。

### F2 · P1：以影片 ID 抢占会话，多客户端互相打断且失败无法回退

- 位置：`backend/internal/playback/manager.go:222-224,957-970,986-1018`。
- 每次启动都先 `takeSessionsForMovie(movieID)`，停止进程并删除目录，然后才探测关键帧、尝试新 profile。会话没有客户端或播放实例归属。
- 触发：客户端 A 正在观看某 HLS 影片，客户端 B 打开同片；或者 A 自己进行需要重建会话的 seek。旧流立即失效。若新编码器启动失败，旧目录已删除。
- 前端 `PlayerPage.vue:2555` 虽然计划在新流就绪后清理旧 session，后端已经提前清理，无法实现平滑切换。
- 建议：用播放实例 / owner 标识管理会话，显式传入被替换的 sessionId；仅替换同一播放实例。采用准备 → 验证可播 → 交接 → 释放流程，并用有界并发控制短暂资源重叠。回归覆盖两客户端同片播放与新流启动失败保留旧流。

### F3 · P1：显式切换 direct 仍会触发自动 HLS 决策

- 位置：`backend/internal/app/app.go:2441-2444`；`src/components/jav-library/PlayerPage.vue:1858-1869,2099-2105`。
- 后端收到 `mode=direct` 后调用 `ResolvePlayback`，没有强制构造原文件 descriptor。开启 ForceStreamPush、源时间轴异常或格式需要 HLS 时，仍会启动 HLS 并返回 `.m3u8`。
- 前端在用户选择 direct 后，把返回值的 `mode` 改成 `direct`，保留 HLS URL；Chromium 接下来通过 `v.src` 加载该 URL，不创建 hls.js 实例。还可能先删除旧会话并白启动一个新的 FFmpeg。
- 建议：后端明确区分自动规划与显式模式请求；显式 direct 返回真实 `/stream` 或明确不可用错误。前端校验响应模式及能力，不以修改 DTO 标签伪造模式切换成功。验收应覆盖 ForceStreamPush=true 的 MP4 手动切 direct。

### F4 · P2：fatal HLS 错误无条件回退到可能无法直放的源

- 位置：`src/components/jav-library/PlayerPage.vue:2334-2341,1086-1124,1606-1623`。
- 所有 fatal error 都走 `fallbackHlsToDirect`，不区分网络、媒体解码和会话过期。回退能力只按扩展名判断，并硬填 `video/mp4` MIME，忽略原有 codec 决策。
- 触发：MKV 或浏览器不支持的 HEVC 依赖 HLS 播放，网络重试耗尽后发生 fatal error。回到源文件不会修复问题；对不支持的 MP4 codec，也可能因扩展名被误判为能播。
- 建议：在 hls.js 自带重试耗尽后执行有界的分类恢复：网络错误重连、媒体错误尝试 recoverMediaError、失效会话按当前位置重建；只有能力校验通过才能回 direct。设置次数上限并保留暂停/续播意图，失败给出可执行的重试入口。

### F5 · P2：固定 60 秒 seek 复用窗口可能先白等 20 秒再重建

- 位置：`src/lib/player-hls-seek.ts:1-6,47-61,75-121`；`src/components/jav-library/PlayerPage.vue:2490-2542`。
- transcode 目标位于已写边界之后 60 秒内，一律先等窗口追上，超时为 20 秒；等待失败才 POST 新会话。
- 例：已写至 20s，目标为 70s，编码速度约 1×。规则允许复用，但 20s 不足以补齐 50s，用户先等待约 20s，然后还要承担新会话启动成本。这里是根据阈值推导的场景，不是测得的真实 seek 延迟。
- 建议：使用已经存在的 encoderSpeed / writtenDurationSec 估算追赶时间，与近期 session 启动成本比较；临近边界短等，明显追不上直接换会话。连续快捷键和拖动采用最后意图优先，取消过期等待及尚未交接的启动。

### F6 · P2：HLS 预热绕不开 no-store，产生重复媒体下载

- 位置：`backend/internal/server/server.go:1106-1109`；`src/lib/hls-player.ts:239-283`；`src/components/jav-library/PlayerPage.vue:879-898,1055-1057`。
- m3u8、init.mp4、m4s 均使用 `no-store, no-cache, must-revalidate`。预热 fetch 将媒体读到 arrayBuffer 后丢弃，hls.js 另行加载同一资源；两条请求也没有共享 loader 或内存缓存。
- 默认 resourceCount=2，fMP4 清单通常意味着额外读取 init.mp4 和首个 m4s，并额外 GET 一次清单。预热可能温暖磁盘缓存，但 HTTP 缓存不能复用这些响应。
- 此外 `prewarmHlsDescriptor` 没传 AbortSignal；helper 的超时在 fetch 返回响应头后已被清理，body 的 text / arrayBuffer 读取不受该超时保护。慢响应体可能继续占连接和内存。
- 建议：优先删除重复媒体预热，保留 HLS 模块预加载；若保留，采用同 loader 的共享数据，或对唯一 session URL 下已完成的分片使用经认证的 private 缓存策略。清单继续保持新鲜。取消与超时应覆盖完整响应体消费。
- 验收：抓取启动与 seek 网络瀑布，核对 init / 首片传输次数、重复字节和首帧时间，不能只验证预热进度条完成。

### F7 · P2：8 秒起播缓冲检查把媒体 duration 当作实际缓冲

- 位置：`src/components/jav-library/PlayerPage.vue:944-950`；`src/lib/player-hls-seek.ts:23-36`。
- 起播调用 `waitForMediaWrittenEnd(v, 8)`，该函数检查 `max(video.duration, bufferedEnd)`。duration 表示媒体时间轴长度，不代表浏览器已经下载并可连续播放的数据；而 bufferedEnd 也没有减去当前播放位置。
- 触发：清单声明 12 秒、浏览器仅缓冲首片，duration 有限时仍立即通过；从会话内非零位置续播也不能证明当前帧后有 8 秒缓冲。
- 建议：分开“服务端已生成范围”“客户端可 seek 范围”“当前位置之后连续 buffered 秒数”。起播使用包含 currentTime 的 buffered range 的 end-currentTime，再按播放倍速换算可播放墙钟时间。保留有界等待，避免为阈值无限阻塞。

### F8 · P2：显式起点未进入首次请求，可能重复启动 HLS

- 位置：`src/views/PlayerView.vue:34`；`backend/internal/app/app.go:2402-2415`；`src/components/jav-library/PlayerPage.vue:1020-1046`。
- 首次 prefetch 只带 movieId 与 codecs，服务端按数据库进度启动 HLS。拿到 descriptor 后，前端才读取 `?t=`，不一致时再次启动。
- 触发：数据库进度在 600s，从萃取帧打开 `?t=1200`；会先准备 600s 的流，再启动 1200s 的流。普通续播位置一致时不会重复，不能泛称所有续播都双启动。
- 建议：首次请求就携带显式起点并在服务端统一处理 near-end 重播规则；或者拆分纯 descriptor 规划与唯一一次显式 start。prefetch key 应包括有效起点与能力参数。

### F9 · P2：全局启动锁覆盖慢探测和全部 profile readiness

- 位置：`backend/internal/playback/manager.go:197-198,246-277,622-649`；`src/api/http-client.ts:64,111-136`。
- 一个会话持有全局 sessionStartMu，直至关键帧探测、profile 尝试和分片就绪完成。转码首片后还可等待第四片最多 12 秒。其他影片的启动也排队，等待 mutex 本身不能被 ctx 取消。
- 前端普通 HTTP 超时为 30 秒。慢输入或失败 profile 会消耗其他客户端的启动预算；锁串行也不等于活跃转码数量受限，成功的不同影片会话可持续累积。
- 建议：短锁只保护注册表，启动使用有界且可取消的队列；按编码资源限制并发而不是全局串行启动。总 deadline 覆盖整条尝试链。若引入异步 starting 状态，明确 session 归属、取消与错误合约。

### F10 · P2：退出页面时在途 descriptor 可能生成无人回收的会话

- 位置：`src/components/jav-library/PlayerPage.vue:1007-1057,1202-1212`；`src/services/adapters/web/web-library-service.ts:1402-1437`。
- loadPlayback 通过 movieId 和 playbackLoadSeq 检查过期结果，但卸载仅增加 hlsWindowWaitGeneration，没有增加 playbackLoadSeq 或设置 disposed，也无法 abort descriptor 请求。
- 触发：HLS 正在准备时离开页面，卸载时 descriptor 仍为 null，因此没有 sessionId 可删。响应稍后到达，闭包中的 movieId / seq 仍匹配，继续接受结果与预热；该会话只能等待 janitor 等后续清理。
- prefetch 也没有路由离开回收；10 秒 TTL 从请求创建时计时，过期只在后续消费时丢弃，未完成请求本身不会取消，可能再启动第二条会话。
- 建议：页面级 AbortController + disposed/generation 检查，在首次 await 返回后、任何 reseek 前检查失效。即使取消与服务端完成竞争，也应释放迟到的 session；prefetch 使用 lease / consume / cancel 生命周期。

### 进一步的性能与播放体验空间（不是本轮已复现故障）

| 方向 | 代码依据与建议 | 验收重点 |
| --- | --- | --- |
| 只转音频、复制视频 | `playback_decision.go:canRemuxToHLS` 与 `manager.go:676-692` 当前只有音视频全 copy 或视频重编码两类；增加 h264 copy + AAC 音频转换，避免只是 DTS/FLAC 等音频不兼容就重编视频。需排除负 PTS 等确实需要视频修复的源。 | 同源视频质量、CPU/GPU、首帧与中段 seek、音画同步；协商 AC3/EAC3 支持。 |
| 弱网/高分辨率档位 | 当前单 rendition，Windows encoder profile 以质量参数为主，没有统一分辨率/码率上限。先提供明确的原画/兼容/省流策略，再评估 ABR 的额外编码成本。 | LAN 带宽低于源码率时的卡顿比、实际码率和切换连续性。 |
| 可恢复的暂停与过期 | idle janitor 以资源访问时间为依据，前端诊断虽能获得 session state，页面只提取 speed / written duration / seek kind。恢复播放时检查 expired / failed，按当前位置有界重建。 | 暂停超过 idle 阈值、锁屏、睡眠恢复后不退到不可播放源。 |
| 按需分段和缓存配额 | event HLS 保留已生成历史；长片反复跨窗口 seek 仍重建进程与目录。先做缓存字节/会话数预算，再单独推进已有 8 月 23 日按需 VOD 计划。 | 缓存峰值、重复 seek 命中率、首帧/seek P95；不要直接删旧分片破坏后退。 |
| 续播可靠性 | `playback-progress-storage.ts:saveProgress` 是独立 fire-and-forget PUT，退出使用普通 fetch；增加同片写入串行合并/版本约束、离线重试和关闭时完整交付机制。 | 请求乱序不回退进度、页面关闭不丢最后位置；不靠提高写入频率补救。 |
| 浏览定位体验 | 目前进度 hover 是时间/标记，没有视频 storyboard；按需生成低分辨率缩略图，结合已有萃取帧辅助定位。 | 控制缩略图生成 IO，不抢当前解码资源；键盘/触控均能定位。 |

### 建议实施顺序与测量

1. **先修行为正确性**：F2 会话归属与交接、F3 显式模式语义、F4 有界错误恢复、F10 取消与回收。
2. **再去掉启动/seek 浪费**：F6 重复预热、F8 一次有效起点启动、F5 动态 seek 决策、F7 连续缓冲检查。
3. **治理持续资源开销**：F1 真正有效的节流及能力确认、F9 并发与 deadline，随后视频 copy + 音频转码及缓存预算。
4. **最后决定较大架构投入**：用同一组真实影片对照后，再推进按需 VOD、弱网档位或已有桌面原生播放器目标。

各项按最小行为单元实现与提交。跨会话替换和取消强相关，应一起定义协议与生命周期，避免前后端各自修一半。

建议记录：点击→descriptor、descriptor→首个解码帧、seek 输入→目标帧的 P50/P95；每分钟 rebuffer 次数/总时长；实际 bufferedAhead；decoder 掉帧；重复下载字节；暂停后 CPU/GPU/分片增长；会话数与缓存峰值。按 direct / remux / transcode、冷/热启动、1×/2×、本机/LAN 分组。当前没有足够基准数据，不能承诺百分比收益。

### 本轮验证与局限

- 前端：6 个文件、85 项测试通过。
  - 根目录：`pnpm test --maxWorkers=2 src/lib/hls-player.test.ts src/lib/player-hls-seek.test.ts src/lib/player-playback-timeline.test.ts src/lib/playback-targets.test.ts src/components/jav-library/PlayerPage.loading.test.ts src/services/adapters/web/web-library-service.test.ts`
- 后端：playback / app / server 三包选定播放相关测试通过。
  - `backend/`：`go test ./internal/playback/... ./internal/app/ ./internal/server/ -run 'Playback|HLS|Transcode|Throttle|Remux|MediaProbe|Encoder|Keyframe' -count=1`
- FFmpeg stdin 节流合成实验已执行，见 F1；不读取真实资料库视频，不产生持久化媒体产物。
- 已有测试覆盖大量纯决策与 mock 事件，尚不能证明真实 FFmpeg 节流、多客户端交接、网络故障恢复和缓存复用正确。
- 未跑全库测试、生产构建、浏览器 e2e、真实影片转码/GPU/LAN 基准。未将推导场景表述为实测性能结果。

---

## 2026-08-16 历史审计（保留原始记录，当前状态见上节）

本文是对当前播放链路（playback descriptor → direct stream / HLS session）与前端播放器的一次全面审计，目标是找出**进一步增强能力**与**进一步优化性能**的机会。所有结论均已对照源码核验，附文件与行号。

## 现状架构速览

- 描述符入口：`GET /api/library/movies/{id}/playback`（`backend/internal/app/app.go:2349`），基于 ffprobe 探测（`backend/internal/playback/media_probe.go`，内存 `sync.Map` 缓存，key = path+size+mtime）做 direct-file / remux-hls / transcode-hls 决策（`backend/internal/app/playback_decision.go:36`）。
- 直连播放：`GET /api/library/movies/{id}/stream` 用 `http.ServeContent` 提供 Range（`backend/internal/server/server.go:835-877`）。
- HLS 会话：`StartHLSSession`（`backend/internal/playback/manager.go:155`）单 rendition、2 秒 TS 分段、event playlist（`hls_list_size 0`）、remux 优先 + nvenc/qsv/amf（仅 Windows 且开启硬件解码时）/videotoolbox（macOS）/libx264 veryfast crf17 兜底的 profile 失败链；混合 seek（输入 `-ss t-2s` + 输出 `-ss 2s`）；idle janitor 3 分钟 / 30 秒扫描。
- 前端：`PlayerPage.vue`（约 2900 行）消费描述符；hls.js/light 按需加载（`src/lib/hls-player.ts`，maxBufferLength 30 / backBufferLength 90）；窗口内 seek 本地 `currentTime`，窗口外 seek = 整会话换新（`PlayerPage.vue:2120-2203`）。

---

## 一、后端性能问题（按影响排序）

### P0-1 续播 / 中段 seek 永远走转码，remux 只在 t=0 生效

- 证据：`manager.go:775-783` `shouldPreferRemuxProfile` 要求 `StartPositionSec <= 0.001`；而 `app.go:2370-2382` 在 HLS 模式下直接把 `progress.PositionSec`（续播位置）作为会话起点；前端窗口外 seek 也带绝对目标调 `POST playback-session`。
- 后果：本产品高度以续播为中心（进度存储、继续观看、侧栏 resume chip），但 **MKV/AVI 的 h264+aac 源一旦续播就整片转码**（libx264 或 nvenc），而 stream-copy remux 本可以近乎 IO 速度完成。CPU 占用从"IO-bound 复用"变成"整片实时编码"。
- 建议：支持中段起播的 remux（`-ss <t> -i src -c copy`，必要时配合 `-start_at_zero` / `-copyts` / `-muxdelay 0` 处理 TS 时间戳与分段起点；`TimelineOriginSec` 机制已有先例）。失败（如非关键帧对齐导致音画异常）再回退 transcode profile，沿用现有失败链模式。
- 相关测试：`manager_profiles_test.go` 已有 "skips remux when starting mid-stream" 用例，需按新行为改写并补充时间戳连续性验证。

### P0-2 FFmpeg 全速运行 + event playlist 全量落盘，无任何节流

- 证据：所有 profile 均无 `-re` / `-readrate` / `-t`（`manager.go:595-624`）；`-hls_playlist_type event` + `-hls_list_size 0` 意味着分段只增不删，直到会话结束（默认 3 分钟 idle 后 janitor 清理）。
- 后果：
  - **磁盘**：每个 HLS 会话向 `CacheDir/playback-sessions/<id>/` 写入接近整片的 TS（remux 接近 IO 极限几分钟跑完；软转码数倍实时）。用户拖动进度条多次 = 旧会话删掉重写整片。5 GB 源 + TS 封装开销 ≈ 每会话 5–6 GB 峰值。
  - **CPU**：转码会话即使客户端暂停，FFmpeg 也继续全速跑完；播放暂停对 CPU/磁盘零反压。
- 建议分三步：
  1. **快速止血（无前端改动）**：给 remux 与 transcode 都加 `-readrate 1.1`。CPU 尖峰被平摊到整个观看期，暂停后最多再以 1.1x 写 3 分钟（idle janitor 已会 kill），磁盘峰值从"整片"降为"已看部分 + 少量提前量"。
  2. **中期（推荐的目标形态）**："转码器跟随客户端"——manager 已在 `ResolveFile` touch 时知道客户端请求到哪个分段（`manager.go:244-254`），当 FFmpeg 领先客户端超过 N 个分段（如 90 秒）时终止进程，客户端逼近已写边缘时用 `-ss <edge>` + `-hls_flags append_list` 续写同一 playlist。此路径与 P0-1 的中段 remux 能力互相复用。
  3. **可选**：滑窗 playlist（`-hls_list_size N` + `delete_segments`）。注意当前前端"窗口内后向 seek"依赖完整 event playlist，滑窗会使已淘汰分段 404，需要把 404 纳入窗口外换会话逻辑或保证很大窗口，改动面最大，优先级最低。

### P1-3 描述符 GET 同步阻塞等待 FFmpeg readiness，且全局互斥

- 证据：`app.go:2377` 在 GET 请求内同步 `StartHLSSession`；readiness 轮询为 playlist 12s + 首分段 8s + playlist 引用 8s + 第二分段 2.5s（`manager.go:560-577`）；profile 失败逐个重试（每个最坏 ~12s+）；所有会话启动被全局 `sessionStartMu` 串行（`manager.go:159`）。
- 后果：首播延迟数秒到数十秒（硬件编码器全部不可用时最坏 ~30s+）；LAN 第二个客户端起播会被第一个的启动阻塞。
- 建议：
  - 描述符立即返回 `mode: hls` + session `starting` 状态，前端轮询 playlist 就绪（descriptor seam 已有 sessionKind/reasonCode 扩展位，改动自然）；
  - 或至少把 readiness 收敛到"首分段非零即返回"（第二分段本就是 best-effort）；
  - 硬件编码器能力改为启动时一次性探测缓存（解析 `ffmpeg -encoders` 或 1 秒 test encode），替代"12 秒超时失败链"；现有 sticky `lastSuccessfulProfile`（`manager.go:98`）已缓解后续起播，但首播仍慢。

### P1-4 ffprobe 结果仅内存缓存，重启即失效

- 证据：`media_probe.go:24-28`、`media_duration.go:19-25` 均为 `sync.Map`，key 含 mtime。
- 后果：后端每次重启后，每部影片首次播放/首接口调用都要付一次 ffprobe 子进程（~百毫秒级）。
- 建议：把 probe 结果（容器/视频编码/音频编码/时长/码率/分辨率，未来可加关键帧表与字幕流清单）持久化到 SQLite。这也是后续 storyboard 缩略图、精确关键帧 seek、字幕能力（见下）的共同地基。

### P2-5 并发 HLS 会话无上限

- 证据：manager 只抢占**同片**会话（`manager.go:184, 809-844`），不同影片可同时各起一个 FFmpeg，无总数上限。
- 后果：LAN 多客户端各自看不同影片时，可能同时跑多个全速转码（叠加 P0-2 更糟）。
- 建议：加 `maxConcurrentTranscodes` 门槛，超限时排队或降级 direct（带 reasonCode）。

### P2-6 直连 stream 每请求 3 次 SQLite 查询 + stat + open

- 证据：`server.go:835-877` → `movie_stream.go:86-100`（GetMovieDetail + ListLibraryPaths + 路径校验 + stat + open）。
- 评估：直连路径本身走 `http.ServeContent`（Windows 上 sendfile），吞吐没问题；查询开销相对磁盘 IO 可忽略，仅建议顺手加 ETag/Cache-Control 与 movieID→location 的短 TTL 缓存，优先级最低。

---

## 二、前端性能问题

### P0-7 首帧前的串行请求链（冷启动续播场景最痛）

现状串行链：`PlayerView.vue:22-45` 若影片不在内存缓存 → `ensureMovieCached`（内部 `ensureLoaded` 会**分页拉全库列表**，`web-library-service.ts` 500/批）→ `PlayerPage` 才挂载 → `GET /playback` 描述符 →（resume 目标不一致时）`POST playback-session` → hls.js 动态 import → manifest。
- 侧栏"继续观看" chip 在应用冷启动直达播放器时，这条链前两步最重。
- 建议：
  1. `ensureMovieCached` 改为先 `loadMovieDetail(id)` 直接命中详情接口，失败再全库 hydrate 兜底；
  2. 详情页 play 按钮点击（甚至 hover 预取）时预取 `GET /playback` 描述符 / 预起 HLS 会话，进入播放器即热；
  3. `PlayerView` 的影片元数据获取与 `PlayerPage` 的描述符获取并行化（播放器 UI 不应阻塞在列表 hydrate 上）。

### P1-8 直连判定缺少客户端能力协商

- 证据：后端判定纯扩展名 + codec 白名单（`playback_decision.go:132-159`），`hevc`/`av1` 的 mp4 一律判 direct；前端唯一的 `canPlayType` 调用是 HLS MIME（`hls-player.ts:61-75`），无逐编码探测。
- 后果：Firefox / 无 HEVC 硬解的 Chromium 上 HEVC mp4 会得到 `canDirectPlay` 描述符 → 播放报 decode error → 走已有的 HLS 回退（`PlayerPage.vue:859-899`）链路，体验差且白付一次失败。
- 建议：前端一次性探测 `canPlayType('video/mp4; codecs="avc1.…"/"hvc1.…"/"av01…"')` 等 codec string 并上报（query 参数或一次性能力 POST 缓存到后端），后端判定引用该能力集。
- 顺带清理：`preferNativePlayer` 设置项目前在前端播放路径**零消费**（仅存储与设置页展示），`launchNativePlayback`（`POST /native-playback`）也无组件调用方——要么实现要么收敛设置项。

### P2-9 timeupdate 高频发布对象（小优化）

- `onTimeUpdate`（~4Hz）每次 `publishActivePlaybackSession` 新建对象并 bump revision（`PlayerPage.vue:236-249`），使侧栏 computed 每 250ms 级失效；量级不大，可节流到 1Hz。watch-time flush 在卸载时可用 `navigator.sendBeacon` 兜底防丢。

---

## 三、能力增强机会（按价值排序）

1. **上一部 / 下一部 + 连播（autoplay next）**：`browse=` 上下文与连播基础设施（进度、会话清理、`onVideoEnded`）齐备，目前 ended 只收尾（`PlayerPage.vue:1310-1317`）。对长片单连续观看价值最大，改动集中在播放器 + 路由。
2. **时间轴 hover 缩略图（storyboard）**：需后端扫库时/懒生成 ffmpeg tile sprite（`-vf fps=1/N,scale=160:-1,tile=`），经描述符或独立端点下发 + 前端 sprite 定位显示。与 P1-4（probe 持久化）和关键帧表（精确 seek 对齐）协同，是"资料库型播放器"的标志性体验。
3. **字幕支持**：当前 ffprobe 只探视频/音频流（`media_probe.go:78-85`），描述符 track 恒空（`app.go:2607-2608`），FFmpeg 命令无 `-map` 字幕。方案：探测字幕流 → 懒提取为 WebVTT（`ffmpeg -map 0:s:N`）→ 描述符暴露 → 前端用原生 `<track>` 外挂（不依赖 hls.js full build 的字幕控制器）。MKV 内嵌字幕（尤其中文）在此内容域常见。
4. **fMP4/CMAF 分段替代 TS**：`-hls_segment_type fmp4`。TS 封装开销 ~10-15%，fMP4 省 LAN 带宽、seek 粒度更细；hls.js/light 支持 fMP4 demux（light 剥离的是 subtitle/EME/alt-audio 控制器）。改动集中在 `manager.go` 分段命名/参数与 segment MIME。
5. **Media Session API**：`navigator.mediaSession` 集成 OS 媒体键 / 锁屏控制 / 元数据展示。当前零引用，工作量小，对 Electron 桌面形态友好。
6. **播放器小功能集**（工作量均为小）：播放速率持久化 + 0.5–3x 细调；数字键 0–9 百分比 seek、Home/End；逐帧步进（`,`/`.`）；时间轴 hover 时间 tooltip（当前仅拖动时显示）；双击全屏；A-B 循环。
7. **手动画质档位**：后端单 rendition、前端 levels 只读。本地场景带宽充裕，优先级低；若做，建议"原码率 / 1080p / 720p"手动档 + hls.js level 写入，而非完整 ABR 阶梯。

---

## 四、建议实施顺序

| 批次 | 内容 | 理由 |
|------|------|------|
| 第一批 | P0-2① `-readrate 1.1`；P0-7 ensureMovieCached 先打详情接口 | 改动极小、无前端协同、立刻降低 CPU/磁盘峰值与冷启动延迟 |
| 第二批 | P0-1 中段 remux；P1-4 probe 持久化 | 续播 CPU 大头；为 storyboard/字幕/关键帧 seek 打地基 |
| 第三批 | P1-3 异步会话启动 + 编码器能力预探测；P1-8 客户端能力上报 | 首播延迟与判定准确性 |
| 第四批 | 能力项 1/2/3（连播、storyboard、字幕）与 fMP4 | 产品体验跃升 |
| 持续 | P2 项与会话并发上限（P2-5，若做 P0-2② 可一并落地） | 防御性治理 |

## 审计中确认无需处理的部分

- 前端清理卫生良好：hls.js 在卸载/换源/换模式时均 destroy，监听器与定时器全量移除，序号守卫防竞态（`PlayerPage.vue:976-1000`）。
- 直连播放走 `http.ServeContent`，Range/If-Range/sendfile 正确。
- 混合 seek（输入 -ss + 输出精确 -ss）设计正确，有测试覆盖。
- hls.js 配置（30s 前向缓冲 / 90s 后向缓冲 / event playlist 从 session origin 起播）与后端 event playlist 语义匹配。

---

## 落地记录（2026-08-16，后端 + 前端性能批次完成）

### 已落地（8 个提交，全部通过 go test / go vet / vitest / typecheck / lint / test:electron / e2e 除下述既有失败）

| 审计项 | 落地方式 | 提交 |
|--------|----------|------|
| P0-1 续播强制转码 | `keyframe_probe.go`：ffprobe packets 探测目标前最后一个关键帧（30s 窗口），remux 会话从关键帧起播；`-avoid_negative_ts make_zero` 归零媒体时间轴；描述符 `startPositionSec`=关键帧、`resumePositionSec`=请求位置，前端沿用既有本地微调逻辑；探测失败回退转码链 | `8aadc8bc` |
| P0-2 FFmpeg 全速运行 | 所有 profile 输入统一 `-readrate 2.5`（覆盖 2x 倍速 + 余量） | `51b31c9a` |
| P1-3 首播慢 / 失败链 12s×N | `encoder_probe.go`：`-encoders` 列表 + lavfi 微型测试编码，按 ffmpeg 命令缓存，Manager 创建时后台预热、会话启动最多等 2s，不可用编码器移出链；能力未知保留 try-and-fail。readiness 超时未动（避免慢硬件回归） | `9469751b` |
| P1-4 probe 重启失效 | migration `0038_media_probe_cache.sql` + `media_probe_cache.go`，size+mtime 守卫，L2 持久层 | `e099d1c1` |
| P2-6 直连无 ETag | `/stream` 响应 `size+mtime` 强 ETag（304 / If-Range 可用） | `62ac909d` |
| P0-7 首帧串行链 | `ensureMovieCached` 详情优先、全库兜底；`prefetchMoviePlayback`（consume-once、10s TTL）挂在 `PlayerView` 挂载路径，与影片 hydrate 并行 | `ffda5c72` / `db66add2` |
| P1-8 无能力协商 | 前端 `playback-capabilities.ts`（tab 会话缓存 canPlayType）→ `GET /playback?clientVideoCodecs=`；后端仅收窄 mp4 家族判定，webm/ogg 不受影响，未上报保持白名单 | `03a43111` |
| P2-9 高频发布 | `updateActivePlaybackSession` 跳过同状态 + 位置 < 0.75s 的发布 | `d1231da6` |

### 重要修正（实现过程中发现）

- **预取入口从 router 迁到 PlayerView**（`db66add2`）：最初把预取挂在 router `beforeEach`，但 router 静态导入 `library-service` 使 `startWebLibraryService()`（含 `curated-frames/stats`）在应用引导期执行，破坏了"PIN 锁定启动不请求受保护资源"的 e2e 门禁。改挂 `PlayerView`（守卫通过后才懒加载）后门禁恢复。

### 尚未落地（后续批次）

- P0-2②③（ahead-margin 跟随客户端 / 滑窗分段 GC）：`-readrate` 已消除全速突发，但完整观看全程磁盘仍会累积整片分段；"领先即停 + `append_list` 续写"是目标形态。
- P1-3 完整版（描述符异步返回 + session starting 轮询）：涉及描述符契约变更，暂以编码器预探测收敛最坏情况。
- P2-5 并发转码会话上限：需要先决定排队还是拒绝策略。
- e2e `375px library controls` 用例在基线提交 `6fb6119d` 上同样失败（已用 worktree 验证），为**本机既有失败**，与播放优化无关，待单独排查。

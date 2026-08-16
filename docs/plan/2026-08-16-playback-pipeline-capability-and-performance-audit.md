# 播放链路与播放器能力 / 性能审计（2026-08-16）

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

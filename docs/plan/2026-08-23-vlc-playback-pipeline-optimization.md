# 对照 VLC 优化 Curated 播放链路（方案）

- 日期：2026-08-23
- 状态：方案（未实施）
- 对照源码：[videolan/vlc](https://github.com/videolan/vlc)（libVLCcore + `modules/`）
- 对照本仓库：`backend/internal/playback/`、`backend/internal/app/playback_decision.go`、`src/components/jav-library/PlayerPage.vue`、`src/lib/player-hls-seek.ts`
- 既有相关计划：`docs/plan/2026-08-16-playback-pipeline-capability-and-performance-audit.md`、`docs/plan/2026-08-22-direct-play-frame-stability.md`、`docs/plan/2026-08-23-hls-transcode-seek.md`
- 实施计划：`docs/plan/2026-08-23-playback-pipeline-improvement-implementation-plan.md`

## 结论（先读这段）

**不要把 VLC 源码整包搬进本仓库，也不要用 VLC 的 `sout`/`transcode` 替换现有 FFmpeg HLS 会话。**

VLC 稳定，是因为它是**本机解码器 + 自有时钟 + 自有 demux**，在进程内把脏时间轴消化掉再画到窗口上。Curated 当前播放链路是 **浏览器 `<video>` / hls.js + 后端 FFmpeg 实时写出 event HLS**。这两套系统的边界完全不同：VLC 的 `clock.c`、`mp4.c`、`input_clock.c` 不能粘到 Go 里当 HLS 管理器用。

能用的只有两类：

1. **学算法，在现有 FFmpeg + HLS 链路上重做**（这是解决你列的三个问题的主路径，也覆盖 LAN / 浏览器）。
2. **可选：桌面端用 libVLC（LGPL 动态链接）当原生播放器**。这能让 Electron 本机播放接近 VLC 体验，但**治不了**浏览器/局域网转码，也和产品文档里已定的 **mpv** 目标冲突，不应作为「把 VLC 大部分逻辑集成进来」的默认方案。

更接近「浏览器里转码播放」的成熟参考其实是 Jellyfin / Plex 的分段转码，不是 VLC。VLC 的价值在 demux、时钟、快跳；Jellyfin 的价值在「seek 不要杀整条编码器」。

---

## 1. 两套架构为什么对不上

```text
VLC（本地播放）
  access → demux(mp4/mkv/…) → decoder → clock/es_out → vout/aout
  seek = DEMUX_SET_TIME（同一 demux/decoder，冲刷后继续）
  脏时间轴在 demux/clock 内校正，用户看到的是已经对齐的帧

Curated（当前）
  ffprobe 决策
    ├─ direct-file  → Range + 浏览器原生 demux/decode
    ├─ remux-hls    → ffmpeg -c copy → fMP4 HLS → hls.js / MSE
    └─ transcode-hls→ ffmpeg 重编码 → fMP4 HLS → hls.js / MSE
  大跨度 seek = 杀掉 FFmpeg，新开一条 HLS 会话（编码器重新预热）
```

产品文档 `docs/product/2026-03-20-jav-libary.md` §8 的桌面目标是 **mpv + FFmpeg**，不是 VLC。Electron 壳已经存在，但渲染器规则仍是：前端**不得默认假设**原生播放器存在；LAN / 浏览器客户端永远需要 HTML5 路径。

所以：即便以后接 libVLC 或 libmpv，**现有 HLS 链路仍必须单独治好**。把 VLC 当银弹会留下浏览器/LAN 这条「屎山」。

---

## 2. 许可证：哪些能「直接用」

| 东西 | 许可 | 能否放进 Curated |
|------|------|------------------|
| 复制 `modules/demux/mp4/mp4.c`、`src/clock/*` 进 Go/Electron | GPLv2+（播放器本体） | **否**。静态或源码级吸收会把整个应用 GPL 化。 |
| 动态链接 **libVLC**（`libvlc.dll` + 插件目录） | LGPLv2.1+ | **可以**，须保持动态链接、能替换引擎、插件与应用分离。体积大约一百多 MB 插件树。 |
| 调用本机已安装的 `vlc.exe`（协议/命令行） | 不吸收源码 | **可以**，等价于现在的 PotPlayer 协议模板，体验是「交给外部播放器」。 |
| 只借鉴算法（关键帧跳、edit list、缓冲/hurry-up）用 FFmpeg 参数重做 | 思想不构成版权复制 | **可以，且是推荐路径**。 |

「把 VLC 大部分逻辑集成进来」在工程上等于再做一个 libVLCcore。那不是优化，是换产品。

---

## 3. VLC 源码里真正有用的点（按你的三个问题）

VLC 仓库结构：`lib/`（libVLC API）、`src/`（libVLCcore：input/clock/playlist）、`modules/`（demux/codec/sout）。有用的不是 UI，是下面这些模块的**行为**。

### 3.1 部分影片播放效果不好

**VLC 怎么做**

- [`modules/demux/mp4/mp4.c`](https://github.com/videolan/vlc/blob/master/modules/demux/mp4/mp4.c)：完整处理 `elst`（edit list）、`ctts`（composition offset，含负 CTS shift）、`moov/moof`。浏览器 Chromium 的 progressive MP4 demux 对同一类脏文件不宽容，所以直放会抖、HLS remux 后会稳。这和 `docs/plan/2026-08-22-direct-play-frame-stability.md` 已确认的根因一致。
- [`src/clock/`](https://github.com/videolan/vlc/tree/master/src/clock) + [`src/clock/input_clock.c`](https://github.com/videolan/vlc/blob/master/src/clock/input_clock.c)：主从时钟、PCR 低通滤波、用 `pts_delay` 吸收抖动；来不及就**停一轨或丢帧**，而不是把错误时间戳交给 GPU 硬解。
- 解码在 VLC 进程内完成，不经过 MSE。

**Curated 现状**

- 脏时间轴启发式只有三条：帧率差 > 2%、`avg_frame_rate` 显式 `0/0`、片头负 PTS。负 PTS 强制 `transcode-hls` + `-fps_mode cfr`（正确，因为 copy remux 留着 B 帧节奏仍会抖）。
- `streamPushEnabled=false` 时脏片仍直放。
- 直放运行时掉帧再切 HLS（计划 P1）**还没做**。
- 转码画质：`libx264 veryfast crf 17`、NVENC `p5 cq 19`、QSV `medium global_quality 20`。偏「好看」而不是「实时」。

**可落地的借鉴（不要拷 C）**

1. 把 ffprobe 再挖一层，对齐 VLC mp4 demux 会看的东西：`elst` 非空、`ctts` 含负 offset、音视频起始 PTS 差过大、GOP 极长。命中就不要直放。
2. 直放数秒内 `droppedVideoFrames` 超阈值 → 自动切 remux/transcode（已有计划，应做）。
3. 转码目标从「尽量原画」改成「实时优先」：软编 `veryfast` + 更高 CRF 或 `zerolatency`；硬编改 `p1`/`fast` 档。画质略降，卡顿会少很多。VLC sout 的 `hurry-up` 就是这个哲学。

**不能直接用**

- 把 `mp4.c` 嵌进 Go。FFmpeg 已经有 demux；问题出在**浏览器** demux，不在我们缺一份 C。
- 用 VLC 的时钟去驱动 hls.js。MSE 的时钟是 Chromium 的。

### 3.2 转码播放跳转很慢

**VLC 怎么做**

- `DEMUX_SET_TIME` / `DEMUX_SET_POSITION`：同一个 input 线程里告诉 demux 跳时间。
- `--input-fast-seeking`：牺牲帧级精度，跳到最近关键帧，**不重启 demux/decoder**。
- 本地文件 `STREAM_CAN_FASTSEEK` 时预读策略更激进（`mp4.c` 里 `b_fastseekable`）。

**Curated 现状（这是慢的结构原因）**

- HLS 是 **event playlist + 一条活着的 FFmpeg**，不是 VOD。
- 目标超出「已写窗口 + 30s」就 `createPlaybackSession`：**杀进程、新目录、编码器预热、等首个 fMP4**。
- 转码中段起播仍是混合 seek：输入 `-ss t-2` + 输出 `-ss 2`。每次换会话额外解码约 2 秒，只为帧精确。
- 关键帧探测只服务 **remux**（`keyframe_probe.go`，30s 窗口）；转码 seek 不用关键帧索引。
- 2026-08-23 已做窗口内复用 / `ended` 不当成整片结束 / `readrate 12`。大跨度拖进度条仍然慢。

VLC **没有**「杀掉编码器再开一条 HLS」这种模型。把 VLC 的 seek 源码搬过来也接不上。要对齐的是 VLC 的**语义**：seek = 同一播放器内快跳到关键帧，而不是重建管道。

**可落地的借鉴**

| 优先级 | 做法 | 对应 VLC / 成熟播放器 |
|--------|------|------------------------|
| P0 | 用户拖进度条用 **fast seek**：只保留输入侧 `-ss` 到目标前最近关键帧，去掉输出侧 2s 精确 `-ss` | `input-fast-seeking` |
| P0 | probe 时持久化关键帧表（或 GOP 间隔），seek 不再现场 `ffprobe -read_intervals` | VLC demux 自带 sample 索引 |
| P1 | 转码会话改 **按关键帧对齐的独立分段按需生成**（VOD playlist），seek 只编码命中段，不杀整条长会话 | Jellyfin/Plex 分段转码；VLC sout 不是好模板 |
| P2 | 同一会话内用 `append_list` 从新关键帧续写，而不是换 `sessionId` | 审计文档里的「跟随客户端」 |

**不要做**

- 用 `vlc --sout '#transcode{…}:std{access=livehttp…}'` 当后端。VLC sout 是推流工具，不是资料库点播会话管理器；许可证、Windows 插件、seek 语义都更差。

### 3.3 播放时转码卡顿

**VLC 怎么做**

- `file-caching` / `network-caching`：先缓冲再播。
- `input_clock_GetJitter`：抖动变大就加大 `pts_delay`，宁可晚播几十到几百毫秒，也不 underflow。
- sout `transcode` 的 `hurry-up`：CPU 跟不上就降质量，保住实时。
- 本地文件可以 pace control：先尽快编码，播放端读缓冲。

**Curated 卡顿的真实机制**

1. 会话在 **1 个分片（2s）** 就绪后就返回，第二片最多再等 2s；前端最多再等 2.5s 攒约 4s。编码器一旦瞬时掉到 <1x，hls.js 立刻 starvation。
2. `-readrate 12` 在编码器已经 < 实时时没有意义，它只限制「读入」；真正瓶颈是 encode。
3. `-hwaccel auto` 解码 + NVENC/QSV 编码抢同一 GPU 时，速度会掉到 0.7–0.9x，表现为周期性卡一下。
4. `crf 17` / NVENC `cq 19` / QSV `medium` 对 1080p+ 长 GOP 源不一定实时。
5. event playlist 在 FFmpeg 写 `#EXT-X-ENDLIST` 前被当成 live；窗口耗尽仍可能误伤（已修一层，未从根上改成 VOD）。

**可落地的借鉴**

1. **起播门槛**：转码会话至少攒 8–12s 稳定超实时（ffmpeg `-progress` 的 `speed=` > 1.1）再让前端起播；追不上就立刻降档（更快 preset / 更高 CRF / 关 hwaccel decode）。
2. **hurry-up 失败链**：会话中途 `speed < 1.0` 持续 N 秒 → 不停播、换更快 profile 从当前关键帧接着写（或提示「已降画质保流畅」）。
3. **解码/编码不要默认双开 GPU**：转码时默认软解 + 硬编，或硬解 + 软编，禁止默认 `hwaccel auto` + `h264_nvenc`。
4. 生产者必须领先消费者：领先 < 6s 时去掉 readrate 限制；领先 > 90s 再节流。现在是固定 12x，两头不靠谱。
5. x264 HLS 友好参数：`keyint` 对齐 2s 分段、`-tune zerolatency` 或关闭 B 帧，减少「分片写完但解码依赖还没到」。

---

## 4. 「集成 VLC」三条路，只推荐一条主路

### 路径 A — 只借鉴，继续 FFmpeg + HLS（推荐，先做）

覆盖桌面、浏览器、LAN。工作量在现有 `playback` 包和播放器 seek，不引入 GPL/LGPL 引擎。

对应三个用户问题：更准的脏时间轴决策 + 直放掉帧兜底；fast keyframe seek +（中期）分段 VOD 转码；起播缓冲 / hurry-up / GPU 互斥。

### 路径 B — Electron 内嵌 libVLC（可选，后做，且不要替换 A）

- 动态链接 LGPL libVLC，HWND/纹理输出到 Electron 窗口。
- **本机文件直解**，脏时间轴、seek、卡顿都会接近 VLC，因为根本不再转码给浏览器。
- 代价：插件树体积、Windows 运行库、字幕/OSD 与现有 Vue chrome 两套 UI、进度仍要回写 SQLite、LAN 客户端用不上。
- 与已文档化的 **mpv `--wid` + JSON IPC** 目标重复。若做原生播放器，**优先 mpv**：IPC 更干净、体积更小、资料库类产品（含本仓库产品设计）已选它。选 VLC 的唯一理由是「某些残片 mpv 也播不好」——那是白名单例外，不是整引擎替换。

### 路径 C — 调用系统 VLC / 协议打开（最低成本，体验最割裂）

类似现有 PotPlayer 模板。不优化应用内播放器，用户离开 Curated chrome。可保留为「外部播放器」选项，**不能**当主方案。

### 明确拒绝

- 把 `videolan/vlc` 当 git submodule 编译进 `curated.exe`。
- 用 VLC sout 替换 `backend/internal/playback/manager.go`。
- 前端 WASM 跑 libVLC。体积、线程、文件路径、PIN 会话都对不上。

---

## 5. 建议实施顺序（优化点清单）

针对当前「屎山感」：决策规则、FFmpeg 命令、HLS 会话生命周期、前端 seek 散落在多处。优化应**按行为切片提交**，不要先做大重构。

### 批次 0 — 诊断（1 次提交级）

播放诊断已有 `sessionKind` / `reasonCode` / profile。补齐用户能感知的三项，方便对症：

- 当前 `speed=`（ffmpeg progress）
- 已写窗口时长 vs 播放头
- 最近一次 seek 是 reuse 还是 swap

没有这个，后面调 `readrate` / CRF 只能猜。

### 批次 1 — 卡顿与画质（对应问题 1 的「转码观感」+ 问题 3）

1. 转码默认实时档：x264 `veryfast` + CRF 21–23（或 `zerolatency`）；NVENC `p1`/`ll`；QSV `fast`。原画档可留设置项。
2. 转码会话禁止默认 `hwaccel auto` + 硬编同时开。
3. 起播改为「至少 8s 缓冲或 speed>1.1」，不要 2s 就交给 hls.js。
4. 动态 readrate：落后就放开，领先过多再限。

### 批次 2 — 转码跳转（对应问题 2）

1. 进度条 seek：fast keyframe input `-ss`，去掉混合 2s accurate seek。
2. 把关键帧索引写入 `media_probe_cache`（已有 probe 表），seek 热路径不再现场扫 packet。
3. 窗口外 seek 仍换会话，但新会话必须走关键帧对齐 + 更快 profile，目标是 1s 内出首片。

### 批次 3 — 播放效果（对应问题 1 的直放抖动）

1. 扩展脏时间轴启发式（elst/ctts/A-V PTS 差）。
2. 直放掉帧自动切 HLS。
3. 反复观看的脏片可后置 sidecar remux（已有计划 P2，非必须）。

### 批次 4 — 结构治理（中期，真正消掉「一条 event 转码」）

按关键帧把转码改成 **VOD 分段按需编码**。这是 Jellyfin 模型，也是转码 seek 变快、卡顿变少的根治。工作量最大，但比「集成 VLC」小一个数量级，且不改产品边界。

### 批次 5 — 桌面原生播放（目标架构，与 VLC 无关）

按产品文档接 **libmpv**，本机直解。浏览器/LAN 继续走批次 1–4 的 HLS。不要为了「VLC 更成熟」把目标引擎改成 libVLC，除非 mpv 实测打不开特定残片再开白名单。

---

## 6. 当前 vs 目标 vs 未决

**当前**

- 描述符三态：`direct-file` / `remux-hls` / `transcode-hls`。
- 脏时间轴 + 负 PTS 规则已部分落地；中段 remux 已按关键帧 copy。
- 转码仍是单 rendition event HLS + 换会话 seek。
- 原生播放器仍是可选协议模板；mpv/libVLC 都未嵌入。

**目标（本方案）**

- 浏览器/LAN：更快、更稳的 FFmpeg HLS（fast seek + 实时档 + 最终分段 VOD）。
- 桌面：libmpv 直解（已有产品目标），不是 libVLC。
- VLC：只作算法参考与「外部播放器」后备。

**未决（需要你拍板后再写代码）**

1. 转码默认画质：保流畅（推荐）还是保 CRF 17 原画感。
2. 批次 4 分段 VOD 是否纳入下一迭代，还是先只做批次 1–2。
3. 桌面原生播放器继续 mpv，还是你坚持 libVLC（不推荐作为默认）。

---

## 7. 一句话对照

| 你的问题 | VLC 为什么看起来没这问题 | 我们该做的 |
|----------|---------------------------|------------|
| 部分影片效果差 | 自有 mp4 demux + clock，不把脏文件交给 Chromium | 更准地识别脏片走 HLS；转码用实时档；掉帧自动切换 |
| 转码跳转慢 | seek 不重建管道，快跳关键帧 | 去掉精确 2s seek；缓存关键帧；中期改分段 VOD |
| 转码卡顿 | 先缓冲、hurry-up、独立时钟 | 起播多攒几秒；speed 掉下来就降档；GPU 编解码不要互抢 |

成熟项目的可靠性，来自**匹配问题的架构**，不是来自把那份 C 拷进仓库。

---

## 8. 对照 video.js / hls.js / dash.js（2026-08-23 补）

三个仓库都是 **浏览器 MSE 播放栈**，和 VLC、和后端 FFmpeg 会话不是一类东西。它们负责「清单 + 分片怎么喂给 `<video>`」，不负责「源文件怎么 demux / 要不要转码 / seek 要不要杀编码器」。

```text
video.js     = 播放器壳（控件、插件、主题）+ 可挂 VHS
hls.js       = HLS 协议引擎（你们已经在用 light 构建）
dash.js      = MPEG-DASH 协议引擎（要后端出 MPD，不是 m3u8）
VHS          = video.js 自带的 HLS/DASH 引擎，和 hls.js 是竞品不是上下游
```

| 库 | 许可 | 你们现在 | 换成它能否治三个问题 |
|----|------|----------|----------------------|
| [hls.js](https://github.com/video-dev/hls.js) | Apache-2.0 / MIT | **已用** `hls.js/light` `^1.6.16`，`src/lib/hls-player.ts` | 引擎已经对了。换 full build 或换版本治不好杀会话 seek |
| [video.js](https://github.com/videojs/video.js) | Apache-2.0 | 未用；自有 `PlayerPage` chrome | 否。它是 UI 框架，会和现有低干扰播放器抢皮 |
| [dash.js](https://github.com/Dash-Industry-Forum/dash.js) | BSD-3 | 未用 | 否，除非后端改出 DASH。协议换皮不消 event 转码 |

### 8.1 hls.js：留下，继续调，不要换引擎

这是三者里**唯一已经在链路上、也该留下**的。Safari 走原生 HLS，其余走 MSE；light 构建刻意丢掉字幕 / EME / 多音轨 / CMCD，和当前单 rendition 本地流匹配。

已对齐 event 转码语义的配置：`autoStartLoad: false` + `startLoad(0)`（不要跳 live edge）、`maxBufferLength 60`、`maxStarvationDelay 12`、cookie 凭证。窗口内 seek / 假 `ended` 也已经在 `player-hls-seek.ts`。

**还能从 hls.js 本体学、但不必换库的点：**

- 转码卡顿时看 `FRAG_LOAD_EMERGENCY_ABORTED` / buffer stall，而不是先怪播放器皮肤。
- 若改分段 VOD，可把 playlist 从 EVENT 改成 VOD（带 `#EXT-X-ENDLIST` 的完整清单），hls.js 的 seek 会变成「加载目标分片」，这才用上它的长处。
- 不要为了「更成熟」换成 full `hls.js`：包体会变大，用不到的控制器还会干扰现有绑定。

**换不掉的：** 目标超出已写窗口时，前端只能 `createPlaybackSession`。hls.js 再强也变不出尚未编码的分片。

### 8.2 video.js：不要引入

video.js 解决的是「网站要一个带控件的通用播放器」。Curated 已经有萃取帧刻度、续播、HLS 会话切换、沉浸 chrome、快捷键。换 video.js 等于重做播放器 UI，还要再接 VHS 或 `videojs-http-streaming`。

VHS 播 HLS 并不比现在的 hls.js 更懂「FFmpeg 还在写 event playlist」。它甚至更偏标准直播：live 窗口外的 seek 会被拧回 live edge（`allowSeeksWithinUnsafeLiveWindow` 才是例外）。我们要的是从会话原点起播、并在窗口增长后向前追，和 VHS 的直播默认值是反的。

唯一可借鉴、仍不必装 video.js 的：VHS 对 underflow 会 seek 到当前时间做 resync。若卡顿表现为 MSE 卡死而不是编码器掉速，可以在现有 `PlayerPage` 里对 stall 做一次小幅 `currentTime` 轻推，作为兜底，而不是换壳。

### 8.3 dash.js：现在不要切 DASH

MPEG-DASH 的 `SegmentTimeline` / CMAF 对**已经打好的 VOD** seek 很友好：播放器按时间算分片 URL，不靠一条不断变长的 event m3u8。这看起来像「跳转会快」。

但前提是分片已经在磁盘上。Curated 转码慢，是因为分片还不存在。把 muxer 从 `-f hls` 改成 DASH MPD，只是换清单格式；编码器预热、混合 seek、`readrate`、GPU 互抢一个都不会消失。还要双栈（Safari 仍更吃 HLS）、描述符、预热、PIN cookie 全改一遍。

批次 4 若做「按关键帧按需分段」，**继续出 HLS VOD m3u8 即可**，hls.js 原生支持。只有出现多客户端 ABR、多码率阶梯、要跟商业 CDN 对齐时，再评估 DASH。本地资料库单 rendition 不需要。

### 8.4 和三个用户问题的对照

| 问题 | 换 video.js / dash.js / 升级 hls.js full | 实际该动的层 |
|------|------------------------------------------|--------------|
| 部分影片效果差 | 无效。直放仍走浏览器 demux；HLS 时间轴仍由 FFmpeg 写出 | 决策 + remux/transcode 参数 |
| 转码跳转慢 | 无效。引擎不能创造未编码分片 | 后端 fast seek + 分段 VOD |
| 转码卡顿 | 最多减轻 MSE underflow 表现，治不了 `speed<1` | 起播缓冲、hurry-up、GPU 互斥 |

### 8.5 修订后的前端结论

- **协议引擎：维持 hls.js/light。**
- **播放器壳：维持自有 PlayerPage，不引入 video.js。**
- **封装格式：维持 fMP4 HLS，不切 dash.js。**
- 前端值得做的只是小优化：stall 轻推、把诊断接到 hls.js 事件、等后端变成真 VOD 后简化 seek。主药仍在 `backend/internal/playback`。

---

## 9. 究竟哪些成熟开源能对上三个问题（2026-08-23）

先说清楚：**没有一个 npm/Go 模块可以 drop-in 修好「浏览器 + 实时 FFmpeg HLS」。** 成熟的是整条产品级转码/播放管道。能抄的是它们的**会话模型与 FFmpeg 策略**，不是把 Jellyfin/Stash 嵌进 Curated。

闭源里 Plex 的分段转码仍然是体验上限；开源里能公开对照、且真的对上问题的如下。

### 9.1 按问题选仓库（不要按明星项目选）

| 问题 | 能真正对上的开源 | 对不上的 |
|------|------------------|----------|
| 部分影片效果差（脏时间轴 / 浏览器播不好） | 桌面：**[mpv](https://github.com/mpv-player/mpv)**（libmpv）。浏览器侧：继续 FFmpeg remux/CFR，决策抄 Jellyfin/Stash 的 direct vs transcode | VLC 整包、video.js、dash.js |
| 转码跳转慢 | **唯一结构性答案**：[Kyoo `transcoder` / Gocoder](https://github.com/zoriya/Kyoo/tree/master/transcoder) — 按源片关键帧切段、**按需只编码被请求的分片**，清单是完整 VOD。原理文：[The challenge of writing a on-demand transcoder](https://zoriya.dev/blogs/transcoder/) | Jellyfin / Stash 仍是「seek = 重启一条从头/从 -ss 开始的 FFmpeg」，跳转会正确一些，但不会变快一个数量级 |
| 转码卡顿 | [Jellyfin](https://github.com/jellyfin/jellyfin) 的 `TranscodingThrottler`（领先则 `p` 暂停、落后则 `u` 继续）+ [jellyfin-ffmpeg](https://github.com/jellyfin/jellyfin-ffmpeg) 硬编补丁；[Stash](https://github.com/stashapp/stash) 先写临时文件再 rename 完整分片，避免把半截 segment 交给播放器 | 换 hls.js/video.js |

### 9.2 三条成熟管道，各自能抄什么

**A. [jellyfin/jellyfin](https://github.com/jellyfin/jellyfin) + [jellyfin/jellyfin-ffmpeg](https://github.com/jellyfin/jellyfin-ffmpeg) + jellyfin-web**

业界默认的「浏览器播私人片库」开源参照。客户端仍是 hls.js。和你们一样：HLS 会话、seek 重启 ffmpeg。

值得抄（不要嵌整个服务器）：

- `EncodingHelper` 的 **fast `-ss` + 转码时 accurate seek**；音视频一个 copy 一个 encode 时用 `-bsf:a noise=drop='lt(pts*tb\,T)'` 对齐（2026 PR [#16580](https://github.com/jellyfin/jellyfin/pull/16580)），不要用你们现在的固定输出侧 2 秒 `-ss`。
- `TranscodingThrottler`：按「已转进度 vs 客户端已拉进度」暂停/恢复 ffmpeg，替代固定 `-readrate 12`。
- jellyfin-ffmpeg：Windows 上 NVENC/QSV/AMF 的稳定组合（含 CUDA 下 `-force_idr` 这类坑）。硬编卡顿优先对照它的命令行，而不是再探一套。
- 直放/转码决策：能 copy 就 copy，浏览器不能解再转。你们已有描述符，缺的是更全的「脏时间轴 / 10bit / 奇怪 audio」表。

**解决不了：** 大跨度拖进度条仍然要等新 ffmpeg 出首片。Jellyfin 用户同样抱怨 seek 慢。它把重启做对了，没有取消重启。

**B. [stashapp/stash](https://github.com/stashapp/stash)**

Go + FFmpeg + 浏览器 HLS，栈和内容域最像。HLS 大修在 [#2322](https://github.com/stashapp/stash/pull/2322) / [#3274](https://github.com/stashapp/stash/pull/3274)。

值得抄：

- **跟随客户端：** 请求的分片超前 → 停当前 ffmpeg，从该时间重开；领先太多 → 停进程，缺片再请求时再开。这就是审计里的「转码器跟随客户端」。
- **完整分片才对外：** ffmpeg 写 tempfile，stash rename 后再给播放器。他们明确说这修掉了 skip/stutter（半截 m4s 被 hls.js 吃到）。
- 混合 seek：AVI/WMV 脏源用 fast+slow；干净源只用 input `-ss`。`pkg/ffmpeg/transcoder/transcode.go`。

**解决不了：** 官方自己也承认转码 seek 慢，因为每次 scrub 都重启转码。Direct stream 才快。和你们同一阶级，实现更完整。

**C. [zoriya/Kyoo](https://github.com/zoriya/Kyoo) 的 Gocoder（`transcoder/`）**

这是开源里**唯一公开把「seek 要快」当成设计目标**的转码器。可独立跑（docker / HTTP），Meelo、Blee 也在用。

模型：

1. ffprobe 扫关键帧（packet flags，不解码），缓存。
2. 服务器**自己写完整 VOD m3u8**（带 `#EXT-X-ENDLIST`），分片 URL 在片还没转出来时就已经在清单里。
3. 播放器 seek 到第 N 段 → HTTP 要 `segment-N` → 才 spawn ffmpeg 转**这一段**（并预热后面几段）。
4. remux 与 transcode 都按**同一组源关键帧**切，时间轴对齐。

这直接打掉「event playlist 只能往前长、窗口外必须换整段会话」。现有 hls.js/light **不用换**。

代价：要维护关键帧表、手写 playlist、按段 ffmpeg（`-f segment` + `-segment_times` 或等价物）。Gocoder 目前主力仍是 MPEG-TS；你们已是 fMP4，要对齐的是模型，不是照抄 `.ts`。作者自己的博文把坑写得很清楚，这是可执行的规格，不是口号。

### 9.3 桌面「播什么都稳」：mpv，不是再找一个 Web 播放器

[mpv-player/mpv](https://github.com/mpv-player/mpv)（LGPL libmpv）是产品文档 §8 已选的目标。脏时间轴、VFR、负 PTS、MKV/AVI 在本机解码，**没有转码 seek、没有转码卡顿**。

这是问题 1 在 Electron 本机上的成熟解。LAN/浏览器仍然要走 A/B/C。IINA、mpv.net 只是壳，参考嵌入方式即可。

### 9.4 已经在用、继续用

- [FFmpeg](https://github.com/FFmpeg/FFmpeg)：唯一编码器。成熟方案都是「怎么调它」，不是换它。
- [hls.js](https://github.com/video-dev/hls.js)：唯一该留的浏览器 HLS 引擎（见第 8 节）。

### 9.5 建议怎么落地（相对第 5 节更具体的参照）

| 批次 | 做的事 | 主要对照 |
|------|--------|----------|
| 1 卡顿 | 完整分片才 serve；硬编命令对齐 jellyfin-ffmpeg；领先暂停/落后恢复 | Stash rename + Jellyfin throttler |
| 2 跳转（过渡） | 转码 seek 改 fast input `-ss` + 音频 bsf trim；缓存关键帧 | Jellyfin EncodingHelper |
| 4 跳转（根治） | 按源关键帧的按需分段 VOD，清单先完整 | **Kyoo Gocoder** |
| 5 桌面效果 | libmpv 直解 | mpv |

**不要：** 把 Jellyfin/Stash/Kyoo 当依赖嵌进来；不要用 VLC sout；不要为了「成熟」换 video.js/dash.js。

**可以：** 读 Gocoder 的 Go 实现当批次 4 的蓝图（GPL/许可证需再核 `transcoder/` 目录许可后再决定是借鉴还是 submodule）；FFmpeg 参数与会话状态机用 MIT/Apache 思路在 `backend/internal/playback` 重写。

## 10. 2026-09-09 Electron 集成 libmpv 可行性补充

状态：技术建议，尚未实现或完成原型验证。当前 `electron/preload.cjs` 仅暴露目录选择；现有 Electron 壳与 Web 播放器不等于已嵌入 libmpv。

Electron 可以通过原生桥接集成 libmpv。libmpv 是 C API 播放库，不能直接作为 Vue 组件或替换 Chromium 的 `<video>` 解码器。控制桥接与视频画面合成是两项独立工作。

| 路线 | 实际集成内容 | 适用范围与主要限制 |
| --- | --- | --- |
| mpv 独立进程 + JSON IPC | Electron 管理 mpv 进程并同步播放状态；可另行验证 `--wid` 窗口嵌入 | 初步验证原生播放较快；不是直接链接 libmpv，内嵌仍有平台和层级问题 |
| libmpv + 原生窗口 | 原生 addon 或独立 helper 加载 libmpv，以 `wid` 指向受管理的原生窗口 | Windows 可先验证子 HWND；不能把任意 DOM 元素当窗口句柄，也不能默认 HTML 控件能覆盖原生画面 |
| libmpv Render API + 自定义合成 | 原生侧维护图形上下文与渲染生命周期，再完成与 Electron 的画面合成 | 更适合深度定制，但 libmpv 的纹理不能直接交给页面 WebGL；GPU 共享、跨进程同步、平台差异需要单独设计验证 |

建议先在 Windows 做有界原型：Electron 主进程管理独立原生播放 helper，helper 加载 libmpv 并拥有播放窗口与渲染线程；Vue 经窄化 preload IPC 发送播放、暂停、seek、音量和轨道选择，订阅状态及错误。独立 helper 有利于隔离原生崩溃，但不会自动解决视频嵌入与合成问题。仅需原生播放能力时，也可先用 mpv 独立进程验证控制契约。

原型优先验证现有产品交互：视频上方的进度条、菜单、萃取帧 HUD 能否正确显示，窗口移动/缩放、跨屏 DPI、全屏与焦点是否稳定；再评估是否采用 Render API 深度合成。不能仅凭“能播放”判断可以替换现有 PlayerPage。

业务层通过统一播放接口选择 Web 或原生实现，前端保持浏览器可用。Go 继续负责资料库、媒体访问授权和持久化；本机可访问文件可以直读，远程后端媒体应通过受控 HTTP 访问，不能把远程机器的本地路径当客户端路径。原生请求需要正确传递现有媒体认证，不能假设自动继承 Chromium Cookie。

原生直解可减少因 Chromium 编解码限制产生的转码，并提供字幕、音轨和播放控制能力；硬解、HDR、seek 性能仍取决于构建、驱动、文件索引和显示链路，不能承诺任意影片无卡顿。截图/萃取帧需要新增原生实现，现有从 HTMLVideoElement 读取 canvas 的路径无法原样复用；浏览器与 LAN 客户端继续保留 direct/HLS。

许可修正：前文将 libmpv 简称为 LGPL 不够准确。mpv 默认整体为 GPLv2+，符合条件且排除 GPL 代码的构建可为 LGPLv2.1+；`-Dgpl=false` 本身不是许可合规保证。分发应核对实际 libmpv 二进制、FFmpeg 依赖与对应分发义务，不能仅凭 DLL 名称判断许可。

官方依据（2026-09-09 核对）：

- [libmpv Client API 与窗口嵌入说明](https://github.com/mpv-player/mpv/blob/master/include/mpv/client.h)：推荐 Render API，同时说明 `wid` 更简单但存在问题。
- [libmpv Render API](https://github.com/mpv-player/mpv/blob/master/include/mpv/render.h)：支持 OpenGL 和软件渲染，并规定线程与上下文要求。
- [mpv 许可与构建条件](https://github.com/mpv-player/mpv/blob/master/Copyright)。

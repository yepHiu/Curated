# 播放链路改善实施计划

> 2026-09-06 复审修复：下文历史落地记录中的 stdin `p/u`、固定 60 秒追赶窗口和预热方式已被后续修复替代。当前实现与验证见 [播放链路审计复审](2026-08-16-playback-pipeline-capability-and-performance-audit.md)：进程级暂停/恢复 + readrate 回退、独立会话及有界可取消启动、显式 direct 与首次起点、无重复媒体预热、实际连续缓冲和最多 4 秒动态追赶、有界恢复及换流回退。

- 日期：2026-08-23
- 状态：阶段 0、1、2 已落地；阶段 3–4 待实施
- 对照研究：`docs/plan/2026-08-23-vlc-playback-pipeline-optimization.md`
- 既有止血：`docs/plan/2026-08-23-hls-transcode-seek.md`、`docs/plan/2026-08-22-direct-play-frame-stability.md`、`docs/plan/2026-08-16-playback-pipeline-capability-and-performance-audit.md`

## 目标

在**不换播放器壳、不嵌 VLC/Jellyfin/Kyoo、不切 DASH**的前提下，改善三个用户问题：

1. 部分影片播放效果差（直放抖动、转码观感不稳）
2. 转码播放跳转慢
3. 转码过程中卡顿

主战场是 `backend/internal/playback` + 描述符决策 + 现有 `hls.js/light`。桌面 libmpv 不在本计划内。

## 已锁定的默认（避免开工后再争论）

| 决策 | 选择 | 原因 |
|------|------|------|
| 转码默认画质 | **实时档**（x264 CRF ~22 / NVENC 更快档），原画档可后置设置项 | 卡顿优先于 CRF 17 |
| 协议与前端 | 维持 fMP4 HLS + `hls.js/light` + 自有 `PlayerPage` | 引擎已经对了 |
| 跳转根治 | 阶段 1–3 先把现有 event 会话做快做稳；**按需分段 VOD 放阶段 4** | 根治工作量大，不能挡止血 |
| 原生播放器 | 本计划不做 libmpv / libVLC | 产品目标另开，且治不了 LAN |
| 许可证 | 只借鉴 Jellyfin/Stash/Gocoder 行为，FFmpeg 参数在本仓库重写 | 不 submodule 整台媒体服务器 |

## 非目标

- 引入 video.js、dash.js、VLC sout
- 默认整库预转码 / 改用户片源
- 多码率 ABR 阶梯（本地单 rendition 足够）
- 把描述符 GET 改成异步 `starting`（可后置；阶段 1 用「多攒几秒再返回」即可）

## 现状锚点（实施时对照）

- 决策：`backend/internal/app/playback_decision.go`（脏时间轴 / 负 PTS → HLS；负 PTS 强制转码 CFR）
- 会话：`backend/internal/playback/manager.go`
  - event playlist，2s fMP4，首片就绪即返回，第二片最多再等 2s
  - remux `-readrate 2.5`，转码 `-readrate 12`
  - 转码中段：混合 seek（输入 `-ss t-2` + 输出 `-ss 2`）
  - remux 中段：已按关键帧 copy（`keyframe_probe.go`）
  - `HardwareDecode` 时 remux/转码都加 `-hwaccel auto`（会和 NVENC/QSV 抢 GPU）
  - 软编 `libx264 veryfast crf 17`；NVENC `p5 cq 19`；QSV `medium global_quality 20`
- 前端：`src/lib/player-hls-seek.ts`（窗口内 +30s 复用）、`src/lib/hls-player.ts`、`PlayerPage.vue`
- 诊断：`GET /api/playback/sessions/{id}` 尚无 ffmpeg `speed=` / 已写窗口

---

## 阶段 0 — 诊断（先能看见，再调参）

**目的：** 后面每阶段用同一组信号验收，避免「感觉卡了」无法归因。

### 任务

1. FFmpeg 加 `-progress pipe:1`（或独立 fd），解析 `speed=`、`out_time_us`。
2. `SessionSnapshot` / `PlaybackSessionStatusDTO` 增加只读字段：
   - `encoderSpeed`（如 `1.24x`）
   - `writtenDurationSec`
   - `lastSeekKind`：`reuse` | `swap`（swap 由 POST session 标记即可；reuse 前端不必上报）
3. 播放器 stats overlay 展示上述三项（已有 profile / sessionKind，补三个数）。

### 改动文件

- `backend/internal/playback/manager.go`、`manager_sessions_test.go`
- `backend/internal/contracts/contracts.go`
- 前端：`src/lib/player-playback-stats*.ts`、`PlayerPage.vue` 中 overlay 绑定
- `API.md`（sessions 诊断字段）

### 验收

- 转码播放中，诊断接口 1s 级能读到非空 `encoderSpeed` 和递增的 `writtenDurationSec`。
- 进度条大跨度跳转后，新会话 `StartPositionSec` 与 overlay 一致。

### 提交

单独一个 commit：诊断字段 + 解析 + 测试。不改编码参数。

---

## 阶段 1 — 卡顿与转码观感（问题 3 + 问题 1 的转码侧）

对照：Stash 完整分片、Jellyfin throttler / jellyfin-ffmpeg 实时档。

### 1.1 不要把半截分片交给 hls.js

FFmpeg HLS 增加 `-hls_flags independent_segments+temp_file`（在现有 `independent_segments` 上叠加 `temp_file`）。`ResolveFile` 只 serve 已 rename 完成的 `.m4s` / `init.mp4`。

**文件：** `manager.go`、`manager_profiles_test.go`（断言 flags）
**验收：** 转码起播与跳转不再出现「播一小段突然卡住再加载」中由截断 m4s 引起的那一类。

### 1.2 GPU 编解码不要默认双开

- remux（`-c copy`）**去掉** `-hwaccel auto`。
- 转码：若 profile 是 `h264_nvenc` / `h264_qsv` / `h264_amf`，默认**软解 + 硬编**。
- 仅纯软编 `libx264` 时才允许 `-hwaccel auto`（或保持关，先测再开）。

**文件：** `buildHLSInputPrefix` / `buildTranscodeProfiles`、`manager_profiles_test.go`
**验收：** nvenc 会话命令行无 `-hwaccel auto`；remux 命令行无 hwaccel。

### 1.3 转码改实时档

| Profile | 现况 | 改为 |
|---------|------|------|
| libx264 | `veryfast` `crf 17` | `veryfast` `crf 22`，保留 `-force_key_frames expr:gte(t,n_forced*2)` |
| h264_nvenc | `p5` `cq 19` | `p4`（或 `ll`）`cq 23` |
| h264_qsv | `medium` `global_quality 20` | `fast` `global_quality 24` |
| h264_amf | `-quality quality` | `-quality speed` |

后置（本阶段可选、不挡合并）：`library-config.cfg` / `PATCH /api/settings` 增加 `transcodePreset: realtime | quality`，`quality` 回到现况参数。默认 `realtime`。

**验收：** 1080p 转码会话诊断里 `encoderSpeed` 多数时间 ≥ 1.1；主观画质允许略软。

### 1.4 起播多攒几秒

转码会话就绪条件从「1 片 + 第二片最多 2s」改为：

- **至少 4 个媒体分片**进入 playlist（约 8s），或
- 已写时长 ≥ 8s
超时上限 12s，超时仍返回（避免硬件全失败时卡死），但前端继续沿用现有 2.5s 缓冲等待。

remux 保持现有「首片即可」，copy 已经远超实时。

**文件：** `manager.go` 常量与 `waitForPlaylistSegmentReference`、`manager_readiness_test.go`
**前端：** 转码时可把 `HLS_STARTUP_BUFFER_SEC` 从 4 提到 8（`player-hls-seek.ts`），超时可略增。

**验收：** 转码开播时 playlist 至少能看到 4 条 `#EXTINF`（机器跟得上时）；跟不上时走 1.3 的实时档，而不是再缩短就绪条件。

### 1.5 跟随播放头的节流（替代死 write 12x）

不要继续用固定 `-readrate 12` 当唯一手段：

- 转码默认**不**加 readrate（让编码器尽快攒领先量）。
- 当 `writtenDurationSec - 播放头` > 90s：向 ffmpeg stdin 写 `p` 暂停（Jellyfin；需确认本机 ffmpeg 支持；不支持则退回 `-readrate 1.1`）。
- 领先 < 12s：写 `u` 恢复。
- 播放头由现有 `ResolveFile` touch / 分段请求推断，不必前端新协议。

**风险：** 暂停后忘记恢复会导致播到窗口尽头卡死。必须有「客户端又来拉接近边缘的分片 → 强制 unpause」和超时保险。

**验收：** 暂停观看 1 分钟，CPU 明显下降；再按播放，12s 内恢复写片且不卡死。

### 阶段 1 提交切分

1. `temp_file` + ResolveFile
2. hwaccel 与实时档（可同 commit，都是 profile 命令行）
3. 起播 4 分片
4. throttler（独立 commit，逻辑最容易回滚）

**落地（2026-08-23）：** `temp_file` + 拒绝 `.tmp`；去掉 `-hwaccel auto`；实时档 crf 22 / NVENC p4 cq 23 / QSV fast 24 / AMF speed；转码起播等 `segment-00003.m4s`（超时 12s 仍返回）；转码去掉 `-readrate 12`，按客户端已拉分片 stdin `p`/`u` 节流；前端起播缓冲 8s / 最多等 4s。

---

## 阶段 2 — 转码跳转变快（问题 2 的过渡解）

对照：Jellyfin EncodingHelper fast `-ss`；你们 remux 已有的关键帧探测。
**还不做** Gocoder 按需分段。大跨度跳转仍换会话，但新会话要在约 1s 级出首片。

### 2.1 用户 seek 改为关键帧 fast seek

`buildSeekPlan` 对 **transcode** 不再使用输出侧 2s accurate `-ss`。

流程：

1. `probeKeyframeAtOrBefore`（已有，30s 窗口）
2. 命中：只加输入 `-ss <keyframe>`，`TimelineOriginSec = keyframe`
3. 未命中：输入 `-ss <target>`（fast），无输出 `-ss`
4. 描述符已有 `startPositionSec`（关键帧）与 `resumePositionSec`（用户点的时间）；前端 remux 路径已在 `loadedmetadata` 微调差值，**转码 swap 复用同一套**，不要再写一套偏移。

AVI/WMV 等探测经常偏的容器：可保留「目标前最多 2s 的慢 seek」作为例外（Stash `SlowSeek`），用 probe 的 container 字段判断，不要当默认。

**文件：** `manager.go` `buildSeekPlan` / `buildTranscodeProfiles`、`keyframe_probe.go`、`manager_profiles_test.go`（改掉「中段转码必须有输出 -ss」的断言）
**验收：** 转码片进度条跳到 10 分钟后，新会话命令行只有输入 `-ss`，无输出 `-ss`；首片就绪时间相对阶段 1 明显下降（诊断 `StartedAt` → 首个 `writtenDurationSec>0`）。

### 2.2 缩小换会话范围（前端，小改）

已有 `HLS_SEEK_REUSE_LEAD_SEC = 30`。转码在 1.5 节领先量更大之后，可把 reuse lead 提到 **60s**（仅 `sessionKind === transcode-hls`）。remux 维持 30s 即可。

**文件：** `player-hls-seek.ts`、`PlayerPage.vue` 传入 kind、测试。
**验收：** 转码中连续快进 10s×3 不创建新 session（诊断 sessionId 不变）。

### 2.3 关键帧探测超时保护

窗口探测已是 8s timeout。转码 swap 热路径若 8s 太长：超时则走 2.1.3 的纯 fast `-ss`，不要卡住「正在准备播放流」。

**验收：** 无关键帧索引的残片仍能跳转，最坏等于现在的杀会话，但不比现在更慢。

### 阶段 2 提交切分

1. 转码 fast seek + 测试
2. 前端 reuse lead 60s
3. 探测超时降级（若未含在 1 里）

**落地（2026-08-23）：** 转码默认输入侧 fast `-ss`（命中关键帧则对齐，未命中则按请求时间）；AVI/WMV/ASF 仍混合慢 seek；同一探测结果同时给 remux 与转码；前端 `transcode-hls` reuse lead 60s。

---

## 阶段 3 — 直放效果（问题 1 的直放侧）

对照：`docs/plan/2026-08-22-direct-play-frame-stability.md` 未做的 P1，以及 VLC mp4 demux 会看的 elst/ctts。

### 3.1 扩充脏时间轴启发式

在现有「帧率差 / avg 0/0 / 负 PTS」之外增加（命中且 `streamPushEnabled` → 走 HLS，规则与现决策函数相同）：

- 片头有界探测里，视频 DTS/PTS 差绝对值持续过大（B 帧 + 坏 ctts 的廉价近似）
- 音视频首包 PTS 差 > 0.5s
- 容器是 mp4/mov 且 `ffprobe` 能看到非空 edit list（若当前 JSON 拿不到，用一次有界 `-show_entries packet=dts_time,pts_time` 扩展现有 `%+#16` probe，避免整片扫描）

`probe_schema` 升级，旧缓存行失效重探（与 0040/0041 同一套路）。新 migration `0042_...`。

**文件：** `media_probe.go`、`frame_rate.go` 或新建 `timeline_heuristics.go`、`playback_decision.go`、`playback_descriptor_test.go`、storage cache
**验收：** 已知抖的直放片自动 `reasonCode=source_timestamps_unstable`；干净 CFR MP4 仍 `direct-file`。

### 3.2 直放运行时掉帧兜底

仅 `direct-file`：起播后 3–5s 内若 `droppedVideoFrames / totalVideoFrames` 持续高于阈值（建议先 8%，排除 `document.hidden`），则 `createPlaybackSession(hls)` 并 seek 到当前进度。与现有 decode-error → HLS 同族。

**文件：** `PlayerPage.vue` 或抽出 `src/lib/player-direct-play-fallback.ts` + 单测
**验收：** 漏检脏片直放数秒后自动切 HLS，不必用户刷新。误伤路径：弱核显可后置「不再自动切换」诊断开关，本阶段不做设置页文案堆砌。

### 阶段 3 提交切分

1. probe 启发式 + migration + 决策测试
2. 前端掉帧兜底

---

## 阶段 4 — 按需分段 VOD（问题 2 根治，独立迭代）

对照：Kyoo Gocoder。**本阶段开工前，阶段 1–2 必须已在真实转码片上验证。**

### 模型

1. 后台或首次播放时提取关键帧表（packet flags，不解码），写入 SQLite（勿每 seek 扫 30s 窗口拼凑）。
2. 服务器生成**完整** VOD `index.m3u8`（`#EXT-X-PLAYLIST-TYPE:VOD` + `#EXT-X-ENDLIST`），分片 URL 在未编码时已存在。
3. `GET .../segment-N.m4s` 若缺失：ffmpeg 只转 `[keyframe_N, keyframe_N+k)`（k=1～3 预热），写完再 serve。
4. 播放器 seek = hls.js 要目标分片，**不再** `createPlaybackSession` 换整条 event 会话。
5. 现有 event 路径保留为 fallback（关键帧表失败 / 残片）。

### 注意

- 继续 fMP4 + 现有 hls.js/light，不要为了对齐 Gocoder 退回 MPEG-TS。
- `-force_key_frames` 必须落在源关键帧时间戳上，不能只按 2s 均分（否则 remux 档与转码档对不齐；即便单 rendition，按段 seek 也会裂）。
- 工作量明显大于阶段 1–3 之和；单独开迭代，单独一组 commit。

### 阶段 4 验收

- 转码片从 0 拖到片中、再拖回，sessionId 可不变或仅按片复用缓存目录，等待时间接近「编码 2–6s 分片」而不是「整条编码器预热」。
- 窗口耗尽不再被当成 `ended` 重拉（VOD 有 ENDLIST）。

---

## 实施顺序与依赖

```text
阶段 0 诊断
    → 阶段 1 卡顿（temp_file / GPU / 实时档 / 起播缓冲 / throttler）
        → 阶段 2 快跳（fast seek + 更大 reuse 窗口）  ← 用户最快感知「跳转没那么慢」
            → 阶段 3 直放效果
                → 阶段 4 按需 VOD（根治跳转）
```

阶段 1 与阶段 3 可部分并行（后端 profile vs 决策/前端兜底），但不要和阶段 4 抢同一套 playlist 语义。

建议先打完 **0 → 1 → 2**，用你们平时会卡的几部转码片验收，再决定 4 是否立刻上。

## 测试与回归

每阶段至少：

- `cd backend && go test ./internal/playback/... ./internal/app/ -count=1`（描述符决策在 app）
- 相关 vitest：`player-hls-seek`、`hls-player`、新增 fallback
- 手测（Web API + stream push 开）：
  1. 干净 H.264 MP4 仍直放
  2. 已知抖的 MP4 走 remux/转码且画面稳
  3. 负 PTS / 必须转码的片：起播不卡、暂停不空转 CPU、拖进度条可接受
  4. remux 片（MKV h264+aac）续播仍 copy，不要被 1.3 误伤成转码

## 文档同步（阶段落地时）

行为进描述符/诊断 DTO 时更新：`API.md`、`CLAUDE.md` 播放段落、`.cursor/rules/project-facts.mdc` Playback Seam。阶段 4 若新增按段路由再改公开 API 表。

---

## 当前 vs 目标

**当前：** event HLS 单会话、混合 2s seek、2s 就绪、CRF 17、可能 GPU 双开、脏片启发式三条。
**本计划目标（阶段 1–3）：** 实时档 + 完整分片 + 够用的起播缓冲 + 跟随节流 + 转码关键帧快跳 + 直放漏检兜底。
**阶段 4 目标：** 转码 seek 不再重建整条管道。
**仍未做：** libmpv 本机直解。

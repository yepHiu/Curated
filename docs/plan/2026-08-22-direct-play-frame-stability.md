# 直放帧率不稳：可做与不可做（2026-08-22）

## 问题

部分 H.264/AAC MP4 走 `direct-file`（`GET /stream` + 浏览器原生 `<video>`）时画面抖动、displayed FPS 不稳；同一文件走 HLS（尤其 `remux-hls` / `transcode-hls`）则稳定。根因是浏览器 progressive MP4 demux 对 VFR、错误 `ctts`/`elst`、音视频时钟漂移不宽容；HLS 路径经 FFmpeg 重写时间轴后再由 MSE 播放。

当前决策（`backend/internal/app/playback_decision.go`）只看扩展名 + 编解码白名单 + 浏览器 `clientVideoCodecs`，**不区分时间戳质量**。能解码的 MP4 一律直放。

## 边界（当前 vs 目标）

- **当前**：直放 = 原文件字节；时间戳不友好的 MP4 自动 HLS。HLS 会话为 fMP4（`init.mp4` + `.m4s`），remux 会保留 edit list 里被丢弃的 priming IDR。仍没有运行时掉帧升格路径。
- **目标（推荐）**：干净 CFR MP4 继续直放；时间戳不友好的 MP4 自动 `remux-hls`，用户无需开强制 stream push。
- **不做**：默认整库转码；用 CSS/`requestVideoFrameCallback` 伪装流畅；在 UI 里解释 VFR。

## 选项对比

| 方案 | 是否仍直放原文件 | 对抖动的效果 | 成本 | 建议 |
|------|------------------|--------------|------|------|
| A. Probe 后对“脏时间轴”改走 remux-hls | 否（对该片改 HLS） | 高（与用户已观察到的 HLS 稳相符） | 探测字段 + 决策规则；会话 CPU/磁盘与现有 HLS 相同 | **首选** |
| B. 直放前几秒掉帧超阈值再切 remux | 先直放再切 | 中高，有一次中断 | 前端 `getVideoPlaybackQuality` + 已有换会话能力 | 次选，给漏检兜底 |
| C. 缓存 sidecar：`-c copy` 生成干净 fMP4 再直放 | 直放的是 sidecar | 中高 | 磁盘、首次等待、路径校验 | 后置；适合反复观看的脏片 |
| D. 只给原文件 `+faststart` / 更大 `preload` | 是 | 低（改善起播，几乎不治 judder） | 低 | 不作为本问题主方案 |
| E. 默认 `transcode-hls` | 否 | 最高（CFR 重编码） | CPU 高 | 仅作 remux 失败或强制 stream push |

**结论：** 无法在“继续把同一份脏 MP4 交给 Chromium 原生 demux”的前提下可靠治好抖动。产品上应把这类片从直放决策里拿掉，走已验证的 remux。

## 推荐切片

### P0：时间戳不友好 → 自动 remux-hls（已落地）

扩展 `MediaInfo` / probe 缓存（`0040_media_probe_frame_rates.sql` 帧率字段；`0041_media_probe_negative_pts.sql` 负 PTS；`probe_schema` 当前为 2）。旧缓存行会在下次播放描述符时重新 ffprobe。描述符路径必须校验 `probe_schema`，不能只靠 size+mtime，否则会一直复用缺字段的旧行。

启发式：

- `avg_frame_rate` 与 `r_frame_rate` 相对差超过 2%；或
- `r_frame_rate` 有效且 `avg_frame_rate` 显式为 `0/0` / `N/A`（空字符串视为未探测，不改决策）；或
- 片头有界视频包 PTS 为负（DVDMS-981：首包 `-0.033s` + `KD_` 丢弃 priming 帧，标称帧率仍是 29.97，纯帧率差打不中）

命中且 `streamPushEnabled` 时：可 copy 且无负 PTS 则 `remux-hls`；**负 PTS 强制 `transcode-hls` + CFR**。`streamPushEnabled=false` 时仍直放。

### P1：直放运行时掉帧兜底

仅 `direct-file`：起播后数秒内若 `droppedVideoFrames / totalVideoFrames` 持续高于阈值（需排除弱核显、后台 tab），切到 remux 会话并 seek 到当前进度。与现有 decode-error → HLS 同族，方向相反（直放 → HLS）。误伤要可关或只记诊断。

### P2：反复观看的 sidecar（可选）

对 P0 命中且用户多次播放的条目，后台 `-c copy -fflags +genpts -avoid_negative_ts make_zero -movflags +faststart` 写入 cache，描述符改指向 sidecar 直放。库文件不改。失败则保持会话 remux。

## 明确无效或低优先

- 加大 Range/`ServeContent` 缓冲：直放已是 sendfile，治不了 demux 时间戳。
- 前端显示“标称 fps”：hls.js 的 `FRAME-RATE` 是标签，不是呈现稳定性。
- 扫描入库时整库重封装：默认过重，与“不改用户片源”冲突。

## 开放问题

- 阈值：用哪几部已知脏片标定，避免把故意 VFR（屏录）全部打去 HLS。
- LAN 多客户端：自动 remux 增加并发会话，需与现有会话上限讨论对齐。
- 用户是否要在设置里保留“强制直放（可能抖动）”。默认应自动，不必先做开关。

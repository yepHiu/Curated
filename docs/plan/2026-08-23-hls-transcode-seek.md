# 转码 HLS 进度条跳转卡顿（2026-08-23）

## 问题

转码影片在进度条上跳转（含快进到下一个点）体验差：

1. 先切换播放片段，等待很久。
2. 到了新片段后先播一小段，突然卡住，接着又在加载播放流。

## 根因（对照当前代码）

HLS 是 **event playlist + 实时转码**，不是整片已经在磁盘上的 VOD。

- 前端 `seekToAbsolutePlaybackTime`：目标超出 `<video>.duration`（已写出的 HLS 窗口，通常只有数秒）就 `createPlaybackSession` 换整段 FFmpeg。
- 默认快进 10 秒、转码会话刚起播时窗口只有 2–4 秒 → **几乎每次快进都杀会话重开**。编码器预热 + 混合 seek（输入 `-ss t-2` + 输出 `-ss 2`）是长等待的主要来源。
- 后端就绪条件只保证首个 fMP4 分片，第二分片最多再等 2.5 秒；`-readrate 2.5` 从第一帧就限速。会话一返回，hls.js 播完仅有的 2 秒就 buffer underrun。
- event playlist 在 FFmpeg 写完 `#EXT-X-ENDLIST` 之前会被当成 live。`<video>` 的 `ended` 可能在「当前窗口播完」时触发，前端会拆掉会话再拉流，表现为「播一小段 → 卡住 → 又在加载」。

大跨度跳转（例如进度条点到几分钟之后）仍然应该换会话；小幅快进不应该。

## 已落地

- **前端**：目标落在已写窗口 + 30 秒以内时复用当前会话，等 duration/buffered 追上再 `currentTime`；超时才换会话。
- **前端**：HLS `ended` 若绝对时间仍早于影片总时长，视为窗口耗尽而不是整片结束，保持会话并在窗口增长后继续播。
- **后端**：转码 profile 用 `-readrate 12`（仍高于 2x 倍速，起播后继续在后台攒提前量）；remux 仍是 `-readrate 2.5`。
- **后端**：HTTP 会话在首个 fMP4 分片入 playlist 后即返回；第二分片最多再等 2 秒，不再为了 12 秒提前量堵住「正在准备播放流」。
- **前端**：新 HLS 会话起播前最多再等 2.5 秒看能否攒到约 4 秒缓冲，超时仍开播。
- **hls.js**：`maxBufferLength` 60 / `maxMaxBufferLength` 120，`maxStarvationDelay` 12 秒。

## 不做

- 不把转码改回直放 / remux（负 PTS 脏时间轴仍要 CFR 转码）。
- 不做滑窗 playlist（会破坏窗口内回跳）。
- 不在描述符 GET 里改成异步 `starting` 状态（改动面更大，可后置）。

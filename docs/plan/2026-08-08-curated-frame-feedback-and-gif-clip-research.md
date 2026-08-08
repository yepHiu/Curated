# 萃取帧反馈与视频片段/GIF 需求调研

日期：2026-08-08  
状态：调研结论，待确认交互后实施

## 结论摘要

这两个需求都能纳入现有播放器体系，但建议分成两个相互独立的交付切片：

1. 先增强单帧截图反馈：保留现有轻量视觉快门，在成功落库后补充短提示音、成功状态和可访问的 `aria-live` 文案。该部分完全可以在当前 Web、Mock 和 Electron 渲染层完成，不需要改后端合约。
2. 再增加“按住截图键截取片段”：把按键视为一个有状态的手势，而不是在 `keydown` 中重复截图。短按仍保存一张萃取帧；超过约 350–500 ms 后进入录制态，`keyup` 结束并生成片段。GIF 建议由已有的 FFmpeg 运行时在后端按原视频时间范围生成，前端只负责采集起止时间、显示进度和下载/分享。

推荐默认约束：最长 6 秒、GIF 10–12 fps、宽度 640 px、无音频；生成失败时保留原片段任务错误，不影响已经保存的单帧。GIF 是分享格式，不应成为内部唯一母版，后续可增加 WebM/MP4 导出以避免 GIF 的 256 色和体积限制。

## 当前实现事实

- 播放器是 `src/components/jav-library/PlayerPage.vue`，使用浏览器 `<video>` 播放后端 stream；架构规则明确当前不是 mpv/IPC 播放链路。
- 全局播放器快捷键在 `onPlaybackKeydown` 中处理；当 `event.code === getCuratedCaptureKeyCode()` 时调用 `runCuratedCapture()`。默认键为 `C`，可在 Settings 中改为单个字母、数字、功能键、Page Up/Down；按键保留名单不能覆盖播放控制。
- `runCuratedCapture()` 当前会先启动约 550 ms 的 `.curated-shutter-ring`，保存成功后显示约 800 ms 的 `+1`。沉浸式控件隐藏时才通过 `usePlayerImmersiveChrome().showCuratedFeedback()` 显示短文案。
- `saveCuratedCaptureFromVideo()` 调用 `captureVideoFrameToPng()`：将当前 `<video>` 画到 canvas，再输出 PNG；Web API 模式上传到 `POST /api/curated-frames`，Mock 模式写入 IndexedDB。额外的 download/directory 保存由 localStorage 设置控制。
- 当前已有跨域失败、视频未就绪、canvas/toBlob 失败等单帧错误文案；没有截图成功提示音，也没有片段/GIF DTO、任务类型、下载接口或数据库表。
- 仓库已有可配置、可打包的 FFmpeg，当前主要用于播放/HLS 与媒体探测。已有的任务管理器和 `/api/tasks/{taskId}` 可作为片段导出异步任务的基础，但现有 API 没有“导出视频片段”能力。

## 需求拆解与建议交互

### 单按：萃取帧

单按应以“轻微但确定”的反馈为目标：

- 画面：沿用现有边缘快门，建议把反馈起点从“开始保存”改成“手势确认”，成功后再显示短暂的“已萃取帧 · 00:12”标签；错误则显示明确失败原因。现有 550 ms 动画可在实现时收敛到约 200–300 ms 的快入快出，避免反馈拖尾；录制态计时则保持直到松开。
- 声音：播放音量/静音状态之外，增加独立的“截图反馈音”开关，默认开启。使用本地极短 click/chime（约 50–100 ms），不依赖 CDN；如果 `AudioContext` 被浏览器挂起，在本次用户键盘/指针操作中 `resume()`，失败时只保留视觉反馈。
- 可访问性：通过 `aria-live="polite"` 报告“已保存萃取帧”或错误；尊重 `prefers-reduced-motion`，不把声音作为唯一反馈。
- 反馈时序：成功提示应在本地/服务端持久化成功后触发，避免出现“看见 +1 但实际未保存”的错觉。保存中的延迟超过约 150 ms 时可显示小型 busy 状态。

### 持续长按：片段

建议采用“短按/长按阈值”状态机：

```text
idle
  └─ keydown/pointerdown → armed（记录 gestureStartSec，暂存一帧）
       ├─ 在阈值前释放 → 提交暂存帧，回到 idle
       └─ 达到阈值 → recording（红点/计时/进度）
            ├─ keyup/pointerup/失焦/切片结束 → processing
            └─ 超过最大时长 → 自动停止并 processing
processing → success / error → idle
```

关键行为：

- `keydown` 必须忽略 `event.repeat`；同时监听 `keyup`、`blur` 和 `visibilitychange`，防止切出窗口后录制状态卡死。
- 在 `keydown` 时记录视频时间和一张内存中的当前帧。这样短按仍能得到按下瞬间的画面，而不必等 `keyup` 后视频已经前进。
- 达到阈值后不再提交那张暂存帧，而是用 `gestureStartSec` 到释放时的 `endSec` 生成片段。视频暂停、时间不可用或时长低于最小值时，降级为单帧或提示原因。
- 录制态必须有明显但不遮挡画面的红点、计时和“松开结束”提示；不能只依赖红色，需同时有文字/图标。结束后显示“正在生成 GIF”，生成完成后提供下载/分享入口。
- 最长时长、阈值、GIF 宽度和帧率应先固定为安全默认值；后续再放到设置中，避免第一版出现大量不可控输出。

## 技术路线评估

| 路线 | 优点 | 主要问题 | 建议 |
| --- | --- | --- | --- |
| 浏览器 `video.captureStream()` + `MediaRecorder` | 纯前端、响应直观 | 输出通常是 WebM/MP4，不是 GIF；`captureStream`/MSE/HLS 支持不完全一致；长片段占内存，编码质量受浏览器影响 | 可作为 Mock 或直播放源的后续降级，不作为 Web API 主路径 |
| Canvas `captureStream()` 后用 GIF wasm 编码 | 可直接生成 GIF、可控制帧率 | 需要持续绘制视频、GIF 编码耗 CPU/内存，跨域 canvas 仍会失败；大分辨率很容易卡顿 | 仅适合离线小片段实验，不作为第一版主路径 |
| 后端按源文件时间范围调用 FFmpeg | 帧准确、质量和大小可控，避开 HLS/MSE 渲染差异；已有 FFmpeg 打包与任务基础 | 需要新增导出 API、异步任务、临时文件清理和下载鉴权；Mock 无后端时不可用 | 推荐主路径（Web API/Electron） |

### 推荐的后端 GIF 管线

前端提交 `movieId`、`startSec`、`endSec` 和受限的 `format/fps/width`；后端从电影记录解析真实源文件路径，做路径范围校验和时长/大小上限，再启动 FFmpeg。示意参数：

```text
-ss <start> -t <duration> -i <source> -an
-vf "fps=10,scale=480:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse=dither=sierra2_4a"
-loop 0 <artifact>.gif
```

参数仅作方向说明，实施时需根据 FFmpeg 版本、源视频像素格式和 Windows 路径逐项测试。产物应放在受控临时/导出目录，不写入 `curated_frames.image_blob`；任务完成后返回一次性或短期有效的下载 URL，并由 janitor 清理过期产物。

## 建议的接口与分层（实施时）

遵循现有 service-contract 边界，先定义 DTO、错误码和事件名，再接 Web adapter；Mock adapter 可以明确返回“片段导出需要 Web API”，或实现直播放源的 MediaRecorder 降级。

- `POST /api/library/movies/{movieId}/clips`：创建片段导出任务。请求至少包含 `startSec`、`endSec`、`format`；服务端强制最大时长、最大输出规格和合法时间范围。
- `GET /api/tasks/{taskId}`：复用现有任务状态，增加 artifact 元数据（文件名、MIME、字节数、过期时间）。
- `GET /api/tasks/{taskId}/artifact`（或等价的受控 clip 路由）：下载已完成 GIF，设置 `Content-Disposition`，禁止任意路径读取。
- 可选 `DELETE /api/tasks/{taskId}`：用户取消仍在处理的导出；若任务系统暂不支持取消，第一版至少在前端卸载时停止轮询并保留服务端清理。

前端建议新增 `use-player-clip-capture` composable，把手势状态、阈值、失焦收尾、反馈和任务轮询隔离于 `PlayerPage.vue`；单帧保存继续复用 `saveCuratedCaptureFromVideo()`。后续若加入剪辑面板，也应复用同一套 clip service，而不是从组件直接调用具体 Web/Mock adapter。

## 分阶段交付

### P0：截图反馈（低风险）

- 增加成功/失败统一反馈状态、短提示音、`aria-live`。
- 增加 localStorage 设置 `captureFeedbackSound`（默认 true），Settings -> 萃取帧显示开关。
- 覆盖快捷键、按钮、手柄三种入口；测试成功、失败、沉浸模式和 reduced-motion。

## P0 细化实施方案：单帧视觉、声音与无障碍反馈

### P0.1 先确定反馈分层

单帧动作不建议只依赖一个 `+1`，也不建议每次成功都弹一个会打断观看的全局 Toast。建议分成三层，按状态选择显示：

| 状态 | 画面内反馈 | 控件区反馈 | 全局 Toast |
| --- | --- | --- | --- |
| 用户刚触发 | 立即显示 200–300 ms 的快门/边缘闪光 | 不改变布局 | 不显示 |
| 保存成功 | 显示 650–800 ms 的“已萃取帧 · 00:12”徽标 | 保留短暂 `+1` 或改为勾选态 | 默认不显示，避免连续截图打扰 |
| 保存失败 | 显示可读的错误徽标，持续到下一次操作或约 3 秒 | 保留当前错误文案，便于用户看到恢复建议 | 仅在控件已隐藏或错误需要用户离开沉浸态时显示 |

这样可以满足“我确实知道发生了截图”，同时不会因为连按截图而产生一串 Toast。画面内徽标应使用 Lucide 图标（相机/勾选/警告）和文字，不能只用颜色表达状态。

声音时序采用“触发确认”而不是“成功落库确认”：用户按下截图键时立即播放一次极短的快门音，表示手势已被接收；真正保存成功/失败仍由后续徽标和 `aria-live` 文案确认。这样既能满足即时反馈，也不会因为 Web API 网络延迟让声音晚半秒以上才出现。若产品更偏好严格的成功确认音，可以把声音调用点改到保存成功分支，但不建议同时播放两次声音。

### P0.2 视觉实现细节

建议把当前 `PlayerPage.vue` 中的 `curatedShutterActive`、`curatedPlusOne` 和 `curatedCaptureError` 统一为一个小型状态模型，避免“快门已亮但保存失败”这类错觉：

```ts
type CaptureFeedbackState =
  | { phase: "idle" }
  | { phase: "capturing"; positionSec: number }
  | { phase: "success"; positionSec: number; at: number }
  | { phase: "error"; message: string; at: number }
```

实施建议：

1. 在 `runCuratedCapture()` 入口先记录当前播放时间，立即设置 `capturing`，并触发边缘反馈；这一步必须在 100 ms 内完成，不等待 canvas 或网络。
2. `saveCuratedCaptureFromVideo()` 成功后再切换到 `success`，显示格式化后的时间码；保存失败则切换到 `error`，同时保留现有错误文案。
3. 新增一个 `CaptureFeedbackOverlay.vue` 或等价的渲染块，放在视频表面层而不是底部控制栏中：这样沉浸模式下控件隐藏，反馈仍然可见。
4. 成功/失败徽标放在视频表面上方约 8%–10% 处，同时保留至少 12 px 顶部安全间距，尽量避开人物主体和底部进度条。
5. Overlay 使用 `pointer-events-none`、现有播放器 z-index 层级和主题 token；不改变视频尺寸、不触发布局重排。
6. 快门动画只修改 `opacity` / `box-shadow`，成功徽标使用 `opacity + transform: scale(0.96 → 1)`；正常动效 200–300 ms，退出时间略短于进入时间。
7. `prefers-reduced-motion: reduce` 下不做缩放和边缘扩散，只保留短暂高对比度状态和文字。
8. 失败徽标必须同时显示图标和错误文案，颜色只做辅助；暗色视频背景上使用带半透明黑底和边框的浮层，保证文字对比度。

建议新增的文案键（需同时加入 `zh-CN`、`en`、`ja`）：

- `player.captureFeedbackCapturing`：正在萃取当前帧…
- `player.captureFeedbackSuccess`：已萃取帧 · {time}
- `player.captureFeedbackError`：萃取失败：{reason}
- `player.captureFeedbackSoundOn` / `player.captureFeedbackSoundOff`
- `settings.captureFeedbackSoundTitle` / `settings.captureFeedbackSoundHint`

### P0.3 声音实现细节

推荐新增 `src/lib/curated-frames/capture-feedback-sound.ts`，使用浏览器原生 Web Audio API 生成极短的本地提示音，而不是引入一段较大的音频资源：

- 复用一个懒创建的 `AudioContext`，首次触发时创建；不要在页面加载时创建。
- 在键盘、按钮或手柄触发的用户交互栈内调用 `context.resume()`；如果浏览器仍拒绝，捕获异常并静默降级为纯视觉反馈。
- 用约 90 ms 的柔和机械快门包络：一小段经过低通滤波的噪声质感叠加较低频的正弦双音；总增益先控制在约 `0.05–0.06`，明显低于系统提示音。第一版只播放“触发确认”声音，失败不再补第二个声音，避免连按时变成刺耳警报。目标是接近 iPhone 截图声的质感，但不直接复制系统音频资源。
- 声音开关独立于视频 `muted` 与播放器音量；用户静音视频时仍可选择保留截图确认音。
- 页面卸载时关闭 AudioContext 或释放节点，避免重复创建上下文。
- 该声音属于增强反馈，不应承担无障碍的唯一通知责任；始终同时更新 `aria-live`。

建议使用 localStorage 保存 `captureFeedbackSound`：

```text
键：jav-curated-capture-feedback-sound-v1
默认：true
可接受值："on" / "off"
```

这是渲染层的即时偏好，与当前截图快捷键和保存方式一样属于本机界面设置；第一阶段不需要新增后端 settings 字段，也不需要修改 SQLite。

### P0.4 无障碍语义与焦点规则

- 在播放器表面放置一个始终存在但视觉隐藏的 `aria-live="polite"` 区域；成功时播报“已萃取帧，时间 00:12”，失败时播报原因。
- 失败状态可以使用 `role="alert"`，成功状态使用 `role="status"`；不要让 Toast 抢夺焦点。
- 截图按钮继续保留明确的 `aria-label`，并增加 `aria-keyshortcuts`（例如 `C`）；快捷键改变后，标签和设置页显示值同步更新。
- 快捷键触发时不要把焦点强行移到播放器或 Toast；用户正在操作进度条、输入框或设置表单时，继续沿用现有的快捷键忽略规则。
- 反馈文案使用可换行布局，不固定单行宽度；放大字体或窄屏时不能覆盖播放/暂停按钮。
- 视觉状态至少由“图标 + 文字 + 动效/边框”中的两种表达，不能仅靠粉色/红色区分成功和失败。

### P0.5 与现有代码的对应改动

第一阶段预计只需要前端改动，建议按以下边界实现：

1. `src/lib/curated-frames/settings-storage.ts`：增加声音偏好的读取、写入和默认值函数。
2. `src/lib/curated-frames/capture-feedback-sound.ts`：封装 Web Audio、resume 失败降级和可测试的 `playCaptureTriggerCue()`。
3. `src/components/jav-library/settings/SettingsCuratedSection.vue`：增加声音开关；通过 props/emit 与 `SettingsPage.vue` 连接，不直接访问业务 adapter。
4. `src/components/jav-library/PlayerPage.vue`：把保存前、成功、失败三种状态接到统一反馈模型；保留现有单帧保存函数，不改变 API/IndexedDB 数据格式。
5. `src/components/jav-library/CaptureFeedbackOverlay.vue`（建议）：集中处理浮层图标、文字、动效和 reduced-motion 样式；如果最终代码量很小，也可以暂留在 `PlayerPage.vue`，但状态逻辑仍应独立。
6. `src/locales/zh-CN.json`、`en.json`、`ja.json` 与 locale 测试：补齐上述文案键。

第一阶段不做：GIF、片段任务、后端接口、持久化通知中心记录、系统级全局快捷键和桌面原生音效。

### P0.6 测试与验收矩阵

单元测试：

- 默认声音开关为开启；写入 `off` 后不调用 AudioContext；非法 localStorage 值回退默认值。
- `playCaptureSuccessCue()` 在 `AudioContext` 不存在、`resume()` reject、节点创建异常时都不抛出到播放器。
- 反馈状态从 `capturing → success` 或 `capturing → error` 的定时器会被下一次截图清理，不出现旧徽标覆盖新状态。
- reduced-motion 媒体查询开启时，反馈仍显示文字但不依赖动画完成。

组件/交互测试：

- 快捷键、底部按钮、游戏手柄三种入口都显示同一种成功反馈。
- 控件可见和沉浸模式两种情况下都能看到成功/失败反馈；控件隐藏时不额外重复弹 Toast。
- 视频未就绪、canvas CORS、IndexedDB/API 写入失败时，用户同时得到视觉错误和 `aria-live` 文案。
- 连续快速触发两次时，第二次反馈会替换/刷新第一条，不留下多个并行定时器。
- 页面卸载、影片切换、全屏切换后没有残留定时器、音频节点或 `aria-live` 旧文案。

浏览器验收：

- Chrome/Edge 开启与关闭声音、视频静音、系统音量较低三种组合。
- 深色/浅色主题、高对比度视频画面、窄屏和全屏模式。
- `prefers-reduced-motion` 和屏幕阅读器（至少 Windows Narrator/NVDA 之一）。
- 反馈首次出现时间不超过 100 ms；正常动效不超过 300 ms；错误文案至少保留 3 秒或直到下一次操作。

### P0.7 建议实施顺序

按最小可验证单元推进：

1. 先抽离/补充 localStorage 声音设置和 i18n 文案。
2. 单独实现并测试声音模块，确认浏览器策略失败时不影响截图。
3. 实现 `CaptureFeedbackOverlay` 静态成功/失败状态，再接入 `runCuratedCapture()`。
4. 将现有快门和 `+1` 迁移到统一状态模型，删除重复定时器。
5. 补齐组件测试、locale 测试和浏览器验收；确认 P0 稳定后再进入长按状态机设计。

### P1：长按导出 GIF（Web API/Electron）

- 实现上述状态机与录制态 UI。
- 新增 clip DTO、任务类型、FFmpeg worker/进程、artifact 下载和清理。
- 默认上限：阈值 400 ms，最短 0.4 s，最长 6 s，10 fps，宽度 640，无音频。
- 完成后提供“下载 GIF”；分享按钮先调用系统 `navigator.share({files})`，不支持时回退下载。

#### P1 UI 状态原型

GIF 片段不建议在视频中央放大遮罩，而采用“上方状态徽标 + 底部录制条 + 播放器下方结果卡片”的三层结构。对应的交互原型为 `docs/plan/2026-08-08-curated-gif-clip-demo.html`。

- **预备态（armed）**：按下截图键但尚未超过 400 ms，顶部显示“继续按住开始录制”；如果快速松开，则仍走单帧截图，不进入片段流程。
- **录制态（recording）**：视频顶部约 8%–10% 处显示红点和“正在截取 GIF”；视频底部浮出一条半透明录制 Dock，包含“松开结束”、进度条和 `00:02.1` 计时。Dock 不覆盖视频中央主体，也不改变播放器布局。
- **生成态（processing）**：松开后移除录制 Dock，只保留顶部“正在生成 GIF…”徽标；播放器仍可继续观看，任务处理不阻塞播放。
- **完成态（success）**：播放器下方出现结果卡片，展示 GIF 缩略图、起止时间、分辨率、帧率和文件大小，并提供“下载 GIF”和“分享”两个动作。
- **失败态（error）**：顶部徽标变为错误状态，结果卡片位置显示原因和“重试”；错误同时通过 `aria-live` 播报，不只依赖红色。
- **沉浸模式**：底部播放器控件可以隐藏，但录制 Dock、顶部状态徽标和生成结果状态仍保持可见；结果卡片在控件恢复后显示在播放器下方。

原型中右侧“状态预览”允许直接切换五种状态，便于确认每种状态是否太抢画面、信息是否足够，以及完成后的下载/分享卡片是否适合当前播放器布局。正式实现时，右侧状态预览不会出现，它只是设计评审工具。

### P2：质量与兼容性增强

- 增加 WebM/MP4 母版导出、GIF 二次导出和质量预设。
- 评估 Mock/纯浏览器 MediaRecorder 降级、预录制缓冲（按下前 0.5–1 s）和片段库管理。
- 增加失败重试、任务取消、磁盘空间预检查与导出历史。

### GIF 分辨率取舍（2026-08-08）

当前 GIF 导出链路使用 FFmpeg 的 `scale=<width>:-1`，后端允许宽度范围为 160–960 px，前端和后端默认值均为 640 px。因此提高分辨率不需要重做录制链路，只需要调整默认参数或增加质量档位。

| 宽度 | 适用场景 | 主要取舍 |
| --- | --- | --- |
| 480 px | 快速分享、移动端预览 | 体积最小、生成最快，但细节和字幕容易变糊 |
| 640 px | 默认分享规格（推荐） | 相比 480 px 清晰度明显提升，体积和编码耗时仍较可控 |
| 720 px | 桌面端观看、画面细节较多 | 画质更好，但 GIF 体积和生成耗时会明显增加 |
| 960 px | 留档或后续再处理 | 接近原始清晰度上限，但不适合作为默认 GIF 分享规格 |

从 480 px 提升到 640 px，像素面积约为 `(640 / 480)^2 ≈ 1.78` 倍，实际文件大小通常会增加约 1.5–2 倍，取决于画面运动量、色彩复杂度和时长。GIF 仍受 256 色、无音频等格式限制，分辨率提高不能完全替代 MP4/WebM 母版。

建议下一步将默认宽度调整为 640 px，同时保留 480/720/960 px 作为后续质量预设；若暂时不增加设置项，也可以先固定 640 px，并继续沿用最长 6 秒、10 fps 的安全上限。对于 720 px 及以上，建议在生成提示中显示“文件较大，生成时间可能更长”，并在后续加入体积预估或质量选择。

### 播放过程中导出是否过重（2026-08-08）

当前实现不是从浏览器正在播放的画面持续录屏，而是松开长按后，由后端另起一个 FFmpeg 任务重新打开原视频，只解码选中的 0.4–6 秒片段，再缩放为 GIF。播放接口本身通过 `http.ServeContent` 提供原文件 Range 读取，不经过后端转码。因此两条链路的关系是“浏览器继续播放 + 后端并行做一次短片段解码”，不会把整部影片载入内存，也不会持续占用录制资源。

这仍然会产生可感知的额外负载：

- 播放端通常由浏览器/显卡负责视频解码；GIF 导出当前使用 FFmpeg 软件解码、Lanczos 缩放和 GIF 编码，会额外消耗 CPU。
- 需要同时读取同一个源文件。SSD 通常影响不大，机械硬盘、网络盘或高码率 4K 文件可能出现读盘竞争和短暂卡顿。
- 任务最长 6 秒、10 fps，内存压力主要来自短片段的编码缓冲，通常低于“持续录制整段视频”；但 720/960 px 会显著增加缩放和编码时间。
- 笔记本低功耗模式、老旧 CPU、播放高码率 4K/HEVC 或同时运行扫描/刮削任务时，卡顿和风扇噪声风险会更高。

因此，**分享 GIF 可以在播放过程中直接生成**，但不建议把“高质量归档”也默认走 GIF。推荐分成两个出口：

1. **快速分享**：640 px、10 fps、最长 6 秒，后台异步生成；默认只允许 1 个导出任务，播放继续进行。
2. **高质量归档**：优先导出 MP4/WebM 母版（尽量保留源分辨率，必要时使用硬件编码）；GIF 的 720/960 px 仅作为用户主动选择的分享变体。

第一版建议加入以下保护：导出任务串行化；导出时不暂停视频，但检测到播放卡顿时允许自动降级到 480/640 px；提供取消任务；避免源视频低于目标宽度时被无意义放大；导出结束后清理临时文件。后续再根据真实设备采样 CPU、GPU、磁盘和生成耗时，决定是否开放 720/960 px。

### 动态萃取帧 / Live Photo 式资源（2026-08-08）

这个方向值得做，但建议将产品概念定义为“动态萃取帧”：一张可长期展示的静态关键帧，附带一个可选的短时动态片段。它在交互上借鉴 Live Photo 的“静态封面 + 按需播放”，但不应把 GIF 本身当作库内唯一资源。

#### 与当前实现的差异

当前 `CuratedFrameItemDTO` 和 IndexedDB 行只有图片元数据、视频时间点和静态图片字节/URL；当前 GIF 任务完成后自动下载，artifact 也会按临时任务策略清理。要在萃取帧库中展示动态内容，需要为静态帧增加一个关联 motion 资源及其生命周期，而不是只在卡片上临时替换图片。

#### 推荐的捕获语义

按键按下时先捕获一张关键帧，并将它作为本次手势的候选封面保存在内存中：

- 短按松开：直接保存这张静态萃取帧；
- 长按超过阈值：复用同一张关键帧作为封面，同时提交关联的动态片段；不再重复截取第二张几乎相同的帧；
- 动态片段生成失败：静态萃取帧仍然保留，动态状态显示为失败并允许重试。

关键帧应位于手势开始附近，而不是等长按结束才截取，这样用户看到的“按下瞬间”与库中封面是一致的，也能避免视频已经播放到另一画面后封面跳变。

#### 库内展示方式

推荐采用“静态优先、交互后播放”的卡片策略：

1. 初始只加载静态图片作为 `poster`，动态资源使用 `preload="none"`；
2. 桌面端鼠标悬停约 250 ms、键盘焦点进入或用户明确点击播放时，才加载并播放短片段；
3. 播放结束后回到静态关键帧，卡片右上角显示一个“动态”图标或短环形标识；
4. 同一时间只允许一个或极少数卡片播放，离开卡片立即暂停并释放媒体资源；
5. 移动端不自动播放，使用点击/长按明确触发，避免滚动列表时同时下载多个片段。

视觉上可以把动态萃取帧做成静态卡片的增强状态，而不是把整个网格变成 GIF 墙。这样既保留萃取帧库的浏览效率，也让动态内容成为“发现彩蛋”的交互。

#### 资源格式与存储

- 库内动态预览优先使用短时 WebM/MP4（静音、循环、带静态 poster），GIF 作为分享下载格式；
- 动态预览建议控制在约 1.5–3 秒，围绕关键帧前后截取，不必默认使用完整 6 秒长按片段；
- GIF/视频 artifact 不建议写入 SQLite BLOB，应放在受控媒体目录并通过受保护的资源路由提供；
- 当前 24 小时临时 artifact 清理策略不适用于“已加入萃取帧库”的动态资源；保存为库资源后需要独立的持久化标记和删除联动；
- 可在 `CuratedFrameItemDTO` 增加 `motion` 字段，至少包含 `status`（none/processing/ready/error）、`contentType`、`durationSec`、`width`、`height` 和受控 `artifactUrl`。

#### 分阶段实现建议

- **第一阶段（低风险验证）**：只在长按 GIF 完成后，将现有 GIF artifact 作为可选动态预览挂到萃取帧；卡片默认静态，悬停/点击才播放；先限制单个动态资源和列表并发，验证用户是否真的愿意观看。
- **第二阶段（体验优化）**：增加 WebM/MP4 短预览、poster、动态资源状态和持久化清理；将 GIF 降级为分享派生格式。
- **第三阶段（归档能力）**：增加“动态萃取帧”详情页、下载原始片段/母版、重新生成不同分辨率，以及按设备性能选择质量。

#### 无障碍与性能约束

动态内容必须服从 `prefers-reduced-motion` 和应用内“动态预览”开关；关闭时始终展示静态 poster。卡片需要用文字或 `aria-label` 明确“这是动态萃取帧”，不能只依靠图标。实现上使用 IntersectionObserver、有限并发、可取消加载和离开即暂停，避免列表滚动触发批量解码。

#### 专用分支首版实现范围

`codex/dynamic-curated-frame-gif` 分支的第一版先验证 GIF 关联闭环：长按 C 时先保存一张静态萃取帧，再以 `curatedFrameId` 提交 GIF 任务；后端将 GIF 持久化到应用缓存目录的 `curated-frame-motions` 子目录，SQLite migration `0037_curated_frame_motion.sql` 保存状态和文件元数据；萃取帧列表返回 `motion` 状态，卡片悬停和详情页按钮按需播放关联 GIF。关联 GIF 不再自动下载到浏览器，删除萃取帧时联动删除文件。WebM/MP4、动态预览开关、跨设备迁移与高质量母版留到后续阶段。

## 风险与必须先定的产品决策

- **GIF 体积/画质**：GIF 无音频、最多 256 色；必须限制时长、帧率和宽度，并在 UI 告知“GIF 适合分享，不适合作为高质量归档”。
- **长按误触**：400 ms 阈值需要真实键盘和鼠标测试；短按应只产生一张帧，长按不能额外留下那张暂存帧。
- **暂停/拖动/切片切换**：录制中若视频暂停、用户拖动进度、切换影片或 HLS 会话失效，应自动结束并提示，而不是生成错误时间范围。
- **浏览器安全策略**：截图已有 CORS/canvas 限制；后端源文件路线可规避渲染跨域，但必须做库路径沙箱校验。浏览器声音也可能因自动播放策略被挂起，故声音只能是增强反馈。
- **存储与隐私**：片段可能比单帧大很多，不能默认写 SQLite BLOB；需明确临时目录、保留期和“下载后是否删除”。
- **Mock 语义**：Mock 没有可靠的真实源文件和 FFmpeg。第一版应在界面上明确“GIF 导出需启用 Web API”，不要伪造成功。

## 验收标准（建议）

- 单按截图后 100 ms 内出现视觉反馈；成功落库后 1 s 内出现成功文案/声音（声音关闭时不播放），失败时显示可理解原因。
- 快捷键短按只保存 1 帧；长按一次只生成 1 个片段，不产生重复帧；松开、失焦、切页都能结束状态。
- 片段实际时长与手势起止时间误差不超过 200 ms；超出最大时长会自动结束。
- GIF 在 Chrome/Electron 中可播放、可下载；输出无音频、规格不超过设置上限；任务失败不会阻塞播放器和后续截图。
- Web API、Mock、Electron 的能力差异在 UI 和文档中明确，且不泄露本地任意路径。

## 参考实现位置

- `src/components/jav-library/PlayerPage.vue`：快捷键、单帧按钮、当前视觉反馈。
- `src/lib/curated-frames/capture.ts`：`<video>` 到 canvas/PNG 的实现。
- `src/lib/curated-frames/save-capture.ts`：Web API/IndexedDB 保存和文件导出策略。
- `src/lib/curated-frames/settings-storage.ts`、`SettingsCuratedSection.vue`：截图快捷键与保存设置。
- `backend/internal/server/playback_curated_handlers.go`：现有萃取帧上传入口。
- `backend/internal/playback/`、`backend/third_party/ffmpeg/`：现有媒体探测、FFmpeg 命令解析与打包运行时。

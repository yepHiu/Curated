# Windows 原生播放器原型与后续集成实施计划

日期：2026-10-05（北京时间）
状态：in-progress
关联需求：REQ-0058（原型验收）、REQ-0059（正式 Desktop 接入）
分支：`codex/windows-native-player-prototype`
基线：`408b8c10`，独立工作目录；本计划与原型不改主工作区。

最新进展：§12 的同窗原型已验证；2026-10-06 用户授权后，§14 的 M1–M3 正式入口、会话协调、专用窗口、逐文件记录和侧栏控制已有源码及本地验证，详见 §14.9。用户指定 CD1 已完成 native/Chromium 各五分钟对照（§14.10）：原生零观察掉帧、Chromium 持续掉帧。Windows 通过本机试用开关选择 native，默认仍 Web。M4 已实现固定开发引擎准备和可选打包校验，但完整对应源码/许可及原生安装包验收尚未完成；M5 主页面内嵌与能力增量保留后续阶段。下方早期阶段记录与已发布包行为不因此被改写。

## 1. 目标与决策

用户已明确要求先形成详细实施计划，然后在专用分支制作 Windows 原型。原型要证明：Desktop 能通过原生引擎读取 Server 的原始媒体，处理之前 Chromium 直放不友好的 MP4，并具备可控、可回收、可诊断的播放链路。前期研究见 [播放链路方案 §11](2026-08-23-vlc-playback-pipeline-optimization.md#11-2026-10-05--在-desktop-解决部分-mp4-播放不稳)。

选择 **mpv 独立进程 + JSON IPC** 作为第一阶段，原生视频使用独立窗口。随后依据样本播放效果和画面合成实验决定 mpv 窗口嵌入或 libmpv/helper。初期引擎使用本机 mpv.exe，不自动安装、不随正式包分发；不能把能启动播放器等同于问题片源已验证流畅。

保留现有 Server / Desktop 分工：Server 管理媒体访问和业务持久化；Desktop 管理原生进程、窗口和控制。原型用独立本地控制页验证，不修改生产 PlayerPage，不依赖桌面端读取 Server 的磁盘路径。

## 2. 第一版范围

### 2.1 本轮交付

- 独立原型启动命令与 Electron 本地控制窗口；开发数据目录与正式 Desktop 分开。
- 选择 mpv.exe、输入 Server 根地址；探测 Curated 身份与认证状态，必要时输入 PIN 解锁。
- 读取有限的影片列表、详情及 files；支持检索、文件选择和指定起播位置。
- 原始媒体通过现有 `/api/library/movies/{movieId}/stream?fileId=...` 访问，不自动建立 FFmpeg/HLS 会话。
- 播放、暂停、绝对 seek、倍速、音量、停止；实时显示时间、时长、播放状态、解码器与掉帧诊断。
- 复用现有进度 PUT，按 movieId/fileId 保存；暂停、停止与播放期间有节制地保存，失败明确显示。
- 独立进程和 IPC 管道有界启动、请求超时、错误状态、退出回收；连续启动不会保留多个编码/播放进程。
- 自动测试及真实 mpv 合成媒体实验，记录哪些已验证、哪些仍需问题影片实测。

### 2.2 后续阶段

应用内视频嵌入、生产 PlayerPage 双实现、观看时长统计、自动连播、萃取/截图、原生小窗、字幕与音轨选择、macOS 适配、自动降档及正式引擎分发属于后续阶段。本轮不修改正式播放器默认路径和 Server 配置，不准备发布版本或上传包。

## 3. 原型架构

```text
本地 Vue 控制页（只调用窄化 preload API）
  → Electron 原型主进程（当前 Server、独立认证 session）
    ├─ 现有 HTTP API：影片列表/详情、PIN、逐文件进度
    ├─ loopback 媒体代理：随机能力地址、Range、背压与取消
    │   → 当前 Server 原始媒体接口
    └─ mpv.exe：独立原生窗口 + Windows named pipe JSON IPC
```

采用主进程媒体代理：mpv 只拿临时 loopback 能力地址，不拿 Server Cookie。代理只允许当前选定影片/文件，转发 Range 等必要请求头；上游使用当前 Server 的 Electron Session，保留 HttpOnly 认证、禁用跨来源重定向。流式传输使用背压，不能把整部影片读入内存。

代理及旧进程在停止、换片、换 Server、关闭原型、渲染进程失败时回收。原型本机与远端统一读取 HTTP；没有客户端路径授权和跨机器路径转换。

## 4. 控制契约与生命周期

renderer 请求只能包含业务输入：Server 根地址、PIN、搜索词、movieId、fileId、起点以及枚举控制动作。禁止传任意 shell 命令、mpv 参数、任意媒体 URL。mpv 可执行路径只允许本地原型窗口通过原生文件选择取得，启动使用 spawn 参数数组与 `shell: false`。

状态：`idle → starting → playing/paused → ended/stopped`，失败进入 `error`。返回状态不包含 Cookie、PIN、原始媒体代理地址或完整 mpv stderr。播放会话标识必须独立于 movieId，所有异步回调按会话身份隔离。

IPC 按 request_id 匹配响应；处理拆包/合包，限制单条消息大小；命令超时拒绝并释放；断管和进程退出拒绝所有待响应请求。有界重试仅用于等 named pipe 创建，不能无限重启引擎。

一个会话对应一个进程、一个管道和一个媒体代理。控制窗口关闭先保存最后有效进度，再发送 quit；超时仅结束自己启动的 mpv 子进程，不按名称杀系统播放器。进度写入不允许把旧文件时间保存到新文件，结束和停止语义分开。

## 5. UI 决策框架

| 项 | 原型决定 |
|---|---|
| 产品面与任务 | 独立开发实验窗口；任务是连接 Server、选择同一片源并验证原生播放 |
| 参考 | 现有 Desktop 连接页与设置卡片；复用 Input/Button/Card/Field/Alert/Select |
| 层级与密度 | 当前影片与播放控制为主；连接及引擎选择为准备动作；诊断为次级信息 |
| 状态 | 未选择引擎、连接中、锁定、列表为空、启动中、播放/暂停、结束、网络/引擎/保存失败 |
| 窄窗口 | 卡片纵向堆叠，字段可换行；控件有标签、键盘焦点和可读边界 |
| 系统影响 | 局部原型，不增加全局 token、不更改正式界面；三语文案集中管理 |

原型为开发工具，可显示引擎和解码统计；这些信息不默认进入正式用户播放流程。

## 6. 分阶段实施与提交

### P0 · 计划、分支与契约

建立本计划和 REQ-0058，明确原型独立入口。定义业务 DTO、控制枚举、状态与失败分类。计划文档与代码分别提交，不推送远端。

### P1 · 原生进程与媒体访问（本轮）

实现 mpv named pipe 管理器及 loopback 媒体代理。验收：真实 mpv 读取合成 MP4，时间推进、暂停保持、seek 生效、倍速/音量可改、停止后进程和连接释放；代理转发 Range/认证、拒绝无效能力地址和重定向、取消不继续下载。

### P2 · 可操作的 Windows 原型（本轮）

实现 Electron 原型主进程、preload 与 Vue 控制页，接入影片文件选择和逐文件进度；添加开发启动命令。验收：类型与构建通过，界面能执行上述路径，缺 mpv/PIN 错误/影片空列表/启动失败均有明确状态。自动化测试用独立 fixture Server，避免修改用户资料库。

### P3 · 问题影片对照实验（原型交付后实测）

同一部片源、同一时间区间、同一机器对照 Chromium direct、现有 HLS 与 mpv raw HTTP。包括 DVDMS-981（如可取得原样本）、干净 CFR H.264/AAC、多文件第二片、HEVC、不完整/索引异常文件。记录片源指纹和 probe 摘要、播放区间、引擎/显卡/驱动、硬解状态、呈现掉帧、音画同步与人工观感。负 PTS 不是文件必然损坏的证明。

最低观察：连续播放 5 分钟、至少 3 个 seek 点、暂停/恢复和两次换片；在问题片上至少复现旧路径症状，再确认原生路径改善。合成视频通过不能替代此项。LAN 另记录码率、吞吐与缓冲，原始流卡顿时要区分网络和解码瓶颈。

### P4 · 应用内视频与业务整合（待 P3 结果）

先验证 HWND/wid 与控件层级、全屏、跨屏 DPI 和焦点；必要时评估 libmpv Render API/helper，不能假设 DOM 自动覆盖原生画面。通过后抽取生产 Web/原生播放接口，接入播放宿主、观看时长、连播、截图/萃取及小窗；保持现有 fileId、绝对媒体时间和源切换取消规则。默认路径切换以完整交互验收为前提。

### P5 · 分发与跨平台（另行排期）

Windows 引擎实际构建/许可/依赖核对、版本管理和安装验收；2026-10-06 的建议是首版与 Desktop 一起更新，暂不引入引擎独立在线更新器，见 §13。macOS 窗口/渲染/硬解单独实现验证。Server 托管 UI 若改动则属于 Server 交付；引擎属于 Desktop，按实际变化独立准备版本。失败保留 Web/HLS 能力。

## 7. 验证矩阵

| 类别 | 关键检查 | 证据 |
|---|---|---|
| 管道与进程 | JSON 拆包、响应关联、超时、失败退出、同会话回收 | Vitest + 真实 mpv 集成 |
| 原始媒体 | Range/206、fileId、认证、背压、请求取消 | HTTP fixture + 真实 mpv |
| 认证隔离 | Cookie 不进 renderer/mpv argv；跨来源重定向拒绝 | 主进程契约与代理测试 |
| 业务 | 第二片选择、起点、暂停/停止保存、旧状态不串片 | 原型服务测试与 fixture |
| UI | 加载/锁定/空/失败、键盘、窄窗口、无水平溢出 | 组件检查与 Electron 窗口检查 |
| 问题片源 | 旧症状复现、原生改善、音画同步 | P3 人工对照记录，未完成不宣称解决 |

命令遵循构建范式：仓库根 `pnpm test:electron`、相关前端单测、`pnpm typecheck`、相关 ESLint、原型构建；本轮若没有 Go 变更不重复执行无关 Go 全量。显示缩放全套不在本轮自动运行；Windows 原型窗口做局部尺寸检查。

## 8. 交付与完成判定

本轮交付是可运行的 Windows 原型、专用分支上的原子提交、启动步骤、实测记录与未验证项；无主分支合并、推送或发布。REQ-0058 的完整验收还包括问题样本对照，不能仅凭合成媒体和类型检查标记为 verified。

后续每个阶段完成后更新本文及台账。遇到本机样本不可取得时仍完成控制和合成链路，记录该缺项并等待实际样本验证；不替换为未经证实的“已修复”。

## 9. 2026-10-05 首版实现与验收记录

P0、P1、P2 的首版已实现；P3 问题影片对照仍待执行。REQ-0058 保持 `in_progress`，本节不表示正式内嵌播放器完成或问题影片已修复。

### 9.1 已实现

- 核心：`electron/native-mpv-player.ts`、`native-media-proxy.ts`、`native-player-contract.ts`。
- 原型服务及入口：`electron/native-player-lab.ts`、`native-player-prototype.ts`、`native-player-preload.cjs`。
- 控制页：`src/native-player-prototype/`。只依赖原型桥接，正常 Desktop/Web 不加载此页或增加能力。
- 构建：`electron/tsconfig.native-player.json` 和 `vite.native-player.config.ts`，产物全部位于忽略的 `.workspace/native-player-dist/`。正常 Desktop 的 tsconfig 排除 `native-*.ts`，生产 preload 不改。
- 会话：Windows named pipe；先建立观察订阅，再通过 loadfile 加载媒体，防止早期坏片源错误丢失；file-loaded 才报告成功；命令、加载与退出均有期限。
- 进度：每五秒最多一个待执行周期检查/保存，暂停时间不变时不重复写；保存 query 和正文均带 fileId。发现 Server 认证锁定后停止原生媒体；网络短暂失败不立即丢掉缓冲。
- 三语界面复用语义色与现有基础组件，主窗口内部滚动。原型资料目录为 `%APPDATA%/Curated Native Playback Prototype`；可用主进程环境 `CURATED_NATIVE_PROFILE` 指向绝对临时路径做验收。

### 9.2 验证证据

- 带 `CURATED_NATIVE_MPV` 的完整 `pnpm test:electron`：13 文件 / 93 项通过，含 2 项真实 Windows mpv 集成测试。未设置引擎变量时真实集成用例明确跳过。
- `pnpm typecheck`、正常 Electron 与原型 Electron 编译、原型相关 ESLint、`pnpm build:native-player-prototype` 通过。
- 真正的 mpv 合成测试验证认证原始 HTTP MP4、Range、暂停、精确跳到 7 秒、1.5 倍速、音量 35、恢复、停止以及错误媒体的回收。
- 真正 Electron 控制窗口使用独立临时 profile 与 HTTP fixture：PIN 解锁成功；第二片读取自身 12 秒续播位置；启动后时间推进；暂停停在 14.866667 秒，再跳到 45 秒仍保持暂停；保存路径 `/api/playback/progress/fixture?fileId=part-2` 和正文 fileId 一致。
- 合成窗口样本为 640×360 / 30 fps H.264，观察到 `d3d11va`，观察时呈现/解码掉帧均为 0；不是长期基准，也不是问题影片实测。
- Playwright CLI 经 CDP 操作实际 Electron 页面；640px 与 1000px 宽度均无水平 document 溢出。修复了独立构建漏扫描 UI 类名及控制页缺少内部滚动；控制页 console 无错误/警告。截图与 fixture 请求记录在本地 `output/playwright/native-player/`，不纳入仓库。
- 新分支依赖安装后 Electron 自动下载很慢；本机已有同版本 42.0.1，验收复用了该运行时。未升级 Electron 或改锁文件。

### 9.3 本机运行

在本分支工作目录的 PowerShell 运行：

```powershell
pnpm dev:native-player
```

可通过选择器选已有 `mpv.exe`。本机已经存在 `C:/ProgramData/chocolatey/lib/mpvio.install/tools/mpv.exe`，也可在启动前设置：

```powershell
$env:CURATED_NATIVE_MPV = 'C:/ProgramData/chocolatey/lib/mpvio.install/tools/mpv.exe'
pnpm dev:native-player
```

运行自己的 Server，在原型里填写根地址、连接/解锁、搜索并选作品和文件；续播秒数可改为 0，再点击原生播放。影片在独立 mpv 窗口显示。原型会保存真实逐文件进度，测试问题片前可记录原进度以便恢复。关闭原型会停止它启动的播放器。

真实引擎回归：

```powershell
$env:CURATED_NATIVE_MPV = 'C:/ProgramData/chocolatey/lib/mpvio.install/tools/mpv.exe'
$env:NODE_OPTIONS = '--no-experimental-webstorage'
pnpm test:electron
```

### 9.4 仍待完成

真实问题 MP4 的旧症状复现与原生对照、5 分钟观察、真实 LAN/HTTPS/吞吐、音画同步、不同显卡/驱动、生产内嵌视频层、萃取/截图、原生小窗和 macOS 均未验收。当前提交可用于 P3 实验，不能据合成视频掉帧为 0 就宣布原问题全部解决。

## 10. 2026-10-05 页面融合与桌面专用播放视图

### 10.1 用户反馈与新增约束

用户已运行原型并反馈能正常、流畅播放，随后询问如何融入现有页面；允许考虑 Desktop 专用播放页，但要求 UI 控件尽可能接近原有播放器。本次反馈记为用户实测证据；没有具体文件、播放区间和诊断记录，P3 的完整对照矩阵仍待补齐。

本节为下一阶段的实施建议，尚未实现嵌入。当前实验控制页不作为正式播放器的 UI；后续沿用 Curated 的图标、字体、主题、按钮、进度条、菜单和交互习惯。

### 10.2 现有代码与真正的嵌入边界

`PlayerPage.vue` 的标题、萃取工具、提示、进度条和底部控件是 DOM 浮层，叠在 HTMLVideoElement 上。播放、帧截图、全屏和浏览器 PiP 也直接依赖该元素；整个组件不能仅替换一个标签就完整支持 mpv。`ActivePlaybackHost.vue` 已统一管理当前播放目标和侧栏操作，可继续作为整合入口。

mpv 支持 `--wid`：Windows 下传入 HWND，mpv 创建自己的子窗口并使用指定窗口为父窗口。需要先创建真正的原生视频容器，并跟随页面的视频区域调整位置/大小；HTML div 没有 HWND。只把 Electron 主窗口句柄传给 mpv 不能让它理解 DOM 的视频区域、层级或菜单位置。

原生子窗口与 Chromium DOM 不共用 CSS 堆叠层级。因此「画面位于页面中」与「任意 HTML 控件覆盖画面」必须分开验收，单纯提高 z-index 不能解决。CSS 圆角/遮罩也不能自动裁切原生画面。

### 10.3 可行路线及推荐顺序

| 路线 | 画面与 UI | 适用阶段与限制 |
|---|---|---|
| 原生容器 + mpv --wid | 视频占页面预留矩形；标题、控制栏和侧栏由现有 Vue UI 绘制 | 首选 Windows 嵌入实验；保留现有进程和认证代理；必须处理原生窗口生命周期、焦点、DPI 和遮挡 |
| Desktop 专用播放视图 | 同一 Curated 窗口中，上方标题/文件选择、中间原生画面、下方现有样式控制栏；也可先用独立 Curated 播放窗口验证 | 推荐首个产品视图；控件先放视频区域之外；新建 Vue 页面本身仍不能省去原生容器 |
| 画面上叠加 Web UI / 自有渲染合成 | 恢复现有标题、进度条、菜单直接覆盖画面的沉浸效果 | 后续技术实验；可评估透明 Web UI 层及窗口合成，或 libmpv Render API/helper；不假设已有稳定的 Electron GPU 纹理桥接 |

推荐组合是第二行的产品形态加第一行的画面实现：先做桌面专用播放视图，复用同一套 UI 组件，中心使用原生视频容器；验证后再接到现有播放宿主与路由。页面融合无需以 libmpv 重写全部播放器为前提。

如果必须逐像素保留当前浮在视频上的渐变控件和弹出菜单，再进入第三行。libmpv Render API 将画面交给应用渲染，仍需要解决与 Electron/Chromium 的合成；它不是 Vue canvas 自动可用的接口。不要按帧通过 JS IPC 复制高清图像作为正式播放方案，以免重新产生带宽、内存和延迟问题。

### 10.4 UI 复用与播放实现分离

目标结构：

```text
共享播放布局、控件、播放列表与业务状态
  ├─ WebVideoSurface + WebPlaybackEngine：现有 HTML video / HLS
  └─ NativeVideoSurface + NativePlaybackEngine：Desktop 原生容器 / mpv
```

名称仅表示拟定职责，不表示上述组件已存在。先从 PlayerPage 分离控制栏、标题/文件选择与播放状态契约，随后按能力选择实现；避免复制约 3,600 行 PlayerPage 后让两个页面各自演进。

已有的 `MoviePartSelect`、`PlayerPlaylistPanel`、`PlayerPlaylistRevealTab`、`PlayerProgressFrameMarkers` 可基于 props/events 复用。`PlayerPlaybackSettingsMenu` 当前硬编码 direct/hls 模式，需要改为按引擎能力展示选项。播放控件目前还写在 PlayerPage 内，须先抽取，不能声称已经完全解耦。

统一契约覆盖播放状态、绝对时间、时长、暂停/恢复、seek、音量/静音、倍速、结束/错误及能力。UI 继续使用相同图标、尺寸、配色、倍速项和快捷键；mpv 自带 OSC 不作为 Curated 控件。原生画面的点击、双击、悬停和键盘焦点需由宿主/受控输入桥映射，不能依赖 DOM 自动收到事件。

进度、观看时长、连播和 fileId 归共享业务层；接入正式宿主时必须确定唯一的进度写入方，不能让现有页面与原型服务同时保存。窗口/页面隐藏和认证失效要同步原生可见性、声音及会话策略，不能仅依赖 v-show 或 DOM 卸载。

### 10.5 功能适配范围

| 功能 | 复用/改造原则 |
|---|---|
| 标题、文件选择、播放列表、进度条、播放/跳转/音量/倍速按钮 | 复用视觉与交互；命令改发统一播放接口；桌面首版可固定放在画面外 |
| 续播、历史、逐文件进度、观看时长、连播 | 保持现有 movieId/fileId 与绝对时间规则，使用原生状态驱动 |
| 全屏 | 由 Desktop 同步原生容器与完整播放器窗口；HTML 元素全屏不能自行管理 mpv 子窗口 |
| 小窗 | 实现受控 Desktop 播放窗口及主页面/侧栏同步；不能调用 HTMLVideoElement PiP 完成 |
| 单帧/源帧与片段萃取 | 源帧/片段可沿用 Server API 加 fileId/时间；截图与逐帧操作需原生适配，不再读取不存在的 video/canvas |
| 沉浸式浮层、菜单、提示与遮挡 | 首版采用画面外区域或受控遮挡策略；完整叠加另做窗口合成验收 |

没有实现的功能通过能力和清楚的状态展示；不能仅为了控件外观接近而保留无效操作。

### 10.6 下一次 Windows 实验的完成标准

1. 在现有原型分支建立最小原生容器/helper，mpv 只绘制该容器，视频和控制页在同一 Curated 窗口中；不显示独立 mpv 窗口或默认 OSC。
2. 控制区用现有 Button/Slider/菜单样式，先验证标题、文件选择、时间、暂停、跳转、倍速和音量；连接/引擎配置与诊断收进准备区，正常观看时不占主要布局。
3. 明确 CSS 像素、Electron DIP、Win32 坐标与页面缩放的转换；验证缩放窗口、最小化/恢复、跨显示器与 100%/125%/150% DPI，避免位移、黑块和覆盖控件。
4. 验证菜单打开、弹窗、播放列表展开、路由离开、换片、锁定和退出。焦点回到视频后快捷键仍一致；页面不可见时不残留原生画面。
5. 成功后提取共享控件并接入正式播放宿主，验证侧栏继续/暂停/停止、历史与连播。若独立原生宿主通过而主页面融合未通过，可先交付独立 Curated 播放窗口并使用同一套 UI；原生容器本身未通过时仍保留当前双窗口实验形态，记录原因再决定合成路线，不能把新建 Vue 页面当作嵌入成功。

官方边界参考：[mpv --wid 文档](https://github.com/mpv-player/mpv/blob/master/DOCS/man/options.rst)、[libmpv Render API](https://github.com/mpv-player/mpv/blob/master/include/mpv/render.h)。后者也明确指出窗口嵌入可能受 GUI 框架和平台影响，推荐渲染 API；本项目先用 wid 验证，是为了复用已工作的独立进程链路，最终路线以实际嵌入/合成验收为准。

## 11. 2026-10-05 开源参考项目核查

用户要求寻找成熟开源项目作为页面融合参考。本次读取了 GitHub 仓库信息、发布记录、README 和相关源码；仅做研究，没有引入第三方代码/依赖，没有编译或运行这些客户端。维护日期是截至本次核查的快照，不代表任一具体功能已在 Curated/Electron 42 验证。

### 11.1 优先级与维护情况

| 优先级 | 项目 | 参考价值 | 核查到的维护/交付证据 | 应用源码许可证 |
|---|---|---|---|---|
| 1 | [Jellyfin Desktop](https://github.com/jellyfin/jellyfin-desktop)，旧名 Jellyfin Media Player | Web 业务界面接原生 libmpv；视频下层、透明 Web UI 上层；最接近 Server + Desktop 结构 | 2021 年建立；默认分支提交 2026-08-31；稳定 v1.12.0 有 Windows x64/x86 安装器，v2.0.0 为预发布，当前 README 另有开发构建 | GPL-2.0 |
| 2 | [Stremio Shell](https://github.com/Stremio/stremio-shell) | Qt WebEngine + libmpv Render API 的成熟 Web UI 叠加实现；原生事件与 Web 命令桥 | 2017 年建立；默认分支 2026-03-27 更新到 4.4.183；有 Windows 构建/安装脚本。GitHub Releases 列表较旧，不能据此称最新版本有 GitHub 安装包 | GPL-3.0 |
| 3 | [mpv.net](https://github.com/mpvnet-player/mpv.net) | Windows 原生宿主、wid、输入、全屏和窗口生命周期 | 2017 年建立；2026-01-09 发布 v7.1.2.0，含 Windows x64 安装器及 x64/ARM64 portable；默认分支提交 2026-02-09 | GPL-2.0 |
| 补充 | [Harbor](https://github.com/harborstremio/harbor) | Tauri/React/Rust + libmpv；Windows 透明 WebView2、mpv 子窗口层级、坐标与 HDR 独立 UI 层，技术问题贴近本项目 | 2026-06-02 建立；V0.9.21 有 Windows 安装器，2026-10 另有 DLL 预发布；属于近期项目，不按长期成熟组件评价。源码快照默认分支提交 2026-08-21 | MIT |
| 基础样例 | [mpv-examples](https://github.com/mpv-player/mpv-examples/tree/master/libmpv) | 官方最小 C#/Qt/QML/SDL 嵌入样例；用于理解渲染 API 与宿主，不作为完整客户端 | 默认分支提交 2024-06-07；无 Release，结构简单可定位。许可证以 libmpv/Copyright 为准，不能仅以 GitHub license=null 判断无许可 | 示例提供多种许可及 public-domain 选项；mpv 库单独核对 |

### 11.2 已核对的源码落点

**Jellyfin Desktop：优先学习如何让同一套 Web 界面切换到原生播放。**

- [稳定 v1.12.0 的 webview.qml](https://github.com/jellyfin/jellyfin-desktop/blob/v1.12.0/src/ui/webview.qml)：MpvVideo 与 WebEngineView 的同一窗口布局。稳定交付与新架构研究分开，不能把最新 master 当稳定发行版。
- [当前 webview.qml，206–238 行](https://github.com/jellyfin/jellyfin-desktop/blob/2cb4a4456fd29b5b62d825ef0e5df93ed6913328/src/ui/webview.qml#L206)：MpvVideoItem 在下层，WebEngineView 的 z=100、backgroundColor=transparent、layer.enabled=true；还记录了最小化恢复的黑屏处理。
- [native/mpvVideoPlayer.js](https://github.com/jellyfin/jellyfin-desktop/blob/2cb4a4456fd29b5b62d825ef0e5df93ed6913328/native/mpvVideoPlayer.js)：把 Web 播放器的命令/状态接到原生 player，并使用 setVideoRectangle 等能力。可参考统一播放适配接口、状态与媒体区域，而不是复制整个客户端。
- [MpvVideoItem.cpp](https://github.com/jellyfin/jellyfin-desktop/blob/2cb4a4456fd29b5b62d825ef0e5df93ed6913328/src/player/MpvVideoItem.cpp)：当前实现通过 mpvqt 集成 vo=libmpv，Windows 下配置 OpenGL。其合成由 Qt 承担，不能将 QML z 值等同于 Electron DOM z-index。

**Stremio Shell：优先学习原生画面与透明 Web UI 的完整合成及通信。**

- [main.qml，299 行起](https://github.com/Stremio/stremio-shell/blob/c3a8bcbf857d5569b6ae7444ead0dc0a0814888b/main.qml#L299)：MpvObject 充满窗口，后面的 WebEngineView 使用透明背景；webChannel/transport 转发命令与事件。
- [mpv.cpp，44 行起](https://github.com/Stremio/stremio-shell/blob/c3a8bcbf857d5569b6ae7444ead0dc0a0814888b/mpv.cpp#L44)：QQuickFramebufferObject Renderer 通过 mpv_render_context_create/render 绘制 OpenGL FBO，再参与 Qt 合成。该案例支持原有浮层控件的目标可行，但它不是 Electron 插件。

**mpv.net：优先学习 Windows 窗口与引擎的生命周期。**

- [Player.cs，69 行起](https://github.com/mpvnet-player/mpv.net/blob/ef45baecbdd8e0a249eca9a621fe608143f75c4b/src/MpvNet/Player.cs#L69)：创建 libmpv 上下文，将 Form 的句柄设置为 wid，并设置输入与事件选项。
- [MainForm.cs](https://github.com/mpvnet-player/mpv.net/blob/ef45baecbdd8e0a249eca9a621fe608143f75c4b/src/MpvNet.Windows/WinForms/MainForm.cs)：窗口、全屏、尺寸、Win32 消息与焦点管理。它的 WinForms/WPF/OSC 界面不作为 Curated UI 来源。

**Harbor：作为现代 Web 技术栈的补充实验参考。**

- [mpv.rs](https://github.com/harborstremio/harbor/blob/0117755855d3f43960bad3f9f62b69ef851d5991/src-tauri/src/mpv.rs)：CSS/native 媒体矩形映射、Windows wid、关闭默认输入/OSC、自有状态事件；mpv_force_below 枚举属于宿主的 mpv 子窗口并调整层级，退出时清理遗留画面。
- [webview_helpers.rs](https://github.com/harborstremio/harbor/blob/0117755855d3f43960bad3f9f62b69ef851d5991/src-tauri/src/webview_helpers.rs)：用 WebView2 Controller 设置透明背景；[hdr_overlay.rs](https://github.com/harborstremio/harbor/blob/0117755855d3f43960bad3f9f62b69ef851d5991/src-tauri/src/hdr_overlay.rs) 单独管理透明 UI 窗口、位置、大小和激活状态。
- Tauri/Windows 使用 WebView2，Electron 使用自身 Chromium；其 Controller API 不能直接调用于 Electron。该项目展示了实际问题和处理方式，不能证明 Curated 已有稳定透明叠加桥。

### 11.3 Electron 搜索结果与不直接采用的项目

- [Kagami/mpv.js](https://github.com/Kagami/mpv.js)：历史上可嵌入 Electron/NW.js，但 README 依赖 Pepper/PPAPI、register-pepper-plugins、NaCl SDK，并要求 no-sandbox；最后 GitHub 发布 v0.3.0 为 2018-07-28，源码推送日期为 2024-01-17。不据旧示例认定兼容当前 Electron 42，不作为直接依赖。
- [stevevista/electron-mpv](https://github.com/stevevista/electron-mpv)：README 也基于 mpv.js/NaCl SDK，仓库推送停在 2021；属于历史实验参考。
- [Node-MPV](https://github.com/j-holub/Node-MPV)：JSON IPC 进程控制包装；可以参考命令与观察状态，但不能单独解决画面嵌入和 DOM 叠加。

在本次搜索范围内，没有核实到一个维护成熟、可直接安装并保证 Electron 42 + Windows 视频/DOM 合成的组件。成熟的完整产品主要证明 Qt 合成路径；现代 WebView2 案例也需要原生桥。因此后续仍需自己的小范围嵌入验收，不承诺安装一个 npm 包即可完成。

### 11.4 对 Curated 下一阶段的具体调整建议

1. 共享 UI 与播放适配接口优先参考 Jellyfin：保留现有 PlayerPage 的视觉与业务，把媒体引擎命令/事件解耦。
2. 在 Windows 实验中同时验证「预留原生区域」与「原生视频下层 + 透明 Web UI 上层」，前者为基线，后者优先验证菜单、提示和渐变控件能否保留原布局。稳定合成可采用后者；未通过则按 §10 使用画面外控件。
3. 用 mpv.net/官方示例检查 HWND、全屏、输入和回收；用 Harbor 的边界问题检查缩放、层级、背景透明度及遗留画面。继续复用当前认证媒体代理与受限控制契约。
4. 保持 Electron/Vue 主架构；若考虑 Qt 原生播放器 helper，则另做范围明确的实验，评估额外运行时、UI 桥接和打包，不因案例使用 Qt 就直接迁移全部 Desktop。
5. 先参考架构和接口，自行实现小型适配。Jellyfin/Stremio/mpv.net 的 GPL 应用代码不直接混入当前 MIT 业务代码；若实际采用其源码需按对应许可处理。Harbor 的 MIT 外壳和官方示例的许可不覆盖实际分发的 libmpv/FFmpeg 构建，分发核对仍属 P5。

本节为调研与实施建议，未把第三方代码运行效果计入本项目的嵌入验收。公开源码快照保存在本地忽略的 `.workspace/native-player-research/`，本文永久记录源码链接与版本，不依赖该缓存存在。

## 12. 2026-10-05 页面融合实施（用户已授权）

用户要求按上述参考开始实施。本轮首先在独立原型入口实现 Windows 原生宿主 + 透明 Electron UI 的同窗播放，保留当前 mpv 进程/认证代理；不将 Qt/GPL 客户端源码加入项目。受控 Win32 helper 创建视频 HWND，mpv 以 wid 绘制；Electron 窗口作为宿主的透明 owned window，仅承载 Curated 控件。宿主与 UI 同步客户端区域、DPI、移动、最小化、全屏与退出。

| 设计项 | 本轮决定 |
|---|---|
| 产品面与主任务 | Desktop 播放；选择媒体后以原生引擎观看 |
| 既有参考 | PlayerPage 的标题、进度条、圆形控制按钮、音量、设置；Jellyfin/Stremio 的视频/Web UI 分层；Windows 宿主参考 mpv.net |
| 层级与密度 | 画面优先；播放时隐藏准备卡片，配置和诊断为次级入口；沿用当前黑色渐变播放浮层 |
| 状态 | 未连接/锁定、缺引擎、未选片、启动/缓冲、播放/暂停/结束/错误、保存失败；窄窗口控件换行 |
| 系统影响 | 提取可复用的播放控件，不改全局 token；正常 Web/生产 Desktop 默认播放保持原路径；独立桥接不提供任意 HWND/命令 |

验收先覆盖真实 Win32/mpv/Electron 合成、窗口移动/缩放/最小化恢复、暂停/seek/倍速/音量、菜单与文件切换、退出与认证锁定。共享控件回归当前 Web 播放页。每个可独立说明的实现与文档分别提交；进度和正式业务接入以本轮实际完成记录为准。

### 12.1 已落地的架构和 UI

- `electron/native-player-host.c` 是独立编写的 Win32 helper，只链接系统库。以父 PID 启动，创建黑色视频宿主，mpv 通过 `--wid` 绘制；不引入 Qt/GPL 应用实现。helper 只接受内部有限动作，叠加窗口必须属于启动它的 Electron PID。父进程死亡或 stdin EOF 自动退出。
- `NativePlayerWindow` 管理有界 NDJSON、请求关联、超时和清理。helper 报告物理客户端矩形/DPI，并通过 SetWindowPos 独占 owned UI 物理位置和尺寸；Electron 主进程仅同步可见性，不再重复转换 DIP 设置边界。透明层自身禁止移动/缩放，修复记录见 §12.4。全屏保留原位置与最大化状态，关闭按保存进度 → 停止 mpv/代理 → 销毁 UI/helper 顺序执行。
- 透明无框 Electron 窗口仅承载 Web 控件，使用原生 owner 关系保持窗口层级，无全局置顶。主进程禁用此入口的后台节流，避免透明窗口遮挡时控件动画/轮询停顿。helper 的初始/最小尺寸使用 DIP，适配本机高 DPI；跨显示器混合 DPI 仍需专项验收。
- renderer 继续只接受有限业务桥，新增 `windowAction(fullscreen|minimize|close)`，不暴露 HWND、任意窗口几何、通用 mpv/shell 调用。已有身份/PIN/媒体代理和逐文件进度合约保留；原生模式禁用 mpv 默认 OSC/输入，Curated 负责鼠标、键盘和控件。
- `PlayerTransportControls` 从既有 PlayerPage 提取，直接共享圆形播放/快进后退按钮、音量和静音；动作插槽保留原 Web 队列、设置、外部播放器、PiP、全屏。原型共享分段选择和设置菜单，native 菜单仅显示倍速；准备弹窗负责连接/解锁/搜索/起播。播放时标题、进度条、底栏和信息面板叠在视频上。
- 分段切换读取该文件续播，保留倍速/音量/静音；单击播放暂停，双击全屏，沿用 Space/J/K/L、方向键、M/F/Escape/D。控件 5 秒隐藏，鼠标和键盘焦点可恢复，菜单/弹窗保持交互。未连接、锁定、缺引擎、空结果、忙碌、结束重播、播放失败和保存失败均有状态。三语设置消息采用函数，保持本地 CSP，不启用 unsafe-eval。

### 12.2 本轮验证证据

| 检查 | 结果 |
|---|---|
| Windows helper 编译 | MinGW gcc 15.2；C11、Wall/Wextra/Werror 通过；仅生成忽略目录内 exe |
| Electron 测试 | 2026-10-06 复验 15 文件 / 98 项通过；包括认证 Range、逐文件/锁定/并发/回收、真实 mpv 坏媒体与控制、4 项真实宿主生命周期/嵌入测试及新增实际 Electron 透明层绑定测试 |
| 真实 Win32 宿主 | ready、尺寸、非法尺寸/外部 PID HWND 拒绝、全屏/恢复、最小化/恢复、EOF、父 PID 退出、mpv wid 绘制与 seek/恢复通过 |
| Web 共享控件回归 | 6 文件 / 40 项通过：PlayerPage progress-hover/loading/i18n/frame-markers、PlayerPlaybackSettingsMenu、PlayerView；native 菜单只显示倍速 |
| 类型、静态与独立构建 | pnpm typecheck、相关 src ESLint、native tsc/Vite 构建通过；Electron 文件未匹配仓库 ESLint 配置，以 tsc/测试验证 |
| 正式入口构建隔离 | VITE_USE_WEB_API=true 的 pnpm build 与生产 electron/tsconfig.json 编译通过，原型不进入生产 main/preload |
| 实际 Electron UI | CLI 操作连接/PIN/搜索/选片/第二片续播；暂停、1.5x、静音、进度条约 45 秒寻址、切回第一片、快捷键、全屏恢复通过；设置菜单无 Direct/HLS 虚假选项，浏览器控制台 0 错误/警告 |
| 原生/Web 合成 | 桌面客户端截图同时包含真实彩色 H.264 合成视频和 Vue 标题/分段/圆形底栏；本机 150% DPI 客户端 1478×1144，最小化/恢复后两层仍显示。页面截图仅用于 UI 辅助检查 |
| 进度身份 | fixture 的第二片从 12 秒起播，暂停 62.466667 秒/后续 seek 保存到 part-2；切换后 part-1 独立保存 44.9 秒；最终退出仍回收自己的进程 |
| 引擎观测 | 合成 H.264 使用 d3d11va；观察会话呈现/解码掉帧计数均为 0，不外推为所有问题影片已修复 |

本地 QA 证据在忽略目录 `output/playwright/native-player/`：`native-composition.png`、`native-restored.png`、`native-controls-web.png`、`fixture-controls-results.json` 与最近一轮 `fixture-results.json`。截图只用于开发检查，不能替代真实问题片源的五分钟以上对照。REQ-0058 维持 in_progress，当前 progress 80。

### 12.3 启动与剩余边界

在本分支根目录运行 `pnpm dev:native-player`。构建需本机 MinGW gcc；可通过绝对 `CURATED_NATIVE_CC` 指定编译器。`CURATED_NATIVE_MPV` 预选现有 mpv.exe，或在准备弹窗选择；Server 仍需独立运行。build-only 为 `pnpm build:native-player-prototype`，产物/profile 继续隔离，详细业务写入效果见 guide。

本轮已完成开发原型中的原生视频/Web UI 合成。正式 Desktop 路由/业务入口与播放引擎适配、萃取帧/录制/原生 PiP、音轨/字幕/HDR、混合 DPI 跨屏、长期 A/V 同步、真实 LAN/HTTPS 和问题 MP4 对照、引擎许可证/分发/安装包、macOS 仍未交付。完整 display-scaling 套件未运行；生产打包、push、合并、发布均未执行。不能将现阶段开发窗口认作正式主页面已切换原生引擎。

本轮实现提交：`014de668`（宿主/透明窗口）、`45113d70`（共享控制栏）、`73d406ed`（高 DPI/后台节流）、`d231d1e1`（Curated 原生播放 UI）。

### 12.4 2026-10-06 透明控件层独立移动/缩放修复

用户在真实启动后发现拖动窗口边缘时透明层单独变化，视频窗口没有跟随。现场 Win32 检查确认旧透明层左上/右下返回 HTTOPLEFT=13、HTBOTTOMRIGHT=17；实际透明层为 1599×816，视频客户端为 1478×1144，已发生分离。owned window 只提供归属和窗口层级，不会自动锁定几何；Electron 默认可移动/缩放导致透明层拥有独立操作能力。另有 helper 的物理 SetWindowPos 和主进程 DIP setBounds 两套几何写入，高 DPI 舍入出现 1 像素误差。

修复明确唯一几何控制方：

1. 透明 BrowserWindow 禁止移动、缩放、最大化、最小化和自行全屏；will-move/will-resize 阻止透明层自身系统操作。可交互的原生外框继续承担窗口操作，页面按钮调用宿主有限动作。
2. Win32 helper 在 attach 首次展示前、WM_MOVE/WM_SIZE/WM_DPICHANGED 时按宿主客户端物理区域同步透明层；主进程移除 screenToDipRect/setBounds，仅在页面 ready 后同步可见性。避免两套坐标往返舍入。
3. 新增 `electron/native-player-overlay.integration.test.ts` 与 Win32 PowerShell probe，运行编译后的真实原型入口，避免复制 BrowserWindow 选项后测试产生假通过。测试只操作自己创建的 helper/overlay 和临时 profile，不使用全局鼠标。PowerShell fixture 保留 UTF-8 BOM，避免 Windows PowerShell 5.1 按本机 ANSI 读取中文注释而影响脚本执行。

本轮实际验证：透明层初始及结束的八个边角均返回 HTCLIENT=1，且无 WS_THICKFRAME；视频宿主边角/标题栏仍返回 13、17、2。初始、连续四次移动/缩放、最小化恢复后的六组物理边界完全相等；透明层随宿主隐藏/恢复，正常关闭后本次 Electron 退出。完整 Electron 15 文件/98 项通过（已配置真实 mpv）；原型 C/TS/Vue 构建及生产 Electron tsc 通过，git diff --check 通过。本轮未修改共享前端控件，40 项前端回归为上轮记录。

此前真实宿主测试和 UI 合成截图证明了正常播放，但没有覆盖透明层自身的系统边角命中，因而漏掉本次独立缩放问题。新增实际 Electron 窗口检查补足这个缺口。代码提交 `efe0a108`；继续保留原型分支，REQ-0058 的真实问题片源与混合 DPI 对照仍待完成。

### 12.5 2026-10-06 左右键触发整屏主题色焦点边框修复

用户发现左右键寻址后，透明显示层出现主题色整圈边框。旧版本实际 Electron fixture 复现：铺满画面的 `data-native-video-surface` 是默认进入 Tab 顺序的 button，包含 `focus-visible:ring-2/ring-inset/ring-primary`。点击使画面层获得焦点，方向键使其匹配 focus-visible，实际 box-shadow 出现主题色 2px inset，浏览器还显示原生 outline；该边框来自 Web 控件焦点样式。

本次局部修复只修改 `NativePlayerPrototype.vue` 的画面点击层：移除整屏 ring 类，设置 outline-none 与 tabindex=-1，避免重复的 Tab 停靠点。保留 button 的语义、名称、单击/双击和现有全局快捷键；底部共享播放、音量和设置控件继续保留正常焦点样式和键盘操作，不修改全局 token 或 Button 基元。

验证记录：pnpm typecheck、该组件 ESLint、独立原型 C/TS/Vue 构建和 git diff --check 通过。修复后的真实 Electron 页面读取到画面层 tabindex=-1、box-shadow=none、outline-style=none。独立 Chromium 页面使用受限 fixture 播放桥验证右键从 40 到 50 秒、左键回 40 秒；Space 与单击切换播放/暂停；Tab 跳过画面层并到达带可见局部焦点 ring 的播放按钮，Enter 可操作该按钮；双击发送进入/退出全屏动作。独立页面验证不等同于重新完成真实引擎/DPI 验收，本次未修改原生协议或引擎代码，也未新增镜像式样式单元测试。

本地忽略目录 `output/playwright/native-player/focus-before.png` 与 `focus-after.png` 记录页面层对照；后者为独立 UI 桥验证，截图不包含原生视频，不作为同窗视频合成证据。代码提交 `0f3ae84d`。当前启动的原型已加载修复后产物。

## 13. 2026-10-06 mpv 随 Desktop 分发建议（尚未实施）

用户询问 mpv 能否与 Desktop 一起分发。结论是可以；安装包和便携 ZIP 都可包含引擎，安装后自动使用随包的绝对路径，用户无需预装 mpv 或选择 exe。当前独立 mpv 进程、标准 JSON IPC 和自有 Win32 宿主可继续沿用，随包分发不要求改为 libmpv。原型仍使用本机已有引擎，生产分发尚未接入，本节不代表已修改发布包或正式播放默认路径。

### 13.1 建议的 Windows 交付方式

首版按现有 Windows Desktop x64 交付，完整资源建议放在 `resources/app/native-player/`，包括自有 `native-player-host.exe`、`mpv/mpv.exe`、该构建所需的 DLL、许可/署名和 `engine.json`。主进程从 `process.resourcesPath` 解析固定的随包路径；开发态保留本地引擎选择。现有包没有 app.asar，该路径可直接执行；未来改为 asar 时可执行文件/DLL 必须保持解包状态。

`engine.json` 应记录引擎版本、平台/架构、上游来源、锁定的构建与依赖提交、SHA-256、构建参数和对应源码位置。来源可以是可核对的 Windows 构建，或自建构建流水线；不从开发机 Chocolatey/PATH 自动抄一个未知版本作为发布输入。构建仅使用固定版本和校验值，禁止发布时取漂移的 latest。

首版让引擎随 Desktop 批次一起安装、更新和回退，不另做播放时在线下载或自动更新器。Engine 版本单独记录，Desktop 测过再升级。引擎属于 Desktop，不放进 Server；更换引擎造成 Desktop 交付变化，按现有组件版本与发布批次政策处理。当前本机 mpv 0.41.0 的 exe 为 115,788,288 字节（110.4 MiB，未压缩）；正式包增量还取决于所选构建、DLL 和压缩方式，尚未测量安装包大小。

现有 `scripts/release/release_lib/windows_components.py` 的 Desktop 验证会拒绝任何 `third_party/` 路径，视为 Server 依赖。接入时在 `stage_desktop` 增加上述明确的 native-player 资源，并验证引擎清单、宿主和必要依赖；继续拒绝 Server 的 Go、frontend-dist、ffmpeg.exe/ffprobe.exe。mpv 内含/链接 FFmpeg 播放库不等于需要再复制 Server 的 FFmpeg 命令行工具。`resources/app/` 也属于现有安装器的托管替换范围，升级需先关闭本次 mpv/helper，避免占用可执行文件和 DLL。

### 13.2 许可结论和需要保存的材料

mpv 官方 v0.41.0 的 [Copyright](https://github.com/mpv-player/mpv/blob/v0.41.0/Copyright) 明确默认 GPL-2.0-or-later；排除全部 GPL-only 文件才可能生成 LGPL-2.1-or-later 构建，`-Dgpl=false` 本身不能保证最终许可。其说明还指出 LGPL 模式主要面向 libmpv，目前不推荐据此构建 mpv CLI。所链接的 FFmpeg 等依赖也会影响实际二进制的许可，因此首版可采用许可材料完整的 GPL mpv CLI，不能只依据名字或 Chocolatey 的 licenseUrl 判定最终许可。

GPL 允许重新分发。随二进制保留实际许可文本、版权声明、依赖材料，并提供与该份二进制完全对应的完整源码（包括适用的依赖、改动与构建脚本/参数）；建议在同一发行批次提供可下载的源码归档。只写 mpv 官网或 master 源码链接不等于提供对应源码。下载页/本地许可入口明确引擎及相关依赖各自的许可，不能将整个包统一标为 MIT。

当前进程间标准命令/状态控制与独立引擎目录有利于保持组件边界。GNU [FAQ 的聚合与通信说明](https://www.gnu.org/licenses/gpl-faq.html#MereAggregation) 说明，同介质分发不同程序不自动使所有代码改为 GPL，但是否构成一个组合程序也看通信语义，不能将“独立进程/IPC”当作一律免除许可义务。保留 Curated 自有 MIT 代码的权利声明与 mpv 各自声明；未来若直接链接 libmpv，需重新核对所用构建和链接方式。GNU [安装器说明](https://www.gnu.org/licenses/gpl-faq.html#GPLCompatInstaller) 明确，单纯安装 GPL 程序的安装器不因此必须采用 GPL。

核对来源：[mpv Windows 下载/构建入口](https://mpv.io/installation/)（多数是第三方构建，官方 CI 主要用于测试）、[mpv v0.41.0 Copyright](https://github.com/mpv-player/mpv/blob/v0.41.0/Copyright)、[FFmpeg 许可及源码说明](https://ffmpeg.org/legal.html)、GNU 上述 FAQ，以及仓库当前 Windows `stage_desktop` / `validate_payload` 和本机引擎版本/体积。尚未选择正式随包构建，也未完成其对应源码核对。

### 13.3 实施与验收顺序

1. 选定 Windows x64 引擎构建、依赖、固定校验值及完整对应源码；补充本地许可材料和 About 中的引擎条目。
2. 增加受限的随包路径解析和版本清单；缺失/损坏引擎给出明确状态，原型本地选择仍可用于开发。
3. 将宿主与引擎纳入 Desktop installer/ZIP staging 和 payload 校验，不自动改变正式 PlayerPage 引擎选择。正式播放接入继续按 P4 完成。
4. 在未安装 mpv、未配置 PATH 的干净 Windows 上验收安装/便携包离线起播、空格/中文路径、H.264/HEVC 硬解、退出回收、升级替换与回退。核对安装包增量和许可/源码下载。

本节记录初始分发方案。2026-10-06 后续已下载并校验固定开发引擎、实现可选 staging 与资料清单校验，见 §14.9；没有准备完整可分发引擎资料、原生安装包或发布。

## 14. 2026-10-06 正式 Desktop 接入实施计划（用户已授权）

本节状态：in-progress。2026-10-06 用户授权按本节实施，继续使用专用分支。REQ-0058 保留原型验收；正式接入跟踪为 REQ-0059。

本节最初回应后续集成问题，细化 P4/P5；随后用户明确要求按计划实施。当前已在专用分支改动正式 Desktop 和对应 Server 前端入口，默认偏好仍 Web，实际完成情况以 §14.9 为准。§10/§11 保留早期研究记录。

实施设计框架：播放窗口保持画面优先与低干扰，共享圆形控制栏、分部与倍速；设置沿用现有纵向卡片，增加本机原生偏好；侧栏沿用当前后台播放卡片。加载、锁定、引擎不可用、播放错误与 Web 切换均有明确路径；不会添加另一套全库搜索或连接准备 UI。主页面内嵌为 M5，首版使用专用窗口，不新增全局主题规则。

以下 14.1–14.8 保留设计与完成条件；其中“接入时/建议”等描述为设计目标，逐项源码与证据见 14.9。M1–M3 已接入，M4 只有准备/校验支持，不能将源码接入等同于首个正式包验收完成。

### 14.1 首个产品形态与用户流程

建议首版以 **Desktop 管理的专用 Curated 播放窗口** 接入现有产品：用户在现有列表、详情、历史、分部选择或萃取帧入口点击播放后，直接打开已经验证的原生视频 + Curated 控件窗口。准备弹窗里的 Server 地址、PIN、搜索和选择 mpv.exe 不再属于正式观看流程；Desktop 使用当前连接、当前登录会话和随包引擎，直接传入所选 movieId/fileId/续播位置。

主窗口继续承担浏览、管理和来源导航。常规点击原生播放后保留来源业务页；播放器窗口关闭时保存并停止播放，恢复/聚焦来源主窗口。重复打开同一目标聚焦已有播放器；换作品或分部先按当前身份保存/停止，再复用播放窗口启动新目标。直接访问 /player 深链也必须经同一选择/启动逻辑；不存在来源业务页时，主窗口可落在所选影片详情，保留历史/萃取等返回上下文。

第一阶段保留当前原生标题栏和已修复的两层几何关系。原生宿主独占物理位置与尺寸，透明层只承载页面控件，不能独立拖动/缩放。取消“原型”产品文案，统一 Desktop 的图标、语言、主题、音量与倍速偏好；继续复用圆形控件、进度条和文件选择。原型准备 UI 可留在开发入口用于诊断，不带入正式普通播放流程。

原生播放器以独立窗口存在时，允许主窗口继续浏览；它不是浏览器 HTMLVideoElement PiP。主窗口关闭到托盘与退出整个 Desktop 继续区分：关闭主窗口不自动误杀仍打开的播放窗口，托盘“退出”、系统会话结束、认证锁定和成功切换 Server 要停止本次播放并回收自己的 mpv/helper。最小化播放窗口保留普通播放器语义，恢复后两层同步；跨路由是否保留实例由实际原生窗口状态决定，不能假装 pipActive=true。

### 14.2 当前代码接入点与待改内容

| 当前接入点 | 当前行为 | 接入时的修改与验收 |
|---|---|---|
| electron/main.ts、preload.cjs | 正式 Desktop 加载当前 Server 的页面，维护 Server 身份/partition、主窗口和托盘；尚无原生播放桥 | 新增受限的原生播放能力、打开/控制/聚焦/关闭和状态订阅；正式 main 管理 native 生命周期，不导入原型的全局 app 初始化与 quit handler |
| native-mpv-player.ts、native-media-proxy.ts、native-player-window.ts | 原型已验证的 mpv/代理/宿主实现 | 提取或复用为生产服务模块，统一资源路径、受控会话和退出顺序；保留独立原型入口以便开发对照 |
| native-player-lab.ts | 拥有独立原型 session、连接/检索/起播与定期进度写入 | 生产适配移除第二套连接/PIN和 exe 选择流程；注入正式当前 Session/Server 上下文，明确唯一写入方，避免复制出另一套播放记录逻辑 |
| src/lib/navigation-intent.ts、PlayerView.vue | 播放意图含 fileId/t/autoplay 和来源；PlayerView 在挂载时 prefetch，可能预先启动 HLS | 在意图/路由入口先完成引擎选择，选择 native 时不挂载 Web 播放器、不运行 Web/HLS 预热；深链、直接访问、同片 seek 与普通点击均走同一规则 |
| use-playback-host.ts、ActivePlaybackHost.vue | 壳层拥有一个 PlayerPage；仅浏览器 PiP 时允许离页保留；控制依赖 PlayerPage 实例 | 增加引擎/展示形态和异步生命周期，native 绑定远程会话 facade，Web 仍绑定现有播放器；窗口实例与浏览器 PiP 分开建模，停止/换片必须等待当前实例回收 |
| use-active-playback-session.ts、SidebarPlaybackEntry.vue | 快照在当前 Vue 运行时；侧栏实时控制依赖 pipActive | 接收 native 状态快照，显示实际文件、进度、暂停/继续/停止及“返回播放器”；聚焦已有窗口不重新 seek 到旧 t，不新建第二个引擎 |
| PlayerPage 与 NativePlayerPrototype.vue | 已共享 transport/分部/倍速菜单；布局和其余业务仍各自实现 | 按需继续提取标题、进度/帧标记、播放列表和通用状态；正式 native 页面接受会话，不承担连接/全库检索；避免复制整份 PlayerPage |
| Windows stage_desktop / payload 检查 | 尚未包含 native 宿主/mpv；将 third_party 全部视为 Server 依赖 | 按 §13 的明确 native-player 路径纳入 Desktop staging/installer/ZIP，固定版本与来源；Mac 构建不调用 Win32 gcc、也不打入 Windows exe |

本表为实施前接入点。当前已新增 `NativePlaybackCoordinator`、Desktop playback contract、正式 main bridge 与 Server UI 可选适配，复用底层 mpv/代理/窗口模块；实现映射见 §14.9。

### 14.3 引擎选择和兼容策略

首版在 Windows Desktop 提供本机偏好“优先使用桌面原生播放器”，先以试用开关接入。该偏好保存在 Desktop 本机 profile，不写 library-config.cfg，也不改变其它设备的播放偏好。能力探测至少包含平台、引擎/宿主就绪、协议版本和功能能力。浏览器、尚未适配的 macOS、旧 Desktop、旧 Server UI 或缺少引擎时继续使用现有 Web direct/HLS 流程。

选择 native 必须发生在请求可能启动 HLS 的 descriptor 之前；HTTP 续播请求显式使用 direct 语义，保留每片进度和“从此帧播放”的绝对时间。普通 native 起播失败时允许在当前有效时间切换到 Web，先停止当前 native 会话再启动 Web，防止两路音视频和重复写入；不在错误恢复中循环切换引擎。401/403 或锁定状态回到现有解锁流程，不把认证失败当作绕过锁定的回退触发条件。

当真实问题 MP4、常用格式/硬解、长时间 A/V 同步、混合 DPI 和完整业务通过验收后，再评估 Windows 新安装默认优先 native；不在原型阶段自动修改已有用户设置。测试/开发可以显式选引擎做对照。

### 14.4 正式认证、IPC 和会话边界

正式播放复用 **当前主窗口实际使用的 Electron Session**（或经过明确绑定的同一 partition），不继续用原型 `-native-prototype` 分区。main.ts 的 connectServer 会因服务器身份改变使用不同 partition，因此不能只重新根据 URL 计算一个 session 然后假设已登录。HTTP 媒体继续通过认证代理转发 Range；Cookie/PIN 不进入 mpv argv 或 renderer。

Server UI 的播放请求只传影片/文件身份、起点、autoplay 和必要的受限来源上下文；Server URL、Session、exe 路径、HWND 和引擎参数由 main 决定。沿用正式可信主 frame 的 sender 校验，候选连接窗口在尚未提交为当前 Server 时不能启动播放。本地 native 控制页使用独立限定 preload，只接受本次 sessionId 的枚举动作；不把原型 connect/search/任意源切换接口整体暴露给远程 Server 页面。

会话身份包含实际 Server 身份/连接代次、movieId、fileId、engine 和独立 sessionId。状态订阅、命令、进度保存、切片和结束回调均检查该身份，拒绝旧窗口/旧 Server 的迟到事件。首次启动创建一个播放窗口与一个引擎会话，重复点击同片聚焦；换片串行，始终只有一个本机活动播放目标。控制 UI 崩溃、helper/mpv 退出都反馈到主窗口并回收本次资源。

切换 Server 时保持现有“先探测/加载候选、成功后替换当前连接”的规则：取消或探测失败不误停原播放；在成功提交切换前按旧身份保存/停止 native，随后清除旧状态、关闭窗口并替换主页面。锁定/媒体 401/403 停止播放，保留既有登录与解锁 UX。退出清理有期限，网络保存失败仍完成本次引擎/代理/宿主回收。

### 14.5 进度、历史、观看时长和播放列表

native 模式由 Desktop 播放协调服务负责持久化进度/观看时长，主窗口和播放窗口只接收同一状态；Web 模式继续由现有播放器写入。停止旧引擎、确认写入责任释放后才启动另一引擎；不能把原型周期写入与 PlayerPage 保存同时打开。保留 Server 现有 fileId/绝对时间合约，不新增每种播放器各一套历史记录。

观看时长复用现有“实际播放墙钟时间”的计算语义，抽离纯计算并注入 HTTP sink，不直接从 Electron main 导入包含浏览器存储依赖的 src 模块。暂停、缓冲/停止、seek、倍速、跨日及窗口最小化分别处理；seek 跳过的媒体秒数不计观看时长，倍速不简单按媒体推进量累计。main 持有的原生会话在主页面不显示时仍能记录实际播放。

分部切换继续读取该片续播并保留音量/静音/倍速；“从萃取帧播放”恢复同一 fileId 与绝对时间。播放队列携带来源快照和普通影片/FC2 类别，不被主窗口后来浏览的页面改写。自然结束按既有连播策略处理；手动停止、错误、锁定和关闭不触发自动播放下一片。

### 14.6 UI 复用与功能能力

| 功能 | 首个正式 native 接入目标 | 后续处理 |
|---|---|---|
| 标题、文件选择、圆形控件、进度、倍速、音量/静音、快捷键、窗口全屏 | 保留当前已共享/验证的 UI，去除准备弹窗 | 继续统一通用布局，局部焦点提示保留在真实控件上 |
| 历史、逐文件续播、观看时长、队列/连播、侧栏控制 | 首个正式业务闭环必需 | 验证多来源/切片、窗口后台、页面重载后的状态恢复 |
| 源帧/片段萃取 | 使用现有 Server 合约及 fileId/绝对时间，可分增量交付 | 将现有萃取 UI 接到引擎能力；即时当前画面截图/逐帧要单独适配 native |
| 桌面小窗 | 专用普通播放窗口先可用 | 紧凑置顶/PiP、窗口恢复与焦点另行实现；不调用浏览器 video PiP 冒充 native 小窗 |
| 音轨/字幕/HDR/画面设置 | 按实际引擎能力显示 | 增加 mpv 属性与控制适配，分别验收，不保留无效按钮 |
| 主窗口内容区域内直接显示 native 画面 | 保留为后续嵌入实验 | 通过局部透明/合成、裁切、侧栏/菜单遮挡、页面滚动、DPI 和焦点验收后再接同一会话 |

原生首版可以在有限功能下明确提供 Web 切换，但生产功能入口不能外观存在、点击无效果。业务能力通过 `PlaybackCapabilities` 显示，与 engine 选择分离，避免当前 native 菜单出现无效 Direct/HLS 或 HTML PiP 动作。

### 14.7 实施里程碑与完成条件

| 顺序 | 可 review/提交的实现单元 | 完成条件 |
|---|---|---|
| M1 | 生产会话协调、限定 bridge、正式 Session 复用、资源路径与生命周期 | 从正式 Desktop 主窗口发受限请求可起播；无需第二次 PIN；旧会话事件不会串入新片；停止/退出可等待并回收 |
| M2 | 统一播放意图、偏好/能力选择、PlayerView/host 接入和原型准备 UI 移除 | 列表/详情/历史/分部/帧入口直接打开原生窗口；native 不额外预热 HLS；重复打开聚焦已有窗口；Web/旧客户端仍可播 |
| M3 | 单一进度写入、观看时长、主窗口/侧栏同步、队列与切片 | fileId/时间/统计正确；主窗口继续浏览时可暂停/停止/返回；换片/连播、锁定/切源/崩溃/关闭均不残留画面或音频 |
| M4 | 随包 mpv/helper、许可/源码、Windows installer/ZIP 与兼容批次 | 无 mpv/PATH 的干净 Windows 安装即播；中文/空格路径、升级/回退/卸载通过；Server/旧 Desktop 兼容有实际检查 |
| M5 | 功能增量与主页面嵌入实验 | 源帧/片段、小窗、轨道/HDR 逐项独立验收；主页面合成通过前持续提供专用播放窗口 |

Windows 问题片源对照和混合 DPI 的 P3 验收可与 M1–M3 开发并行，但在正式默认优先 native 或发布前必须完成。M1–M4 与核心播放/业务/安装验收构成首个可交付闭环；每个最小单元提交并记录证据，当前原型分支继续隔离。不靠“exe 已复制进安装器”判定接入完成。

### 14.8 Desktop / Server 发布兼容矩阵

正式 Desktop 当前加载 Server 提供的 Web UI。让原有播放按钮调用新增 Desktop bridge，通常也需要发布 Server 内的前端入口/能力检测更新；mpv/helper 和本地播放 UI 属于 Desktop，媒体/进度/统计仍通过 Server 现有 HTTP 合约。本阶段不以增加公共业务 API 或 library-config 字段为前提。

| 组合 | 预期行为 |
|---|---|
| 新 Windows Desktop + 支持 native 入口的新 Server UI | 可选原生专用窗口，复用当前登录/资料库；Web 回退可用 |
| 新 Desktop + 旧 Server UI | 原有 Web 播放仍工作；不能假设旧页面会自动调用新 bridge |
| 旧 Desktop/浏览器 + 新 Server UI | 可选桥检测失败后继续 Web 播放，不访问不存在的 IPC |
| macOS Desktop + 新 Server UI | Windows native 能力明确不可用，使用当前 Web 路径，打包不执行 Windows 宿主构建 |

Server 前端与 Desktop 交付若都实际改变，按现有 Curated 日期批次独立准备两个组件版本并可同批发布；不把 mpv 纳入 Server 包。当前已在隔离分支实施正式可选入口与打包准备代码；没有版本递进、push、合并或发布。新旧组合中的实际网络/安装验收仍需完成。


### 14.9 2026-10-06 实施记录与验收边界

本轮正式接入需求 REQ-0059 保持 `in_progress`（70），原型 REQ-0058 保持 `in_progress`（80）。M1–M3 有本地源码和自动/合成样本证据；M4 尚无完整可分发引擎资料和安装验收，整项需求不标 verified/released。专用分支保持隔离，未修改主工作区、library-config.cfg、版本或发布渠道。

| 单元 | 已实现源码/行为 | 验证与剩余条件 |
|---|---|---|
| M1 会话与窗口 | `playback-coordinator.ts` 为唯一原生会话/写入所有者；`playback-desktop.ts` 复用当前主窗口实际 Session，限定主 frame 和本地控制页；独立 sessionId/连接代次，串行替换、停止/退出；复用原生 helper 几何权威 | 实际正式 Electron main 用 HttpOnly Cookie 播放受保护第二片，无二次 PIN；旧命令拒绝、替换/EOF/锁定/回收自动测试通过；真实切 Server/崩溃/网络验收待完成 |
| M2 入口与 UI | router 在 PlayerView/HLS prefetch 前检测 bridge/本机偏好；列表/详情/历史/分部/帧意图走同一入口，主窗口保留来源；本地 `DesktopPlayer.vue` 共享圆形 transport、分部/倍速菜单；Web 显式回退 | 深链与入口适配、无能力/Web 回退/锁定路由自动测试通过；同目标重新打开在实际 40 秒位置保持同一会话、不回退至旧 12 秒；实际透明画面层 CSS 为 outline none/boxShadow none |
| M3 业务同步 | direct descriptor、逐文件进度/历史/每日观看时长；冻结队列与受限来源上下文，刷新可恢复；分部/下一作品连播；侧栏暂停/继续/停止/聚焦；volume/speed 本机保存 | 实际 PUT query/body 均为 part-2；d3d11va 合成 H.264 观察到零掉帧；四次实际日时长上报为 4.902/5.040/4.990/4.011 秒；main 刷新/侧栏命令/Web 来源恢复测试通过 |
| M4 分发准备 | `native-player:prepare` 固定 URL/archive/exe SHA，开发资源放 ignored `.workspace/native-player/`；Windows staging 可读取绝对 `CURATED_NATIVE_BUNDLE`，校验 runtime/完整对应源码/构建材料/许可/组件清单和 SHA，按清单复制；未配置则原包继续 Web | 21 项打包测试通过；上游 20261005 Release 仅有运行时/开发二进制资产，所下载包未包含完整对应源码与许可集合；固定 QA pin `distributionReady=false`，不能直接用于正式分发；未制作或验收原生 installer/ZIP |

正式窗口提供标题、进度、分部、圆形播放/快进退、倍速、音量/静音、前后作品、连播、全屏、诊断、关闭和 Web 切换；准备地址/PIN/搜索/exe 选择留在独立原型入口。即时截图/逐帧、源帧/片段 UI、原生 PiP、音轨/字幕/HDR 和主页面内嵌尚未接入本地原生页，相关功能可切回现有 Web 使用。当前前后作品为冻结 ID 队列按钮，尚无完整标题列表面板。

原生窗口继续采用 helper 的物理 SetWindowPos，透明层不可独立拖动/缩放。主窗口关闭到托盘不会关闭正在播放的专用窗口；播放器关闭停止并保存，恢复主窗口。锁定（状态 unlocked=false、媒体或认证 HTTP 401/403）停止并关闭原生资源，不触发 Web 绕过锁定。实际合成 Server 锁定后发布 SERVER_LOCKED/windowOpen=false，自己的 mpv/helper 已回收。

共享观看时长纯计算已修复本地午夜拆分和失败批次重试，重试不会再次累计已确认日块。mpv 逐帧 time-pos 的 30/60fps 推进不能用原 0.05 秒阈值过滤，现计入任何有效正向推进，仍忽略 seek、暂停及不推进样本。相关 Web 回归通过，实际原生每 5 秒批次有正常秒数。

#### 本轮验证

- `CURATED_NATIVE_MPV=<已校验 QA mpv> pnpm test:electron`：17 文件、109 测试通过，包含真实 mpv/Win32 host/实际透明叠层及新增正式 coordinator/bridge/统计测试。
- `pnpm build`（含 vue-tsc 与 bundle 监管）、`pnpm build:electron:main`、生产 Electron tsc 和改动前端局部 ESLint 通过；入口服务测试新增页面刷新/侧栏命令/旧快照/Web 来源恢复验证通过。
- 相关前端播放器、auth guard、设置、locale、watch tracker 和 Desktop 入口定向回归均通过；这些批次重叠，不累加成独立总数。
- 最新独立前端回归批次（PlayerView、Desktop service、locale、ActivePlaybackHost、watch tracker、Playback settings）共 6 文件、52 测试通过。
- `python -m unittest scripts.release.tests.test_native_player scripts.release.tests.test_component_cd`：21 测试通过。
- 真实正式 main 采用隔离临时 profile 与本机合成 HTTP fixture；实际硬解/第二片/进度/时长/重复聚焦/锁定清理已观察。该 fixture 没有替代真实问题 MP4、实际资料库、LAN/HTTPS 或安装环境验收。
- 用户随后提供的问题 CD1 五分钟对照见 §14.10；未运行 `test:display`、混合 DPI 跨屏、整片长播、历史安装器/升级/卸载或 macOS 构建。

#### 继续完成首个交付闭环

1. 为选择的 Windows 引擎构建获得精确依赖版本、补丁、构建输入、完整对应源码和许可集合，形成能通过 `engine-manifest.json` 校验的分发目录；若上游无法证明该二进制的完整对应资料，改用能保留这些材料的自有固定构建。
2. 通过 `CURATED_NATIVE_BUNDLE` 组装原生 Desktop installer/ZIP，在未装 mpv/PATH 的干净 Windows 验证安装即播、中文/空格路径、退出、升级/回退/卸载及旧 Server UI 兼容。
3. 在 CD1 对照基础上，对更多 MP4、常用 HEVC、长时 A/V 同步、真实 LAN/HTTPS、成功/失败 Server 切换、渲染器/引擎崩溃及混合 DPI 做实际验收并记录证据，然后决定 Windows 默认偏好与发布批次。
4. M5 的萃取/PiP/轨道/HDR/主页面嵌入按能力独立增量；嵌入实验通过前专用窗口持续可用。

正式启动与可选分发目录格式见 guide 的 Windows Desktop native playback integration。源码入口与 Server 前端需要配套更新；已安装旧包不会因本分支代码自动获得能力。

分发来源复核：已读取所选 [上游构建](https://github.com/shinchiro/mpv-winbuild-cmake/actions/runs/37245752254) 的 x86_64 日志 artifact，以及固定 builder commit `05a60b3cfd04e3e3b89918f4a27f3dde2935dff2` 的 mpv/FFmpeg recipe 和 GCC workflow。FFmpeg 配置日志明确为 GPL v3 or later；recipe 依赖许多静态库，工作流用 latest 容器及仓库/工具链缓存。日志和顶层两个源码仓库不能代替所有精确依赖、补丁、构建材料与完整对应源码；未据此把 QA 二进制标成 distributionReady。

其它完成细节：媒体先以暂停加载，在 main 应用本机 volume/speed 后按 autoplay 恢复，避免起播瞬间使用默认声音/速度；设置页仅修改 preferNative，不能用旧快照覆盖播放中保存的音量/倍速；成功提交 Server 切换的停止阶段与退出阶段拒绝新的主窗口起播请求。Win32 helper 嵌入现有 Curated Desktop 图标，构建还需 gcc 同目录的 MinGW windres；产物不依赖运行时资源编译器。


### 14.10 用户指定 FC2-4985807-CD1 的实际对照（2026-10-06）

用户提供 `fc2-4985807-cd1` 后，已在当前开发 Server 查到作品 `fc2-4985807`、文件 `fc2-4985807:primary`（CD1）。源文件约 2.32 GB、50 分 31 秒：H.264 High 1920×1080、yuv420p、30000/1001 fps，AAC-LC 音频。使用正式 Electron main、隔离临时 profile、受限 native bridge，实际原始媒体经测试 HTTP relay 和生产认证媒体代理流式读取；没有整片下载/读入内存，业务记录写入隔离 sink，正式历史/进度没有改动。

先 native 后 Chromium HTMLVideoElement direct，各从 60 秒开始连续观察 300 秒；浏览器页可见、readyState=4，测试都静音。没有使用正式 PlayerPage/HLS；这是同机顺序诊断，不是严格受控性能基准。两套引擎的掉帧计数口径不同，不能将其当作完全相同的指标或直接推导具体 Chromium 内部原因。

| 检查 | 结果 |
|---|---|
| native 连续播放 | 300.010 秒，时间到 359.792767；mpv d3d11va；frame-drop-count=0，decoder-frame-drop-count=0 |
| native 暂停与远 seek | seek 至 1200 秒，结果 1199.9988，保持 paused，1 秒内验证到目标 |
| native 日观看时长 | 隔离 sink 合计 299.927 秒，seek 跳过的时间未计入 |
| native A/V 诊断 | 最后约 110 秒间隔抽样 11 次，avsync 最大绝对值 0.000008 秒；这仅为引擎属性，不代替主观听辨或整片长期同步验收 |
| Chromium direct | 300.0007 秒，时间到 359.926428；totalVideoFrames=9037，droppedVideoFrames=2498（27.6419%）；首个样本仅掉 6 帧，之后持续累计，结束时仍 readyState=4 |
| 生命周期 | native 转 Web 前关闭原生资源；比较完成后暂停 Web 并关闭本次临时 QA 实例；正式 Desktop/Server 未退出 |

该样本在当前机器上复现 Chromium 直放掉帧，native 同片连续播放没有观察到掉帧，支持目前专用窗口方案对这一问题片的改善；不能泛化为所有 MP4/电脑/网络已修复，也不能只据计数判定是 codec、硬解还是合成器的具体缺陷。采样结果摘要保存为 `docs/ops/2026-10-06-windows-native-playback-qa.json`；本机原始日志在 ignored `output/playwright/desktop-player/problem-file-results.json` 和 `problem-file-avsync.json`。本轮没有生成媒体截图/图像。

REQ-0058 保留 in_progress 80（混合 DPI、更多真实场景等仍未通过）；REQ-0059 保留 in_progress 70（M4 分发资料/安装验收未完成）。其它格式/HEVC、真实 LAN/HTTPS、混合 DPI、50 分钟整片主观同步与成功/失败 Server 切换/崩溃仍待验收。

### 14.11 测试运行改用已编译页面（2026-10-06）

用户确认日常测试不需要热更新。`pnpm desktop:test:build` 一次构建 Web API 页面到 `.workspace/desktop-test-ui` 和 Electron；`pnpm desktop:test` 直接运行产物，继续使用独立测试 profile。Electron 内轻量 HTTP 托管复用 `5183`，静态资源不实时编译、不监听源码；同源 `/api` 以流转发到既有开发 Server `8080`，保留 Cookie/上传/Range/SSE。不替换共享 Server 的页面、不另起 Vite/Node 编译进程。正式包及远端页面来源保留。

旧本机启动入口转到新脚本，旧 Vite 启动入口停用。源码修改后先停止原生播放并完全退出测试 Desktop，再构建/启动；Windows 无法覆盖运行中的宿主 EXE。关闭窗口仍是隐藏到托盘。监听冲突明确报错。20 项定向测试覆盖真实静态响应、SPA 深链接、缓存/MIME、缺失资源、认证/上传/Range 流代理、跨来源/路径越界、上游失败与退出释放，以及原有前端选择和主窗口生命周期；前端类型检查、生产构建和 Electron 构建通过。

实际测试窗口加载 `/assets/index-*.js`，没有 Vite 客户端或 `/src/main.ts`；首页和影片页刷新后挂载正常，`/api/auth/status` 和 `/api/health` 返回 200，原生 capability available，既有 preferNative/volume/speed 保留。本轮没有开始真实影片或改写播放进度。旧 Vite PID 31044 已结束；5183 由测试 Electron 自身持有。新主进程物理内存约 150 MB，5.04 秒采样使用 0.078 秒 CPU；此前独立 Vite 约 680 MB 物理内存、5 秒约 12.4 秒 CPU。这是短时诊断，不是整套 Desktop 内存/播放基准。

### 14.12 原生播放器 UI 与窗口控制统一（2026-10-06）

历史记录：本节自定义窗口标题栏及三个窗口按钮已按用户后续决定由 §14.13 取代；分部选择和底部按钮样式继续保留。下述验证与拖动限制只描述当时的自绘标题栏版本。

按用户要求，正式 `DesktopPlayer` 的分部选择使用 `surface-muted` 实色背景；底部所有图标动作均为圆形，自动连播、诊断、静音、全屏开启及设置菜单展开时使用 `primary` 18% 弱填充。顶部将最小化、最大化/还原、关闭纳入 Curated UI；关闭为 `danger` 红色叉号和 15% 透明红底，悬停 25%，保留焦点与三语标签。独立原生页在 html 设置深色主题，确保 body portal 的分部菜单沿用相同背景。共享 Web 基元和全局 token 不改。

正式 helper 使用可选 `--custom-chrome`，隐藏系统标题栏，同时保留原生缩放边缘与系统窗口操作。顶部 pointerdown 和双击发送受限 drag/maximize 动作到自己创建的 Win32 宿主；透明 Electron 仍不可独立移动/缩放。宿主报告 maximized；聚焦只恢复最小化窗口，不再无故取消最大化。窗口状态变化才额外发布快照，普通移动/尺寸报告不连续重复发布状态。独立旧原型继续使用原有系统标题栏。

验证：Electron 18 文件/117 测试通过（含此轮新增的真实 custom-chrome 最大化、最小化恢复、重新聚焦保持最大化、全屏恢复及受限动作/状态通知测试）；相关前端 4 文件/24 测试、前端类型检查、局部 ESLint 与 Electron/播放页构建通过。使用正式 main、隔离认证合成片源观察到实色分部选择、圆形按钮/选中底色、红色关闭按钮以及唯一自绘顶部栏；实际 UI 最大化/还原/最小化/侧栏聚焦流程通过。原始本机结果在 ignored `output/playwright/desktop-player/native-chrome-results.json`。

最后检查了 body portal：分部菜单在深色 html 下为语义深色表面，设置菜单展开时触发按钮为 primary 18% 圆形填充。合成片源 UI 截图保存为 ignored `output/playwright/desktop-player/native-chrome-preview.png`；此轮没有截取用户影片画面。临时合成 QA Desktop/宿主/mpv 已有序退出，用户日常测试 Desktop 保留运行并可加载新版播放器。

实际鼠标拖动的自动验收未完成：Windows 窗口工具将视频宿主与不同进程的透明控件识别为两个输入目标，刷新后再次尝试仍报 `point ... is over electron.exe ... not target window native-player-host.exe`，未执行拖动。此项需在测试窗口手动确认；不能把此轮截图或宿主命令测试当成真实拖动验收。未运行完整 display-scaling、跨屏混合 DPI 或新增分发验收。

### 14.13 恢复 Windows 系统标题栏，保留播放器 UI 样式（2026-10-06）

按用户后续要求，正式播放器恢复 Windows 系统标题栏及原生最小化、最大化/还原、关闭按钮，移除播放器顶部的三个自定义按钮。取消 helper 的 `--custom-chrome` 分支与自定义非客户区绘制；renderer 不再提供 drag/maximize 命令和相关文案。标题栏拖动/双击、缩放和窗口按钮由原生宿主管理，透明 Electron 仍仅覆盖宿主客户区，不独立移动。保留此前的 maximized 状态通知与聚焦保持最大化行为。

§14.12 的分部选择 `surface-muted` 实色背景、底部图标圆形样式、开启/菜单展开时的 `primary` 18% 弱填充，以及本地页面和 body portal 深色主题均继续保留。顶部恢复影片标题/番号与分部选择的紧凑布局。系统关闭按钮由 Windows 绘制，播放器内没有重复的红色关闭按钮。

验证：`pnpm build:electron:main`、原型 helper 编译、`pnpm typecheck`、播放器和文案的局部 ESLint 通过；设置实际 mpv 后 `pnpm test:electron` 18 文件/117 项通过，含系统标题栏宿主的最大化、聚焦、最小化恢复和全屏恢复。正式 main 配合隔离 profile/认证合成片源，观察到系统标题栏的三个原生按钮，renderer 内对应按钮计数为 0。实际水平拖动系统标题栏后宿主位置改变，视频与透明控件仍同窗对齐；原生最大化、最小化及主窗口聚焦恢复后保持最大化通过。窗口工具对拖动终点落在 Electron 客户区的尝试拒绝输入，改为标题栏内水平拖动完成实际操作；此结果不覆盖跨屏混合 DPI。

实测分部触发器背景为 `rgb(18, 24, 39)`，展开菜单为深色语义表面；自动连播开启和设置菜单展开时仍为圆形 `primary` 18% 弱底。点击系统关闭按钮后快照为 `windowOpen=false/status=stopped`，自己的 helper/mpv 已回收。新预览为 ignored `output/playwright/desktop-player/native-system-chrome-preview.jpg`，仅包含合成测试片源。临时 QA 实例已退出，日常隔离测试 Desktop 已重启到本轮构建，保持原生偏好/音量/倍速；没有重新播放真实影片。未运行 display-scaling、安装/发布或真实 LAN/HTTPS 验收。

### 14.14 窗口标题使用当前影片名称（2026-10-06）

历史记录：本节影片名称标题已按用户后续要求改为仅显示当前影片番号，见 §14.14.1。

按用户要求，正式原生播放窗口标题直接使用当前影片名称，缺少名称时依次回退到番号、Curated，不附加固定产品前后缀。协调器在每次起播、切片及队列切换时设置原生宿主与透明控件窗口标题；HTML 的固定页面标题不能覆盖窗口标题。名称来自已获取的影片详情，不新增 renderer 原生动作或 HTTP API。

helper 通过内部 `title` 命令调用 `SetWindowTextW`。标题以 UTF-16 单元的十六进制字段传递，避免中文/日文、引号及反斜线受 JSON 转义影响；去除控制字符并限制为 480 个 UTF-16 单元，使消息保持在现有 2048 字节管道上限内。

验证：Electron/播放器/helper 构建及 18 文件/117 项 Electron 测试通过。测试 Desktop 已重启；此前打开的 STAR-684 在保存的位置约 3:16 恢复为暂停，读取真实 Win32 窗口列表确认宿主标题与影片详情中的日文名称完全一致。此轮未截取真实影片画面。

#### 14.14.1 窗口标题改为当前影片番号（2026-10-06）

按用户后续要求，正式原生播放器的 Windows 窗口标题改为仅显示当前影片番号，不附加影片名称、分部文件名或产品前后缀。使用详情中已获取的 code，去除首尾空白；缺失番号时回退 Curated。沿用协调器每次起播、切片和队列切换的设置流程，同时更新原生宿主与透明控件窗口；播放器页面内的影片名称展示保持原样。不新增 IPC 或 HTTP 端点。

验证：既有协调器 27 项测试及 desktop:test:build（含 Electron TypeScript 编译）通过。正式 main、隔离 profile 与认证合成片源打开原生窗口后，按本次 Electron 父 PID 筛选 native-player-host 并读取真实 Win32 MainWindowTitle，确认为 TEST-001，未显示合成片源的影片名称。合成验证实例已退出，日常测试 Desktop 已重启；重启前没有活动原生播放，不自动起播旧影片。未截图/萃取用户影片，未改版本、push、合并或发布。

### 14.15 正式原生窗口补齐静态萃取帧（2026-10-06）

用户指出原生播放器缺少萃取帧后，本次补齐静态图片入库。底部增加符合既有圆形控件样式的相机按钮，默认快捷键 C；打开/聚焦播放器时从可信主页面读取已配置的萃取键与快门音效偏好。忽略修饰键、重复 keydown、输入框/滑条及原图预览中的快捷键，不引入长按片段行为。沿用中文/英文/日文反馈。

主进程串行验证当前 session 和认证，再由 mpv 临时暂停画面，读取准确 `time-pos`，执行 `screenshot-to-file <本次临时 PNG> video`，恢复原来的暂停/播放状态。截图只包含视频，无系统标题栏、Curated 控件或字幕，保持源视频尺寸。临时目录由 mkdtemp 生成，不接受 renderer 路径；PNG 验证 12 MiB 与既有像素上限，读取后删除自己的文件/空目录。截图编码有独立 10 秒期限，超时先结束本次引擎，防止迟到写入留存文件。

`DesktopPlayerBridge` 新增受限 `capture(sessionId,retryId?)`、`capturePreferences()`，主窗口播放 bridge 不增加任意截图/文件能力。`NativePlaybackCoordinator` 冻结影片、fileId、标题、演员、真实帧时间、采集时间和 UUID，通过当前主窗口真实 Session 以 multipart 写既有 `POST /api/curated-frames`。不新增 Server HTTP 端点或 library-config 字段。最多一个采集/上传任务和一张失败缓存；重复请求拒绝为 CAPTURE_BUSY。上传失败保留原图，同 ID/同 metadata/同图重试以兼容 Server 幂等回执；成功后释放主进程缓存并递增 captureRevision。401/403 停止播放，切片/换片/退出清理旧缓存，旧会话完成结果不显示到新影片。

UI 复用 `CaptureReceipt` 与 `FrameImageViewer`：显示保存中、成功、失败、时间和缩略图；失败可重试保存，缩略图可打开本窗口内的原图 Dialog，成功回执 8 秒后收起。预览期间不收起回执；切换会话清理预览/回执。回执采用实色语义背景，位置随底部控制栏实际高度调整，窄窗控件换行也保持间距。独立页面显式生成这两个共享组件的 Tailwind 样式。主页面 `desktop-playback-service` 仅在 captureRevision 增加时刷新帧库，不因普通时间事件或旧快照重复刷新。

验证：`pnpm desktop:test:build`、`pnpm build:desktop-player`、`pnpm typecheck` 与定向 ESLint 通过；`pnpm test:electron` 18 文件/123 项通过，包含真实 mpv/Win32 宿主截图为 320×180 源尺寸、7 秒帧、暂停保持、播放恢复和临时目录清理，以及 multipart 身份/时间/演员、失败同图重试、并发拒绝、旧分部拒绝、401/403 关闭。定向前端 4 文件/13 项通过，包含回执生命周期、首次失败的响应式重试按钮、晚到会话结果隔离、偏好与主页面刷新，以及 Web 保存/i18n 回归。

正式 Electron main + 隔离 profile + 认证合成 Server 验证：选 part-2、暂停于 12 秒，相机按钮及 C 产生 640×360 PNG，上传带当前 HttpOnly Cookie、正确 fileId/时间/演员；模拟 500 后回执显示原图和重试，重试上传同 ID/同图片 SHA，成功回执出现；配置 V 后按 V 同样采集，原图预览加载为 640×360，原生窗口截图确认回执背景与控制栏间距。验收中修复了重试 computed 未跟踪候选结果与共享组件缺少本地 Tailwind source 两处实际 UI 问题。本机证据保存在 ignored `output/playwright/desktop-player/qa-results.json` 与 `captured-frame.png`，仅为合成测试片源；没有萃取或保存用户真实影片画面。

本次范围为静态帧入 Server 萃取帧库。长按片段/GIF、自动额外下载/目录导出、逐帧、原生 PiP/轨道/HDR/主页面内嵌仍未实现；可切 Web 使用对应已有功能。没有新增分发/安装/真实 LAN 或 HTTPS 验收，不改变 M4 与总体 REQ-0058/0059 的未完成边界。无版本递进、push、合并或发布。

### 14.16 丰富原生播放诊断（2026-10-06）

设计框架：沿用 Web PlayerPage 的两列播放信息布局和原生 Info / D 入口，在视频上按需打开紧凑实色语义面板。标题/关闭为首层，播放、视频、音频、网络缓存按组浏览，标题行复制/保存为圆形次要动作。窄窗限高内部滚动，保留底部控制栏；加载、无音轨、属性不可用显示 `—`，暂停与缓存闲置保留真实零值，负 A/V 偏差保留符号。Esc 先关闭面板，操作不触发画面暂停。仅新增原生业务组件，不改全局 tokens 或 Web 诊断布局。

数据口径：mpv 的视频/音频码率为 bits/s 估算，源帧率与显示刷新率区分；预计文件帧数不能当实际呈现帧数或掉帧率分母。当前原始 HTTP/Range 媒体为一条音视频共享流，代理按背压计数实际接收字节、总媒体请求/Range 请求，1 秒采样真实传输速度与有界历史。避免伪造 DASH 分段或独立音视频下载速度。仅公开来源 host:port、协议及 MIME，复制/保存由 main 构建允许字段报告，不包含完整 URL、能力 token、Cookie、PIN、路径或 HWND。切片、失败、关闭清理旧会话统计与定时器。

实现：增加 optional 强类型诊断 DTO，main 每秒并行读取允许的 mpv 属性，零/缺失/非法值与 signed avsync 分别投影；每个事件深拷贝，停止/失败/退出清采样任务。代理使用 Transform 流式计数，不缓存整片，单调时钟 1 秒速率与最多 30 点曲线；HEAD 不计 body，seek 重读计入本次流量。协调器合并引擎/当前代理，active 检查阻止旧会话污染。仅本地播放 bridge 提供 copyDiagnostics / saveDiagnostics，main 生成白名单 JSON 并经系统剪贴板/native save dialog 输出；取消正常返回，重复保存有界拒绝，对话框期间源变化拒绝旧报告。

UI：四组两列字段和轻量速度曲线，源 FPS 与显示 Hz 分开；两种掉帧并列，预计文件帧数明确估计，解码像素格式可能为 NV12/D3D11。中文/英文/日文标签，缺失值 `—`，0 B/s/0 掉帧/负 ms 保留。初版底部导出按钮在 640×480 检查中裁剪，改为标题行圆形复制/保存/关闭，数据区内部滚动，避让实际底栏；客户区高度 ≤560px 时面板上边距缩至 16px，以保留可读信息区。640×480 和 626×444 renderer 检查中，数据区分别为134/98px，全部操作可见且无横向溢出。Esc/D 恢复 Info 焦点，拦截面板按键与点击的播放冒泡；无变更系统标题栏三个原生窗口按钮。UI 范式同步 ui-spec 与 ui-component-spec；最终合成样本预览位于 ignored `output/playwright/desktop-player/native-playback-info-preview.png`。

验证：`pnpm desktop:test:build`、更新后的 `pnpm build:desktop-player`、`pnpm typecheck` 与定向 ESLint 通过。19 文件/129 项 Electron 测试通过（真实 mpv 合成无音轨视频参数/关键帧码率、允许字段投影、HEAD/重复 Range 的字节、1 秒速率/闲置归零/有界 history/停止回收/重新起播清零、旧会话隔离与报告白名单）；4 文件/11 项前端定向测试通过（信息格式单位、missing/zero/负 A/V、三语、曲线、面板动作/反馈/Esc、已有萃取/主页面服务）。

正式 production main + 隔离 profile + 认证 640×360/30fps H.264/AAC 合成 Server 实测：mpv d3d11va/gpu-next、48 kHz 单声道、视频约 2.03 Mbps 与音频约 67.56 Kbps 估算、显示 160 Hz、两类掉帧 0、预计 3600 帧、约 107.8 秒前向缓存、实际代理接收 35.03 MiB/2 Range 请求、预读结束后0 B/s。真实剪贴板复制后粘贴并解析白名单报告通过；切至 part-1 生成新 session/清 history，旧 session 导出拒绝。面板实色背景 `rgb(13,15,26)`、底栏 12px 间隔及 native/Web 合成截图检查通过。narrow renderer 640×480 与最小宿主客户区近似下内部滚动/操作完整/无横向溢出及 Esc/D 焦点验证，不等同跨屏混合 DPI。

native JSON 保存对话框已实际显示；Windows 窗口工具将其识别为另一进程 owned modal，按恢复流程刷新/重试仍拒绝操作（point is over electron.exe, not native-player-host.exe），未声称完成真实对话框输入。另起隔离 fixture 仅注入 dialog 的 cancelled/已选 QA 路径结果，其余沿用正式 main handler；取消不显示错误/成功反馈，真实磁盘白名单 JSON 写入与 UI 成功反馈通过。本机诊断文件 `.workspace/diagnostics-qa-saved.json` 与 ignored QA 记录均为合成数据。

本轮没有真实影片截图/萃取、共享 Server 写入或正式应用退出；原生/Web 标题栏和萃取能力保留。日常测试 Desktop 重启到本轮产物；重启前没有活动播放（resume=null），因此不自动起播。未运行 test:display、混合 DPI、真实 LAN/HTTPS、安装/发布验收；无版本、push、合并或发布。

#### 14.16.1 修复实时信息文字跳动（2026-10-06）

用户确认现象为数字、单位与文字来回跳动。对正在打开的面板只读采样 8 秒，外框坐标/尺寸、行位置及滚动位置稳定；A/V 偏差与接收速度文本宽度却随数据变化。原因是格式化经 `Number(toFixed(...))` 删除末尾零，整数位数、符号与单位长度的变化又推移后续文字。此次修复保留小数位（默认 2 位、FPS/Hz 3 位、计数整数）、A/V 始终显示符号；数字/单位拆成固定 `9ch`/`5ch` 宽、16px 高的独立槽，数字右对齐，缺失值也保留单位占位。组合指标的分隔符位置固定；曲线始终预留空间，数据区固定滚动条占位并关闭滚动锚定。首次压力检查发现空单位仍影响行内基线，最终改为固定高度、垂直居中，缺失到实测值的切换也保持稳定。

验证：定向前端 2 文件/5 测试、`pnpm typecheck`、定向 ESLint、`pnpm desktop:test:build` 和最终 `pnpm build:desktop-player` 通过。使用正式 production main、隔离 profile、认证合成片源，在 986×763、640×480、626×444 renderer 尺寸各连续采样 8 秒；所有数字槽、单位槽、分隔符、面板边界和滚动值保持不变，实际视频/音频码率、A/V、缓存值仍持续更新。窄窗滚动位置 650px 保持；数据区高度 134px/98px，无横向溢出。另在同一 QA DOM 同步临时替换文本，检查 `—`/空单位、`0.00 B/s`、`1023.99 KiB/s`、`1.00 MiB/s`、`-1234.56 ms`、`+0.00 ms`，全部槽位与分隔符坐标不变；压力检查后恢复文本，未改产品数据。证据在 ignored `output/playwright/desktop-player/info-jitter-{normal,640,626}.log`。此为布局验证，不等同跨屏混合 DPI 或新性能基准；此次未改 Electron/Server 逻辑，未重复完整引擎测试。

测试 Desktop 已退出、重建并重新启动；重启前保存当前影片/分部/队列及进度，恢复 `fc2-4985807:primary` 于 2932.663067 秒（48:52）为暂停，重新打开播放信息供检查。没有截取或萃取用户影片画面。未改版本、push、合并或发布。

#### 14.16.2 统一值列对齐与英文状态值（2026-10-06）

用户指出数字与普通文字的起点不一致，并要求播放状态结果直接显示 paused、playing 等英文。原因是 §14.16.1 的固定数字槽使用右对齐，而普通文字使用左对齐。本次将数字槽内文字改为左对齐，使全部值从同一列左缘开始；保留 9ch 数字槽、5ch 单位槽、固定高度与小数位，继续稳定实时采样的单位/分隔符位置。播放状态直接使用 NativePlayerState.status 的小写英文值；标题、字段标签和其它反馈继续使用当前界面语言，仅修改原生诊断组件，不改全局 token、共享 Web 面板或主进程。

验证：信息格式/组件交互定向 2 文件/5 项通过，其中既有用例覆盖中文、英文、日文界面的 paused/playing 原始状态值；typecheck、定向 ESLint、desktop:test:build 与 diff 检查通过。在正式测试 Desktop 的实际原生页面中，986×763、640×480、626×444 renderer 尺寸的 28 行值均与值列左缘一致，每个尺寸连续采样 4 秒，数字/单位/分隔符与滚动位置稳定，无横向溢出。同步临时替换 —/空单位、0.00 B/s、1023.99 KiB/s、1.00 MiB/s、-1234.56 ms、+0.00 ms 后，槽位与分隔符边界均不变，检查后恢复原始文本和窗口 viewport。证据保存在 ignored output/playwright/desktop-player/info-align-layout.log。此为局部布局检查，未运行 test:display 或混合 DPI 验收，未重复无变化的 Electron/Server 引擎测试。

测试 Desktop 已退出、重建、重启，保留原有影片/分部/队列与来源，恢复 fc2-4985807:primary 于 2932.663067 秒（48:52）为暂停；播放信息面板保持打开并实际显示 paused。没有截图/萃取用户影片或生成 GIF，没有修改版本、push、合并或发布。原生诊断的左对齐与状态显示约定同步回写 UI 规范及组件规则。

### 14.17 原生 GIF 截取实施（2026-10-06）

按用户要求补齐 GIF。设计范围为现有原生播放面：短按相机/已配置萃取键保持静态帧，长按沿用 Web 400ms 阈值与 0.4–6 秒媒体时长；增加圆形 GIF 开始/停止入口便于发现。反馈复用 Web 播放面的实色语义面板，录制时间优先，处理进度与取消次之，成功/失败就地显示；窄窗避让实际底栏。暂停/缓冲不按墙钟虚增片长，跳转、换源、失焦取消未提交录制。仅为原生局部组合，不改全局颜色或系统标题栏。

主进程冻结会话/分部与实际起始帧，释放时读取 mpv 真实结束位置；先保存静态帧，再通过当前 Electron Session 请求已有 Server clips/task/cancel HTTP 合约，以 GIF、10fps、640px 和同一 curatedFrameId 导出。Server 负责生成及入帧库，Desktop 不另装 FFmpeg、不下载整片。任务轮询有期限与失败上限，取消/切片/关闭清理旧任务，完成后刷新主页面帧库。Renderer 不接收 Cookie、Server 地址、路径或任意任务访问能力。验收使用合成片源，保留用户测试播放位置。

实现：仅本地 bridge 新增 prepareCapture/commitCapture/discardCapture/cancelClip/retryClip；prepare 冻结一张未上传 PNG 与真实 time-pos、60 秒未提交回收，短按 commit 保存静态帧，GIF commit 以真实结束位置约束范围，再调用既有 clips?fileId。失败上传保留同图/同 ID 及 GIF 范围，失败生成重试已保存帧/冻结范围；不再次采集。main 管理一个任务监视器，后台轮询不占用整段任务的串行播放队列，150 秒期限/连续 5 次状态失败后停止，换源/关闭取消自己的任务；401/403 停止原生播放。取消请求失败不宣称已取消，保留监视与重试取消入口。GIF 就绪路径须匹配当前 frameId 的 motion 端点，完成增加 captureRevision。6 秒上限减去 1 微秒，避免 JSON 浮点相减略大于 6 被 Server 拒绝。

UI：复用 usePlayerClipCapture 的媒体时钟状态机，起点以主进程冻结帧时间修正；短按相机/键释放提交一帧，长按释放或媒体达到上限提交 GIF，自动完成后的再次释放不额外采集。GIF 按钮在播放中开始、录制中停止，暂停时独立入口禁用；长按萃取键但媒体未前进足够时仅静态帧。实色 NativeClipCaptureFeedback 以三语显示录制时长、生成进度/取消、成功/失败/重试，依实际底栏高度保持 12px 间隔，与图片回执不重叠。用户原有系统标题栏、萃取图片预览、诊断面板保持可用。

验证：`pnpm desktop:test:build`、最终 Electron 编译、typecheck 与定向 ESLint 通过。完整 Electron 19 文件/140 项通过（含真实 mpv 暂停/播放结束时间读取），最后同文件显式 seekExisting 也清理录制候选的补充定向 27 项通过；前端 4 文件/23 项通过（短/长按、实际起点、媒体六秒自动提交/松键抑制、暂停、取消/迟到结果、准备失败恢复、GIF 状态/重试、Web/i18n 与主页面刷新）；既有 Go Server clips/文件归属/格式/范围/取消定向测试通过。未运行 test:display 或重新做混合 DPI、真实 LAN/HTTPS/安装测试。

正式 production main + 隔离 profile + 认证合成 Server fixture 通过当前 Session 调用真实 HTTP/Range；fixture 按现有 Server 合约真实执行 FFmpeg GIF 生成（并非共享开发 Server 业务入库）。GIF 按钮录制区间 73.7–76.333333 秒，输出 640×360、10fps、26 帧/2.6 秒、GIF89a，UI 显示已入帧库，captureRevision 增加两次（静态/动图）。实际短按只上传一帧且无 clips 请求；实际 C 长按在 109.833333–115.833333 自动完成，按键抬起后累计仅一张新帧/一个 GIF 任务；6 秒 GIF 为 60 帧。保留静态帧的生成中任务按取消后 fixture 为 cancelled；640×480/626×444 面板均距底栏约 12px、取消按钮完整可见、无横向溢出。末尾 1 微秒上限与取消失败/认证分支修正由定向用例覆盖，最终产物已重建。证据在 ignored `output/playwright/desktop-player/native-gif-ui.log`、`native-gif-inputs.log`、`native-gif-qa-results.json`、`native-gif-qa.gif`。

测试 Desktop 已重建/启动；保留 `fc2-4985807:primary`、队列/来源于 2932.663067 秒（48:52），恢复为暂停，读取 UI 确认 GIF 按钮可见且暂停时禁用。没有截图/萃取用户影片或向共享 Server 生成 GIF。当前原生只支持 GIF 片段格式，MP4/WebM/额外本机导出/逐帧等范围继续待做；REQ-0058/0059 状态、分发资料/安装边界不变。无版本、push、合并或发布。

#### 14.17.1 截取成功反馈自动退出与位置对齐 Web（2026-10-06）

用户反馈静态帧/GIF 成功提示常驻且位置过高。本次沿用 Web 播放面的局部反馈：静态帧成功回执 2 秒退出，打开原图预览期间暂停计时、关闭后重新计时；GIF 成功或已取消提示 1.6 秒退出。生成中与失败结果保留取消/重试动作，手动关闭仍可用。位置按 Web 原有静态帧桌面底部 96px/窄屏 176px、GIF 128px 起算，仅按实际进度条与按钮区域及底部 padding 向上避让 12px；排除原生底栏上方的 48px 渐变留白和时间行，不改变全局 token 或 Web 布局。

原因与实现：原来 GIF 的 8 秒计时器只重置局部录制 phase，却仍由持久 snapshot.clip 控制显示，终态一直可见；静态提交若遇到旧 clip 也可能跳过 expiry。现在 composable 管理独立的图片/GIF 计时器和按 session/frameId/phase 识别的本地隐藏标记，页面统一使用该可见状态；连续收到相同终态快照不重开提示或重置计时。GIF 完成/手动关闭同时清理其起始帧回执，避免退出后又显示静态帧；新捕获、换会话及卸载清理旧计时器。失败重试仍使用同帧/同 ID。反馈下移后的实际点击验证发现 footer 透明渐变截获 GIF 取消点击，最终将 GIF 反馈提升到与现有 CaptureReceipt 一致的局部 z-20，保留正常按钮命中。

验证：前端定向 2 文件/19 项通过，包括静态 2 秒到期/大图保留、GIF 处理中保留/成功 1.6 秒退出、重复终态快照不复现、关联静态帧不复现、旧 GIF 计时器不删除新回执、手动关闭、既有短/长按/媒体时长/重试/会话隔离；typecheck、定向 ESLint、desktop:test:build、最终 build:desktop-player 与 diff 检查通过。仅 renderer/业务组件修改，未重复无变化的 Electron/Server 测试。

正式 production main + 隔离 profile + 认证合成 Server/真实 mpv/真实 GIF 生成验证：986×763、640×480、626×444 下静态回执均比原先下降约 80px，与进度/按钮区域保持约 12px 间距，全部操作可见且无横向溢出；成功回执自动退出。大图打开超过 2 秒仍保留、关闭后自动退出；真实 GIF 成功后自动退出，main 仍保留 saved，但后续快照不再显示 GIF 或其起始帧。失败图片等待超过 2 秒仍可重试、成功后退出；hold 任务处理反馈保持、真实点击取消后自动退出。最终在 640×480/626×444 再验证 GIF 录制反馈及取消按钮中心命中、实际取消与退出，确认 z-20 生效。证据位于 ignored output/playwright/desktop-player/capture-feedback-ui.log 和 capture-feedback-narrow.log；全部截帧/GIF 为合成素材，没有截图或萃取用户影片，没有写入共享 Server 的帧库。

验收 fixture 已退出，日常测试 Desktop 已重启到本轮产物。重启前不存在活动原生窗口（保存 intent 为 null），因此不自动起播旧影片。未运行 test:display、混合 DPI、真实 LAN/HTTPS 或安装验收；UI 约定同步规范与组件规则。未改版本、push、合并或发布。


### 14.18 原生引擎分发材料与生产流水线（2026-10-07）

用户明确本次生产包必须包含原生播放器，先补齐分发材料与流水线。此前开发 QA pin 保持 distributionReady=false，不能因开发播放成功就作为生产来源。Windows stage_desktop 改为强制 prepared CURATED_NATIVE_BUNDLE；缺引擎/源码/构建/许可、SHA 或 DLL 闭包失败均拒绝出包。macOS 继续 Web；本机 preferNative 默认 false 不变。

生产来源采用 MSYS2 UCRT64 mpv 0.41.0-8，运行时 FFmpeg libraries 9.0.2-1。windows-x64-production.json 固定 105 个提供运行文件的二进制包（131 文件）、153 份源码快照和 731 份 Cargo crate 的 URL/SHA；开发 shinchiro pin 与生产 lock 分开。每个二进制 .BUILDINFO 的 pkgbuild_sha256sum 与对应源码 PKGBUILD 原文 SHA 完全一致，保存完整 .BUILDINFO/.PKGINFO、版本/许可证目录、原始上游源码包或准确 Git 源码、补丁与构建配方。头文件/静态依赖快照与 Rust 锁文件的 crates-io 依赖均包含，恢复脚本生成离线 Cargo vendor/checksum/config；不把可变化的仓库首页作为对应源码。

源码归档保留原始文件字节，通过 SHA blobs 去重，使用 complete-sources.tar.zst（1435.88 MiB）。whisper.cpp 原始 Git 缓存约 710 MB，固定在 60c0be6ac8fa71b1a2ae2dd938a31a34a508e774 并保留 184 层/标签及 9 位缩写后约 9 MB；assert 原提交、源码树及 v1.9.4-183-g60c0be6ac 均一致，只省略无关更早历史，不改变源码/PKGBUILD。CURATED-SOURCE-SNAPSHOT.json 保存原始源码包 SHA 与还原策略。包内 120 个许可/说明/恢复文件及 884 个源码组件条目，mpv 的 GPL/Copyright 与 FFmpeg COPYING 原文独立保存。引擎总资料 1620.94 MiB，离线材料使本次 Windows 包明显大于旧 Web-only 包；后续若需更小交付，应评估可控的精简构建，不能删掉必需对应源码来缩包。

staging 只复制声明文件，重新 hash 并检查 x64 PE normal/delay DLL import 闭包，除明确 Windows 系统 DLL/API set 外必须有本地 DLL。正式运行始终选择 resources/app/native-player/mpv.exe，忽略开发 CURATED_NATIVE_MPV；不把 Server 的 ffmpeg.exe/ffprobe.exe 带入 Desktop。CD 显式装 UCRT64 gcc/windres，按 lock 缓存下载输入，Python 3.12 安装固定 Zstandard wheel，准备/校验后出包；Windows 作业期限为 90 分钟。windows_smoke 在真实安装器生命周期之外增加安装目录原生播放验证，以合成 H.264/AAC/HttpOnly Cookie/HTTP Range、最小 PATH、无效开发引擎覆盖完成播放、seek、PNG、换片和退出回收；不依赖开发机 mpv 或 MSYS PATH。

本地已通过：19 文件/140 项真实候选引擎 Electron 测试；312 文件/1688 项前端；全量 lint/typecheck、生产 Web 与 Electron 构建、Go test ./... 和 vet；actionlint 检查新 CD 工作流。最终 stage_desktop 生成的正式结构 payload 又通过最小 PATH 的实际认证播放、媒体时钟、seek、320×180 原图、part replacement、安装目录 executable 和无遗留子进程检查。全部媒体均为合成样本，没有截取用户影片或写共享 Server。

本地完整源码已从归档离线恢复，Git 长路径选项下核对 whisper.cpp 为原提交、原树 e08fcfd8890eccc6d7e13b2b809c4a483c34516f 和原版本号；恢复的 Cargo vendor/config 与校验材料均存在。真实 Inno 编译 260.515 秒成功，QA installer 1723432848 bytes（1643.59 MiB）；最终 102 项 release pipeline 测试通过。之后从干净提交准备 20261007 批次并触发不可变标签 CD，实际 Windows 安装/升级/卸载与 Mac 验收成功才允许公开。具体 run/tag/version 将回填本节。本地 payload 成功不等同已发布；更多片源、整片 A/V、混合 DPI、真实 LAN/HTTPS 仍沿原边界继续待验。

发布前生产依赖审计发现 Vue server-renderer 和 source-map-js 两项高危，已分别升级 Vue 3.5.42 与 source-map-js 1.2.2；官方 registry 复审高危归零（剩余两项 low），pnpm 11.0.0 frozen-lockfile 安装通过。锁文件的 Vue 编译器/Babel 依赖按新版要求更新，保留原平台 libc 约束，没有改产品偏好。运行时 e2e 同时暴露 Desktop 路由适配的静态影片服务 import 在认证前启动读取；现在只有已选 native 的播放导航才动态导入，模块加载失败也进入单次 Web 回退。现有锁定启动验收重新通过，并补充模块加载边界与失败回退验证。

补丁后复验：312 文件/1688 项前端全量通过（首轮与其它检查并行造成超时，限制本机 4 workers 后独立复跑成功）；新增失败回退后相关 3 文件/11 项通过；19 文件/140 项真实候选引擎 Electron 通过；全量 lint、最终 typecheck/Web 构建、Electron/播放器构建与 9 项浏览器运行时通过。更新 Vue 后重新生成正式结构 payload，再次通过最小 PATH、无效开发 mpv 覆盖下的安装目录引擎、认证 Range、播放/seek/PNG/换片/退出回收；源码资料与引擎未变化，无需重复大归档准备。生产触发前依旧未公开新包。

2026-10-07 01:09（北京时间）已完成本次用户授权的生产触发：专用分支 codex/windows-native-player-prototype 与不可变 release-20261007 标签一同推送，标签固定提交 571245b21ad70716aa615e0104f86cfa8ba9dd5f，未合并或更新 master。批次 Server 1.7.9 → 1.7.10、Desktop 0.2.4 → 0.2.5，来源摘要与发布说明核对一致；[CD run 37501360378](https://github.com/yepHiu/Curated/actions/runs/37501360378) 已 in_progress，目前验证源码与说明，之后依次执行精确提交质量检查、Windows 原生资料/安装验收及 Mac 包验收，全部成功才公开并推进更新通道。这里记录的是触发成功，不是安装包已公开；失败时保留标签和旧公共版本，按流水线诊断处理。Windows 包内含完整离线对应源码，因此本地 QA 安装包约 1.60 GiB，最终生产大小以 CD 产物为准。

### 14.19 安装包精简与源码独立交付（2026-10-07）

用户指出不需要把大份源码资料放进安装包。已取消尚未发布的 run 37501360378（最终 cancelled），保留原不可变 release-20261007 标签及原公共版本，不发布 1.7 GB 安装器。新版安装目录只包含 mpv/实际所需 DLL、许可、小型构建记录和准确源码下载说明；完整源码、原始构建输入、许可与还原脚本作为同一 Release 的 Curated-Desktop-Native-Sources-<Desktop version>-windows-x64.zip 单独提供，不进入更新器的安装包选择列表。

实现沿用已经校验的 MSYS2 引擎与完整准备目录，不删除对应源码、不替换第三方许可、不更改播放行为。准备资料仍核对准确配方/源码/二进制 SHA 和完整来源；运行 manifest 区分精简目录并保存同一官方 Release 源码资产 URL/SHA，SOURCE-DOWNLOAD.txt 明示无需下载源码即可播放。独立 ZIP 保留已压缩完整归档，采用 ZIP_STORED 避免再次压缩；Windows 构建记录保存 nativeSources。新批次声明 desktopNativeSources，发布 staging/公开前校验源码 ZIP 成员、材料 SHA、准确引擎 lock、批次/版本/commit，以及上传完整资产的结果；源码包存在于 release.json 与 SHA256SUMS，但不进入 desktop.json 的应用安装列表。真实安装的合成播放 gate 同时确认安装目录无大源码归档且下载说明存在。旧批次与旧标签不改写。

本地真实 Inno 编译通过，用时 114.859 秒；安装器 221382516 bytes（约 221 MB / 211 MiB），对比原 1723432848 bytes 减少约 87%。安装后 payload 653061110 bytes（约 623 MiB），独立来源 ZIP 1508785338 bytes（约 1.41 GiB）。正式结构精简 payload 在最小 PATH、无效开发 mpv override、认证 HTTP Range 下通过播放、seek、320×180 PNG、换片、安装目录引擎及退出回收；未截图/截帧用户影片，未写共享 Server。源码资产缺失/损坏/异批/错误引擎 lock、安装目录误带归档与更新清单隔离都有发布脚本回归覆盖；最终测试与新批次 run 记录随后回填。

精简交付的全量 release pipeline 105 项通过，新增批次声明/下载说明后相关 40 项再次通过；真实 1.41 GiB 来源 ZIP 也通过发布校验器与本仓库准确引擎 lock 对照。仅打包/发布脚本修改，无前端/Go/播放实现变化；没有重复执行无变化的 UI 或媒体功能全量测试。已准备并推送新不可变 release-20261007-2，固定提交 c5f40bf7aa2d357c67fc82b5eb497afaa92599ff，版本仍为 Server 1.7.10 / Desktop 0.2.5（前一批未公开，不移动旧标签）。[CD run 37505997682](https://github.com/yepHiu/Curated/actions/runs/37505997682) 已触发并排队，实际生产安装/升级/播放器验收与全资产上传成功后自动公开。运行状态与本地 QA 大小不代表已发布或最终生产文件字节数。

### 14.20 Windows 发布 Go 缓存策略优化（2026-10-07）

用户要求优化发布作业的 Go 缓存耗时。[Windows job 112417874727](https://github.com/yepHiu/Curated/actions/runs/37505997682/job/112417874727) 共 25 分 44 秒，Setup Go 本身仅 21 秒；结尾 Post setup-go 保存缓存耗时 7 分 35 秒，约占作业 29.5%。日志中 tar/zstd 命令从 18:10:39 UTC 执行至约 18:18:12，上传 246633924 bytes 仅约 2 秒。证据定位到本地读取、归档和压缩阶段，不能进一步断言全部时间由压缩 CPU 或文件扫描造成。本次缓存未命中，缓存列表显示相同 key 在多个不同发布 ref 下分别保存，跨发布标签不能保证复用。

实施：只在 `.github/workflows/cd-release.yml` 的 Windows build 作业为 `actions/setup-go@v5` 显式设置 `cache: false`，移除已无用途的 cache-dependency-path。保留 backend/go.mod 的 Go 版本选择及 Go 默认本机缓存路径；同一作业内多次构建仍复用本机模块和编译缓存。Linux backend-quality 的现有缓存、pnpm 缓存及原生引擎输入缓存保持现状；构建、安装和播放器验收门禁继续执行。发布标签作业不再恢复或保存 setup-go 缓存，因此不再因这项 post 归档阻塞后续发布。

代价与验收：Windows 发布需要冷下载依赖、冷编译；净收益以未来采用此提交的 CD 运行实测为准，不把本次 7 分 35 秒直接承诺为净节省。下一次核对 Go setup 无缓存恢复、无 Post setup-go 缓存保存，并比较 Windows 作业整体耗时与实际构建耗时。若后续冷下载成为瓶颈，再根据日志评估只读恢复模块缓存；本次不新增缓存维护流水线。actionlint v1.7.7 已通过 CD 工作流检查（关闭外部 shellcheck/pyflakes 集成）；本次仅修改 CI 配置，无需重跑 UI、Go 业务测试或生成生产包。修改在专用分支本地提交，不改版本或既有不可变标签；现有 run 使用其原始工作流，新策略需推送并由后续包含该修改的发布运行采用。


### 14.21 合入本地 master（2026-10-07）

用户要求将原生播放器代码合入 master。已保留两边提交历史，使用非快进合并将 `codex/windows-native-player-prototype`（c176822f）合入本地主线，合并提交为 `4ac3c9c9`；随后合入 `codex/fix-release-native-lock`（e4dcd138），合并提交为 `91cba070`，纳入跨平台 LF/CRLF 引擎 lock 哈希兼容修复。主线既有 FC2 刮削与 AI 缓存修复保留；三个文档合并冲突均保留双方各自内容，没有代码冲突。

合并前未提交的离线转码方案和索引修改先暂存保护，再恢复为未提交状态。旧 VLC 研究文档的修改已包含在原生分支中，恢复时保留较新的已批准/已实现状态说明，避免退回历史提案状态。原生播放仍为 Windows 可选能力，`preferNative=false`；本次未改变既有版本和不可变发布标签，未推送远程、生成安装包或触发发布。

本地主线验证：

- `pnpm install --frozen-lockfile` 成功；`pnpm lint` 通过。
- Web API 模式 `pnpm build`（含 typecheck）通过，体积报告无提醒或超限。
- `pnpm build:electron` 通过，包括 Electron TypeScript、连接页、Win32 helper 和原生播放器页面。
- `pnpm test:electron`：16 文件 / 132 项通过；3 文件 / 8 项真实引擎或独立原型集成测试因本次环境未启用相应输入跳过。本次未重复真实媒体/安装包验收。
- 显式 `VITE_USE_WEB_API=false`、`NODE_OPTIONS=--no-experimental-webstorage` 的 `pnpm test --maxWorkers 4`：312 文件 / 1689 项全部通过。首轮受本机 `.env` 的 Web API 模式影响，25 项设置测试失败；切回测试所需 Mock 模式后全量通过，没有修改产品权限逻辑。
- `python -m unittest discover -s scripts/release/tests -p 'test_*.py'`：107 项通过。
- `pnpm test:e2e` 首轮 7 项通过、2 项导航超时；失败项定向复跑后均通过，其中 Mock 演员页至设置页导航在单 worker 复跑通过。9 项最终均通过，记录首轮与复跑结果，不宣称首轮全绿。

本次验证覆盖源码集成和构建回归；既有混合 DPI、更多片源、整片 A/V、真实 LAN/HTTPS 待验范围仍以此前章节为准。


### 14.22 产品分支最新提交同步（2026-10-07）

用户要求将分支代码全部合入 master，随后明确更正：独立看板不合入 master。以最新指示为准，`codex/local-project-dashboard` 保留独立，不纳入产品主线。

重新检查全部本地分支，原生引擎修复分支新增 8a506c9c / e47b64df，包含冻结发布资产的恢复说明及 `.github/workflows/recover-release.yml`。已通过 `3303cb3b` 非快进合并纳入本地 master；该工作流使用修正后的工具校验原始不可变来源及成功门禁，再恢复既有资产，不修改原始标签。

`git fetch origin` 后再次检查：除明确排除的独立看板外，全部本地产品分支与远端产品分支的现有提交均已包含在 master。`origin/release-channels` 是独立更新通道数据分支，不属于产品源码，不合并。主线没有 `scripts/project-dashboard`，独立看板未引入。原有离线转码方案与索引的未提交修改已恢复。

本次仅新增发布恢复工作流与说明；`python -m unittest discover -s scripts/release/tests -p 'test_component_cd.py'` 的 19 项通过。前一节产品构建与全量回归结果仍适用，产品代码及依赖未变，不重复运行。恢复工作流在来源分支已通过 actionlint v1.7.7（见恢复计划）；本次没有触发工作流、推送远程或执行实际发布。


### 14.23 最新 master 新批次打包触发（2026-10-07）

用户要求再次触发新版本包。发布前读取稳定 Release/通道，当前公开基线仍为 Server 1.7.9 / Desktop 0.2.4；20261007 与 20261007-2 未公开。基于集成后的 master 准备新批次 `release-20261007-3`，Server 1.7.10 / Desktop 0.2.5，不额外递增未发布版本，不移动旧标签。说明同时包含 Windows 原生播放、独立源码资产、FC2 元数据、AI 请求缓存前缀、跨平台 lock 校验和 Windows Go 缓存策略修复。独立看板仍排除。

批次及说明已提交为 `3bf7790665ced4a0f6c7d11af88f2b40321c8abd`，master 与新标签已原子推送；15 项 batch 测试通过，元数据/源码摘要/版本表校验通过。用户原有未提交文档已恢复，未带入发布源码。标签触发 [CD run 37513511993](https://github.com/yepHiu/Curated/actions/runs/37513511993)，已核对 headSha 与标签提交一致，记录时为 pending。后续完整质量门禁、Windows 两端与 Mac Desktop 构建/安装验收全部通过后由标签 CD 正式发布。本节仅确认触发成功，不代表包已生成或公开；实际结果以该运行及 Release 为准。

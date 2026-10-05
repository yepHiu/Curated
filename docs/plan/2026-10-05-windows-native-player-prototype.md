# Windows 原生播放器原型与后续集成实施计划

日期：2026-10-05（北京时间）
状态：in-progress
关联需求：REQ-0058
分支：`codex/windows-native-player-prototype`
基线：`408b8c10`，独立工作目录；本计划与原型不改主工作区。

最新进展：§12 的 Windows 同窗原型已实现并验证。mpv 绘制自有 Win32 宿主，透明 Electron owned window 叠加复用的 Curated 控件。下方第一版范围与 §9—§11 保留阶段记录；当前实现、启动要求与剩余边界以 §12 和 guide 为准。生产 PlayerPage 只提取共享控制栏，默认播放引擎未切换。

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

Windows 引擎实际构建/许可/依赖核对、独立更新和安装验收；macOS 窗口/渲染/硬解单独实现验证。Server 托管 UI 若改动则属于 Server 交付；引擎属于 Desktop，按实际变化独立准备版本。失败保留 Web/HLS 能力。

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
- `NativePlayerWindow` 管理有界 NDJSON、请求关联、超时和清理。helper 报告物理客户端矩形/DPI，并在本地移动消息中同步 owned UI 位置；Electron 主进程转换为 DIP，同步可见性。全屏保留原位置与最大化状态，关闭按保存进度 → 停止 mpv/代理 → 销毁 UI/helper 顺序执行。
- 透明无框 Electron 窗口仅承载 Web 控件，使用原生 owner 关系保持窗口层级，无全局置顶。主进程禁用此入口的后台节流，避免透明窗口遮挡时控件动画/轮询停顿。helper 的初始/最小尺寸使用 DIP，适配本机高 DPI；跨显示器混合 DPI 仍需专项验收。
- renderer 继续只接受有限业务桥，新增 `windowAction(fullscreen|minimize|close)`，不暴露 HWND、任意窗口几何、通用 mpv/shell 调用。已有身份/PIN/媒体代理和逐文件进度合约保留；原生模式禁用 mpv 默认 OSC/输入，Curated 负责鼠标、键盘和控件。
- `PlayerTransportControls` 从既有 PlayerPage 提取，直接共享圆形播放/快进后退按钮、音量和静音；动作插槽保留原 Web 队列、设置、外部播放器、PiP、全屏。原型共享分段选择和设置菜单，native 菜单仅显示倍速；准备弹窗负责连接/解锁/搜索/起播。播放时标题、进度条、底栏和信息面板叠在视频上。
- 分段切换读取该文件续播，保留倍速/音量/静音；单击播放暂停，双击全屏，沿用 Space/J/K/L、方向键、M/F/Escape/D。控件 5 秒隐藏，鼠标和键盘焦点可恢复，菜单/弹窗保持交互。未连接、锁定、缺引擎、空结果、忙碌、结束重播、播放失败和保存失败均有状态。三语设置消息采用函数，保持本地 CSP，不启用 unsafe-eval。

### 12.2 本轮验证证据

| 检查 | 结果 |
|---|---|
| Windows helper 编译 | MinGW gcc 15.2；C11、Wall/Wextra/Werror 通过；仅生成忽略目录内 exe |
| Electron 测试 | 14 文件 / 97 项通过；包括认证 Range、逐文件/锁定/并发/回收、真实 mpv 坏媒体与控制，以及 4 项真实宿主生命周期/嵌入测试 |
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

# Windows 原生播放器原型与后续集成实施计划

日期：2026-10-05（北京时间）
状态：in-progress
关联需求：REQ-0058
分支：`codex/windows-native-player-prototype`
基线：`408b8c10`，独立工作目录；本计划与原型不改主工作区。

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

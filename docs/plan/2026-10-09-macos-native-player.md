# macOS 原生播放器适配

日期：2026-10-09（北京时间）
状态：本机自包含包已实现并验证；签名公证与公开发行待完成
关联需求：REQ-0061（macOS 适配）、REQ-0059（共用正式播放链路）

## 目标

保留现有 Curated 播放控件、认证代理、分片续播、观看统计和萃取功能，在 macOS 的同一播放窗口内显示原生视频。Windows 的 Win32 宿主保持现有实现。新 Mac 正式安装默认使用原生播放器，已有偏好保留；开发/Windows 默认不变，原生启动失败保留现有 Web 回退。

## 实现边界

- macOS 的 NSView 指针不能作为跨进程 HWND 使用，因此采用主进程 Node-API 适配层、libmpv Render API 和 Cocoa 视频视图；透明 Electron 控件在同一窗口上层。
- 使用私有临时目录中的 Unix domain socket 复用现有 mpv JSON IPC。媒体地址、认证及 dylib 路径不进入 renderer。
- Cocoa 窗口负责红黄绿按钮、移动、缩放、最小化和系统全屏；渲染尺寸使用 backing pixels。
- 初始适配使用开发机 libmpv；随后按用户“无需安装 MPV”的要求完成下述自包含包。正式打包只接受固定来源/SHA 的准备资料，不直接复制任意本机库。

## 验证

类型编译、Electron 播放业务回归、真实 mpv Unix socket 控制/清理，以及真实 Electron + Cocoa + libmpv 合成视频测试。窗口内实际视频、控件、缩放和关闭须验证，不能以无画面的 headless 测试代替。

## 状态

2026-10-09 初始适配验证记录（后续自包含包见下节），未发布。

- 构建：`pnpm build:electron:main`、`pnpm typecheck`、定向 Vue ESLint 通过。
- Electron：启用 Mac 原生与本机 CLI 引擎后 143 项通过、6 项 Windows 专属用例跳过；Mac 专项实际启动生产播放 bridge，覆盖认证 Range、暂停/seek/倍速/音量、GPU PNG、系统全屏/还原、窗口 resize/最小化恢复、分片重建、关闭进度保存及 socket 清理。
- 打包逻辑：7 项 Mac staging/DMG 配置测试通过；未打正式安装包或发布。
- 本机目视：Apple Silicon、mpv 0.41.0、960×540/30fps 合成 H.264。窗口及全屏视频正常显示，HUD 同窗叠加，原生红黄绿可见，Cmd+W 关闭。诊断读取到 VideoToolbox 硬解。
- 修复了 Render API 默认模式下硬件帧无法经 software screenshot 转换的问题：启用 advanced control，并在主线程持有 GL context 处理 update callback；隐藏窗口也处理截图更新。
- 公开 API 与库配置不变。新增 readonly windowChrome 仅服务本地 HUD 的 Mac 标题避让。

## 后续边界

Mac 包现已自包含引擎，不需要用户本机 libmpv。尚未验收 Developer ID 签名/公证、互联网下载的 Gatekeeper 首次启动、Intel 实机、HDR/轨道/PiP、混合 DPI 多屏、整片音画同步；GIF 的业务调度复用既有验证，本次 Mac 专项验证至原图捕获，没有使用真实 Server 重新验完整 GIF 生成。Mac libmpv 位于主进程，崩溃隔离不同于 Windows 独立 mpv 进程。

## 自包含交付 — 用户无需安装 MPV

用户要求：首次使用者安装 Curated 后即可使用原生播放器，不自行安装 MPV/Homebrew、不配置环境变量。

- Mac 生产资源解析只接受 app 内 `native-player-macos/libmpv.dylib`，开发变量不能覆盖。新正式 Mac profile 默认 native；已保存的 Web/native 选择、音量和倍速不变。开发态仍可使用本机引擎。
- `prepare_macos_native.py` 使用仓库固定 bottle/source/resource/recipe SHA，离线缓存命中时不重新解析 latest。重定位全部依赖/ID 为同目录 `@loader_path`，解析 bottle 中版本 symlink，移除 rpath，签名后检查 arm64、闭包、最低系统和资料。
- 48 个 dylib 加一个稳定入口（共 49 个文件），约 66 MiB；对应源代码、补丁、shaderc 资源、header 资源、配方/SBOM 与许可直接随 app。当前 DMG 约 442 MiB、ZIP 426 MiB，源码资料贡献部分体积；本轮没有改现有 Release 资产分类或更新器下载协议。
- 目标 Apple Silicon/macOS 15+，Info.plist 与库最低版本检查一致。引擎随 Desktop 升级。没有新增 Server API、数据库迁移或 library-config 字段。
- CD 先准备固定引擎，缺材料阻断打包；之后验证独立启动、隔离原生播放和现有升级链路。公开发布仍需现有所有门禁。

### 实际验证

- 已创建独立输出 `release/macos-native-validation-20261009/Curated-Desktop-0.2.6-macos-arm64.dmg` 与 ZIP，未覆盖原有包。属于本地验证产物，不代表版本发布。
- 打包构建通过，codesign deep/strict、DMG checksum、实际挂载后的 Finder 结构与签名检查通过。
- ZIP 解压到带空格的独立目录，使用新临时 profile 和认证 HTTP/Range 合成源播放，PATH 限于系统目录，开发引擎变量故意指向不存在文件。
- macOS sandbox 阻止加载 libmpv 的主进程读取 `/opt/homebrew`、`/usr/local`，测试主动验证限制生效。只允许 Frameworks 下的 Chromium helper 建立其自己的沙盒，以避免嵌套 seatbelt 初始化失败；没有给生产程序添加关闭沙盒参数。
- 实测播放时钟、暂停、seek、320×180 PNG 上传、分片重建与关闭清理均通过，新 Mac 默认原生偏好通过；独立无 Server 启动另测。
- Electron 基础回归 143 通过、9 项真实引擎/平台专属用例跳过；真实安装播放通过上面的独立测试覆盖。新增偏好保留、打包不回退 Homebrew、固定资料/配方、外部依赖/rpath/最低系统校验测试。
- 发布脚本最终全量 115 项（3 项跳过）通过；Mac 定向 15 项通过，Electron 类型编译与 actionlint 工作流检查通过。

### 仍需区分的交付边界

本机证据证明无需另装 MPV；不是完全干净物理机/所有系统与所有影片的保证。包目前仍是 ad-hoc 签名，未持有并执行 Apple Developer ID 签名/公证，互联网下载后可能被 Gatekeeper 拦截；这是达到新用户“下载即开”的剩余发布条件。未推送、未公开发布。

来源资料包含固定主源码、补丁、构建资源与配方。libplacebo 的 header-only fast_float 版本依据原 bottle 构建日期的 Homebrew 配方追溯，原 SBOM 没有记录该构建依赖；具体依据与重构限制写入随包 MACOS-SOURCES/BUILDING。没有声称字节级复现原 Homebrew 编译环境。

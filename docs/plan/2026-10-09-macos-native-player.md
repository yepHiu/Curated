# macOS 原生播放器适配

日期：2026-10-09（北京时间）
状态：in-progress
关联需求：REQ-0061（macOS 适配）、REQ-0059（共用正式播放链路）

## 目标

保留现有 Curated 播放控件、认证代理、分片续播、观看统计和萃取功能，在 macOS 的同一播放窗口内显示原生视频。Windows 的 Win32 宿主保持现有实现。默认偏好仍为 Web，原生启动失败保留现有 Web 回退。

## 实现边界

- macOS 的 NSView 指针不能作为跨进程 HWND 使用，因此采用主进程 Node-API 适配层、libmpv Render API 和 Cocoa 视频视图；透明 Electron 控件在同一窗口上层。
- 使用私有临时目录中的 Unix domain socket 复用现有 mpv JSON IPC。媒体地址、认证及 dylib 路径不进入 renderer。
- Cocoa 窗口负责红黄绿按钮、移动、缩放、最小化和系统全屏；渲染尺寸使用 backing pixels。
- 本轮先使用本机安装的 libmpv（Homebrew 或显式绝对路径），不将本机 Homebrew 二进制直接复制到发布包。完整自包含引擎的锁定构建、依赖闭包、来源/许可和安装验收需独立完成。

## 验证

类型编译、Electron 播放业务回归、真实 mpv Unix socket 控制/清理，以及真实 Electron + Cocoa + libmpv 合成视频测试。窗口内实际视频、控件、缩放和关闭须验证，不能以无画面的 headless 测试代替。

## 状态

2026-10-09：本机适配已实现并通过验证，未发布。完整自包含引擎分发仍待独立完成。

- 构建：`pnpm build:electron:main`、`pnpm typecheck`、定向 Vue ESLint 通过。
- Electron：启用 Mac 原生与本机 CLI 引擎后 143 项通过、6 项 Windows 专属用例跳过；Mac 专项实际启动生产播放 bridge，覆盖认证 Range、暂停/seek/倍速/音量、GPU PNG、系统全屏/还原、窗口 resize/最小化恢复、分片重建、关闭进度保存及 socket 清理。
- 打包逻辑：7 项 Mac staging/DMG 配置测试通过；未打正式安装包或发布。
- 本机目视：Apple Silicon、mpv 0.41.0、960×540/30fps 合成 H.264。窗口及全屏视频正常显示，HUD 同窗叠加，原生红黄绿可见，Cmd+W 关闭。诊断读取到 VideoToolbox 硬解。
- 修复了 Render API 默认模式下硬件帧无法经 software screenshot 转换的问题：启用 advanced control，并在主线程持有 GL context 处理 update callback；隐藏窗口也处理截图更新。
- 公开 API 与库配置不变。新增 readonly windowChrome 仅服务本地 HUD 的 Mac 标题避让。

## 后续边界

Mac 包含适配层和本地 UI，但用户仍需本机 libmpv。尚未验收自包含引擎包、Intel 实机、HDR/轨道/PiP、混合 DPI 多屏、整片音画同步；GIF 的业务调度复用既有验证，本次 Mac 专项验证至原图捕获，没有使用真实 Server 重新验完整 GIF 生成。Mac libmpv 位于主进程，崩溃隔离不同于 Windows 独立 mpv 进程。

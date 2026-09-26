# Curated Full 1.6.0

本次发行将 Curated 拆分为独立 Server 和 Desktop。完整安装包默认安装 Server 1.6.0 + Desktop 0.1.0；退出 Desktop 不再停止 Server。

- Windows x64：Full、Server、Desktop 各提供独立命名的 EXE 和 ZIP。Full EXE 内含原版组件安装器，Full ZIP 是离线安装套件。
- macOS：Desktop 0.1.0 仅支持 Apple Silicon，提供 DMG/ZIP；连接已有 Server，不含 Go、FFmpeg 或业务 Web UI。应用仅 ad-hoc 签名，未进行 Apple 公证。
- Server 与 Desktop 独立更新；Desktop 连接窗口在没有连接服务器时也能检查自身更新。Full 复用已安装的同版/更高版本组件。
- 所有下载的组件、版本、平台、架构与 SHA-256 记录在 `release.json`、组件 manifest 和 `SHA256SUMS.txt`。

Windows 推荐下载 `Curated-Full-Setup-1.6.0-windows-x64.exe`。仅运行服务端可下载 `Curated-Server-Setup-1.6.0-windows-x64.exe`；已有 Server 的电脑使用 `Curated-Desktop-Setup-0.1.0-windows-x64.exe`。Mac 使用 `Curated-Desktop-0.1.0-macos-arm64.dmg`。

旧客户端的 Latest 通道仍保留兼容一体包；本发行使用独立组件更新源，并不是旧一体包的自动覆盖安装。

## 旧版迁移

先创建并验证备份，记录数据目录、自定义配置和端口；停用旧登录启动，完整退出旧 Curated，再卸载旧程序并保留资料库。以原用户安装 Full，默认沿用 `%LOCALAPPDATA%\Curated`。如原先使用自定义 `CURATED_DATA_DIR` 或 `-config`，安装时使用 `/NOLAUNCH=1`，先带原配置启动 Server，再连接 Desktop。安装器检测到旧安装身份会阻止直接重叠安装；不会自动迁移另一账户或复制旧认证令牌。

详细说明及恢复步骤：[发布与迁移手册](https://github.com/yepHiu/Curated/blob/full-v1.6.0/docs/guide.md#8-release-and-packaging)。

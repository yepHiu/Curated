# Curated Full 1.7.1

包含 Server 1.7.0 与 Desktop 0.2.0。合并 Desktop/Server worktree 的全部提交，并保留已发布的独立组件安装、命名和更新通道。

- Desktop 使用新的本地连接页，支持最近连接、局域网发现、主题、此设备代理与登录自启动；退出 Desktop 不会停止 Server。
- Server 增加稳定身份与局域网发现。仅启用 LAN 访问后广播；可在 Server 本机关闭自动发现，重启生效。手动输入地址始终可用。
- 保留旧 Desktop 的服务器记录和会话；修复连接开启 PIN 的旧 Server、首次绑定新身份、开发版连接配置迁移的兼容性问题。
- 延续远程设置与本机原生操作限制，Server 和 Desktop 独立更新；旧一体包 Latest 通道不受影响。

## 下载

| 平台 | 安装包 | 便携 / 离线包 |
|---|---|---|
| Windows x64 Full | `Curated-Full-Setup-1.7.1-windows-x64.exe` | `Curated-Full-1.7.1-windows-x64.zip` |
| Windows x64 Server | `Curated-Server-Setup-1.7.0-windows-x64.exe` | `Curated-Server-1.7.0-windows-x64.zip` |
| Windows x64 Desktop | `Curated-Desktop-Setup-0.2.0-windows-x64.exe` | `Curated-Desktop-0.2.0-windows-x64.zip` |
| macOS Apple Silicon Desktop | `Curated-Desktop-0.2.0-macos-arm64.dmg` | `Curated-Desktop-0.2.0-macos-arm64.zip` |

Full Windows 安装器包含原版 Server/Desktop 独立安装器。全部资产版本、来源及 SHA-256 见 `release.json`、组件 manifest 和 `SHA256SUMS.txt`。

## 升级与兼容性

- **已拆分版 Server 1.6.0 / Desktop 0.1.0**：沿用原安装身份和位置，支持覆盖升级。CD 强制执行上一公开版安装、数据库完整性和升级检查；真实 Mac Desktop 升级验证连接记录、Cookie、本地存储保留。此次不新增 SQLite 迁移。升级前仍建议创建并验证备份，退出旧程序。
- **旧一体包 ≤1.5.8**：仍须手动迁移，不能直接重叠安装。先备份、停止旧启动项并完全退出，卸载程序时保留资料库，以原用户安装 Full。自定义 `CURATED_DATA_DIR` / `-config` 使用 `/NOLAUNCH=1`，先带原配置启动 Server。跨账户、自定义配置和旧共享会话不自动迁移。
- 新 Desktop 可连接旧 Server，包括开启 PIN 的 1.5.x/1.6.x；原 PIN 解锁机制不变。新 Server 保留旧 Desktop 的 health/桥接接口。未来不支持的协议会明确拒绝连接。
- Server UUID 保存在 `<databasePath>.server-id`，移动同一 Server 时一并保留；首次升级绑定保留会话，已绑定 UUID 后身份变化会要求确认并重新解锁。损坏身份文件不自动重置。
- 自动发现依赖同一局域网 IPv4 多播，防火墙/VLAN 不通时请手动连接。macOS 仅支持 Apple Silicon，使用 ad-hoc 签名，尚未 Apple 公证。

详细迁移与兼容性边界见 [使用手册](https://github.com/yepHiu/Curated/blob/full-v1.7.1/docs/guide.md)。

1.7.0 候选在 Windows 升级测试临时 SQLite 句柄清理处被拦截，未公开发布。本版修复测试句柄释放，重跑全部门禁；Server 1.7.0 与 Desktop 0.2.0 产品版本不变。

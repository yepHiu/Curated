[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "chinesesimp"; MessagesFile: "{#CuratedWindowsSupport}\languages\ChineseSimplified.isl"

[CustomMessages]
english.LaunchApp=Launch %1
chinesesimp.LaunchApp=启动 %1
english.LegacyDetected=An older all-in-one Curated installation was found. Use Curated Full 1.7.2 or later to back up and upgrade this installation automatically.
chinesesimp.LegacyDetected=检测到旧版 Curated 一体安装。请使用 Curated Full 1.7.2 或更高版本，自动备份数据并升级旧版本。
english.NewerInstalled=A newer Curated %1 is already installed. Downgrade was cancelled.
chinesesimp.NewerInstalled=已安装更新版本的 Curated %1，已取消降级安装。
english.UpgradeTitle=Upgrade your existing Curated
chinesesimp.UpgradeTitle=升级现有 Curated
english.UpgradeSubtitle=Keep your original library and settings
chinesesimp.UpgradeSubtitle=保留原有资料库和设置
english.UpgradeDetails=Confirm the original data directory (containing data and config). Setup will close the old Curated, verify a backup, uninstall the old program and install Server and Desktop. Windows may request permission to remove the old program. Your media files stay in place.
chinesesimp.UpgradeDetails=请确认原数据目录（包含 data 和 config）。安装程序将关闭旧版 Curated、创建并校验备份、卸载旧程序，然后安装 Server 和 Desktop。卸载旧程序时 Windows 可能请求管理员权限。您的媒体文件保持原位。
english.DataDirectory=Original data directory:
chinesesimp.DataDirectory=原数据目录：
english.ConfigTitle=Custom Server configuration
chinesesimp.ConfigTitle=自定义 Server 配置
english.ConfigSubtitle=Only needed if the old Server used -config
chinesesimp.ConfigSubtitle=仅在旧 Server 使用 -config 参数时需要
english.ConfigDetails=Select the same runtime configuration file if you previously started Server with -config. Otherwise leave this blank. Custom filesystem paths must be absolute and outside the old program directory.
chinesesimp.ConfigDetails=如果之前通过 -config 启动 Server，请选择原运行配置文件，否则留空。配置中的文件路径须为绝对路径，且位于旧程序目录之外。
english.ConfigFile=Original runtime configuration (optional):
chinesesimp.ConfigFile=原运行配置文件（可选）：
english.ConfigFilter=Configuration files|*.json;*.cfg;*.yaml|All files|*.*
chinesesimp.ConfigFilter=配置文件|*.json;*.cfg;*.yaml|所有文件|*.*
english.UpgradeFailed=The Curated upgrade could not finish. Resolve the following error and run Full again. Your migration records and backups are retained.
chinesesimp.UpgradeFailed=Curated 升级未完成。请处理以下错误后重新运行 Full。迁移记录和备份已保留。
english.VersionUnreadable=%1: installed version could not be read.
chinesesimp.VersionUnreadable=%1：无法读取已安装版本。
english.InstallerStartFailed=%1: could not start installer.
chinesesimp.InstallerStartFailed=%1：无法启动安装程序。
english.InstallerFailed=%1: installer failed with code %2.
chinesesimp.InstallerFailed=%1：安装失败，错误码 %2。
english.ServerFailed=Desktop was not changed. Existing components and library data were kept. Resolve the error and run Full again.
chinesesimp.ServerFailed=Desktop 未更改，现有组件及资料库数据已保留。请处理错误后重新运行 Full。
english.DesktopFailed=Server is installed. Existing components and library data were kept. Run Full again to retry Desktop.
chinesesimp.DesktopFailed=Server 已安装，现有组件及资料库数据已保留。请重新运行 Full 以重试 Desktop 安装。
english.CloseFailed=Could not close the running Curated. Exit it from the tray and retry. No program files have been replaced.
chinesesimp.CloseFailed=无法关闭正在运行的 Curated。请从托盘退出后重试。程序文件尚未替换。

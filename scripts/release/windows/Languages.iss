[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "chinesesimp"; MessagesFile: "{#CuratedWindowsSupport}\languages\ChineseSimplified.isl"

[CustomMessages]
english.LaunchApp=Launch %1
chinesesimp.LaunchApp=启动 %1
english.LegacyDetected=An older all-in-one Curated installation was found. Use the published Curated Full 1.7.3 once to migrate, then update Server and Desktop independently.
chinesesimp.LegacyDetected=检测到旧版 Curated 一体安装。请先使用已发布的 Curated Full 1.7.3 完成一次迁移，再分别更新 Server 和 Desktop。
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

english.InstallIntroduction=Install %1 %2. Choose the installation folder on the next page.
chinesesimp.InstallIntroduction=即将安装 %1 %2。您可以在下一页选择安装位置。

english.ExistingVersion=Installed version: %1%nVersion to install: %2
chinesesimp.ExistingVersion=已安装版本：%1%n本次安装版本：%2

english.TargetVersion=Version to install: %1
chinesesimp.TargetVersion=本次安装版本：%1

english.DesktopPurpose=Desktop connects to an existing Curated Server. It does not install or stop Server.
chinesesimp.DesktopPurpose=Desktop 用于连接已有的 Curated Server，不包含 Server。

english.ServerPurpose=Server manages your library and provides its web interface. Desktop is installed separately.
chinesesimp.ServerPurpose=Server 管理资料库并提供网页界面。Desktop 可另外安装。

english.ChooseDirectory=Choose a program folder. Library data and media are stored separately.
chinesesimp.ChooseDirectory=请选择程序安装位置。资料库数据和媒体文件与程序分开保存。

english.KeepDirectory=To update or reinstall, keep the existing folder:%n%1%nMoving an existing installation is not supported. Return to the folder page and use this location.
chinesesimp.KeepDirectory=升级或重新安装需沿用原位置：%n%1%n当前不支持直接迁移已安装程序。请返回目录页并使用原位置。

english.DirectoryNotEmpty=Choose an empty folder dedicated to this component. The selected folder already contains files.
chinesesimp.DirectoryNotEmpty=请选择此模块专用的空文件夹。所选文件夹已包含文件，不能直接覆盖。

english.FreshInstall=&Install
chinesesimp.FreshInstall=安装(&I)

english.UpgradeInstall=&Upgrade
chinesesimp.UpgradeInstall=升级(&I)

english.Reinstall=&Reinstall
chinesesimp.Reinstall=重新安装(&I)

english.DesktopKeepData=Saved Server connections and client settings are kept. Server is not updated.
chinesesimp.DesktopKeepData=将保留已保存的服务器连接和客户端设置，不会更新 Server。

english.ServerKeepData=Your existing library, settings and media files are kept. Desktop is not updated.
chinesesimp.ServerKeepData=将保留现有资料库、设置和媒体文件，不会更新 Desktop。

english.OverwriteQuestion=Continue to replace the installed program files with this version?
chinesesimp.OverwriteQuestion=是否继续覆盖原程序文件并升级到本次版本？

english.ReinstallQuestion=This version is already installed. Reinstall its program files without resetting user data?
chinesesimp.ReinstallQuestion=已安装相同版本。是否重新安装程序文件并保留用户数据？

english.RunningTitle=Close the running application
chinesesimp.RunningTitle=关闭正在运行的程序

english.RunningSubtitle=Program files will not be replaced until the application exits.
chinesesimp.RunningSubtitle=确认程序退出后才会替换文件。

english.RunningDetails=%1 is running, possibly in the tray. Select Close app to close it and continue installation, or exit it yourself and select Check again.
chinesesimp.RunningDetails=%1 正在运行，可能已隐藏到托盘。点击“关闭程序”退出并继续安装，或自行从托盘退出再点击“重新检查”。

english.ServerImpact=Clients will temporarily disconnect. Playback, uploads and background work may be interrupted. Finish current work before continuing.
chinesesimp.ServerImpact=客户端将暂时断开连接，播放、上传和后台任务可能中断。请完成当前操作后继续。

english.DesktopImpact=Save unfinished edits before continuing. Desktop windows and playback will close. Server will keep running.
chinesesimp.DesktopImpact=请先保存未完成的编辑。Desktop 窗口和播放将关闭，Server 会继续运行。

english.CloseContinue=&Close app
chinesesimp.CloseContinue=关闭程序(&I)

english.ContinueInstall=&Install
chinesesimp.ContinueInstall=继续安装(&I)

english.Recheck=Check again
chinesesimp.Recheck=重新检查

english.StoppedDetails=The application has exited. Continue to install.
chinesesimp.StoppedDetails=程序已退出，可以继续安装。

english.ProbeFailed=Unable to verify or close this installation safely. Check the details, exit it from its tray if necessary, then try again. No program files have been replaced.
chinesesimp.ProbeFailed=无法确认或关闭此安装实例。请查看详情，必要时从托盘退出后重新检查。程序文件尚未替换。

english.ClosingTitle=Closing the application
chinesesimp.ClosingTitle=正在关闭程序

english.ClosingDetails=Requesting a normal exit. Please wait. Program files have not been replaced.
chinesesimp.ClosingDetails=正在请求正常退出，请稍候。程序文件尚未替换。

english.RunningAgain=The application is running. Go back and continue to the close step. Silent installations require /CLOSECURATED=1 to allow shutdown. No files have been replaced.
chinesesimp.RunningAgain=程序正在运行，请返回并进入关闭步骤。静默安装需显式传入 /CLOSECURATED=1 才允许退出程序。文件尚未替换。

english.InstallationChanged=The installed version or location changed while Setup was open. Cancel and restart Setup to confirm the new installation.
chinesesimp.InstallationChanged=安装器打开期间，已安装版本或位置发生变化。请取消并重新运行安装器，确认新的安装信息。

english.InstallationUnreadable=The existing installation version or location cannot be read. No program was closed or replaced. Resolve the installation registration before retrying.
chinesesimp.InstallationUnreadable=无法读取现有安装的版本或位置，未关闭或替换程序。请检查安装登记信息后重试。

english.DowngradeDetails=Installed version: %1%nThis package: %2%nA newer version is already installed. Downgrade was cancelled.
chinesesimp.DowngradeDetails=已安装版本：%1%n本安装包版本：%2%n已安装较新版本，不能使用本安装包覆盖降级。

english.CancelledAfterStop=Setup has requested application shutdown. If you cancel, the original program files remain but the application may already be closed. You can reopen the original version from the Start menu.
chinesesimp.CancelledAfterStop=安装器已请求退出程序。取消后原程序文件仍在，但程序可能已经关闭，您可以从开始菜单重新打开原版本。

english.InvalidDirectory=The installation folder cannot be used. Choose a dedicated local folder and review the details below.
chinesesimp.InvalidDirectory=此安装位置不可用。请选择专用的本地文件夹，并查看下方原因。

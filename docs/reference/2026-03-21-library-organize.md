# 库目录整理与周期扫描

## 持久化：`config/library-config.cfg`

与 `-config` 指向的服务端主配置（HTTP 地址、数据库路径等）**分开**存放。文件为 **JSON**，用于可持久化的「库行为」开关；后续可把更多设置项合并进同一文件（写入时保留未知键）。

| 字段 | 说明 |
|------|------|
| `organizeLibrary` | 默认 **`true`**（若文件不存在或省略该字段，启动时也按 `true` 处理）。`true`/`false` 由前端 **Settings → 整理入库** 通过 `PATCH /api/settings` 更新，成功后**原子写回**本文件。 |
| `metadataMovieProvider` | 影片 Metatube 源；空字符串表示自动。由设置页或 `PATCH /api/settings` 更新。 |
| `autoLibraryWatch` | 默认 **`true`**。为 **`true`** 且主配置允许目录监听时，库根下新文件经 **fsnotify** 防抖后会触发与 **`POST /api/scans`** 同类的扫描链（任务元数据常带 `trigger: fsnotify`），并可能对新增条目排队刮削。为 **`false`** 时**不**因监听排队扫描；**手动扫描、周期 `autoScanIntervalSeconds` 全库扫描**不受影响。由设置页「自动刮削元数据」或 `PATCH /api/settings` 的 `autoLibraryWatch` 更新。 |
| `launchAtLogin` | 默认 **`false`**。由设置页「通用」或 `PATCH /api/settings` 更新；在支持的 Windows 运行时中会同步当前用户 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` 项，命令行为 `curated(.exe) -mode tray -autostart`。Windows 登录触发的这次启动会**静默进入托盘**，只拉起本地服务与托盘图标，**不会自动打开浏览器页面**。 |
| `logDir` | 后端启动后按日向该目录轮转写日志文件（与主配置 `logDir` 同源字段，由本文件合并覆盖）。空或省略表示使用默认目录，而不是关闭文件日志：**release** 默认 `LOCALAPPDATA\Curated\logs`，**dev** 默认 `backend/runtime/logs`。由设置页 **通用** 或 `PATCH /api/settings` 的 `backendLog` 更新；**重启后端**后 Zap 才按新目录/级别落盘。 |
| `logFilePrefix` | 日志文件名前缀，默认行为见 `internal/logging`（省略或空则使用 `curated`）。设置页不写入该键，需手写本文件或主配置。 |
| `logMaxAgeDays` | 日志文件保留天数；`0` 或省略时由日志模块使用默认（7 天）。 |
| `logLevel` | Zap 级别（如 `debug`/`info`/`warn`/`error`）；非法值会导致启动合并失败。 |

路径解析：在 `backend` 目录下启动时为 `../config/library-config.cfg`，否则为 `config/library-config.cfg`（相对当前工作目录）。

## 目录监听（fsnotify）与主配置

| 主配置字段 | 说明 |
|------------|------|
| `libraryWatchEnabled` | `null`/省略视为**开启**监听；显式 `false` 时**不**启动 fsnotify（与 `autoLibraryWatch` 无关，总闸在 yaml）。 |
| `libraryWatchDebounceMs` | 合并监听事件的防抖毫秒数；`0` 使用默认（约 1500ms）。 |

增删改库路径后，HTTP 层会尝试 **`ReloadLibraryWatches`**，使监听目录与数据库中的根路径一致。

## 配置（Curated 后端主 JSON，`-config`）

| 字段 | 说明 |
|------|------|
| `organizeLibrary` | 可选。若存在，先被读入主配置；启动时再由 `library-config.cfg` **覆盖**（若库设置文件含该键）。行为：`true` 时将视频整理到 `{父目录}/{番号}/{番号}.扩展名` 并写入 NFO/海报等到该番号目录；`false` 时资产在 `cacheDir/{movieId}/`。 |
| `autoScanIntervalSeconds` | 大于 `0` 时，按该间隔（秒）对库路径自动执行一次与 `POST /api/scans` 相同的全量扫描。`0` 表示关闭。 |

同一时间只允许 **一个** 库扫描在跑：手动 `POST /api/scans`、stdio `scan.start` 与周期扫描共用互斥；若已有扫描进行中再次触发，HTTP 返回 **409 Conflict**（`COMMON_CONFLICT`），stdio 返回对应错误响应。

## 风险

- 会 **移动/重命名磁盘上的视频文件**，请先备份重要数据。
- 若目标路径已存在其他文件，该条目会跳过并记录原因（避免覆盖）。
- 文件被播放器占用时移动可能失败。

## 番号识别

文件名清洗与解析见 `internal/scanner/number.go`（含站点前缀、`-C` 等后缀处理）。

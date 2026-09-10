# 库目录整理与周期扫描

## Comic library settings keys

The optional comic module uses separate keys in `config/library-config.cfg`; they do not replace or overload movie library settings.

| Field | Default | Meaning |
|------|---------|---------|
| `comicLibraryEnabled` | `false` | Enables comic routes and frontend comic navigation. When false, comic entry points stay hidden. |
| `autoComicLibraryWatch` | `true` | Enables independent fsnotify-driven comic scans when `comicLibraryEnabled=true` and the global `libraryWatchEnabled` gate permits watchers. |
| `defaultComicImportLibraryPathId` | empty | Comic library path id used by `POST /api/import/comics`. |
| `comicReader` | `{ "mode": "page", "fit": "contain", "direction": "ltr" }` | Global comic reader defaults. `mode` is `page` or `scroll`; `fit` is `contain` or `width`; `direction` is `ltr` or `rtl`. |
| `comicCache` | `{ "maxBytes": 2147483648 }` | Separate comic cache limit. `0` normalizes to the default 2GB; negative values mean unlimited. |

Comic archive behavior:

- Supported archive extensions are `.zip` and `.cbz`.
- Supported page image extensions are `.jpg`, `.jpeg`, `.png`, `.webp`, and `.gif`.
- The first naturally sorted image is the cover.
- Comic import copies archives into `defaultComicImportLibraryPathId`, never deletes the source archive, and never overwrites conflicts.
- Comic auto watch scans existing `.zip` / `.cbz` archives placed under configured comic roots. It does not copy files and does not watch movie library paths.
- Comic cache cleanup deletes only comic cache files/index rows and must not delete source `.zip` / `.cbz` archives.

## Photo library settings keys

The optional photo book module uses separate keys in `config/library-config.cfg`; they do not replace or overload movie or comic library settings.

| Field | Default | Meaning |
|------|---------|---------|
| `photoLibraryEnabled` | `false` | Enables photo routes and frontend photo navigation. When false, photo entry points stay hidden. |
| `autoPhotoLibraryWatch` | `true` | Enables independent fsnotify-driven photo scans when `photoLibraryEnabled=true` and the global `libraryWatchEnabled` gate permits watchers. `photowatch` listens to configured photo roots for `.zip` / `.cbz` changes and queues `scan.photos`. |
| `defaultPhotoImportLibraryPathId` | empty | Photo library path id reserved for future photo imports. |
| `photoViewer` | `{ "mode": "page", "fit": "contain", "direction": "ltr" }` | Global photo viewer defaults. `mode` is `page` or `scroll`; `fit` is `contain` or `width`; `direction` is `ltr` or `rtl`. |
| `photoCache` | `{ "maxBytes": 5368709120 }` | Separate photo cache limit. The cache endpoints and concrete cache cleanup are pending the photo content/backend slice. |

Photo library current slice:

- Configured roots are stored in the independent `photo_library_paths` table.
- Implemented APIs include `GET/POST/PATCH/DELETE /api/library/photos/paths`, `POST /api/library/photos/scans`, `GET /api/library/photos`, `GET /api/library/photos/{photoId}`, and page image/thumbnail routes under `/api/library/photos/books/{photoId}/...`.
- Scanned books/pages are stored in independent `photo_books`, `photo_pages`, `photo_tags`, `photo_book_tags`, `photo_viewing_progress`, `photo_viewing_preferences`, and `photo_cache_entries` tables.
- Photo path changes call independent `ReloadPhotoLibraryWatches`; the photo watcher must not touch comic watchers or queue `scan.comics`.

## 持久化：`config/library-config.cfg`

与 `-config` 指向的服务端主配置（HTTP 地址、数据库路径等）**分开**存放。文件为 **JSON**，用于可持久化的「库行为」开关；后续可把更多设置项合并进同一文件（写入时保留未知键）。

| 字段 | 说明 |
|------|------|
| `organizeLibrary` | 默认 **`true`**（若文件不存在或省略该字段，启动时也按 `true` 处理）。`true`/`false` 由前端 **Settings → 整理入库** 通过 `PATCH /api/settings` 更新，成功后**原子写回**本文件。 |
| `metadataMovieProvider` | 影片 Metatube 源；空字符串表示自动。由设置页或 `PATCH /api/settings` 更新。 |
| `defaultImportLibraryPathId` | 默认导入目标库路径 id。由设置页「影片存储」或 `PATCH /api/settings` 更新；`POST /api/import/movies` 会把浏览器选择的影片复制到该库根目录下，不移动或删除源文件。 |
| `autoLibraryWatch` | 默认 **`true`**。为 **`true`** 且主配置允许目录监听时，库根下新文件经 **fsnotify** 防抖后会触发与 **`POST /api/scans`** 同类的扫描链（任务元数据常带 `trigger: fsnotify`），并可能对新增条目排队刮削。为 **`false`** 时**不**因监听排队扫描；**手动扫描、周期 `autoScanIntervalSeconds` 全库扫描**不受影响。由设置页「自动刮削元数据」或 `PATCH /api/settings` 的 `autoLibraryWatch` 更新。 |
| `autoComicLibraryWatch` | 默认 **`true`**。为 **`true`** 且漫画库已启用、主配置允许目录监听时，漫画存储路径下新增或变更的 `.zip` / `.cbz` 会经独立漫画 watcher 防抖后触发 `scan.comics`（任务元数据常带 `trigger: fsnotify`）。为 **`false`** 时不因监听排队漫画扫描；手动漫画扫描与漫画导入后的扫描不受影响。 |
| `autoPhotoLibraryWatch` | 默认 **`true`**。为 **`true`** 且写真库已启用、主配置允许目录监听时，独立写真 watcher 会监听写真存储路径下新增或变更的 `.zip` / `.cbz`，并经防抖后触发 `scan.photos`。 |
| `photoLibraryEnabled` / `defaultPhotoImportLibraryPathId` / `photoViewer` / `photoCache` | 写真库开关、未来默认导入目标、浏览器默认设置和缓存上限；由设置页「写真库」或 `PATCH /api/settings` 更新。写真路径列表来自独立 SQLite 表 `photo_library_paths`。 |
| `launchAtLogin` | 默认 **`false`**。由设置页「通用」或 `PATCH /api/settings` 更新；在支持的 Windows 运行时中会同步当前用户 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` 项，命令行为 `curated(.exe) -mode tray -autostart`。Windows 登录触发的这次启动会**静默进入托盘**，只拉起本地服务与托盘图标，**不会自动打开浏览器页面**。 |
| `autoDownloadUpdates` | 默认 **`false`**。由设置页「通用」或 `PATCH /api/settings` 更新；开启后，启动阶段的后台更新检查若发现较新的 installer，会自动下载并完成 SHA256 校验；安装仍须用户在 Settings -> About 显式确认，不会自动静默安装。 |
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

增删改影片库路径后，HTTP 层会尝试 **`ReloadLibraryWatches`**；增删漫画库路径后会尝试 **`ReloadComicLibraryWatches`**，使监听目录与数据库中的根路径一致。增删写真库路径后会尝试 **`ReloadPhotoLibraryWatches`**，由独立 `photowatch` 重建写真存储路径监听并在新增或变更归档时排队 `scan.photos`。

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

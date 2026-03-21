# 库目录整理与周期扫描

## 持久化：`config/library-config.cfg`

与 `-config` 指向的服务端主配置（HTTP 地址、数据库路径等）**分开**存放。文件为 **JSON**，用于可持久化的「库行为」开关；后续可把更多设置项合并进同一文件（写入时保留未知键）。

| 字段 | 说明 |
|------|------|
| `organizeLibrary` | 默认 **`true`**（若文件不存在或省略该字段，启动时也按 `true` 处理）。`true`/`false` 由前端 **Settings → 整理入库** 通过 `PATCH /api/settings` 更新，成功后**原子写回**本文件。 |

路径解析：在 `backend` 目录下启动时为 `../config/library-config.cfg`，否则为 `config/library-config.cfg`（相对当前工作目录）。

## 配置（`javd` 主 JSON，`-config`）

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

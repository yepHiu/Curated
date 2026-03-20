# 库目录整理与周期扫描

## 配置（`javd` JSON）

| 字段 | 说明 |
|------|------|
| `organizeLibrary` | `true` 时：扫描识别番号后，将视频移动到 `{视频所在目录}/{番号}/{番号}.扩展名`；刮削后的 **NFO** 与 **海报/预览图** 写入该番号目录。`false` 时保持旧行为（资产在 `cacheDir/{movieId}/`）。 |
| Web Settings | 前端 **Settings → 库目录整理** 开关调用 `PATCH /api/settings`，在**进程内存**中覆盖上述行为；重启 `javd` 后恢复为配置文件中的值。 |
| `autoScanIntervalSeconds` | 大于 `0` 时，按该间隔（秒）对库路径自动执行一次与 `POST /api/scans` 相同的全量扫描。`0` 表示关闭。 |

同一时间只允许 **一个** 库扫描在跑：手动 `POST /api/scans`、stdio `scan.start` 与周期扫描共用互斥；若已有扫描进行中再次触发，HTTP 返回 **409 Conflict**（`COMMON_CONFLICT`），stdio 返回对应错误响应。

## 风险

- 会 **移动/重命名磁盘上的视频文件**，请先备份重要数据。
- 若目标路径已存在其他文件，该条目会跳过并记录原因（避免覆盖）。
- 文件被播放器占用时移动可能失败。

## 番号识别

文件名清洗与解析见 `internal/scanner/number.go`（含站点前缀、`-C` 等后缀处理）。

# 导入后扫描因 UNIQUE(location) 中断

- 日期：2026-08-23
- 状态：开发仓库已落地（方案 1+2）；方案 3（导入后优先扫刚落地文件）未做
- 环境：桌面正式包 Curated 1.5.4（`BuildStamp=20260817.165606`）仍含该缺陷，需发版后才会在正式库生效

## 结论

导入任务本身会把文件复制进资料库并标记 `import.movies` 成功。真正丢片发生在随后的 `scan.library`：扫描写库时撞上 `movies.location` 的 UNIQUE 约束，整次扫描按 walk 失败返回，排在冲突文件之后的新片永远不会入库。

## 正式版证据

今日两次导入（`C:\Users\wujiahui\AppData\Local\Curated\logs\curated-20260823.log` + `curated.db`）：

| 时间 | 导入任务 | 扫描结果 | 用户可见结果 |
|---|---|---|---|
| 01:08–01:14 | `import.movies-1787418532407959500` 成功（2 个大文件） | 立刻 `scan.library` → walk 失败：`UNIQUE constraint failed: movies.location (2067)` | 文件在磁盘，库里没有 |
| 15:09–15:10 | `import.movies-1787468947879706200` 成功 | 同上错误，但 `JUR-098` 已先入库并刮削 | 看起来「这次成功了」 |

磁盘 `E:\JAV\Curated` 上未入库的视频：

- `hhd800.com@JUR-530.mp4`
- `SONE-305-C.mp4`
- `SONE-521-C.mp4`

冲突文件（扫描字典序第 161 个，正好是失败点）：

- 路径：`E:\JAV\Curated\SIRO-5705\SIRO-5705.mp4`（仍在磁盘）
- 数据库：`movies.id=siro-5705`，`trashed_at=2026-08-18T15:39:42Z`

`PersistScanMovie` 查 location/code 时排除了回收站行，随后又 INSERT 同一路径，触发 UNIQUE，扫描中止。`JUR-098` 排在 `SIRO-5705` 之前所以能入库；`SONE-*` / `hhd800.com@*` 排在之后所以不能。

自 2026-05 起该错误已出现 137 次，1.5.4 之后每次扫描几乎都失败，只是日志里没有 Error 行，只写在 `scan_jobs` 表。

## 已落地（开发仓库）

1. 单文件 persist 失败改为 skip `persist_failed` + Error 日志，`OnFileDetected` 返回 nil，整次扫描继续。整次 `Scan()` 失败时打 `scan.library failed` Error 日志。
2. `PersistScanMovie` 按 location/code 查找时包含回收站行：
   - 回收站占用路径：skip `trashed_already_indexed`（同番号）或 `trashed_path_indexed`（不同番号）
   - 回收站占用番号：skip `trashed_code_indexed`
   - INSERT/UPDATE 仍撞 UNIQUE：skip `unique_constraint`
   - **不自动恢复**回收站影片
3. 测试：`TestPersistScanMovieSkipsTrashedPathAndAllowsLaterImport`、`TestIntegration_RunScan_TrashedLocationDoesNotAbortLaterImports`（ABC-100 + 已回收 SIRO-5705 + SONE-305-C，期望后两部可入库且任务 completed）。`go test ./internal/storage/ ./internal/app/` 已通过。

## 未做

- 导入提交后优先扫描刚落地的文件（加固项，不阻塞本修复）。

## 正式库后续

发版前，本机正式库仍会被 SIRO-5705 回收站占用路径卡住。修好的开发后端发版或对正式库再扫一次后，那 3 个已在磁盘上的文件才会入库。在那之前也可以从回收站彻底删除 SIRO-5705（或改掉它占用的 `location`）后再手动扫描。

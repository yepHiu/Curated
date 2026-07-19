# Curated Data Longevity 实施计划

日期：2026-07-20
状态：in-progress
关联需求：REQ-0014～REQ-0018
上位计划：`docs/plan/2026-07-19-project-feature-quality-audit.md`

## 执行进度

| 切片 | 状态 | 证据 |
|---|---|---|
| C1 备份核心与离线 CLI | verified | `bd2e9b59`、`4d0fd1f0`；storage / backup / maintenance / cmd 测试与全量 Go test/vet 通过 |
| C2 Settings Maintenance | verified | `e7958490`；受 PIN 保护的 create / verify / preflight API、Web/Mock service contract、三语 Settings UI、目标 Vitest、全量 167/684 Vitest、4/28 Electron、4 项 Chromium e2e、typecheck/lint/build 与全量 Go test/vet 均通过 |
| C3 路径迁移 CLI | verified | `1d9b2070`；只读 plan、7 列白名单、Windows/UNC/Unix 与跨平台映射、目标/冲突检查、迁移前已验证备份、单事务 apply + audit、binding reset、目标/全量 Go test/vet 与真实临时 SQLite CLI 演练通过 |
| C4 上传 session 持久化 | verified | `3e77766e`；SQLite session/file/chunk ledger、重启恢复、提交中断协调、24h sliding TTL、审计 janitor、离线目标延迟与全量 Go/前端/Electron/e2e/build 门禁通过 |
| C5 Library Health / 修复队列 | not-started | REQ-0017、REQ-0018 |

## 1. 目标与边界

Milestone C 的完成条件不是“增加几个维护按钮”，而是证明以下长期数据承诺成立：

1. 用户能够为运行中的 SQLite 创建一致备份，并验证备份未损坏、未缺文件、迁移版本可识别。
2. 恢复前能够看到兼容性、目标路径、覆盖与回滚风险；真正替换 SQLite 只允许在 HTTP/Electron 业务服务启动前执行。
3. 库根或挂载路径变化时，可以先 dry-run，再在事务内迁移已存路径，并保留可审计结果。
4. 大文件上传 session 在后端重启后仍可恢复；过期或孤立 staging 有统一 janitor，而不是静默堆积。
5. Library Health 能集中展示数据库、存储、源文件、资源、导入残留和元数据失败，并通过任务化修复队列执行安全行动。

默认备份不包含媒体源文件。第一版备份范围是 SQLite 与存在时的 `library-config.cfg`；媒体资产范围在 manifest 中显式标为未包含，后续再增加可选用户资产范围。

## 2. 备份格式与恢复安全

备份文件扩展名使用 `.curated-backup`，内部为 ZIP：

```text
manifest.json
database/curated.db
config/library-config.cfg   # 原文件存在时
```

`manifest.json` 至少记录：格式版本、创建时间、Curated build/channel、SQLite schema migration 列表、每个条目的相对路径、字节数和 SHA-256，以及媒体/用户资产是否包含。

创建数据库快照使用 SQLite `VACUUM INTO`，不得在有写入活动时直接复制数据库文件。验证必须同时完成：

- ZIP 路径和条目唯一性检查，拒绝绝对路径与 `..`；
- manifest schema 与必需条目检查；
- 大小与 SHA-256 检查；
- 临时打开备份数据库并执行 `PRAGMA quick_check`；
- 执行 `PRAGMA foreign_key_check`；
- 读取 `schema_migrations` 并与 manifest 一致性比较。

恢复流程先做 preflight。preflight 不写目标，返回备份验证结果、目标是否存在、所需字节数、迁移版本差异和警告。真正恢复必须离线执行，并遵循：同目录临时文件 → 当前文件回滚副本 → 原子替换 → 重新验证；任何失败都尽可能恢复旧文件。

## 3. 实施切片

### C1：备份核心与离线 CLI

- `internal/storage` 增加一致 SQLite snapshot 能力。
- 新增聚焦的 `internal/backup` 包，负责 package create / verify / preflight / restore。
- `cmd/curated` 只解析维护参数，把执行委托给内部维护 runner。
- CLI 支持 create、verify、preflight、restore；restore 需要显式确认参数。
- 使用临时目录测试篡改 checksum、损坏 DB、路径穿越、版本不一致和恢复回滚。

### C2：Settings Maintenance

- 增加受 PIN 保护的 create / verify / preflight API；不增加在线替换数据库端点。
- Electron 目录选择只用于选择备份位置，不扩大 preload 业务能力。
- Settings 展示备份范围、时间、大小、schema、校验结果和“恢复需重启到维护模式”的明确提示。

### C3：路径迁移 CLI

- 枚举所有存储绝对路径的表/列，建立白名单，不对任意文本字段做字符串替换。
- `dry-run` 输出每个表的影响数量、样例、冲突、越界与不存在目标。
- `apply` 在单事务中执行，并把操作摘要写入迁移审计记录或外部结果文件。
- Windows 路径比较处理盘符大小写和分隔符；禁止部分目录名误匹配。

完成结果（2026-07-20）：

- 已实现 `path-migrate-plan` / `path-migrate-apply`，两者均要求 Curated 完全退出并取得运行时数据库锁；plan 不运行 schema migration，也不写数据库。
- 白名单覆盖 `library_paths.path`、`movies.location`、`scan_items.path`、`media_assets.local_path`、`actors.avatar_local_path`、`library_path_storage_bindings.root_path`、`app_update_status.downloaded_file_path`；自由文本、URL 与历史审计不参与替换。
- Windows drive / UNC 比较大小写不敏感并统一分隔符，Unix 保持大小写；按路径段匹配，支持 Windows→Unix，拒绝相对路径、`..`、等价前缀和目标嵌套在源下的危险映射。
- plan 结构化返回各列影响数、空值/越界/非法值、目标 missing/unchecked/error、唯一性冲突、最多 5 个样例、errors/warnings 与 `canApply`；输出顺序稳定。
- apply 要求新备份目标与 `-confirm-path-migration`，先创建并验证迁移前 `.curated-backup`，再于单事务内重新 plan、更新路径、清除旧存储 binding、运行 `quick_check` / `foreign_key_check`、创建并写入 `path_migration_audits`；审计写入失败测试证明全部路径更新回滚。
- missing/unchecked 默认阻止；人工确认跨平台布局后可显式 `-allow-missing-paths`，但 wrong-type、I/O error 和冲突始终不可覆盖。
- 真实临时 SQLite 演练证明 plan 后原路径与 audit count 保持不变；apply 后 3 行更新、binding 清零、审计落库，独立 `backup-verify`、`quick_check` 和 `foreign_key_check` 通过。

### C4：上传 session 持久化

- SQLite 保存 upload、file、chunk 范围、状态、目标根、staging 路径、字节计数与时间戳。
- 每次创建、chunk 落盘、commit/abort 状态转换都以可恢复顺序持久化。
- 启动时重建 session；校验 staging 文件实际大小，不信任仅数据库记录。
- janitor 只清理明确过期、已 abort/commit 或无法恢复且已记录诊断的目录，不扫描和删除任意隐藏目录。

完成结果（2026-07-20）：

- migration `0029_movie_import_upload_sessions.sql` 新增 `movie_import_upload_sessions`、`movie_import_upload_files`、`movie_import_upload_chunks` 与 `movie_import_upload_cleanup_audits`；repository 以事务维护 chunk 去重/重叠约束、文件与会话计数、条件过期、状态转换和 cleanup audit。
- chunk 写入顺序为目标范围写盘、边界校验、`Sync`、`Close`，再写 SQLite chunk row 并更新计数；启动恢复从 chunk 行重新推导 received bytes，不把预分配文件长度当成完成证据。写盘后、记账前退出时，客户端可安全覆盖式重传同一范围。
- 启动恢复 `uploading` / `committing` session、原 task ID、进度和 metadata；提交逐文件记录 committed marker，可继续 staging 尚存的移动，也可识别 final 已移动但 marker 尚未写入的中断窗口。冲突文件不覆盖，无法安全协调的状态标为 `unrecoverable`。
- 目标盘离线时 session 保留并暴露 `recoveryStatus=unavailable`；目标恢复后 GET/PUT/commit 可重新协调。janitor 不在离线时谎报 staging 已移除，恢复在线后才审计清理。
- 默认 active TTL 24h、janitor interval 15m、orphan grace 24h、terminal cleanup delay 1m。janitor 只清理 SQLite 登记的终态/确实过期 session，或严格位于配置库根下且命名为 `.curated-import/upload_<16 lowercase hex>` 的旧孤立目录；symlink、无效命名、新鲜 orphan、任意隐藏目录和最终目标文件均不删除。
- manifest 限制为 2 MiB、最多 10,000 文件，拒绝未知字段、尾随 JSON、重复目标路径、空文件和总量溢出；DTO 同步 `expiresAt`、`recoveryStatus`、`recoveryError` 与 file `state`，并增加 `IMPORT_UPLOAD_PERSIST_FAILED`、`IMPORT_UPLOAD_UNRECOVERABLE`、`IMPORT_UPLOAD_EXPIRED`。
- 验证证据：`go test ./...`、`go vet ./...`、`npx -y pnpm@11.0.0 typecheck`、`lint`、`test`（167 files / 684 tests）、`test:electron`（4 files / 28 tests）、`test:e2e`（4 Chromium tests）、`build`、`build:electron:main` 全部通过。`go test -race` 因本机未启用 CGO 未运行成功，因此不列为通过证据。

### C5：Library Health 与修复队列

- 第一版检测：SQLite quick/foreign-key、异常存储绑定、丢失/不可读/零长度源文件、缺资源、重复、孤儿用户状态、失败刮削与 `.curated-import` 残留。
- 检测与修复分离；默认只读，不在扫描阶段自动删除或覆盖。
- 修复使用现有 task lifecycle，支持重新扫描、重新刮削、清理已确认孤儿和导出诊断。

## 4. 验证与提交策略

每个切片必须：

- 在 `backend/` 运行目标包测试、`go test ./...` 与 `go vet ./...`；
- 若有前端，运行 typecheck、lint、目标 Vitest、全量 Vitest、e2e 与 build；
- 更新 `API.md`、README 三语版、`CLAUDE.md`、`project-facts.mdc` 和架构 HTML 中受影响的端点与事实；
- 精确暂存并按最小行为单元提交；未经用户明确要求不 push。

Milestone C 最终审计需要逐条对应 REQ-0014～REQ-0018 的 acceptance criteria 和权威测试证据，不能用“相关测试大多通过”替代逐项证明。

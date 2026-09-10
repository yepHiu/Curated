# 漫画库与写真库 Beta 主线整合

日期：2026-09-10

状态：verified

关联需求：REQ-0046、REQ-0047

## 用户确认的范围

将 `codex/comic-library-mvp` 中已提交漫画库和工作区未提交的漫画增强、写真库实现合入 master，以 Beta 供试用。漫画和写真库各有默认关闭的开关，统一位于「设置 → 实验性功能」。开启后才显示该库的路径、阅读/浏览、缓存配置，不新增正式设置导航项。

## 整合边界

- 保留原实验 worktree 的工作文件、分支和暂存区；独立 index 保存源快照至 `codex/comic-photo-beta-source`，在 `codex/comic-photo-beta` worktree 整合当前 master。
- 复用服务端 `comicLibraryEnabled` / `photoLibraryEnabled` 持久化设置，允许先启用再添加路径。扫描仍需有效已配置路径；关闭隐藏入口和配置，并停止相应目录监听，不删除内容或路径。
- 保留主线 AI、影片、消息中心、鉴权与备份行为；旧设置链接改向 experimental。漫画/写真使用独立服务、端点、表，不能混入 Movie。
- 新迁移使用 0046–0048，不复用主线 0026–0028 编号。原 SQL 的 IF NOT EXISTS 保留对已有实验表的兼容；不复制或迁移原 worktree 的运行数据库、密钥和真实媒体。
- 漫画支持 ZIP/CBZ 扫描、导入、阅读、进度/偏好、收藏评分标签、缓存；写真支持已实现的压缩包扫描、列表/详情与浏览。写真缩略图当前复用原图，缓存设置保留为 Beta 配置，尚无独立写真缓存清理实现。

## 验证与交付

按仓库统一命令执行类型检查、lint、前端及后端测试、vet、构建与相关浏览器回归。用临时合成压缩包和 SQLite 验证启用 → 配置路径 → 扫描 → 阅读 → 关闭 → 重启后保留。保留现有 bundle budget，不重跑未经要求的 display-scaling 长套件。最终合入 master，仅提交本任务，不 push。

### 验收记录（2026-09-10）

- 前端全量 Vitest：259 文件 / 1200 测试通过；后续新增拼音提取 3 项及写真返回导航 6 项通过。最终受影响文件复验 53 项通过，导航与拼音复验 25 项通过。
- `pnpm typecheck`、`pnpm lint`、`pnpm build` 通过；`pnpm test:e2e --workers=1` 全部 5 项通过（含 PIN 启动保护、Mock 边界、375px 和备份流程）。新增媒体 settings 读取已纳入解锁后请求断言，锁屏前无新增受保护请求。
- 后端 `go test ./...`、`go vet ./...` 通过。
- 在 8186 独立临时后端、SQLite 和生产前端上，用两份各含 3 张合成 PNG 的 CBZ 经 UI 添加路径、扫描。漫画/写真列表、详情、阅读/浏览和翻页成功；图片实际加载。修复写真顶栏返回到来源详情/写真库。
- 默认关闭时路径、阅读/浏览、缓存配置和两个侧栏入口均隐藏；无路径时可启用。关闭后两个内容 API 返回 HTTP 400 及各自 DISABLED 错误，直接访问 `/comics`、`/photos` 均回到 experimental。
- 再次开启并重启临时后端后，两个启用标志、各 1 条路径、各 1 本 3 页内容、漫画第 2 页进度（index=1）均保留。旧 `section=photos` 链接也会打开 experimental。
- 桌面和 375px 无横向溢出。截图及临时日志保存在整合 worktree 的 `.workspace/beta/`；未对真实媒体或用户运行数据库做操作。
- 拼音纯数据提取后，生产浏览器实测“中国 / zhongguo”和“重庆 / cq”均命中 `[0,1]`，无关查询返回 `null`。字典作为内容哈希 JSON 随 dist 发布，保持原算法；这是减少可执行 JS，不是删除词条或减少全部数据。未提高任何构建预算。

### 交付提交

- `b31e7a35`：保留源 worktree 历史的 Beta 整合 merge commit。
- `28f6c9f8`：拼音词典数据文件与 Mock 适配器构建裁剪。
- 文档、PRD 台账与验收记录单独提交；随后 fast-forward 到 master，不 push。
- 原始 `codex/comic-library-mvp` 工作区和未提交修改保留；源快照为 `b1108e02`。不恢复该工作区里与本次媒体 Beta 无关的影片详情尺寸修改。

最终生产包（raw / gzip 字节）：初始 JS {'rawBytes': 395391, 'gzipBytes': 141145}；总 JS {'rawBytes': 2125381, 'gzipBytes': 652825}。

# 「添加影片」保留并完善 — 实施计划

- 日期：2026-08-17
- 决策来源：`2026-08-17-movie-import-optimization-research.md` §8（保留功能 C2 路线 + 入口弱化；LAN 远程上传真实但低优先级）。
- 原则：不新增大块架构；让既有断点续传后端（4 张表、janitor、恢复机制）第一次产出真实用户价值；顺手修掉调研发现的缺陷。
- **进度（2026-08-17）：M1、M2、M3 已完成并通过全量测试**（后端 `go test ./...` / `go vet`，前端 typecheck / lint / Vitest 全绿）。实现比计划更省的两处：分块明细直接来自内存 session 的 chunks 映射（恢复流程会重建），未新增 repository 查询；空间探测抽为共享包 `internal/diskutil`，backup preflight 同步改用。
- **M3 落地摘要**：会话账本 `src/lib/movie-import-upload-ledger.ts`（localStorage `curated-movie-import-uploads-v1`，24h TTL 修剪、损坏载荷自清理、`relativePath+size+lastModified` 指纹精确集合匹配）；`endpoints.ts` 的 `importMovies` 新增 `resumeUploadId`（GET 状态 → 校验 `uploading` → 按 `chunks` 明细跳块 + 从 `bytesReceived` 起算进度 → commit）与 `onUploadSessionCreated` 回调，并暴露 `getMovieImportUpload` / `deleteMovieImportUpload`；服务层契约新增 `listResumableMovieImports()` / `abandonMovieImportUpload()`（Web 适配器实现账本对账与 404/终态清理，Mock 恒空）；`MovieImportDialog` 打开时加载"可继续的上传"（文件数/大小/已接收百分比/有效期），选中文件与会话指纹完全匹配时展示提示并把提交切到续传，放弃采用两次点击确认。跳块前校验本地文件 size 与会话记录一致、已存分片 offset/size 与本地切块布局一致，不一致时明确报错引导新建导入。M4（并发分块、统一进度视图）未开始。

## 里程碑总览

| 里程碑 | 内容 | 规模 | 价值 |
|---|---|---|---|
| M1 | 入口弱化（按钮样式） | 极小（1 文件 + 快照更新） | 降低入口存在感 |
| M2 | 后端第一批：session 状态暴露分块明细 + 磁盘空间预检 + 扩展名统一 | 中（3 个后端改动点） | 修复静默失败缺陷；为续传铺路 |
| M3 | 前端续传完整实现（会话账本 + 恢复 UI + 精确跳块） | 大（前端核心工作） | 断点续传兑现价值 |
| M4 | 可选增强：并发分块上传；导入→扫描→刮削统一进度视图 | 中 | 性能与体验 |

依赖关系：M3 的"精确跳块"依赖 M2 的分块明细接口；M1、M2 相互独立可并行。

---

## M1 入口弱化

现状：`src/components/jav-library/MovieImportDialog.vue:233-239` 触发按钮为 `variant="default"`（`bg-primary` 主题色），是顶栏右侧按钮组（`AppShell.vue:887-905`）里唯一非 ghost 的按钮。

方案：与旁边主题切换按钮同一视觉语言——

```vue
<Button
  data-import-trigger
  type="button"
  variant="ghost"
  class="min-h-11 rounded-full text-muted-foreground hover:bg-muted/70 hover:text-foreground lg:min-h-9"
  :aria-label="t('import.trigger')"
>
```

- 忙碌态进度环同步调整：`text-primary-foreground` → `text-foreground`、轨道 `text-primary-foreground/35` → `text-foreground/35`（ghost 底上 primary-foreground 不可见）。
- 备选：`variant="secondary"`（`bg-secondary`，弱化但仍为填充按钮）——若 ghost 在实际视觉中过弱再切换，改动同样只有一行。
- 验收：亮/暗主题下按钮不抢视觉焦点；hover 有反馈；`MovieImportDialog.test.ts` 断言 `variant` 的用例同步更新。

## M2 后端第一批

### M2.1 session 状态暴露分块明细（M3 的前置）

现状：`GET /api/import/movies/uploads/{uploadId}`（`movie_import_upload_handlers.go`）只返回每文件 `bytesReceived`/`complete`（`contracts.go:972-980`），前端无法得知哪些块已落盘，只能假设"连续前缀"。

改动：
- `MovieImportUploadFileDTO` 增加 `Chunks []MovieImportUploadChunkDTO`（`index/offset/size`），GET 与 create 响应均填充。
- `storage/movie_import_upload_repository.go` 新增 `ListMovieImportUploadChunks(uploadID, fileID)`（表 `movie_import_upload_chunks` 已有 `idx_movie_import_upload_chunks_range`）。
- 量级安全：chunk 数 = ceil(size/32MB)，100GB 文件约 3,200 条，JSON 体积可接受；如担心可在 DTO 上懒加载（仅 GET 时填充）。
- `src/api/types.ts`、`guards.ts` 同步 DTO。

### M2.2 磁盘空间预检

现状：multipart 与 session 两条路径都**没有**事前空间检查，写满后靠错误字符串分类成 `IMPORT_NOT_ENOUGH_SPACE`；`createSizedStagingFile` 的 truncate 预留在 Windows 上不保证分配。

改动：
- 复用 `internal/backup` 的磁盘空间探测（`disk_space_windows.go` / `disk_space_unix.go`）。
- multipart 入口：解析 `totalBytes` 后、写第一个文件前检查目标根剩余空间 > `totalBytes + margin`（margin 建议 512MB），不足直接 400 `IMPORT_NOT_ENOUGH_SPACE`。
- session create：manifest 总量同样预检（现已有 `validateMovieImportUploadRecord` 汇总量，复用其结果）。
- 注意：预检是尽力而为（多任务并发仍可能写满），保留既有事后错误分类。

### M2.3 扩展名白名单统一（缺陷修复）

现状四处三套（详见调研 §3.1）：scanner 5 种（`scanner/service.go:37` 硬编码）、watcher 5 种（`librarywatch/watcher.go:50`）、后端导入 15 种（`server.go:2785`）、前端 16 种（`MovieImportDialog.vue:37`）。

改动：
- 新建单一事实来源：`backend/internal/contracts`（或独立小包）导出 `SupportedVideoExtensions`；scanner `NewService`、watcher `videoExtensions`、`isSupportedImportVideoPath` 全部引用；前端在 `src/api/types.ts` 或独立常量文件镜像同一列表（构建期无法共享 Go 常量，用注释锚点双向标注）。
- **对齐方向：向上对齐到导入的 15 种**。理由：播放管线基于 ffmpeg 探测容器，不依赖扩展名；番号提取只读文件名；导入已向用户承诺这 15 种。`.iso` 保留但需在联调中验证播放（ffmpeg 对 ISO9660/UDF 支持有限），若不可播则从统一列表移除并同步导入文案。
- 安全性确认：staging 文件名 `<fileID>.part`、multipart 临时文件 `.<taskID>.tmp` 均被 watcher 的 `.part/.tmp`/隐藏前缀过滤排除，扩展名扩围不影响这两条防误扫机制。
- 验收：导入 `.rmvb`/`.wmv` 后扫描能入库；手动复制同扩展名进库根能被监听收录；三处来源改动都有单测。

## M3 前端续传完整实现（核心）

### 设计概要

会话账本 + 指纹匹配 + 精确跳块。不引入 File System Access API 作为 v1 依赖（见 M3.5）。

**1) 会话账本（localStorage `curated-movie-import-uploads-v1`）**

结构（仅元数据，KB 级）：

```jsonc
[{
  "uploadId": "upload_ab12...",
  "targetLibraryPathId": "3",
  "chunkSize": 33554432,
  "files": [{ "relativePath": "ABC-123.mp4", "size": 8589934592, "lastModified": 1755400000000 }],
  "createdAt": "...", "lastActiveAt": "..."
}]
```

- 由 `src/lib/movie-import-upload-ledger.ts`（新文件）管理：add / remove / list / touch。
- 写入时机：`shouldUseResumableImport` 命中并成功 create session 后。
- 移除时机：commit 成功、DELETE 中止、GET 到终态（committed/aborted/expired/unrecoverable）、账本条目超过 24h 未活跃（与后端 TTL 对齐）。

**2) 恢复入口 UI（`MovieImportDialog.vue`）**

- 对话框打开时：读账本 → 逐条 `GET /uploads/{uploadId}` → `state === "uploading"` 且未过期的条目展示"可继续的上传"区块：文件数、总大小、目标路径、`expiresAt` 倒计时、已接收百分比（`bytesReceived/totalBytes`）。
- 每条提供两个动作：
  - **继续**：进入文件选择，用户重新选中相同文件（浏览器安全模型决定刷新后无法保留 `File` 引用）。
  - **放弃**：`DELETE /uploads/{uploadId}` + 账本移除（后端会 `Fail(IMPORT_CANCELLED)` 并由 janitor 清 staging）。
- 选中文件后指纹匹配（`relativePath + size + lastModified` 三元组，与后端 manifest 字段一致）：
  - **完全匹配**（文件集合与账本条目一致）→ 复用 session，进入跳块续传。
  - **不匹配** → 正常新建 session（旧 session 留在账本中等 TTL 或用户手动放弃；同一目标的冲突预检由后端 create 时 409 兜底）。
- v1 不做部分匹配复用（同 session 内部分文件重选），账本条目按整体文件集处理。

**3) 跳块续传（`src/api/endpoints.ts`）**

- `uploadMovieFileChunks` 改造：接收服务端 `chunks` 明细（M2.1），跳过已完成的 `(fileId, chunkIndex)`，只 PUT 缺失块；进度基线从 `bytesReceived` 起算，UI 显示"从 X% 处继续"。
- 复用既有幂等与重试（同 chunkIndex 同 range 200 幂等、3 次线性退避）。
- 会话不可恢复（`recoveryStatus: unrecoverable`、404、state 非 uploading）→ 账本移除 + 提示新建。

**4) Mock 适配**

`mock-library-service.ts` 补一个内存版会话账本行为（create/GET/DELETE + 跳块计数），保证 Mock 模式演示链路不回归。

### 已知限制（v1 接受，不处理）

- 刷新后必须重新选择文件才能续传（浏览器安全模型）；仅 Chromium 系可缓解（M3.5）。
- 多标签页同时续传同一 session：PUT 幂等使重复块无害，但计数/进度可能跳变；不做跨 tab 锁。
- 换浏览器/换设备无法恢复（账本是本机 localStorage），过期由 TTL 自然清理。

### M3.5（可选）Chromium 免重选续传

File System Access API 的 handle 可存 IndexedDB，重开对话框时重新授权即可拿到 `File`，无需重新浏览目录。仅在 Electron/Chrome/Edge 生效；作为渐进增强放在 M3 之后单独评估。

## M4 可选增强（单独立项，本计划仅登记）

1. **并发分块**：`uploadMovieFileChunks` 加 2-3 并发窗口（PUT 幂等 + range 寻址天然支持乱序），提高远程大文件吞吐。
2. **导入→扫描→刮削统一进度视图**：利用任务链元数据（import 的 `scanTaskId`、scrape 的 `parentScanTaskId`）在 Dock 中串成单条流水，替代三种独立 toast，顺带移除 `use-library-watch-toasts.ts` 的 15 秒冗余抑制 hack。

## 测试与验收清单

- 后端单测：分块明细 DTO 填充；空间预检（mock 磁盘空间）；扩展名统一后 scanner/watcher 收录 `.rmvb`/`.wmv`；`.part/.tmp` 仍被排除。
- 前端单测：账本增删与 TTL 清理；指纹匹配三分支（匹配/不匹配/会话终态）；跳块后进度基线；放弃动作调用 DELETE。
- 手工联调：LAN 场景上传 2GB+ 文件中断网/关页 → 重开对话框 → 重选文件 → 从断点继续 → commit → 扫描 → 刮削全链路。
- i18n：`import.*` 新增恢复相关 key，zh-CN / en / ja 三语言同步。

## 涉及文件清单（预估）

- M1：`MovieImportDialog.vue`、`MovieImportDialog.test.ts`
- M2：`contracts.go`、`movie_import_upload_handlers.go`、`movie_import_upload_repository.go`、`server.go`（导入区段 + 空间预检）、`scanner/service.go`、`librarywatch/watcher.go`、对应 `*_test.go`
- M3：新 `src/lib/movie-import-upload-ledger.ts`、`src/api/endpoints.ts`、`src/api/types.ts`、`src/api/guards.ts`、`MovieImportDialog.vue`、`mock-library-service.ts`、3 个 locale JSON、相关 `*.test.ts`

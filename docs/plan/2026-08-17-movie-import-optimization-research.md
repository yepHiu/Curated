# 「添加影片」功能优化调研

- 日期：2026-08-17
- 背景：产品主认为"添加影片"（Movie Import）功能偏鸡肋——不如直接把影片文件复制进库目录，让 fsnotify 监听 + 扫描 + 自动刮削管线自己处理。本文调研该功能的现状、真实价值、维护成本与具体问题，并给出后续优化方向与实施建议。
- 关联历史文档：`2026-05-01-movie-import-implementation-plan.md`、`2026-05-01-movie-import-ui-design.md`、`2026-05-02-movie-import-performance-architecture.md`、`2026-05-02-movie-import-resumable-upload-implementation-plan.md`

## TL;DR

1. 两条链路**最终效果完全一致**：都是 `PersistScanMovie` 入库 → `scrape.movie` 刮削 → NFO/资产。"添加影片"只是多了一层 HTTP 上传 + 受控复制，对**本机桌面场景**确实基本冗余。
2. 发现一个**缺陷级问题**：导入接受 15 种扩展名，但扫描器/监听只认 5 种（mp4/mkv/avi/mov/ts）。导入 `.iso`/`.rmvb`/`.wmv` 等文件会"成功"复制进库目录后**永远不被收录**，纯静默失败。
3. 断点续传的后端机制（约 2,500 行 + 4 张表 + janitor + 启动恢复）**前端从未真正用过**：前端每次新建 session、总是从 chunk 0 重传，关掉页面无法恢复。复杂度与收益严重不匹配。
4. 功能唯一的不可替代价值是 **LAN 远程上传**（手机/其他电脑经浏览器入库，无需 SMB 访问库目录）。
5. 推荐方向：**扶持"复制进目录"成为本机主路径**（补稳定性检查、统一扩展名），同时把导入功能**按场景拆分**——桌面端改为"本机路径导入"（服务端本地复制，不走 HTTP 传输），远程端保留上传但砍掉或补齐断点续传。

---

## 1. 现状：两条链路怎么走

### 1.1 手动复制进目录（fsnotify 链）

```
用户复制文件到任意库根/任意子目录
  → fsnotify Create/Write（internal/librarywatch/watcher.go）
  → isVideoPath 过滤（.part/.tmp/隐藏文件排除，仅 5 种扩展名）
  → 1.5s 防抖（Write 事件不断重置 timer）
  → startLibraryScan(trigger: "fsnotify")
  → PersistScanMovie（SQLite 去重/入库）
  → imported/updated → enqueueScrape → scrape.movie → 元数据 + 资产
  → SSE task.updated → 前端弹"库目录监听"toast + 刷新列表
```

### 1.2 添加影片（import.movies 链）

```
顶栏 MovieImportDialog（拖拽/多选/文件夹选择）
  → 前端格式过滤（16 种）+ relativePath:size 去重 + canImport 存储状态阻断
  → 单文件 <512MB：multipart POST /api/import/movies（XHR 进度）
  → 单文件 ≥512MB：session 上传（POST manifest → 串行 PUT 32MB chunk ×3 重试 → commit）
  → 后端复制到"默认导入库路径"（multipart: tmp+rename；session: .curated-import staging + 硬链接提交）
  → 永不覆盖冲突文件；任务元数据带 stage/进度/错误明细
  → 显式 StartScan([targetRoot])（scanTaskId 写入任务）
  → 与 1.1 完全相同的扫描/入库/刮削管线
```

### 1.3 链路对比

| 维度 | 手动复制 | 添加影片 |
|---|---|---|
| 目标位置 | 任意库根的任意子目录 | 仅"默认导入库路径"（设置里单选，需改设置才能换目标） |
| 数据路径 | 原生文件复制（磁盘速度） | 浏览器 → HTTP(loopback) → tmp/staging → rename/硬链接 |
| 预检 | 无 | 扩展名白名单、冲突 409、路径防逃逸、存储在线检查 |
| 写入原子性 | 无（Explorer 以最终名写入，可能被中途扫描） | tmp+rename / staging+commit，监听层过滤 `.tmp/.part` |
| 进度反馈 | 只有扫描/刮削 toast | 上传进度 + 导入任务 Dock + 扫描 toast + 刮削 toast |
| 扫描触发 | fsnotify 防抖 | 显式 StartScan；fsnotify 随后还会再触发一次（互斥 + 去重兜底） |
| 远程可用性 | 需要文件系统访问（SMB/本地） | 浏览器即可（LAN + PIN） |

## 2. 维护成本盘点

后端（生产 ~3,200 行 + 测试 ~1,523 行）：

| 模块 | 行数 | 说明 |
|---|---|---|
| `server/movie_import_upload_runtime.go` | 932 | 恢复、对账、janitor、安全校验（断点续传专属） |
| `server/movie_import_upload_handlers.go` | 800 | 5 个 session 端点（断点续传专属） |
| `storage/movie_import_upload_repository.go` | 741 | SQLite repository（断点续传专属） |
| `server.go` 导入区段 | ~473 | multipart 端点 + 复制/清洗辅助 |
| `migrations/0029_*.sql` | 70 | 4 张表 |
| library health 集成 | ~122 | staging 残留 finding + 确认式清理 |

前端（生产 ~1,000-1,300 行 + 测试 ~1,000 行）：`MovieImportDialog.vue`（417 行）、`endpoints.ts` 上传逻辑（~150 行）、tracker/Dock import 分支（~115 行）、设置页默认目标选择器（~100 行）、storage-status 告警（119 行）、mock 适配、`import.*` i18n key ×3 语言（每语言 ~40 个）。

**断点续传 session 层占后端导入成本的约 75-80%。**

## 3. 发现的具体问题

### 3.1 缺陷级：扩展名三处不一致（必须修）

- 扫描器 `backend/internal/scanner/service.go:37-43`：mp4/mkv/avi/mov/ts（硬编码，无配置）
- 监听 `backend/internal/librarywatch/watcher.go:50-52`：同上 5 种
- 导入白名单 `backend/internal/server/server.go:2785-2792`：15 种（多出 m4v/wmv/webm/m2ts/flv/mpeg/mpg/ogv/rmvb/iso）
- 前端 `MovieImportDialog.vue:37-53`：16 种（比后端导入还多一种）

后果：
- **导入侧**：导入 `.iso`/`.rmvb` 等文件 → 上传成功、复制成功、任务 completed → 扫描跳过 → 影片永远不出现，且文件已占双份磁盘（原位置 + 库目录）。
- **手动复制侧**：这 10 种扩展名复制进库目录 → 监听和扫描都无视。
- 播放管线本身基于 ffmpeg 探测容器，并不限于 5 种扩展名，瓶颈纯粹在扫描发现层。

### 3.2 断点续传：后端重、前端不用（最大的复杂度错配）

- 前端 `endpoints.ts`：每次 `importMovies` 都 `POST /uploads` 新建 session；总是从 chunk 0 全量上传；不持久化 `uploadId`；不利用响应中的 `bytesReceived`/`complete` 跳过已传块。
- 即：上传中途关页面/断网 → 用户唯一选择是重新开始整个上传。后端 24h TTL + janitor 只负责清理垃圾，不产生用户价值。
- 后端为"可恢复"付出的成本：4 张表、CAS 状态机、启动对账、committing 中断恢复、离线目标盘延迟清理、孤儿扫描、library health 残留处置——约 2,540 行生产代码。

### 3.3 其他可优化点

- **无磁盘空间预检**：导入失败后才分类为 `IMPORT_NOT_ENOUGH_SPACE`；backup 包已有 `Statfs`/`GetDiskFreeSpaceEx` 实现可复用。
- **串行 chunk 上传**：无并发窗口，多 GB 文件单流传输。
- **目标单一**：只能导入到 `defaultImportLibraryPathId`，换目标要去设置页改。
- **进度割裂**：导入 → 扫描 → 刮削是三个独立任务、三种 toast，没有串联视图（任务链其实已有 `scanTaskId`/`parentScanTaskId` 可用来串）。
- **双重扫描触发**：显式 StartScan + fsnotify 各排队一次，靠互斥与去重兜底；前端被迫做 15 秒"无变化"toast 抑制（`use-library-watch-toasts.ts`），是链路设计不干净的直接证据。
- **multipart 路径无重试**（session 路径有 3 次线性退避）。

### 3.4 手动复制路径自身的短板（扶持它为主路径需要补的）

- **无文件稳定性检查**：`findVideoFiles` 只看扩展名。以最终文件名复制大文件时，Create 即触发防抖，扫描可能在复制中途进行——半成品被入库（Windows 文件锁使 organize 的 rename 失败从而侥幸不移动文件，但中断的复制会留下"已入库的坏文件"）。
- **网络盘不可靠**：watcher 包注释明示；`autoScanIntervalSeconds` 默认 0（关闭），网络卷上可能既无事件也无周期扫描。
- **番号识别失败的文件静默跳过**：只在扫描计数里体现，没有"待处理/收件箱"界面帮助改名重试。
- **同番号不同目录**：第二个文件 UPDATE location 指向新路径，旧文件成为磁盘孤儿。

## 4. 功能真实价值评估

| 场景 | 添加影片是否不可替代 | 判断 |
|---|---|---|
| 本机桌面（Electron/loopback），库在本机/直连盘 | 否。Explorer 复制更快、目标更灵活、同样全自动 | 冗余（用户的"鸡肋"感来源） |
| LAN 远程浏览器（手机/笔记本 → 后端） | **是**。无 SMB 时浏览器上传是唯一入库手段 | 真实价值，与项目在 LAN/PIN/connected-clients 上的投入一致 |
| 库在 NAS/网络卷 | fsnotify 不可靠，导入的显式 StartScan 反而更确定 | 有条件价值（但上传到网络卷的双写/速度是新问题） |
| 引导式/防错需求（冲突预检、原子落盘、进度） | 有价值但可以用更轻的手段获得（见方案 C） | 部分可替代 |

结论：功能不是全无价值，而是**价值集中在远程场景，成本却主要花在为本机场景服务的断点续传上**。

## 5. 优化方向选项

### 方案 A：彻底退役 HTTP 导入，全面转向目录导入

- 删除 `MovieImportDialog`、`/api/import/*`、0029 四张表（保留 migration 历史仅停用）、janitor、health 集成、相关 i18n/测试。
- 全部投入补强 fsnotify/扫描链：统一扩展名、稳定性检查、网络卷周期扫描建议、识别失败 triage。
- 收益：净删 ~4,500+ 行生产代码，产品概念收敛为"库目录即真相"。
- 代价：失去 LAN 远程上传；与已有 LAN/PIN/connected-clients 方向相悖。

### 方案 B：保留现状，只修缺陷（保守）

- 修 3.1 扩展名对齐 + 3.3 小项；不动架构。
- 不解决"鸡肋"感与复杂度错配。

### 方案 C：按场景拆分（推荐起点）

**本机（Electron）→ "本机路径导入"：**
- preload 增加文件/文件夹选择能力（现为仅 `pickDirectory`）。
- 新端点 `POST /api/import/local`：后端从用户选中的**本机绝对路径**做受控复制（tmp+rename、冲突预检、空间预检、进度、scan 链接）——复用现有 multipart 端点的全部辅助函数，去掉 HTTP 传输。
- 速度等价 Explorer 复制，但保留原子性/防冲突/统一进度；浏览器上传在本机场景退役。

**远程（LAN 浏览器）→ 保留上传，二选一：**
- C1：砍掉断点续传 session 层（multipart-only + `totalBytes` 上限 + 明确的"远程大文件请用桌面端"引导）——净删 ~2,540 行 + 4 张表。
- C2：把断点续传做完整（前端持久化 uploadId + 指纹到 localStorage/IndexedDB，重开对话框时 GET session 状态、跳过已传块、真恢复）——让已有后端投资兑现价值。
- 取决于"远程上传大文件"是否真实使用场景；若几乎不用，选 C1。

### 方案 D：收件箱（Inbox）模型（远期可选）

- 所有新文件先落"待整理"区，triage UI 展示未识别番号/待刮削/冲突项，用户确认后整理入库。
- 解决 3.4 的静默跳过与识别失败问题，但产品改动大，建议独立立项。

## 6. 推荐路线（分阶段）

**P0（无论选哪个方向都要做，缺陷修复）**
1. 统一扩展名：抽一个共享白名单常量（contracts 或独立包），scanner/watch/import/前端四处引用。倾向**向 15 种对齐扫描器**（播放管线基于 ffmpeg，容器不是瓶颈；番号从文件名提取与容器无关），并对 `.iso`（无番号场景多）单独确认产品意图。
2. 修导入"成功但永不收录"的体验：至少在导入任务中报告"扩展名暂不被扫描收录"而不是 success。

**P1（扶持目录导入为主路径）**
3. 扫描稳定性检查：候选文件 size+mtime 双采样（间隔数秒）不变才入库；或扫描结束时对首轮新文件复查。杜绝半成品入库与中断复制留下的脏记录。
4. 网络卷场景：storage-status 已能识别卷类型，可在设置里对网络卷根提示开启 `autoScanIntervalSeconds`（或自动给默认值）。
5. 识别失败可见性：fsnotify/扫描 toast 或通知中心给出 `number_not_recognized` 计数与文件名清单入口。

**P2（导入功能重构，先做产品决策）**
6. 决策点 1：LAN 远程上传是否真实使用？否 → 方案 A 或 C1；是 → C2。
7. 决策点 2：是否愿意为桌面本机导入加 preload 能力？是 → 方案 C 桌面部分。
8. 若选 C1/A：退役顺序 = 前端入口下线 → API 标记 deprecated 一个版本 → 删后端与表 → 清理 health/i18n/测试。

**P3（体验统一，可选）**
9. 用任务链（import.scanTaskId / scrape.parentScanTaskId）做"导入 → 扫描 → 刮削"单一进度视图，替代三种独立 toast；顺带消灭 15 秒冗余抑制 hack。

## 7. 需要产品拍板的决策点

1. 扫描扩展名向 15 种对齐，还是导入收敛到 5 种？（建议前者；`.iso` 单独定）
2. LAN 远程上传是否是保留"添加影片"的充分理由？
3. 桌面端是否接受 preload 扩展（文件选择器）以启用本机路径导入？
4. 断点续传：砍（C1）还是补完（C2）？

## 8. 决策记录（2026-08-17）

- **LAN 远程上传是真实场景，但优先级不高**：功能保留，不退役（排除方案 A/C1）。
- **入口弱化**：顶栏"添加影片"按钮从主题色（`variant="default"`）改为弱化样式，降低存在感（具体方案见实施计划 M1）。
- **方向定为 C2（保留并完善）**：把前端续传补完整，让后端已有的断点续传机制兑现价值；同时做后端优化（扩展名统一、空间预检、session 状态暴露分块明细等）。
- 实施计划：`2026-08-17-movie-import-improvement-implementation-plan.md`。

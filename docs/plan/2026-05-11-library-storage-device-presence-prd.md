# Curated 影片存储硬盘在线检测 PRD

日期：2026-05-11
状态：Accepted for Windows-first implementation
范围：Curated 本地后端 + Web 前端设置页 + 通知中心

## 0. 已确认决策

- 第一版优先支持 Windows 系统，目标是把外置硬盘盒 / USB 硬盘 / Windows 盘符场景做扎实。
- macOS 和 Linux 保留为未来适配项，第一版只做路径可用性 fallback，不承诺完整卷身份绑定体验。
- 第一版不尝试控制硬盘供电，只检测已登记库路径所在存储是否在线、是否仍是原设备、路径是否可访问。

## 1. 背景

Curated 允许用户登记多个影片存储目录。当前系统默认这些目录在应用启动后可访问，并在目录监听、手动扫描、导入影片、默认导入目录等流程中直接使用路径。

实际使用中，影片库可能存放在外置硬盘盒、USB 硬盘、移动硬盘或其他需要独立供电的设备上。这类设备的上下电行为不一定跟随电脑启动。用户可能遇到以下情况：

- 电脑已启动，Curated 已启动，但外置硬盘盒未上电。
- 盘符暂时不存在，或盘符存在但不是原来的硬盘。
- 存储目录路径不可访问，导致自动监听、扫描、导入或播放相关操作失败。
- 当前系统缺少明确提醒，用户需要自己发现“硬盘没上电 / 没连接”。

这个需求的核心不是“让 Curated 给硬盘供电”，而是让 Curated 识别已登记库路径绑定的存储设备当前是否在线，并在异常时明确提醒用户。

## 2. 问题陈述

当 Curated 启动时，如果已登记的影片存储目录所在硬盘未上电或未连接，用户应该在第一时间知道：

- 哪个存储目录不可用。
- 可能原因是硬盘未上电、未连接、盘符不存在、目录不存在、权限不足，还是盘符被其他设备占用。
- 应该怎么处理，例如给硬盘盒上电、重新连接硬盘、确认盘符、进入设置页重新检测。

当前系统只保存库路径本身，没有保存路径所在卷 / 设备的身份，也没有启动时的库路径健康检测机制。

## 3. 目标

1. 启动时检测所有已登记影片存储目录的可用性。
2. 尽可能识别“路径所在硬盘 / 卷是否是上次登记的同一个设备”。
3. 如果硬盘离线、目录不可达或卷不匹配，通过新的 toast 和通知中心提醒用户。
4. 在设置「影片存储」中展示每个存储目录的在线状态，并提供手动重新检测入口。
5. 避免因为硬盘临时离线而删除影片记录、错误重建资料库、误触发大规模扫描。
6. 给扫描、目录监听、导入影片等依赖库路径的流程提供统一的“路径健康状态”基础。

## 4. 非目标

第一版不做以下内容：

- 不尝试给硬盘盒上电，不发送 USB / SATA / 电源控制命令。
- 不做 SMART 健康检测、坏道检测、硬盘寿命评估。
- 不做磁盘格式化、修复、挂载、盘符重分配。
- 不删除任何影片记录或磁盘文件。
- 不把设备序列号上传到外部服务；所有信息只保存在本机。
- 不要求用户必须绑定硬盘身份才能继续使用；绑定失败时退化为路径存在性检测。

## 5. 用户故事

### US1：启动时发现外置硬盘未上电

作为用户，我启动电脑后 Curated 自动启动，但外置硬盘盒还没开机。Curated 应提示“存储目录所在硬盘不在线”，并告诉我是哪一个库路径。

验收标准：

- 应用启动后自动检测。
- 如果路径根盘符不存在或卷不可访问，通知中心出现一条存储异常通知。
- toast 文案清楚说明“可能未上电或未连接”。
- 点击通知可进入设置「影片存储」。

### US2：硬盘后来上电，用户重新检测

作为用户，我打开硬盘盒电源后，希望在设置页点击“重新检测”并看到目录恢复在线。

验收标准：

- 设置页每个库路径都有状态。
- 点击重新检测后刷新状态。
- 从离线恢复在线时可显示成功 toast，并更新通知中心记录或新增恢复通知。

### US3：盘符存在，但不是原来的硬盘

作为用户，我希望 Curated 不要仅因为 `E:\Movies` 存在就认为库正常，因为 `E:` 可能被另一个 U 盘占用了。

验收标准：

- 添加库路径或首次健康检测成功时，记录该路径所在卷的身份信息。
- 后续检测时，如果相同盘符对应的卷身份变化，状态显示为“卷不匹配”。
- 不自动扫描这个路径，避免把错误硬盘上的内容纳入资料库。

### US4：硬盘在线但目录被改名或删除

作为用户，如果硬盘在线但影片目录不存在，我希望提示区分“硬盘不在线”和“目录缺失”。

验收标准：

- 卷在线但路径 `stat` 返回不存在时，状态为 `path_missing`。
- 文案提示“硬盘在线，但目录不存在或已移动”。

### US5：路径无权限

作为用户，如果路径因为权限问题不可访问，我希望看到权限受限提示。

验收标准：

- 权限错误归类为 `permission_denied`。
- 提醒用户检查目录权限或以合适权限运行 Curated。

## 6. 状态模型

新增库路径存储状态：

| 状态 | 含义 | 示例提醒 |
| --- | --- | --- |
| `online` | 卷在线，路径可访问，且绑定身份匹配 | 存储目录在线 |
| `offline` | 卷 / 根路径不可用 | 硬盘可能未上电或未连接 |
| `volume_mismatch` | 当前盘符在线，但不是登记时的同一个卷 | 盘符可能被其他设备占用 |
| `path_missing` | 卷在线，但库目录不存在 | 目录可能被移动或删除 |
| `permission_denied` | 路径存在但不可读取 | 权限不足 |
| `unknown` | 检测失败或平台不支持完整检测 | 无法确认存储状态 |

## 7. 设备识别策略

### 7.1 Windows 第一优先级

Curated 当前重点运行环境是 Windows，本需求第一版应优先做好 Windows。

建议识别字段：

- 路径根：如 `E:\`。
- 卷标：Volume Label。
- 卷序列号：Volume Serial Number。
- 文件系统：NTFS / exFAT 等。
- Drive Type：fixed / removable / remote / cdrom / ramdisk / unknown。
- 可选扩展：设备路径、总容量、可用容量。

建议后端使用 Windows API 获取卷信息：

- `GetDriveTypeW` 判断驱动器类型。
- `GetVolumeInformationW` 获取卷标、卷序列号、文件系统。
- `GetLogicalDrives` 或等价能力判断盘符是否存在。

第一版不强依赖 WMI。WMI 信息更丰富，但权限、性能和系统差异更复杂，可作为后续增强。

### 7.2 非 Windows 平台

macOS / Linux 第一版做 best-effort：

- 检查路径是否存在、是否目录、是否可读。
- 尝试识别挂载点。
- 如无法稳定获得卷 UUID，则不做强绑定，只做路径可用性检测。

### 7.3 绑定策略

添加库路径或首次检测成功时，记录“期望卷身份”：

- `libraryPathId`
- `rootPath`
- `volumeSerial` / `volumeUuid`
- `volumeLabel`
- `fileSystem`
- `driveType`
- `boundAt`
- `lastSeenAt`

后续检测时：

1. 先解析库路径所在根。
2. 检查根是否在线。
3. 读取当前卷身份。
4. 如果已绑定身份存在且当前身份不同，返回 `volume_mismatch`。
5. 如果卷身份匹配，再检查库目录路径本身。

如果某些外置硬盘盒无法提供稳定卷序列号，则降级为路径根 + 卷标 + 文件系统 + 路径存在性组合判断，并在状态详情中标记 `identityConfidence: "low"`。

## 8. 后端设计

### 8.1 新增领域模块

建议新增：

- `backend/internal/storagehealth`
- 或 `backend/internal/volumehealth`

职责：

- 解析库路径所在卷 / 挂载点。
- 调用平台相关探测器。
- 输出统一 DTO。
- 不直接操作扫描、刮削、删除等业务。

平台文件：

- `probe_windows.go`
- `probe_posix.go`
- `probe_stub.go`

### 8.2 数据持久化

建议新增表，而不是直接扩展 `library_paths`：

`library_path_storage_bindings`

字段建议：

- `library_path_id TEXT PRIMARY KEY`
- `root_path TEXT NOT NULL`
- `volume_id TEXT`
- `volume_label TEXT`
- `file_system TEXT`
- `drive_type TEXT`
- `identity_confidence TEXT NOT NULL DEFAULT 'unknown'`
- `bound_at TEXT`
- `last_seen_at TEXT`
- `last_checked_at TEXT`
- `last_status TEXT`
- `last_error TEXT`
- `updated_at TEXT NOT NULL`

原因：

- 库路径本身仍保持简单。
- 健康检测状态可以独立演进。
- 后续可加入容量、设备型号、网络挂载等信息，不污染核心路径表。

### 8.3 API

建议新增接口：

#### `GET /api/library/paths/storage-status`

返回当前缓存的库路径存储状态；如果没有缓存，可触发轻量检测或返回 `unknown`。

响应：

```json
{
  "items": [
    {
      "libraryPathId": "library-1",
      "path": "E:\\Movies",
      "title": "External HDD",
      "status": "offline",
      "message": "The storage device for this library path appears offline.",
      "checkedAt": "2026-05-11T10:00:00Z",
      "rootPath": "E:\\",
      "driveType": "removable",
      "volumeLabel": "CURATED_VAULT",
      "fileSystem": "NTFS",
      "identityConfidence": "high",
      "expectedVolumeId": "ABCD-1234",
      "currentVolumeId": "",
      "canRescan": false,
      "canImport": false
    }
  ]
}
```

#### `POST /api/library/paths/storage-status/check`

触发 fresh check，可选 body：

```json
{
  "libraryPathIds": ["library-1"]
}
```

如果省略 `libraryPathIds`，检测全部库路径。接口更新缓存和绑定表。

#### `POST /api/library/paths/{id}/storage-binding/rebind`

当用户确认当前硬盘就是新的正确硬盘时，重新绑定该路径的卷身份。

使用场景：

- 用户换了硬盘。
- 用户重新格式化硬盘导致卷序列号变化。
- 硬盘盒桥接导致身份变化，用户明确接受当前设备。

### 8.4 启动检测

后端初始化完成数据库和 App 后，执行一次非阻塞检测：

- 启动后短延迟，例如 1-3 秒，避免拖慢 HTTP 服务启动。
- 每个路径检测设置超时，例如 800ms-1500ms，避免网络盘或休眠盘阻塞启动。
- 检测结果写入缓存 / 表。
- 不因检测失败阻止后端启动。

前端启动后主动调用 `POST /api/library/paths/storage-status/check` 或 `GET /api/library/paths/storage-status`，再根据结果发通知。

## 9. 前端设计

### 9.1 服务层

扩展：

- `src/api/types.ts`
- `src/api/endpoints.ts`
- `src/services/contracts/library-service.ts`
- Web adapter
- Mock adapter

新增类型：

```ts
export type LibraryPathStorageStatus =
  | "online"
  | "offline"
  | "volume_mismatch"
  | "path_missing"
  | "permission_denied"
  | "unknown"

export interface LibraryPathStorageStatusDTO {
  libraryPathId: string
  path: string
  title: string
  status: LibraryPathStorageStatus
  message: string
  checkedAt: string
  rootPath?: string
  driveType?: string
  volumeLabel?: string
  fileSystem?: string
  identityConfidence?: "high" | "medium" | "low" | "unknown"
  expectedVolumeId?: string
  currentVolumeId?: string
  canRescan: boolean
  canImport: boolean
}
```

### 9.2 设置页 UI

设置「影片存储」：

- 每个存储目录行增加状态 badge。
- 离线 / 卷不匹配 / 权限受限状态使用 warning/destructive 视觉。
- 在线状态显示 muted success。
- 增加“重新检测”按钮。
- 对 `volume_mismatch` 增加“重新绑定当前硬盘”动作，但必须有确认弹窗。

建议文案：

- `online`：存储在线
- `offline`：硬盘不在线，可能未上电或未连接
- `volume_mismatch`：当前盘符不是上次登记的硬盘
- `path_missing`：硬盘在线，但目录不存在
- `permission_denied`：目录权限不足
- `unknown`：无法确认存储状态

### 9.3 启动通知

前端启动后检测存储状态：

- 如果存在 `offline` / `volume_mismatch` / `path_missing` / `permission_denied`，发一条聚合 toast。
- 同步写入通知中心。
- 通知 source 指向 `/settings?section=library`。
- 通知中心建议新增类型 `storage`；如果短期不扩展类型，可先用 `system`。

聚合策略：

- 1 个异常：`“存储目录「{title}」所在硬盘不在线。”`
- 多个异常：`“{n} 个影片存储目录当前不可用。”`

去重策略：

- 同一启动会话内只提醒一次。
- 通知中心按 `libraryPathId + status` 去重或更新。
- 从异常恢复在线时，可新增一条 `storageRestored` 通知，或仅更新设置页状态。第一版建议只 toast 成功，不写通知中心，减少噪音。

### 9.4 通知中心扩展

建议扩展：

```ts
export type NotificationType = "scan" | "scrape" | "update" | "error" | "system" | "storage"

export interface NotificationSource {
  taskId?: string
  movieId?: string
  libraryPathId?: string
  route?: string
}
```

新增标题：

- `notificationCenter.titles.storageOffline`
- `notificationCenter.titles.storageMismatch`
- `notificationCenter.titles.storageRestored`

## 10. 与扫描 / 监听 / 导入的关系

第一版检测机制不应只停留在提示，还应避免明显错误操作。

建议规则：

| 场景 | 行为 |
| --- | --- |
| 手动重新扫描离线路径 | 阻止启动任务，提示先连接硬盘 |
| 全库扫描中某些路径离线 | 跳过离线路径，任务结果可为 partial_failed，并写明跳过路径 |
| fsnotify 监听启动 | 对离线路径不启动 watcher；恢复在线后用户可重新检测并重启 watcher |
| 影片导入目标路径离线 | 阻止导入，提示默认导入目录所在硬盘不在线 |
| 播放已有影片但文件路径离线 | 详情页 / 播放页提示源文件所在存储不可用 |

MVP 可先实现“提醒 + 设置页状态 + 手动扫描/导入阻止”，后续再补全 watcher 自动恢复。

## 11. 成功指标

定性指标：

- 用户启动 Curated 后，不需要手动猜测硬盘是否在线。
- 用户能明确知道哪个库路径不可用。
- 用户不会因为外置硬盘未上电而误以为资料库损坏或影片丢失。

可量化指标：

- 启动检测不会让 HTTP 服务启动时间增加超过 1 秒。
- 单个本地盘检测目标耗时小于 200ms；异常路径检测有超时保护。
- 同一会话内同一异常不重复弹 toast。
- 离线库路径不会触发误扫描或误删除。

## 12. 里程碑计划

### Phase 1：后端存储健康探测

- 新增 `storagehealth` 模块。
- Windows 实现卷信息读取。
- 非 Windows 实现路径可用性 fallback。
- 新增绑定表 migration。
- 新增状态检测 service。
- 新增 API：状态查询、手动检测、重新绑定。
- 单元测试覆盖状态分类和绑定匹配。

### Phase 2：前端状态展示

- 增加 API 类型和 service contract。
- Web adapter 接入真实接口。
- Mock adapter 提供在线 / 离线 / 卷不匹配示例。
- 设置页库路径列表展示状态 badge。
- 增加重新检测按钮和重新绑定确认。

### Phase 3：启动提醒与通知中心

- AppShell 或启动 composable 调用存储状态检测。
- 接入 `pushAppToast` 和通知中心。
- 新增 `storage` 通知类型或先复用 `system`。
- 做会话去重。

### Phase 4：流程保护

- 手动扫描前检查路径状态。
- 默认导入路径离线时阻止导入。
- 全库扫描跳过离线路径并给出 partial result。
- fsnotify watcher 跳过离线路径，恢复在线后的自动 watcher 恢复作为后续增强。

### Phase 5：文档与发布说明

- 更新 `project-facts.mdc`。
- 更新 `README.md` / `README.zh-CN.md` / `README.ja-JP.md` 的功能摘要。
- 更新 `docs/reference/architecture-and-implementation.html`。
- 增加 release notes。

## 13. 测试计划

### 后端

- Windows volume probe 单元测试：用接口抽象 mock WinAPI 返回。
- 路径分类测试：
  - 根不存在 -> `offline`
  - 根存在路径不存在 -> `path_missing`
  - 权限错误 -> `permission_denied`
  - 卷 ID 不同 -> `volume_mismatch`
  - 卷 ID 相同路径可读 -> `online`
- API 测试：
  - GET 返回缓存状态。
  - POST check 更新状态。
  - rebind 更新绑定。
- scanner/import guard 测试：
  - 离线默认导入路径阻止导入。
  - 全库扫描跳过离线路径。

### 前端

- service adapter 测试：DTO 映射与状态缓存。
- 设置页组件测试：不同状态 badge 和动作显示。
- 通知 composable 测试：
  - 启动异常只提醒一次。
  - 多路径异常聚合。
  - 通知 source 指向设置页。
- Mock 模式测试：可模拟离线路径。

## 14. 风险与取舍

1. USB 硬盘盒桥接芯片可能隐藏真实硬盘身份。
   - 处理：第一版绑定卷序列号和卷标，不强依赖硬件序列号。

2. 盘符变化会影响路径可用性。
   - 处理：第一版提醒用户路径不可用；后续可做“发现同卷新盘符并建议修复路径”。

3. 检测路径可能唤醒休眠硬盘。
   - 处理：优先读取卷信息，减少递归访问；只 `stat` 库根，不扫描目录内容。

4. 网络路径或 NAS 检测可能慢。
   - 处理：每路径超时保护；remote drive 状态文案区分本地外置盘。

5. 通知过多影响体验。
   - 处理：启动聚合提醒，会话级去重，通知中心按路径去重。

## 15. 默认产品决策

如果没有进一步修改，建议按以下默认决策实现：

- 第一优先级支持 Windows 本地盘符，非 Windows 做路径可用性 fallback。
- 启动时自动检测并提醒，但不阻止应用启动。
- 离线或卷不匹配的路径不参与扫描、导入和 watcher。
- 检测结果显示在设置「影片存储」的每个路径行内。
- 通知中心新增 `storage` 类型。
- 不自动修改库路径，不自动删除影片记录。
- 用户手动确认后才允许重新绑定当前硬盘身份。

## 16. 开放问题

1. 如果硬盘恢复在线，是否需要自动重新启动 fsnotify watcher，还是只在用户点击“重新检测”后恢复？
2. 如果发现同一个卷换了盘符，是否要提供“一键更新库路径盘符”的能力？
3. 通知恢复在线时是否写入通知中心，还是只显示短 toast？
4. 默认是否对固定内置盘也做同样提醒，还是仅对 removable / external drive 强提醒？
5. 是否需要在侧边栏或全局顶部显示“存储异常”入口，还是只放在通知中心和设置页？

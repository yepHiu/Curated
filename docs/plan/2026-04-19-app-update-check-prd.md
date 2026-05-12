# App Update Check PRD

Status: Draft, expanded with auto download / install feasibility
Date: 2026-04-19
Last updated: 2026-05-13

## 2026-05-13 Implementation Update

- `autoDownloadUpdates` 已落地为 Settings -> General 中的持久化开关，默认 `false`，经 `GET/PATCH /api/settings` 与 `config/library-config.cfg` 读写。
- 启动后的后台更新检查若发现较新的 installer，且该开关已开启，会自动下载并完成 SHA256 校验。
- 该能力只覆盖“后台下载并准备安装包”；真正安装仍在 Settings -> About 中由用户显式确认，不等同于默认静默安装。

## Background

Curated 目前在 Settings -> About 中已经能展示：

- 应用构建版本 `version`
- 安装包版本 `installerVersion`

但当前 About 页只能告诉用户“我现在装的是哪个版本”，还不能回答：

- 远端仓库当前最新发布版本是什么
- 我当前安装的版本是否已经落后
- 如果有新版本，应该去哪里下载和更新

这会带来两个问题：

1. 用户需要手动打开 GitHub Release 页面比对版本，路径长且容易忘。
2. 生产包已经有稳定的安装包版本号，但没有围绕它构建“检查更新”闭环，About 页信息价值不完整。

本 PRD 定义一个面向正式安装包的“更新检查”能力：

- 支持自动检查更新
- 支持手动点击“检查更新”
- 能在 About 页明确展示“当前版本 vs 最新版本”结果
- 发现新版本后，引导用户前往官方 Release 页面下载更新

当前项目的 GitHub Release 页面仓库地址为：

- 仓库 Releases 首页：`https://github.com/yepHiu/Curated/releases`
- 当前示例 Release：`https://github.com/yepHiu/Curated/releases/tag/v1.2.7`

## Product Goal

让正式安装包用户可以在 Curated 内部低成本确认是否有可用更新，并在发现更新时获得明确、可信、低打扰的升级引导。

## 2026-05-03 Scope Update

本需求已从“只打开 GitHub Release 页面”扩展为“优先直接下载最新 Windows `.exe` 安装包”：

- 后端仍以 GitHub latest Release 作为唯一更新来源。
- 当 latest Release 的 assets 中包含 `.exe` 安装包时，API 返回 `installerDownloadUrl`。
- Settings -> About 的主操作按钮优先链接该 `.exe` 安装包，让浏览器直接开始下载。
- 如果 latest Release 没有可识别的 `.exe` asset，界面继续降级为打开官方 Release 页面。
- 该阶段“不默认后台下载 / 不默认静默安装”的边界仍成立；自 2026-05-13 起，Curated 仅在用户显式开启 `autoDownloadUpdates` 后执行后台下载与 SHA256 校验，仍不会自动替换程序或默认静默安装。

## 2026-05-12 Auto Download / Install Feasibility Update

新的明确需求是把更新能力继续扩展到：

- 自动发现新版本
- 自动下载安装包
- 自动安装
- 低打扰 / 后台 / 静默更新
- 也允许用户在界面中显式触发更新安装流程

当前本地产物签名状态实测：

- `release/installer/Curated-Setup-1.4.4.exe`: `Get-AuthenticodeSignature` 返回 `NotSigned`
- `release/Curated/curated.exe`: `Get-AuthenticodeSignature` 返回 `NotSigned`

这意味着 Curated 目前没有生产级 Authenticode 信任链。自动更新仍可先实现下载、SHA256 校验、用户确认安装；但如果要把“静默自动安装”作为默认能力，签名发布和安装前签名校验应作为前置条件。

结论：**可行，但不能把“静默更新”理解为绕过用户知情、绕过系统权限或绕过安全校验的隐蔽替换。** 对 Curated 当前 Windows + Go 后端 + Inno Setup 安装包结构来说，推荐定义为：

- 后台自动检查：可直接沿用现有 `GET /api/app-update/status` 与 `POST /api/app-update/check`。
- 后台自动下载：可行，应由后端下载到 `LOCALAPPDATA\Curated\updates\` 或当前运行目录旁的安全缓存目录。
- 包完整性校验：必须做。当前 `pnpm release:publish` 已生成 `release/manifest/release.json`，其中包含 installer / portable artifact 的 `sha256`；后续应把 manifest 作为 GitHub Release asset 发布，并让后端用它校验下载结果。GitHub Release asset 响应本身也可携带 `digest` 字段，但不能假设历史 release 都有，应优先使用 Curated 自己生成的 manifest。
- 数字签名校验：强烈建议做。下载后至少校验 Authenticode 签名与发布者；发布流程需要给 `Curated-Setup-<version>.exe` 做代码签名并时间戳。
- 显式安装：可行，优先落地。用户点击“下载并安装”后，后端启动已下载的 installer。
- 静默安装：技术上可行，Inno Setup 支持 `/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /SP- /CLOSEAPPLICATIONS` 等参数，但当前安装器默认安装到 `{autopf}\Curated`，且 Inno Setup 的 `PrivilegesRequired` 默认是 `admin`，因此从未提权的 Curated 进程启动安装器时，Windows UAC 仍会出现。这不是应用可以也不应该规避的系统安全边界。
- 真正无感自动安装：只有在以下架构之一成立时才现实：
  - 初次安装时选择 per-user 安装目录并将 Inno `PrivilegesRequired=lowest`，之后更新不需要写入 Program Files。
  - 初次安装时安装一个受信任、已签名、最小权限范围的 updater service / scheduled task，由它负责后续提权安装。
  - 未来迁移到带成熟更新器的桌面壳或安装框架，但这会改变当前“Go 后端托管 Web UI”的发布边界。

因此，本阶段建议把目标拆成三个级别：

1. **MVP：显式触发下载并安装**
   - 用户在 Settings -> About 点击“下载并安装”。
   - 后端发现版本、下载 installer、校验 hash / 签名、启动安装器。
   - 安装器可以用普通 UI 或 `/SILENT`，默认不要使用完全无界面的 `/VERYSILENT`。
   - 如果需要管理员权限，允许 Windows UAC 正常弹出。

2. **增强：后台自动下载，用户确认安装**
   - 启动后低频检查更新。
   - 发现新版本后后台下载并校验。
   - 下载完成后只给非阻塞提示：`新版本已准备好`。
   - 用户点击“立即安装”或“退出后安装”再执行安装流程。
   - 这是最适合当前 Curated 的默认路线，低打扰但仍透明。

3. **高级：用户明确开启的静默安装**
   - Settings 中新增开关，例如 `自动下载更新`、`下载完成后静默安装`。
   - 仅在安装包已签名、hash 校验通过、版本高于本地、release channel 匹配时允许。
   - 如果当前安装模式需要 admin，仍然会出现 UAC；不能承诺完全无感。
   - 对 Program Files 安装模式，后台只能做到“静默传参 + 系统权限提示正常出现”。

### Feasibility Matrix

| 能力 | 当前基础 | 可行性 | 推荐阶段 | 关键风险 |
| --- | --- | --- | --- | --- |
| 自动发现新版本 | 已实现 GitHub latest Release 检查、SQLite 缓存、侧边栏提示 | 高 | 已完成 / 保持 | GitHub 网络、代理、限流、tag 语义版本 |
| 自动获取 installer URL | 已实现 `installerDownloadUrl` 解析 | 高 | 已完成 / 保持 | asset 命名不稳定，需要 manifest 兜底 |
| 后台下载安装包 | 已实现 opt-in `autoDownloadUpdates` 后台准备安装包 | 高 | 已完成 / 保持 | 断点续传、磁盘空间、取消、缓存清理 |
| SHA256 校验 | 已实现下载后 SHA256 完整性校验 | 高 | 已完成 / 保持 | manifest 发布流程仍需继续固定 |
| Authenticode 签名校验 | 尚未实现签名发布和校验 | 中高 | Phase 1 / 2 前置 | 证书成本、CI secret 管理、时间戳 |
| 显式触发安装 | 已实现 About 页用户确认后的 installer 启动 | 高 | 已完成 / 保持 | 关闭当前进程、单实例锁、安装日志、失败恢复 |
| 静默安装 | Inno Setup 支持静默参数 | 中 | Phase 2 | UAC 仍可能出现；错误提示可能被压低 |
| 完全无感自动安装 | 当前 Program Files + admin installer 架构不满足 | 低到中 | Phase 3+ | 需要 per-user install 或 updater service |
| 增量更新 | 当前发布物是完整 installer / portable zip | 低 | 暂不做 | 包格式、回滚、差分生成与校验复杂度 |

### Recommended Target Behavior

默认行为应是“自动发现 + 自动下载准备 + 用户确认安装”，而不是默认完全静默替换：

- 启动后延迟检查更新，遵守现有 24 小时缓存策略。
- 发现更新后，若用户开启 `autoDownloadUpdates`，后端排队下载任务。
- 下载完成且校验通过后，通知中心显示：`新版本已下载，准备安装`。
- About 页主 CTA 从 `下载最新安装包` 变为：
  - 未下载：`下载并安装`
  - 下载中：`正在下载...`
  - 已就绪：`立即安装`
  - 安装中：`正在启动安装程序...`
- 用户点击安装后：
  - 后端记录安装尝试状态。
  - 后端启动 installer。
  - 对显式安装默认使用普通 UI 或 `/SILENT /NORESTART /SP- /CLOSEAPPLICATIONS`。
  - 对用户开启的静默安装才使用 `/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /SP- /CLOSEAPPLICATIONS`。
  - 当前进程需要主动释放单实例锁并退出，避免安装器无法替换 `curated.exe`。

### Proposed Backend Model

建议新增 `internal/appupdate` 的第二层状态机，不复用现有 `status` 字段承载所有下载和安装状态：

- Release check state：继续使用现有 `unsupported | up-to-date | update-available | error`。
- Installer artifact state：新增 `not-started | downloading | downloaded | verified | failed | installing | install-launched`。

建议新增 DTO 字段：

- `artifactStatus`
- `downloadedVersion`
- `downloadedFileName`
- `downloadedBytes`
- `totalBytes`
- `downloadProgress`
- `sha256`
- `signatureStatus`
- `installReady`
- `installMode`
- `lastInstallAttemptAt`
- `lastInstallError`

建议新增接口：

- `POST /api/app-update/download`
  - 启动或恢复 installer 下载。
- `POST /api/app-update/install`
  - 启动安装器。
  - body 示例：`{ "mode": "interactive" | "silent", "restartApp": true }`。
- `DELETE /api/app-update/downloaded-installer`
  - 清理已下载但未安装的安装包。

首版实现暂不提供取消下载接口；若下载改为长时间后台异步任务，再补 `POST /api/app-update/cancel-download`。

下载任务建议纳入现有任务系统，以便通知中心和进度 dock 可以复用 task 事件：

- task type：`app-update.download`
- task metadata：`version`、`fileName`、`downloadedBytes`、`totalBytes`、`sha256`、`stage`

### Installer Launch Plan

Windows installer 启动建议由后端新增 `internal/appupdate/install_windows.go`：

- 使用 `exec.CommandContext` 启动 installer。
- 安装参数分层：
  - 显式普通安装：无静默参数，或仅带 `/NORESTART /SP-`。
  - 显式低打扰安装：`/SILENT /NORESTART /SP- /CLOSEAPPLICATIONS`。
  - 用户授权静默安装：`/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /SP- /CLOSEAPPLICATIONS`。
- 如果需要保留安装日志，带 `/LOG="<logPath>"`，但不要把 Inno log 当作稳定机器可解析格式。
- 启动 installer 后，Curated 应准备退出：
  - 停止 HTTP server 接新请求。
  - 关闭 tray。
  - 释放 single-instance mutex。
  - 允许 installer 关闭 `curated.exe`。
- 安装器模板当前已有 `CloseApplications=yes`、`CloseApplicationsFilter=curated.exe`、`RestartApplications=no`。未来如果希望安装完成后自动重启 Curated，需要显式设计 restart handoff，而不是依赖 Inno 的默认行为。

### Security And Trust Requirements

自动下载和安装属于高风险路径，必须把以下要求列为实现前置：

- 只接受 HTTPS 下载 URL。
- 下载 URL 必须属于配置允许的 release host，默认仅允许 `github.com/yepHiu/Curated/releases/download/...`。
- 下载文件必须写入受控缓存目录，文件名按远端版本和 asset 名生成，避免路径穿越。
- 下载完成后必须校验 SHA256。
- 需要尽快引入 Windows code signing；安装器和主 exe 都应签名。
- 若已签名，安装前校验 Authenticode 证书链与发布者。
- 只允许升级到版本号高于当前 `installerVersion` 的 release。
- 不允许 draft / prerelease 自动安装到 stable channel。
- 静默安装必须是用户明确开启的设置，默认关闭。
- 所有自动下载、校验、安装尝试都写入日志，About 页能展示最近失败原因。

### Product Setting Proposal

建议新增设置项，默认值保守：

- `checkForUpdatesAutomatically`: `true`
- `autoDownloadUpdates`: `false` 或后续成熟后改为 `true`
- `autoInstallUpdates`: `false`
- `silentInstallUpdates`: `false`
- `updateChannel`: `stable`

默认推荐：

- 自动检查：开。
- 自动下载：第一版默认关，功能稳定后可改为开。
- 自动安装：默认关。
- 静默安装：默认关，且必须要求包签名校验通过。

### Release Pipeline Impact

为了支撑自动安装，发布流程需要补齐：

- `pnpm release:publish` 生成的 `release/manifest/release.json` 必须作为 GitHub Release asset 上传。
- manifest 中 artifact 的 `path` 不应作为远端客户端字段使用；应新增 `downloadUrl` 或让客户端按 GitHub asset name 匹配。
- manifest 应包含：
  - `version`
  - `channel`
  - `artifacts[].type`
  - `artifacts[].fileName`
  - `artifacts[].sha256`
  - `artifacts[].size`
  - `artifacts[].downloadUrl`（发布后可由 release uploader 回写，或客户端从 asset 列表匹配）
  - `minimumSupportedVersion`（可选，未来数据库迁移或破坏性升级时使用）
- 安装器应签名并时间戳。
- 如果 CI/CD 后续接管 release，需要把签名证书和 GitHub release upload 权限纳入 secret 管理。

### Documentation References

- GitHub REST API 的 latest release 接口返回最新 published full release，且 response assets 中包含 `browser_download_url` 与 `digest` 等字段：https://docs.github.com/en/rest/releases/releases#get-the-latest-release
- Inno Setup 官方命令行参数支持 `/SILENT`、`/VERYSILENT`、`/SUPPRESSMSGBOXES`、`/NORESTART`、`/CLOSEAPPLICATIONS` 等：https://jrsoftware.org/ishelp/topic_setupcmdline.htm
- Inno Setup 官方退出码定义可用于安装结果归类；非 0 代表未成功完成：https://jrsoftware.org/ishelp/topic_setupexitcodes.htm
- Inno Setup `PrivilegesRequired` 默认是 `admin`，会影响是否触发 Windows UAC：https://jrsoftware.org/ishelp/topic_setup_privilegesrequired.htm
- Microsoft SignTool 可用于签名、时间戳和验证文件签名，`verify` 会检查证书链、吊销状态和策略：https://learn.microsoft.com/en-us/windows/win32/seccrypto/signtool

### Open Product Decision

需要用户确认的关键产品决策：

**Curated 的默认自动更新体验要做到哪一级？**

推荐选择：**自动发现 + 可选自动下载 + 用户确认安装**。这能满足低打扰，同时不引入绕过权限和用户知情的风险。若未来要做到完全无感安装，应单独评估 per-user 安装模式或受限 updater service。

## Non-Goals

本期不做以下内容：

- 未经用户知情和授权的隐蔽更新
- 绕过 Windows UAC、SmartScreen、签名校验或系统权限边界
- 增量补丁升级
- 多更新通道切换（如 preview / beta / nightly）
- 非 GitHub Release 来源的更新检查
- 开发态 / Mock 模式下伪造“有更新”结果

说明：2026-05-13 后，`应用内下载 installer`、`显式触发安装` 以及 opt-in `autoDownloadUpdates` 后台准备安装包能力均已落地；`用户授权静默安装` 仍属于后续候选范围，不属于当前默认能力。

## Scope

本期范围仅覆盖：

- Settings -> About 页面中的版本与更新状态展示
- 启动后的自动更新检查
- 用户主动触发的手动更新检查
- 后端访问远端 GitHub Release 信息并返回统一结果
- 更新可用时的引导下载动作

## User Stories

### Story 1: 手动检查更新

作为正式安装包用户，我进入 About 页面后，希望点击“检查更新”按钮，就能立即知道当前安装包是不是最新版本。

### Story 2: 自动提示更新

作为正式安装包用户，我不想每次都自己去 GitHub 看 release；当有新版本时，应用应在合适时机自动提示我。

### Story 3: 明确的升级入口

作为发现有新版本的用户，我希望能直接点击按钮跳到官方 Release 下载页，而不是再自己搜索仓库。

### Story 4: 正常降级

作为开发环境或非正式打包用户，如果没有可靠的安装包版本号，应用应明确告诉我“当前环境不支持更新检查”，而不是给出误导性结果。

## Options Considered

### Option A: 前端直接请求 GitHub Releases API

优点：

- 实现路径最短
- 不需要新增后端接口

缺点：

- 会遇到跨域、限流、代理、认证和失败重试的一致性问题
- 难以复用现有后端代理配置
- 版本比对与缓存逻辑散落在前端，不利于后续桌面化演进

### Option B: 后端统一代理远端 Release 检查，前端只展示结果

优点：

- 可以复用现有后端网络能力和代理配置
- 版本比较、超时、缓存、错误处理都集中在一处
- 前端只关心 UI 状态，边界清晰

缺点：

- 需要新增后端接口与缓存状态

### Option C: 基于 release manifest 自建更新源

优点：

- 可以完全控制返回结构
- 未来适合扩展到多通道和增量更新

缺点：

- 当前用户明确希望比对远端仓库 Release
- 现阶段会额外引入一套“发布元数据托管约束”，超出本期目标

## Recommended Approach

采用 Option B。

即：

- 后端作为唯一“更新检查”执行者
- 后端定期访问 GitHub Releases API 的 latest release 信息
- 前端 About 页只展示统一 DTO，并提供手动触发按钮
- 自动检查仅做“发现并提醒”，下载与安装仍由浏览器打开官方 Release 页面完成

这是当前仓库最稳妥的方案，因为 Curated 已经是“本地前端 + Go 后端”的运行结构，且已有代理、健康检查、版本注入和 About 页展示基础。

## Product Decisions

- 更新源固定为该仓库的 GitHub Releases。
- 当前仓库固定为 `yepHiu/Curated`。
- 默认跳转与查询基地址固定为 `https://github.com/yepHiu/Curated/releases`。
- 本期只比较稳定正式版 release，不比较 draft 或 prerelease。
- 当前安装版本来源固定为 `installerVersion`。
- 若当前运行环境没有 `installerVersion`，视为“不支持更新检查”。
- 自动检查默认开启，但仅在正式安装包环境下执行。
- 手动检查始终可点击；若环境不支持，则返回明确原因。
- 发现新版本后，默认动作为“打开官方 Release 页面”，不在应用内直接下载。

## Functional Requirements

### 1. About 页展示更新状态

About 页除了现有的：

- App version
- Installer version

还需要新增“更新状态”信息块，至少覆盖以下状态：

- `未检查`
- `检查中`
- `已是最新版本`
- `发现新版本`
- `检查失败`
- `当前环境不支持检查更新`

当状态为“发现新版本”时，页面至少展示：

- 当前安装版本
- 远端最新版本
- 远端 release 标题或版本标签
- 发布时间
- “前往下载更新”按钮

当状态为“已是最新版本”时，页面至少展示：

- 当前安装版本
- 最近一次检查时间

当状态为“检查失败”时，页面至少展示：

- 失败状态文案
- 最近一次尝试检查时间
- “重试检查”按钮

### 2. 手动检查更新

About 页新增一个明确按钮：

- 文案建议：`检查更新`

行为要求：

- 用户点击后，立即发起一次强制检查
- 本次检查不受自动检查缓存周期限制
- UI 进入 `检查中` 状态
- 检查结束后立即更新状态展示
- 若发现新版本，可同时弹出轻量 toast，提示用户可前往下载

### 3. 自动检查更新

自动检查的产品行为如下：

- 仅在正式安装包环境下触发
- 应用启动后延迟一小段时间触发，避免与首屏加载抢资源
- 若最近一次成功检查仍在有效期内，则本次启动直接复用缓存结果，不重复打远端

推荐策略：

- 启动后约 `10-30 秒` 发起后台检查
- 成功结果缓存 `24 小时`
- 手动检查不受 `24 小时` 限制

自动检查发现新版本时：

- 不打断当前操作流
- 采用非阻塞提示
- About 页进入“发现新版本”状态
- 可选地在设置导航或 About 卡片上出现轻量提示标记

### 4. 版本比较规则

版本比较必须基于安装包语义版本，而不是构建时间戳。

规则：

- 本地版本使用 `installerVersion`
- 远端版本优先读取 GitHub Release 的 `tag_name`
- 允许 `v1.2.7` 与 `1.2.7` 归一化后比较
- 比较逻辑采用标准语义版本规则

如果远端 tag 不是可识别的语义版本：

- 本次检查记为失败
- 不允许用字符串字典序进行错误比较

### 5. 远端发布信息展示

当发现更新时，前端建议展示如下最小信息：

- 最新版本号
- 发布时间
- 发布页面链接
- 可选摘要：Release notes 首段或简短说明

本期不要求把完整 release note 内嵌到 About 页，但应保留以后扩展的结构空间。

### 6. 环境降级行为

以下情况不执行正常更新比较，而是降级展示：

- `VITE_USE_WEB_API=false`
- 后端健康检查拿不到 `installerVersion`
- 当前为开发态 / 非正式打包运行态

降级要求：

- 不显示“有更新”或“已是最新”这类误导性结论
- 明确展示“当前环境不支持更新检查”
- 手动检查按钮可隐藏，或保留但返回清晰提示

推荐做法：

- dev / mock 环境保留只读说明，不显示手动检查按钮

## UX Design

### About 页信息结构

推荐在 About 区块中形成三层信息：

1. `App version`
2. `Installer version`
3. `Update status`

当发现新版本时，`Update status` 卡片建议包含：

- 状态标题：`发现可用更新`
- 当前版本：`当前 1.2.7`
- 最新版本：`最新 1.2.8`
- 发布时间：`发布于 2026-04-20`
- 操作按钮：
  - `前往下载更新`
  - `重新检查`

当当前已是最新时：

- 状态标题：`当前已是最新版本`
- 辅助文案：`最近检查时间 ...`
- 操作按钮：`重新检查`

当检查失败时：

- 状态标题：`更新检查失败`
- 辅助文案：网络错误、超时或远端响应异常
- 操作按钮：`重试检查`

### 提示层级

本期不建议使用强制弹窗。

推荐优先级：

- About 页面内的状态卡片作为主承载
- 自动检查发现更新时，补一个轻量 toast
- 在主页面左侧品牌 / logo 入口附近增加一个轻量全局提示标签
- 后续若用户明确需要更强提醒，再评估启动后 banner 或全局提示点

### Global Logo Update Badge

考虑到用户并不一定会频繁进入 Settings -> About，推荐在主页面左侧导航顶部的品牌区，也就是当前 `Curated` logo 附近，增加一个轻量更新提示标签。

当前代码结构里，这个位置更贴近全局入口，而不是某个具体页面正文，因此适合作为“发现新版本后的持续提醒点”。

推荐方式如下：

- 当更新状态为 `update-available` 时，在侧边栏顶部 `Curated` logo 右侧展示一个小标签
- 标签文案优先使用短文案，例如：
  - `New`
  - `更新`
  - `v1.2.8`
- 推荐默认文案使用 `New`，hover / tooltip 再补充完整信息，避免 logo 区域被版本号撑宽
- 点击该标签或点击 logo 区域时，默认跳转到 `Settings -> About`，由 About 页面承接完整版本详情与“前往下载更新”动作

推荐视觉约束：

- 这是“轻提示”，不是主 CTA
- 不使用大面积红色警告样式
- 优先使用与品牌区协调的小号 pill badge 或小圆角标签
- 与 logo 保持并列关系，不覆盖品牌文字本身

推荐状态规则：

- 仅在 `update-available` 时显示
- `up-to-date`、`checking`、`error`、`unsupported` 状态下不显示该标签
- 若用户尚未触发自动检查且还没有结果，不预先显示占位提示

紧凑侧边栏行为：

- 当侧边栏处于 compact 状态且品牌文字收起时，不建议强塞完整文字标签
- compact 模式下可退化为一个小圆点或小角标
- 用户 hover 后通过 tooltip 展示：
  - `发现新版本`
  - `当前 1.2.7，最新 1.2.8`

交互定位：

- logo 区域标签只负责“提醒用户有更新”
- 真正的状态详情、发布时间、下载入口仍然放在 About 页面
- 这样可以避免首页导航区承载过多信息，保持主界面清爽

## Backend Design

## Remote Source

后端使用 GitHub Releases latest release 接口作为远端来源。

当前产品固定对应以下 GitHub 仓库：

- owner: `yepHiu`
- repo: `Curated`
- release page base URL: `https://github.com/yepHiu/Curated/releases`
- example tag URL: `https://github.com/yepHiu/Curated/releases/tag/v1.2.7`

后端职责：

- 请求远端 latest release
- 过滤 draft / prerelease
- 解析并归一化版本号
- 与本地 `installerVersion` 比较
- 生成统一返回 DTO
- 负责超时、失败、缓存与日志

## API Proposal

新增更新检查接口：

- `GET /api/app-update/status`
- `POST /api/app-update/check`

建议语义：

- `GET /status`
  - 读取当前缓存状态
  - 如果缓存缺失或过期，可由后端决定是否后台刷新
- `POST /check`
  - 强制立即检查
  - 用于用户手动点击“检查更新”

建议返回字段：

- `supported`
- `channel`
- `installedVersion`
- `latestVersion`
- `hasUpdate`
- `status`
- `checkedAt`
- `publishedAt`
- `releaseName`
- `releaseUrl`
- `releaseNotesSnippet`
- `source`
- `errorMessage`

补充约束：

- `releaseUrl` 必须返回该仓库下的官方 Release 页面链接
- 当检测到新版本时，前端“前往下载更新”按钮默认打开该 `releaseUrl`
- 若后端没有拿到更精确的单版本页面链接，至少返回仓库 Releases 首页 `https://github.com/yepHiu/Curated/releases`

其中 `status` 建议统一枚举为：

- `unsupported`
- `idle`
- `checking`
- `up-to-date`
- `update-available`
- `error`

## Caching And Persistence

后端需要持久化最近一次检查结果，避免每次启动都重复请求 GitHub。

推荐要求：

- 最近一次成功结果持久化保存
- 最近一次失败结果也保留基础信息，便于 About 页展示
- 自动检查优先复用缓存
- 手动检查可绕过缓存

推荐实现：

- 使用 SQLite 新增轻量状态表
- 每次检查完成后更新最近一条状态

至少需要保存：

- `installed_version`
- `latest_version`
- `status`
- `checked_at`
- `published_at`
- `release_url`
- `release_name`
- `error_message`

## Network And Proxy

由于仓库已有后端代理配置能力，本功能必须复用后端网络设置。

要求：

- 远端检查走后端统一 HTTP 客户端
- 若用户配置了代理，更新检查也应通过代理访问 GitHub
- 需要设置明确超时与错误分类，避免 About 页长时间悬挂在 `检查中`

建议：

- 请求超时控制在 `5-10 秒`
- 对 DNS 失败、超时、非 200 响应分别记录日志

## Frontend Design

前端不直接访问 GitHub。

前端职责：

- About 页读取更新状态 DTO
- 根据状态渲染不同 UI
- 提供“检查更新”按钮
- 在发现更新时提供“前往下载更新”按钮

前端建议新增一个小型更新状态 composable 或 helper，负责：

- 页面首次加载状态
- 手动检查 loading 状态
- DTO 到文案 / badge / CTA 的映射

## State Machine

前端 / 后端协作时，更新检查的核心状态机如下：

1. `unsupported`
2. `idle`
3. `checking`
4. `up-to-date`
5. `update-available`
6. `error`

状态迁移规则：

- 正式安装包首次进入 About 页：
  `idle -> checking -> up-to-date | update-available | error`
- 自动检查命中有效缓存：
  直接进入缓存对应状态，不重新跑 `checking`
- 用户点击“检查更新”：
  当前状态 -> `checking` -> 新结果状态
- dev / mock / 无 `installerVersion`：
  直接 `unsupported`

## Acceptance Criteria

满足以下条件即可认为该功能达到 v1 可发布标准：

1. 正式安装包打开 About 页时，能看到当前安装包版本和更新状态。
2. 用户点击“检查更新”后，能立即看到加载态，并在结束后得到明确结果。
3. 若远端 GitHub Release 版本高于本地安装包版本，页面明确展示“发现可用更新”。
4. 用户点击“前往下载更新”后，会打开官方 Release 页面。
5. 应用启动后的自动检查不会在 24 小时内重复频繁打远端。
6. 代理开启时，更新检查仍可正常工作。
7. 开发态 / Mock / 无安装包版本环境不会显示误导性的升级结果。
8. 远端异常、超时、限流或版本解析失败时，用户看到的是明确失败状态，而不是无限 loading。

## Test Scope

### Backend tests

- `installerVersion` 缺失时返回 `unsupported`
- 远端版本高于本地时返回 `update-available`
- 远端版本等于本地时返回 `up-to-date`
- 远端 tag 无法解析时返回 `error`
- draft / prerelease 不应被当作 latest stable update
- 缓存命中时不会重复请求远端
- 强制检查可以绕过缓存

### Frontend tests

- About 页正确渲染 6 类状态
- 手动检查按钮的 loading / success / error 展示正确
- `update-available` 状态下显示下载入口
- `unsupported` 状态下不显示误导性 CTA

### Manual verification

- 正式 release 包启动后，在网络正常情况下可自动拿到更新状态
- 在存在更高版本的测试场景下，About 页提示正确
- 断网或代理配置错误时，About 页显示失败并可重试

## Rollout Strategy

建议分两步上线：

### Phase 1

- About 页展示更新状态
- 手动检查更新
- 自动检查更新
- 打开 GitHub Release 页面下载

### Phase 2

- 全局轻提示或设置页导航角标
- 侧边栏品牌 logo 邻近的更新提示 badge / compact 角标
- 可选“跳过该版本”或“稍后提醒”
- 可选 preview / beta 渠道支持

### Phase 3

- 已实现：应用内下载 installer
- 待补：下载进度、取消、失败重试
- 已实现：下载后 SHA256 校验；manifest 解析与发布流程约束继续补齐
- 已实现：About 页提供“下载并安装 / 立即安装”
- 待补：安装前退出当前 Curated 进程并释放单实例锁

### Phase 4

- 已实现：opt-in `autoDownloadUpdates` 自动后台下载更新包
- 待补：下载完成后非阻塞通知
- 已实现：用户确认后安装
- 待补：用户明确开启后支持 Inno Setup 静默参数
- 待补：引入安装器和主程序代码签名校验

### Phase 5

- 评估 per-user 安装或 updater service
- 只有在架构允许时再承诺完全无感安装
- 评估回滚、最小支持版本、数据库迁移兼容和发布通道

## Docs Impact

如果进入实现，需要同步更新：

- `API.md`
- `.cursor/rules/project-facts.mdc`
- `README.md`
- `README.zh-CN.md`
- `README.ja-JP.md`

并补充：

- About 页更新检查能力说明
- 自动检查与手动检查行为
- 正式安装包与开发环境的行为差异

# Curated 全项目特性、优化与质量审计

日期：2026-07-19  
状态：in-progress（Milestone A～C 已完成；Milestone D / Personalization 进行中）  
关联需求：REQ-0003～REQ-0022  
范围：Vue 3 / TypeScript / Vite 前端、shadcn-vue UI、Go HTTP 后端、SQLite、Electron 桌面壳、发布脚本、测试、文档与产品路线

## 执行快照（2026-07-21）

| 里程碑 | 状态 | 已验证证据 / 下一步 |
|---|---|---|
| Milestone A / Release Safety | verified | 脱敏发布配置、loopback/LAN guard、Origin/Host、PIN 限速、可信会话、依赖审计、SQLite 外键均已提交并通过门禁 |
| Milestone B / Runtime Correctness | verified | active adapter、锁屏 hydrate、CI/e2e、375px 触控、事实文档与 Bundle hard budget 已完成；C5 闭环时进一步将首屏 JS 优化为 333469 raw / 120624 gzip |
| Milestone C / Data Longevity | verified | REQ-0014～REQ-0018 全部 verified：一致备份/离线恢复、可审计路径迁移、重启安全分块上传、只读离线安全 Library Health、持久化有界元数据修复与确认式审计清理；浏览器 QA、168/694 Vitest、4/28 Electron、4/4 Chromium e2e、Go test/vet、双构建、PRD 与 Bundle hard budget 全部通过 |
| Milestone D / Personalization | in-progress | 独立执行计划见 [2026-07-20-personalization-implementation-plan.md](2026-07-20-personalization-implementation-plan.md)；D1 Saved Views、D2 推荐解释/负反馈、D3 actor canonical merge 与 D4 Personal Insights 均已实现。D4 后完整非浏览器门禁为 180/745 Vitest、4/28 Electron、Go test/vet、双构建、类型/Lint 全通过；首屏 `346882/124701`、全量 `2127607/704237` 仍低于未放宽硬预算，Insights dynamic chunk 为 `9767/3277`。应用内 Browser 无实例且 Playwright `spawn EPERM`，故 REQ-0019～0022 均保持 implemented / 90，Milestone D 仍未 verified |
| Milestone E / Strategic Branch | not-started | 完成前只选择一个主分支，不并行铺开多个新产品域 |

## 1. 结论先行

Curated 已经具备成熟本地媒体资料库的主要骨架，不再适合按“早期原型”思路持续横向堆功能。当前已实现的能力包括：虚拟化资料库、影片详情与用户偏好、演员库、标签与回收站、导入与分片上传、扫描与元数据刮削、任务与 SSE、播放描述符与 HLS、萃取帧、每日推荐、PIN 锁、存储在线检测、应用内更新、Electron 托盘与 Windows 发布链。

当前最值得做的事情不是立即再开一个大领域，而是先完成四个发布级风险闭环：

1. 阻止发布包复制本机 `library-config.cfg`，消除代理、路径、播放器命令与 provider 配置泄漏。
2. 收紧默认网络暴露、CORS 与 PIN 解锁安全，明确 Desktop 与 Server/LAN 两种运行模式。
3. 升级存在已知高危漏洞的 Vite、`picomatch`、`defu` / `reka-ui` 依赖链。
4. 启用 SQLite 外键并修复既有孤儿数据；当前运行库已实际发现 8 条外键违规记录。

完成这四项后，下一批最有价值的产品方向是“资料库健康与修复、备份/恢复/迁移、大文件导入恢复、可信设备会话管理”。这些能力会提高用户长期使用 Curated 的信心，也同时为容器化、WebDAV、Android 或公开分发打基础。

## 2. 总体判断

| 维度 | 当前判断 | 主要依据 | 下一步重点 |
|---|---|---|---|
| 产品完整度 | 良好 | 浏览、整理、导入、播放、元数据、推荐、隐私锁、更新与桌面壳均已形成主流程 | 从“加入口”转向“长期数据治理与恢复” |
| 正确性 | 良好，但有两个启动链缺陷 | 类型、Lint、前后端测试全绿；实际运行发现 Mock 仍请求 Web API、锁屏产生 423 噪声 | 消除模块导入副作用与锁屏前 hydrate |
| 数据可靠性 | 需要尽快加固 | SQLite 文件 `quick_check=ok`，但外键未开启且已有 8 条孤儿记录 | 外键、清理迁移、备份、恢复、完整性检查 |
| 安全与发布 | 当前最大短板 | 默认全网卡监听、宽松 CORS、PIN 默认关闭、无解锁限速、本机配置进入 release example、5 个 high 漏洞 | 作为下一版 release blocker |
| 前端可维护性 | 中等 | 组件与服务边界总体正确，但少数页面仍为超大编排器 | 按行为边界继续拆分，不做大爆炸式重写 |
| 后端可维护性 | 中等 | 包结构清晰、测试较多，但 `server.go` / `app.go` 继续承担过多职责 | 拆 route/handler/service，维持合约稳定 |
| 自动化质量 | 单测强，交付门禁弱 | 661 个前端测试、28 个 Electron 测试、Go 全包测试通过；仓库没有实际 CI workflow | 建 Windows/Linux CI、依赖审计、基础 e2e |
| UI/UX | 桌面良好，移动端需专项收口 | 桌面信息层级与视觉一致性较好；375px 无横向溢出，但排序区换行、标签裁切、触控尺寸不足 | 一轮移动端与无障碍专项 |
| 文档治理 | 事实文档部分良好，历史文档明显漂移 | `project-facts.mdc` 较新；feature inventory / 产品文档仍称 Electron、SSE 未实现 | 建“事实源 + 需求台账 + 历史方案”状态治理 |

## 3. 本次验证范围与结果

### 3.1 仓库规模快照

| 项 | 数量 / 规模 |
|---|---:|
| 非构建产物文件 | 约 914 |
| Vue SFC | 162 |
| 前端 / Electron TypeScript | 309 |
| Go 源文件 | 197 |
| 测试 / spec 文件 | 约 251 |
| SQLite migrations | 26 |
| `docs/plan` Markdown | 111 |
| PRD CSV 中真实产品需求 | 1 条（另 1 条是 PRD 流程本身） |

### 3.2 自动化检查

| 检查 | 结果 | 证据 |
|---|---|---|
| `pnpm typecheck` | 通过 | `vue-tsc -b` 无错误 |
| `pnpm lint` | 通过 | ESLint 无错误 |
| `pnpm test` | 通过 | 162 个测试文件、661 个测试全部通过 |
| `pnpm test:electron` | 通过 | 4 个文件、28 个测试全部通过 |
| Python 脚本测试 | 通过 | 6 个测试通过 |
| `go test ./...` | 通过 | 所有 Go 包通过 |
| `go vet ./...` | 通过 | 无项目诊断 |
| `pnpm build` | 通过，有 bundle 告警 | 2802 modules；`vendor` 908.56 kB，超过 500 kB 告警阈值 |
| `pnpm build:electron:main` | 通过 | Electron TypeScript 与 preload copy 完成 |
| `pnpm audit --prod --audit-level=high` | 失败 | 共 10 个漏洞：5 high、5 moderate |
| SQLite `PRAGMA quick_check` | 通过 | 返回 `ok` |
| SQLite `PRAGMA foreign_key_check` | 失败 | 8 条孤儿记录 |

### 3.3 Go 覆盖率信号

覆盖率不应单独作为质量目标，但可以帮助决定下一批测试投资。

高覆盖包：

- `scanner` 84.1%
- `curatedexport` 82.8%
- `executil` 83.3%
- `clienttracker` 80.1%
- `logging` 79.4%
- `tasks` 72.4%

中等覆盖包：

- `storage` 60.6%
- `assets` 61.5%
- `appupdate` 62.5%
- `server` 48.0%
- `playback` 43.4%
- `storagehealth` 44.9%
- `app` 41.3%
- `config` 39.5%

风险较高但覆盖偏低的包：

- `metatube` adapter 3.2%
- `desktop` 5.6%
- `cmd/curated` 12.7%
- `version` 13.2%
- `library` 26.2%
- `librarywatch` 29.9%
- `nativeplayer` 33.3%
- `browserheaders`、`curatedthumb`、`devmetrics`、`shellopen`、`webui` 没有直接测试

建议优先补“发布启动、桌面进程、元数据 adapter 边界、Web 静态托管、原生播放器命令、安全 header”测试，而不是对所有包强制统一覆盖率阈值。

### 3.4 渲染与交互验证

Browser 插件已安装，但当前会话可用浏览器列表为空，因此按前端测试规范回退到 Playwright CLI。

UI/UX 审查技能包提供了完整检查表，但其 `scripts/search.py` 实际安装路径缺失，无法运行附带的 design-system / domain 搜索命令；本次改为直接执行该技能的可访问性、触控、响应式、性能、主题与交互检查表，不影响代码与实际页面证据。

验证环境：

- Web API：`http://127.0.0.1:5173/`
- Mock 审计 mode：`http://127.0.0.1:5174/`
- 桌面视口：1280 × 720
- 移动视口：375 × 812

通过项：

- 页面标题为 `Curated`，首屏不是空白页，也没有 Vite error overlay。
- Web API 启用 PIN 时正确重定向 `/lock?redirect=/`。
- Mock 首页与资料库可以渲染完整示例数据。
- “首页 → 全部影片”导航成功，URL 更新为 `/library?...`。
- 移动端导航抽屉可以打开，存在可访问名称“导航”和明确关闭按钮。
- 375px 下 `documentElement.scrollWidth === innerWidth`，没有页面级横向溢出。
- 当前可见影片图片均有 `alt`，当前可见按钮均有可辨识名称。

发现项：

- Mock UI 明确显示“演示模式（无后端）”，仍产生一次真实 `/api/library/movies` 请求并返回 423。
- 锁屏产生 `/api/playback/progress` 与 `/api/library/played-movies` 两条 423 console error。
- 移动端排序 Tabs 的“评分”换到第二行，排序区与“批量管理”视觉对齐不稳定。
- 移动端卡片末尾标签 / `+1` 在窄卡中出现裁切，信息密度与卡片宽度不匹配。
- 多个移动端目标小于 44px：菜单 36×36、通知 36×36、主题 switch 32×18、排序 tab 高 38、批量管理高 32、侧栏导航行高 40。
- 资料库页可见标题从影片卡片 `h3` 开始，没有页面级 `h1` / `h2`，不利于屏幕阅读器理解页面结构。
- dev 性能条会覆盖移动端底部内容；它只在开发态出现，不是生产缺陷，但会影响开发阶段的移动端视觉判断。

## 4. P0：下一版发布前必须处理

### P0-1 发布配置与隐私泄漏

#### 事实

`scripts/release/release_lib/build_steps.py` 当前把：

```text
config/library-config.cfg
```

直接复制成：

```text
resources/app/runtime/config/library-config.example.cfg
```

`config/` 被 Git 忽略，仓库中没有受控的 `library-config.example.cfg`。当前本机配置已检测到以下非空状态：

- 代理已配置；
- 原生播放器命令已配置；
- 默认导入库 ID 已配置；
- 日志目录已配置；
- provider chain 有 3 项。

这意味着 release 包可能携带本机网络配置、路径、命令和资料库标识。

#### 建议首切片

1. 新增 tracked、脱敏、最小化的 `config/library-config.example.cfg`。
2. release builder 只允许复制该 example，禁止读取本机 `library-config.cfg`。
3. 打包前增加静态审计：拒绝绝对路径、代理凭据、本机用户名、库 ID、播放器绝对命令和非中性 provider。
4. 增加 release 脚本测试，构造含敏感值的本机配置并证明产物不含它们。
5. 对已有 installer / portable 包做一次离线字符串审计；按仓库规则保留既有产物，不主动删除。
6. 面向公开分发时增加 Authenticode 签名、签名验证与 SBOM；SHA256 只提供完整性，不应被描述为发布者身份认证。

#### 验收

- 干净 clone 无本机配置也能构建 release。
- release staging 中 example 内容完全来自 tracked 文件。
- 打包测试能阻止敏感值进入产物。

### P0-2 默认网络暴露、CORS 与 PIN 安全

#### 事实

- dev / release 默认地址分别是 `:8080` / `:8081`，即监听所有网卡。
- PIN 默认关闭；PIN 关闭时所有 `/api/*` 业务接口无需认证。
- `withCORS` 对任意非空 `Origin` 原样回显 `Access-Control-Allow-Origin`，并允许 credentials。
- PIN 允许 4～8 位数字，但解锁接口没有失败次数限制、退避或临时锁定。
- `lanRequiresPin` 当前只在 DTO、SQLite 与设置页中读写，认证中间件没有读取它或区分 local / remote request；开关的名称与实际行为不一致。
- trusted-forever cookie 最长 10 年；当前没有设备会话列表与远程撤销 UI。
- cookie 是 `HttpOnly + SameSite=Lax`，本地 HTTP 下没有 `Secure`；这对纯 loopback 可以接受，但不能直接等同于可信 LAN / Internet 安全。

#### 建议首切片

1. Desktop 默认改为 `127.0.0.1:8080/8081`。
2. 新增显式 `serverMode` / `lanEnabled`，只有用户主动开启才绑定 LAN。
3. LAN 开启前强制完成 PIN 或更强认证初始化，不能只显示提示。
4. CORS 改为明确 allowlist；对状态变更请求校验 `Origin` / `Host`。
5. 为 `setup-pin` / `unlock` 增加按 IP + client key 的限速、指数退避与审计日志。
6. 明确定义并真正实现 `lanRequiresPin`；如果产品不需要“本机免 PIN、LAN 需 PIN”的分叉，就删除这个误导性开关并使用更简单的全局锁语义。
7. 新增 auth session 列表、当前设备标识、逐个撤销与“撤销其他设备”。
8. 容器 / 远程访问继续要求反向代理 TLS 与外层身份认证；PIN 只作为第二层应用锁。

#### 验收

- 新安装桌面版默认只能从本机访问。
- 未设置 PIN 时无法开启 LAN mode。
- 未在 allowlist 的浏览器 Origin 不能读取 API response。
- PIN 暴力尝试触发可测试的 429 / retry-after 行为。
- 用户能看到并撤销 trusted-forever session。

### P0-3 依赖漏洞升级

#### 事实

生产依赖审计报告 10 个漏洞，其中 5 个 high：

- Vite 当前 8.0.0；Windows 文件读取 / `server.fs.deny` 绕过需要至少升级到 8.0.16，当前 latest 为 8.1.5。
- `picomatch` 当前 4.0.3；ReDoS 修复版本为 4.0.4。
- `reka-ui` 当前 2.9.2，通过 `defu` 6.1.4 引入 prototype pollution；`defu` 修复版本为 6.1.5，`reka-ui` 当前 latest 为 2.10.1。

#### 建议首切片

1. 优先升级 Vite / `@tailwindcss/vite` / Tailwind 同一兼容批次。
2. 升级 `reka-ui` 并确认 lockfile 中 `defu >= 6.1.5`。
3. 确认所有 `picomatch` 路径均解析到 `>=4.0.4`。
4. 跑完整 typecheck、lint、Vitest、Electron、Go、build 与桌面冒烟。
5. 在 CI 中加入 `pnpm audit --prod --audit-level=high`。

不要把 `vue-i18n 9 → 11`、`vue-virtual-scroller beta → 3`、Electron 42 → 43 等大版本迁移混入安全修复批次。

#### 验收

- high 漏洞归零。
- build 不再使用 Vite 8.0.0 / `picomatch` 4.0.3 / `defu` 6.1.4。
- 现有 689 个前端 + Electron 测试继续通过。

### P0-4 SQLite 外键与孤儿数据

#### 事实

`NewSQLiteStore()` 使用 `sql.Open("sqlite", path)` 并限制单连接，但没有开启 `PRAGMA foreign_keys=ON`。当前运行库检查结果：

```text
PRAGMA foreign_keys = 0
PRAGMA quick_check = ok
```

`PRAGMA foreign_key_check` 已发现 8 条违规：

- `playback_progress` 4 条；
- `library_played_movies` 4 条；
- 均指向不存在的 `movies`。

数据库文件没有损坏，但 schema 中的 `ON DELETE CASCADE` 当前没有按预期执行。

#### 建议首切片

1. 在连接初始化时显式开启并验证 `foreign_keys=ON`，连接失败时不要静默继续。
2. 新增 store-level 测试，证明删除 movie 会 cascade 清理相关表。
3. 新增幂等数据修复 migration / maintenance command，清理既有孤儿行。
4. 升级前先生成 SQLite 在线备份或停机备份，并记录修复行数。
5. 评估设置 `busy_timeout` 与 WAL；当前是 `journal_mode=delete`、`busy_timeout=0`、单连接，应先 benchmark 再调整。
6. 增加 Settings Maintenance 的 `quick_check` / `foreign_key_check` 只读诊断入口。

#### 验收

- 新连接 `PRAGMA foreign_keys` 返回 1。
- `PRAGMA foreign_key_check` 返回空。
- 删除影片后播放进度、已播放、评论、每日时长等引用表无孤儿记录。
- 修复过程可回滚且不会删除有效用户数据。

## 5. P1：稳定性、长期数据与交付能力

### P1-1 消除 Web / Mock adapter 导入副作用

#### 根因

`src/services/library-service.ts` 顶层同时静态导入 Mock 和 Web adapter。即使最后返回 Mock，`web-library-service.ts` 仍在模块加载时执行：

```ts
export const webLibraryService = createWebLibraryService()
```

而 `createWebLibraryService()` 立即 `ensureLoaded()`，因此 Mock 模式产生真实 API 请求。

#### 建议

- 移除 adapter constructor 的网络副作用；提供显式 `start()` / `ensureLoaded()`。
- 只由选中的 adapter 启动。
- 如果需要 bundle 隔离，使用异步 service bootstrap 或 adapter registry，但不要让 views 直接动态导入具体 adapter。
- 新增集成测试：`VITE_USE_WEB_API=false` 时 mock 全流程不得调用 `fetch`。

### P1-2 锁屏启动顺序与 console 健康

`main.ts` mount 后无条件启动 auth idle monitor，然后在 idle callback hydrate 播放进度与已播放列表。Web API 已锁定时，这两项请求返回 423，虽然 UI 最终正确进入锁屏，但 console 每次都有错误。

建议：

1. 启动先读取 auth status。
2. 只有 `unlocked=true` 才 hydrate 受保护业务状态。
3. 解锁成功后统一触发一次 hydrate。
4. 423 作为认证状态转换处理，不进入普通 resource error / console error。
5. e2e 覆盖“锁定启动 → 无受保护请求 → 解锁 → hydrate 一次”。

### P1-3 大文件导入 Phase 2

当前 resumable upload 的 session 与 chunk map 保存在进程内：

```go
sessions map[string]*movieImportUploadSession
```

字节虽然写入 `.curated-import/<uploadId>`，但进程重启后 session 丢失，也没有启动 janitor 统一治理残留 staging。

建议首切片：

- SQLite 持久化 upload manifest、状态、chunk bitmap、目标路径与更新时间；
- 启动时重建可恢复 session；
- janitor 清理超时/已中止/无法恢复的 staging；
- UI 显示速度、ETA、已重试次数；
- 第二切片再做暂停/继续与跨页面恢复。

### P1-4 备份、恢复与路径迁移

审计时 Curated 已保存评分、收藏、评论、播放进度、PIN 设置、推荐状态、萃取帧、路径绑定与更新状态，但没有受支持的备份/恢复工作流。执行更新：REQ-0014 已完成一致备份、包验证、恢复预检、受 PIN 保护的 Settings 维护入口与显式离线恢复；REQ-0015 已完成只读 dry-run、白名单路径映射、Windows/UNC/Unix 与 Windows→Unix 语义、目标/冲突检查、迁移前验证备份、单事务 apply/audit、binding reset 和回滚证据。两项均通过对应全量门禁，本节已闭环。

建议：

- Settings Maintenance 增加“创建备份包”和“验证备份包”；
- SQLite 使用安全 backup API / `VACUUM INTO`，不要在写入中直接复制 db 文件；
- 备份 manifest 记录 schema version、应用版本、配置版本、资产范围与校验和；
- 恢复先预检版本、磁盘空间、目标为空/可覆盖策略；
- 增加离线路径映射 CLI，支持 Windows → Linux / 容器迁移并可 dry-run、回滚；
- 默认不把媒体原文件打进备份，只备份索引与用户资产，提供可选范围。

### P1-5 资料库健康与修复工作台

这是当前最推荐的新产品功能。它能把散落的技术诊断变成用户可操作的长期维护体验。

第一版聚合：

- 离线 / 卷不匹配路径；
- 重复番号、重复文件指纹或同片多版本；
- 缺封面、缺预览、缺演员头像；
- 最近刮削失败及 error category；
- 源视频丢失、不可读、零长度；
- 孤儿播放进度 / 评论 / tag link；
- 残留 `.curated-import`；
- `quick_check` / `foreign_key_check` 摘要。

第一版只做“检测 + 定向行动”，不要直接自动删除：重新刮削、重新扫描、清理孤儿状态、打开路径、忽略、导出诊断。

### P1-6 CI 与通用 e2e 门禁

仓库已有丰富测试，但没有 `.github/workflows`。`tests/` 只有 display scaling spec，且仓库规则不允许日常自动运行耗时的 `test:display`。

建议最小 CI：

- Windows：pnpm install frozen lockfile、typecheck、lint、Vitest、Electron test、build、Go test、release script tests；
- Linux：Go test、前端 build、未来 container build；
- 依赖审计：high 漏洞阻断；
- 基础 e2e：Mock 首页、资料库导航、锁屏、Settings、404；
- artifact：失败截图与 console log；
- display scaling 保留为手动 / release workflow，不放每次 commit。

### P1-7 文档事实同步与需求台账治理

当前 `docs/features/2026-05-03-feature-inventory.md` 仍标记：

- Electron shell 未实现；
- SSE 未实现；
- 当前是 Web-first；
- 只有旧路由集。

`docs/product/2026-03-20-jav-libary.md` 也仍写“当前没有 Electron、preload、服务层”。这些与代码和 `project-facts.mdc` 明显冲突。

同时，`docs/plan` 有 111 个 Markdown 方案，而 `docs/prd/requirements.csv` 只有 1 条真实产品需求，无法表示已批准、进行中、已验证与放弃的路线。

建议：

1. 更新 feature inventory 到当前版本，新增 Security、Electron、SSE、Connected Clients、Storage Health、Installer Update、Actor Detail。
2. 产品文档的历史段落明确标注“历史快照”，或更新 Current State。
3. 从本报告选中的 P0/P1 才进入 PRD CSV；未批准灵感继续留在 plan，不全部塞入台账。
4. 为 plan 增加状态：research / proposed / approved / in-progress / verified / superseded。
5. 在文档检查中验证 Current / Target / Pending 关键词与关键事实。

## 6. P1/P2：性能与可维护性优化

### 6.1 前端大文件拆分

当前主要复杂度热点：

| 文件 | 行数 | 复杂度信号 |
|---|---:|---|
| `PlayerPage.vue` | 2749 | 76 个函数、36 个 computed、7 个 watch、14 处事件监听 |
| `SettingsPage.vue` | 2308 | 37 个函数、31 个 computed、7 个 watch |
| `CuratedFramesLibrary.vue` | 1718 | 分页、选择、编辑、导出、删除编排仍集中 |
| `LibraryView.vue` | 951 | 路由、选择、批量操作与数据状态集中 |
| `AppShell.vue` | 932 | 壳层同时承载较多全局系统 |

建议按行为拆，而不是按模板片段拆：

- Player：media lifecycle、HLS session、progress/watch-time、chrome/input、capture、stats 分为可独立测试的 composables。
- Settings：父组件改成 section controller + DTO draft，减少几十个 save handler 的重复编排。
- Curated Frames：把 query/pagination、selection/export、edit dialog orchestration 分开。
- Library：把 route query state、batch mutations、gamepad selection 分开。

不要为拆文件引入全局 Pinia 大迁移；项目现有 composable + service 边界仍适合渐进重构。

### 6.2 后端大文件拆分

| 文件 | 行数 | 主要职责混合 |
|---|---:|---|
| `backend/internal/server/server.go` | 3147 | 路由注册、影片/演员、播放、设置、导入、路径、任务、provider、CORS |
| `backend/internal/app/app.go` | 2685 | 配置、监听、扫描、刮削、播放、更新、HTTP 装配、stdio command |
| `contracts.go` | 1016 | 多领域 DTO / command / error code |
| `playback/manager.go` | 1089 | session 生命周期与 FFmpeg 编排 |

建议：

- `server.go` 先按 handler 文件拆：settings、library paths、provider/proxy、movie CRUD；保留一个集中 route registrar。
- `app.go` 按 service facade 拆：SettingsService、ScanCoordinator、ScrapeCoordinator、PlaybackFacade、UpdateFacade。
- `contracts` 按领域拆文件但保持同一 package，避免一次性 transport 重构。
- 每次只拆一个行为边界，保持 API、错误码和测试不变。

### 6.3 Bundle 与首屏性能

优化前生产构建主要体积：

- `vendor`: 908.56 kB / gzip 336.53 kB；
- app entry: 195.59 kB / gzip 65.78 kB；
- `reka-ui`: 181.42 kB；
- lucide icons: 129.48 kB；
- Settings chunk: 190.11 kB；
- global CSS: 171.09 kB。

建议：

1. 用 bundle analyzer 确认 908 kB vendor 的真实组成。
2. 避免把不活跃 adapter 同时 eager import；修复 Web/Mock 副作用也能帮助拆包。
3. 检查 icon barrel import，继续保持按图标导入。
4. Settings 子区可按 tab 延迟加载，但不要牺牲状态一致性。
5. 大型仅播放器依赖继续维持按需加载。
6. 建体积预算：initial gzip、单 route gzip、字体总量，并在 CI 记录趋势。

执行结果（2026-07-20）：已移除 catch-all `vendor`，把 HLS、拼音搜索与图片轮播拆为具名 chunk，详情图集 inner viewer 改为异步加载；`pnpm build` 通过 `vite.bundle-budget.ts` 强制首屏、全量与关键 chunk hard budget，并生成无绝对路径的 `dist/bundle-analysis.json`。当前首屏静态闭包为 **448553 raw / 156004 gzip**，全量 JS 为 **2153344 raw / 713410 gzip**；HLS、拼音搜索和 Settings 均不在首屏闭包。

### 6.4 shadcn-vue 与 UI 规范收口

shadcn-vue CLI 识别项目为 Vite + TypeScript + Tailwind v4，配置与 alias 正常。当前业务 UI 总体遵守语义 token，但仍有一些渐进清理项：

- 部分业务组件直接使用 raw `<button>`，可以在不破坏特定交互的前提下复用 Button variant。
- 若干 `space-y-*` 与 `w-* h-*` 组合偏离当前 shadcn 规范。
- NotificationCenter 仍有硬编码状态色；应映射到 semantic status tokens。
- `DialogScrollContent` 关闭图标仍用 `w-4 h-4`，可与当前 icon sizing 规范统一。
- 不应为了“规则全绿”机械替换所有自定义播放器控件；播放器和虚拟卡片属于需要保留行为特性的例外面。

## 7. UI/UX 专项建议

### 7.1 保留的优点

- 桌面端信息架构清楚：侧栏、顶栏、内容面分工稳定。
- 深色主题、粉色品牌色与中性色层次一致，语义色基础较完整。
- 影片卡 poster-first，详情 / 播放器方向与项目规范一致。
- 侧栏和移动抽屉均有明确当前态。
- 图片 alt、按钮命名、移动导航 dialog title 的基础可访问性良好。
- 375px 无页面级横向滚动。

### 7.2 移动端首批修复

1. 排序区在 375px 使用单行横向滚动、紧凑 Select 或两行但等宽对齐，不让“评分”独占第二行 223px。
2. 批量管理与排序保持同一高度和对齐基线。
3. 影片卡在单列移动端适度增宽，或让 tag row 明确 `overflow-hidden + +N`，避免末项半截露出。
4. 菜单、通知、主题、Tabs、批量管理、侧栏行的可点击区域至少 44×44 CSS px；视觉图标可更小，但 hit area 不能更小。
5. 资料库页面增加 screen-reader-only `h1`，卡片标题可继续为 `h3`，中间增加结构性 `h2` 或调整等级。
6. 检查 200% text zoom 与中英文长文案；不只检查 DPR。
7. 为 `prefers-reduced-motion`、键盘 tab 顺序、焦点返回、dialog escape 增加 e2e。

### 7.3 非紧急视觉打磨

- 搜索框、添加影片、通知与主题控件在窄宽度下可以采用优先级折叠，避免第二行过密。
- 移动单列海报可以扩大视觉占比，减少两侧无效留白。
- 对状态色做 WCAG 对比度自动检查，尤其是 muted text、状态点与粉色小字号。
- dev 性能条在移动视口默认收起，或为内容预留仅开发态 bottom inset。

## 8. 推荐的新特性需求

### 8.1 第一梯队：现在最值得做

| 优先级 | 特性 | 用户价值 | 第一切片 |
|---|---|---|---|
| P1 | 资料库健康与修复 | 让长期使用后的重复、缺图、失联、刮削失败、孤儿数据变得可见可修 | 只读健康扫描 + 定向行动 |
| P1 | 备份 / 恢复 / 迁移 | 用户敢长期积累评分、评论、进度和萃取帧 | SQLite 安全备份 + manifest + 恢复预检 |
| P1 | 导入会话持久恢复 | 多 GB 导入不因刷新或重启作废 | session SQLite + startup janitor + ETA |
| P1 | 可信设备与会话管理 | 补齐 PIN 与 Connected Clients 的安全闭环 | session 列表、撤销、撤销其他设备 |
| P1 | 元数据修复队列 | 把逐片重试变成按错误分类的批量修复 | missing/failed 聚合 + 批量重试 |

### 8.2 第二梯队：在可靠性基线后增加产品价值

| 优先级 | 特性 | 用户价值 | 第一切片 |
|---|---|---|---|
| P2 | 智能集合 / Saved Views | 保存“未看、五星、某演员、某标签、最近导入、4K”等常用筛选 | localStorage 保存查询，稳定后迁 SQLite |
| P2 | 推荐解释与反馈 | 让每日推荐可理解、可调教 | “为什么推荐” + 少推荐 actor/studio/tag |
| P2 | 演员别名、规范名与合并 | 解决同一演员多写法、不同 provider 名称不一致 | canonical actor + alias 表 + dry-run merge |
| P2 | 观看洞察 | 利用已有 watch-time、rating、history 形成用户价值 | 按演员/标签/片商的时长与完成率 |
| P2 | 播放诊断产品化 | 用户能知道 direct/remux/transcode、失败原因与硬件回退 | stats overlay 增加用户可读 session drill-down |
| P2 | Electron 原生播放器窄桥 | 提升浏览器不易解码格式体验 | preload 仅暴露受控 launch capability |
| P2 | 萃取帧集合 / Storyboard | 把单帧升级为可整理和分享的素材集 | collection 模型 + ZIP/contact sheet 导出 |

### 8.3 条件性战略方向

#### 如果目标是公开分发

`2026-07-18-curated-compliance-and-metadata-source-decoupling.md` 中的 P0/P1 应立即前移；公开核心默认不访问成人元数据源，release 配置必须可复现且无本机状态。若要真正形成中性公开 Core，需要完成外部 provider extension 边界，而不是隐藏开关。

#### 如果目标是 Linux / fnOS Server

先做网络安全、备份、路径迁移、上传恢复和 Intel GPU capability spike，再做 Docker FPK。不要先把当前 Windows release 脚本硬改成多平台发布器。

#### 如果目标是 WebDAV

先验证系统已挂载路径的兼容性；原生 WebDAV 先只读 MVP，不要一开始实现导入、MOVE、删除和整理。SQLite 与 runtime 继续放本地盘。

### 8.4 暂不建议立即开做

- 漫画库完整 MVP：设计已经充分，但会新增独立数据表、扫描器、阅读器、缓存、导入与路由，是一个产品域，不应与当前 P0/P1 并行。
- Android 原生 App：LAN 安全、设备会话、播放契约与 Server mode 还未稳定。
- 原生 WebDAV 写入：MOVE/PUT/覆盖/Range/鉴权差异很大，应晚于只读 PoC。
- 完整多用户与权限系统：当前单用户本地产品先做好设备会话与访问范围即可。
- 深度 WebHID / node-hid / DualSense LED 与自适应扳机：价值低于数据可靠性和播放诊断。

## 9. 推荐实施顺序

### Milestone A：Release Safety（1～2 周）

1. 脱敏 example config + release 敏感字段审计。
2. 依赖漏洞升级。
3. Desktop loopback 默认 + CORS allowlist。
4. PIN unlock 限速与 LAN enable guard。
5. SQLite foreign keys + 孤儿数据修复。

退出标准：可以明确回答“这个包不会泄漏开发机配置、不会默认暴露到 LAN、依赖无 high 漏洞、数据库引用一致”。

### Milestone B：Runtime Correctness（1～2 周）

1. Web/Mock adapter 无副作用 bootstrap。
2. 锁屏前不 hydrate 受保护状态。
3. 基础 e2e 与 CI。
4. feature inventory / current-state 文档同步。
5. 移动端触控与排序栏修复。
6. Bundle 分块、分析报告与 hard budget。

退出标准：Mock 不发后端请求；锁屏 console 干净；PR 必须经过自动门禁；375px 主流程可操作。

### Milestone C：Data Longevity（2～4 周）

1. 备份与恢复预检。
2. 路径迁移 CLI。
3. 上传 session 持久化与 janitor。
4. Library Health 第一版。
5. 元数据修复队列。

退出标准：用户可以备份、验证、恢复；重启不丢上传；资料库问题有统一入口。

### Milestone D：Personalization（2～4 周）

1. Saved Views。
2. 推荐解释与负反馈。
3. 演员 alias/canonical merge。
4. Insights。

退出标准：Curated 从“能管理媒体”提升为“能适应用户习惯并解释推荐”。

### Milestone E：战略分支

只选择一个主分支：

- 公共 Core / metadata extension；
- Linux / fnOS Server；
- WebDAV 只读；
- 漫画库；
- Android 客户端。

不要同时启动多个新运行形态或产品域。

## 10. 建议拆成的最小修改单元

1. `fix(release): use tracked sanitized library config example`
2. `test(release): reject sensitive local config in staged package`
3. `fix(storage): enable sqlite foreign key enforcement`
4. `fix(storage): repair orphan playback state`
5. `fix(security): bind desktop backend to loopback by default`
6. `fix(security): restrict cors origins and validate state-changing requests`
7. `feat(security): rate limit pin unlock attempts`
8. `feat(security): list and revoke trusted sessions`
9. `chore(deps): patch vite picomatch and defu advisories`
10. `fix(frontend): avoid initializing inactive web adapter`
11. `fix(frontend): hydrate protected state after unlock`
12. `test(e2e): cover mock navigation and locked startup`
13. `ci: add frontend backend build and audit gates`
14. `fix(ui): improve mobile library sorting and touch targets`
15. `docs: refresh current feature inventory and roadmap status`

## 11. 最终建议

如果只选择一个下一步，不应先做漫画、Android 或原生 WebDAV，而应做一个“Release Safety + Data Integrity”批次：发布配置脱敏、网络/CORS/PIN 加固、依赖漏洞升级、SQLite 外键与孤儿清理。

如果再选择一个用户可见特性，优先做“资料库健康与修复工作台”，并把备份/恢复作为同一长期数据可信方向。它能复用当前已有的存储状态、任务、provider health、错误分类和通知系统，投入产出比高于再增加一个孤立页面。

Curated 当前的优势是基础能力已经足够丰富；下一阶段的产品质量来自“默认安全、数据可恢复、问题可诊断、长期可维护”，而不是功能数量本身。

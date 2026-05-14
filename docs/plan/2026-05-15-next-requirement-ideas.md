# Curated 后续需求灵感清单

日期：2026-05-15

本文是基于当前仓库代码、`.cursor/rules`、`docs/features`、近期 `docs/plan` 与 git 历史整理出的候选需求清单。它不是已批准实施计划，主要用于后续筛选方向、拆 PRD 或生成更细实施计划。

## 观察信号

- Electron 桌面壳、托盘常驻、release 打包链已经推进到默认生产入口，但窗口状态记忆、托盘日志入口、离线资源、本地播放器深度集成仍是自然下一步。
- 后端本质上已经是本地 HTTP + LAN 可访问服务，但当前没有完整鉴权、连接设备可见性和访问范围控制；如果继续强化远程访问或 Server Mode，这会变成前置需求。
- 通知中心、任务系统、导入、更新、存储健康已经有很多后台事件，但前端仍主要靠轮询和 localStorage；SSE/WebSocket 与后端持久通知可以形成更一致的事件面。
- 大文件导入已具备 resumable chunk endpoint，但文档仍标记 SQLite 会话持久化、启动清理、速度/ETA、暂停/恢复和 benchmark 为后续项。
- 首页推荐、演员库、标签、收藏、评分、观看历史和策展帧已经有大量“个性化信号”，但尚未形成更明确的智能集合、推荐反馈和库卫生工作流。
- `docs/features/2026-05-03-feature-inventory.md` 仍把 Electron shell 标为 Target，已经落后于当前事实；`docs/prd/requirements.csv` 目前只有 PRD 工作流本身，需求台账还没有承载真实产品条目。
- 代码层仍存在几个大文件：`PlayerPage.vue` 约 2733 行，`SettingsPage.vue` 约 2241 行，`backend/internal/server/server.go` 约 3125 行，`backend/internal/app/app.go` 约 2694 行。继续拆分会降低后续功能接入风险。

## 候选需求

| 优先级 | 方向 | Idea | 用户价值 | 第一切片 |
|---|---|---|---|---|
| P0 | 安全 / LAN | LAN 访问控制与本机/局域网开关 | 避免默认暴露本地媒体库；给后续远程访问、Server Mode、安全提示打底 | `settings` 增加监听范围与访问 token；后端对非 loopback 请求校验 token；README/API 同步 |
| P0 | 安全 / 可见性 | Connected Clients 连接设备列表 | 用户能看到当前有哪些浏览器/设备在访问 Curated，降低 LAN 场景的不透明感 | 落地 `docs/features/2026-05-03-connected-clients.md` 的 in-memory tracker + Settings Overview 卡片 |
| P1 | 桌面体验 | Electron 窗口状态记忆 + 托盘诊断菜单 | 桌面端更像原生应用；用户能直接打开日志、设置、Web 入口并理解后端归属 | 保存窗口位置/大小/最大化；托盘增加“打开日志目录”“复制后端地址”“当前状态” |
| P1 | 桌面体验 | 完全离线桌面资源 | 安装包在断网环境下播放器、字体、HLS 都能正常工作 | 将 hls.js 改 npm 动态 import；字体本地化；增加离线 smoke check |
| P1 | 导入 | 大文件导入 Phase 2 | 多 GB 导入在刷新、重启、断网后可恢复，UI 能显示速度和 ETA | 上传会话 SQLite 持久化、启动 janitor、暂停/继续、速度/ETA、残留 staging 清理 |
| P1 | 事件系统 | SSE 事件流 | 减少任务、更新、导入、存储健康、通知中心的轮询碎片 | 新增 `GET /api/events`；先推送 task/update/storage 事件；前端封装 `use-backend-events` |
| P1 | 资料库卫生 | 重复影片 / 文件冲突治理 | 用户能发现重复入库、同番号多版本、路径冲突、损坏资产，而不是靠手动排查 | 新增“Library Health”页或 Settings 卡片：重复番号、重复文件名、缺封面/预览、失联路径 |
| P1 | 元数据 | 元数据修复队列 | 把刮削失败、缺头像、缺封面、provider 降级变成可批量重试的工作流 | 后端聚合 missing/failed 状态；前端展示修复队列；支持批量重试和按错误类型过滤 |
| P2 | 播放器 | Electron 原生播放器桥 | 对浏览器不易解码的格式，用 mpv/PotPlayer 原生播放，同时保留进度同步 | preload 暴露窄 `launchNativePlayer`；设置页配置可执行路径；播放 descriptor 决策 native/hls/direct |
| P2 | 播放器 | HLS / 转码诊断面板 | 用户能知道为什么直放失败、当前走 remux 还是 transcode、硬件编码是否生效 | Player stats 增加 session drilldown，复用 `/api/playback/sessions/{id}` |
| P2 | 发现 | 智能集合 / Saved Views | 把常用筛选变成可复用入口：未看、五星、某演员、某标签、最近导入、4K 等 | 保存查询条件到 localStorage/SQLite；侧边栏或首页展示自定义集合 |
| P2 | 推荐 | 推荐反馈与解释 | 推荐不是黑盒；用户可以“少推荐这个演员/厂商/标签”“稍后再说” | recommendation state 增加 dislike/skip reason；首页卡片显示简短解释与反馈按钮 |
| P2 | 演员库 | 演员别名 / 合并 / 纠错 | 解决同一演员多写法、刮削名不一致、手动资料修正问题 | 新增 actor aliases 表；Actor Profile 提供合并入口；movie actor filter 走 canonical name |
| P2 | 策展帧 | 帧集合 / Storyboard | 把单张截图升级为可整理、可分享的视觉素材集 | 新增 collection/board 模型；帧可加入集合；支持按集合导出 ZIP 或 HTML contact sheet |
| P2 | 统计 | 观看洞察页 | 利用已存在 watch-time、history、rating 形成个人库分析 | 新增 Insights：按演员/厂商/标签观看时长、完成率、近期中断、偏好变化 |
| P3 | 多端 | 移动端 / 平板布局专项 | LAN 访问时手机/平板体验更可用 | 针对首页、库瀑布流、详情、播放器做 mobile smoke + 响应式修复 |
| P3 | 运维 | 备份 / 迁移 / 恢复 | 用户敢长期使用：数据库、配置、缓存、封面可备份和迁移 | Settings Maintenance 增加导出备份包、恢复前预检、版本兼容说明 |
| P3 | 工程 | 需求台账和文档事实同步 | 降低“文档说未实现但代码已实现”的认知成本 | 更新 feature inventory；把筛选后的需求写入 `docs/prd/requirements.csv`；补状态字段与来源 |
| P3 | 工程 | 大文件继续拆分 | 降低后续接入播放器/导入/设置功能的回归概率 | 分批拆 `PlayerPage.vue`、`SettingsPage.vue`、`server.go`、`app.go`，每批保持可测试边界 |

## 我建议优先看的 5 个

1. **LAN 访问控制 + Connected Clients**
   这是安全与产品可信度的基础。现在 Curated 已经是本机 HTTP 服务，并且 Electron 常驻后 LAN 入口会更自然，先补“谁能访问、谁正在访问”很划算。

2. **大文件导入 Phase 2**
   导入是媒体库的入口。现有 chunk endpoint 已经铺好路，继续做持久化、janitor、暂停/恢复、速度/ETA，能显著提升真实大库体验。

3. **SSE 事件流**
   当前任务、通知、导入、更新、存储健康都在各自轮询或本地处理。SSE 可以作为统一事件底座，先替代任务进度和通知来源，后面再扩展播放 session。

4. **资料库卫生 / 元数据修复队列**
   用户使用越久，重复、缺图、刮削失败、失联路径越多。把这些问题变成一个可操作的“修复台”，比继续加单点按钮更像成熟产品。

5. **智能集合 / 推荐反馈**
   现有标签、演员、评分、收藏、历史、每日推荐已经有足够数据。做 Saved Views 和推荐反馈，能把 Curated 从“库管理器”推进到“个人化浏览入口”。

## 可作为快速切片的低风险项

- 更新 `docs/features/2026-05-03-feature-inventory.md`，把 Electron shell、通知中心、存储健康、导入现状同步到当前事实。
- 托盘菜单增加“打开日志目录”和“复制 Web 地址”。
- 导入 UI 增加速度 / ETA，不先做 SQLite 会话持久化。
- Settings Overview 加 Connected Clients 的本机 in-memory MVP，不先做 token 鉴权。
- 首页推荐卡片增加“为什么推荐”文案，先用现有算法信号解释，不改推荐算法。

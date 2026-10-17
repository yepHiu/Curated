# AI 字幕识别、翻译与播放实施计划

日期：2026-10-16
状态：in-progress
关联需求：REQ-0062；用户于 2026-10-16 授权开始实施

## 实施记录（2026-10-16，源码集成完成；完整实机验收待补）

用户已授权实施。本节记录最终实现；下文设计目标在冲突处以本节和源码为准。REQ-0062 登记 implemented / 90，尚未标记 verified。

| 阶段 | 当前结果 | 证据与限制 |
|---|---|---|
| P0 | 供应商短样和 Windows 引擎通过 | 千问3.1真实约25秒音频；Flash/Plus各两句ID严格对齐；Windows mpv真实加载双语、样式/延时/关闭命令通过。macOS字幕实机未测。 |
| P1/P2 | 后端生成链路已集成 | 迁移0066、密钥脱敏、本机管理、幂等队列、10分钟FLAC分片、临时上传、异步识别、严格翻译、检查点恢复、保存三件套及清理。合成媒体真实FFmpeg提取+fake provider端到端通过。 |
| P3 | 设置与详情弹窗已集成 | 单一模型选择、视频/CD与音轨选择、明确开始、真实阶段数量、取消/重试、关闭继续、成品下载/定位、中英日；Mock不伪造云端成功。390px弹窗与1440px设置截图已检查。 |
| P4 | Web字幕源码已集成 | 有界SRT解析、绝对时间、重叠cue、正延时、纯文本、共享样式；用例覆盖非零时间/换片/迟到响应隔离。真实整片直连/HLS视觉矩阵待补。 |
| P5 | 原生字幕源码已集成 | main按当前Session下载受管资产、私有临时SRT、mpv调度、退出清理、受控命令、截图video-only。Windows真实引擎通过，macOS视觉待补。 |
| P6 | 生命周期和文档已集成 | 整理复制受管字幕、永久删除保护用户修改、离线路径迁移同事务同步源路径、文档更新。跨平台恢复、整片质量及磁盘满/NAS等硬件故障验收待补。 |

实际存储与恢复：

- SQLite `subtitle_jobs` 一张元数据表；不永久保存对白。供应商taskId、识别cue和翻译进度集中在 `<cacheDir>/subtitle-jobs/<id>/checkpoint.json`，持久写入后才继续；音频同工作区。成功清理，失败/取消约24小时清理。
- requestId幂等、同文件单一活动任务，最多排队8个文件、串行worker。提交意图在付费提交前持久化，未知提交响应不盲目重发；已保存taskId继续轮询。
- 成品三件套独占创建/fsync，同名加任务后缀；旧成品保留。译文已齐的保存重试不要求密钥、不调用模型；失败翻译保留识别及已译句子。
- 固定10分钟分片，无重叠切片。翻译优先合并全部待译句子，容量不足才拆分，使用 `subtitle-translation-v3` 独立TXT；模型/endpoint/提示词版本冻结。原文与译文共用时间边界。字幕翻译token进入现有AI用量报表subtitle渠道；ASR金额没有虚构估算。
- 百炼临时上传继续使用，每个云端文件48小时；「验证密钥」只读上传授权，不上传、不调用ASR，界面明确说明测试范围。
- 任务由App生命周期管理，弹窗每2秒轮询元数据；**当前未投影通用任务中心或SSE**。不把阶段数量换算为虚假的整体百分比。
- 播放器样式按设备存储；模式/偏移按Server origin、fileId和资产版本隔离。Web与原生均在打开时读取成品，**同一播放会话重新生成后需重新打开播放器加载新版**；自动热更新留待补充。
- 整理时复制已登记/hash匹配的成品并保留原件；不是递归移动整个sub。离线路径迁移在更新movie_files的同一事务中同步字幕JSON sourcePath；不复制媒体，也不重新计算资产哈希，备份恢复必须包括媒体sub。

验证记录（日志为忽略的本地工作区资料，不含密钥/对白）：

| 检查 | 结果 |
|---|---|
| `go test ./...` | 全部通过；`.workspace/subtitle-go-all.log` |
| `go vet ./...` | 通过；`.workspace/subtitle-go-vet.log` |
| 字幕/Server/config/prompts定向测试 | 通过；`.workspace/subtitle-backend-final.log` |
| 前端全量 `VITE_USE_WEB_API=false pnpm test --maxWorkers=4` | 318文件、1747测试全部通过；`.workspace/subtitle-front-all-mock.log` |
| storage/maintenance/subtitles生命周期定向测试 | 通过；`.workspace/subtitle-lifecycle-final.log` |
| 新增Web时间轴/版本隔离测试 | 3项通过；`.workspace/subtitle-playback-web-final.log` |
| Electron全量 | 151通过、5跳过；`.workspace/subtitle-electron-final.log` |
| Windows真实mpv集成 | 2通过（headless命令验证，不等同视觉验收）；`.workspace/subtitle-native-final.log` |
| `pnpm test:e2e` | 9通过；`.workspace/subtitle-e2e.log` |
| typecheck / lint / Web build / Electron main build | 通过；`.workspace/subtitle-*-final.log`及`subtitle-web-build.log` |
| 真实千问短样 | 上传→异步识别→签名结果下载，Flash/Plus非空译文及句子ID对齐通过；`.workspace/subtitle-live-provider.log` |

测试环境说明：根目录开发 `.env` 开启Web API时，全量单元测试中原有设置测试没有模拟本机权限探测，出现25项设置断言失败，以及2项并行播放器超时。未改开发配置；按Mock单元环境全量重跑全部通过，两个播放器文件单独重跑也通过。不能把默认Web环境的失败记录抹去。新增测试随后定向复核通过。未运行需单独授权的display suite。

真实短样验证修复了两个实际兼容问题：签名OSS结果GET不可带JSON Content-Type，否则签名403；翻译提示词须精确约束输出translation字段，避免Flash返回text。供应商调用已付少量短样费用，没有进行整片付费生成；未更改运行Server的密钥、业务库或服务配置。

已完成的本地提交：c106ef2b（计划/原型）；b5dfc569（非思考参数）；9d270370（后端生成）；cbcc9fa7（共享SRT）；22c049eb（设置/弹窗）。e911a819（Web/原生播放器）；16c3a4eb（路径迁移）；bc567c70（合成媒体/保存重试/整理测试）。文档独立提交，未推送。

交付边界：源码集成与自动检查完成，未构建发布安装包、未替换当前Server。用户体验需重建/重启Server，原生控制需更新Desktop。剩余实机验收包括macOS字幕视觉、整片Flash/Plus质量与真实用量比较、全量播放/故障矩阵和跨平台媒体恢复；不得宣称这些已经通过。

## 1. 范围、依据与当前状态

本计划把已讨论的产品行为拆成可执行工作。需求背景、价格核对及实验记录见 [ASR 调研与集成讨论](2026-10-15-realtime-translation-asr.md)，界面基准见 [HTML 交互原型](2026-10-16-subtitle-ui-prototype.html)。本计划是后续实施入口；旧调研中「永久保存结构化对白」「成品放 Server 资产目录」「先选服务再选模型」的早期建议不再适用。

用户已指定先设计原型，确认后再实施。现已获得开始实施授权，登记 REQ-0062。原型通过不等于功能验收；按各阶段实际证据更新进度。

### 1.1 本次交付范围

| 项目 | 实施基准 |
|---|---|
| 入口 | 影片详情页更多菜单「AI 字幕识别与翻译」 |
| 任务 | 一个实际视频文件一个任务；明确开始；弹窗关闭后继续；可取消、查看结果和针对失败阶段重试 |
| 识别 | 百炼北京入口，固定 `qwen-audio-3.1-asr-flash-filetrans`，日语 |
| 翻译 | 一个「翻译模型」选择器：用户配置的 AI 模型、`qwen-flash`、`qwen-plus`；目标简体中文 |
| 上传 | 先沿用已验证的百炼临时上传，封装为可替换适配器；OSS 不作为第一版前置条件 |
| 成品 | 日语 SRT、中文 SRT、中文 TXT，统一保存到 `<视频所在目录>/sub/` |
| 长期数据 | Server SQLite 仅保存任务、分段状态、模型/用量与成品元数据，不永久保存对白正文 |
| 中间数据 | 音频与结构化对白检查点放 Server 任务工作区，成功保存成品后清理 |
| 播放器 | Web 直连/HLS、Windows/macOS 原生；中文/日文/双语/关闭、字号、背景、位置、偏移 |
| 文件获取 | 下载；Server 本机有桌面能力时在文件夹中显示 |

首期不包含实时跟随播放头识别、全库批量、自动开始、字幕编辑器、任意外部字幕导入、字幕烧录、词级卡拉 OK、说话人分离、云端字幕备份、系统画中画字幕。正常页面全屏必须支持。外部播放器能否自动发现 `sub` 取决于其自身能力，不承诺统一自动加载。

### 1.2 默认行为

- 默认继承 AI 设置中的翻译模型；新配置默认「用户配置的 AI 模型」，未配置则提示选择千问或完成 AI 配置。千问中建议 Plus 起步，Flash 为经济选项。
- 任务弹窗可为本次改选模型；不会反写默认设置。开始时冻结模型、源文件修订、音轨、语言和提示词版本。
- 本机可创建云端任务；远端首期只读任务、播放与下载。远端发起属于后续明确扩展，不顺带加入权限开关。
- 确认动作只有首次启用云功能、开始生成与明确重新生成等必要决策点；不堆叠确认弹窗。取消处说明已提交请求仍可能计费。
- 同文件活动任务复用；有已完成结果时先展示成品，再提供明确重新生成动作。

## 2. 实现边界与代码依据

| 现有位置 | 已核对事实 | 本次接入方式 |
|---|---|---|
| `backend/internal/tasks/manager.go` | 内存注册表，支持 Restore 和 task.updated | 复用投影与事件，持久化由字幕业务负责 |
| `backend/internal/app/tag_organization.go` | 独立于前端连接、服从 App 生命周期的持久任务 | 参考 worker 生命周期及恢复模式，不复用标签业务表 |
| `backend/internal/storage/movie_files.go` | 稳定 fileId、文件选择、路径重定位 | 以 fileId 关联，接入重定位与源修订检查 |
| `backend/internal/llm/client.go` | 文本 chat completion；请求结构没有任意 provider 参数 | 复用文本客户端；以类型化能力补非思考参数，避免影响现有 Agent |
| `backend/internal/server/movie_clip_handlers.go` | 既有媒体生成 HTTP 模式 | 参考 ID 校验、任务和错误返回；不复用片段生成状态 |
| `src/lib/player-playback-timeline.ts` 与 `PlayerPage.vue` | 已有绝对播放时间和 HLS 原点换算 | cue 使用源文件绝对毫秒，以现有换算驱动 |
| `electron/playback-contract.ts`、`playback-coordinator.ts` | session/revision、认证 Session、窄桥接 | 增加字幕 ID/枚举/偏移的收敛命令 |
| `electron/native-media-proxy.ts` | 单媒体受控代理，使用认证 fetcher | 借鉴 Session 绑定，不扩成任意 URL 代理 |
| `electron/native-mpv-player.ts` | mpv 属性观察、命令边界、截图 | 添加受控字幕加载、轨选择、样式与清理 |

Server 拥有配置、付费调用、FFmpeg、任务、最终文件；Desktop main 拥有认证字幕获取与原生加载。renderer 不接收凭证，不提交任意磁盘路径、供应商 URL 或 mpv 命令。

## 3. 契约与数据设计

### 3.1 配置

在现有 `library-config.cfg` 及 GET/PATCH `/api/settings` 增加独立 `subtitleProvider`，拟包含 `enabled`、固定 provider/region/model、`apiKey`、`translationModel`。`translationModel` 为 `configured-ai | qwen-flash | qwen-plus`。密钥更新采用保留/替换/显式清除语义，读取仅返回 `apiKeyConfigured`，不回传原文。

识别和千问翻译共用百炼 Key；`configured-ai` 复用已有 AI 配置与文本客户端，不读取 Agent 会话历史或启用工具。共享现有可适用的限流、审计与用量机制；不要求为了字幕开启 Agent 的修改标签权限，不绕过既有模型调用限制。

任务冻结非敏感配置和提示词版本，凭证仅通过 Server 配置引用读取。Key 更新可恢复鉴权；模型或提示词变化须显式创建新的翻译修订，不能混进原任务。显式选择的模型未就绪应明确失败，不偷偷切模型。

### 3.2 表与检查点

拟增加以下元数据表，最终迁移编号在实施时取得：

- `subtitle_jobs`：jobId、requestId、movieId/fileId、sourceRevision、音轨索引、配置快照（无密钥）、state/stage、计数、错误、时间、检查点有效期、用量。
- `subtitle_job_parts`：jobId/partIndex、源起止时间、音频指纹、提交意图、providerTaskId、ASR/翻译状态、尝试次数、用量、临时文件相对引用；不保存永久对白。
- `subtitle_assets`：assetId、fileId、jobId/生成版本、语言/格式、相对视频目录路径、字节数、SHA-256、发布状态与来源类型。

不新增永久 `subtitle_cues` 表。工作区 JSON 检查点保存 cueId、绝对 beginMs/endMs、原文、译文和逐句状态，使用临时文件 + 原子替换，Schema 带版本。启动恢复核对文件与数据库；任何已声明可复用但检查点缺失的结果都必须降级为可理解状态，不假装可继续。

任务唯一性采用 requestId 幂等约束，以及同 fileId 活动任务约束，创建与取消持久化；不能仅靠进程 mutex 防重复。身份核对包含源路径对应文件、大小、mtime、时长和音轨，避免读取整片做哈希。源文件变化后任务阻塞，旧字幕标可能过期。

### 3.3 状态机

状态：`queued → running → completed | partial_failed | failed | blocked | cancelled`。

阶段：`preparing → extracting → uploading → recognizing → translating → saving → done`。重启恢复先进入对应检查点阶段；保存失败为 `blocked/saving` 并保留成品，不应显示已成功。提供独立的 `checkpointExpired`、`assetMissing` 信息，不把旧 completed 任务记录当作当前文件仍然存在的证据。

进入取消状态先落盘，再终止本机进程和后续请求；提交/发布处再次核对，防止迟到响应发布。重试先校验检查点、配置兼容性和源修订，返回具体可执行范围：未完成 ASR、剩余翻译、仅保存。

### 3.4 拟议 API

| 方法与路径 | 输入/输出 | 权限 |
|---|---|---|
| GET/PATCH `/api/settings` | subtitleProvider、能力摘要 | 写入沿用本机管理策略；远端只取必要可用性 |
| POST `/api/subtitles/provider/test` | 识别鉴权/文本连接测试，区分验证范围 | 本机；真实短音频验证有明确小额用量提示 |
| POST `/api/library/movies/{movieId}/subtitle-jobs` | fileId、requestId、翻译模型、音轨选择、显式重新生成；202 + job | 本机创建；校验影片/文件归属 |
| GET `/api/library/movies/{movieId}/subtitle-jobs` | fileId + 分页任务摘要 | 既有认证 |
| GET `/api/subtitle-jobs/{jobId}` | state/stage、真实计数、错误、成品、可执行重试范围 | 既有认证 |
| POST `/api/subtitle-jobs/{jobId}/cancel`、`/retry` | 幂等动作；Server 决定实际恢复范围 | 本机写操作 |
| GET `/api/library/movies/{movieId}/subtitles?fileId=...` | 当前可用资产、版本、内容类型 | 既有认证 |
| GET `/api/subtitles/{assetId}/content` | SRT/TXT；可下载；ETag/修订 | 既有认证，ID 解析路径，限制体积 |
| POST `/api/subtitles/{assetId}/reveal` | 无任意路径参数；在文件夹中选中成品 | 直接本机且有桌面能力 |

通过任务事件只传有界摘要；正文按需读取，不在 SSE/snapshot 传整片对白。锁定、切服务器后不继续读取旧来源。错误码区分未配置、无音轨、源变化、提交结果未知、余额/鉴权/限流、审核、翻译部分失败、检查点过期、目录不可写、成品缺失。支持客户端重连补读，事件不作为唯一数据来源。

## 4. 执行链路和资源控制

1. 创建任务时解析实际 fileId、探测音轨与源修订；单音轨默认选择，多音轨显示可选轨。与当前播放音轨无法对应时使用明确的默认轨并允许改选。
2. Server 用既有 FFmpeg 路径抽取 16 kHz 单声道压缩音频；工作区随机 ID 命名。限制全局一个字幕任务，分段最多两个并发起步，单独限制文本并发；避免播放时多路抽取抢占资源。
3. 沿用约 3–5 分钟有界切片，优先临近静音；记录真实起止和编码偏移。首版不为了追求无缝拼接盲加重叠；若增加重叠，按绝对时间去重并记录重复用量。
4. 使用百炼临时上传后及时提交固定模型，上传与调用同账号同模型。临时 URL 48 小时、100 QPS 不可扩容及官方生产警示保留在技术文档；不让用户配置 OSS，不把上传链接当成最终存储。
5. 云端 taskId 收到即持久化，重启查询原任务；网络/429/5xx 有界退避。提交已到服务端但响应丢失时，如缺少供应商幂等/查询凭据，标记提交结果未知，不自动再次付费提交。用户显式重试时说明可能重复计费。
6. ASR 回包归一化 cue，验证时间边界、空结果和输出完整性；无对白是合法结果。纯喘息/短应答不能靠简单词表一概删除。
7. 翻译按 token 预算分批，输入 cueId、日语原文、有限相邻上下文及稳定译名表，输出 cueId/译文，不允许模型修改时间码。校验缺项、重复 ID、空译文，逐级缩批，最终失败只标该句。
8. 千问使用非思考模式；现有客户端按明确类型扩展，不给 renderer 任意参数通道。若用户配置模型缺乏可用的结构化输出，使用文本 JSON 校验/重试，不假设所有兼容接口有同样能力。
9. ASR 已完成后重试翻译不再次上传/识别；原文检查点已清理时可解析最终日语 SRT。原文成品也缺失时明确需要重新识别。
10. 保存最终 SRT/TXT、校验、发布后清理中间数据。已有字幕播放不调用模型。请求取消和服务退出终止本地 FFmpeg，并有界等待子进程结束。

阶段进度按真实提取时长、上传字节、完成分段和句子更新。无供应商进度时显示等待/分段计数，不做虚假细粒度百分比，不承诺固定完成时长。ASR 与翻译用量分开，价格参考带日期，不把样本成本当整片账单。

## 5. 文件生命周期与影片联动

### 5.1 路径和命名

```text
<影片目录>/
  DEMO-001.CD1.mp4
  DEMO-001.CD2.mp4
  sub/
    DEMO-001.CD1.ja.srt
    DEMO-001.CD1.zh-CN.srt
    DEMO-001.CD1.zh-CN.txt
    DEMO-001.CD2.ja.srt
    DEMO-001.CD2.zh-CN.srt
    DEMO-001.CD2.zh-CN.txt
```

若同目录存在同 stem 不同扩展的视频或同名用户字幕，使用短 fileId/版本后缀生成不冲突名称，记录实际资产路径；不能覆盖其它文件或依赖猜测覆盖归属。`sub` 已有目录复用；如果是文件、符号链接/重解析点导致逃出预期目录等异常，明确拒绝该保存位置。所有文件名与目录从 Server 受控数据计算。

### 5.2 发布与恢复

- 字幕内容写入目标同卷的临时文件，flush/close、校验编码、时间边界和非空正文（合法无对白除外）、大小与哈希，再原子重命名。NAS/跨卷不能假设 rename 原子，须按实际能力设计失败恢复。
- 多文件发布没有单次文件系统事务。用待发布版本记录和发布日志逐个登记，全部就绪后切换当前资产版本；启动时对账。旧完整版本在新版本失败前保持可用，不能先覆盖旧三件套再尝试回滚。
- 同名旧受管版本替换要在全部新文件准备好后受控执行；新内容可先用版本后缀保存以避免覆盖窗口。只删除数据库确认归属且未被外部修改的旧受管文件，不清空 `sub`。
- 写入失败保留待保存文件，提供仅重试保存与下载。弹窗显示剩余保留期。文件管理器按资产 ID 定位实际成品。

### 5.3 清理、扫描、移动与备份

- 工作目录建议位于 Server `cacheDir/subtitle-jobs/<jobId>/`，但与普通缓存清理隔离：运行任务持有租约/锁，不能被清理动作删掉；Server 重启根据持久任务恢复租约。
- 成功发布并校验后立即清理音频、JSON 和临时译文。失败/取消/待保存任务建议默认保留 24 小时，记录 expiresAt；过期后任务记录继续存在，但明确恢复能力已经失效。清理范围限制在本工作目录并检查归属。
- 媒体扫描、监听、整理预览和导入应排除生成的字幕目录内容，避免将 TXT 当元数据输入。不能仅前端隐藏。
- 影片移动/改名只跟随处理该 fileId 的受管字幕，不整体移动共享 `sub`。规划目标路径、同名冲突、失败恢复，与 `RelocateMovieFile` 及离线路径迁移规则保持一致；纯重扫发现移动时对账关联。
- 第一版影片删除沿用原有删除语义，受管字幕明确纳入预览/结果；只移除库记录不能顺带删除磁盘成品。用户删除成品或外部修改后标记缺失/修改，不能从任务记录伪造恢复。
- 应用数据库备份只含任务和成品关联。SRT/TXT 必须随影片目录备份；恢复数据库不能保证字幕存在。测试数据库+媒体共同恢复和仅数据库恢复两条路径，不新增把媒体目录全量打包的行为。

## 6. 前端实施

### 6.1 设置与详情任务

- 在 `src/api/types.ts`、`src/api/endpoints.ts`、`services/contracts/library-service.ts` 补 DTO/方法；Web/Mock 都有明确实现。Mock 不上传、无真实定位，提供可验证状态，不把模拟结果伪装成落盘成功。
- 在 `SettingsAISection.vue` 接字幕配置，保持现有设置卡片范式；固定识别模型只读展示。翻译单一下拉框，正式界面尽量展示当前配置的具体模型名，千问项展示 Flash/Plus 及定位。
- 新建业务弹窗与 composable（如 `MovieSubtitleDialog.vue`、`use-subtitle-job.ts`），从 `DetailPanel` → `DetailPage` → `DetailView` 沿现有事件装配；愿望单无本地文件场景不误开放生成。
- 首次点击不发付费请求；开始按钮防连点并携带 requestId。关闭停止 UI 订阅但不取消 Server 任务；重新打开从服务读取，断线重连补读。
- 完成列出三个成品和播放、下载、本机定位；部分翻译失败准确统计；保存失败有仅重试保存；未配置可跳 AI 设置；远端明确只读与下载。
- 页面文本加入中/英/日 locale，复用 Dialog、Progress、Select、Toast 等基元；错误就地显示，完成/失败消息不重复，播放中遵守已有静默政策。
- 不带入原型的评审导航、状态选择器、模拟数据、假连接结果、两句示例下载或“原型提示”。

### 6.2 Web 播放

- 新建有界 SRT 解析/配对与播放器字幕 composable；使用纯文本，不用 `v-html`，处理 BOM、CRLF、多行、非法时间和重复/重叠 cue。
- 通过当前 fileId 查询成品，SRT 解析成内存 cue；共享进度、样式和配对语义，查找避免每帧线性扫整片。
- 使用现有 `player-playback-timeline` 的源文件绝对时间，跨 HLS 会话重建/非零起点/seek 使用相同原点。原文与译文按稳定时间边界配对，外部修改后无法配对则提示并回退单轨。
- 字幕层不拦截点击，避让控制栏和安全区；页面全屏随播放器容器显示。模式/字号/背景/位置不重建播放会话；有新成品时刷新轨列表。
- 样式偏好按设备保存；轨选择与偏移按 Server 身份+fileId+字幕版本保存。正值使字幕更晚出现；双语同时偏移；换片不能沿用错误偏移。

### 6.3 Desktop 原生播放

- 在 `playback-contract`/preload/coordinator 增加受控字幕列表、选择和样式命令；校验 sessionId、连接 generation、revision、资产 ID、枚举和偏移范围。
- main 用当前认证 Session 从 Server 下载选定字幕到应用私有临时目录，限制体积与格式；renderer 不拿凭证/路径，mpv 不拿 Server Cookie。
- mpv 加载中文、日语及按同一 cue 派生的双语轨，负责时间调度；显示样式映射同一套配置，先验证 SRT/ASS、字体和边距在 Win/Mac 都可用。
- 新字幕加载成功再替换当前轨；换片、切 Server、锁定、窗口结束时取消下载并清理临时轨，迟到请求不进入新 session。生成完成更新轨但不中断视频。
- 截图继续视频原画面不含字幕，验证 mpv 截图命令；Web 与原生切换保留字幕模式/偏移，不新增播放记录写入者。
- 若当前 libmpv 渲染路径不能正确支持字幕，先提供证据与局部替代验证，不直接以低频 renderer 时间快照驱动字幕冒充同步实现。

## 7. 分阶段工作清单与通过条件

各阶段只在前置条件满足后进入下一阶段。按最小行为单元提交，文档与业务尽量分开；不 push、不改发布版本、不自动发布。

| 阶段 | 具体任务与主要落点 | 交付物和通过条件 |
|---|---|---|
| P0：契约与可行性验证 | 复核 3.1 参数/上传/轮询/时间戳；Flash/Plus 非思考参数；Win/Mac mpv 加载与截图小样 | 短样可生成对应字幕并在引擎显示；记录实际限制、价格和响应；上传先用临时接口，无需 OSS |
| P1：配置、任务和文件基础 | config/contracts、storage 新迁移、`internal/subtitles` 状态/检查点/资产发布；App worker 生命周期 | fake provider 下创建/幂等/取消/重启恢复/保存失败/旧版本保留通过；任务不依赖浏览器连接 |
| P2：识别与翻译完整链路 | 百炼适配器、FFmpeg 切片、llm 模型参数、独立 prompts TXT、计费统计、清理器 | 本机短样三件套写入 sub；翻译失败不重识别；无永久对白表；音频与 JSON 能正确清理 |
| P3：前端配置和任务交互 | API/LibraryService Web+Mock、AI 设置、详情事件与弹窗、消息策略、三语 | 所有任务状态可通过真实 API 到达；刷新恢复；远端只读；下载/定位能力正确 |
| P4：Web 字幕 | SRT parser、字幕层、共享设置、HLS 时间轴、设备偏好 | 直连/HLS 非零起播、seek、暂停、倍速、全屏、换片和偏移通过，无播放中断 |
| P5：原生字幕 | Electron 收敛 IPC、认证获取/缓存、mpv 轨道与样式、Win/Mac 生命周期 | Windows 与 macOS 原生都可原文/中文/双语；锁定与切 Server 清理；截图不含字幕 |
| P6：媒体联动与最终验收 | scan/organize/path migration/delete 联动、恢复对账、整片质量、文档与打包验证 | 数据生命周期矩阵、真实整片和故障验收通过，实施事实与手册一致 |

P1/P2 内的建议提交单元：配置与契约；任务仓储/迁移；受管文件发布；worker 恢复；百炼调用；翻译与输出；清理与用量。P3、P4、P5 各自按独立行为提交，测试与对应实现同提。迁移/模型调用/播放器不得以半成功状态宣称第一版交付。

## 8. 验证与验收矩阵

| 场景 | 必须验证的结果 |
|---|---|
| 并发/重复开始 | 同 requestId 复用，同 fileId 只有一个活动任务 |
| 提取/上传/识别/翻译中重启 | 各检查点恢复；已获得 taskId 不重复提交；未知提交状态不盲重试 |
| 翻译超时/限流/余额不足 | 明确原因、有限重试，只处理未完成范围，不串成假成功 |
| 无对白/审核/部分未译 | 无对白合法空结果；审核与空结果区分；部分失败准确计数 |
| 关闭/取消/断网 | 关闭不取消；取消持久化且迟到结果不发布；重连补读 |
| 只读目录/磁盘满/断开 NAS | 已生成结果保留，重试仅保存，超时后准确说明临时结果过期 |
| 多文件/同 stem/同名用户字幕 | 三件套与 fileId 正确对应，不串 CD、不覆盖用户文件 |
| 保存中崩溃/重新生成失败 | 启动对账；旧完整版本仍可用，无半套新字幕被标为当前 |
| 移动/改名/删除/路径迁移 | 只操作对应受管文件，共享 sub 中其它影片字幕不受影响 |
| 清理/备份/恢复 | 活动任务不误删；成功清理中间文件；数据库单独恢复显示缺失；媒体+数据库恢复可播放 |
| Web 直连/HLS | 非零起点、seek、暂停、倍速、全屏与正负偏移对齐 |
| Windows/macOS 原生 | 三种语言模式、样式、字幕刷新、跨引擎偏好及视频原画面截图 |
| 锁定/切 Server/远端 | 不读取旧连接字幕、不暴露密钥和临时 URL；无任意路径/URL IPC |
| 质量与成本 | 同一段清晰日语/短应答/人名对白对比 Flash/Plus；记录漏译、ID 对齐、人名、耗时、真实 token，不凭模型定位宣布质量胜出 |

自动测试优先故障恢复、付费幂等、文件隔离、原点与版本竞态。供应商使用 fake HTTP/异步任务 fixture，不让单测真实付费；人工短样和整片真实验收单独记录。当平台实机不可用，明确未验证，不用跨编译代替原生显示验收。

执行规范以 `docs/ops/2026-04-08-agent-build-and-test.md` 为准：

```text
仓库根目录：
  pnpm typecheck
  pnpm lint
  pnpm test <相关文件>
  pnpm test:electron
  pnpm test:e2e
  pnpm build
  pnpm build:electron:main

backend/ 目录：
  go test ./internal/subtitles/... ./internal/storage/... ./internal/app/... ./internal/server/... ./internal/llm/... ./internal/backup/...
  go test ./...
  go vet ./...
```

先运行相关检查，再在最终集成运行全量门禁；通过且无新修改不重复扩测。Go 缓存保持默认，不放仓库内。Windows 开发编译使用既有 `pnpm backend:build:dev`；发布标签/跨平台包测试走已有流程，不在本计划阶段发布。UI 用浏览器截图验证桌面/窄屏、键盘、Dialog 焦点、暗色可读性；未经明确同意不运行 `pnpm test:display`。

## 9. 文档与交付完成条件

实现时同步：

- `docs/prd/requirements.csv`：实施开始登记 requirement；implemented/verified 以实际证据更新，不提前写 100%。
- `.cursor/rules/project-facts.mdc` 与必要的 `workspace-quick-reference.mdc`：配置、API、任务及当前播放器支持范围。
- 根 README 短入口、`docs/guide.md`：配置、生成、字幕设置、sub 文件随片备份、远端限制和失败恢复。
- `docs/reference/2026-03-21-library-organize.md`：新增 cfg 键、sub 排除和移动/改名行为。
- `CLAUDE.md` API 清单、`docs/reference/architecture-and-implementation.html` 实现说明及对照表。
- `docs/reference/2026-03-24-frontend-ui-spec.md`：稳定的字幕层、设置、位置避让规则；相关 message-catalog 与 locale。
- 本计划：逐阶段记录完成项、提交、测试及剩余项；原型保持评审参考，不改成生产入口。

第一版完成须同时满足「配置 → 发起 → 可恢复生成 → sub 三件套 → Web/Win/Mac 显示 → 下载/定位 → 失败与清理」闭环。真实验证不能完成的环节列为未完成，不以静态原型代替。

## 10. 开工顺序

本计划评审后，从 P0 开始收敛供应商与 mpv 契约；通过后先完成 P1/P2 后端闭环，再接 P3/P4/P5，最后 P6。第一版保留当前已验证的临时上传方案，不另起 OSS 配置工程。若 P0 发现真实不可用的供应商/引擎约束，给出复现和具体调整范围，再修订计划，而不是静默换模型或缩减用户已要求的功能。

## 2026-10-16 UI 反馈修正

用户反馈新增字幕设置/弹窗不符合UI规范，详情分片胶囊被长文件名撑大。已按现有AI设置卡片统一图标、Server归属、短标题、gap-3/嵌套p-4块、说明Tooltip及胶囊操作；把字幕卡片移入同一gap-6布局，修复与上一张卡片紧贴的问题。字幕弹窗操作右对齐、按内容宽度、手机44px；下载/定位为圆形图标动作；字段标签关联实际控件ID。

详情仅移除wrap-file-name，复用MoviePartSelect既有桌面sm:w-64、单行省略和title全文，保留窄屏44px、下拉全名和选择行为。不修改全局Select默认尺寸。

侧边栏问题核对结果：当前字幕独立worker运行，关闭弹窗后继续；尚未投影通用任务中心/侧栏，不会显示任务进度。再次打开详情字幕弹窗会恢复真实进度。本轮仅修正UI和说明现状，不宣称已接入侧栏。

验证：typecheck/lint通过；详情、字幕弹窗、AI设置、三语资源相关64项测试通过；浏览器1440px设置和390px弹窗/设置检查无横向溢出，弹窗主要操作44px。未运行display scaling suite；未调用付费模型。截图output/playwright/subtitle-*-ui-fixed*.png为忽略的本地复核资料。

### 字幕设置逐项排列与密钥显示（2026-10-16）

启用云识别、固定识别模型、百炼密钥、翻译模型改为四个独立纵向设置行，取消桌面双列。密钥默认password，眼睛切换显示/隐藏；已保存密钥仅在点击显示时通过新增本机限定POST /api/subtitles/provider/key按需读取，响应no-store，普通GET仍脱敏。隐藏时释放读取的密钥，保存后重置显示状态；远端不可调用。沿用现有认证/本机管理边界，不在Mock保存密钥。

### 按播放设置范式修正行布局（2026-10-16）

用户明确“一行行”以播放设置为参考。字幕设置改为桌面左侧名称、右侧当前值/开关/输入/下拉的横向行，采用播放设置相同的border-border/50、bg-muted/5、p-4与gap-3；移除外围重复嵌套表面。API Key与翻译模型控件右对齐，窄屏上下排列，保留眼睛显示与隐藏。typecheck与定向eslint通过；1440px/390px浏览器检查无横向溢出。此轮不修改工作区内已存在的播放及其他设置页面改动。

### 移除清除密钥开关（2026-10-16）

按用户反馈移除“清除已保存的密钥”设置行及前端状态。保留四个主要设置项；留空保存不提交apiKey，保留已存密钥，输入新密钥则替换。眼睛显示/隐藏行为保持。

### 可选字幕识别模型（2026-10-16）

字幕识别模型改为右侧下拉：千问3.1文件转写 `qwen-audio-3.1-asr-flash-filetrans`（默认）和千问3文件转写 `qwen3-asr-flash-filetrans`。`subtitleProvider.recognitionModel`随设置保存；旧配置缺失默认3.1。任务创建时冻结recognitionModel，重试/恢复继续使用原模型，旧任务缺失仍按3.1处理。上传授权使用对应模型；3.1提交file_urls/language_hints并解析results，3提交file_url/language并解析result。测试密钥按当前已保存的识别模型检查上传授权。无新增端点/数据表，识别模型进入任务JSON元数据。

配置校验、两代协议分支（含千问3单结果）、字幕/Server测试通过；38项相关前端测试、typecheck、定向ESLint通过。浏览器确认两个选项及切换；未进行本轮付费云调用，两代模型既有真实demo记录见2026-10-15调研。

### 关闭时收起配置并保留原值（2026-10-16）

云端字幕识别关闭时隐藏识别模型、密钥、翻译模型及密钥验证按钮，保留开关与保存操作。仅条件渲染控件，不重置配置或密钥草稿；重新打开恢复原选项。保存关闭状态不提交空apiKey，因此保留Server已有密钥。专项交互测试覆盖隐藏/恢复、草稿保留和关闭保存不清密钥，通过；typecheck和组件ESLint通过。

### 缩短翻译模型选择器与括号提示（2026-10-16）

按用户反馈，设置页字幕翻译模型选择器桌面宽度收紧到240px（同时限定max-width，避免Field默认子项全宽规则覆盖），窄屏仍自适应。选项改为“千问 Flash（更经济）”“千问 Plus（更均衡）”，中英日使用独立完整文案，替换型号与提示之间的圆点。共享组件同步影响任务弹窗选项。浏览器实测宽度240px并验证选择Flash；12项相关测试与定向ESLint通过。

### 保留千问官方模型名称（2026-10-16）

根据用户反馈，识别模型直接显示qwen-audio-3.1-asr-flash-filetrans、qwen3-asr-flash-filetrans，不翻译模型名。翻译模型使用qwen-flash、qwen-plus，保留括号内经济/均衡提示。三语同步，模型请求标识不变。

### 字幕设置自动保存与测试入口（2026-10-16）

移除保存按钮，复用useAISettingsAutosave串行保存：开关/模型立即保存，密钥700ms防抖，失焦与离开页面flush，失败保留草稿并提供重试；空密钥不提交，仍保留Server密钥。输入控件不因后台保存锁定，避免打字中断。测试按钮改为“测试”，允许测试新输入的密钥，先等待保存成功再执行测试，失败不继续。结果明确验证的是当前识别模型的音频上传权限，未冒充完整识别/翻译或余额测试。13项相关测试、typecheck与组件ESLint通过，包括关闭保留和测试前保存顺序。

### 字幕弹窗历史与全局进度入口（2026-10-16）

设计框架：字幕弹窗以选择文件、查看进度和生成字幕为主；成品复用 TagOrganizationDialog 的紧凑可展开历史行，按生成时间显示，展开后下载或定位。视频文件、音轨和翻译模型使用相同胶囊触发器与触控高度。删除底部关闭按钮，重新生成使用简短文案。侧栏复用 SidebarTagOrganizationEntry 的位置、密度与折叠形态，AppShell 持有独立观察器和恢复弹窗，不依赖详情组件生命周期。新增只读 GET /api/subtitle-jobs 读取持久任务快照，刷新后仍可恢复；观察失败保留最后快照并显示断线。覆盖加载、空历史、运行中、完成、失败、成品缺失及窄屏状态。


2026-10-16：字幕弹窗成品复用 AI 整理的可展开历史列表；选择器统一胶囊样式，底部取消关闭按钮，重新生成文案简化。AppShell 观察 GET /api/subtitle-jobs 的持久任务快照；侧栏与播放/AI整理相同区域显示进度，点击按 movieId/fileId 恢复弹窗，关闭弹窗不停止观察。断线保留最后快照；无新增配置或迁移。

验证完成：pnpm build（含 typecheck，体积监管零提醒/超限）、定向 ESLint、5 文件 48 项 Vitest、backend 下 go test ./internal/server -run Subtitle -count=1 通过。Playwright 在真实页面使用模拟任务响应检查 1440px/390px、下拉尺寸一致、历史展开、Escape关闭后侧栏恢复及折叠侧栏恢复；窄屏无横向溢出。当前运行的旧开发 Server 不含新增端点，需重启新版 Server；未中断它，也未触发付费模型。未运行完整 display-scaling 套件。截图保存在忽略的 output/playwright/subtitle-history-sidebar-*.png。


## 2026-10-17 千问 3 结果链接兼容修复

真实任务在 recognizing 阶段报 recognition result unavailable。只读查询原云端任务确认其已 SUCCEEDED，但 result.transcription_url 使用 HTTP，被本地仅 HTTPS 的供应商地址校验拒绝。将同一签名链接协议升级为 HTTPS 后，实际下载返回 200，包含 67 条句子，确认无需重新识别该片段。

修复仅在转写结果下载入口升级协议，随后继续校验既有供应商域名、无用户信息及无自定义端口限制；原始签名路径和查询不重编码，不允许明文下载。回归测试覆盖千问 3 HTTP 结果、签名保留、无凭据下载和非受信地址拒绝。go test ./internal/subtitles 与 go vet ./internal/subtitles 通过。修复提交 f590ba92。

本轮不重启正在执行其它字幕任务的开发后端，不自动重试失败任务；重启后修复生效。没有新建云端识别请求，排查仅查询既有任务和下载既有结果。当前用户另发起的千问 3.1 任务继续运行。


## 2026-10-17 翻译优先整批完成

用户要求尽可能一次完成翻译。取消按识别音频分片、每30条顺序翻译的实现；识别仍按10分钟切音频，全部识别后按播放时间合并所有未译cue，已译cue沿用检查点。输入与估算序列化共用TranslationInput，只提交id/text。

发送前按文本容量贪心合并：configured-ai沿用AI设置contextWindow；缺省及千问专用入口保守使用65536上下文预算。输入以序列化UTF-8字节数估算，包含提示词、1024余量与窗口5%余量；输出估算为每条原文Unicode字符数+16，预算上限min(32768, contextWindow/2)。这些是分组估算，不是供应商实测token；沿用最新移除max_tokens的实现，请求不声明固定输出上限。单条超过估算预算仍单独交给供应商裁决，不截断或丢弃对白。

全部可容纳则一次请求；超出估算预算才分成尽可能大的连续组。若供应商明确返回上下文超限或finish_reason=length，当前组二分并顺序处理，最小到一条；所有有效子组立即落检查点。鉴权、限流、审核、普通网络错误与ID/JSON校验失败不自动拆分，保留原错误。供应商以stop返回却漏译仍会失败，不把残缺结果当作完成。

翻译HTTP超时由3分钟延长至15分钟，继续响应任务取消。所有ID仍须唯一且完整对应，取消后拒绝发布；已完成译文的保存重试不需要云配置。提示词、检查点版本、API、数据库和library-config字段均未改变；已有旧提示词任务仍受既有版本冻结限制。

验证：go test ./internal/subtitles ./internal/app ./internal/llm通过；go vet同三包通过；开发后端构建通过。测试包括跨两识别片段的180条待译只发一次且保留原译文、500条短句无需分组、长文本按容量拆分、截断/上下文错误缩小请求、成功前半检查点恢复只重试后半、审核/鉴权/限流/超时不额外请求、应用层180条请求不含max_tokens，以及取消不发布成品。首次整包运行app未通过，随后包含新增应用测试的整包运行通过；未修改无关用例。使用本地fixture，未发起本轮付费云端识别或翻译，整片真实模型质量与耗时仍待验证。


实现提交73fde3c1。确认任务中心running/queued均为0后，平滑停止旧开发后端40572并启动新开发二进制backend/runtime/curated-dev-subtitle.exe，PID43504；127.0.0.1:8080健康检查成功，任务中心历史仍可读取。新策略已在开发后端生效，没有自动重试历史失败任务或新增付费调用。


## 2026-10-17 整批翻译漏句与调用状态修复

17:11的JUQ-219任务（460f0681）报translation omitted cues，482条识别句子全部已就绪、ID唯一，检查点已译0条。对应17:10:14的deepseek-v4-flash调用耗时82.897秒，输入9529/输出23554/总33083 Token，旧逻辑记录completed，但后续句子完整性校验失败。未留存供应商原始正文，不能据此断言实际漏了几句或模型为何漏句。17:02同模型的JUR-075已完成609/609，证明新版整批流程可用。

排查时8080运行PID29484，由go run ./cmd/curated于16:59启动，使用runtime/curated.db；已实际使用v3提示词和整批翻译。因此最新失败不是未重启造成。根因是整批返回缺少部分cue，而服务原先将整组译文作废；上层调用观察又只涵盖JSON解析，早于完整性校验，导致用量记录看似成功。

改动：共用ValidateTranslations在应用层记录结束前检查ID/非空/完整性，漏句产生含应有/已收到数量的IncompleteTranslationError并返回有效子集。用量行记录failed/incomplete_translation；未知/重复ID或空译文为failed/invalid_translation，Token按实测保留。服务层再次独立校验，只接受有效子集，先写检查点并更新真实进度，再对遗漏ID最多补译两轮；已有译文不重复提交。两轮后仍不齐则保留成果报错，空结果立即失败，审核/鉴权/限流/格式/错误ID不触发补译。上下文超限和length二分逻辑继续保留，取消仍拒绝发布。整批成功时维持一次调用。

本次不修改提示词版本、供应商配置、API或数据库；旧检查点继续兼容，历史调用状态不回写。JUQ-219原失败响应未保存，无法恢复其当时返回的部分译文；手动重试仍复用16段识别结果，首次重新翻译482条，之后才应用新的遗漏补译逻辑。未自动重试或新建付费调用。

验证：go test ./internal/subtitles ./internal/llm通过；go test ./internal/app -run 'Subtitle|AIObservations' -count=1通过；go vet ./internal/subtitles ./internal/app ./internal/llm通过；pnpm backend:build:dev通过。新增回归验证漏译结果的保存顺序、只补遗漏ID、最多两轮及检查点恢复、空/错ID结果不重试，以及用量表对漏译/重复/未知/空译文的失败状态与Token保留。实现提交ab7b9488。

确认任务中心running/queued为0后平滑停止旧go run后端，启动标准backend/runtime/curated-dev.exe（PID6788），健康检查、8080监听及任务历史读取通过，仍使用runtime/curated.db；修复已在开发后端生效。

### 失败字幕任务支持整次重试（2026-10-17）

按用户要求，任务中心和影片字幕入口共用的TaskDetailContent保留“重试未完成部分”，为failed/blocked/cancelled增加“重新执行整个任务”。确认弹窗说明从头识别、翻译和再次计费。新操作使用当前字幕默认识别/翻译模型与最新提示词，保留原文件和音轨；创建新任务并切换详情/深链，旧记录、检查点和成品不修改，过期检查点也可重新生成。原检查点续试及提示词版本保护不变。

新增本机限定POST /api/subtitle-jobs/{jobId}/restart，返回新SubtitleJob；复用现有Create校验/队列限制和同文件活动任务去重。运行中/已完成任务不允许整次重试。无新配置或数据库迁移。

验证：前端相关3文件17项测试、typecheck和定向ESLint通过；后端字幕/Server整包测试及go vet通过，开发后端构建通过。新增后端用例覆盖检查点存在/过期、当前配置和提示词、旧历史成品保留、重复点击、非法状态及真实FFmpeg合成音频从头执行识别翻译（供应商使用本地fixture）。未自动重试用户的付费任务。

确认任务中心running/queued均为0后平滑停止旧开发后端6788，替换并启动标准backend/runtime/curated-dev.exe（PID11732）。8080健康检查和任务历史读取通过，使用原runtime/curated.db；新接口已在开发后端生效。

### 整次重试保持任务身份与需处理状态修正（2026-10-17）

用户指出初版整次重试生成不同任务ID，并使旧失败记录留在需处理中。修正为POST restart原记录同ID重新排队：保留requestId/createdAt/文件/音轨及已有成品，清空阶段计数、错误、供应商错误和过期时间，冻结当前字幕模型与提示词。增加内部attemptId切换检查点目录，不先破坏旧检查点，续试/重启恢复/清理均使用当前attempt；同ID多次发布使用独立文件后缀保护原文件。重复点击已活动任务直接返回；同文件另有活动任务或队列满则拒绝。worker运行前重读持久记录，避免使用旧快照。

前端去除切换新ID/路由的分支，操作成功刷新原任务及中心统计。初版已产生的接替任务不改号、不重跑；Server启动识别restart-请求，仅在源版本/文件/音轨一致且先前失败记录唯一时标记旧记录supersededBy。旧诊断与时间戳不改，仍在历史可读，并有接替任务链接；统计、筛选和Mock投影均排除需处理，旧记录不再提供续试/整次重试。歧义不猜测。沿用JSON元数据，无配置或表迁移。

新attempt生成的成品ID包含本轮版本标识，旧资产ID继续兼容；保持任务ID并不复用字幕版本，避免播放器时间偏移跨版本串用。原文件变化时不把旧资产登记到新版源文件。

验证：前端3文件18项测试、typecheck及定向ESLint通过；pnpm build通过，体积监管0项提醒/超限；Go字幕、storage、Server整包测试和vet通过，开发二进制已构建。覆盖同ID重排队/需处理汇总与筛选、配置/检查点过期、重复点击、同文件冲突、取消/恢复、新attempt检查点及成品读取、旧接替关系的幂等修复和歧义保护。仅本地合成音频与provider fixture，不额外发起付费任务。

实现提交25fe6f01。切换前JUR-067翻译请求仍在执行，先保留原后端等待；随后用户明确选择立即重启并接受当前请求可能重复计费。切换前已将开发SQLite在线快照保存到忽略目录.workspace/subtitle-restart-backend/pre-reconcile.db。

启动核对进一步修正：启动worker前将持久running记录恢复为queued，保留ID、attemptId、阶段/计数、时间和检查点，worker真正领取后才标记running，避免重启后显示多个实际未执行的running任务。新增幂等恢复与进度保留测试；字幕整包测试、go vet及开发构建通过。实现提交f61f396d。

开发后端最终运行标准backend/runtime/curated-dev.exe，PID31908，8080健康检查通过。实际任务中心total=11/running=1/queued=1/attention=0/history=9，attention筛选为空；旧JUR-067与STAR-684失败记录分别关联81eb047719854a118fdd0eb2b1ca529d、7d4aacf47305fe88509782d0ded3f50c。JUR-067从739/1165条译文继续翻译，STAR-684排队并保留4/14段识别检查点；没有声称整片已完成。

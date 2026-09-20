# Curated Agent 持续任务体验：研究与实现

## 目标与范围

用户希望参考 MiniMax Code 等开源 Agent 重塑 Curated 的使用体验，首要问题是达到上下文上限就结束任务。本次直接实现上下文管理基础：保留完整聊天记录，维护独立的工作上下文，在可恢复的容量不足时整理并接续。任务后台执行、断线自动恢复、持久任务计划等列为下一阶段，不能把本次实现描述为完整通用 Agent 框架或无限上下文。

## 源码研究（2026-09-21）

MiniMax Code 阅读基线：`73a2581c6c7525628342f33b53907d4f7bdc146e`。git 浅克隆传输中断后，通过 GitHub tree API 与固定 revision 的 raw 源码完成定向阅读；未运行其安装脚本，也未引入其运行时或依赖。

| 源码 | 观察到的机制 | Curated 的适配 |
| --- | --- | --- |
| [compact-context.ts](https://github.com/MiniMax-AI/minimax-code/blob/73a2581c6c7525628342f33b53907d4f7bdc146e/packages/local-runtime-v2/src/service/turn-system/compaction/algorithm/compact-context.ts) | 工具内容裁剪/归档、检查点生成、压缩后重新检查是否能放入请求 | 单轮上下文主动整理、替换后重新检查字节预算；保留近期完整工具批次 |
| [checkpoint-prompt.ts](https://github.com/MiniMax-AI/minimax-code/blob/73a2581c6c7525628342f33b53907d4f7bdc146e/packages/local-runtime-v2/src/service/turn-system/compaction/execution/checkpoint-prompt.ts) | 目标、约束、已完成工作、当前状态、阻塞、待办；历史内容视为不可信数据 | 独立无工具摘要请求；保留用户目标与未完成工作，不提升历史内容权限 |
| [history-reduction.ts](https://github.com/MiniMax-AI/minimax-code/blob/73a2581c6c7525628342f33b53907d4f7bdc146e/packages/local-runtime-v2/src/service/turn-system/compaction/algorithm/history-reduction.ts) | 工具调用和结果一致性检查，不接受孤立结果/未结束批次 | 只在整批工具执行后整理；保留的调用与结果整批复制，旧批次作为序列化数据摘要 |
| [provider-budget.ts](https://github.com/MiniMax-AI/minimax-code/blob/73a2581c6c7525628342f33b53907d4f7bdc146e/packages/agent-modules/context-manager/src/provider-budget.ts) | 模型窗口、输出预留、安全余量与主动触发门槛 | 按配置的总窗口扣除输出预留及安全余量，在输入预算的 80% 触发；JSON 字节估算不伪装成真实 token 测量 |
| [OpenCode session/compaction.ts](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/session/compaction.ts) | 历史摘要、近期内容保留、工具结果清理、overflow 恢复 | 交叉验证“摘要 + 近期窗口 + 有界恢复”的分层方向；独立 Go 实现 |

以上是机制参考，不是代码移植；没有将任一项目的复杂度、工具权限或多 Agent 调度直接搬入媒体资料库。

## 本次已实现

### 1. 持久会话记忆

- SQLite migration `0052_ai_context_checkpoints.sql` 新增按会话保存的 `through_seq` 与 `summary`，删除会话时级联删除。
- 模型近期窗口随 contextWindow 缩放，候选上限 200 条，默认保留最多 24 条及约 22.4 KiB 正文；窗口之前的记录从旧检查点向前分批读取并总结，包含最早目标，而不是只处理最近候选。
- 摘要每次只在模型生成成功并保存成功后推进位置；旧请求不能覆盖更新位置。摘要最多 6 KiB，输入数据最多 24 KiB；超大历史单条使用明确标注省略的 UTF-8 摘录。
- 每次摘要最多 45 秒，一次历史准备总计最多 60 秒；很长的旧会话可在后续请求继续补齐。失败不改变原始消息或检查点位置，界面显示记忆不完整。
- 持久摘要不存入用户可见回答，也不返回给界面。它是历史线索，不是事实、权限或写入回执。精确记录仍需通过当前轮工具核实。
- 同一会话串行处理聊天写入与检查点，等待期间可以取消；不同会话独立。此锁是当前后端进程级，不支持多进程同时写同一资料库。

### 2. 单轮执行中的上下文整理

- 请求门槛随配置的模型窗口变化：输入预算扣除输出预留与 5% 安全余量后，在 80% 时先整理，超过输入预算则拒绝不可压缩输入。计数使用序列化 JSON 的 UTF-8 字节估算，不是模型真实 token；具体公式见下文模型容量配置。
- 系统规则、修正限制和最新用户输入原样保留。最近一个较小工具批次（低于 12 KiB）完整保留；更早批次或超大结果转成摘要数据，不拼接残缺 JSON，不重复执行工具。
- 原请求的服务端事实引用存储与写入确认规则继续有效；摘要不能创建新引用。摘要生成失败时使用有界原文摘录降级，提示部分上下文缺失。
- 对明确的供应商上下文超限 HTTP 错误允许一次压缩后重试；认证、限流、普通参数错误不会被当成上下文不足反复请求。
- 压缩后仍放不下（例如最新输入本身过大），返回 `context_input_too_large`，提示缩短当前请求并继续原会话。不会要求创建新会话。
- 提示词升为 `agent-system-v7`，说明检查点不可信、应接续任务、缺失精确信息需重新查询。

### 3. 可见、可恢复的过程反馈

- `POST /api/ai/chat` 新增 `context_status`，`context.phase` 为 `compacting | ready | limited`。全程共用一个 `message_start` 与单调事件序号。
- 复用 `AgentChatProcess` 的次级文字反馈，显示“正在整理对话记忆，随后自动继续”“已整理，原始记录仍可查看”或记忆不完整提示。中英日三语；`role=status` / `aria-live=polite`。
- 状态随 `events_json` 保存；重新打开历史不会重启 spinner。结束、取消和失败都清除活动状态。前端按当前流身份过滤回调，旧请求不能覆盖新会话。
- UI 范围是现有 Agent 过程反馈，不改壳层、布局、设计 token 或主要操作。继续以发送/停止为主要行动；Mock 保留模拟聊天，不声称生成持久模型记忆。

## 验证与限制

针对性测试覆盖：超大工具结果后继续完成；完整工具批次配对；系统规则与最新 Unicode 输入保留；摘要失败降级；取消不提交；供应商 overflow 只恢复一次；101 条历史回填最早目标；检查点复用、单调保存与会话删除；同会话等待取消；SSE 状态转发；历史恢复不留转圈。

最终检查结果见本文件末尾的验收记录。未调用用户实际配置的付费模型；确定性 streamer 与 SQLite 回归验证控制流和持久化，不能替代具体模型摘要质量评估。

实际限制：摘要有损，不能保证所有早期条件永远保留；工具长结果的原始全文没有新增归档库，必要时重新查询。预算已接入模型窗口配置，但字节估算尚未依据 tokenizer 或真实 usage 校准。关闭面板、断网或进程退出仍会停止当前运行，历史与已保存摘要可续聊，但不能自动从任意工具步骤恢复。

## 后续演进顺序（尚未实现）

1. **模型容量与成本**：总上下文窗口配置已实现；后续补充独立 output limit，结合实际 prompt usage 校准估算，展示实际调用与整理成本，不提供伪精确百分比。
2. **持久任务计划**：把目标、约束、已完成步骤、待办与阻塞保存为结构化任务状态；计划展示与执行结果绑定，避免模型只输出待办而无真实进展。
3. **可恢复运行**：将 HTTP 连接与任务生命周期解耦，保存 run/checkpoint/event cursor；重连可追读。任何写工具恢复必须先查 durable receipt，禁止盲目重放。
4. **工具结果归档和检索**：为长结果生成有界归档引用，提供只读分页回读；与事实引用权限分离，精确信息可按需取回。
5. **质量评估**：用长对话、目标变更、取消/重连、写入确认、不同语言与模型的合成场景持续验证；评估摘要漂移和重复执行率，再考虑委派式并行任务。

这些阶段不应通过默认提高工具步数、简单放大窗口或追加“请继续”提示来替代。

## 本次验收记录

- `backend/`：`go test ./...`、`go vet ./...` 通过；随后增加摘要异常字符预算防护、缺失历史说明、真实本地 HTTP/SSE 集成回归后，相关 app/run/prompts 测试与 app/run vet 再次通过。
- 根目录：`pnpm lint` 通过；`pnpm test` 为 275 个文件、1324 项测试通过；Agent/transport/i18n 针对性回归 9 个文件、71 项通过。
- `VITE_USE_WEB_API=true pnpm build`（PowerShell 先设置环境变量）通过，包含类型检查与未放宽的 bundle hard budget。
- `pnpm test:e2e` 初次为 1 通过、4 失败：Mock navigation / Personal Insights 的 page.goto 冷启动超时；locked startup / maintenance backup 在 `unknownApiRequests` 断言发现测试桩遗漏 `GET /api/ai/settings`。这两个场景未打开 Agent 面板；现有 AppShell/useAIGovernanceSync 发起该请求，本次没有改动启动同步或测试桩。
- 用 `pnpm exec playwright test --config playwright.e2e.config.ts --workers=1 --grep 'Mock navigation|personal insights'` 复跑两个超时场景，2 项全部通过。上述两个测试桩断言仍为未通过项，未以放宽断言掩盖。未运行付费真实模型、发布打包或生产实例升级。
- 本地 HTTP/SSE 集成测试确认：摘要会传入后续模型请求，SSE 仅一个 message_start、seq 连续，摘要不泄露到可见 transcript 或事件中。


## 模型容量配置（2026-09-21）

用户追加需求：在「设置 → AI → 配置模型」明确设置所导入模型的上下文容量，并提供主流模型参考值。已新增 `aiProvider.contextWindow`，沿用现有 `GET/PATCH /api/settings` 与 library-config 持久化；不新增端点或迁移。数字输入和预设遵循现有自动保存，550ms 停顿、失焦及离开页面时保存合法草稿，错误值阻止保存/连通测试并提示。Web 下一次聊天快照生效，进行中请求保持原预算；Mock 保存到 localStorage。

- 总窗口单位 tokens；默认 **65,536**，接受 **32,768–2,097,152** 整数。下界是当前工具/摘要预算的产品支持范围，不表示模型行业下限。旧配置缺失或 0 使用默认；PATCH 显式 0 拒绝，省略保持原值。
- 预设只填入容量，不修改模型 ID、地址或密钥；手动编辑容量会转为自定义。预设名称只是容量参考，供应商的实际部署/中转额度可能更小，应按实际值覆盖。Claude 等预设不增加原生协议支持，仍使用现有 OpenAI 兼容接口。
- 令 W 为总窗口，输出 O=min(32768, floor(W/4))，安全余量 S=floor(W/20)，输入 I=W−O−S，主动整理阈值 T=floor(I×4/5)。聊天与摘要请求传 `max_tokens=O`。例如 W=204800，O=32768、S=10240、I=161792、T=129433。
- 模型输入以序列化 UTF-8 JSON 字节数计入预算，每字节保守占一个单位；尚未用 tokenizer/usage 校准，不是精确 token 占用率。默认窗口 W=65536 的 I=45876、T=36700，预留输出后比此前固定 64 KiB 更保守。
- 历史正文预算 floor(I/2)，条数 min(200, floor(24×W/65536))；服务端最多读取 200 条候选，其余由增量摘要衔接。摘要输入 min(24576, I−2048) 字节，输出摘要最多 6 KiB。HTTP 消息长度等已有独立防护继续有效。
- 本次预算仅适用于聊天和会话摘要；独立翻译/润色 Action 与连通探针保持原策略。仍不提供无限上下文保证、运行断线自动恢复或每个 provider 的独立输出上限设置。

### 预设来源

于 2026-09-21 读取以下官方页面；文档名义 1M/200K 使用保守十进制，官方给出精确整数时采用原值。预设代码保留官方来源链接。

| 模型 | tokens | 官方来源 |
| --- | ---: | --- |
| DeepSeek Flash | 1,000,000 | [Models & Pricing](https://api-docs.deepseek.com/quick_start/pricing) |
| DeepSeek V4 Pro | 1,000,000 | [Models & Pricing](https://api-docs.deepseek.com/quick_start/pricing) |
| MiniMax M3 | 1,000,000 | [Models](https://platform.minimax.io/docs/guides/models-intro) |
| MiniMax M2.7 / M2.5 | 204,800 | [Text generation](https://platform.minimax.io/docs/guides/text-generation) |
| GLM-5 | 200,000 | [GLM-5](https://docs.z.ai/guides/llm/glm-5) |
| GPT-5.4 | 1,050,000 | [GPT-5.4](https://developers.openai.com/api/docs/models/gpt-5.4) |
| Claude Sonnet 5 / Opus 5 | 1,000,000 | [Models overview](https://platform.claude.com/docs/en/about-claude/models/overview) |

### 容量配置验证

- Go 回归覆盖配置解析/范围校验、旧配置默认、保存重载、无关部分更新保留容量、非法 PATCH 不改变配置、不同容量的请求准入及历史保留；本地 HTTP/SSE 集成验证配置传入历史准备和模型请求的 max_tokens。
- 前端覆盖已存值回填、预设不覆盖模型/密钥、手工覆盖、非法/小数/越界拒绝、自动保存与连通测试顺序、三语键完整性；4 文件 85 项针对性测试与类型检查通过。
- 最终全量：`pnpm lint`、`pnpm test`（276 文件 / 1332 测试）、`go test ./...`、`go vet ./...`、`VITE_USE_WEB_API=true pnpm build`（含类型检查及原 bundle budget）全部通过。
- Playwright CLI 浏览器检查（独立 Mock）：桌面 1280×720 / 窄屏 390×844 显示通过；选择 MiniMax M2.7 自动保存 204800，刷新仍保留；小数 65536.5 标为无效，刷新恢复之前的合法值。桌面截图保存在本地 `.workspace/context-config-desktop.png`。未调用用户真实模型或更改实际模型配置。
- 本轮未重跑完整 runtime e2e；此前两个 `/api/ai/settings` 测试桩遗漏仍见上节记录，不能将本次测试描述为整个仓库所有 e2e 已通过。

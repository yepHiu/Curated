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
| [provider-budget.ts](https://github.com/MiniMax-AI/minimax-code/blob/73a2581c6c7525628342f33b53907d4f7bdc146e/packages/agent-modules/context-manager/src/provider-budget.ts) | 模型窗口、输出预留、安全余量与主动触发门槛 | 先在现有 JSON 字节预算的 75% 触发，保留硬边界；不伪装成真实 token 测量 |
| [OpenCode session/compaction.ts](https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/session/compaction.ts) | 历史摘要、近期内容保留、工具结果清理、overflow 恢复 | 交叉验证“摘要 + 近期窗口 + 有界恢复”的分层方向；独立 Go 实现 |

以上是机制参考，不是代码移植；没有将任一项目的复杂度、工具权限或多 Agent 调度直接搬入媒体资料库。

## 本次已实现

### 1. 持久会话记忆

- SQLite migration `0052_ai_context_checkpoints.sql` 新增按会话保存的 `through_seq` 与 `summary`，删除会话时级联删除。
- 模型继续使用最多 24 条、约 24 KiB 的近期窗口，但窗口之前的记录从旧检查点向前分批读取并总结，包含最早目标，而不是只处理最近 80 条。
- 摘要每次只在模型生成成功并保存成功后推进位置；旧请求不能覆盖更新位置。摘要最多 6 KiB，输入数据最多 24 KiB；超大历史单条使用明确标注省略的 UTF-8 摘录。
- 每次摘要最多 45 秒，一次历史准备总计最多 60 秒；很长的旧会话可在后续请求继续补齐。失败不改变原始消息或检查点位置，界面显示记忆不完整。
- 持久摘要不存入用户可见回答，也不返回给界面。它是历史线索，不是事实、权限或写入回执。精确记录仍需通过当前轮工具核实。
- 同一会话串行处理聊天写入与检查点，等待期间可以取消；不同会话独立。此锁是当前后端进程级，不支持多进程同时写同一资料库。

### 2. 单轮执行中的上下文整理

- 请求估算超过 48 KiB 时先整理，64 KiB 仍是硬边界。数字是现有序列化 JSON 的 UTF-8 字节估算，不是模型真实 token，也不是供应商最大窗口。
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

实际限制：摘要有损，不能保证所有早期条件永远保留；工具长结果的原始全文没有新增归档库，必要时重新查询。字节估算尚未按模型配置窗口或依据真实 usage 校准。关闭面板、断网或进程退出仍会停止当前运行，历史与已保存摘要可续聊，但不能自动从任意工具步骤恢复。

## 后续演进顺序（尚未实现）

1. **模型容量与成本**：为兼容 provider 明确配置 context/output limits；结合实际 prompt usage 校准估算，展示实际调用与整理成本，不提供伪精确百分比。
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

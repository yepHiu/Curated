# Curated 正式版 Agent 提前结束排查

## 范围与结论

2026-09-09 只读检查正式版日志、SQLite 会话/事件/工具审计/运行统计/写入回执，并对照仓库实现。未调用模型、未改正式库或配置、未重启服务。以下时间均为北京时间。

- 正式版本：1.5.5；后端 buildStamp：20260906.154156。
- 模型：deepseek-v4-flash；治理配置 stepLimit=15，enabled=true，readOnly=false。
- 会话：ses_03910e2a18aa2e363b84eef638cea063。
- 核心问题：00:58:45 发起的“查找相关作品并保存书签”请求耗尽工具预算，未进入展示和书签预览；结束提示未准确报告步数耗尽。01:00:59 重试则正常生成预览，并在 01:02:33 确认保存。

## 事件证据

| 请求起止时间 | 运行统计 | 实际结果 |
| --- | --- | --- |
| 00:58:45–00:59:16 | 31.496 秒；模型调用 5 次；工具调用 15 次；partial；error_code 为空 | get_library_overview 1 次、search_movies 14 次。没有执行 present_movies 或 create_saved_view。正文末尾出现 DSML 工具标记和 present_movies 调用文本，未成为结构化工具事件。 |
| 01:00:59–01:01:31 | 32.159 秒；模型调用 2 次；工具调用 8 次；completed；error_code 为空 | 实际执行 present_movies 和 create_saved_view；工具结果为 previewed，发出 confirm_required；message_done 原因是 A write preview is waiting for UI confirmation. |
| 01:02:33 | ai_apply_receipts 存在当前会话 create_saved_view 成功回执 | library_saved_views 新增“内衣·ランジェリー”，过滤标签为 ランジェリー。 |

两轮均持久化了 message_done；所有已执行检索工具审计均成功。证据支持应用主动收尾，不支持将本次归因为 90 秒无数据超时、用户取消或进程崩溃。HTTP 200 本身不能证明任务完成，结论来自结构化事件与回执。

## 实现原因

1. `backend/internal/agent/run/loop.go:74` 在 steps 达到 stepLimit 后追加收尾提示并设置 tool_choice=none。步数按每个工具调用累计，不按模型调用轮数累计，因此一批多个检索会快速耗尽预算。
2. `loop.go:114` 先判断没有结构化 ToolCalls 的回复并直接 return；步数上限的专门 partial 分支位于 `loop.go:130`。模型遵循 tool_choice=none 时通常会先进入通用结束分支。本次因检索曾返回 truncated，被归为泛化的“Some requested evidence could not be fully retrieved.”，掩盖了真实的预算耗尽。若没有 truncated/failure，现有代码还可能标记 completed。
3. 前一轮正文包含 `<｜｜DSML｜｜tool_calls>` 等内容，但审计没有对应工具执行。客户端只从标准 delta.tool_calls 累积可执行调用，不把正文当指令。原始上游响应未保存，因此无法进一步断言这些标记由模型还是兼容服务转换产生。
4. `loop.go:215` 将“写入预览等待确认”标记为 completed，并立即返回。这代表当前生成流结束，不代表资料已经写入。最终写入需要 UI 确认；本次已有成功回执。
5. `src/components/agent-window/AgentWindow.vue:398` 的确认流程会更新卡片状态、刷新书签，但不会自动续聊或补发模型总结，因此确认后不会再出现一段模型完成答复。
6. `backend/internal/app/agent_runtime.go:302` 的模型历史只投影 user/assistant 正文，旧工具证据不随历史回传，且当前轮实体引用会重置。重试中模型将旧 ID 称为“臆造”并不能证明旧 ID 是假的；上一轮检索摘要已有相同 ID，属于历史证据缺失下的不可靠自我解释。

## 排查时的修复建议（实施进度见下文）

- 优先修复结束状态：记录工具预算耗尽事实，使正常收尾也返回明确的 partial 和稳定原因代码；展示“已用完 15 次工具调用，展示/书签步骤尚未完成”。以可控 streamer 测试覆盖“达到上限后返回纯文本”的路径。
- 区分流结束、待确认、业务写入完成；书签预览显示待确认，成功回执才显示保存完成；确认成功后可追加确定性的完成提示，不必再次调用模型。
- 为检索、展示和写预览合理分配调用预算；避免重复探测消耗全部额度。提高上限只能缓解，不能修复结束状态与未完成任务的识别。
- 识别正文中的工具协议泄漏并提示异常，保留诊断分类；不得直接执行正文里的 DSML。补充未完成工具意图的回归场景。
- 后续完善跨轮证据摘要，明确旧轮已查到的事实与本轮重新验证要求，避免模型把缺失证据误说成先前捏造。

## 后续讨论：默认不限制工具步数

用户提出是否需要步数限制。调整建议为：取消默认 15 次工具调用的硬停止，将步数上限作为可选高级设置，默认不限制。前文“合理分配调用预算”仅适用于用户主动设置上限的情况，不再作为默认运行方式。保留准确的结束状态修复。

理由：工具调用次数不代表任务完成度或真实成本。本次 15 次工具调用仅耗时约 31 秒，模型调用为 5 次；反复进行跨语言搜索虽有优化空间，但达到第 15 次不能证明后续无法完成任务。工具步数可用于统计与诊断，不应默认替用户决定任务结束。对于“主流 Agent 都没有限制”的判断，本次未做产品逐项核验；不将没有显式步数设置等同于不存在上下文、运行时间、额度或内部执行边界。

建议运行语义：

- 默认持续执行到任务完成、需要用户补充/确认、用户停止，或出现明确的执行故障；流结束、业务完成、待确认分别展示。
- 始终提供停止按钮及可读进度，允许用户在高级设置中主动启用工具次数上限。
- 保留单次工具期限、网络无数据超时、写入权限/确认/幂等和上下文容量保护；这些与累计步数上限独立。
- 检测反复相同参数、相同结果且没有任务进展的循环，先提示模型改变方法；持续无进展时清楚暂停。分页、合法轮询、结果随时间变化不能仅按相似调用误判。第一阶段先覆盖确定性的重复与连续失败，不声称能准确判断所有语义进展。
- 可展示耗时与提供方报告的 usage；提供方缺少 usage 时标为未知。以后若增加用量预算，应由用户明确配置，不以任意新步数或固定时长暗中替代旧限制。
- 上下文达到当前容量边界仍需明确部分完成与恢复入口；默认不限制步数不承诺无限上下文。后续再做证据摘要与上下文压缩，保留任务目标、未完成项与确认权限边界。

实现注意：当前 config 校验限定 1–30，前端也限定 1–30；Gateway 与 Loop 均执行限制，core.Settings 将非正数回退到 15。因此直接设为 0 或仅修改 UI 都不会实现无限步数。应明确禁用上限的配置语义，贯通配置、服务契约、前端和两处执行判断，并测试旧配置兼容。已有用户显式设置应保留；旧值 15 无法单凭数值判断是默认还是用户选择，不应无说明地批量改为不限制。

本节记录当时的讨论建议；用户随后授权实施步数开关和状态修复，实际范围见下节。

## 2026-09-09 实施结果

用户要求：默认不限步数、用户可选上限；修复结束状态与待确认状态。已完成代码修改，未打安装包、未替换正式进程、未改正式配置。

### 最终行为

- 配置与 Web/Mock 默认值统一为 `stepLimit: 0`；设置页增加“限制每轮工具调用次数”开关，关闭时隐藏次数输入，开启时可设 1–30。开关立即自动保存，数字仍按既有延时/失焦保存；加载已有正数不写回，保留用户选择。
- Gateway 和 Loop 都将 0 视为不限。旧配置 15 仍保留为已开启上限；升级后用户可关掉开关取消限制。
- 用户选择上限后，达到上限在下一次模型调用前直接发出 `partial`，附 `reasonCode: tool_step_limit`，保留已返回的工具证据。移除强制模型以 `tool_choice=none` 收尾的额外请求，避免未执行完的工具批次进入下一次模型请求。
- 写入预览终态为 `needs_confirmation/confirmation_required`。实时界面按确认卡片状态显示待确认、已保存或已放弃；成功保存有确定性反馈，不额外调用模型。
- 确认失败不会显示成功；保留重试和放弃按钮。修复原来出现错误文案时按钮一起消失的问题。
- 历史接口以成功回执投影 `completed/write_applied`；无成功回执的历史预览投影 `needs_input/confirmation_unavailable`，要求重新生成预览，不恢复写权限。兼容旧 `completed` + 等待确认 reason 的历史。
- AI 运行列表支持 `needs_confirmation` 筛选。运行统计记录生成终止时状态，聊天状态和成功回执反映确认结果，不修改原始统计事件。
- 新增状态与提示有中、英、日三语。设置页复用现有 Switch/Field/Input 和自动保存；聊天状态复用现有终态反馈，使用语义状态色，无全局布局或设计令牌变更。

### 验证

- `pnpm test -- src/components/jav-library/settings/SettingsAISection.test.ts`：21 项通过。
- 五个相关前端测试文件（AgentWindow、session races、confirmation outcome、restore history、AI settings）：63 项通过，覆盖待确认、确认失败重试、已保存、旧历史兼容、开关与自动保存。
- `pnpm typecheck`、`pnpm lint`：通过。
- `VITE_USE_WEB_API=true` 下 `pnpm build`：通过，含 bundle budget 校验。
- 后端 config/core/run 首批测试通过；本次完整 Agent/app/server 相关测试在隔离副本通过，包含超过旧 15 次继续执行、上限批次截断、确认回执跨重启恢复。`go vet ./internal/agent/... ./internal/app ./internal/server ./internal/config` 通过。
- 隔离原因：主工作区另一任务同时修改影片展示/答案协议，造成两个旧 present_movies 测试断言失败。隔离副本基于 `ec933dab`，仅叠加本次状态修复；未覆盖或提交另一任务的未完成代码。
- 没有调用真实付费模型，没有操作正式资料库。未运行需要另行授权的 `pnpm test:display`，未声称完成多浏览器/显示缩放视觉验收。

### 提交与边界

- `5677a26c`：`feat(agent): make tool-step limits opt-in`
- `9826e005`：`fix(agent): distinguish execution limits and confirmation outcomes`
- 同步 project-facts、README 三语言入口、guide、library-config 说明、CLAUDE API 列表与架构功能对照。
- 本次不增加循环检测、上下文压缩或正文 DSML 解析；网络/工具超时、上下文容量、取消和写入确认继续独立生效。未推送远程。

## 数据来源

- `%LOCALAPPDATA%/Curated/logs/curated-20260909.log`
- `%LOCALAPPDATA%/Curated/data/curated.db`（SQLite URI mode=ro）
- `%LOCALAPPDATA%/Curated/config/library-config.cfg`（仅提取模型名和治理配置）
- `C:/Program Files (x86)/Curated/resources/app/package.json`

未复制密钥、确认 token 或原始提供方请求。未改业务代码；未运行构建、测试或真实模型复现。本报告的代码归因基于当前源码与正式事件吻合；未逆向正式二进制。

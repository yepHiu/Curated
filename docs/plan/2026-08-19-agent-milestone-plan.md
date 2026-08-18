# Agent 执行里程碑计划（E1–E4）

日期：2026-08-19
状态：proposed（执行计划；对应需求 REQ-0029～REQ-0043，均 `idea`）
上游：[`2026-08-18-agent-charter.md`](2026-08-18-agent-charter.md)（宪法，B1–B7 基建不变）· [`2026-08-19-agent-user-prd.md`](2026-08-19-agent-user-prd.md)（用户需求）

## 0. 总原则（2026-08-19 用户决策）

1. **早期不做重 UI**：当前处于想法验证期，交互投入最小化；
2. **实验性门控（feature gate）**：所有 Agent UI 默认隐藏，开关开启后才出现；关闭即全部消失，对普通用户零感知（P-07 的产品化表达）；
3. **早期载体 = 浮动 Agent Window**：Cursor 风格的对话浮窗，覆盖在页面上；正式 Copilot 抽屉留到毕业期（E4）再决策；
4. **基建不缩水**：后端仍按 charter B1–B7 全量推进，gating 只影响前端呈现，不影响网关/审计/权限；
5. **MCP（B4）不受 gating 影响**：外部客户端自带交互，仅依赖 B2+B3，可在 E2 后任意时点独立交付。

## 1. 已确认的 UI 需求（E1 范围）

1. **设置页侧栏新增「实验性功能」类目**：作为通用实验容器（未来其他实验功能也入驻），Agent 是第一个入驻者；组件 `SettingsExperimentalSection.vue` + `src/lib/settings-nav.ts` 新条目。
2. **「启用 Agent 能力」开关**：默认关；持久化为每浏览器状态（localStorage `curated-agent-experimental-v1`）；**gating 是纯前端呈现层**——后端端点始终注册且受 PIN 保护；毕业时迁移为后端全局设置键。
3. **gating 规则**：开关关 = 顶栏无 ✦ 入口、无就地按钮、无窗口；开 = 上述全部出现并保持可用；重启记住状态；关闭开关时进行中的流式请求前端 abort + 后端 ctx 取消。
4. **Agent Window（浮动对话窗口）**：
   - 开启开关后自动弹出一次，之后经顶栏 ✦ 图标（`NotificationCenter` 旁，gated）唤起；
   - 可拖动（标题栏）、可关闭（按钮/Escape）、位置与尺寸记忆（localStorage）、移动端全屏化；
   - 内容：流式消息区 + 输入框（Enter 发送）；E1 单会话，E2 起会话持久化与切换；
   - 视觉与工程约束照旧：shadcn-vue + 设计令牌、dark、三语 i18n、≥44px 触控、**懒加载不进首屏 bundle**；
   - 组件落点：`src/components/agent-window/`（AgentWindow / ChatStream / ChatComposer）+ `src/composables/use-agent-window.ts`、`use-experimental-agent.ts`（gating 状态）。

## 2. E1 实验一期：开关 + Provider + 能对话

**目标**：打通「开关 → 配 provider → 浮窗里连续对话」全链路，验证 provider 与流式基建。

| 交付 | 内容 | 对应 |
|---|---|---|
| 实验性功能设置区 | 类目 + 启用开关 | REQ-0043 |
| Provider 配置（实验区内） | providerKind/baseURL/apiKey/model + 连通测试（交互照搬 proxy ping，出站走 proxyenv）；apiKey 走 `PATCH /api/settings` 入 `library-config.cfg`，永不进 release 样例 | B1、REQ-0039 实验形态 |
| `/api/ai/chat` v0 | SSE 流式（`message_start/text_delta/message_done/error`），**无工具调用**（纯 LLM 对话） | B5-lite |
| Agent Window | §1.4 定义的最小形态 | REQ-0043 |

**验收**：开关关时全应用无任何 Agent UI；开→入口与窗口出现并记忆位置；未配 provider 时窗口内引导去实验区配置，配 Ollama 后可多轮对话；Mock 模式假流式；关闭开关中止进行中流。测试：provider 客户端 fake-server 单测、SSE handler 单测、gating/window 组件测试。

## 3. E2 实验二期：工具网关 + 能查库

**目标**：Agent Window 里的问题开始触达真实库数据（只读）。

| 交付 | 内容 | 对应 |
|---|---|---|
| 工具网关 | `agent/core` 七步管道 + 权限三档 + ConfirmStore + 审计（migration `0039_ai_agent.sql`）+ 全环节单测（P-12，恶意剧本自此期开始） | B2 |
| 查询域工具 | 11 组只读工具 + 投影/脱敏/游标 | B3 |
| 完整 AgentLoop | 多步循环、工具卡渲染、会话持久化与切换 | B5 完整 |
| 首批只读场景 | 统计问答、查找、上下文指代、**会话式选片推荐** | REQ-0032、REQ-0040 |

**验收**："这个月看了多久""XX 演员没看的有几部"答案数字真实可对账；候选推荐来自真实检索且条件可见；工具调用内联卡可见；步数触顶明确汇报；审计表含全部调用；Mock 假工具结果。

## 4. E3 实验三期：写能力 + 就地动作（能干活）

**目标**：Agent 从"能问"到"能做"，写路径唯一入口 = preview→apply。

| 交付 | 内容 | 对应 |
|---|---|---|
| 确认协议 + 写域工具 | 确认卡（字段级 diff → 应用/放弃）；用户数据写域工具上线 | B6 |
| Action 通道 + presets | `/api/ai/actions/{name}`；首批：润色笔记、清洗简介、翻译标题、Insights 解读；**就地按钮全部受 gating**（开关开启才显示） | B7、REQ-0029/0030/0041/0031 |
| 一句话视图 | NL → 筛选确认卡 → 创建 Saved View | REQ-0033 |

**验收**：确认前零写入、token 校验、审计含 preview/apply；就地按钮 diff 预览后走既有端点语义写回；批量 >25 转任务并经 ScanProgressDock 展示。

## 5. E4 毕业期：转正与对外

**目标**：从实验功能毕业为正式功能，补齐治理与自动化需求。

| 交付 | 内容 | 对应 |
|---|---|---|
| MCP 对外 | stdio `-mode mcp` + `/api/mcp`（只读默认）；设置区"复制配置 JSON"。**技术依赖仅 B2+B3，可在 E2 后与 E3 并行，不必等 E4** | B4、REQ-0038 |
| 设置毕业 | 实验区配置迁入正式「AI」分区（隐私/权限/用量/审计视图完整形态）；启用开关策略毕业（保持显式开启，但迁为全局设置键） | REQ-0039 完整形态 |
| 交互毕业（待定） | Agent Window → 正式 Copilot 右侧抽屉，或保留 Window 形态；届时按使用数据决策（charter §7.6 待定项同步收敛） | charter §7 |
| 治理需求 | 标签归一化、演员归并、AI 用户标签与偏好画像 | REQ-0034/0035/0042 |
| 批量与自动化 | 批量维护代理、每周库报告（经通知中心 + 消息目录登记） | REQ-0036/0037 |

## 6. 与旧 M0–M4 / 批次 A–D 的映射

| 旧序列 | 新归属 | 变化说明 |
|---|---|---|
| M0 基建 | E1 | provider 配置落位实验区，而非正式 AI 分区 |
| M1 就地动作 | E3 | **延后**：先验证对话形态的价值，再铺按钮 |
| M2 Copilot 只读 | E1+E2 | 载体由抽屉改为浮动 Agent Window |
| M3 写与治理 | E3（写/确认）+ E4（治理需求） | 拆分：写机制先行，治理需求随毕业期 |
| M4 对外与自动化 | E4 | MCP 可提前至 E2 后并行 |

PRD 批次列（A–D）继续作为价值分组标签，执行顺序以本文 E1–E4 为准。

## 7. 每期 DoD（质量门，不随实验身份降低）

- `pnpm typecheck` / `pnpm lint` / `pnpm test` / `pnpm build`；后端 `go test ./...` / `go vet ./...`；
- charter P-12：网关管道单测、Agent 循环恶意剧本（自 E2 起）、提示词 golden transcript（自 E2 起）；
- gating 验证：开关关闭时构建产物中不渲染任何 Agent UI（组件测试覆盖）；
- 文档同步义务（charter §9.3）：`project-facts.mdc` / `API.md`（新端点）/ 本计划勾选进度；需求状态在 `requirements.csv` 同步推进（E1 启动时 REQ-0043/0039 至少 `specified`）。

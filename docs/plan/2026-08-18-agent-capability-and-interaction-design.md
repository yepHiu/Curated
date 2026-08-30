# Agent 能力、原子工具与交互布局设计（2026-08-18）

> 状态：设计文档（未实施）。上接 `2026-08-17-ai-integration-research.md`（生态调研），本文回答三个问题：
> 1. 接入 Agent 能力后能拿来做什么（能力分层与场景）；
> 2. 要提供什么样的原子能力（工具注册表设计）；
> 3. 怎么把 Agent 布局进现有系统、怎么做交互（三条交互面 + 基建 + 分期）。
> 基建深化（安全工具网关 / 原子能力体系 / Agent 运行时 / 实施里程碑 B1–B7）见 `2026-08-18-agent-infrastructure-design.md`。
> **2026-08-19 更新**：E1–E3 实验期交互载体为「实验性功能开关 + 浮动 Agent Window」，本文所述交互面为毕业期（E4）目标形态；执行顺序见 `2026-08-19-agent-milestone-plan.md`。
> 代码锚点均基于当前实现。

---

## 1. Agent 能力能拿来做什么

### 1.1 能力分层（决定架构复杂度）

按"需要的 Agent 能力"分层，越往下越简单、越先落地：

| 层级 | 能力 | 输入 → 输出 | 是否需要工具调用 |
|---|---|---|---|
| L1 纯文本变换 | 润色/清洗/翻译/扩写 | 文本 → 文本 | 否（单次调用） |
| L2 检索增强 | 查库后回答/生成 | 问题 → 查询工具 → 答案 | 单步或少量 |
| L3 多步规划 | 任务分解 + 连续工具 + 写操作 | 指令 → 计划 → 执行 → 汇报 | 是（Agent 本体） |
| L4 持续任务 | 定时/后台执行 | 触发器 → 报告/清单 | 是 + 任务系统 |

**核心产品判断**：Agent 的价值不在"能聊天"，而在三件事——
1. **翻译**：把自然语言翻译成系统已有的结构化操作（筛选条件、PATCH 字段、Saved View）；
2. **模糊判断**：做规则代码做不好的事（清洗带广告的刮削摘要、判断两个演员名是否同一人、生成解释文案）；
3. **编排**：把 N 次手动点击串成一句指令（批量重刮、批量打标、治理清单）。

### 1.2 场景清单（按层级标注）

**L1（就地按钮，最快见效）**
- 影片笔记润色（`MovieCommentSection` + `library_movie_comments`；不做扩写/翻译）
- 刮削摘要清洗去广告 → 写 `user_summary`（展示覆盖列本就是干这个的）
- 标题/厂牌建议（`MovieEditDialog` 的 AI 预填）

**L2（问答与报告）**
- 统计问答："这个月看了多久？比上个月呢？"（调 insights 工具）
- 库情问答："我评分最高的十部是哪些"、"XX 演员我还看过哪些没看的"
- Insights 页"AI 解读本期数据"（把 overview/breakdown 的数字变成自然语言洞察）
- 影片对比："这两部选一部今晚看，给我理由"

**L3（指令式任务）**
- 自然语言筛选 → 应用到资料库或创建 Saved View："建一个 90 分钟内、我评分 4 星以上、还没看过的睡前短片视图"
- 治理清单："找出疑似重复的演员给我一份合并建议"、"把乱七八糟的标签归并成规范清单"（产出走现有 merge-preview 确认流）
- 批量操作："把没有封面的片子都重新刮一遍，完了告诉我哪些失败"（刮削走现有任务系统，Agent 只负责发起 + 汇总）
- 萃取帧整理："给这部片的萃取帧补上场景标签"

**L4（后台/定时）**
- 每周口味报告、库健康巡检建议（接 `tasks.Manager` 后台生成，产出进通知中心）

---

## 2. 原子能力设计（给 Agent 的工具注册表）

### 2.1 设计原则

1. **一个动词 + 一个资源 = 一个工具**。不造复合工具（"整理我的库"这种复合是 Agent 规划层的事，不是工具层的事）。
2. **读写分离，三档权限**：
   - `read`：默认放行；
   - `write-preview`：返回变更预览 + 一次性 `confirmToken`；
   - `write-apply`：必须携带有效 token 才执行（token 模式照搬 actor merge，含过期时间）。
   低风险单字段写（评分、收藏）可以合并 preview/apply 为直接写，但默认仍经 UI 确认卡。
3. **工具面向 Agent，不等于 REST 翻版**：返回摘要化 DTO 而非原始行（`search_movies` 返回轻量行而非全字段），字段名稳定，列表必须带分页游标；实体间用 ID 引用。
4. **有界性**：`limit` 有上限；结果截断时置 `truncated: true` 并提示游标；长操作返回 `taskId` 由 `get_task_status` 轮询（复用现有任务与 SSE 体系），绝不阻塞等待。
5. **description 即文档**：每个工具的描述写清语义、参数含义、边界（"日期一律 IANA 时区"、"actor 多值为 AND"），Agent 全靠它工作。

### 2.2 工具清单（v1 建议，共 22 个）

**查询域（read，7 个）**

| 工具 | 语义 | 对应现有能力 |
|---|---|---|
| `get_library_overview` | 库容量、路径在线状态、最近扫描、健康摘要——Agent 的"定位起点" | `GET /api/health` + library paths + health 汇总 |
| `search_movies` | 条件检索；参数镜像列表 API（q/tag/actor/studio/playState/userRating/resolution/addedAfter/limit/offset）；返回轻量行 + `nextCursor` | `GET /api/library/movies` |
| `get_movie_detail` | 单片全量：元数据、用户数据、笔记、标签、进度状态 | `GET /api/library/movies/{id}` + comment |
| `list_actors` / `get_actor_profile` | 演员检索 / 档案（含 alias、用户标签、外链） | `GET /api/library/actors*` |
| `get_insights_overview` / `get_insights_breakdown` | 统计聚合（range/tz/dimension/limit 参数照搬） | `GET /api/insights/*` |
| `get_watch_history` | 按天回看历史 | playback progress/watch-time |
| `search_curated_frames` / `get_curated_frames_stats` | 萃取帧检索与统计 | `GET /api/curated-frames*` |
| `get_task_status` | 异步任务状态（Agent 发起批量后的跟进手段） | `GET /api/tasks/{id}` |

**用户数据写域（write，7 个）**

| 工具 | 语义 | 备注 |
|---|---|---|
| `set_user_rating` | 评分 0–5 | 低风险；单字段 |
| `set_favorite` | 收藏布尔 | 低风险；单字段 |
| `save_movie_comment` | 笔记整条替换 | 与现有 UPSERT 语义一致 |
| `update_movie_display_overrides` | user_title/user_summary/… 部分更新 | preview 返回前后 diff |
| `modify_movie_tags` | 标签增量 add/remove | |
| `create_saved_view` / `delete_saved_view` | 按筛选 schema v1 建视图 | 建议必走 preview（展示解析出的筛选条件） |

**治理域（preview→apply 强制，4 个）**

| 工具 | 语义 |
|---|---|
| `scan_duplicate_movies` | 只读发现：疑似重复影片组 |
| `suggest_actor_merges` | 疑似同一演员清单（LLM 判断 + 现有 merge-preview 管线产出 token） |
| `apply_actor_merge` | 带 token + profile 冲突选择执行合并（完全复用现有机制） |
| `suggest_tag_normalizations` | 标签归一化建议清单（应用走批量标签写） |

**任务域（异步，2 个）**

| 工具 | 语义 |
|---|---|
| `trigger_movie_scrape` | 单部或批量重刮 → 返回 taskId |
| `trigger_library_scan` | 全库/单路径扫描 → 返回 taskId（高危，默认关） |

### 2.3 通用契约

- 出参统一：`{ ok, data | error: { code, message }, truncated?, nextCursor? }`
- 确认协议：`write-preview` → `{ changes: [{ path, before, after }], confirmToken, expiresAt }`；`write-apply(taskId/toolCallId, confirmToken)` 校验后单事务执行；过期/参数变化即失效。
- 错误码进入 `contracts.go` 体系：`AI_TOOL_INVALID_ARGS` / `AI_TOOL_NOT_FOUND` / `AI_CONFIRM_REQUIRED` / `AI_CONFIRM_EXPIRED` / `AI_PROVIDER_UNAVAILABLE` / `AI_RATE_LIMITED`。
- Agent 循环护栏：单会话最大工具步数（建议 15）、写操作每分钟上限、总 token 预算。
- 脱敏：provider 为云端时按设置剥离 `location`、库路径等字段（本地模型不脱敏）。
- **同一注册表三处消费**：内嵌 Agent（function calling）、`POST /api/ai/actions/{name}`（前端按钮直调单工具）、MCP server（对外导出）。工具只写一份定义。

---

## 3. 布局与交互

### 3.0 总览：三条交互面 + 一条基建

```
┌─ AppShell ────────────────────────────────────────────────┐
│ 顶栏：… [导入] [通知] [✦Copilot] [主题]     ← A. 全局抽屉入口 │
│ ┌──────┬─────────────────────────────────┬──────────────┐ │
│ │侧栏  │  主内容区                         │ B. Copilot   │ │
│ │browse│  · 详情页"AI 润色/清洗"按钮        │    右侧抽屉   │ │
│ │yours │  · 资料库批量栏"AI 归一化标签"     │   (聊天+工具卡)│ │
│ │      │  · Insights"AI 解读"             │              │ │
│ └──────┴─────────────────────────────────┴──────────────┘ │
│ 底部：ScanProgressDock（批量 AI 任务进度复用此处）            │
└───────────────────────────────────────────────────────────┘
  C. 对外：MCP server（stdio `-mode mcp` + `/api/mcp`）→ Claude Desktop/Cursor/ZCode
  D. 基建：Settings → AI section（provider/脱敏/权限/用量/审计）
```

### 3.1 A. 全局 Copilot 抽屉（主交互面）

- **入口**：顶栏右侧 `NotificationCenter` 旁新增 Sparkles 图标按钮（`AppShell.vue:890-905` 区域）；不占侧栏导航位。
- **形态**：右侧 Sheet 抽屉（桌面 ~400px、移动端全屏），overlay 模式照搬现有 mobile sidebar（`AppShell.vue:924-951`）。可与主内容并排查看，支持"边看边问"。
- **上下文注入**：打开时把当前路由 + 当前影片/演员/筛选状态作为 system context（"用户正在看 ABC-123 详情页"），使"这部""这个演员"这类指代可用；注入内容对用户可见可清除（隐私透明）。
- **消息流渲染**：
  - 流式文本（SSE）；
  - 工具调用渲染为内联状态卡：`正在搜索资料库…` → 折叠的结果摘要（"命中 23 部，前 5：…"）；
  - 写操作渲染为**确认卡**：字段级 before/after diff + [应用] [放弃]，点击应用才带 token 重放；
  - 异步任务卡：显示 taskId 进度（复用 `use-scan-task-tracker` 模式），完成后可展开结果。
- **会话**：持久化（`ai_chat_sessions`）；抽屉内可新建/切换/清空。PIN 锁定下随 `/api/*` 一同被 423 保护。
- **降级**：未配置 provider 时入口显示但引导去设置；Mock 模式返回假流式 + 假工具结果（保证前端可独立开发）。

### 3.2 B. 就地动作（无聊天单发，L1/L2 的载体）

统一走 `POST /api/ai/actions/{name}`：无会话、无工具循环、后端 prompt 模板 + 必要上下文拼装，返回结构化结果。首批：

| 位置 | 动作 | 结果交互 |
|---|---|---|
| 详情页笔记区（`MovieCommentSection`） | 润色 | diff 预览 → 应用写回（走既有 PUT） |
| 详情页元数据（`MovieEditDialog`） | 清洗刮削摘要 | diff 预览 → 写 `user_summary` |
| 资料库批量栏（`LibraryBatchActionBar`） | 批量归一化标签 / 批量补摘要 | 转后台任务，进度走 `ScanProgressDock` |
| Insights 页（`PersonalInsightsPage`） | AI 解读本期数据 | 页面内流式/一次性渲染成卡片 |
| 演员详情页 | 疑似同一人建议 | 列表 → 逐条走现有 `ActorMergeDialog` |

就地动作用按钮 loading + 结果替换即可，不需要流式基建，是 M1 最小可交付形态。

### 3.3 C. 外部 Agent（MCP，零产品 UI）

- 同一工具注册表经 `internal/mcp`（官方 `modelcontextprotocol/go-sdk`）导出：
  - **stdio**：`curated.exe -mode mcp`（挂在现有 stdio 模式旁），MCP 客户端一条命令即接；
  - **Streamable HTTP**：`/api/mcp`，复用 PIN 中间件与 CORS/Host 校验。
- Settings → AI 提供"复制 MCP 配置 JSON"按钮，降低上手成本。
- MCP 侧默认只开放 read 工具 + 治理 preview；write-apply 仅在用户于设置中显式允许后暴露。

### 3.4 D. 基建与治理（Settings 新 section「AI」）

- provider 配置：`providerKind`（openai-compatible）/ `baseURL` / `apiKey` / `model`，出站走现有 `proxy` 配置，连通性测试交互照搬 `proxy/ping-*`；
- 脱敏开关（云端 provider 时剥离路径等字段）、写权限分级（"低风险直接写"默认关）、Agent 步数/频控阈值；
- 用量统计（token/请求数）与**审计视图**：`ai_tool_invocations` 表只读列表（时间、工具、参数摘要、结果状态、confirmToken 消耗）。

### 3.5 代码结构落点

**后端**（遵循 app/server/storage 三层惯例）：

```
backend/internal/llm/      # provider 接口 + OpenAI 兼容实现（openai-go，自定义 baseURL，走 proxyenv）
backend/internal/agent/    # ToolRegistry（定义+handler→app 层）/ AgentLoop（步数上限+SSE 事件）/ PermissionGate / ConfirmStore
backend/internal/mcp/      # ToolRegistry → MCP server 桥接（stdio + streamable HTTP）
backend/internal/server/   # POST /api/ai/chat(SSE) · POST /api/ai/actions/{name} · /api/ai/sessions CRUD · /api/mcp
migrations/00XX_ai_*.sql   # ai_chat_sessions / ai_tool_invocations
```

**前端**（遵循 contract + web/mock 双适配边界，views 不 import 具体 adapter）：

```
src/services/contracts/ai-service.ts            # chat 流 / actions / sessions
src/services/adapters/web/ + adapters/mock/      # 各一份；mock 出假流式
src/components/copilot/                          # CopilotDrawer / ChatMessageList / ToolCallCard / WriteConfirmCard / ChatInput
src/composables/use-copilot.ts                   # 抽屉开关 + 会话状态 + SSE 消费（注意：不走 30s 超时的 http-client，独立 fetch+ReadableStream）
```

### 3.6 关键交互决策（已定建议）

| 决策点 | 建议 | 理由 |
|---|---|---|
| Copilot 常驻侧栏 vs 抽屉 | **抽屉** | 不动现有 grid 布局；overlay 模式有现成实现 |
| 聊天入口进侧栏导航 | 不进 | 顶栏图标足够；侧栏分组保持信息架构干净 |
| 写操作确认 | 默认全部确认卡；设置可放行"低风险直接写"（评分/收藏） | 信任分级，安全默认 |
| 聊天流式 | 必须 | 长回答无流式体验不可用；SSE 基建现成 |
| 就地动作用聊天会话 | 不用 | 单发 action 更简单、可测试、Mock 友好 |
| MCP 对外写权限 | 默认只读 + preview | 对外面更大，先开放读 |

### 3.7 分期落地

- **M0 基建**：Settings AI section + `internal/llm` + 连通性测试。无用户可见 AI 功能。
- **M1 就地动作**：`/api/ai/actions/{name}` + 详情页润色/清洗 + Insights 解读（L1/L2，无工具循环）。
- **M2 Copilot 只读**：工具注册表（查询域）+ `/api/ai/chat` SSE + CopilotDrawer（聊天 + 只读问答 + 上下文注入）。
- **M3 写能力与治理**：确认协议 + 写域/治理域工具 + 确认卡交互 + 审计。
- **M4 对外与自动化**：MCP 导出（stdio + HTTP）+ 批量任务 + 定时报告（L4）。

每期都是可独立发布的最小单元，符合最小修改提交工作流。

# AI 接入调研（2026-08-17）

> 状态：调研文档（未实施）。产品：Curated。范围：AI/LLM 能力接入的场景、对外接口、Agent 集成方式与现成方案、可拓展的需求方向。
> 后续深化：能力分层、原子工具注册表与交互布局设计见 `2026-08-18-agent-capability-and-interaction-design.md`。
> 结论基于当前代码事实（`backend/` Go + SQLite、`src/` Vue3 + shadcn-vue、服务层 web/mock 双适配）与 2026-08 外部生态调研。

## 0. 现状基线（与 AI 接入相关的事实）

- 代码中**完全没有** AI/LLM 相关代码或依赖，接入是全新增量，无历史包袱。
- 数据基础好：影片元数据（`movies` 含 `user_*` 展示覆盖列、`user_rating`、`is_favorite`）、演员（`actors` + alias/merge 体系）、标签、影片笔记（`library_movie_comments`，每片一条）、观看数据（`playback_daily_watch_time`、`playback_progress`、`library_played_movies`）、萃取帧（`curated_frames` 含标签/演员 JSON）、推荐快照与反馈。
- 已有聚合统计管道：`GET /api/insights/overview|breakdown`（观看时长、完播率、评分、actor/studio/tag 维度）。
- 已有流式基建：`GET /api/events`（SSE，task.updated 推送），可直接复用为 LLM 流式输出通道。
- 已有 stdio 运行模式（`-mode stdio`，如 `settings.get`），是未来挂 MCP stdio transport 的天然入口。
- 已有出站代理体系（`library-config.cfg` 的 `proxy` → `internal/proxyenv`），LLM 请求可复用。
- 已有 PIN 应用锁 + 会话中间件（`/api/*` 423 保护）、写操作确认范式（actor merge 的 preview → token → confirm 模式）。
- 隐私定位：本地优先的桌面应用，且媒体内容为成人向——**云端 LLM 的内容政策是现实风险**（OpenAI/Anthropic/Gemini 对成人内容元数据处理可能拒绝），本地模型（Ollama/LM Studio）或对此宽容的 provider 是重要选项。

---

## 1. 接入场景

### 1.1 分析 / 统计类

| 场景 | 数据来源（现状） | AI 增量 |
|---|---|---|
| 个人洞察总结 | `/api/insights/*` 聚合管道 | 自然语言报告：观看趋势、口味变化、完播率解读、"收藏了 200 天没看"之类的行为洞察 |
| 库健康/整理建议 | Library Health 扫描结果、重复检测 | 把 finding 翻译成人话 + 给出行动建议（哪些该修、哪些该删） |
| 推荐解释与重排 | 推荐是规则算法（v8）+ reason codes | LLM 重排候选、生成个性化解释文案；用户反馈（not_interested/snooze/less）可作为 LLM 上下文 |
| 影片/演员问答 | 元数据 + 笔记 + 萃取帧标签 | "这个演员我还看过哪些"式的对话查询，本质是 RAG over 本地库 |
| 导入/扫描兜底 | `scanner/number.go` 从文件名提番号 | 识别失败时用 LLM 解析非标准文件名（ Release/无码/FC2 等命名变体） |

### 1.2 编辑类（"编辑消息"）

现有落点非常直接，都有端点和存储：

- **影片笔记润色**：`MovieCommentSection.vue` + `GET/PUT /api/library/movies/{id}/comment`（`library_movie_comments`）。加一个"AI 润色"按钮，diff 预览后保存。后续产品确认不做扩写/翻译（见 REQ-0029）。
- **刮削摘要清洗**：刮削来的 `summary` 常含广告/水印/推广文字。AI 清洗后写入 `user_summary`（展示覆盖列已存在，设计上就是为这类覆盖准备的）。`MovieEditDialog.vue` 同理可加标题/厂牌建议。
- **标签归一化与翻译**：`tags` 来自多 provider 刮削，混杂（中/英/日、同义标签）。AI 批量归并建议（复用 actor merge 的 preview→confirm 交互范式）。
- **萃取帧打标签**：`curated_frames` 的 `tags_json`，AI 根据影片上下文建议标签。
- **演员归并建议**：alias/canonical merge 体系已很完整（preview/audit/token），LLM 辅助判断"疑似同一人"（拼写变体、不同写法），产出仍走既有 merge-preview 流程。

### 1.3 对话助手（全局 Copilot）

一个跨页面的聊天入口，自然语言驱动现有能力：

- **自然语言筛选**："2023 年以后、XX 演员出演、我评分 4 星以上、还没看过的" → LLM 结构化输出转成 `GET /api/library/movies` 的既有 query 参数（或直接创建 Saved View——筛选 v1 schema 已版本化）。
- **统计问答**："这个月看了多久？" → 调 insights 工具回答。
- **操作代理**：收藏、评分、触发刮削、开播放器（写操作需确认）。

---

## 2. 接口需求（给 AI 提供什么接口）

### 2.1 设计原则

1. **只读先行**：第一批工具全部只读，写操作走"preview → 用户确认 → apply"（项目已有成熟范式：actor merge、Library Health repairs）。
2. **复用现有 API 语义**：工具参数即现有列表 API 的 query 参数，不另造查询语言。
3. **脱敏分级**：给云端 LLM 的上下文可配置剥离磁盘路径、本地目录结构；本地 LLM 无此顾虑。
4. **PIN/会话保护照常生效**；分页强制（limit 上限），防止模型把全库拉进上下文。

### 2.2 工具清单建议（映射现有端点）

只读：
- `search_movies`（参数 = 列表 API 的 q/tag/actor/studio/playState/userRating/resolution/addedAfter/limit/offset）
- `get_movie_detail(movieId)`（含评分、收藏、笔记、演员、标签）
- `list_actors(q, actorTag, sort)` / `get_actor_profile(name)`
- `insights_overview(range, tz)` / `insights_breakdown(dimension, range, limit)`
- `get_watch_history(days)` / `list_curated_frames(movieId|actor|tag)` / `curated_frames_stats`
- `get_library_health_summary()`（只读扫描结果）

写入（需确认）：
- `update_movie_display_overrides(movieId, user_title/user_summary/...)`（= PATCH movie）
- `set_user_rating` / `set_favorite` / `put_movie_comment`
- `suggest_tag_merges(preview)` → 走既有确认流
- `create_saved_view(name, filters)`（筛选 schema v1 直接可用）

### 2.3 接口形态（三层，可叠加）

1. **Function calling / tools JSON Schema**（对内嵌 Agent）：工具定义即 2.2 清单，OpenAI 兼容 `tools` 格式，任何 OpenAI 兼容模型（OpenAI/DeepSeek/Qwen/Ollama/LM Studio）都能用。
2. **MCP server**（对外）：把 Curated 暴露为 [MCP](https://modelcontextprotocol.io) server，Claude Desktop / Cursor / ZCode 等通用 Agent 客户端即可直接查询、操作本地库。Go 有官方 SDK [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)（与 Google 合作维护，已 1.0+）。transport 两种都现实：
   - **stdio**：`curated.exe -mode mcp` 挂在现有 stdio 模式旁边，用户在 MCP 客户端里配一条命令即可；
   - **Streamable HTTP**：挂在现有 `/api` 下（如 `/api/mcp`），复用 PIN 中间件与 CORS 策略，LAN 内可访问。
3. **REST 上下文端点**（最简）：`GET /api/ai/context?movieId=...` 返回打包好的"该片全量上下文"（元数据+笔记+统计），供用户粘贴到任意 ChatGPT/网页版使用。成本最低，可作为 Phase 0。

---

## 3. Agent 集成

### 3.(a) 集成架构（推荐：后端内嵌 Agent）

推荐把 Agent 做进 Go 后端（而非前端直连 LLM API）：密钥不出本机后端、工具直接调 `internal/app` 层方法、与 PIN/代理/任务体系天然整合。

```
Vue 前端                     Go 后端                            外部
─────────                    ─────────                          ──────
聊天 UI (shadcn-vue 自建)  →  POST /api/ai/chat (SSE 流式)  →   internal/llm provider 抽象
"AI 润色"按钮              →  POST /api/ai/actions/{name}        (OpenAI 兼容 baseURL/key/model：
                                │                                  OpenAI/DeepSeek/Qwen/Ollama/
                                ▼                                  LM Studio 皆可)
                          internal/agent
                          ├─ tool registry (JSON Schema + handler→app 层)
                          ├─ agent loop (function calling 循环, 最大步数限制)
                          └─ 权限门 (只读放行/写需 confirm token)

                          internal/mcp (可选) = 同一批工具暴露为 MCP server
```

落地要点：

- **Provider 抽象**：只需实现"OpenAI 兼容 chat completions + tools"一个接口，即可覆盖几乎所有云端/本地模型。设置放 Settings 新 section（`baseURL`/`apiKey`/`model`/`providerKind`），连通性测试复用现有 `proxy/ping-*` 的交互模式，出站走现有 `proxy` 配置。
- **SDK 选择**：官方 [openai/openai-go](https://github.com/openai/openai-go) 支持 `NewStreaming` + `ChatCompletionAccumulator`（流式 + 工具调用增量合并都处理好了），且支持自定义 baseURL。工具数固定在 10~20 个的场景，**自写 agent loop（约 200~300 行）比上框架更可控**。
- **流式通道**：`POST /api/ai/chat` 返回 SSE（复用 `/api/events` 的写法），事件含 `text_delta` / `tool_call` / `tool_result` / `done` / `error`；前端已有 SSE 消费基建（`use-scan-task-tracker`）。
- **写操作确认**：agent 工具返回 `needs_confirmation` + preview payload，前端弹确认框（复用 merge-preview 交互），确认后带 token 重放——LLM 永远拿不到直接写权限。
- **会话与审计**：新迁移 `ai_chat_sessions` / `ai_tool_invocation_audits` 表；工具调用全记录，出问题可追溯。
- **前端**：Vue 生态没有成熟的 LLM 聊天组件库（Nuxt UI chat 组件与 Vercel AI SDK/Nuxt 绑定较深），且项目已有成熟的 shadcn-vue 体系与服务层契约模式——**用 shadcn-vue 自建聊天组件、走 `contracts/library-service.ts` 同款 contract + web/mock 双适配**最贴合现状。Mock 适配器可返回假流式，保证无后端时 UI 可开发。
- **降级**：未配置 provider / 无网络时按钮置灰并引导到设置页。

### 3.(b) 现成方案对比

**Go 内嵌框架：**

| 方案 | 定位 | 评价 |
|---|---|---|
| 自写 loop + openai-go | 轻量 | 工具少、逻辑直白时最可控，无框架升级负担。**首选起步方案** |
| [cloudwego/eino](https://github.com/cloudwego/eino) | 字节开源，Go 全家桶（ChatModel/Tool/编排/Agent），10k+ stars，生产验证 | Go 生态当前最强框架；当工具多、需要多步编排/并发图时值得引入。支持 OpenAI 兼容模型 |
| tmc/langchaingo | LangChain 的 Go 移植 | 原型快，但维护与 Agent 能力弱于 Eino，2025 年后社区共识是 Eino 更优先 |
| modelcontextprotocol/go-sdk（官方）/ mark3labs/mcp-go | MCP server SDK | 对外暴露 Curated 为 MCP 用，官方 SDK 已 1.0+，与 Google 合作维护 |

**外挂平台（不改或少改产品代码）：**

| 平台 | 适合 | 与 Curated 的结合方式 |
|---|---|---|
| Claude Desktop / Cursor / ZCode + MCP | 零 UI 成本 | Curated 出一个 MCP server，用户在通用 Agent 里聊天操作库；**性价比最高的第一步** |
| Dify | 低代码 LLM 应用/RAG | 把 Curated REST（已有 `API.md`）注册为 Dify 自定义工具，搭聊天/知识库应用；适合快速试验，不适合随安装包分发 |
| n8n | 流程自动化（开源 Zapier） | HTTP 节点调 Curated API，做定时报告（每周库报告、健康摘要邮件）；侧重视自动化而非对话 |
| Open WebUI + Ollama | 本地聊天前端 | 本地模型聊天 UI 现成；可挂工具/MCP 调 Curated；适合"全家桶本地化"用户，非产品内置 |

**建议组合**：短期 = MCP server（对外，Claude Desktop/Cursor 直连）+ 自写 agent loop（对内，聊天与按钮动作）；工具数量和编排复杂度上来后再评估 Eino。

---

## 4. 需求拓展（后续可开发方向）

按"数据已有、增量小 → 想象空间大"排序：

1. **自然语言 Saved View**：一句话建筛选视图（"睡前 90 分钟内、评分 4 星以上、没看过的"），落库走现有 `library_saved_views`（schema v1 已版本化，可直接扩展）。
2. **智能去重/归并**：影片重复检测建议、演员疑似同一人建议（merge 管道现成）。
3. **语义搜索 / 嵌入**：对 summary、笔记、标签做本地嵌入（Ollama embedding 模型），实现跨语言搜索（日文标题搜中文词）、"找类似这部"的相似推荐（现关联推荐是规则匹配）。
4. **多模态（本地模型）**：萃取帧精彩度评分、自动选封面、帧内容描述辅助检索。
5. **字幕与语音**（大工程）：Whisper 本地转字幕 → 字幕全文检索 → "找到提到 XX 的片段"跳转时间轴。
6. **定时 AI 报告**：每周口味变化报告、库整理清单，接入现有任务系统（`tasks.Manager`）后台生成，产出落 PDF/MD 或推送通知中心。
7. **导入预处理兜底**：番号识别失败时 LLM 解析文件名；元数据刮削全失败时基于文件名的 LLM 猜测填充。
8. **多语言翻译**：summary/笔记/标签翻译（i18n 基建已有 zh/en/ja）。
9. **用量与隐私面板**：token 用量统计、脱敏开关（云端 provider 时剥离标题/演员名/路径）、按 provider 的成本估算。

---

## 5. 风险与注意

- **内容政策**：云端主流 API 对成人内容元数据可能拒绝处理；设置页应明示"本地模型更适合本库内容"，provider 探测时给出内容政策提示。
- **隐私**：默认不把任何数据送云端；云端 provider 需显式开启 + 脱敏选项。
- **密钥管理**：apiKey 存 `library-config.cfg`（已有 PATCH /api/settings 原子写回机制），注意不进 release 样例配置（现有脱敏基线治理可复用）。
- **流式中断与超时**：LLM 长响应需独立于现有 30s http-client 超时的 SSE 通道；Mock 模式需模拟流式以便前端开发。
- **Windows 本地模型资源**：Ollama 与转码会话（FFmpeg）可能同时吃 GPU/CPU，设置页应提示资源冲突。

## 6. 建议路线（供后续立项参考）

- **Phase 0**：Settings 增加 AI/LLM section（provider 配置 + 连通性测试），无任何 AI 功能，纯基建。
- **Phase 1**：两个最快见效点——`GET /api/ai/context` 上下文端点（粘贴外部用）+ insights 页"AI 总结"按钮（只读、单次调用）。
- **Phase 2**：编辑类（笔记润色、summary 清洗），按钮式、diff 预览、走既有 PATCH 端点。
- **Phase 3**：聊天 Agent（自写 loop + 工具注册表 + SSE + 写确认）与 MCP server（stdio 模式挂载）。
- **Phase 4**：语义搜索/嵌入、定时报告、多模态等拓展方向。

## 参考链接

- MCP 官方 Go SDK：<https://github.com/modelcontextprotocol/go-sdk> · <https://modelcontextprotocol.io/docs/2026-07-28/sdk>
- openai-go 官方 SDK（流式 + 工具调用）：<https://github.com/openai/openai-go> · Function Calling 指南 <https://developers.openai.com/api/docs/guides/function-calling>
- Ollama OpenAI 兼容 API：<https://docs.ollama.com/api/openai-compatibility> · <https://ollama.com/blog/openai-compatibility>
- Eino（CloudWeGo）：<https://github.com/cloudwego/eino> · <https://www.cloudwego.io/zh/docs/eino/overview/>
- Eino vs LangChain 对比（CSDN）：<https://blog.csdn.net/gitblog_00717/article/details/151092964> ·（知乎）<https://zhuanlan.zhihu.com/p/1989291175090345165>
- Dify / n8n / Open WebUI 选型：<https://developer.cloud.tencent.com/article/2541767> · <https://zhuanlan.zhihu.com/p/1898775808660710158>
- Vue LLM 聊天 UI 生态综述（2026）：<https://alexander-lukashov.medium.com/the-overview-of-ui-libraries-for-ai-chat-interfaces-in-2026-146a1492114a>

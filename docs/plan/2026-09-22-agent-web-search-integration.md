# Curated Agent 联网搜索接入评估

日期：2026-09-22。状态：研究与实施建议，尚未接入或调用付费搜索服务。

## 结论

可以引入联网搜索，并参考成熟开源 Agent 的实现。补充核实 OpenCode、Gemini CLI 与 Cline 后，建议为 Curated 增加独立搜索服务接口，第一阶段优先评估 Exa 远端 MCP（免 Key 限流入口与自带 Key）；MiniMax、Parallel、Claude 原生搜索作为可选适配。无需把完整编码 Agent 作为 Curated 主运行时。该结论取代本文件初版“MiniMax MCP 优先”的建议。

## 当前实现

- Go Agent 已有工具注册表、执行网关、事件流与调用审计，可以承载搜索工具。
- 当前只有 `search_provider_titles` 和受本轮来源 URL 限制的 `get_source_page`，没有通用网页搜索，也没有 MCP 客户端。
- 当前模型层使用 OpenAI 兼容 chat completions。更换为 MiniMax 或 Claude 模型名称，不会自动附带搜索服务。
- 系统提示词与早期 Agent charter 明确排除了通用网页搜索。落地本次能力时需同步更新产品范围、提示词和文档，不能只注册工具。

## 官方能力与接入路径

| 路径 | 已核实能力 | 对 Curated 的影响 |
|---|---|---|
| MiniMax Code 开源实现 | `web_search` 通过 LocalMatrixClient 调用远端 Matrix 工具服务；网页读取另有 HTTP 实现 | 可借鉴工具定义和结果处理；开源代码不意味着远端搜索服务可匿名使用 |
| MiniMax 官方 Web Search MCP | 提供 `web_search(query)`；文档使用 `uvx minimax-coding-plan-mcp`，配置 API key 和 host；当前要求 Token Plan seat 或 Credits 权限 | 增加受管理的 MCP stdio 客户端与进程生命周期；桌面部署需处理 uvx/Python 依赖，也可进一步核实公开 HTTP 合约后实现 Go 适配 |
| Claude API Web Search | Claude 服务端执行搜索并返回结果、引用和用量 | 增加 Anthropic Messages 协议与服务端工具事件适配，或独立的 Claude 搜索子请求；不能把原生工具直接塞进现有 OpenAI 兼容 function schema |
| Claude Code / Agent SDK | 官方支持 CLI、Python、TypeScript 程序化运行，复用其 Agent 工具与执行循环 | 可以作为可选桥接，但增加运行时、认证、输出解析与嵌套 Agent 成本；不建议仅为搜索默认依赖整个编码 Agent |

MiniMax Code 阅读版本：`ae65651df5f97ae1085ab4e19964f4b78c769a4e`。仓库许可证为 MIT；如复制代码须保留许可证声明。这里仅做定向阅读，未引入其源码或依赖。

Claude 搜索官方页面本次成功读取到价格为每 1,000 次搜索 10 美元，另计相关 token；仅作研究时点参考。账户可用性、模型支持与配额应在实际接入时验证。后续再次抓取部分 Claude 文档返回 403，本次未做凭据或运行时验证。不得默认把 Claude Code 订阅等同于可供产品后端复用的搜索 API 凭据。

## 建议的实现边界

1. 在 Go 服务层定义独立 `WebSearchProvider`，与聊天模型配置分离。模型通过普通函数工具发出查询，后端选择搜索服务并返回规范化结果，因此聊天模型与搜索供应商不必相同。
2. 注册 `search_web`，返回有界结果集，包含来源 ID、标题、URL、摘要、抓取时间与失败原因。扩展现有网页读取能力处理本轮搜索返回的公网 URL，保留超时、大小限制、私网拒绝与重定向校验。
3. 扩展来源引用结构和回答核验规则。网页摘要不能直接当作已读取全文；普通网页引用不能直接变成本地影片 ID。库外资料经过身份对账后才能关联本地卡片。
4. 在设置中增加联网搜索开关、服务商、凭据和连通测试。凭据只由后端使用；每轮搜索次数和预算独立于工具总步数配置，查询不默认携带整个资料库或聊天记录。
5. 结果状态按尚未解决的问题判断。已成功修正的 `submit_answer` 校验失败不能永久污染最终状态；搜索无结果、网页无法读取、配额不足、引用失败应分别解释，不能只显示无原因的“部分完成”。

## 分阶段交付建议

- 第一阶段：独立搜索配置、Exa 远端 MCP 适配、`search_web`、网页读取和来源引用，配套取消、超时、配额错误与 Mock 验证。无需为此引入 Python 或完整编码 Agent；生产接入仍应遵循服务方当前 MCP 协议，而不是机械复制某个客户端的简化调用。
- 第二阶段：按真实需求增加 MiniMax、Parallel 或 Claude API 搜索适配；原生模型搜索需明确服务端工具循环、引用事件、计费和上下文管理。需要自托管时再评估 SearXNG。
- Claude Code 桥接作为单独可选能力，仅在需要其完整 Agent 工具流程时考虑。

验收应覆盖：搜索成功带可追溯链接；无结果不宣称事实不存在；网络/配额失败有具体原因；修正成功恢复正常完成状态；搜索结果不能伪造本地资源身份；后端退出清理托管 MCP 进程；未配置搜索时现有本地功能正常。

## 来源

- [MiniMax Web Search MCP](https://platform.minimax.io/docs/token-plan/mcp-guide)
- [MiniMax Code 搜索客户端](https://github.com/MiniMax-AI/minimax-code/blob/ae65651df5f97ae1085ab4e19964f4b78c769a4e/packages/local-runtime/src/web-search/local-web-search-client.ts)
- [MiniMax Code 搜索工具定义](https://github.com/MiniMax-AI/minimax-code/blob/ae65651df5f97ae1085ab4e19964f4b78c769a4e/packages/agent-tools/src/shared/web-search.ts)
- [Claude API Web Search](https://platform.claude.com/docs/en/agents-and-tools/tool-use/web-search-tool)
- [Claude Code 程序化运行](https://code.claude.com/docs/en/headless)

本评估不改变已有运行配置、不安装 CLI 或 MCP 服务、不执行真实付费搜索。

## 成熟开源项目补充调查（2026-09-22）

### OpenCode

阅读 dev 分支固定版本 `e059ac5918f3e2c798de029b9df4cede617466ed`，不将其等同于用户已安装的稳定版本。

- `packages/opencode/src/tool/websearch.ts` 将搜索封装为独立工具，当前实现包含 Exa 和 Parallel 两个后端，可通过 `OPENCODE_WEBSEARCH_PROVIDER` 指定。
- `mcp-websearch.ts` 通过 HTTP JSON-RPC `tools/call` 调用 `https://mcp.exa.ai/mcp` 的 `web_search_exa` 或 `https://search.parallel.ai/mcp` 的 `web_search`；解析 JSON / SSE 结果。支持可选 `EXA_API_KEY`，Parallel 分支支持可选 `PARALLEL_API_KEY`。这是远端搜索服务，不是本地搜索引擎。
- `webfetch.ts` 是独立 HTTP 抓取工具，支持文本、Markdown、HTML，含超时和响应大小限制；HTML 使用解析器提取文本或转换为 Markdown，不等于完整浏览器渲染。
- Exa 官方文档明确提供 Keyless 模式：免登录、免 Key、免费但限流；也支持 OAuth 和 API key。不能把免 Key 解读为无限额度或服务可用性保证。Parallel 本次只核实客户端实现，未验证免 Key 的服务额度。
- 适合借鉴的是搜索与网页读取分离、搜索后端可替换、取消/超时、结果裁剪。Curated 不需要照搬其运行框架、实验分流，也不默认传送会话标识或模型名。

### 其他参考项目

| 项目与固定版本 | 已核实机制 | 对 Curated 的参考价值 |
|---|---|---|
| Gemini CLI `cfbcaa8df13ea4610bb379b377b56d62980c0032` | `web-search.ts` 通过 Gemini client 的 `web-search` 模型配置调用 Google Search grounding，读取 groundingChunks / groundingSupports 并生成引用 | 参考“事实段落关联来源”的引用处理；搜索服务仍来自 Google |
| Cline `5d765e27797387a30ffdd246e519fbee11799c44` | 官方 SDK 的 `examples/plugins/web-search.ts` 展示独立插件调用 Exa Search HTTP API，使用独立 `EXA_API_KEY`，规范化标题、URL、摘要、日期等 | 参考独立搜索 provider 和结构化结果；这是官方插件示例，不能据此概括全部 Cline 产品的默认搜索后端 |
| SearXNG 官方 Search API | 自托管实例提供 `/search?q=...&format=json`，需在配置中启用 JSON，很多公共实例禁用该格式 | 可选自托管聚合搜索，但仍依赖上游引擎和运维，不是获得独立的全网索引 |

开源 Agent 通常开放工具封装、网页读取和 Agent 编排；搜索索引由外部服务或上游引擎提供。通用 MCP 是接入协议，并不是搜索数据来源。成熟实现可以减少工程试错，实际服务质量仍需用 Curated 的目标查询验证。

补充来源：

- [OpenCode 搜索工具源码](https://github.com/anomalyco/opencode/blob/e059ac5918f3e2c798de029b9df4cede617466ed/packages/opencode/src/tool/websearch.ts)
- [OpenCode MCP 搜索调用](https://github.com/anomalyco/opencode/blob/e059ac5918f3e2c798de029b9df4cede617466ed/packages/opencode/src/tool/mcp-websearch.ts)
- [OpenCode 网页读取](https://github.com/anomalyco/opencode/blob/e059ac5918f3e2c798de029b9df4cede617466ed/packages/opencode/src/tool/webfetch.ts)
- [OpenCode 工具文档](https://opencode.ai/docs/tools/)
- [Exa MCP 与认证方式](https://exa.ai/docs/reference/exa-mcp)
- [Gemini CLI 搜索源码](https://github.com/google-gemini/gemini-cli/blob/cfbcaa8df13ea4610bb379b377b56d62980c0032/packages/core/src/tools/web-search.ts)
- [Cline 搜索插件示例](https://github.com/cline/cline/blob/5d765e27797387a30ffdd246e519fbee11799c44/sdk/examples/plugins/web-search.ts)
- [SearXNG Search API](https://docs.searxng.org/dev/search_api.html)

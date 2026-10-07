# Agent 提示词缓存优化（DeepSeek）

日期：2026-10-07（北京时间）

状态：in-progress。关联 REQ-0060；源码优化与本地回归已落地，真实服务的命中率与费用对照待观察。未发布或替换运行中的 Server。

## 问题与依据

用户反馈 Agent 缓存命中率低、费用难以承受，并确认使用 DeepSeek。未提供具体模型、原始命中率或是否经第三方代理，不能宣称达到某个百分比或节省金额。

已读取 [DeepSeek Context Caching 官方文档](https://api-docs.deepseek.com/guides/kv_cache)。缓存自动工作；后续请求必须完整匹配已经持久化的缓存前缀单元。当前文档说明请求输入/输出结束位置、检测到的共同前缀和长输入固定间隔都可能形成单元。缓存建立需要数秒，采用 best-effort；空闲后通常数小时至数天清理。不能将旧版固定 token 粒度当成当前保证。

源码发现：页面路由、实体、筛选、提及和截断说明改写第一条 system，页面变化会在历史之前改变前缀。润色和展示翻译把原文放进 system 和 user 两次。洞察 JSON 虽只放一次，但在 system 中，规则消息也随数据变化。工具注册表已按名称排序，Go JSON 编码稳定排序字符串 map 键，无需再调整。用量客户端此前不解析缓存计数，应用内总 token 不能证明缓存效果。

## 已落地的修改

1. 聊天按固定 system（领域、安全规则、响应语言）→ 已有摘要/历史 → 本轮页面 system → 最新 user 组装。空页面不生成补充消息；截断说明进入本轮补充消息，固定规则保持一致。
2. 页面继续使用既有白名单投影和净化，不持久化为历史授权。system 角色使长任务压缩保留当前页面、筛选、提及和截断说明。
3. 独立动作的 system 只包含规则和受控类型/语言参数。源资料通过 `action-source.txt` 包装后在 user 中发送一次；洞察发送完整统计 JSON，保留只读与事实约束。
4. 聊天版本为 `agent-system-v8`，动作为 `agent-actions-v2`。所有正文沿用 embed TXT，重新编译/重启后生效。
5. 流式与普通响应解析 DeepSeek 的 `prompt_cache_hit_tokens` / `prompt_cache_miss_tokens`，兼容 `prompt_tokens_details.cached_tokens`。缺失值保持未知；负值、超输入量或总和矛盾不作为可信缓存统计。命中是输入的子集，不重复累计总 token。
6. 每次有可信命中计数的调用写 `AI prompt cache usage` Info 日志：runId、channel、model、promptVersion、promptTokens、promptCacheHitTokens，以及服务报告时的 promptCacheMissTokens。不记录提示词或源资料。无新增数据库字段、HTTP 端点、配置或报表列。

## 验收与成本观察

本地回归覆盖页面切换/清除后的共享前缀、截断不改基础规则、压缩保留书库页面且不带错误影片 ID、源文本只发送一次且模板符号/CRLF 保留、流式/非流式缓存计数、冷缓存的零与缺失字段的区别、异常计数及日志不重复累计 token。

更新 Server 后用同一 endpoint/model 观察，区分聊天与动作、首轮与后续轮，比较相近输入长度和工具使用量。先正常使用数轮，给服务建立缓存的时间。按同一窗口计算 `sum(hit) / sum(promptTokens)`，只纳入有可信 hit 的调用；不要简单平均每次百分比，也不要将未知值当零。费用还含未命中输入与输出，不能只凭缓存比例推导费用。

历史压缩/截断、模型或工具 schema 变化、工具结束/修正阶段缩减可用工具、语言切换和服务缓存淘汰仍会减少复用。持久历史只保存发布正文和来源快照，不重放完整工具轨迹；本次不增加原始工具轨迹持久化。动作规则较短，去重收益明确，缓存收益可能有限。冷缓存、新提示词首次调用不保证命中。

## 验证记录

`backend/` 的 `go test ./internal/agent/... ./internal/llm/... ./internal/app/...`、全量 `go test ./...`、`go vet ./...` 均通过；补充缓存日志回归通过；`git diff --check` 通过。未调用收费模型，尚无真实命中率提升数据。

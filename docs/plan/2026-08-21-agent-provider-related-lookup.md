# Agent 源站周边：评价、演员介绍、库外作品

日期：2026-08-21
状态：implemented（Phase A/B/C；无 `search_web`）
关联需求：REQ-0044
上游：[`2026-08-18-agent-charter.md`](2026-08-18-agent-charter.md) · [`2026-08-19-agent-milestone-plan.md`](2026-08-19-agent-milestone-plan.md) E2 查询域

## 0. 结论

给 Agent 补「影片评价、演员介绍、该演员在库外的其他作品」，**不接通用网页搜索**（Brave / Google / Tavily），也**不做天气、新闻类问答**。

网上周边沿着 **Metatube 已经给出的源站 URL 与检索** 往下挖：刮削落库的 `homepage` / 站点评分，加上按演员或番号再搜一批源站条目，必要时只读取本轮已出现的 https 源站页。这是 L2 检索增强，放大现有刮削能力，不引入新的搜索引擎产品。

## 1. 问题

当前 Agent 只能看见本地库行。`get_movie_detail` / `get_actor_profile` 丢掉了刮削已经写入的源站入口；演员档案也不存片单。用户问「评价怎么样」「她还拍过什么」时，模型要么编造，要么只能复述库内那几部。

同时 Metatube 其实已经带回不少周边锚点：

| 数据 | 落库情况 | Agent 现状 |
|---|---|---|
| 影片 `movies.homepage` | 刮削写入 | 详情工具未暴露；列表 SQL 也未选出 |
| 影片站点评分 `movies.rating` | 写入，API 为 `metadataRating` | 详情工具未暴露 |
| 影片 `movies.provider` | 写入 | 详情 DTO 有 `metadataProvider`，工具未暴露 |
| 演员 `actors.homepage` / `summary` | 刮削写入 | 工具有简介，无源站主页 |
| 用户外链 `externalLinks` | 有 | 工具已返回 |
| Meta 的 `series` | 刮削结构体有，**SQLite 未存** | 本期不做持久化 |

用户明确不要通用搜索。评价和长介绍在源站页上；库外作品在 Meta 按演员/番号检索里。

## 2. 价值主张与铁律

- **翻译**：把「她还拍过什么 / 评价如何」翻成源站检索与已知 URL 读取。
- **模糊判断**：用源站摘要回答口碑与介绍，并标明哪些番号已在库内。
- **不引入**独立搜索产品；底层能力是刮削已经在用的 provider。

约束：

- **P-01** 只走注册表。禁止模型自由 HTTP。
- **P-04** 先扩展既有 `get_*` 参数/返回；新能力用既有动词 × 新名词：`search_provider_titles`、`get_source_page`。不注册 `search_web`。
- **P-06 / 1.3** 查询打向用户已配置的刮削源站（现有 `proxyenv`），不把番号丢给第三方搜索 SaaS。云端 LLM 仍脱敏路径类字段；homepage URL 可保留。
- **P-09** 源站 HTML/摘要一律 `<source>`；工具层 `wrapSource`。
- **P-10** 列表上限（默认 15，硬顶 25）+ `truncated`；读超时 10s；失败截断告知，不崩非 AI 功能（P-07）。
- **P-11 / P-12** Mock 假结果；网关与 FakeChatClient 剧本单测。

## 3. 明确不做

- 通用 `search_web(query)`、天气/新闻/百科闲聊。
- 任意 URL 抓取（SSRF）；`file://`、内网、非 https。
- 把库外番号伪装成本地 `movieId`，或对其调用 `present_movies`。
- 新的用户可见影片卡组件（实验期少 UI：库外作品以工具摘要 + 助手叙述呈现）。
- 把 `series` 列补进 SQLite（另立需求）。
- 就地按钮、MCP 专用通道、新 HTTP 端点（仍走 `/api/ai/chat` 工具循环）。

## 4. 分三期交付

```
用户问评价 / 介绍 / 还拍过什么
        │
        ├─ Phase A  get_movie_detail / get_actor_profile
        │            + homepage、metadataRating、provider
        │
        ├─ Phase B  search_provider_titles(actorName | movieId)
        │            Metatube SearchMovie* → 番号/标题/源站/评分
        │            与本地 code 对账 → inLibrary / movieId?
        │
        └─ Phase C  get_source_page(url)
                     url ∈ 本轮已知源站 URL
                     抽文本、限体积、wrapSource
```

### Phase A — 暴露已有锚点（最小、无新出站）

扩展 `movieDetailCard` / `get_actor_profile` 返回：

- 影片：`homepage`、`metadataRating`、`metadataProvider`（已刮才有）
- 演员：刮削 `homepage`（与用户 `externalLinks` 分开）

为此需让详情查询选出 `movies.homepage`（今日 SELECT 未包含该列）。不必改资料库海报 UI。

系统提示补一句：评价与介绍优先用这些字段和后续源站工具，禁止编造库外番号为本地 id。

### Phase B — `search_provider_titles`（库外作品的主路径）

**参数**（至少其一，且必须能锚定到本轮已检索实体）：

- `actorName`：来自 `list_actors` / `get_actor_profile` / 影片 `actors`
- `movieId`：来自本轮 `search_movies` / `get_movie_detail` / `get_watch_history`（与 `present_movies` 同一 MovieRef 约束）
- `limit`：默认 15，最大 25

`movieId` 时后端用该片 `code` + 主演名拼检索词（优先演员名搜片单，番号用于去重当前片）。禁止自由 `query` 字符串，避免滑向通用搜索。

**返回**（`wrapSource`）：

```json
{
  "query": { "actorName": "…", "code": "ABC-123" },
  "items": [
    {
      "code": "ABC-124",
      "title": "…",
      "provider": "javbus",
      "homepage": "https://…",
      "score": 4.2,
      "inLibrary": true,
      "movieId": "local-id-if-any"
    }
  ],
  "truncated": false
}
```

`inLibrary=false` 时省略 `movieId`。助手可用文本列出库外作品；仅 `inLibrary=true` 的 id 可进 `present_movies`。

**底层**：在 `scraper.Service` 增加有界检索（封装现有 `SearchMovie` / `SearchMovieAll`，按演员名或番号），`internal/app` 对账 `movies.code`（现有番号规范化）。出站走刮削同一代理与超时。不在 Agent 循环里触发完整 `scrape.movie` 任务。

### Phase C — `get_source_page`（长评 / 长介绍，可选）

**参数**：`url`（https，本轮允许名单内）。

允许名单（会话内、当轮工具结果并集）：

- 本轮 `get_movie_detail.homepage`
- 本轮 `get_actor_profile.homepage` 与 `externalLinks`
- 本轮 `search_provider_titles[].homepage`

额外硬规则：只允许 https；拒绝 loopback / 链路本地 / 私有网段；不跟随跳到未在名单中的 host；响应上限约 32 KiB 抽出文本；去脚本与标签；超时 10s；失败返回工具错误。出站走 `proxyenv`。

Phase C 可在 A+B 验证后再做。没有它，站点评分 + 源站 URL + 库外番号列表已经能回答大部分「评价 / 还拍过什么」。

## 5. 前端与 Mock

- `src/lib/agent-tool-labels.ts` 增加产品化名称（如「查找源站作品」「阅读源站页面」），过程条可折叠。
- Mock adapter 对 `search_provider_titles` 返回若干假番号（部分 `inLibrary`），对 `get_source_page` 返回短假正文。
- 不新增独立卡片组件。实验开关与 PIN 保护不变。

## 6. 验收要点（写入 REQ-0044）

1. 已刮削影片/演员的 Agent 详情含源站 `homepage` 与影片站点评分；未刮削时字段缺省，不编造 URL。
2. `search_provider_titles` 必须带本轮已见的 `movieId` 或 `actorName`；自由文本查询被 schema 拒绝。
3. 返回条目含 `code` / `title` / `inLibrary`；仅库内条目带 `movieId`；`present_movies` 拒绝库外 id。
4. 不存在 `search_web`；天气、新闻无法通过本工具发出。
5. `get_source_page`（若已实现）拒绝允许名单外的 URL 与非 https。
6. 刮削源站失败时工具返回错误摘要，非 AI 功能不受影响。
7. Mock 可独立跑通「还拍过什么」对话；Go 工具与 FakeChatClient 剧本覆盖「无锚点调用被拒」「库外 id 不能 present」。

## 7. 实施任务

1. ~~详情查询选出并映射 `movies.homepage`；扩展 Agent `get_movie_detail` / `get_actor_profile` 返回与单测。~~
2. ~~系统提示：源站周边规则 + 禁止伪造 movieId。~~
3. ~~`scraper.Service` 有界检索接口 + Metatube 实现 + 与本地 code 对账。~~
4. ~~注册 `search_provider_titles`；MovieRef 或等价本轮实体约束；审计走现有网关。~~
5. ~~前端工具文案三语 + Mock 假结果 + 标签单测。~~
6. ~~`get_source_page`：允许名单、SSRF 护栏、HTML 抽文本、单测。~~
7. 同步 `project-facts.mdc` / 工具清单叙述（落地后）；本提案改为 `implemented`。

## 8. 风险

- 成人源站 HTML 结构不稳定，Phase C 抽取质量参差 → 先上 A+B。
- 按演员名搜片可能夹杂同名或广告条目 → 返回 `provider` + `code`，提示词要求注明不确定项。
- 源站超时与 P-10 10s 冲突 → 单次检索、截断列表，不在循环内重试整链刮削。
- 云端 LLM 可能拒绝成人页面正文 → Phase C 默认仍走用户已选 provider；失败当工具错误，不改脱敏策略。

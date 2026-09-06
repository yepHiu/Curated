# Agent 对话中的影片卡（E2 补片）

日期：2026-08-20
状态：implemented
上游：[`2026-08-19-agent-user-prd.md`](2026-08-19-agent-user-prd.md) REQ-0040 · [`2026-08-19-agent-milestone-plan.md`](2026-08-19-agent-milestone-plan.md) E2

## 问题

E2 已经能用 `search_movies` / `get_movie_detail` 查到真实影片，但 Agent Window 只把工具结果收成一行截断 JSON。会话式选片（REQ-0040）要的是「3 张带理由的候选卡，可点进详情」，这一层还没做。

资料库影片卡是 Curated 的识别符号。对话里如果只剩纯文字番号，选片体验会掉一截。

## 结论（先拍板）

**不要再做一个查库工具。** 数据源继续是现有只读查询。  
**要在对话 UI 上做一层「展示协议」。** 推荐卡片挂在助手回复下面，而不是塞进工具调试卡。

REQ-0040 原文写「无新工具、前端新增推荐卡片渲染单元」。那句话要防的是再造一套搜索 API。下面的 `present_movies` 若落地，性质是 **UI 投影**，不是查询域工具：参数里的 `movieId` 必须来自本轮已经检索到的 ID。

## 为什么不能只改工具卡

```
用户：今晚想看轻松的、一小时内、没看过的
        │
        ▼
 search_movies  →  内部工作集（可能 20 部）
 get_insights_* →  口味依据
        │
        ▼
 助手要展示的只有 3 部 + 一句理由
```

工具结果 = 模型的工作集。助手回复 = 给用户看的选片板。两者不能混成一张卡，否则 20 部海报会把对话撑爆，用户也分不清哪几部是推荐。

也不要让模型在正文里写 `[[movie:id]]` 或 Markdown 小组件：容易编造 ID，和系统提示「Never invent movie IDs」对着干。

## 推荐形态

1. **查询工具照旧**（`search_movies`、`get_movie_detail`、`get_watch_history`）。
2. **新增一个 UI 投影工具** `present_movies`（或同名）：
   - 不打 SQLite；
   - 参数：`items: [{ movieId, reason? }]`，上限 6（推荐场景默认 3）；
   - 网关校验：`movieId` 必须出现在本会话本轮（或最近一步）查询结果里，否则拒绝；
   - 循环发出单独 SSE，例如 `movie_cards`，前端把卡片贴在随后的助手气泡下。
3. **卡片视觉**：不要复用资料库竖版 `MovieCard`（收藏、批量、双击进播放器对窄浮窗太重）。对齐 `PlayerPlaylistCard` / `PlaybackHistoryCard` 的横条：左海报、右标题 / 番号 / 演员，可选一行理由。点击进详情，浮窗保持打开（页面上下文已有 `movieId`，「这部」能接上）。播放作为次要动作，不要双击语义。
4. **工具卡继续当痕迹**：可折叠的「已搜索资料库」，不要在那里铺海报墙。

本机 loopback 下海报仍走 `/api/library/movies/{id}/asset/cover`。若开发后端不在 Vite 默认代理的 8080，相对路径会 502；对话卡片会碰到同一条路径问题，需要与播放 URL 一样跟 `VITE_API_BASE_URL`。

## 不做（本期）

- 在对话里嵌完整资料库网格 / 虚拟滚动；
- 演员卡、萃取帧卡（同一协议以后可复用 `present_actors`）；
- 把推荐结果写进首页每日推荐或资料库；
- 用 markdown 围栏当 UI 协议。

## 验收

- 「今晚看什么」给出最多 3～6 张真实影片卡，理由只引用已检索字段；
- 点击进详情，Agent Window 不关；
- 未检索到的 ID 不能出卡；
- 工具调试卡仍在，但不替代推荐板；
- Mock 模式有假卡片；三语、dark、触控高度照旧。

# Agent E3.1：笔记 Action 与确认写

日期：2026-08-21
状态：implemented（E3 首刀；清洗简介 / Insights 解读 / 一句话视图仍未做）
上游：[`2026-08-19-agent-milestone-plan.md`](2026-08-19-agent-milestone-plan.md) E3 · REQ-0029

## 范围

- 写路径唯一入口：`save_movie_comment` 的 write-preview → 用户确认 → apply。
- Action 通道：`POST /api/ai/actions/polish_comment`（模型自识别原文语言，保持同语言润色）。
- 确认：`POST /api/ai/confirm`。Chat SSE 增加 `confirm_required` 后挂起循环。
- 详情页笔记按钮与 Agent Window 确认卡均受实验开关 gating。
- 模型不能在 chat 通道自行 apply（总开关默认关）。

## 不做

- 笔记 **扩写** / **翻译**（2026-08-21 产品确认：REQ-0029 只做润色；简介/标题翻译是 REQ-0030 / REQ-0041）。
- `clean_summary` / MCP（本切片当时；标题翻译、Insights、Saved View 已在 E3.2–E3.4）。
- 正式 Settings AI 分区与写权限总开关 UI（仍默认仅 preview；人点确认走 action 通道 apply）。

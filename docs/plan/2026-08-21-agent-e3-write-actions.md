# Agent E3.2–E3.4：展示覆盖、洞察解读、一句话视图

日期：2026-08-21
状态：implemented
上游：[`2026-08-19-agent-milestone-plan.md`](2026-08-19-agent-milestone-plan.md) E3 · REQ-0030 / REQ-0041 / REQ-0031 / REQ-0033

## 范围

- 写工具 `update_movie_display_overrides`：只写 `user_title` / `user_summary`，确认前零写入。
- Action `translate_summary`、`translate_title`：详情编辑对话框输入框内的独立 ghost 按钮，各自只改简介或标题，走 preview→confirm。
- Action `insights_narrative`：Insights 页只读解读，经查询工具取真实聚合后再生成文本，无确认卡。
- 写工具 `create_saved_view`：Agent Window 把自然语言解析成 Saved View v1 筛选，确认卡展示 name + JSON filters 后再创建。缺 `schemaVersion`、片长分钟数、多余导航字段会在入网关前规范化。瞬时导航字段不进 schema。

## 不做

- 批量清洗/翻译、MCP、正式 Settings AI 分区、写权限总开关 UI。
- 精确到分钟的片长筛选（现有 runtime 桶仍是 short/standard/long）。

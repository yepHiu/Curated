# Curated 规划文档状态约定

`docs/plan/` 保存研究、方案、路线图与实施计划，但“存在一个文件”不等于该方向已经获批或正在执行。当前需求状态的唯一台账是 `docs/prd/requirements.csv`；当前实现事实以源码、测试和 `.cursor/rules/project-facts.mdc` 为准。

## 状态值

新建或实质更新规划文档时，应在标题后的前几行写入 `状态：<value>`，使用以下值之一：

| 状态 | 含义 |
|---|---|
| `research` | 调研材料，尚未形成实施建议 |
| `proposed` | 已形成方案，但尚未批准进入需求台账 |
| `approved` | 已批准，等待排期或依赖 |
| `in-progress` | 已进入实现；必须有关联的 PRD requirement |
| `verified` | 实现与验收证据齐全；PRD 状态至少为 `verified` |
| `superseded` | 已被新文档或新决策取代；必须注明替代来源 |

历史文档不要求一次性机械补齐状态；在重新采用、更新或引用它作为当前路线时必须补齐。没有状态的旧文档默认视为历史资料，不能据此声称功能已批准或正在开发。

## 当前主计划

质量审计主计划是 [`2026-07-19-project-feature-quality-audit.md`](2026-07-19-project-feature-quality-audit.md)。Milestone A～C 已验证；Milestone D（REQ-0019～REQ-0022）已随 1.4.12 实现，真实 Chromium QA 仍待补。1.5.1 之后当前选定的下一执行入口是 [`2026-08-14-scrape-governance-implementation-plan.md`](2026-08-14-scrape-governance-implementation-plan.md)，对应 REQ-0023。消息中心改版 REQ-0024 与消息政策台账 REQ-0025 已开始落地（[`2026-08-14-message-center-repair-plan.md`](2026-08-14-message-center-repair-plan.md)、[`../prd/message-catalog.md`](../prd/message-catalog.md)），不抢刮削调度实现。资料库筛选强化 REQ-0026 见 [`2026-08-14-library-filter-strengthening.md`](2026-08-14-library-filter-strengthening.md)。SFW 合规产品已决定另建新项目，不在本仓库改造；容器、WebDAV、漫画库、Android 与 metadata sidecar 在未进入 PRD 前不并行实施。

## 状态流转

1. `research` / `proposed` 文档可以只记录问题、证据和备选方案。
2. 决定实施时，在 `requirements.csv` 新增或更新稳定 `REQ-xxxx`，再把文档标为 `approved` 或 `in-progress`。
3. `implemented` 不等于 `verified`；PRD 必须保留实现引用和测试引用。
4. 只有验收标准均有权威证据时，文档与 PRD 才能标为 `verified`。
5. 方向被替换时保留原文，标为 `superseded` 并链接替代文档，不删除决策历史。

## 最小文档头示例

```markdown
# 标题

日期：2026-07-20
状态：proposed
关联需求：REQ-xxxx
```

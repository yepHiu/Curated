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
| `paused` | 暂缓推进；必须注明暂停决定及恢复条件，保留已有研究和实现状态 |
| `verified` | 实现与验收证据齐全；PRD 状态至少为 `verified` |
| `superseded` | 已被新文档或新决策取代；必须注明替代来源 |

历史文档不要求一次性机械补齐状态；在重新采用、更新或引用它作为当前路线时必须补齐。没有状态的旧文档默认视为历史资料，不能据此声称功能已批准或正在开发。

## 当前主计划

质量审计主计划是 [`2026-07-19-project-feature-quality-audit.md`](2026-07-19-project-feature-quality-audit.md)。Milestone A～C 已验证；Milestone D（REQ-0019～REQ-0022）已随 1.4.12 实现，真实 Chromium QA 仍待补。1.5.1 之后当前选定的下一执行入口是 [`2026-08-14-scrape-governance-implementation-plan.md`](2026-08-14-scrape-governance-implementation-plan.md)，对应 REQ-0023。消息中心改版 REQ-0024 与消息政策台账 REQ-0025 已开始落地（[`2026-08-14-message-center-repair-plan.md`](2026-08-14-message-center-repair-plan.md)、[`../prd/message-catalog.md`](../prd/message-catalog.md)），不抢刮削调度实现。资料库筛选强化 REQ-0026 见 [`2026-08-14-library-filter-strengthening.md`](2026-08-14-library-filter-strengthening.md)。退役标签浏览页 REQ-0028 见 [`2026-08-16-retire-tags-browse-page.md`](2026-08-16-retire-tags-browse-page.md)（in-progress）。演员空资料自动补刮 REQ-0027 见 [`2026-08-16-actor-missing-profile-auto-scrape.md`](2026-08-16-actor-missing-profile-auto-scrape.md)。SFW 合规产品已决定另建新项目，不在本仓库改造；容器、WebDAV、Android 与 metadata sidecar 在未进入 PRD 前不并行实施。漫画库与写真库已获准作为默认关闭的 Beta 合入主线，对应 REQ-0046 / REQ-0047，见 [`2026-09-10-comic-photo-beta-integration.md`](2026-09-10-comic-photo-beta-integration.md)。

## 合规化当前决定：独立项目推进，本仓库暂缓（2026-10-04）

用户已另开独立项目推进合规化，明确要求本仓库先搁置相关工作。[合规化与元数据源解耦研究](2026-07-18-curated-compliance-and-metadata-source-decoupling.md) 状态改为 `paused`；其中普通影视/成人内容分区、启用向导、能力模块与公开核心中性化等构思暂缓，不作为本仓库当前实施或排期依据。恢复条件为用户明确重新提出在本仓库推进；既有刮削治理等需求继续按各自 PRD 状态处理。

2026-10-03—2026-10-04 的讨论保留在同文第 15 节（普通电影、电视剧兼容与成人内容隔离）、第 15.13 节（开启流程）和第 15.14 节（扩展包导入与加载），供以后参考。当时仅形成构思，没有批准实施、增加 PRD requirement 或改变现有运行行为；本次只更新规划状态与项目记忆。

## 多分片影片与 FC2 独立展示（2026-10-04）

用户确认「同一番号的多个文件」。[多文件影片与 FC2 独立展示](2026-10-04-multipart-movies-and-fc2-library.md) 与 REQ-0056 / REQ-0057 已为 `verified`：独立 FC2 浏览，共用资源；文件选择、数字排序、独立进度、萃取来源和旧库迁移已实现并通过本地验证。尚未生产发布；旧索引丢失路径通过升级后重扫补回。合规化仍暂缓。

## FC2 元数据可靠性（2026-10-06）

[FC2 元数据获取可靠性](2026-10-06-fc2-metadata-reliability.md) 为 `verified`：独立来源、字段补全、访问错误与图片来源上下文，已完成 FC2-3977618 在线只读验证及后端全量检查；需新 Server 编译重启/升级生效，未生产发布。

## Windows 原生播放原型（2026-10-05）

用户已要求在专用分支先制作 Windows 原型。[`2026-10-05-windows-native-player-prototype.md`](2026-10-05-windows-native-player-prototype.md) 为当前实施计划，状态 `in-progress`，关联 REQ-0058。§12 已实现原生视频与 Curated 控件同窗，复用圆形控制栏，并修复透明层几何绑定与整面焦点边框；合成媒体和实际 Electron fixture 已通过。2026-10-06 已按 §14 用户授权实施正式 main 会话/入口/本地专用窗口与记录/侧栏，新增 REQ-0059（in_progress 70），见 §14.9；§13/14 的引擎分发已有开发下载与可选 staging 校验，但完整对应源码/许可及原生安装验收待完成。用户指定 CD1 已完成两引擎各五分钟对照（§14.10），native 零观察掉帧、Chromium 掉帧 27.64%；P3 更多片源、混合 DPI 与完整业务验收仍待完成，主页面内嵌和 macOS 为后续阶段。历史 VLC 对照研究不因此整体进入实施。

## 状态流转

1. `research` / `proposed` 文档可以只记录问题、证据和备选方案。
2. 决定实施时，在 `requirements.csv` 新增或更新稳定 `REQ-xxxx`，再把文档标为 `approved` 或 `in-progress`。
3. `implemented` 不等于 `verified`；PRD 必须保留实现引用和测试引用。
4. 只有验收标准均有权威证据时，文档与 PRD 才能标为 `verified`。
5. 方向被替换时保留原文，标为 `superseded` 并链接替代文档，不删除决策历史。
6. 暂缓时标为 `paused`，记录决定日期、范围与恢复条件；既有 PRD 的实现或验收状态不因此被改写。

## 最小文档头示例

```markdown
# 标题

日期：2026-07-20
状态：proposed
关联需求：REQ-xxxx
```

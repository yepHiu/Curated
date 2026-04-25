# Curated 文档目录说明

本目录为**仓库内**说明与规划类材料；**对外公开 API** 以仓库根目录 `API.md` 为准，**产品入口说明**以根目录 `README.md`（及 `README.zh-CN.md` / `README.ja-JP.md`）为准。

## 一级子目录

| 目录 | 内容 |
|------|------|
| **`reference/`** | 长期有效的规范与事实：前后端与 UI 规范、合约约束、**项目记忆**（`2026-03-20-project-memory`）、`library-config` 与资料库整理说明、**实现与功能对照** `architecture-and-implementation.html` 等 |
| **`product/`** | 产品设计/域设计长文（如主设计文档、演员库设计等） |
| **`ops/`** | 研发与发版向：Agent 构建与测试说明、**打包历史 CSV** 与相关 Markdown、CI/CD 与双库/假库/转型类方案文档等 |
| **`review/`** | 一次性架构评审、代码/前端质量与性能类审计（历史快照性质） |
| **`plan/`** | 排期与实施计划、分阶段技术方案（持续更新同主题时优先**改**旧文） |
| **`prd/`** | 需求表与 PRD 材料（如 `requirements.csv`） |
| **`release-notes/`** | 发版说明 |

`docs/film-scanner/` 若存在，为**参考/实验**材料，不视为生产运行时代码树的一部分。

## 新文档放哪里

- 实施计划、里程碑、技术落地清单 → `docs/plan/`
- 可长期当「规范/真相」引用的内容 → `docs/reference/`
- 产品愿景与域模型、与具体 sprint 无强绑定的长文 → `docs/product/`
- 构建、发布台账、环境/CI、运维向说明 → `docs/ops/`
- 仅反映某次评审结论文档 → `docs/review/`

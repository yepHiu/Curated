# Agent / Cursor 说明

**产品正式名称：Curated**（仓库目录 / `package.json` 的 `name` 等可能仍为 `jav-shadcn`）。

本仓库的 **Cursor 规则与项目记忆** 集中在 **`.cursor/rules/*.mdc`**。

| 优先级 | 文件 | 用途 |
|--------|------|------|
| 高 | `workspace-quick-reference.mdc` | 启动命令、代理、`VITE_USE_WEB_API` |
| 高 | `project-facts.mdc` | 前后端目录、API 概览、分层 |
| 高 | `architecture-boundaries.mdc` | 已实现 vs 目标桌面架构 |
| 中 | `project-standards.mdc` | 规则索引入口 |
| 高 | `git-commit-workflow.mdc` | 最小修改单元提交；仅用户明确要求时再 `git push` |
| 按需 | `ui-component-spec.mdc`、`vue-frontend-standards.mdc`、`jav-library-frontend-patterns.mdc` | 前端 UI 与实现 |
| 按需 | `backend-go-standards.mdc`、`backend-api-contracts.mdc` 等 | 后端与合约 |

**UI 设计规范（代码级）**：`docs/frontend-ui-spec.md`。

**规划类文档**：任务规划、实施计划等 Markdown 放在 **`docs/plan/`**（无则创建该目录）。
**新增规则**：凡是输出“方案”“计划”“路线图”“实施建议”等可沉淀内容时，除了在对话中回复，还要同步保存为 **`docs/plan/*.md`** 文档；若同主题已有文档，则优先更新原文档而不是重复创建。

修改架构或新增重要端点后，请同步更新 **`project-facts.mdc`**（必要时 **`workspace-quick-reference.mdc`**）、**`README.md`** 配置与 API 摘要、**`docs/library-organize.md`**（若涉及 **`library-config.cfg`**）、**`CLAUDE.md`** API 列表，以及 **`docs/architecture-and-implementation.html`**（实现说明与功能对照表）。

**Mock / Web 与收藏、评分**：Mock 模式下收藏与用户评分通过 **`localStorage`（`jav-library-movie-prefs`）** 跨页面刷新保留；写入 SQLite 的持久化需 **`VITE_USE_WEB_API=true`** 并运行后端（见 **`workspace-quick-reference.mdc`**）。

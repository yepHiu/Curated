# Agent / Cursor 说明

本仓库的 **Cursor 规则与项目记忆** 集中在 **`.cursor/rules/*.mdc`**。

| 优先级 | 文件 | 用途 |
|--------|------|------|
| 高 | `workspace-quick-reference.mdc` | 启动命令、代理、`VITE_USE_WEB_API` |
| 高 | `project-facts.mdc` | 前后端目录、API 概览、分层 |
| 高 | `architecture-boundaries.mdc` | 已实现 vs 目标桌面架构 |
| 中 | `project-standards.mdc` | 规则索引入口 |
| 按需 | `vue-frontend-standards.mdc`、`jav-library-frontend-patterns.mdc` | 前端实现 |
| 按需 | `backend-go-standards.mdc`、`backend-api-contracts.mdc` 等 | 后端与合约 |

修改架构或新增重要端点后，请同步更新 **`project-facts.mdc`**（必要时 **`workspace-quick-reference.mdc`**）。

**Mock / Web 与收藏、评分**：Mock 模式下收藏与用户评分通过 **`localStorage`（`jav-library-movie-prefs`）** 跨页面刷新保留；写入 SQLite 的持久化需 **`VITE_USE_WEB_API=true`** 并运行后端（见 **`workspace-quick-reference.mdc`**）。

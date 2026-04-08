## Context

- 产品：**Curated**（Vue 3 + TS + Vite + Go + SQLite）。资料库页使用虚拟滚动，但列表数据来自 **全量预取到内存** 的 `moviesState`（Web 适配器），规模上限由 `MAX_MOVIES_PREFETCH` 与分页批大小共同约束。
- 设置页、库页、详情页逻辑随功能迭代增长，部分单文件过长。

## Goals / Non-Goals

**Goals:**

- 为「性能 / 质量 / 体验」三类问题提供**可排序**的改进路径（先测再改、先 P0 再 P1）。
- 将大库列表行为写成可测试的 **spec**，避免口头约定。

**Non-Goals:**

- 在本 change 内确定具体排期或负责人。
- 不强制统一技术选型（例如是否引入服务端分页）——由后续独立 change 决策。

## Decisions

1. **大库性能**：优先**测量**（浏览器 Performance / Network、后端 `list` SQL 与索引、Payload 大小），再在「服务端分页 / 虚拟数据源 / 提高上限」之间选型；避免未测量就重写列表。
2. **质量**：`SettingsPage.vue` 等巨型组件的拆分策略采用 **按路由子区块 / composable / 子组件** 渐进式迁移，与功能开发同 PR 时遵守最小单元提交。
3. **体验**：关键路径（库→详情→播放、设置保存）用**一致反馈**（toast、保存中态、错误可重试）验收，而非一次性大改版。

## Risks / Trade-offs

- **[Risk] 全量列表改为分页** → 筛选/收藏/虚拟滚动与 URL 状态需联动，工作量大；**Mitigation**：分阶段；先加「仅首屏窗口数据」或「可配置上限」再演进。
- **[Risk] 过度拆分组件** → 跳转与 prop 爆炸；**Mitigation**：以领域边界（库路径、代理、播放）为切分单位。
- **[Risk] 仅文档无代码** → 审计被搁置；**Mitigation**：`tasks.md` 中保留可勾选跟进项，由后续 sprint 领取。

## Migration Plan

- 不适用（本 change 为规划与 spec 工件）。

## Open Questions

- 产品是否接受「大库仅索引前 N 条用于浏览」的明确说明（与当前 `MAX_MOVIES_PREFETCH` 行为对齐）？
- 设置页是否需「显式保存」与「自动保存」双模式（部分用户偏好可确认感）？

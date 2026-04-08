## Why

Curated 已具备完整前后端与主要用户路径，但缺少一份**集中、可执行**的优化视图：大库场景下的前端内存与请求策略、超大单文件组件的可维护性、以及浏览→详情→播放→设置链路上的体验摩擦。本变更建立**结构化审计结论**与后续 spec/design 输入，便于按优先级分期落地。

## What Changes

- 新增本 change 下的 **proposal / design / specs / tasks**，记录当前代码事实上的优化空间（非一次性改代码）。
- **性能**：资料库 Web 适配器将影片列表最多预取 **10_000** 条、每批 **500**（`web-library-service.ts`）；超大库时内存与首包时间仍是主要瓶颈；虚拟列表已用于海报墙，但**数据源仍是全量数组**。
- **质量**：`SettingsPage.vue` 等单文件体量极大（三千行级），设置与业务逻辑耦合度高，回归与审阅成本高；前后端契约分散在 `contracts`、`types`、`endpoints`，需持续与实现同步（已有规则，执行上仍可加强）。
- **体验（用户路径）**：
  - **资料库**：首屏等待全量列表拉取、筛选/标签与 URL 状态同步、批量操作与长任务反馈。
  - **详情 / 播放**：刮削与封面的加载态、失败可恢复性；播放器与 HLS/直链 分支的认知成本。
  - **设置**：选项多、分组长，滚动位置与自动保存已部分处理，仍可能产生「是否已保存」的不确定感。
- **不在本 change 内直接修改产品代码**（除非后续单独 propose）。

## Capabilities

### New Capabilities

- `client-library-scale`: 约束大库下浏览器端列表加载与可感性能的预期行为（见 `specs/client-library-scale/spec.md`）。

### Modified Capabilities

- （无）仓库内 `openspec/specs/` 尚无基线 spec；本次为新增能力说明，非对既有 spec 的 delta。

## Impact

- **文档与规划**：仅增加 `openspec/changes/product-health-audit-2026/` 下工件；**不**改变运行时行为。
- **后续工作**：`/opsx:apply` 或手工实现时，按 `tasks.md` 分期；真正代码改动需**单独**最小单元提交（见 `.cursor/rules/git-commit-workflow.mdc`）。

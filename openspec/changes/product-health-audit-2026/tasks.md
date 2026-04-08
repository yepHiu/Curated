## 1. Measurement & backlog

- [ ] 1.1 使用 DevTools 与后端日志，在「1k / 5k / 10k+ 影片」样本下记录资料库首屏时间、内存、`/api/library/movies` 请求次数与 payload 体积
- [ ] 1.2 将本 audit 的 P0/P1 项同步到 `docs/plan/` 下一条主题 backlog（若已有同主题则更新，避免重复文档）

## 2. Performance (candidate implementations — separate PRs)

- [ ] 2.1 Spike：服务端分页或游标 API 与现有 `mode`/`q`/`actor`/`tag` 查询的兼容性评估
- [ ] 2.2 评估降低 `MAX_MOVIES_PREFETCH` 或改为「仅当前筛选结果窗口」的可行性与产品文案
- [ ] 2.3 后端：为列表查询确认索引与 `SELECT` 字段（避免过大 payload）

## 3. Quality

- [ ] 3.1 为 `SettingsPage.vue` 制定分块拆分清单（按 Tab/section 对应子组件），每次迁移保持行为一致
- [ ] 3.2 增加关键路径 Vitest/Go 测试缺口梳理（设置 PATCH、列表边界、播放 descriptor）

## 4. User experience

- [ ] 4.1 资料库：首屏加载与空/错误态文案统一；长任务（扫描/刮削）进度与完成 toast 一致性走查
- [ ] 4.2 详情→播放：断网/代理失败时的可重试提示与返回路径
- [ ] 4.3 设置：自动保存「已保存」反馈与 `settings` 相关错误 toast 走查（含滚动保留）

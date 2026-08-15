# README 短入口与 docs/guide 手册

日期：2026-08-16
状态：verified

## 决策

根目录三份 README 只做公开短入口（产品是什么、怎么跑起来、文档表）。原先堆在 README 里的配置、备份、路径迁移、功能长文和发版说明迁到 [`docs/guide.md`](../guide.md)。该手册同时作为 `docs/` 各篇文章的引用索引。

`API.md` 仍是唯一公开 HTTP API 参考。`docs/README.md` 只说明子目录怎么用，不重复手册正文。

## 约定

- 用户可见行为变了：同步三份短 README 的入口描述，以及 `docs/guide.md` 的操作/索引。
- 不要把操作长文写回根 README。
- 新的跨目录文章优先补进 `docs/guide.md` 第 9 节索引表。
- 旧方案 [`2026-04-12-readme-and-api-docs-redesign.md`](2026-04-12-readme-and-api-docs-redesign.md) 与其实施计划标为 superseded；`API.md` 拆分仍然有效。

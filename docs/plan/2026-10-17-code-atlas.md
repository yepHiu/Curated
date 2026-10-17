# Curated 代码地形图

## 目的与设计

为当前仓库生成可离线打开的单文件 HTML，帮助维护者理解代码规模、目录分布、函数长度和静态依赖。范围是 `jav-shadcn`，不自动合并相邻 Android / 插件仓库。

这是独立的开发观察工具，放在 docs 下，不加入产品导航、不改变正式 Vue 组件与设计令牌。采用深色技术地图、目录面积图、依赖关系图和可搜索排行；主要操作是点击目录下钻。提供键盘访问、空结果、窄屏布局和明确的统计口径。设计参考 ui-ux-pro-max 的可访问性与图表原则；该技能本机未包含 search.py，采用已读指南直接实施。

## 实施范围

- 读取 Git 已跟踪及未忽略的新文件，统计当前磁盘内容；排除依赖、运行数据、构建产物及报告自身。
- 文件按源码、测试、文档、配置与资源分类。源码物理行数包含注释/空行；不冒充有效代码行数。
- TypeScript / JavaScript / Vue script 与 Go 使用语法树识别函数（含匿名回调），显示位置、跨度和分支指标。其他语言先展示文件体量，不伪造函数覆盖。
- 分支指标为 1 + 条件/循环/case/catch/逻辑短路等语法分支，跳过嵌套函数；它是阅读热点线索，不是完整控制流或质量评分。
- TypeScript / JavaScript 本地导入解析到文件；Go 本地 import 解析到包。聚合为模块关系，明确不代表 HTTP / IPC / 运行时函数调用图。
- 输出报告与可重复运行的生成命令，不依赖外部 CDN。

## 验证

2026-10-18 完成：

- `node --test scripts/analysis/code-atlas.test.mjs`：6 项通过。覆盖注释/字符串不产生分支、嵌套函数隔离、Vue 原始行号、解析失败、回调标签不包含调用参数、本地导入解析及 Go 方法/闭包/select/case。
- `pnpm code:atlas`：真实仓库生成成功，解析失败 0，未解析本地导入 0。
- Playwright CLI Chromium：17 项断言通过，覆盖目录下钻、函数面积、分支排序、分页、函数名搜索、文件详情及原始行号、Escape 关闭、空搜索、测试范围、依赖图、模块聚焦、文件清单、断网后筛选、375px 与 812px 横屏无页面溢出。
- 目视检查桌面地图、依赖图和移动布局。移动表格局部横向滚动，保持路径可读。最终页面无浏览器控制台错误。
- 截图保存在 ignored `output/playwright/code-atlas-*.png`；临时验收脚本放在 ignored `.workspace/`。没有运行产品全量构建/测试；本工具不修改产品运行代码。

## 交付

- `docs/reference/code-atlas.html`：数据与样式/脚本内嵌的 HTML 快照，报告自身不参与统计。
- `scripts/analysis/code-atlas.mjs`、`go-atlas.go`：统计生成器；`code-atlas.template.html`：可维护的页面模板。
- `pnpm code:atlas`：仓库根目录更新命令，需要现有 Node 依赖与 Go。
- `docs/guide.md`：使用入口与统计口径。未新增业务 API、数据迁移或产品设计系统规则。

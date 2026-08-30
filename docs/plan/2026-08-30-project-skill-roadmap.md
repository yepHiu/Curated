# Curated 项目 Skill 建议路线图

日期：2026-08-30
状态：全部路线图 Skill 已创建

## 结论

Curated 目前已经有 `curated-dev-start`、`curated-backend-runtime`、`curated-packaging`、`prd-csv` 与 OpenSpec 工作流 Skill。下一批不应重复封装启动或发布命令，而应固化最容易遗漏的跨层一致性、SQLite 安全操作与验证闭环。

## 优先级建议

| 优先级 | 建议 Skill | 触发场景 | 应固化的能力 | 价值 |
| --- | --- | --- | --- | --- |
| 已落地 | `curated-feature-slice` | 新增或修改用户可见功能 | 契约优先；Web 与 Mock adapter、服务接口、视图/组件、i18n、单元测试、必要 E2E、文档同步的完整检查单 | 防止“后端已做、Mock/前端/文档漏做”的跨层漂移 |
| 已落地 | `curated-api-contract-change` | 增加或修改 `/api`、DTO、错误码、任务接口 | 定位 Go contracts/handler/service、前端 service contract、Web/Mock adapter、`API.md`/`CLAUDE.md`/project facts；生成影响清单并执行针对性测试 | 保持 HTTP、前端抽象与未来 IPC/stdio 的可演进性 |
| 已落地 | `curated-sqlite-migration` | 新增 migration、数据修复或表结构调整 | 编号冲突检查、迁移幂等性/外键/回滚思路、repository 与测试改动、`foreign_key_check`、真实升级路径的验证建议 | 保护用户长期数据，避免一次迁移破坏本地资料库 |
| 已落地 | `curated-maintenance-safety` | backup、restore、path migration、Library Health 修复/清理 | 只读预检优先、运行时锁、备份目标不可覆盖、显式确认、受控删除范围、审计记录与复核输出 | 把高风险运维操作变成可靠且可审计的流程 |
| 已落地 | `curated-ui-governance` | 新页面/面板、壳层/布局、共享组件/令牌、交互范式或 UI 评审 | 在编码前建立产品面、主任务、信息层级、状态矩阵和系统影响；以 UI 宪法约束 token、组件分层、密度与可访问性；必要时回写规范 | 让 UI 从“实现时检查”前移为“设计时治理”，逐步形成统一设计语言 |
| 已落地 | `curated-ui-acceptance` | 修改 Vue 页面、组件、响应式布局或交互状态 | 依据 UI spec 检查 token、可访问性、空/加载/错误态、键盘操作、i18n、Mock/Web 两种状态；默认运行轻量测试，display suite 仅在获授权时运行 | 降低视觉回归与“只有正常态可用”的问题；初期可作为 `curated-ui-governance` 的验证阶段 |
| 已落地 | `curated-e2e-scenario` | 新功能需要浏览器级验证、回归 bug | 将自然语言验收标准转换为 Mock 与 Web API stub 独立场景；选择现有 Playwright 配置、定位稳定断言与非脆弱选择器 | 让 E2E 覆盖真实用户路径，而不是重复单测 |
| 已落地 | `curated-localization-message` | 新增用户文案、错误码、通知中心消息 | 同步 `en`/`zh-CN`/`ja` locale，核查插值、语气与命名空间；若进入消息中心，同步 message catalog 并运行对应 lint | 防止语言缺漏及前端文案、后端错误码、消息台账失联 |
| 已落地 | `curated-doc-sync` | 修改重要端点、架构、配置或用户可见行为 | 根据影响范围精确列出并更新 README 短入口、guide、API、CLAUDE、project facts、架构 HTML/资料库配置说明 | 将当前分散的文档同步规则转成可执行的最小清单 |
| 已落地 | `curated-security-boundary-review` | 认证、CORS/LAN、文件路径、上传、FFmpeg、Electron preload 相关改动 | 威胁模型小检查：来源与路径校验、权限/确认、敏感日志、错误泄露、CORS/Host、preload 最小暴露；给出风险与回归测试 | 适合本地优先、可选 LAN 与媒体文件读写的安全边界 |
| 已落地 | `curated-release-readiness` | 准备正式版本发布 | 在既有 `curated-packaging` 之前做只读门禁：工作区状态、版本基线、变更日志、必跑测试、FFmpeg/runtime 前置条件、签名/产物核查表 | 补齐发布前判断；不替代已有打包 Skill |
| 已落地 | `curated-regression-selector` | 小改动后不知道该跑哪些测试 | 按改动文件和领域映射到最小必要的 Vitest、Electron、Playwright、Go test/vet、build 命令；说明未覆盖风险 | 减少全量测试成本，同时让验证选择有根据 |

## 推荐落地顺序

1. 先做 `curated-ui-governance` 与 `curated-feature-slice`：前者在 UI 编码前锁定产品语言，后者保证跨层实现不遗漏。
2. 接着做 `curated-sqlite-migration` 与 `curated-maintenance-safety`：两者直接保护用户数据，且触发频率不高、最值得用硬护栏约束。
3. 再做 `curated-api-contract-change`、`curated-localization-message`：把跨端和产品表面的一致性拆成可维护的专项流程。
4. 最后按团队痛点补独立 `e2e`、文档、安全和发布门禁类 Skill。

## 设计边界

- Skill 应编排现有脚本和规则，不复制它们。例如发布继续调用 `curated-packaging`，开发环境继续调用 `curated-dev-start`。
- 每个 Skill 都应包含：适用触发语、读取范围、不可跨越的安全边界、最小验证命令、完成时需要同步的文档/台账。
- 不要把所有规则塞进一个“Curated 全能开发”Skill；它会过长且难以稳定触发。以变更类型拆分更可靠。
- `curated-doc-sync` 与 `curated-regression-selector` 可以先作为 `curated-feature-slice` 的内部章节；只有重复使用后再拆出独立 Skill。

## `curated-feature-slice` 已实现的最小骨架

该 Skill 在 UI 治理入口之后落地，采用以下结构：

1. 明确触发条件：新增、扩展或修改用户可见功能。
2. 读取规则：workspace quick reference、project facts、相应前端/后端规则、构建测试范式。
3. 影响面盘点：contracts → service interface → Web/Mock adapters → UI/composable → locale/message → tests → docs。
4. 实现硬约束：组件不能直连具体 adapter；长任务使用 task/progress；Mock 行为保持可用且有意地区分无法模拟的能力。
5. 验证选择：按改动范围选择 pnpm、Playwright、Go 和 Electron 检查；不擅自运行昂贵 display suite。
6. 收尾输出：变更矩阵、运行过的验证、未运行项和原因、同步过的文档。

## 暂不建议新建

- `curated-dev-start`、`curated-backend-runtime`、`curated-packaging`、`prd-csv`：仓库已有等价 Skill，应完善现有版本而不是重复创建。
- 泛化的 Git、Vue、Go、代码审查 Skill：已有 Cursor rules 已覆盖基础约束；除非出现稳定的项目特有检查脚本，否则收益有限。

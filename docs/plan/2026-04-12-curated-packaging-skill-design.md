# Curated Packaging Skill Design

## 1. Goal

在当前仓库内新增一个仅供本仓库使用的打包 skill，让 agent 能通过自然语言理解以下发布需求：

- 打生产包
- 打整机包
- 打安装包
- 只打安装包
- 打便携包
- 只打便携包
- 预览这次打包版本
- 把 minor 升到 2 再打包
- 把 major 升到 2 再打生产包

该 skill 的核心要求是：

- 只在当前仓库使用，不做全局安装
- 支持自然语言直接触发
- 默认先预览，再执行
- 预览时明确说明将执行什么、预计版本号是什么
- 执行时复用仓库现有发布脚本，不重新实现发布链路

## 2. Scope

本 skill 覆盖以下模式：

- `publish`
  - 整机发布，生成便携包 + 安装包 + manifest
- `installer`
  - 只生成安装包
- `portable`
  - 只生成便携包
- `preview`
  - 只预览预计版本和即将执行的动作，不真正打包
- `set-base`
  - 只调整 `major.minor.0` 版本基线，不打包

本 skill 不负责：

- 改造现有安装包内容
- 改造现有发布产物结构
- 改变现有版本规则之外的运行时版本展示逻辑
- 做全局 skill 安装或发布到外部 skill 市场

## 3. Repository Placement

该 skill 放置在仓库内：

```text
.cursor/skills/curated-packaging/
```

原因：

- 用户明确要求该 skill 仅在当前仓库使用
- 避免污染全局 skill 目录
- 方便和仓库内发布脚本、规则文档一起演进

## 4. Triggering Model

skill 以自然语言触发为主，不要求用户显式提到 skill 名称。

触发样例：

- “打生产包”
- “打整机包”
- “只打安装包”
- “只打便携包”
- “预览这次打包版本”
- “把 minor 升到 2 再打安装包”
- “把 major 升到 2 再打生产包”

skill 在触发后需要先把请求映射为内部模式：

- `publish`
- `installer`
- `portable`
- `preview`
- `set-base`

## 5. Release Rules The Skill Must Respect

该 skill 必须完全服从仓库当前已确认的生产包版本规则：

- 生产包版本的唯一自动化来源是 `scripts/release/version.json`
- 当前版本基线为 `1.1.0`
- `pnpm release:portable`、`pnpm release:installer`、`pnpm release:publish` 在未显式传入 `-Version` 时都会自动执行 `patch + 1`
- `major` / `minor` 只允许人工调整，并在调整时把 `patch` 重置为 `0`
- `pnpm release:publish` 只允许在入口处分配一次新版本，再把同一个版本传给便携包、安装包和 manifest
- 打包台账写入 `docs/2026-04-02-package-build-history.md`

skill 不得绕开这些规则自行计算并写入版本文件；它只能预览、调用现有命令并验证结果。

## 6. Interaction Flow

### 6.1 Two-Stage Flow

该 skill 固定采用“两阶段”流程：

1. 先预览
2. 再执行

### 6.2 Preview Requirements

预览阶段必须明确输出：

- 本次请求识别出的模式
- 当前版本基线
- 预计使用的版本号
- 该预计版本号为什么会是这个值
- 将执行哪些命令
- 将生成哪些产物
- 是否会写入发布台账
- 是否会自动 bump patch
- 如果包含 `set-base`，则先显示将如何调整基线，再显示打包预计版本

预览阶段应使用一致的结构化格式，便于用户确认。

推荐字段：

- `mode`
- `currentBaseVersion`
- `predictedVersion`
- `commands`
- `artifacts`
- `willWriteHistory`
- `willBumpPatch`
- `baseChange`

### 6.3 Execution Requirements

执行阶段只在预览输出之后进行。

执行时：

- `publish` 调用 `pnpm release:publish`
- `installer` 调用 `pnpm release:installer`
- `portable` 调用 `pnpm release:portable`
- `set-base` 调用 `pnpm release:version:set-base -- --Major <major> --Minor <minor>`

如果请求是“先升 minor / major 再打包”，执行顺序必须是：

1. 先调整基线
2. 再执行对应打包动作

## 7. Expected Preview Semantics

示例：当前基线为 `1.1.0`

### 7.1 只打安装包

预览必须说明：

- 模式：`installer`
- 当前基线：`1.1.0`
- 预计版本：`1.1.1`
- 将调用安装包打包命令
- 将写入打包台账

### 7.2 只打便携包

预览必须说明：

- 模式：`portable`
- 当前基线：`1.1.0`
- 预计版本：`1.1.1`
- 将调用便携包打包命令
- 将写入打包台账

### 7.3 打生产包

预览必须说明：

- 模式：`publish`
- 当前基线：`1.1.0`
- 预计版本：`1.1.1`
- 整机发布只会分配一次版本，不会让安装包和便携包分别再 bump 一次
- 将生成便携包、安装包和 manifest
- 将写入打包台账

### 7.4 把 minor 升到 2 再打生产包

预览必须说明：

- 先执行 `set-base`，把基线调整到 `1.2.0`
- 再执行 `publish`
- 本次整机发布预计版本为 `1.2.1`

## 8. Skill File Layout

建议结构如下：

```text
.cursor/skills/curated-packaging/
  SKILL.md
  scripts/
    preview-package.ps1
    execute-package.ps1
```

### 8.1 `SKILL.md`

职责：

- 描述 skill 的使用场景与触发条件
- 规定自然语言如何映射到内部模式
- 强制“先预览，再执行”
- 强制遵守仓库现有生产包版本规则
- 规定执行后必须检查产物和台账

### 8.2 `scripts/preview-package.ps1`

职责：

- 读取 `scripts/release/version.json`
- 结合请求模式计算预计版本
- 生成统一的结构化预览信息
- 不做真正打包

该脚本不直接改写生产包版本文件；它只做预览和说明。

### 8.3 `scripts/execute-package.ps1`

职责：

- 根据模式调用现有打包命令
- 支持 `publish`、`installer`、`portable`、`set-base`
- 复用仓库已有脚本与 npm scripts
- 不重写现有发布链路

## 9. Why This Structure

采用“skill 文档 + 两个辅助脚本”的原因：

- `SKILL.md` 保持简洁，专注触发条件和工作流
- 预览和执行职责分离
- 避免把版本推导逻辑写死在文档里
- 避免对现有发布脚本做过度重构
- 后续如果新增“只生成 manifest”或“只预览不执行”等能力，扩展成本较低

## 10. Validation Plan

### 10.1 Skill-Level Scenarios

至少验证以下自然语言场景：

- “打生产包”
- “只打安装包”
- “只打便携包”
- “把 minor 升到 2 再打生产包”

### 10.2 Script-Level Scenarios

至少验证以下行为：

- 从 `1.1.0` 预览出 `1.1.1`
- 先切到 `1.2.0` 再预览出 `1.2.1`
- `publish` 只分配一次版本，不重复 bump

## 11. Implementation Notes

后续实现阶段应优先复用现有能力：

- 生产包版本源：`scripts/release/version.json`
- 版本分配逻辑：`scripts/release/versioning.mjs`
- PowerShell 公共逻辑：`scripts/release/release-common.ps1`
- 现有打包命令：`pnpm release:portable`、`pnpm release:installer`、`pnpm release:publish`
- 手动版本基线调整：`pnpm release:version:set-base`

skill 的预览脚本应尽量调用现有版本工具，而不是复制一套平行规则。

## 12. Final Decision Summary

本设计的最终结论：

- 使用仓库内私有 skill，而不是全局 skill
- 自然语言直接触发
- 默认“先预览，再执行”
- 预览必须说明将执行什么、预计版本号是什么
- 整机发布使用 `publish` 模式，且只能分配一次版本
- 通过 skill 文档配合两个轻量脚本落地

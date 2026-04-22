# PRD CSV Skill Proposal

## 背景

当前痛点不是“缺少一个写 PRD 的地方”，而是零散需求进入项目后缺少稳定的整理、去重、澄清、排期、实现追踪和回看机制。一个轻量的 PRD CSV 配合 Codex skill 很适合解决这个问题，因为它能同时满足：

- 人可以直接在表格里编辑。
- Codex 可以稳定读写、排序、筛选和补全。
- Git 可以记录每次需求变化。
- 需求可以和代码提交、测试、文档更新建立关联。

## 推荐方向

推荐采用“CSV 总账 + 必要时补充 Markdown 详情”的方式。

CSV 负责结构化字段和状态流转，是唯一总索引；Markdown 只用于复杂需求的详细说明、验收清单、设计草图或决策记录。这样既不会把 CSV 变成很难维护的超长文本，也不会让 PRD 分散到一堆无法追踪的文档里。

## 可选方案

### 方案 A：单一 PRD CSV

所有需求都在一个 CSV 中管理，例如 `docs/prd/requirements.csv`。

优点是最简单，Codex 容易维护，适合早期快速启动。缺点是复杂需求的背景、边界、验收标准会让单元格变长，长期可读性会下降。

### 方案 B：PRD CSV + 详情 Markdown

CSV 中每行是一条需求，复杂需求通过 `detail_doc` 字段链接到 `docs/prd/details/REQ-xxxx.md`。

优点是兼顾表格管理和深度说明，适合持续演进的产品项目。缺点是需要约定什么时候必须创建详情文档。

### 方案 C：外部表格系统 + 仓库镜像

用飞书多维表格、Notion、Airtable 等作为主 PRD，再定期导出或同步到仓库。

优点是表格体验更好，适合多人协作。缺点是自动化和 Git 追踪会复杂一些，也更依赖外部服务。

## 建议选择

建议先采用方案 B，但落地时按方案 A 的复杂度启动：

- 第一阶段只创建一个 `docs/prd/requirements.csv`。
- 只有当单条需求超过 CSV 可读范围时，再创建 `docs/prd/details/REQ-xxxx.md`。
- Codex skill 先只负责 CSV 的规范读写、润色、拆分、状态更新和一致性检查。
- 等流程稳定后，再考虑接入飞书多维表格或自动生成看板。

## CSV 字段建议

建议第一版字段不要太多，但要覆盖“是什么、为什么、做到哪了、如何验收、和代码如何对应”。

```csv
id,title,type,area,priority,status,progress,source,problem,proposal,acceptance_criteria,dependencies,owner,target_version,implementation_refs,test_refs,detail_doc,updated_at,notes
```

字段说明：

- `id`：稳定需求 ID，例如 `REQ-0001`，不要复用，不要因标题变化而变化。
- `title`：一句话需求标题，尽量可搜索。
- `type`：`feature`、`bug`、`ux`、`refactor`、`docs`、`ops` 等。
- `area`：模块范围，例如 `homepage`、`player`、`settings`、`backend-api`。
- `priority`：`P0`、`P1`、`P2`、`P3`。
- `status`：需求状态，建议使用固定枚举。
- `progress`：开发进度，可以是 `0` 到 `100`，也可以用 `not-started`、`in-progress`、`blocked` 等语义值；推荐第一版用数字。
- `source`：来源，例如用户口述、bug 复盘、review、release note。
- `problem`：用户痛点或业务问题。
- `proposal`：当前解决方案摘要。
- `acceptance_criteria`：验收标准，建议用分号分隔多条短句。
- `dependencies`：依赖的需求 ID、接口、外部条件。
- `owner`：负责人，个人项目可先填 `user` 或 `codex`。
- `target_version`：目标版本或里程碑，例如 `1.3.0`。
- `implementation_refs`：相关提交、PR、文件或计划文档。
- `test_refs`：相关测试命令、测试文件、验证记录。
- `detail_doc`：复杂需求的详情文档路径。
- `updated_at`：最后更新时间，使用 `YYYY-MM-DD`。
- `notes`：临时备注，避免污染主字段。

## 状态流转建议

建议状态使用固定枚举，避免后面无法统计。

```text
idea -> triaged -> specified -> planned -> in_progress -> implemented -> verified -> released
```

异常状态：

```text
blocked
deferred
rejected
superseded
```

每个状态的含义：

- `idea`：刚记录，还没有整理。
- `triaged`：确认值得保留，已初步归类和定优先级。
- `specified`：问题、方案、验收标准已经写清楚。
- `planned`：已有实现计划或进入某个版本范围。
- `in_progress`：正在开发。
- `implemented`：代码已完成，但还没有完整验证或发布。
- `verified`：验证通过。
- `released`：已经进入发布版本。
- `blocked`：被依赖项、技术问题或决策卡住。
- `deferred`：暂缓，不在当前阶段做。
- `rejected`：明确不做。
- `superseded`：被其他需求替代，必须在 `notes` 或 `dependencies` 中写替代 ID。

## Skill 行为建议

可以创建一个 `prd-csv` skill，让 Codex 按以下协议工作。

### 录入新需求

当你口述一个需求时，Codex 应该：

1. 判断这是新需求、已有需求补充、bug、体验问题还是实现任务。
2. 搜索 CSV 中是否已有相似需求，避免重复创建。
3. 把口述内容润色为结构化字段。
4. 如果信息不足，最多问 1 到 3 个关键问题。
5. 分配新的 `REQ-xxxx`。
6. 写入 CSV。
7. 在回复中给出新增行摘要和下一步建议。

### 修改已有需求

当你说“修改某个需求”时，Codex 应该：

1. 按 ID 优先定位，标题模糊匹配作为兜底。
2. 保留 `id` 不变。
3. 更新受影响字段。
4. 如果需求含义发生重大变化，在 `notes` 中记录变化原因。
5. 不静默删除验收标准，除非你明确要求。

### 开发状态追踪

当某个功能被实现时，Codex 应该：

1. 找到对应 `REQ-xxxx`。
2. 更新 `status`、`progress`、`implementation_refs`、`test_refs`。
3. 如果验收未完成，不应直接标记为 `verified`。
4. 如果实现过程中发现需求变更，先更新 PRD，再继续实现记录。

### 一致性检查

Codex 可以定期执行 PRD lint：

- 检查重复 ID。
- 检查非法状态值。
- 检查 `implemented` 但没有 `implementation_refs` 的需求。
- 检查 `verified` 但没有 `test_refs` 的需求。
- 检查 `specified` 及以后状态但缺少 `acceptance_criteria` 的需求。
- 检查 `superseded` 但没有替代说明的需求。
- 检查目标版本为空但优先级为 `P0` 或 `P1` 的需求。

## 推荐的文件结构

```text
docs/
  prd/
    requirements.csv
    README.md
    details/
      REQ-0001.md
```

`README.md` 用来说明字段含义、状态流转和 Codex 操作约定。`details/` 只在复杂需求需要更长说明时使用。

## 使用示例

你可以这样对 Codex 说：

```text
把这个需求加入 PRD：我希望设置页能显示当前安装包版本，并提示是否有新版。
```

Codex 应该输出类似：

```text
已新增 REQ-0007：设置页显示安装包版本和更新状态。
状态为 idea，优先级暂定 P1，模块为 settings。
我把验收标准拆成：显示当前版本；能检查新版；失败时有非阻塞提示；开发环境能模拟检查链路。
```

你也可以说：

```text
把 REQ-0007 标记为 implemented，相关计划是 docs/plan/2026-04-19-app-update-check-prd.md，验证命令是 pnpm build。
```

## 特别提示

### 1. 不要把 PRD CSV 当任务看板的替代品

CSV 应该记录“产品需求事实”，不是每天的开发流水账。任务拆分、实现细节、临时 bug 可以放在计划文档或 issue 中，再通过 `implementation_refs` 关联回来。

### 2. 每条需求必须有可验收结果

如果一条需求无法写出验收标准，它大概率还只是一个想法，不应该进入开发状态。

### 3. ID 稳定比标题优雅更重要

标题可以反复改，`REQ-xxxx` 不能改。后续提交信息、计划文档、测试记录都应该引用这个 ID。

### 4. 状态要保守更新

`implemented` 不等于 `verified`，`verified` 不等于 `released`。这三个状态分开能避免“我以为做完了”的错觉。

### 5. Codex 更新 PRD 时应该先 diff 再写入

PRD 是产品事实源，不能让 Agent 随意重排、重写整张表。Skill 应要求最小修改：只改必要行，只改必要字段，保留用户手动编辑。

## 下一步建议

如果要落地，建议按以下顺序推进：

1. 新建 `docs/prd/requirements.csv` 和 `docs/prd/README.md`。
2. 写一个仓库内或用户级的 `prd-csv` skill，明确读写 CSV、去重、润色、状态更新、lint 的规则。
3. 先手动录入 5 到 10 条已有需求，验证字段是否够用。
4. 增加一个轻量 lint 脚本，例如 `scripts/prd_lint.py`，用于检查 ID、状态、引用和必填字段。
5. 后续再考虑把 CSV 渲染为 Markdown 表格、HTML 看板，或同步到飞书多维表格。


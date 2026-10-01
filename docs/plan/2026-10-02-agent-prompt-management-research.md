# Agent 提示词管理：开源调研与 Curated 优化建议

日期：2026-10-02（北京时间）

状态：源码调研与建议；已按用户确认实施独立 TXT 迁移。配置覆盖、任务版本冻结与数据库迁移尚未实施。

## 1. 结论

成熟开源 Agent 没有统一采用数据库或在线提示词平台。此次核对的四个项目分别使用独立 TXT、YAML、专用 TypeScript 模块，默认提示词均由源码仓库管理，再由程序加载或组装。独立文件不等于运行时热更新；内置模板和用户扩展指令也应分开看。

Curated 已有 `system.md + go:embed`，方向合理。优先优化提示词分散、任务级版本固定、约束一致性和质量评估，而不是先增加在线编辑功能。

建议：统一提示词资源目录，以 Markdown 存正文、Go 保存注册信息和类型化渲染逻辑，继续 embed 交付；为每种用途独立标识版本及内容哈希。全库整理在启动时固定所用提示词包，恢复时沿用同一版本。

## 2. 调研范围与证据

通过 GitHub API 获取各仓库默认分支的完整 commit SHA，并从该 SHA 的 raw 源码读取模板和调用点。以下链接固定到此次读取的提交，避免分支后续变化影响结论。没有运行这些项目，也没有用真实模型验证其效果；结论限定为所列源码中的存放、加载及组装方式，不表示整个项目只有这些提示词。

| 项目 | 默认分支 | 本次源码快照 | 主要存放方式 |
|---|---|---|---|
| OpenCode | dev | `aa481b8f5652f5576c55f914a64ed270e7daa7e0` | 独立 TXT，代码导入、按模型和 Agent 用途选择 |
| Cline | main | `0809928ab28783c0d2b41c1e56edaf0951dadcab` | 本次核对的 shared SDK 路径为 TypeScript 模板字符串及集中构建函数 |
| Hugging Face smolagents | main | `c30b115286e000e98711fae5e85993547b73d826` | 包内 YAML，按 Agent 类型加载、Jinja 渲染 |
| SWE-agent | main | `3ea751c087f32b16e039a2233dd6eefecef325d5` | 实验配置 YAML 与 Python 模板配置类，Jinja 渲染 |

### 2.1 OpenCode：独立文本与选择逻辑分离

- `packages/opencode/src/session/prompt/*.txt` 存主提示词，包括默认版本及不同模型族版本。
- `packages/opencode/src/session/system.ts` 直接导入 TXT；`provider()` 按模型信息选择文本。
- `packages/opencode/src/agent/prompt/*.txt` 存 compaction、summary、title、explore 等用途文本；`agent.ts` 将其配置给对应 Agent。
- `session/instruction.ts` 另行读取项目/全局 `AGENTS.md` 等扩展指令及配置中的 instructions。这一层与内置基础提示词有不同来源。
- 借鉴：正文独立审阅、按用途组织、扩展指令独立加载。TXT 被源码导入本身不意味着编辑安装目录中的文件就能热更新。

证据：[主模板选择](https://github.com/anomalyco/opencode/blob/aa481b8f5652f5576c55f914a64ed270e7daa7e0/packages/opencode/src/session/system.ts#L6-L50)、[Agent 模板装配](https://github.com/anomalyco/opencode/blob/aa481b8f5652f5576c55f914a64ed270e7daa7e0/packages/opencode/src/agent/agent.ts#L214-L283)、[扩展指令读取](https://github.com/anomalyco/opencode/blob/aa481b8f5652f5576c55f914a64ed270e7daa7e0/packages/opencode/src/session/instruction.ts)。

### 2.2 Cline：代码字符串也可以集中管理

- 本次 SDK 路径 `sdk/packages/shared/src/prompt/system/act.ts` 与 `yolo.ts` 存模式模板字符串，`index.ts` 集中导出。
- `prompt/cline.ts` 的 `buildClineSystemPrompt()` 选择基础模板，并拼接模式规则、调用方 rules、平台、工作目录等变量。
- 支持每请求 `overridePrompt`；`cline.test.ts` 检查不同模式、规则顺序及覆盖行为。
- 借鉴：提示词写在代码中并非天然问题，专用模块、统一构建入口和清晰的覆盖语义更重要。这里的覆盖能力不应直接照搬为 Curated 的写权限机制。

证据：[基础模板](https://github.com/cline/cline/blob/0809928ab28783c0d2b41c1e56edaf0951dadcab/sdk/packages/shared/src/prompt/system/act.ts)、[构建与覆盖](https://github.com/cline/cline/blob/0809928ab28783c0d2b41c1e56edaf0951dadcab/sdk/packages/shared/src/prompt/cline.ts#L129-L205)、[模式测试](https://github.com/cline/cline/blob/0809928ab28783c0d2b41c1e56edaf0951dadcab/sdk/packages/shared/src/prompt/cline.test.ts)。

### 2.3 smolagents：YAML 管理多个阶段的模板

- `src/smolagents/prompts/code_agent.yaml`、`toolcalling_agent.yaml`、`structured_code_agent.yaml` 存不同 Agent 的模板。
- 单个 YAML 含 system_prompt、planning、managed_agent、final_answer 等阶段。
- `agents.py` 使用 `importlib.resources` 读取包内 YAML，再由 `yaml.safe_load` 解析；构造参数允许提供自定义 prompt_templates。
- `populate_template()` 使用 Jinja 的 `StrictUndefined`，缺变量时报错；自定义模板也会检查必要键。
- 借鉴：阶段化组织、渲染变量严格校验、测试时注入模板。Curated 的用途数量较少，不必为了相同效果引入 Python/Jinja 或复杂 YAML。

证据：[模板 YAML](https://github.com/huggingface/smolagents/blob/c30b115286e000e98711fae5e85993547b73d826/src/smolagents/prompts/toolcalling_agent.yaml)、[严格渲染](https://github.com/huggingface/smolagents/blob/c30b115286e000e98711fae5e85993547b73d826/src/smolagents/agents.py#L102-L107)、[包内加载](https://github.com/huggingface/smolagents/blob/c30b115286e000e98711fae5e85993547b73d826/src/smolagents/agents.py#L1241-L1249)。

### 2.4 SWE-agent：提示词属于可复现的运行配置

- `config/default.yaml` 在 `agent.templates` 下配置 system_template、instance_template、next_step_template 等正文。
- `sweagent/agent/agents.py` 的 `TemplateConfig` 定义模板字段；部分观察截断、超时等默认文本仍在 Python 中。
- 运行时使用 Jinja `Template(...).render(...)`，将模板与问题、工作目录、观察结果等参数组合成消息。
- 借鉴：将“这次运行用了什么模板配置”作为完整运行条件看待。不能据此推断它有数据库提示词后台或所有变量都执行 StrictUndefined 校验。

证据：[默认 YAML](https://github.com/SWE-agent/SWE-agent/blob/3ea751c087f32b16e039a2233dd6eefecef325d5/config/default.yaml)、[模板配置类](https://github.com/SWE-agent/SWE-agent/blob/3ea751c087f32b16e039a2233dd6eefecef325d5/sweagent/agent/agents.py#L60-L79)、[系统消息渲染](https://github.com/SWE-agent/SWE-agent/blob/3ea751c087f32b16e039a2233dd6eefecef325d5/sweagent/agent/agents.py#L608-L615)。

## 3. Curated 当前实现与缺口

当前工作区 HEAD：`bd85476e26f72bb536c6736839eccc85a2b422f6`；本次只读核对后端，原有播放器相关未提交改动不在调研范围内。

### 3.1 已具备的基础

- `backend/internal/agent/prompts/system.md` 经 `system.go` embed，静态正文与页面上下文构建已经分离。
- AI 调用记录已有 `promptVersion`；主聊天 `agent-system-v7`，动作共用 `agent-actions-v1`，整理分别为 `topic-vocabulary-v3` / `topic-classification-v1`。
- 题材输出有严格 JSON 解码、字段与引用验证，写入有权限网关、范围、版本及资料指纹检查；这些硬约束应继续留在程序中。
- 已有模板断言、分类校验测试及脚本模型驱动的 Agent eval runner，不能说项目完全没有评估。

### 3.2 优化空间

1. **存放分散**：主聊天在 prompts 包；题材文本在 app/tag_organization_classifier.go；记忆摘要和回答纠正在 run 包；动作文本在 prompts/actions.go。审查和复用需要跨层搜索。
2. **版本追踪过粗**：多个动作共用一个版本号。题材版本在调用点手工指定，甚至用提示词字符串相等来判定用途；正文与版本元数据容易不同步。
3. **后台任务没有冻结提示词**：整理任务持久保存 locale、词表及进度，但当前任务表没有所用提示词包或哈希字段。重启后使用当前程序常量，升级可能使同一个任务前后采用不同策略。只在调用记录中记版本不能解决恢复一致性。
4. **动态源资料进入 system 正文**：润色、翻译及洞察构建函数把笔记、简介、JSON 放进 system 的 `<source>`，部分又在 user 消息重复发送。标签整理已经采用静态 system + 独立 user JSON，更适合作为统一模式。角色分离有助于减少重复并明确数据来源，但不能保证杜绝提示注入；标签引用验证和写入权限仍是独立保障。
5. **提示词期望与校验上限不同**：题材归纳要求每次至多 12 个新增题材、描述至多 80 字符、至多 5 个别名；通用校验器允许 160 个词汇、1000 字节描述、20 个别名。160 可能是累计词表上限，不应直接降为 12；应拆分“单响应校验”和“累计词表校验”，统一其他计量单位与限制。这是策略一致性需要明确，不宜直接认定所有差异都是缺陷。
6. **真实质量评估尚未闭环**：现有确定性测试能验证代码边界，不能代替真实模型的题材质量验收。需要固定样本、版本、模型和可比较指标。

## 4. 推荐结构（未来方案）

继续沿用 `backend/internal/agent/prompts/`，逐步形成：

```text
prompts/
  system.md                    # 保留现有主聊天模板路径
  assets/
    topics/vocabulary.md
    topics/classification.md
    actions/polish-comment.md
    actions/translate-display.md
    actions/insights-narrative.md
    memory/checkpoint.md
    chat/answer-correction.md
  catalog.go                   # embed、用途 ID、版本、哈希、契约元信息
  render.go                    # 类型化变量、严格渲染与消息装配
```

这是目标目录，不是当前已存在的全部文件。不必将每一句短提示、工具说明都拆成文件；优先迁移较长或独立演进的策略文本。

- 为每种用途注册 `ID + Version + SHA256 + OutputContractVersion`；优先在 Go 中声明，避免为少量静态元数据添加 YAML 依赖。
- 对模板静态正文及模板组合方式计算稳定哈希，不把原始笔记、影片资料、API key 或含隐私的完整渲染结果写进审计记录。动态资料的指纹另行记录，不与模板哈希混为一谈。
- 业务代码通过明确用途获取模板对象，记录对象提供的版本；不要根据字符串内容或相等关系推断版本。
- system 消息存稳定规则；locale 等受控参数可在受控片段中加入；影片/笔记/观察等源资料统一作为序列化数据传入 user/tool 消息。模板只渲染一次，源资料中的 `{{...}}` 不得再次作为模板求值。
- 输出类型、schema、可写字段、权限、事务与引用校验继续由 Go 保证。提示词里涉及的数值从共同的契约常量注入，避免两套数值长期漂移。
- 正式版本继续 embed；修改内置模板仍需编译和发布，这是可复现交付的选择。

### 4.1 固定任务提示词包

创建整理任务时保存归纳、分类模板及契约的精确身份。每次调用审计记录实际使用身份，并与任务冻结值核对。恢复时使用原包；旧包不可用时明确阻塞，禁止静默替换成新策略。

首版可采用“内置版本保留 + job 保存版本/哈希”的方式，跨版兼容由程序显式控制。若后续确有外部模板需求，再考虑存储已验证的不可变模板快照或本地版本包。仅保存哈希不能重建正文，不能单独作为可恢复方案。已有未记录版本的任务需要明确兼容/阻塞策略，不能假装有历史快照。

### 4.2 暂缓在线编辑与模型分支

当前不建议先建设数据库提示词 CRUD、在线市场、远程拉取或全局热更新。它们增加配置漂移、恢复一致性与错误模板处置成本；也不能直接改善归类准确率。

如果调试需求明确，可以后续加仅开发环境启用的本地覆盖，严格校验用途、变量与契约，记录来源和哈希，按运行/任务冻结快照，默认关闭。正式发布保持内置模板。

不同模型模板只在固定样本证明有稳定收益后新增，先使用统一语义模板和通用 fallback；避免单凭模型名称字符串大量分叉。以上为 Curated 的建议，不是四个开源项目共同实施的规则。

## 5. 实施顺序与验收建议

### 第一阶段：集中资源、保持行为

将长提示词迁入专用资源目录；新增 catalog；调用点改为显式用途。迁移时保持现有文本和消息角色一致，便于区分资源重构与行为修改。

验收：比较迁移前后实际模板/消息内容，检查用途和版本映射、embed 资源完整性及依赖方向；运行相关 prompts、run、app 测试。无需为每句正文写镜像测试。

### 第二阶段：修复可复现与契约问题

独立提交任务提示词包持久化与恢复策略；独立提交动作消息的数据分离；独立提交单响应/累计词表的约束统一。新增数据库字段需遵循仓库迁移与文档同步规则。

验收覆盖：跨程序版本恢复不切换模板、缺旧包明确阻塞、取消与重试保持模板身份、数据不被二次渲染、JSON/引用/权限边界仍有效、描述长度的单位一致。不要一次同时改组织形式和题材策略。

### 第三阶段：真实模型质量回归

使用合成或脱敏固定样本，覆盖明确题材、否定描述、模糊宣传、同义词、多语言、资料冲突、无证据和注入文本。记录模板身份、模型、上下文配置、样本版本、运行时间与用量；真实模型运行有成本，不属于本次只读调研。

对比结构有效率、引用有效率、题材精确率、可归类样本召回率、无证据样本误归类率、同义词准确性、跨批词表稳定性及成本。采用多次运行，避免把单次随机结果当作改进。硬权限/NFO 隔离由确定性测试验证。

最初调研阶段仅交付文档。用户随后确认采用 OpenCode 风格的独立 TXT，实施记录见下一节；未启动正式库整理或调用用户配置的模型。


## 6. 2026-10-02 实施记录：采用独立 TXT

用户明确选择 OpenCode 风格的独立 TXT，覆盖上文第四节的 Markdown 资源建议。当前统一使用 `backend/internal/agent/prompts/` 下的 14 个平铺 TXT，未增加 assets 子目录或 YAML。

- 主聊天 `system.md` 改为 `system.txt`；语言、页面、筛选、提及片段分别独立。
- 原 app 包中的归纳/分类、run 包中的摘要/纠正/截断说明，以及连接探针迁入 TXT。
- 原 actions.go 内的润色/翻译/洞察规则迁入带受控变量的 TXT，保留原消息角色与源资料位置。
- `templates.go` embed 全部 TXT，启动时一次解析，模板缺变量明确失败；只执行一次，数据中的模板符号保留原文。去掉每个文件的末尾一个 LF，其余空白保留；上下文需要的末尾空格在模板内显式表达，避免编辑器删除尾空格。
- 题材模板通过 `Definition{Text, Version}` 提供；原 `topicComplete` 的正文比较被显式版本元信息取代。既有版本未变，因为本次没有改模型看到的文本。
- 新增迁移前消息哈希基线与静态规则比对，覆盖默认语言、多行/空白、上下文、源资料中的 `{{...}}`；新增缺变量拒绝测试。
- 硬权限、引用验证、长度策略、数据库与 HTTP 合约保持原实现；在线编辑、热更新、任务冻结版本、动作独立版本、哈希审计和真实模型质量回归仍属后续工作。

详细维护入口：`docs/guide.md` 的 AI prompt TXT resources。

验证结果：在 `backend/` 执行 `go test ./internal/agent/... ./internal/app/...`、`go test ./...`、`go vet ./...` 均通过；代表性消息及静态规则与迁移前基线一致，`git diff --check` 通过。本次没有运行真实 Provider 请求。

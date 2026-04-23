# 仓库目录结构优化方案

> 范围：针对 `Curated`（仓库目录仍为 `jav-shadcn`）的仓库目录布局做评估与优化规划。本文重点关注结构清晰度、可发现性和维护成本；除非后续明确排期，否则本文本身不要求立刻进行代码迁移。

## 1. 当前快照

基于 `2026-04-24` 当前工作区状态：

- 当前工作区顶层可见目录数：`24`
- 顶层目录中实际包含 tracked 仓库文件的目录数：`9`
- 当前工作区递归目录总数：`5256`
- 当前工作区递归文件总数：`35394`

这里要先说明两点：

- 递归统计出来的总目录数和总文件数，被本地缓存、生成产物和运行数据明显放大了，所以**不能直接拿来判断仓库架构是否健康**。
- 如果从 tracked file 分布来看，仓库的主要内容实际上集中在：
  - `docs/`（`380` 个 tracked 文件）
  - `src/`（`284` 个 tracked 文件）
  - `backend/`（`230` 个 tracked 文件）
  - `.cursor/`（`31` 个 tracked 文件）
  - `scripts/`（`17` 个 tracked 文件）

还有几个值得注意的数据点：

- 单独一个 `docs/film-scanner/` 就有 `291` 个 tracked 文件，这意味着 `docs/` 里相当大一块内容实际上不是传统意义上的文档，而更像一个可运行的实验或子项目。
- `src/components/` 有 `135` 个文件，`src/lib/` 有 `90` 个文件，说明前端业务逻辑已经开始分散到多个技术层目录里。
- `backend/internal/storage/` 有 `51` 个文件，是后端里最密集的区域之一。

## 2. 总体判断

### 目前已经比较合理的地方

- 仓库仍然保持了 `src/` 和 `backend/` 这条明确主边界，这个结构是对的，建议保留。
- `scripts/` 已经按用途分成了 `dev/`、`prd/`、`release/`，方向是对的。
- 前端公共 UI 在 `src/components/ui/` 下，这条边界很清楚，应该继续保留。
- 后端 Go 代码统一放在 `backend/internal/` 下，整体上符合 Go 项目的常见组织方式，只要后续继续维持好分组语义，就还是可维护的。

### 目前偏弱的地方

1. 根目录日常导航噪音过大。
2. `docs/` 承载了过多不同职责。
3. 前端越来越像按“技术层”组织，而不是按“功能域”组织。
4. 后端 `internal/` 下包的数量越来越多，但分组语义还不够稳定。
5. 一些本地缓存和运行产物仍然出现在 repo 内部，和现有文档约束不完全一致。

## 3. 主要问题

### 问题 A：根目录同时混放了源码、运行产物、缓存和流程状态目录

当前根目录除了源码和文档目录外，还能看到这类本地目录：

- `.codex-temp/`
- `.gocache/`
- `.pnpm-store/`
- `.tmp/`
- `dist/`
- `log/`
- `output/`
- `release/`
- `videos_test/`
- `openspec/`
- `config/`

这些目录即使大部分都已经被 Git ignore，也仍然会显著增加日常浏览和理解成本。一个需要先“过滤噪音”才能看懂的根目录，本身就是效率问题。

补充约束说明：

- 根目录 `videos_test/` 视为固定测试素材目录，需要保留原位。它在导航上属于“根目录噪音来源”的一种，但**不属于这份计划里的迁移目标**。

### 问题 B：`docs/` 已经过载

当前 `docs/` 同时承载了：

- 长期保留的架构/产品说明
- review / audit 类文档
- 实施计划
- release notes
- PRD 材料
- `docs/film-scanner/` 这种带可运行内容的实验/工具目录
- 混在 `docs/plan/` 里的独立 HTML 原型文件

这是当前 tracked 目录结构里最明显的问题。`docs/` 这个名字的语义应该是“文档目录”，但现在里面已经掺杂了工具、实验和一次性原型。

### 问题 C：前端业务逻辑分散在太多技术层目录

目前前端结构还没到不可维护，但一个功能点相关的代码经常分散在：

- `src/views/`
- `src/components/jav-library/`
- `src/composables/`
- `src/lib/`
- `src/api/`
- `src/services/`
- `src/domain/`

这种方式在项目早中期没问题，但随着 homepage recommendations、actor profiles、player pipeline、history、curated frames、settings 等功能并行演进，跨目录跳转成本会越来越高。

### 问题 D：后端包名整体还行，但分组可以更清楚

当前后端已经有一些比较明确的领域包：

- `library`
- `playback`
- `scanner`
- `scraper`
- `desktop`
- `server`
- `storage`

但与此同时，也有很多基础设施或横切能力包和它们处在同一层级：

- `assets`
- `browserheaders`
- `config`
- `executil`
- `logging`
- `proxyenv`
- `shellopen`
- `version`
- `webui`

随着规模继续增长，所有这些包都平铺在 `internal/` 下，会让目录列表越来越长，语义也越来越分散。

### 问题 E：repo 内部缓存治理还不够严格

当前工作区里可以看到这类 repo 内本地缓存或运行目录：

- 根目录 `.gocache/`
- `backend/.gocache/`
- `backend/.tmp-go/`
- `backend/runtime/`

其中 `backend/runtime/` 是有明确用途的，用于开发态二进制和运行数据，可以保留；但 repo 内 Go cache 这类目录，恰恰是现有文档已经明确说要避免的内容。它不仅是“看起来乱”，还会拉高文件总数、拖慢工具扫描、增加误操作概率。

## 4. 建议的总体方向

我建议采用**保守分阶段重排**，而不是一次性大改。

原因很直接：

- 当前核心顶层结构并没有错到需要推翻重来。
- 真正的问题主要集中在“目录职责混杂”“命名语义不清”“内容放错位置”。
- 分阶段治理可以在较低风险下拿到大部分收益，避免一次性大迁移带来的 import、路径、文档、脚本联动成本。

## 5. 目标结构原则

### 根目录原则

根目录应该让人几秒钟内就能回答三个问题：

1. 产品源码在哪里？
2. 项目文档在哪里？
3. 本地运行/缓存/临时产物应该放在哪里？

建议的根目录意图：

- 作为主要 tracked 根目录长期保留：
  - `src/`
  - `backend/`
  - `docs/`
  - `scripts/`
  - `public/`
  - `icon/`
- 配置文件只在它们确实是 repo 级别配置时保留在根目录，比如 `package.json`、`vite.config.ts`、`components.json`、`README*`、`API.md`、规则文件。
- 实验或内部工具不要继续塞在 `docs/` 里，可以迁移到类似：
  - `tools/`
  - 或 `experiments/`
- 本地专用目录如果可以，最好集中到一个统一的 ignored 命名空间，例如：
  - `.local/`
  - 或 `.workspace/`

### 文档目录原则

`docs/` 应该只放文档和轻量引用资源，不应该再混入可运行子项目。

建议逐步收敛为：

- `docs/architecture/`
- `docs/guides/`
- `docs/plan/`
- `docs/prd/`
- `docs/reviews/`
- `docs/release-notes/`
- `docs/reference/`

这里不建议一口气大迁移，应该渐进式收敛。

### 前端目录原则

共享基础设施继续共享，但业务功能代码要尽量靠拢。

建议的目标方向：

- `src/features/library/`
- `src/features/actors/`
- `src/features/player/`
- `src/features/history/`
- `src/features/home/`
- `src/features/curated-frames/`
- `src/features/settings/`
- `src/shared/ui/`，或者继续保留 `src/components/ui/`
- `src/shared/lib/`，或者继续保留 `src/lib/` 作为跨 feature 的工具层
- `src/app/` 负责 router、shell、bootstrapping 等全局内容

这个方向应该按 feature 逐步迁移，不能一次性整体搬迁。

### 后端目录原则

不要为了视觉统一就重写符合 Go 习惯的包结构。只有在“更容易理解”的情况下，才值得做分组调整。

建议方向：

- 领域包继续保持领域导向
- 给平台/基础设施类能力建立更明确的归属位置
- 不要让越来越多小 utility 包长期直接平铺在 `internal/` 下

例如可以朝这个方向演进：

```text
backend/internal/
  app/
  domain/
    library/
    playback/
    scanner/
    scraper/
    desktop/
    appupdate/
  platform/
    config/
    logging/
    assets/
    executil/
    proxyenv/
    shellopen/
    version/
    webui/
  storage/
  server/
  contracts/
```

这只是方向，不是强制一模一样落地。比如 `storage/` 和 `server/` 保持顶层也完全可以接受。

## 6. 具体优化计划

### 阶段 1：根目录清理与目录政策固化

优先级：最高  
风险：低  
目标：让仓库根目录一眼能看懂

动作：

1. 在文档里补一段简短的根目录政策说明：
   - 哪些是 tracked 的源码/文档目录
   - 哪些是允许存在的运行目录
   - 哪些是本地 ignored 目录
   - 哪些是明确允许保留在根目录的固定例外，比如 `videos_test/`
2. 停止在 repo 内生成 Go cache：
   - 清理 repo 内缓存生成路径
   - 和现有 build/test 规则文档保持一致
3. 盘点现有 ignored 本地目录，能收拢的尽量收拢：
   - 尽量用一个统一的本地工作区目录承载，而不是多个散落根目录
4. 明确 `config/` 是否需要继续作为根目录本地目录存在，还是迁移到更清晰的 ignored 命名空间下。

预期效果：

- 根目录误导性目录减少
- 文件浏览器和命令行扫描更清爽
- contributor 和 agent 更不容易误判项目主结构

### 阶段 2：处理 `docs/` 过载问题

优先级：最高  
风险：低到中  
目标：让 `docs/` 重新只表达“文档”

动作：

1. 把 `docs/film-scanner/` 从 `docs/` 挪出：
   - 推荐目标：`tools/film-scanner/`
   - 备选目标：`experiments/film-scanner/`
2. 把 `docs/plan/` 里的一次性 HTML 原型文件挪到更合适的位置：
   - 推荐：`docs/prototypes/`
   - 或 `experiments/ui-prototypes/`
3. 为 `docs/` 建立更清楚的子目录语义：
   - architecture
   - guides
   - plan
   - prd
   - reviews
   - release notes
4. 在 `docs/` 下放一个简短 README 或索引文档，明确以后不同类型文档该往哪放。

预期效果：

- `docs/` 更容易检索
- 新增文档时不容易再出现“放哪都像对、放哪都不太对”的情况
- 长期文档、计划文档、实验产物之间的边界更清楚

### 阶段 3：前端按功能域逐步收敛

优先级：中  
风险：中  
目标：减少改一个功能时的跨目录跳转成本

动作：

1. 先冻结当前真正共享的公共层：
   - 保留 `src/components/ui/`
   - 保留全局 app shell / router / bootstrap 的独立位置
2. 引入 `src/features/`，作为新增或持续演进功能的目标位置。
3. 一次只迁一个功能域，优先处理最分散的：
   - `actors`
   - `player`
   - `curated-frames`
   - `home`
4. 对每个迁移的 feature，尽量把这些内容放在一起：
   - feature page/view
   - feature components
   - feature composables
   - feature-specific types / mappers
5. 只把真正跨 feature 复用的内容保留在 `src/lib/` 和 `src/services/`。

预期效果：

- feature 开发效率更高
- 不相关功能之间耦合降低
- `src/lib/` 这种“公共杂物箱”进一步膨胀的风险降低

### 阶段 4：后端包分组语义优化

优先级：中  
风险：中  
目标：在不破坏稳定性的前提下提升后端目录可读性

动作：

1. 先梳理哪些包明显属于：
   - domain
   - transport
   - platform / infrastructure
   - persistence
2. 对基础设施类包做更明确的归类。
3. 除非收益明显，否则不要去挪动已经稳定、引用很多的高频包。
4. 只有在“一个目录既大又混杂”时，才考虑继续拆分。

预期效果：

- 后端目录心智模型更清楚
- `backend/internal/` 扫描效率更高
- 风险远小于全面重构

## 7. 目前不建议动的地方

下面这些地方不应该为了“看起来更干净”就去改：

- 不要把前后端合并成一个混合式 app 目录。
- 不要移除 `backend/` 这条 Go 代码边界。
- 不要拆掉 `src/components/ui/`；共享 UI 原语仍然需要稳定边界。
- 不要移动根目录 `videos_test/`；它是批准保留的固定测试素材目录。
- 不要围绕递归总目录数做表面优化；这个数字被缓存和运行产物严重放大了。
- 不要为了统一命名而大面积重命名那些已经和业务能力一一对应的 Go 包。

## 8. 建议执行顺序

推荐顺序：

1. 根目录治理规则
2. `docs/` 清理
3. 前端选择一个 feature 做试点迁移
4. 等前三步稳定后，再做后端包分组优化

这个顺序的好处是：前面两步见效快、风险低；后面的结构迁移建立在规则已经明确的基础上，不会变成一轮没有边界的整理工程。

## 9. 可选方案

### 方案 A：只做保守清理

做：

- 根目录治理
- `docs/` 清理
- 不做大的源码重组

适合：

- 你现在最在意的是立刻提升清晰度，同时把风险压到最低

### 方案 B：平衡型优化

做：

- 根目录治理
- `docs/` 清理
- 前端引入 `src/features/` 并先迁一个试点 feature
- 后端只做轻量分组收敛

适合：

- 你希望拿到比较明显的中长期收益，但不想承受“一次性大重构”的成本

### 方案 C：全面结构重排

做：

- 根目录治理
- `docs/` 清理
- 前端大范围搬迁
- 后端大范围重分组

适合：

- 你愿意接受短期明显扰动，以换取一次更彻底的目录重置

推荐选择：**方案 B**

因为它对当前仓库来说，收益和迁移风险的比例最合适。

## 10. 总结

这个仓库目前**不是核心架构错了**，而是以下几个问题开始变得明显：

1. 根目录本地目录和生成目录太多，观感噪音高
2. `docs/` 已经过载，尤其是 `docs/film-scanner/`
3. 前端业务代码开始分散在多个技术层目录里
4. 后端 `internal/` 未来需要更清楚的分组语义

所以正确做法不是“把整个仓库重新设计一遍”，而是：

- 先把根目录清干净
- 让 `docs/` 重新只表达文档
- 前端开始渐进式 feature 化
- 后端只在真正有价值的地方做分组优化

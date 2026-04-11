# 返回机制评估（2026-04-12）

## 结论

当前返回机制有明确优化空间，但不需要推倒重来。

现状已经具备两个优点：

- 不依赖浏览器原生 `history.back()`，因此深链进入 `detail` / `player` 时仍能给出可预测的回退目标。
- 列表上下文已经通过 route query 持久化，且列表滚动位置已有按 browse-context 的内存恢复机制。

但当前方案也存在结构性问题：

- “返回”语义分散在多个层次：`query.from`、`buildMovieRouteQuery()`、`AppShell` 顶栏判断、列表 `selected`/滚动恢复，各自成立，但没有统一抽象。
- `player` 页对来源的识别是特判式的，目前只显式支持 `history` 和 `curated-frames`，普通 browse 路径会固定回到 `detail`，不一定符合用户真实心智。
- 当前系统更像“显式目标跳转”而不是“导航状态机”，所以应用返回、浏览器返回、刷新后恢复三者语义并不一致。
- 滚动恢复只存在于前端运行时内存，刷新或新标签页后丢失。

## 现状判断

### 1. 当前方案不是错，只是边界开始变脆

在页面数量较少时，`AppShell` 按路由名和 `query.from` 手工判断返回目标是可接受的。

但现在已经至少出现这些来源类型：

- 资料库 browse
- 详情页
- 历史页
- 萃取帧页
- 特殊过滤后的详情回跳

来源一多，靠壳层硬编码分支会继续膨胀，后续再加演员库、设置子页、播放器内新入口时，复杂度会继续上升。

### 2. 最明显的不一致点在 `player`

当前 `player` 的顶栏返回规则是：

- `from=history` -> `history`
- `from=curated-frames` -> `curated-frames`
- 否则 -> 当前影片 `detail`

这会带来一个体验特征：

- 从列表直接进入播放器，用户点返回时并不会回列表，而是先回详情。

这个行为未必错误，但它不是“自然回到来处”，而是“产品定义的默认回退链”。如果产品就想这样，可以保留；如果目标是“尽量回到用户上一步所在工作面”，这里就是第一优先级优化点。

### 3. 当前 query 已经承担了太多职责

同一套 query 现在同时承担：

- browse 来源模式
- 搜索 / 标签 / 演员 / 厂商过滤
- 列表 tab
- 当前选中影片
- 播放起点 `t`
- 特殊来源标记 `from=history` / `from=curated-frames`

这让“路由上下文”和“返回语义”耦合较深。问题不是字段多，而是没有区分：

- 哪些字段是“展示状态”
- 哪些字段是“返回意图”
- 哪些字段是“临时动作参数”

## 推荐优化方向

### 方向 A：保守优化，继续沿用现有模型

适合短期，改动最小。

做法：

- 保留 query 作为 browse context 主载体。
- 保留 `AppShell` 顶栏显式返回按钮。
- 只把返回目标计算逻辑从 `AppShell` 中提炼成单独 composable / helper。
- 明确定义普通 browse -> player 的返回策略，到底是回 `detail` 还是回 browse。
- 把 `history` / `curated-frames` 这种来源特判，收敛成统一来源枚举，而不是散落字符串。

收益：

- 风险低。
- 可以先把复杂度压下来，不改变主要交互。

不足：

- 仍然不是统一导航模型。
- 浏览器返回与应用返回仍会保持双轨语义。

### 方向 B：建立统一的 “navigation intent” 层

这是我更推荐的中期方向。

做法：

- 在前端建立一个统一的导航上下文模型，例如：
  - `sourcePage`
  - `sourceMode`
  - `sourceFilters`
  - `returnTarget`
  - `selectedMovieId`
- `detail` 和 `player` 不再各自猜测从哪里来，而是消费统一的导航 intent。
- `AppShell` 只渲染“返回目标”，不再自己拼装业务规则。
- `buildPlayerRouteFromBrowse()` / `buildPlayerRouteFromHistory()` / `buildPlayerRouteFromCuratedFrame()` 收敛成一层更统一的 route-builder。

收益：

- 返回逻辑集中，后续新增入口不会继续把判断写回壳层。
- 更容易测试。
- 更容易定义“应用返回”和“浏览器返回”的差异边界。

成本：

- 需要调整多个入口函数和 query 约定。
- 需要补一轮路由层测试。

### 方向 C：做真正的应用内导航历史栈

这是最强但也最重的方案，目前不建议立刻上。

做法：

- 维护前端自己的 navigation stack。
- 顶栏返回优先消费应用历史栈，而不是只根据当前 route 推导。
- query 主要负责可分享 / 可刷新恢复的状态，返回语义由 history store 负责。

优点：

- 最接近“用户真实来路”。

缺点：

- 和浏览器原生历史栈存在双栈同步问题。
- 需要处理刷新、深链、replace、批量筛选变更等一整套边界。

## 推荐优先级

### P1：先做

- 提炼统一的返回目标解析函数，移出 `AppShell`。
- 把来源字符串收敛成统一枚举或 helper。
- 明确普通 browse -> player 的产品语义。

### P2：随后做

- 把 `detail` / `player` / `history` / `curated-frames` 的 route-builder 统一成一层导航 helper。
- 明确区分“展示 query”和“返回语义 query”。

### P3：再评估是否需要

- 是否引入真正的前端 navigation stack。
- 是否把滚动恢复从运行时内存升级到 `sessionStorage` 级别。

## 我的判断

有优化空间，而且值得做。

但我不建议直接切到“自建历史栈”。当前最合适的路径是：

1. 先把现有规则集中化。
2. 再抽出统一 navigation intent。
3. 最后再看是否还有必要引入更重的历史机制。

也就是说，当前问题的核心不是“没有历史栈”，而是“返回语义已经跨多个文件散落，缺少单一真相来源”。

## Implementation Status

- 2026-04-12: P1 and P2 implemented.
- `AppShell` now consumes a centralized back-link resolver from `src/lib/navigation-intent.ts`.
- Browse context and return intent are now separated:
  - `browse=` carries library-mode context for detail/player routes.
  - `back=` carries player return semantics (`browse`, `detail`, `history`, `curated-frames`).
- `src/lib/player-route.ts` is now a thin wrapper over the unified navigation-intent builder layer.
- `LibraryView` launches player with `back=browse`; `DetailView` launches player with `back=detail`.
- Legacy `from=history` / `from=curated-frames` is still parsed as a compatibility fallback when resolving player back targets.

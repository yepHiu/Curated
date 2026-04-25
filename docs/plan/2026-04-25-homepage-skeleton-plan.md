# 首页骨架屏方案（草案）

## 1. 背景

当前首页在以下场景下容易出现首屏体验不稳定的问题：

- 新开标签页进入首页
- 应用冷启动后首次进入首页
- 影片实际存储在机械硬盘、网络盘或响应较慢的介质上

现状并不是“完全没有加载占位”，而是：

- 卡片图片层已经有 `MediaStill.vue` 提供的图片级 skeleton
- 但首页整体没有页面级 / portal 级的 loading skeleton

这会导致一个问题：

- 图片还没来得及稳定展示时，首页真实布局已经开始渲染
- 如果电影数据本身还没 ready，页面会先以不完整结构出现，随后再整体跳变

本次目标不是让硬盘读取变快，而是让首页在数据未 ready 时先稳定展示页面结构，等数据就绪后再切换到真实内容。

## 2. 当前代码现状

### 2.1 已有能力

- `MediaStill.vue` 已经支持图片解码前显示 skeleton
- 首页卡片、Hero 图片、推荐位图片都已经复用了这层图片级占位

### 2.2 现有缺口

- `HomeView.vue` 当前直接基于 `libraryService.movies.value` 构建首页模型
- `HomepagePortal.vue` / `HomeHeroCarousel.vue` / `HomeSectionRow.vue` 当前只负责真实内容渲染
- 首页没有“页面数据未 ready 时统一显示骨架屏”的机制

也就是说，现在首页只有“图片级 skeleton”，还没有“首页级 skeleton”。

## 3. 本次确认的交互规则

本次规则已经明确：

- **只要首页数据还没 ready，就统一显示骨架屏**
- 不限定为“首次冷启动才显示”
- 不限定为“第一次打开标签页才显示”

首页渲染语义要明确区分三种状态：

- `loading`：显示首页骨架屏
- `ready`：显示真实首页内容
- `empty`：显示真实空库空态

这一点很关键，因为不能再继续把：

- `movies.length === 0`

误当成：

- “当前库为空”

它也可能只是：

- 数据还没加载完

## 4. `loaded` / `ready` 的判断依据

### 4.1 不是看什么

首页的 `loaded` / `ready` 不应该以下列条件作为判断依据：

- 图片是否已经加载完成
- 首页组件是否已经 mounted
- 每日推荐接口是否已经返回
- 某个 section 是否单独渲染完成

### 4.2 正确的判断依据

正确依据应该只有一个：

- **首页依赖的电影主列表首轮加载是否已经完成**

也就是说，这个状态属于 `libraryService` 这一层，而不属于首页 UI 自己猜测。

### 4.3 为什么必须补这个底层信号

如果没有显式的 loaded / ready 状态，前端只能看到：

- `movies = []`

但这个空数组至少可能代表三种完全不同的语义：

1. 还没开始加载
2. 正在加载，但尚未返回
3. 已经加载完成，而且库里真的没有影片

如果这三种情况都用 `movies.length === 0` 来判断，首页必然会在慢盘或慢接口场景下误把“加载中”显示成“空库”。

## 5. 结合当前仓库的最佳落点

当前仓库其实已经有一个接近的底层信号：

- [web-library-service.ts](/C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/services/adapters/web/web-library-service.ts:79) 内部已有私有变量 `loaded = false`
- [ensureLoaded()](/C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/services/adapters/web/web-library-service.ts:178) 在 `loadActiveMovies()` 完成后把它置为 `true`
- [reloadMoviesFromApiImmediate()](/C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/services/adapters/web/web-library-service.ts:162) 在重载完成后也会把它置为 `true`

所以后续真正要做的，不是重新发明一个首页专用状态，而是：

- **把 service 层已有的电影主列表加载结果暴露给上层**

## 6. 推荐状态设计

### 6.1 最小可落地版本

可以先补一个只读布尔状态：

- `moviesLoaded: boolean`

语义定义如下：

- `false`
  - 电影主列表首轮加载尚未完成
- `true`
  - 电影主列表首轮加载已经完成，并且 `moviesState` 已经得到一次明确结果
  - 这个结果可以是非空，也可以是空数组

首页据此切换：

- `!moviesLoaded` -> skeleton
- `moviesLoaded && movies.length === 0` -> empty
- `moviesLoaded && movies.length > 0` -> real content

### 6.2 更完整、长期更稳的版本

更推荐直接定义为枚举状态：

- `moviesLoadState: "idle" | "loading" | "ready" | "error"`

语义如下：

- `idle`
  - service 已创建，但首轮拉取尚未开始
- `loading`
  - 正在执行首轮 `loadActiveMovies()`
- `ready`
  - 首轮电影列表拉取成功并已写入 `moviesState`
  - 即使结果为空数组，也仍然属于 `ready`
- `error`
  - 首轮电影列表拉取失败

首页切换规则就可以写成：

- `idle/loading` -> skeleton
- `ready && movies.length > 0` -> 真实首页
- `ready && movies.length === 0` -> 真空库空态
- `error` -> 错误态或降级 skeleton

## 7. 为什么首页优先看 movies，而不是每日推荐

[HomeView.vue](/C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/views/HomeView.vue:19) 当前是这样构建首页模型的：

- 主数据源是 `libraryService.movies.value`
- 每日推荐只是 `homepageDailyRecommendations.snapshot.value ?? undefined`

而 [use-homepage-daily-recommendations.ts](/C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/composables/use-homepage-daily-recommendations.ts:21) 本身也有独立的 `loading`

但从首页是否“能稳定成型”这个角度看，真正的一手依赖仍然是电影主列表。

因此首页骨架屏的主判定应当是：

- **movies 是否 ready**

而不是：

- daily recommendations 是否 ready

推荐位可以在真实首页内部继续渐进更新，但首页外层骨架切换不应依赖它。

## 8. 推荐实施方案

推荐先做 **方案 A：页面级骨架屏 + 保留现有图片级 skeleton**

### 做法

1. 在 `libraryService` 暴露 `moviesLoaded` 或 `moviesLoadState`
2. 新增 `HomepagePortalSkeleton.vue`
3. 在 `HomeView.vue` 中按状态切换：
   - `loading` -> `HomepagePortalSkeleton`
   - `ready` -> `HomepagePortal`
   - `empty` -> 空态
4. 保留 `MediaStill.vue` 的图片级 skeleton 作为第二层渐进体验

### 为什么先做这个

- 改动范围最小
- 能直接解决“首页整体结构先塌后稳”的问题
- 与现有图片级 skeleton 是互补关系
- 不需要一开始就把每个 section 都做成复杂的分区加载状态

## 9. 验证标准

需要验证以下场景：

1. 冷启动后进入首页
2. 新开标签页进入首页
3. 慢速磁盘 / 慢响应场景

验证通过标准：

- 首屏先出现稳定的首页骨架结构
- 不再先出现碎片化真实布局
- 图片慢一点没关系，但页面框架不能塌
- 数据 ready 后从 skeleton 平稳切换到真实首页
- 空库场景不会被误判成 loading，loading 场景也不会被误判成空库

## 10. 结论

回到你刚才问的那个点，答案是明确的：

- **是的，这里需要补一个更底层的加载信号**

这个 `loaded` 的判断依据应该是：

- **电影主列表首轮加载是否已经完成**

而不是：

- `movies.length`
- 图片是否渲染出来
- 首页某个组件是否 mounted

如果只是要尽快落地，先补 `moviesLoaded` 就够用；如果希望这套语义后续可以复用到库页、标签页、演员页，直接上 `moviesLoadState` 会更稳。

## 11. 当前落地进度

截至本轮实现，已完成：

- 在 `LibraryService` 合约中补充 `moviesLoaded`
- 在 `web-library-service.ts` 中把电影主列表首轮加载状态改为可响应消费的 `moviesLoaded`
- 在 `mock-library-service.ts` 中补齐 `moviesLoaded=true` 的语义
- 新增首页页面级骨架屏组件 `HomepagePortalSkeleton.vue`
- `HomeView.vue` 已按以下三态切换：
  - `!moviesLoaded` -> 骨架屏
  - `moviesLoaded && movies.length === 0` -> 独立首页空态
  - `moviesLoaded && movies.length > 0` -> 真实首页门户
- 新增独立空态组件 `HomepageEmptyState.vue`
- 已补充回归测试，覆盖：
  - 未 ready 时显示 skeleton
  - ready 且有数据时显示真实首页
  - ready 且空库时显示独立 empty state

本轮验证已通过：

- `pnpm typecheck`
- `pnpm lint`
- `pnpm test -- src/views/HomeView.test.ts src/services/adapters/mock/mock-library-service.test.ts src/components/jav-library/settings/SettingsLoggingSection.test.ts`

### 下一步建议

如果继续优化首页体验，建议按以下顺序往下做：

1. 观察骨架屏与真实首页切换时的视觉稳定性，必要时微调 skeleton 的尺寸、密度和间距
2. 评估是否需要把 `moviesLoaded` 升级为更完整的 `moviesLoadState`
3. 如果未来首页需要“主动刷新中”反馈，再区分“首轮未 ready”和“已 ready 但后台 refresh 中”

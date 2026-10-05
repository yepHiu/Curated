# Homepage Portal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a real homepage portal to Curated, make it the default `/` entry, and keep its content driven by deterministic library data instead of hard-coded view logic.

**Architecture:** Introduce a dedicated homepage view and homepage-specific UI components under `src/components/jav-library`, but keep selection and recommendation logic in a pure `src/lib` module so it can be tested independently. The homepage should read from existing library, playback-progress, and played-movies state, while routing and sidebar navigation remain aligned with the current AppShell structure.

**Tech Stack:** Vue 3 SFCs, TypeScript, vue-router, vue-i18n, shadcn-vue primitives, Vitest, existing local storage helpers and library service adapters.

---

### Task 1: Homepage Data Assembly

**Files:**
- Create: `src/lib/homepage-portal.ts`
- Create: `src/lib/homepage-portal.test.ts`
- Read: `src/domain/movie/types.ts`
- Read: `src/lib/playback-progress-storage.ts`
- Read: `src/lib/played-movies-storage.ts`
- Read: `src/lib/random-sample.ts`

- [x] **Step 1: Write the failing test**

Write tests for:
- daily hero chooses exactly 8 movies with deterministic date-seeded ordering
- continue watching only includes unfinished progress rows
- recommendation scoring favors favorite / user-rated / recent-signal matches over unrelated movies
- recent imports are sorted by `addedAt` descending

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/lib/homepage-portal.test.ts`
Expected: FAIL because `src/lib/homepage-portal.ts` does not exist yet.

- [x] **Step 3: Write minimal implementation**

Implement pure helpers that:
- normalize and sort candidate movie pools
- build homepage sections from existing movie records plus playback / played state
- keep deterministic behavior from date seed and stable tie-breaking

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/lib/homepage-portal.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/lib/homepage-portal.ts src/lib/homepage-portal.test.ts
git commit -m "feat: add homepage portal data assembly"
```

### Task 2: Homepage View Composition

**Files:**
- Create: `src/views/HomeView.vue`
- Create: `src/components/jav-library/HomepagePortal.vue`
- Create: `src/components/jav-library/HomeHeroCarousel.vue`
- Create: `src/components/jav-library/HomeSectionRow.vue`
- Create: `src/components/jav-library/HomeContinueRow.vue`
- Read: `src/components/jav-library/MovieCard.vue`
- Read: `src/components/jav-library/MediaStill.vue`
- Read: `src/layouts/AppShell.vue`

- [x] **Step 1: Write the failing test**

Add a view-level smoke test that mounts `HomeView` with mocked library and playback state and asserts:
- the hero region renders
- the progress rail renders 8 segments
- recent and recommendation sections render expected headings

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/views/HomeView.test.ts`
Expected: FAIL because `HomeView.vue` and homepage components do not exist yet.

- [x] **Step 3: Write minimal implementation**

Build the homepage as:
- one full-bleed hero region with internal text column
- supporting section rows below the hero
- section components that reuse existing `MovieCard` or `MediaStill` rather than inventing a second card system

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/views/HomeView.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/views/HomeView.vue src/views/HomeView.test.ts src/components/jav-library/HomepagePortal.vue src/components/jav-library/HomeHeroCarousel.vue src/components/jav-library/HomeSectionRow.vue src/components/jav-library/HomeContinueRow.vue
git commit -m "feat: add homepage portal view"
```

### Task 3: Router And Sidebar Integration

**Files:**
- Modify: `src/router/index.ts`
- Modify: `src/domain/library/types.ts`
- Modify: `src/components/jav-library/AppSidebar.vue`
- Modify: `src/layouts/AppShell.vue`

- [x] **Step 1: Write the failing test**

Add a router / navigation test that verifies:
- `/` resolves to homepage instead of redirecting to library
- sidebar includes the homepage item
- homepage is treated as a primary browse surface for header behavior

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/lib/homepage-routing.test.ts`
Expected: FAIL because route names and sidebar metadata are not updated yet.

- [x] **Step 3: Write minimal implementation**

Update:
- `AppPage` union with `"home"`
- router root child route to `HomeView`
- sidebar browse group so homepage appears first
- shell header logic so homepage does not show the back button like a detail page

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/lib/homepage-routing.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/router/index.ts src/domain/library/types.ts src/components/jav-library/AppSidebar.vue src/layouts/AppShell.vue src/lib/homepage-routing.test.ts
git commit -m "feat: wire homepage into app navigation"
```

### Task 4: Localization And Finish Pass

**Files:**
- Modify: `src/locales/en.json`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/ja.json`
- Modify: `docs/plan/2026-04-12-homepage-portal-design.md`

- [x] **Step 1: Write the failing test**

Extend homepage smoke coverage so translated keys used by `HomeView` are asserted to exist via rendered output for the default locale.

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/views/HomeView.test.ts`
Expected: FAIL due to missing locale keys.

- [x] **Step 3: Write minimal implementation**

Add locale entries for:
- sidebar / nav homepage label
- hero labels
- section labels and empty states
- recommendation rationale and continue-watching wording

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/views/HomeView.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/locales/en.json src/locales/zh-CN.json src/locales/ja.json docs/plan/2026-04-12-homepage-portal-design.md
git commit -m "feat: localize homepage portal"
```

### Final Verification

- [x] Run homepage lib tests: `pnpm test -- src/lib/homepage-portal.test.ts`
- [x] Run homepage view tests: `pnpm test -- src/views/HomeView.test.ts`
- [x] Run targeted navigation tests: `pnpm test -- src/lib/homepage-routing.test.ts`
- [x] Run typecheck: `pnpm typecheck`
- [x] Run full frontend tests if targeted tests stay green: `pnpm test`

### Incremental Refinement: Hero Carousel Motion

- [x] Replace the single-image hero body with a slide track that keeps the active frame centered while exposing previous and next frames on the left and right edges.
- [x] Keep rail navigation below the hero container and drive the same animated horizontal transform for autoplay and manual selection.
- [x] Move visible movie metadata into each frame, limiting the frame copy to `title` and `code` so long titles truncate cleanly without pushing the layout.
- [x] Cover the motion structure with `src/components/jav-library/HomeHeroCarousel.test.ts` and re-run homepage view and routing regressions.
- [x] Remove the visible outer hero card container so the slide track uses a full-width stage and begins from the page edges instead of scrolling inside a narrow centered box.
- [x] Make the hero carousel loop seamlessly by prepending and appending clone slides, then snapping the track back to the canonical index after wraparound transitions complete.
- [x] Restore the progress rail to a smaller centered footprint instead of stretching it across the entire full-width hero stage.
- [x] Fix wraparound stutter by deriving slide visual state from the logical movie index instead of the physical track index, so the clone slide and canonical slide keep the same active/adjacent styling during the post-wrap snap.
- [x] Split hero transition timing so autoplay can stay smoother and slower while manual rail / preview clicks react faster.
- [x] Rebalance hero preview depth for light mode by removing hard-coded black drop shadows and using theme-aware shadows plus brightness / saturation falloff on side previews.
- [x] Make homepage taste radar chips clickable and route them into the appropriate browse filters (`tags` for exact tags, `library` for actor/studio filters).

### 2026-10-02：首页与影片页双向滚动（历史实现，2026-10-05 已取消）

- 首页底部继续向下滚动进入影片页；影片列表顶部继续向上滚动返回首页，并恢复首页离开时的位置。触屏对应为首页底部向上划、影片列表顶部向下划。
- 影片列表使用 80px 的连续滚动/触屏阈值，忽略缩放和横向手势；滚轮间隔超过 250ms 重置累计值。收藏、最近加入、回收站与批量选择模式不触发返回。
- AppShell 使用反向抽屉动画露出首页，尊重减少动画偏好。HomeView 保持单一元素根节点，使过渡钩子正常完成，支持反复来回切换。
- 验证：相关 54 项单测、Chromium 两次往返滚动及位置恢复、类型检查、改动源码 ESLint 通过。首页完整测试中原有轮播宽度断言仍不匹配（测试期待 `max-w-[54rem]`，既有组件为 `max-w-[28rem]`），与本次交互无关。

### 2026-10-02：双向滚动过渡内容交叠修正（已实现）

- 进入中的影片页原来为透明背景且未定位，页面空隙会透出仍在退出的首页。双页 opacity 动画又产生额外合成，形成内容交替的视觉效果。
- 双向过渡期间两页都绝对定位到同一容器，使用实色背景固定层级；影片页保留 `bg-background`，避免动画结束移除类名后背景变化。只动画抽屉的 transform，退出层禁用指针交互。
- Vue Transition 显式保留双页 560ms，减少动画偏好下为 0ms，替代原有用于保留静止页面的 opacity 动画。
- 验证：Chromium 35 帧采样确认影片页及列表视口高度稳定、移动方向单调、影片页不透出下层；三次往返、首页位置恢复和减少动画模式通过。现有双向滚动 e2e、46 项相关单测、类型检查和改动源码 ESLint 通过。

### 2026-10-02：影片列表底部回到顶部修正（已实现）

- 复现：从 `scrollTop=31326` 点击「回到顶部」，原生 smooth 滚动被 DynamicScroller 高度测量后的滚动锚点调整中断，停在 `28662`。
- 共用滚动 composable 改为 500ms 的 requestAnimationFrame 动画，以即时位置写入持续推进到顶部；减少动画偏好直接到顶。用户滚轮、触摸、指针或键盘输入，以及容器/浏览键切换和卸载，均终止动画。
- 返回顶部或用户滚动会使旧位置恢复回调失效，避免延迟恢复把列表拉回旧位置。
- 浏览器连续两次从底部点击均到达 `scrollTop=0`，按钮隐藏，继续上滚可返回首页。18 项相关单测覆盖虚拟列表修正、用户中断、减少动画偏好和过期恢复；类型检查与改动文件 ESLint 通过。

### 2026-10-05：取消首页与影片页滚动切换

- 按用户要求取消首页底部继续下滚进入影片页、影片页顶部继续上滚返回首页的交互；鼠标滚轮、触控板与触屏滑动均只作用于当前页面。
- 移除 HomepagePortal 与 VirtualMovieMasonry 的边界手势处理、影片页返回首页的属性/事件传递与注入入口，以及不再使用的反向抽屉动画。
- 保留首页底部「继续浏览影片」按钮及其进入动画、侧栏导航、页面内部正常滚动、影片列表「回到顶部」和详情页返回后的滚动位置恢复。
- 将原双向滚动测试更新为边界滚动不跳转验证，并保留按钮与侧栏显式导航验证；操作说明同步更新至 `docs/guide.md`。
- 验证：7 个测试文件共 61 项相关单测、`pnpm typecheck`、改动源码 ESLint 通过；Chromium e2e 验证首页底部/影片页顶部的滚轮与触屏操作不跳转，底部按钮与侧栏导航正常。

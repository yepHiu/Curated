# 影片详情返回列表时的布局闪烁

日期：2026-10-04。状态：源码修复及本地 Chromium 验证完成。

## 问题与证据

用户反馈从影片详情返回列表时，详情布局短暂变形，再显示列表。

`AppShell.vue` 的所有子路由共用 Vue `Transition`，只有首页与列表的连续滚动设置具名抽屉动画。普通导航虽然没有设置动画名称，仍默认启用 CSS 过渡；Vue 会给旧页添加 `v-leave-from` / `v-leave-active`，等待后续动画帧后才移除。

在本地 Mock 页面用 `requestAnimationFrame` 逐帧记录，返回列表后有两帧详情和列表同时挂载。详情根节点宽度从约 931 CSS px 变为约 979 CSS px，且已经应用列表路由的壳层间距。这会造成短暂的详情重排和两页同时占位。三个返回入口的初始回归测试也捕获了这个问题。

## 修复

为共享 `Transition` 设置 `:css="Boolean(homeLibraryDrawerTransition)"`：只有明确设置首页/列表抽屉动画时启用 CSS 过渡。普通路由切换同步移除旧页，不增加新的返回动画或计时器。

继续使用现有路由、返回目标、筛选参数与内部滚动机制。本次没有变更 API、配置、全局样式或页面布局尺寸。

## 本地验证

- 浏览器逐帧复查：列表出现的第一帧，旧详情节点已移除，内容区只有一个页面根节点。
- `pnpm test:e2e --grep 'movie detail returns|home and movie library'`：4 项通过，覆盖顶栏返回、Esc、浏览器后退，以及首页/列表双向连续滚动与抽屉动画。
- `pnpm test src/layouts/AppShell.test.ts src/views/DetailView.test.ts src/lib/homepage-routing.test.ts`：29 项通过。
- `pnpm typecheck`：通过。
- `pnpm exec eslint src/layouts/AppShell.vue`：通过。仓库现有 ESLint 配置不包含 `tests/e2e`，该文件通过 Playwright 执行验证。
- `git diff --check`：通过。

验证范围为本地 Chromium 导航生命周期；未运行长耗时显示缩放套件，也未执行发布。

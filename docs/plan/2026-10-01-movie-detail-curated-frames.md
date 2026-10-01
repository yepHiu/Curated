# 影片详情页萃取帧区块

## 范围与实现

- 在本地影片详情页的样本图区下方增加「萃取帧」；心愿单等只读资料页不显示。
- 复用 `listCuratedFramesPage({ movieId, limit: 12 })` 的 Web / Mock 双模式数据入口，按现有截取时间倒序展示，使用游标或偏移分页。
- 网格复用 `CuratedFrameCard`，展示缩略图、动态帧预览和影片时间点；点击打开与萃取帧库共用的 `CuratedFrameDetailDialog`。
- 共享弹窗从 `CuratedFramesLibrary` 抽出，完整保留图片轮播、影片信息、标签自动保存、原始/水印/动态导出、从时间点播放、确认删除、100% 查看和图片重试。详情页传入本片已加载帧，帧库保留时间线/演员/影片分组顺序及演员导出上下文。
- `useCuratedFrameExport` 统一帧库批量/右键与共享弹窗的原始和水印导出流程，仍通过 LibraryService 处理服务端导出。
- 提供加载、空状态、列表失败重试及加载更多；分页失败保留已加载内容。
- 切换影片时重置列表并销毁旧弹窗，丢弃过期异步结果；清理 Mock object URL。详情弹窗打开期间的帧库修订延后到关闭时刷新列表，标签保存回执立即更新卡片元数据，避免保存标签关闭弹窗或撤销正在显示的 Blob URL。
- 无新增 API、配置或数据库迁移；复用现有主题卡片与中英日文案。

## 验证

组件测试覆盖影片筛选、缩略图与原图分离、分页重试和去重、切换影片竞态、空状态、Mock URL 清理，以及只读详情不挂载区块。

2026-10-01 验证结果：

- 详情区块、DetailPage、DetailView、三语词典共 27 项测试通过；`pnpm typecheck`、相关文件 ESLint 和 `git diff --check` 通过。
- macOS Chromium 桌面与 375×812 手机视口检查：本机真实帧库为空，已确认空状态；使用仅当前浏览器生效的 13 张合成帧验证 12→13 分页、缩略图/原图分离、前后帧切换和原图弹框。未写入真实资料库。
- 局部截图保存到 `.workspace/detail-frame-viewer.png` 和 `.workspace/detail-frame-mobile.png`；未执行完整跨浏览器/DPR 的 `test:display` 套件。

## 共享组件回归（2026-10-01）

- 萃取帧组件、详情区块、导出、帧库导航与服务边界等 30 个测试文件、94 项测试通过；类型检查、相关 ESLint 通过。测试按 CI 约定使用 `NODE_OPTIONS=--no-experimental-webstorage`，避免本机 Node 原生 localStorage 与 jsdom 冲突。
- 新增共享弹窗行为验证：当前/相邻原图加载、方向键单步切换、标签失败保留草稿并重试、演员分组重复帧定位及导出归属、动态预览、带时间点播放、删除确认与失败保留。
- 浏览器使用拦截请求的合成帧验证两个页面共用弹窗、标签保存后保持打开及切换后保留、删除后列表更新；未修改真实资料库。共享弹窗截图：`.workspace/shared-frame-detail-desktop.png`、`.workspace/shared-frame-detail-mobile.png`。

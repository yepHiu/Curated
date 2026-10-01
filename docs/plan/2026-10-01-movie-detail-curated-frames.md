# 影片详情页萃取帧区块

## 范围与实现

- 在本地影片详情页的样本图区下方增加「萃取帧」；心愿单等只读资料页不显示。
- 复用 `listCuratedFramesPage({ movieId, limit: 12 })` 的 Web / Mock 双模式数据入口，按现有截取时间倒序展示，使用游标或偏移分页。
- 网格展示缩略图和影片时间点，点击用 `FrameImageViewer` 查看原图，支持按钮和左右方向键切换当前已加载帧、100% 查看与图片重试。
- 提供加载、空状态、列表失败重试及加载更多；分页失败保留已加载内容。
- 切换影片或帧库修订时重置列表，丢弃过期异步结果；清理 Mock object URL。
- 无新增 API、配置或数据库迁移；复用现有主题卡片与中英日文案。

## 验证

组件测试覆盖影片筛选、缩略图与原图分离、分页重试和去重、切换影片竞态、空状态、Mock URL 清理，以及只读详情不挂载区块。

2026-10-01 验证结果：

- 详情区块、DetailPage、DetailView、三语词典共 27 项测试通过；`pnpm typecheck`、相关文件 ESLint 和 `git diff --check` 通过。
- macOS Chromium 桌面与 375×812 手机视口检查：本机真实帧库为空，已确认空状态；使用仅当前浏览器生效的 13 张合成帧验证 12→13 分页、缩略图/原图分离、前后帧切换和原图弹框。未写入真实资料库。
- 局部截图保存到 `.workspace/detail-frame-viewer.png` 和 `.workspace/detail-frame-mobile.png`；未执行完整跨浏览器/DPR 的 `test:display` 套件。

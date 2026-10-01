# 远程 Desktop 页面不跳转：诊断与恢复

## 现场证据

- 用户使用 Desktop，点击愿望单后留在原页面，另提及萃取帧不可用。
- 本机保存的远程 Server 可访问，health 确认正式版 Server 1.7.6；已安装 Mac Desktop 为 0.2.1。
- Desktop 对应该 Server 的 HTTP 缓存存在两组旧 WishlistView / CuratedFramesView 脚本名；当前 Server 入口引用的是不同的脚本名。
- 请求缓存中的四个旧脚本，均得到 `200 text/html`（SPA 入口），而非 JavaScript。旧页面在升级后加载这些脚本时会被模块加载器拒绝，路由在提交前中止。
- 新浏览器访问同一 Server，愿望单和萃取帧均成功跳转。当前源码的 Web API 生产构建也正常。没有证据表明这些业务被远程权限限制。
- 无法通过本机 UI 自动化读取运行中的 Desktop 窗口（工具超时）；旧缓存只能证明存在失效引用，尚未直接获取用户点击当时的 renderer 错误。已请用户刷新后复验，不把缓存假设写成完成的现场修复。

## 已实现的修改

1. 在前端启动时注册 router.onError，路由加载失败显示持久提示及「刷新应用」操作。保持原页面，避免自动中断播放、上传或未保存编辑。
2. 明确刷新后打开原目标路由，保留 query；一次成功导航清除旧提示。不清除 Cookie、localStorage 或 IndexedDB。Toaster 从 AppShell 移到 App 根部，使首次进入页面失败时也能显示提示。
3. Server 静态托管对缺失的 `/assets/` 文件返回 no-store 404，普通 SPA fallback 和现有文件缓存策略保持原行为。
4. 更新中／英／日提示文案。

## 验证与交付边界

- 真实 Router 测试覆盖愿望单／萃取帧模块加载失败、原路由保留、刷新操作提示、后续成功导航清除提示；刷新辅助函数覆盖 hash 和 query 保留。
- Go webui 测试覆盖缺失 JS／CSS 404、入口 no-store、已有 hash 资源 immutable 与 SPA fallback。
- 目标前端测试 5 文件 41 用例、ESLint、类型检查、Web API 生产构建通过。测试使用文档约定的 `NODE_OPTIONS=--no-experimental-webstorage`，避免 Node 原生 localStorage 干扰 jsdom；首次未设置该参数时 AppShell 套件初始化失败，按标准参数重跑通过。
- 使用隔离浏览器对生产构建注入 `text/html` 失效脚本响应：点击愿望单后保留资料库页面，出现持久刷新提示；恢复脚本响应后点击「刷新应用」，新文档成功打开 `/wishlist`，显示愿望单标题、筛选、空态及计数。
- 首次直达 `/wishlist` 并注入脚本 404 时，根部刷新提示仍可见，不依赖 AppShell 已挂载。
- 修改仅在本地源码；没有发布或修改运行中的 Server。已加载的旧前端不会自动获得新提示，现阶段需要 ⌘R／Ctrl+R 或重连。

## 现场恢复

在 Desktop 按 macOS ⌘R / Windows Ctrl+R，再点击愿望单或萃取帧；也可重连同一 Server。如果仍失败，继续获取 Desktop renderer 控制台错误，区分脚本缺失、网络异常与其他运行时错误。

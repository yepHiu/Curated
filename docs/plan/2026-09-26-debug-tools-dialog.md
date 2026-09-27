# Debug 工具弹窗

## 设计决定

开发壳层的 DEV / PERF 旁增加 DEBUG 按钮，打开页面内 Dialog。复用 Dialog、Tabs、SettingsHint、SettingsScopeBadge 和既有设置卡片，标题短、密度紧凑，说明在标题下方 Tooltip，内容限高并内部滚动。

- 操作：配置重新读取、已保存代理连通检测、每日推荐刷新、同机强制 HLS 开关。
- 性能：复用现有监控实例及请求统计，暂停、清空、复制摘要；原 Perf 条点击进入同一个弹窗。
- 日志：复用服务端日志配置和客户端日志等级；先读取配置，失败展示重试，禁止未初始化时保存。
- Debug 仅开发环境提供；设置页移除开发触发项，正式环境仍保留日志设置。扫描、备份等日常维护留在原位置。
- 弹窗打开时隐藏底部监控条，徽标降到遮罩下；关闭后恢复焦点。窄屏允许换行，执行中禁用重复操作，失败就地反馈。
- 普通播放设置不再写入强制 HLS 值，仅显式关闭推流时同时清除强制开关，避免旧草稿覆盖 Debug 操作。

## 验证

- 相关 dev/settings/service boundary/i18n 测试：48 文件、223 项通过；补充 Perf 单实例与关闭推流清理 HLS 回归后，针对性 21 项通过。
- 改动文件 ESLint、`pnpm typecheck`、Web API 生产构建通过；体积监管无提醒/超限。
- 独立 Playwright Chrome 浏览器，1280×720 与 375×812，默认 DPR/100% 缩放：检查三个 Tab、窄屏边界和内部滚动、隐藏 Perf 后 Debug 可达、Perf 直达性能页、Escape 后两个入口的焦点恢复。截图保存在本地 `.workspace/debug-{dialog,performance,logs,narrow}.png`。
- 浏览器只读验证，没有触发真实推荐刷新、外部连通测试或配置写入；动作成功/失败、防重复、Mock/remote 限制由组件测试覆盖。
- 未运行 `pnpm test:display`，未覆盖 Safari/Firefox、其它 DPR 与操作系统缩放。

## 模拟远程端

Debug 操作分区新增客户端开关，使用共享的开发态内存状态立即覆盖前端本机能力判断。路径、播放高级项、强制 HLS 和服务器更新操作跟随现有远端限制；关闭仅恢复真实探测结果，不授予真实远端本机权限。启用时 DEBUG 标签显示 remote，关闭弹窗或切换路由仍有效，刷新页面清除。不改网络目标、服务器配置或服务端权限，不模拟网络延迟或断线。入口、弹窗及状态 setter/read 均受 DEV 限制，正式构建不能启用。

验证：8 个相关测试文件、71 项通过，覆盖开发/生产开关、真实远端不能提权、Mock/本机即时切换、播放保存不携带隐藏字段，以及更新写操作拦截。类型检查、改动文件 ESLint 和 Web API 生产构建通过。

## 连接页 Debug 工具（2026-09-27）

连接页属于 Desktop 本地壳层，断开 Server 时仍需可用。开发版连接页沿用内容页右下角 `DEBUG` 入口和限高 Dialog，但只提供本机状态与连接诊断：Desktop 构建信息、当前/上次连接、已保存数量、运行中的代理模式、无凭据的服务器身份探测，以及连接页 DevTools。探测复用正常连接的地址规范化与身份校验，不保存地址、不切换连接、不携带会话。正式版隐藏入口，主进程拒绝诊断 IPC；调用者必须是随包连接页主 frame。弹窗有加载、失败、重试和键盘关闭/焦点返回，保持连接表单为页面主任务。

验证：连接页及新 Debug 弹窗 7 项测试、前端类型检查、相关 ESLint、Electron/launcher 构建通过。真实 Windows Electron 独立临时 profile 验证只读探测、本机 fixture 的 `/api/server-info` 无 Cookie、记录/当前连接不变、DevTools 打开、无效地址失败、Escape 焦点恢复和正式状态 IPC 拒绝。深浅色及 520×540 最小窗口、当前系统 DPR 1.5 下无横向越界且内容可滚动；截图 `.workspace/connection-debug/`。未运行 `pnpm test:display` 或完整跨平台缩放矩阵；未生成安装包或发布。

独立 Chrome 1280×720 验证开关即时生效、DEBUG · REMOTE 状态、关闭重开保留、强制 HLS 隐藏/恢复、刷新页面重置；截图 `.workspace/debug-remote.png`。未运行完整显示缩放套件。

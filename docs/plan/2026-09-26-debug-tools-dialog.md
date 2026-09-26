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

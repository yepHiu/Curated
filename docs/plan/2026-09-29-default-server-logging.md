# 服务端日志默认目录与本机访问

用户要求日志目录交由应用管理，设置页提供本机 Desktop 打开入口，服务端策略只允许本机配置；前端日志设置保持原样。

## 已实现

- 配置加载统一采用应用默认日志目录，兼容但忽略旧 `logDir`；保存策略清除此库配置键，不搬移或删除旧文件。
- 移除绝对路径输入和选择目录控件。本机 Desktop 显示打开按钮；本机 Web 仍可改保留时间、日志等级。加载未完成时禁用服务端操作。
- 新增无路径参数的 `POST /api/settings/logs/reveal`；Server 自己创建/打开默认目录，沿用直接 loopback 权限，拒绝远端、转发和远端 Origin。
- 前端日志设置及默认值不变。保留时间、等级仍在 Server 重启后生效。

## 验证范围

组件测试覆盖本机 Desktop 打开/失败重试、本机 Web、远端 Desktop、策略保存不提交目录。Go 测试覆盖主/库配置中的旧路径忽略、保存时清除旧键、远端策略拒绝、打开目录接口权限与系统打开失败。正式构建的默认目录另以 release tag 配置测试验证。实际文件管理器弹窗需启动新版 Server 后验收。

## 验证结果

- 日志组件与 Web 服务测试：52 项通过。
- `go test ./internal/config ./internal/app ./internal/server` 通过。
- `go test -tags release ./internal/config` 通过。
- 本次前端文件 ESLint 通过。
- `pnpm build` 通过（含全量 TypeScript 检查，体积报告 0 项提醒/超限）。
- 未启动新版 Server 操作真实系统文件管理器，未执行跨端显示缩放套件。

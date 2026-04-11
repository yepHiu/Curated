# 整包打包历史

用于记录 Curated（仓库目录仍可能显示为 `jav-shadcn`）每次整包打包的关键信息，便于后续追踪构建产物、回溯问题和核对发布内容。

## 记录规则

- 本文件是 Curated 的打包版本台账；后续只要发生实际打包产物输出，就继续在文末追加，不覆盖历史记录。
- 每次执行整包打包后追加一条记录。
- 如果一次打包同时生成多个产物，尽量合并在同一条记录中描述。
- `版本` 优先填写打包脚本实际使用的 `-Version`；如果与 `package.json` 的 `version` 不一致，在 `备注` 中说明。
- 整机安装包与正式发布流程在开包前必须先读取本台账最近一条有效发布记录，再沿着该版本历史确定新版本；不要绕开台账单独起版本号。
- 同一次整机发布中的安装包文件名、发布清单、命令参数 `-Version` 与本台账的 `版本` 列必须保持一致。
- `提交 / 分支` 建议记录打包时的 commit short SHA 与当前分支，便于后续回溯。
- `状态` 建议使用：`成功`、`失败`、`已验证`、`待验证`。
- `产物路径` 写相对仓库路径，便于跨机器查看。
- `备注` 记录构建环境、特殊参数、已知问题、补丁说明等。

## 常用脚本

- 前端构建：`pnpm build`
- 前端发布包：`pnpm release:frontend`
- 后端发布包：`pnpm release:backend`
- 便携包：`pnpm release:portable`
- 安装包：`pnpm release:installer`
- 发布流程：`pnpm release:publish`

## 历史记录

| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 2026-04-01 | 0.0.0-local | `a52143a` / `master` | `release:publish` | `release/portable/Curated-0.0.0-local-windows-x64.zip`；`release/installer/Curated-Setup-0.0.0-local.exe` | 成功 | Codex | BuildStamp=`20260401.151209`；同时生成 `release/manifest/release.json`；为兼容当前 Windows 环境，发布脚本改为前端使用 `vite --configLoader native`、后端使用仓库内 `.gocache/` |
| 2026-04-02 | 0.0.1-master | `c517ddb` / `master` | `release:publish` | `release/portable/Curated-0.0.1-master-windows-x64.zip`；`release/installer/Curated-Setup-0.0.1-master.exe` | 成功 | Codex（补记） | BuildStamp=`20260402.160705`；`release/manifest/release.json` 生成时间为 `2026-04-02T16:07:26Z`；SHA256 已写入 manifest；本条根据现存 manifest、产物时间戳与 git 历史补记。 |

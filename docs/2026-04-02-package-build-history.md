# 整包打包历史

用于记录 Curated（仓库目录仍可能显示为 `jav-shadcn`）每次整包打包的关键信息，便于后续追踪构建产物、回溯问题和核对发布内容。

## 记录规则

- 本文件是 Curated 的打包版本台账；后续只要发生实际打包产物输出，就继续在文末追加，不覆盖历史记录。
- 每次执行整包打包后追加一条记录。
- 如果一次打包同时生成多个产物，尽量合并在同一条记录中描述。
- `版本` 填写本次打包实际使用的生产包版本号；该版本号默认来自 `scripts/release/version.json` 自动分配的 `major.minor.patch`，而不是 `package.json` 的 `version`。
- 当前生产包版本基线为 `1.1.0`。`pnpm release:portable`、`pnpm release:installer`、`pnpm release:publish` 在未显式传入 `-Version` 时都会自动把 `patch` 加 1；`major` / `minor` 仅允许通过 `scripts/release/set-version-base.ps1` 人工调整，并在调整时把 `patch` 重置为 `0`。
- 同一次整机发布中的安装包文件名、便携包文件名、`release/manifest/release.json`、命令参数 `-Version` 与本台账的 `版本` 列必须保持一致。
- `提交 / 分支` 建议记录打包时的 commit short SHA 与当前分支，便于后续回溯。
- `状态` 建议使用：`成功`、`失败`、`部分成功`、`已验证`、`待验证`。
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

| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 变更内容 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 2026-04-01 | 0.0.0-local | `a52143a` / `master` | `release:publish` | `release/portable/Curated-0.0.0-local-windows-x64.zip`；`release/installer/Curated-Setup-0.0.0-local.exe` | 成功 | Codex | 历史记录补齐前未采集 | BuildStamp=`20260401.151209`；同时生成 `release/manifest/release.json`；为兼容当前 Windows 环境，发布脚本改为前端使用 `vite --configLoader native`、后端使用仓库内 `.gocache/` |
| 2026-04-02 | 0.0.1-master | `c517ddb` / `master` | `release:publish` | `release/portable/Curated-0.0.1-master-windows-x64.zip`；`release/installer/Curated-Setup-0.0.1-master.exe` | 成功 | Codex（补记） | 历史记录补齐前未采集 | BuildStamp=`20260402.160705`；`release/manifest/release.json` 生成时间为 `2026-04-02T16:07:26Z`；SHA256 已写入 manifest；本条根据现存 manifest、产物时间戳与 git 历史补记。 |
| 2026-04-12 | 1.1.1 | 6d16780c / master | release:publish | release/portable/Curated-1.1.1-windows-x64.zip | 失败 | wujiahui | 历史记录补齐前未采集 | versionSource=explicit; BuildStamp=20260412.062321; error=The property 'ArtifactPaths' cannot be found on this object. Verify that the property exists. |
| 2026-04-12 | 1.1.1 | 6d16780c / master | release:publish | release/portable/Curated-1.1.1-windows-x64.zip; release/installer/Curated-Setup-1.1.1.exe; release/manifest/release.json | 失败 | wujiahui | 历史记录补齐前未采集 | versionSource=explicit; BuildStamp=20260412.063010; error=The property 'Count' cannot be found on this object. Verify that the property exists. |
| 2026-04-12 | 1.1.1 | 6d16780c / master | release:publish | release/portable/Curated-1.1.1-windows-x64.zip; release/installer/Curated-Setup-1.1.1.exe; release/manifest/release.json | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=explicit; BuildStamp=20260412.063217; portable=versionSource=explicit; installer=versionSource=explicit |
| 2026-04-12 | 1.1.2 | 6d16780c / master | release:portable | release/portable/Curated-1.1.2-windows-x64.zip | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=auto-patch |
| 2026-04-12 | 1.1.2 | 6d16780c / master | release:portable | release/portable/Curated-1.1.2-windows-x64.zip | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=explicit |
| 2026-04-12 | 1.1.2 | 6d16780c / master | release:installer | release/installer/Curated-Setup-1.1.2.exe | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=explicit |
| 2026-04-12 | 1.1.3 | d58e8d85 / master | release:publish | release/portable/Curated-1.1.3-windows-x64.zip; release/installer/Curated-Setup-1.1.3.exe; release/manifest/release.json | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=auto-patch; BuildStamp=20260412.085211; portable=versionSource=explicit; installer=versionSource=explicit |
| 2026-04-12 | 1.1.4 | d58e8d85 / master | release:publish | release/portable/Curated-1.1.4-windows-x64.zip; release/installer/Curated-Setup-1.1.4.exe; release/manifest/release.json | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=auto-patch; BuildStamp=20260412.100459; portable=versionSource=explicit; installer=versionSource=explicit |
| 2026-04-13 | 1.2.1 | 8b1ef9b0 / master | release:publish | release/portable/Curated-1.2.1-windows-x64.zip; release/installer/Curated-Setup-1.2.1.exe; release/manifest/release.json | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=auto-patch; BuildStamp=20260412.173621; portable=versionSource=explicit; installer=versionSource=explicit |
| 2026-04-13 | 1.2.1 | 8b1ef9b0 / master | release:installer | release/installer/Curated-Setup-1.2.1.exe | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=explicit |
| 2026-04-13 | 1.2.2 | cacfa022 / master | release:publish | release/portable/Curated-1.2.2-windows-x64.zip; release/installer/Curated-Setup-1.2.2.exe; release/manifest/release.json | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=auto-patch; BuildStamp=20260412.180713; portable=versionSource=explicit; installer=versionSource=explicit |
| 2026-04-14 | 1.2.3 | 89131f08 / master | release:publish | release/portable/Curated-1.2.3-windows-x64.zip; release/installer/Curated-Setup-1.2.3.exe; release/manifest/release.json | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=auto-patch; BuildStamp=20260413.162833; portable=versionSource=explicit; installer=versionSource=explicit |
| 2026-04-16 | 1.2.4 | 7a7b2981 / master | release:publish | release/portable/Curated-1.2.4-windows-x64.zip; release/installer/Curated-Setup-1.2.4.exe; release/manifest/release.json | 成功 | wujiahui | 历史记录补齐前未采集 | versionSource=auto-patch; BuildStamp=20260415.165606; portable=versionSource=explicit; installer=versionSource=explicit |
| 2026-04-17 | 1.2.5 | 4319daf7 / master | release:publish | release/portable/Curated-1.2.5-windows-x64.zip; release/installer/Curated-Setup-1.2.5.exe; release/manifest/release.json | 成功 | wujiahui | 4319daf7 docs(plan): home and detail sidebar optimization follow-up<br>063793dd feat(frontend): home scroll preserve, sidebar layout, expandable text<br>51e59511 docs(plan): curated frame menu and home detail sidebar notes<br>0f8d371f feat(frontend): context menu for curated frames library<br>c58003e9 chore(installer): remove AppMutex from Inno Setup template<br>696916fc docs: package ledger change column and packaging strategy<br>b877ad1a feat(scripts): auto-fill package history change column<br>06898596 docs: homepage daily recommendations API and plans<br>b2b2f092 chore(release): bump package version to 1.2.4 and record publish<br>f4e71dc2 feat(frontend): homepage daily picks, portal, and player shortcuts<br>b71b1775 feat(backend): persist homepage daily recommendations | versionSource=auto-patch; BuildStamp=20260416.164557; portable=versionSource=explicit; installer=versionSource=explicit |
| 2026-04-17 | 1.2.6 | 4319daf7 / master | release:publish | release/portable/Curated-1.2.6-windows-x64.zip; release/installer/Curated-Setup-1.2.6.exe; release/manifest/release.json | 成功 | wujiahui | 无代码差异（同一提交重复打包） | versionSource=auto-patch; BuildStamp=20260416.165419; portable=versionSource=explicit; installer=versionSource=explicit |

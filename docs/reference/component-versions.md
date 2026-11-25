# 组件版本与 macOS 引擎

这里是现行版本和 macOS 原生引擎边界的唯一说明。其它 README、手册、`CLAUDE.md` 和 `.cursor/rules` 里的日期段落如果写出具体版本号，只代表写那段时的快照。数字以本节列出的文件为准；文件和本节不一致时，改文件的那个变更应同时改本节表格。

查看当前 Server / Desktop 源码版本：

```bash
pnpm release:version:show
```

该命令只读下面两个 JSON，不访问网络（`scripts/release/prepare_batch.py --versions`）。

## 现行源码版本

| 组件 | 唯一版本文件 | 本节核对时的值 | 谁读取 |
| --- | --- | --- | --- |
| Server | `backend/internal/version/server.json` 的 `current.major/minor/patch` | 1.8.0 | `backend/internal/version/version.go` 用 `go:embed` 作为健康检查里的产品版本 |
| Desktop | `scripts/release/versions/desktop.json` 的同样字段 | 0.2.7 | `scripts/release/release_lib/components.py`、`batches.py`、`macos_desktop.py` |

不要把下面两个文件当成现行 Server 或 Desktop 版本：

- `scripts/release/version.json` 仍是旧一体包版本文件（本节核对时为 1.5.8）。`version.go` 里的 `InstallerVersion` 注释仍把它当作发布注入例子。
- `scripts/release/versions/full.json` 是历史 Full 组件记录（本节核对时为 1.7.3），不是 Server 健康检查版本。

`package.json` 的 `version` 也不是产品版本。

## Server / Desktop 拆分

独立安装包和旧一体包迁移已经在源码里：

- Windows 组件包：`scripts/release/component_cd.py`、`scripts/release/release_lib/windows_components.py`
- Apple Silicon Desktop：`scripts/release/release_lib/macos_desktop.py`
- 旧一体包迁移辅助：`backend/cmd/curated-migrate`

仓库里提交的 `electron/release-config.json` 仍是 `distribution: legacy`、`updateFeed: null`。因此默认的开发 Electron 构建不会自己去拉正式 Desktop 更新源。这和「拆分包已经能构建」是两件事。

## macOS libmpv

正式 Mac Desktop 内置 libmpv，用户不需要安装 mpv 或 Homebrew。

- 打包前要准备绝对路径 `CURATED_NATIVE_MAC_BUNDLE`。`scripts/release/release_lib/macos_native.py` 要求包内有 `libmpv.dylib`，并把依赖收成同目录 `@loader_path`。该文件写明：Homebrew 只作为构建时的 bottle 来源，安装后的应用不再探测它。
- 未打包的开发态才使用 `CURATED_NATIVE_LIBMPV`，否则依次尝试 `/opt/homebrew/opt/mpv/lib/libmpv.dylib` 和 `/usr/local/opt/mpv/lib/libmpv.dylib`。已打包应用忽略这些覆盖，只加载 `native-player-macos/libmpv.dylib`。见 `electron/playback-resources.ts`。
- 锁定文件是 `scripts/release/native-player/macos-arm64-production.json`：`engine` 为 `libmpv`，`arch` 为 `arm64`，`minimumMacOS` 为 `15.0`，`packages.mpv.version` 为 `0.41.0_7`（上游源码标签 `v0.41.0`）。
- 准备脚本用 `codesign --sign -` 做 ad-hoc 签名。本仓库的该脚本里没有 Apple 公证步骤。
- `electron/native/include/mpv/` 只有 ISC 头文件，不是随包运行的 dylib。

Windows 正式原生播放用的是独立 `mpv.exe` 和另一份 lock，不是这份 macOS libmpv。

某个 GitHub Release 是否已经公开、用户下载到的包是否包含当时的引擎，要以该 Release 的资产为准。本节只描述当前源码会怎样打包和加载。

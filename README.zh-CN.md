<p align="center">
  <img src="icon/curated-wordmark.png" alt="Curated" width="520" />
</p>

<p align="center">
  <a href="README.md">English</a> | 简体中文 | <a href="README.ja-JP.md">日本語</a>
</p>

<p align="center">
  <img alt="Vue 3" src="https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vuedotjs&logoColor=white">
  <img alt="TypeScript 5.x" src="https://img.shields.io/badge/TypeScript-5.x-3178c6?style=flat-square&logo=typescript&logoColor=white">
  <img alt="Vite 8.x" src="https://img.shields.io/badge/Vite-8.x-646cff?style=flat-square&logo=vite&logoColor=white">
  <img alt="Go 1.25+" src="https://img.shields.io/badge/Go-1.25+-00add8?style=flat-square&logo=go&logoColor=white">
  <img alt="SQLite modernc" src="https://img.shields.io/badge/SQLite-modernc-003b57?style=flat-square&logo=sqlite&logoColor=white">
  <img alt="Tailwind CSS v4" src="https://img.shields.io/badge/Tailwind_CSS-v4-06b6d4?style=flat-square&logo=tailwindcss&logoColor=white">
  <img alt="shadcn-vue" src="https://img.shields.io/badge/shadcn--vue-ui-111111?style=flat-square">
  <img alt="Windows tray ready" src="https://img.shields.io/badge/Windows-tray_ready-0078d4?style=flat-square&logo=windows&logoColor=white">
</p>

# Curated

Server 使用深色底图标，Desktop 使用浅灰白底图标（`#F5F6F8`），中央图案保持一致。详见[品牌规范](docs/reference/curated-brand-guidelines.md)与[资源预览](icon/brand/index.html)。

Desktop 可从设置或托盘保存、管理和切换服务器；详见[服务器连接说明](docs/guide.md#desktop-server-connections)。

萃取帧现支持截图预览、重试与撤销、源文件高清帧，以及 GIF/MP4/WebM 片段；详见[操作手册](docs/guide.md#curated-capture-and-inspection)。

Curated 是本地优先的媒体资料库：Vue 3 前端、Go + SQLite 后端，以及 Electron 桌面壳。产品正式名称是 **Curated**。仓库目录和 npm 包名仍可能使用 **`jav-shadcn`**。

本 README 只做公开入口。启动细节、配置、打包，以及 `docs/` 里各篇文章的索引，见 **[docs/guide.md](docs/guide.md)**。HTTP API 参考是 **[API.md](API.md)**。

## 下载

下一版 Windows 安装流程已在源码实现（Server 1.7.4 / Desktop 0.2.2）：独立安装位置页、明确的版本与覆盖确认、仅在程序运行时提示关闭。详见[安装流程](docs/guide.md#windows-installation-flow)；源码版本不代表已经发布。

Full/Server 1.7.3、Desktop 0.2.1 已补齐 Windows 安装器中文与自动退出，Full 在备份校验后自动卸载旧一体版；新包已通过 Windows/Mac CD 验收并发布，支持范围见[迁移指南](docs/guide.md#migrating-an-old-all-in-one-installation)。

Release 按 **Curated YYYYMMDD** 命名，同日后续追加 **-2**、**-3**。一次 Release 包含有更新的 Server、Desktop 或两端独立包；两端分别递进版本、检查更新。详见[发布操作](docs/guide.md#8-release-and-packaging)。

- 漫画库 / 写真库 **Beta**：在「设置 → 资料库 → 漫画库 / 写真库」分别开启，开启后才显示配置。写真库页支持批量收藏、追加标签和删除索引。见 [试用说明](docs/guide.md#comic-and-photo-library-beta)。
- **Windows 安装器：** `Curated-Server-Setup-<version>-windows-x64.exe` 与 `Curated-Desktop-Setup-<version>-windows-x64.exe`；各有独立 ZIP。
- **Apple Silicon Desktop：** `Curated-Desktop-<version>-macos-arm64.dmg` / `.zip`。

不再出新 Full 包。首个日期批次发布后，Latest 跟随日期批次；旧一体版先通过 [Full 1.7.3](https://github.com/yepHiu/Curated/releases/tag/full-v1.7.3) 完成一次迁移，旧独立 Server 通过过渡版接入新更新渠道。

## 亮点

- 浏览器插件联动：在「设置 → 访问与连接 → 网络与设备」开启即可使用，无需凭证；新版插件同步愿望单状态，已加入显示绿色；加入时保存来源页，详情以 JAVDB 等站点名显示超链接；关闭联动后保留已有愿望单。详见 [使用说明](docs/guide.md#wishlist)。

- 本地优先：Vue 3 SPA、Go HTTP API、SQLite，以及面向 Windows 的 Electron 托盘壳。
- macOS 桌面端使用融合标题栏，保留原生红黄绿窗口按钮。
- 双模式开发：真实 Web API 与 Mock UI 共用同一服务层。
- 内容区右侧 Agent 面板，支持自动整理上下文与持久会话摘要；作品引用经后端填入事实并在展示前校验；工具调用默认不限，可选上限；写入明确区分待确认与已保存，AI 设置支持模型上下文容量配置与预设，并提供权限、实际 token 用量、耗时与审计；详见[操作手册](docs/guide.md#ai-settings-and-governance)。
- 资料库浏览、刮削、导入、播放、萃取帧、演员身份、每日推荐与个人洞察。
- 演员合并支持搜索选人、选择保留身份与预览最终资料；见[演员合并说明](docs/guide.md#merging-duplicate-actors)。
- 独立 HLS 会话、指定位置续播与有界错误恢复，详见[播放说明](docs/guide.md#playback-recovery)。
- 可选 PIN 锁、可验证备份、路径迁移、资料库健康与有界修复。
- 基于 GitHub Releases 的更新检查与安装器下载；正式包内置 FFmpeg 与本地 `hls.js`。

## 快速开始

环境：与 Vite 8 兼容的当前 Node.js LTS、**pnpm**、Go `1.25.4+`。

```bash
cd backend && go run ./cmd/curated
```

```bash
pnpm install
pnpm dev
```

- 后端默认：`http://127.0.0.1:8080`（仅 loopback），健康名 `curated-dev`。
- 前端默认：`http://localhost:5173`。
- 根目录 `.env` 设置 `VITE_USE_WEB_API=true` 走真实 API，否则为 Mock。
- 桌面壳：`pnpm dev:electron`。
- Windows 开发二进制：`pnpm backend:build:dev` → `backend/runtime/curated-dev.exe`。

备份、恢复、路径迁移、配置项和发布打包见 [docs/guide.md](docs/guide.md)。CD 构建 Windows Server / Desktop 独立包与 Apple Silicon Mac Desktop，支持批次标签发布和手动草稿运行，详见[发布操作](docs/guide.md#8-release-and-packaging)。

## 文档

| 文档 | 用途 |
| --- | --- |
| [docs/guide.md](docs/guide.md) | 详细手册与文档索引 |
| [API.md](API.md) | 公开 HTTP API 参考 |
| [docs/features/2026-05-03-feature-inventory.md](docs/features/2026-05-03-feature-inventory.md) | 已实现 / 目标功能目录 |
| [docs/README.md](docs/README.md) | `docs/` 子目录怎么用 |

## 仓库结构

```text
src/        Vue SPA
backend/    Go 模块 curated-backend（`cmd/curated`）
electron/   桌面壳 MVP
config/     资料库级运行配置（保留在仓库根目录）
docs/       手册、规范、产品、运维、计划、PRD
icon/       品牌源文件（wordmark / appicon / mark）
```

源码、测试、必需资源以及全部文档和原型保留在 Git；本地产物遵循[仓库跟踪规则](docs/guide.md#repository-tracking-policy)。

构建体积监管允许正常增长、提醒异常增长，仅在严重超限时阻断；详见[体积报告与规则](docs/guide.md#build-size-monitoring)。

## 说明

- 当前阶段是 Web 优先 + 最小 Electron 壳。更深的 IPC、mpv 与广泛原生桥接仍是目标方向。
- `docs/film-scanner/` 是参考材料，不是生产模块树。
- 远程端存储目录只读，增删改和默认目录设置在服务器所在电脑完成；上传仍使用服务端目标，见[目录管理说明](docs/guide.md#remote-storage-directory-access)。
- 远程端保留设备偏好；全局设置、安全、AI 管理和备份维护在 Server 本机操作，见[远程设置说明](docs/guide.md#remote-settings-access)。
- 服务端日志固定使用默认目录，本机 Desktop 可直接打开，见[日志设置](docs/guide.md#server-logging)。
- 远程 Desktop 仅提供自身更新入口；Server 安装操作仅限明确本机直连。Desktop 目前提供手动下载，见[更新说明](docs/guide.md#update-target-and-remote-connections)。
- 关于页分别显示 Server 与 Desktop 的独立三段数字版本，并保留 Server 构建时间戳；组件包名规则已落地，独立安装器和迁移仍待完成，见[操作指南](docs/guide.md#desktop-version-and-component-release-planning)。

## 参与过本仓库的模型

后续若有大模型接手本仓库工作，请把名称追加到下面。

- Composer 2.5
- DeepSeek V4
- GPT 5.5
- GPT 5.6 Terra sol
- Grok 4.6

Server identity and LAN discovery, Desktop device settings, upgrade compatibility and release packaging: [guide](docs/guide.md).

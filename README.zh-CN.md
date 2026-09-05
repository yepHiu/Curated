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

Curated 是本地优先的媒体资料库：Vue 3 前端、Go + SQLite 后端，以及 Electron 桌面壳。产品正式名称是 **Curated**。仓库目录和 npm 包名仍可能使用 **`jav-shadcn`**。

本 README 只做公开入口。启动细节、配置、打包，以及 `docs/` 里各篇文章的索引，见 **[docs/guide.md](docs/guide.md)**。HTTP API 参考是 **[API.md](API.md)**。

## 下载

正式 Windows 安装包和便携包发布在 **[GitHub Releases](https://github.com/yepHiu/Curated/releases)**。请使用 [最新 Release](https://github.com/yepHiu/Curated/releases/latest)。

- **安装器（推荐）：** `Curated-Setup-<version>.exe`
- **便携包：** `Curated-<version>-windows-x64.zip`

已安装的应用也可以在 设置 → 关于 中检查并下载新的安装器。

## 亮点

- 本地优先：Vue 3 SPA、Go HTTP API、SQLite，以及面向 Windows 的 Electron 托盘壳。
- 双模式开发：真实 Web API 与 Mock UI 共用同一服务层。
- 资料库浏览、刮削、导入、播放、萃取帧、演员身份、每日推荐与个人洞察。
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

备份、恢复、路径迁移、配置项和发布打包见 [docs/guide.md](docs/guide.md)。

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

## 说明

- 当前阶段是 Web 优先 + 最小 Electron 壳。更深的 IPC、mpv 与广泛原生桥接仍是目标方向。
- `docs/film-scanner/` 是参考材料，不是生产模块树。

## 参与过本仓库的模型

后续若有大模型接手本仓库工作，请把名称追加到下面。

- Composer 2.5
- DeepSeek V4
- GPT 5.5
- GPT 5.6 Terra sol
- Grok 4.6

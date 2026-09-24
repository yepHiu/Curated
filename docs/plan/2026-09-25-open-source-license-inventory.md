# Curated “关于”页开源组件许可清单

2026-09-25，按仓库当前依赖和 Windows 打包产物核对“关于 → 开源项目使用许可”的扩展范围。

## 展示决定

- 默认保留 Curated MIT 与三套字体的简短条目；另外用可展开的“更多开源组件”列出 15 个有代表性的前端、Go 后端及桌面运行时组件。展开后只显示名称与许可类型，完整许可和署名集中在本地 `src/assets/licenses/ThirdParty_NOTICES.txt`。
- 清单基于 `package.json`、`backend/go.mod` 中实际使用的组件与 `release/Curated/resources/app/third_party/ffmpeg/` 当前 Windows 包。它是精选目录，**不是完整依赖树或 SBOM**；例如构建工具和间接依赖并未全部逐项展示。
- HarmonyOS Sans SC 不是开放字体；仍沿用用户指定的区块标题，并在单独条目展示 Huawei 署名及其独立许可，避免误写为 MIT 或 OFL。
- FFmpeg 许可随具体二进制变化。核对的 Windows 包为 `8.0.1-full_build-www.gyan.dev`，`ffmpeg -L` 声明 GPL 第 3 版或以后版本，许可原文取自该包附带的 `LICENSE`。今后更换打包来源时，应重新核对二进制、更新该条目及随前端分发的原文。

## 核对来源

| 范围 | 组件 | 许可原文来源 |
| --- | --- | --- |
| 前端 | Vue、Vue Router、Vue I18n、Reka UI、hls.js、DOMPurify、Lucide、pinyin-pro | 当前 `node_modules` 对应版本中的 `LICENSE`；DOMPurify 同时保留 Apache-2.0 与 MPL-2.0 两份原文。 |
| 后端 | GORM、go-sqlite、fsnotify、MetaTube SDK、Zap | `backend/go.mod` 对应版本的 Go module 缓存中的 `LICENSE`。GORM 经当前后端依赖间接引入。 |
| 桌面 | Electron、FFmpeg | 当前 `node_modules/electron/LICENSE`；当前 Windows 包的 FFmpeg `LICENSE` 与 `ffmpeg -L`。 |

`ThirdParty_NOTICES.txt` 保留版本、许可标识、核对来源和原始全文，随 `dist` 本地打包。独立的字体与项目许可文件仍提供各自的入口。

# Curated 品牌资源整理与交付

2026-09-27。用户要求参照此前已经完成的字体、图标与配色，整理 Curated、Curated Server、Curated Desktop、Curated App / Curated Web 的品牌资源；并明确 App 是 Android / iOS 移动客户端。

## 已采用的依据

- 沿用 `icon/curated-wordmark.png` 与 `icon/curated-mark.png` 原始图案和主字标。
- 沿用品牌粉色 `#FE628E`、Outfit 品牌字体、Noto Sans / HarmonyOS Sans SC / Noto Sans JP 正文分工。
- Server / Web 保留现有深色 `#141826`；Desktop / App 使用本轮已确认的浅灰白 `#F5F6F8`。
- Server、Desktop、Web 的产品边界来自之前的拆分记录；App 的移动定位由用户本轮确认。

## 交付

1. `docs/reference/curated-brand-guidelines.md`：正式名称、字体、配色、图案、组合字标、保护区、格式、平台适配与维护规则。
2. `icon/brand/brand.json`：机器可读品牌清单。
3. 五套透明组合字标 PNG 与混合 SVG；原主字标不重绘，后缀用 Outfit 600。
4. 四套明确产品命名的图标副本；移动端额外提供 1024 px 无透明方图与 Android 透明前景。
5. 离线预览页、品牌总览图、Outfit 预览字体与许可。
6. `scripts/dev/generate-brand-assets.py`：可重复生成资源，使用锁定的本地字体包。

## 验证范围

检查五套名称、PNG / SVG 结构、SVG 内嵌主字标来源、字标后缀基线、图标副本与既有源图一致、移动图标无透明度、许可同步以及重生成结果一致。视觉检查总览图的明暗背景效果。此次仅整理品牌资源，现有页面主题与各端运行时图标引用保持已接入链路；未把移动资源写入另一仓库，也未重打生产包。

## 后续原始设计源

仓库现有主字标与核心图案仅找到 PNG。SVG 交付明确标为内嵌原 PNG + 后缀路径的混合格式；取得真正矢量设计源后可替换内嵌部分，不能以自动描摹悄然改变核心图案。

## 2026-09-27 浅色底字标暗边修正

用户指出白底字标存在黑色模糊边。像素检查发现历史 `icon/curated-wordmark.png` 的边缘含深色背景预混 RGB：主色虽为 `#FE628E`，仍有 3,737 个完全不透明像素不是该粉色。旧图归档为 `icon/brand/source/curated-wordmark-original.png`。新增 `clean-brand-wordmark.py`，按品牌粉色与旧深色底恢复透明覆盖率，更新品牌源图与 `src/icon/` 水印副本，再重生成五套 PNG / SVG 与预览。浅色和深色背景均已视觉检查；原实色粉色像素及墨迹边界不变。

## 2026-09-27 透明背景导出

应用户要求，在 `icon/brand/transparent/` 中明确交付五套无底板字标 PNG / SVG、五个按产品命名的无底板核心图案 PNG 和一个通用图案混合 SVG。字标与总资源包现有透明 PNG / SVG 内容一致；单独目录方便直接选择透明资源。透明图案沿用原 mark 的 alpha 和位置，可见 RGB 统一为 `#FE628E`。离线棋盘格预览只用于显示 alpha，不烘焙进图。正式应用图标及移动商店源图仍保留平台要求的底色。

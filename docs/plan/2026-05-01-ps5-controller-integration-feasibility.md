# PS5 DualSense 手柄集成 — 可行性研究

## 想法概述

让 Curated 媒体库应用支持通过 PS5 DualSense 手柄进行操控：用手柄的十字键/摇杆导航界面、用按键触发常用操作、在播放器页面用手柄控制播放。

## 技术通道总览

手柄与 Web/桌面应用通信有 **三条通道**，能力逐级递增：

| 通道 | 可用环境 | 连接方式 | 能拿到什么 |
|---|---|---|---|
| **Gamepad API** (W3C 标准) | 所有现代浏览器、Electron | USB + 蓝牙 | 标准按键、摇杆轴、扳机压力值、基础双马达震动 |
| **WebHID API** (Chromium 专有) | Chrome / Edge / Electron 渲染进程 | 仅 USB | 以上全部 + 触摸板、陀螺仪、自适应扳机、RGB LED、高级触觉反馈 |
| **node-hid** (Node.js 原生) | Electron 主进程 | USB + 蓝牙 | 完全等同于 WebHID，且不受浏览器安全策略限制 |

### Gamepad API — 当前阶段即可用

这是 W3C 标准，**不依赖 Electron**。所有主流浏览器都支持。PS5 DualSense 连接后自动映射为 `"standard"` 布局：

```
事件: gamepadconnected / gamepaddisconnected
轮询: navigator.getGamepads() 每帧读取
```

**可获取的输入：**
- 17 个标准按键：  、  、  、  、L1/L2/R1/R2、Share、Options、L3/R3、PS、D-pad 4 方向、触控板按下
- 4 个模拟轴：左摇杆 X/Y、右摇杆 X/Y
- L2/R2 扳机作为压力感应按键（0.0–1.0 模拟值）
- 基础震动：`vibrationActuator.playEffect("dual-rumble", { duration, strongMagnitude, weakMagnitude })`（仅 Chromium）

**拿不到的：** 触摸板滑动/多点触控、陀螺仪、自适应扳机阻力、LED 灯控制、高级触觉波形

### WebHID API — 进阶功能

仅 Chromium 系（Chrome/Edge/Electron），且需要 HTTPS 或 localhost，用户需主动授权。第三方库 `dualsense-ts` 封装了完整的 DualSense HID 协议：

**额外可获取：**
- 触摸板：多点触控，1920×1080 分辨率，可作为精准光标
- 陀螺仪 + 加速度计：3 轴各，可用于倾斜感应
- 自适应扳机控制：设置 L2/R2 的阻力模式（0–7 级）
- RGB LED：5 区独立控制（玩家指示灯）
- 高级触觉：自定义波形，非简单双马达震动

### node-hid — Electron 主进程方案

未来 Electron 阶段，`dualsense-ts` + `node-hid` 在主进程运行，通过 IPC 桥接到渲染进程。比 WebHID 额外优势：
- 支持蓝牙连接（WebHID 仅 USB）
- 无需用户授权弹窗
- 可后台持续监听（窗口失焦也不中断）

## 分阶段可行性

### 阶段 0：当前 Web 阶段（Vue SPA + 浏览器）

**可行。** 使用 Gamepad API 即可实现基础手柄操控。

- Chrome/Edge：完美支持，含震动
- Firefox：支持，但 `hapticActuators` 实验性
- Safari：基本按键/轴可用，无震动

**实现方式：** 在 Vue 应用层添加一个 `useGamepad()` composable，在 `requestAnimationFrame` 循环中轮询 `navigator.getGamepads()`，将手柄输入转化为应用事件。

### 阶段 1：Electron 阶段（未来目标架构）

**可行且体验更好。** 可叠加使用三条通道：

- **基础操控**：Gamepad API（蓝牙 + USB 均可，零配置）
- **进阶功能**：WebHID 或 node-hid（触摸板当鼠标、自适应扳机反馈、LED 指示播放状态）

在 Electron 中推荐主进程 `node-hid` 方案，因为：
1. 蓝牙可用
2. 无需用户手动授权
3. 窗口失焦仍可监听（后台播放控制）
4. 可触发系统级媒体键模拟

## 操控映射设计（草案）

### 全局导航层

| 手柄输入 | 操作 |
|---|---|
| D-pad / 左摇杆 | 焦点移动（焦点框在 UI 元素间移动） |
|   (Cross) | 确认 / 进入 |
|   (Circle) | 返回 / 取消 |
|   (Triangle) | 上下文菜单 |
|   (Square) | 多选 / 批量操作 |
| L1 / R1 | 切换标签页（Library → Favorites → Actors → …） |
| L2 / R2 | 页面快速滚动（模拟值控制速度） |
| Options | 打开设置 |
| Share | 搜索 |

### 播放器层

| 手柄输入 | 操作 |
|---|---|
|   | 播放 / 暂停 |
|   | 停止 / 退出播放器 |
| D-pad ← → | 快退 / 快进 |
| D-pad ↑ ↓ | 音量增 / 减 |
| L1 / R1 | 上一章节 / 下一章节 |
| L2 / R2 | 精细调速（模拟扳机值映射到倍速 0.5x–4x） |
| 左摇杆 ← → | 进度条拖动 |
| 右摇杆 ↑ ↓ | 音量微调 |
| PS 键 | 显示 / 隐藏播放控制 OSD |
| 触控板滑动 | 鼠标光标定位 |
| 触控板按下 | 鼠标左键点击 |

### 反馈（震动）

| 场景 | 震动模式 |
|---|---|
| 焦点移动到新元素 | 弱触觉（20ms） |
| 操作确认 | 中等触觉（50ms） |
| 错误/不可操作 | 双脉冲震动 |
| 播放/暂停切换 | 确认震动 |

## 实现路径

### 推荐方案：渐进式三步走

**第一步 — Gamepad 导航内核（可在当前 Web 阶段完成）**

新增 `src/composables/use-gamepad.ts`：

```
- 监听 gamepadconnected / gamepaddisconnected
- rAF 循环轮询 gamepad 状态
- 按键去抖（防止重复触发）
- 摇杆死区处理
- 输出响应式 ref：当前按键状态、摇杆值
- 提供 onPress / onAxis 事件注册接口
```

新增 `src/composables/use-focus-navigation.ts`：

```
- 维护当前焦点元素 ID
- D-pad 方向映射到可聚焦元素列表的导航
- 焦点高亮 UI 组件（FocusRing.vue）
- 与 vue-router 联动：焦点在导航栏时切换路由
```

**第二步 — 播放器手柄控制（当前 Web 阶段）**

在 `PlayerPage.vue` 中复用已有的键盘快捷键架构，将 Gamepad 按键映射到相同的处理函数：
-   → `togglePlayPause()`
- D-pad ← → → `seekDelta()`
- D-pad ↑ ↓ → `adjustVolume()`

**第三步 — Electron 深度集成（未来阶段）**

- 主进程使用 `dualsense-ts` + `node-hid`
- 触摸板映射为系统光标（全局鼠标模式）
- 自适应扳机用于播放器：L2 阻力随播放速度变化
- LED 灯指示播放状态（绿色=播放中，红色=暂停，蓝色=空闲）
- 系统级手柄热键（全局媒体键）

## 风险与限制

| 风险 | 影响 | 缓解 |
|---|---|---|
| Safari 不支持 Gamepad API 震动 | macOS 用户体验降级 | 检测能力，静默降级 |
| 蓝牙连接下 Gamepad API 偶现轴漂移 | 误操作 | 摇杆死区阈值（默认 0.15） |
| WebHID 仅 USB 且需用户授权 | 进阶功能覆盖受限 | 基础操控走 Gamepad API 不受影响 |
| 焦点导航对虚拟滚动列表的兼容性 | 焦点可能落在未渲染元素上 | 虚拟滚动需预留焦点占位，焦点移动时触发 scrollTo |
| 手柄与鼠标/键盘同时操作的冲突 | 焦点状态混乱 | 检测最后输入设备类型，自动切换焦点导航模式 |
| 非标准浏览器（如 WebView）可能不支持 Gamepad API | 部分环境下无法使用 | 手柄功能始终是可选的增强，不影响核心功能 |

## 现有基础设施兼容性

项目中已有键盘快捷键体系（`src/lib/player-shortcuts.ts`、`SettingsCuratedShortcutSection.vue`），播放器键盘处理在 `PlayerPage.vue` 的 `onPlaybackKeydown()` 中。手柄集成可以：

1. 复用相同的 action 函数（`togglePlayPause`、`seekDelta`、`adjustVolume` 等）
2. 快捷键配置 UI 可扩展支持手柄按键重映射
3. 焦点导航系统是全新的——当前应用没有键盘焦点导航，这是手柄集成的核心新工作

## 结论

**技术上完全可行。** PS5 DualSense 手柄的绝大部分功能在不同的技术层次上都有成熟的 API 支持。

- **当前阶段（Web）：** Gamepad API 即可实现按钮操控和基础震动，覆盖 90% 的操控需求
- **未来阶段（Electron）：** 通过 `dualsense-ts` 解锁触摸板、自适应扳机、LED 等高级特性

核心工作量在于**焦点导航系统**——这是手柄操控的前提，也是当前应用所没有的。其他部分工作量可控，与现有代码冲突很小。

建议先实现第一步（Gamepad 导航内核 + 播放器控制），做一个最小可行原型来验证手感，再决定是否投入后续阶段。

## 实施计划（2026-05-02）

### 目标与边界

本轮实施目标是先交付 **Web 阶段可用的 DualSense / 标准手柄 MVP**：在不引入 Electron、WebHID、node-hid 和后端改动的前提下，用 Gamepad API 支持播放器控制，并为后续全局焦点导航打好可测试的输入层基础。

本轮不做：
- WebHID 授权、触控板、陀螺仪、自适应扳机、LED 控制。
- Electron 主进程、IPC、node-hid、系统级媒体键。
- 完整手柄按键重映射 UI。默认映射先固化在前端常量中，等 MVP 手感验证后再扩展设置页。

### 推荐路径

采用“先播放器，后全局导航”的渐进方案。

1. **输入内核先行**：新增纯 TypeScript 的 Gamepad 标准布局、按键边沿检测、摇杆死区、方向重复节流与震动能力检测。核心逻辑放在 `src/lib/gamepad/`，用 Vitest 覆盖，不依赖 Vue 生命周期。
2. **播放器 MVP**：在 `PlayerPage.vue` 内复用现有 action 函数，不复制播放器逻辑。Gamepad 输入映射到 `togglePlayPause()`、`seekDelta()`、`adjustVolume()`、`toggleMute()`、`toggleFullscreen()`、`runCuratedCapture()` 等现有动作。
3. **全局导航试点**：播放器 MVP 稳定后，在 `AppShell.vue` 挂载全局焦点导航，但在 `player` 路由禁用全局导航，避免与播放器控制冲突。焦点导航优先使用真实 DOM focus 与现有 `focus-visible` 样式，再用 `data-controller-focused` 做手柄焦点补强。
4. **资料库网格专项处理**：虚拟瀑布流不要直接依赖 DOM 全量列表。后续在 `LibraryView.vue` / `VirtualMovieMasonry.vue` 里基于 `visibleMovies`、当前选中影片、列数估算和滚动定位实现网格移动，避免焦点落到未渲染节点。

### 任务拆分

#### Task 1：Gamepad 输入模型与纯函数

**文件**
- Create: `src/lib/gamepad/standard-gamepad.ts`
- Create: `src/lib/gamepad/gamepad-input.ts`
- Create: `src/lib/gamepad/gamepad-input.test.ts`

**内容**
- 定义标准布局按钮索引：`cross`、`circle`、`square`、`triangle`、`l1`、`r1`、`l2`、`r2`、`share`、`options`、`l3`、`r3`、`dpadUp`、`dpadDown`、`dpadLeft`、`dpadRight`、`psOrHome`、`touchpad`。
- 定义标准轴索引：`leftX`、`leftY`、`rightX`、`rightY`。
- 实现：
  - `applyDeadzone(value, threshold)`：默认阈值 `0.18`，过滤蓝牙轴漂移。
  - `directionFromAxes(x, y, threshold)`：把摇杆转换为 `up/down/left/right/null`。
  - `diffButtonEdges(previous, current)`：输出本帧刚按下 / 刚松开按钮。
  - `createRepeatGate(initialDelayMs, repeatMs)`：用于 D-pad / 摇杆方向长按重复。
  - `supportsDualRumble(gamepad)`：能力检测，不支持时静默降级。

**测试**
- `applyDeadzone(0.1, 0.18)` 返回 `0`。
- `applyDeadzone(0.5, 0.18)` 保留非零方向值。
- 按钮从未按下到按下只产生一次 press edge。
- 摇杆斜向输入按较大绝对轴值解析主方向。
- repeat gate 首次触发、等待初始延迟、再按间隔重复。

#### Task 2：Vue composable 封装 Gamepad API

**文件**
- Create: `src/composables/use-gamepad.ts`
- Create: `src/composables/use-gamepad.test.ts`

**内容**
- 监听 `window.gamepadconnected` / `window.gamepaddisconnected`。
- 使用 `requestAnimationFrame` 轮询 `navigator.getGamepads()`。
- 默认选择第一个 `mapping === "standard"` 的手柄；没有标准映射时可显示连接状态但不触发动作。
- 暴露：
  - `connected`
  - `gamepadId`
  - `lastInputAt`
  - `onButtonPress(button, handler)`
  - `onDirectionPress(direction, handler, options?)`
  - `rumble(pattern)`
  - `stop()`
- 组件卸载时取消 rAF 和事件监听。

**测试**
- fake `navigator.getGamepads()` 返回手柄时，`connected` 变为 true。
- 同一按钮持续按住只触发一次 `onButtonPress`。
- D-pad 长按按 repeat gate 重复触发。
- `rumble()` 在不支持 `vibrationActuator` 时不抛错。

#### Task 3：播放器手柄控制 MVP

**文件**
- Create: `src/composables/use-player-gamepad-controls.ts`
- Create: `src/composables/use-player-gamepad-controls.test.ts`
- Modify: `src/components/jav-library/PlayerPage.vue`
- Optional Modify: `src/locales/zh-CN.json`, `src/locales/en.json`, `src/locales/ja-JP.json`

**映射**
- Cross / Space 等价：播放 / 暂停。
- Circle / Escape 等价：退出全屏；非全屏时按现有返回意图离开播放器。
- D-pad Left / Right：调用 `seekDelta(-seekBackwardStep)` / `seekDelta(seekForwardStep)`。
- D-pad Up / Down：调用 `adjustVolume(+5)` / `adjustVolume(-5)`。
- Square：调用 `runCuratedCapture()`。
- Triangle：切换详细统计或打开现有播放设置入口，具体以当前 UI 可复用性为准，避免新做一套菜单。
- L1 / R1：短跳转，初版可复用 seek 步长的 3 倍；若后续有章节数据，再升级为章节切换。
- Options：切换播放器控制层显示状态或打开设置菜单。

**实现要点**
- `PlayerPage.vue` 只创建一组 `PlayerGamepadActions`，把现有函数传给 composable。
- 不在 composable 中直接读写视频元素，避免绕过播放器状态机。
- 当上下文菜单、输入框、滑块正在交互时，手柄播放热键仍可用，但不会把方向输入转发给全局焦点导航。
- 每次确认动作可调用 `rumble({ duration: 35, weakMagnitude: 0.35, strongMagnitude: 0.15 })`；不支持震动时无感降级。

**测试**
- mock `useGamepad()`，验证 Cross 触发 `togglePlayPause`。
- 验证 D-pad Left / Right 使用当前设置中的 seek 步长。
- 验证 D-pad Up / Down 调整音量。
- 验证 Circle 在全屏状态优先退出全屏。

#### Task 4：全局焦点导航基础设施

**文件**
- Create: `src/composables/use-gamepad-focus-navigation.ts`
- Create: `src/lib/gamepad/focus-navigation.ts`
- Create: `src/lib/gamepad/focus-navigation.test.ts`
- Modify: `src/layouts/AppShell.vue`
- Modify: `src/style.css`

**内容**
- 建立轻量焦点导航，不引入全局状态库。
- 通过选择器发现可控元素：`button:not(:disabled)`、`a[href]`、`[role="button"]`、`[data-gamepad-focusable]`，同时跳过隐藏、禁用、`aria-hidden`、不可见尺寸为 0 的节点。
- 用元素 `getBoundingClientRect()` 做空间导航：方向键选择目标方向上角度偏差最小、距离最近的元素。
- 用 Cross 调用当前焦点元素的 `click()`。
- 用 Circle 调用 `router.back()` 或关闭已打开的弹窗 / 抽屉。
- 用 L1 / R1 在主导航路由间切换：`home -> library -> actors -> tags -> curated-frames -> history -> settings`。
- 最后输入设备为鼠标 / 键盘时清除 `data-controller-focused`，避免手柄焦点框与鼠标 hover 状态混杂。

**测试**
- 给定一组矩形，向右选择正确目标。
- 隐藏或 disabled 元素不会成为候选。
- 当前焦点不存在时，选择离视口中心最近的候选。
- 输入模式从 gamepad 切到 pointer 后清理焦点状态。

#### Task 5：资料库网格导航专项

**文件**
- Modify: `src/views/LibraryView.vue`
- Modify: `src/components/jav-library/LibraryPage.vue`
- Modify: `src/components/jav-library/VirtualMovieMasonry.vue`
- Modify: `src/components/jav-library/MovieCard.vue`
- Create: `src/lib/gamepad/library-grid-navigation.ts`
- Create: `src/lib/gamepad/library-grid-navigation.test.ts`

**内容**
- `LibraryView.vue` 已有 `visibleMovies` 与 URL `selected` 同步，应复用它作为手柄网格焦点状态。
- D-pad 左右移动 `selectedMovie` 的 index ±1。
- D-pad 上下移动 index ± 当前列数。
- Cross 打开详情，Square 在批量模式下切换选择，否则进入批量模式并选中当前影片。
- L2 / R2 按压力值映射为滚动速度，先做离散 page up / page down，后续再做模拟速度滚动。
- `VirtualMovieMasonry.vue` 暴露按 movie id 滚动到目标块的能力，确保虚拟列表中目标卡片会被渲染。
- `MovieCard.vue` 增加 `data-gamepad-focusable` 和当前手柄选中态样式，但不破坏现有鼠标点击与键盘 focus。

**测试**
- 可见列表中从 index 0 向右到 index 1。
- 当前列数为 5 时，向下从 index 2 到 index 7。
- 列表边界不会越界。
- 批量模式下 Square 调用 `toggleBatchSelect` 而不是打开详情。

#### Task 6：设置、提示与文档

**文件**
- Modify: `src/components/jav-library/settings/SettingsCuratedSection.vue` 或新建小节组件
- Modify: `src/locales/zh-CN.json`, `src/locales/en.json`, `src/locales/ja-JP.json`
- Modify: `README.md`
- Modify: `.cursor/rules/project-facts.mdc`

**内容**
- 增加一个轻量设置项：启用 / 禁用手柄控制。第一版可用 `localStorage` 保存浏览器本机偏好，key 建议为 `curated-gamepad-controls-v1`。
- 设置页显示当前能力状态：未连接、已连接、浏览器不支持、震动不可用。
- README 只补充当前 Web 阶段能力，不写 Electron / WebHID 已实现。
- `project-facts.mdc` 同步记录“Gamepad API Web MVP 已实现，Electron 深度集成仍为未来目标”。

### 验证计划

自动化验证：
- `pnpm test -- src/lib/gamepad/gamepad-input.test.ts`
- `pnpm test -- src/lib/gamepad/focus-navigation.test.ts`
- `pnpm test -- src/lib/gamepad/library-grid-navigation.test.ts`
- `pnpm test -- src/composables/use-player-gamepad-controls.test.ts`
- `pnpm typecheck`
- `pnpm lint`
- `pnpm test`

手动验证：
- Chrome / Edge 下用 USB 和蓝牙分别连接 DualSense。
- 打开播放器，验证 Cross、Circle、D-pad、Square、L1/R1、Options。
- 在不支持震动或浏览器拒绝震动时，确认控制功能不受影响。
- 在资料库大列表中验证上下左右移动不会跳到未渲染卡片或丢失 URL `selected`。
- 使用鼠标点击后确认手柄焦点框自动消失，再用手柄输入后恢复。

### 风险控制

- **避免一上来做完整空间导航**：播放器 MVP 可以先独立交付，确认 Gamepad API、按键映射和震动降级可靠后，再进入全局导航。
- **避免虚拟列表焦点错位**：资料库网格导航基于数据 index 和 URL `selected`，不要直接依赖当前 DOM。
- **避免与播放器键盘快捷键分叉**：播放器手柄控制只调用现有 action，不复制播放状态逻辑。
- **避免桌面架构误假设**：当前 Vue 代码不假设 Electron、node-hid 或原生播放器存在；未来 Electron 集成应通过独立 service / IPC 边界进入。

### 我的执行顺序

如果进入实现，我会按下面顺序做：

1. 先写 `src/lib/gamepad/` 的纯函数和单测，确认输入边沿、死区、repeat、震动能力检测可靠。
2. 再写 `use-gamepad.ts`，用 fake Gamepad / fake rAF 做 composable 单测。
3. 接入 `PlayerPage.vue`，只传入现有播放器 action，完成播放器 MVP。
4. 跑播放器相关测试、`pnpm typecheck`，并用浏览器手动验证一次。
5. 再做全局焦点导航与资料库网格导航，不和播放器 MVP 混在一个提交里。
6. 最后补设置开关、文档和项目事实说明，跑 `pnpm lint`、`pnpm test`、`pnpm build` 做收口。

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

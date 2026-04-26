# Player Playback Settings Implementation Plan

Date: 2026-04-26
Status: In Progress

## Goal

在播放器页新增一个紧凑型“播放设置”按钮与下拉菜单，用于控制当前播放会话的：

- 播放速度
- 播放模式（`Direct` / `HLS`）

本能力仅作用于当前播放器会话，不修改 Settings 页的全局播放偏好。

## Scope

本次实现采用已确认的方案 A：

- 入口位于播放器右下角控制区
- 点击后打开小型 dropdown menu
- 菜单内包含两组单选项：
  - 播放速度预设
  - 播放模式预设

## Implementation Outline

### 1. 新增独立菜单组件

新增 `src/components/jav-library/PlayerPlaybackSettingsMenu.vue`：

- 负责渲染播放设置按钮
- 使用现有 `dropdown-menu` 组件族
- 暴露以下输入：
  - `disabled`
  - `playbackRate`
  - `playbackMode`
  - `canSwitchToDirect`
  - `switchingMode`
- 暴露以下事件：
  - `update:playbackRate`
  - `update:playbackMode`

同时新增 `src/components/jav-library/PlayerPlaybackSettingsMenu.test.ts` 覆盖：

- 按钮与菜单分组渲染
- 当前速度 / 当前模式高亮
- `Direct` 不可用时禁用
- 点击选项后正确抛出更新事件

### 2. PlayerPage 接入本次会话速度状态

修改 `src/components/jav-library/PlayerPage.vue`：

- 新增本次会话级别的 `playbackRate` 状态，默认 `1`
- 在 video 元素可用、媒体源切换、metadata 加载后都同步 `video.playbackRate`
- 保持为页面内状态，不做持久化

### 3. PlayerPage 接入本次会话模式切换

在 `PlayerPage.vue` 中新增一个统一的切换方法，目标：

- 从当前播放位置切到 `Direct` 或 `HLS`
- 尽可能保留当前位置
- 切换前若在播放，则切换后自动恢复播放
- 失败时保留当前会话并给出 toast

实现上复用现有 `libraryService.createPlaybackSession(movieId, mode, startPositionSec?)`：

- `HLS`：继续走后端 session 创建
- `Direct`：走相同入口，后端会回退到 `ResolvePlayback()`

### 4. 文案补充

修改：

- `src/locales/en.json`
- `src/locales/zh-CN.json`
- `src/locales/ja.json`

新增播放器设置相关文案：

- 设置按钮 aria
- 菜单标题
- 播放速度
- 播放模式
- `Direct`
- `HLS`
- `Direct` 不可用提示

### 5. 验证

执行：

- 针对新增菜单组件的 Vitest
- `pnpm typecheck`

必要时补一次相关组件测试或轻量 lint 校验。

## Files Expected To Change

- `docs/plan/2026-04-26-player-playback-settings-implementation-plan.md`
- `src/components/jav-library/PlayerPage.vue`
- `src/components/jav-library/PlayerPlaybackSettingsMenu.vue`
- `src/components/jav-library/PlayerPlaybackSettingsMenu.test.ts`
- `src/locales/en.json`
- `src/locales/zh-CN.json`
- `src/locales/ja.json`

# 通知中心 — 需求梳理与产品方案

## 原始需求转述

1. 在顶栏外观切换（太阳/月亮）旁边增加一个「铃铛/闹钟」图标入口
2. 点击铃铛展开下拉列表，展示当前系统未读通知
3. 用户打开下拉即视为已读（或点击已读标记），已读通知自动从列表中消失
4. 通知保留一周历史，用户可回溯查看
5. 现有的 toast 位置从底部居中移到屏幕右上角

## 现状摸底

### 当前 toast 体系

| 维度 | 现状 |
|---|---|
| 组件 | vue-sonner (`<Toaster>`) |
| 位置 | `bottom-center`，在 Sonner.vue 中硬编码 |
| 使用点 | 扫描完成/失败 (`useScanTaskTracker`)、库目录监听 (`useLibraryWatchToasts`)、更新检查 (`SettingsAppUpdateSection`)、首页刷新 (`SettingsHomepageDevTools`)、刮削 (`ActorProfileCard`)、导出 (`CuratedFramesLibrary`) |
| 通知生命周期 | toast 弹出 → 4.5 秒后自动消失，无持久化 |
| 去重 | `useLibraryWatchToasts` 用 sessionStorage 记录已弹过的 taskId，仅防同会话重复弹出 |

### 当前顶栏右侧布局（AppShell.vue 796–814 行）

```
[ Sun icon ] [ Switch ] [ Moon icon ]
```

外观切换是一个独立区块，左侧有 `border-l` 分隔线。

### 现有通知来源（隐式）

目前系统里并没有「通知」这一抽象层。实际会产生用户可见消息的来源：

| 来源 | 触发方式 | 当前处理 |
|---|---|---|
| 手动扫描 | 用户点击扫描 → 后台任务 → toast | `useScanTaskTracker` → `pushAppToast` |
| 库目录自动监听 (fsnotify) | 文件变化 → 自动扫描 → toast | `useLibraryWatchToasts` → `pushAppToast` |
| 刮削完成 | 自动/手动刮削 → toast | 同上 |
| 应用更新可用 | `GET /api/app-update/status` | Settings 页面内展示，无主动推送 |
| HTTP 请求错误 | 网络异常 | 静默（console 除外） |
| 后台任务异常 | 任务失败 | toast（仅扫描/刮削类有） |

## 通知中心设计

### 核心理念

**通知 ≠ Toast。** Toast 是瞬时浮层，通知是持久化的消息记录。通知中心是二者的桥梁：

- 每条 `pushAppToast` 调用同时写入一条通知记录
- 用户在通知中心里看到所有未读通知
- 已读通知从列表中消失，但保留一周可回溯

### 通知数据模型

```
Notification {
  id: string               // 唯一标识，如 nanoid()
  type: "scan" | "scrape" | "update" | "error" | "system"
  severity: "info" | "success" | "warning" | "error"
  title: string            // 简短标题，如「扫描完成」
  message: string          // 详细描述
  timestamp: number        // Date.now()
  read: boolean            // 是否已读
  source?: {               // 可选：关联的实体
    taskId?: string
    movieId?: string
    route?: string          // 点击后跳转的路由
  }
}
```

### 通知生命周期

```
事件发生 → 创建通知(未读) → 同时弹出 toast
                ↓
         用户打开通知中心 → 可见即已读
                ↓
         已读通知隐藏（不再出现在未读列表）
                ↓
         保留 7 天，之后自动清理
```

关键规则：
- **打开下拉 = 所有可见通知标记已读**（无需逐条确认）
- 关闭下拉时，已读通知从列表中消失
- **已读不等于删除**，一周内可回溯

### 存储方案

**当前阶段（Web SPA）：** `localStorage`

```typescript
// Key: curated-notification-center-v1
// Value: Notification[]
```

- 前端独立管理，无需后端改动
- 上限 200 条，超出裁剪最旧
- 7 天过期自动清理（下次写入或打开时执行）

**未来阶段（Electron / 后端支持）：** 可迁移到后端 SQLite `notifications` 表，好处是多设备同步。但这在当前阶段不是必须。

### UI 布局

#### 顶栏入口

```
[ Sun ] [ Switch ] [ Moon ]  |  [ Bell icon + 未读红点 ]
```

- 铃铛图标：`Bell` (lucide-vue-next)
- 有未读通知时显示红色圆点 badge（`size-2 bg-red-500 rounded-full`）
- 无未读时仅有铃铛图标
- 位置：外观切换开关右侧，用 `border-l` 分隔

#### 下拉面板

点击铃铛展开一个 DropdownMenu（复用现有 shadcn-vue DropdownMenu 组件）：

```
┌─────────────────────────────────┐
│  通知中心            [全部已读]   │  ← 标题栏
├─────────────────────────────────┤
│  ● 扫描完成                     │  ← 未读标记 ●
│  资料库扫描完成 — 新增 3 部影片   │
│  2 分钟前                       │
├─────────────────────────────────┤
│  ● 刮削失败                     │
│  元数据刮削失败 — ABC-123        │
│  15 分钟前                      │
├─────────────────────────────────┤
│  ● 更新可用                     │
│  Curated v2.1.0 已发布          │
│  1 小时前                       │
├─────────────────────────────────┤
│  已读通知 (3)           查看全部 →│  ← 折叠的历史入口
├─────────────────────────────────┤
│                                 │
│  暂无新通知                     │  ← 空状态
│                                 │
└─────────────────────────────────┘
```

面板规格：
- 宽度 380px，最大高度 480px
- 内部可滚动（ScrollArea）
- 每条通知可点击：如果有 `source.route` 则跳转
- 每条通知右侧有 `X` 按钮可单独关闭
- 「全部已读」按钮在标题栏右侧
- 底部折叠展示最近的已读通知（3 条），点击「查看全部」展开完整历史

#### 完整历史视图

点击「查看全部」后，下拉面板切换为全历史模式：

```
┌─────────────────────────────────┐
│  ← 返回     通知历史      清空   │
├─────────────────────────────────┤
│  扫描完成                  ✓ 已读 │
│  资料库扫描完成 — 新增 3 部影片   │
│  2 分钟前                       │
├─────────────────────────────────┤
│  刮削失败                  ✓ 已读 │
│  ...                           │
├─────────────────────────────────┤
│  (7 天前的通知自动不显示)        │
└─────────────────────────────────┘
```

#### Toast 位置变更

`Sonner.vue` 中 `position` 从 `"bottom-center"` 改为 `"top-right"`：

```typescript
position: "top-right"
```

同时 CSS 类从 `!bottom-6` 改为 `!top-4 !right-4`。

### 通知来源映射

将现有 toast 调用点统一注册为通知：

| 现有 toast | 通知 type | severity |
|---|---|---|
| 手动扫描完成 | `scan` | `success` |
| 手动扫描失败/部分失败 | `scan` | `warning` / `error` |
| fsnotify 扫描完成 | `scan` | `success` |
| fsnotify 刮削完成 | `scrape` | `success` |
| fsnotify 刮削失败 | `scrape` | `error` |
| 应用更新可用 | `update` | `info` |
| 添加库路径成功 | `system` | `success` |
| 删除库路径成功 | `system` | `success` |
| 设置保存成功/失败 | `system` | `success` / `error` |
| 代理测试结果 | `system` | `success` / `error` |
| 演员刮削结果 | `scrape` | `success` / `error` |
| 演员标签更新 | `system` | `success` |
| 萃取帧创建/删除 | `system` | `success` |
| 萃取帧导出 | `system` | `success` |
| 播放进度保存失败 | `error` | `error` |
| 网络请求失败（全局拦截） | `error` | `error` |

### 对 pushAppToast 的改造

保持现有 API 签名不变，内部增强：

```typescript
export function pushAppToast(
  message: string,
  options?: {
    variant?: AppToastVariant
    durationMs?: number
    // 新增：通知中心相关
    notification?: {
      type: NotificationType
      title: string
      source?: NotificationSource
    }
  },
): string  // 返回通知 ID，调用方可忽略
```

- 如果传入 `notification`，则同时写入 localStorage 通知记录
- 如果不传，仅弹出 toast（兼容旧行为）
- 渐进式迁移：先将高价值来源（扫描、刮削、更新）加上通知，其余保持不变

### 清理策略

在通知中心打开时执行清理：

```typescript
function cleanupNotifications(notifications: Notification[]): Notification[] {
  const cutoff = Date.now() - 7 * 24 * 60 * 60 * 1000  // 7 天前
  return notifications
    .filter(n => n.timestamp > cutoff)
    .slice(-200)  // 最多保留 200 条
}
```

### 组件拆分

```
src/
  composables/
    use-notification-center.ts    # 核心逻辑：读写通知、未读计数、标记已读、清理
  components/
    notification-center/
      NotificationBell.vue        # 铃铛图标 + 红点 badge
      NotificationDropdown.vue    # 下拉面板（未读列表 + 已读折叠）
      NotificationHistoryList.vue # 完整历史列表
      NotificationItem.vue        # 单条通知渲染
```

### 实施范围评估

| 模块 | 工作量估计 | 复杂度 |
|---|---|---|
| `use-notification-center` composable | 中 | 低 — 纯 localStorage 读写 |
| `NotificationBell` + 集成到 AppShell | 小 | 低 — 插入现有顶栏 |
| `NotificationDropdown` 下拉面板 | 中 | 中 — 需要处理好打开/关闭时的已读逻辑 |
| `NotificationHistoryList` 历史视图 | 小 | 低 — 列表渲染 |
| Toast 位置迁移 (`bottom-center` → `top-right`) | 极小 | 低 — 改两个值 |
| `pushAppToast` 扩展 | 小 | 低 — 向后兼容 |
| 逐来源接入通知 | 中 | 低 — 机械式改动 |
| i18n 文案 | 小 | 低 |

**总规模：** 约 4–5 个新文件 + 少量现有文件改动，整体可控。

### 不在当前范围

- 后端持久化通知（保留到 Electron 阶段评估）
- WebSocket 实时推送通知
- 通知分组/聚合
- 通知偏好设置（哪些类型免打扰）
- 桌面原生通知（Notification API）

### 与手柄集成的关联

通知中心的下拉面板天然支持键盘/手柄焦点导航（见 [PS5 手柄可行性研究](./2026-05-01-ps5-controller-integration-feasibility.md)），铃铛入口可被焦点选中，下拉项用 D-pad 上下选择， 确认跳转， 关闭。

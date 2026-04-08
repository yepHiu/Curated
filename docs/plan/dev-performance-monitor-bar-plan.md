# 开发环境底部性能监测条设计方案

## 目标

在开发环境下，为 Curated 增加一个“浮在页面表面、不参与页面布局”的底部性能监测条，用于观察前后端联调阶段的实时指标与请求状态，并提供最小控制能力。

约束如下：

- 仅在开发环境显示
- 固定浮层，不挤压现有页面内容，不影响现有布局流
- 默认状态为“薄状态条 + 点击展开”
- 控制能力保持最小集
  - 暂停/恢复采样
  - 清空当前统计
  - 复制最近摘要
  - 展开查看请求明细

已确认的范围：

- 指标范围以“前后端联调指标”为主
- 默认状态采用“薄状态条 + 点击展开”
- 控制能力采用最小集
- 指标口径采用：
  - CPU：后端提供系统级 CPU 占用指标
  - 解码：前端提供播放体验级解码指标，而不是追求浏览器拿不到的显卡 Video Decode 引擎占用率

非目标：

- 不在第一版中追求显卡视频解码引擎真实占用率
- 不把整个浮条做成独立诊断页
- 不纳入播放器原生流分片级抓包

---

## 方案对比与结论

### 方案 A：直接在 `AppShell` 内部做单组件悬浮条

做法：

- 在 `src/layouts/AppShell.vue` 里直接挂一个 `DevPerformanceBar.vue`
- 组件内部自己采样、自己渲染、自己处理展开收起

优点：

- 接入最快
- 改动文件少
- 很适合验证第一版交互

缺点：

- 指标采集、UI、状态管理全堆在一个组件里，后续会很快变脏
- 不利于复用到播放页、设置页或单独诊断页
- 请求拦截与 UI 耦合过重

适用：

- 只打算做一次性调试条

### 方案 B：`AppShell` 挂载浮层，采集逻辑拆成 composable + monitor 模块

做法：

- `AppShell` 只负责挂载开发环境浮层
- 新增 composable / monitor 模块负责：
  - 前端指标采样
  - HTTP 请求观测
  - 健康检查轮询
  - 摘要计算与明细缓存
- UI 组件只消费聚合后的监测状态

优点：

- 壳层接入点稳定，符合当前项目结构
- 采集逻辑与展示逻辑分离，后续可继续扩展为独立诊断页
- 容易控制“开发环境可见、生产环境剔除”

缺点：

- 首次实现文件会比方案 A 多
- 需要设计一层轻量数据模型

适用：

- 当前这个项目，且后续大概率还会继续扩展播放链路/联调诊断能力

### 方案 C：独立 devtools 面板页，底部条仅做入口

做法：

- 底部只放极简摘要和打开按钮
- 详细信息全部放到单独诊断页或抽屉

优点：

- 主界面最干净
- 明细可以做得很完整

缺点：

- 第一版反馈链路太长
- 每次都要点击进入，不适合“边操作边看”
- 当前需求明确要一个“直接浮在表面”的监测条，这个方案不够贴合

适用：

- 后续第二阶段扩展

## 推荐方案

推荐使用 **方案 B**。

原因：

1. 入口应该放在 `AppShell`，因为它是当前整个 SPA 的稳定壳层，且已经挂了 `ScanProgressDock`、`Toaster` 和开发环境角标。
2. 指标采样不能直接写死在 UI 组件里，否则很快会演变成新的“大组件”。
3. 你这次要的是“前后端联调指标”，说明后续不仅会看 FPS，还会看 API 耗时、失败率、后端健康状态。这个范围天然需要一层独立监测状态模型。

---

## 设计方案

### 1. 挂载位置与显示方式

挂载位置：

- 在 `src/layouts/AppShell.vue` 中，与 `ScanProgressDock` 同级挂载
- 使用 `Teleport to="body"` + `fixed bottom-0 inset-x-0 z-*`

显示方式：

- 默认显示一条薄状态条，悬浮于视口底部
- 不占据文档流空间，不影响 `AppShell` 内部滚动和任何页面布局
- 点击后展开为更高的面板，面板仍然固定在底部表面
- 收起后恢复为单行摘要条

推荐视觉形态：

- 薄条高度 `32px` 到 `40px`
- 半透明背景 + 轻微模糊
- 左侧显示总体状态
- 中部显示关键摘要指标
- 右侧放最小控制按钮

交互结论：

- 默认收起为薄条
- 点击薄条后展开
- 展开态从底部向上生长
- 不给现有页面增加任何用于“让位”的底部 padding

### 2. 指标范围与口径

第一版只做联调最有价值的指标，不追求全量。

前端指标：

- 当前页面 FPS 采样值
- 最近 30 秒长任务次数
- JS 堆内存占用，浏览器支持时显示
- 当前路由名称
- 最近一次路由切换耗时
- 视频播放质量指标
  - `getVideoPlaybackQuality()` 可用时的掉帧数和总帧数
  - 播放器已有统计中的帧率
  - 最近卡顿次数或 `waiting` 次数

说明：

- 这里的“解码性能占用”按播放体验口径实现，不把它误写成浏览器实际无法稳定获得的 GPU 解码引擎占用率
- 如果用户当前在播放页，浮条优先显示播放器页可取得的解码体验指标；在非播放页则显示 `N/A`

后端联调指标：

- 最近 30 秒请求总数
- 最近 30 秒失败请求数
- 当前活跃请求数
- 最近 30 秒平均耗时
- 最近最慢的 5 条请求
- `/api/health` 最近一次健康检查状态与耗时
- 当前整机 CPU 占用率
- 当前 Curated 后端进程 CPU 占用率，若当前平台实现成本可控

说明：

- CPU 指标走本地 Go 后端采集，因为浏览器拿不到稳定的系统级 CPU 占用率
- 如果进程级 CPU 采集在当前平台实现成本过高，第一版可先保留整机 CPU 占用率，进程级作为后续增强项

显示原则：

- 收起态只显示 4 到 6 个最关键摘要
- 展开态才显示请求列表和更完整信息

### 3. 最小控制集

收起态：

- 点击整条展开

展开态：

- `Pause`：暂停采样与健康检查轮询，但保留当前数据显示
- `Resume`：恢复采样
- `Clear`：清空当前窗口统计与请求明细
- `Copy Summary`：复制一段文本摘要，方便贴给我或贴进 issue
- `Collapse`：收起回薄条

### 4. 数据模型

建议新增一层开发环境监测 store，而不是把状态散在组件里。

核心状态建议：

- `enabled`
- `expanded`
- `paused`
- `sampling`
  - `startedAt`
  - `lastUpdatedAt`
- `frontend`
  - `fps`
  - `longTaskCount30s`
  - `memoryUsedMB`
  - `routeName`
  - `lastRouteChangeMs`
  - `videoDroppedFrames`
  - `videoTotalFrames`
  - `videoWaitingCount`
- `backend`
  - `activeRequestCount`
  - `requestCount30s`
  - `failedRequestCount30s`
  - `avgLatencyMs30s`
  - `healthStatus`
  - `healthLatencyMs`
  - `systemCpuPercent`
  - `backendCpuPercent`
- `recentRequests`
  - `method`
  - `path`
  - `status`
  - `durationMs`
  - `startedAt`
  - `requestId`

数据窗口：

- 摘要统计基于最近 30 秒滑动窗口
- 请求明细最多保留最近 50 条

### 5. 架构与边界

这层监测功能建议拆成三层：

1. `AppShell` 挂载层
   - 只负责在开发环境渲染浮条
   - 不承担采样或业务逻辑
2. monitor / composable 采集层
   - 管理采样定时器、请求统计、健康检查、播放器状态读取
3. UI 展示层
   - 只读取聚合后的状态并渲染

这样可以确保：

- 业务页面完全不知道监测条的存在
- 浮条不改变原有路由页结构
- 未来可以把同一份采样状态复用到单独诊断页

### 6. 技术落点

推荐新增文件：

- `src/components/dev/DevPerformanceBar.vue`
  - 底部浮层 UI
- `src/composables/use-dev-performance-monitor.ts`
  - 统一监测状态、控制与摘要计算
- `src/lib/dev-performance/request-monitor.ts`
  - 请求观测与滑动窗口缓存
- `src/lib/dev-performance/frontend-monitor.ts`
  - FPS / long task / memory / route timing 采样
- `src/lib/dev-performance/health-monitor.ts`
  - `/api/health` 轮询

需要修改：

- `src/layouts/AppShell.vue`
  - 开发环境下挂载该浮层
- `src/api/http-client.ts`
  - 为 `get/post/patch/put/delete/postBlob` 注入统一请求观测埋点
- `src/main.ts`
  - 视情况初始化开发监测器，或交由 `AppShell` 首次挂载时启动
- `backend/internal/server/server.go`
  - 新增开发环境性能摘要接口，或扩展现有健康接口的开发字段
- `backend/internal/contracts/contracts.go`
  - 增加开发环境性能摘要 DTO
- `backend/internal/app/app.go`
  - 提供 CPU 指标读取与 DTO 映射

### 7. 采样与拦截策略

请求监测：

- 不改业务 endpoint 层
- 只在 `http-client.ts` 做统一埋点
- 记录：
  - method
  - url/path
  - start time
  - end time
  - duration
  - status
  - 是否失败

前端性能：

- FPS：用 `requestAnimationFrame` 滑动采样
- Long Task：优先 `PerformanceObserver('longtask')`
- Memory：优先读 `performance.memory`，不支持则显示 `N/A`
- Route timing：监听 `vue-router` 导航开始/结束，记录单次耗时

健康检查：

- 开发环境下每 10 秒轮询一次 `/api/health`
- 若当前暂停采样，则暂停轮询
- 若连续失败，则状态转为 `down`

CPU 指标：

- 由 Go 后端提供一个开发环境专用摘要接口
- 采样周期建议与健康检查一致或略慢，例如 `10s`
- 前端只消费结构化摘要，不在浏览器自己猜测 CPU 占用

播放器体验指标：

- 复用播放页已有的统计基础能力，不重复造一套播放器采样
- 非播放页不主动创建 `<video>` 相关监听
- 监测条应容忍“当前无播放器上下文”的状态

### 8. 不影响页面布局的保证

这部分要明确做成硬约束：

- 组件必须 `Teleport to="body"`
- 根节点必须 `fixed`
- 不给 `AppShell`、页面内容区、路由页增加 `padding-bottom`
- 只有浮层自己处理 `pointer-events`
- 收起态宽度占满视口，展开态高度向上生长

### 9. 风险与边界

需要明确控制的边界：

- 只在 `import.meta.env.DEV` 下显示
- 只监测经过 `http-client.ts` 的 API 请求
  - `<video>` 原生流请求、HLS 分片请求默认不纳入第一版
- 浏览器不支持 `performance.memory` 或 `longtask` 时，显示降级值
- 不做持久化存储，刷新后清空
- 后端 CPU 指标需要新增开发环境摘要接口或开发字段
- 若播放器统计在当前路由上下文不可用，则解码体验指标显示 `N/A`

### 10. 后续扩展位

如果第一版验证有效，下一阶段可继续做：

- 纳入 HLS / 视频元素专属指标
- 增加“仅看错误请求”筛选
- 增加“复制最近 N 条请求 JSON”
- 独立诊断抽屉或诊断页面

---

## 实施计划概览

### Phase 1：搭底座

- 新增 `use-dev-performance-monitor` 与状态模型
- 在 `AppShell` 开发环境下挂载空白浮层
- 完成收起/展开/暂停/清空/复制摘要的交互框架

### Phase 2：接前端指标

- 接入 FPS 采样
- 接入 Long Task 采样
- 接入 Memory 采样
- 接入路由切换耗时
- 接入播放页可用的解码体验指标

### Phase 3：接后端联调指标

- 在 `http-client.ts` 注入统一请求监测
- 完成最近 30 秒摘要计算
- 展开态展示最近请求明细
- 接入 `/api/health` 轮询状态
- 接入后端 CPU 摘要接口

### Phase 4：打磨与验证

- 校验不同页面下不影响布局
- 校验移动端/窄屏下不遮挡主要操作区
- 校验暂停/恢复/清空/复制摘要行为
- 增加最小单元测试或纯函数测试

---

## 自检结论

已自检以下几点：

- 范围没有和你最新确认的“CPU 用系统级、解码用体验级”相冲突
- 架构上没有把监测逻辑硬塞进 `AppShell`
- 设计上保持了“浮在表面、不影响布局”的硬约束
- 第一版范围足够小，不会把播放器深度诊断和联调浮条混成一个巨型功能

---

## 验证建议

开发完成后建议至少验证：

- `library`
- `detail`
- `player`
- `settings`

重点观察：

- 浮层始终不挤压页面
- 底部悬浮层不破坏播放器控制区
- 请求计数与健康状态会随联调状态变化
- 暂停/恢复后指标行为符合预期

---

## 结论

这个功能最合适的做法是：

- 在 `AppShell` 挂一个开发环境专用底部浮层
- 用独立 composable / monitor 模块管理指标采集
- 收起态做成薄条摘要，展开态看请求明细和控制项
- 第一版抓最关键的前后端联调指标，并用“体验级解码指标”补足播放侧观察能力

这样可以最快形成稳定、可扩展、不会污染业务布局的开发诊断入口。

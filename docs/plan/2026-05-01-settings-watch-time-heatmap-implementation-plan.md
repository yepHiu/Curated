# 设置概览每日观看时间统计实施方案

> 状态：已按最新要求调整。热力图可视化展示取消，保留播放器观看时长采集、Web API / Mock 聚合、设置概览中的观看统计摘要。

HTML 原型：`docs/plan/2026-05-01-settings-watch-time-heatmap-prototype.html`

## 目标

在 Curated 的 `Settings -> 概览` 中展示「每日观看时间」统计卡。用户不需要看到逐日色块图，只需要快速了解最近 3 个月的观看投入情况。

保留的数据能力：

- 播放器按有效播放增量上报观看时长。
- Web API 模式写入 SQLite 日聚合表。
- Mock 模式写入 `localStorage`。
- 设置概览读取最近 91 天数据并计算统计摘要。

取消的展示能力：

- 不渲染逐日格子、月份行、颜色图例、日期 tooltip。
- 不做按日期钻取。

## UI 方案

位置仍放在 `Settings -> 概览` 的现有统计卡下方，作为一张独立卡片。

卡片内容：

- 标题：`每日观看时间`
- 副标题：`过去 3 个月的有效观看时长统计。`
- 四项摘要指标：
  - 本周观看
  - 过去 3 个月
  - 最长连续
  - 最高单日
- 空态：无观看记录时显示简短说明，不显示图表占位。
- 加载态：只显示统计卡骨架，不显示图表骨架。

响应式：

- 桌面端：四项指标一行展示。
- 中小宽度：指标降为 2 列或 1 列。
- 不再需要横向滚动区域。

## 数据口径

采用方案 B：按日、按影片累加观看时长。

后端表：

```sql
CREATE TABLE IF NOT EXISTS playback_daily_watch_time (
  day_key TEXT NOT NULL,
  movie_id TEXT NOT NULL,
  watched_sec REAL NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (day_key, movie_id),
  FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);
```

接口：

- `GET /api/playback/watch-time/daily?days=91`
- `POST /api/playback/watch-time/daily`

`GET` 默认与上限均为 91 天。前端仍用返回的每日聚合数据计算本周观看、最近 3 个月总时长、最长连续天数、最高单日。

## 前端实现

保留：

- `src/lib/playback-watch-time-tracker.ts`
- `src/lib/playback-watch-time-storage.ts`
- `src/lib/watch-time-heatmap.ts` 中的日期、摘要和格式化工具
- `SettingsPage.vue` 中的加载与刷新流程
- `SettingsWatchTimeHeatmap.vue` 作为现有组件文件，当前只渲染统计摘要

调整：

- `SettingsWatchTimeHeatmap.vue` 移除逐日格子、月份行、颜色图例和相关样式。
- 文案从“观看时间热力图”改为“观看时间统计”。
- 测试断言不再检查 91 个可视格子，只检查统计摘要存在且图表 DOM 不存在。

## 测试

前端重点：

- `SettingsWatchTimeHeatmap.test.ts`：统计摘要存在，逐日图表 DOM 不存在。
- `watch-time-heatmap.test.ts`：保留 91 天窗口和摘要计算测试，作为统计数据口径保障。
- `playback-watch-time-storage.test.ts`：默认读取 91 天，Web API / Mock 均保留。
- `SettingsOverviewSection.test.ts`：概览页继续渲染观看统计卡。

后端重点：

- `playback_watch_time_handlers_test.go`：GET/POST 仍按 91 天窗口工作。
- `playback_watch_time_test.go`：日聚合与窗口查询保留。

## 验收

- 设置概览展示四项观看统计。
- 页面不出现逐日格子、月份标签、颜色图例或横向滚动图表。
- 播放器仍会记录有效观看时长。
- Web API 模式请求 `GET /api/playback/watch-time/daily?days=91`。
- Mock 模式刷新后仍能从 `localStorage` 读取统计数据。

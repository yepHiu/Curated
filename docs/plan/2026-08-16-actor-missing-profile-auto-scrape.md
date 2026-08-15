# 演员空资料自动补刮

日期：2026-08-16
状态：implemented
关联需求：REQ-0027
上位方案：[2026-04-12-actor-auto-scrape-options.md](2026-04-12-actor-auto-scrape-options.md)、[2026-04-12-actor-auto-scrape-implementation.md](2026-04-12-actor-auto-scrape-implementation.md)

## 问题

`autoActorProfileScrape` 已经能在影片元数据刮削成功后，为该片仍缺头像和简介的演员排队 `scrape.actor`。资料库里更早入库、从未随影片再刮削的空资料演员不会进入队列，用户会觉得「自动刮削演员」没有覆盖已有演员。

## 方案

继续复用同一个设置，默认关闭。开启后增加有界后台补刮：

- 查询条件与现有资格一致：活跃（非回收站）影片关联、头像与简介都为空
- 每批最多 50 人，按参演部数优先
- 启动约 20 秒后跑第一次，之后每 15 分钟再扫一批
- 打开设置开关后立刻触发一次
- 自动尝试失败后冷却 24 小时；影片刮削触发的补刮不受冷却限制
- 仍走全局 `scraper.maxConcurrent` 槽位和 in-flight pending 去重
- 演员页按需自动刮削保持不变

## 非目标

- 不新增第二个设置项
- 不在关闭开关时强制刮削
- 不把失败演员做成需要用户处理的 needs-you 消息
- 不在本切片改 provider 选择或 Metatube 调度

## 验收

- 开启开关后，历史空资料演员会在后台分批进入 `scrape.actor`
- 已有头像或简介的演员不会被 sweep 再排队
- 关闭开关后 sweep 与影片后补刮都不排队
- 同一演员在冷却期内不会被 sweep 反复打源；手动或影片刮削仍可立即再试

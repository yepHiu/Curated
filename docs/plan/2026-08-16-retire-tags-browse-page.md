# 退役资料库「标签浏览」页

日期：2026-08-16  
状态：in-progress  
关联需求：REQ-0028  
依赖：REQ-0026（资料库筛选选择板已覆盖标签多选 AND）

## 判断

侧栏「标签」和 `/tags` 上的标签云，原先是进入精确 `tag=` 筛选的专用入口。资料库筛选现在已经提供：

- 用户标签 / INFO 标签分组
- 库存数量、搜索、多选 AND
- 与演员 / 厂牌同一套选择板
- 顶栏可关闭芯片

再保留一条平行导航，会让「资料库 / 收藏 / 标签」看起来像三套应用，而标签页本身并不比筛选多出产品能力。

标签页现在还多做的事只有：把元数据标签和用户标签铺成两片云，以及把网格按标签字符串排一次序。这些都不是独立浏览场景，可以去掉。

## 要去掉

- 侧栏浏览分组里的「标签」项
- `/tags` 作为独立浏览页，以及 `LibraryPage` 在 `mode=tags` 时的标签云卡片
- 首页口味雷达点标签时跳到 `name: "tags"`
- 详情 / 列表点标签时若来源是 tags 页、继续停在 tags 页的特殊规则

## 要保留

- 影片的用户标签、INFO 标签，以及详情 / 卡片上的标签展示
- 资料库筛选里的标签选择板、URL `tag=`、Saved Views 的 `tag` 字段
- 萃取帧库标签、演员用户标签（不是资料库浏览页）

## 兼容

旧书签和已保存视图不能摔死。

| 入口 | 做法 |
| --- | --- |
| `/#/tags`、`/#/tags?tag=Drama` | 路由重定向到 `library`，保留 `tag` 及其他可保存 query |
| Saved Views `mode=tags` | 应用时落到 `library`，`tag` 等筛选不变；读取仍接受 `mode=tags`，不必升 schemaVersion |
| `GET /api/library/movies?mode=tags` | 继续当作活动库别名（与现在相同：返回集合同 `library`），本轮不删 API 值 |
| 首页 / 详情点标签 | 一律进入 `library?tag=...` |

## 明确不做

- 不删标签数据，不改标签编辑
- 不把演员库或萃取帧库的标签入口并进这次
- 不在资料库网格上方再做一块标签云「补偿」；筛选选择板就是唯一浏览入口

## 落地

1. `/tags` 重定向到 `library`，query 原样并入；`canonicalLibraryRouteMode("tags")` 把浏览 / Saved Views / 详情回跳一律落到 `library`
2. 侧栏去掉标签项；首页口味雷达改走资料库
3. 删掉 `LibraryPage` 的 tags 云布局，以及 `LibraryView` 对 `mode=tags` 的特殊排序
4. `getDetailBrowseTargetMode("tags", ...)` 与旧 `browse=tags` 一律落到 `library`
5. 应用 Saved View 时把 `mode=tags` 映射为 `library`；新保存不再写出 `mode=tags`
6. 同步规则、功能清单与实现对照里的 `/tags` 行

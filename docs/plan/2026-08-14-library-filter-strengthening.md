# 资料库筛选强化

状态：in-progress  
关联需求：REQ-0026

## 问题

现有资料库筛选看起来有 Saved Views，但日常不好用：

- 「我的评分」是精确匹配，选 4 星找不到 4.5 / 5 星。
- 年份、时长、未刮削、无封面、未评分都筛不了。
- 标签 / 演员 / 厂牌只能从别的页面点进去，筛选弹层里改不了。
- 资料库 / 收藏 / 最近入库页几乎看不到当前生效条件，尤其是从详情点进来的标签。

## 本轮落地

不改 Saved Views schemaVersion，只在 v1 上做加法，并修正评分语义。

1. `userRating` 改为最低分（大于等于 N）；新增 `unrated`。
2. 新增 `year`（`YYYY` / `unknown`）、`runtime`（short / standard / long）、`catalog`（unscraped / no-cover）。
3. 筛选弹层可填标签、演员、厂牌，并带资料库联想。
4. 顶栏用可关闭芯片展示全部生效条件。
5. 前后端 ListMovies / Saved Views / Mock 客户端筛选对齐。

## 语义

| 字段 | 含义 |
| --- | --- |
| userRating | 本地用户评分大于等于 N，不用刮削分替代 |
| unrated | 没有本地用户评分；与 userRating 互斥，未评分优先 |
| year | 有效发行年；unknown 表示不在 1800-3000 |
| runtime | short 小于 90 分钟；standard 为 90-150；long 大于 150；时长 0 不匹配 |
| catalog=unscraped | 无演员且无元数据标签 |
| catalog=no-cover | 无封面且无缩略图 |

已保存的「正好 4 星」视图会变成「4 星及以上」。这是有意的兼容，比精确匹配更符合个人库用法。

## 明确不做

- 多标签 AND/OR、排除条件、自由区间滑杆。
- 把扫描进度或消息中心搬进筛选。
- 新的 Saved Views schemaVersion。

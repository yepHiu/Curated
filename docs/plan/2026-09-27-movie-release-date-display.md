# 影片详情页发售日期展示

## 数据能力

现有 Metatube 刮削适配器将提供商返回的 `info.ReleaseDate` 归一为 `YYYY-MM-DD`，存入 SQLite `movies.release_date`。详情接口已返回可选字段 `releaseDate`，Web mapper 已传入前端 `Movie`，无需新增接口或数据库迁移。具体影片能否取得日期取决于提供商是否返回该字段；空值不代表没有发售日期。

「编辑影片信息」已有发行日编辑功能。用户覆盖值优先于刮削底稿，清空覆盖值恢复刮削日期。Mock 日期是演示数据，不可用于确认真实影片日期。

## 展示方案与实现

- 在详情页标题下方的片商、年份、分辨率信息行，优先显示「发售日期 YYYY-MM-DD」。
- 有完整日期时替代独立年份；日期为空时继续显示已有年份，不推算月日。
- 使用 HTML `time` 元素，直接展示日期字符串，避免时区转换。
- 中、英、日文分别使用「发售日期」「Release date」「発売日」。
- 共用 `DetailPanel` 的详情视图同步获得展示，包括只读视图。

## 使用方式

已有日期的影片刷新前端后直接显示。缺少日期时可在更多操作中选择「刷新元数据」，或通过「编辑影片信息」填写发行日；刷新能否补全取决于上游数据。

## 验证结果

- `pnpm typecheck` 通过。
- `pnpm test src/components/jav-library/DetailPanel.test.ts src/components/jav-library/DetailPage.test.ts src/i18n/locales.test.ts`：3 个文件、33 项既有测试通过。
- `pnpm exec eslint src/components/jav-library/DetailPanel.vue` 通过。
- 未执行真实影片在线刮削或浏览器视觉验收；数据能力结论来自现有链路代码。

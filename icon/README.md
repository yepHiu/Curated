# Curated 品牌源图

`icon/` 只放设计源文件。文件名全部小写 kebab-case：`curated-<用途>.png`。口头提及时用中间那个词即可。

| 怎么叫 | 文件 | 是什么 |
| --- | --- | --- |
| **wordmark** | `curated-wordmark.png` | 标志 + “Curated” 文字，透明底。README 顶部用这个。 |
| **appicon** | `curated-appicon.png` | 圆角深色底 + 标志。托盘、安装包、Web favicon 的源图。 |
| **mark** | `curated-mark.png` | 只有核心标志（四角星 + 加号 + 圆点），透明底。 |

前端运行时副本在 `src/icon/`，文件名与上表相同。

改 `appicon` 之后，还要同步派生：

- `public/Curated-icon.png`
- `backend/frontend-dist/Curated-icon.png`
- `backend/internal/assets/curated.ico`

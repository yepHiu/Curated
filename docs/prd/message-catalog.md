# 消息政策台账

`docs/prd/message-catalog.csv` 是 Curated 消息怎么提示、怎么显示、要不要进消息中心的产品台账。需求台账仍是 `requirements.csv`；本表只回答：**假如有这条消息，系统该怎么对待它。**

日期：2026-08-14  
关联需求：REQ-0025、REQ-0024

## 谁维护

产品经理（或代理按产品决定代写）在加新提示前先增一行，再改代码。不要先写 toast / 消息中心，再事后补行。

运行时政策在 `src/lib/message-policy.ts`，ID 必须与 CSV 的 `id` 一致。校验：

```powershell
python scripts/prd/message_catalog_lint.py docs/prd/message-catalog.csv
pnpm test -- src/lib/message-policy.test.ts
```

## 字段

| 字段 | 含义 |
|---|---|
| `id` | 稳定 ID，如 `MSG-0001`。一经占用不复用。 |
| `title` | 一行名称，便于扫表 |
| `area` | 产品域，如 `library-scan`、`settings-ops` |
| `trigger` | 什么时候出现 |
| `level` | `silent` / `notify` / `needs-you` / `now` |
| `toast` | `yes` 或 `no`：当场浮层 |
| `center` | `none` / `recent` / `needs-you` / `now` |
| `badge` | `none` 或 `needs-you`：顶栏红点 |
| `until` | `immediate` / `open` / `session` / `7d` / `resolved` |
| `group` | 可选。同一 group 在消息中心合并成一行 |
| `cta` | 可选跳转，空表示没有。跳转 ≠ 待办 |
| `status` | `specified` / `implemented` / `deprecated` |
| `source_files` | 主要写入点 |
| `updated_at` | `YYYY-MM-DD` |
| `notes` | 残留说明 |

## 层级规则

| level | toast | center | badge | 含义 |
|---|---|---|---|---|
| `silent` | 可有可无 | `none` | `none` | 当场反馈或完全不说 |
| `notify` | 通常 `yes` | `recent` | `none` | 通知我，打开面板可已读 |
| `needs-you` | 通常 `yes` | `needs-you` | `needs-you` | 要处理；打开面板不清空 |
| `now` | 通常 `no` | `now` | `none` | 活状态，不落历史 |

设置页运维（备份、Health、provider ping、代理探测、CSV 导出、重绑成功）默认 `silent`，结果留在设置页。若将来要进中心，只能改成 `notify`，不能改成 `needs-you`。

## 加一行时

1. 分配下一个 `MSG-xxxx`。
2. 先选 `level`，再填 toast / center / badge，不要三套口各写各的。
3. 把 ID 加进 `src/lib/message-policy.ts`。
4. 调用方传 `messageId`，由政策层决定写不写中心。
5. 跑 lint 和 `message-policy` 测试。

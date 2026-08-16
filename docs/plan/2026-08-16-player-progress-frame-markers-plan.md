# 播放器进度条萃取帧标记 — 实施计划

- 日期：2026-08-16
- 状态：**已实施**（提交：`806df1d8` 聚类纯函数 → `4436cba5` 标记组件 + i18n → `48f8aa40` PlayerPage 接线；全量 `pnpm test` 782 通过）
- 前置：需求文档 `docs/plan/2026-08-16-player-progress-curated-frame-markers.md`（已确认：竖线刻度穿轨 / 点击跳簇内最早帧 / 全部萃取帧）
- UI 稿：`docs/plan/2026-08-16-player-progress-frame-markers-wireframe.html`
- 范围：纯前端，无后端改动

## 1. 技术方案总览

```
PlayerPage.vue（现有）
  ├─ movie.id 就绪 → listCuratedFramesPage({ movieId, limit: 200 })   ← 两种模式统一入口
  │    （total 超出则补拉；失败 debug log 后静默降级为无标记）
  ├─ 播放内萃取保存成功（返回 { id, positionSec }）→ 追加标记
  ↓ markers: { id, positionSec }[] + durationSec
PlayerProgressFrameMarkers.vue（新增，自包含）
  ├─ ResizeObserver 自测宽度 W（首测前 W=0 → 退化为不合并渲染，测量后自动纠正）
  ├─ clusterFrameMarkers(markers, durationSec, W)  ← 纯函数
  └─ 渲染刻度层：absolute + pointer-events-none，刻度自身可交互
       ├─ 单帧刻度：3px 竖线穿轨，hover 显示 formatClock 时间
       └─ 合并簇：稍宽刻度 + ×n 角标，hover 显示「n 个萃取帧 · 最早 hh:mm:ss」
  @seek(sec) → PlayerPage.seekToAbsolutePlaybackTime(sec)
```

坐标系：刻度定位 `left: (positionSec / durationSec × 100)%`，与现有 buffered 覆盖层（`PlayerPage.vue` 内 `z-[11]` 层）同一参照系——reka-ui 轨道 `inset-x-0` 全宽，无需 thumb 宽度补偿。

## 2. 文件清单

### 新增

| 文件 | 内容 |
|------|------|
| `src/lib/player-frame-markers.ts` | 纯函数聚类与类型 |
| `src/lib/player-frame-markers.test.ts` | 聚类单测 |
| `src/components/jav-library/PlayerProgressFrameMarkers.vue` | 刻度覆盖层组件（自测量、渲染、hover 提示、点击 emit） |
| `src/components/jav-library/PlayerProgressFrameMarkers.test.ts` | 组件测试 |

### 修改

| 文件 | 改动 |
|------|------|
| `src/components/jav-library/PlayerPage.vue` | ① movie 就绪后加载帧列表；② 在 `progressSliderRootRef` 容器内、buffered 层之后渲染标记组件（`z-[12]`）；③ `@seek` 接 `seekToAbsolutePlaybackTime`；④ `savePendingCuratedFrame()` 成功后追加 `{ id, positionSec }`（按 id 去重）；⑤ `totalDurationSec <= 0` 时不渲染 |
| `src/locales/zh-CN.json` / `en.json` / `ja.json` | `player.frameMarkerAria`（「跳转到萃取帧 {time}」）、`player.frameMarkerClusterAria`（「跳转到 {count} 个萃取帧中最早的一个（{time}）」）、`player.frameMarkerClusterTip`（「{count} 个萃取帧 · 最早 {time}」） |

### 落地后同步（功能合并后另起 docs 提交）

`project-facts.mdc`（播放器小节补一句）、`docs/guide.md` 操作说明、README 无需动（短入口不含功能细节）。

## 3. 纯函数契约（`src/lib/player-frame-markers.ts`）

```ts
export interface FrameMarkerInput {
  id: string
  positionSec: number
}

export interface FrameMarkerCluster {
  /** 簇内成员按时间升序 */
  items: FrameMarkerInput[]
  /** 展示位置（0–1），簇内成员像素坐标的中点 */
  ratio: number
  /** 点击跳转目标：簇内最早帧 */
  seekToSec: number
}

/** 默认最小间距 12px（刻度视觉宽 + 间隙），可调 */
export function clusterFrameMarkers(
  markers: readonly FrameMarkerInput[],
  durationSec: number,
  trackWidthPx: number,
  options?: { minSeparationPx?: number },
): FrameMarkerCluster[]
```

边界约定：

- `durationSec <= 0` → 返回 `[]`（播放器本就禁用进度条）。
- `trackWidthPx <= 0`（首测前）→ 每帧各成一簇，不合并；位置按时间中点自身。
- `positionSec < 0 || > durationSec` → 过滤，不参与。
- 贪心分簇以**簇首帧**为窗口基准：`px(next) - px(clusterFirst) < minSeparationPx` 则入簇；保证单簇宽度有界，超长密集段自然裂为相邻多簇。
- 稳定输入（按时间排序后处理）保证同宽度下输出确定，便于测试与快照。

## 4. 组件要点（`PlayerProgressFrameMarkers.vue`）

- Props：`markers: readonly FrameMarkerInput[]`、`durationSec: number`；Emits：`seek: [sec: number]`。
- 根元素：`absolute inset-x-0 top-1/2 z-[12] h-0 pointer-events-none`，挂在 `progressSliderRootRef` 的 relative 容器内（buffered 层之上，不遮挡 Slider 交互）。
- 刻度元素：`pointer-events-auto`，单帧 `w-[2px]` 竖线穿轨（高约 10px，上下各出 2px）；主色 `bg-primary`（`#fe628e`）+ 1px 深色描边保证在亮轨道上也可见。
- 命中区：伪元素扩展至 ≥44px 高、≥12px 宽（触控标准），不影响视觉宽度。
- 合并簇：刻度稍宽（3px）+ 挂在刻度正下方的数量文本（`×5`，`text-[10px]` 主色纯文本，完整低于轨道、不遮挡刻线）。
- hover 提示：纯 CSS/受控浮层显示 `formatClock` 时间；簇显示「n 个萃取帧 · 最早 hh:mm:ss」。无缩略图（二期候选）。
- 可访问性：刻度为 `button`，`aria-label` 用 i18n key；`tabindex` 参与焦点顺序但位于 Slider 之后。
- 宽度自测：`onMounted` 对根元素挂 `ResizeObserver` → 本地 ref；`onBeforeUnmount` 断开。
- 沉浸模式 / 全屏：组件位于底部控制栏容器内，自动跟随既有 chrome 过渡显隐，无需额外逻辑。

## 5. 提交切分（最小修改单元）

1. `feat: add pure clustering util for player frame markers` — lib + 单测。
2. `feat: add player progress frame marker overlay component` — 组件 + 组件测试 + 三语言 i18n key。
3. `feat: mark curated frames on player progress bar` — PlayerPage 接线（加载、渲染、seek、保存后追加）+ PlayerPage 测试补充。
4. `docs: sync player frame markers feature docs` — project-facts / guide / 本计划状态更新。

每步后跑 `pnpm typecheck && pnpm lint && pnpm test`；单文件迭代用 `pnpm test -- <file>`。

## 6. 测试清单

**lib 单测（`player-frame-markers.test.ts`）**

- 空列表 → `[]`；`durationSec <= 0` → `[]`。
- 单帧 / 互相远离的多帧 → 每帧一簇，`ratio = positionSec / duration`，`seekToSec = positionSec`。
- 密集（如 1s 内 5 帧且 12px 内）→ 1 簇 5 成员，`seekToSec` = 最早帧，`ratio` = 像素中点换算值。
- 边界：与簇首距离恰 `< 12px` 合并、`≥ 12px` 不合并；链式密集但跨度 > 12px 的序列裂为多簇。
- `trackWidthPx = 0` → 全部不合并。
- 脏数据：负数 / 超 `durationSec` 的 `positionSec` 被过滤。
- 输入顺序打乱 → 输出与排序后一致（稳定性）。

**组件测试**

- 给定 markers + duration → 渲染对应数量刻度；簇显示 `×n` 角标。
- 点击刻度 → emit `seek(seekToSec)`。
- 根层存在 `pointer-events-none`、刻度存在 `pointer-events-auto` 类（不拦截拖动）。

**PlayerPage 补充**

- Mock 服务返回该片帧 → 刻度渲染出现；点击刻度调用 seek（沿用现有 PlayerPage 测试基建）。

## 7. 手动验证

1. Mock 模式（`pnpm dev`）：挑一部有萃取帧的影片进播放器 → 刻度齐全；点刻度跳转；拖动进度不被刻度阻挡。
2. 构造密集帧（连续萃取 3–5 张相近帧）→ 合并簇 + 角标；缩放窗口 / 全屏 → 分簇随宽度重算。
3. 播放中按「萃取」保存 → 新刻度立即出现。
4. Web API 模式（`VITE_USE_WEB_API=true` + 后端）重复 1–3。
5. 沉浸模式显隐、键盘 tab 焦点、44px 触控命中。

## 8. 风险与备注

- 刻度与 thumb 重叠：thumb（`size-4` 白色）视觉层级更高，3px 细刻度可接受；如仍干扰，二期可给 hover 态加刻度提亮。
- Bundle 预算：纯函数 + 轻组件，增量远低于首屏/全量预算线，不需豁免。
- 帧数据按播放会话加载：其他页面删帧后本会话不自动移除（需求文档已列为非目标）。
- `positionSec` 与 HLS / 直连时长同域（媒体绝对秒），无换算风险；若个别影片元数据时长与实际微差，刻度最多贴近边缘，无功能影响。

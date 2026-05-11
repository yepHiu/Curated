# 详情页评论自动保存实现方案

> 日期：2026-05-11
> 范围：详情页 `MovieCommentSection` 前端交互；复用现有 Web API / Mock 服务契约，不新增后端接口。

## 需求理解

详情页中的「我的评论」目前需要用户手动点击「保存」。用户在 textarea 中输入较长评论时，如果切换影片、离开详情页、刷新页面或忘记点击保存，已有草稿可能丢失。自动保存机制的目标是让评论编辑更接近笔记体验：输入后后台自动持久化，并清楚展示当前保存状态。

该功能应覆盖 Web API 与 Mock 两种模式，因为现有 `LibraryService` 已把二者统一为：

- Web API：`GET/PUT /api/library/movies/{id}/comment`
- Mock：`localStorage`（`jav-library-movie-comment-v1`）

不建议新增后端端点；现有 `PUT` 覆盖式保存已经足够支撑自动保存。

## 当前实现事实

- 入口组件：`src/components/jav-library/DetailPage.vue`，以 `<MovieCommentSection :movie-id="movie.id" :readonly="commentReadonly" />` 挂载评论区。
- 评论组件：`src/components/jav-library/MovieCommentSection.vue`
  - `load()`：按 `movieId` 调 `libraryService.getMovieComment()`，写入 `draft` 与 `updatedAt`。
  - `save()`：校验 `MAX_MOVIE_COMMENT_RUNES` 后，将 `draft.value.trim()` 通过 `libraryService.putMovieComment()` 保存，再用服务端返回值覆盖 `draft` 与 `updatedAt`。
  - 回收站影片 `readonly=true` 时禁用 textarea 与保存按钮。
- 服务契约：`src/services/contracts/library-service.ts`
  - `getMovieComment(movieId): Promise<MovieCommentDTO>`
  - `putMovieComment(movieId, body): Promise<MovieCommentDTO>`
- 当前没有 `MovieCommentSection.test.ts`，详情页测试只 stub 了评论组件。

## 推荐方案

采用「防抖自动保存 + 队列串行保存 + 保留手动保存按钮」。

核心行为：

1. 评论加载完成后记录一份 `lastSavedBody`，代表已持久化内容。
2. 用户编辑 `draft` 后，如果内容和 `lastSavedBody` 不同，进入 `dirty` 状态。
3. 使用 `watchDebounced` 监听 `draft`，停止输入约 800ms 后自动保存；设置 `maxWait: 5000`，长时间持续输入也会周期性落盘。
4. 若保存请求进行中又发生新输入，不并发发 PUT，而是标记 `saveQueued=true`；当前请求结束后再保存最新草稿。
5. 保存成功后只更新 `lastSavedBody` 和 `updatedAt`，不要用响应体覆盖当前 `draft`，除非当前草稿仍等于本次提交内容，避免旧响应覆盖用户新输入。
6. 保留「保存」按钮作为立即保存入口；按钮文案可以根据状态显示「保存」「保存中…」「已自动保存」之一。
7. 切换 `movieId` 或组件卸载前，如果仍有未保存且合法的草稿，尝试 flush 一次；不要阻塞路由太久，也不要在只读状态保存。

## 状态模型

建议在 `MovieCommentSection.vue` 中新增这些状态：

- `lastSavedBody = ref("")`：最近一次成功持久化的正文，使用与保存 payload 相同的 normalize 结果。
- `hydrating = ref(false)`：加载评论期间为 true，用于阻止 `draft` watcher 误触发保存。
- `saveQueued = false`：保存中收到新变更时置 true。
- `commentSavedFlash = ref(false)`：保存成功后的短暂反馈。
- `commentSavedFlashTimer: ReturnType<typeof setTimeout> | null`
- `commentSavePromise: Promise<void> | null`：串行化保存请求。

派生状态：

- `normalizedDraft = computed(() => draft.value.trim())`
- `hasUnsavedChanges = computed(() => normalizedDraft.value !== lastSavedBody.value)`
- `isTooLong = computed(() => runeCount.value > MAX_MOVIE_COMMENT_RUNES)`

保留现有 `saving/loadError/saveError/updatedAt`，避免扩大模板改动。

## 保存算法

建议把当前 `save()` 拆成两层：

```ts
async function performSaveComment() {
  if (props.readonly) return
  const id = props.movieId.trim()
  if (!id) return
  if (isTooLong.value) {
    saveError.value = t("detailPage.commentTooLong", { n: MAX_MOVIE_COMMENT_RUNES })
    return
  }

  const bodyToSave = normalizedDraft.value
  if (bodyToSave === lastSavedBody.value) return

  saving.value = true
  saveError.value = ""
  try {
    const dto = await libraryService.putMovieComment(id, { body: bodyToSave })
    lastSavedBody.value = dto.body
    updatedAt.value = dto.updatedAt
    if (normalizedDraft.value === bodyToSave) {
      draft.value = dto.body
    }
    flashCommentSaved()
  } catch (err) {
    saveError.value = formatClientErr(err, t("detailPage.commentSaveError"))
  } finally {
    saving.value = false
  }
}

async function saveCommentNow() {
  if (commentSavePromise) {
    saveQueued = true
    return commentSavePromise
  }

  commentSavePromise = (async () => {
    do {
      saveQueued = false
      await performSaveComment()
    } while (saveQueued && hasUnsavedChanges.value)
  })()

  try {
    await commentSavePromise
  } finally {
    commentSavePromise = null
  }
}
```

这里的关键点是 `bodyToSave` 捕获当前要提交的内容；响应回来时只有在用户没有继续输入的情况下才回写 `draft`。否则旧请求只更新 `lastSavedBody/updatedAt`，随后队列会保存最新内容。

## Watcher 设计

加载 watcher 继续按 `movieId` 触发，但需要在加载期间屏蔽自动保存：

```ts
async function load() {
  const id = props.movieId.trim()
  if (!id) return
  loading.value = true
  hydrating.value = true
  loadError.value = ""
  saveError.value = ""
  try {
    const dto = await libraryService.getMovieComment(id)
    draft.value = dto.body
    lastSavedBody.value = dto.body
    updatedAt.value = dto.updatedAt
  } catch (err) {
    loadError.value = formatClientErr(err, t("detailPage.commentLoadError"))
  } finally {
    hydrating.value = false
    loading.value = false
  }
}
```

自动保存 watcher：

```ts
watchDebounced(
  draft,
  async () => {
    if (hydrating.value || loading.value || props.readonly) return
    if (isTooLong.value || !hasUnsavedChanges.value) return
    await saveCommentNow()
  },
  { debounce: 800, maxWait: 5000 },
)
```

组件卸载前建议做一次尽力 flush：

```ts
onBeforeUnmount(() => {
  if (!props.readonly && !isTooLong.value && hasUnsavedChanges.value) {
    void saveCommentNow()
  }
  if (commentSavedFlashTimer) clearTimeout(commentSavedFlashTimer)
})
```

说明：浏览器不保证异步请求在卸载后一定完成，所以这只是降低丢失概率；主要可靠性仍来自输入过程中的防抖自动保存。

## UI 文案与反馈

建议新增 i18n key：

- `detailPage.commentAutoSaved`
  - zh-CN：`已自动保存`
  - en：`Autosaved`
  - ja：`自動保存しました`
- `detailPage.commentUnsaved`
  - zh-CN：`有未保存修改`
  - en：`Unsaved changes`
  - ja：`未保存の変更があります`

模板反馈建议：

- 字数与更新时间仍放在 textarea 下方。
- 如果 `hasUnsavedChanges && !saving && !saveError`，显示「有未保存修改」。
- 如果 `saving`，显示「保存中…」。
- 如果 `commentSavedFlash && !hasUnsavedChanges`，显示「已自动保存」。
- 保留手动按钮；按钮 disabled 条件改为 `saving || !hasUnsavedChanges || isTooLong`。

不建议弹 toast；评论输入是高频操作，toast 会打扰且容易堆叠。

## 文件改动范围

推荐只改前端：

- 修改：`src/components/jav-library/MovieCommentSection.vue`
  - 引入 `watchDebounced`、`onBeforeUnmount`
  - 增加自动保存状态、串行保存队列、保存成功短反馈
  - 保留手动保存入口
- 创建：`src/components/jav-library/MovieCommentSection.test.ts`
  - 覆盖加载、自动保存、防抖、队列、防旧响应覆盖、超长限制、只读禁保存。
- 修改：`src/locales/zh-CN.json`
- 修改：`src/locales/en.json`
- 修改：`src/locales/ja.json`

不需要修改：

- `backend/`：现有 GET/PUT 和长度限制已满足需求。
- `src/services/contracts/library-service.ts`：契约已满足需求。
- `src/services/adapters/web/web-library-service.ts` / `mock-library-service.ts`：调用路径不变。
- `project-facts.mdc` / README / API 文档：如果只增加前端自动保存体验，不改变 API 或架构事实，暂不需要同步。

## 测试方案

新增 `MovieCommentSection.test.ts`，使用 fake timers 和 mock `useLibraryService()`：

1. 加载时填充 textarea，不触发 `putMovieComment`。
2. 用户输入后 800ms 内不保存，超过 debounce 后保存一次。
3. 持续输入时最多按 `maxWait` 保存，最终保存最新正文。
4. 保存中继续输入：第一次请求返回后不能覆盖 textarea 中的新草稿，并会排队保存最新正文。
5. 超过 `MAX_MOVIE_COMMENT_RUNES` 时不保存，显示 `commentTooLong`。
6. `readonly=true` 时不自动保存、不显示可用保存行为。
7. 手动点击保存按钮会立即保存，不必等待 debounce。

推荐验证命令：

```bash
pnpm vitest run src/components/jav-library/MovieCommentSection.test.ts --configLoader native --pool threads
pnpm typecheck
pnpm lint
```

若后续实现调整了详情页布局或 textarea 视觉结构，再按 UI 变更需要补充浏览器检查；本方案本身主要是行为变更。

## 取舍与风险

- 自动保存会增加 PUT 次数，但评论输入是单 textarea，800ms debounce + 5s maxWait 足够克制。
- 继续 trim 保存会保持当前后端/前端行为；如果产品希望保留首尾空格，应作为单独行为变更评估。
- 卸载前 flush 不能保证 100% 完成；更可靠的保护来自用户输入过程中已经周期性保存。
- 失败后保留 `hasUnsavedChanges`，用户继续输入或点击保存会重试；不做本地离线队列，避免引入新的跨模式同步复杂度。

## 实施顺序

1. 写 `MovieCommentSection.test.ts`，先覆盖“加载不保存”和“防抖后自动保存”。
2. 实现 `lastSavedBody/hydrating/watchDebounced/saveCommentNow`，跑测试转绿。
3. 增加保存中继续输入的队列测试，再补 `saveQueued/commentSavePromise`。
4. 增加 UI 状态与 i18n 文案测试，再更新三份 locale。
5. 跑 `MovieCommentSection.test.ts`、`pnpm typecheck`、`pnpm lint`。

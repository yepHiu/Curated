<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { watchDebounced } from "@vueuse/core"
import { Loader2, Sparkles } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { HttpClientError } from "@/api/http-client"
import { MAX_BOOK_COMMENT_RUNES, type AIActionPreviewDTO, type BookCommentDTO } from "@/api/types"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useAIActionRequest } from "@/composables/use-ai-action-request"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useAIService } from "@/services/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"
import { useComicLibraryService } from "@/services/comic-library-service"
import { usePhotoLibraryService } from "@/services/photo-library-service"

const props = defineProps<{
  kind: "comics" | "photos"
  entityId: string
}>()

const { t, locale } = useI18n()
const comicService = useComicLibraryService()
const photoService = usePhotoLibraryService()
const aiService = useAIService()
const { run: runAIAction, pending: aiActionPending, cancel: cancelAIAction } = useAIActionRequest(aiService)
const { writeEnabled: agentEnabled } = useExperimentalAgent()

const draft = ref("")
const updatedAt = ref("")
const loading = ref(false)
const saving = ref(false)
const loadError = ref("")
const saveError = ref("")
const lastSavedBody = ref("")
const hydrating = ref(false)
const commentSavedFlash = ref(false)
let commentSavedFlashTimer: ReturnType<typeof setTimeout> | null = null
let commentSavePromise: Promise<void> | null = null
let saveQueued = false

const aiBusy = ref(false)
const aiError = ref("")
const previewOpen = ref(false)
const preview = ref<AIActionPreviewDTO | null>(null)

/** 按 Unicode 标量计数字符长度，与后端 rune 上限对齐。 */
function countRunes(s: string): number {
  return [...s].length
}

const runeCount = computed(() => countRunes(draft.value))
const normalizedDraft = computed(() => draft.value.trim())
const isTooLong = computed(() => runeCount.value > MAX_BOOK_COMMENT_RUNES)
const hasUnsavedChanges = computed(() => normalizedDraft.value !== lastSavedBody.value)
const fieldId = computed(() => `book-comment-body-${props.kind}`)
const showAgentActions = computed(() => agentEnabled.value)

const updatedLabel = computed(() => {
  const iso = updatedAt.value.trim()
  if (!iso) return ""
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(d)
})

/** 把客户端或未知错误整理成可读文案。 */
function formatClientErr(err: unknown, fallback: string) {
  if (err instanceof HttpClientError) {
    return err.apiError?.message?.trim() || err.message || fallback
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return fallback
}

/** 按当前库类型读取一本备注。 */
async function loadComment(id: string): Promise<BookCommentDTO> {
  if (props.kind === "photos") {
    return await photoService.getPhotoComment(id)
  }
  return await comicService.getComicComment(id)
}

/** 按当前库类型覆盖保存一本备注。 */
async function saveComment(id: string, body: string): Promise<BookCommentDTO> {
  if (props.kind === "photos") {
    return await photoService.putPhotoComment(id, { body })
  }
  return await comicService.putComicComment(id, { body })
}

/** 加载当前书的已保存备注到编辑框。 */
async function load() {
  const id = props.entityId.trim()
  if (!id) {
    return
  }
  loading.value = true
  hydrating.value = true
  loadError.value = ""
  saveError.value = ""
  try {
    const dto = await loadComment(id)
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

/** 短暂展示“已自动保存”状态。 */
function flashCommentSaved() {
  commentSavedFlash.value = true
  if (commentSavedFlashTimer) clearTimeout(commentSavedFlashTimer)
  commentSavedFlashTimer = setTimeout(() => {
    // 保存成功提示只短暂显示，随后恢复为上次保存时间。
    commentSavedFlash.value = false
    commentSavedFlashTimer = null
  }, 2200)
}

/** 在未超长且有改动时把草稿写入当前书。 */
async function performSaveComment(entityId = props.entityId.trim()) {
  const id = entityId.trim()
  if (!id) {
    return
  }
  if (isTooLong.value) {
    saveError.value = t("detailPage.commentTooLong", { n: MAX_BOOK_COMMENT_RUNES })
    return
  }
  const bodyToSave = normalizedDraft.value
  if (bodyToSave === lastSavedBody.value) {
    return
  }
  saving.value = true
  saveError.value = ""
  try {
    const dto = await saveComment(id, bodyToSave)
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

/** 串行保存，避免快速输入触发重叠写入。 */
async function saveCommentNow(entityId = props.entityId.trim()) {
  if (commentSavePromise) {
    saveQueued = true
    return commentSavePromise
  }

  commentSavePromise = (async () => {
    // 把排队的后续保存排成串行，避免草稿互相覆盖。
    do {
      saveQueued = false
      await performSaveComment(entityId)
    } while (saveQueued && hasUnsavedChanges.value)
  })()

  try {
    await commentSavePromise
  } finally {
    commentSavePromise = null
  }
}

watch(() => props.entityId, cancelAIAction)
watch(agentEnabled, cancelAIAction)

/** 对当前草稿调用 polish_comment，成功后弹出确认预览。 */
async function polishComment() {
  if (!showAgentActions.value || aiBusy.value) {
    return
  }
  const id = props.entityId.trim()
  const body = normalizedDraft.value
  if (!id || !body) {
    aiError.value = t("detailPage.commentAiEmpty")
    return
  }
  if (isTooLong.value) {
    aiError.value = t("detailPage.commentTooLong", { n: MAX_BOOK_COMMENT_RUNES })
    return
  }
  aiBusy.value = true
  aiError.value = ""
  try {
    const request = props.kind === "photos"
      ? { photoId: id, body }
      : { comicId: id, body }
    const dto = await runAIAction("polish_comment", request)
    if (dto.noop) {
      aiError.value = t("detailPage.commentAiNoop")
      return
    }
    preview.value = dto
    previewOpen.value = true
  } catch (err) {
    if (err instanceof AIServiceError && err.code === "AI_CANCELLED") return
    if (err instanceof AIServiceError && err.code === "AI_PROVIDER_UNAVAILABLE") {
      aiError.value = t("detailPage.commentAiUnconfigured")
    } else {
      aiError.value = formatClientErr(err, t("detailPage.commentAiError"))
    }
  } finally {
    aiBusy.value = false
  }
}

/** 确认预览后把润色正文写回当前书。 */
async function applyCommentPreview() {
  const current = preview.value
  const entityId = props.entityId
  if (!current?.confirmToken || !current.sessionId) {
    previewOpen.value = false
    return
  }
  aiBusy.value = true
  aiError.value = ""
  try {
    const applied = await aiService.confirmTool({
      sessionId: current.sessionId,
      name: current.name,
      arguments: current.arguments ?? (
        props.kind === "photos"
          ? { photoId: entityId, body: current.proposedText }
          : { comicId: entityId, body: current.proposedText }
      ),
      confirmToken: current.confirmToken,
    })
    const data = applied.replayed
      ? await loadComment(entityId)
      : applied.data as { body?: string; updatedAt?: string } | undefined
    if (props.entityId !== entityId || preview.value !== current) return
    const body = data?.body ?? current.proposedText ?? ""
    draft.value = body
    lastSavedBody.value = body
    if (data?.updatedAt) {
      updatedAt.value = data.updatedAt
    }
    previewOpen.value = false
    preview.value = null
    flashCommentSaved()
  } catch (err) {
    aiError.value = formatClientErr(err, t("detailPage.commentAiApplyError"))
  } finally {
    aiBusy.value = false
  }
}

/** 关闭润色预览且不写入。 */
function discardCommentPreview() {
  previewOpen.value = false
  preview.value = null
}

watchDebounced(
  draft,
  async () => {
    // 防抖后自动保存当前草稿，超长时只提示不提交。
    if (hydrating.value || loading.value) {
      return
    }
    if (isTooLong.value) {
      saveError.value = t("detailPage.commentTooLong", { n: MAX_BOOK_COMMENT_RUNES })
      return
    }
    if (!hasUnsavedChanges.value) {
      return
    }
    saveError.value = ""
    await saveCommentNow()
  },
  { debounce: 800, maxWait: 5000 },
)

watch(
  () => [props.kind, props.entityId] as const,
  async ([, nextId], previous) => {
    // 切换书籍前先尽量保存上一本的未提交草稿。
    const previousId = previous?.[1]
    if (
      previousId &&
      previousId.trim() &&
      previousId.trim() !== nextId.trim() &&
      !isTooLong.value &&
      hasUnsavedChanges.value
    ) {
      await saveCommentNow(previousId)
    }
    await load()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  // 离开详情页时尽量把未超长的草稿写入当前书。
  if (!isTooLong.value && hasUnsavedChanges.value) {
    void saveCommentNow()
  }
  if (commentSavedFlashTimer) clearTimeout(commentSavedFlashTimer)
})
</script>

<template>
  <Card class="gap-4 rounded-3xl border-border/70 bg-card/85" data-book-comment-section>
    <CardHeader>
      <CardTitle>{{ t("detailPage.commentTitle") }}</CardTitle>
    </CardHeader>
    <CardContent class="flex flex-col gap-2">
      <p
        v-if="loading"
        class="text-sm text-muted-foreground"
      >
        {{ t("detailPage.commentLoading") }}
      </p>
      <p
        v-else-if="loadError"
        class="text-sm text-destructive"
      >
        {{ loadError }}
      </p>
      <template v-else>
        <label class="sr-only" :for="fieldId">{{ t("detailPage.commentTitle") }}</label>
        <div
          class="rounded-xl border border-border/60 bg-muted/40 text-foreground shadow-sm outline-none transition-[color,box-shadow] focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
          data-comment-field
        >
          <textarea
            :id="fieldId"
            v-model="draft"
            rows="6"
            :placeholder="t('detailPage.commentPlaceholder')"
            class="min-h-[140px] w-full resize-y border-0 bg-transparent px-3 py-2 text-sm text-foreground outline-none placeholder:text-muted-foreground"
          />
          <div
            v-if="showAgentActions"
            class="flex items-center justify-end px-1 pb-1"
            data-comment-ai-actions
          >
            <Button
              type="button"
              variant="ghost"
              size="sm"
              class="min-h-11 rounded-lg text-muted-foreground hover:text-foreground md:h-8 md:min-h-8"
              :disabled="aiBusy || !normalizedDraft"
              data-comment-ai-polish
              @click="polishComment"
            >
              <Loader2 v-if="aiBusy" class="size-4 motion-safe:animate-spin" />
              <Sparkles v-else class="size-4" />
              {{ t("detailPage.commentAiPolish") }}
            </Button>
            <Button v-if="aiActionPending" type="button" variant="ghost" size="sm" data-ai-action-cancel @click="cancelAIAction">
              {{ t("common.cancel") }}
            </Button>
          </div>
        </div>
        <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>
            {{ t("detailPage.commentRuneCount", { n: runeCount, max: MAX_BOOK_COMMENT_RUNES }) }}
          </span>
          <span v-if="saving">
            {{ t("detailPage.commentSaving") }}
          </span>
          <span v-else-if="hasUnsavedChanges && !saveError">
            {{ t("detailPage.commentUnsaved") }}
          </span>
          <span v-else-if="commentSavedFlash">
            {{ t("detailPage.commentAutoSaved") }}
          </span>
          <span v-else-if="updatedLabel">
            {{ t("detailPage.commentUpdatedAt", { time: updatedLabel }) }}
          </span>
        </div>
        <p
          v-if="saveError"
          class="text-sm text-destructive"
        >
          {{ saveError }}
        </p>
        <p v-if="aiError" class="text-sm text-destructive">{{ aiError }}</p>
      </template>
    </CardContent>
  </Card>
  <Dialog :open="previewOpen" @update:open="previewOpen = $event">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t("detailPage.commentAiPreviewTitle") }}</DialogTitle>
        <DialogDescription>{{ t("detailPage.commentAiPreviewDesc") }}</DialogDescription>
      </DialogHeader>
      <div class="grid gap-3 text-sm">
        <div>
          <p class="mb-1 text-xs text-muted-foreground">{{ t("detailPage.commentAiBefore") }}</p>
          <p class="whitespace-pre-wrap rounded-lg bg-muted/50 p-2">{{ preview?.originalText || "—" }}</p>
        </div>
        <div>
          <p class="mb-1 text-xs text-muted-foreground">{{ t("detailPage.commentAiAfter") }}</p>
          <p class="whitespace-pre-wrap rounded-lg bg-muted/50 p-2">{{ preview?.proposedText || "—" }}</p>
        </div>
      </div>
      <DialogFooter class="gap-2">
        <Button type="button" variant="outline" data-comment-ai-discard @click="discardCommentPreview">
          {{ t("detailPage.commentAiDiscard") }}
        </Button>
        <Button type="button" :disabled="aiBusy" data-comment-ai-apply @click="applyCommentPreview">
          {{ t("detailPage.commentAiApply") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

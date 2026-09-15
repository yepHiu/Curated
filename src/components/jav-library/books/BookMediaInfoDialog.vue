<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { Loader2, Sparkles } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { MAX_BOOK_TITLE_RUNES, type AIActionPreviewDTO } from "@/api/types"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { useAIActionRequest } from "@/composables/use-ai-action-request"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useAIService } from "@/services/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"

const props = defineProps<{
  book: {
    title: string
    sourceFileName: string
    location: string
    pageCount: number
    addedAt: string
    updatedAt: string
  }
  kind: "comics" | "photos"
  entityId: string
  busy?: boolean
}>()

const emit = defineEmits<{
  saveTitle: [title: string, done: (err?: unknown) => void]
  applied: []
}>()

const open = defineModel<boolean>("open", { required: true })
const { t, locale } = useI18n()
const aiService = useAIService()
const { run: runAIAction, pending: aiActionPending, cancel: cancelAIAction } = useAIActionRequest(aiService)
const { writeEnabled: agentEnabled } = useExperimentalAgent()

const titleDraft = ref("")
const saveError = ref("")
const saving = ref(false)
const aiBusy = ref(false)
const preview = ref<AIActionPreviewDTO | null>(null)
const previewOpen = ref(false)
const showAgentActions = computed(() => agentEnabled.value)

/** 把 ISO 时间格式化为本地可读标签。 */
function dateLabel(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString()
}

const rows = computed(() => [
  { label: t("bookBrowser.sourceFile"), value: props.book.sourceFileName || "—" },
  { label: t("bookBrowser.location"), value: props.book.location || "—" },
  { label: t("bookBrowser.fileFormat"), value: props.book.sourceFileName.match(/\.([^.]+)$/)?.[1]?.toUpperCase() || "—" },
  { label: t("bookBrowser.pages"), value: String(props.book.pageCount) },
  { label: t("bookBrowser.addedAt"), value: dateLabel(props.book.addedAt) },
  { label: t("bookBrowser.updatedAt"), value: dateLabel(props.book.updatedAt) },
])

const trimmedTitle = computed(() => titleDraft.value.trim())
const titleChanged = computed(() => trimmedTitle.value !== props.book.title.trim())
const canSave = computed(() => Boolean(trimmedTitle.value) && !saving.value && !props.busy && !aiBusy.value)

/** 打开弹窗时用当前展示标题填充草稿。 */
function syncDraftFromBook() {
  saveError.value = ""
  titleDraft.value = props.book.title
}

watch(
  () => props.entityId,
  () => {
    // 换书时关闭弹窗并清空翻译预览。
    open.value = false
    saveError.value = ""
    previewOpen.value = false
    preview.value = null
  },
)

watch(open, (isOpen) => {
  // 打开时同步草稿；关闭时取消进行中的翻译请求。
  if (isOpen) {
    syncDraftFromBook()
  } else {
    cancelAIAction()
    previewOpen.value = false
    preview.value = null
  }
}, { immediate: true })

watch(() => props.book.title, (title) => {
  // 外部刷新展示标题时，未保存中就覆盖草稿。
  if (open.value && !saving.value && !aiBusy.value) {
    titleDraft.value = title
  }
})

watch([() => props.entityId, open], cancelAIAction)
watch(agentEnabled, cancelAIAction)

/** 校验并提交展示标题。 */
function submitTitle() {
  saveError.value = ""
  if (!trimmedTitle.value) {
    saveError.value = t("bookBrowser.titleRequired")
    return
  }
  if ([...trimmedTitle.value].length > MAX_BOOK_TITLE_RUNES) {
    saveError.value = t("bookBrowser.titleTooLong")
    return
  }
  saving.value = true
  emit("saveTitle", trimmedTitle.value, (err?: unknown) => {
    // 保存回调负责解除忙碌并在失败时就地提示。
    saving.value = false
    if (err) {
      saveError.value =
        err instanceof Error && err.message.trim() ? err.message : t("bookBrowser.titleSaveFailed")
      return
    }
    open.value = false
  })
}

/** 请求标题翻译预览。 */
async function runTitleTranslation() {
  if (!showAgentActions.value || aiBusy.value || !trimmedTitle.value) {
    return
  }
  aiBusy.value = true
  saveError.value = ""
  try {
    const body =
      props.kind === "comics"
        ? { comicId: props.entityId, body: trimmedTitle.value, locale: locale.value }
        : { photoId: props.entityId, body: trimmedTitle.value, locale: locale.value }
    const dto = await runAIAction("translate_title", body)
    if (dto.noop) {
      saveError.value = t("detailPanel.movieAiTranslateNoop")
      return
    }
    preview.value = dto
    previewOpen.value = true
  } catch (err) {
    if (err instanceof AIServiceError && err.code === "AI_CANCELLED") return
    if (err instanceof AIServiceError && err.code === "AI_PROVIDER_UNAVAILABLE") {
      saveError.value = t("detailPanel.movieAiUnconfigured")
    } else {
      saveError.value = err instanceof Error && err.message.trim() ? err.message : t("detailPanel.movieAiError")
    }
  } finally {
    aiBusy.value = false
  }
}

/** 确认标题翻译预览并通知详情刷新。 */
async function applyTitlePreview() {
  const current = preview.value
  if (!current?.confirmToken || !current.sessionId) {
    previewOpen.value = false
    return
  }
  aiBusy.value = true
  saveError.value = ""
  try {
    const applied = await aiService.confirmTool({
      sessionId: current.sessionId,
      name: current.name,
      arguments: current.arguments ?? (
        props.kind === "comics"
          ? { comicId: props.entityId, title: current.proposedText }
          : { photoId: props.entityId, title: current.proposedText }
      ),
      confirmToken: current.confirmToken,
    })
    if (preview.value !== current) return
    titleDraft.value = applied.replayed ? titleDraft.value : (current.proposedText ?? titleDraft.value)
    previewOpen.value = false
    preview.value = null
    emit("applied")
  } catch (err) {
    saveError.value = err instanceof Error && err.message.trim() ? err.message : t("detailPanel.movieAiApplyError")
  } finally {
    aiBusy.value = false
  }
}

/** 关闭标题翻译预览，不写入。 */
function discardTitlePreview() {
  previewOpen.value = false
  preview.value = null
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent data-book-media-info class="max-h-[min(90dvh,40rem)] min-w-0 overflow-y-auto rounded-3xl border-border/70 sm:max-w-lg">
      <DialogHeader class="min-w-0 pr-5">
        <DialogTitle>{{ t("bookBrowser.mediaInfo") }}</DialogTitle>
        <DialogDescription class="sr-only">{{ t("bookBrowser.mediaInfo") }}</DialogDescription>
      </DialogHeader>
      <div class="flex min-w-0 flex-col gap-4 py-2">
        <p
          v-if="saveError"
          role="alert"
          class="rounded-xl border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ saveError }}
        </p>
        <div class="grid gap-2">
          <label class="text-sm font-medium" for="book-media-title">{{ t("bookBrowser.displayTitle") }}</label>
          <div
            class="rounded-xl border border-border/60 bg-muted/40 text-foreground shadow-sm outline-none transition-[color,box-shadow] focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
            data-book-media-title-field
          >
            <Input
              id="book-media-title"
              v-model="titleDraft"
              class="rounded-xl border-0 bg-transparent shadow-none focus-visible:border-0 focus-visible:ring-0"
              autocomplete="off"
              data-book-media-title
            />
            <div v-if="showAgentActions" class="flex items-center justify-end px-1 pb-1">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                class="min-h-11 rounded-lg text-muted-foreground hover:text-foreground md:h-8 md:min-h-8"
                :disabled="aiBusy || !trimmedTitle"
                data-book-media-ai-translate
                @click="runTitleTranslation"
              >
                <Loader2 v-if="aiBusy" class="size-4 motion-safe:animate-spin" />
                <Sparkles v-else class="size-4" />
                {{ t("detailPanel.movieAiTranslateTitle") }}
              </Button>
              <Button
                v-if="aiActionPending"
                type="button"
                variant="ghost"
                size="sm"
                data-ai-action-cancel
                @click="cancelAIAction"
              >
                {{ t("common.cancel") }}
              </Button>
            </div>
          </div>
        </div>
        <dl class="flex min-w-0 flex-col gap-4 text-sm">
          <div v-for="row in rows" :key="row.label" class="grid min-w-0 gap-1 sm:grid-cols-[6rem_minmax(0,1fr)] sm:gap-4">
            <dt class="text-muted-foreground">{{ row.label }}</dt>
            <dd class="min-w-0 select-text break-all leading-relaxed">{{ row.value }}</dd>
          </div>
        </dl>
      </div>
      <DialogFooter class="gap-3">
        <DialogClose as-child>
          <Button type="button" variant="outline" class="rounded-full" :disabled="saving">{{ t("common.close") }}</Button>
        </DialogClose>
        <Button
          type="button"
          class="rounded-full"
          data-book-media-save
          :disabled="!canSave || !titleChanged"
          @click="submitTitle"
        >
          {{ saving ? t("detailPanel.movieEditSaving") : t("bookBrowser.saveTitle") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
  <Dialog :open="previewOpen" @update:open="previewOpen = $event">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t("detailPanel.movieAiPreviewTitle") }}</DialogTitle>
        <DialogDescription>{{ t("detailPanel.movieAiPreviewDesc") }}</DialogDescription>
      </DialogHeader>
      <div class="grid gap-3 text-sm">
        <div>
          <p class="mb-1 text-xs text-muted-foreground">{{ t("detailPanel.movieAiBefore") }}</p>
          <p class="whitespace-pre-wrap rounded-lg bg-muted/50 p-2">{{ preview?.originalText || "—" }}</p>
        </div>
        <div>
          <p class="mb-1 text-xs text-muted-foreground">{{ t("detailPanel.movieAiAfter") }}</p>
          <p class="whitespace-pre-wrap rounded-lg bg-muted/50 p-2">{{ preview?.proposedText || "—" }}</p>
        </div>
      </div>
      <DialogFooter class="gap-2">
        <Button type="button" variant="outline" data-book-media-ai-discard @click="discardTitlePreview">
          {{ t("detailPanel.movieAiDiscard") }}
        </Button>
        <Button type="button" :disabled="aiBusy" data-book-media-ai-apply @click="applyTitlePreview">
          {{ t("detailPanel.movieAiApply") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

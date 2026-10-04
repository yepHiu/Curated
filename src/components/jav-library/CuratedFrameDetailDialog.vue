<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, useId, watch } from "vue"
import { useFocusWithin, onClickOutside, useEventListener } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { ChevronLeft, ChevronRight, Download, Film, PlayCircle, Plus, Sparkles, Trash2, X } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Carousel, CarouselContent, CarouselItem, type CarouselApi } from "@/components/ui/carousel"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import FrameImageViewer from "@/components/jav-library/FrameImageViewer.vue"
import CuratedFrameDeleteConfirmDialog from "@/components/jav-library/CuratedFrameDeleteConfirmDialog.vue"
import type { CuratedFrameRecord } from "@/domain/curated-frame/types"
import { deleteCuratedFrame, listCuratedFrameTagSuggestions, updateCuratedFrameTags } from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import { curatedFrameImageUrl } from "@/lib/curated-frame-image-url"
import { useUserTagSuggestKeyboard } from "@/composables/use-user-tag-suggest-keyboard"
import { filterUserTagSuggestions } from "@/lib/user-tag-suggestions"
import { commitCuratedFrameTags, shouldCommitCuratedFrameTagDraft, shouldShowCuratedFrameTagRetry, type CuratedFrameTagSaveStatus } from "@/lib/curated-frames/p2-state"
import { findAdjacentCuratedFrameDialogEntry, type CuratedFrameDialogDirection, type CuratedFrameDialogNavigationEntry, type CuratedFrameDialogItem as RowWithUrl } from "@/lib/curated-frames/dialog-navigation"
import { mergeCuratedFramesQuery, serializeCuratedFrameTagFilters } from "@/lib/library-query"
import { buildPlayerRouteFromCuratedFrame } from "@/lib/player-route"
import { pushAppToast } from "@/composables/use-app-toast"
import { useLibraryService } from "@/services/library-service"
import { useCuratedFrameExport, type CuratedExportMode } from "@/composables/use-curated-frame-export"
import { triggerDownloadBlob } from "@/lib/curated-frames/export-file"

const props = withDefaults(defineProps<{
  entries: readonly CuratedFrameDialogNavigationEntry<RowWithUrl>[]
  nearDuplicateIds?: readonly string[]
}>(), { nearDuplicateIds: () => [] })
const emit = defineEmits<{
  "update:open": [open: boolean]
  deleted: [ids: string[]]
  tagsSaved: [payload: { id: string; tags: string[] }]
}>()
const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()
const { exportFrames } = useCuratedFrameExport()
const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"
const noActorLabel = computed(() => t("curated.noActor"))
const curatedFrameNearDuplicateThresholdSec = 3
const maxFrameTags = 64
const maxFrameTagRunes = 64
const dialogOpenedFromActor = ref<string | null>(null)
const dialogExportError = ref("")
const exportBusy = ref(false)
let disposed = false

function preferredCuratedExportFormat() {
  return libraryService.curatedFrameExportFormat.value ?? "jpg"
}

const dialogOpen = ref(false)
const selected = ref<CuratedFrameRecord | null>(null)
const dialogMotionPlaying = ref(false)
const dialogCarouselApi = ref<CarouselApi | null>(null)
const dialogCarouselUserSelecting = ref(false)
const dialogTags = ref<string[]>([])
const dialogTagSaveStatus = ref<CuratedFrameTagSaveStatus>("idle")
const dialogTagSaveError = ref("")
const lastSavedDialogTags = ref<string[]>([])
const dialogTagSaveDebounceMs = 250

const dialogNavigationEntries = computed(() => props.entries)
const previousDialogEntry = computed(() =>
  findAdjacentCuratedFrameDialogEntry(
    dialogNavigationEntries.value,
    selected.value?.id,
    dialogOpenedFromActor.value,
    "previous",
  ),
)
const nextDialogEntry = computed(() =>
  findAdjacentCuratedFrameDialogEntry(
    dialogNavigationEntries.value,
    selected.value?.id,
    dialogOpenedFromActor.value,
    "next",
  ),
)
const canNavigateDialogPrevious = computed(() => previousDialogEntry.value !== null)
const canNavigateDialogNext = computed(() => nextDialogEntry.value !== null)
const selectedDialogNavigationIndex = computed(() => {
  const id = selected.value?.id
  if (!id) {
    return -1
  }
  return dialogNavigationEntries.value.findIndex(
    (entry) => entry.item.row.id === id && entry.sectionActor === dialogOpenedFromActor.value,
  )
})

let dialogTagSaveTimer: ReturnType<typeof setTimeout> | null = null
let dialogTagSaveInFlight = false
let detachDialogCarouselListeners: (() => void) | null = null
let dialogCarouselUserSelectingResetTimer: ReturnType<typeof setTimeout> | null = null

/** 与详情页「我的标签」一致：内联添加 */
const newUserTagDraft = ref("")
const userTagFormError = ref("")
const userTagInputOpen = ref(false)
const newUserTagInputRef = ref<HTMLInputElement | null>(null)
const userTagInlineZoneRef = ref<HTMLElement | null>(null)
const userTagSuggestRootRef = ref<HTMLElement | null>(null)
const userTagSuggestListRef = ref<HTMLElement | null>(null)
const tagSuggestDomId = useId()
const { focused: userTagSuggestRowFocused } = useFocusWithin(userTagSuggestRootRef)

/** 仅萃取帧库内已出现过的标签，与影片库标签无关 */
const userTagSuggestionCandidates = ref<string[]>([])

async function reloadTagSuggestions() {
  try {
    userTagSuggestionCandidates.value = await listCuratedFrameTagSuggestions()
  } catch {
    userTagSuggestionCandidates.value = []
  }
}

watch([dialogOpen, curatedFramesRevision, locale], () => {
  if (dialogOpen.value) void reloadTagSuggestions()
})
watch(dialogOpen, (open) => emit("update:open", open), { flush: "sync" })

const filteredUserTagSuggestions = computed(() =>
  filterUserTagSuggestions(
    userTagSuggestionCandidates.value,
    newUserTagDraft.value,
    new Set(dialogTags.value),
    { limit: 10 },
  ),
)

const showUserTagSuggestions = computed(
  () =>
    userTagInputOpen.value &&
    userTagSuggestRowFocused.value &&
    newUserTagDraft.value.trim() !== "" &&
    filteredUserTagSuggestions.value.length > 0,
)

function resetTagInputState() {
  newUserTagDraft.value = ""
  userTagFormError.value = ""
  userTagInputOpen.value = false
}

function sameDialogTags(a: string[], b: string[]) {
  if (a.length !== b.length) {
    return false
  }
  return a.every((tag, index) => tag === b[index])
}

function resetDialogTagSaveState(savedTags: string[] = []) {
  if (dialogTagSaveTimer) {
    clearTimeout(dialogTagSaveTimer)
    dialogTagSaveTimer = null
  }
  dialogTagSaveInFlight = false
  lastSavedDialogTags.value = [...savedTags]
  dialogTagSaveStatus.value = "idle"
  dialogTagSaveError.value = ""
}

function resetDialogState() {
  selected.value = null
  dialogMotionPlaying.value = false
  resetDialogCarouselUserSelecting()
  resetTagInputState()
  resetDialogTagSaveState()
  dialogOpen.value = false
}

function dialogEntryImageUrl(entry: CuratedFrameDialogNavigationEntry<RowWithUrl>): string {
  const index = dialogNavigationEntries.value.indexOf(entry)
  if (Math.abs(index - selectedDialogNavigationIndex.value) > 1) return entry.item.url
  return entry.item.row.imageBlob ? entry.item.url : curatedFrameImageUrl(entry.item.row.id)
}

function isDialogEntryCurrent(entry: CuratedFrameDialogNavigationEntry<RowWithUrl>) {
  return entry.item.row.id === selected.value?.id && entry.sectionActor === dialogOpenedFromActor.value
}

function resetDialogCarouselUserSelecting() {
  if (dialogCarouselUserSelectingResetTimer) {
    clearTimeout(dialogCarouselUserSelectingResetTimer)
    dialogCarouselUserSelectingResetTimer = null
  }
  dialogCarouselUserSelecting.value = false
}

function markDialogCarouselUserSelecting() {
  dialogCarouselUserSelecting.value = true
  if (dialogCarouselUserSelectingResetTimer) {
    clearTimeout(dialogCarouselUserSelectingResetTimer)
  }
  dialogCarouselUserSelectingResetTimer = setTimeout(() => {
    dialogCarouselUserSelectingResetTimer = null
    dialogCarouselUserSelecting.value = false
  }, 1200)
}

async function syncDialogCarouselToSelected(options: { jump?: boolean } = {}) {
  await nextTick()
  const api = dialogCarouselApi.value
  const index = selectedDialogNavigationIndex.value
  if (!api || index < 0) {
    return
  }
  api.scrollTo(index, options.jump ?? false)
}

async function selectDialogEntryFromCarousel(entry: CuratedFrameDialogNavigationEntry<RowWithUrl>) {
  if (isDialogEntryCurrent(entry)) {
    return
  }
  if (!(await persistDialogTags({ toastOnError: true }))) {
    await syncDialogCarouselToSelected({ jump: true })
    return
  }
  openDialog(entry.item, entry.sectionActor)
}

function onDialogCarouselInit(api: CarouselApi) {
  if (!api) {
    return
  }
  detachDialogCarouselListeners?.()
  dialogCarouselApi.value = api
  const handleSelect = () => {
    const entry = dialogNavigationEntries.value[api.selectedScrollSnap()]
    if (!entry || !dialogOpen.value || !dialogCarouselUserSelecting.value) {
      return
    }
    resetDialogCarouselUserSelecting()
    void selectDialogEntryFromCarousel(entry)
  }
  const handleReInit = () => {
    void syncDialogCarouselToSelected({ jump: true })
  }
  api.on("select", handleSelect)
  api.on("reInit", handleReInit)
  detachDialogCarouselListeners = () => {
    api.off("select", handleSelect)
    api.off("reInit", handleReInit)
  }
  void syncDialogCarouselToSelected({ jump: true })
}

function resolveSingleFrameActorNameForExport(
  frame: CuratedFrameRecord,
  fromActorSection: string | null,
): string | undefined {
  if (
    fromActorSection &&
    fromActorSection !== noActorLabel.value &&
    frame.actors.some((actor) => actor.trim() === fromActorSection)
  ) {
    return fromActorSection
  }
  return undefined
}

function openDialog(item: RowWithUrl, fromActorSection: string | null = null) {
  const { imageBlob, ...meta } = item.row
  void imageBlob
  dialogOpenedFromActor.value = fromActorSection
  dialogExportError.value = ""
  selected.value = meta
  dialogMotionPlaying.value = false
  dialogTags.value = [...item.row.tags]
  resetTagInputState()
  resetDialogTagSaveState(item.row.tags)
  dialogOpen.value = true
}

async function navigateDialogFrame(direction: CuratedFrameDialogDirection) {
  const target = direction === "previous" ? previousDialogEntry.value : nextDialogEntry.value
  if (!target) {
    return
  }
  if (!(await persistDialogTags({ toastOnError: true }))) {
    return
  }
  openDialog(target.item, target.sectionActor)
}

function isEditableKeyboardTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false
  }
  const tag = target.tagName.toLowerCase()
  return tag === "input" || tag === "textarea" || tag === "select" || target.isContentEditable
}

useEventListener(
  "keydown",
  (event: KeyboardEvent) => {
    if (
      !dialogOpen.value ||
      deleteConfirmOpen.value ||
      event.defaultPrevented ||
      event.altKey ||
      event.ctrlKey ||
      event.metaKey ||
      event.shiftKey ||
      isEditableKeyboardTarget(event.target)
    ) {
      return
    }
    if (event.key === "ArrowLeft") {
      event.preventDefault()
      event.stopPropagation()
      void navigateDialogFrame("previous")
    } else if (event.key === "ArrowRight") {
      event.preventDefault()
      event.stopPropagation()
      void navigateDialogFrame("next")
    }
  },
  { capture: true },
)

watch(
  [dialogOpen, selectedDialogNavigationIndex, dialogCarouselApi],
  ([open, index]) => {
    if (!open || index < 0) {
      return
    }
    void syncDialogCarouselToSelected()
  },
)

watch(
  dialogTags,
  (nextTags) => {
    if (!selected.value) {
      return
    }
    dialogTagSaveError.value = ""
    if (!shouldCommitCuratedFrameTagDraft({
      tags: nextTags,
      lastSavedTags: lastSavedDialogTags.value,
      saveInFlight: dialogTagSaveInFlight,
    })) {
      dialogTagSaveStatus.value = sameDialogTags(nextTags, lastSavedDialogTags.value)
        ? "idle"
        : dialogTagSaveStatus.value
      return
    }
    dialogTagSaveStatus.value = "dirty"
    if (dialogTagSaveTimer) {
      clearTimeout(dialogTagSaveTimer)
    }
    dialogTagSaveTimer = setTimeout(() => {
      dialogTagSaveTimer = null
      void persistDialogTags()
    }, dialogTagSaveDebounceMs)
  },
  { deep: true },
)

async function persistDialogTags(options: { toastOnError?: boolean } = {}) {
  const frame = selected.value
  if (!frame) {
    return true
  }
  if (dialogTagSaveTimer) {
    clearTimeout(dialogTagSaveTimer)
    dialogTagSaveTimer = null
  }
  if (dialogTagSaveInFlight) {
    return false
  }
  if (!shouldCommitCuratedFrameTagDraft({
    tags: dialogTags.value,
    lastSavedTags: lastSavedDialogTags.value,
    saveInFlight: dialogTagSaveInFlight,
  })) {
    dialogTagSaveStatus.value = sameDialogTags(dialogTags.value, lastSavedDialogTags.value)
      ? "idle"
      : dialogTagSaveStatus.value
    return true
  }
  dialogTagSaveInFlight = true
  dialogTagSaveStatus.value = "saving"
  dialogTagSaveError.value = ""
  const result = await commitCuratedFrameTags({
    frameId: frame.id,
    tags: dialogTags.value,
    lastSavedTags: lastSavedDialogTags.value,
    update: updateCuratedFrameTags,
  })
  if (disposed) return false
  if (!result.ok) {
    dialogTagSaveStatus.value = "error"
    dialogTagSaveError.value = t("curated.tagSaveFailed")
    if (options.toastOnError) {
      pushAppToast(dialogTagSaveError.value, { variant: "destructive" })
    }
    dialogTagSaveInFlight = false
    return false
  }
  dialogTagSaveInFlight = false
  lastSavedDialogTags.value = [...result.lastSavedTags]
  emit("tagsSaved", { id: frame.id, tags: [...result.lastSavedTags] })
  dialogTagSaveStatus.value = result.status
  if (selected.value?.id === frame.id) {
    selected.value = { ...selected.value, tags: [...result.lastSavedTags] }
  }
  if (selected.value?.id === frame.id && !sameDialogTags(dialogTags.value, lastSavedDialogTags.value)) {
    dialogTagSaveStatus.value = "dirty"
    dialogTagSaveTimer = setTimeout(() => {
      dialogTagSaveTimer = null
      void persistDialogTags()
    }, dialogTagSaveDebounceMs)
  }
  return true
}

async function saveDialogTagsNow() {
  await persistDialogTags({ toastOnError: true })
}

async function handleDialogOpenChange(v: boolean) {
  if (!v) {
    const ok = await persistDialogTags({ toastOnError: true })
    if (!ok) {
      dialogOpen.value = true
      return
    }
    resetDialogState()
    return
  }
  dialogOpen.value = v
}

function cancelUserTagInput() {
  userTagInputOpen.value = false
  newUserTagDraft.value = ""
  userTagFormError.value = ""
}

async function onUserTagAddButtonClick() {
  userTagFormError.value = ""
  if (!userTagInputOpen.value) {
    userTagInputOpen.value = true
    await nextTick()
    newUserTagInputRef.value?.focus()
    return
  }
  const t = newUserTagDraft.value.trim()
  if (!t) {
    return
  }
  addUserTag()
}

function addUserTagWithValue(raw: string) {
  userTagFormError.value = ""
  const tagText = raw.trim()
  if (!tagText) {
    return
  }
  if ([...tagText].length > maxFrameTagRunes) {
    userTagFormError.value = t("curated.tagMaxRunes", { n: maxFrameTagRunes })
    return
  }
  if (dialogTags.value.includes(tagText)) {
    newUserTagDraft.value = ""
    return
  }
  if (dialogTags.value.length >= maxFrameTags) {
    userTagFormError.value = t("curated.tagMaxCount", { n: maxFrameTags })
    return
  }
  dialogTags.value = [...dialogTags.value, tagText]
  newUserTagDraft.value = ""
}

function addUserTag() {
  addUserTagWithValue(newUserTagDraft.value)
}

const { highlightIndex, onTagSuggestKeydown } = useUserTagSuggestKeyboard({
  showSuggestions: showUserTagSuggestions,
  suggestions: filteredUserTagSuggestions,
  listRootRef: userTagSuggestListRef,
  commitTag: (tag) => addUserTagWithValue(tag),
  commitDraft: () => addUserTag(),
})

onClickOutside(userTagInlineZoneRef, () => {
  if (!userTagInputOpen.value) {
    return
  }
  cancelUserTagInput()
})

function removeUserTag(tag: string) {
  dialogTags.value = dialogTags.value.filter((x) => x !== tag)
}

function pickUserTagSuggestion(tag: string) {
  newUserTagDraft.value = tag
  userTagFormError.value = ""
  void nextTick(() => newUserTagInputRef.value?.focus())
}

/** 在本页用独立 cft 参数筛选萃取帧，不进入影片库 tag */
async function browseCuratedFramesByTag(tag: string) {
  const t = tag.trim()
  if (!t || !selected.value) {
    return
  }
  if (!(await persistDialogTags({ toastOnError: true }))) {
    return
  }
  resetDialogState()
  await router.push({
    name: "curated-frames",
    query: mergeCuratedFramesQuery(route.query, {
      cft: serializeCuratedFrameTagFilters([t]),
    }),
  })
}

/** 从帧来源文件及片内时间开始播放。 */
async function playFromFrame() {
  if (!selected.value) return
  if (!(await persistDialogTags({ toastOnError: true }))) {
    return
  }
  const { movieId, positionSec, fileId } = selected.value
  resetDialogState()
  await router.push(buildPlayerRouteFromCuratedFrame(movieId, positionSec, fileId))
}

async function exportSingleFromDialog() {
  await exportSingleFromDialogWithMode(libraryService.curatedFrameExportMode.value)
}

async function exportSingleFromDialogWatermarked() {
  await exportSingleFromDialogWithMode("watermarked")
}

function curatedFrameGifFilename(frame: CuratedFrameRecord): string {
  const source = frame.code.trim() || frame.id.trim() || "curated-frame"
  const safeName = source.replace(/[<>:"/\\|?*\u0000-\u001f]/g, "-")
  const extension = frame.motion?.contentType === "video/mp4" ? "mp4" : frame.motion?.contentType === "video/webm" ? "webm" : "gif"
  return `curated-${safeName}.${extension}`
}

async function exportSingleGifFromDialog() {
  const motion = selected.value?.motion
  const artifactUrl = motion?.status === "ready" ? motion.artifactUrl : undefined
  if (!selected.value || !artifactUrl) {
    dialogExportError.value = t("curated.exportGifUnavailable")
    return
  }
  exportBusy.value = true
  dialogExportError.value = ""
  try {
    const response = await fetch(artifactUrl, { credentials: "include" })
    if (!response.ok) {
      throw new Error(`GIF export request failed: ${response.status}`)
    }
    const blob = await response.blob()
    triggerDownloadBlob(blob, curatedFrameGifFilename(selected.value))
  } catch (err) {
    console.error("[curated-frames] GIF export failed", err)
    dialogExportError.value = t("curated.exportGifFailed")
  } finally {
    exportBusy.value = false
  }
}

async function exportSingleFromDialogWithMode(mode: CuratedExportMode) {
  if (!selected.value) {
    return
  }
  const actorName = resolveSingleFrameActorNameForExport(selected.value, dialogOpenedFromActor.value)
  if (!(await persistDialogTags({ toastOnError: true }))) return
  exportBusy.value = true
  dialogExportError.value = ""
  try {
    await exportFrames(props.entries.map((entry) => entry.item), [selected.value.id], actorName, preferredCuratedExportFormat(), mode)
  } catch {
    dialogExportError.value = t(mode === "watermarked" ? "curated.exportWatermarkedFailed" : "curated.exportFailed")
  } finally {
    exportBusy.value = false
  }
}

const deleteConfirmOpen = ref(false)
const deleteFrameBusy = ref(false)
const deleteFrameError = ref("")
const deleteTarget = ref<CuratedFrameRecord | null>(null)
const deleteTargetLabel = computed(() => deleteTarget.value?.code.trim() || deleteTarget.value?.title || t("curated.deleteLabel"))

function openDeleteConfirmFromDialog() {
  if (!selected.value) return
  deleteTarget.value = selected.value
  deleteFrameError.value = ""
  deleteConfirmOpen.value = true
}

async function executeDeleteCuratedFrame() {
  const frame = deleteTarget.value
  if (!frame || deleteFrameBusy.value) return
  deleteFrameBusy.value = true
  deleteFrameError.value = ""
  try {
    await deleteCuratedFrame(frame.id)
    deleteConfirmOpen.value = false
    resetDialogState()
    emit("deleted", [frame.id])
  } catch {
    deleteFrameError.value = t("curated.deleteFailed")
  } finally {
    deleteFrameBusy.value = false
  }
}

function dismissDeleted(ids: readonly string[]) {
  if (selected.value && ids.includes(selected.value.id)) resetDialogState()
}

function formatClock(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "00:00"
  const s = Math.floor(seconds % 60)
  const m = Math.floor(seconds / 60) % 60
  const h = Math.floor(seconds / 3600)
  const pad = (n: number) => String(n).padStart(2, "0")
  if (h > 0) return `${pad(h)}:${pad(m)}:${pad(s)}`
  return `${pad(m)}:${pad(s)}`
}

function formatCapturedAt(iso: string) {
  try {
    const d = new Date(iso)
    return d.toLocaleString(locale.value, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    })
  } catch {
    return iso
  }
}

function isNearDuplicateFrame(frameId: string) {
  return props.nearDuplicateIds.includes(frameId)
}

onUnmounted(() => {
  disposed = true
  if (dialogTagSaveTimer) clearTimeout(dialogTagSaveTimer)
  resetDialogCarouselUserSelecting()
  detachDialogCarouselListeners?.()
})

defineExpose({ open: openDialog, dismissDeleted })
</script>

<template>
    <Dialog :open="dialogOpen" @update:open="handleDialogOpenChange">
      <!-- 覆盖 DialogContent 默认 sm:max-w-lg，否则整窗约 512px 宽，左侧预览会被压成一条 -->
      <DialogContent
        class="h-[min(94dvh,960px)] max-h-[min(94dvh,960px)] w-[min(98vw,92rem)] max-w-[min(98vw,92rem)] gap-0 overflow-hidden border-border/70 p-0 sm:max-w-[min(98vw,92rem)]"
      >
        <DialogDescription class="sr-only">{{ t("curated.title") }}</DialogDescription>
        <div
          class="grid h-full min-h-0 w-full grid-cols-1 grid-rows-[auto_minmax(0,1fr)] md:grid-rows-1 md:grid-cols-[minmax(0,2.4fr)_minmax(16rem,22rem)] lg:grid-cols-[minmax(0,2.75fr)_minmax(17rem,24rem)]"
        >
          <div
            class="relative flex h-[min(45dvh,560px)] w-full min-w-0 items-center justify-center bg-black md:h-full md:min-h-0"
          >
            <Carousel
              class="h-full w-full min-w-0 overflow-hidden"
              :opts="{ align: 'center', loop: false }"
              @pointerdown.capture="markDialogCarouselUserSelecting"
              @init-api="onDialogCarouselInit"
            >
              <CarouselContent class="h-full ml-0">
                <CarouselItem
                  v-for="entry in dialogNavigationEntries"
                  :key="`${entry.sectionActor ?? 'frames'}-${entry.item.row.id}`"
                  class="h-[min(45dvh,560px)] pl-0 md:h-[min(94dvh,960px)]"
                  :aria-current="isDialogEntryCurrent(entry) ? 'true' : undefined"
                >
                  <div class="flex h-full w-full min-w-0 items-center justify-center bg-black">
                    <FrameImageViewer
                      v-if="!isDialogEntryCurrent(entry) || !dialogMotionPlaying || entry.item.row.motion?.status !== 'ready' || !entry.item.row.motion.artifactUrl"
                      :src="dialogEntryImageUrl(entry)"
                      :alt="`${entry.item.row.code} ${entry.item.row.positionSec}s`"
                      :active="isDialogEntryCurrent(entry)"
                      class="max-md:[&>div.absolute]:right-12"
                    />
                    <img
                      v-else-if="entry.item.row.motion.contentType === 'image/gif'"
                      :src="entry.item.row.motion.artifactUrl"
                      :alt="t('curated.motionAvailable')"
                      class="box-border h-full w-full object-contain p-2 sm:p-4"
                      draggable="false"
                    />
                    <video
                      v-else
                      :src="entry.item.row.motion.artifactUrl"
                      :poster="dialogEntryImageUrl(entry)"
                      class="box-border h-full w-full object-contain p-2 sm:p-4"
                      autoplay
                      muted
                      loop
                      playsinline
                      controls
                    />
                  </div>
                </CarouselItem>
              </CarouselContent>
            </Carousel>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="absolute top-1/2 left-3 size-10 -translate-y-1/2 rounded-xl bg-black/45 text-white shadow-lg ring-1 ring-white/15 transition hover:bg-black/65 hover:text-white focus-visible:ring-white/60 disabled:opacity-25 dark:hover:bg-black/65 sm:left-4 sm:size-11 sm:rounded-2xl"
              :disabled="!canNavigateDialogPrevious"
              :aria-label="t('curated.previousFrame')"
              @click="navigateDialogFrame('previous')"
            >
              <ChevronLeft class="size-5 sm:size-6" aria-hidden="true" />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="absolute top-1/2 right-3 size-10 -translate-y-1/2 rounded-xl bg-black/45 text-white shadow-lg ring-1 ring-white/15 transition hover:bg-black/65 hover:text-white focus-visible:ring-white/60 disabled:opacity-25 dark:hover:bg-black/65 sm:right-4 sm:size-11 sm:rounded-2xl"
              :disabled="!canNavigateDialogNext"
              :aria-label="t('curated.nextFrame')"
              @click="navigateDialogFrame('next')"
            >
              <ChevronRight class="size-5 sm:size-6" aria-hidden="true" />
            </Button>
            <Button
              v-if="selected?.motion?.status === 'ready' && selected.motion.artifactUrl"
              type="button"
              variant="ghost"
              size="sm"
              class="absolute bottom-4 left-1/2 z-10 -translate-x-1/2 rounded-full bg-black/65 text-white shadow-lg ring-1 ring-white/15 hover:bg-black/80 hover:text-white"
              :aria-pressed="dialogMotionPlaying"
              @click.stop="dialogMotionPlaying = !dialogMotionPlaying"
            >
              <PlayCircle class="size-4" aria-hidden="true" />
              {{ dialogMotionPlaying ? t("curated.stopMotion") : t("curated.playMotion") }}
            </Button>
          </div>
          <div
            class="flex min-h-0 flex-col gap-5 overflow-y-auto border-t border-border/70 p-5 sm:p-6 md:max-h-full md:border-t-0 md:border-l"
          >
            <DialogHeader class="space-y-1.5 text-left">
              <DialogTitle class="line-clamp-3 text-lg font-semibold leading-snug sm:text-xl">
                {{ selected?.title }}
              </DialogTitle>
            </DialogHeader>

            <dl class="space-y-3 text-sm">
              <div>
                <dt class="text-muted-foreground">{{ t("curated.fieldCode") }}</dt>
                <dd class="font-medium">{{ selected?.code }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">{{ t("curated.fieldActors") }}</dt>
                <dd>{{ selected?.actors?.length ? selected.actors.join("、") : "—" }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">{{ t("curated.fieldPosition") }}</dt>
                <dd>{{ selected ? formatClock(selected.positionSec) : "—" }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">{{ t("curated.fieldCapturedAt") }}</dt>
                <dd>{{ selected ? formatCapturedAt(selected.capturedAt) : "—" }}</dd>
              </div>
            </dl>

            <p
              v-if="selected && isNearDuplicateFrame(selected.id)"
              class="rounded-2xl border border-amber-300/70 bg-amber-50/80 px-3 py-2 text-xs text-amber-900 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-100"
            >
              {{ t("curated.duplicateReviewDialogHint", { threshold: curatedFrameNearDuplicateThresholdSec }) }}
            </p>

            <div class="flex flex-col gap-3">
              <p class="text-sm font-medium">{{ t("curated.tagsSectionTitle") }}</p>
              <div class="flex flex-wrap items-center gap-2">
                <Badge
                  v-for="tag in dialogTags"
                  :key="`frame-${tag}`"
                  variant="outline"
                  as-child
                  class="group rounded-full border-primary/35 bg-primary/5 pl-2 pr-1 text-foreground"
                >
                  <span class="inline-flex max-w-full items-center gap-0.5 rounded-[inherit] py-0.5 pl-1">
                    <button
                      type="button"
                      class="min-w-0 max-w-[12rem] cursor-pointer truncate rounded-md px-1.5 py-0.5 text-left text-xs font-medium transition hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                      :aria-label="t('curated.ariaFilterInLibrary', { tag })"
                      @click="browseCuratedFramesByTag(tag)"
                    >
                      {{ tag }}
                    </button>
                    <button
                      type="button"
                      class="inline-flex size-6 shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive"
                      :aria-label="t('curated.ariaRemoveTag', { tag })"
                      @click.stop="removeUserTag(tag)"
                    >
                      <X class="size-3.5" />
                    </button>
                  </span>
                </Badge>

                <div ref="userTagInlineZoneRef" class="flex max-w-full flex-wrap items-center gap-2">
                  <Button
                    type="button"
                    variant="secondary"
                    class="h-[29px] min-h-[29px] shrink-0 rounded-2xl px-3 py-0 text-xs has-[>svg]:px-2.5 [&_svg:not([class*='size-'])]:size-3.5"
                    @click="onUserTagAddButtonClick"
                  >
                    <Plus data-icon="inline-start" />
                    {{ t("common.add") }}
                  </Button>
                  <div
                    v-if="userTagInputOpen"
                    ref="userTagSuggestRootRef"
                    class="relative max-w-full min-w-[min(100%,12rem)] flex-1 sm:flex-initial"
                  >
                    <div
                      class="flex h-9 w-full items-center gap-0.5 rounded-2xl border border-border/80 bg-background/80 pl-3 pr-0.5 shadow-sm"
                    >
                      <input
                        ref="newUserTagInputRef"
                        v-model="newUserTagDraft"
                        type="text"
                        maxlength="64"
                        autocomplete="off"
                        role="combobox"
                        :aria-expanded="showUserTagSuggestions"
                        :aria-activedescendant="
                          highlightIndex >= 0 ? `${tagSuggestDomId}-opt-${highlightIndex}` : undefined
                        "
                        aria-autocomplete="list"
                        :aria-controls="showUserTagSuggestions ? `${tagSuggestDomId}-list` : undefined"
                        :placeholder="t('curated.newTagPlaceholder')"
                        class="placeholder:text-muted-foreground h-8 min-w-0 flex-1 border-0 bg-transparent px-0 text-sm shadow-none outline-none focus-visible:ring-0"
                        @keydown="onTagSuggestKeydown"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        class="size-8 shrink-0 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
                        :aria-label="t('curated.ariaCancelTagInput')"
                        @click="cancelUserTagInput"
                      >
                        <X class="size-4" />
                      </Button>
                    </div>
                    <ul
                      v-if="showUserTagSuggestions"
                      :id="`${tagSuggestDomId}-list`"
                      ref="userTagSuggestListRef"
                      class="absolute top-full left-0 z-50 mt-1 max-h-60 w-full min-w-[min(100%,12rem)] overflow-y-auto rounded-2xl border border-border/80 bg-popover/98 py-1 text-popover-foreground shadow-lg backdrop-blur-sm"
                      role="listbox"
                      :aria-label="t('curated.tagSuggestAria')"
                    >
                      <li v-for="(s, si) in filteredUserTagSuggestions" :key="s">
                        <button
                          :id="`${tagSuggestDomId}-opt-${si}`"
                          type="button"
                          role="option"
                          :data-tag-suggest-idx="si"
                          class="w-full truncate px-3 py-2 text-left text-sm transition-colors hover:bg-accent hover:text-accent-foreground"
                          :class="highlightIndex === si ? 'bg-muted' : ''"
                          :aria-selected="highlightIndex === si"
                          @mousedown.prevent="pickUserTagSuggestion(s)"
                        >
                          {{ s }}
                        </button>
                      </li>
                    </ul>
                  </div>
                </div>
              </div>
              <p v-if="userTagFormError" class="text-sm text-destructive">{{ userTagFormError }}</p>
              <div class="flex items-center justify-between gap-3">
                <p
                  v-if="dialogTagSaveStatus === 'saving'"
                  class="text-xs text-muted-foreground"
                >
                  {{ t("common.saving") }}
                </p>
                <p
                  v-else-if="dialogTagSaveStatus === 'error'"
                  class="text-xs text-destructive"
                  role="alert"
                >
                  {{ dialogTagSaveError }}
                </p>
                <span v-else class="text-xs text-muted-foreground"></span>
                <Button
                  v-if="shouldShowCuratedFrameTagRetry(dialogTagSaveStatus)"
                  type="button"
                  variant="outline"
                  size="sm"
                  class="h-8 rounded-xl px-3"
                  @click="saveDialogTagsNow"
                >
                  {{ t("curated.tagSaveRetry") }}
                </Button>
              </div>
            </div>

            <p v-if="dialogExportError" class="text-sm text-destructive" role="alert">{{ dialogExportError }}</p>

            <div class="mt-auto flex shrink-0 flex-col gap-2 border-t border-border/60 pt-5">
              <div
                class="grid grid-cols-3 gap-2"
                role="group"
                :aria-label="t('curated.exportActionsAria')"
              >
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  class="h-10 min-w-0 justify-center gap-1 rounded-xl px-2 text-xs"
                  :disabled="!useWebApi || exportBusy || dialogTagSaveStatus === 'saving'"
                  :title="!useWebApi ? t('curated.exportRequiresApi') : undefined"
                  @click="exportSingleFromDialog"
                >
                  <Download class="size-4 shrink-0" aria-hidden="true" />
                  <span class="truncate">{{ exportBusy ? t("curated.exportWorking") : t("curated.export") }}</span>
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  class="h-10 min-w-0 justify-center gap-1 rounded-xl px-2 text-xs"
                  :disabled="exportBusy || dialogTagSaveStatus === 'saving'"
                  @click="exportSingleFromDialogWatermarked"
                >
                  <Sparkles class="size-4 shrink-0" aria-hidden="true" />
                  <span class="truncate">{{ exportBusy ? t("curated.exportWorking") : t("curated.exportWatermarked") }}</span>
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  class="h-10 min-w-0 justify-center gap-1 rounded-xl px-2 text-xs"
                  :disabled="!selected?.motion || selected.motion.status !== 'ready' || !selected.motion.artifactUrl || exportBusy || dialogTagSaveStatus === 'saving'"
                  :title="!selected?.motion || selected.motion.status !== 'ready' || !selected.motion.artifactUrl ? t('curated.exportGifUnavailable') : undefined"
                  @click="exportSingleGifFromDialog"
                >
                  <Film class="size-4 shrink-0" aria-hidden="true" />
                  <span class="truncate">{{ exportBusy ? t("curated.exportWorking") : t("curated.exportGif") }}</span>
                </Button>
              </div>
              <Button
                type="button"
                size="sm"
                class="h-10 w-full justify-center gap-1.5 rounded-xl"
                :disabled="dialogTagSaveStatus === 'saving'"
                @click="playFromFrame"
              >
                <PlayCircle class="size-4 shrink-0" aria-hidden="true" />
                {{ t("curated.playFromTime") }}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                class="h-10 w-full justify-center gap-1.5 rounded-xl border-destructive/50 text-destructive hover:bg-destructive/10"
                @click="openDeleteConfirmFromDialog"
              >
                <Trash2 class="size-4 shrink-0" aria-hidden="true" />
                {{ t("curated.deleteThisFrame") }}
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>

    <CuratedFrameDeleteConfirmDialog
      v-model:open="deleteConfirmOpen"
      :label="deleteTargetLabel"
      :error="deleteFrameError"
      :busy="deleteFrameBusy"
      @confirm="executeDeleteCuratedFrame"
    />
</template>

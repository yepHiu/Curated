<script setup lang="ts">
import { useFocusWithin, onClickOutside } from "@vueuse/core"
import { computed, nextTick, onUnmounted, ref, useId, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { api } from "@/api/endpoints"
import type { CuratedFrameFacetItemDTO, PostCuratedFramesExportBody } from "@/api/types"
import {
  Camera,
  CheckSquare,
  Download,
  ListChecks,
  PlayCircle,
  Plus,
  Trash2,
  X,
} from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import CuratedFrameContextMenu from "@/components/jav-library/CuratedFrameContextMenu.vue"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { CuratedFrameRecord } from "@/domain/curated-frame/types"
import {
  getCuratedFrameSearchQuery,
  getCuratedFrameTagQuery,
  mergeCuratedFramesQuery,
} from "@/lib/library-query"
import type { CuratedFrameDbRow } from "@/lib/curated-frames/db"
import {
  deleteCuratedFrame,
  listCuratedFrameTagFacets,
  listCuratedFrameTagSuggestions,
  listCuratedFramesPage,
  updateCuratedFrameTags,
} from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import { useUserTagSuggestKeyboard } from "@/composables/use-user-tag-suggest-keyboard"
import { filterUserTagSuggestions } from "@/lib/user-tag-suggestions"
import { curatedFrameImageUrl, curatedFrameThumbnailUrl } from "@/lib/curated-frame-image-url"
import { triggerDownloadBlob } from "@/lib/curated-frames/export-file"
import { deleteCuratedFramesBatch } from "@/lib/curated-frames/batch-delete"
import {
  buildCuratedFrameNearDuplicateIndex,
  findCuratedFrameNearDuplicateGroups,
  type CuratedFrameNearDuplicateGroup,
} from "@/lib/curated-frames/near-duplicates"
import {
  commitCuratedFrameTags,
  shouldCommitCuratedFrameTagDraft,
  shouldShowCuratedFrameTagRetry,
  type CuratedFrameTagSaveStatus,
} from "@/lib/curated-frames/p2-state"
import { visibleCuratedFrameTagFacets } from "@/lib/curated-frames/tag-facets"
import { buildPlayerRouteFromCuratedFrame } from "@/lib/player-route"
import { pushAppToast } from "@/composables/use-app-toast"
import { useLibraryService } from "@/services/library-service"

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"
const curatedPageLimit = 60
const curatedTagFilterPreviewLimit = 16

/** 与资料库「批量管理」一致：勾选卡片并配合底部工具栏导出 */
const batchMode = ref(false)

const mainTab = ref<"timeline" | "actors" | "movies">("timeline")

type ExportSelectionBucket = "none" | "anonymous" | "named"
const exportSelectionBucket = ref<ExportSelectionBucket>("none")
const namedActorForExport = ref<string | null>(null)
const selectedFrameIds = ref<string[]>([])
const exportToolbarError = ref("")
const exportBusy = ref(false)
const batchDeleteBusy = ref(false)
const dialogOpenedFromActor = ref<string | null>(null)
const dialogExportError = ref("")

function isFrameSelected(id: string) {
  return selectedFrameIds.value.includes(id)
}

function clearExportSelection() {
  selectedFrameIds.value = []
  exportSelectionBucket.value = "none"
  namedActorForExport.value = null
  exportToolbarError.value = ""
}

function exitBatchMode() {
  batchMode.value = false
  clearExportSelection()
}

watch(mainTab, () => {
  batchMode.value = false
  clearExportSelection()
})

function toggleFrameSelection(id: string, sectionActor?: string) {
  exportToolbarError.value = ""
  const idx = selectedFrameIds.value.indexOf(id)
  if (idx >= 0) {
    selectedFrameIds.value = selectedFrameIds.value.filter((x) => x !== id)
    if (selectedFrameIds.value.length === 0) {
      exportSelectionBucket.value = "none"
      namedActorForExport.value = null
    }
    return
  }
  if (selectedFrameIds.value.length >= batchExportMax) {
    exportToolbarError.value = t("curated.exportSelectMax")
    return
  }

  if (mainTab.value === "actors" && sectionActor !== undefined) {
    const anonymous = sectionActor === noActorLabel.value
    if (exportSelectionBucket.value === "none") {
      exportSelectionBucket.value = anonymous ? "anonymous" : "named"
      if (!anonymous) {
        namedActorForExport.value = sectionActor
      }
    } else if (exportSelectionBucket.value === "anonymous") {
      if (!anonymous) {
        exportToolbarError.value = t("curated.exportActorMixed")
        return
      }
    } else if (namedActorForExport.value !== sectionActor || anonymous) {
      exportToolbarError.value = t("curated.exportActorMixed")
      return
    }
  }

  selectedFrameIds.value = [...selectedFrameIds.value, id]
}

const batchExportMax = 20

function selectAllVisibleUpTo20() {
  clearExportSelection()
  const cap = listWithUrls.value.slice(0, batchExportMax)
  selectedFrameIds.value = cap.map((x) => x.row.id)
}

/** 「按演员」某分组内全选（与单选勾选同一导出桶规则） */
function selectAllInActorSection(actorLabel: string) {
  exportToolbarError.value = ""
  const entry = actorGroups.value.find(([a]) => a === actorLabel)
  if (!entry) {
    return
  }
  const [, items] = entry
  const ids = [...new Set(items.map((x) => x.row.id))]
  let chosen = ids
  if (chosen.length > batchExportMax) {
    chosen = ids.slice(0, batchExportMax)
    pushAppToast(t("curated.batchSelectGroupCapped", { max: batchExportMax }), { variant: "warning" })
  }
  selectedFrameIds.value = chosen
  const anonymous = actorLabel === noActorLabel.value
  exportSelectionBucket.value = anonymous ? "anonymous" : "named"
  namedActorForExport.value = anonymous ? null : actorLabel
}

/** 「按影片」某分组内全选 */
function selectAllInMovieSection(movieKey: string) {
  exportToolbarError.value = ""
  const g = movieGroups.value.find((x) => x.movieKey === movieKey)
  if (!g) {
    return
  }
  const ids = [...new Set(g.items.map((x) => x.row.id))]
  let chosen = ids
  if (chosen.length > batchExportMax) {
    chosen = ids.slice(0, batchExportMax)
    pushAppToast(t("curated.batchSelectGroupCapped", { max: batchExportMax }), { variant: "warning" })
  }
  selectedFrameIds.value = chosen
  exportSelectionBucket.value = "none"
  namedActorForExport.value = null
}

function getCappedUniqueIdsForGroupItems(items: RowWithUrl[]): string[] {
  return [...new Set(items.map((x) => x.row.id))].slice(0, batchExportMax)
}

/** 分组头勾选：与「全选本组」一致，按导出上限截断 */
function isGroupHeaderFullySelected(items: RowWithUrl[]): boolean {
  const target = getCappedUniqueIdsForGroupItems(items)
  if (target.length === 0) {
    return false
  }
  const sel = selectedFrameIds.value
  if (sel.length !== target.length) {
    return false
  }
  const tset = new Set(target)
  return sel.every((id) => tset.has(id))
}

/** 取消分组头勾选后，在「按演员」下根据仍选中的 id 推断导出桶 */
function reconcileActorsTabExportBucket() {
  if (mainTab.value !== "actors") {
    return
  }
  const sel = selectedFrameIds.value
  if (sel.length === 0) {
    exportSelectionBucket.value = "none"
    namedActorForExport.value = null
    return
  }
  const selSet = new Set(sel)
  const candidates: string[] = []
  for (const [label, groupItems] of actorGroups.value) {
    const gids = new Set(groupItems.map((x) => x.row.id))
    if ([...selSet].every((id) => gids.has(id))) {
      candidates.push(label)
    }
  }
  if (candidates.length === 0) {
    exportSelectionBucket.value = "none"
    namedActorForExport.value = null
    return
  }
  const prefer = namedActorForExport.value
  let label = candidates[0]!
  if (prefer && candidates.includes(prefer)) {
    label = prefer
  }
  const anonymous = label === noActorLabel.value
  exportSelectionBucket.value = anonymous ? "anonymous" : "named"
  namedActorForExport.value = anonymous ? null : label
}

function onActorGroupHeaderCheckboxChange(actor: string, items: RowWithUrl[], ev: Event) {
  const checked = (ev.target as HTMLInputElement).checked
  exportToolbarError.value = ""
  if (checked) {
    selectAllInActorSection(actor)
    return
  }
  const groupIds = new Set(items.map((x) => x.row.id))
  selectedFrameIds.value = selectedFrameIds.value.filter((id) => !groupIds.has(id))
  if (selectedFrameIds.value.length === 0) {
    exportSelectionBucket.value = "none"
    namedActorForExport.value = null
  } else {
    reconcileActorsTabExportBucket()
  }
}

function onMovieGroupHeaderCheckboxChange(movieKey: string, items: RowWithUrl[], ev: Event) {
  const checked = (ev.target as HTMLInputElement).checked
  exportToolbarError.value = ""
  if (checked) {
    selectAllInMovieSection(movieKey)
    return
  }
  const groupIds = new Set(items.map((x) => x.row.id))
  selectedFrameIds.value = selectedFrameIds.value.filter((id) => !groupIds.has(id))
  if (selectedFrameIds.value.length === 0) {
    exportSelectionBucket.value = "none"
    namedActorForExport.value = null
  }
}

function actorNameForExportRequest(): string | undefined {
  if (mainTab.value !== "actors") {
    return undefined
  }
  if (exportSelectionBucket.value !== "named" || !namedActorForExport.value) {
    return undefined
  }
  return namedActorForExport.value
}

type CuratedExportFormat = NonNullable<PostCuratedFramesExportBody["format"]>

function preferredCuratedExportFormat(): CuratedExportFormat {
  return libraryService.curatedFrameExportFormat.value ?? "jpg"
}

async function runExport(
  ids: string[],
  actorName: string | undefined,
  format: CuratedExportFormat,
  errorTarget: "toolbar" | "dialog" | "toast",
) {
  exportBusy.value = true
  if (errorTarget === "toolbar") {
    exportToolbarError.value = ""
  } else if (errorTarget === "dialog") {
    dialogExportError.value = ""
  }
  const body: PostCuratedFramesExportBody = { ids, format }
  if (actorName !== undefined && actorName !== "") {
    body.actorName = actorName
  }
  try {
    const { blob, filename } = await api.postCuratedFramesExport(body)
    triggerDownloadBlob(blob, filename)
  } catch (err) {
    console.error("[curated-frames] export failed", err)
    const msg = t("curated.exportFailed")
    if (errorTarget === "toolbar") {
      exportToolbarError.value = msg
    } else if (errorTarget === "dialog") {
      dialogExportError.value = msg
    } else {
      pushAppToast(msg, { variant: "destructive" })
    }
  } finally {
    exportBusy.value = false
  }
}

async function exportSelected() {
  if (selectedFrameIds.value.length === 0) {
    return
  }
  await runExport(
    selectedFrameIds.value,
    actorNameForExportRequest(),
    preferredCuratedExportFormat(),
    "toolbar",
  )
}

async function exportSingleFromDialog() {
  if (!selected.value) {
    return
  }
  const actorName = resolveSingleFrameActorNameForExport(selected.value, dialogOpenedFromActor.value)
  await runExport([selected.value.id], actorName, preferredCuratedExportFormat(), "dialog")
}

async function exportSingleFromContextMenu() {
  const menu = frameContextMenu.value
  if (!menu) {
    return
  }
  closeFrameContextMenu()
  const actorName = resolveSingleFrameActorNameForExport(menu.frame, menu.fromActorSection)
  await runExport([menu.frame.id], actorName, preferredCuratedExportFormat(), "toast")
}

const maxFrameTags = 64
const maxFrameTagRunes = 64

interface RowWithUrl {
  row: CuratedFrameDbRow
  url: string
}

const rawRows = ref<CuratedFrameDbRow[]>([])
const listWithUrls = ref<RowWithUrl[]>([])
const totalRows = ref(0)
const rowsLoading = ref(false)
const rowsLoadingMore = ref(false)
const curatedTagFacets = ref<CuratedFrameFacetItemDTO[]>([])
const tagFiltersExpanded = ref(false)

function revokeAllUrls() {
  for (const x of listWithUrls.value) {
    if (x.url.startsWith("blob:")) {
      URL.revokeObjectURL(x.url)
    }
  }
  listWithUrls.value = []
}

function currentCuratedQuery() {
  return getCuratedFrameSearchQuery(route.query).trim()
}

function currentCuratedTagFilter() {
  return getCuratedFrameTagQuery(route.query).trim()
}

async function reloadFromDb() {
  rowsLoading.value = true
  try {
    const page = await listCuratedFramesPage({
      q: currentCuratedQuery(),
      tag: currentCuratedTagFilter(),
      limit: curatedPageLimit,
      offset: 0,
    })
    rawRows.value = page.items
    totalRows.value = page.total
  } finally {
    rowsLoading.value = false
  }
}

async function loadMoreRows() {
  if (rowsLoadingMore.value || rawRows.value.length >= totalRows.value) {
    return
  }
  rowsLoadingMore.value = true
  try {
    const page = await listCuratedFramesPage({
      q: currentCuratedQuery(),
      tag: currentCuratedTagFilter(),
      limit: curatedPageLimit,
      offset: rawRows.value.length,
    })
    rawRows.value = [...rawRows.value, ...page.items]
    totalRows.value = page.total
  } finally {
    rowsLoadingMore.value = false
  }
}

watch(
  [
    () => curatedFramesRevision.value,
    () => getCuratedFrameSearchQuery(route.query),
    () => getCuratedFrameTagQuery(route.query),
  ],
  () => {
    void reloadFromDb()
  },
  { immediate: true },
)

watch(
  rawRows,
  () => {
    revokeAllUrls()
    listWithUrls.value = rawRows.value.map((row) => ({
      row,
      url: row.imageBlob ? URL.createObjectURL(row.imageBlob) : curatedFrameThumbnailUrl(row.id),
    }))
  },
  { immediate: true, deep: true },
)

onUnmounted(() => {
  if (dialogTagSaveTimer) {
    clearTimeout(dialogTagSaveTimer)
    dialogTagSaveTimer = null
  }
  revokeAllUrls()
})

const isEmpty = computed(() => !rowsLoading.value && listWithUrls.value.length === 0)
const activeTagFilter = computed(() => getCuratedFrameTagQuery(route.query).trim())
const hasActiveFrameFilters = computed(
  () => currentCuratedQuery() !== "" || activeTagFilter.value !== "",
)
const isLibraryEmpty = computed(
  () =>
    !rowsLoading.value &&
    listWithUrls.value.length === 0 &&
    totalRows.value === 0 &&
    !hasActiveFrameFilters.value &&
    curatedTagFacets.value.length === 0,
)
const isFilteredEmpty = computed(
  () => !rowsLoading.value && listWithUrls.value.length === 0 && !isLibraryEmpty.value,
)
const hasMoreRows = computed(() => rawRows.value.length < totalRows.value)
const visibleCuratedTagFacets = computed(() => {
  const facets = visibleCuratedFrameTagFacets(
    curatedTagFacets.value,
    curatedTagFilterPreviewLimit,
    tagFiltersExpanded.value,
  )
  const active = activeTagFilter.value
  if (!active || tagFiltersExpanded.value || facets.some((item) => item.name === active)) {
    return facets
  }
  const activeFacet = curatedTagFacets.value.find((item) => item.name === active)
  return activeFacet
    ? [activeFacet, ...facets.slice(0, Math.max(curatedTagFilterPreviewLimit - 1, 0))]
    : facets
})
const hiddenCuratedTagFacetCount = computed(() =>
  Math.max(curatedTagFacets.value.length - curatedTagFilterPreviewLimit, 0),
)
const curatedFrameNearDuplicateThresholdSec = 3
const nearDuplicateGroups = computed(() =>
  findCuratedFrameNearDuplicateGroups(rawRows.value, curatedFrameNearDuplicateThresholdSec),
)
const nearDuplicateFrameIds = computed(() => buildCuratedFrameNearDuplicateIndex(nearDuplicateGroups.value))
const nearDuplicateSummaryGroups = computed(() => nearDuplicateGroups.value.slice(0, 3))

/** 「按演员」视图下跨分组全选会混演员，与导出规则冲突，故不提供全选可见 */
const batchShowSelectVisible = computed(() => mainTab.value !== "actors")

const noActorLabel = computed(() => t("curated.noActor"))
const noMovieLabel = computed(() => t("curated.noMovie"))

/** 无 movieId 时归入同一组，避免 Map 用空串作 key 歧义 */
const UNKNOWN_MOVIE_KEY = "__curated_no_movie__"

const actorGroups = computed(() => {
  const none = noActorLabel.value
  const m = new Map<string, RowWithUrl[]>()
  for (const item of listWithUrls.value) {
    const acts = item.row.actors.length > 0 ? item.row.actors : [none]
    for (const a of acts) {
      const k = a.trim() || none
      if (!m.has(k)) m.set(k, [])
      m.get(k)!.push(item)
    }
  }
  for (const arr of m.values()) {
    arr.sort((x, y) => y.row.capturedAt.localeCompare(x.row.capturedAt))
  }
  return [...m.entries()].sort(([a], [b]) =>
    a.localeCompare(b, locale.value, { numeric: true }),
  )
})

interface MovieGroupSection {
  movieKey: string
  heading: string
  sub: string
  items: RowWithUrl[]
}

const movieGroups = computed((): MovieGroupSection[] => {
  void locale.value
  const none = noMovieLabel.value
  const m = new Map<string, RowWithUrl[]>()
  for (const item of listWithUrls.value) {
    const mid = item.row.movieId.trim()
    const key = mid || UNKNOWN_MOVIE_KEY
    if (!m.has(key)) {
      m.set(key, [])
    }
    m.get(key)!.push(item)
  }
  for (const arr of m.values()) {
    arr.sort((x, y) => y.row.capturedAt.localeCompare(x.row.capturedAt))
  }
  const entries = [...m.entries()].sort(([, ia], [, ib]) => {
    const ca = ia[0]?.row.capturedAt ?? ""
    const cb = ib[0]?.row.capturedAt ?? ""
    return cb.localeCompare(ca)
  })
  return entries.map(([movieKey, items]) => {
    const r = items[0]!.row
    const code = r.code.trim()
    const title = r.title.trim()
    const isUnknown = movieKey === UNKNOWN_MOVIE_KEY
    if (isUnknown) {
      const line = [code, title].filter(Boolean).join(code && title ? " · " : "")
      return { movieKey, heading: none, sub: line, items }
    }
    if (code) {
      return {
        movieKey,
        heading: code,
        sub: title && title !== code ? title : "",
        items,
      }
    }
    return {
      movieKey,
      heading: title || movieKey,
      sub: "",
      items,
    }
  })
})

const dialogOpen = ref(false)
const selected = ref<CuratedFrameRecord | null>(null)
const selectedImageUrl = ref("")
type CuratedFrameContextMenuState = {
  x: number
  y: number
  frame: CuratedFrameRecord
  fromActorSection: string | null
}

const frameContextMenu = ref<CuratedFrameContextMenuState | null>(null)
const dialogTags = ref<string[]>([])
const dialogTagSaveStatus = ref<CuratedFrameTagSaveStatus>("idle")
const dialogTagSaveError = ref("")
const lastSavedDialogTags = ref<string[]>([])
const dialogTagSaveDebounceMs = 250

let dialogTagSaveTimer: ReturnType<typeof setTimeout> | null = null
let dialogTagSaveInFlight = false

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

watch(
  [() => curatedFramesRevision.value, () => locale.value],
  () => {
    void reloadTagSuggestions()
    void reloadTagFacets()
  },
  { immediate: true },
)

async function reloadTagFacets() {
  try {
    curatedTagFacets.value = await listCuratedFrameTagFacets(locale.value)
  } catch {
    curatedTagFacets.value = []
  }
}

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
  selectedImageUrl.value = ""
  resetTagInputState()
  resetDialogTagSaveState()
  dialogOpen.value = false
}

function closeFrameContextMenu() {
  frameContextMenu.value = null
}

function onFrameContextMenu(
  event: MouseEvent,
  item: RowWithUrl,
  fromActorSection: string | null = null,
) {
  frameContextMenu.value = {
    x: event.clientX,
    y: event.clientY,
    frame: item.row,
    fromActorSection,
  }
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
  closeFrameContextMenu()
  dialogOpenedFromActor.value = fromActorSection
  dialogExportError.value = ""
  selected.value = meta
  selectedImageUrl.value = imageBlob ? item.url : curatedFrameImageUrl(item.row.id)
  dialogTags.value = [...item.row.tags]
  resetTagInputState()
  resetDialogTagSaveState(item.row.tags)
  dialogOpen.value = true
}

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

function isCuratedTagFilterActive(tag: string) {
  return activeTagFilter.value === tag.trim()
}

function setCuratedFrameTagFilter(tag: string | undefined) {
  const normalized = tag?.trim()
  void router.replace({
    name: "curated-frames",
    query: mergeCuratedFramesQuery(route.query, { cft: normalized || undefined }),
  })
}

function toggleCuratedFrameTagFilter(tag: string) {
  const normalized = tag.trim()
  if (!normalized) {
    return
  }
  setCuratedFrameTagFilter(isCuratedTagFilterActive(normalized) ? undefined : normalized)
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
    query: mergeCuratedFramesQuery(route.query, { cft: t }),
  })
}

async function playFromFrame() {
  if (!selected.value) return
  if (!(await persistDialogTags({ toastOnError: true }))) {
    return
  }
  const { movieId, positionSec } = selected.value
  resetDialogState()
  await router.push(buildPlayerRouteFromCuratedFrame(movieId, positionSec))
}

/** 删除萃取帧（确认弹窗） */
const deleteConfirmOpen = ref(false)
const deleteTargetIds = ref<string[]>([])
const deleteTargetLabel = ref("")
const deleteFrameBusy = ref(false)
const deleteFrameError = ref("")

function resetDeleteConfirmState() {
  deleteConfirmOpen.value = false
  deleteTargetIds.value = []
  deleteTargetLabel.value = ""
}

function frameLabelForDelete(id: string) {
  const row = rawRows.value.find((item) => item.id === id)
  if (!row) {
    return t("curated.deleteLabel")
  }
  return row.code.trim() || row.title.trim().slice(0, 48) || t("curated.deleteLabel")
}

function openDeleteConfirmFromDialog() {
  if (!selected.value) return
  closeFrameContextMenu()
  deleteFrameError.value = ""
  deleteTargetIds.value = [selected.value.id]
  deleteTargetLabel.value = frameLabelForDelete(selected.value.id)
  deleteConfirmOpen.value = true
}

function openDeleteConfirmFromContextMenu() {
  const menu = frameContextMenu.value
  if (!menu) {
    return
  }
  closeFrameContextMenu()
  deleteFrameError.value = ""
  deleteTargetIds.value = [menu.frame.id]
  deleteTargetLabel.value = frameLabelForDelete(menu.frame.id)
  deleteConfirmOpen.value = true
}

function openDeleteConfirmForSelectedFrames() {
  const ids = [...new Set(selectedFrameIds.value)]
  if (ids.length === 0) {
    return
  }
  deleteFrameError.value = ""
  deleteTargetIds.value = ids
  deleteTargetLabel.value =
    ids.length === 1
      ? frameLabelForDelete(ids[0]!)
      : t("curated.deleteSelectedLabel", { n: ids.length })
  deleteConfirmOpen.value = true
}

function applyDeletedFrameIds(deletedIds: readonly string[]) {
  if (deletedIds.length === 0) {
    return
  }

  const deletedSet = new Set(deletedIds)
  selectedFrameIds.value = selectedFrameIds.value.filter((id) => !deletedSet.has(id))
  if (selectedFrameIds.value.length === 0) {
    exportSelectionBucket.value = "none"
    namedActorForExport.value = null
  } else if (mainTab.value === "actors") {
    reconcileActorsTabExportBucket()
  }

  if (selected.value?.id && deletedSet.has(selected.value.id)) {
    selected.value = null
    selectedImageUrl.value = ""
    resetTagInputState()
    dialogOpen.value = false
  }
}

async function executeDeleteCuratedFrame() {
  const ids = [...deleteTargetIds.value]
  if (ids.length === 0) return
  deleteFrameError.value = ""
  deleteFrameBusy.value = true
  batchDeleteBusy.value = ids.length > 1
  try {
    const result = await deleteCuratedFramesBatch(ids, deleteCuratedFrame)
    applyDeletedFrameIds(result.deletedIds)

    if (result.ok) {
      resetDeleteConfirmState()
      return
    }

    console.error("[curated-frames] delete failed", result.error)
    deleteFrameError.value = result.deletedIds.length > 0
      ? t("curated.deletePartialFailed", { done: result.deletedIds.length, total: ids.length })
      : t("curated.deleteFailed")
  } catch (err) {
    console.error("[curated-frames] delete failed", err)
    deleteFrameError.value = t("curated.deleteFailed")
  } finally {
    deleteFrameBusy.value = false
    batchDeleteBusy.value = false
  }
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
  return nearDuplicateFrameIds.value.has(frameId)
}

function formatNearDuplicateGroup(group: CuratedFrameNearDuplicateGroup<RowWithUrl["row"]>) {
  const lead = group.items[0]
  if (!lead) {
    return ""
  }
  const label = lead.code.trim() || lead.title.trim() || lead.movieId.trim()
  const positions = group.items.map((item) => formatClock(item.positionSec)).join(" / ")
  return `${label} · ${positions}`
}

defineExpose({
  batchMode,
  isEmpty,
  selectedFrameIds,
  exportBusy,
  batchDeleteBusy,
  batchShowSelectVisible,
  exportToolbarError,
  exitBatchMode,
  clearExportSelection,
  selectAllVisibleUpTo20,
  deleteSelectedFrames: () => {
    openDeleteConfirmForSelectedFrames()
  },
  exportSelected,
})
</script>

<template>
  <div
    class="relative isolate mx-auto flex h-full min-h-0 w-full max-w-6xl flex-col gap-6 px-3 sm:px-6"
  >
    <h1 class="pt-2 text-2xl font-semibold tracking-tight">{{ t("curated.title") }}</h1>

    <div
      v-if="nearDuplicateGroups.length > 0"
      class="rounded-3xl border border-amber-300/70 bg-amber-50/80 p-4 text-sm text-amber-950 shadow-sm dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-100"
    >
      <p class="font-medium">
        {{ t("curated.duplicateReviewTitle", { count: nearDuplicateGroups.length }) }}
      </p>
      <p class="mt-1 text-amber-900/80 dark:text-amber-100/80">
        {{ t("curated.duplicateReviewBody", { threshold: curatedFrameNearDuplicateThresholdSec }) }}
      </p>
      <div class="mt-3 flex flex-wrap gap-2">
        <span
          v-for="group in nearDuplicateSummaryGroups"
          :key="`${group.movieId}-${group.items.map((item) => item.id).join('-')}`"
          class="rounded-full border border-amber-400/60 bg-background/70 px-2.5 py-1 text-xs text-foreground"
        >
          {{ formatNearDuplicateGroup(group) }}
        </span>
      </div>
    </div>

    <div
      v-if="isLibraryEmpty"
      class="flex flex-col items-center justify-center gap-3 rounded-3xl border border-dashed border-border/70 bg-muted/20 py-16 text-center"
    >
      <Camera class="size-12 text-muted-foreground" />
      <p class="text-sm text-muted-foreground">{{ t("curated.empty") }}</p>
    </div>

    <Tabs
      v-else
      v-model="mainTab"
      class="flex min-h-0 w-full min-w-0 flex-1 flex-col gap-4 overflow-hidden"
    >
      <div class="flex shrink-0 flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-3">
          <TabsList class="h-auto w-fit max-w-full flex-wrap rounded-2xl bg-muted/60 p-1">
            <TabsTrigger value="timeline" class="rounded-xl px-4 py-2">{{ t("curated.tabTimeline") }}</TabsTrigger>
            <TabsTrigger value="actors" class="rounded-xl px-4 py-2">{{ t("curated.tabActors") }}</TabsTrigger>
            <TabsTrigger value="movies" class="rounded-xl px-4 py-2">{{ t("curated.tabMovies") }}</TabsTrigger>
          </TabsList>
          <p class="text-xs text-muted-foreground">
            {{ t("curated.pageSummary", { shown: rawRows.length, total: totalRows }) }}
          </p>
        </div>
        <div class="flex shrink-0 flex-wrap items-center gap-2">
          <template v-if="!batchMode">
            <Button
              type="button"
              variant="outline"
              size="sm"
              class="gap-1.5 rounded-xl"
              @click="batchMode = true"
            >
              <ListChecks class="size-4 opacity-80" aria-hidden="true" />
              {{ t("library.batchManage") }}
            </Button>
          </template>
          <template v-else>
            <Button
              v-if="batchShowSelectVisible"
              type="button"
              variant="outline"
              size="sm"
              class="gap-1.5 rounded-xl"
              :disabled="listWithUrls.length === 0"
              @click="selectAllVisibleUpTo20"
            >
              <CheckSquare class="size-4 opacity-80" aria-hidden="true" />
              {{ t("library.batchSelectVisible") }}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              class="gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground"
              @click="exitBatchMode"
            >
              <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
              {{ t("library.batchExitToolbar") }}
            </Button>
          </template>
        </div>
      </div>

      <section
        class="shrink-0 rounded-3xl border border-border/70 bg-card/85 px-4 py-3 shadow-lg shadow-black/5"
        :aria-label="t('curated.tagFilterTitle')"
      >
        <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div class="min-w-0 space-y-2">
            <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">
              {{ t("curated.tagFilterTitle") }}
            </p>
            <div v-if="curatedTagFacets.length > 0" class="flex flex-wrap gap-2">
              <Badge
                as-child
                :variant="!activeTagFilter ? 'default' : 'secondary'"
                :class="[
                  'rounded-full border px-3 py-1 text-sm font-normal transition-colors',
                  !activeTagFilter
                    ? 'border-primary/40'
                    : 'cursor-pointer border-border/60 bg-secondary/70 hover:bg-secondary hover:text-secondary-foreground',
                ]"
              >
                <button
                  type="button"
                  class="inline-flex max-w-full min-w-0 items-center gap-1.5"
                  :aria-pressed="!activeTagFilter"
                  :aria-label="t('curated.ariaClearFrameTagFilter')"
                  @click="setCuratedFrameTagFilter(undefined)"
                >
                  {{ t("curated.tagFilterAll") }}
                </button>
              </Badge>
              <Badge
                v-for="tag in visibleCuratedTagFacets"
                :key="tag.name"
                as-child
                :variant="isCuratedTagFilterActive(tag.name) ? 'default' : 'secondary'"
                :class="[
                  'max-w-[14rem] rounded-full border px-3 py-1 text-sm font-normal transition-colors',
                  isCuratedTagFilterActive(tag.name)
                    ? 'border-primary/40'
                    : 'cursor-pointer border-border/60 bg-secondary/70 hover:bg-secondary hover:text-secondary-foreground',
                ]"
              >
                <button
                  type="button"
                  class="inline-flex max-w-full min-w-0 items-center gap-1.5"
                  :aria-pressed="isCuratedTagFilterActive(tag.name)"
                  :aria-label="t('curated.ariaFilterFrameTag', { tag: tag.name, count: tag.count })"
                  @click="toggleCuratedFrameTagFilter(tag.name)"
                >
                  <span class="truncate">{{ tag.name }}</span>
                  <span
                    class="tabular-nums text-xs opacity-80"
                    :class="
                      isCuratedTagFilterActive(tag.name)
                        ? 'text-primary-foreground/90'
                        : 'text-muted-foreground'
                    "
                  >
                    · {{ tag.count }}
                  </span>
                </button>
              </Badge>
            </div>
            <p v-else class="text-sm text-muted-foreground">
              {{ t("curated.tagFilterEmpty") }}
            </p>
          </div>
          <Button
            v-if="hiddenCuratedTagFacetCount > 0"
            type="button"
            variant="ghost"
            size="sm"
            class="h-8 shrink-0 rounded-full px-3 text-xs text-muted-foreground hover:text-foreground"
            @click="tagFiltersExpanded = !tagFiltersExpanded"
          >
            {{
              tagFiltersExpanded
                ? t("curated.tagFilterShowLess")
                : t("curated.tagFilterShowMore", { count: hiddenCuratedTagFacetCount })
            }}
          </Button>
        </div>
      </section>

      <div
        class="min-h-0 flex-1 overflow-y-auto pb-2 pr-3 [scrollbar-gutter:stable] sm:pr-4"
      >
      <div
        v-if="isFilteredEmpty"
        class="mb-4 flex flex-col items-center justify-center gap-3 rounded-3xl border border-dashed border-border/70 bg-muted/20 py-12 text-center"
      >
        <Camera class="size-10 text-muted-foreground" />
        <p class="text-sm text-muted-foreground">{{ t("curated.tagFilterNoMatches") }}</p>
        <Button
          v-if="activeTagFilter"
          type="button"
          variant="outline"
          size="sm"
          class="rounded-2xl"
          @click="setCuratedFrameTagFilter(undefined)"
        >
          {{ t("curated.tagFilterAll") }}
        </Button>
      </div>
      <TabsContent value="timeline" class="mt-0 outline-none">
        <div
          class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4"
        >
          <div
            v-for="item in listWithUrls"
            :key="item.row.id"
            class="group relative min-w-0 overflow-hidden rounded-2xl border bg-card/90 shadow-md transition hover:border-primary/40 hover:shadow-lg"
            :class="isNearDuplicateFrame(item.row.id) ? 'border-amber-400/70' : 'border-border/70'"
          >
            <label
              v-if="batchMode"
              class="absolute top-2 left-2 z-10 flex cursor-pointer items-center justify-center rounded-md p-1.5 text-primary transition-colors hover:bg-foreground/12 focus-within:outline-none focus-within:ring-2 focus-within:ring-ring dark:hover:bg-black/50"
              :title="t('curated.exportToggleAria')"
              @click.stop
            >
              <input
                type="checkbox"
                class="size-4 cursor-pointer rounded accent-primary"
                :checked="isFrameSelected(item.row.id)"
                :aria-label="t('curated.exportToggleAria')"
                @change="toggleFrameSelection(item.row.id)"
              />
            </label>
            <span
              v-if="isNearDuplicateFrame(item.row.id)"
              class="absolute top-2 right-2 z-10 rounded-full bg-amber-500/90 px-2 py-0.5 text-[11px] font-medium text-white shadow-sm"
            >
              {{ t("curated.duplicateReviewBadge") }}
            </span>
            <button
              type="button"
              class="block w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              @contextmenu.prevent="onFrameContextMenu($event, item)"
              @click="openDialog(item)"
            >
              <div class="relative aspect-video w-full bg-black/80">
                <img
                  :src="item.url"
                  :alt="item.row.code"
                  class="h-full w-full object-contain"
                  loading="lazy"
                />
              </div>
              <div class="space-y-1 p-3">
                <p class="line-clamp-2 text-sm font-medium">{{ item.row.title }}</p>
                <p class="text-xs text-muted-foreground">
                  {{ item.row.code }} · {{ formatClock(item.row.positionSec) }}
                </p>
              </div>
            </button>
          </div>
        </div>
      </TabsContent>

      <TabsContent value="actors" class="mt-0 outline-none">
        <div class="flex flex-col gap-8">
          <section v-for="[actor, items] in actorGroups" :key="actor">
            <div class="mb-3">
              <label
                v-if="batchMode"
                class="flex cursor-pointer items-start gap-2.5"
              >
                <input
                  type="checkbox"
                  class="mt-1 size-4 shrink-0 cursor-pointer rounded accent-primary"
                  :disabled="items.length === 0"
                  :checked="isGroupHeaderFullySelected(items)"
                  :aria-label="t('curated.batchSelectActorGroup')"
                  @change="onActorGroupHeaderCheckboxChange(actor, items, $event)"
                />
                <span class="min-w-0 text-lg font-semibold leading-snug">{{ actor }}</span>
              </label>
              <h2
                v-else
                class="text-lg font-semibold"
              >
                {{ actor }}
              </h2>
            </div>
            <div
              class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4"
            >
              <div
                v-for="item in items"
                :key="`${actor}-${item.row.id}`"
                class="group relative min-w-0 overflow-hidden rounded-2xl border bg-card/90 shadow-md transition hover:border-primary/40 hover:shadow-lg"
                :class="isNearDuplicateFrame(item.row.id) ? 'border-amber-400/70' : 'border-border/70'"
              >
                <label
                  v-if="batchMode"
                  class="absolute top-2 left-2 z-10 flex cursor-pointer items-center justify-center rounded-md p-1.5 text-primary transition-colors hover:bg-foreground/12 focus-within:outline-none focus-within:ring-2 focus-within:ring-ring dark:hover:bg-black/50"
                  :title="t('curated.exportToggleAria')"
                  @click.stop
                >
                  <input
                    type="checkbox"
                    class="size-4 cursor-pointer rounded accent-primary"
                    :checked="isFrameSelected(item.row.id)"
                    :aria-label="t('curated.exportToggleAria')"
                    @change="toggleFrameSelection(item.row.id, actor)"
                  />
                </label>
                <span
                  v-if="isNearDuplicateFrame(item.row.id)"
                  class="absolute top-2 right-2 z-10 rounded-full bg-amber-500/90 px-2 py-0.5 text-[11px] font-medium text-white shadow-sm"
                >
                  {{ t("curated.duplicateReviewBadge") }}
                </span>
                <button
                  type="button"
                  class="block w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  @contextmenu.prevent="onFrameContextMenu($event, item, actor)"
                  @click="openDialog(item, actor)"
                >
                  <div class="relative aspect-video w-full bg-black/80">
                    <img
                      :src="item.url"
                      :alt="item.row.code"
                      class="h-full w-full object-contain"
                      loading="lazy"
                    />
                  </div>
                  <div class="space-y-1 p-3">
                    <p class="line-clamp-2 text-sm font-medium">{{ item.row.title }}</p>
                    <p class="text-xs text-muted-foreground">
                      {{ item.row.code }} · {{ formatClock(item.row.positionSec) }}
                    </p>
                  </div>
                </button>
              </div>
            </div>
          </section>
        </div>
      </TabsContent>

      <TabsContent value="movies" class="mt-0 outline-none">
        <div class="flex flex-col gap-8">
          <section v-for="g in movieGroups" :key="g.movieKey">
            <div class="mb-3">
              <label
                v-if="batchMode"
                class="flex cursor-pointer items-start gap-2.5"
              >
                <input
                  type="checkbox"
                  class="mt-1 size-4 shrink-0 cursor-pointer rounded accent-primary"
                  :disabled="g.items.length === 0"
                  :checked="isGroupHeaderFullySelected(g.items)"
                  :aria-label="t('curated.batchSelectMovieGroup')"
                  @change="onMovieGroupHeaderCheckboxChange(g.movieKey, g.items, $event)"
                />
                <span class="min-w-0 flex-1">
                  <span class="block text-lg font-semibold leading-snug">{{ g.heading }}</span>
                  <p
                    v-if="g.sub"
                    class="mt-0.5 line-clamp-2 text-sm text-muted-foreground"
                  >
                    {{ g.sub }}
                  </p>
                </span>
              </label>
              <template v-else>
                <h2 class="text-lg font-semibold">{{ g.heading }}</h2>
                <p
                  v-if="g.sub"
                  class="mt-0.5 line-clamp-2 text-sm text-muted-foreground"
                >
                  {{ g.sub }}
                </p>
              </template>
            </div>
            <div
              class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4"
            >
              <div
                v-for="item in g.items"
                :key="`${g.movieKey}-${item.row.id}`"
                class="group relative min-w-0 overflow-hidden rounded-2xl border bg-card/90 shadow-md transition hover:border-primary/40 hover:shadow-lg"
                :class="isNearDuplicateFrame(item.row.id) ? 'border-amber-400/70' : 'border-border/70'"
              >
                <label
                  v-if="batchMode"
                  class="absolute top-2 left-2 z-10 flex cursor-pointer items-center justify-center rounded-md p-1.5 text-primary transition-colors hover:bg-foreground/12 focus-within:outline-none focus-within:ring-2 focus-within:ring-ring dark:hover:bg-black/50"
                  :title="t('curated.exportToggleAria')"
                  @click.stop
                >
                  <input
                    type="checkbox"
                    class="size-4 cursor-pointer rounded accent-primary"
                    :checked="isFrameSelected(item.row.id)"
                    :aria-label="t('curated.exportToggleAria')"
                    @change="toggleFrameSelection(item.row.id)"
                  />
                </label>
                <span
                  v-if="isNearDuplicateFrame(item.row.id)"
                  class="absolute top-2 right-2 z-10 rounded-full bg-amber-500/90 px-2 py-0.5 text-[11px] font-medium text-white shadow-sm"
                >
                  {{ t("curated.duplicateReviewBadge") }}
                </span>
                <button
                  type="button"
                  class="block w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  @contextmenu.prevent="onFrameContextMenu($event, item)"
                  @click="openDialog(item)"
                >
                  <div class="relative aspect-video w-full bg-black/80">
                    <img
                      :src="item.url"
                      :alt="item.row.code"
                      class="h-full w-full object-contain"
                      loading="lazy"
                    />
                  </div>
                  <div class="space-y-1 p-3">
                    <p class="line-clamp-2 text-sm font-medium">{{ item.row.title }}</p>
                    <p class="text-xs text-muted-foreground">
                      {{ item.row.code }} · {{ formatClock(item.row.positionSec) }}
                    </p>
                  </div>
                </button>
              </div>
            </div>
          </section>
        </div>
      </TabsContent>
      <div v-if="hasMoreRows" class="flex justify-center py-5">
        <Button
          type="button"
          variant="outline"
          class="rounded-2xl"
          :disabled="rowsLoadingMore"
          @click="loadMoreRows"
        >
          {{ rowsLoadingMore ? t("common.loading") : t("curated.loadMore") }}
        </Button>
      </div>
      </div>
    </Tabs>

    <Dialog :open="dialogOpen" @update:open="handleDialogOpenChange">
      <!-- 覆盖 DialogContent 默认 sm:max-w-lg，否则整窗约 512px 宽，左侧预览会被压成一条 -->
      <DialogContent
        class="h-[min(94vh,960px)] max-h-[min(94vh,960px)] w-[min(98vw,92rem)] max-w-[min(98vw,92rem)] gap-0 overflow-hidden border-border/70 p-0 sm:max-w-[min(98vw,92rem)]"
      >
        <div
          class="grid h-full min-h-0 w-full grid-cols-1 grid-rows-1 md:grid-cols-[minmax(0,2.4fr)_minmax(16rem,22rem)] lg:grid-cols-[minmax(0,2.75fr)_minmax(17rem,24rem)]"
        >
          <div
            class="relative flex h-[min(52vh,560px)] w-full min-w-0 items-center justify-center bg-black md:h-full md:min-h-0"
          >
            <img
              v-if="selectedImageUrl"
              :src="selectedImageUrl"
              alt=""
              class="box-border h-full w-full object-contain p-2 sm:p-4"
            />
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

            <div
              class="mt-auto flex shrink-0 flex-col gap-4 border-t border-border/60 pt-5"
            >
              <div
                v-if="useWebApi"
                class="grid grid-cols-1 gap-2"
              >
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  class="h-10 w-full justify-center gap-1.5 rounded-xl px-2"
                  :disabled="exportBusy || dialogTagSaveStatus === 'saving'"
                  @click="exportSingleFromDialog"
                >
                  <Download class="size-4 shrink-0" aria-hidden="true" />
                  <span class="truncate">{{ exportBusy ? t("curated.exportWorking") : t("curated.export") }}</span>
                </Button>
              </div>
              <div class="flex flex-col gap-2">
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
        </div>
      </DialogContent>
    </Dialog>

    <CuratedFrameContextMenu
      v-if="frameContextMenu"
      :frame="frameContextMenu.frame"
      :x="frameContextMenu.x"
      :y="frameContextMenu.y"
      :use-web-api="useWebApi"
      @close="closeFrameContextMenu"
      @export="exportSingleFromContextMenu"
      @delete="openDeleteConfirmFromContextMenu"
    />

    <Dialog v-model:open="deleteConfirmOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t("curated.deleteCard") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("curated.deleteConfirm", { label: deleteTargetLabel }) }}
          </DialogDescription>
        </DialogHeader>
        <p v-if="deleteFrameError" class="text-sm text-destructive" role="alert">
          {{ deleteFrameError }}
        </p>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl" :disabled="deleteFrameBusy">
              {{ t("curated.cancel") }}
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            class="rounded-2xl"
            :disabled="deleteFrameBusy"
            @click="executeDeleteCuratedFrame"
          >
            {{ deleteFrameBusy ? t("curated.deleteWorking") : t("curated.confirmDelete") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

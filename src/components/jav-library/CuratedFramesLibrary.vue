<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import type { CuratedFrameFacetItemDTO } from "@/api/types"
import { Button } from "@/components/ui/button"
import CuratedFrameDetailDialog from "@/components/jav-library/CuratedFrameDetailDialog.vue"
import { useCuratedFrameExport, type CuratedExportFormat, type CuratedExportMode } from "@/composables/use-curated-frame-export"
import CuratedFrameTimelineTab from "@/components/jav-library/CuratedFrameTimelineTab.vue"
import CuratedFrameActorsTab from "@/components/jav-library/CuratedFrameActorsTab.vue"
import CuratedFrameMoviesTab from "@/components/jav-library/CuratedFrameMoviesTab.vue"
import CuratedFrameTagFilterBar from "@/components/jav-library/CuratedFrameTagFilterBar.vue"
import CuratedFrameLibraryToolbar from "@/components/jav-library/CuratedFrameLibraryToolbar.vue"
import CuratedFrameDeleteConfirmDialog from "@/components/jav-library/CuratedFrameDeleteConfirmDialog.vue"
import CuratedFrameEmptyState from "@/components/jav-library/CuratedFrameEmptyState.vue"
import CuratedFrameContextMenu from "@/components/jav-library/CuratedFrameContextMenu.vue"
import { Tabs, TabsContent } from "@/components/ui/tabs"
import type { CuratedFrameRecord } from "@/domain/curated-frame/types"
import {
  getCuratedFrameSearchQuery,
  getCuratedFrameTagFilters,
  mergeCuratedFramesQuery,
  serializeCuratedFrameTagFilters,
} from "@/lib/library-query"
import type { CuratedFrameDbRow } from "@/lib/curated-frames/db"
import {
  deleteCuratedFrame,
  listCuratedFrameTagFacets,
  listCuratedFramesPage,
} from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import { curatedFrameThumbnailUrl } from "@/lib/curated-frame-image-url"
import { deleteCuratedFramesBatch } from "@/lib/curated-frames/batch-delete"
import {
  buildCuratedFrameNearDuplicateIndex,
  findCuratedFrameNearDuplicateGroups,
} from "@/lib/curated-frames/near-duplicates"
import {
  clearCuratedFrameExportSelection,
  reconcileCuratedFrameActorExportSelection,
  toggleCuratedFrameExportSelection,
  type CuratedFrameExportSelectionState,
} from "@/lib/curated-frames/selection"
import {
  buildCuratedFrameDialogNavigationEntries,
} from "@/lib/curated-frames/dialog-navigation"
import { pushAppToast } from "@/composables/use-app-toast"
import { useLibraryService } from "@/services/library-service"

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"
const curatedPageLimit = 60

/** 与资料库「批量管理」一致：勾选卡片并配合底部工具栏导出 */
const batchMode = ref(false)

const mainTab = ref<"timeline" | "actors" | "movies">("timeline")

const exportSelectionBucket = ref<CuratedFrameExportSelectionState["exportSelectionBucket"]>("none")
const namedActorForExport = ref<string | null>(null)
const selectedFrameIds = ref<string[]>([])
const exportToolbarError = ref("")
const exportBusy = ref(false)
const batchDeleteBusy = ref(false)

function currentExportSelectionState(): CuratedFrameExportSelectionState {
  return {
    selectedFrameIds: selectedFrameIds.value,
    exportSelectionBucket: exportSelectionBucket.value,
    namedActorForExport: namedActorForExport.value,
  }
}

function applyExportSelectionState(next: CuratedFrameExportSelectionState) {
  selectedFrameIds.value = next.selectedFrameIds
  exportSelectionBucket.value = next.exportSelectionBucket
  namedActorForExport.value = next.namedActorForExport
}

function clearExportSelection() {
  applyExportSelectionState(clearCuratedFrameExportSelection())
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
  const result = toggleCuratedFrameExportSelection(currentExportSelectionState(), {
    id,
    mainTab: mainTab.value,
    sectionActor,
    max: batchExportMax,
    anonymousActorLabel: noActorLabel.value,
  })
  if (result.error === "max") {
    exportToolbarError.value = t("curated.exportSelectMax")
    return
  }
  if (result.error === "mixed-actor") {
    exportToolbarError.value = t("curated.exportActorMixed")
    return
  }
  applyExportSelectionState(result.state)
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

/** 取消分组头勾选后，在「按演员」下根据仍选中的 id 推断导出桶 */
function reconcileActorsTabExportBucket() {
  if (mainTab.value !== "actors") {
    return
  }
  const result = reconcileCuratedFrameActorExportSelection({
    selectedFrameIds: selectedFrameIds.value,
    actorGroups: actorGroups.value.map(([label, items]) => [
      label,
      items.map((item) => item.row.id),
    ]),
    currentNamedActorForExport: namedActorForExport.value,
    anonymousActorLabel: noActorLabel.value,
  })
  exportSelectionBucket.value = result.exportSelectionBucket
  namedActorForExport.value = result.namedActorForExport
}

function onActorGroupHeaderSelectionChange(
  actor: string,
  items: readonly RowWithUrl[],
  checked: boolean,
) {
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

function onMovieGroupHeaderSelectionChange(
  movieKey: string,
  items: readonly RowWithUrl[],
  checked: boolean,
) {
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


function preferredCuratedExportFormat(): CuratedExportFormat {
  return libraryService.curatedFrameExportFormat.value ?? "jpg"
}

async function runExport(
  ids: string[],
  actorName: string | undefined,
  format: CuratedExportFormat,
  errorTarget: "toolbar" | "toast",
  mode: CuratedExportMode = "raw",
) {
  exportBusy.value = true
  if (errorTarget === "toolbar") {
    exportToolbarError.value = ""
  }
  try {
    await exportFrames(listWithUrls.value, ids, actorName, format, mode)
  } catch (err) {
    console.error("[curated-frames] export failed", err)
    const msg = mode === "watermarked"
      ? t("curated.exportWatermarkedFailed")
      : t("curated.exportFailed")
    if (errorTarget === "toolbar") {
      exportToolbarError.value = msg
    } else {
      pushAppToast(msg, { variant: "destructive" })
    }
  } finally {
    exportBusy.value = false
  }
}

async function exportSelected() {
  await exportSelectedWithMode(libraryService.curatedFrameExportMode.value)
}

async function exportSelectedWatermarked() {
  await exportSelectedWithMode("watermarked")
}

async function exportSelectedWithMode(mode: CuratedExportMode) {
  if (selectedFrameIds.value.length === 0) {
    return
  }
  await runExport(
    selectedFrameIds.value,
    actorNameForExportRequest(),
    preferredCuratedExportFormat(),
    "toolbar",
    mode,
  )
}

async function exportSingleFromContextMenu() {
  await exportSingleFromContextMenuWithMode(libraryService.curatedFrameExportMode.value)
}

async function exportSingleFromContextMenuWatermarked() {
  await exportSingleFromContextMenuWithMode("watermarked")
}

async function exportSingleFromContextMenuWithMode(mode: CuratedExportMode) {
  const menu = frameContextMenu.value
  if (!menu) {
    return
  }
  closeFrameContextMenu()
  const actorName = resolveSingleFrameActorNameForExport(menu.frame, menu.fromActorSection)
  await runExport([menu.frame.id], actorName, preferredCuratedExportFormat(), "toast", mode)
}

interface RowWithUrl {
  row: CuratedFrameDbRow
  url: string
}

const rawRows = ref<CuratedFrameDbRow[]>([])
const listWithUrls = ref<RowWithUrl[]>([])
const totalRows = ref(0)
let rowsNextCursor: string | undefined
const rowsLoading = ref(false)
const rowsLoadingMore = ref(false)
const rowsLoadError = ref(false)
let rowsQueryVersion = 0
const rowsScrollRoot = ref<HTMLElement | null>(null)
const rowsLoadMoreSentinel = ref<HTMLElement | null>(null)
const curatedTagFacets = ref<CuratedFrameFacetItemDTO[]>([])
let rowsLoadMoreObserver: IntersectionObserver | null = null

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

function currentCuratedTagFilters() {
  return getCuratedFrameTagFilters(route.query)
}

async function reloadFromDb() {
  const version = ++rowsQueryVersion
  rowsLoading.value = true
  rowsLoadingMore.value = false
  rowsLoadError.value = false
  try {
    const page = await listCuratedFramesPage({
      q: currentCuratedQuery(),
      tags: currentCuratedTagFilters(),
      limit: curatedPageLimit,
      offset: 0,
    })
    if (version !== rowsQueryVersion) return
    rawRows.value = page.items
    totalRows.value = page.total
    rowsNextCursor = page.nextCursor
  } catch {
    if (version === rowsQueryVersion) rowsLoadError.value = true
  } finally {
    if (version === rowsQueryVersion) {
      rowsLoading.value = false
      await nextTick()
      if (!rowsLoadError.value) maybeAutoLoadMoreRows()
    }
  }
}

async function loadMoreRows() {
  if (rowsLoadError.value || rowsLoading.value || rowsLoadingMore.value || rawRows.value.length >= totalRows.value) {
    return
  }
  rowsLoadingMore.value = true
  const version = rowsQueryVersion
  try {
    const page = await listCuratedFramesPage({
      q: currentCuratedQuery(),
      tags: currentCuratedTagFilters(),
      limit: curatedPageLimit,
      offset: rawRows.value.length,
      cursor: rowsNextCursor,
      skipTotal: Boolean(rowsNextCursor),
    })
    if (version !== rowsQueryVersion) return
    const known = new Set(rawRows.value.map((row) => row.id))
    rawRows.value = [...rawRows.value, ...page.items.filter((row) => !known.has(row.id))]
    if (page.total >= 0) totalRows.value = page.total
    rowsNextCursor = page.nextCursor
    if (page.items.length === 0) totalRows.value = rawRows.value.length
  } catch {
    if (version === rowsQueryVersion) rowsLoadError.value = true
  } finally {
    if (version === rowsQueryVersion) {
      rowsLoadingMore.value = false
      await nextTick()
      if (!rowsLoadError.value) maybeAutoLoadMoreRows()
    }
  }
}

watch(
  [
    () => curatedFramesRevision.value,
    () => getCuratedFrameSearchQuery(route.query),
    () => serializeCuratedFrameTagFilters(getCuratedFrameTagFilters(route.query)) ?? "",
  ],
  () => {
    void reloadFromDb()
  },
  { immediate: true },
)

watch(
  rawRows,
  () => {
    const previous = new Map(listWithUrls.value.map((item) => [item.row.id, item]))
    listWithUrls.value = rawRows.value.map((row) => {
      const old = previous.get(row.id)
      previous.delete(row.id)
      if (old && old.row.imageBlob === row.imageBlob) return { row, url: old.url }
      if (old?.url.startsWith('blob:')) URL.revokeObjectURL(old.url)
      return { row, url: row.imageBlob ? URL.createObjectURL(row.imageBlob) : curatedFrameThumbnailUrl(row.id) }
    })
    for (const old of previous.values()) if (old.url.startsWith('blob:')) URL.revokeObjectURL(old.url)
  },
  { immediate: true, deep: true },
)

onUnmounted(() => {
  rowsQueryVersion++
  rowsLoadMoreObserver?.disconnect()
  rowsLoadMoreObserver = null
  revokeAllUrls()
})

const isEmpty = computed(() => !rowsLoading.value && listWithUrls.value.length === 0)
const activeTagFilters = computed(() => getCuratedFrameTagFilters(route.query))
const hasActiveFrameFilters = computed(
  () => currentCuratedQuery() !== "" || activeTagFilters.value.length > 0,
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

function isRowsLoadMoreSentinelNearViewport() {
  const target = rowsLoadMoreSentinel.value
  if (!target) {
    return false
  }
  const margin = 640
  const targetRect = target.getBoundingClientRect()
  const root = rowsScrollRoot.value
  if (!root) {
    return targetRect.top <= window.innerHeight + margin && targetRect.bottom >= -margin
  }
  const rootRect = root.getBoundingClientRect()
  return targetRect.top <= rootRect.bottom + margin && targetRect.bottom >= rootRect.top - margin
}

function maybeAutoLoadMoreRows() {
  if (!hasMoreRows.value || rowsLoading.value || rowsLoadingMore.value) {
    return
  }
  if (isRowsLoadMoreSentinelNearViewport()) {
    void loadMoreRows()
  }
}

function observeRowsLoadMoreSentinel() {
  rowsLoadMoreObserver?.disconnect()
  rowsLoadMoreObserver = null

  const target = rowsLoadMoreSentinel.value
  if (!target || typeof IntersectionObserver === "undefined") {
    return
  }

  rowsLoadMoreObserver = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        maybeAutoLoadMoreRows()
      }
    },
    {
      root: rowsScrollRoot.value,
      rootMargin: "640px 0px",
      threshold: 0,
    },
  )
  rowsLoadMoreObserver.observe(target)
}

watch([rowsScrollRoot, rowsLoadMoreSentinel], observeRowsLoadMoreSentinel, { flush: "post" })

const curatedFrameNearDuplicateThresholdSec = 3
const nearDuplicateGroups = computed(() =>
  findCuratedFrameNearDuplicateGroups(rawRows.value, curatedFrameNearDuplicateThresholdSec),
)
const nearDuplicateFrameIds = computed(() => buildCuratedFrameNearDuplicateIndex(nearDuplicateGroups.value))
const nearDuplicateFrameIdList = computed(() => [...nearDuplicateFrameIds.value])

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

const frameDialogRef = ref<InstanceType<typeof CuratedFrameDetailDialog> | null>(null)
const { exportFrames } = useCuratedFrameExport()
const dialogNavigationEntries = computed(() => buildCuratedFrameDialogNavigationEntries({
  mainTab: mainTab.value,
  listItems: listWithUrls.value,
  actorGroups: actorGroups.value,
  movieGroups: movieGroups.value,
}))
type CuratedFrameContextMenuState = { x: number; y: number; frame: CuratedFrameRecord; fromActorSection: string | null }
const frameContextMenu = ref<CuratedFrameContextMenuState | null>(null)

watch([curatedFramesRevision, locale], () => { void reloadTagFacets() }, { immediate: true })
async function reloadTagFacets() {
  try { curatedTagFacets.value = await listCuratedFrameTagFacets(locale.value) }
  catch { curatedTagFacets.value = [] }
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

function openFrameCardDialog(item: RowWithUrl, fromActorSection?: string) {
  closeFrameContextMenu()
  frameDialogRef.value?.open(item, fromActorSection ?? null)
}

function applyFrameTags({ id, tags }: { id: string; tags: string[] }) {
  rawRows.value = rawRows.value.map((row) => row.id === id ? { ...row, tags: [...tags] } : row)
}

function onFrameCardContextMenu(
  event: MouseEvent,
  item: RowWithUrl,
  fromActorSection?: string,
) {
  onFrameContextMenu(event, item, fromActorSection ?? null)
}

function setCuratedFrameTagFilters(tags: readonly string[]) {
  void router.replace({
    name: "curated-frames",
    query: mergeCuratedFramesQuery(route.query, {
      cft: serializeCuratedFrameTagFilters(tags),
    }),
  })
}

function clearCuratedFrameTagFilters() {
  setCuratedFrameTagFilters([])
}

function toggleCuratedFrameTagFilter(tag: string) {
  const normalized = tag.trim()
  if (!normalized) {
    return
  }
  const key = normalized.toLocaleLowerCase()
  const next = activeTagFilters.value.filter((item) => item.toLocaleLowerCase() !== key)
  if (next.length === activeTagFilters.value.length) {
    next.push(normalized)
  }
  setCuratedFrameTagFilters(next)
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

  frameDialogRef.value?.dismissDeleted(deletedIds)
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
  exportSelectedWatermarked,
})
</script>

<template>
  <div
    class="relative isolate mx-auto flex h-full min-h-0 w-full max-w-[min(100%,120rem)] flex-col gap-6 px-3 sm:px-6"
  >
    <Button v-if="rowsLoadError && isLibraryEmpty" variant="outline" class="mx-auto" @click="reloadFromDb">{{ t('curated.retryLoad') }}</Button>
    <CuratedFrameEmptyState
      v-if="isLibraryEmpty && !rowsLoadError"
      variant="library"
      :show-clear-filter="false"
    />

    <Tabs
      v-else-if="!isLibraryEmpty"
      v-model="mainTab"
      class="flex min-h-0 w-full min-w-0 flex-1 flex-col gap-4 overflow-hidden"
    >
      <CuratedFrameLibraryToolbar
        :shown-count="rawRows.length"
        :total-rows="totalRows"
        :batch-mode="batchMode"
        :show-select-visible="batchShowSelectVisible"
        :select-visible-disabled="listWithUrls.length === 0"
        @enter-batch-mode="batchMode = true"
        @select-visible="selectAllVisibleUpTo20"
        @exit-batch-mode="exitBatchMode"
      >
        <template #actions-start>
          <CuratedFrameTagFilterBar
            :facets="curatedTagFacets"
            :selected-tags="activeTagFilters"
            @clear="clearCuratedFrameTagFilters"
            @toggle-tag="toggleCuratedFrameTagFilter"
          />
        </template>
      </CuratedFrameLibraryToolbar>

      <div
        ref="rowsScrollRoot"
        class="min-h-0 flex-1 overflow-y-auto pb-2 pr-3 [scrollbar-gutter:stable] sm:pr-4"
        @scroll.passive="maybeAutoLoadMoreRows"
      >
      <CuratedFrameEmptyState
        v-if="isFilteredEmpty && !rowsLoadError"
        variant="filtered"
        :show-clear-filter="activeTagFilters.length > 0"
        @clear-filter="clearCuratedFrameTagFilters"
      />
      <TabsContent value="timeline" class="mt-0 outline-none">
        <CuratedFrameTimelineTab
          :items="listWithUrls"
          :batch-mode="batchMode"
          :selected-ids="selectedFrameIds"
          :near-duplicate-ids="nearDuplicateFrameIdList"
          @toggle-selection="toggleFrameSelection"
          @contextmenu="onFrameCardContextMenu"
          @open="openFrameCardDialog"
        />
      </TabsContent>

      <TabsContent value="actors" class="mt-0 outline-none">
        <CuratedFrameActorsTab
          :actor-groups="actorGroups"
          :batch-mode="batchMode"
          :selected-ids="selectedFrameIds"
          :near-duplicate-ids="nearDuplicateFrameIdList"
          :select-group-aria-label="t('curated.batchSelectActorGroup')"
          :batch-export-max="batchExportMax"
          @group-selection-change="onActorGroupHeaderSelectionChange"
          @toggle-selection="toggleFrameSelection"
          @contextmenu="onFrameCardContextMenu"
          @open="openFrameCardDialog"
        />
      </TabsContent>

      <TabsContent value="movies" class="mt-0 outline-none">
        <CuratedFrameMoviesTab
          :movie-groups="movieGroups"
          :batch-mode="batchMode"
          :selected-ids="selectedFrameIds"
          :near-duplicate-ids="nearDuplicateFrameIdList"
          :select-group-aria-label="t('curated.batchSelectMovieGroup')"
          :batch-export-max="batchExportMax"
          @group-selection-change="onMovieGroupHeaderSelectionChange"
          @toggle-selection="toggleFrameSelection"
          @contextmenu="onFrameCardContextMenu"
          @open="openFrameCardDialog"
        />
      </TabsContent>
      <Button v-if="rowsLoadError" variant="outline" class="mx-auto" @click="reloadFromDb">
        {{ t('curated.retryLoad') }}
      </Button>
      <div
        v-if="hasMoreRows"
        ref="rowsLoadMoreSentinel"
        class="flex min-h-20 items-center justify-center py-5 text-sm text-muted-foreground"
        aria-live="polite"
      >
        <span v-if="rowsLoadingMore">{{ t("common.loading") }}</span>
      </div>
      </div>
    </Tabs>

    <CuratedFrameDetailDialog
      ref="frameDialogRef"
      :entries="dialogNavigationEntries"
      :near-duplicate-ids="nearDuplicateFrameIdList"
      @deleted="applyDeletedFrameIds"
      @tags-saved="applyFrameTags"
    />

    <CuratedFrameContextMenu
      v-if="frameContextMenu"
      :frame="frameContextMenu.frame"
      :x="frameContextMenu.x"
      :y="frameContextMenu.y"
      :use-web-api="useWebApi"
      @close="closeFrameContextMenu"
      @export="exportSingleFromContextMenu"
      @export-watermarked="exportSingleFromContextMenuWatermarked"
      @delete="openDeleteConfirmFromContextMenu"
    />

    <CuratedFrameDeleteConfirmDialog
      v-model:open="deleteConfirmOpen"
      :label="deleteTargetLabel"
      :error="deleteFrameError"
      :busy="deleteFrameBusy"
      @confirm="executeDeleteCuratedFrame"
    />
  </div>
</template>

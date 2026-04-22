<script setup lang="ts">
import { computed, ref, shallowRef, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { HttpClientError } from "@/api/http-client"
import type { PatchMovieBody } from "@/api/types"
import LibraryBatchActionBar from "@/components/jav-library/LibraryBatchActionBar.vue"
import LibraryPage from "@/components/jav-library/LibraryPage.vue"
import MovieDeleteConfirmDialog from "@/components/jav-library/MovieDeleteConfirmDialog.vue"
import MovieEditDialog from "@/components/jav-library/MovieEditDialog.vue"
import MovieLibraryContextMenu from "@/components/jav-library/MovieLibraryContextMenu.vue"
import { pushAppToast } from "@/composables/use-app-toast"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import { isTerminalTaskStatus, waitForTrackedTaskTerminal } from "@/composables/wait-tracked-task"
import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import {
  buildClearLibraryActorFilterQuery,
  getBrowseSourceMode,
  getLibraryActorExactQuery,
  getLibrarySearchQuery,
  getLibraryStudioExactQuery,
  getLibraryTabQuery,
  getLibraryTagExactQuery,
  getSelectedMovieQuery,
  mergeLibraryQuery,
  resolveLibraryMode,
} from "@/lib/library-query"
import { bumpMovieImageVersion } from "@/lib/image-version"
import { buildLibraryBrowseScrollKey } from "@/lib/library-scroll-key"
import { buildDetailRouteFromBrowse, buildPlayerRouteFromBrowseIntent } from "@/lib/navigation-intent"
import { isMovieRecentlyAdded } from "@/lib/library-stats"
import { movieSearchHaystack } from "@/lib/movie-search"
import { buildUserTagSuggestionPool } from "@/lib/user-tag-suggestions"
import {
  compareByAddedAtDesc,
  compareByRatingDesc,
  compareByReleaseDateDesc,
} from "@/lib/movie-sort"
import { loadMovieDetail } from "@/services/adapters/web/web-library-service"
import { useLibraryService } from "@/services/library-service"

const USE_WEB_API = import.meta.env.VITE_USE_WEB_API === "true"

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()
const scanTaskTracker = useScanTaskTracker()

const metadataRefreshBusy = ref(false)

/** 资料库演员资料卡：与详情页「我的标签」同结构的联想池（全库影片 userTags） */
const actorUserTagSuggestionPool = computed(() =>
  buildUserTagSuggestionPool(libraryService.movies.value, []),
)

const BATCH_SELECT_VISIBLE_MAX = 100

const batchMode = ref(false)
const batchSelectedIds = shallowRef<Set<string>>(new Set())
const batchScrapeBusy = ref(false)
const batchScrapeProgress = ref<{ current: number; total: number } | null>(null)
const batchOperationBusy = ref(false)
const batchScrapeRunning = ref(false)

const batchSelectedIdsList = computed(() => [...batchSelectedIds.value])
const batchSelectedCount = computed(() => batchSelectedIds.value.size)

type LibraryContextMenuState = { movie: Movie; x: number; y: number }
const libraryContextMenu = shallowRef<LibraryContextMenuState | null>(null)

const libraryEditTarget = shallowRef<Movie | null>(null)
const libraryEditOpen = ref(false)
const trashConfirmOpen = ref(false)
const permanentConfirmOpen = ref(false)
const trashPendingId = ref<string | null>(null)
const permanentPendingId = ref<string | null>(null)

function clampMenuPosition(clientX: number, clientY: number) {
  const pad = 8
  const w = 220
  const h = 260
  const maxX = Math.max(pad, window.innerWidth - w - pad)
  const maxY = Math.max(pad, window.innerHeight - h - pad)
  return {
    x: Math.min(Math.max(pad, clientX), maxX),
    y: Math.min(Math.max(pad, clientY), maxY),
  }
}

function onLibraryContextMenu(payload: { event: MouseEvent; movie: Movie }) {
  const { x, y } = clampMenuPosition(payload.event.clientX, payload.event.clientY)
  libraryContextMenu.value = { movie: payload.movie, x, y }
}

function closeLibraryContextMenu() {
  libraryContextMenu.value = null
}


function formatClientError(err: unknown, fallback: string) {
  if (err instanceof HttpClientError) {
    return err.apiError?.message?.trim() || err.message || fallback
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return fallback
}

watch(scanTaskTracker.activeTask, async (task) => {
  if (!USE_WEB_API || !task) return
  if (batchScrapeRunning.value) return
  if (!isTerminalTaskStatus(task.status)) return
  if (task.type !== "scrape.movie") return
  const mid =
    task.metadata && typeof task.metadata.movieId === "string" ? task.metadata.movieId : undefined
  if (!mid) return

  if (task.status === "completed") {
    bumpMovieImageVersion(mid)
    await loadMovieDetail(mid)
  } else if (task.status === "failed" || task.status === "partial_failed") {
    pushAppToast(task.errorMessage?.trim() || t("detail.scrapeFailed"), { variant: "destructive" })
  }
})

function onContextEdit() {
  const m = libraryContextMenu.value?.movie
  if (!m) return
  libraryEditTarget.value = m
  libraryEditOpen.value = true
}

async function handleContextRefreshMetadata() {
  const id = libraryContextMenu.value?.movie.id
  if (!id) return
  metadataRefreshBusy.value = true
  try {
    const task = await libraryService.refreshMovieMetadata(id)
    if (!task?.taskId) {
      pushAppToast(USE_WEB_API ? t("detail.refreshTaskFail") : t("detail.refreshMockMode"), {
        variant: "warning",
      })
      return
    }
    scanTaskTracker.start(task.taskId)
  } catch (err) {
    const message =
      err instanceof HttpClientError
        ? (err.apiError?.message ?? err.message)
        : err instanceof Error
          ? err.message
          : t("detail.refreshFailGeneric")
    pushAppToast(message, { variant: "destructive" })
    console.error("[LibraryView] refresh metadata failed", err)
  } finally {
    metadataRefreshBusy.value = false
  }
}

async function handleContextRevealInFileManager() {
  const id = libraryContextMenu.value?.movie.id
  if (!id) return
  try {
    await libraryService.revealMovieInFileManager(id)
    pushAppToast(t("detail.revealSuccess"), { variant: "success", durationMs: 3200 })
  } catch (err) {
    if (err instanceof Error && err.message === "MOCK_REVEAL_NOT_SUPPORTED") {
      pushAppToast(t("detail.revealMockMode"), { variant: "warning" })
      return
    }
    pushAppToast(formatClientError(err, t("detail.revealFailGeneric")), { variant: "destructive" })
    console.error("[LibraryView] reveal in file manager failed", err)
  }
}

function handleContextMoveToTrash() {
  const id = libraryContextMenu.value?.movie.id
  if (!id) return
  trashPendingId.value = id
  trashConfirmOpen.value = true
}

function handleContextDeletePermanently() {
  const id = libraryContextMenu.value?.movie.id
  if (!id) return
  permanentPendingId.value = id
  permanentConfirmOpen.value = true
}

async function handleContextRestore() {
  const id = libraryContextMenu.value?.movie.id
  if (!id) return
  try {
    await libraryService.restoreMovie(id)
  } catch (err) {
    pushAppToast(formatClientError(err, t("detail.restoreFailGeneric")), { variant: "destructive" })
    console.error("[LibraryView] restore movie failed", err)
  }
}

async function handleDeleteMovieFromLibrary(id: string) {
  try {
    await libraryService.deleteMovie(id)
    await router.replace({
      name: getBrowseSourceMode(route.query),
      query: mergeLibraryQuery(route.query, { selected: undefined }),
    })
  } catch (err) {
    const message =
      err instanceof HttpClientError
        ? (err.apiError?.message ?? err.message)
        : err instanceof Error
          ? err.message
          : t("detail.deleteFailGeneric")
    pushAppToast(message, { variant: "destructive" })
    console.error("[LibraryView] move to trash failed", err)
  }
}

async function handleDeleteMoviePermanentlyFromLibrary(id: string) {
  try {
    await libraryService.deleteMoviePermanently(id)
    await router.replace({
      name: "trash",
      query: mergeLibraryQuery(route.query, { selected: undefined }),
    })
  } catch (err) {
    pushAppToast(formatClientError(err, t("detail.permanentDeleteFailGeneric")), {
      variant: "destructive",
    })
    console.error("[LibraryView] permanent delete failed", err)
  }
}

function onTrashConfirmDialog() {
  const id = trashPendingId.value
  trashPendingId.value = null
  if (id) {
    void handleDeleteMovieFromLibrary(id)
  }
}

function onPermanentConfirmDialog() {
  const id = permanentPendingId.value
  permanentPendingId.value = null
  if (id) {
    void handleDeleteMoviePermanentlyFromLibrary(id)
  }
}

function patchMovieDisplayForLibraryEdit(body: PatchMovieBody, done: (err?: unknown) => void) {
  const id = libraryEditTarget.value?.id
  if (!id) {
    done(new Error("no movie"))
    return
  }
  void (async () => {
    try {
      await libraryService.patchMovie(id, body)
      done()
    } catch (err) {
      console.error("[LibraryView] patch movie display failed", err)
      done(err)
    }
  })()
}

function clearBatchSelection() {
  batchSelectedIds.value = new Set()
}

function enterBatchMode() {
  batchMode.value = true
  closeLibraryContextMenu()
}

function exitBatchMode() {
  batchMode.value = false
  clearBatchSelection()
}

function toggleBatchSelect(movieId: string) {
  const id = movieId.trim()
  if (!id) return
  const next = new Set(batchSelectedIds.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  batchSelectedIds.value = next
}

function selectAllVisibleInBatch() {
  const ids = visibleMovies.value.map((m) => m.id)
  if (ids.length > BATCH_SELECT_VISIBLE_MAX) {
    pushAppToast(t("library.batchSelectVisibleCap", { max: BATCH_SELECT_VISIBLE_MAX }), {
      variant: "warning",
    })
    batchSelectedIds.value = new Set(ids.slice(0, BATCH_SELECT_VISIBLE_MAX))
    return
  }
  batchSelectedIds.value = new Set(ids)
}

watch(
  [
    () => resolveLibraryMode(route),
    () => getLibraryTabQuery(route.query),
    () => getLibraryTagExactQuery(route.query),
    () => getLibrarySearchQuery(route.query),
    () => getLibraryActorExactQuery(route.query),
    () => getLibraryStudioExactQuery(route.query),
  ],
  () => {
    clearBatchSelection()
  },
)

async function runBatchRefreshMetadata() {
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0) return
  batchScrapeRunning.value = true
  batchScrapeBusy.value = true
  let ok = 0
  let fail = 0
  try {
    for (let i = 0; i < ids.length; i++) {
      const id = ids[i]!
      batchScrapeProgress.value = { current: i + 1, total: ids.length }
      try {
        const task = await libraryService.refreshMovieMetadata(id)
        if (!task?.taskId) {
          fail++
          continue
        }
        scanTaskTracker.start(task.taskId)
        const final = await waitForTrackedTaskTerminal(
          () => scanTaskTracker.activeTask.value,
          task.taskId,
        )
        if (final.status === "completed") {
          bumpMovieImageVersion(id)
          await loadMovieDetail(id)
          ok++
        } else {
          fail++
        }
      } catch {
        fail++
      }
    }
  } finally {
    batchScrapeProgress.value = null
    batchScrapeBusy.value = false
    batchScrapeRunning.value = false
  }
  pushAppToast(t("library.batchScrapeSummary", { ok, fail }), {
    variant: fail > 0 && ok === 0 ? "destructive" : ok === 0 ? "warning" : "default",
  })
}

async function runBatchAddFavorite() {
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0) return
  batchOperationBusy.value = true
  let fail = 0
  try {
    for (const id of ids) {
      try {
        await libraryService.toggleFavorite(id, true)
      } catch {
        fail++
      }
    }
  } finally {
    batchOperationBusy.value = false
  }
  pushAppToast(t("library.batchFavoriteSummary", { ok: ids.length - fail, fail }), {
    variant: fail === ids.length ? "destructive" : fail > 0 ? "warning" : "success",
  })
}

async function runBatchRemoveFavorite() {
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0) return
  batchOperationBusy.value = true
  let fail = 0
  try {
    for (const id of ids) {
      try {
        await libraryService.toggleFavorite(id, false)
      } catch {
        fail++
      }
    }
  } finally {
    batchOperationBusy.value = false
  }
  pushAppToast(t("library.batchUnfavoriteSummary", { ok: ids.length - fail, fail }), {
    variant: fail === ids.length ? "destructive" : fail > 0 ? "warning" : "success",
  })
}

async function runBatchAddUserTag(tag: string) {
  const trimmed = tag.trim()
  if (!trimmed) return
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0) return
  batchOperationBusy.value = true
  let fail = 0
  try {
    for (const id of ids) {
      try {
        const movie = libraryService.getMovieById(id)
        const base = movie?.userTags ?? []
        if (base.includes(trimmed)) continue
        await libraryService.patchMovie(id, { userTags: [...base, trimmed] })
      } catch {
        fail++
      }
    }
  } finally {
    batchOperationBusy.value = false
  }
  pushAppToast(t("library.batchTagSummary", { ok: ids.length - fail, fail, tag: trimmed }), {
    variant: fail === ids.length ? "destructive" : fail > 0 ? "warning" : "success",
  })
}

async function runBatchMoveToTrash() {
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0) return
  batchOperationBusy.value = true
  let fail = 0
  try {
    for (const id of ids) {
      try {
        await libraryService.deleteMovie(id)
      } catch {
        fail++
      }
    }
    try {
      await router.replace({
        name: getBrowseSourceMode(route.query),
        query: mergeLibraryQuery(route.query, { selected: undefined }),
      })
    } catch (err) {
      pushAppToast(formatClientError(err, t("detail.deleteFailGeneric")), { variant: "destructive" })
    }
    clearBatchSelection()
    exitBatchMode()
  } finally {
    batchOperationBusy.value = false
  }
  pushAppToast(t("library.batchTrashSummary", { ok: ids.length - fail, fail }), {
    variant: fail === ids.length ? "destructive" : fail > 0 ? "warning" : "default",
  })
}

async function runBatchRestore() {
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0) return
  batchOperationBusy.value = true
  let fail = 0
  try {
    for (const id of ids) {
      try {
        await libraryService.restoreMovie(id)
      } catch {
        fail++
      }
    }
    clearBatchSelection()
  } finally {
    batchOperationBusy.value = false
  }
  pushAppToast(t("library.batchRestoreSummary", { ok: ids.length - fail, fail }), {
    variant: fail === ids.length ? "destructive" : fail > 0 ? "warning" : "success",
  })
}

async function runBatchPermanentDelete() {
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0) return
  batchOperationBusy.value = true
  let fail = 0
  try {
    for (const id of ids) {
      try {
        await libraryService.deleteMoviePermanently(id)
      } catch {
        fail++
      }
    }
    try {
      await router.replace({
        name: "trash",
        query: mergeLibraryQuery(route.query, { selected: undefined }),
      })
    } catch (err) {
      pushAppToast(formatClientError(err, t("detail.permanentDeleteFailGeneric")), {
        variant: "destructive",
      })
    }
    clearBatchSelection()
    exitBatchMode()
  } finally {
    batchOperationBusy.value = false
  }
  pushAppToast(t("library.batchPermanentSummary", { ok: ids.length - fail, fail }), {
    variant: fail === ids.length ? "destructive" : fail > 0 ? "warning" : "default",
  })
}

const libraryMode = computed<LibraryMode>(() => resolveLibraryMode(route))
const libraryScrollKey = computed(() => buildLibraryBrowseScrollKey(route))

watch(
  libraryMode,
  (mode) => {
    if (USE_WEB_API && mode === "trash") {
      void libraryService.ensureTrashLoaded()
    }
  },
  { immediate: true },
)

const libraryMovies = computed(() =>
  libraryMode.value === "trash" ? libraryService.trashedMovies.value : libraryService.movies.value,
)
const searchQuery = computed(() => getLibrarySearchQuery(route.query))
const tagExactQuery = computed(() => getLibraryTagExactQuery(route.query).trim())
const actorExactQuery = computed(() => getLibraryActorExactQuery(route.query).trim())
const studioExactQuery = computed(() => getLibraryStudioExactQuery(route.query).trim())
/**
 * 小写 -> 库内规范演员名（用于 q 与演员名匹配）。
 * 仅在「无 actor= 且顶栏 q 非空」时需要解析；有 `actor=` 或 q 为空时跳过全库扫描，避免大库下卡主线程。
 */
const actorCanonicalByLower = computed(() => {
  if (actorExactQuery.value) {
    return new Map<string, string>()
  }
  const q = searchQuery.value.trim()
  if (!q) {
    return new Map<string, string>()
  }
  const m = new Map<string, string>()
  for (const movie of libraryMovies.value) {
    for (const raw of movie.actors) {
      const name = raw.trim()
      if (!name) continue
      const key = name.toLowerCase()
      if (!m.has(key)) {
        m.set(key, name)
      }
    }
  }
  return m
})

/** 未带 `actor=` 时，若整段 `q` 与某演员名一致（忽略大小写），视为按演员浏览 */
const actorResolvedFromSearch = computed(() => {
  if (actorExactQuery.value) {
    return ""
  }
  const q = searchQuery.value.trim()
  if (!q) {
    return ""
  }
  return actorCanonicalByLower.value.get(q.toLowerCase()) ?? ""
})

/** 演员资料卡标题：URL `actor` 优先，否则为 `q` 解析出的演员名 */
const actorProfileDisplayName = computed(
  () => actorExactQuery.value || actorResolvedFromSearch.value,
)

const activeTab = computed<LibraryTab>(() => getLibraryTabQuery(route.query))
const selectedMovieId = computed(() => getSelectedMovieQuery(route.query))

const queryFilteredMovies = computed(() => {
  const qRaw = searchQuery.value.trim()
  const queryLower = qRaw.toLowerCase()
  const mode = libraryMode.value
  const raw = libraryMovies.value

  let list: Movie[]
  if (mode === "trash") {
    // 回收站不按搜索 / 标签 / 演员 / 厂商筛选，仅展示全部已删除条目
    return [...raw]
  } else if (mode === "favorites") {
    list = raw.filter((movie) => movie.isFavorite)
  } else if (mode === "recent") {
    list = raw
      .filter((movie) => isMovieRecentlyAdded(movie.addedAt))
      .slice()
      .sort((left, right) => right.addedAt.localeCompare(left.addedAt))
  } else if (mode === "tags") {
    list = raw.slice().sort((left, right) => left.tags.join("").localeCompare(right.tags.join("")))
  } else {
    list = [...raw]
  }

  const actorViaQ = actorResolvedFromSearch.value
  const useQAsActorOnly = Boolean(actorViaQ)
  if (queryLower && !useQAsActorOnly) {
    list = list.filter((movie) => movieSearchHaystack(movie).includes(queryLower))
  }

  const tagExact = tagExactQuery.value
  if (tagExact) {
    list = list.filter(
      (movie) => movie.tags.includes(tagExact) || movie.userTags.includes(tagExact),
    )
  }

  const actorFromParam = actorExactQuery.value
  if (actorFromParam) {
    list = list.filter((movie) => movie.actors.includes(actorFromParam))
  } else if (actorViaQ) {
    list = list.filter((movie) => movie.actors.includes(actorViaQ))
  }

  const studioExact = studioExactQuery.value
  if (studioExact) {
    list = list.filter((movie) => movie.studio.trim() === studioExact)
  }

  return list
})

/** 回收站不使用 tab 子筛选（顶栏无「入库时间 / 发售日期 / 评分」） */
const effectiveTab = computed<LibraryTab>(() =>
  libraryMode.value === "trash" ? "all" : activeTab.value,
)

const visibleMovies = computed(() => {
  switch (effectiveTab.value) {
    case "new":
      return queryFilteredMovies.value.slice().sort(compareByReleaseDateDesc)
    case "top-rated":
      return queryFilteredMovies.value.slice().sort(compareByRatingDesc)
    default:
      return queryFilteredMovies.value.slice().sort(compareByAddedAtDesc)
  }
})

const selectedMovie = computed(() => {
  if (selectedMovieId.value) {
    const routeMovie = visibleMovies.value.find((movie) => movie.id === selectedMovieId.value)
    if (routeMovie) {
      return routeMovie
    }
  }

  return visibleMovies.value[0] ?? undefined
})

const replaceQuery = async (
  nextQuery: Partial<Record<"q" | "tab" | "selected" | "from", string | undefined>>,
) => {
  await router.replace({
    name: libraryMode.value,
    query: mergeLibraryQuery(route.query, nextQuery),
  })
}

watch(
  [selectedMovie, () => route.query.selected],
  ([movie]) => {
    const nextSelected = movie?.id
    const normalizedSelected = getSelectedMovieQuery(route.query)

    if (nextSelected === normalizedSelected) {
      return
    }

    void replaceQuery({
      selected: nextSelected,
    })
  },
  { immediate: true, flush: "post" },
)

const updateActiveTab = async (value: LibraryTab) => {
  await replaceQuery({
    tab: value,
    selected: selectedMovie.value?.id ?? visibleMovies.value[0]?.id,
  })
}

const selectMovie = async (movieId: string) => {
  await replaceQuery({
    selected: movieId,
  })
}

const openDetails = async (movieId: string) => {
  await router.push(buildDetailRouteFromBrowse(movieId, route.query, libraryMode.value))
}

const openPlayer = async (movieId?: string) => {
  const nextMovieId = movieId ?? selectedMovie.value?.id

  if (!nextMovieId) {
    return
  }

  await router.push(
    buildPlayerRouteFromBrowseIntent(nextMovieId, route.query, libraryMode.value, "browse"),
  )
}

const toggleFavorite = async (payload: { movieId: string; nextValue: boolean }) => {
  try {
    await libraryService.toggleFavorite(payload.movieId, payload.nextValue)
  } catch (err) {
    console.error("[LibraryView] toggle favorite failed", err)
  }
}

/** Tags 页标签云：与详情 `browseByTag` 一致 */
const browseByExactTag = async (tag: string) => {
  const trimmed = tag.trim()
  if (!trimmed) return
  await router.replace({
    name: libraryMode.value,
    query: mergeLibraryQuery(route.query, {
      tag: trimmed,
      q: undefined,
      actor: undefined,
      studio: undefined,
      tab: "all",
      selected: undefined,
    }),
  })
}

const clearExactTagFilter = async () => {
  await router.replace({
    name: libraryMode.value,
    query: mergeLibraryQuery(route.query, { tag: undefined }),
  })
}

const clearExactActorFilter = async () => {
  await router.replace({
    name: libraryMode.value,
    query: buildClearLibraryActorFilterQuery(route.query, actorProfileDisplayName.value),
  })
}

const clearExactStudioFilter = async () => {
  await router.replace({
    name: libraryMode.value,
    query: mergeLibraryQuery(route.query, { studio: undefined }),
  })
}

/** 回收站不展示 URL 带入的演员 / 厂商筛选条（与无搜索一致）；标签高亮由 LibraryPage 读 route */
const activeActorForPage = computed(() =>
  libraryMode.value === "trash" ? "" : actorProfileDisplayName.value,
)
const activeStudioForPage = computed(() =>
  libraryMode.value === "trash" ? "" : studioExactQuery.value,
)
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
    <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden px-4 py-4 sm:px-5 lg:px-6 lg:py-5 xl:px-7">
      <LibraryPage
        :mode="libraryMode"
        :all-movies="libraryMovies"
        :visible-movies="visibleMovies"
        :selected-movie="selectedMovie"
        :active-tab="effectiveTab"
        :batch-mode="batchMode"
        :batch-selected-ids="batchSelectedIdsList"
        :active-actor-filter="activeActorForPage"
        :active-studio-filter="activeStudioForPage"
        :actor-user-tag-suggestions="actorUserTagSuggestionPool"
        :scroll-preserve-key="libraryScrollKey"
        @update:active-tab="updateActiveTab"
        @select="selectMovie"
        @open-details="openDetails"
        @open-player="openPlayer"
        @toggle-favorite="toggleFavorite"
        @browse-by-exact-tag="browseByExactTag"
        @clear-exact-tag-filter="clearExactTagFilter"
        @clear-exact-actor-filter="clearExactActorFilter"
        @clear-exact-studio-filter="clearExactStudioFilter"
        @context-menu="onLibraryContextMenu"
        @enter-batch-mode="enterBatchMode"
        @exit-batch-mode="exitBatchMode"
        @select-all-visible-in-batch="selectAllVisibleInBatch"
        @toggle-batch-select="toggleBatchSelect"
      />
    </div>

    <LibraryBatchActionBar
      v-if="batchMode"
      :mode="libraryMode"
      :selected-count="batchSelectedCount"
      :use-web-api="USE_WEB_API"
      :scrape-progress="batchScrapeProgress"
      :scrape-busy="batchScrapeBusy"
      :operation-busy="batchOperationBusy"
      :user-tag-suggestions="actorUserTagSuggestionPool"
      @exit="exitBatchMode"
      @clear-selection="clearBatchSelection"
      @select-all-visible="selectAllVisibleInBatch"
      @add-favorite="runBatchAddFavorite"
      @remove-favorite="runBatchRemoveFavorite"
      @add-user-tag="runBatchAddUserTag"
      @refresh-metadata="runBatchRefreshMetadata"
      @move-to-trash="runBatchMoveToTrash"
      @restore="runBatchRestore"
      @permanent-delete="runBatchPermanentDelete"
    />

    <MovieLibraryContextMenu
      v-if="libraryContextMenu"
      :movie="libraryContextMenu.movie"
      :x="libraryContextMenu.x"
      :y="libraryContextMenu.y"
      :metadata-refresh-busy="metadataRefreshBusy"
      @close="closeLibraryContextMenu"
      @edit="onContextEdit"
      @refresh-metadata="handleContextRefreshMetadata"
      @reveal-in-file-manager="handleContextRevealInFileManager"
      @move-to-trash="handleContextMoveToTrash"
      @restore="handleContextRestore"
      @delete-permanently="handleContextDeletePermanently"
    />

    <MovieEditDialog
      v-if="libraryEditTarget"
      v-model:open="libraryEditOpen"
      :key="libraryEditTarget.id"
      :movie="libraryEditTarget"
      :patch-movie-display="patchMovieDisplayForLibraryEdit"
    />

    <MovieDeleteConfirmDialog
      v-model:open="trashConfirmOpen"
      variant="trash"
      @confirm="onTrashConfirmDialog"
    />

    <MovieDeleteConfirmDialog
      v-model:open="permanentConfirmOpen"
      variant="permanent"
      @confirm="onPermanentConfirmDialog"
    />
  </div>
</template>

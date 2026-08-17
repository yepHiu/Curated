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
import { pushAppToast, pushAppToastLoading } from "@/composables/use-app-toast"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import { isTerminalTaskStatus, waitForTrackedTaskTerminal } from "@/composables/wait-tracked-task"
import type { LibraryMode } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import {
  buildClearLibraryActorFilterQuery,
  buildSavedViewFiltersV1,
  getBrowseSourceMode,
  getLibraryActorExactFilters,
  getLibrarySearchQuery,
  getLibrarySortQuery,
  getLibraryStudioExactFilters,
  getLibraryTabQuery,
  getLibraryTagExactFilters,
  mergeLibraryQuery,
  resolveLibraryMode,
  serializeLibraryTagFilters,
} from "@/lib/library-query"
import { applyLibraryBatchToggle } from "@/lib/library-batch-selection"
import { bumpMovieImageVersion } from "@/lib/image-version"
import { buildLibraryBrowseScrollKey } from "@/lib/library-scroll-key"
import { buildDetailRouteFromBrowse, buildPlayerRouteFromBrowseIntent } from "@/lib/navigation-intent"
import { isMovieRecentlyAdded } from "@/lib/library-stats"
import { movieSearchHaystack } from "@/lib/movie-search"
import { filterMoviesBySavedView } from "@/lib/library-saved-view-filter"
import { getProgress, playbackProgressRevision } from "@/lib/playback-progress-storage"
import { hasPlayedMovie, playedMovieCount } from "@/lib/played-movies-storage"
import { buildUserTagSuggestionPool } from "@/lib/user-tag-suggestions"
import { compareMoviesByLibrarySort } from "@/lib/movie-sort"
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
const libraryLoadError = computed(() => libraryService.loadError.value)

const BATCH_SELECT_VISIBLE_MAX = 100

const batchMode = ref(false)
const batchSelectedIds = shallowRef<Set<string>>(new Set())
const batchAnchorId = ref<string | null>(null)
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
    await libraryService.loadMovieDetail(mid)
  }
})

function onContextEdit() {
  const m = libraryContextMenu.value?.movie
  if (!m) return
  libraryEditTarget.value = m
  libraryEditOpen.value = true
}

async function handleContextRefreshMetadata() {
  const movie = libraryContextMenu.value?.movie
  if (!movie) return
  metadataRefreshBusy.value = true
  const code = movie.code?.trim() || movie.id
  const loadingToastId = pushAppToastLoading(t("toasts.manualMovieScrapeStarted", { code }))
  try {
    const task = await libraryService.refreshMovieMetadata(movie.id)
    if (!task?.taskId) {
      pushAppToast(USE_WEB_API ? t("detail.refreshTaskFail") : t("detail.refreshMockMode"), {
        variant: "warning",
        id: loadingToastId,
      })
      return
    }
    scanTaskTracker.start(task.taskId, {
      notifyMovieScrape: true,
      hideProgressDock: true,
      loadingToastId,
      scrapeCode: code,
    })
  } catch (err) {
    pushAppToast(t("toasts.manualMovieScrapeStartFailed", { code }), {
      variant: "destructive",
      id: loadingToastId,
    })
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
      const message = formatClientError(err, t("detailPanel.movieEditSaveFailed"))
      pushAppToast(message, { variant: "destructive" })
      done(new Error(message))
    }
  })()
}

function clearBatchSelection() {
  batchSelectedIds.value = new Set()
  batchAnchorId.value = null
}

function enterBatchMode() {
  batchMode.value = true
  closeLibraryContextMenu()
}

function exitBatchMode() {
  batchMode.value = false
  clearBatchSelection()
}

function toggleBatchSelect(payload: { movieId: string; shiftKey?: boolean }) {
  const result = applyLibraryBatchToggle({
    selectedIds: batchSelectedIds.value,
    orderedIds: visibleMovies.value.map((movie) => movie.id),
    movieId: payload.movieId,
    shiftKey: payload.shiftKey === true,
    anchorId: batchAnchorId.value,
    maxCount: BATCH_SELECT_VISIBLE_MAX,
  })
  batchSelectedIds.value = result.selectedIds
  batchAnchorId.value = result.anchorId
  if (result.truncated) {
    pushAppToast(t("library.batchSelectVisibleCap", { max: BATCH_SELECT_VISIBLE_MAX }), {
      variant: "warning",
    })
  }
}

function selectAllVisibleInBatch() {
  const ids = visibleMovies.value.map((m) => m.id)
  if (ids.length > BATCH_SELECT_VISIBLE_MAX) {
    pushAppToast(t("library.batchSelectVisibleCap", { max: BATCH_SELECT_VISIBLE_MAX }), {
      variant: "warning",
    })
    batchSelectedIds.value = new Set(ids.slice(0, BATCH_SELECT_VISIBLE_MAX))
    batchAnchorId.value = ids[0] ?? null
    return
  }
  batchSelectedIds.value = new Set(ids)
  batchAnchorId.value = ids[0] ?? null
}

watch(
  [
    () => resolveLibraryMode(route),
    () => getLibraryTabQuery(route.query),
    () => getLibrarySortQuery(route.query),
    () => serializeLibraryTagFilters(getLibraryTagExactFilters(route.query)) ?? "",
    () => getLibrarySearchQuery(route.query),
    () => serializeLibraryTagFilters(getLibraryActorExactFilters(route.query)) ?? "",
    () => serializeLibraryTagFilters(getLibraryStudioExactFilters(route.query)) ?? "",
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
        scanTaskTracker.start(task.taskId, { hideProgressDock: true })
        const final = await waitForTrackedTaskTerminal(
          () => scanTaskTracker.activeTask.value,
          task.taskId,
        )
        if (final.status === "completed") {
          bumpMovieImageVersion(id)
          await libraryService.loadMovieDetail(id)
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
const actorExactFilters = computed(() => getLibraryActorExactFilters(route.query))
const actorExactQuery = computed(() =>
  actorExactFilters.value.length === 1 ? actorExactFilters.value[0]!.trim() : "",
)
const canonicalActorExactQuery = ref("")
let actorResolveSequence = 0

watch(
  actorExactQuery,
  async (rawName) => {
    const sequence = ++actorResolveSequence
    canonicalActorExactQuery.value = rawName
    if (!rawName) return
    try {
      const profile = await libraryService.getActorProfile(rawName)
      if (sequence !== actorResolveSequence) return
      canonicalActorExactQuery.value = profile.name
      if (profile.name !== rawName) {
        await router.replace({
          name: libraryMode.value,
          query: mergeLibraryQuery(route.query, { actor: profile.name }),
        })
      }
    } catch {
      // Keep the original query so ordinary not-found handling and empty states remain stable.
    }
  },
  { immediate: true },
)
const effectiveActorExactQuery = computed(
  () => canonicalActorExactQuery.value || actorExactQuery.value,
)
const studioExactFilters = computed(() => getLibraryStudioExactFilters(route.query))
const savedViewFilters = computed(() => {
  const filters = buildSavedViewFiltersV1(libraryMode.value, route.query)
  if (actorExactFilters.value.length === 1 && effectiveActorExactQuery.value) {
    return { ...filters, actor: effectiveActorExactQuery.value }
  }
  return filters
})
/**
 * 小写 -> 库内规范演员名（用于 q 与演员名匹配）。
 * 仅在「无 actor= 且顶栏 q 非空」时需要解析；有 `actor=` 或 q 为空时跳过全库扫描，避免大库下卡主线程。
 */
const actorCanonicalByLower = computed(() => {
  if (actorExactFilters.value.length > 0) {
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
  if (actorExactFilters.value.length > 0) {
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
  () => effectiveActorExactQuery.value || actorResolvedFromSearch.value,
)

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
  } else {
    list = [...raw]
  }

  const actorViaQ = actorResolvedFromSearch.value
  const useQAsActorOnly = Boolean(actorViaQ)
  if (queryLower && !useQAsActorOnly) {
    list = list.filter((movie) => movieSearchHaystack(movie).includes(queryLower))
  }

  if (actorViaQ && actorExactFilters.value.length === 0) {
    list = list.filter((movie) => movie.actors.includes(actorViaQ))
  }

  // These revisions make the Saved View result reactive after playback writes or hydrate.
  void playbackProgressRevision.value
  void playedMovieCount.value
  return filterMoviesBySavedView(list, savedViewFilters.value, {
    hasPlayedMovie,
    getProgress,
  })
})

const visibleMovies = computed(() => {
  const sort = getLibrarySortQuery(route.query)
  return queryFilteredMovies.value
    .slice()
    .sort((left, right) => compareMoviesByLibrarySort(left, right, sort))
})

const openPlayer = async (movieId?: string) => {
  const nextMovieId = movieId?.trim()

  if (!nextMovieId) {
    return
  }

  await router.push(
    buildPlayerRouteFromBrowseIntent(nextMovieId, route.query, libraryMode.value, "browse"),
  )
}

const openDetails = async (movieId: string) => {
  await router.push(buildDetailRouteFromBrowse(movieId, route.query, libraryMode.value))
}


const toggleFavorite = async (payload: { movieId: string; nextValue: boolean }) => {
  try {
    await libraryService.toggleFavorite(payload.movieId, payload.nextValue)
  } catch (err) {
    pushAppToast(formatClientError(err, t("library.favoriteToggleFailed")), {
      variant: "destructive",
    })
    console.error("[LibraryView] toggle favorite failed", err)
  }
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

/** 回收站不展示 URL 带入的演员 / 厂商筛选条（与无搜索一致） */
const activeActorForPage = computed(() =>
  libraryMode.value === "trash" ? "" : actorProfileDisplayName.value,
)
const activeStudioForPage = computed(() =>
  libraryMode.value === "trash" ? "" : studioExactFilters.value.length === 1 ? studioExactFilters.value[0]! : "",
)
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
    <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden px-[var(--app-page-px)] py-[var(--app-page-py)] sm:px-[var(--app-page-px-sm)] lg:px-[var(--app-page-px-lg)] lg:py-[var(--app-page-py-lg)] xl:px-[var(--app-page-px-xl)]">
      <p
        v-if="libraryLoadError"
        data-library-load-error
        role="alert"
        class="mb-4 rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      >
        {{ libraryLoadError }}
      </p>
      <LibraryPage
        :mode="libraryMode"
        :visible-movies="visibleMovies"
        :batch-mode="batchMode"
        :batch-selected-ids="batchSelectedIdsList"
        :active-actor-filter="activeActorForPage"
        :active-studio-filter="activeStudioForPage"
        :actor-user-tag-suggestions="actorUserTagSuggestionPool"
        :scroll-preserve-key="libraryScrollKey"
        @open-details="openDetails"
        @open-player="openPlayer"
        @toggle-favorite="toggleFavorite"
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

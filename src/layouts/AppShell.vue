<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { onClickOutside, onKeyStroke, useMediaQuery, watchDebounced } from "@vueuse/core"
import { LayoutDashboard, Menu, Moon, PanelLeftClose, PanelLeftOpen, Search, Sun, X } from "lucide-vue-next"
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router"
import AppSidebar from "@/components/jav-library/AppSidebar.vue"
import DevEnvironmentBadge from "@/components/dev/DevEnvironmentBadge.vue"
import DevPerformanceBar from "@/components/dev/DevPerformanceBar.vue"
import { Toaster } from "@/components/ui/sonner"
import ScanProgressDock from "@/components/jav-library/ScanProgressDock.vue"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import {
  buildLibrarySearchSuggestions,
  librarySearchSuggestionsHasAny,
} from "@/lib/library-search-suggestions"
import { resolveNavigationBackLink } from "@/lib/navigation-intent"
import {
  ACTORS_SEARCH_QUERY_KEY,
  getActorsSearchQuery,
  mergeActorsQuery,
} from "@/lib/actors-route-query"
import {
  getCuratedFrameSearchQuery,
  getLibraryActorExactQuery,
  getLibrarySearchQuery,
  getLibraryStudioExactQuery,
  getLibraryTagExactQuery,
  getSelectedMovieQuery,
  isLibraryBrowseRoute,
  mergeCuratedFramesQuery,
  mergeLibraryQuery,
  resolveLibraryMode,
} from "@/lib/library-query"
import { useLibraryWatchToasts } from "@/composables/use-library-watch-toasts"
import { useTheme } from "@/composables/use-theme"
import { devPerformanceBarHidden, setDevPerformanceBarHidden } from "@/lib/dev-performance/visibility"
import { useLibraryService } from "@/services/library-service"

useLibraryWatchToasts()

const { resolvedMode, setThemePreference } = useTheme()

function onShellAppearanceSwitch(useDark: boolean) {
  setThemePreference(useDark ? "dark" : "light")
}
const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const isDev = import.meta.env.DEV

/** 与 Tailwind lg 对齐：以下视为「窄屏」，侧栏自动收起，用抽屉进入导航 */
const isLgUp = useMediaQuery("(min-width: 1024px)")
/** 桌面宽屏下用户可切换窄条 / 完整侧栏；切到窄屏时会强制展开（见 watch） */
const desktopSidebarCollapsed = ref(false)
const mobileSidebarOpen = ref(false)

watch(isLgUp, (wide) => {
  mobileSidebarOpen.value = false
  if (wide) {
    desktopSidebarCollapsed.value = false
  }
})

watch(
  () => route.fullPath,
  () => {
    if (!isLgUp.value) {
      mobileSidebarOpen.value = false
    }
  },
)

watch(
  [mobileSidebarOpen, isLgUp],
  () => {
    const lock = mobileSidebarOpen.value && !isLgUp.value
    document.body.style.overflow = lock ? "hidden" : ""
  },
  { flush: "sync" },
)

onBeforeUnmount(() => {
  document.body.style.overflow = ""
})

const shellGridClass = computed(() => {
  const base =
    "grid h-full min-h-0 grid-cols-1 lg:transition-[grid-template-columns] lg:duration-300 lg:ease-in-out motion-reduce:lg:transition-none"
  if (!isLgUp.value) {
    return base
  }
  return `${base} ${desktopSidebarCollapsed.value ? "lg:grid-cols-[4.75rem_minmax(0,1fr)]" : "lg:grid-cols-[304px_minmax(0,1fr)]"}`
})

const currentMovie = computed(() => {
  const routeMovieId = typeof route.params.id === "string" ? route.params.id : undefined
  const selectedMovieId = getSelectedMovieQuery(route.query)
  const candidateId = routeMovieId ?? selectedMovieId

  return candidateId ? libraryService.getMovieById(candidateId) : undefined
})

const currentMovieId = computed(() => currentMovie.value?.id)
const isLibraryRoute = computed(() => isLibraryBrowseRoute(route))
/** 回收站不显示资料库顶栏搜索 */
const showLibraryBrowseSearch = computed(
  () => isLibraryRoute.value && resolveLibraryMode(route) !== "trash",
)
const isActorsRoute = computed(() => route.name === "actors")
const isPrimaryBrowseRoute = computed(() => isLibraryRoute.value || isActorsRoute.value)
const isCuratedFramesRoute = computed(() => route.name === "curated-frames")
const useFlushWorkspaceFrame = computed(() =>
  ["library", "favorites", "tags", "trash", "history", "curated-frames"].includes(
    String(route.name ?? ""),
  ),
)

const showHeaderBack = computed(() => !isPrimaryBrowseRoute.value)
const routerViewFrameClass = computed(() =>
  useFlushWorkspaceFrame.value
    ? "flex h-full min-h-0 min-w-0 flex-col overflow-hidden"
    : "flex h-full min-h-0 min-w-0 flex-col overflow-hidden px-4 py-4 sm:px-5 lg:px-6 lg:py-5 xl:px-7",
)

const headerBackIntent = computed(() => resolveNavigationBackLink(route, currentMovieId.value))
const headerBackTarget = computed(() => headerBackIntent.value.to)
const headerBackLabel = computed(() => {
  void locale.value
  return t(headerBackIntent.value.labelKey)
})

/** 输入即时更新；同步到 URL 防抖，避免每键一次 router.replace 导致整库重算/重绘 */
const searchDraft = ref("")

const librarySearchRootRef = ref<HTMLElement | null>(null)
const librarySuggestionsOpen = ref(false)
const debouncedSuggestNeedle = ref("")

watchDebounced(
  searchDraft,
  (v) => {
    debouncedSuggestNeedle.value = v
  },
  { debounce: 150 },
)

const librarySuggestGroups = computed(() =>
  buildLibrarySearchSuggestions(debouncedSuggestNeedle.value, libraryService.movies.value),
)

/** 下拉列表渲染行：分组标题 + 可选条目（含扁平下标供键盘高亮） */
type LibrarySuggestRow =
  | { rowType: "header"; title: "actors" | "tags" | "codes" }
  | {
      rowType: "item"
      kind: "actor" | "tag" | "code"
      flatIndex: number
      label: string
      canonical?: string
      code?: string
    }

const librarySuggestRows = computed((): LibrarySuggestRow[] => {
  const g = librarySuggestGroups.value
  if (!librarySearchSuggestionsHasAny(g)) {
    return []
  }
  const rows: LibrarySuggestRow[] = []
  let flatIndex = 0
  if (g.actors.length) {
    rows.push({ rowType: "header", title: "actors" })
    for (const s of g.actors) {
      rows.push({
        rowType: "item",
        kind: "actor",
        flatIndex: flatIndex++,
        label: s.canonical,
        canonical: s.canonical,
      })
    }
  }
  if (g.tags.length) {
    rows.push({ rowType: "header", title: "tags" })
    for (const s of g.tags) {
      rows.push({
        rowType: "item",
        kind: "tag",
        flatIndex: flatIndex++,
        label: s.canonical,
        canonical: s.canonical,
      })
    }
  }
  if (g.codes.length) {
    rows.push({ rowType: "header", title: "codes" })
    for (const s of g.codes) {
      rows.push({
        rowType: "item",
        kind: "code",
        flatIndex: flatIndex++,
        label: s.code,
        code: s.code,
      })
    }
  }
  return rows
})

const librarySuggestItemCount = computed(
  () => librarySuggestRows.value.filter((r) => r.rowType === "item").length,
)

const librarySuggestHighlightIndex = ref(-1)

watch(debouncedSuggestNeedle, () => {
  librarySuggestHighlightIndex.value = -1
})

watch(librarySuggestionsOpen, (open) => {
  if (!open) {
    librarySuggestHighlightIndex.value = -1
  }
})

watch(librarySuggestItemCount, (n) => {
  if (n === 0) {
    librarySuggestHighlightIndex.value = -1
    return
  }
  if (librarySuggestHighlightIndex.value >= n) {
    librarySuggestHighlightIndex.value = n - 1
  }
})

watch(librarySuggestHighlightIndex, async (idx) => {
  if (idx < 0 || !librarySearchRootRef.value) {
    return
  }
  await nextTick()
  librarySearchRootRef.value
    .querySelector<HTMLElement>(`[data-suggest-idx="${idx}"]`)
    ?.scrollIntoView({ block: "nearest" })
})

function suggestTitleKey(section: "actors" | "tags" | "codes") {
  if (section === "actors") {
    return "shell.searchSuggestActors"
  }
  if (section === "tags") {
    return "shell.searchSuggestTags"
  }
  return "shell.searchSuggestCodes"
}

watch(searchDraft, (v) => {
  if (!v.trim()) {
    librarySuggestionsOpen.value = false
  }
})

watch(isLibraryRoute, (lib) => {
  if (!lib) {
    librarySuggestionsOpen.value = false
  }
})

onClickOutside(librarySearchRootRef, () => {
  librarySuggestionsOpen.value = false
})

function onLibrarySearchFocus() {
  debouncedSuggestNeedle.value = searchDraft.value
  if (searchDraft.value.trim()) {
    librarySuggestionsOpen.value = true
  }
}

function onLibrarySearchInput() {
  if (searchDraft.value.trim()) {
    librarySuggestionsOpen.value = true
  }
}

function applyLibrarySuggestActor(canonical: string) {
  librarySuggestionsOpen.value = false
  void router.replace({
    name: route.name ?? "library",
    query: mergeLibraryQuery(route.query, {
      actor: canonical,
      q: undefined,
      tag: undefined,
      studio: undefined,
    }),
  })
}

function applyLibrarySuggestTag(canonical: string) {
  librarySuggestionsOpen.value = false
  void router.replace({
    name: route.name ?? "library",
    query: mergeLibraryQuery(route.query, {
      tag: canonical,
      q: undefined,
      actor: undefined,
      studio: undefined,
    }),
  })
}

function applyLibrarySuggestCode(code: string) {
  librarySuggestionsOpen.value = false
  void router.replace({
    name: route.name ?? "library",
    query: mergeLibraryQuery(route.query, {
      q: code,
      tag: undefined,
      actor: undefined,
      studio: undefined,
    }),
  })
}

function applyLibrarySuggestRow(row: Extract<LibrarySuggestRow, { rowType: "item" }>) {
  if (row.kind === "actor" && row.canonical) {
    applyLibrarySuggestActor(row.canonical)
  } else if (row.kind === "tag" && row.canonical) {
    applyLibrarySuggestTag(row.canonical)
  } else if (row.kind === "code" && row.code) {
    applyLibrarySuggestCode(row.code)
  }
}

function onLibrarySearchKeydown(e: KeyboardEvent) {
  if (!librarySuggestionsOpen.value || !searchDraft.value.trim()) {
    return
  }
  const n = librarySuggestItemCount.value
  if (n === 0) {
    return
  }

  if (e.key === "ArrowDown") {
    e.preventDefault()
    if (librarySuggestHighlightIndex.value < 0) {
      librarySuggestHighlightIndex.value = 0
    } else {
      librarySuggestHighlightIndex.value = Math.min(librarySuggestHighlightIndex.value + 1, n - 1)
    }
    return
  }

  if (e.key === "ArrowUp") {
    e.preventDefault()
    if (librarySuggestHighlightIndex.value <= 0) {
      librarySuggestHighlightIndex.value = -1
    } else {
      librarySuggestHighlightIndex.value -= 1
    }
    return
  }

  if (e.key === "Enter") {
    const idx = librarySuggestHighlightIndex.value
    if (idx < 0) {
      return
    }
    const row = librarySuggestRows.value.find(
      (r): r is Extract<LibrarySuggestRow, { rowType: "item" }> =>
        r.rowType === "item" && r.flatIndex === idx,
    )
    if (row) {
      e.preventDefault()
      applyLibrarySuggestRow(row)
    }
  }
}

onKeyStroke("Escape", (e) => {
  if (librarySuggestionsOpen.value && showLibraryBrowseSearch.value) {
    e.preventDefault()
    librarySuggestionsOpen.value = false
    return
  }
  if (mobileSidebarOpen.value && !isLgUp.value) {
    e.preventDefault()
    mobileSidebarOpen.value = false
  }
})

watch(
  [
    () => route.name,
    () => route.query.q,
    () => route.query.tag,
    () => route.query.actor,
    () => route.query.studio,
  ],
  () => {
    if (!isLibraryRoute.value) {
      return
    }
    if (!showLibraryBrowseSearch.value) {
      searchDraft.value = ""
      return
    }
    const qPart = getLibrarySearchQuery(route.query)
    const tagPart = getLibraryTagExactQuery(route.query).trim()
    const actorPart = getLibraryActorExactQuery(route.query).trim()
    const studioPart = getLibraryStudioExactQuery(route.query).trim()
    const next = qPart || tagPart || actorPart || studioPart
    if (next !== searchDraft.value) {
      searchDraft.value = next
    }
  },
  { immediate: true },
)

watchDebounced(
  searchDraft,
  (value) => {
    if (!showLibraryBrowseSearch.value) {
      return
    }
    const normalized = value.trim()
    const currentQ = getLibrarySearchQuery(route.query).trim()
    const currentTag = getLibraryTagExactQuery(route.query).trim()
    const currentActor = getLibraryActorExactQuery(route.query).trim()
    const currentStudio = getLibraryStudioExactQuery(route.query).trim()
    const syncedToTextSearch =
      normalized === currentQ && (currentQ !== "" || (!currentTag && !currentActor && !currentStudio))
    const syncedToTagFilter =
      currentTag !== "" && normalized === currentTag && currentQ === "" && !currentActor && !currentStudio
    const syncedToActorFilter =
      currentActor !== "" &&
      normalized === currentActor &&
      currentQ === "" &&
      !currentTag &&
      !currentStudio
    const syncedToStudioFilter =
      currentStudio !== "" &&
      normalized === currentStudio &&
      currentQ === "" &&
      !currentTag &&
      !currentActor
    if (
      syncedToTextSearch ||
      syncedToTagFilter ||
      syncedToActorFilter ||
      syncedToStudioFilter
    ) {
      return
    }
    void router.replace({
      name: route.name ?? "library",
      query: mergeLibraryQuery(route.query, {
        q: normalized || undefined,
        tag: undefined,
        actor: undefined,
        studio: undefined,
      }),
    })
  },
  { debounce: 280 },
)

function clearLibrarySearch() {
  searchDraft.value = ""
  librarySuggestionsOpen.value = false
  if (!showLibraryBrowseSearch.value) {
    return
  }
  void router.replace({
    name: route.name ?? "library",
    query: mergeLibraryQuery(route.query, {
      q: undefined,
      tag: undefined,
      actor: undefined,
      studio: undefined,
    }),
  })
}

/** 萃取帧库专用顶栏搜索（`cfq`） */
const searchDraftFrames = ref("")

watch(
  [() => route.name, () => route.query.cfq],
  () => {
    if (!isCuratedFramesRoute.value) {
      return
    }
    const part = getCuratedFrameSearchQuery(route.query)
    if (part !== searchDraftFrames.value) {
      searchDraftFrames.value = part
    }
  },
  { immediate: true },
)

watchDebounced(
  searchDraftFrames,
  (value) => {
    if (!isCuratedFramesRoute.value) {
      return
    }
    const normalized = value.trim()
    const current = getCuratedFrameSearchQuery(route.query).trim()
    if (normalized === current) {
      return
    }
    void router.replace({
      name: "curated-frames",
      query: mergeCuratedFramesQuery(route.query, {
        cfq: normalized || undefined,
      }),
    })
  },
  { debounce: 280 },
)

function clearCuratedFramesSearch() {
  searchDraftFrames.value = ""
  if (!isCuratedFramesRoute.value) {
    return
  }
  void router.replace({
    name: "curated-frames",
    query: mergeCuratedFramesQuery(route.query, { cfq: undefined }),
  })
}

/** 演员库顶栏搜索（`actorsQ`） */
const searchDraftActors = ref("")

watch(
  [() => route.name, () => route.query[ACTORS_SEARCH_QUERY_KEY]],
  () => {
    if (!isActorsRoute.value) {
      return
    }
    const part = getActorsSearchQuery(route.query)
    if (part !== searchDraftActors.value) {
      searchDraftActors.value = part
    }
  },
  { immediate: true },
)

watchDebounced(
  searchDraftActors,
  (value) => {
    if (!isActorsRoute.value) {
      return
    }
    const normalized = value.trim()
    const current = getActorsSearchQuery(route.query).trim()
    if (normalized === current) {
      return
    }
    void router.replace({
      name: "actors",
      query: mergeActorsQuery(route.query, {
        [ACTORS_SEARCH_QUERY_KEY]: normalized || undefined,
      }),
    })
  },
  { debounce: 280 },
)

function clearActorsSearch() {
  searchDraftActors.value = ""
  if (!isActorsRoute.value) {
    return
  }
  void router.replace({
    name: "actors",
    query: mergeActorsQuery(route.query, {
      [ACTORS_SEARCH_QUERY_KEY]: undefined,
    }),
  })
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden bg-background text-foreground">
    <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <div
        data-shell-layout="split"
        :class="shellGridClass"
      >
        <div
          v-if="isLgUp"
          class="min-h-0 min-w-0 overflow-hidden border-r border-sidebar-border/80 bg-sidebar/95"
        >
          <AppSidebar
            class="min-h-0"
            :compact="desktopSidebarCollapsed"
          />
        </div>

        <section
          class="flex min-h-0 min-w-0 flex-col overflow-hidden bg-background/95"
        >
          <!-- min-h 与中间栏 h-10 搜索框 + 上下 py-4 对齐，避免仅「返回」时顶栏变矮（如观看历史） -->
          <div
            class="flex min-h-[4.5rem] flex-wrap items-center justify-between gap-3 border-b border-border/60 px-4 py-3.5 sm:px-5 lg:px-6 lg:py-4"
          >
            <div class="flex flex-wrap items-center gap-2">
              <Button
                v-if="isLgUp"
                type="button"
                variant="ghost"
                size="icon"
                class="hidden shrink-0 rounded-2xl text-muted-foreground hover:text-foreground lg:inline-flex"
                data-sidebar-toggle="desktop"
                :title="desktopSidebarCollapsed ? t('nav.expandSidebar') : t('nav.collapseSidebar')"
                :aria-label="desktopSidebarCollapsed ? t('nav.expandSidebar') : t('nav.collapseSidebar')"
                @click="desktopSidebarCollapsed = !desktopSidebarCollapsed"
              >
                <PanelLeftOpen v-if="desktopSidebarCollapsed" class="size-5" />
                <PanelLeftClose v-else class="size-5" />
              </Button>
              <Button
                v-if="!isLgUp"
                type="button"
                variant="ghost"
                size="icon"
                class="shrink-0 rounded-2xl text-muted-foreground hover:text-foreground lg:hidden"
                :aria-label="t('shell.openMenu')"
                @click="mobileSidebarOpen = true"
              >
                <Menu class="size-5" />
              </Button>
              <Button v-if="showHeaderBack" as-child variant="secondary" class="rounded-2xl">
                <RouterLink :to="headerBackTarget">
                  <LayoutDashboard data-icon="inline-start" />
                  {{ headerBackLabel }}
                </RouterLink>
              </Button>
            </div>

            <div class="flex flex-1 justify-center px-0 lg:px-3">
              <div
                v-if="showLibraryBrowseSearch"
                ref="librarySearchRootRef"
                class="relative w-full max-w-xl"
              >
                <Search class="pointer-events-none absolute top-1/2 left-3 z-[1] -translate-y-1/2 text-muted-foreground" />
                <Input
                  v-model="searchDraft"
                  class="h-10 rounded-2xl border-border/70 bg-background/70 pl-10 transition-[border-color,background-color,box-shadow] hover:border-primary/60 hover:bg-background/85 hover:ring-1 hover:ring-primary/30"
                  :class="searchDraft.trim() ? 'pr-10' : ''"
                  :placeholder="t('shell.searchLibraryPlaceholder')"
                  autocomplete="off"
                  role="combobox"
                  :aria-expanded="librarySuggestionsOpen && !!searchDraft.trim()"
                  :aria-activedescendant="
                    librarySuggestHighlightIndex >= 0
                      ? `library-search-suggest-${librarySuggestHighlightIndex}`
                      : undefined
                  "
                  aria-autocomplete="list"
                  aria-controls="library-search-suggest-list"
                  @focus="onLibrarySearchFocus"
                  @input="onLibrarySearchInput"
                  @keydown="onLibrarySearchKeydown"
                />
                <Button
                  v-if="searchDraft.trim()"
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="absolute top-1/2 right-1.5 z-[1] size-8 -translate-y-1/2 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
                  :aria-label="t('shell.clearSearch')"
                  @click="clearLibrarySearch"
                >
                  <X class="size-4" />
                </Button>
                <div
                  v-show="librarySuggestionsOpen && searchDraft.trim()"
                  id="library-search-suggest-list"
                  class="absolute top-full right-0 left-0 z-50 mt-1 overflow-hidden rounded-2xl border border-border/80 bg-popover text-popover-foreground shadow-lg shadow-black/15"
                  role="listbox"
                  :aria-label="t('shell.searchSuggestPanel')"
                >
                  <ScrollArea class="max-h-72">
                    <div class="p-1.5">
                      <template
                        v-if="librarySearchSuggestionsHasAny(librarySuggestGroups)"
                      >
                        <template
                          v-for="(row, ri) in librarySuggestRows"
                          :key="
                            row.rowType === 'header'
                              ? `h-${row.title}-${ri}`
                              : `i-${row.flatIndex}`
                          "
                        >
                          <Separator
                            v-if="row.rowType === 'header' && ri > 0"
                            class="my-1.5"
                          />
                          <p
                            v-if="row.rowType === 'header'"
                            class="px-2 pb-1 text-xs font-medium text-muted-foreground"
                          >
                            {{ t(suggestTitleKey(row.title)) }}
                          </p>
                          <button
                            v-else
                            :id="`library-search-suggest-${row.flatIndex}`"
                            type="button"
                            :data-suggest-idx="row.flatIndex"
                            class="flex w-full rounded-xl px-2 py-2 text-left text-sm transition-colors hover:bg-muted"
                            :class="
                              librarySuggestHighlightIndex === row.flatIndex ? 'bg-muted' : ''
                            "
                            role="option"
                            :aria-selected="librarySuggestHighlightIndex === row.flatIndex"
                            @mousedown.prevent
                            @click="applyLibrarySuggestRow(row)"
                          >
                            {{ row.label }}
                          </button>
                        </template>
                      </template>
                      <p
                        v-else
                        class="px-2 py-3 text-sm text-muted-foreground"
                      >
                        {{ t("shell.searchSuggestEmpty") }}
                      </p>
                    </div>
                  </ScrollArea>
                </div>
              </div>
              <div
                v-else-if="isCuratedFramesRoute"
                class="relative w-full max-w-xl"
              >
                <Search class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground" />
                <Input
                  v-model="searchDraftFrames"
                  class="h-10 rounded-2xl border-border/70 bg-background/70 pl-10 transition-[border-color,background-color,box-shadow] hover:border-primary/60 hover:bg-background/85 hover:ring-1 hover:ring-primary/30"
                  :class="searchDraftFrames.trim() ? 'pr-10' : ''"
                  :placeholder="t('shell.searchCuratedPlaceholder')"
                />
                <Button
                  v-if="searchDraftFrames.trim()"
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="absolute top-1/2 right-1.5 size-8 -translate-y-1/2 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
                  :aria-label="t('shell.clearCuratedSearch')"
                  @click="clearCuratedFramesSearch"
                >
                  <X class="size-4" />
                </Button>
              </div>
              <div
                v-else-if="isActorsRoute"
                class="relative w-full max-w-xl"
              >
                <Search class="pointer-events-none absolute top-1/2 left-3 z-[1] -translate-y-1/2 text-muted-foreground" />
                <Input
                  v-model="searchDraftActors"
                  class="h-10 rounded-2xl border-border/70 bg-background/70 pl-10 transition-[border-color,background-color,box-shadow] hover:border-primary/60 hover:bg-background/85 hover:ring-1 hover:ring-primary/30"
                  :class="searchDraftActors.trim() ? 'pr-10' : ''"
                  :placeholder="t('actors.searchPlaceholder')"
                  autocomplete="off"
                />
                <Button
                  v-if="searchDraftActors.trim()"
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="absolute top-1/2 right-1.5 z-[1] size-8 -translate-y-1/2 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
                  :aria-label="t('shell.clearSearch')"
                  @click="clearActorsSearch"
                >
                  <X class="size-4" />
                </Button>
              </div>
            </div>

            <div
              class="flex shrink-0 items-center gap-1.5 border-border/50 sm:border-l sm:pl-3 lg:pl-4"
              :title="t('shell.themeToggleHint')"
            >
              <Sun
                class="size-4 shrink-0 text-muted-foreground"
                aria-hidden="true"
              />
              <Switch
                :model-value="resolvedMode === 'dark'"
                class="motion-safe:transition-colors motion-safe:duration-200"
                :aria-label="t('shell.themeToggleAria')"
                @update:model-value="onShellAppearanceSwitch"
              />
              <Moon
                class="size-4 shrink-0 text-muted-foreground"
                aria-hidden="true"
              />
            </div>
          </div>

          <div class="min-h-0 min-w-0 flex-1 overflow-hidden">
            <div
              data-router-view-frame
              :class="routerViewFrameClass"
            >
              <RouterView />
            </div>
          </div>
        </section>
      </div>
    </div>

    <ScanProgressDock />
    <DevPerformanceBar v-if="isDev" />
    <Toaster :theme="resolvedMode" />

    <Teleport to="body">
      <div
        v-if="!isLgUp"
        class="fixed inset-0 z-[100] lg:hidden"
        :class="mobileSidebarOpen ? 'pointer-events-auto' : 'pointer-events-none'"
        :aria-hidden="!mobileSidebarOpen"
      >
        <button
          type="button"
          class="absolute inset-0 bg-black/50 transition-opacity duration-200"
          :class="mobileSidebarOpen ? 'opacity-100' : 'opacity-0'"
          tabindex="-1"
          :aria-label="t('shell.closeMenu')"
          @click="mobileSidebarOpen = false"
        />
        <aside
          class="absolute top-0 bottom-0 left-0 flex w-[min(304px,88vw)] max-w-full flex-col border-r border-border/60 bg-sidebar/98 shadow-2xl shadow-black/30 backdrop-blur-md transition-transform duration-200 ease-out"
          :class="mobileSidebarOpen ? 'translate-x-0' : '-translate-x-full'"
          role="dialog"
          aria-modal="true"
          :aria-label="t('shell.navDialog')"
        >
          <div class="min-h-0 flex-1 overflow-hidden p-3 pt-4">
            <AppSidebar />
          </div>
        </aside>
      </div>
    </Teleport>

    <DevEnvironmentBadge
      v-if="isDev"
      :show-perf-restore="devPerformanceBarHidden"
      @show-performance-monitor="setDevPerformanceBarHidden(false)"
    />
  </div>
</template>

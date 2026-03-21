<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { onKeyStroke, useMediaQuery, watchDebounced } from "@vueuse/core"
import { LayoutDashboard, Menu, Search, X } from "lucide-vue-next"
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router"
import AppSidebar from "@/components/jav-library/AppSidebar.vue"
import ScanProgressDock from "@/components/jav-library/ScanProgressDock.vue"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  buildMovieRouteQuery,
  getBrowseSourceMode,
  getLibraryActorExactQuery,
  getLibrarySearchQuery,
  getLibraryTagExactQuery,
  getSelectedMovieQuery,
  isLibraryRouteName,
  mergeLibraryQuery,
} from "@/lib/library-query"
import { useLibraryService } from "@/services/library-service"

const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

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

onKeyStroke("Escape", (e) => {
  if (mobileSidebarOpen.value && !isLgUp.value) {
    e.preventDefault()
    mobileSidebarOpen.value = false
  }
})

onBeforeUnmount(() => {
  document.body.style.overflow = ""
})

const shellGridClass = computed(() => {
  const base = "grid h-full min-h-0 gap-4 grid-cols-1"
  if (!isLgUp.value) {
    return base
  }
  return `${base} ${desktopSidebarCollapsed.value ? "lg:grid-cols-[4.5rem_minmax(0,1fr)]" : "lg:grid-cols-[280px_minmax(0,1fr)]"}`
})

const currentMovie = computed(() => {
  const routeMovieId = typeof route.params.id === "string" ? route.params.id : undefined
  const selectedMovieId = getSelectedMovieQuery(route.query)
  const candidateId = routeMovieId ?? selectedMovieId

  return candidateId ? libraryService.getMovieById(candidateId) : undefined
})

const currentMovieId = computed(() => currentMovie.value?.id)
const isLibraryRoute = computed(() => isLibraryRouteName(route.name))
const isDetailRoute = computed(() => route.name === "detail")
const isPlayerRoute = computed(() => route.name === "player")

const showHeaderBack = computed(() => !isLibraryRoute.value)

const headerBackTarget = computed(() => {
  if (isPlayerRoute.value && currentMovieId.value) {
    return {
      name: "detail",
      params: { id: currentMovieId.value },
      query: buildMovieRouteQuery(
        route.query,
        getBrowseSourceMode(route.query),
        currentMovieId.value,
      ),
    }
  }

  if (isDetailRoute.value) {
    return {
      name: getBrowseSourceMode(route.query),
      query: mergeLibraryQuery(route.query, {
        selected: currentMovieId.value,
      }),
    }
  }

  return { name: "library" }
})

const headerBackLabel = computed(() =>
  isPlayerRoute.value && currentMovieId.value ? "Back to details" : "Back to library",
)

/** 输入即时更新；同步到 URL 防抖，避免每键一次 router.replace 导致整库重算/重绘 */
const searchDraft = ref("")

watch(
  [() => route.name, () => route.query.q, () => route.query.tag, () => route.query.actor],
  () => {
    if (!isLibraryRoute.value) {
      return
    }
    const qPart = getLibrarySearchQuery(route.query)
    const tagPart = getLibraryTagExactQuery(route.query).trim()
    const actorPart = getLibraryActorExactQuery(route.query).trim()
    const next = qPart || tagPart || actorPart
    if (next !== searchDraft.value) {
      searchDraft.value = next
    }
  },
  { immediate: true },
)

watchDebounced(
  searchDraft,
  (value) => {
    if (!isLibraryRoute.value) {
      return
    }
    const normalized = value.trim()
    const currentQ = getLibrarySearchQuery(route.query).trim()
    const currentTag = getLibraryTagExactQuery(route.query).trim()
    const currentActor = getLibraryActorExactQuery(route.query).trim()
    if (
      normalized === currentQ &&
      !currentTag &&
      !currentActor
    ) {
      return
    }
    void router.replace({
      name: route.name ?? "library",
      query: mergeLibraryQuery(route.query, {
        q: normalized || undefined,
        tag: undefined,
        actor: undefined,
      }),
    })
  },
  { debounce: 280 },
)

function clearLibrarySearch() {
  searchDraft.value = ""
  if (!isLibraryRoute.value) {
    return
  }
  void router.replace({
    name: route.name ?? "library",
    query: mergeLibraryQuery(route.query, {
      q: undefined,
      tag: undefined,
      actor: undefined,
    }),
  })
}
</script>

<template>
  <div class="h-screen overflow-hidden bg-background text-foreground">
    <div class="h-full w-full px-3 py-3 lg:px-4 lg:py-4">
      <div :class="shellGridClass">
        <AppSidebar
          v-if="isLgUp"
          class="min-h-0"
          :compact="desktopSidebarCollapsed"
          show-collapse-toggle
          @toggle-compact="desktopSidebarCollapsed = !desktopSidebarCollapsed"
        />

        <section
          class="flex min-h-0 min-w-0 flex-col overflow-hidden rounded-[2rem] border border-border/70 bg-background/92 shadow-xl shadow-black/8"
        >
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border/70 px-4 py-4 lg:px-5">
            <div class="flex flex-wrap items-center gap-2">
              <Button
                v-if="!isLgUp"
                type="button"
                variant="secondary"
                size="icon"
                class="shrink-0 rounded-2xl lg:hidden"
                aria-label="打开导航菜单"
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
                v-if="isLibraryRoute"
                class="relative w-full max-w-lg"
              >
                <Search class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground" />
                <Input
                  v-model="searchDraft"
                  class="h-10 rounded-2xl border-border/70 bg-background/70 pl-10 transition-[border-color,background-color,box-shadow] hover:border-primary/60 hover:bg-background/85 hover:ring-1 hover:ring-primary/30"
                  :class="searchDraft.trim() ? 'pr-10' : ''"
                  placeholder="Search by code, title, actor, or tag"
                />
                <Button
                  v-if="searchDraft.trim()"
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="absolute top-1/2 right-1.5 size-8 -translate-y-1/2 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
                  aria-label="清除搜索"
                  @click="clearLibrarySearch"
                >
                  <X class="size-4" />
                </Button>
              </div>
            </div>
          </div>

          <div class="min-h-0 min-w-0 flex-1 overflow-hidden p-4 lg:p-5 xl:p-6">
            <RouterView />
          </div>
        </section>
      </div>
    </div>

    <ScanProgressDock />

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
          aria-label="关闭导航菜单"
          @click="mobileSidebarOpen = false"
        />
        <aside
          class="absolute top-0 bottom-0 left-0 flex w-[min(280px,88vw)] max-w-full flex-col border-r border-border/70 bg-sidebar/98 shadow-2xl shadow-black/30 backdrop-blur-md transition-transform duration-200 ease-out"
          :class="mobileSidebarOpen ? 'translate-x-0' : '-translate-x-full'"
          role="dialog"
          aria-modal="true"
          aria-label="导航"
        >
          <div class="min-h-0 flex-1 overflow-hidden p-3 pt-4">
            <AppSidebar />
          </div>
        </aside>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { watchDebounced } from "@vueuse/core"
import { LayoutDashboard, Search } from "lucide-vue-next"
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router"
import AppSidebar from "@/components/jav-library/AppSidebar.vue"
import ScanProgressDock from "@/components/jav-library/ScanProgressDock.vue"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  buildMovieRouteQuery,
  getBrowseSourceMode,
  getLibrarySearchQuery,
  getSelectedMovieQuery,
  isLibraryRouteName,
  mergeLibraryQuery,
} from "@/lib/library-query"
import { useLibraryService } from "@/services/library-service"

const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

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
  [() => route.name, () => route.query.q],
  () => {
    if (!isLibraryRoute.value) {
      return
    }
    const next = getLibrarySearchQuery(route.query)
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
    const current = getLibrarySearchQuery(route.query)
    if (normalized === current.trim()) {
      return
    }
    void router.replace({
      name: route.name ?? "library",
      query: mergeLibraryQuery(route.query, {
        q: normalized || undefined,
      }),
    })
  },
  { debounce: 280 },
)
</script>

<template>
  <div class="h-screen overflow-hidden bg-background text-foreground">
    <div class="h-full w-full px-3 py-3 lg:px-4 lg:py-4">
      <div class="grid h-full min-h-0 gap-4 xl:grid-cols-[280px_minmax(0,1fr)]">
        <AppSidebar />

        <section
          class="flex min-h-0 flex-col overflow-hidden rounded-[2rem] border border-border/70 bg-background/92 shadow-xl shadow-black/8"
        >
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border/70 px-4 py-4 lg:px-5">
            <div class="flex flex-wrap items-center gap-2">
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
                  placeholder="Search by code, title, actor, or tag"
                />
              </div>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-hidden p-4 lg:p-5 xl:p-6">
            <RouterView />
          </div>
        </section>
      </div>
    </div>

    <ScanProgressDock />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, shallowRef, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute } from "vue-router"
import type { ActorListItemDTO } from "@/api/types"
import ActorLibraryCard from "@/components/jav-library/ActorLibraryCard.vue"
import { getActorsSearchQuery } from "@/lib/actors-route-query"
import { useLibraryService } from "@/services/library-service"

const { t } = useI18n()
const route = useRoute()
const libraryService = useLibraryService()

const PAGE_SIZE = 48

const actors = shallowRef<ActorListItemDTO[]>([])
const total = ref(0)
const loading = ref(false)
const loadingMore = ref(false)
const loadError = ref("")
const scrollRoot = ref<HTMLElement | null>(null)
const loadMoreSentinel = ref<HTMLElement | null>(null)
let loadMoreObserver: IntersectionObserver | null = null

const listBase = computed(() => ({
  q: getActorsSearchQuery(route.query).trim() || undefined,
  sort: "movieCount" as const,
  limit: PAGE_SIZE,
}))

async function fetchFirstPage() {
  loading.value = true
  loadingMore.value = false
  loadError.value = ""
  try {
    const res = await libraryService.listActors({ ...listBase.value, offset: 0 })
    total.value = res.total
    actors.value = res.actors
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : t("actors.loadError")
    actors.value = []
    total.value = 0
  } finally {
    loading.value = false
    await nextTick()
    maybeAutoLoadMore()
  }
}

watch(
  () => getActorsSearchQuery(route.query),
  () => {
    void fetchFirstPage()
  },
  { immediate: true },
)

async function loadMore() {
  if (actors.value.length >= total.value || loading.value || loadingMore.value) {
    return
  }
  loadingMore.value = true
  loadError.value = ""
  try {
    const res = await libraryService.listActors({
      ...listBase.value,
      offset: actors.value.length,
    })
    total.value = res.total
    actors.value = [...actors.value, ...res.actors]
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : t("actors.loadError")
  } finally {
    loadingMore.value = false
    await nextTick()
    maybeAutoLoadMore()
  }
}

const hasMore = computed(() => actors.value.length < total.value)

function isLoadMoreSentinelNearViewport() {
  const target = loadMoreSentinel.value
  if (!target) {
    return false
  }
  const margin = 480
  const targetRect = target.getBoundingClientRect()
  const root = scrollRoot.value
  if (!root) {
    return targetRect.top <= window.innerHeight + margin && targetRect.bottom >= -margin
  }
  const rootRect = root.getBoundingClientRect()
  return targetRect.top <= rootRect.bottom + margin && targetRect.bottom >= rootRect.top - margin
}

function maybeAutoLoadMore() {
  if (!hasMore.value || loading.value || loadingMore.value || loadError.value) {
    return
  }
  if (isLoadMoreSentinelNearViewport()) {
    void loadMore()
  }
}

function observeLoadMoreSentinel() {
  loadMoreObserver?.disconnect()
  loadMoreObserver = null

  const target = loadMoreSentinel.value
  if (!target || typeof IntersectionObserver === "undefined") {
    return
  }

  loadMoreObserver = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        maybeAutoLoadMore()
      }
    },
    {
      root: scrollRoot.value,
      rootMargin: "480px 0px",
      threshold: 0,
    },
  )
  loadMoreObserver.observe(target)
}

watch([scrollRoot, loadMoreSentinel], observeLoadMoreSentinel, { flush: "post" })

onBeforeUnmount(() => {
  loadMoreObserver?.disconnect()
  loadMoreObserver = null
})
</script>

<template>
  <div
    class="mx-auto flex h-full min-h-0 w-full max-w-[min(100%,88rem)] flex-col gap-4 px-2 py-4 sm:px-4 lg:px-6 lg:py-6"
  >
    <header class="shrink-0 space-y-1">
      <h1 class="text-xl font-semibold tracking-tight lg:text-2xl">
        {{ t("actors.title") }}
      </h1>
      <p class="text-sm text-muted-foreground">
        {{ t("actors.subtitle") }}
      </p>
    </header>

    <div v-if="loadError" class="rounded-xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
      {{ loadError }}
    </div>

    <div
      v-else-if="!loading && actors.length === 0"
      class="flex flex-1 flex-col items-center justify-center rounded-2xl border border-dashed border-border/70 bg-muted/20 px-6 py-16 text-center text-muted-foreground"
    >
      <p class="text-sm">
        {{ t("actors.empty") }}
      </p>
    </div>

    <div
      v-else
      ref="scrollRoot"
      class="min-h-0 flex-1 overflow-y-auto"
      @scroll.passive="maybeAutoLoadMore"
    >
      <div
        v-if="loading && actors.length === 0"
        class="flex items-center justify-center py-20 text-sm text-muted-foreground"
      >
        {{ t("actors.loading") }}
      </div>
      <template v-else>
        <div
          data-actor-grid
          class="grid w-full min-w-0 grid-cols-[repeat(auto-fill,minmax(9.25rem,9.5rem))] justify-center gap-4 pb-4 [&>*]:min-w-0"
        >
          <ActorLibraryCard
            v-for="a in actors"
            :key="a.name"
            :actor="a"
          />
        </div>
        <div
          v-if="hasMore"
          ref="loadMoreSentinel"
          class="flex min-h-16 items-center justify-center py-4 text-sm text-muted-foreground"
          aria-live="polite"
        >
          <span v-if="loadingMore">{{ t("actors.loading") }}</span>
        </div>
      </template>
    </div>
  </div>
</template>

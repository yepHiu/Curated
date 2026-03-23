<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { X } from "lucide-vue-next"
import type { ActorListItemDTO } from "@/api/types"
import ActorLibraryCard from "@/components/jav-library/ActorLibraryCard.vue"
import { Button } from "@/components/ui/button"
import { getActorsSearchQuery, getActorsTagQuery, mergeActorsQuery } from "@/lib/actors-route-query"
import { useLibraryService } from "@/services/library-service"

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const PAGE_SIZE = 48

/** 详情页「我的标签」同款联想池：影片 userTags + 当前页演员标签 */
const actorTagSuggestionPool = computed(() => {
  const s = new Set<string>()
  for (const m of libraryService.movies.value) {
    for (const u of m.userTags ?? []) {
      const x = u.trim()
      if (x) s.add(x)
    }
  }
  for (const a of actors.value) {
    for (const u of a.userTags ?? []) {
      const x = u.trim()
      if (x) s.add(x)
    }
  }
  return [...s].sort((a, b) => a.localeCompare(b, "zh-CN", { numeric: true }))
})

const actors = ref<ActorListItemDTO[]>([])
const total = ref(0)
const loading = ref(false)
const loadingMore = ref(false)
const loadError = ref("")

const listBase = computed(() => ({
  q: getActorsSearchQuery(route.query).trim() || undefined,
  actorTag: getActorsTagQuery(route.query).trim() || undefined,
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
  }
}

watch(
  () =>
    [getActorsSearchQuery(route.query), getActorsTagQuery(route.query)] as const,
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
  }
}

function onTagsUpdated(row: ActorListItemDTO) {
  const i = actors.value.findIndex((a) => a.name === row.name)
  if (i >= 0) {
    actors.value = actors.value.map((a, j) => (j === i ? { ...row } : a))
  }
}

const hasMore = computed(() => actors.value.length < total.value)

const activeActorTagTrimmed = computed(() => getActorsTagQuery(route.query).trim())

function clearActorTagFilter() {
  void router.replace({
    name: "actors",
    query: mergeActorsQuery(route.query, { actorTag: undefined }),
  })
}

function onFilterByActorTag(payload: { tag: string }) {
  const tag = payload.tag.trim()
  if (!tag) {
    return
  }
  const cur = getActorsTagQuery(route.query).trim()
  if (cur === tag) {
    clearActorTagFilter()
    return
  }
  void router.replace({
    name: "actors",
    query: mergeActorsQuery(route.query, { actorTag: tag }),
  })
}
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

    <div
      v-if="activeActorTagTrimmed"
      class="flex flex-wrap items-center gap-2 rounded-xl border border-border/60 bg-muted/30 px-3 py-2.5"
    >
      <p class="min-w-0 text-sm text-muted-foreground">
        {{ t("actors.filteredByTag", { tag: activeActorTagTrimmed }) }}
      </p>
      <Button
        type="button"
        variant="outline"
        size="sm"
        class="h-8 shrink-0 gap-1 rounded-full"
        :aria-label="t('actors.ariaClearTagFilter')"
        @click="clearActorTagFilter"
      >
        <X class="size-3.5" />
        {{ t("actors.clearTagFilter") }}
      </Button>
    </div>

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

    <div v-else class="min-h-0 flex-1 overflow-y-auto">
      <div
        v-if="loading && actors.length === 0"
        class="flex items-center justify-center py-20 text-sm text-muted-foreground"
      >
        {{ t("actors.loading") }}
      </div>
      <template v-else>
        <div
          class="grid w-full min-w-0 grid-cols-2 gap-4 pb-4 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-5 [&>*]:min-w-0"
        >
          <ActorLibraryCard
            v-for="a in actors"
            :key="a.name"
            :actor="a"
            :user-tag-suggestions="actorTagSuggestionPool"
            @tags-updated="onTagsUpdated"
            @filter-by-actor-tag="onFilterByActorTag"
          />
        </div>
        <div v-if="hasMore" class="flex justify-center py-4">
          <Button
            type="button"
            variant="outline"
            class="rounded-xl"
            :disabled="loadingMore"
            @click="loadMore"
          >
            {{ loadingMore ? t("actors.loading") : t("actors.loadMore") }}
          </Button>
        </div>
      </template>
    </div>
  </div>
</template>

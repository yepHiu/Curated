<script setup lang="ts">
import { computed, onMounted, ref, shallowRef, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import ComicBatchActionBar from "@/components/jav-library/comics/ComicBatchActionBar.vue"
import ComicLibraryPage from "@/components/jav-library/comics/ComicLibraryPage.vue"
import { pushAppToast } from "@/composables/use-app-toast"
import { filterComics } from "@/lib/comic-search"
import { sortComics, type ComicLibrarySortValue } from "@/lib/comic-sort"
import { buildComicReaderRouteFromSource } from "@/lib/navigation-intent"
import { useComicLibraryService } from "@/services/comic-library-service"

const route = useRoute()
const router = useRouter()
const comicService = useComicLibraryService()
const { t } = useI18n()
const BATCH_SELECT_VISIBLE_MAX = 100

const searchQuery = computed(() =>
  typeof route.query.q === "string" ? route.query.q : "",
)
const activeSort = computed<ComicLibrarySortValue>(() => {
  const raw = typeof route.query.sort === "string" ? route.query.sort : "addedAt"
  if (["fileName", "favorite"].includes(raw)) {
    return raw as ComicLibrarySortValue
  }
  return "addedAt"
})
const visibleComics = computed(() =>
  sortComics(
    filterComics(comicService.comics.value, {
      q: searchQuery.value,
    }),
    activeSort.value,
  ),
)
const batchMode = ref(false)
const batchSelectedIds = shallowRef<Set<string>>(new Set())
const batchOperationBusy = ref(false)
const batchSelectedIdsList = computed(() => [...batchSelectedIds.value])
const batchSelectedCount = computed(() => batchSelectedIds.value.size)

onMounted(() => {
  void Promise.resolve(comicService.refreshSettings())
    .then(() => comicService.reloadComicsFromApi())
    .catch((error) => {
      console.warn("[ComicsView] failed to load comics", error)
    })
})

function updateSearch(value: string) {
  const q = value.trim()
  void router.replace({
    name: "comics",
    query: {
      ...route.query,
      filter: undefined,
      q: q || undefined,
    },
  })
}

function updateSort(value: ComicLibrarySortValue) {
  void router.replace({
    name: "comics",
    query: {
      ...route.query,
      filter: undefined,
      sort: value === "addedAt" ? undefined : value,
    },
  })
}

function openDetails(comicId: string) {
  void router.push({ name: "comic-detail", params: { id: comicId } })
}

function openReader(comicId: string, pageIndex: number) {
  void router.push(buildComicReaderRouteFromSource(comicId, pageIndex, route.fullPath))
}

function toggleFavorite(payload: { comicId: string; nextValue: boolean }) {
  void comicService.patchComic(payload.comicId, { favorite: payload.nextValue })
}

function clearBatchSelection() {
  batchSelectedIds.value = new Set()
}

function enterBatchMode() {
  batchMode.value = true
}

function exitBatchMode() {
  batchMode.value = false
  clearBatchSelection()
}

function toggleBatchSelect(comicId: string) {
  const id = comicId.trim()
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
  const ids = visibleComics.value.map((comic) => comic.id)
  if (ids.length > BATCH_SELECT_VISIBLE_MAX) {
    pushAppToast(t("comics.batchSelectVisibleCap", { max: BATCH_SELECT_VISIBLE_MAX }), {
      variant: "warning",
    })
    batchSelectedIds.value = new Set(ids.slice(0, BATCH_SELECT_VISIBLE_MAX))
    return
  }
  batchSelectedIds.value = new Set(ids)
}

watch([searchQuery, activeSort], () => {
  clearBatchSelection()
})

watch(visibleComics, (comics) => {
  const visibleIds = new Set(comics.map((comic) => comic.id))
  const next = new Set([...batchSelectedIds.value].filter((id) => visibleIds.has(id)))
  if (next.size !== batchSelectedIds.value.size) {
    batchSelectedIds.value = next
  }
})

async function runComicBatch(
  summaryKey: string,
  action: (comicId: string) => Promise<void>,
  afterSuccess?: () => void,
) {
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0) return
  batchOperationBusy.value = true
  let fail = 0
  try {
    for (const id of ids) {
      try {
        await action(id)
      } catch {
        fail++
      }
    }
    afterSuccess?.()
  } finally {
    batchOperationBusy.value = false
  }
  pushAppToast(t(summaryKey, { ok: ids.length - fail, fail }), {
    variant: fail === ids.length ? "destructive" : fail > 0 ? "warning" : "success",
  })
}

async function runBatchAddFavorite() {
  await runComicBatch("comics.batchFavoriteSummary", async (id) => {
    await comicService.patchComic(id, { favorite: true })
  })
}

async function runBatchRemoveFavorite() {
  await runComicBatch("comics.batchUnfavoriteSummary", async (id) => {
    await comicService.patchComic(id, { favorite: false })
  })
}

async function runBatchAddTag(tag: string) {
  const trimmed = tag.trim()
  if (!trimmed) return
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0) return
  batchOperationBusy.value = true
  let fail = 0
  try {
    for (const id of ids) {
      try {
        const comic = comicService.comics.value.find((item) => item.id === id)
        const base = comic?.tags ?? []
        if (base.includes(trimmed)) continue
        await comicService.patchComic(id, { tags: [...base, trimmed] })
      } catch {
        fail++
      }
    }
  } finally {
    batchOperationBusy.value = false
  }
  pushAppToast(t("comics.batchTagSummary", { ok: ids.length - fail, fail, tag: trimmed }), {
    variant: fail === ids.length ? "destructive" : fail > 0 ? "warning" : "success",
  })
}

async function runBatchDeleteComics() {
  await runComicBatch(
    "comics.batchDeleteSummary",
    async (id) => {
      await comicService.deleteComic(id)
    },
    () => {
      clearBatchSelection()
      exitBatchMode()
    },
  )
}
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
    <div
      data-comics-view-content
      class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden px-[var(--app-page-px)] py-[var(--app-page-py)] sm:px-[var(--app-page-px-sm)] lg:px-[var(--app-page-px-lg)] lg:py-[var(--app-page-py-lg)] xl:px-[var(--app-page-px-xl)]"
    >
      <ComicLibraryPage
        :comics="visibleComics"
        :active-sort="activeSort"
        :search-query="searchQuery"
        :load-error="comicService.loadError.value ?? ''"
        :batch-mode="batchMode"
        :batch-selected-ids="batchSelectedIdsList"
        @update-search="updateSearch"
        @update:sort="updateSort"
        @open-details="openDetails"
        @open-reader="openReader"
        @toggle-favorite="toggleFavorite"
        @enter-batch-mode="enterBatchMode"
        @exit-batch-mode="exitBatchMode"
        @select-all-visible-in-batch="selectAllVisibleInBatch"
        @toggle-batch-select="toggleBatchSelect"
      />
    </div>

    <ComicBatchActionBar
      v-if="batchMode"
      :selected-count="batchSelectedCount"
      :operation-busy="batchOperationBusy"
      @exit="exitBatchMode"
      @clear-selection="clearBatchSelection"
      @select-all-visible="selectAllVisibleInBatch"
      @add-favorite="runBatchAddFavorite"
      @remove-favorite="runBatchRemoveFavorite"
      @add-tag="runBatchAddTag"
      @delete-comics="runBatchDeleteComics"
    />
  </div>
</template>

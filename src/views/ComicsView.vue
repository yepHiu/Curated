<script setup lang="ts">
import { computed, onMounted } from "vue"
import { useRoute, useRouter } from "vue-router"
import ComicLibraryPage from "@/components/jav-library/comics/ComicLibraryPage.vue"
import type { ComicReadStatus } from "@/domain/comic/types"
import { filterComics } from "@/lib/comic-search"
import { useComicLibraryService } from "@/services/comic-library-service"

type ComicLibraryFilterValue = "all" | "favorite" | ComicReadStatus

const route = useRoute()
const router = useRouter()
const comicService = useComicLibraryService()

const searchQuery = computed(() =>
  typeof route.query.q === "string" ? route.query.q : "",
)
const activeFilter = computed<ComicLibraryFilterValue>(() => {
  const raw = typeof route.query.filter === "string" ? route.query.filter : "all"
  if (["favorite", "unread", "reading", "read"].includes(raw)) {
    return raw as ComicLibraryFilterValue
  }
  return "all"
})
const visibleComics = computed(() =>
  filterComics(comicService.comics.value, {
    q: searchQuery.value,
    favorite: activeFilter.value === "favorite",
    readStatus: activeFilter.value === "favorite" ? "all" : activeFilter.value,
  }),
)

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
      q: q || undefined,
    },
  })
}

function updateActiveFilter(value: ComicLibraryFilterValue) {
  void router.replace({
    name: "comics",
    query: {
      ...route.query,
      filter: value === "all" ? undefined : value,
    },
  })
}

function openDetails(comicId: string) {
  void router.push({ name: "comic-detail", params: { id: comicId } })
}

function openReader(comicId: string, pageIndex: number) {
  void router.push({
    name: "comic-reader",
    params: { id: comicId, pageIndex: String(Math.max(0, pageIndex)) },
  })
}

function toggleFavorite(payload: { comicId: string; nextValue: boolean }) {
  void comicService.patchComic(payload.comicId, { favorite: payload.nextValue })
}
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 flex-col px-[var(--app-page-px)] py-[var(--app-page-py)] sm:px-[var(--app-page-px-sm)] lg:px-[var(--app-page-px-lg)] lg:py-[var(--app-page-py-lg)] xl:px-[var(--app-page-px-xl)]">
    <ComicLibraryPage
      :comics="visibleComics"
      :active-filter="activeFilter"
      :search-query="searchQuery"
      :load-error="comicService.loadError.value ?? ''"
      @update-search="updateSearch"
      @update-active-filter="updateActiveFilter"
      @open-details="openDetails"
      @open-reader="openReader"
      @toggle-favorite="toggleFavorite"
    />
  </div>
</template>

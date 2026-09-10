<script setup lang="ts">
import { computed, onMounted } from "vue"
import { useRoute, useRouter } from "vue-router"
import PhotoLibraryPage from "@/components/jav-library/photos/PhotoLibraryPage.vue"
import { filterPhotos } from "@/lib/photo-search"
import { sortPhotos, type PhotoLibrarySortValue } from "@/lib/photo-sort"
import { usePhotoLibraryService } from "@/services/photo-library-service"

const route = useRoute()
const router = useRouter()
const photoService = usePhotoLibraryService()

const searchQuery = computed(() =>
  typeof route.query.q === "string" ? route.query.q : "",
)
const activeSort = computed<PhotoLibrarySortValue>(() => {
  const raw = typeof route.query.sort === "string" ? route.query.sort : "addedAt"
  if (["fileName", "favorite"].includes(raw)) {
    return raw as PhotoLibrarySortValue
  }
  return "addedAt"
})
const visiblePhotos = computed(() =>
  sortPhotos(
    filterPhotos(photoService.photos.value, {
      q: searchQuery.value,
    }),
    activeSort.value,
  ),
)

onMounted(() => {
  void Promise.resolve(photoService.refreshSettings())
    .then(() => photoService.reloadPhotosFromApi())
    .catch((error) => {
      console.warn("[PhotosView] failed to load photos", error)
    })
})

function updateSearch(value: string) {
  const q = value.trim()
  void router.replace({
    name: "photos",
    query: {
      ...route.query,
      q: q || undefined,
    },
  })
}

function updateSort(value: PhotoLibrarySortValue) {
  void router.replace({
    name: "photos",
    query: {
      ...route.query,
      sort: value === "addedAt" ? undefined : value,
    },
  })
}

function openDetails(photoId: string) {
  void router.push({ name: "photo-detail", params: { id: photoId } })
}

function openViewer(photoId: string, pageIndex: number) {
  void router.push({
    name: "photo-viewer",
    params: { id: photoId, pageIndex: String(pageIndex) },
    query: { returnTo: route.fullPath },
  })
}
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
    <div
      data-photos-view-content
      class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden px-[var(--app-page-px)] py-[var(--app-page-py)] sm:px-[var(--app-page-px-sm)] lg:px-[var(--app-page-px-lg)] lg:py-[var(--app-page-py-lg)] xl:px-[var(--app-page-px-xl)]"
    >
      <PhotoLibraryPage
        :photos="visiblePhotos"
        :active-sort="activeSort"
        :search-query="searchQuery"
        :load-error="photoService.loadError.value ?? ''"
        @update-search="updateSearch"
        @update:sort="updateSort"
        @open-details="openDetails"
        @open-viewer="openViewer"
      />
    </div>
  </div>
</template>

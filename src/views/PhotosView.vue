<script setup lang="ts">
import { computed, onMounted } from "vue"
import { useRoute, useRouter } from "vue-router"
import PhotoLibraryPage from "@/components/jav-library/photos/PhotoLibraryPage.vue"
import {
  isLegacyFavoriteSort,
  parsePhotoLibraryBrowse,
  patchBookLibraryQuery,
  photoLibraryHasConstraints,
  type BookLibrarySortValue,
} from "@/lib/book-library-query"
import { filterPhotos } from "@/lib/photo-search"
import { sortPhotos } from "@/lib/photo-sort"
import { usePhotoLibraryService } from "@/services/photo-library-service"

const route = useRoute()
const router = useRouter()
const photoService = usePhotoLibraryService()

const browse = computed(() => parsePhotoLibraryBrowse(route.query))
const searchQuery = computed(() => browse.value.q)
const activeSort = computed(() => browse.value.sort)
const visiblePhotos = computed(() =>
  sortPhotos(
    filterPhotos(photoService.photos.value, {
      q: browse.value.q,
      tag: browse.value.tag,
    }),
    browse.value.sort,
  ),
)
const hasConstraints = computed(() => photoLibraryHasConstraints(browse.value))

/** 用规范化 query 替换当前写真墙地址，保留精确标签等约束。 */
function replaceBrowseQuery(patch: Parameters<typeof patchBookLibraryQuery>[1]) {
  void router.replace({
    name: "photos",
    query: patchBookLibraryQuery(route.query, patch),
  })
}

/** 进页只补齐尚未加载的设置与列表，扫描/导入仍走强制 reload。 */
function hydrateLibrary() {
  if (isLegacyFavoriteSort(route.query)) {
    replaceBrowseQuery({ sort: "addedAt" })
  }
  void photoService.ensurePhotosLoaded().catch((error) => {
    console.warn("[PhotosView] failed to load photos", error)
  })
}

/** 加载失败重试时强制重拉列表。 */
function reloadLibrary() {
  void photoService.reloadPhotosFromApi().catch((error) => {
    console.warn("[PhotosView] failed to reload photos", error)
  })
}
onMounted(hydrateLibrary)

/** 更新壳层投影下来的自由文本搜索。 */
function updateSearch(value: string) {
  replaceBrowseQuery({ q: value })
}

/** 只改排序。 */
function updateSort(value: BookLibrarySortValue) {
  replaceBrowseQuery({ sort: value })
}

/** 清除精确标签筛选。 */
function clearTag() {
  replaceBrowseQuery({ tag: "" })
}

/** 清空搜索和标签，保留当前排序。 */
function clearFilters() {
  replaceBrowseQuery({ q: "", tag: "" })
}

/** 打开写真集详情。 */
function openDetails(photoId: string) {
  void router.push({ name: "photo-detail", params: { id: photoId } })
}

/** 带着当前墙面地址进入查看器，便于返回。 */
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
        :tag="browse.tag"
        :has-constraints="hasConstraints"
        :loading="!photoService.photosLoaded.value"
        :load-error="photoService.loadError.value ?? ''"
        @retry="reloadLibrary"
        @update-search="updateSearch"
        @clear-tag="clearTag"
        @clear-filters="clearFilters"
        @update:sort="updateSort"
        @open-details="openDetails"
        @open-viewer="openViewer"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, shallowRef, watch } from "vue"
import { useI18n } from "vue-i18n"
import MediaBatchActionBar from "@/components/jav-library/MediaBatchActionBar.vue"
import PhotoLibraryPage from "@/components/jav-library/photos/PhotoLibraryPage.vue"
import { pushAppToast } from "@/composables/use-app-toast"
import { useRoute, useRouter } from "vue-router"
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
const { t } = useI18n()
const BATCH_SELECT_VISIBLE_MAX = 100
const batchMode = ref(false)
const batchSelectedIds = shallowRef<Set<string>>(new Set())
const batchOperationBusy = ref(false)
const batchSelectedIdsList = computed(() => [...batchSelectedIds.value])
const batchSelectedCount = computed(() => batchSelectedIds.value.size)

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
/** Keep selections scoped to the current visible photo wall. */
function clearBatchSelection() {
  batchSelectedIds.value = new Set()
}
function enterBatchMode() {
  batchMode.value = true
}
function exitBatchMode() {
  if (batchOperationBusy.value) return
  batchMode.value = false
  clearBatchSelection()
}
function toggleBatchSelect(photoId: string) {
  if (batchOperationBusy.value) return
  const next = new Set(batchSelectedIds.value)
  if (next.has(photoId)) next.delete(photoId)
  else next.add(photoId)
  batchSelectedIds.value = next
}
function selectAllVisibleInBatch() {
  if (batchOperationBusy.value) return
  const ids = visiblePhotos.value.map((photo) => photo.id)
  if (ids.length > BATCH_SELECT_VISIBLE_MAX) {
    pushAppToast(t("photos.batchSelectVisibleCap", { max: BATCH_SELECT_VISIBLE_MAX }), { variant: "warning" })
  }
  batchSelectedIds.value = new Set(ids.slice(0, BATCH_SELECT_VISIBLE_MAX))
}
watch(() => [browse.value.q, browse.value.tag, browse.value.sort], clearBatchSelection)
watch(visiblePhotos, (photos) => {
  const visible = new Set(photos.map((photo) => photo.id))
  const next = new Set([...batchSelectedIds.value].filter((id) => visible.has(id)))
  if (next.size !== batchSelectedIds.value.size) batchSelectedIds.value = next
})

async function runPhotoBatch(summaryKey: string, action: (id: string) => Promise<unknown>, tag?: string) {
  const ids = [...batchSelectedIds.value]
  if (!ids.length || batchOperationBusy.value) return
  batchOperationBusy.value = true
  const failed = new Set<string>()
  try {
    for (const id of ids) {
      try { await action(id) } catch { failed.add(id) }
    }
  } finally {
    batchOperationBusy.value = false
  }
  if (summaryKey === "photos.batchDeleteSummary") {
    batchSelectedIds.value = new Set(failed)
    if (failed.size === 0) exitBatchMode()
  }
  pushAppToast(t(summaryKey, { ok: ids.length - failed.size, fail: failed.size, tag }), {
    variant: failed.size === ids.length ? "destructive" : failed.size ? "warning" : "success",
  })
}
function runBatchAddFavorite() {
  return runPhotoBatch("photos.batchFavoriteSummary", (id) => photoService.patchPhoto(id, { favorite: true }))
}
function runBatchRemoveFavorite() {
  return runPhotoBatch("photos.batchUnfavoriteSummary", (id) => photoService.patchPhoto(id, { favorite: false }))
}
async function runBatchAddTag(tag: string) {
  const trimmed = tag.trim()
  if (!trimmed) return
  await runPhotoBatch("photos.batchTagSummary", async (id) => {
    const photo = photoService.photos.value.find((item) => item.id === id)
    if (!photo || photo.tags.includes(trimmed)) return
    await photoService.replacePhotoTags(id, [...photo.tags, trimmed])
  }, trimmed)
}
function runBatchDeletePhotos() {
  return runPhotoBatch("photos.batchDeleteSummary", (id) => photoService.deletePhoto(id))
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
        :batch-mode="batchMode"
        :batch-selected-ids="batchSelectedIdsList"
        @retry="reloadLibrary"
        @update-search="updateSearch"
        @clear-tag="clearTag"
        @clear-filters="clearFilters"
        @update:sort="updateSort"
        @open-details="openDetails"
        @open-viewer="openViewer"
        @enter-batch-mode="enterBatchMode"
        @exit-batch-mode="exitBatchMode"
        @select-all-visible-in-batch="selectAllVisibleInBatch"
        @toggle-batch-select="toggleBatchSelect"
      />
    </div>
    <MediaBatchActionBar kind="photos" v-if="batchMode" :selected-count="batchSelectedCount" :operation-busy="batchOperationBusy" @clear-selection="clearBatchSelection" @add-favorite="runBatchAddFavorite" @remove-favorite="runBatchRemoveFavorite" @add-tag="runBatchAddTag" @delete-selection="runBatchDeletePhotos" />
  </div>
</template>

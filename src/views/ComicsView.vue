<script setup lang="ts">
import { computed, onMounted, ref, shallowRef, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import MediaBatchActionBar from "@/components/jav-library/MediaBatchActionBar.vue"
import ComicLibraryPage from "@/components/jav-library/comics/ComicLibraryPage.vue"
import { pushAppToast } from "@/composables/use-app-toast"
import type { ComicReadStatus } from "@/domain/comic/types"
import {
  comicLibraryHasConstraints,
  isLegacyFavoriteSort,
  parseComicLibraryBrowse,
  patchBookLibraryQuery,
  type BookLibrarySortValue,
} from "@/lib/book-library-query"
import { filterComics } from "@/lib/comic-search"
import { sortComics } from "@/lib/comic-sort"
import { buildComicReaderRouteFromSource } from "@/lib/navigation-intent"
import { useComicLibraryService } from "@/services/comic-library-service"

const route = useRoute()
const router = useRouter()
const comicService = useComicLibraryService()
const { t } = useI18n()
const BATCH_SELECT_VISIBLE_MAX = 100

const browse = computed(() => parseComicLibraryBrowse(route.query))
const searchQuery = computed(() => browse.value.q)
const activeSort = computed(() => browse.value.sort)
const visibleComics = computed(() =>
  sortComics(
    filterComics(comicService.comics.value, {
      q: browse.value.q,
      tag: browse.value.tag,
      favorite: browse.value.favorite,
      readStatus: browse.value.readStatus,
    }),
    browse.value.sort,
  ),
)
const hasConstraints = computed(() => comicLibraryHasConstraints(browse.value))
const batchMode = ref(false)
const batchSelectedIds = shallowRef<Set<string>>(new Set())
const batchOperationBusy = ref(false)
const batchSelectedIdsList = computed(() => [...batchSelectedIds.value])
const batchSelectedCount = computed(() => batchSelectedIds.value.size)

/** 用规范化 query 替换当前漫画墙地址，不丢掉其它书库约束。 */
function replaceBrowseQuery(patch: Parameters<typeof patchBookLibraryQuery>[1]) {
  void router.replace({
    name: "comics",
    query: patchBookLibraryQuery(route.query, patch),
  })
}

/** 进页只补齐尚未加载的设置与列表，扫描/导入仍走强制 reload。 */
function hydrateLibrary() {
  if (isLegacyFavoriteSort(route.query)) {
    replaceBrowseQuery({ favorite: true, sort: "addedAt" })
  }
  void comicService.ensureComicsLoaded().catch((error) => {
    console.warn("[ComicsView] failed to load comics", error)
  })
}

/** 加载失败重试时强制重拉列表。 */
function reloadLibrary() {
  void comicService.reloadComicsFromApi().catch((error) => {
    console.warn("[ComicsView] failed to reload comics", error)
  })
}
onMounted(hydrateLibrary)

/** 更新壳层投影下来的自由文本搜索。 */
function updateSearch(value: string) {
  replaceBrowseQuery({ q: value })
}

/** 只改排序，不把筛选写进 sort。 */
function updateSort(value: BookLibrarySortValue) {
  replaceBrowseQuery({ sort: value })
}

/** 清除精确标签筛选。 */
function clearTag() {
  replaceBrowseQuery({ tag: "" })
}

/** 切换是否只看收藏。 */
function updateFavorite(value: boolean) {
  replaceBrowseQuery({ favorite: value })
}

/** 切换阅读状态筛选。 */
function updateReadStatus(value: ComicReadStatus | "all") {
  replaceBrowseQuery({ readStatus: value })
}

/** 清空搜索、标签、收藏和阅读状态，保留当前排序。 */
function clearFilters() {
  replaceBrowseQuery({
    q: "",
    tag: "",
    favorite: false,
    readStatus: "all",
  })
}

/** 打开漫画详情。 */
function openDetails(comicId: string) {
  void router.push({ name: "comic-detail", params: { id: comicId } })
}

/** 带着当前墙面地址进入阅读器，便于返回。 */
function openReader(comicId: string, pageIndex: number) {
  void router.push(buildComicReaderRouteFromSource(comicId, pageIndex, route.fullPath))
}

/** 卡片收藏开关走漫画服务，不经过墙面筛选 query。 */
function toggleFavorite(payload: { comicId: string; nextValue: boolean }) {
  void comicService.patchComic(payload.comicId, { favorite: payload.nextValue })
}

/** 清空当前批量选择。 */
function clearBatchSelection() {
  batchSelectedIds.value = new Set()
}

/** 进入漫画墙选择态。 */
function enterBatchMode() {
  batchMode.value = true
}

/** 退出选择态并丢掉当前勾选。 */
function exitBatchMode() {
  batchMode.value = false
  clearBatchSelection()
}

/** 勾选或取消勾选一本可见漫画。 */
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

/** 全选当前墙上可见漫画，超出上限时只取前 N 本。 */
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

watch(
  () => [browse.value.q, browse.value.tag, browse.value.favorite, browse.value.readStatus, browse.value.sort],
  () => {
    // 筛选或排序变化后丢掉已不在墙上的选择。
    clearBatchSelection()
  },
)

watch(visibleComics, (comics) => {
  const visibleIds = new Set(comics.map((comic) => comic.id))
  const next = new Set([...batchSelectedIds.value].filter((id) => visibleIds.has(id)))
  if (next.size !== batchSelectedIds.value.size) {
    batchSelectedIds.value = next
  }
})

/** 逐本执行批量操作并汇总成功/失败条数。 */
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

/** 批量把所选漫画标为收藏。 */
/** 批量把所选漫画标为收藏。 */
/** 批量把所选漫画标为收藏。 */
/** 批量把所选漫画标为收藏。 */
async function runBatchAddFavorite() {
  await runComicBatch("comics.batchFavoriteSummary", async (id) => {
    // 把所选漫画标为收藏。
    await comicService.patchComic(id, { favorite: true })
  })
}

/** 批量取消所选漫画的收藏。 */
/** 批量取消所选漫画的收藏。 */
/** 批量取消所选漫画的收藏。 */
/** 批量取消所选漫画的收藏。 */
async function runBatchRemoveFavorite() {
  await runComicBatch("comics.batchUnfavoriteSummary", async (id) => {
    // 取消所选漫画的收藏。
    await comicService.patchComic(id, { favorite: false })
  })
}

/** 给尚未带有该标签的所选漫画追加同一标签。 */
/** 给尚未带有该标签的所选漫画追加同一标签。 */
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

/** 批量删除所选漫画索引，成功后退出选择态。 */
async function runBatchDeleteComics() {
  await runComicBatch(
    "comics.batchDeleteSummary",
    async (id) => {
      // 从漫画库移除所选索引，不删除源压缩包。
      await comicService.deleteComic(id)
    },
    () => {
      // 删除成功后退出选择态，避免对着空选择继续操作。
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
        :tag="browse.tag"
        :favorite="browse.favorite"
        :read-status="browse.readStatus"
        :has-constraints="hasConstraints"
        :loading="!comicService.comicsLoaded.value"
        :load-error="comicService.loadError.value ?? ''"
        :batch-mode="batchMode"
        :batch-selected-ids="batchSelectedIdsList"
        @retry="reloadLibrary"
        @update-search="updateSearch"
        @clear-tag="clearTag"
        @update-favorite="updateFavorite"
        @update-read-status="updateReadStatus"
        @clear-filters="clearFilters"
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

    <MediaBatchActionBar
      kind="comics"
      v-if="batchMode"
      :selected-count="batchSelectedCount"
      :operation-busy="batchOperationBusy"
      @clear-selection="clearBatchSelection"
      @add-favorite="runBatchAddFavorite"
      @remove-favorite="runBatchRemoveFavorite"
      @add-tag="runBatchAddTag"
      @delete-selection="runBatchDeleteComics"
    />
  </div>
</template>

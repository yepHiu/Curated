<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { Filter } from "lucide-vue-next"
import { useLibraryService } from "@/services/library-service"
import type { WishlistPage } from "@/domain/wishlist/types"
import WishlistGrid from "@/components/jav-library/wishlist/WishlistGrid.vue"
import WishlistPluginStatus from "@/components/jav-library/wishlist/WishlistPluginStatus.vue"
import { buildMovieGridChunkStyle } from "@/lib/movie-grid-template"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import MediaEmptyState from "@/components/jav-library/MediaEmptyState.vue"
import { DropdownMenu, DropdownMenuContent, DropdownMenuRadioGroup, DropdownMenuRadioItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
const { t } = useI18n()
const route = useRoute(), router = useRouter(), service = useLibraryService().wishlist
const page = ref<WishlistPage>({ items: [], total: 0, pendingCount: 0 })
const loading = ref(true), error = ref(false), draft = ref(String(route.query.q ?? ""))
const gridStyle = buildMovieGridChunkStyle({ minTrackWidth: "var(--movie-grid-min-track)", gap: "var(--movie-grid-gap)" })
const status = computed(() => { /* 从 URL 保留用户筛选。 */ return String(route.query.status ?? "pending") })
const hasOtherItems = ref(false)
const hasConstraints = computed(() => Boolean(String(route.query.q ?? "").trim() || route.query.cursor ||
  (status.value !== "all" && (status.value !== "pending" || hasOtherItems.value))))
/** Return to the entire wishlist so items in other states are visible too. */
function clearFilters() {
  clearTimeout(searchTimer)
  draft.value = ""
  void router.replace({ query: { ...route.query, q: undefined, cursor: undefined, status: "all" } })
}
let alive = true
let generation = 0, timer: ReturnType<typeof setTimeout> | undefined
let searchTimer: ReturnType<typeof setTimeout> | undefined
/** 加载当前页，丢弃旧筛选的迟到响应。 */
async function load(quiet = false) {
  const current = ++generation
  if (!quiet) loading.value = true
  try {
    const result = await service.list({ status: status.value, q: String(route.query.q ?? ""), cursor: String(route.query.cursor ?? ""), limit: 60 })
    if (current !== generation) return
    // An empty default tab may still have completed wishes. Check before calling the collection empty.
    let otherItems = false
    if (!result.items.length && status.value === "pending" && !String(route.query.q ?? "").trim() && !route.query.cursor) {
      const all = await service.list({ status: "all", limit: 1 })
      otherItems = all.total > 0
    }
    if (current !== generation) return
    hasOtherItems.value = otherItems
    page.value = result; error.value = false
  } catch { if (current === generation) error.value = true }
  finally { if (current === generation) loading.value = false }
}
/** 页面在前台且仍有工作时轮询，避免后台页面持续请求。 */
async function poll() { if (!document.hidden && page.value.items.some((item) => { /* 仅处理中的条目需要轮询。 */ return ["queued", "running"].includes(item.enrichmentState) })) await load(true); if (alive) timer = setTimeout(poll, 6000) }
/** 筛选或搜索改变后从第一页重新查询。 */
function setFilter(value: unknown) { if (typeof value === "string" && value) void router.replace({ query: { ...route.query, status: value, cursor: undefined } }) }
/** 搜索防抖，避免每个按键触发联网。 */
function search() { clearTimeout(searchTimer); searchTimer = setTimeout(() => { /* 写 URL 以支持返回恢复。 */ void router.replace({ query: { ...route.query, q: draft.value || undefined, cursor: undefined } }) }, 300) }
/** 打开详情并记住返回目标与当前行。 */
function open(id: string) { void router.push({ name: "wishlist-detail", params: { id }, query: { back: route.fullPath } }) }
/** 切换游标分页，不累积整个库的图像。 */
function nextPage() { void router.push({ query: { ...route.query, cursor: page.value.nextCursor } }) }
/** 页面恢复前台时重新同步入库状态。 */
function visible() { if (!document.hidden) void load(true) }
watch(() => route.fullPath, () => { /* 路由筛选控制唯一加载入口。 */ draft.value = String(route.query.q ?? ""); void load() }, { immediate: true })
onMounted(() => { /* 注册前台刷新和轮询。 */ timer = setTimeout(poll, 6000); document.addEventListener("visibilitychange", visible) })
onBeforeUnmount(() => { /* 卸载后禁止迟到写入或重新排队。 */ alive = false; generation++; clearTimeout(timer); clearTimeout(searchTimer); document.removeEventListener("visibilitychange", visible) })
</script>
<template>
  <section class="flex h-full min-h-0 min-w-0 flex-1 flex-col gap-3">
    <h1 class="sr-only">{{ t('wishlist.title') }}</h1>
    <div class="flex min-w-0 flex-wrap items-center justify-between gap-3">
      <WishlistPluginStatus />
      <div class="flex min-w-0 flex-1 flex-wrap items-center justify-end gap-1.5">
        <Input v-model="draft" class="min-h-11 w-full rounded-full sm:w-64 lg:min-h-8" :placeholder="t('wishlist.search')" :aria-label="t('wishlist.search')" @input="search" />
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="outline" class="min-h-11 shrink-0 rounded-full px-3 lg:min-h-8" :aria-label="t('wishlist.filter')">
              <Filter data-icon="inline-start" aria-hidden="true" />
              {{ t(`wishlist.filters.${status}`) }}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="rounded-2xl border-border/70">
            <DropdownMenuRadioGroup :model-value="status" @update:model-value="setFilter">
              <DropdownMenuRadioItem v-for="filter in ['pending', 'in_library', 'completed', 'all']" :key="filter" :value="filter" class="min-h-11 lg:min-h-8">
                {{ t(`wishlist.filters.${filter}`) }}
              </DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
    <div v-if="loading" class="grid min-h-0 overflow-hidden pr-2" :style="gridStyle" :aria-label="t('common.loading')">
      <Skeleton v-for="n in 12" :key="n" class="aspect-[2/3] w-full max-w-[var(--movie-card-max-width)] justify-self-center rounded-[1.2rem]" />
    </div>
    <div v-else-if="error" role="alert" class="flex flex-wrap items-center gap-3">
      <p>{{ t('wishlist.loadFailed') }}</p>
      <Button variant="outline" class="min-h-11 rounded-full lg:min-h-8" @click="load()">{{ t('wishlist.reload') }}</Button>
    </div>
    <MediaEmptyState v-else-if="!page.items.length" :filtered="hasConstraints" :description="t('wishlist.emptyHint')">
      <template v-if="hasConstraints" #default>
        <Button variant="outline" class="min-h-11 rounded-full sm:min-h-8" @click="clearFilters">{{ t('bookBrowser.clearFilters') }}</Button>
      </template>
    </MediaEmptyState>
    <WishlistGrid v-else :key="route.fullPath" class="min-h-0 flex-1" :items="page.items" :scroll-preserve-key="route.fullPath" @open="open" />
    <div class="mt-auto flex flex-wrap items-center justify-end gap-2">
      <span class="text-xs text-muted-foreground">{{ t('wishlist.total', { count: page.total }) }}</span>
      <Button v-if="route.query.cursor" variant="outline" class="min-h-11 rounded-full lg:min-h-8" @click="router.replace({ query: { ...route.query, cursor: undefined } })">{{ t('wishlist.firstPage') }}</Button>
      <Button v-if="page.nextCursor" variant="outline" class="min-h-11 rounded-full lg:min-h-8" @click="nextPage">{{ t('wishlist.next') }}</Button>
    </div>
  </section>
</template>

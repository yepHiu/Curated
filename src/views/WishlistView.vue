<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { useElementSize } from "@vueuse/core"
import { RecycleScroller } from "vue-virtual-scroller"
import { useLibraryService } from "@/services/library-service"
import type { WishlistItem, WishlistPage } from "@/domain/wishlist/types"
import WishlistCard from "@/components/jav-library/wishlist/WishlistCard.vue"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription } from "@/components/ui/empty"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
const { t } = useI18n()
const route = useRoute(), router = useRouter(), service = useLibraryService().wishlist
const page = ref<WishlistPage>({ items: [], total: 0, pendingCount: 0 })
const loading = ref(true), error = ref(false), draft = ref(String(route.query.q ?? ""))
const container = ref<HTMLElement | null>(null), scroller = ref<InstanceType<typeof RecycleScroller> | null>(null)
const { width } = useElementSize(container)
const columns = computed(() => { /* 与容器宽度一起调整海报密度。 */ return Math.max(2, Math.floor(width.value / 165)) })
const rowHeight = computed(() => { /* 2:3 海报加底部文字和间距。 */ return Math.ceil((width.value / columns.value - 16) * 1.5 + 76) })
const rows = computed(() => { /* 以行虚拟化，避免加载页数改变 DOM 规模。 */ const out: { id: string; items: WishlistItem[] }[] = []; for (let i = 0; i < page.value.items.length; i += columns.value) out.push({ id: page.value.items[i]!.id, items: page.value.items.slice(i, i + columns.value) }); return out })
const status = computed(() => { /* 从 URL 保留用户筛选。 */ return String(route.query.status ?? "pending") })
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
function open(id: string) { const element = scroller.value?.$el as HTMLElement | undefined; sessionStorage.setItem(`wishlist-scroll:${route.fullPath}`, String(element?.scrollTop ?? 0)); void router.push({ name: "wishlist-detail", params: { id }, query: { back: route.fullPath } }) }
/** 切换游标分页，不累积整个库的图像。 */
function nextPage() { void router.push({ query: { ...route.query, cursor: page.value.nextCursor } }) }
/** 页面恢复前台时重新同步入库状态。 */
function visible() { if (!document.hidden) void load(true) }
watch(() => route.fullPath, () => { /* 路由筛选控制唯一加载入口。 */ void load() }, { immediate: true })
onMounted(() => { /* 注册前台刷新和轮询。 */ timer = setTimeout(poll, 6000); document.addEventListener("visibilitychange", visible) })
watch(scroller, (value) => { /* 虚拟容器就绪后恢复返回位置。 */ if (value?.$el) (value.$el as HTMLElement).scrollTop = Number(sessionStorage.getItem(`wishlist-scroll:${route.fullPath}`) ?? 0) })
onBeforeUnmount(() => { /* 卸载后禁止迟到写入或重新排队。 */ alive = false; generation++; clearTimeout(timer); clearTimeout(searchTimer); document.removeEventListener("visibilitychange", visible) })
</script>
<template>
  <section ref="container" class="flex h-full min-h-0 flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2"><h1 class="font-curated text-xl font-semibold">{{ t('wishlist.title') }}</h1><Badge variant="secondary">{{ page.pendingCount }}</Badge></div>
      <Button variant="outline" @click="router.push({ name: 'settings', query: { section: 'network' } })">{{ t('wishlist.connect') }}</Button>
    </div>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <Tabs :model-value="status" :aria-label="t('wishlist.filter')" @update:model-value="setFilter"><TabsList><TabsTrigger v-for="filter in ['pending', 'in_library', 'completed', 'all']" :key="filter" :value="filter">{{ t(`wishlist.filters.${filter}`) }}</TabsTrigger></TabsList></Tabs>
      <Input v-model="draft" class="max-w-sm" :placeholder="t('wishlist.search')" :aria-label="t('wishlist.search')" @input="search" />
    </div>
    <div v-if="loading" class="grid grid-cols-2 gap-4 md:grid-cols-5"><Skeleton v-for="n in 5" :key="n" class="aspect-[2/3]" /></div>
    <div v-else-if="error" role="alert" class="flex items-center gap-3"><p>{{ t('wishlist.loadFailed') }}</p><Button variant="outline" @click="load()">{{ t('wishlist.retry') }}</Button></div>
    <Empty v-else-if="!page.items.length"><EmptyHeader><EmptyTitle>{{ t('wishlist.empty') }}</EmptyTitle><EmptyDescription>{{ t(service.integrationsAvailable ? 'wishlist.emptyHint' : 'wishlist.mockHint') }}</EmptyDescription></EmptyHeader></Empty>
    <RecycleScroller v-else ref="scroller" class="min-h-0 flex-1" :items="rows" :item-size="rowHeight" key-field="id">
      <template #default="{ item: row }"><div class="grid gap-4 pb-4" :style="{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }"><WishlistCard v-for="item in (row as { items: WishlistItem[] }).items" :key="item.id" :item="item" :image="item.assets[0] ? service.assetUrl(item.assets[0].thumbnailUrl) : undefined" @open="open" /></div></template>
    </RecycleScroller>
    <div class="flex items-center justify-end gap-2"><span class="text-sm text-muted-foreground">{{ t('wishlist.total', { count: page.total }) }}</span><Button v-if="route.query.cursor" variant="outline" @click="router.replace({ query: { ...route.query, cursor: undefined } })">{{ t('wishlist.firstPage') }}</Button><Button v-if="page.nextCursor" variant="outline" @click="nextPage">{{ t('wishlist.next') }}</Button></div>
  </section>
</template>

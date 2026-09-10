<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useElementSize } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { ChevronLeft, ChevronRight, Ellipsis } from "lucide-vue-next"
import { PaginationRoot, PaginationList, PaginationListItem, PaginationEllipsis, PaginationPrev, PaginationNext } from "reka-ui"
import { Button } from "@/components/ui/button"
import { previewColumns, previewStart } from "@/lib/book-preview"
import BookPreviewTile from "./BookPreviewTile.vue"
const props = defineProps<{ bookId: string; pages: readonly { index: number; thumbUrl?: string; imageUrl?: string }[]; total: number; kind: "comics" | "photos" }>()
const emit = defineEmits<{ open: [index: number] }>()
const { t } = useI18n()
const container = ref<HTMLElement | null>(null)
const { width } = useElementSize(container)
const columns = computed(() => previewColumns(width.value))
const size = computed(() => columns.value * 2)
const anchor = ref(0)
watch(() => props.bookId, () => { anchor.value = 0 }, { immediate: true })
const start = computed(() => previewStart(anchor.value, size.value, props.pages.length))
const pages = computed(() => props.pages.slice(start.value, start.value + size.value))
const currentBatch = computed(() => Math.floor(start.value / size.value) + 1)
function selectBatch(batch: number) {
  anchor.value = previewStart((batch - 1) * size.value, size.value, props.pages.length)
}
</script>
<template>
  <section ref="container" data-book-page-preview class="flex min-w-0 flex-col gap-4 rounded-3xl border border-border/70 bg-card/85 p-5 sm:p-6">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 class="text-lg font-semibold">{{ t(`${kind}.previewTitle`) }}</h2>
      <p class="text-xs tabular-nums text-muted-foreground" aria-live="polite">{{ t('bookBrowser.previewRange', { start: pages.length ? pages[0]!.index + 1 : 0, end: pages.length ? pages[pages.length - 1]!.index + 1 : 0, total }) }}</p>
    </div>
    <div v-if="pages.length" data-book-preview-grid class="grid gap-3" :style="{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }">
      <BookPreviewTile v-for="page in pages" :key="`${bookId}:${page.index}`" :src="page.thumbUrl || page.imageUrl"
        :label="t(`${kind}.previewPageAlt`, { page: page.index + 1 })" :page-number="page.index + 1" :retry-label="t('bookBrowser.retryImage')"
        :data-comic-page-preview="kind === 'comics' ? '' : undefined" :data-photo-page-preview="kind === 'photos' ? '' : undefined"
        @open="emit('open', page.index)" />
    </div>
    <p v-else class="py-6 text-sm text-muted-foreground">{{ t(`${kind}.previewEmpty`) }}</p>
    <PaginationRoot
      v-if="pages.length"
      :page="currentBatch"
      :items-per-page="size"
      :total="props.pages.length"
      :sibling-count="width < 480 ? 0 : 1"
      show-edges
      :aria-label="t('bookBrowser.previewPagination')"
      class="flex flex-wrap items-center justify-center gap-2"
      @update:page="selectBatch"
    >
      <PaginationPrev as-child :aria-label="t('bookBrowser.previousBatch')">
        <Button data-preview-previous variant="outline" class="min-h-11 rounded-full">
          <ChevronLeft data-icon="inline-start" aria-hidden="true" />{{ t('bookBrowser.previousBatch') }}
        </Button>
      </PaginationPrev>
      <PaginationList v-slot="{ items }" class="order-last flex w-full flex-wrap items-center justify-center gap-1 sm:order-none sm:w-auto">
        <template v-for="(item, index) in items" :key="item.type === 'page' ? item.value : `ellipsis-${index}`">
          <PaginationListItem v-if="item.type === 'page'" :value="item.value" as-child :aria-label="t('bookBrowser.batchNumber', { batch: item.value })">
            <Button :data-preview-batch="item.value" :variant="item.value === currentBatch ? 'default' : 'outline'" class="min-h-11 min-w-11 rounded-full px-3 tabular-nums">
              {{ item.value }}
            </Button>
          </PaginationListItem>
          <PaginationEllipsis v-else class="flex size-6 shrink-0 items-center justify-center text-muted-foreground">
            <Ellipsis class="size-4" aria-hidden="true" />
            <span class="sr-only">{{ t('bookBrowser.moreBatches') }}</span>
          </PaginationEllipsis>
        </template>
      </PaginationList>
      <PaginationNext as-child :aria-label="t('bookBrowser.nextBatch')">
        <Button data-preview-next variant="outline" class="min-h-11 rounded-full">
          {{ t('bookBrowser.nextBatch') }}<ChevronRight data-icon="inline-end" aria-hidden="true" />
        </Button>
      </PaginationNext>
    </PaginationRoot>
  </section>
</template>

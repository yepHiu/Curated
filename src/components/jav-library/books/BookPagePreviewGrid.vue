<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useElementSize } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { ChevronLeft, ChevronRight } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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
const jump = ref<string | number>(1)
watch(() => props.bookId, () => { anchor.value = 0; jump.value = 1 }, { immediate: true })
const start = computed(() => previewStart(anchor.value, size.value, props.pages.length))
const pages = computed(() => props.pages.slice(start.value, start.value + size.value))
const end = computed(() => Math.min(props.pages.length, start.value + size.value))
const jumpValid = computed(() => Number.isInteger(Number(jump.value)) && Number(jump.value) >= 1 && Number(jump.value) <= props.total)
function move(delta: number) {
  anchor.value = previewStart(start.value + delta * size.value, size.value, props.pages.length)
  jump.value = (props.pages[anchor.value]?.index ?? 0) + 1
}
function locate() {
  if (!jumpValid.value) return
  const position = props.pages.findIndex(page => page.index === Number(jump.value) - 1)
  if (position >= 0) anchor.value = position
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
        :label="t(`${kind}.previewPageAlt`, { page: page.index + 1 })" :page-number="page.index + 1" :current="page.index === Number(jump) - 1" :retry-label="t('bookBrowser.retryImage')"
        :data-comic-page-preview="kind === 'comics' ? '' : undefined" :data-photo-page-preview="kind === 'photos' ? '' : undefined"
        @open="emit('open', page.index)" />
    </div>
    <p v-else class="py-6 text-sm text-muted-foreground">{{ t(`${kind}.previewEmpty`) }}</p>
    <div v-if="pages.length" class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <Button data-preview-previous variant="outline" class="min-h-11 rounded-full" :disabled="start === 0" :aria-label="t('bookBrowser.previousBatch')" @click="move(-1)"><ChevronLeft data-icon="inline-start" />{{ t('bookBrowser.previousBatch') }}</Button>
        <Button data-preview-next variant="outline" class="min-h-11 rounded-full" :disabled="end >= props.pages.length" :aria-label="t('bookBrowser.nextBatch')" @click="move(1)">{{ t('bookBrowser.nextBatch') }}<ChevronRight data-icon="inline-end" /></Button>
      </div>
      <form class="flex min-w-0 items-center gap-2" @submit.prevent="locate">
        <Input v-model="jump" data-preview-jump type="number" min="1" :max="total" step="1" class="min-h-11 w-20" :aria-label="t('bookBrowser.pageNumber')" />
        <Button type="submit" variant="outline" class="min-h-11 rounded-full" :disabled="!jumpValid">{{ t('bookBrowser.locatePage') }}</Button>
        <Button type="button" variant="ghost" class="min-h-11 rounded-full" :disabled="!jumpValid" @click="emit('open', Number(jump) - 1)">{{ t('bookBrowser.openPage') }}</Button>
      </form>
    </div>
  </section>
</template>

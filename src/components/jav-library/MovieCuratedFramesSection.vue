<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { ChevronLeft, ChevronRight } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog"
import MediaStill from "@/components/jav-library/MediaStill.vue"
import FrameImageViewer from "@/components/jav-library/FrameImageViewer.vue"
import { listCuratedFramesPage, type CuratedFrameDbRow } from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import { curatedFrameImageUrl, curatedFrameThumbnailUrl } from "@/lib/curated-frame-image-url"
import { formatTimecodeLabel } from "@/lib/player-playback-stats-format"

const props = defineProps<{ movieId: string }>()
const { t } = useI18n()
const pageSize = 12
type FrameEntry = { row: CuratedFrameDbRow; thumbnail: string; original: string }
const entries = ref<FrameEntry[]>([])
const total = ref(0)
const loading = ref(false)
const failed = ref(false)
const viewerOpen = ref(false)
const selectedIndex = ref(0)
const selected = computed(() => entries.value[selectedIndex.value])
const hasMore = computed(() => entries.value.length < total.value)
let nextCursor: string | undefined
let offset = 0
let generation = 0

function releaseImages() {
  for (const entry of entries.value) {
    if (entry.original.startsWith("blob:")) URL.revokeObjectURL(entry.original)
  }
  entries.value = []
}

async function loadMore() {
  if (loading.value || !props.movieId.trim()) return
  const requestGeneration = generation
  loading.value = true
  failed.value = false
  try {
    const page = await listCuratedFramesPage({
      movieId: props.movieId,
      limit: pageSize,
      offset,
      cursor: nextCursor,
      skipTotal: Boolean(nextCursor),
    })
    if (requestGeneration !== generation) return
    const known = new Set(entries.value.map((entry) => entry.row.id))
    const added: FrameEntry[] = []
    for (const row of page.items) {
      if (known.has(row.id)) continue
      known.add(row.id)
      const original = row.imageBlob ? URL.createObjectURL(row.imageBlob) : curatedFrameImageUrl(row.id)
      added.push({ row, original, thumbnail: row.imageBlob ? original : curatedFrameThumbnailUrl(row.id) })
    }
    entries.value = [...entries.value, ...added]
    offset += page.items.length
    nextCursor = page.nextCursor
    if (page.total >= 0) total.value = page.total
    if (page.items.length === 0) total.value = entries.value.length
  } catch {
    if (requestGeneration === generation) failed.value = true
  } finally {
    if (requestGeneration === generation) loading.value = false
  }
}

watch([() => props.movieId, curatedFramesRevision], () => {
  generation++
  viewerOpen.value = false
  selectedIndex.value = 0
  releaseImages()
  total.value = 0
  offset = 0
  nextCursor = undefined
  loading.value = false
  failed.value = false
  void loadMore()
}, { immediate: true })

onBeforeUnmount(() => {
  generation++
  releaseImages()
})

function openFrame(index: number) {
  selectedIndex.value = index
  viewerOpen.value = true
}

function moveFrame(delta: number) {
  selectedIndex.value = Math.max(0, Math.min(entries.value.length - 1, selectedIndex.value + delta))
}
</script>

<template>
  <Card data-movie-curated-frames class="min-w-0 rounded-3xl border-border/70 bg-card/85">
    <CardHeader class="flex flex-wrap items-center justify-between gap-2">
      <CardTitle>{{ t("curated.title") }}</CardTitle>
      <span v-if="total > 0" class="text-xs text-muted-foreground">
        {{ t("curated.pageSummary", { shown: entries.length, total }) }}
      </span>
    </CardHeader>
    <CardContent class="space-y-4" :aria-busy="loading">
      <div v-if="entries.length" class="grid grid-cols-[repeat(auto-fill,minmax(min(100%,15rem),1fr))] gap-3">
        <button
          v-for="(entry, index) in entries"
          :key="entry.row.id"
          type="button"
          :data-movie-frame="entry.row.id"
          :aria-label="t('detailPage.curatedFrameOpen', { time: formatTimecodeLabel(entry.row.positionSec) })"
          class="min-w-0 overflow-hidden rounded-2xl border border-border/70 bg-muted/30 text-left transition-colors hover:border-primary/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          @click="openFrame(index)"
        >
          <div class="relative aspect-video">
            <MediaStill :src="entry.thumbnail" :alt="entry.row.code" fit="contain" />
          </div>
          <p class="px-3 py-2 text-xs tabular-nums text-muted-foreground">{{ formatTimecodeLabel(entry.row.positionSec) }}</p>
        </button>
      </div>
      <p v-if="loading" role="status" class="text-sm text-muted-foreground">{{ t("common.loading") }}</p>
      <p v-else-if="!failed && !entries.length" class="text-sm text-muted-foreground">{{ t("detailPage.curatedFramesEmpty") }}</p>
      <div v-if="failed" class="flex flex-wrap items-center gap-3" role="alert">
        <p class="text-sm text-destructive">{{ t("detailPage.curatedFramesLoadError") }}</p>
        <Button variant="outline" @click="loadMore">{{ t("curated.retryLoad") }}</Button>
      </div>
      <Button v-else-if="hasMore" variant="outline" :disabled="loading" @click="loadMore">{{ t("curated.loadMore") }}</Button>
    </CardContent>
  </Card>

  <Dialog v-model:open="viewerOpen">
    <DialogContent
      class="flex h-[min(90dvh,60rem)] w-[94vw] max-w-[94vw] flex-col gap-3 overflow-hidden sm:max-w-[min(94vw,90rem)]"
    >
      <div
        data-movie-frame-viewer
        class="flex h-full min-h-0 flex-col gap-3"
        @keydown.left.stop.prevent="moveFrame(-1)"
        @keydown.right.stop.prevent="moveFrame(1)"
      >
        <DialogTitle class="pr-8">{{ t("curated.title") }} · {{ selected?.row.code }}</DialogTitle>
        <DialogDescription class="sr-only">{{ t("detailPage.curatedFrameOpen", { time: formatTimecodeLabel(selected?.row.positionSec) }) }}</DialogDescription>
        <FrameImageViewer
          v-if="viewerOpen && selected"
          :src="selected.original"
          :alt="`${selected.row.code} · ${formatTimecodeLabel(selected.row.positionSec)}`"
          class="min-h-0 flex-1"
        />
        <div class="flex shrink-0 items-center justify-center gap-4">
          <Button variant="outline" size="icon" class="size-11" :disabled="selectedIndex <= 0" :aria-label="t('curated.previousFrame')" @click="moveFrame(-1)">
            <ChevronLeft class="size-4" aria-hidden="true" />
          </Button>
          <span aria-live="polite" class="text-sm tabular-nums text-muted-foreground">{{ formatTimecodeLabel(selected?.row.positionSec) }} · {{ selectedIndex + 1 }} / {{ entries.length }}</span>
          <Button variant="outline" size="icon" class="size-11" :disabled="selectedIndex >= entries.length - 1" :aria-label="t('curated.nextFrame')" @click="moveFrame(1)">
            <ChevronRight class="size-4" aria-hidden="true" />
          </Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>

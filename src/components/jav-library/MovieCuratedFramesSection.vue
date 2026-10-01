<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import CuratedFrameCard from "@/components/jav-library/CuratedFrameCard.vue"
import CuratedFrameDetailDialog from "@/components/jav-library/CuratedFrameDetailDialog.vue"
import { listCuratedFramesPage } from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import { curatedFrameThumbnailUrl } from "@/lib/curated-frame-image-url"
import type { CuratedFrameDialogItem } from "@/lib/curated-frames/dialog-navigation"
import { buildCuratedFrameNearDuplicateIndex, findCuratedFrameNearDuplicateGroups } from "@/lib/curated-frames/near-duplicates"
import { formatTimecodeLabel } from "@/lib/player-playback-stats-format"

const props = defineProps<{ movieId: string }>()
const { t } = useI18n()
const pageSize = 12
const frameDialogRef = ref<InstanceType<typeof CuratedFrameDetailDialog> | null>(null)
const entries = ref<CuratedFrameDialogItem[]>([])
const total = ref(0)
const loading = ref(false)
const failed = ref(false)
const viewerOpen = ref(false)
let reloadOnClose = false
const dialogEntries = computed(() => entries.value.map((item) => ({ item, sectionActor: null })))
const nearDuplicateIds = computed(() => [...buildCuratedFrameNearDuplicateIndex(findCuratedFrameNearDuplicateGroups(entries.value.map((entry) => entry.row), 3))])
const hasMore = computed(() => entries.value.length < total.value)
let nextCursor: string | undefined
let offset = 0
let generation = 0

function releaseImages() {
  for (const entry of entries.value) {
    if (entry.url.startsWith("blob:")) URL.revokeObjectURL(entry.url)
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
    const added: CuratedFrameDialogItem[] = []
    for (const row of page.items) {
      if (known.has(row.id)) continue
      known.add(row.id)
      const url = row.imageBlob ? URL.createObjectURL(row.imageBlob) : curatedFrameThumbnailUrl(row.id)
      added.push({ row, url })
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

function reload() {
  generation++
  viewerOpen.value = false
  reloadOnClose = false
  releaseImages()
  total.value = 0
  offset = 0
  nextCursor = undefined
  loading.value = false
  failed.value = false
  void loadMore()
}

watch(() => props.movieId, reload, { immediate: true })
watch(curatedFramesRevision, () => {
  // Tag autosave also increments the revision. Keep the active dialog and its URLs alive.
  if (viewerOpen.value) reloadOnClose = true
  else reload()
})
watch(viewerOpen, (open) => {
  if (!open && reloadOnClose) reload()
})

onBeforeUnmount(() => {
  generation++
  releaseImages()
})

function applyFrameTags({ id, tags }: { id: string; tags: string[] }) {
  entries.value = entries.value.map((entry) => entry.row.id === id
    ? { ...entry, row: { ...entry.row, tags: [...tags] } }
    : entry)
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
        <CuratedFrameCard
          v-for="entry in entries"
          :key="entry.row.id"
          :data-movie-frame="entry.row.id"
          :row="entry.row"
          :image-url="entry.url"
          :position-label="formatTimecodeLabel(entry.row.positionSec)"
          :batch-mode="false"
          :selected="false"
          :near-duplicate="nearDuplicateIds.includes(entry.row.id)"
          @open="frameDialogRef?.open(entry)"
        />
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

  <CuratedFrameDetailDialog
    :key="movieId"
    ref="frameDialogRef"
    :entries="dialogEntries"
    :near-duplicate-ids="nearDuplicateIds"
    @update:open="viewerOpen = $event"
    @tags-saved="applyFrameTags"
  />
</template>

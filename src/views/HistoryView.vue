<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { RouterLink, useRouter } from "vue-router"
import { CheckSquare, ListChecks, Trash2, X } from "lucide-vue-next"
import PlaybackHistoryCard from "@/components/jav-library/PlaybackHistoryCard.vue"
import { HttpClientError } from "@/api/http-client"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { pushAppToast } from "@/composables/use-app-toast"
import type { Movie } from "@/domain/movie/types"
import { groupPlaybackRowsByLocalDay } from "@/lib/playback-history-groups"
import { buildPlayerRouteFromHistory } from "@/lib/player-route"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"
import {
  listSortedByUpdatedDesc,
  playbackProgressRevision,
  removeProgress,
} from "@/lib/playback-progress-storage"
import { useLibraryService } from "@/services/library-service"

const { t, locale } = useI18n()
const router = useRouter()
const libraryService = useLibraryService()

interface HistoryRow {
  entry: PlaybackProgressEntry
  movie: Movie
  updatedAt: string
}

const removeDialogOpen = ref(false)
const removing = ref(false)
const pendingRemoveRow = ref<HistoryRow | null>(null)
const batchMode = ref(false)
const batchBusy = ref(false)
const batchRemoveDialogOpen = ref(false)
const batchSelectedIds = ref<Set<string>>(new Set())

const historyRows = computed((): HistoryRow[] => {
  void playbackProgressRevision.value
  const sorted = listSortedByUpdatedDesc()
  const out: HistoryRow[] = []
  for (const entry of sorted) {
    const movie = libraryService.getMovieById(entry.movieId)
    if (!movie) continue
    out.push({ entry, movie, updatedAt: entry.updatedAt })
  }
  return out
})

const dayBuckets = computed(() => {
  const loc = locale.value
  return groupPlaybackRowsByLocalDay(historyRows.value, {
    locale: loc,
    labels: {
      today: t("history.today"),
      yesterday: t("history.yesterday"),
    },
  })
})

const isEmpty = computed(() => historyRows.value.length === 0)
const batchSelectedCount = computed(() => batchSelectedIds.value.size)

async function openFromHistory(row: HistoryRow) {
  if (batchMode.value) {
    toggleBatchSelect(row.entry.movieId)
    return
  }
  const pos = Math.max(0, Math.floor(row.entry.positionSec))
  await router.push(buildPlayerRouteFromHistory(row.movie.id, pos))
}

function requestRemoveRow(row: HistoryRow) {
  pendingRemoveRow.value = row
  removeDialogOpen.value = true
}

async function confirmRemoveRow() {
  const row = pendingRemoveRow.value
  if (!row || removing.value) return
  removing.value = true
  try {
    await removeProgress(row.entry.movieId)
    removeDialogOpen.value = false
    pendingRemoveRow.value = null
    pushAppToast(t("history.deleteSuccess"), { variant: "success", durationMs: 3200 })
  } catch (err) {
    const message =
      err instanceof HttpClientError && err.apiError?.message
        ? err.apiError.message
        : err instanceof Error && err.message
          ? err.message
          : t("history.deleteFailed")
    pushAppToast(message, { variant: "destructive" })
  } finally {
    removing.value = false
  }
}

function clearBatchSelection() {
  batchSelectedIds.value = new Set()
}

function enterBatchMode() {
  batchMode.value = true
}

function exitBatchMode() {
  batchMode.value = false
  clearBatchSelection()
}

function toggleBatchSelect(movieId: string) {
  const id = movieId.trim()
  if (!id) return
  const next = new Set(batchSelectedIds.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  batchSelectedIds.value = next
}

function selectAllVisibleInBatch() {
  batchSelectedIds.value = new Set(historyRows.value.map((row) => row.entry.movieId))
}

function requestBatchRemove() {
  if (batchSelectedCount.value <= 0 || batchBusy.value) return
  batchRemoveDialogOpen.value = true
}

async function confirmBatchRemove() {
  const ids = [...batchSelectedIds.value]
  if (ids.length === 0 || batchBusy.value) return
  batchBusy.value = true
  let fail = 0
  try {
    for (const id of ids) {
      try {
        await removeProgress(id)
      } catch {
        fail++
      }
    }
    batchRemoveDialogOpen.value = false
    clearBatchSelection()
    exitBatchMode()
    pushAppToast(t("history.batchDeleteSummary", { ok: ids.length - fail, fail }), {
      variant: fail === ids.length ? "destructive" : fail > 0 ? "warning" : "success",
    })
  } finally {
    batchBusy.value = false
  }
}

watch(
  historyRows,
  (rows) => {
    const allowed = new Set(rows.map((row) => row.entry.movieId))
    const next = new Set([...batchSelectedIds.value].filter((id) => allowed.has(id)))
    if (next.size !== batchSelectedIds.value.size) {
      batchSelectedIds.value = next
    }
    if (batchMode.value && rows.length === 0) {
      exitBatchMode()
    }
  },
  { deep: false },
)
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
    <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-y-auto">
      <div
        class="mx-auto flex w-full max-w-4xl flex-col gap-6 px-3 pb-6 sm:px-6 lg:px-8"
      >
        <header class="flex flex-col gap-3">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <h1 class="text-2xl font-semibold tracking-tight">{{ t("history.title") }}</h1>
              <p class="text-sm text-muted-foreground">
                {{ t("history.subtitle") }}
              </p>
            </div>
            <div
              v-if="!isEmpty"
              class="flex shrink-0 flex-wrap items-center gap-2"
            >
              <template v-if="!batchMode">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="shrink-0 gap-1.5 rounded-xl"
                  @click="enterBatchMode"
                >
                  <ListChecks class="size-4 opacity-80" aria-hidden="true" />
                  {{ t("history.batchManage") }}
                </Button>
              </template>
              <template v-else>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="shrink-0 gap-1.5 rounded-xl"
                  @click="selectAllVisibleInBatch"
                >
                  <CheckSquare class="size-4 opacity-80" aria-hidden="true" />
                  {{ t("history.batchSelectVisible") }}
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  class="shrink-0 gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground"
                  @click="exitBatchMode"
                >
                  <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
                  {{ t("history.batchExitToolbar") }}
                </Button>
              </template>
            </div>
          </div>
        </header>

    <div
      v-if="isEmpty"
      class="flex flex-col items-center justify-center gap-4 rounded-3xl border border-dashed border-border/70 bg-card/50 px-6 py-16 text-center"
    >
      <p class="max-w-sm text-sm text-muted-foreground">
        {{ t("history.empty") }}
      </p>
      <Button as-child variant="secondary" class="rounded-2xl">
        <RouterLink :to="{ name: 'library' }">{{ t("history.goLibrary") }}</RouterLink>
      </Button>
    </div>

    <template v-else>
      <section
        v-for="bucket in dayBuckets"
        :key="bucket.dayKey"
        class="flex flex-col gap-3"
      >
        <h2
          class="sticky top-0 z-[1] bg-background/90 px-1 py-2 text-xs font-semibold tracking-wider text-muted-foreground uppercase backdrop-blur-sm"
        >
          {{ bucket.label }}
        </h2>
        <!-- 单列瀑布流：按时间自上而下逐条排列，卡片占满内容区宽度 -->
        <div class="flex w-full flex-col gap-4">
          <PlaybackHistoryCard
            v-for="row in bucket.rows"
            :key="row.entry.movieId + row.entry.updatedAt"
            :movie="row.movie"
            :entry="row.entry"
            :batch-mode="batchMode"
            :selected="batchSelectedIds.has(row.entry.movieId)"
            @click="openFromHistory(row)"
            @remove="requestRemoveRow(row)"
            @toggle-select="toggleBatchSelect(row.entry.movieId)"
          />
        </div>
      </section>
        </template>
      </div>
    </div>

    <Dialog v-model:open="removeDialogOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t("history.deleteTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{
              t("history.deleteConfirm", {
                title: pendingRemoveRow?.movie.title ?? t("history.deleteLabelFallback"),
              })
            }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl" :disabled="removing">
              {{ t("common.cancel") }}
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            class="rounded-2xl"
            :disabled="removing"
            @click="confirmRemoveRow"
          >
            {{ removing ? t("history.deleting") : t("history.deleteAction") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="batchRemoveDialogOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t("history.batchDeleteTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("history.batchDeleteConfirm", { n: batchSelectedCount }) }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl" :disabled="batchBusy">
              {{ t("common.cancel") }}
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            class="rounded-2xl"
            :disabled="batchBusy"
            @click="confirmBatchRemove"
          >
            {{ batchBusy ? t("history.deleting") : t("history.batchDeleteAction") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <div
      v-if="batchMode"
      role="toolbar"
      :aria-label="t('history.batchToolbarAria')"
      class="w-full shrink-0 overflow-hidden border-t border-border/70 bg-card/95 px-3 py-3 shadow-[0_-8px_30px_rgba(0,0,0,0.12)] backdrop-blur-md sm:px-4 rounded-b-[calc(1.75rem-1rem)] sm:rounded-b-[calc(1.75rem-1.25rem)] lg:rounded-b-[calc(1.75rem-1.5rem)] xl:rounded-b-none"
      style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom))"
    >
      <div
        class="mx-auto flex w-full max-w-4xl flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between"
      >
        <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-muted-foreground">
          <span class="font-medium text-foreground">
            {{ t("history.batchSelected", { n: batchSelectedCount }) }}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="h-8 rounded-lg px-2"
            :disabled="batchSelectedCount === 0 || batchBusy"
            @click="clearBatchSelection"
          >
            {{ t("history.batchClearSelection") }}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="h-8 rounded-lg px-2"
            :disabled="batchBusy"
            @click="selectAllVisibleInBatch"
          >
            {{ t("history.batchSelectVisible") }}
          </Button>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="gap-1.5 rounded-xl text-destructive hover:bg-destructive/10"
            :disabled="batchSelectedCount <= 0 || batchBusy"
            @click="requestBatchRemove"
          >
            <Trash2 class="size-4" aria-hidden="true" />
            {{ t("history.batchDeleteAction") }}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground disabled:opacity-40"
            :disabled="batchBusy"
            @click="exitBatchMode"
          >
            <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
            {{ t("library.batchExit") }}
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>

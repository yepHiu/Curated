<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Trash2 } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import type { Movie } from "@/domain/movie/types"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"

const { t } = useI18n()

const props = defineProps<{
  movie: Movie
  entry: PlaybackProgressEntry
  batchMode?: boolean
  selected?: boolean
  showRemoveAction?: boolean
}>()

const emit = defineEmits<{
  click: []
  remove: []
  toggleSelect: []
}>()

function onBatchCheckboxChange() {
  emit("toggleSelect")
}

/** 宽版海报：优先封面图（常为整幅封套/横长图），缩略图多为竖版裁切 */
const posterSrc = computed(() => props.movie.coverUrl || props.movie.thumbUrl)

const useWideCoverCrop = computed(() => Boolean(props.movie.coverUrl))

const progressPercent = computed(() => {
  const dur = props.entry.durationSec
  if (!dur || dur <= 0) return null
  return Math.min(100, Math.max(0, (props.entry.positionSec / dur) * 100))
})

const actorsLine = computed(() => {
  const list = props.movie.actors ?? []
  if (!list.length) return "—"
  return list.slice(0, 4).join(" · ")
})
</script>

<template>
  <div
    class="group relative w-full overflow-hidden rounded-[1.2rem] border border-border/70 bg-card/80 py-0 text-left shadow-md shadow-black/5 transition-[border-color,box-shadow] duration-150 hover:border-primary/25 hover:shadow-lg motion-reduce:transition-none"
    :class="props.batchMode && props.selected ? 'border-primary/40 ring-1 ring-primary/20' : ''"
  >
    <button type="button" class="block w-full p-3 text-left focus-visible:outline-none" @click="emit('click')">
      <div class="flex items-center gap-4">
        <div class="flex min-w-0 flex-1 flex-col justify-center gap-1.5 py-0.5">
          <p class="line-clamp-2 text-sm font-semibold leading-snug sm:text-base">
            {{ movie.title }}
          </p>
          <p class="line-clamp-2 text-xs text-muted-foreground sm:text-sm">
            {{ actorsLine }}
          </p>
          <p class="text-[11px] text-muted-foreground/80 sm:text-xs">
            {{ movie.code }}
          </p>
        </div>

        <div
          class="relative aspect-[2.25/1] w-[min(46%,15.5rem)] shrink-0 overflow-hidden rounded-[0.95rem] border border-border/60 bg-muted/30 sm:aspect-[2.4/1] sm:w-[min(44%,17rem)]"
          :title="t('curated.widePoster')"
        >
          <label
            v-if="props.batchMode"
            class="absolute top-2 right-2 z-[4] flex cursor-pointer items-center justify-center rounded-md border border-border/45 bg-background/25 p-1.5 shadow-sm backdrop-blur-md backdrop-saturate-150 dark:border-white/20 dark:bg-black/30"
            @click.stop
          >
            <input
              type="checkbox"
              class="size-4 cursor-pointer rounded accent-primary"
              :checked="props.selected"
              :aria-label="
                props.selected
                  ? t('history.batchDeselectAria', { title: movie.title })
                  : t('history.batchSelectAria', { title: movie.title })
              "
              @change="onBatchCheckboxChange"
            />
          </label>

          <Button
            v-if="!props.batchMode && props.showRemoveAction !== false"
            type="button"
            variant="ghost"
            size="icon"
            class="absolute top-2 right-2 z-[3] size-8 rounded-md border border-border/45 bg-background/25 text-muted-foreground shadow-sm backdrop-blur-md backdrop-saturate-150 hover:bg-background/40 hover:text-destructive dark:border-white/20 dark:bg-black/30"
            :aria-label="t('history.deleteAria', { title: movie.title })"
            @click.stop="emit('remove')"
          >
            <Trash2 class="size-4" aria-hidden="true" />
          </Button>

          <img
            v-if="posterSrc"
            :src="posterSrc"
            :alt="movie.title"
            class="absolute inset-0 z-[1] size-full object-cover"
            :class="useWideCoverCrop ? 'object-[76%_center] sm:object-right' : 'object-center'"
            loading="lazy"
          />
          <div
            v-else
            class="absolute inset-0 z-[1] flex items-center justify-center bg-gradient-to-br from-primary/25 via-accent/30 to-card text-[10px] text-muted-foreground"
          >
            {{ t("common.noArt") }}
          </div>
          <div
            class="pointer-events-none absolute inset-0 z-[2] bg-gradient-to-t from-black/45 via-transparent to-black/20"
            aria-hidden="true"
          />
          <div
            class="absolute right-0 bottom-0 left-0 z-[2] h-1 bg-black/50"
            aria-hidden="true"
          >
            <div
              class="h-full bg-primary transition-[width] duration-300 motion-reduce:transition-none"
              :style="{ width: progressPercent != null ? `${progressPercent}%` : '0%' }"
            />
          </div>
        </div>
      </div>
    </button>
  </div>
</template>

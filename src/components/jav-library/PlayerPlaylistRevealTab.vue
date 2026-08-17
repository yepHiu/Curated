<script setup lang="ts">
import { ChevronLeft, ChevronRight } from "lucide-vue-next"
import { useI18n } from "vue-i18n"

const { t } = useI18n()

withDefaults(
  defineProps<{
    visible?: boolean
    expanded?: boolean
  }>(),
  {
    visible: false,
    expanded: false,
  },
)

const emit = defineEmits<{
  enter: []
  leave: []
  open: []
  close: []
}>()
</script>

<template>
  <button
    v-if="expanded"
    type="button"
    data-player-playlist-tab
    class="flex h-16 w-6 items-center justify-center rounded-l-md border border-r-0 border-border bg-background text-foreground"
    :aria-label="t('player.playlistCloseAria')"
    @click.stop="emit('close')"
  >
    <ChevronRight class="size-4" aria-hidden="true" />
  </button>
  <div
    v-else
    data-player-playlist-hotzone
    class="absolute inset-y-0 right-0 z-20 flex w-10 items-center justify-end"
    @mouseenter="emit('enter')"
    @mouseleave="emit('leave')"
    @click.stop
  >
    <button
      v-if="visible"
      type="button"
      data-player-playlist-tab
      class="flex h-16 w-6 items-center justify-center rounded-l-md border border-r-0 border-border bg-background text-foreground"
      :aria-label="t('player.playlistOpenAria')"
      @click.stop="emit('open')"
    >
      <ChevronLeft class="size-4" aria-hidden="true" />
    </button>
  </div>
</template>

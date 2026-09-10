<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { PhotoBook } from "@/domain/photo/types"

const props = withDefaults(
  defineProps<{
    photo: PhotoBook
    previewLimit?: number
  }>(),
  {
    previewLimit: 12,
  },
)

const emit = defineEmits<{
  openViewer: [pageIndex: number]
}>()

const { t } = useI18n()

const previewPages = computed(() => (props.photo.pages ?? []).slice(0, props.previewLimit))
</script>

<template>
  <section class="flex min-w-0 flex-col gap-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 class="text-lg font-semibold tracking-normal">{{ t("photos.previewTitle") }}</h2>
      <p class="text-sm text-muted-foreground">
        {{ t("photos.previewCount", { count: previewPages.length, total: photo.pageCount }) }}
      </p>
    </div>

    <div
      v-if="previewPages.length"
      class="grid grid-cols-[repeat(auto-fill,minmax(5.5rem,1fr))] gap-3 sm:grid-cols-[repeat(auto-fill,minmax(7rem,1fr))]"
    >
      <button
        v-for="page in previewPages"
        :key="page.index"
        type="button"
        data-photo-page-preview
        class="group aspect-[2/3] min-w-0 overflow-hidden rounded-lg bg-muted text-left transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
        @click="emit('openViewer', page.index)"
      >
        <img
          v-if="page.thumbUrl || page.imageUrl"
          :src="page.thumbUrl || page.imageUrl"
          :alt="t('photos.previewPageAlt', { page: page.index + 1 })"
          class="h-full w-full object-cover"
          loading="lazy"
        >
      </button>
    </div>

    <p
      v-else
      class="rounded-xl border border-dashed border-border/70 bg-muted/20 px-4 py-6 text-sm text-muted-foreground"
    >
      {{ t("photos.previewEmpty") }}
    </p>
  </section>
</template>

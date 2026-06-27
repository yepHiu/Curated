<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { ComicBook } from "@/domain/comic/types"

const props = withDefaults(
  defineProps<{
    comic: ComicBook
    previewLimit?: number
  }>(),
  {
    previewLimit: 12,
  },
)

const emit = defineEmits<{
  openReader: [pageIndex: number]
}>()

const { t } = useI18n()

const previewPages = computed(() => (props.comic.pages ?? []).slice(0, props.previewLimit))
</script>

<template>
  <section class="flex min-w-0 flex-col gap-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 class="text-lg font-semibold tracking-normal">{{ t("comics.previewTitle") }}</h2>
      <p class="text-sm text-muted-foreground">
        {{ t("comics.previewCount", { count: previewPages.length, total: comic.pageCount }) }}
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
        data-comic-page-preview
        class="group flex min-w-0 flex-col gap-1.5 rounded-xl border border-border/70 bg-card/70 p-1.5 text-left transition-colors hover:border-primary/35 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
        @click="emit('openReader', page.index)"
      >
        <span class="aspect-[2/3] overflow-hidden rounded-lg bg-muted">
          <img
            v-if="page.thumbUrl || page.imageUrl"
            :src="page.thumbUrl || page.imageUrl"
            :alt="t('comics.previewPageAlt', { page: page.index + 1 })"
            class="h-full w-full object-cover"
            loading="lazy"
          >
        </span>
        <span class="px-1 text-xs tabular-nums text-muted-foreground">
          {{ t("comics.previewPage", { page: page.index + 1 }) }}
        </span>
      </button>
    </div>

    <p v-else class="rounded-xl border border-dashed border-border/70 bg-muted/20 px-4 py-6 text-sm text-muted-foreground">
      {{ t("comics.previewEmpty") }}
    </p>
  </section>
</template>

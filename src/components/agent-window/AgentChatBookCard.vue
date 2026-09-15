<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { AIAgentBookCardDTO } from "@/api/types"
import { resolveMediaUrl } from "@/api/media-url"

const props = defineProps<{
  book: AIAgentBookCardDTO
}>()

const emit = defineEmits<{
  open: [book: AIAgentBookCardDTO]
}>()

const { t } = useI18n()

/** 把同源封面地址解析成可展示的图片 URL。 */
const coverSrc = computed(() => {
  const src = (props.book.coverUrl || "").trim()
  return src ? resolveMediaUrl(src) : ""
})

/** 卡片主键：漫画用 comicId，写真用 photoId。 */
const bookId = computed(() => props.book.comicId || props.book.photoId || "")
/** 最多展示三个标签，避免横条被挤出。 */
const tagsLine = computed(() => (props.book.tags ?? []).slice(0, 3).join(" · "))
</script>

<template>
  <button
    type="button"
    class="flex min-h-11 w-full overflow-hidden rounded-xl border border-border bg-card text-left text-card-foreground transition-[border-color] duration-150 hover:border-primary/30 motion-reduce:transition-none"
    :data-agent-book-card="bookId"
    :aria-label="t('agentWindow.openBook')"
    @click="emit('open', book)"
  >
    <div class="relative aspect-[2/3] w-14 shrink-0 overflow-hidden bg-muted/40">
      <img
        v-if="coverSrc"
        :src="coverSrc"
        :alt="book.title || bookId"
        class="absolute inset-0 size-full object-cover"
        loading="lazy"
        decoding="async"
        referrerpolicy="no-referrer"
      />
      <span
        v-else
        class="absolute inset-0 flex items-center justify-center px-1 text-center text-[9px] leading-tight text-muted-foreground"
      >
        {{ t("common.noArt") }}
      </span>
    </div>
    <div class="flex min-w-0 flex-1 flex-col justify-center gap-0.5 px-3 py-2">
      <p class="line-clamp-1 text-sm font-medium leading-snug">
        {{ book.title || bookId }}
      </p>
      <p class="truncate text-[11px] text-muted-foreground">
        {{ [t(`agentWindow.mention.${book.kind}`), tagsLine].filter(Boolean).join(" · ") }}
      </p>
    </div>
  </button>
</template>

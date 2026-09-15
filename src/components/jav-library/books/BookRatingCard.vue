<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Star } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import MovieRatingStars from "@/components/jav-library/MovieRatingStars.vue"

const props = defineProps<{
  rating: number | null
  disabled?: boolean
}>()

const emit = defineEmits<{
  commit: [value: number | null]
}>()

const { t } = useI18n()

const hasRating = computed(() => typeof props.rating === "number")
const starValue = computed(() => (typeof props.rating === "number" ? props.rating : 0))
const combinedLabel = computed(() =>
  typeof props.rating === "number"
    ? t("detailPanel.combined", { n: props.rating.toFixed(1) })
    : t("detailPanel.unrated"),
)

/** 把半星选择写成当前本地分。 */
function commitFromStars(value: number) {
  emit("commit", value)
}

/** 清除本地分，回到未评。 */
function clearRating() {
  if (!hasRating.value || props.disabled) return
  emit("commit", null)
}
</script>

<template>
  <div
    data-book-rating-card
    class="w-[250px] max-w-full rounded-2xl border border-border/70 bg-background/50 p-3"
  >
    <p class="text-xs text-muted-foreground">{{ t("detailPanel.rating") }}</p>
    <p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm font-semibold">
      <Star class="size-4 shrink-0 text-primary" aria-hidden="true" />
      <span :class="hasRating ? undefined : 'font-normal text-muted-foreground'">
        {{ combinedLabel }}
      </span>
    </p>
    <div class="mt-2 flex flex-wrap items-center gap-2">
      <span class="text-xs text-muted-foreground">{{ t("detailPanel.myRating") }}</span>
      <MovieRatingStars
        :model-value="starValue"
        :disabled="disabled"
        @commit="commitFromStars"
      />
      <Button
        type="button"
        variant="ghost"
        size="sm"
        class="h-7 rounded-full px-2 text-xs"
        data-book-rating-clear
        :disabled="disabled || !hasRating"
        @click="clearRating"
      >
        {{ t("detailPanel.clearLocalRating") }}
      </Button>
    </div>
  </div>
</template>

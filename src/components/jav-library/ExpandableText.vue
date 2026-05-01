<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"

const props = withDefaults(
  defineProps<{
    text: string
    collapsedLines?: number
    forceExpandable?: boolean
    expandLabel?: string
    collapseLabel?: string
  }>(),
  {
    collapsedLines: 5,
    forceExpandable: false,
  },
)

const { t } = useI18n()
const expanded = ref(false)

watch(
  () => props.text,
  () => {
    expanded.value = false
  },
)

const normalizedText = computed(() => props.text.trim())
const shouldShowToggle = computed(
  () => props.forceExpandable || normalizedText.value.length > 180,
)
const toggleLabel = computed(() =>
  expanded.value
    ? props.collapseLabel ?? t("movie.collapseSummary")
    : props.expandLabel ?? t("movie.expandSummary"),
)
</script>

<template>
  <div v-if="normalizedText" class="min-w-0">
    <p
      data-expandable-content
      class="text-pretty text-sm leading-6 text-muted-foreground"
      :class="!expanded && shouldShowToggle ? `line-clamp-${collapsedLines}` : ''"
    >
      {{ normalizedText }}
    </p>
    <button
      v-if="shouldShowToggle"
      data-expandable-toggle
      type="button"
      class="mt-2 inline-flex text-sm font-medium text-primary underline-offset-4 hover:underline"
      @click="expanded = !expanded"
    >
      {{ toggleLabel }}
    </button>
  </div>
</template>

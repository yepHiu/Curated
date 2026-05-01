<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { CuratedFrameFacetItemDTO } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  facets: readonly CuratedFrameFacetItemDTO[]
  visibleFacets: readonly CuratedFrameFacetItemDTO[]
  activeTag: string
  hiddenCount: number
  expanded: boolean
}>()

const emit = defineEmits<{
  clear: []
  toggleTag: [tag: string]
  updateExpanded: [expanded: boolean]
}>()

const { t } = useI18n()

function isActive(tag: string): boolean {
  return props.activeTag === tag.trim()
}
</script>

<template>
  <section
    class="shrink-0 rounded-3xl border border-border/70 bg-card/85 px-4 py-3 shadow-lg shadow-black/5"
    :aria-label="t('curated.tagFilterTitle')"
  >
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0 space-y-2">
        <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          {{ t("curated.tagFilterTitle") }}
        </p>
        <div v-if="facets.length > 0" class="flex flex-wrap gap-2">
          <Badge
            as-child
            :variant="!activeTag ? 'default' : 'secondary'"
            :class="[
              'rounded-full border px-3 py-1 text-sm font-normal transition-colors',
              !activeTag
                ? 'border-primary/40'
                : 'cursor-pointer border-border/60 bg-secondary/70 hover:bg-secondary hover:text-secondary-foreground',
            ]"
          >
            <button
              type="button"
              class="inline-flex max-w-full min-w-0 items-center gap-1.5"
              :aria-pressed="!activeTag"
              :aria-label="t('curated.ariaClearFrameTagFilter')"
              @click="emit('clear')"
            >
              {{ t("curated.tagFilterAll") }}
            </button>
          </Badge>
          <Badge
            v-for="tag in visibleFacets"
            :key="tag.name"
            as-child
            :variant="isActive(tag.name) ? 'default' : 'secondary'"
            :class="[
              'max-w-[14rem] rounded-full border px-3 py-1 text-sm font-normal transition-colors',
              isActive(tag.name)
                ? 'border-primary/40'
                : 'cursor-pointer border-border/60 bg-secondary/70 hover:bg-secondary hover:text-secondary-foreground',
            ]"
          >
            <button
              type="button"
              class="inline-flex max-w-full min-w-0 items-center gap-1.5"
              :aria-pressed="isActive(tag.name)"
              :aria-label="t('curated.ariaFilterFrameTag', { tag: tag.name, count: tag.count })"
              @click="emit('toggleTag', tag.name)"
            >
              <span class="truncate">{{ tag.name }}</span>
              <span
                class="tabular-nums text-xs opacity-80"
                :class="
                  isActive(tag.name)
                    ? 'text-primary-foreground/90'
                    : 'text-muted-foreground'
                "
              >
                · {{ tag.count }}
              </span>
            </button>
          </Badge>
        </div>
        <p v-else class="text-sm text-muted-foreground">
          {{ t("curated.tagFilterEmpty") }}
        </p>
      </div>
      <Button
        v-if="hiddenCount > 0"
        type="button"
        variant="ghost"
        size="sm"
        class="h-8 shrink-0 rounded-full px-3 text-xs text-muted-foreground hover:text-foreground"
        @click="emit('updateExpanded', !expanded)"
      >
        {{
          expanded
            ? t("curated.tagFilterShowLess")
            : t("curated.tagFilterShowMore", { count: hiddenCount })
        }}
      </Button>
    </div>
  </section>
</template>

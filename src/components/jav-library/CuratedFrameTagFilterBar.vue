<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Check, Tags } from "lucide-vue-next"
import type { CuratedFrameFacetItemDTO } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

const props = defineProps<{
  facets: readonly CuratedFrameFacetItemDTO[]
  selectedTags: readonly string[]
}>()

const emit = defineEmits<{
  clear: []
  toggleTag: [tag: string]
}>()

const { t } = useI18n()

const selectedSet = computed(() => {
  const set = new Set<string>()
  for (const tag of props.selectedTags) {
    const trimmed = tag.trim()
    if (trimmed) set.add(trimmed.toLocaleLowerCase())
  }
  return set
})

const selectedCount = computed(() => props.selectedTags.length)
const filterActive = computed(() => selectedCount.value > 0)

const buttonLabel = computed(() => {
  if (selectedCount.value === 0) {
    return t("curated.tagFilterTitle")
  }
  if (selectedCount.value === 1) {
    return props.selectedTags[0]!
  }
  return t("curated.tagFilterSelectedCount", { count: selectedCount.value })
})

const menuFacets = computed(() => {
  const selectedNames = new Set(selectedSet.value)
  const fromFacets = [...props.facets]
  for (const tag of props.selectedTags) {
    const trimmed = tag.trim()
    if (!trimmed) continue
    if (fromFacets.some((facet) => facet.name.toLocaleLowerCase() === trimmed.toLocaleLowerCase())) {
      continue
    }
    fromFacets.unshift({ name: trimmed, count: 0 })
  }
  return fromFacets.sort((left, right) => {
    const leftSelected = selectedNames.has(left.name.toLocaleLowerCase())
    const rightSelected = selectedNames.has(right.name.toLocaleLowerCase())
    if (leftSelected !== rightSelected) {
      return leftSelected ? -1 : 1
    }
    return right.count - left.count || left.name.localeCompare(right.name)
  })
})

function isSelected(tag: string): boolean {
  return selectedSet.value.has(tag.trim().toLocaleLowerCase())
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        type="button"
        size="sm"
        data-curated-tag-filter-toggle
        class="min-h-11 max-w-[14rem] gap-1.5 rounded-xl sm:min-h-8"
        :variant="filterActive ? 'secondary' : 'outline'"
        :aria-label="t('curated.tagFilterTitle')"
        :aria-pressed="filterActive"
      >
        <Tags class="size-4 shrink-0 opacity-80" aria-hidden="true" />
        <span class="min-w-0 truncate">{{ buttonLabel }}</span>
        <Badge v-if="selectedCount > 1" variant="outline" class="shrink-0 tabular-nums">
          {{ selectedCount }}
        </Badge>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent
      align="start"
      class="w-64 rounded-xl"
      data-curated-tag-filter-menu
    >
      <DropdownMenuLabel class="font-normal text-muted-foreground">
        {{ t("curated.tagFilterTitle") }}
      </DropdownMenuLabel>
      <p
        v-if="selectedCount > 0"
        class="px-2 pb-1 text-xs leading-relaxed text-muted-foreground"
        data-curated-tag-filter-selected
      >
        {{ t("curated.tagFilterSelectedList", { tags: selectedTags.join(" · ") }) }}
      </p>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem
          data-curated-tag-filter-clear
          :disabled="selectedCount === 0"
          @click="emit('clear')"
        >
          <Check v-if="selectedCount === 0" aria-hidden="true" />
          <Tags v-else aria-hidden="true" />
          {{ t("curated.tagFilterAll") }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <DropdownMenuSeparator />
      <DropdownMenuGroup v-if="menuFacets.length > 0" class="max-h-64 overflow-y-auto">
        <DropdownMenuCheckboxItem
          v-for="tag in menuFacets"
          :key="tag.name"
          :checked="isSelected(tag.name)"
          :data-curated-tag-filter-option="tag.name"
          @select.prevent
          @click="emit('toggleTag', tag.name)"
        >
          <span class="min-w-0 flex-1 truncate">{{ tag.name }}</span>
          <span class="shrink-0 tabular-nums text-xs text-muted-foreground">
            · {{ tag.count }}
          </span>
        </DropdownMenuCheckboxItem>
      </DropdownMenuGroup>
      <p v-else class="px-2 py-1.5 text-sm text-muted-foreground">
        {{ t("curated.tagFilterEmpty") }}
      </p>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

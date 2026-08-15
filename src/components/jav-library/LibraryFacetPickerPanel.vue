<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Check, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export interface FacetPickerRow {
  name: string
  count: number
}

export interface FacetPickerGroup {
  id: string
  label?: string
  ariaKey?: string
  rows: readonly FacetPickerRow[]
}

const props = defineProps<{
  title: string
  searchLabel: string
  emptyLabel: string
  selected: readonly string[]
  groups: readonly FacetPickerGroup[]
  testId: string
}>()

const emit = defineEmits<{
  clear: []
  toggle: [value: string]
}>()

const { t } = useI18n()
const query = ref("")

watch(
  () => props.title,
  () => {
    query.value = ""
  },
)

const selectedSet = computed(() => {
  const set = new Set<string>()
  for (const value of props.selected) {
    const trimmed = value.trim()
    if (trimmed) set.add(trimmed.toLocaleLowerCase())
  }
  return set
})

const selectedCount = computed(() => props.selected.length)
const queryNeedle = computed(() => query.value.trim().toLocaleLowerCase())

function matchesQuery(name: string): boolean {
  const needle = queryNeedle.value
  if (!needle) return true
  return name.toLocaleLowerCase().includes(needle)
}

const visibleGroups = computed(() =>
  props.groups
    .map((group) => ({
      ...group,
      rows: group.rows.filter((row) => matchesQuery(row.name)),
    }))
    .filter((group) => group.rows.length > 0),
)

const hasAnyOptions = computed(() => props.groups.some((group) => group.rows.length > 0))
const hasVisibleOptions = computed(() => visibleGroups.value.length > 0)

function isSelected(name: string): boolean {
  return selectedSet.value.has(name.trim().toLocaleLowerCase())
}
</script>

<template>
  <div
    class="flex h-[min(36rem,75vh)] w-[min(40rem,calc(100vw-2rem))] shrink-0 flex-col overflow-hidden rounded-xl border border-border/60 bg-background/40"
    :data-library-facet-picker="testId"
    :data-library-tag-filter-menu="testId === 'library-tag-filter' ? '' : undefined"
  >
    <div class="flex items-center gap-2 px-3 pt-3 pb-2">
      <p class="min-w-0 flex-1 text-sm font-semibold text-foreground">
        {{ title }}
      </p>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        class="h-8 shrink-0 px-2"
        :data-library-facet-picker-clear="testId"
        :data-library-tag-filter-clear="testId === 'library-tag-filter' ? '' : undefined"
        :disabled="selectedCount === 0"
        @click="emit('clear')"
      >
        {{ t("library.savedViewFacetClear") }}
      </Button>
    </div>

    <div
      v-if="selectedCount > 0"
      class="flex flex-wrap gap-1.5 px-3 pb-2"
      :data-library-facet-picker-selected="testId"
      :data-library-tag-filter-selected="testId === 'library-tag-filter' ? '' : undefined"
    >
      <button
        v-for="value in selected"
        :key="value"
        type="button"
        class="inline-flex min-h-8 max-w-full items-center gap-1 rounded-full border border-border/70 bg-muted/60 px-2.5 text-xs text-foreground hover:bg-muted"
        :data-library-facet-picker-chip="value"
        :data-library-tag-filter-chip="testId === 'library-tag-filter' ? value : undefined"
        :aria-label="t('library.savedViewClearChipAria', { label: value })"
        @click="emit('toggle', value)"
      >
        <span class="min-w-0 truncate">{{ value }}</span>
        <X class="size-3.5 shrink-0 opacity-70" aria-hidden="true" />
      </button>
    </div>

    <div v-if="hasAnyOptions" class="px-3 pb-2">
      <Input
        v-model="query"
        type="search"
        :data-library-facet-picker-search="testId"
        :data-library-tag-filter-search="testId === 'library-tag-filter' ? '' : undefined"
        class="h-9 min-h-9 rounded-xl"
        :placeholder="searchLabel"
        :aria-label="searchLabel"
      />
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto px-1.5 pb-2">
      <p
        v-if="!hasAnyOptions"
        class="px-2 py-1.5 text-sm text-muted-foreground"
      >
        {{ emptyLabel }}
      </p>
      <template v-else-if="hasVisibleOptions">
        <section
          v-for="(group, index) in visibleGroups"
          :key="group.id"
          class="flex flex-col gap-1"
          :class="index > 0 ? 'mt-2' : ''"
          :data-library-facet-picker-group="group.id"
        >
          <p
            v-if="group.label"
            class="px-2 pt-1 pb-1 text-xs font-medium text-muted-foreground"
          >
            {{ group.label }}
          </p>
          <div
            class="grid grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] gap-0.5"
            data-library-tag-filter-grid
          >
            <button
              v-for="row in group.rows"
              :key="`${group.id}-${row.name}`"
              type="button"
              class="flex min-w-0 items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm text-foreground hover:bg-muted/70"
              :aria-pressed="isSelected(row.name)"
              :aria-label="
                group.ariaKey
                  ? t(group.ariaKey, {
                      tag: row.name,
                      actor: row.name,
                      studio: row.name,
                      count: row.count,
                    })
                  : row.name
              "
              :data-library-facet-picker-option="row.name"
              :data-library-tag-filter-option="testId === 'library-tag-filter' ? row.name : undefined"
              :data-library-tag-filter-kind="testId === 'library-tag-filter' ? group.id : undefined"
              @click="emit('toggle', row.name)"
            >
              <Check
                v-if="isSelected(row.name)"
                class="size-3.5 shrink-0 text-primary"
                aria-hidden="true"
              />
              <span v-else class="size-3.5 shrink-0" aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate">{{ row.name }}</span>
              <span
                class="shrink-0 tabular-nums text-xs text-muted-foreground"
                data-library-tag-filter-count
              >
                {{ row.count }}
              </span>
            </button>
          </div>
        </section>
      </template>
    </div>
  </div>
</template>

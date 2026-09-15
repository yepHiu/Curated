<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { ArrowUpDown, Check, Filter, X } from "lucide-vue-next"
import type { ComicReadStatus } from "@/domain/comic/types"
import type { BookLibrarySortValue } from "@/lib/book-library-query"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type BookLibraryKind = "photos" | "comics"

type FilterChip = {
  key: string
  label: string
  clear: () => void
}

const props = withDefaults(
  defineProps<{
    kind: BookLibraryKind
    count: number
    sort: BookLibrarySortValue
    searchQuery?: string
    tag?: string
    favorite?: boolean
    readStatus?: ComicReadStatus | "all"
    batchMode?: boolean
    batchSelectedCount?: number
  }>(),
  {
    searchQuery: "",
    tag: "",
    favorite: false,
    readStatus: "all",
    batchMode: false,
    batchSelectedCount: 0,
  },
)

const emit = defineEmits<{
  sort: [value: BookLibrarySortValue]
  clearSearch: []
  clearTag: []
  updateFavorite: [value: boolean]
  updateReadStatus: [value: ComicReadStatus | "all"]
}>()

const { t } = useI18n()

const sortOptions: { value: BookLibrarySortValue; labelKey: string }[] = [
  { value: "addedAt", labelKey: "sortByAdded" },
  { value: "fileName", labelKey: "sortByFileName" },
]

const readStatusOptions: { value: ComicReadStatus | "all"; labelKey: string }[] = [
  { value: "all", labelKey: "comics.filterAll" },
  { value: "unread", labelKey: "comics.filterUnread" },
  { value: "reading", labelKey: "comics.filterReading" },
  { value: "read", labelKey: "comics.filterRead" },
]

const sortLabel = computed(() => t(`${props.kind}.${sortOptions.find((option) => option.value === props.sort)?.labelKey ?? "sortByAdded"}`))
const filterCount = computed(() => {
  let count = 0
  if (props.kind === "comics" && props.favorite) count += 1
  if (props.kind === "comics" && props.readStatus !== "all") count += 1
  return count
})
const chips = computed<FilterChip[]>(() => {
  const next: FilterChip[] = []
  const query = props.searchQuery.trim()
  if (query) {
    next.push({
      key: "q",
      label: query,
      clear: () => emit("clearSearch"),
    })
  }
  const tag = props.tag.trim()
  if (tag) {
    next.push({
      key: "tag",
      label: tag,
      clear: () => emit("clearTag"),
    })
  }
  if (props.kind === "comics" && props.favorite) {
    next.push({
      key: "favorite",
      label: t("comics.filterFavorite"),
      clear: () => emit("updateFavorite", false),
    })
  }
  if (props.kind === "comics" && props.readStatus !== "all") {
    const option = readStatusOptions.find((item) => item.value === props.readStatus)
    next.push({
      key: "readStatus",
      label: t(option?.labelKey ?? "comics.filterAll"),
      clear: () => emit("updateReadStatus", "all"),
    })
  }
  return next
})
const countLabel = computed(() => {
  if (props.batchMode) {
    return t("bookBrowser.selectedCount", {
      selected: props.batchSelectedCount,
      total: props.count,
    })
  }
  return t("bookBrowser.bookCount", { count: props.count })
})

/** 只接受书库已声明的排序值。 */
function selectSort(value: unknown) {
  if (value === "addedAt" || value === "fileName") {
    emit("sort", value)
  }
}

/** 切换漫画墙是否只看收藏。 */
function toggleFavorite(checked: boolean) {
  emit("updateFavorite", checked)
}

/** 切换漫画墙阅读状态筛选。 */
function selectReadStatus(value: unknown) {
  if (value === "unread" || value === "reading" || value === "read" || value === "all") {
    emit("updateReadStatus", value)
  }
}

/** 清空当前 chip 对应的约束。 */
function clearChip(chip: FilterChip) {
  chip.clear()
}
</script>

<template>
  <header class="flex min-w-0 w-full flex-wrap items-center justify-end gap-1.5">
    <h1 class="sr-only">{{ t(`nav.${kind}`) }}</h1>
    <p class="mr-auto min-w-0 text-sm tabular-nums text-muted-foreground">
      {{ countLabel }}
    </p>
    <div
      v-if="!batchMode && chips.length > 0"
      data-book-filter-chips
      class="flex min-w-0 flex-1 flex-nowrap items-center justify-end gap-1.5 overflow-x-auto"
    >
      <Badge
        v-for="chip in chips"
        :key="chip.key"
        as-child
        variant="secondary"
      >
        <button
          type="button"
          class="min-h-11 max-w-[10rem] sm:min-h-8"
          :data-book-filter-chip="chip.key"
          :aria-label="t('library.savedViewClearChipAria', { label: chip.label })"
          @click="clearChip(chip)"
        >
          <span class="min-w-0 truncate">{{ chip.label }}</span>
          <X data-icon="inline-end" aria-hidden="true" />
        </button>
      </Badge>
    </div>
    <template v-if="!batchMode">
      <DropdownMenu v-if="kind === 'comics'">
        <DropdownMenuTrigger as-child>
          <Button
            type="button"
            data-book-filter-trigger
            class="min-h-11 max-w-[13rem] shrink-0 rounded-full px-3 sm:min-h-8"
            :variant="filterCount > 0 ? 'secondary' : 'outline'"
            :aria-label="t('library.savedViewFilters')"
            :aria-pressed="filterCount > 0"
          >
            <Filter data-icon="inline-start" aria-hidden="true" />
            <span class="min-w-0 truncate">{{ t("library.savedViewFilters") }}</span>
            <Badge v-if="filterCount > 0" variant="secondary">{{ filterCount }}</Badge>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-56 rounded-2xl border-border/70" data-book-filter-menu>
          <DropdownMenuLabel class="font-normal text-muted-foreground">
            {{ t("library.savedViewFilters") }}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuCheckboxItem
            data-book-filter-favorite
            :checked="favorite"
            @update:checked="toggleFavorite"
          >
            {{ t("comics.filterFavorite") }}
          </DropdownMenuCheckboxItem>
          <DropdownMenuSeparator />
          <DropdownMenuLabel class="font-normal text-muted-foreground">
            {{ t("comics.filterReadStatus") }}
          </DropdownMenuLabel>
          <DropdownMenuRadioGroup :model-value="readStatus" @update:model-value="selectReadStatus">
            <DropdownMenuRadioItem
              v-for="option in readStatusOptions"
              :key="option.value"
              :value="option.value"
              :data-book-filter-read-status="option.value"
            >
              {{ t(option.labelKey) }}
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button
            type="button"
            data-book-sort-trigger
            class="min-h-11 max-w-[13rem] shrink-0 rounded-full px-3 sm:min-h-8"
            :variant="sort !== 'addedAt' ? 'secondary' : 'outline'"
            :aria-label="t('library.savedViewSort')"
            :aria-pressed="sort !== 'addedAt'"
          >
            <ArrowUpDown data-icon="inline-start" aria-hidden="true" />
            <span class="min-w-0 truncate">{{ sortLabel }}</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-56 rounded-2xl border-border/70" data-book-sort-menu>
          <DropdownMenuLabel class="font-normal text-muted-foreground">
            {{ t("library.savedViewSort") }}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem
              v-for="option in sortOptions"
              :key="option.value"
              :data-comic-sort-option="kind === 'comics' ? option.value : undefined"
              :data-photo-sort-option="kind === 'photos' ? option.value : undefined"
              @select="selectSort(option.value)"
            >
              <Check v-if="sort === option.value" aria-hidden="true" />
              <ArrowUpDown v-else aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate">{{ t(`${kind}.${option.labelKey}`) }}</span>
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </template>
    <slot />
  </header>
</template>

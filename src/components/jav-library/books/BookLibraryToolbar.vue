<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { ArrowUpDown, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuRadioGroup, DropdownMenuRadioItem } from "@/components/ui/dropdown-menu"
type Sort = "addedAt" | "fileName" | "favorite"
const props = defineProps<{ kind: "photos" | "comics"; count: number; sort: Sort; searchQuery?: string }>()
const emit = defineEmits<{ sort: [value: Sort]; clearSearch: [] }>()
const { t } = useI18n()
const options: {value: Sort; label: string}[] = [{ value: "addedAt", label: "sortByAdded" }, {value: "fileName", label: "sortByFileName"}, {value: "favorite", label: "sortByFavorite"}]
function select(value: unknown) { if (value === "addedAt" || value === "fileName" || value === "favorite") emit("sort", value) }
</script>
<template>
  <header class="flex flex-wrap items-center justify-between gap-3 pb-1">
    <div class="flex min-w-0 flex-wrap items-center gap-2">
      <h1 class="sr-only">{{ t(`nav.${kind}`) }}</h1>
      <p class="text-sm tabular-nums text-muted-foreground">{{ t('bookBrowser.bookCount', { count }) }}</p>
      <Button v-if="searchQuery" variant="secondary" class="min-h-11 max-w-full rounded-full lg:min-h-8" :aria-label="t(`${kind}.clearSearch`)" @click="emit('clearSearch')"><span class="max-w-40 truncate">{{ searchQuery }}</span><X data-icon="inline-end" /></Button>
    </div>
    <div class="flex flex-wrap items-center justify-end gap-2">
      <DropdownMenu>
        <DropdownMenuTrigger as-child><Button data-book-sort-trigger variant="outline" class="min-h-11 rounded-full lg:min-h-8"><ArrowUpDown data-icon="inline-start" />{{ t(`${kind}.${options.find(option => option.value === props.sort)!.label}`) }}</Button></DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuRadioGroup :model-value="sort" @update:model-value="select">
            <DropdownMenuRadioItem v-for="option in options" :key="option.value" :value="option.value" :data-comic-sort-option="kind === 'comics' ? option.value : undefined" :data-photo-sort-option="kind === 'photos' ? option.value : undefined">{{ t(`${kind}.${option.label}`) }}</DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <slot />
    </div>
  </header>
</template>

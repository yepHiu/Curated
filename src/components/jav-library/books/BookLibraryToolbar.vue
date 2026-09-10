<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { ArrowUpDown, Check, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator } from "@/components/ui/dropdown-menu"
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
            <span class="min-w-0 truncate">{{ sort === 'addedAt' ? t('library.savedViewSort') : t(`${kind}.${options.find(option => option.value === props.sort)!.label}`) }}</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-56 rounded-2xl border-border/70" data-book-sort-menu>
          <DropdownMenuLabel class="font-normal text-muted-foreground">
            {{ t("library.savedViewSort") }}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem
              v-for="option in options"
              :key="option.value"
              :data-comic-sort-option="kind === 'comics' ? option.value : undefined"
              :data-photo-sort-option="kind === 'photos' ? option.value : undefined"
              @select="select(option.value)"
            >
              <Check v-if="sort === option.value" aria-hidden="true" />
              <ArrowUpDown v-else aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate">{{ t(`${kind}.${option.label}`) }}</span>
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <slot />
    </div>
  </header>
</template>

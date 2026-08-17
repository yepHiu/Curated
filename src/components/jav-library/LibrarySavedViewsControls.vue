<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import {
  ArrowUpDown,
  Bookmark,
  Building2,
  Check,
  Filter,
  LoaderCircle,
  Pencil,
  RefreshCw,
  Save,
  Tags,
  Trash2,
  User,
  X,
} from "lucide-vue-next"
import type { SavedViewDTO, SavedViewFiltersV1 } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import LibraryFacetPickerPanel from "@/components/jav-library/LibraryFacetPickerPanel.vue"
import LibraryTagFilterControl from "@/components/jav-library/LibraryTagFilterControl.vue"
import { pushAppToast } from "@/composables/use-app-toast"
import {
  aggregateMetadataTagCounts,
  aggregateNamedCounts,
  aggregateUserTagCounts,
  withCurrentNamedCounts,
} from "@/lib/library-stats"
import {
  buildSavedViewFiltersV1,
  buildSavedViewRouteTarget,
  getLibraryActorExactFilters,
  getLibraryAddedWithinDaysQuery,
  getLibraryCatalogQuery,
  getLibraryPlayStateQuery,
  getLibraryResolutionQuery,
  getLibraryRuntimeQuery,
  getLibrarySortQuery,
  getLibraryStudioExactFilters,
  getLibraryTagExactFilters,
  getLibraryUnratedQuery,
  getLibraryUserRatingQuery,
  getLibraryYearQuery,
  mergeLibraryQuery,
  normalizeLibrarySortFilter,
  parseLibraryTagFilterText,
  resolveLibraryMode,
  serializeLibraryTagFilters,
} from "@/lib/library-query"
import { librarySortKeys, libraryTabFromSortKey } from "@/lib/movie-sort"
import { useLibraryService } from "@/services/library-service"

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const busy = ref(false)
const savedViewsMenuOpen = ref(false)
const createPanelOpen = ref(false)
const createNameDraft = ref("")
const createNameInputRef = ref<{ $el?: HTMLElement } | null>(null)
const renameDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const renameTarget = ref<SavedViewDTO | null>(null)
const deleteTarget = ref<SavedViewDTO | null>(null)
const nameDraft = ref("")

const mode = computed(() => resolveLibraryMode(route))
const savedViews = computed(() => libraryService.savedViews.value)
const currentFilters = computed(() => buildSavedViewFiltersV1(mode.value, route.query))
const sort = computed(() => getLibrarySortQuery(route.query))
const playState = computed(() => getLibraryPlayStateQuery(route.query))
const userRating = computed(() => getLibraryUserRatingQuery(route.query))
const unrated = computed(() => getLibraryUnratedQuery(route.query))
const resolution = computed(() => getLibraryResolutionQuery(route.query))
const addedWithinDays = computed(() => getLibraryAddedWithinDaysQuery(route.query))
const year = computed(() => getLibraryYearQuery(route.query))
const runtime = computed(() => getLibraryRuntimeQuery(route.query))
const catalog = computed(() => getLibraryCatalogQuery(route.query))
const tagFilters = computed(() => getLibraryTagExactFilters(route.query))
const actorFilters = computed(() => getLibraryActorExactFilters(route.query))
const studioFilters = computed(() => getLibraryStudioExactFilters(route.query))
const filterOpen = ref(false)
type LibraryFacetKind = "tag" | "actor" | "studio"
const activeFacet = ref<LibraryFacetKind | null>(null)
watch(filterOpen, (open) => {
  if (!open) activeFacet.value = null
})
const ratingSelectValue = computed(() => {
  if (unrated.value) return "unrated"
  return userRating.value === undefined ? "any" : String(userRating.value)
})
const advancedFilterCount = computed(
  () =>
    Number(playState.value !== "all") +
    Number(userRating.value !== undefined && !unrated.value) +
    Number(unrated.value) +
    Number(resolution.value !== "") +
    Number(addedWithinDays.value !== undefined) +
    Number(year.value !== "") +
    Number(runtime.value !== "") +
    Number(catalog.value !== "") +
    Number(tagFilters.value.length > 0) +
    Number(actorFilters.value.length > 0) +
    Number(studioFilters.value.length > 0),
)
const sortActive = computed(() => sort.value !== "added")
const sortButtonLabel = computed(() =>
  sortActive.value
    ? t(`library.savedViewSortValue.${sort.value}`)
    : t("library.savedViewSort"),
)
const yearOptions = computed(() => {
  const years = new Set<number>()
  for (const movie of libraryService.movies.value) {
    if (Number.isInteger(movie.year) && movie.year >= 1800 && movie.year <= 3000) {
      years.add(movie.year)
    }
  }
  return [...years].sort((left, right) => right - left)
})
const userTagSuggestions = computed(() =>
  aggregateUserTagCounts(libraryService.movies.value, locale.value),
)
const metadataTagSuggestions = computed(() =>
  aggregateMetadataTagCounts(libraryService.movies.value, locale.value),
)
const actorSuggestions = computed(() =>
  withCurrentNamedCounts(
    aggregateNamedCounts(
      libraryService.movies.value.flatMap((movie) => movie.actors),
      locale.value,
    ),
    actorFilters.value,
    locale.value,
  ),
)
const studioSuggestions = computed(() =>
  withCurrentNamedCounts(
    aggregateNamedCounts(
      libraryService.movies.value.map((movie) => movie.studio),
      locale.value,
    ),
    studioFilters.value,
    locale.value,
  ),
)
const actorPickerGroups = computed(() => [
  {
    id: "actors",
    ariaKey: "library.ariaFilterActor",
    rows: actorSuggestions.value,
  },
])
const studioPickerGroups = computed(() => [
  {
    id: "studios",
    ariaKey: "library.ariaFilterStudio",
    rows: studioSuggestions.value,
  },
])

function facetTriggerLabel(count: number, selectedKey: string): string {
  if (count === 0) {
    return t("library.savedViewAny")
  }
  return t(selectedKey, { count })
}

function toggleFacet(kind: LibraryFacetKind) {
  activeFacet.value = activeFacet.value === kind ? null : kind
}

const canonicalFilterKey = (filters: SavedViewFiltersV1) => JSON.stringify(filters)

const activeSavedViewId = computed(() => {
  const current = canonicalFilterKey(currentFilters.value)
  return savedViews.value.find((item) => canonicalFilterKey(item.filters) === current)?.id ?? ""
})

function errorMessage(error: unknown): string {
  return error instanceof Error && error.message.trim()
    ? error.message
    : t("library.savedViewOperationFailed")
}

async function refreshSavedViews() {
  try {
    await libraryService.refreshSavedViews()
  } catch (error) {
    pushAppToast(errorMessage(error), { variant: "destructive" })
  }
}

onMounted(() => {
  void refreshSavedViews()
})

async function updateAdvancedFilters(
  patch: Partial<
    Record<
      | "playState"
      | "userRating"
      | "unrated"
      | "resolution"
      | "addedWithinDays"
      | "year"
      | "runtime"
      | "catalog"
      | "tag"
      | "actor"
      | "studio"
      | "sort"
      | "tab",
      string | undefined
    >
  >,
) {
  await router.replace({
    name: mode.value,
    query: mergeLibraryQuery(route.query, {
      ...patch,
      selected: undefined,
    }),
  })
}

function selectString(value: unknown): string | undefined {
  return typeof value === "string" || typeof value === "number" ? String(value) : undefined
}

function onPlayStateChange(value: unknown) {
  const next = selectString(value)
  if (!next) return
  void updateAdvancedFilters({ playState: next })
}

function onRatingChange(value: unknown) {
  const next = selectString(value)
  if (!next) return
  if (next === "any") {
    void updateAdvancedFilters({ userRating: undefined, unrated: undefined })
    return
  }
  if (next === "unrated") {
    void updateAdvancedFilters({ userRating: undefined, unrated: "1" })
    return
  }
  void updateAdvancedFilters({ userRating: next, unrated: undefined })
}

function onResolutionChange(value: unknown) {
  const next = selectString(value)
  if (!next) return
  void updateAdvancedFilters({ resolution: next === "any" ? undefined : next })
}

function onAddedWindowChange(value: unknown) {
  const next = selectString(value)
  if (!next) return
  void updateAdvancedFilters({ addedWithinDays: next === "any" ? undefined : next })
}

function onYearChange(value: unknown) {
  const next = selectString(value)
  if (!next) return
  void updateAdvancedFilters({ year: next === "any" ? undefined : next })
}

function onRuntimeChange(value: unknown) {
  const next = selectString(value)
  if (!next) return
  void updateAdvancedFilters({ runtime: next === "any" ? undefined : next })
}

function onCatalogChange(value: unknown) {
  const next = selectString(value)
  if (!next) return
  void updateAdvancedFilters({ catalog: next === "any" ? undefined : next })
}

function toggleTagFilter(tag: string) {
  const normalized = tag.trim()
  if (!normalized) {
    return
  }
  const key = normalized.toLocaleLowerCase()
  const next = tagFilters.value.filter((item) => item.toLocaleLowerCase() !== key)
  if (next.length === tagFilters.value.length) {
    next.push(normalized)
  }
  void updateAdvancedFilters({ tag: serializeLibraryTagFilters(next) })
}

function clearTagFilters() {
  void updateAdvancedFilters({ tag: undefined })
}

function toggleDelimitedFilter(
  key: "actor" | "studio",
  current: readonly string[],
  value: string,
) {
  const normalized = value.trim()
  if (!normalized) {
    return
  }
  const needle = normalized.toLocaleLowerCase()
  const next = current.filter((item) => item.toLocaleLowerCase() !== needle)
  if (next.length === current.length) {
    next.push(normalized)
  }
  void updateAdvancedFilters({ [key]: serializeLibraryTagFilters(next) })
}

function toggleActorFilter(actor: string) {
  toggleDelimitedFilter("actor", actorFilters.value, actor)
}

function clearActorFilters() {
  void updateAdvancedFilters({ actor: undefined })
}

function toggleStudioFilter(studio: string) {
  toggleDelimitedFilter("studio", studioFilters.value, studio)
}

function clearStudioFilters() {
  void updateAdvancedFilters({ studio: undefined })
}

function onSortChange(value: unknown) {
  const next = normalizeLibrarySortFilter(selectString(value) ?? "")
  if (!next) return
  const tab = libraryTabFromSortKey(next)
  if (tab === "none") {
    void updateAdvancedFilters({ sort: next, tab: undefined })
    return
  }
  void updateAdvancedFilters({
    sort: undefined,
    tab: tab === "all" ? undefined : tab,
  })
}

function clearAdvancedFilters() {
  void updateAdvancedFilters({
    playState: undefined,
    userRating: undefined,
    unrated: undefined,
    resolution: undefined,
    addedWithinDays: undefined,
    year: undefined,
    runtime: undefined,
    catalog: undefined,
    tag: undefined,
    actor: undefined,
    studio: undefined,
  })
}

const activeFilterChips = computed(() => {
  const chips: { key: string; label: string; clear: () => void }[] = []
  if (playState.value !== "all") {
    chips.push({
      key: "playState",
      label: t(`library.savedViewPlay.${playState.value}`),
      clear: () => void updateAdvancedFilters({ playState: undefined }),
    })
  }
  if (unrated.value) {
    chips.push({
      key: "unrated",
      label: t("library.savedViewUnrated"),
      clear: () => void updateAdvancedFilters({ unrated: undefined }),
    })
  } else if (userRating.value !== undefined) {
    chips.push({
      key: "userRating",
      label: t("library.savedViewRatingAtLeast", { value: userRating.value }),
      clear: () => void updateAdvancedFilters({ userRating: undefined }),
    })
  }
  if (resolution.value) {
    chips.push({
      key: "resolution",
      label: resolution.value.toUpperCase(),
      clear: () => void updateAdvancedFilters({ resolution: undefined }),
    })
  }
  if (addedWithinDays.value !== undefined) {
    chips.push({
      key: "addedWithinDays",
      label: t("library.savedViewAddedDays", { days: addedWithinDays.value }),
      clear: () => void updateAdvancedFilters({ addedWithinDays: undefined }),
    })
  }
  if (year.value === "unknown") {
    chips.push({
      key: "year",
      label: t("library.savedViewYearUnknown"),
      clear: () => void updateAdvancedFilters({ year: undefined }),
    })
  } else if (year.value) {
    chips.push({
      key: "year",
      label: year.value,
      clear: () => void updateAdvancedFilters({ year: undefined }),
    })
  }
  if (runtime.value) {
    chips.push({
      key: "runtime",
      label: t(`library.savedViewRuntimeValue.${runtime.value}`),
      clear: () => void updateAdvancedFilters({ runtime: undefined }),
    })
  }
  if (catalog.value) {
    chips.push({
      key: "catalog",
      label: t(`library.savedViewCatalogValue.${catalog.value}`),
      clear: () => void updateAdvancedFilters({ catalog: undefined }),
    })
  }
  if (tagFilters.value.length > 0) {
    for (const tag of tagFilters.value) {
      chips.push({
        key: `tag:${tag.toLocaleLowerCase()}`,
        label: t("library.savedViewSummaryTag", { value: tag }),
        clear: () => toggleTagFilter(tag),
      })
    }
  }
  for (const actor of actorFilters.value) {
    chips.push({
      key: `actor:${actor.toLocaleLowerCase()}`,
      label: t("library.savedViewSummaryActor", { value: actor }),
      clear: () => toggleActorFilter(actor),
    })
  }
  for (const studio of studioFilters.value) {
    chips.push({
      key: `studio:${studio.toLocaleLowerCase()}`,
      label: t("library.savedViewSummaryStudio", { value: studio }),
      clear: () => toggleStudioFilter(studio),
    })
  }
  return chips
})

function focusCreateNameInput() {
  const input = createNameInputRef.value?.$el
  if (input instanceof HTMLInputElement) {
    input.focus()
    input.select()
  }
}

function onCreatePanelOpenAutoFocus(event: Event) {
  event.preventDefault()
  void nextTick(() => {
    focusCreateNameInput()
  })
}

watch(createPanelOpen, (open) => {
  if (!open) return
  createNameDraft.value = ""
  void nextTick(() => {
    focusCreateNameInput()
  })
})

function openRenameDialog(item: SavedViewDTO) {
  renameTarget.value = item
  nameDraft.value = item.name
  renameDialogOpen.value = true
}

async function submitCreate() {
  if (!createNameDraft.value.trim() || busy.value) {
    return
  }
  busy.value = true
  try {
    await libraryService.createSavedView(createNameDraft.value, currentFilters.value)
    pushAppToast(t("library.savedViewCreated"), { variant: "success" })
    createPanelOpen.value = false
    savedViewsMenuOpen.value = false
  } catch (error) {
    pushAppToast(errorMessage(error), { variant: "destructive" })
  } finally {
    busy.value = false
  }
}

async function submitRename() {
  if (!renameTarget.value || !nameDraft.value.trim() || busy.value) {
    return
  }
  busy.value = true
  try {
    await libraryService.updateSavedView(renameTarget.value.id, { name: nameDraft.value })
    pushAppToast(t("library.savedViewRenamed"), { variant: "success" })
    renameDialogOpen.value = false
  } catch (error) {
    pushAppToast(errorMessage(error), { variant: "destructive" })
  } finally {
    busy.value = false
  }
}

async function applySavedView(item: SavedViewDTO) {
  savedViewsMenuOpen.value = false
  await router.push(buildSavedViewRouteTarget(item.filters))
}

function onSavedViewDoubleClick(item: SavedViewDTO) {
  void applySavedView(item)
}

async function updateSavedViewFilters(item: SavedViewDTO) {
  if (busy.value) return
  busy.value = true
  try {
    await libraryService.updateSavedView(item.id, { filters: currentFilters.value })
    pushAppToast(t("library.savedViewUpdated"), { variant: "success" })
  } catch (error) {
    pushAppToast(errorMessage(error), { variant: "destructive" })
  } finally {
    busy.value = false
  }
}

function openDeleteDialog(item: SavedViewDTO) {
  deleteTarget.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value || busy.value) return
  busy.value = true
  try {
    await libraryService.deleteSavedView(deleteTarget.value.id)
    deleteDialogOpen.value = false
    pushAppToast(t("library.savedViewDeleted"), { variant: "success" })
  } catch (error) {
    pushAppToast(errorMessage(error), { variant: "destructive" })
  } finally {
    busy.value = false
  }
}

function filterSummary(filters: SavedViewFiltersV1): string {
  const entries: string[] = []
  if (filters.q) entries.push(t("library.savedViewSummarySearch", { value: filters.q }))
  const actors = parseLibraryTagFilterText(filters.actor)
  if (actors.length > 0) {
    entries.push(t("library.savedViewSummaryActor", { value: actors.join(" · ") }))
  }
  if (filters.tag) {
    const tags = parseLibraryTagFilterText(filters.tag)
    if (tags.length > 0) {
      entries.push(t("library.savedViewSummaryTag", { value: tags.join(" · ") }))
    }
  }
  const studios = parseLibraryTagFilterText(filters.studio)
  if (studios.length > 0) {
    entries.push(t("library.savedViewSummaryStudio", { value: studios.join(" · ") }))
  }
  if (filters.playState && filters.playState !== "all") {
    entries.push(t(`library.savedViewPlay.${filters.playState}`))
  }
  if (filters.unrated) {
    entries.push(t("library.savedViewUnrated"))
  } else if (filters.userRating !== undefined) {
    entries.push(t("library.savedViewRatingAtLeast", { value: filters.userRating }))
  }
  if (filters.resolution) entries.push(filters.resolution.toUpperCase())
  if (filters.addedWithinDays) {
    entries.push(t("library.savedViewAddedDays", { days: filters.addedWithinDays }))
  }
  if (filters.year === "unknown") {
    entries.push(t("library.savedViewYearUnknown"))
  } else if (filters.year) {
    entries.push(filters.year)
  }
  if (filters.runtime) {
    entries.push(t(`library.savedViewRuntimeValue.${filters.runtime}`))
  }
  if (filters.catalog) {
    entries.push(t(`library.savedViewCatalogValue.${filters.catalog}`))
  }
  if (filters.sort && filters.sort !== "added") {
    entries.push(t(`library.savedViewSortValue.${filters.sort}`))
  }
  return entries.join(" · ") || t("library.savedViewAllLibrary")
}
</script>

<template>
  <div
    data-library-saved-view-controls
    class="flex min-w-0 max-w-full flex-nowrap items-center justify-end gap-1.5"
  >
    <div
      v-if="mode !== 'trash' && activeFilterChips.length > 0"
      data-library-filter-chips
      class="flex min-w-0 flex-1 flex-nowrap items-center justify-end gap-1.5 overflow-x-auto"
    >
      <Badge
        v-for="chip in activeFilterChips"
        :key="chip.key"
        as-child
        variant="secondary"
      >
        <button
          type="button"
          class="min-h-11 max-w-[10rem] sm:min-h-8"
          :data-library-filter-chip="chip.key"
          :aria-label="t('library.savedViewClearChipAria', { label: chip.label })"
          @click="chip.clear"
        >
          <span class="min-w-0 truncate">{{ chip.label }}</span>
          <X data-icon="inline-end" aria-hidden="true" />
        </button>
      </Badge>
    </div>
    <div
      data-library-saved-view-actions
      class="flex shrink-0 flex-nowrap items-center justify-end gap-1.5"
    >
    <Popover v-if="mode !== 'trash'" v-model:open="filterOpen">
      <PopoverTrigger as-child>
        <Button
          type="button"
          variant="outline"
          class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
          :aria-label="t('library.savedViewFilters')"
        >
          <Filter data-icon="inline-start" aria-hidden="true" />
          {{ t("library.savedViewFilters") }}
          <Badge v-if="advancedFilterCount > 0" variant="secondary">
            {{ advancedFilterCount }}
          </Badge>
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="end"
        class="w-auto max-w-[calc(100vw-2rem)] rounded-2xl border-border/70 p-3"
        data-library-filter-menu
      >
        <div class="flex flex-wrap items-start gap-3" data-library-filter-popover>
          <LibraryTagFilterControl
            v-if="activeFacet === 'tag'"
            :metadata-tags="metadataTagSuggestions"
            :user-tags="userTagSuggestions"
            :selected-tags="tagFilters"
            @clear="clearTagFilters"
            @toggle-tag="toggleTagFilter"
          />
          <LibraryFacetPickerPanel
            v-else-if="activeFacet === 'actor'"
            test-id="library-actor-filter"
            :title="t('library.savedViewActor')"
            :search-label="t('library.savedViewActorSearch')"
            :empty-label="t('library.savedViewActorEmpty')"
            :selected="actorFilters"
            :groups="actorPickerGroups"
            @clear="clearActorFilters"
            @toggle="toggleActorFilter"
          />
          <LibraryFacetPickerPanel
            v-else-if="activeFacet === 'studio'"
            test-id="library-studio-filter"
            :title="t('library.savedViewStudio')"
            :search-label="t('library.savedViewStudioSearch')"
            :empty-label="t('library.savedViewStudioEmpty')"
            :selected="studioFilters"
            :groups="studioPickerGroups"
            @clear="clearStudioFilters"
            @toggle="toggleStudioFilter"
          />
          <div class="flex w-[min(28rem,calc(100vw-2rem))] shrink-0 flex-col gap-4">
            <div class="flex max-h-[min(32rem,70vh)] flex-col gap-4 overflow-y-auto">
          <p class="text-sm font-semibold text-foreground">
            {{ t("library.savedViewFiltersTitle") }}
          </p>

          <div class="grid gap-3 sm:grid-cols-2">
            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground">
              {{ t("library.savedViewPlayState") }}
                <Select :model-value="playState" @update:model-value="onPlayStateChange">
                  <SelectTrigger class="w-full rounded-xl">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="all">{{ t("library.savedViewPlay.all") }}</SelectItem>
                    <SelectItem value="unwatched">{{ t("library.savedViewPlay.unwatched") }}</SelectItem>
                    <SelectItem value="in-progress">{{ t("library.savedViewPlay.in-progress") }}</SelectItem>
                    <SelectItem value="completed">{{ t("library.savedViewPlay.completed") }}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </label>

            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground">
              {{ t("library.savedViewUserRating") }}
              <Select
                :model-value="ratingSelectValue"
                @update:model-value="onRatingChange"
              >
                <SelectTrigger class="w-full rounded-xl">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="any">{{ t("library.savedViewAny") }}</SelectItem>
                    <SelectItem value="unrated">{{ t("library.savedViewUnrated") }}</SelectItem>
                    <SelectItem v-for="value in [5, 4, 3, 2, 1]" :key="value" :value="String(value)">
                      {{ t("library.savedViewRatingAtLeast", { value }) }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </label>

            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground">
              {{ t("library.savedViewResolution") }}
              <Select
                :model-value="resolution || 'any'"
                @update:model-value="onResolutionChange"
              >
                <SelectTrigger class="w-full rounded-xl">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="any">{{ t("library.savedViewAny") }}</SelectItem>
                    <SelectItem value="4k">4K</SelectItem>
                    <SelectItem value="1080p">1080p</SelectItem>
                    <SelectItem value="720p">720p</SelectItem>
                    <SelectItem value="480p">480p</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </label>

            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground">
              {{ t("library.savedViewAddedWindow") }}
              <Select
                :model-value="addedWithinDays === undefined ? 'any' : String(addedWithinDays)"
                @update:model-value="onAddedWindowChange"
              >
                <SelectTrigger class="w-full rounded-xl">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="any">{{ t("library.savedViewAny") }}</SelectItem>
                    <SelectItem v-for="days in [7, 30, 90, 365]" :key="days" :value="String(days)">
                      {{ t("library.savedViewAddedDays", { days }) }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </label>

            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground">
              {{ t("library.savedViewYear") }}
              <Select
                :model-value="year || 'any'"
                @update:model-value="onYearChange"
              >
                <SelectTrigger class="w-full rounded-xl">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="any">{{ t("library.savedViewAny") }}</SelectItem>
                    <SelectItem value="unknown">{{ t("library.savedViewYearUnknown") }}</SelectItem>
                    <SelectItem v-for="option in yearOptions" :key="option" :value="String(option)">
                      {{ option }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </label>

            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground">
              {{ t("library.savedViewRuntime") }}
              <Select
                :model-value="runtime || 'any'"
                @update:model-value="onRuntimeChange"
              >
                <SelectTrigger class="w-full rounded-xl">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="any">{{ t("library.savedViewAny") }}</SelectItem>
                    <SelectItem value="short">{{ t("library.savedViewRuntimeValue.short") }}</SelectItem>
                    <SelectItem value="standard">{{ t("library.savedViewRuntimeValue.standard") }}</SelectItem>
                    <SelectItem value="long">{{ t("library.savedViewRuntimeValue.long") }}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </label>

            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground sm:col-span-2">
              {{ t("library.savedViewCatalog") }}
              <Select
                :model-value="catalog || 'any'"
                @update:model-value="onCatalogChange"
              >
                <SelectTrigger class="w-full rounded-xl">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="any">{{ t("library.savedViewAny") }}</SelectItem>
                    <SelectItem value="unscraped">{{ t("library.savedViewCatalogValue.unscraped") }}</SelectItem>
                    <SelectItem value="no-cover">{{ t("library.savedViewCatalogValue.no-cover") }}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </label>

            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground sm:col-span-2">
              {{ t("library.savedViewTag") }}
              <Button
                type="button"
                size="sm"
                data-library-tag-filter-toggle
                class="h-9 w-full justify-start gap-1.5 rounded-xl"
                :variant="tagFilters.length > 0 || activeFacet === 'tag' ? 'secondary' : 'outline'"
                :aria-label="t('library.savedViewTag')"
                :aria-pressed="tagFilters.length > 0"
                :aria-expanded="activeFacet === 'tag'"
                @click="toggleFacet('tag')"
              >
                <Tags class="size-4 shrink-0 opacity-80" aria-hidden="true" />
                <span class="min-w-0 truncate">
                  {{ facetTriggerLabel(tagFilters.length, "library.savedViewTagSelectedCount") }}
                </span>
              </Button>
            </label>

            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground">
              {{ t("library.savedViewActor") }}
              <Button
                type="button"
                size="sm"
                data-library-actor-filter-toggle
                class="h-9 w-full justify-start gap-1.5 rounded-xl"
                :variant="actorFilters.length > 0 || activeFacet === 'actor' ? 'secondary' : 'outline'"
                :aria-label="t('library.savedViewActor')"
                :aria-pressed="actorFilters.length > 0"
                :aria-expanded="activeFacet === 'actor'"
                @click="toggleFacet('actor')"
              >
                <User class="size-4 shrink-0 opacity-80" aria-hidden="true" />
                <span class="min-w-0 truncate">
                  {{ facetTriggerLabel(actorFilters.length, "library.savedViewActorSelectedCount") }}
                </span>
              </Button>
            </label>

            <label class="flex min-w-0 flex-col gap-1.5 text-xs font-medium text-muted-foreground">
              {{ t("library.savedViewStudio") }}
              <Button
                type="button"
                size="sm"
                data-library-studio-filter-toggle
                class="h-9 w-full justify-start gap-1.5 rounded-xl"
                :variant="studioFilters.length > 0 || activeFacet === 'studio' ? 'secondary' : 'outline'"
                :aria-label="t('library.savedViewStudio')"
                :aria-pressed="studioFilters.length > 0"
                :aria-expanded="activeFacet === 'studio'"
                @click="toggleFacet('studio')"
              >
                <Building2 class="size-4 shrink-0 opacity-80" aria-hidden="true" />
                <span class="min-w-0 truncate">
                  {{ facetTriggerLabel(studioFilters.length, "library.savedViewStudioSelectedCount") }}
                </span>
              </Button>
            </label>
          </div>

          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="self-end"
            :disabled="advancedFilterCount === 0"
            @click="clearAdvancedFilters"
          >
            {{ t("library.savedViewClearFilters") }}
          </Button>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>

    <DropdownMenu v-if="mode !== 'trash'">
      <DropdownMenuTrigger as-child>
        <Button
          type="button"
          data-library-sort-toggle
          class="min-h-11 max-w-[13rem] shrink-0 rounded-full px-3 sm:min-h-8"
          :variant="sortActive ? 'secondary' : 'outline'"
          :aria-label="t('library.savedViewSort')"
          :aria-pressed="sortActive"
        >
          <ArrowUpDown data-icon="inline-start" aria-hidden="true" />
          <span class="min-w-0 truncate">{{ sortButtonLabel }}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        class="w-56 rounded-2xl border-border/70"
        data-library-sort-menu
      >
        <DropdownMenuLabel class="font-normal text-muted-foreground">
          {{ t("library.savedViewSort") }}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem
            v-for="option in librarySortKeys"
            :key="option"
            :data-library-sort-option="option"
            @click="onSortChange(option)"
          >
            <Check v-if="sort === option" aria-hidden="true" />
            <ArrowUpDown v-else aria-hidden="true" />
            <span class="min-w-0 flex-1 truncate">
              {{ t(`library.savedViewSortValue.${option}`) }}
            </span>
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>

    <DropdownMenu v-model:open="savedViewsMenuOpen">
      <DropdownMenuTrigger as-child>
        <Button
          type="button"
          variant="outline"
          data-library-saved-views-toggle
          class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
          :aria-label="t('library.savedViews')"
        >
          <Bookmark data-icon="inline-start" aria-hidden="true" />
          {{ t("library.savedViews") }}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        class="w-64 rounded-2xl border-border/70"
        data-library-saved-views-menu
      >
        <DropdownMenuGroup>
          <DropdownMenuSub v-model:open="createPanelOpen">
            <DropdownMenuSubTrigger
              :disabled="savedViews.length >= 50 || busy"
              data-library-saved-view-create-trigger
            >
              <Save aria-hidden="true" />
              <span class="min-w-0 flex-1 whitespace-normal text-left leading-snug">
                {{ t("library.savedViewSaveCurrent") }}
              </span>
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent
              class="w-64 overflow-visible rounded-2xl border-border/70 p-3"
              data-library-saved-view-create-panel
              :align-offset="-4"
              :align-flip="false"
              @openAutoFocus="onCreatePanelOpenAutoFocus"
            >
              <form class="flex flex-col gap-3" @submit.prevent="submitCreate" @keydown.stop>
                <label class="flex flex-col gap-2 text-sm font-medium text-foreground">
                  {{ t("library.savedViewName") }}
                  <Input
                    ref="createNameInputRef"
                    v-model="createNameDraft"
                    maxlength="40"
                    autocomplete="off"
                    data-library-saved-view-create-name
                    :placeholder="t('library.savedViewNamePlaceholder')"
                    :disabled="busy"
                    @pointerdown.stop
                  />
                </label>
                <div class="flex justify-end">
                  <Button
                    type="submit"
                    class="min-h-8 rounded-full px-4"
                    :disabled="busy || !createNameDraft.trim()"
                    @click="submitCreate"
                  >
                    <LoaderCircle v-if="busy" data-icon="inline-start" class="animate-spin" aria-hidden="true" />
                    {{ t("library.savedViewSave") }}
                  </Button>
                </div>
              </form>
            </DropdownMenuSubContent>
          </DropdownMenuSub>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuLabel v-if="savedViews.length === 0" class="font-normal text-muted-foreground">
          {{ t("library.savedViewEmpty") }}
        </DropdownMenuLabel>
        <DropdownMenuGroup v-else>
          <DropdownMenuSub v-for="item in savedViews" :key="item.id">
            <DropdownMenuSubTrigger
              :data-library-saved-view-item="item.id"
              @dblclick.prevent.stop="onSavedViewDoubleClick(item)"
            >
              <Check v-if="activeSavedViewId === item.id" aria-hidden="true" />
              <Bookmark v-else aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate">{{ item.name }}</span>
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent class="w-60 rounded-2xl border-border/70" data-library-saved-view-item-menu>
              <DropdownMenuLabel class="flex flex-col gap-1">
                <span class="truncate text-foreground">{{ item.name }}</span>
                <span class="line-clamp-2 font-normal text-muted-foreground">
                  {{ filterSummary(item.filters) }}
                </span>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem @click="applySavedView(item)">
                  <Check aria-hidden="true" />
                  {{ t("library.savedViewApply") }}
                </DropdownMenuItem>
                <DropdownMenuItem :disabled="busy" @click="updateSavedViewFilters(item)">
                  <RefreshCw aria-hidden="true" />
                  {{ t("library.savedViewUpdateFilters") }}
                </DropdownMenuItem>
                <DropdownMenuItem @click="openRenameDialog(item)">
                  <Pencil aria-hidden="true" />
                  {{ t("library.savedViewRename") }}
                </DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem variant="destructive" @click="openDeleteDialog(item)">
                  <Trash2 aria-hidden="true" />
                  {{ t("library.savedViewDelete") }}
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuSubContent>
          </DropdownMenuSub>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
      <slot />
    </div>
  </div>

  <Dialog v-model:open="renameDialogOpen">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>
          {{ t("library.savedViewRenameTitle") }}
        </DialogTitle>
        <DialogDescription class="text-pretty">
          {{ t("library.savedViewRenameDescription") }}
        </DialogDescription>
      </DialogHeader>
      <label class="flex flex-col gap-2 text-sm font-medium text-foreground">
        {{ t("library.savedViewName") }}
        <Input
          v-model="nameDraft"
          maxlength="40"
          autocomplete="off"
          :placeholder="t('library.savedViewNamePlaceholder')"
          @keydown.enter.prevent="submitRename"
        />
      </label>
      <DialogFooter class="gap-3">
        <DialogClose as-child>
          <Button type="button" variant="outline" :disabled="busy">
            {{ t("common.cancel") }}
          </Button>
        </DialogClose>
        <Button type="button" :disabled="busy || !nameDraft.trim()" @click="submitRename">
          <LoaderCircle v-if="busy" data-icon="inline-start" class="animate-spin" aria-hidden="true" />
          {{ t("library.savedViewRename") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <Dialog v-model:open="deleteDialogOpen">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t("library.savedViewDeleteTitle") }}</DialogTitle>
        <DialogDescription class="text-pretty">
          {{ t("library.savedViewDeleteDescription", { name: deleteTarget?.name ?? "" }) }}
        </DialogDescription>
      </DialogHeader>
      <DialogFooter class="gap-3">
        <DialogClose as-child>
          <Button type="button" variant="outline" :disabled="busy">
            {{ t("common.cancel") }}
          </Button>
        </DialogClose>
        <Button type="button" variant="destructive" :disabled="busy" @click="confirmDelete">
          <LoaderCircle v-if="busy" data-icon="inline-start" class="animate-spin" aria-hidden="true" />
          {{ t("library.savedViewDelete") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import {
  Bookmark,
  Check,
  ChevronDown,
  ChevronUp,
  Filter,
  LoaderCircle,
  Pencil,
  RefreshCw,
  Save,
  Trash2,
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
import { pushAppToast } from "@/composables/use-app-toast"
import {
  buildSavedViewFiltersV1,
  buildSavedViewRouteTarget,
  getLibraryAddedWithinDaysQuery,
  getLibraryPlayStateQuery,
  getLibraryResolutionQuery,
  getLibraryUserRatingQuery,
  mergeLibraryQuery,
  resolveLibraryMode,
} from "@/lib/library-query"
import { useLibraryService } from "@/services/library-service"

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const busy = ref(false)
const editDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const editMode = ref<"create" | "rename">("create")
const editTarget = ref<SavedViewDTO | null>(null)
const deleteTarget = ref<SavedViewDTO | null>(null)
const nameDraft = ref("")

const mode = computed(() => resolveLibraryMode(route))
const savedViews = computed(() => libraryService.savedViews.value)
const currentFilters = computed(() => buildSavedViewFiltersV1(mode.value, route.query))
const playState = computed(() => getLibraryPlayStateQuery(route.query))
const userRating = computed(() => getLibraryUserRatingQuery(route.query))
const resolution = computed(() => getLibraryResolutionQuery(route.query))
const addedWithinDays = computed(() => getLibraryAddedWithinDaysQuery(route.query))
const advancedFilterCount = computed(
  () =>
    Number(playState.value !== "all") +
    Number(userRating.value !== undefined) +
    Number(resolution.value !== "") +
    Number(addedWithinDays.value !== undefined),
)

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
  patch: Partial<Record<"playState" | "userRating" | "resolution" | "addedWithinDays", string | undefined>>,
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
  void updateAdvancedFilters({ userRating: next === "any" ? undefined : next })
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

function clearAdvancedFilters() {
  void updateAdvancedFilters({
    playState: undefined,
    userRating: undefined,
    resolution: undefined,
    addedWithinDays: undefined,
  })
}

function openCreateDialog() {
  editMode.value = "create"
  editTarget.value = null
  nameDraft.value = ""
  editDialogOpen.value = true
}

function openRenameDialog(item: SavedViewDTO) {
  editMode.value = "rename"
  editTarget.value = item
  nameDraft.value = item.name
  editDialogOpen.value = true
}

async function submitEdit() {
  if (!nameDraft.value.trim() || busy.value) {
    return
  }
  busy.value = true
  try {
    if (editMode.value === "create") {
      await libraryService.createSavedView(nameDraft.value, currentFilters.value)
      pushAppToast(t("library.savedViewCreated"), { variant: "success" })
    } else if (editTarget.value) {
      await libraryService.updateSavedView(editTarget.value.id, { name: nameDraft.value })
      pushAppToast(t("library.savedViewRenamed"), { variant: "success" })
    }
    editDialogOpen.value = false
  } catch (error) {
    pushAppToast(errorMessage(error), { variant: "destructive" })
  } finally {
    busy.value = false
  }
}

async function applySavedView(item: SavedViewDTO) {
  await router.push(buildSavedViewRouteTarget(item.filters))
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

async function moveSavedView(item: SavedViewDTO, direction: -1 | 1) {
  const index = savedViews.value.findIndex((candidate) => candidate.id === item.id)
  const target = index + direction
  if (index < 0 || target < 0 || target >= savedViews.value.length || busy.value) {
    return
  }
  const ids = savedViews.value.map((candidate) => candidate.id)
  ;[ids[index], ids[target]] = [ids[target]!, ids[index]!]
  busy.value = true
  try {
    await libraryService.reorderSavedViews(ids)
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
  if (filters.actor) entries.push(t("library.savedViewSummaryActor", { value: filters.actor }))
  if (filters.tag) entries.push(t("library.savedViewSummaryTag", { value: filters.tag }))
  if (filters.studio) entries.push(t("library.savedViewSummaryStudio", { value: filters.studio }))
  if (filters.playState && filters.playState !== "all") {
    entries.push(t(`library.savedViewPlay.${filters.playState}`))
  }
  if (filters.userRating !== undefined) {
    entries.push(t("library.savedViewRatingValue", { value: filters.userRating }))
  }
  if (filters.resolution) entries.push(filters.resolution.toUpperCase())
  if (filters.addedWithinDays) {
    entries.push(t("library.savedViewAddedDays", { days: filters.addedWithinDays }))
  }
  return entries.join(" · ") || t("library.savedViewAllLibrary")
}
</script>

<template>
  <div
    data-library-saved-view-controls
    class="flex min-w-0 flex-wrap items-center justify-end gap-2"
  >
    <Popover v-if="mode !== 'trash'">
      <PopoverTrigger as-child>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="min-h-11 rounded-xl sm:min-h-8"
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
        class="w-[min(22rem,calc(100vw-2rem))] rounded-2xl border-border/70"
      >
        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-1">
            <p class="text-sm font-semibold text-foreground">
              {{ t("library.savedViewFiltersTitle") }}
            </p>
            <p class="text-xs leading-relaxed text-muted-foreground">
              {{ t("library.savedViewFiltersDescription") }}
            </p>
          </div>

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
                :model-value="userRating === undefined ? 'any' : String(userRating)"
                @update:model-value="onRatingChange"
              >
                <SelectTrigger class="w-full rounded-xl">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="any">{{ t("library.savedViewAny") }}</SelectItem>
                    <SelectItem v-for="value in [5, 4, 3, 2, 1, 0]" :key="value" :value="String(value)">
                      {{ t("library.savedViewRatingValue", { value }) }}
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
      </PopoverContent>
    </Popover>

    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="min-h-11 rounded-xl sm:min-h-8"
          :aria-label="t('library.savedViews')"
        >
          <Bookmark data-icon="inline-start" aria-hidden="true" />
          {{ t("library.savedViews") }}
          <Badge v-if="savedViews.length > 0" variant="secondary">{{ savedViews.length }}</Badge>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" class="w-64 rounded-xl">
        <DropdownMenuGroup>
          <DropdownMenuItem :disabled="savedViews.length >= 50 || busy" @click="openCreateDialog">
            <Save aria-hidden="true" />
            {{ t("library.savedViewSaveCurrent") }}
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuLabel v-if="savedViews.length === 0" class="font-normal text-muted-foreground">
          {{ t("library.savedViewEmpty") }}
        </DropdownMenuLabel>
        <DropdownMenuGroup v-else>
          <DropdownMenuSub v-for="(item, index) in savedViews" :key="item.id">
            <DropdownMenuSubTrigger>
              <Check v-if="activeSavedViewId === item.id" aria-hidden="true" />
              <Bookmark v-else aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate">{{ item.name }}</span>
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent class="w-60 rounded-xl">
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
                <DropdownMenuItem :disabled="index === 0 || busy" @click="moveSavedView(item, -1)">
                  <ChevronUp aria-hidden="true" />
                  {{ t("library.savedViewMoveUp") }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  :disabled="index === savedViews.length - 1 || busy"
                  @click="moveSavedView(item, 1)"
                >
                  <ChevronDown aria-hidden="true" />
                  {{ t("library.savedViewMoveDown") }}
                </DropdownMenuItem>
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
  </div>

  <Dialog v-model:open="editDialogOpen">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>
          {{
            editMode === "create"
              ? t("library.savedViewCreateTitle")
              : t("library.savedViewRenameTitle")
          }}
        </DialogTitle>
        <DialogDescription class="text-pretty">
          {{
            editMode === "create"
              ? t("library.savedViewCreateDescription")
              : t("library.savedViewRenameDescription")
          }}
        </DialogDescription>
      </DialogHeader>
      <label class="flex flex-col gap-2 text-sm font-medium text-foreground">
        {{ t("library.savedViewName") }}
        <Input
          v-model="nameDraft"
          maxlength="40"
          autocomplete="off"
          :placeholder="t('library.savedViewNamePlaceholder')"
          @keydown.enter.prevent="submitEdit"
        />
      </label>
      <p v-if="editMode === 'create'" class="text-xs leading-relaxed text-muted-foreground">
        {{ filterSummary(currentFilters) }}
      </p>
      <DialogFooter class="gap-3">
        <DialogClose as-child>
          <Button type="button" variant="outline" :disabled="busy">
            {{ t("common.cancel") }}
          </Button>
        </DialogClose>
        <Button type="button" :disabled="busy || !nameDraft.trim()" @click="submitEdit">
          <LoaderCircle v-if="busy" data-icon="inline-start" class="animate-spin" aria-hidden="true" />
          {{ editMode === "create" ? t("library.savedViewSave") : t("library.savedViewRename") }}
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

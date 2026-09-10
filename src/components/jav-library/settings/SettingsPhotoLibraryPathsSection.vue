<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { FolderArchive } from "lucide-vue-next"
import type { PhotoLibrarySetting } from "@/domain/photo/types"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import SettingsLibraryPathAddDialog from "@/components/jav-library/settings/SettingsLibraryPathAddDialog.vue"
import SettingsPhotoLibraryPathActions from "@/components/jav-library/settings/SettingsPhotoLibraryPathActions.vue"

const props = defineProps<{
  paths: readonly PhotoLibrarySetting[]
  defaultImportLibraryPathId: string
  addPathDialogOpen: boolean
  newPath: string
  newPathTitle: string
  pickDirectoryBusy: boolean
  directoryHintDisplay: string
  pathAddError: string
  addBusy: boolean
  canSaveNewPath: boolean
  defaultSaving: boolean
  scanPathBusy: string | null
  dialogContentClass: string
}>()

const emit = defineEmits<{
  "update:addPathDialogOpen": [open: boolean]
  "update:newPath": [path: string]
  "update:newPathTitle": [title: string]
  clearError: []
  browse: []
  submit: []
  scanPath: [path: PhotoLibrarySetting]
  removePath: [id: string]
  changeDefaultImportPath: [id: string]
}>()

const { t } = useI18n()

const defaultImportPathSelectValue = computed(() =>
  props.paths.some((path) => path.id === props.defaultImportLibraryPathId)
    ? props.defaultImportLibraryPathId
    : undefined,
)

const selectedDefaultImportPath = computed(() => {
  const id = defaultImportPathSelectValue.value
  if (!id) return undefined
  return props.paths.find((path) => path.id === id)
})

function defaultImportPathTriggerLabel(path: PhotoLibrarySetting): string {
  const title = path.title.trim()
  if (!title || title === path.path) {
    return path.path
  }
  return `${title} · ${path.path}`
}

function onDefaultChange(value: unknown) {
  if (typeof value !== "string" || value === props.defaultImportLibraryPathId) return
  emit("changeDefaultImportPath", value)
}
</script>

<template>
  <div
    data-photo-paths
    class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
  >
    <div class="flex flex-col gap-2">
      <div class="flex items-center gap-2">
        <FolderArchive class="size-4 text-primary" aria-hidden="true" />
        <p class="text-sm font-semibold text-foreground">
          {{ t("settings.photoLibraryPathsTitle") }}
        </p>
      </div>
      <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
        {{ t("settings.photoLibraryPathsDesc") }}
      </p>
    </div>

    <div
      class="flex flex-col gap-3 rounded-lg border border-border/50 bg-background/30 p-3 sm:flex-row sm:items-center sm:justify-between"
      :aria-busy="props.defaultSaving"
    >
      <div class="flex min-w-0 flex-col gap-1">
        <p class="text-sm font-medium text-foreground">
          {{ t("settings.photoDefaultImportPath") }}
        </p>
        <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
          {{ t("settings.photoDefaultImportPathDesc") }}
        </p>
      </div>
      <Select
        :model-value="defaultImportPathSelectValue"
        :disabled="props.paths.length === 0 || props.defaultSaving"
        @update:model-value="onDefaultChange"
      >
        <SelectTrigger
          size="sm"
          class="h-9 w-full min-w-0 rounded-xl border-border/50 sm:w-72 sm:shrink-0"
          :aria-label="t('settings.photoDefaultImportPath')"
        >
          <SelectValue :placeholder="t('settings.photoDefaultImportPathNone')">
            <span
              v-if="selectedDefaultImportPath"
              class="block min-w-0 flex-1 truncate text-left"
            >
              {{ defaultImportPathTriggerLabel(selectedDefaultImportPath) }}
            </span>
          </SelectValue>
        </SelectTrigger>
        <SelectContent align="end" class="rounded-xl border-border/50">
          <SelectItem
            v-for="path in props.paths"
            :key="path.id"
            class="rounded-lg"
            :value="path.id"
          >
            <span class="flex min-w-0 flex-col gap-0.5">
              <span class="truncate text-sm">{{ path.title || path.path }}</span>
              <span class="truncate font-mono text-xs text-muted-foreground">
                {{ path.path }}
              </span>
            </span>
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div v-if="props.paths.length > 0" class="flex flex-col gap-3">
      <div
        v-for="path in props.paths"
        :key="path.id"
        class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="min-w-0">
          <p class="truncate text-sm font-medium text-foreground">{{ path.title || path.path }}</p>
          <p class="break-all text-sm text-muted-foreground">{{ path.path }}</p>
        </div>
        <SettingsPhotoLibraryPathActions
          :path="path"
          :scan-busy="props.scanPathBusy === path.path"
          @scan="emit('scanPath', $event)"
          @remove="emit('removePath', $event)"
        />
      </div>
    </div>
    <p v-else class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
      {{ t("settings.photoLibraryPathsEmpty") }}
    </p>

    <div class="flex flex-wrap justify-start gap-2 pt-1">
      <SettingsLibraryPathAddDialog
        :open="props.addPathDialogOpen"
        :new-path="props.newPath"
        :new-path-title="props.newPathTitle"
        :pick-directory-busy="props.pickDirectoryBusy"
        :directory-hint-display="props.directoryHintDisplay"
        :path-add-error="props.pathAddError"
        :add-busy="props.addBusy"
        :can-save-new-path="props.canSaveNewPath"
        :content-class="props.dialogContentClass"
        :trigger-label="t('settings.photoLibraryPathAdd')"
        :dialog-title="t('settings.photoLibraryPathDialogTitle')"
        :dialog-description="t('settings.photoLibraryPathDialogDesc')"
        :path-label="t('settings.photoLibraryPathLabel')"
        :path-placeholder="t('settings.photoLibraryPathPlaceholder')"
        path-input-id="new-photo-lib-path"
        :title-label="t('settings.photoLibraryPathTitleLabel')"
        :title-placeholder="t('settings.photoLibraryPathTitlePlaceholder')"
        title-input-id="new-photo-lib-title"
        :example-paths="['D:\\Photos', '/home/user/Photos']"
        @update:open="emit('update:addPathDialogOpen', $event)"
        @update:new-path="emit('update:newPath', $event)"
        @update:new-path-title="emit('update:newPathTitle', $event)"
        @clear-error="emit('clearError')"
        @browse="emit('browse')"
        @submit="emit('submit')"
      />
    </div>
  </div>
</template>

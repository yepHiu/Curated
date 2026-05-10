<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { CircleHelp, Database, RefreshCw } from "lucide-vue-next"
import type { LibraryPathDTO, LibraryPathStorageStatusDTO } from "@/api/types"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  TooltipContent,
  TooltipPortal,
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
} from "reka-ui"
import SettingsLibraryPathAddDialog from "@/components/jav-library/settings/SettingsLibraryPathAddDialog.vue"
import SettingsLibraryPathList from "@/components/jav-library/settings/SettingsLibraryPathList.vue"
import SettingsLibraryPathRemoveDialog from "@/components/jav-library/settings/SettingsLibraryPathRemoveDialog.vue"
import SettingsLibraryPathToolbar from "@/components/jav-library/settings/SettingsLibraryPathToolbar.vue"

const props = defineProps<{
  scanFeedbackError: string
  paths: readonly LibraryPathDTO[]
  storageStatuses: readonly LibraryPathStorageStatusDTO[]
  storageStatusBusy: boolean
  storageStatusError: string
  storageBindingBusy: string | null
  defaultImportLibraryPathId: string
  defaultImportPathSaving: boolean
  defaultImportPathError: string
  batchMode: boolean
  hasMetadataPathSelection: boolean
  metadataRefreshBusy: boolean
  metadataRefreshSuccess: string
  metadataRefreshError: string
  selectedMetadataRefreshPaths: readonly string[]
  removePathDialogOpen: boolean
  removePathPending: LibraryPathDTO | null
  removePathBusy: boolean
  editingLibraryPathId: string | null
  editLibraryTitleDraft: string
  editTitleBusy: boolean
  editTitleError: string
  revealPathBusy: string | null
  scanPathBusy: string | null
  addPathDialogOpen: boolean
  newPath: string
  newPathTitle: string
  pickDirectoryBusy: boolean
  directoryHintDisplay: string
  pathAddError: string
  addBusy: boolean
  canSaveNewPath: boolean
  dialogContentClass: string
}>()

const emit = defineEmits<{
  "update:removePathDialogOpen": [open: boolean]
  "update:editLibraryTitleDraft": [title: string]
  "update:addPathDialogOpen": [open: boolean]
  "update:newPath": [path: string]
  "update:newPathTitle": [title: string]
  enterBatchMode: []
  selectAll: []
  clearSelection: []
  refreshMetadata: []
  exitBatchMode: []
  confirmRemove: []
  saveTitle: [id: string]
  cancelEdit: []
  toggleMetadataPathSelection: [path: string]
  reveal: [path: LibraryPathDTO]
  edit: [path: LibraryPathDTO]
  rescan: [path: LibraryPathDTO]
  checkStorage: []
  rebindStorage: [path: LibraryPathDTO]
  remove: [path: LibraryPathDTO]
  changeDefaultImportLibraryPath: [id: string]
  clearError: []
  browse: []
  submit: []
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
  return props.paths.find((p) => p.id === id)
})

/** Reka SelectItemText flattens nested spans into one string for the trigger; keep separator here. */
function defaultImportPathTriggerLabel(path: LibraryPathDTO): string {
  const title = (path.title ?? "").trim()
  const diskPath = path.path
  if (!title || title === diskPath) {
    return diskPath
  }
  return `${title} · ${diskPath}`
}

function onDefaultImportPathChange(value: unknown) {
  if (typeof value !== "string" || value === props.defaultImportLibraryPathId) return
  emit("changeDefaultImportLibraryPath", value)
}
</script>

<template>
  <div class="flex w-full flex-col gap-6">
    <p
      v-if="scanFeedbackError"
      class="rounded-2xl border border-destructive/35 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      role="alert"
    >
      {{ scanFeedbackError }}
    </p>

    <div class="break-inside-avoid">
      <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="space-y-3 pb-2">
          <CardTitle class="flex items-center gap-2.5 text-lg font-semibold tracking-tight">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <Database class="size-[1.15rem]" />
            </span>
            {{ t("settings.storageCardTitle") }}
          </CardTitle>
          <CardDescription
            class="text-xs leading-relaxed text-pretty text-muted-foreground"
          >
            {{ t("settings.storageCardDesc") }}
          </CardDescription>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-2">
          <div
            class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-start sm:justify-between"
            :aria-busy="defaultImportPathSaving"
          >
            <div class="min-w-0 space-y-1">
              <div class="flex flex-wrap items-center gap-1.5">
                <p class="text-sm font-medium text-foreground">
                  {{ t("settings.defaultImportPathLabel") }}
                </p>
                <TooltipProvider :delay-duration="280">
                  <TooltipRoot>
                    <TooltipTrigger as-child>
                      <button
                        type="button"
                        data-default-import-path-help
                        class="inline-flex size-8 shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-muted/60 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                      >
                        <CircleHelp class="size-4" aria-hidden="true" />
                        <span class="sr-only">{{
                          t("settings.defaultImportPathHelpAria")
                        }}</span>
                      </button>
                    </TooltipTrigger>
                    <TooltipPortal>
                      <TooltipContent
                        side="top"
                        :side-offset="6"
                        class="z-50 max-w-[min(22rem,calc(100vw-2rem))] rounded-xl border border-border/50 bg-popover px-3 py-2 text-xs leading-relaxed text-pretty text-popover-foreground shadow-lg"
                      >
                        {{ t("settings.defaultImportPathDesc") }}
                      </TooltipContent>
                    </TooltipPortal>
                  </TooltipRoot>
                </TooltipProvider>
              </div>
              <p
                v-if="defaultImportPathSaving"
                class="text-xs text-muted-foreground motion-safe:animate-pulse"
              >
                {{ t("common.saving") }}
              </p>
            </div>
            <Select
              :model-value="defaultImportPathSelectValue"
              :disabled="defaultImportPathSaving || paths.length === 0"
              @update:model-value="onDefaultImportPathChange"
            >
              <SelectTrigger
                size="sm"
                class="h-9 w-full min-w-0 rounded-xl border-border/50 sm:w-72 sm:shrink-0"
                :aria-label="t('settings.defaultImportPathLabel')"
              >
                <SelectValue
                  :placeholder="
                    paths.length > 0
                      ? t('settings.defaultImportPathPlaceholder')
                      : t('settings.defaultImportPathNone')
                  "
                >
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
                  v-for="path in paths"
                  :key="`default-import-path-${path.id}`"
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
          <p v-if="defaultImportPathError" class="text-sm text-destructive" role="alert">
            {{ defaultImportPathError }}
          </p>

          <SettingsLibraryPathToolbar
            :batch-mode="batchMode"
            :library-paths-count="paths.length"
            :has-metadata-path-selection="hasMetadataPathSelection"
            :metadata-refresh-busy="metadataRefreshBusy"
            @enter-batch-mode="emit('enterBatchMode')"
            @select-all="emit('selectAll')"
            @clear-selection="emit('clearSelection')"
            @refresh-metadata="emit('refreshMetadata')"
            @exit-batch-mode="emit('exitBatchMode')"
          />

          <p v-if="storageStatusError" class="text-sm text-destructive" role="alert">
            {{ storageStatusError }}
          </p>

          <SettingsLibraryPathRemoveDialog
            :open="removePathDialogOpen"
            :pending="removePathPending"
            :busy="removePathBusy"
            :content-class="dialogContentClass"
            @update:open="emit('update:removePathDialogOpen', $event)"
            @confirm="emit('confirmRemove')"
          />

          <p v-if="metadataRefreshSuccess" class="text-sm text-primary">
            {{ metadataRefreshSuccess }}
          </p>
          <p
            v-if="metadataRefreshError"
            class="text-sm text-destructive"
            role="alert"
          >
            {{ metadataRefreshError }}
          </p>

          <SettingsLibraryPathList
            :edit-library-title-draft="editLibraryTitleDraft"
            :paths="paths"
            :storage-statuses="storageStatuses"
            :storage-binding-busy="storageBindingBusy"
            :batch-mode="batchMode"
            :selected-metadata-refresh-paths="selectedMetadataRefreshPaths"
            :editing-library-path-id="editingLibraryPathId"
            :edit-title-busy="editTitleBusy"
            :edit-title-error="editTitleError"
            :reveal-path-busy="revealPathBusy"
            :scan-path-busy="scanPathBusy"
            @update:edit-library-title-draft="emit('update:editLibraryTitleDraft', $event)"
            @save-title="emit('saveTitle', $event)"
            @cancel-edit="emit('cancelEdit')"
            @toggle-metadata-path-selection="emit('toggleMetadataPathSelection', $event)"
            @reveal="emit('reveal', $event)"
            @edit="emit('edit', $event)"
            @rescan="emit('rescan', $event)"
            @rebind-storage="emit('rebindStorage', $event)"
            @remove="emit('remove', $event)"
          />

          <div class="flex flex-wrap justify-start gap-2 pt-1">
            <SettingsLibraryPathAddDialog
              :open="addPathDialogOpen"
              :new-path="newPath"
              :new-path-title="newPathTitle"
              :pick-directory-busy="pickDirectoryBusy"
              :directory-hint-display="directoryHintDisplay"
              :path-add-error="pathAddError"
              :add-busy="addBusy"
              :can-save-new-path="canSaveNewPath"
              :content-class="dialogContentClass"
              @update:open="emit('update:addPathDialogOpen', $event)"
              @update:new-path="emit('update:newPath', $event)"
              @update:new-path-title="emit('update:newPathTitle', $event)"
              @clear-error="emit('clearError')"
              @browse="emit('browse')"
              @submit="emit('submit')"
            />
            <Button
              type="button"
              variant="outline"
              class="h-8 min-w-28 rounded-2xl px-3"
              :disabled="storageStatusBusy"
              data-check-storage-status
              @click="emit('checkStorage')"
            >
              <RefreshCw
                data-icon="inline-start"
                :class="storageStatusBusy ? 'animate-spin' : ''"
                aria-hidden="true"
              />
              {{
                storageStatusBusy
                  ? t("settings.storageStatusChecking")
                  : t("settings.storageStatusRecheck")
              }}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

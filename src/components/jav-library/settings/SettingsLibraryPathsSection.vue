<script setup lang="ts">
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { useLibraryPathAccess } from "@/composables/use-library-path-access"
import SettingsReadOnlyLibraryPaths from "./SettingsReadOnlyLibraryPaths.vue"
import { useI18n } from "vue-i18n"
import { Database, Download, RefreshCw } from "lucide-vue-next"
import type { LibraryPathDTO, LibraryPathStorageStatusDTO } from "@/api/types"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import SettingsLibraryPathAddDialog from "@/components/jav-library/settings/SettingsLibraryPathAddDialog.vue"
import SettingsLibraryPathList from "@/components/jav-library/settings/SettingsLibraryPathList.vue"
import SettingsLibraryPathRemoveDialog from "@/components/jav-library/settings/SettingsLibraryPathRemoveDialog.vue"

defineProps<{
  scanFeedbackError: string
  paths: readonly LibraryPathDTO[]
  storageStatuses: readonly LibraryPathStorageStatusDTO[]
  storageStatusBusy: boolean
  storageStatusError: string
  storageBindingBusy: string | null
  movieCsvExportBusy: boolean
  movieCsvExportError: string
  defaultImportLibraryPathId: string
  defaultImportPathSaving: boolean
  defaultImportPathError: string
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
  confirmRemove: []
  saveTitle: [id: string]
  cancelEdit: []
  reveal: [path: LibraryPathDTO]
  edit: [path: LibraryPathDTO]
  rescan: [path: LibraryPathDTO]
  exportMoviesCsv: []
  checkStorage: []
  rebindStorage: [path: LibraryPathDTO]
  remove: [path: LibraryPathDTO]
  changeDefaultImportLibraryPath: [id: string]
  clearError: []
  browse: []
  submit: []
}>()

const { t } = useI18n()
const { canManagePaths } = useLibraryPathAccess()

</script>

<template>
  <SettingsReadOnlyLibraryPaths
    v-if="!canManagePaths"
    :title="t('settings.storageCardTitle')"
    :paths="paths"
    :default-import-library-path-id="defaultImportLibraryPathId"
    :storage-statuses="storageStatuses"
  />
  <div v-else class="flex w-full flex-col gap-6">
    <p
      v-if="scanFeedbackError"
      class="rounded-2xl border border-destructive/35 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      role="alert"
    >
      {{ scanFeedbackError }}
    </p>

    <div class="break-inside-avoid">
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
          <span
            class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
            aria-hidden="true"
          >
            <Database class="size-[1.15rem]" />
          </span>
          <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
            <span>{{ t("settings.storageCardTitle") }}</span>
            <SettingsScopeBadge scope="server" />
          </CardTitle>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <p v-if="defaultImportPathError" class="text-sm text-destructive" role="alert">
            {{ defaultImportPathError }}
          </p>

          <p v-if="storageStatusError" class="text-sm text-destructive" role="alert">
            {{ storageStatusError }}
          </p>
          <p v-if="movieCsvExportError" class="text-sm text-destructive" role="alert">
            {{ movieCsvExportError }}
          </p>

          <SettingsLibraryPathRemoveDialog
            :open="removePathDialogOpen"
            :pending="removePathPending"
            :busy="removePathBusy"
            :content-class="dialogContentClass"
            @update:open="emit('update:removePathDialogOpen', $event)"
            @confirm="emit('confirmRemove')"
          />

          <SettingsLibraryPathList
            :edit-library-title-draft="editLibraryTitleDraft"
            :paths="paths"
            :storage-statuses="storageStatuses"
            :storage-binding-busy="storageBindingBusy"
            :default-import-library-path-id="defaultImportLibraryPathId"
            :default-import-path-saving="defaultImportPathSaving"
            :editing-library-path-id="editingLibraryPathId"
            :edit-title-busy="editTitleBusy"
            :edit-title-error="editTitleError"
            :reveal-path-busy="revealPathBusy"
            :scan-path-busy="scanPathBusy"
            @update:edit-library-title-draft="emit('update:editLibraryTitleDraft', $event)"
            @save-title="emit('saveTitle', $event)"
            @cancel-edit="emit('cancelEdit')"
            @change-default-import-library-path="emit('changeDefaultImportLibraryPath', $event)"
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
              :disabled="movieCsvExportBusy"
              data-export-movie-csv
              @click="emit('exportMoviesCsv')"
            >
              <Download data-icon="inline-start" aria-hidden="true" />
              {{
                movieCsvExportBusy
                  ? t("settings.movieCsvExporting")
                  : t("settings.movieCsvExport")
              }}
            </Button>
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

<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Database } from "lucide-vue-next"
import type { LibraryPathDTO } from "@/api/types"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import SettingsLibraryPathAddDialog from "@/components/jav-library/settings/SettingsLibraryPathAddDialog.vue"
import SettingsLibraryPathList from "@/components/jav-library/settings/SettingsLibraryPathList.vue"
import SettingsLibraryPathRemoveDialog from "@/components/jav-library/settings/SettingsLibraryPathRemoveDialog.vue"
import SettingsLibraryPathToolbar from "@/components/jav-library/settings/SettingsLibraryPathToolbar.vue"

defineProps<{
  scanFeedbackError: string
  paths: readonly LibraryPathDTO[]
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
  remove: [path: LibraryPathDTO]
  clearError: []
  browse: []
  submit: []
}>()

const { t } = useI18n()
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
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

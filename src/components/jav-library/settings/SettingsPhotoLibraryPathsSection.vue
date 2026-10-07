<script setup lang="ts">
import { useLibraryPathAccess } from "@/composables/use-library-path-access"
import SettingsReadOnlyLibraryPaths from "./SettingsReadOnlyLibraryPaths.vue"
import { useI18n } from "vue-i18n"
import type { PhotoLibrarySetting } from "@/domain/photo/types"
import { cn } from "@/lib/utils"
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
const { canManagePaths } = useLibraryPathAccess()

</script>

<template>
  <SettingsReadOnlyLibraryPaths
    v-if="!canManagePaths"
    :title="t('settings.photoLibraryPathsTitle')"
    :paths="paths"
    :default-import-library-path-id="defaultImportLibraryPathId"
  />
  <div v-else
    data-photo-paths
    class="flex flex-col gap-3"
  >
    <div v-if="props.paths.length > 0" class="flex flex-col gap-2" :aria-busy="props.defaultSaving">
      <div
        v-for="path in props.paths"
        :key="path.id"
        :class="cn(
          'flex min-w-0 items-center gap-2 rounded-lg border bg-muted/5 px-3 py-1.5',
          path.id === defaultImportLibraryPathId ? 'border-success/70' : 'border-border/50',
        )"
        :data-library-path="path.id"
        :data-default-import-path="path.id === defaultImportLibraryPathId || undefined"
      >
        <div class="flex min-w-0 flex-1 items-center gap-3" :title="path.path">
          <p v-if="path.title && path.title !== path.path" class="max-w-[35%] truncate text-sm font-medium" :title="path.title">{{ path.title }}</p>
          <p class="min-w-0 flex-1 truncate text-sm text-muted-foreground">{{ path.path }}</p>
        </div>
        <span v-if="path.id === defaultImportLibraryPathId" class="sr-only">{{ t('settings.defaultImportPathLabel') }}</span>
        <SettingsPhotoLibraryPathActions
          :path="path"
          :is-default="path.id === defaultImportLibraryPathId"
          :default-import-path-saving="props.defaultSaving"
          :scan-busy="props.scanPathBusy === path.path"
          @set-default="emit('changeDefaultImportPath', $event)"
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

<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { LibraryPathDTO, LibraryPathStorageStatusDTO } from "@/api/types"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import SettingsLibraryPathActions from "./SettingsLibraryPathActions.vue"

const props = defineProps<{
  paths: readonly LibraryPathDTO[]
  storageStatuses: readonly LibraryPathStorageStatusDTO[]
  storageBindingBusy: string | null
  defaultImportLibraryPathId: string
  defaultImportPathSaving: boolean
  editingLibraryPathId: string | null
  editLibraryTitleDraft: string
  editTitleBusy: boolean
  editTitleError: string
  revealPathBusy: string | null
  scanPathBusy: string | null
}>()

const emit = defineEmits<{
  "update:editLibraryTitleDraft": [title: string]
  saveTitle: [id: string]
  cancelEdit: []
  changeDefaultImportLibraryPath: [id: string]
  reveal: [path: LibraryPathDTO]
  edit: [path: LibraryPathDTO]
  rescan: [path: LibraryPathDTO]
  rebindStorage: [path: LibraryPathDTO]
  remove: [path: LibraryPathDTO]
}>()

const { t } = useI18n()

const storageStatusByPathId = computed(() => {
  const map = new Map<string, LibraryPathStorageStatusDTO>()
  for (const status of props.storageStatuses) {
    if (status.libraryPathId) {
      map.set(status.libraryPathId, status)
    }
  }
  return map
})

function updateTitleDraft(value: unknown) {
  emit("update:editLibraryTitleDraft", typeof value === "string" ? value : String(value ?? ""))
}

function storageStatusFor(path: LibraryPathDTO): LibraryPathStorageStatusDTO | undefined {
  return storageStatusByPathId.value.get(path.id)
}

function storageStatusLabelKey(status: LibraryPathStorageStatusDTO["status"]): string {
  switch (status) {
    case "online":
      return "settings.storageStatusOnline"
    case "offline":
      return "settings.storageStatusOffline"
    case "volume_mismatch":
      return "settings.storageStatusVolumeMismatch"
    case "path_missing":
      return "settings.storageStatusPathMissing"
    case "permission_denied":
      return "settings.storageStatusPermissionDenied"
    default:
      return "settings.storageStatusUnknown"
  }
}

function storageStatusAllowsRescan(path: LibraryPathDTO): boolean {
  const status = storageStatusFor(path)
  return !status || status.canRescan
}

function canRebindStorage(status?: LibraryPathStorageStatusDTO): boolean {
  return status?.status === "volume_mismatch"
}
</script>

<template>
  <div class="flex flex-col gap-2" :aria-busy="defaultImportPathSaving">
    <div
      v-for="path in paths"
      :key="path.id"
      :class="cn(
        'min-w-0 rounded-lg border bg-muted/5 px-3 py-1.5',
        path.id === defaultImportLibraryPathId ? 'border-success/70' : 'border-border/50',
        editingLibraryPathId === path.id && 'py-3',
      )"
      :data-library-path="path.id"
      :data-default-import-path="path.id === defaultImportLibraryPathId || undefined"
    >
      <template v-if="editingLibraryPathId === path.id">
        <div class="flex flex-col gap-3">
          <div class="flex flex-col gap-3">
            <p class="text-xs font-medium text-muted-foreground">{{ t("settings.pathReadonly") }}</p>
            <p class="break-all font-mono text-sm text-muted-foreground">{{ path.path }}</p>
          </div>
          <div class="flex flex-col gap-3">
            <SettingsHint :text="t('settings.editTitleHint')">
              <label class="text-sm font-medium" :for="`edit-title-${path.id}`">
                {{ t("settings.pathTitleLabel") }}
              </label>
            </SettingsHint>
            <Input
              :id="`edit-title-${path.id}`"
              :model-value="editLibraryTitleDraft"
              class="rounded-xl"
              :placeholder="t('settings.displayName')"
              autocomplete="off"
              @update:model-value="updateTitleDraft"
              @keydown.enter.prevent="emit('saveTitle', path.id)"
            />
            <p v-if="editTitleError" class="text-sm text-destructive">
              {{ editTitleError }}
            </p>
          </div>
          <div class="flex flex-wrap gap-3">
            <Button
              type="button"
              class="rounded-2xl"
              :disabled="editTitleBusy"
              :data-save-library-path-title="path.id"
              @click="emit('saveTitle', path.id)"
            >
              {{ editTitleBusy ? t("common.saving") : t("settings.saveTitle") }}
            </Button>
            <Button
              type="button"
              variant="outline"
              class="rounded-2xl"
              :disabled="editTitleBusy"
              :data-cancel-library-path-title="path.id"
              @click="emit('cancelEdit')"
            >
              {{ t("common.cancel") }}
            </Button>
          </div>
        </div>
      </template>
      <template v-else>
        <div class="flex min-w-0 items-center gap-2">
          <div class="flex min-w-0 flex-1 items-center gap-3" :title="path.path">
            <p v-if="path.title && path.title !== path.path" class="max-w-[35%] truncate text-sm font-medium" :title="path.title">{{ path.title }}</p>
            <p class="min-w-0 flex-1 truncate text-sm text-muted-foreground">{{ path.path }}</p>
          </div>
          <span v-if="path.id === defaultImportLibraryPathId" class="sr-only">{{ t('settings.defaultImportPathLabel') }}</span>
          <Badge
            v-if="storageStatusFor(path)"
            :variant="storageStatusFor(path)!.status === 'online' ? 'success' : storageStatusFor(path)!.status === 'unknown' ? 'secondary' : 'warning'"
            :title="t(`settings.storageStatusMessages.${storageStatusFor(path)!.status}`)"
            :aria-label="t(`settings.storageStatusMessages.${storageStatusFor(path)!.status}`)"
          >
            {{ t(storageStatusLabelKey(storageStatusFor(path)!.status)) }}
          </Badge>
          <div class="library-path-toolbar shrink-0">
            <SettingsLibraryPathActions
              :path="path"
              :is-default="path.id === defaultImportLibraryPathId"
              :default-import-path-saving="defaultImportPathSaving"
              :reveal-busy="revealPathBusy === path.id"
              :scan-busy="scanPathBusy === path.path"
              :scan-disabled="!storageStatusAllowsRescan(path)"
              :can-rebind="canRebindStorage(storageStatusFor(path))"
              :rebind-busy="storageBindingBusy === path.id"
              @set-default="emit('changeDefaultImportLibraryPath', $event)"
              @rebind-storage="emit('rebindStorage', $event)"
              @reveal="emit('reveal', $event)"
              @edit="emit('edit', $event)"
              @rescan="emit('rescan', $event)"
              @remove="emit('remove', $event)"
            />
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

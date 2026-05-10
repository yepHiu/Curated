<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { LibraryPathDTO, LibraryPathStorageStatusDTO } from "@/api/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import SettingsLibraryPathActions from "./SettingsLibraryPathActions.vue"

const props = defineProps<{
  paths: readonly LibraryPathDTO[]
  storageStatuses: readonly LibraryPathStorageStatusDTO[]
  storageBindingBusy: string | null
  batchMode: boolean
  selectedMetadataRefreshPaths: readonly string[]
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
  toggleMetadataPathSelection: [path: string]
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

function isMetadataPathChecked(path: string): boolean {
  return props.selectedMetadataRefreshPaths.includes(path)
}

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

function storageStatusClass(status: LibraryPathStorageStatusDTO["status"]): string {
  if (status === "online") {
    return "border-emerald-500/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
  }
  if (status === "unknown") {
    return "border-border bg-muted text-muted-foreground"
  }
  return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300"
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
  <div class="flex flex-col gap-3">
    <div
      v-for="path in paths"
      :key="path.id"
      class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
    >
      <template v-if="editingLibraryPathId === path.id">
        <div class="flex flex-col gap-3">
          <div class="flex flex-col gap-3">
            <p class="text-xs font-medium text-muted-foreground">{{ t("settings.pathReadonly") }}</p>
            <p class="break-all font-mono text-sm text-muted-foreground">{{ path.path }}</p>
          </div>
          <div class="flex flex-col gap-3">
            <label class="text-sm font-medium" :for="`edit-title-${path.id}`">{{
              t("settings.pathTitleLabel")
            }}</label>
            <Input
              :id="`edit-title-${path.id}`"
              :model-value="editLibraryTitleDraft"
              class="rounded-xl"
              :placeholder="t('settings.displayName')"
              autocomplete="off"
              @update:model-value="updateTitleDraft"
              @keydown.enter.prevent="emit('saveTitle', path.id)"
            />
            <p class="text-xs text-muted-foreground">
              {{ t("settings.editTitleHint") }}
            </p>
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
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-1 items-start gap-3">
            <input
              v-if="batchMode"
              type="checkbox"
              class="mt-1 size-4 shrink-0 cursor-pointer rounded border border-input accent-primary"
              :checked="isMetadataPathChecked(path.path)"
              :aria-label="t('settings.includeInMetadataRefresh', { title: path.title })"
              @change="emit('toggleMetadataPathSelection', path.path)"
            />
            <div class="flex min-w-0 flex-1 flex-col gap-3">
              <div class="flex min-w-0 flex-wrap items-center gap-2">
                <p class="min-w-0 font-medium">{{ path.title }}</p>
                <span
                  v-if="storageStatusFor(path)"
                  class="inline-flex min-h-6 items-center rounded-full border px-2 py-0.5 text-xs font-medium"
                  :class="storageStatusClass(storageStatusFor(path)!.status)"
                >
                  {{ t(storageStatusLabelKey(storageStatusFor(path)!.status)) }}
                </span>
              </div>
              <p class="break-all text-sm text-muted-foreground">{{ path.path }}</p>
              <div
                v-if="storageStatusFor(path) && storageStatusFor(path)!.status !== 'online'"
                class="flex flex-col gap-2 rounded-lg border border-amber-500/20 bg-amber-500/5 px-3 py-2 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between"
              >
                <span class="min-w-0 leading-relaxed">
                  {{ storageStatusFor(path)!.message }}
                </span>
                <Button
                  v-if="canRebindStorage(storageStatusFor(path))"
                  type="button"
                  size="sm"
                  variant="outline"
                  class="h-8 shrink-0 rounded-lg"
                  :disabled="storageBindingBusy === path.id"
                  :data-rebind-storage="path.id"
                  @click="emit('rebindStorage', path)"
                >
                  {{
                    storageBindingBusy === path.id
                      ? t("settings.storageStatusRebinding")
                      : t("settings.storageStatusRebind")
                  }}
                </Button>
              </div>
            </div>
          </div>
          <div class="library-path-toolbar flex flex-wrap items-center gap-2">
            <SettingsLibraryPathActions
              :path="path"
              :reveal-busy="revealPathBusy === path.id"
              :scan-busy="scanPathBusy === path.path"
              :scan-disabled="!storageStatusAllowsRescan(path)"
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

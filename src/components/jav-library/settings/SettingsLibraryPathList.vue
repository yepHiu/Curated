<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { LibraryPathDTO } from "@/api/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import SettingsLibraryPathActions from "./SettingsLibraryPathActions.vue"

const props = defineProps<{
  paths: readonly LibraryPathDTO[]
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
  remove: [path: LibraryPathDTO]
}>()

const { t } = useI18n()

function isMetadataPathChecked(path: string): boolean {
  return props.selectedMetadataRefreshPaths.includes(path)
}

function updateTitleDraft(value: unknown) {
  emit("update:editLibraryTitleDraft", typeof value === "string" ? value : String(value ?? ""))
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
              <p class="font-medium">{{ path.title }}</p>
              <p class="break-all text-sm text-muted-foreground">{{ path.path }}</p>
            </div>
          </div>
          <div class="library-path-toolbar flex flex-wrap items-center gap-2">
            <SettingsLibraryPathActions
              :path="path"
              :reveal-busy="revealPathBusy === path.id"
              :scan-busy="scanPathBusy === path.path"
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

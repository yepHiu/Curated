<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { FolderArchive, Trash2 } from "lucide-vue-next"
import type { ComicLibrarySetting } from "@/domain/comic/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

defineProps<{
  paths: readonly ComicLibrarySetting[]
  defaultImportLibraryPathId: string
  addBusy: boolean
  defaultSaving: boolean
  error: string
}>()

const emit = defineEmits<{
  addPath: [path: string, title: string]
  removePath: [id: string]
  changeDefaultImportPath: [id: string]
}>()

const { t } = useI18n()
const newPath = ref("")
const newTitle = ref("")

function submitPath() {
  emit("addPath", newPath.value, newTitle.value)
}

function onDefaultChange(value: unknown) {
  if (typeof value === "string") {
    emit("changeDefaultImportPath", value)
  }
}
</script>

<template>
  <div
    data-comic-paths
    class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
  >
    <div class="flex flex-col gap-2">
      <div class="flex items-center gap-2">
        <FolderArchive class="size-4 text-primary" aria-hidden="true" />
        <p class="text-sm font-semibold text-foreground">
          {{ t("settings.comicLibraryPathsTitle") }}
        </p>
      </div>
      <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
        {{ t("settings.comicLibraryPathsDesc") }}
      </p>
    </div>

    <div class="grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(10rem,16rem)_auto]">
      <Input
        v-model="newPath"
        :placeholder="t('settings.comicLibraryPathPlaceholder')"
        :aria-label="t('settings.comicLibraryPathLabel')"
      />
      <Input
        v-model="newTitle"
        :placeholder="t('settings.comicLibraryPathTitlePlaceholder')"
        :aria-label="t('settings.comicLibraryPathTitleLabel')"
      />
      <Button
        type="button"
        size="sm"
        class="h-9"
        :disabled="addBusy"
        @click="submitPath"
      >
        {{ addBusy ? t("settings.comicLibraryPathAdding") : t("settings.comicLibraryPathAdd") }}
      </Button>
    </div>

    <p v-if="error" class="text-sm text-destructive">{{ error }}</p>

    <div
      class="flex flex-col gap-3 rounded-lg border border-border/50 bg-background/30 p-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <div class="flex min-w-0 flex-col gap-1">
        <p class="text-sm font-medium text-foreground">
          {{ t("settings.comicDefaultImportPath") }}
        </p>
        <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
          {{ t("settings.comicDefaultImportPathDesc") }}
        </p>
      </div>
      <Select
        :model-value="defaultImportLibraryPathId"
        :disabled="paths.length === 0 || defaultSaving"
        @update:model-value="onDefaultChange"
      >
        <SelectTrigger
          size="sm"
          class="h-9 w-full min-w-[12rem] rounded-xl border-border/50 sm:w-56"
          :aria-label="t('settings.comicDefaultImportPath')"
        >
          <SelectValue :placeholder="t('settings.comicDefaultImportPathNone')" />
        </SelectTrigger>
        <SelectContent align="end" class="rounded-xl border-border/50">
          <SelectItem
            v-for="path in paths"
            :key="path.id"
            :value="path.id"
          >
            {{ path.title || path.path }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div v-if="paths.length > 0" class="flex flex-col gap-2">
      <div
        v-for="path in paths"
        :key="path.id"
        class="flex flex-col gap-2 rounded-lg border border-border/50 bg-background/30 p-3 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="min-w-0">
          <p class="truncate text-sm font-medium text-foreground">{{ path.title || path.path }}</p>
          <p class="truncate text-xs text-muted-foreground">{{ path.path }}</p>
        </div>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          class="h-8 shrink-0"
          :aria-label="t('settings.comicLibraryPathRemove')"
          @click="emit('removePath', path.id)"
        >
          <Trash2 class="size-4" aria-hidden="true" />
          {{ t("settings.comicLibraryPathRemove") }}
        </Button>
      </div>
    </div>
    <p v-else class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
      {{ t("settings.comicLibraryPathEmpty") }}
    </p>
  </div>
</template>

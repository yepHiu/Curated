<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { FolderOpen, FolderPlus } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"

defineProps<{
  open: boolean
  newPath: string
  newPathTitle: string
  pickDirectoryBusy: boolean
  directoryHintDisplay: string
  pathAddError: string
  addBusy: boolean
  canSaveNewPath: boolean
  contentClass: string
}>()

const emit = defineEmits<{
  "update:open": [open: boolean]
  "update:newPath": [path: string]
  "update:newPathTitle": [title: string]
  clearError: []
  browse: []
  submit: []
}>()

const { t } = useI18n()

function updateNewPath(value: unknown) {
  emit("update:newPath", typeof value === "string" ? value : String(value ?? ""))
}

function updateNewPathTitle(value: unknown) {
  emit("update:newPathTitle", typeof value === "string" ? value : String(value ?? ""))
}
</script>

<template>
  <Dialog
    :open="open"
    @update:open="emit('update:open', $event)"
  >
    <DialogTrigger as-child>
      <Button type="button" class="rounded-2xl">
        <FolderPlus data-icon="inline-start" />
        {{ t("settings.addPath") }}
      </Button>
    </DialogTrigger>

    <DialogContent :class="contentClass">
      <DialogHeader>
        <DialogTitle>{{ t("settings.addPathDialogTitle") }}</DialogTitle>
        <DialogDescription>
          {{ t("settings.addPathDialogDesc") }}
          <span class="font-mono text-xs">D:\Media\JAV</span> 或
          <span class="font-mono text-xs">/home/user/Videos</span>。
        </DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-3">
          <label class="text-sm font-medium" for="new-lib-path">{{ t("settings.absolutePath") }}</label>
          <div class="flex flex-col gap-3 sm:flex-row sm:items-stretch">
            <Input
              id="new-lib-path"
              :model-value="newPath"
              class="rounded-xl sm:min-w-0 sm:flex-1"
              placeholder="D:\Media\JAV\Library"
              autocomplete="off"
              @update:model-value="updateNewPath"
              @input="emit('clearError')"
            />
            <Button
              type="button"
              variant="secondary"
              class="rounded-2xl sm:shrink-0"
              :disabled="pickDirectoryBusy"
              data-add-path-browse
              @click="emit('browse')"
            >
              <FolderOpen data-icon="inline-start" />
              {{ pickDirectoryBusy ? t("settings.picking") : t("settings.pickFolder") }}
            </Button>
          </div>
          <p
            v-if="directoryHintDisplay"
            class="text-sm leading-relaxed text-muted-foreground whitespace-pre-line"
          >
            {{ directoryHintDisplay }}
          </p>
          <p
            v-if="newPath.trim() && !isAbsoluteLibraryPath(newPath)"
            class="text-sm text-destructive"
          >
            {{ t("settings.notAbsolute") }}
          </p>
          <p v-if="pathAddError" class="text-sm text-destructive">
            {{ pathAddError }}
          </p>
        </div>
        <div class="flex flex-col gap-3">
          <label class="text-sm font-medium" for="new-lib-title">{{
            t("settings.optionalPathTitle")
          }}</label>
          <Input
            id="new-lib-title"
            :model-value="newPathTitle"
            class="rounded-xl"
            :placeholder="t('settings.displayName')"
            autocomplete="off"
            @update:model-value="updateNewPathTitle"
          />
        </div>
      </div>

      <DialogFooter>
        <DialogClose as-child>
          <Button type="button" variant="outline" class="rounded-2xl">
            {{ t("common.cancel") }}
          </Button>
        </DialogClose>
        <Button
          type="button"
          class="rounded-2xl"
          :disabled="addBusy || !canSaveNewPath"
          :title="addBusy || canSaveNewPath ? undefined : t('settings.savePathDisabledTitle')"
          data-add-path-submit
          @click="emit('submit')"
        >
          {{ addBusy ? t("common.saving") : t("settings.savePath") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed } from "vue"
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

const props = defineProps<{
  open: boolean
  newPath: string
  newPathTitle: string
  pickDirectoryBusy: boolean
  directoryHintDisplay: string
  pathAddError: string
  addBusy: boolean
  canSaveNewPath: boolean
  contentClass: string
  triggerLabel?: string
  dialogTitle?: string
  dialogDescription?: string
  pathLabel?: string
  pathPlaceholder?: string
  pathInputId?: string
  titleLabel?: string
  titlePlaceholder?: string
  titleInputId?: string
  examplePaths?: readonly string[]
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

const pathInputId = computed(() => props.pathInputId ?? "new-lib-path")
const titleInputId = computed(() => props.titleInputId ?? "new-lib-title")
const displayedExamplePaths = computed(() =>
  props.examplePaths?.length ? props.examplePaths : ["D:\\Media\\JAV", "/home/user/Videos"],
)

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
      <Button type="button" class="h-8 min-w-28 rounded-2xl px-3">
        <FolderPlus data-icon="inline-start" />
        {{ triggerLabel ?? t("settings.addPath") }}
      </Button>
    </DialogTrigger>

    <DialogContent :class="contentClass">
      <DialogHeader>
        <DialogTitle>{{ dialogTitle ?? t("settings.addPathDialogTitle") }}</DialogTitle>
        <DialogDescription>
          {{ dialogDescription ?? t("settings.addPathDialogDesc") }}
          <template
            v-for="(example, index) in displayedExamplePaths"
            :key="example"
          >
            <span class="font-mono text-xs">{{ example }}</span>
            <span v-if="index < displayedExamplePaths.length - 1"> / </span>
          </template>
        </DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-3">
          <label class="text-sm font-medium" :for="pathInputId">
            {{ pathLabel ?? t("settings.absolutePath") }}
          </label>
          <div class="flex flex-col gap-3 sm:flex-row sm:items-stretch">
            <Input
              :id="pathInputId"
              :model-value="newPath"
              class="rounded-xl sm:min-w-0 sm:flex-1"
              :placeholder="pathPlaceholder ?? 'D:\\Media\\JAV\\Library'"
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
          <label class="text-sm font-medium" :for="titleInputId">
            {{ titleLabel ?? t("settings.optionalPathTitle") }}
          </label>
          <Input
            :id="titleInputId"
            :model-value="newPathTitle"
            class="rounded-xl"
            :placeholder="titlePlaceholder ?? t('settings.displayName')"
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

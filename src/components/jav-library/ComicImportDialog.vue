<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { BookOpen, FileArchive, UploadCloud, X } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type { ComicImportUploadProgress, LibraryPathStorageStatusDTO } from "@/api/types"
import { pushAppToast } from "@/composables/use-app-toast"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Progress } from "@/components/ui/progress"
import { useComicLibraryService } from "@/services/comic-library-service"

const { t } = useI18n()
const comicService = useComicLibraryService()
const taskTracker = useScanTaskTracker()

const props = withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })
const emit = defineEmits<{ busy: [value: boolean]; completed: [] }>()
const open = defineModel<boolean>("open", { default: false })
const selectedFiles = ref<File[]>([])
const skippedCount = ref(0)
const importError = ref("")
const dragActive = ref(false)
const busy = ref(false)
watch(busy, (value) => emit("busy", value), { flush: "sync" })
const uploadProgress = ref<ComicImportUploadProgress | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

const archiveExtensions = new Set([".zip", ".cbz"])

const defaultImportPathId = computed(() =>
  comicService.defaultComicImportLibraryPathId.value.trim(),
)
const targetLibraryPath = computed(() =>
  comicService.comicLibraryPaths.value.find((path) => path.id === defaultImportPathId.value),
)
const targetStorageStatus = computed<LibraryPathStorageStatusDTO | undefined>(() =>
  comicService.comicLibraryPathStorageStatuses.value.find(
    (status) => status.libraryPathId === defaultImportPathId.value,
  ),
)
const hasDefaultImportPath = computed(() => Boolean(targetLibraryPath.value))
const targetStorageReady = computed(() => targetStorageStatus.value?.canImport !== false)
const targetStorageUnavailableMessage = computed(() => {
  const status = targetStorageStatus.value
  if (!status || status.canImport !== false) return ""
  return status.message?.trim() || t("import.storageUnavailable")
})
const selectedTotalBytes = computed(() =>
  selectedFiles.value.reduce((sum, file) => sum + file.size, 0),
)
const canSubmit = computed(() =>
  selectedFiles.value.length > 0 &&
  hasDefaultImportPath.value &&
  targetStorageReady.value &&
  !busy.value,
)
const progressValue = computed(() => {
  if (!busy.value) return selectedFiles.value.length > 0 ? 100 : 0
  return uploadProgress.value?.percent ?? 0
})
const triggerProgressDashOffset = computed(() => {
  const clamped = Math.min(100, Math.max(0, progressValue.value))
  return String(100 - clamped)
})

watch(open, (next) => {
  if (next) {
    importError.value = ""
    void Promise.resolve(comicService.refreshSettings()).then(() => {
      const id = defaultImportPathId.value
      if (id) {
        void comicService.checkComicLibraryPathStorageStatus([id])
      }
    }).catch((error) => {
      console.warn("[comic-import] comic settings refresh failed", error)
    })
  }
}, { immediate: true })

function fileExtension(name: string): string {
  const idx = name.lastIndexOf(".")
  return idx >= 0 ? name.slice(idx).toLowerCase() : ""
}

function relativePathForFile(file: File): string {
  return (file as File & { webkitRelativePath?: string }).webkitRelativePath?.trim() || file.name
}

function isComicArchiveFile(file: File): boolean {
  return archiveExtensions.has(fileExtension(relativePathForFile(file)))
}

function addFiles(files: File[]) {
  importError.value = ""
  const next = [...selectedFiles.value]
  const seen = new Set(next.map((file) => `${relativePathForFile(file)}:${file.size}`))
  let skipped = 0
  for (const file of files) {
    const key = `${relativePathForFile(file)}:${file.size}`
    if (!isComicArchiveFile(file) || seen.has(key)) {
      skipped += 1
      continue
    }
    next.push(file)
    seen.add(key)
  }
  selectedFiles.value = next
  skippedCount.value = skipped
}

function clearSelection() {
  selectedFiles.value = []
  skippedCount.value = 0
  uploadProgress.value = null
  importError.value = ""
  if (fileInputRef.value) fileInputRef.value.value = ""
}

function removeFile(index: number) {
  selectedFiles.value = selectedFiles.value.filter((_, i) => i !== index)
}

function openFilePicker() {
  fileInputRef.value?.click()
}

function onFileInput(event: Event) {
  const input = event.target as HTMLInputElement
  addFiles(Array.from(input.files ?? []))
  input.value = ""
}

function onDrop(event: DragEvent) {
  dragActive.value = false
  addFiles(Array.from(event.dataTransfer?.files ?? []))
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  let n = value
  let unit = 0
  while (n >= 1024 && unit < units.length - 1) {
    n /= 1024
    unit += 1
  }
  return unit === 0 ? `${Math.trunc(n)} ${units[unit]}` : `${n.toFixed(1)} ${units[unit]}`
}

function errorMessage(err: unknown): string {
  if (err instanceof HttpClientError) {
    return err.apiError?.message || err.message
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return t("import.failedMessage")
}

async function submitImport() {
  if (!canSubmit.value) return
  const id = defaultImportPathId.value
  if (id) {
    await comicService.checkComicLibraryPathStorageStatus([id])
    if (!targetStorageReady.value) {
      importError.value = targetStorageUnavailableMessage.value || t("import.storageUnavailable")
      pushAppToast(importError.value, { variant: "warning", durationMs: 6500 })
      return
    }
  }

  importError.value = ""
  busy.value = true
  uploadProgress.value = { loaded: 0, total: selectedTotalBytes.value, percent: 0 }
  try {
    const task = await comicService.importComics(selectedFiles.value, {
      onUploadProgress(progress) {
        uploadProgress.value = progress
      },
    })
    if (task?.taskId) {
      taskTracker.start(task.taskId)
    }
    pushAppToast(t("import.queuedToast"), { variant: "success", durationMs: 2600 })
    clearSelection()
    emit("completed")
    open.value = false
  } catch (err) {
    importError.value = errorMessage(err)
    pushAppToast(importError.value, { variant: "destructive", durationMs: 6500 })
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <component :is="props.embedded ? 'div' : Dialog" v-model:open="open">
    <DialogTrigger v-if="!props.embedded" as-child>
      <Button
        data-comic-import-trigger
        type="button"
        variant="secondary"
        class="rounded-full"
        :aria-label="t('import.comicTrigger')"
      >
        <svg
          v-if="busy"
          data-comic-import-trigger-progress
          data-icon="inline-start"
          class="size-4 -rotate-90 text-primary-foreground"
          viewBox="0 0 36 36"
          role="progressbar"
          :aria-valuenow="progressValue"
          aria-valuemin="0"
          aria-valuemax="100"
          :aria-label="t('import.importing')"
        >
          <circle
            class="text-primary-foreground/35"
            cx="18"
            cy="18"
            r="15.9155"
            fill="none"
            stroke="currentColor"
            stroke-width="3"
          />
          <circle
            cx="18"
            cy="18"
            r="15.9155"
            fill="none"
            stroke="currentColor"
            stroke-width="3"
            stroke-linecap="round"
            stroke-dasharray="100"
            :stroke-dashoffset="triggerProgressDashOffset"
          />
        </svg>
        <BookOpen v-else data-icon="inline-start" />
        {{ t("import.comicTrigger") }}
      </Button>
    </DialogTrigger>

    <component :is="props.embedded ? 'div' : DialogContent"
      :class="props.embedded ? 'flex min-w-0 flex-col gap-4' : 'w-[min(calc(100vw-2rem),42rem)] max-w-[calc(100vw-2rem)] min-w-0 overflow-x-hidden rounded-3xl border-border/50 sm:max-w-2xl'"
    >
      <DialogHeader v-if="!props.embedded" class="min-w-0">
        <DialogTitle>{{ t("import.comicDialogTitle") }}</DialogTitle>
        <DialogDescription>
          {{ t("import.comicDialogDescription") }}
        </DialogDescription>
      </DialogHeader>

      <div class="flex min-w-0 max-w-full flex-col gap-4 overflow-x-hidden">
        <div class="rounded-xl border border-border/70 bg-muted/25 px-3 py-2.5 text-sm">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <span class="text-muted-foreground">{{ t("import.comicTargetPath") }}</span>
            <span
              v-if="targetLibraryPath"
              class="max-w-full truncate font-mono text-xs text-foreground"
            >
              {{ targetLibraryPath.path }}
            </span>
            <span v-else class="text-xs text-destructive">
              {{ t("import.comicNoDefaultPath") }}
            </span>
          </div>
        </div>

        <p
          v-if="targetStorageUnavailableMessage"
          class="text-sm text-amber-700 dark:text-amber-300"
        >
          {{ t("import.storageUnavailable") }} {{ targetStorageUnavailableMessage }}
        </p>

        <div
          class="flex min-h-44 flex-col items-center justify-center gap-3 rounded-2xl border border-dashed p-6 text-center transition-colors"
          :class="dragActive ? 'border-primary bg-primary/10' : 'border-border bg-muted/20'"
          @dragover.prevent="dragActive = true"
          @dragleave.prevent="dragActive = false"
          @drop.prevent="onDrop"
        >
          <UploadCloud class="size-9 text-muted-foreground" aria-hidden="true" />
          <div class="flex flex-col gap-1">
            <p class="text-sm font-medium">{{ t("import.comicDropTitle") }}</p>
            <p class="text-xs text-muted-foreground">{{ t("import.comicDropHint") }}</p>
          </div>
          <Button type="button" variant="outline" size="sm" @click="openFilePicker">
            <FileArchive data-icon="inline-start" />
            {{ t("import.comicChooseFiles") }}
          </Button>
          <input
            ref="fileInputRef"
            data-comic-import-file-input
            class="sr-only"
            type="file"
            multiple
            accept=".zip,.cbz"
            @change="onFileInput"
          >
        </div>

        <div v-if="selectedFiles.length" class="flex min-w-0 max-w-full flex-col gap-2 overflow-hidden">
          <div class="flex min-w-0 items-center justify-between gap-2 text-xs text-muted-foreground">
            <span class="min-w-0 truncate">
              {{ t("import.comicSelectedSummary", { count: selectedFiles.length, size: formatBytes(selectedTotalBytes) }) }}
            </span>
            <Button type="button" variant="ghost" size="sm" @click="clearSelection">
              {{ t("import.clear") }}
            </Button>
          </div>
          <div class="max-h-40 min-w-0 max-w-full overflow-y-auto overflow-x-hidden rounded-xl border border-border/70">
            <div
              v-for="(file, index) in selectedFiles"
              :key="`${relativePathForFile(file)}-${file.size}`"
              data-comic-import-file-row
              class="flex w-full max-w-full min-w-0 items-center gap-3 overflow-hidden border-b border-border/50 px-3 py-2 text-sm last:border-b-0"
            >
              <span
                data-comic-import-file-name
                class="min-w-0 basis-0 flex-1 overflow-hidden truncate font-mono text-xs"
                :title="relativePathForFile(file)"
              >
                {{ relativePathForFile(file) }}
              </span>
              <div
                data-comic-import-file-actions
                class="flex shrink-0 items-center gap-2 text-xs tabular-nums text-muted-foreground"
              >
                <span class="whitespace-nowrap">{{ formatBytes(file.size) }}</span>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="size-7 rounded-lg"
                  :aria-label="t('import.removeFile')"
                  @click="removeFile(index)"
                >
                  <X class="size-3.5" />
                </Button>
              </div>
            </div>
          </div>
        </div>

        <p v-if="skippedCount > 0" class="text-xs text-muted-foreground">
          {{ t("import.comicSkippedUnsupported", { count: skippedCount }) }}
        </p>

        <div v-if="busy" class="flex flex-col gap-2">
          <Progress :model-value="progressValue" />
          <p class="text-xs text-muted-foreground">
            {{ t("import.copyingProgress", {
              percent: progressValue,
              loaded: formatBytes(uploadProgress?.loaded ?? 0),
              total: formatBytes(uploadProgress?.total ?? selectedTotalBytes),
            }) }}
          </p>
        </div>

        <p v-if="importError" class="text-sm text-destructive" role="alert">
          {{ importError }}
        </p>
      </div>

      <DialogFooter class="min-w-0 gap-2 sm:justify-between">
        <p class="min-w-0 text-xs text-muted-foreground">
          {{ t("import.comicFooterHint") }}
        </p>
        <Button
          data-comic-import-submit
          type="button"
          :disabled="!canSubmit"
          @click="submitImport"
        >
          {{ busy ? t("import.importing") : t("import.comicSubmit") }}
        </Button>
      </DialogFooter>
    </component>
  </component>
</template>

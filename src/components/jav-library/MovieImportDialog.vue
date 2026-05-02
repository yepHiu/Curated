<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { FilePlus2, FolderInput, UploadCloud, X } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type { MovieImportUploadProgress } from "@/api/types"
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
import { useLibraryService } from "@/services/library-service"

const { t } = useI18n()
const libraryService = useLibraryService()
const taskTracker = useScanTaskTracker()

const open = ref(false)
const selectedFiles = ref<File[]>([])
const dragActive = ref(false)
const busy = ref(false)
const uploadProgress = ref<MovieImportUploadProgress | null>(null)
const importError = ref("")
const skippedCount = ref(0)

const fileInputRef = ref<HTMLInputElement | null>(null)
const folderInputRef = ref<HTMLInputElement | null>(null)

const videoExtensions = new Set([
  ".mp4",
  ".m4v",
  ".mkv",
  ".avi",
  ".mov",
  ".wmv",
  ".webm",
  ".ts",
  ".m2ts",
  ".flv",
  ".mpeg",
  ".mpg",
  ".ogv",
  ".rmvb",
  ".iso",
])

const defaultImportPathId = computed(() => libraryService.defaultImportLibraryPathId.value.trim())
const targetLibraryPath = computed(() =>
  libraryService.libraryPaths.value.find((path) => path.id === defaultImportPathId.value),
)
const hasDefaultImportPath = computed(() => Boolean(targetLibraryPath.value))
const canSubmit = computed(() =>
  selectedFiles.value.length > 0 && hasDefaultImportPath.value && !busy.value,
)
const progressValue = computed(() => {
  if (!busy.value) return selectedFiles.value.length > 0 ? 100 : 0
  return uploadProgress.value?.percent ?? 0
})
const triggerProgressDashOffset = computed(() => {
  const clamped = Math.min(100, Math.max(0, progressValue.value))
  return String(100 - clamped)
})
const selectedTotalBytes = computed(() =>
  selectedFiles.value.reduce((sum, file) => sum + file.size, 0),
)

watch(open, (next) => {
  if (next) {
    importError.value = ""
    void libraryService.refreshSettings()
  }
})

function fileExtension(name: string): string {
  const idx = name.lastIndexOf(".")
  return idx >= 0 ? name.slice(idx).toLowerCase() : ""
}

function relativePathForFile(file: File): string {
  return (file as File & { webkitRelativePath?: string }).webkitRelativePath?.trim() || file.name
}

function isVideoFile(file: File): boolean {
  return videoExtensions.has(fileExtension(relativePathForFile(file)))
}

function addFiles(files: File[]) {
  importError.value = ""
  const next = [...selectedFiles.value]
  const seen = new Set(next.map((file) => `${relativePathForFile(file)}:${file.size}`))
  let skipped = 0
  for (const file of files) {
    const key = `${relativePathForFile(file)}:${file.size}`
    if (!isVideoFile(file) || seen.has(key)) {
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
  if (folderInputRef.value) folderInputRef.value.value = ""
}

function removeFile(index: number) {
  selectedFiles.value = selectedFiles.value.filter((_, i) => i !== index)
}

function openFilePicker() {
  fileInputRef.value?.click()
}

function openFolderPicker() {
  folderInputRef.value?.click()
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
  importError.value = ""
  busy.value = true
  uploadProgress.value = { loaded: 0, total: selectedTotalBytes.value, percent: 0 }
  try {
    const task = await libraryService.importMovies(selectedFiles.value, {
      onUploadProgress(progress) {
        uploadProgress.value = progress
      },
    })
    if (task?.taskId) {
      taskTracker.start(task.taskId)
    }
    pushAppToast(t("import.queuedToast"), { variant: "success", durationMs: 2600 })
    clearSelection()
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
  <Dialog v-model:open="open">
    <DialogTrigger as-child>
      <Button
        data-import-trigger
        type="button"
        variant="secondary"
        class="rounded-2xl"
        :aria-label="t('import.trigger')"
      >
        <svg
          v-if="busy"
          data-import-trigger-progress
          data-icon="inline-start"
          class="size-4 -rotate-90 text-primary"
          viewBox="0 0 36 36"
          role="progressbar"
          :aria-valuenow="progressValue"
          aria-valuemin="0"
          aria-valuemax="100"
          :aria-label="t('import.importing')"
        >
          <circle
            class="text-muted-foreground/30"
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
        <FilePlus2 v-else data-icon="inline-start" />
        {{ t("import.trigger") }}
      </Button>
    </DialogTrigger>

    <DialogContent class="rounded-3xl border-border/50 sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ t("import.dialogTitle") }}</DialogTitle>
        <DialogDescription>
          {{ t("import.dialogDescription") }}
        </DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-4">
        <div class="rounded-xl border border-border/70 bg-muted/25 px-3 py-2.5 text-sm">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <span class="text-muted-foreground">{{ t("import.targetPath") }}</span>
            <span
              v-if="targetLibraryPath"
              class="max-w-full truncate font-mono text-xs text-foreground"
            >
              {{ targetLibraryPath.path }}
            </span>
            <span v-else class="text-xs text-destructive">
              {{ t("import.noDefaultPath") }}
            </span>
          </div>
        </div>

        <div
          class="flex min-h-44 flex-col items-center justify-center gap-3 rounded-2xl border border-dashed p-6 text-center transition-colors"
          :class="dragActive ? 'border-primary bg-primary/10' : 'border-border bg-muted/20'"
          @dragover.prevent="dragActive = true"
          @dragleave.prevent="dragActive = false"
          @drop.prevent="onDrop"
        >
          <UploadCloud class="size-9 text-muted-foreground" aria-hidden="true" />
          <div class="flex flex-col gap-1">
            <p class="text-sm font-medium">{{ t("import.dropTitle") }}</p>
            <p class="text-xs text-muted-foreground">{{ t("import.dropHint") }}</p>
          </div>
          <div class="flex flex-wrap justify-center gap-2">
            <Button type="button" variant="outline" size="sm" @click="openFilePicker">
              <FilePlus2 data-icon="inline-start" />
              {{ t("import.chooseFiles") }}
            </Button>
            <Button type="button" variant="outline" size="sm" @click="openFolderPicker">
              <FolderInput data-icon="inline-start" />
              {{ t("import.chooseFolder") }}
            </Button>
          </div>
          <input
            ref="fileInputRef"
            data-import-file-input
            class="sr-only"
            type="file"
            multiple
            accept="video/*,.mkv,.iso,.m2ts,.rmvb"
            @change="onFileInput"
          >
          <input
            ref="folderInputRef"
            data-import-folder-input
            class="sr-only"
            type="file"
            multiple
            webkitdirectory
            @change="onFileInput"
          >
        </div>

        <div v-if="selectedFiles.length" class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
            <span>{{ t("import.selectedSummary", { count: selectedFiles.length, size: formatBytes(selectedTotalBytes) }) }}</span>
            <Button type="button" variant="ghost" size="sm" @click="clearSelection">
              {{ t("import.clear") }}
            </Button>
          </div>
          <div class="max-h-40 overflow-y-auto rounded-xl border border-border/70">
            <div
              v-for="(file, index) in selectedFiles"
              :key="`${relativePathForFile(file)}-${file.size}`"
              class="flex items-center justify-between gap-3 border-b border-border/50 px-3 py-2 text-sm last:border-b-0"
            >
              <span class="min-w-0 truncate font-mono text-xs">{{ relativePathForFile(file) }}</span>
              <div class="flex shrink-0 items-center gap-2 text-xs text-muted-foreground">
                <span>{{ formatBytes(file.size) }}</span>
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
          {{ t("import.skippedUnsupported", { count: skippedCount }) }}
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

      <DialogFooter class="gap-2 sm:justify-between">
        <p class="text-xs text-muted-foreground">
          {{ t("import.footerHint") }}
        </p>
        <Button
          data-import-submit
          type="button"
          :disabled="!canSubmit"
          @click="submitImport"
        >
          {{ busy ? t("import.importing") : t("import.submit") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

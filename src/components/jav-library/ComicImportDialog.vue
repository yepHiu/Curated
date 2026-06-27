<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { BookOpen, FileArchive, UploadCloud, X } from "lucide-vue-next"
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
import { useComicLibraryService } from "@/services/comic-library-service"

const { t } = useI18n()
const comicService = useComicLibraryService()

const open = ref(false)
const selectedFiles = ref<File[]>([])
const skippedCount = ref(0)
const importError = ref("")
const dragActive = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

const archiveExtensions = new Set([".zip", ".cbz"])

const defaultImportPathId = computed(() =>
  comicService.defaultComicImportLibraryPathId.value.trim(),
)
const targetLibraryPath = computed(() =>
  comicService.comicLibraryPaths.value.find((path) => path.id === defaultImportPathId.value),
)
const hasDefaultImportPath = computed(() => Boolean(targetLibraryPath.value))
const selectedTotalBytes = computed(() =>
  selectedFiles.value.reduce((sum, file) => sum + file.size, 0),
)
const canSubmit = computed(() => selectedFiles.value.length > 0 && hasDefaultImportPath.value)

watch(open, (next) => {
  if (next) {
    importError.value = ""
    void Promise.resolve(comicService.refreshSettings()).catch((error) => {
      console.warn("[comic-import] comic settings refresh failed", error)
    })
  }
})

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

function submitImport() {
  if (!canSubmit.value) return
  importError.value = t("import.comicImportPending")
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogTrigger as-child>
      <Button
        data-comic-import-trigger
        type="button"
        variant="secondary"
        class="rounded-full"
        :aria-label="t('import.comicTrigger')"
      >
        <BookOpen data-icon="inline-start" />
        {{ t("import.comicTrigger") }}
      </Button>
    </DialogTrigger>

    <DialogContent class="rounded-3xl border-border/50 sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ t("import.comicDialogTitle") }}</DialogTitle>
        <DialogDescription>
          {{ t("import.comicDialogDescription") }}
        </DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-4">
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

        <div v-if="selectedFiles.length" class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
            <span>
              {{ t("import.comicSelectedSummary", { count: selectedFiles.length, size: formatBytes(selectedTotalBytes) }) }}
            </span>
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
          {{ t("import.comicSkippedUnsupported", { count: skippedCount }) }}
        </p>

        <p v-if="importError" class="text-sm text-destructive" role="alert">
          {{ importError }}
        </p>
      </div>

      <DialogFooter class="gap-2 sm:justify-between">
        <p class="text-xs text-muted-foreground">
          {{ t("import.comicFooterHint") }}
        </p>
        <Button
          data-comic-import-submit
          type="button"
          :disabled="!canSubmit"
          @click="submitImport"
        >
          {{ t("import.comicSubmit") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

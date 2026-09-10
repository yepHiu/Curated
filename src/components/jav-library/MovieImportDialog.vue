<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import ImportTargetPath from "./ImportTargetPath.vue"
import { FilePlus2, FolderInput, UploadCloud, X } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type { LibraryPathStorageStatusDTO, MovieImportUploadProgress, ImportMovieCodeCheckItemDTO } from "@/api/types"
import { pushAppToast } from "@/composables/use-app-toast"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import { Badge } from "@/components/ui/badge"
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
import { matchesMovieImportUploadFiles } from "@/lib/movie-import-upload-ledger"
import type { ResumableMovieImportSession } from "@/services/contracts/library-service"
import { useLibraryService } from "@/services/library-service"

const { t } = useI18n()
const libraryService = useLibraryService()
const taskTracker = useScanTaskTracker()

const props = withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })
const emit = defineEmits<{ busy: [value: boolean]; completed: [] }>()
const open = defineModel<boolean>("open", { default: false })
const selectedFiles = ref<File[]>([])
const dragActive = ref(false)
const busy = ref(false)
watch(busy, (value) => emit("busy", value), { flush: "sync" })
const resumableSessions = ref<ResumableMovieImportSession[]>([])
const abandonConfirmId = ref("")
const abandonBusyId = ref("")
const uploadProgress = ref<MovieImportUploadProgress | null>(null)
const importError = ref("")
const skippedCount = ref(0)
const checkItems = ref<ImportMovieCodeCheckItemDTO[]>([])
let codeCheckSeq = 0

const fileInputRef = ref<HTMLInputElement | null>(null)
const folderInputRef = ref<HTMLInputElement | null>(null)

// Mirrors backend/internal/contracts.SupportedVideoExtensions (scanner / watch /
// import share one whitelist). Keep both lists in sync.
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
const targetStorageStatus = computed<LibraryPathStorageStatusDTO | undefined>(() =>
  libraryService.libraryPathStorageStatuses.value.find(
    (status) => status.libraryPathId === defaultImportPathId.value,
  ),
)
const hasDefaultImportPath = computed(() => Boolean(targetLibraryPath.value))
const targetStorageReady = computed(() => targetStorageStatus.value?.canImport !== false)
const canSubmit = computed(() =>
  selectedFiles.value.length > 0 &&
  hasDefaultImportPath.value &&
  targetStorageReady.value &&
  !busy.value,
)
const targetStorageUnavailableMessage = computed(() => {
  const status = targetStorageStatus.value
  if (!status || status.canImport !== false) return ""
  return status.message?.trim() || t("import.storageUnavailable")
})
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

const matchedResumeSession = computed<ResumableMovieImportSession | null>(() => {
  if (!selectedFiles.value.length) return null
  const fingerprints = selectedFiles.value.map((file) => ({
    relativePath: relativePathForFile(file),
    size: file.size,
    lastModified: file.lastModified,
  }))
  for (const session of resumableSessions.value) {
    if (matchesMovieImportUploadFiles(session.files, fingerprints)) return session
  }
  return null
})

watch(open, (next) => {
  if (next) {
    importError.value = ""
    abandonConfirmId.value = ""
    void libraryService.refreshSettings().then(() => {
      const id = defaultImportPathId.value
      if (id) {
        void libraryService.checkLibraryPathStorageStatus([id])
      }
    })
    void refreshResumableSessions()
  }
}, { immediate: true })

async function refreshResumableSessions() {
  try {
    resumableSessions.value = await libraryService.listResumableMovieImports()
  } catch {
    // 列表加载失败不阻塞导入主流程
    resumableSessions.value = []
  }
}

function sessionPercent(session: ResumableMovieImportSession): number {
  if (session.totalBytes <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((session.bytesReceived / session.totalBytes) * 100)))
}

function formatSessionExpiry(session: ResumableMovieImportSession): string {
  const expiresAtMs = session.expiresAt ? Date.parse(session.expiresAt) : Number.NaN
  if (!Number.isFinite(expiresAtMs)) return ""
  const remainingMs = expiresAtMs - Date.now()
  if (remainingMs <= 0) return t("import.resumableExpiresMinutes", { minutes: 0 })
  const hours = Math.floor(remainingMs / 3_600_000)
  if (hours >= 1) return t("import.resumableExpiresHours", { hours })
  const minutes = Math.max(1, Math.ceil(remainingMs / 60_000))
  return t("import.resumableExpiresMinutes", { minutes })
}

async function abandonSession(session: ResumableMovieImportSession) {
  if (busy.value || abandonBusyId.value) return
  if (abandonConfirmId.value !== session.uploadId) {
    abandonConfirmId.value = session.uploadId
    return
  }
  abandonConfirmId.value = ""
  abandonBusyId.value = session.uploadId
  try {
    await libraryService.abandonMovieImportUpload(session.uploadId)
    resumableSessions.value = resumableSessions.value.filter(
      (item) => item.uploadId !== session.uploadId,
    )
  } catch (err) {
    pushAppToast(errorMessage(err), { variant: "destructive", durationMs: 6500 })
  } finally {
    abandonBusyId.value = ""
  }
}

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

function checkItemForFile(file: File): ImportMovieCodeCheckItemDTO | undefined {
  const name = relativePathForFile(file)
  return checkItems.value.find((item) => item.name === name)
}

function fileAlreadyImported(file: File): boolean {
  return (checkItemForFile(file)?.matches.length ?? 0) > 0
}

async function refreshCodeCheck(files: File[]) {
  const seq = ++codeCheckSeq
  if (files.length === 0) {
    checkItems.value = []
    return
  }
  try {
    const result = await libraryService.checkImportMovieCodes(
      files.map((file) => relativePathForFile(file)),
    )
    if (seq !== codeCheckSeq) return
    checkItems.value = result.items
  } catch {
    if (seq !== codeCheckSeq) return
    checkItems.value = []
  }
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
  void refreshCodeCheck(next)
}

function clearSelection() {
  selectedFiles.value = []
  skippedCount.value = 0
  uploadProgress.value = null
  importError.value = ""
  checkItems.value = []
  codeCheckSeq += 1
  if (fileInputRef.value) fileInputRef.value.value = ""
  if (folderInputRef.value) folderInputRef.value.value = ""
}

function removeFile(index: number) {
  const file = selectedFiles.value[index]
  selectedFiles.value = selectedFiles.value.filter((_, i) => i !== index)
  if (!file) return
  const name = relativePathForFile(file)
  checkItems.value = checkItems.value.filter((item) => item.name !== name)
  if (selectedFiles.value.length === 0) {
    checkItems.value = []
  }
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
  const id = defaultImportPathId.value
  if (id) {
    await libraryService.checkLibraryPathStorageStatus([id])
    if (!targetStorageReady.value) {
      importError.value = targetStorageUnavailableMessage.value || t("import.storageUnavailable")
      pushAppToast(importError.value, {
        variant: "warning",
        durationMs: 6500,
        notification: {
          messageId: "MSG-0010",
          type: "storage",
          title: t("notificationCenter.titles.storageOffline"),
          source: { route: "/settings?section=library", libraryPathId: id },
        },
      })
      return
    }
  }
  importError.value = ""
  busy.value = true
  const resumeSession = matchedResumeSession.value
  if (resumeSession) {
    uploadProgress.value = {
      loaded: resumeSession.bytesReceived,
      total: resumeSession.totalBytes,
      percent: sessionPercent(resumeSession),
    }
  } else {
    uploadProgress.value = { loaded: 0, total: selectedTotalBytes.value, percent: 0 }
  }
  try {
    const task = await libraryService.importMovies(selectedFiles.value, {
      onUploadProgress(progress) {
        uploadProgress.value = progress
      },
      resumeUploadId: resumeSession?.uploadId,
    })
    if (task?.taskId) {
      taskTracker.start(task.taskId)
    }
    if (resumeSession) {
      resumableSessions.value = resumableSessions.value.filter(
        (item) => item.uploadId !== resumeSession.uploadId,
      )
    }
    pushAppToast(t("import.queuedToast"), { variant: "success", durationMs: 2600 })
    clearSelection()
    emit("completed")
    open.value = false
  } catch (err) {
    importError.value = errorMessage(err)
    pushAppToast(importError.value, { variant: "destructive", durationMs: 6500 })
    // 失败后核对一次会话状态：终态会话会被服务层从账本剔除
    void refreshResumableSessions()
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <component :is="props.embedded ? 'div' : Dialog" v-model:open="open">
    <DialogTrigger v-if="!props.embedded" as-child>
      <Button
        data-import-trigger
        type="button"
        variant="ghost"
        class="min-h-11 rounded-full text-muted-foreground hover:bg-muted/70 hover:text-foreground lg:min-h-9"
        :aria-label="t('import.trigger')"
      >
        <svg
          v-if="busy"
          data-import-trigger-progress
          data-icon="inline-start"
          class="size-4 -rotate-90 text-foreground"
          viewBox="0 0 36 36"
          role="progressbar"
          :aria-valuenow="progressValue"
          aria-valuemin="0"
          aria-valuemax="100"
          :aria-label="t('import.importing')"
        >
          <circle
            class="text-foreground/35"
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

    <component :is="props.embedded ? 'div' : DialogContent" :class="props.embedded ? 'flex min-w-0 flex-col gap-4' : 'rounded-3xl border-border/50 sm:max-w-2xl'">
      <DialogHeader v-if="!props.embedded">
        <DialogTitle>{{ t("import.dialogTitle") }}</DialogTitle>
        <DialogDescription>
          {{ t("import.dialogDescription") }}
        </DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-4">
        <ImportTargetPath
          :path="targetLibraryPath?.path"
          :empty-message="t('import.noDefaultPath')"
        />

        <p
          v-if="targetStorageUnavailableMessage"
          class="text-sm text-amber-700 dark:text-amber-300"
        >
          {{ t("import.storageUnavailable") }} {{ targetStorageUnavailableMessage }}
        </p>

        <div
          v-if="resumableSessions.length"
          data-import-resumable
          class="flex flex-col gap-2 rounded-xl border border-border/70 bg-muted/25 px-3 py-2.5"
        >
          <span class="text-xs font-medium text-muted-foreground">
            {{ t("import.resumableTitle") }}
          </span>
          <div
            v-for="session in resumableSessions"
            :key="session.uploadId"
            data-import-resumable-item
            class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border/50 bg-background/60 px-2.5 py-2"
          >
            <div class="flex min-w-0 flex-col gap-0.5">
              <span class="truncate text-sm">
                {{ t("import.resumableSummary", {
                  count: session.files.length,
                  size: formatBytes(session.totalBytes),
                }) }}
              </span>
              <span class="text-xs text-muted-foreground">
                {{ t("import.resumableProgress", { percent: sessionPercent(session) }) }}
                <template v-if="formatSessionExpiry(session)"> · {{ formatSessionExpiry(session) }}</template>
              </span>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              class="h-8 shrink-0 rounded-lg px-2 text-xs text-muted-foreground hover:text-destructive"
              :disabled="busy || abandonBusyId === session.uploadId"
              :data-import-resumable-abandon="session.uploadId"
              @click="abandonSession(session)"
            >
              {{ abandonConfirmId === session.uploadId
                ? t("import.abandonConfirm")
                : t("import.abandon") }}
            </Button>
          </div>
          <p class="text-xs text-muted-foreground">
            {{ t("import.resumableHint") }}
          </p>
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
          <div class="max-h-48 overflow-y-auto rounded-xl border border-border/70">
            <div
              v-for="(file, index) in selectedFiles"
              :key="`${relativePathForFile(file)}-${file.size}`"
              class="flex items-center justify-between gap-3 border-b border-border/50 px-3 py-2 text-sm last:border-b-0"
              :data-import-code-row="relativePathForFile(file)"
            >
              <div class="flex min-w-0 flex-1 items-center gap-2">
                <span class="min-w-0 truncate font-mono text-xs">{{ relativePathForFile(file) }}</span>
                <Badge
                  v-if="fileAlreadyImported(file)"
                  data-import-already-imported
                  variant="warning"
                  class="shrink-0"
                >
                  {{ t("import.alreadyImported") }}
                </Badge>
              </div>
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

        <p
          v-if="matchedResumeSession"
          data-import-resume-match
          class="rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-xs text-foreground"
        >
          {{ t("import.resumableMatched", { percent: sessionPercent(matchedResumeSession) }) }}
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
    </component>
  </component>
</template>

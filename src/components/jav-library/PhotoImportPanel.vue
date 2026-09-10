<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import ImportTargetPath from "./ImportTargetPath.vue"
import { FileArchive, UploadCloud, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { usePhotoLibraryService } from "@/services/photo-library-service"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import { pushAppToast } from "@/composables/use-app-toast"
import type { PhotoImportUploadProgress } from "@/api/types"

const props = defineProps<{ active: boolean }>()
const emit = defineEmits<{ busy: [value: boolean]; completed: [] }>()
const { t } = useI18n()
const service = usePhotoLibraryService()
const tracker = useScanTaskTracker()
const files = ref<File[]>([])
const input = ref<HTMLInputElement | null>(null)
const busy = ref(false)
const error = ref("")
const skipped = ref(0)
const dragActive = ref(false)
const progress = ref<PhotoImportUploadProgress | null>(null)
const target = computed(() => service.photoLibraryPaths.value.find(path => path.id === service.defaultPhotoImportLibraryPathId.value))
const canSubmit = computed(() => service.photoLibraryEnabled.value && !!target.value && files.value.length > 0 && !busy.value)
watch(busy, value => emit("busy", value), { flush: "sync" })
watch(() => props.active, async value => {
  if (value) {
    try { await service.refreshSettings() } catch (cause) { error.value = errorText(cause) }
  }
}, { immediate: true })

/** Only ZIP/CBZ files enter this independent photo upload. */
function addFiles(incoming: File[]) {
  if (busy.value) return
  error.value = ""
  for (const file of incoming) {
    if (!/\.(zip|cbz)$/i.test(file.name)) { skipped.value++; continue }
    if (!files.value.some(item => item.name === file.name && item.size === file.size && item.lastModified === file.lastModified)) files.value.push(file)
  }
}
function selectFiles(event: Event) {
  const element = event.target as HTMLInputElement
  addFiles(Array.from(element.files ?? [])); element.value = ""
}
function dropFiles(event: DragEvent) { dragActive.value = false; addFiles(Array.from(event.dataTransfer?.files ?? [])) }
function errorText(cause: unknown) { return cause instanceof Error ? cause.message : t("import.failedMessage") }

/** Upload first, then follow the independent photo scan; failed files stay reviewable. */
async function submit() {
  if (!canSubmit.value) return
  busy.value = true
  error.value = ""
  try {
    const task = await service.importPhotos(files.value, { onUploadProgress: value => { progress.value = value } })
    const scanTaskId = task?.metadata?.scanTaskId
    if (typeof scanTaskId === "string" && scanTaskId) tracker.start(scanTaskId)
    if (task && task.status !== "completed") {
      error.value = t("import.photoResultError", { completed: task.metadata?.completedFiles ?? 0, failed: task.metadata?.failedFiles ?? 0, message: task.errorMessage || task.message || "" })
      return
    }
    pushAppToast(t("import.queuedToast"), { variant: "success", durationMs: 2600 })
    files.value = []; skipped.value = 0
    emit("completed")
  } catch (cause) { error.value = errorText(cause) }
  finally { busy.value = false }
}
</script>

<template>
  <div data-photo-import-panel class="flex min-w-0 flex-col gap-4">
    <ImportTargetPath
      :path="target?.path"
      :empty-message="t('import.photoNoDefaultPath')"
    />

    <div class="flex min-h-44 flex-col items-center justify-center gap-3 rounded-2xl border border-dashed p-6 text-center"
      :class="dragActive ? 'border-primary bg-primary/10' : 'border-border bg-muted/20'"
      @dragover.prevent="dragActive = !busy" @dragleave.prevent="dragActive = false" @drop.prevent="dropFiles">
      <UploadCloud class="size-9 text-muted-foreground" aria-hidden="true" />
      <p class="text-sm font-medium">{{ t("import.photoDropTitle") }}</p>
      <p class="text-xs text-muted-foreground">{{ t("import.photoDropHint") }}</p>
      <Button type="button" variant="outline" :disabled="busy" @click="input?.click()"><FileArchive data-icon="inline-start" />{{ t("import.photoChooseFiles") }}</Button>
      <input ref="input" data-photo-import-input type="file" accept=".zip,.cbz" multiple class="hidden" :disabled="busy" @change="selectFiles" />
    </div>
    <div v-if="files.length" class="flex max-h-48 flex-col gap-2 overflow-y-auto">
      <div v-for="(file, index) in files" :key="file.name + file.lastModified + file.size" class="flex min-w-0 items-center gap-2 rounded-lg border border-border px-3 py-2">
        <span class="min-w-0 flex-1 truncate text-sm" :title="file.name">{{ file.name }}</span>
        <Button type="button" variant="ghost" size="icon" :disabled="busy" :aria-label="t('import.removeFile')" @click="files.splice(index, 1)"><X class="size-4" /></Button>
      </div>
    </div>
    <p v-if="skipped" class="text-xs text-muted-foreground">{{ t("import.photoSkippedUnsupported", { count: skipped }) }}</p>
    <Progress v-if="busy" :model-value="progress?.percent ?? 0" :aria-label="t('import.importing')" />
    <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <p class="text-xs text-muted-foreground">{{ t("import.photoFooterHint") }}</p>
      <Button data-photo-import-submit type="button" :disabled="!canSubmit" @click="submit">{{ busy ? t("import.importing") : t("import.photoSubmit") }}</Button>
    </div>
  </div>
</template>

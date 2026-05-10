<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"

const { t } = useI18n()
const { progressTask, progressPollError, dismiss } = useScanTaskTracker()

const emptyValue = "-"
const visible = computed(() => progressTask.value != null || progressPollError.value != null)
const isImportTask = computed(() => progressTask.value?.type === "import.movies")

function metaNum(m: Record<string, unknown> | undefined, key: string): string {
  const n = metaNumberValue(m, key)
  return Number.isFinite(n) ? String(Math.trunc(n)) : emptyValue
}

function metaText(m: Record<string, unknown> | undefined, key: string): string {
  const v = m?.[key]
  return typeof v === "string" && v.trim() ? v : emptyValue
}

function metaNumberValue(m: Record<string, unknown> | undefined, key: string): number {
  if (!m) return Number.NaN
  const v = m[key]
  if (typeof v === "number" && Number.isFinite(v)) {
    return v
  }
  if (typeof v === "string" && v !== "") {
    const n = Number(v)
    return Number.isFinite(n) ? n : Number.NaN
  }
  return Number.NaN
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return emptyValue
  const units = ["B", "KB", "MB", "GB", "TB"]
  let n = value
  let unit = 0
  while (n >= 1024 && unit < units.length - 1) {
    n /= 1024
    unit += 1
  }
  return unit === 0 ? `${Math.trunc(n)} ${units[unit]}` : `${n.toFixed(1)} ${units[unit]}`
}

const dockTitle = computed(() => {
  if (progressPollError.value && !progressTask.value) return t("scan.statusLabel")
  const task = progressTask.value
  if (!task) return ""
  if (task.type === "import.movies") {
    if (task.status === "completed") return t("import.completed")
    if (task.status === "failed" || task.status === "partial_failed") return t("import.finished")
    return t("import.scanning")
  }
  if (task.status === "completed") return t("scan.completed")
  if (task.status === "failed" || task.status === "partial_failed") return t("scan.finished")
  return t("scan.scanning")
})

const progressModel = computed(() => {
  const task = progressTask.value
  if (!task) return 0
  return Math.min(100, Math.max(0, task.progress))
})

const detailLine = computed(() => progressTask.value?.message ?? "")

const importCurrentFile = computed(() => metaText(progressTask.value?.metadata, "currentFileName"))

const importCopiedBytes = computed(() => {
  const copied = metaNumberValue(progressTask.value?.metadata, "copiedBytes")
  const total = metaNumberValue(progressTask.value?.metadata, "totalBytes")
  if (Number.isFinite(total) && total > 0) {
    return `${formatBytes(copied)} / ${formatBytes(total)}`
  }
  return formatBytes(copied)
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="animate-in fade-in-0 slide-in-from-bottom-4 fixed right-4 bottom-4 z-50 w-[min(100%-2rem,22rem)] duration-300"
    >
      <Card class="border-border/80 bg-card/95 shadow-xl backdrop-blur-sm">
        <CardHeader class="flex flex-row items-start justify-between gap-2 space-y-0 pb-2">
          <CardTitle class="text-base font-semibold">{{ dockTitle }}</CardTitle>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            class="shrink-0 rounded-xl"
            :aria-label="t('scan.close')"
            @click="dismiss"
          >
            <X class="size-4" />
          </Button>
        </CardHeader>
        <CardContent class="flex flex-col gap-3">
          <template v-if="progressTask">
            <Progress :model-value="progressModel" />
            <p v-if="detailLine" class="text-xs leading-snug text-muted-foreground">
              {{ detailLine }}
            </p>

            <div
              v-if="isImportTask"
              class="grid grid-cols-2 gap-x-3 gap-y-1 text-xs text-muted-foreground"
            >
              <span v-once>{{ t("import.currentFile") }}</span>
              <span class="truncate text-right font-mono text-foreground">
                {{ importCurrentFile }}
              </span>
              <span v-once>{{ t("import.files") }}</span>
              <span class="text-right font-mono text-foreground">
                {{ metaNum(progressTask.metadata, "completedFiles") }} /
                {{ metaNum(progressTask.metadata, "totalFiles") }}
              </span>
              <span v-once>{{ t("import.copiedBytes") }}</span>
              <span class="text-right font-mono text-foreground">{{ importCopiedBytes }}</span>
              <span v-once>{{ t("import.failedFiles") }}</span>
              <span class="text-right font-mono text-foreground">{{
                metaNum(progressTask.metadata, "failedFiles")
              }}</span>
            </div>

            <div v-else class="grid grid-cols-2 gap-x-3 gap-y-1 text-xs text-muted-foreground">
              <span v-once>{{ t("scan.processed") }}</span>
              <span class="text-right font-mono text-foreground">
                {{ metaNum(progressTask.metadata, "scanProcessed") }} /
                {{ metaNum(progressTask.metadata, "scanTotal") }}
              </span>
              <span v-once>{{ t("scan.newItems") }}</span>
              <span class="text-right font-mono text-foreground">{{
                metaNum(progressTask.metadata, "scanImported")
              }}</span>
              <span v-once>{{ t("scan.updated") }}</span>
              <span class="text-right font-mono text-foreground">{{
                metaNum(progressTask.metadata, "scanUpdated")
              }}</span>
              <span v-once>{{ t("scan.skipped") }}</span>
              <span class="text-right font-mono text-foreground">{{
                metaNum(progressTask.metadata, "scanSkipped")
              }}</span>
            </div>

            <p
              v-if="(progressTask.status === 'failed' || progressTask.status === 'partial_failed') && progressTask.errorMessage"
              class="text-xs text-destructive"
            >
              {{ progressTask.errorMessage }}
            </p>
          </template>
          <p v-else-if="progressPollError" class="text-sm text-destructive">
            {{ progressPollError }}
          </p>
        </CardContent>
      </Card>
    </div>
  </Teleport>
</template>

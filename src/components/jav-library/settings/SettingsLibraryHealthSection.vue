<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue"
import { useI18n } from "vue-i18n"
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Download,
  FileWarning,
  LoaderCircle,
  RefreshCw,
  ShieldCheck,
  Wrench,
} from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type {
  LibraryHealthFindingDTO,
  LibraryHealthRepairDTO,
  LibraryHealthReportDTO,
} from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Progress } from "@/components/ui/progress"
import { Separator } from "@/components/ui/separator"
import { pushAppToast } from "@/composables/use-app-toast"
import { statusPanelClass } from "@/lib/ui/status-tone"
import { useLibraryService } from "@/services/library-service"

type RepairCategory = "metadata_missing" | "metadata_failed"

const props = defineProps<{
  supported: boolean
}>()

const { t, locale } = useI18n()
const libraryService = useLibraryService()
const report = ref<LibraryHealthReportDTO | null>(null)
const reportBusy = ref(false)
const actionError = ref("")
const repair = ref<LibraryHealthRepairDTO | null>(null)
const repairDialogOpen = ref(false)
const pendingRepairCategory = ref<RepairCategory>("metadata_missing")
const repairSubmitting = ref(false)
let componentAlive = true

const visibleFindings = computed(() => report.value?.findings.slice(0, 100) ?? [])
const hiddenFindingCount = computed(() => {
  if (!report.value) return 0
  return Math.max(0, report.value.summary.totalFindings - visibleFindings.value.length)
})
const sortedCategoryCounts = computed(() => {
  const entries = Object.entries(report.value?.summary.categoryCounts ?? {})
  return entries.sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).slice(0, 10)
})
const repairCount = computed(() => report.value?.summary.categoryCounts[pendingRepairCategory.value] ?? 0)
const repairProgress = computed(() => {
  const current = repair.value
  if (!current || current.totalItems <= 0) return 0
  return Math.round((current.completedItems / current.totalItems) * 100)
})
const repairActive = computed(() => {
  const status = repair.value?.status
  return status === "pending" || status === "running"
})

onBeforeUnmount(() => {
  componentAlive = false
})

function formatError(error: unknown): string {
  if (error instanceof HttpClientError && error.apiError?.message) return error.apiError.message
  if (error instanceof Error && error.message.trim()) return error.message
  return t("settings.libraryHealthUnknownError")
}

function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date)
}

function reportStatusVariant(status: LibraryHealthReportDTO["status"]) {
  if (status === "healthy") return "success" as const
  if (status === "critical") return "danger" as const
  return "warning" as const
}

function severityVariant(severity: LibraryHealthFindingDTO["severity"]) {
  if (severity === "critical") return "danger" as const
  if (severity === "warning") return "warning" as const
  return "info" as const
}

function repairStatusVariant(status: LibraryHealthRepairDTO["status"]) {
  if (status === "completed") return "success" as const
  if (status === "partial_failed") return "warning" as const
  if (status === "failed" || status === "cancelled") return "danger" as const
  return "info" as const
}

function repairItemStatusLabel(status: LibraryHealthRepairDTO["items"][number]["status"]): string {
  return t(`settings.libraryHealthRepairItemStatus_${status}`)
}

function categoryLabel(category: string): string {
  const keyByCategory: Record<string, string> = {
    database_integrity: "settings.libraryHealthCategoryDatabaseIntegrity",
    foreign_key_violation: "settings.libraryHealthCategoryForeignKey",
    storage_unavailable: "settings.libraryHealthCategoryStorage",
    source_missing: "settings.libraryHealthCategorySourceMissing",
    source_unreadable: "settings.libraryHealthCategorySourceUnreadable",
    source_empty: "settings.libraryHealthCategorySourceEmpty",
    asset_missing: "settings.libraryHealthCategoryAssetMissing",
    asset_download_failed: "settings.libraryHealthCategoryAssetFailed",
    duplicate_code: "settings.libraryHealthCategoryDuplicateCode",
    duplicate_source: "settings.libraryHealthCategoryDuplicateSource",
    orphan_user_state: "settings.libraryHealthCategoryOrphanState",
    metadata_missing: "settings.libraryHealthCategoryMetadataMissing",
    metadata_failed: "settings.libraryHealthCategoryMetadataFailed",
    movie_poster_missing: "settings.libraryHealthCategoryPosterMissing",
    actor_avatar_missing: "settings.libraryHealthCategoryActorAvatar",
    import_staging_residue: "settings.libraryHealthCategoryImportResidue",
  }
  const key = keyByCategory[category]
  return key ? t(key) : category.replaceAll("_", " ")
}

async function runHealthScan() {
  if (!props.supported || reportBusy.value || repairActive.value) return
  actionError.value = ""
  reportBusy.value = true
  try {
    report.value = await libraryService.scanLibraryHealth()
  } catch (error) {
    actionError.value = formatError(error)
  } finally {
    reportBusy.value = false
  }
}

function openRepairDialog(category: RepairCategory) {
  if (!props.supported || repairActive.value) return
  pendingRepairCategory.value = category
  repairDialogOpen.value = true
}

async function confirmRepair() {
  if (!props.supported || repairSubmitting.value || repairCount.value <= 0) return
  actionError.value = ""
  repairSubmitting.value = true
  try {
    repair.value = await libraryService.startLibraryHealthRepair({
      action: "rescrape_metadata",
      categories: [pendingRepairCategory.value],
      limit: Math.min(25, repairCount.value),
      confirm: true,
    })
    repairDialogOpen.value = false
    await pollRepair(repair.value.repairId)
  } catch (error) {
    actionError.value = formatError(error)
  } finally {
    repairSubmitting.value = false
  }
}

async function pollRepair(repairId: string) {
  while (componentAlive) {
    const current = await libraryService.getLibraryHealthRepair(repairId)
    repair.value = current
    if (!["pending", "running"].includes(current.status)) {
      pushAppToast(
        current.status === "completed"
          ? t("settings.libraryHealthRepairCompletedToast")
          : t("settings.libraryHealthRepairFinishedWithErrorsToast"),
        { variant: current.status === "completed" ? "success" : "warning" },
      )
      await runHealthScan()
      return
    }
    await new Promise((resolve) => window.setTimeout(resolve, 750))
  }
}

function downloadDiagnostics() {
  if (!report.value) return
  const payload = JSON.stringify(report.value, null, 2)
  const blob = new Blob([payload], { type: "application/json;charset=utf-8" })
  const href = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = href
  link.download = `curated-library-health-${report.value.scannedAt.replaceAll(":", "-")}.json`
  link.click()
  URL.revokeObjectURL(href)
}
</script>

<template>
  <div class="break-inside-avoid">
    <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
        <span
          class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <Activity class="size-[1.15rem]" />
        </span>
        <CardTitle class="min-w-0 text-lg tracking-tight">
          {{ t("settings.libraryHealthTitle") }}
        </CardTitle>
        <CardDescription class="col-start-2 text-pretty text-xs leading-relaxed text-muted-foreground sm:text-sm">
          {{ t("settings.libraryHealthDescription") }}
        </CardDescription>
      </CardHeader>

      <CardContent class="flex flex-col gap-3 pt-0">
        <div class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 flex-col gap-2">
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.libraryHealthScanTitle") }}
            </p>
            <p class="text-pretty text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.libraryHealthScanHint") }}
            </p>
          </div>
          <div class="flex shrink-0 flex-wrap gap-2">
            <Button
              v-if="report"
              type="button"
              variant="outline"
              class="h-auto min-h-11 rounded-2xl px-4"
              data-settings-comfortable-control
              data-library-health-export
              :disabled="reportBusy"
              @click="downloadDiagnostics"
            >
              <Download data-icon="inline-start" />
              {{ t("settings.libraryHealthExport") }}
            </Button>
            <Button
              type="button"
              class="h-auto min-h-11 rounded-2xl px-5 font-medium"
              data-settings-comfortable-control
              data-library-health-scan
              :disabled="!supported || reportBusy || repairActive"
              @click="runHealthScan"
            >
              <LoaderCircle v-if="reportBusy" data-icon="inline-start" class="animate-spin" />
              <RefreshCw v-else data-icon="inline-start" />
              {{ reportBusy ? t("settings.libraryHealthScanning") : t("settings.libraryHealthScanAction") }}
            </Button>
          </div>
        </div>

        <p v-if="!supported" class="text-xs leading-relaxed text-info sm:text-sm" role="status">
          {{ t("settings.libraryHealthWebRequired") }}
        </p>
        <p v-if="actionError" class="text-xs leading-relaxed text-danger sm:text-sm" role="alert">
          {{ actionError }}
        </p>

        <template v-if="report">
          <div class="grid grid-cols-2 gap-2 lg:grid-cols-4">
            <div class="flex min-w-0 flex-col gap-1 rounded-lg border border-border/50 bg-muted/10 p-3">
              <span class="text-xs text-muted-foreground">{{ t("settings.libraryHealthTotal") }}</span>
              <strong class="text-xl font-semibold tabular-nums text-foreground">{{ report.summary.totalFindings }}</strong>
            </div>
            <div class="flex min-w-0 flex-col gap-1 rounded-lg border border-border/50 bg-muted/10 p-3">
              <span class="text-xs text-muted-foreground">{{ t("settings.libraryHealthCritical") }}</span>
              <strong class="text-xl font-semibold tabular-nums text-foreground">{{ report.summary.criticalFindings }}</strong>
            </div>
            <div class="flex min-w-0 flex-col gap-1 rounded-lg border border-border/50 bg-muted/10 p-3">
              <span class="text-xs text-muted-foreground">{{ t("settings.libraryHealthWarnings") }}</span>
              <strong class="text-xl font-semibold tabular-nums text-foreground">{{ report.summary.warningFindings }}</strong>
            </div>
            <div class="flex min-w-0 flex-col gap-1 rounded-lg border border-border/50 bg-muted/10 p-3">
              <span class="text-xs text-muted-foreground">{{ t("settings.libraryHealthOfflineSkipped") }}</span>
              <strong class="text-xl font-semibold tabular-nums text-foreground">{{ report.summary.skippedOfflineFiles }}</strong>
            </div>
          </div>

          <div class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="flex min-w-0 items-center gap-2">
                <ShieldCheck class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                <p class="text-sm font-semibold text-foreground">{{ t("settings.libraryHealthIntegrityTitle") }}</p>
              </div>
              <Badge :variant="reportStatusVariant(report.status)">
                {{ t(`settings.libraryHealthStatus_${report.status}`) }}
              </Badge>
            </div>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.libraryHealthIntegritySummary", {
                quick: report.database.quickCheckOk ? "ok" : "failed",
                foreignKeys: report.database.foreignKeyCount,
                scannedAt: formatDate(report.scannedAt),
              }) }}
            </p>
            <div v-if="sortedCategoryCounts.length" class="flex flex-wrap gap-2">
              <Badge v-for="[category, count] in sortedCategoryCounts" :key="category" variant="outline">
                {{ categoryLabel(category) }} · {{ count }}
              </Badge>
            </div>
          </div>

          <div
            v-if="report.status === 'healthy'"
            :class="statusPanelClass('success')"
            role="status"
          >
            <div class="flex items-start gap-3">
              <CheckCircle2 class="mt-0.5 size-5 shrink-0 text-success" aria-hidden="true" />
              <div class="flex min-w-0 flex-col gap-1">
                <p class="text-sm font-semibold text-foreground">{{ t("settings.libraryHealthHealthyTitle") }}</p>
                <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">{{ t("settings.libraryHealthHealthyHint") }}</p>
              </div>
            </div>
          </div>

          <div v-if="repair" class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/10 p-4" aria-live="polite">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <p class="text-sm font-semibold text-foreground">{{ t("settings.libraryHealthRepairProgressTitle") }}</p>
              <Badge :variant="repairStatusVariant(repair.status)">
                {{ t(`settings.libraryHealthRepairStatus_${repair.status}`) }}
              </Badge>
            </div>
            <Progress :model-value="repairProgress" :aria-label="t('settings.libraryHealthRepairProgressTitle')" />
            <p class="text-xs tabular-nums text-muted-foreground sm:text-sm">
              {{ t("settings.libraryHealthRepairProgressSummary", {
                completed: repair.completedItems,
                total: repair.totalItems,
                succeeded: repair.succeededItems,
                failed: repair.failedItems,
              }) }}
            </p>
            <ul v-if="repair.items.length && !repairActive" class="flex flex-col gap-2" :aria-label="t('settings.libraryHealthRepairResults')">
              <li v-for="item in repair.items" :key="item.findingId" class="flex flex-col gap-1 rounded-lg border border-border/40 bg-background/30 px-3 py-2 sm:flex-row sm:items-center sm:justify-between">
                <span class="min-w-0 truncate text-sm text-foreground">{{ item.label }}</span>
                <Badge :variant="item.status === 'succeeded' ? 'success' : 'danger'">{{ repairItemStatusLabel(item.status) }}</Badge>
              </li>
            </ul>
          </div>

          <div v-if="report.summary.categoryCounts.metadata_missing || report.summary.categoryCounts.metadata_failed" class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex min-w-0 flex-col gap-2">
              <p class="text-sm font-semibold text-foreground">{{ t("settings.libraryHealthMetadataRepairTitle") }}</p>
              <p class="text-pretty text-xs leading-relaxed text-muted-foreground sm:text-sm">{{ t("settings.libraryHealthMetadataRepairHint") }}</p>
            </div>
            <div class="flex shrink-0 flex-wrap gap-2">
              <Button
                v-if="report.summary.categoryCounts.metadata_missing"
                type="button"
                variant="outline"
                class="h-auto min-h-11 rounded-2xl px-4"
                data-settings-comfortable-control
                data-library-health-repair-missing
                :disabled="repairActive"
                @click="openRepairDialog('metadata_missing')"
              >
                <Wrench data-icon="inline-start" />
                {{ t("settings.libraryHealthRepairMissing", { count: report.summary.categoryCounts.metadata_missing }) }}
              </Button>
              <Button
                v-if="report.summary.categoryCounts.metadata_failed"
                type="button"
                variant="outline"
                class="h-auto min-h-11 rounded-2xl px-4"
                data-settings-comfortable-control
                data-library-health-repair-failed
                :disabled="repairActive"
                @click="openRepairDialog('metadata_failed')"
              >
                <RefreshCw data-icon="inline-start" />
                {{ t("settings.libraryHealthRepairFailed", { count: report.summary.categoryCounts.metadata_failed }) }}
              </Button>
            </div>
          </div>

          <div v-if="visibleFindings.length" class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4">
            <div class="flex flex-wrap items-end justify-between gap-2">
              <div class="flex min-w-0 flex-col gap-1">
                <p class="text-sm font-semibold text-foreground">{{ t("settings.libraryHealthFindingsTitle") }}</p>
                <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">{{ t("settings.libraryHealthFindingsHint") }}</p>
              </div>
              <span class="text-xs tabular-nums text-muted-foreground">{{ visibleFindings.length }} / {{ report.summary.totalFindings }}</span>
            </div>
            <Separator />
            <ul class="flex flex-col gap-2">
              <li v-for="finding in visibleFindings" :key="finding.id" class="flex min-w-0 flex-col gap-2 rounded-lg border border-border/40 bg-background/30 p-3">
                <div class="flex flex-wrap items-center gap-2">
                  <Badge :variant="severityVariant(finding.severity)">{{ categoryLabel(finding.category) }}</Badge>
                  <strong class="min-w-0 truncate text-sm font-medium text-foreground">{{ finding.label }}</strong>
                </div>
                <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">{{ finding.message }}</p>
                <code v-if="finding.path" class="break-all text-xs text-muted-foreground">{{ finding.path }}</code>
              </li>
            </ul>
            <p v-if="hiddenFindingCount > 0 || report.truncated" class="text-xs leading-relaxed text-muted-foreground">
              {{ t("settings.libraryHealthFindingsLimited", { count: hiddenFindingCount }) }}
            </p>
          </div>
        </template>

        <div v-else-if="!reportBusy" class="flex flex-col items-start gap-2 rounded-lg border border-dashed border-border/60 bg-muted/5 p-4">
          <FileWarning class="size-5 text-muted-foreground" aria-hidden="true" />
          <p class="text-sm font-semibold text-foreground">{{ t("settings.libraryHealthIdleTitle") }}</p>
          <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">{{ t("settings.libraryHealthIdleHint") }}</p>
        </div>
      </CardContent>
    </Card>

    <Dialog v-model:open="repairDialogOpen">
      <DialogContent class="rounded-2xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t("settings.libraryHealthRepairConfirmTitle") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("settings.libraryHealthRepairConfirmDescription", {
              count: Math.min(25, repairCount),
              category: categoryLabel(pendingRepairCategory),
            }) }}
          </DialogDescription>
        </DialogHeader>
        <div class="flex items-start gap-3 rounded-lg border border-warning/25 bg-warning/[0.07] p-3 text-sm text-foreground">
          <AlertTriangle class="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
          <p class="text-xs leading-relaxed sm:text-sm">{{ t("settings.libraryHealthRepairConfirmWarning") }}</p>
        </div>
        <DialogFooter class="gap-3">
          <Button type="button" variant="outline" class="min-h-11 rounded-2xl" :disabled="repairSubmitting" @click="repairDialogOpen = false">
            {{ t("common.cancel") }}
          </Button>
          <Button
            type="button"
            class="min-h-11 rounded-2xl"
            data-library-health-repair-confirm
            :disabled="repairSubmitting"
            @click="confirmRepair"
          >
            <LoaderCircle v-if="repairSubmitting" data-icon="inline-start" class="animate-spin" />
            <Wrench v-else data-icon="inline-start" />
            {{ repairSubmitting ? t("settings.libraryHealthRepairStarting") : t("settings.libraryHealthRepairConfirmAction") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

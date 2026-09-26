<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  DatabaseBackup,
  FileCheck2,
  FolderOpen,
  RotateCw,
  ShieldCheck,
} from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type {
  BackupManifestDTO,
  BackupRestorePreflightDTO,
  BackupVerificationDTO,
} from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Separator } from "@/components/ui/separator"
import { pushAppToast } from "@/composables/use-app-toast"
import {
  buildBackupFilename,
  ensureBackupExtension,
  joinBackupDestination,
} from "@/lib/backup-path"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import { pickLibraryDirectory } from "@/lib/pick-directory"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{
  supported: boolean
}>()

const { t, locale } = useI18n()
const libraryService = useLibraryService()
const backupDirectoryDraft = ref("")
const backupPathDraft = ref("")
const directoryError = ref("")
const directoryPersistenceError = ref("")
const pathError = ref("")
const pickerHint = ref("")
const actionError = ref("")
const busyAction = ref<"pick" | "create" | "verify" | "preflight" | null>(null)
const createdManifest = ref<BackupManifestDTO | null>(null)
const verification = ref<BackupVerificationDTO | null>(null)
const preflight = ref<BackupRestorePreflightDTO | null>(null)

const busy = computed(() => busyAction.value !== null)

let lastSyncedDirectory = ""
watch(
  () => libraryService.backupDirectory.value,
  (value) => {
    const next = value.trim()
    const current = backupDirectoryDraft.value.trim()
    if (current === "" || current === lastSyncedDirectory) {
      backupDirectoryDraft.value = next
    }
    lastSyncedDirectory = next
  },
  { immediate: true },
)

function formatError(error: unknown): string {
  if (error instanceof HttpClientError && error.apiError?.message) {
    return error.apiError.message
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message
  }
  return t("settings.backupUnknownError")
}

function validatedBackupPath(): string | null {
  pathError.value = ""
  const candidate = ensureBackupExtension(backupPathDraft.value)
  if (!candidate || !isAbsoluteLibraryPath(candidate)) {
    pathError.value = t("settings.backupPathAbsolute")
    return null
  }
  backupPathDraft.value = candidate
  return candidate
}

function validatedBackupDirectory(): string | null {
  directoryError.value = ""
  const candidate = backupDirectoryDraft.value.trim()
  if (!candidate || !isAbsoluteLibraryPath(candidate)) {
    directoryError.value = t("settings.backupDirectoryAbsolute")
    return null
  }
  backupDirectoryDraft.value = candidate
  return candidate
}

async function pickBackupDirectory() {
  if (!props.supported || busy.value) return
  directoryError.value = ""
  pickerHint.value = ""
  busyAction.value = "pick"
  try {
    const outcome = await pickLibraryDirectory()
    if (outcome.status === "ok") {
      backupDirectoryDraft.value = outcome.path
    } else if (outcome.status === "hint") {
      pickerHint.value = outcome.message
    } else if (outcome.status === "unsupported") {
      pickerHint.value = t("settings.backupPickerUnsupported")
    }
  } finally {
    busyAction.value = null
  }
}

async function createAndVerifyBackup() {
  if (!props.supported || busy.value) return
  const backupDirectory = validatedBackupDirectory()
  if (!backupDirectory) return
  const backupPath = joinBackupDestination(backupDirectory, buildBackupFilename())
  actionError.value = ""
  directoryPersistenceError.value = ""
  pickerHint.value = ""
  preflight.value = null
  busyAction.value = "create"
  try {
    createdManifest.value = await libraryService.createBackup(backupPath)
    backupPathDraft.value = backupPath
    try {
      await libraryService.setBackupDirectory(backupDirectory)
    } catch (error) {
      directoryPersistenceError.value = t("settings.backupDirectorySaveFailed", {
        error: formatError(error),
      })
    }
    verification.value = await libraryService.verifyBackup(backupPath)
    if (!verification.value.valid) {
      actionError.value = t("settings.backupCreatedVerificationFailed")
      return
    }
    pushAppToast(t("settings.backupCreatedToast"), {
      variant: "success",
      durationMs: 3600,
    })
  } catch (error) {
    actionError.value = formatError(error)
  } finally {
    busyAction.value = null
  }
}

async function verifyExistingBackup() {
  if (!props.supported || busy.value) return
  const backupPath = validatedBackupPath()
  if (!backupPath) return
  actionError.value = ""
  createdManifest.value = null
  preflight.value = null
  busyAction.value = "verify"
  try {
    verification.value = await libraryService.verifyBackup(backupPath)
  } catch (error) {
    actionError.value = formatError(error)
  } finally {
    busyAction.value = null
  }
}

async function preflightExistingBackup() {
  if (!props.supported || busy.value) return
  const backupPath = validatedBackupPath()
  if (!backupPath) return
  actionError.value = ""
  createdManifest.value = null
  busyAction.value = "preflight"
  try {
    preflight.value = await libraryService.preflightBackupRestore(backupPath)
    verification.value = preflight.value.verification
  } catch (error) {
    actionError.value = formatError(error)
  } finally {
    busyAction.value = null
  }
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${new Intl.NumberFormat(locale.value, { maximumFractionDigits: index === 0 ? 0 : 1 }).format(value / 1024 ** index)} ${units[index]}`
}
</script>

<template>
  <section
    aria-labelledby="settings-backup-title"
    class="flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
    data-settings-maintenance-block="backup"
  >
    <div class="flex flex-col gap-2">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <SettingsHint :text="[t('settings.backupCardDesc'), t('settings.backupOfflineRestoreHint')].join('\n\n')">
          <h3 id="settings-backup-title" class="min-w-0 text-sm font-semibold text-foreground">
            {{ t("settings.backupCardTitle") }}
          </h3>
        </SettingsHint>
        <Badge v-if="!supported" variant="secondary">
          {{ t("settings.backupWebRequired") }}
        </Badge>
      </div>
    </div>

    <FieldGroup class="gap-4">
      <Field class="gap-3" :data-invalid="Boolean(directoryError)" :data-disabled="!supported || busy">
        <SettingsHint :text="t('settings.backupDirectoryHint')">
          <FieldLabel for="settings-backup-directory">
            {{ t("settings.backupDirectoryLabel") }}
          </FieldLabel>
        </SettingsHint>
        <div class="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
          <Input
            id="settings-backup-directory"
            v-model="backupDirectoryDraft"
            :disabled="!supported || busy"
            :aria-invalid="Boolean(directoryError)"
            aria-describedby="settings-backup-directory-help"
            :placeholder="t('settings.backupDirectoryPlaceholder')"
            autocomplete="off"
            class="min-w-0 flex-1"
            data-settings-backup-directory
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="h-auto min-h-11 sm:h-8 sm:min-h-8"
            :disabled="!supported || busy"
            data-settings-comfortable-control
            data-settings-backup-pick
            @click="pickBackupDirectory"
          >
            <FolderOpen data-icon="inline-start" />
            {{ t("settings.backupPickDirectory") }}
          </Button>
        </div>
        <p v-if="directoryError" id="settings-backup-directory-help" class="text-xs text-destructive" role="alert">
          {{ directoryError }}
        </p>
        <p v-else-if="pickerHint" id="settings-backup-directory-help" class="text-xs text-muted-foreground">
          {{ pickerHint }}
        </p>
        <span v-else id="settings-backup-directory-help" class="sr-only">{{ t("settings.backupDirectoryHint") }}</span>
        <div class="flex flex-wrap justify-end gap-2">
          <Button
            type="button"
            size="sm"
            class="h-auto min-h-11 sm:h-8 sm:min-h-8"
            :disabled="!supported || busy"
            data-settings-comfortable-control
            data-settings-backup-create
            @click="createAndVerifyBackup"
          >
            <RotateCw v-if="busyAction === 'create'" data-icon="inline-start" class="animate-spin" />
            <DatabaseBackup v-else data-icon="inline-start" />
            {{ t("settings.backupCreate") }}
          </Button>
        </div>
      </Field>

      <Separator />

      <Field class="gap-3" :data-invalid="Boolean(pathError)" :data-disabled="!supported || busy">
        <SettingsHint :text="t('settings.backupPathHint')">
          <FieldLabel for="settings-backup-path">
            {{ t("settings.backupPathLabel") }}
          </FieldLabel>
        </SettingsHint>
        <Input
          id="settings-backup-path"
          v-model="backupPathDraft"
          :disabled="!supported || busy"
          :aria-invalid="Boolean(pathError)"
          aria-describedby="settings-backup-path-help"
          :placeholder="t('settings.backupPathPlaceholder')"
          autocomplete="off"
          data-settings-backup-path
        />
        <p v-if="pathError" id="settings-backup-path-help" class="text-xs text-destructive" role="alert">
          {{ pathError }}
        </p>
        <span v-else id="settings-backup-path-help" class="sr-only">{{ t("settings.backupPathHint") }}</span>
        <div class="flex flex-wrap justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="h-auto min-h-11 sm:h-8 sm:min-h-8"
            :disabled="!supported || busy"
            data-settings-comfortable-control
            data-settings-backup-verify
            @click="verifyExistingBackup"
          >
            <FileCheck2 data-icon="inline-start" />
            {{ t("settings.backupVerify") }}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="h-auto min-h-11 sm:h-8 sm:min-h-8"
            :disabled="!supported || busy"
            data-settings-comfortable-control
            data-settings-backup-preflight
            @click="preflightExistingBackup"
          >
            <ShieldCheck data-icon="inline-start" />
            {{ t("settings.backupPreflight") }}
          </Button>
        </div>
      </Field>
    </FieldGroup>

    <p v-if="actionError" class="text-sm text-destructive" role="alert">
      {{ actionError }}
    </p>
    <p v-if="directoryPersistenceError" class="text-sm text-warning" role="alert">
      {{ directoryPersistenceError }}
    </p>

    <template v-if="createdManifest || verification || preflight">
      <Separator />
      <div class="flex flex-col gap-3" aria-live="polite">
        <div v-if="createdManifest" class="flex flex-wrap items-center gap-2 text-sm">
          <Badge variant="success">{{ t("settings.backupCreated") }}</Badge>
          <span class="text-muted-foreground">
            {{ t("settings.backupManifestSummary", {
              files: createdManifest.files.length,
              migrations: createdManifest.schemaMigrations.length,
            }) }}
          </span>
        </div>

        <div v-if="verification" class="flex flex-col gap-2 text-sm">
          <div class="flex flex-wrap items-center gap-2">
            <Badge :variant="verification.valid ? 'success' : 'danger'">
              {{ verification.valid ? t("settings.backupValid") : t("settings.backupInvalid") }}
            </Badge>
            <span class="text-muted-foreground">
              {{ t("settings.backupIntegritySummary", {
                quick: verification.databaseIntegrity.quickCheck || '—',
                foreignKeys: verification.databaseIntegrity.foreignKeyViolations,
              }) }}
            </span>
          </div>
          <ul v-if="verification.errors.length" class="list-disc ps-5 text-destructive">
            <li v-for="error in verification.errors" :key="error">{{ error }}</li>
          </ul>
          <ul v-if="verification.warnings.length" class="list-disc ps-5 text-muted-foreground">
            <li v-for="warning in verification.warnings" :key="warning">{{ warning }}</li>
          </ul>
        </div>

        <div v-if="preflight" class="flex flex-col gap-2 text-sm">
          <div class="flex flex-wrap items-center gap-2">
            <Badge :variant="preflight.canRestore ? 'success' : 'danger'">
              {{ preflight.canRestore ? t("settings.backupPreflightReady") : t("settings.backupPreflightBlocked") }}
            </Badge>
            <span class="text-muted-foreground">
              {{ t("settings.backupCapacitySummary", {
                required: formatBytes(preflight.requiredBytes),
                available: preflight.availableBytesKnown ? formatBytes(preflight.availableBytes) : '—',
              }) }}
            </span>
          </div>
          <ul v-if="preflight.errors.length" class="list-disc ps-5 text-destructive">
            <li v-for="error in preflight.errors" :key="error">{{ error }}</li>
          </ul>
          <ul v-if="preflight.warnings.length" class="list-disc ps-5 text-muted-foreground">
            <li v-for="warning in preflight.warnings" :key="warning">{{ warning }}</li>
          </ul>
        </div>
      </div>
    </template>
  </section>
</template>

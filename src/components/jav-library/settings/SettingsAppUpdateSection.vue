<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { ArrowUpRight, Download, Loader2, Play, RefreshCw } from "lucide-vue-next"
import { pushAppToast } from "@/composables/use-app-toast"
import { useAppUpdate } from "@/composables/use-app-update"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { statusPanelClass, statusTextClass } from "@/lib/ui/status-tone"
import { cn } from "@/lib/utils"

const props = withDefaults(
  defineProps<{
    backendVersionDisplay?: string | null
    backendVersionStatus?: "default" | "loading" | "error"
  }>(),
  {
    backendVersionDisplay: "",
    backendVersionStatus: "default",
  },
)

const { t } = useI18n()
const {
  summary,
  status,
  loading,
  downloading,
  installing,
  hasUpdateBadge,
  ensureLoaded,
  checkNow,
  checkNowSilent,
  downloadInstaller,
  installUpdate,
} = useAppUpdate()
const releaseNotesExpanded = ref(false)

onMounted(() => {
  ensureLoaded()
})

const panelTone = computed(() => {
  switch (status.value) {
    case "update-available":
      return "warning"
    case "up-to-date":
      return "success"
    case "error":
      return "danger"
    default:
      return "info"
  }
})

const title = computed(() => {
  switch (status.value) {
    case "checking":
      return t("settings.appUpdateChecking")
    case "update-available":
      return t("settings.appUpdateAvailableTitle")
    case "up-to-date":
      return t("settings.appUpdateCurrentTitle")
    case "error":
      return t("settings.appUpdateErrorTitle")
    case "unsupported":
      return t("settings.appUpdateUnsupportedTitle")
    default:
      return t("settings.appUpdateIdleTitle")
  }
})

const description = computed(() => {
  const current = summary.value
  switch (status.value) {
    case "checking":
      return t("settings.appUpdateCheckingBody")
    case "update-available":
      return t("settings.appUpdateAvailableBody")
    case "up-to-date":
      return t("settings.appUpdateCurrentBody")
    case "error":
      return current?.errorMessage?.trim() || t("settings.appUpdateErrorBody")
    case "unsupported":
      return current?.errorMessage?.trim() || t("settings.appUpdateUnsupportedBody")
    default:
      return t("settings.appUpdateIdleBody")
  }
})

const backendVersionDisplay = computed(() => props.backendVersionDisplay?.trim() ?? "")
const installerVersionSummary = computed(() => {
  const installed = summary.value?.installedVersion?.trim() ?? ""
  const latest = summary.value?.latestVersion?.trim() ?? ""

  if (installed && latest) {
    return installed === latest ? installed : `${installed} -> ${latest}`
  }
  return installed || latest
})
const releaseUrl = computed(() => summary.value?.releaseUrl?.trim() ?? "")
const installerDownloadUrl = computed(() => summary.value?.installerDownloadUrl?.trim() ?? "")
const installerSha256 = computed(() => summary.value?.installerSha256?.trim() ?? "")
const artifactStatus = computed(() => summary.value?.artifactStatus?.trim() ?? "")
const installReady = computed(
  () => summary.value?.installReady === true && artifactStatus.value === "verified",
)
const canDownloadInstaller = computed(
  () =>
    status.value === "update-available" &&
    !!installerDownloadUrl.value &&
    !!installerSha256.value &&
    !installReady.value,
)
const releaseTitle = computed(() => summary.value?.releaseName?.trim() || summary.value?.latestVersion || "")
const releaseNotesSnippet = computed(() => summary.value?.releaseNotesSnippet?.trim() ?? "")

const lastSilentNotesRefreshAt = ref(0)

async function toggleReleaseNotes() {
  const next = !releaseNotesExpanded.value
  releaseNotesExpanded.value = next
  if (!next || !releaseNotesSnippet.value) {
    return
  }
  const now = Date.now()
  if (now - lastSilentNotesRefreshAt.value < 60_000) {
    return
  }
  lastSilentNotesRefreshAt.value = now
  await checkNowSilent()
}

async function handleCheckNow() {
  const next = await checkNow()
  if (next?.status === "update-available") {
    pushAppToast(
      t("settings.appUpdateToastAvailable", {
        version: next.latestVersion ?? "-",
      }),
      {
        variant: "warning",
        notification: {
          type: "update",
          title: t("notificationCenter.titles.updateAvailable"),
          source: { route: "/settings?section=about" },
        },
      },
    )
    return
  }
  if (next?.status === "up-to-date") {
    pushAppToast(t("settings.appUpdateToastCurrent"), { variant: "success" })
    return
  }
  if (next?.status === "error") {
    pushAppToast(next.errorMessage?.trim() || t("settings.appUpdateErrorBody"), {
      variant: "destructive",
    })
  }
}

async function handleDownloadInstaller() {
  const next = await downloadInstaller()
  if (next?.artifactStatus === "verified" && next.installReady) {
    pushAppToast(t("settings.appUpdateInstallReadyAction"), { variant: "success" })
    return
  }
  if (next?.artifactStatus === "failed" || next?.status === "error") {
    pushAppToast(next.lastInstallError?.trim() || next.errorMessage?.trim() || t("settings.appUpdateErrorBody"), {
      variant: "destructive",
    })
  }
}

async function handleInstallUpdate() {
  const next = await installUpdate("interactive")
  if (next?.artifactStatus === "install-launched") {
    pushAppToast(t("settings.appUpdateInstallingAction"), { variant: "success" })
    return
  }
  if (next?.lastInstallError?.trim() || next?.status === "error") {
    pushAppToast(next.lastInstallError?.trim() || next.errorMessage?.trim() || t("settings.appUpdateErrorBody"), {
      variant: "destructive",
    })
  }
}
</script>

<template>
  <section
    class="rounded-xl border border-border/60 bg-background/40 p-4"
    :class="statusPanelClass(panelTone)"
    data-app-update-section
  >
    <div class="flex flex-col gap-4">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0 space-y-2">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.appUpdateSectionTitle") }}
            </p>
            <Badge
              v-if="hasUpdateBadge"
              variant="secondary"
              class="max-h-5 shrink-0 !h-5 !min-h-0 !py-0 rounded-full border border-amber-600/35 bg-amber-500/20 px-1.5 text-[10px] font-semibold uppercase leading-none tracking-normal text-amber-950 dark:border-amber-500/30 dark:bg-amber-500/15 dark:text-amber-100"
            >
              New
            </Badge>
          </div>
          <p :class="cn('text-sm font-medium', statusTextClass(panelTone))">
            {{ title }}
          </p>
          <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
            {{ description }}
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Button
            v-if="status !== 'unsupported'"
            type="button"
            variant="outline"
            class="rounded-2xl"
            :disabled="loading"
            data-app-update-check
            @click="handleCheckNow"
          >
            <Loader2 v-if="loading" class="mr-2 size-4 animate-spin" aria-hidden="true" />
            <RefreshCw v-else class="mr-2 size-4" aria-hidden="true" />
            {{ loading ? t("settings.appUpdateCheckingAction") : t("settings.appUpdateCheckAction") }}
          </Button>

          <Button
            v-if="installReady"
            type="button"
            class="rounded-2xl"
            :disabled="installing"
            data-app-update-install
            @click="handleInstallUpdate"
          >
            <Loader2 v-if="installing" class="mr-2 size-4 animate-spin" aria-hidden="true" />
            <Play v-else class="mr-2 size-4" aria-hidden="true" />
            {{ installing ? t("settings.appUpdateInstallingAction") : t("settings.appUpdateInstallReadyAction") }}
          </Button>

          <Button
            v-else-if="canDownloadInstaller"
            type="button"
            class="rounded-2xl"
            :disabled="downloading"
            data-app-update-download-installer
            @click="handleDownloadInstaller"
          >
            <Loader2 v-if="downloading" class="mr-2 size-4 animate-spin" aria-hidden="true" />
            <Download v-else class="mr-2 size-4" aria-hidden="true" />
            {{
              downloading
                ? t("settings.appUpdateDownloadingAction")
                : t("settings.appUpdateDownloadAndInstallAction")
            }}
          </Button>

          <Button
            v-if="releaseUrl"
            as-child
            class="rounded-2xl"
            :variant="status === 'update-available' ? 'secondary' : 'outline'"
          >
            <a
              :href="releaseUrl"
              target="_blank"
              rel="noopener noreferrer"
              data-app-update-release
            >
              <ArrowUpRight class="mr-2 size-4" aria-hidden="true" />
              {{ t("settings.appUpdateOpenReleaseAction") }}
            </a>
          </Button>
        </div>
      </div>

      <dl class="grid gap-3 sm:grid-cols-2">
        <div
          v-if="backendVersionDisplay"
          class="rounded-lg border border-border/50 bg-background/55 px-3 py-2.5"
        >
          <dt class="text-xs font-medium text-muted-foreground">
            {{ t("settings.aboutVersionLabel") }}
          </dt>
          <dd
            class="mt-1 text-sm"
            :class="[
              props.backendVersionStatus === 'error' ? 'text-destructive' : 'text-foreground',
              props.backendVersionStatus === 'default' ? 'font-mono' : '',
            ]"
          >
            {{ backendVersionDisplay }}
          </dd>
        </div>

        <div
          v-if="installerVersionSummary"
          class="rounded-lg border border-border/50 bg-background/55 px-3 py-2.5"
        >
          <dt class="text-xs font-medium text-muted-foreground">
            {{ t("settings.aboutInstallerVersionLabel") }}
          </dt>
          <dd class="mt-1 font-mono text-sm text-foreground">
            {{ installerVersionSummary }}
          </dd>
        </div>
      </dl>

      <div
        v-if="releaseTitle || releaseNotesSnippet"
        class="rounded-lg border border-dashed border-border/60 bg-background/25 px-3 py-3"
      >
        <p v-if="releaseTitle" class="text-sm font-semibold text-foreground">
          {{ releaseTitle }}
        </p>
        <div
          v-if="releaseNotesSnippet && !releaseNotesExpanded"
          key="notes-collapsed"
          data-app-update-release-notes
          class="mt-1 line-clamp-3 whitespace-pre-wrap break-words text-xs leading-relaxed text-muted-foreground sm:text-sm"
        >
          {{ releaseNotesSnippet }}
        </div>
        <div
          v-else-if="releaseNotesSnippet"
          key="notes-expanded"
          data-app-update-release-notes
          class="mt-1 block max-h-[min(85vh,44rem)] min-h-0 overflow-y-auto overflow-x-hidden whitespace-pre-wrap break-words rounded-md border border-border/40 bg-background/50 px-3 py-2 text-xs leading-relaxed text-muted-foreground [overflow-anchor:none] sm:text-sm"
        >
          {{ releaseNotesSnippet }}
        </div>
        <Button
          v-if="releaseNotesSnippet"
          type="button"
          variant="ghost"
          class="mt-2 h-auto rounded-xl px-2 py-1 text-xs text-muted-foreground"
          data-app-update-release-notes-toggle
          @click="toggleReleaseNotes"
        >
          {{
            releaseNotesExpanded
              ? t("settings.appUpdateReleaseNotesCollapse")
              : t("settings.appUpdateReleaseNotesExpand")
          }}
        </Button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { ArrowUpRight, Loader2, RefreshCw } from "lucide-vue-next"
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
const { summary, status, loading, hasUpdateBadge, ensureLoaded, checkNow } = useAppUpdate()
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
const releaseTitle = computed(() => summary.value?.releaseName?.trim() || summary.value?.latestVersion || "")
const releaseNotesSnippet = computed(() => summary.value?.releaseNotesSnippet?.trim() ?? "")

async function handleCheckNow() {
  const next = await checkNow()
  if (next?.status === "update-available") {
    pushAppToast(
      t("settings.appUpdateToastAvailable", {
        version: next.latestVersion ?? "-",
      }),
      { variant: "warning" },
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
              class="rounded-full border border-amber-500/30 bg-amber-500/15 text-[11px] font-semibold uppercase tracking-[0.18em] text-amber-100"
            >
              New
            </Badge>
          </div>
          <p :class="cn('text-sm font-medium', statusTextClass(panelTone))">
            {{ title }}
          </p>
          <p class="text-sm leading-6 text-muted-foreground">
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
            v-if="releaseUrl"
            as-child
            class="rounded-2xl"
            :variant="status === 'update-available' ? 'default' : 'secondary'"
          >
            <a :href="releaseUrl" target="_blank" rel="noopener noreferrer" data-app-update-download>
              <ArrowUpRight class="mr-2 size-4" aria-hidden="true" />
              {{ t("settings.appUpdateDownloadAction") }}
            </a>
          </Button>
        </div>
      </div>

      <dl class="grid gap-3 sm:grid-cols-2">
        <div
          v-if="backendVersionDisplay"
          class="rounded-lg border border-border/50 bg-background/55 px-3 py-2.5"
        >
          <dt class="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
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
          <dt class="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
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
        <p v-if="releaseTitle" class="text-sm font-medium text-foreground">
          {{ releaseTitle }}
        </p>
        <p
          v-if="releaseNotesSnippet"
          data-app-update-release-notes
          class="mt-1 text-sm leading-6 text-muted-foreground"
          :class="{ 'line-clamp-1': !releaseNotesExpanded }"
        >
          {{ releaseNotesSnippet }}
        </p>
        <Button
          v-if="releaseNotesSnippet"
          type="button"
          variant="ghost"
          class="mt-2 h-auto rounded-xl px-2 py-1 text-xs text-muted-foreground"
          data-app-update-release-notes-toggle
          @click="releaseNotesExpanded = !releaseNotesExpanded"
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

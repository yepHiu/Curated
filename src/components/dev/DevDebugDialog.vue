<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Switch } from "@/components/ui/switch"
import SettingsHint from "@/components/jav-library/settings/SettingsHint.vue"
import SettingsScopeBadge from "@/components/jav-library/settings/SettingsScopeBadge.vue"
import SettingsHomepageDevTools from "@/components/jav-library/settings/SettingsHomepageDevTools.vue"
import SettingsLoggingSection from "@/components/jav-library/settings/SettingsLoggingSection.vue"
import { useLibraryService } from "@/services/library-service"
import { useServerLocalAccess } from "@/composables/use-server-local-access"

const open = defineModel<boolean>("open", { default: false })
const tab = defineModel<string>("tab", { default: "actions" })
const { t } = useI18n()
const service = useLibraryService()
const { isServerLocal } = useServerLocalAccess()
const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"
let returnFocus: HTMLElement | null = null
function restoreFocus(event: Event) {
  event.preventDefault()
  void nextTick(() => {
    const target = returnFocus?.isConnected && returnFocus.getClientRects().length
      ? returnFocus
      : document.querySelector<HTMLElement>("[data-dev-debug-trigger]")
    target?.focus()
  })
}
const ready = ref(false)
const loading = ref(false)
const loadError = ref("")
const saving = ref(false)
const saveError = ref("")
const forceHls = computed(() => Boolean(service.playerSettings.value.forceStreamPush))
const streamEnabled = computed(() => service.playerSettings.value.streamPushEnabled !== false)
const probeBusy = ref(false)
const probeResult = ref("")
const probeFailed = ref(false)

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : t("debug.failed")
}

async function loadSettings() {
  if (loading.value || saving.value) return
  loading.value = true
  ready.value = false
  loadError.value = ""
  try {
    await service.refreshSettings()
    ready.value = true
  } catch (error) {
    loadError.value = errorMessage(error)
  } finally {
    loading.value = false
  }
}

watch(open, (value) => {
  if (value) {
    returnFocus = document.querySelector<HTMLElement>(
      tab.value === "performance" ? "[data-dev-performance-trigger]" : "[data-dev-debug-trigger]",
    )
    void loadSettings()
  }
  else tab.value = "actions"
}, { immediate: true })

async function setForceHls(value: boolean) {
  if (!ready.value || !isServerLocal.value || !streamEnabled.value || saving.value) return
  saving.value = true
  saveError.value = ""
  try {
    await service.patchPlayerSettings({ forceStreamPush: value })
  } catch (error) {
    saveError.value = errorMessage(error)
  } finally {
    saving.value = false
  }
}

async function probe(target: "metadata" | "network") {
  if (!useWebApi || probeBusy.value) return
  probeBusy.value = true
  probeResult.value = ""
  probeFailed.value = false
  try {
    const result = await (target === "metadata" ? service.pingProxyJavbus() : service.pingProxyGoogle())
    probeFailed.value = !result.ok
    probeResult.value = `${t(target === "metadata" ? "debug.metadata" : "debug.network")} · ${result.ok ? t("debug.passed") : t("debug.failed")} · ${result.latencyMs} ms${result.message ? ` · ${result.message}` : ""}`
  } catch (error) {
    probeFailed.value = true
    probeResult.value = errorMessage(error)
  } finally {
    probeBusy.value = false
  }
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent @close-auto-focus="restoreFocus" class="flex max-h-[calc(100dvh-2rem)] flex-col overflow-hidden p-4 sm:max-w-3xl sm:p-6">
      <DialogHeader class="shrink-0 pr-6">
        <DialogTitle>{{ t("debug.title") }}</DialogTitle>
        <DialogDescription class="sr-only">{{ t("debug.description") }}</DialogDescription>
      </DialogHeader>
      <Tabs v-model="tab" class="flex min-h-0 flex-col gap-4">
        <TabsList class="grid w-full shrink-0 grid-cols-3">
          <TabsTrigger value="actions">{{ t("debug.actions") }}</TabsTrigger>
          <TabsTrigger value="performance">{{ t("debug.performance") }}</TabsTrigger>
          <TabsTrigger value="logs">{{ t("debug.logs") }}</TabsTrigger>
        </TabsList>
        <div class="min-h-0 overflow-y-auto overscroll-contain">
          <TabsContent value="actions" class="mt-0 space-y-3">
            <section class="rounded-xl border border-border bg-card p-4">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <SettingsHint :text="t('debug.reloadHint')">
                  <h3 class="text-sm font-semibold">{{ t("debug.settings") }}</h3>
                </SettingsHint>
                <Button variant="outline" size="sm" :disabled="loading || saving" @click="loadSettings">
                  {{ loading ? t("debug.loading") : t("debug.reload") }}
                </Button>
              </div>
              <p v-if="ready" class="mt-2 text-xs text-muted-foreground" role="status">{{ t("debug.loaded") }}</p>
            </section>
            <section class="space-y-3 rounded-xl border border-border bg-card p-4">
              <div class="flex flex-wrap items-center gap-2">
                <SettingsHint :text="t('debug.networkHint')">
                  <h3 class="text-sm font-semibold">{{ t("debug.connectivity") }}</h3>
                </SettingsHint>
                <SettingsScopeBadge scope="server" />
              </div>
              <p v-if="!useWebApi" class="text-xs text-muted-foreground">{{ t("debug.webOnly") }}</p>
              <div class="flex flex-wrap gap-2" :aria-busy="probeBusy">
                <Button variant="outline" size="sm" :disabled="!useWebApi || probeBusy" @click="probe('metadata')">{{ t("debug.metadata") }}</Button>
                <Button variant="outline" size="sm" :disabled="!useWebApi || probeBusy" @click="probe('network')">{{ t("debug.network") }}</Button>
              </div>
              <p v-if="probeBusy" class="text-xs text-muted-foreground" role="status">{{ t("debug.running") }}</p>
              <p v-if="probeResult" class="break-words text-xs" :class="probeFailed ? 'text-destructive' : 'text-muted-foreground'" role="status">{{ probeResult }}</p>
            </section>
            <section v-if="isServerLocal" class="space-y-3 rounded-xl border border-border bg-card p-4">
              <div class="flex items-center justify-between gap-3">
                <div class="flex flex-wrap items-center gap-2">
                  <SettingsHint :text="t('settings.playbackForceStreamPushHint')">
                    <h3 id="debug-force-hls-label" class="text-sm font-semibold">{{ t("settings.playbackForceStreamPush") }}</h3>
                  </SettingsHint>
                  <SettingsScopeBadge scope="server" />
                </div>
                <Switch :model-value="forceHls" :disabled="!ready || !streamEnabled || saving" aria-labelledby="debug-force-hls-label" @update:model-value="setForceHls" />
              </div>
              <p v-if="ready && !streamEnabled" class="text-xs text-muted-foreground">{{ t("debug.streamDisabled") }}</p>
              <p v-if="saving" class="text-xs text-muted-foreground" role="status">{{ t("common.saving") }}</p>
              <p v-if="saveError" class="text-xs text-destructive" role="alert">{{ saveError }}</p>
            </section>
            <SettingsHomepageDevTools v-if="useWebApi" />
          </TabsContent>
          <TabsContent value="performance" class="mt-0 min-w-0"><slot name="performance" /></TabsContent>
          <TabsContent value="logs" force-mount v-show="tab === 'logs'" class="mt-0">
            <SettingsLoggingSection v-if="ready" :auto-save-ready="ready" />
            <p v-else-if="loading" class="text-sm text-muted-foreground" role="status">{{ t("debug.loading") }}</p>
          </TabsContent>
          <div v-if="loadError && tab !== 'performance'" class="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-destructive/30 p-3">
            <p class="text-sm text-destructive" role="alert">{{ loadError }}</p>
            <Button variant="outline" size="sm" @click="loadSettings">{{ t("debug.retry") }}</Button>
          </div>
        </div>
      </Tabs>
    </DialogContent>
  </Dialog>
</template>

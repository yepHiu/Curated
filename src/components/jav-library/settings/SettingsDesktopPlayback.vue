<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { MonitorPlay } from "lucide-vue-next"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { Alert, AlertTitle } from "@/components/ui/alert"
import type { DesktopPlaybackCapabilities } from "../../../../electron/playback-contract"
const bridge = window.javLibrary?.playback
const capability = ref<DesktopPlaybackCapabilities>()
const saving = ref(false)
const error = ref("")
const { t } = useI18n()
onMounted(async () => { if (bridge) capability.value = await bridge.capabilities().catch(() => undefined) })
async function change(preferNative: boolean) {
  if (!bridge || !capability.value || saving.value) return
  saving.value = true; error.value = ""
  try { capability.value = await bridge.setPreferences({ ...capability.value.preferences, preferNative }) }
  catch { error.value = t("desktopPlayback.saveFailed") }
  finally { saving.value = false }
}
</script>
<template>
  <Card v-if="capability">
    <CardHeader class="grid grid-cols-[auto_1fr] items-center gap-x-3"><MonitorPlay class="text-primary" /><CardTitle>{{ t('desktopPlayback.title') }}</CardTitle></CardHeader>
    <CardContent>
      <div class="flex items-center justify-between gap-4 rounded-xl border border-border/60 bg-muted/20 p-4">
        <div class="flex flex-col gap-2"><label for="prefer-desktop-native" class="text-sm font-medium">{{ t('desktopPlayback.preferNative') }}</label><p class="text-xs leading-relaxed text-muted-foreground">{{ capability.available ? t('desktopPlayback.localPreference') : t('desktopPlayback.unavailable') }}</p></div>
        <Switch id="prefer-desktop-native" :model-value="capability.preferences.preferNative" :disabled="!capability.available || saving" @update:model-value="change" />
      </div>
      <Alert v-if="error" variant="destructive" class="mt-4"><AlertTitle>{{ error }}</AlertTitle></Alert>
    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Captions, Eye, EyeOff } from "lucide-vue-next"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import SettingsHint from "./SettingsHint.vue"
import SubtitleModelSelect from "../SubtitleModelSelect.vue"
import { useSubtitleService } from "@/services/subtitle-service"
import type { SubtitleSettings } from "@/services/contracts/subtitle-service"
const { t } = useI18n()
const service = useSubtitleService()
const cfg = ref<SubtitleSettings>()
const key = ref("")
const clearKey = ref(false)
const keyVisible = ref(false)
const revealedKey = ref("")
async function toggleKey() {
  if (keyVisible.value) { keyVisible.value = false; revealedKey.value = ""; return }
  if (!key.value && cfg.value?.apiKeyConfigured) {
    busy.value = true; error.value = ""
    try { revealedKey.value = await service.revealKey() }
    catch (e) { error.value = e instanceof Error ? e.message : String(e); return }
    finally { busy.value = false }
  }
  keyVisible.value = true
}
function editKey(value: string | number) { key.value = String(value); revealedKey.value = "" }
const busy = ref(false)
const error = ref("")
const saved = ref(false)
const tested = ref(false)
async function testProvider() {
  busy.value = true; error.value = ""; tested.value = false
  try { await service.testProvider(); tested.value = true }
  catch (e) { error.value = e instanceof Error ? e.message : String(e) }
  finally { busy.value = false }
}
onMounted(async () => { try { cfg.value = await service.settings() } catch (e) { error.value = String(e) } })
async function save() {
  if (!cfg.value) return
  busy.value = true; error.value = ""; saved.value = false
  try { cfg.value = await service.saveSettings({ enabled: cfg.value.enabled, translationModel: cfg.value.translationModel, ...(key.value || clearKey.value ? { apiKey: clearKey.value ? "" : key.value } : {}) }); key.value = ""; revealedKey.value = ""; keyVisible.value = false; clearKey.value = false; saved.value = true }
  catch (e) { error.value = e instanceof Error ? e.message : String(e) }
  finally { busy.value = false }
}
</script>
<template>
  <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm" data-subtitle-settings-card>
    <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
      <span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><Captions class="size-4" /></span>
      <CardTitle class="flex min-w-0 flex-wrap items-center gap-2 text-lg tracking-tight">
        <span>{{ t('subtitles.title') }}</span>
        <SettingsScopeBadge scope="server" />
      </CardTitle>
    </CardHeader>
    <CardContent class="flex min-w-0 flex-col gap-3 pt-0">
      <Alert v-if="error" variant="destructive"><AlertDescription>{{ error }}</AlertDescription></Alert>
      <FieldGroup v-if="cfg" class="gap-3">
        <Field orientation="horizontal" class="items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4" :data-disabled="!cfg.canManage || busy || undefined">
          <SettingsHint :text="t('subtitles.cloudNotice')"><FieldLabel for="subtitle-enabled" class="text-sm font-semibold text-foreground">{{ t('subtitles.enable') }}</FieldLabel></SettingsHint>
          <Switch id="subtitle-enabled" v-model="cfg.enabled" :disabled="!cfg.canManage || busy" />
        </Field>
        <Field class="min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
          <FieldLabel class="min-w-0 flex-1 text-sm font-semibold text-foreground">{{ t('subtitles.recognitionModel') }}</FieldLabel>
          <p class="min-w-0 text-sm text-muted-foreground sm:max-w-[60%] sm:text-right">{{ t('subtitles.providerDescription') }}</p>
        </Field>
        <Field class="min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4" :data-disabled="!cfg.canManage || busy || clearKey || undefined">
          <div class="min-w-0 flex-1">
            <SettingsHint :text="t('subtitles.keyHint')"><FieldLabel for="subtitle-key" class="text-sm font-semibold text-foreground">{{ t('subtitles.apiKey') }}</FieldLabel></SettingsHint>
          </div>
          <div class="flex w-full min-w-0 items-center gap-2 sm:w-80 sm:max-w-[60%] sm:shrink-0">
            <Input id="subtitle-key" :model-value="key || (keyVisible ? revealedKey : '')" :type="keyVisible ? 'text' : 'password'" autocomplete="new-password" :disabled="!cfg.canManage || busy || clearKey" :placeholder="cfg.apiKeyConfigured ? '••••••••••••••••' : t('subtitles.enterKey')" @update:model-value="editKey" />
            <Button type="button" variant="ghost" size="icon" class="shrink-0 rounded-full" :disabled="!cfg.canManage || busy || clearKey" :aria-label="t(keyVisible ? 'subtitles.hideKey' : 'subtitles.showKey')" :aria-pressed="keyVisible" @click="toggleKey"><EyeOff v-if="keyVisible" aria-hidden="true" /><Eye v-else aria-hidden="true" /></Button>
          </div>
        </Field>
        <Field class="min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4" :data-disabled="!cfg.canManage || busy || undefined">
          <div class="min-w-0 flex-1">
            <SettingsHint :text="t('subtitles.modelHint')"><FieldLabel for="subtitle-model" class="text-sm font-semibold text-foreground">{{ t('subtitles.translationModel') }}</FieldLabel></SettingsHint>
          </div>
          <div class="w-full min-w-0 sm:w-80 sm:max-w-[60%] sm:shrink-0">
            <SubtitleModelSelect id="subtitle-model" v-model="cfg.translationModel" :configured-model="cfg.configuredModel" :disabled="!cfg.canManage || busy" />
          </div>
        </Field>
        <Field v-if="cfg.apiKeyConfigured" orientation="horizontal" class="items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4" :data-disabled="!cfg.canManage || busy || undefined">
          <FieldLabel for="subtitle-clear" class="text-sm font-semibold text-foreground">{{ t('subtitles.clearKey') }}</FieldLabel>
          <Switch id="subtitle-clear" v-model="clearKey" :disabled="!cfg.canManage || busy" />
        </Field>
      </FieldGroup>
      <p v-if="cfg && !cfg.canManage" class="text-sm text-muted-foreground">{{ t('subtitles.localOnly') }}</p>
      <div class="flex flex-col gap-3 rounded-lg border border-border/40 bg-background/30 p-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0 flex-1 text-xs text-muted-foreground" role="status">
          <p v-if="tested">{{ t('subtitles.testPassed') }}</p>
          <p v-else-if="saved">{{ t('subtitles.saved') }}</p>
        </div>
        <div class="flex flex-wrap justify-end gap-2">
          <Button variant="outline" size="sm" class="rounded-full" :disabled="!cfg?.canManage || !cfg.apiKeyConfigured || !!key || clearKey || busy" @click="testProvider">{{ t('subtitles.testProvider') }}</Button>
          <Button size="sm" class="rounded-full" :disabled="!cfg?.canManage || busy" @click="save">{{ busy ? t('common.loading') : t('subtitles.save') }}</Button>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

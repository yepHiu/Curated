<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Captions } from "lucide-vue-next"
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
  try { cfg.value = await service.saveSettings({ enabled: cfg.value.enabled, translationModel: cfg.value.translationModel, ...(key.value || clearKey.value ? { apiKey: clearKey.value ? "" : key.value } : {}) }); key.value = ""; clearKey.value = false; saved.value = true }
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
      <section v-if="cfg" class="flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4">
        <FieldGroup class="gap-3">
          <Field orientation="horizontal" class="rounded-lg border border-border/40 bg-background/30 px-3 py-2" :data-disabled="!cfg.canManage || busy || undefined">
            <SettingsHint :text="t('subtitles.cloudNotice')"><FieldLabel for="subtitle-enabled">{{ t('subtitles.enable') }}</FieldLabel></SettingsHint>
            <Switch id="subtitle-enabled" v-model="cfg.enabled" :disabled="!cfg.canManage || busy" />
          </Field>
          <Field :data-disabled="!cfg.canManage || busy || undefined">
            <FieldLabel>{{ t('subtitles.recognitionModel') }}</FieldLabel>
            <p class="break-words text-sm text-muted-foreground">{{ t('subtitles.providerDescription') }}</p>
          </Field>
          <FieldGroup class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <Field :data-disabled="!cfg.canManage || busy || clearKey || undefined">
              <SettingsHint :text="t('subtitles.keyHint')"><FieldLabel for="subtitle-key">{{ t('subtitles.apiKey') }}</FieldLabel></SettingsHint>
              <Input id="subtitle-key" v-model="key" type="password" autocomplete="new-password" :disabled="!cfg.canManage || busy || clearKey" :placeholder="cfg.apiKeyConfigured ? t('subtitles.keyConfigured') : t('subtitles.enterKey')" />
            </Field>
            <Field :data-disabled="!cfg.canManage || busy || undefined">
              <SettingsHint :text="t('subtitles.modelHint')"><FieldLabel for="subtitle-model">{{ t('subtitles.translationModel') }}</FieldLabel></SettingsHint>
              <SubtitleModelSelect id="subtitle-model" v-model="cfg.translationModel" :configured-model="cfg.configuredModel" :disabled="!cfg.canManage || busy" />
            </Field>
          </FieldGroup>
          <Field v-if="cfg.apiKeyConfigured" orientation="horizontal" class="rounded-lg border border-border/40 bg-background/30 px-3 py-2" :data-disabled="!cfg.canManage || busy || undefined">
            <FieldLabel for="subtitle-clear">{{ t('subtitles.clearKey') }}</FieldLabel>
            <Switch id="subtitle-clear" v-model="clearKey" :disabled="!cfg.canManage || busy" />
          </Field>
        </FieldGroup>
      </section>
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

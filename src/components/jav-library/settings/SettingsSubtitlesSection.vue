<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Alert, AlertDescription } from "@/components/ui/alert"
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
  <Card>
    <CardHeader><CardTitle>{{ t('subtitles.title') }}</CardTitle><CardDescription>{{ t('subtitles.providerDescription') }}</CardDescription></CardHeader>
    <CardContent class="flex flex-col gap-4">
      <Alert v-if="error" variant="destructive"><AlertDescription>{{ error }}</AlertDescription></Alert>
      <FieldGroup v-if="cfg" :disabled="!cfg.canManage || busy">
        <Field orientation="horizontal"><FieldLabel for="subtitle-enabled">{{ t('subtitles.enable') }}</FieldLabel><Switch id="subtitle-enabled" v-model="cfg.enabled" :disabled="!cfg.canManage || busy" /></Field>
        <Field><FieldLabel for="subtitle-key">{{ t('subtitles.apiKey') }}</FieldLabel><Input id="subtitle-key" v-model="key" type="password" autocomplete="new-password" :disabled="!cfg.canManage || busy || clearKey" :placeholder="cfg.apiKeyConfigured ? t('subtitles.keyConfigured') : t('subtitles.enterKey')" /><FieldDescription>{{ t('subtitles.keyHint') }}</FieldDescription></Field>
        <Field v-if="cfg.apiKeyConfigured" orientation="horizontal"><FieldLabel for="subtitle-clear">{{ t('subtitles.clearKey') }}</FieldLabel><Switch id="subtitle-clear" v-model="clearKey" :disabled="!cfg.canManage || busy" /></Field>
        <Field><FieldLabel>{{ t('subtitles.translationModel') }}</FieldLabel><SubtitleModelSelect v-model="cfg.translationModel" :configured-model="cfg.configuredModel" :disabled="!cfg.canManage || busy" /><FieldDescription>{{ t('subtitles.modelHint') }}</FieldDescription></Field>
      </FieldGroup>
      <p class="text-sm text-muted-foreground">{{ t('subtitles.storageHint') }}</p>
      <p v-if="tested" role="status" class="text-sm text-muted-foreground">{{ t('subtitles.testPassed') }}</p>
      <p v-if="cfg && !cfg.canManage" class="text-sm text-muted-foreground">{{ t('subtitles.localOnly') }}</p>
    </CardContent>
    <CardFooter class="flex-wrap gap-3"><Button :disabled="!cfg?.canManage || busy" @click="save">{{ busy ? t('common.loading') : t('subtitles.save') }}</Button><Button variant="outline" :disabled="!cfg?.canManage || !cfg.apiKeyConfigured || !!key || clearKey || busy" @click="testProvider">{{ t('subtitles.testProvider') }}</Button><span v-if="saved" role="status" class="text-sm text-muted-foreground">{{ t('subtitles.saved') }}</span></CardFooter>
  </Card>
</template>

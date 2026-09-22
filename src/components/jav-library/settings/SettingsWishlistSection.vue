<script setup lang="ts">
import { ref } from "vue"
import { Plug } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { useLibraryService } from "@/services/library-service"
import { Switch } from "@/components/ui/switch"
import { Field, FieldContent, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card"

const { t } = useI18n()
const service = useLibraryService()
const busy = ref(false)
const error = ref(false)

/** 保存成功后更新开关；失败保留原状态并允许重试。 */
async function save(enabled: boolean) {
  if (busy.value || !service.wishlist.integrationsAvailable) return
  busy.value = true
  error.value = false
  try {
    await service.setBrowserPluginEnabled(enabled)
  } catch {
    error.value = true
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
    <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
      <span class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true">
        <Plug class="size-[1.15rem]" />
      </span>
      <CardTitle class="min-w-0 text-lg tracking-tight">{{ t('wishlist.connect') }}</CardTitle>
      <CardDescription class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm">{{ t('wishlist.pluginHint') }}</CardDescription>
    </CardHeader>
    <CardContent class="flex flex-col gap-3 pt-0">
      <FieldGroup class="rounded-lg border border-border/50 bg-muted/5 p-4">
        <Field orientation="horizontal" class="has-[>[data-slot=field-content]]:items-center">
          <FieldContent>
            <FieldLabel for="browser-plugin-enabled">{{ t('wishlist.pluginEnabled') }}</FieldLabel>
            <FieldDescription id="browser-plugin-description">{{ t('wishlist.pluginEnabledHint') }}</FieldDescription>
          </FieldContent>
          <Switch
            id="browser-plugin-enabled"
            :model-value="service.browserPluginEnabled.value"
            :disabled="busy || !service.wishlist.integrationsAvailable"
            aria-describedby="browser-plugin-description"
            @update:model-value="save"
          />
        </Field>
      </FieldGroup>
      <p v-if="!service.wishlist.integrationsAvailable" class="text-sm text-muted-foreground">{{ t('wishlist.mockHint') }}</p>
      <p v-if="error" role="alert" class="text-sm text-destructive">{{ t('wishlist.pluginSaveError') }}</p>
    </CardContent>
  </Card>
</template>

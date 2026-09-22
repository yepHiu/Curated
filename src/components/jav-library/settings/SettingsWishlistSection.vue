<script setup lang="ts">
import { ref } from "vue"
import { Plug } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { useLibraryService } from "@/services/library-service"
import { Switch } from "@/components/ui/switch"
import { Field, FieldContent, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"

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
    </CardHeader>
    <CardContent class="flex flex-col gap-3 pt-0">
      <p
        v-if="!service.wishlist.integrationsAvailable"
        class="rounded-xl border border-border/60 bg-muted/10 px-3 py-2 text-xs leading-relaxed text-muted-foreground sm:text-sm"
      >
        {{ t('wishlist.mockHint') }}
      </p>
      <FieldGroup class="rounded-lg border border-border/50 bg-muted/5 p-4 shadow-sm shadow-black/5">
        <Field
          orientation="horizontal"
          class="justify-between has-[>[data-slot=field-content]]:items-center"
          :aria-busy="busy"
        >
          <FieldContent class="min-w-0 gap-3">
            <FieldLabel for="browser-plugin-enabled" class="text-sm font-semibold text-foreground">
              {{ t('wishlist.pluginEnabled') }}
            </FieldLabel>
            <FieldDescription
              id="browser-plugin-description"
              class="text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm group-has-[[data-orientation=horizontal]]/field:text-pretty nth-last-2:mt-0"
            >
              {{ t('wishlist.pluginEnabledHint') }}
            </FieldDescription>
            <p v-if="busy" role="status" class="text-xs text-muted-foreground motion-safe:animate-pulse">
              {{ t('common.saving') }}
            </p>
          </FieldContent>
          <Switch
            id="browser-plugin-enabled"
            class="shrink-0 motion-safe:transition-colors motion-safe:duration-200"
            :model-value="service.browserPluginEnabled.value"
            :disabled="busy || !service.wishlist.integrationsAvailable"
            aria-describedby="browser-plugin-description"
            @update:model-value="save"
          />
        </Field>
      </FieldGroup>
      <p v-if="error" role="alert" class="text-sm text-destructive">{{ t('wishlist.pluginSaveError') }}</p>
    </CardContent>
  </Card>
</template>

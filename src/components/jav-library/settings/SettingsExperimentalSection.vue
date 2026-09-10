<script setup lang="ts">
import { defineAsyncComponent } from "vue"
import { useI18n } from "vue-i18n"
import { RouterLink } from "vue-router"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
defineProps<{ useWebApi: boolean }>()
const { t } = useI18n()
/** 按需加载漫画 Beta 设置，保持常规设置页包体独立。 */
const SettingsComicLibrarySection = defineAsyncComponent(() => import("./SettingsComicLibrarySection.vue"))
/** 按需加载写真 Beta 设置。 */
const SettingsPhotoLibrarySection = defineAsyncComponent(() => import("./SettingsPhotoLibrarySection.vue"))
</script>

<template>
  <div class="flex flex-col gap-3">
    <Card>
      <CardHeader><CardTitle>{{ t('settings.experimentalTitle') }}</CardTitle></CardHeader>
      <CardContent class="flex flex-col items-start gap-3">
        <p class="text-sm text-muted-foreground">{{ t('aiSettings.moved') }}</p>
        <Button as-child variant="outline"><RouterLink :to="{ name: 'settings', query: { section: 'ai' } }">{{ t('aiSettings.title') }}</RouterLink></Button>
      </CardContent>
    </Card>
    <SettingsComicLibrarySection />
    <SettingsPhotoLibrarySection />
  </div>
</template>

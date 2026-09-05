<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { RouterLink } from "vue-router"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
defineProps<{ useWebApi: boolean }>()
const { t } = useI18n()
</script>

<template>
  <Card>
    <CardHeader><CardTitle>{{ t('settings.experimentalTitle') }}</CardTitle></CardHeader>
    <CardContent class="flex flex-col items-start gap-3">
      <p class="text-sm text-muted-foreground">{{ t('aiSettings.moved') }}</p>
      <Button as-child variant="outline"><RouterLink :to="{ name: 'settings', query: { section: 'ai' } }">{{ t('aiSettings.title') }}</RouterLink></Button>
    </CardContent>
  </Card>
</template>

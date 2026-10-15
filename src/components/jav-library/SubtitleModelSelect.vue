<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Select, SelectTrigger, SelectValue, SelectContent, SelectGroup, SelectItem } from "@/components/ui/select"
import type { SubtitleModel } from "@/services/contracts/subtitle-service"
defineProps<{ modelValue: SubtitleModel; configuredModel?: string; disabled?: boolean }>()
const emit = defineEmits<{ "update:modelValue": [value: SubtitleModel] }>()
const { t } = useI18n()
</script>
<template>
  <Select :model-value="modelValue" :disabled="disabled" @update:model-value="emit('update:modelValue', $event as SubtitleModel)">
    <SelectTrigger :aria-label="t('subtitles.translationModel')"><SelectValue /></SelectTrigger>
    <SelectContent><SelectGroup>
      <SelectItem value="configured-ai">{{ t('subtitles.configuredModel') }}{{ configuredModel ? ` · ${configuredModel}` : '' }}</SelectItem>
      <SelectItem value="qwen-flash">Qwen Flash · {{ t('subtitles.economic') }}</SelectItem>
      <SelectItem value="qwen-plus">Qwen Plus · {{ t('subtitles.balanced') }}</SelectItem>
    </SelectGroup></SelectContent>
  </Select>
</template>

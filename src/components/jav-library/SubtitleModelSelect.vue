<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Select, SelectTrigger, SelectValue, SelectContent, SelectGroup, SelectItem } from "@/components/ui/select"
import type { SubtitleModel } from "@/services/contracts/subtitle-service"
defineProps<{ id?: string; modelValue: SubtitleModel; configuredModel?: string; disabled?: boolean; pill?: boolean }>()
const emit = defineEmits<{ "update:modelValue": [value: SubtitleModel] }>()
const { t } = useI18n()
</script>
<template>
  <Select :model-value="modelValue" :disabled="disabled" @update:model-value="emit('update:modelValue', $event as SubtitleModel)">
    <SelectTrigger :id="id" class="w-full min-w-0" :class="{ 'min-h-11 rounded-full px-4 sm:min-h-9': pill }" :aria-label="t('subtitles.translationModel')"><SelectValue /></SelectTrigger>
    <SelectContent><SelectGroup>
      <SelectItem value="configured-ai">{{ t('subtitles.configuredModel') }}{{ configuredModel ? ` · ${configuredModel}` : '' }}</SelectItem>
      <SelectItem value="qwen-flash">{{ t('subtitles.flashOption') }}</SelectItem>
      <SelectItem value="qwen-plus">{{ t('subtitles.plusOption') }}</SelectItem>
    </SelectGroup></SelectContent>
  </Select>
</template>

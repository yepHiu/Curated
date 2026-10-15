<script setup lang="ts">
import { Captions } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import type { SubtitleDisplay, SubtitleDisplayMode } from "../../../electron/subtitle-format"
const props = defineProps<{ modelValue: SubtitleDisplay; available: string[]; disabled?: boolean; error?: string }>()
const emit = defineEmits<{ "update:modelValue": [value: SubtitleDisplay]; "update:open": [value: boolean] }>()
const { t } = useI18n()
function update(patch: Partial<SubtitleDisplay>) { emit('update:modelValue', { ...props.modelValue, ...patch }) }
function numeric(field: "size" | "position" | "offset", value: string | number) {
  const n = Number(value), bounds = { size: [16, 64], position: [0, 40], offset: [-60, 60] }[field]!
  if (Number.isFinite(n)) update({ [field]: Math.max(bounds[0]!, Math.min(bounds[1]!, n)) })
}
</script>
<template>
  <Popover @update:open="emit('update:open', $event)">
    <PopoverTrigger as-child><Button size="icon" variant="secondary" class="size-9 shrink-0 rounded-full" :disabled="disabled" :aria-label="t('subtitles.displaySettings')"><Captions /></Button></PopoverTrigger>
    <PopoverContent align="end" class="max-h-[75dvh] w-72 overflow-y-auto" @click.stop>
      <FieldGroup>
        <Field><FieldLabel>{{ t('subtitles.displaySettings') }}</FieldLabel>
          <Select :model-value="modelValue.mode" @update:model-value="update({ mode: $event as SubtitleDisplayMode })"><SelectTrigger :aria-label="t('subtitles.displaySettings')"><SelectValue /></SelectTrigger><SelectContent><SelectGroup>
            <SelectItem value="off">{{ t('subtitles.off') }}</SelectItem>
            <SelectItem value="zh-CN" :disabled="!available.includes('zh-CN')">{{ t('subtitles.chinese') }}</SelectItem>
            <SelectItem value="ja" :disabled="!available.includes('ja')">{{ t('subtitles.japanese') }}</SelectItem>
            <SelectItem value="bilingual" :disabled="!available.includes('zh-CN') || !available.includes('ja')">{{ t('subtitles.bilingual') }}</SelectItem>
          </SelectGroup></SelectContent></Select>
          <FieldDescription v-if="error">{{ t('subtitles.loadFailed') }}</FieldDescription>
          <FieldDescription v-else-if="!available.length">{{ t('subtitles.noSubtitles') }}</FieldDescription>
        </Field>
        <Field><FieldLabel>{{ t('subtitles.fontSize') }}</FieldLabel><Input type="number" :model-value="modelValue.size" min="16" max="64" :aria-label="t('subtitles.fontSize')" @update:model-value="numeric('size', $event)" /></Field>
        <Field orientation="horizontal"><FieldLabel>{{ t('subtitles.background') }}</FieldLabel><Switch :model-value="modelValue.background" :aria-label="t('subtitles.background')" @update:model-value="update({ background: $event })" /></Field>
        <Field><FieldLabel>{{ t('subtitles.position') }}</FieldLabel><Input type="number" :model-value="modelValue.position" min="0" max="40" :aria-label="t('subtitles.position')" @update:model-value="numeric('position', $event)" /></Field>
        <Field><FieldLabel>{{ t('subtitles.offset') }}</FieldLabel><Input type="number" :model-value="modelValue.offset" min="-60" max="60" step="0.1" :aria-label="t('subtitles.offset')" @update:model-value="numeric('offset', $event)" /><FieldDescription>{{ t('subtitles.offsetHint') }}</FieldDescription></Field>
      </FieldGroup>
    </PopoverContent>
  </Popover>
</template>

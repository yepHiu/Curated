<script setup lang="ts">
import { computed, inject, ref, useId, watch, type Ref } from "vue"
import { Captions, X } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import Button from "@/components/jav-library/PlayerButton.vue"
import { Button as CloseButton } from "@/components/ui/button"
import SettingsHint from "@/components/jav-library/settings/SettingsHint.vue"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field"
import { Separator } from "@/components/ui/separator"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import type { SubtitleDisplay, SubtitleDisplayMode } from "../../../electron/subtitle-format"

const props = defineProps<{ modelValue: SubtitleDisplay; available: string[]; disabled?: boolean; error?: string; native?: boolean }>()
const emit = defineEmits<{ "update:modelValue": [value: SubtitleDisplay]; "update:open": [value: boolean] }>()
const { t } = useI18n()
const id = useId()
const portalTo = inject<Ref<HTMLElement | null | undefined> | undefined>("playerTooltipPortal", undefined)
const open = ref(false)
const modeLabels: Record<SubtitleDisplayMode, string> = {
  off: "subtitles.off", "zh-CN": "subtitles.chinese", ja: "subtitles.japanese", bilingual: "subtitles.bilingual",
}
const enabled = computed(() => props.modelValue.mode === "bilingual"
  ? props.available.includes("zh-CN") && props.available.includes("ja")
  : props.available.includes(props.modelValue.mode))
type NumericField = "size" | "position" | "offset"
const draft = ref({ size: String(props.modelValue.size), position: String(props.modelValue.position), offset: String(props.modelValue.offset) })
for (const field of ["size", "position", "offset"] as const) {
  watch(() => props.modelValue[field], value => { draft.value[field] = String(value) })
}
watch(open, value => emit("update:open", value))
watch(() => props.disabled, value => { if (value) open.value = false })

function update(patch: Partial<SubtitleDisplay>) {
  emit("update:modelValue", { ...props.modelValue, ...patch })
}

function commitNumeric(field: NumericField) {
  const value = String(draft.value[field]).trim()
  const n = Number(value)
  const bounds = { size: [16, 64], position: [0, 40], offset: [-60, 60] }[field]!
  const next = value && Number.isFinite(n) ? Math.max(bounds[0]!, Math.min(bounds[1]!, n)) : props.modelValue[field]
  draft.value[field] = String(next)
  if (next !== props.modelValue[field]) update({ [field]: next })
}
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button size="icon" :variant="native ? 'ghost' : 'secondary'" class="size-9 shrink-0 rounded-full" :disabled="disabled" :aria-pressed="enabled" :aria-label="t('subtitles.displaySettings')">
        <Captions aria-hidden="true" />
      </Button>
    </PopoverTrigger>
    <PopoverContent
      :portal-to="portalTo ?? undefined"
      side="top"
      align="end"
      :side-offset="8"
      :collision-padding="12"
      class="max-h-[min(75dvh,var(--reka-popover-content-available-height))] w-80 max-w-[calc(100vw-1.5rem)] overflow-y-auto rounded-2xl border-border/60 bg-popover p-4 text-popover-foreground shadow-lg"
      :aria-labelledby="`${id}-title`"
      @click.stop
    >
      <div class="flex items-center justify-between gap-3">
        <h2 :id="`${id}-title`" class="text-sm font-semibold">{{ t('subtitles.displaySettings') }}</h2>
        <CloseButton variant="ghost" size="icon-sm" class="shrink-0 rounded-full" :aria-label="t('common.close')" @click="open = false"><X aria-hidden="true" /></CloseButton>
      </div>
      <Separator class="my-3" />
      <FieldGroup class="gap-4">
        <Field class="gap-2" :data-invalid="!!error || undefined">
          <FieldLabel :for="`${id}-mode`" class="sr-only">{{ t('subtitles.displaySettings') }}</FieldLabel>
          <Select :model-value="modelValue.mode" @update:model-value="update({ mode: $event as SubtitleDisplayMode })">
            <SelectTrigger :id="`${id}-mode`" class="min-h-11 w-full border-border/60 bg-muted/40 text-foreground sm:min-h-10" :aria-describedby="error || !available.length ? `${id}-status` : undefined">
              <SelectValue>{{ t(modeLabels[modelValue.mode]) }}</SelectValue>
            </SelectTrigger>
            <SelectContent :portal-to="portalTo ?? undefined" :side-offset="8" :collision-padding="12" class="rounded-xl">
              <SelectGroup>
                <SelectItem value="off">{{ t('subtitles.off') }}</SelectItem>
                <SelectItem value="zh-CN" :disabled="!available.includes('zh-CN')">{{ t('subtitles.chinese') }}</SelectItem>
                <SelectItem value="ja" :disabled="!available.includes('ja')">{{ t('subtitles.japanese') }}</SelectItem>
                <SelectItem value="bilingual" :disabled="!available.includes('zh-CN') || !available.includes('ja')">{{ t('subtitles.bilingual') }}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
          <FieldDescription v-if="error" :id="`${id}-status`" role="alert" class="text-destructive">{{ t('subtitles.loadFailed') }}</FieldDescription>
          <FieldDescription v-else-if="!available.length" :id="`${id}-status`" role="status">{{ t('subtitles.noSubtitles') }}</FieldDescription>
        </Field>
        <Field orientation="horizontal" class="justify-between gap-3">
          <FieldLabel :for="`${id}-size`" class="min-w-0 text-sm">{{ t('subtitles.fontSize') }}</FieldLabel>
          <Input :id="`${id}-size`" type="number" class="min-h-11 w-24 shrink-0 tabular-nums sm:min-h-10" v-model="draft.size" min="16" max="64" @blur="commitNumeric('size')" @keydown.enter.prevent="commitNumeric('size')" />
        </Field>
        <Field orientation="horizontal" class="min-h-11 justify-between gap-3 sm:min-h-10">
          <FieldLabel :for="`${id}-background`" class="min-w-0 text-sm">{{ t('subtitles.background') }}</FieldLabel>
          <Switch :id="`${id}-background`" :aria-label="t('subtitles.background')" :model-value="modelValue.background" @update:model-value="update({ background: $event })" />
        </Field>
        <Field orientation="horizontal" class="justify-between gap-3">
          <FieldLabel :for="`${id}-position`" class="min-w-0 text-sm">{{ t('subtitles.position') }}</FieldLabel>
          <Input :id="`${id}-position`" type="number" class="min-h-11 w-24 shrink-0 tabular-nums sm:min-h-10" v-model="draft.position" min="0" max="40" @blur="commitNumeric('position')" @keydown.enter.prevent="commitNumeric('position')" />
        </Field>
        <Field orientation="horizontal" class="justify-between gap-3">
          <SettingsHint :text="t('subtitles.offsetHint')" :portal-to="portalTo ?? undefined">
            <FieldLabel :for="`${id}-offset`" class="min-w-0 flex-1 text-sm">{{ t('subtitles.offset') }}</FieldLabel>
          </SettingsHint>
          <Input :id="`${id}-offset`" type="number" class="min-h-11 w-24 shrink-0 tabular-nums sm:min-h-10" v-model="draft.offset" min="-60" max="60" step="0.1" :aria-describedby="`${id}-offset-hint`" @blur="commitNumeric('offset')" @keydown.enter.prevent="commitNumeric('offset')" />
          <FieldDescription :id="`${id}-offset-hint`" class="sr-only">{{ t('subtitles.offsetHint') }}</FieldDescription>
        </Field>
      </FieldGroup>
    </PopoverContent>
  </Popover>
</template>

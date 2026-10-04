<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { AcceptableValue } from "reka-ui"
import type { MovieFile } from "@/domain/movie/types"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

const props = defineProps<{ files: MovieFile[]; modelValue?: string; disabled?: boolean; portalTo?: HTMLElement }>()
const emit = defineEmits<{ 'update:modelValue': [fileId: string] }>()
const { t } = useI18n()

/** 只接受实际存在的分片，Reka 的空值不能发起播放请求。 */
function selectPart(value: AcceptableValue) {
 if (typeof value === 'string' && props.files.some(/* 只接受列表内实际文件标识。 */ file => file.id === value)) emit('update:modelValue', value)
}
</script>

<template>
 <Select :model-value="modelValue" :disabled="disabled" @update:model-value="selectPart">
  <SelectTrigger class="w-full min-w-0 min-h-11 sm:min-h-9 sm:w-64" :aria-label="t('player.selectPart')" data-movie-part-select>
   <SelectValue class="min-w-0 truncate" :placeholder="t('player.selectPart')" />
  </SelectTrigger>
  <SelectContent :portal-to="portalTo" class="max-w-[calc(100vw-2rem)]">
   <SelectGroup>
    <SelectItem v-for="(file, index) in files" :key="file.id" :value="file.id" :title="file.fileName">
     <span class="block min-w-0 truncate">{{ t('player.partLabel', { number: file.partIndex || index + 1 }) }} · {{ file.fileName }}</span>
    </SelectItem>
   </SelectGroup>
  </SelectContent>
 </Select>
</template>

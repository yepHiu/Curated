<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { computed } from "vue"
import { cn } from "@/lib/utils"
import type { AcceptableValue } from "reka-ui"
import type { MovieFile } from "@/domain/movie/types"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

const props = defineProps<{ files: Pick<MovieFile, 'id' | 'partIndex' | 'fileName'>[]; modelValue?: string; disabled?: boolean; portalTo?: HTMLElement; wrapFileName?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [fileId: string] }>()
const { t } = useI18n()
const selectedFile = computed(() => props.files.find(file => file.id === props.modelValue))

function fileLabel(file: Pick<MovieFile, 'id' | 'partIndex' | 'fileName'>, index: number) {
 return `${t('player.partLabel', { number: file.partIndex || index + 1 })} · ${file.fileName}`
}

/** 只接受实际存在的分片，Reka 的空值不能发起播放请求。 */
function selectPart(value: AcceptableValue) {
 if (typeof value === 'string' && props.files.some(/* 只接受列表内实际文件标识。 */ file => file.id === value)) emit('update:modelValue', value)
}
</script>

<template>
 <Select :model-value="modelValue" :disabled="disabled" @update:model-value="selectPart">
  <!-- 详情页与播放器共用胶囊形选择条，窄屏保留 44px 触控高度。 -->
  <SelectTrigger :class="cn('w-full min-w-0 min-h-11 rounded-full px-4 sm:min-h-9', wrapFileName ? 'data-[size=default]:h-auto text-left whitespace-normal *:data-[slot=select-value]:line-clamp-none' : 'sm:w-64')" :title="selectedFile?.fileName" :aria-label="t('player.selectPart')" data-movie-part-select>
   <SelectValue class="min-w-0 flex-1" :placeholder="t('player.selectPart')">
    <span v-if="selectedFile" :class="cn('block min-w-0', wrapFileName ? 'whitespace-normal break-all' : 'truncate')">{{ fileLabel(selectedFile, files.indexOf(selectedFile)) }}</span>
   </SelectValue>
  </SelectTrigger>
  <SelectContent :portal-to="portalTo" class="max-w-[calc(100vw-2rem)]">
   <SelectGroup>
    <SelectItem v-for="(file, index) in files" :key="file.id" :value="file.id" :title="file.fileName">
     <span class="block min-w-0 whitespace-normal break-all">{{ fileLabel(file, index) }}</span>
    </SelectItem>
   </SelectGroup>
  </SelectContent>
 </Select>
</template>

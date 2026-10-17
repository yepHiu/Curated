<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"

const props = defineProps<{ offset: number; total: number; disabled?: boolean }>()
const emit = defineEmits<{ change: [offset: number] }>()
const { t } = useI18n()
const pages = computed(() => Math.max(1, Math.ceil(props.total / 25)))
</script>

<template>
  <nav v-if="total > 25 || offset > 0" :aria-label="t('tasks.items')" class="flex flex-wrap items-center justify-between gap-2" data-items-pagination>
    <Button variant="outline" size="sm" class="min-h-11 rounded-full sm:min-h-9" :disabled="disabled || offset === 0" @click="emit('change', Math.max(0, offset - 25))">{{ t('bookAI.previous') }}</Button>
    <span class="text-xs tabular-nums text-muted-foreground" role="status">{{ offset / 25 + 1 }} / {{ pages }}</span>
    <Button variant="outline" size="sm" class="min-h-11 rounded-full sm:min-h-9" :disabled="disabled || offset + 25 >= total" @click="emit('change', offset + 25)">{{ t('bookAI.next') }}</Button>
  </nav>
</template>

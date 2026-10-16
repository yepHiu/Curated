<script setup lang="ts">
import { defineAsyncComponent, nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"

const props = defineProps<{ taskKey?: string; title?: string; description?: string }>()
const open = defineModel<boolean>("open", { default: false })
const emit = defineEmits<{ closeAutoFocus: [event: Event] }>()
const { t } = useI18n()
const scrollEl = ref<HTMLElement>()
const TaskDetailContent = defineAsyncComponent(() => import("./TaskDetailContent.vue"))
watch(() => props.taskKey, async () => {
  if (!open.value) return
  await nextTick()
  if (scrollEl.value) {
    scrollEl.value.scrollTop = 0
    scrollEl.value.focus({ preventScroll: true })
  }
})
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent
      class="flex max-h-[90dvh] flex-col gap-0 overflow-hidden p-0"
      :class="taskKey ? 'sm:max-w-5xl' : 'sm:max-w-xl'"
      @close-auto-focus="emit('closeAutoFocus', $event)"
    >
      <DialogHeader class="shrink-0 border-b border-border px-5 py-4 pr-12 sm:px-6 sm:pr-12">
        <DialogTitle>{{ title || t('tasks.detailTitle') }}</DialogTitle>
        <DialogDescription :class="{ 'sr-only': !description }">{{ description || t('tasks.detailTitle') }}</DialogDescription>
        <slot name="actions" />
      </DialogHeader>
      <div ref="scrollEl" class="min-h-0 overflow-y-auto overscroll-contain p-4 outline-none sm:p-6" tabindex="-1" data-task-detail-scroll>
        <template v-if="open">
          <TaskDetailContent v-if="taskKey" :key="taskKey" :task-key="taskKey" />
          <slot v-else />
        </template>
      </div>
    </DialogContent>
  </Dialog>
</template>

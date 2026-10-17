<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { ChevronRight } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"

defineProps<{ title: string; status: string; warning?: boolean }>()
const { t } = useI18n()
const open = ref(false)
</script>

<template>
  <Dialog v-model:open="open">
    <DialogTrigger as-child>
      <Button variant="ghost" class="h-auto min-h-11 min-w-0 flex-1 justify-start gap-2 rounded-lg px-2 py-2" data-processing-item>
        <span class="min-w-0 flex-1 truncate text-left">{{ title }}</span>
        <Badge :variant="warning ? 'warning' : 'secondary'">{{ status }}</Badge>
        <ChevronRight aria-hidden="true" />
      </Button>
    </DialogTrigger>
    <DialogContent class="flex max-h-[85dvh] flex-col overflow-hidden sm:max-w-2xl">
      <DialogHeader class="shrink-0 pr-6">
        <DialogTitle>{{ t('tasks.items') }}</DialogTitle>
        <DialogDescription>{{ status }}</DialogDescription>
      </DialogHeader>
      <div class="flex min-h-0 flex-col gap-4 overflow-y-auto overscroll-contain break-words [overflow-wrap:anywhere]" data-processing-item-detail>
        <p class="whitespace-pre-wrap text-sm font-medium">{{ title }}</p>
        <slot />
      </div>
    </DialogContent>
  </Dialog>
</template>

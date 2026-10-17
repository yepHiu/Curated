<script setup lang="ts">
import { ref } from "vue"
import { ChevronRight } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from "@/components/ui/collapsible"

defineProps<{ title: string; status: string; warning?: boolean }>()
const open = ref(false)
</script>

<template>
  <Collapsible v-model:open="open" class="min-w-0 flex-1">
    <CollapsibleTrigger as-child>
      <Button variant="ghost" class="h-auto min-h-11 w-full min-w-0 justify-start gap-2 rounded-lg px-2 py-2" data-processing-item>
        <span class="min-w-0 flex-1 truncate text-left">{{ title }}</span>
        <Badge :variant="warning ? 'warning' : 'secondary'">{{ status }}</Badge>
        <ChevronRight aria-hidden="true" class="transition-transform motion-reduce:transition-none" :class="{ 'rotate-90': open }" />
      </Button>
    </CollapsibleTrigger>
    <CollapsibleContent>
      <div class="flex min-w-0 flex-col gap-3 px-2 pt-2 pb-4 break-words [overflow-wrap:anywhere]" data-processing-item-detail>
        <p class="whitespace-pre-wrap text-sm font-medium">{{ title }}</p>
        <slot />
      </div>
    </CollapsibleContent>
  </Collapsible>
</template>

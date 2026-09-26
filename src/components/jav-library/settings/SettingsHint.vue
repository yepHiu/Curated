<script setup lang="ts">
import { ref } from "vue"
import { TooltipContent, TooltipPortal, TooltipProvider, TooltipRoot, TooltipTrigger } from "reka-ui"

defineProps<{ text: string }>()
const open = ref(false)

function showFromKeyboard(event: KeyboardEvent) {
  if (event.target !== event.currentTarget || !["Enter", " "].includes(event.key)) return
  if ((event.currentTarget as HTMLElement).matches("button, a, input, select, textarea")) return
  event.preventDefault()
  open.value = !open.value
}
</script>

<template>
  <TooltipProvider :delay-duration="350">
    <TooltipRoot v-model:open="open" disable-closing-trigger>
      <TooltipTrigger
        as-child
        tabindex="0"
        class="cursor-default rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        data-settings-hint-trigger
        @click="open = !open"
        @keydown="showFromKeyboard"
      >
        <slot />
      </TooltipTrigger>
      <TooltipPortal>
        <TooltipContent
          side="bottom"
          align="start"
          :align-flip="false"
          :side-offset="6"
          :collision-padding="12"
          class="z-50 max-w-[min(22rem,calc(100vw-2rem))] whitespace-pre-line rounded-xl border border-border/60 bg-popover px-3 py-2 text-xs font-normal leading-relaxed text-pretty text-popover-foreground shadow-lg"
        >
          {{ text }}
          <slot name="extra" />
        </TooltipContent>
      </TooltipPortal>
    </TooltipRoot>
  </TooltipProvider>
</template>

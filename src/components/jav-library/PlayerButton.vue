<script setup lang="ts">
import { inject, useAttrs, type Ref } from "vue"
import { Primitive, TooltipContent, TooltipPortal, TooltipProvider, TooltipRoot, TooltipTrigger, useForwardExpose } from "reka-ui"
import { Button } from "@/components/ui/button"

defineOptions({ inheritAttrs: false })
const props = defineProps<{ tooltip?: string; unstyled?: boolean; ariaLabel?: string; title?: string }>()
const attrs = useAttrs()
const portalTo = inject<Ref<HTMLElement | null | undefined> | undefined>("playerTooltipPortal", undefined)
function tooltipLabel() { return props.tooltip || String(props.ariaLabel || props.title || "") }
const { forwardRef } = useForwardExpose()
</script>

<template>
  <TooltipProvider :delay-duration="350">
    <TooltipRoot :disabled="!tooltipLabel()" :disable-hoverable-content="true">
      <TooltipTrigger as-child>
        <component :is="unstyled ? Primitive : Button" :ref="forwardRef" :as="unstyled ? 'button' : undefined" v-bind="attrs" :aria-label="props.ariaLabel" :title="undefined"><slot /></component>
      </TooltipTrigger>
      <TooltipPortal :to="portalTo ?? undefined">
        <TooltipContent :aria-label="tooltipLabel()" side="top" :side-offset="8" :collision-padding="12" class="pointer-events-none z-[120] max-w-[min(20rem,calc(100vw-1.5rem))] rounded-lg border border-border/60 bg-popover px-3 py-1.5 text-xs text-popover-foreground shadow-lg">
          {{ tooltipLabel() }}
        </TooltipContent>
      </TooltipPortal>
    </TooltipRoot>
  </TooltipProvider>
</template>

<script setup lang="ts">
import { ChevronRight, Loader2, Sparkles } from "lucide-vue-next"
import { computed } from "vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  getViewportWidth,
  type DesignLabPlaygroundState,
} from "@/lib/design-lab/playground-definitions"

const props = defineProps<{
  state: DesignLabPlaygroundState
}>()

function radiusClass(radius: "md" | "xl" | "full"): string {
  switch (radius) {
    case "xl":
      return "rounded-[18px]"
    case "full":
      return "rounded-full"
    default:
      return "rounded-[10px]"
  }
}

const viewportWidth = computed(() => getViewportWidth(props.state))
const buttonClass = computed(() => [
  radiusClass(props.state.button.radius),
  props.state.button.fullWidth ? "w-full" : "",
].filter(Boolean))
const inputClass = computed(() => [
  radiusClass(props.state.input.radius),
  props.state.input.density === "compact" ? "min-h-9 py-1.5 text-sm" : "",
].filter(Boolean))
const cardClass = computed(() => [
  props.state.card.radius === "xl" ? "rounded-[18px]" : "rounded-[14px]",
  props.state.card.shadow === "lg"
    ? "shadow-[var(--shadow-lg)]"
    : props.state.card.shadow === "md"
      ? "shadow-[var(--shadow-md)]"
      : "shadow-[var(--shadow-sm)]",
  props.state.card.bordered ? "border border-border/70" : "border-transparent",
])
</script>

<template>
  <div class="rounded-2xl border border-border/70 bg-surface-elevated p-4 lg:p-5">
    <div class="mb-3 flex items-center justify-between gap-3">
      <div>
        <h3 class="text-sm font-semibold text-foreground">Preview Canvas</h3>
        <p class="text-xs text-muted-foreground">{{ viewportWidth }}px viewport</p>
      </div>
      <div class="rounded-full border border-border/70 px-3 py-1 font-mono text-xs text-muted-foreground">
        {{ state.componentId }}
      </div>
    </div>

    <div class="overflow-x-auto rounded-2xl border border-dashed border-border/70 bg-background/80 p-4">
      <div
        class="mx-auto flex min-h-64 items-center justify-center rounded-2xl border border-border/60 bg-surface p-6"
        :style="{ width: `${viewportWidth}px`, maxWidth: '100%' }"
      >
        <template v-if="state.componentId === 'button'">
          <Button
            :variant="state.button.variant"
            :size="state.button.size"
            :class="buttonClass"
            :disabled="state.button.disabled || state.button.loading"
          >
            <Loader2
              v-if="state.button.loading"
              data-icon="inline-start"
              class="animate-spin"
            />
            <Sparkles
              v-else-if="state.button.leadingIcon"
              data-icon="inline-start"
            />
            {{ state.button.loading ? "Saving" : state.button.label }}
            <ChevronRight v-if="state.button.trailingIcon" data-icon="inline-end" />
          </Button>
        </template>

        <template v-else-if="state.componentId === 'input'">
          <div class="flex w-full max-w-xl items-center gap-3 rounded-2xl border border-border/70 bg-surface px-3 py-2">
            <span v-if="state.input.prefixLabel" class="text-sm text-muted-foreground">Q</span>
            <Input
              :model-value="state.input.value"
              :placeholder="state.input.placeholder"
              :class="inputClass"
              :aria-invalid="state.input.invalid ? 'true' : undefined"
              :disabled="state.input.disabled"
              :readonly="state.input.readonly"
            />
            <span v-if="state.input.suffixText" class="text-xs text-muted-foreground">⌘K</span>
          </div>
        </template>

        <template v-else-if="state.componentId === 'tag'">
          <Badge :variant="state.tag.variant" :class="radiusClass(state.tag.radius)">
            <Sparkles v-if="state.tag.leadingIcon" />
            {{ state.tag.label }}
          </Badge>
        </template>

        <Card
          v-else
          class="w-full max-w-xl bg-surface"
          :class="cardClass"
        >
          <CardHeader :class="state.card.padding === 'lg' ? 'gap-3' : 'gap-2'">
            <CardTitle>{{ state.card.title }}</CardTitle>
            <CardDescription>{{ state.card.description }}</CardDescription>
          </CardHeader>
          <CardContent :class="state.card.dense ? 'pt-0 text-sm' : 'text-sm leading-6'">
            Prototype content area
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
</template>

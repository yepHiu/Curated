<script setup lang="ts">
import { useAppToastHost } from "@/composables/use-app-toast"

const { toastOpen, toastMessage, toastVariant } = useAppToastHost()

function variantClass(v: string) {
  if (v === "success") {
    return "border-emerald-500/40 bg-emerald-950/90 text-emerald-50"
  }
  if (v === "destructive") {
    return "border-destructive/50 bg-destructive/95 text-destructive-foreground"
  }
  return "border-border/80 bg-card/95 text-foreground shadow-lg shadow-black/20"
}
</script>

<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed bottom-6 left-1/2 z-[110] flex w-[min(92vw,28rem)] -translate-x-1/2 justify-center"
      role="status"
      aria-live="polite"
    >
      <Transition
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="translate-y-2 opacity-0"
        enter-to-class="translate-y-0 opacity-100"
        leave-active-class="transition duration-150 ease-in"
        leave-from-class="translate-y-0 opacity-100"
        leave-to-class="translate-y-1 opacity-0"
      >
        <div
          v-if="toastOpen"
          :class="[
            'pointer-events-auto max-w-full rounded-2xl border px-4 py-3 text-sm leading-snug backdrop-blur-md',
            variantClass(toastVariant),
          ]"
        >
          {{ toastMessage }}
        </div>
      </Transition>
    </div>
  </Teleport>
</template>

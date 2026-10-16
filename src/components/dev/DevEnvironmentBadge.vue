<script setup lang="ts">
import { devRemoteSimulation } from "@/lib/dev-remote-simulation"

const isDev = import.meta.env.DEV
const emit = defineEmits<{
  showDebugTools: []
}>()
</script>

<template>
  <div
    v-if="isDev"
    class="pointer-events-none fixed right-3 bottom-3 z-40 flex items-center gap-1.5"
    aria-label="Development environment"
  >
    <span
      class="select-none rounded-md border border-amber-500/40 bg-amber-500/15 px-2 py-1 font-mono text-[0.65rem] font-bold tracking-widest text-amber-800 uppercase shadow-sm dark:border-amber-400/35 dark:bg-amber-400/10 dark:text-amber-300"
      aria-hidden="true"
    >
      dev
    </span>
    <button
      type="button"
      class="pointer-events-auto select-none rounded-md border border-border bg-background px-2 py-1 font-mono text-[0.65rem] font-bold tracking-widest text-muted-foreground uppercase shadow-sm transition-colors hover:bg-accent hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      data-dev-debug-trigger
      aria-label="Open debug tools"
      aria-haspopup="dialog"
      @click="emit('showDebugTools')"
    >
      debug{{ devRemoteSimulation ? " · remote" : "" }}
    </button>
  </div>
</template>

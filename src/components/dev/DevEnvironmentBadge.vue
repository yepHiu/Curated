<script setup lang="ts">
withDefaults(
  defineProps<{
    showPerfRestore?: boolean
  }>(),
  {
    showPerfRestore: false,
  },
)

const emit = defineEmits<{
  showPerformanceMonitor: []
}>()
</script>

<template>
  <div
    class="pointer-events-none fixed right-3 bottom-3 z-[90] flex items-center gap-1.5"
    aria-label="Development environment"
  >
    <span
      class="select-none rounded-md border border-amber-500/40 bg-amber-500/15 px-2 py-1 font-mono text-[0.65rem] font-bold tracking-widest text-amber-800 uppercase shadow-sm dark:border-amber-400/35 dark:bg-amber-400/10 dark:text-amber-300"
      aria-hidden="true"
    >
      dev
    </span>
    <button
      v-if="showPerfRestore"
      type="button"
      class="pointer-events-auto select-none rounded-md border border-amber-500/40 bg-amber-500/15 px-2 py-1 font-mono text-[0.65rem] font-bold tracking-widest text-amber-800 uppercase shadow-sm transition-colors hover:bg-amber-500/25 hover:text-amber-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/50 dark:border-amber-400/35 dark:bg-amber-400/10 dark:text-amber-300 dark:hover:bg-amber-400/18 dark:hover:text-amber-200 dark:focus-visible:ring-amber-400/45"
      aria-label="Show performance monitor"
      title="Show performance monitor"
      @click="emit('showPerformanceMonitor')"
    >
      perf
    </button>
  </div>
</template>

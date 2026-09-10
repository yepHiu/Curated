<script setup lang="ts">
import { ref, watch } from "vue"
import { useIntersectionObserver } from "@vueuse/core"
import { ImageOff, RotateCcw } from "lucide-vue-next"
import { Skeleton } from "@/components/ui/skeleton"
const props = defineProps<{ src?: string; label: string; pageNumber: number; current?: boolean; retryLabel: string }>()
const emit = defineEmits<{ open: [] }>()
const target = ref<HTMLElement | null>(null)
const visible = ref(typeof IntersectionObserver === "undefined")
const loaded = ref(false)
const failed = ref(false)
const attempt = ref(0)
useIntersectionObserver(target, ([entry]) => { if (entry?.isIntersecting) visible.value = true }, { rootMargin: "160px" })
watch(() => props.src, () => { loaded.value = false; failed.value = false; attempt.value = 0 })
function activate() { if (failed.value) { failed.value = false; attempt.value++ } else emit("open") }
</script>
<template>
  <button ref="target" type="button" data-book-preview-tile
    class="group relative aspect-[2/3] min-w-0 overflow-hidden rounded-xl border border-border/60 bg-muted/30 transition-colors hover:border-primary/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    :class="current ? 'border-primary/70 ring-1 ring-primary/40' : ''" :aria-current="current ? 'page' : undefined" :aria-label="failed ? retryLabel : label" @click="activate">
    <Skeleton v-if="src && !loaded && !failed" class="absolute inset-0 rounded-none" />
    <img v-if="visible && src && !failed" :key="attempt" :src="src" :alt="label" loading="lazy" decoding="async" fetchpriority="low"
      class="size-full object-contain transition-opacity motion-reduce:transition-none" :class="loaded ? 'opacity-100' : 'opacity-0'" @load="loaded = true" @error="failed = true" />
    <span v-if="failed || !src" class="absolute inset-0 flex items-center justify-center text-muted-foreground"><RotateCcw v-if="failed" class="size-6" /><ImageOff v-else class="size-6" /></span>
    <span class="absolute bottom-2 right-2 rounded-md bg-background/90 px-1.5 py-0.5 text-xs tabular-nums text-foreground shadow-sm">{{ pageNumber }}</span>
  </button>
</template>

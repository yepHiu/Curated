<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { Skeleton } from "@/components/ui/skeleton"
import { buildVersionedImageUrl } from "@/lib/image-version"

type MediaStillLoadPayload = {
  naturalWidth: number
  naturalHeight: number
}

const props = withDefaults(
  defineProps<{
    src?: string
    alt: string
    // fill: fill the parent box. intrinsic: keep the image's own aspect ratio.
    layout?: "fill" | "intrinsic"
    fit?: "cover" | "contain"
    // Omit fetchpriority unless explicitly requested so the browser can schedule normally.
    fetchPriority?: "high" | "low" | "auto"
    loading?: "lazy" | "eager"
    // Used to force-refresh images after a rescrape.
    version?: number
  }>(),
  { layout: "fill", fit: "cover", loading: "lazy", version: 0 },
)

const emit = defineEmits<{
  // Fired after the image finishes decoding, including cache hits.
  load: [payload: MediaStillLoadPayload]
  error: []
}>()

const failed = ref(false)
const loaded = ref(false)

const versionedSrc = computed(() => buildVersionedImageUrl(props.src, props.version))

watch(
  () => props.src,
  () => {
    failed.value = false
    loaded.value = false
  },
)

const handleLoad = (event: Event) => {
  loaded.value = true
  const target = event.target as HTMLImageElement | null
  emit("load", {
    naturalWidth: target?.naturalWidth ?? 0,
    naturalHeight: target?.naturalHeight ?? 0,
  })
}

const handleError = () => {
  failed.value = true
  emit("error")
}
</script>

<template>
  <div
    :class="[
      'min-h-0 min-w-0',
      layout === 'intrinsic' ? 'relative w-full' : 'size-full',
    ]"
  >
    <Skeleton
      v-if="versionedSrc && !loaded"
      class="absolute inset-0 z-0 rounded-none"
    />

    <img
      v-if="versionedSrc && !failed"
      :src="versionedSrc"
      :alt="alt"
      referrerpolicy="no-referrer"
      :loading="loading"
      decoding="async"
      :fetchpriority="fetchPriority"
      :class="[
        'pointer-events-none z-[1] select-none transition-opacity duration-300 motion-reduce:transition-none',
        loaded ? 'opacity-100' : 'opacity-0',
        layout === 'intrinsic'
          ? 'relative block h-auto w-full max-h-none min-[480px]:max-h-[min(85vh,120rem)] object-contain align-top'
          : ['absolute inset-0 size-full', fit === 'cover' ? 'object-cover' : 'object-contain'],
      ]"
      @load="handleLoad"
      @error="handleError"
    />
  </div>
</template>

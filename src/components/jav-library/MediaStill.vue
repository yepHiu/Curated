<script setup lang="ts">
import { ref, watch } from "vue"

const props = withDefaults(
  defineProps<{
    src?: string
    alt: string
    /** fill：铺满父级（需固定比例父容器）；intrinsic：按图片自然比例缩放，高度随图变化 */
    layout?: "fill" | "intrinsic"
    fit?: "cover" | "contain"
  }>(),
  { layout: "fill", fit: "cover" },
)

const failed = ref(false)

watch(
  () => props.src,
  () => {
    failed.value = false
  },
)
</script>

<template>
  <img
    v-if="src && !failed"
    :src="src"
    :alt="alt"
    referrerpolicy="no-referrer"
    loading="lazy"
    decoding="async"
    :class="
      layout === 'intrinsic'
        ? 'pointer-events-none block h-auto w-full max-h-none min-[480px]:max-h-[min(85vh,120rem)] select-none object-contain align-top'
        : [
            'pointer-events-none select-none',
            fit === 'cover' ? 'size-full object-cover' : 'size-full object-contain',
          ]
    "
    @error="failed = true"
  />
</template>

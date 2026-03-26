<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { Skeleton } from "@/components/ui/skeleton"
import { buildVersionedImageUrl } from "@/lib/image-version"

const props = withDefaults(
  defineProps<{
    src?: string
    alt: string
    /** fill：铺满父级（需固定比例父容器）；intrinsic：按图片自然比例缩放，高度随图变化 */
    layout?: "fill" | "intrinsic"
    fit?: "cover" | "contain"
    /** 未设置时不写 fetchpriority，由浏览器默认调度（详情页主图/预览勿用 low） */
    fetchPriority?: "high" | "low" | "auto"
    loading?: "lazy" | "eager"
    /** 图片版本号 - 用于强制刷新重新搜刮后的海报 */
    version?: number
  }>(),
  { layout: "fill", fit: "cover", loading: "lazy", version: 0 },
)

const emit = defineEmits<{
  /** 图片解码完成（含缓存命中时尽快触发），供父级控制叠层（如渐变）避免盖住骨架 */
  load: []
  error: []
}>()

const failed = ref(false)
const loaded = ref(false)

/** 带版本号的图片 URL */
const versionedSrc = computed(() => buildVersionedImageUrl(props.src, props.version))

watch(
  () => props.src,
  () => {
    failed.value = false
    loaded.value = false
  },
)

const handleLoad = () => {
  loaded.value = true
  emit("load")
}

const handleError = () => {
  failed.value = true
  emit("error")
}
</script>

<template>
  <!-- fill：勿写 relative，避免与父级 absolute 合并时被 position 覆盖，导致留在 flex 流里按图片固有宽度收缩 -->
  <div
    :class="[
      'min-h-0 min-w-0',
      layout === 'intrinsic' ? 'relative w-full' : 'size-full',
    ]"
  >
    <!-- 骨架屏占位 - 防止布局跳动 (CLS)；z-0 在底层 -->
    <Skeleton
      v-if="versionedSrc && !loaded"
      class="absolute inset-0 z-0 rounded-none"
    />

    <!-- 图片层级高于骨架屏，加载完成后淡入显示 -->
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

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Loader2 } from 'lucide-vue-next'
const props = withDefaults(defineProps<{ src: string; alt: string; active?: boolean }>(), { active: true })
const { t } = useI18n()
const loaded = ref(false), failed = ref(false), zoomed = ref(false), retry = ref(0)
const dimensions = ref('')
watch(() => props.src, () => { loaded.value = false; failed.value = false; zoomed.value = false })
watch(() => props.active, () => { zoomed.value = false })
function onLoad(event: Event) {
  const img = event.target as HTMLImageElement
  dimensions.value = `${img.naturalWidth} × ${img.naturalHeight}`
  loaded.value = true
}
</script>
<template>
  <div class="relative h-full w-full min-w-0 overflow-hidden [contain:inline-size]">
    <div class="flex h-full w-full overflow-auto" :class="zoomed ? 'items-start justify-start' : 'items-center justify-center'" :tabindex="zoomed ? 0 : undefined" :aria-label="alt">
      <img v-if="!failed" :key="retry" :src="src" :alt="alt" decoding="async" draggable="false" :class="zoomed ? 'max-w-none shrink-0' : 'h-full w-full object-contain p-2 sm:p-4'" @load="onLoad" @error="failed = true" />
    </div>
    <div v-if="active !== false" class="absolute right-2 top-2 flex items-center gap-2 rounded-md bg-background/95 p-1 text-foreground" @click.stop @pointerdown.stop>
      <Loader2 v-if="!loaded && !failed" class="size-4 animate-spin motion-reduce:animate-none" :aria-label="t('common.loading')" />
      <Button v-if="failed" size="sm" variant="outline" @click="failed = false; retry++">{{ t('curated.retryLoad') }}</Button>
      <template v-else-if="loaded">
        <span class="hidden text-xs sm:inline">{{ dimensions }}</span>
        <Button size="sm" variant="ghost" :aria-pressed="zoomed" @click="zoomed = !zoomed">{{ zoomed ? t('curated.fitImage') : '100%' }}</Button>
      </template>
    </div>
  </div>
</template>

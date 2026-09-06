<script setup lang="ts">
import { ref, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
const props = defineProps<{ height: number; columns: number }>()
const host = ref<HTMLElement | null>(null)
const visible = ref(true)
const focused = ref(false)
let observer: IntersectionObserver | undefined
async function revealRow() {
  visible.value = true
  await nextTick()
  host.value?.querySelector<HTMLElement>('button, a[href], [tabindex="0"]')?.focus()
}
onMounted(() => {
  if (typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver(([entry]) => { visible.value = Boolean(entry?.isIntersecting) }, { rootMargin: '600px 0px' })
  if (host.value) observer.observe(host.value)
})
onBeforeUnmount(() => observer?.disconnect())
</script>
<template>
  <div ref="host" :style="{ minHeight: `${props.height}px`, gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }" class="grid gap-4" @focusin="focused = true" @focusout="focused = Boolean(host?.contains($event.relatedTarget as Node))">
    <slot v-if="visible || focused" />
    <button v-else type="button" class="col-span-full min-h-11 text-sm text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring" @focus="revealRow" @click="revealRow">{{ t('curated.showFrameRow') }}</button>
  </div>
</template>

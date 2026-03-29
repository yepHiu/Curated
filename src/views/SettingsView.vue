<script setup lang="ts">
import { onBeforeUnmount, provide, ref, watch } from "vue"
import SettingsPage from "@/components/jav-library/SettingsPage.vue"
import {
  noteSettingsScrollPosition,
  resetSettingsScrollSnapshot,
  SETTINGS_SCROLL_EL_KEY,
} from "@/composables/use-settings-scroll-preserve"

const settingsScrollEl = ref<HTMLElement | null>(null)
provide(SETTINGS_SCROLL_EL_KEY, settingsScrollEl)

let detachScrollListener: (() => void) | undefined

watch(
  settingsScrollEl,
  (el) => {
    detachScrollListener?.()
    detachScrollListener = undefined
    if (!el) return
    const onScroll = () => noteSettingsScrollPosition(el)
    el.addEventListener("scroll", onScroll, { passive: true })
    noteSettingsScrollPosition(el)
    detachScrollListener = () => el.removeEventListener("scroll", onScroll)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  detachScrollListener?.()
  resetSettingsScrollSnapshot()
})
</script>

<template>
  <div class="h-full min-h-0 min-w-0 overflow-hidden pr-2">
    <SettingsPage />
  </div>
</template>

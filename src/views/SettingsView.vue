<script setup lang="ts">
import { onBeforeUnmount, onMounted, provide, ref } from "vue"
import SettingsPage from "@/components/jav-library/SettingsPage.vue"
import {
  noteSettingsScrollPosition,
  resetSettingsScrollSnapshot,
  SETTINGS_SCROLL_EL_KEY,
  SETTINGS_SCROLL_ROOT_ID,
} from "@/composables/use-settings-scroll-preserve"

const settingsScrollEl = ref<HTMLElement | null>(null)
provide(SETTINGS_SCROLL_EL_KEY, settingsScrollEl)

let detachScrollListener: (() => void) | undefined

onMounted(() => {
  const root =
    settingsScrollEl.value ?? (document.getElementById(SETTINGS_SCROLL_ROOT_ID) as HTMLElement | null)
  if (!root) return
  const onScroll = () => noteSettingsScrollPosition(root)
  root.addEventListener("scroll", onScroll, { passive: true })
  noteSettingsScrollPosition(root)
  detachScrollListener = () => root.removeEventListener("scroll", onScroll)
})

onBeforeUnmount(() => {
  detachScrollListener?.()
  resetSettingsScrollSnapshot()
})
</script>

<template>
  <div
    :id="SETTINGS_SCROLL_ROOT_ID"
    ref="settingsScrollEl"
    class="h-full min-h-0 overflow-x-hidden overflow-y-auto overscroll-contain pr-2 [overflow-anchor:none]"
  >
    <SettingsPage />
  </div>
</template>

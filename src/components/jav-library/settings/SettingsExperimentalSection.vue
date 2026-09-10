<script setup lang="ts">
import { defineAsyncComponent } from "vue"
/** 按需加载漫画 Beta 设置，保持常规设置页包体独立。 */
const SettingsComicLibrarySection = defineAsyncComponent(() => import("./SettingsComicLibrarySection.vue"))
/** 按需加载写真 Beta 设置。 */
const SettingsPhotoLibrarySection = defineAsyncComponent(() => import("./SettingsPhotoLibrarySection.vue"))
</script>

<template>
  <div class="flex flex-col gap-3">
    <SettingsComicLibrarySection />
    <SettingsPhotoLibrarySection />
  </div>
</template>

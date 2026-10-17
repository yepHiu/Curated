<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import type { AtlasSnapshot } from "./types";
import { mountAtlas } from "./explorer.mjs";
import markup from "./explorer.html?raw";
import css from "./explorer.css?raw";

const props = defineProps<{ snapshot: AtlasSnapshot }>();
const container = ref<HTMLDivElement>();
let shadow: ShadowRoot | undefined;
let explorer: ReturnType<typeof mountAtlas> | undefined;

function render() {
  if (!shadow) return;
  const state = explorer ? { ...explorer.state } : undefined;
  explorer?.destroy();
  // Only build-time static markup enters this host. The explorer escapes all scan metadata.
  shadow.innerHTML = `<style>${css}</style>${markup}`;
  explorer = mountAtlas(shadow, props.snapshot, state, true);
}
onMounted(() => {
  shadow = container.value?.attachShadow({ mode: "open" });
  render();
});
watch(() => props.snapshot, render);
onBeforeUnmount(() => explorer?.destroy());
</script>

<template><div ref="container" class="atlas-explorer" /></template>

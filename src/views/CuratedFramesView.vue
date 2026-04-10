<script setup lang="ts">
import type { ComputedRef, Ref } from "vue"
import { computed, ref, unref } from "vue"
import CuratedFramesBatchActionBar from "@/components/jav-library/CuratedFramesBatchActionBar.vue"
import CuratedFramesLibrary from "@/components/jav-library/CuratedFramesLibrary.vue"

const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"

type CuratedFramesLibraryExpose = {
  batchMode: Ref<boolean>
  isEmpty: ComputedRef<boolean>
  selectedFrameIds: Ref<string[]>
  exportBusy: Ref<boolean>
  batchDeleteBusy: Ref<boolean>
  batchShowSelectVisible: ComputedRef<boolean>
  exportToolbarError: Ref<string>
  exitBatchMode: () => void
  clearExportSelection: () => void
  selectAllVisibleUpTo20: () => void
  deleteSelectedFrames: () => void
  exportSelectedWebp: () => void | Promise<void>
  exportSelectedPng: () => void | Promise<void>
}

const curatedLibRef = ref<CuratedFramesLibraryExpose | null>(null)

const showCuratedBatchBar = computed(() => {
  const c = curatedLibRef.value
  if (!c) return false
  return unref(c.batchMode) && !unref(c.isEmpty)
})

const batchSelectedCount = computed(() => unref(curatedLibRef.value?.selectedFrameIds)?.length ?? 0)

const batchExportBusy = computed(() => unref(curatedLibRef.value?.exportBusy) ?? false)
const batchDeleteBusy = computed(() => unref(curatedLibRef.value?.batchDeleteBusy) ?? false)

const batchShowSelectVisible = computed(() => unref(curatedLibRef.value?.batchShowSelectVisible) ?? false)

const batchExportError = computed(() => unref(curatedLibRef.value?.exportToolbarError) ?? "")

function onCuratedBatchExit() {
  curatedLibRef.value?.exitBatchMode()
}

function onCuratedBatchClear() {
  curatedLibRef.value?.clearExportSelection()
}

function onCuratedBatchSelectAllVisible() {
  curatedLibRef.value?.selectAllVisibleUpTo20()
}

function onCuratedBatchExportWebp() {
  void curatedLibRef.value?.exportSelectedWebp()
}

function onCuratedBatchExportPng() {
  void curatedLibRef.value?.exportSelectedPng()
}

function onCuratedBatchDeleteSelected() {
  curatedLibRef.value?.deleteSelectedFrames()
}
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
    <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden pr-2">
      <CuratedFramesLibrary ref="curatedLibRef" class="min-h-0 min-w-0 flex-1" />
    </div>

    <CuratedFramesBatchActionBar
      v-if="showCuratedBatchBar"
      :selected-count="batchSelectedCount"
      :export-busy="batchExportBusy"
      :delete-busy="batchDeleteBusy"
      :show-select-visible="batchShowSelectVisible"
      :use-web-api="useWebApi"
      :export-error="batchExportError"
      @exit="onCuratedBatchExit"
      @clear-selection="onCuratedBatchClear"
      @select-all-visible="onCuratedBatchSelectAllVisible"
      @delete-selected="onCuratedBatchDeleteSelected"
      @export-webp="onCuratedBatchExportWebp"
      @export-png="onCuratedBatchExportPng"
    />
  </div>
</template>

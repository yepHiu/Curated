<script setup lang="ts">
import { onClickOutside, useEventListener } from "@vueuse/core"
import { Download, Trash2 } from "lucide-vue-next"
import { computed, nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { CuratedFrameRecord } from "@/domain/curated-frame/types"
import { cn } from "@/lib/utils"

const props = defineProps<{
  frame: CuratedFrameRecord
  x: number
  y: number
  useWebApi: boolean
}>()

const emit = defineEmits<{
  close: []
  exportWebp: []
  exportPng: []
  delete: []
}>()

const { t } = useI18n()

const menuRef = ref<HTMLElement | null>(null)

const itemClass = cn(
  "relative flex w-full cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm outline-hidden select-none transition-colors",
  "hover:bg-accent hover:text-accent-foreground",
  "focus-visible:bg-accent focus-visible:text-accent-foreground",
  "disabled:pointer-events-none disabled:opacity-50",
  "[&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 [&_svg:not([class*='text-'])]:text-muted-foreground",
)

const destructiveItemClass = cn(
  itemClass,
  "text-destructive hover:bg-destructive/10 hover:text-destructive dark:hover:bg-destructive/20",
  "focus-visible:bg-destructive/10 focus-visible:text-destructive dark:focus-visible:bg-destructive/20",
  "[&_svg]:!text-destructive",
)

const exportHint = computed(() => (props.useWebApi ? undefined : t("curated.exportRequiresApi")))

function close() {
  emit("close")
}

function onExportWebp() {
  if (!props.useWebApi) return
  emit("exportWebp")
  close()
}

function onExportPng() {
  if (!props.useWebApi) return
  emit("exportPng")
  close()
}

function onDelete() {
  emit("delete")
  close()
}

onClickOutside(menuRef, () => close())

useEventListener(
  typeof window !== "undefined" ? window : null,
  "scroll",
  () => close(),
  true,
)

useEventListener("keydown", (event: KeyboardEvent) => {
  if (event.key === "Escape") {
    close()
  }
})

watch(
  () => [props.x, props.y, props.frame.id] as const,
  () => {
    void nextTick(() => {
      menuRef.value?.focus?.()
    })
  },
  { immediate: true },
)
</script>

<template>
  <Teleport to="body">
    <div
      ref="menuRef"
      role="menu"
      tabindex="-1"
      class="bg-popover text-popover-foreground fixed z-[100] min-w-[11rem] rounded-md border p-1 shadow-md outline-none"
      :style="{ left: `${x}px`, top: `${y}px` }"
      @contextmenu.prevent
    >
      <button
        type="button"
        role="menuitem"
        :class="itemClass"
        :disabled="!useWebApi"
        :title="exportHint"
        data-curated-frame-context-action="export-webp"
        @click="onExportWebp"
      >
        <Download class="size-4 shrink-0" aria-hidden="true" />
        {{ t("curated.exportWebp") }}
      </button>
      <button
        type="button"
        role="menuitem"
        :class="itemClass"
        :disabled="!useWebApi"
        :title="exportHint"
        data-curated-frame-context-action="export-png"
        @click="onExportPng"
      >
        <Download class="size-4 shrink-0" aria-hidden="true" />
        {{ t("curated.exportPng") }}
      </button>
      <button
        type="button"
        role="menuitem"
        :class="destructiveItemClass"
        data-curated-frame-context-action="delete"
        @click="onDelete"
      >
        <Trash2 class="size-4 shrink-0" aria-hidden="true" />
        {{ t("curated.deleteThisFrame") }}
      </button>
    </div>
  </Teleport>
</template>

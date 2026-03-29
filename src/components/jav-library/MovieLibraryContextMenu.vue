<script setup lang="ts">
import { onClickOutside, useEventListener } from "@vueuse/core"
import { computed, nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { FolderOpen, Pencil, RefreshCw, Trash2 } from "lucide-vue-next"
import type { Movie } from "@/domain/movie/types"
import { cn } from "@/lib/utils"

const props = defineProps<{
  movie: Movie
  x: number
  y: number
  metadataRefreshBusy?: boolean
}>()

const emit = defineEmits<{
  close: []
  edit: []
  refreshMetadata: []
  revealInFileManager: []
  moveToTrash: []
  restore: []
  deletePermanently: []
}>()

const { t } = useI18n()

const menuRef = ref<HTMLElement | null>(null)

const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"
const isTrashed = computed(() => Boolean(props.movie.trashedAt?.trim()))
const canRevealInFileManager = computed(
  () => useWebApi && Boolean(props.movie.location?.trim()) && !isTrashed.value,
)

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

function close() {
  emit("close")
}

function onEdit() {
  emit("edit")
  close()
}

function onRefresh() {
  if (props.metadataRefreshBusy) return
  emit("refreshMetadata")
  close()
}

function onReveal() {
  if (!canRevealInFileManager.value) return
  emit("revealInFileManager")
  close()
}

function onTrash() {
  emit("moveToTrash")
  close()
}

function onRestore() {
  emit("restore")
  close()
}

function onPermanent() {
  emit("deletePermanently")
  close()
}

onClickOutside(menuRef, () => close())

useEventListener(
  typeof window !== "undefined" ? window : null,
  "scroll",
  () => close(),
  true,
)

useEventListener("keydown", (e: KeyboardEvent) => {
  if (e.key === "Escape") {
    close()
  }
})

watch(
  () => [props.x, props.y, props.movie.id] as const,
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
      @mouseleave="close"
    >
      <template v-if="!isTrashed">
        <button
          type="button"
          role="menuitem"
          :class="itemClass"
          @click="onEdit"
        >
          <Pencil
            class="size-4 shrink-0"
            aria-hidden="true"
          />
          {{ t("detailPanel.editMovie") }}
        </button>
        <button
          type="button"
          role="menuitem"
          :class="itemClass"
          :disabled="metadataRefreshBusy"
          @click="onRefresh"
        >
          <RefreshCw
            class="size-4 shrink-0"
            :class="metadataRefreshBusy ? 'animate-spin' : ''"
            aria-hidden="true"
          />
          {{
            metadataRefreshBusy
              ? t("detailPanel.scrapeSubmitting")
              : t("detailPanel.refreshMetadata")
          }}
        </button>
        <button
          type="button"
          role="menuitem"
          :class="itemClass"
          :disabled="!canRevealInFileManager"
          :title="
            !useWebApi
              ? t('detailPanel.revealInFileManagerMockHint')
              : !movie.location?.trim()
                ? t('detailPanel.revealInFileManagerNoPath')
                : undefined
          "
          @click="onReveal"
        >
          <FolderOpen
            class="size-4 shrink-0"
            aria-hidden="true"
          />
          {{ t("detailPanel.revealInFileManager") }}
        </button>
        <button
          type="button"
          role="menuitem"
          :class="destructiveItemClass"
          @click="onTrash"
        >
          <Trash2
            class="size-4 shrink-0"
            aria-hidden="true"
          />
          {{ t("detailPanel.moveToTrash") }}
        </button>
      </template>
      <template v-else>
        <button
          type="button"
          role="menuitem"
          :class="itemClass"
          @click="onRestore"
        >
          {{ t("detailPanel.restoreMovie") }}
        </button>
        <button
          type="button"
          role="menuitem"
          :class="destructiveItemClass"
          @click="onPermanent"
        >
          {{ t("detailPanel.deleteMoviePermanently") }}
        </button>
      </template>
    </div>
  </Teleport>
</template>

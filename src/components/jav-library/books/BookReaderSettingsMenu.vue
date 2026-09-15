<script setup lang="ts">
import { computed, ref } from "vue"
import { Settings2 } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

/** 阅读器底栏菜单相对整条工具栏的垂直间距，避免遮住 HUD。 */
const READER_TOOLBAR_MENU_SIDE_OFFSET = 8
/** 菜单与视口边缘的碰撞内边距，给上方留出呼吸距离。 */
const READER_TOOLBAR_MENU_COLLISION_PADDING = 8

type ReaderMode = "page" | "scroll"
type ReaderFit = "contain" | "width"
type ReaderDirection = "ltr" | "rtl"

const props = withDefaults(
  defineProps<{
    kind: "comics" | "photos"
    mode: ReaderMode
    fit: ReaderFit
    direction: ReaderDirection
    showStitch?: boolean
    stitched?: boolean
  }>(),
  {
    showStitch: false,
    stitched: false,
  },
)

const emit = defineEmits<{
  "update:mode": [value: ReaderMode]
  "update:fit": [value: ReaderFit]
  "update:direction": [value: ReaderDirection]
  stitchPrevious: []
  stitchNext: []
  clearStitch: []
}>()

const { t } = useI18n()

const modeLabelKey = computed(() =>
  props.kind === "photos" ? "settings.photoViewerMode" : "settings.comicReaderMode",
)
const fitLabelKey = computed(() =>
  props.kind === "photos" ? "settings.photoViewerFit" : "settings.comicReaderFit",
)
const directionLabelKey = computed(() =>
  props.kind === "photos" ? "settings.photoViewerDirection" : "settings.comicReaderDirection",
)
const modePageKey = computed(() =>
  props.kind === "photos" ? "settings.photoViewerModePage" : "settings.comicReaderModePage",
)
const modeScrollKey = computed(() =>
  props.kind === "photos" ? "settings.photoViewerModeScroll" : "settings.comicReaderModeScroll",
)
const fitContainKey = computed(() =>
  props.kind === "photos" ? "settings.photoViewerFitContain" : "settings.comicReaderFitContain",
)
const fitWidthKey = computed(() =>
  props.kind === "photos" ? "settings.photoViewerFitWidth" : "settings.comicReaderFitWidth",
)
const directionLtrKey = computed(() =>
  props.kind === "photos" ? "settings.photoViewerDirectionLtr" : "settings.comicReaderDirectionLtr",
)
const directionRtlKey = computed(() =>
  props.kind === "photos" ? "settings.photoViewerDirectionRtl" : "settings.comicReaderDirectionRtl",
)
const stitchVisible = computed(() => props.showStitch && props.mode === "page")

/** 切换翻页或滚动模式。 */
function selectMode(value: unknown) {
  if (value === "page" || value === "scroll") {
    emit("update:mode", value)
  }
}

/** 切换完整页或适应宽度。 */
function selectFit(value: unknown) {
  if (value === "contain" || value === "width") {
    emit("update:fit", value)
  }
}

/** 切换从左到右或从右到左。 */
function selectDirection(value: unknown) {
  if (value === "ltr" || value === "rtl") {
    emit("update:direction", value)
  }
}

const capturedToolbarEl = ref<HTMLElement | null>(null)
/** 用触发按钮所属的底栏卡片作为菜单定位锚点。 */
const menuAnchor = computed(() => capturedToolbarEl.value ?? undefined)

/** 从组件实例或原生节点取出可用的 HTML 元素。 */
function unwrapElement(el: unknown): HTMLElement | null {
  if (el instanceof HTMLElement) return el
  if (el && typeof el === "object" && "$el" in el) {
    const root = (el as { $el: unknown }).$el
    return root instanceof HTMLElement ? root : null
  }
  return null
}

/** 记下触发按钮所属的底栏卡片，供菜单按整条工具栏定位。 */
function bindTrigger(el: unknown) {
  if (el == null) {
    capturedToolbarEl.value = null
    return
  }
  capturedToolbarEl.value = unwrapElement(el)?.closest("[data-book-reader-chrome]") ?? null
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        :ref="bindTrigger"
        type="button"
        variant="ghost"
        size="icon"
        class="rounded-lg"
        data-reader-settings-trigger
        :aria-label="t('bookBrowser.readerSettings')"
        :title="t('bookBrowser.readerSettings')"
      >
        <Settings2 />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent
      side="top"
      align="center"
      :side-offset="READER_TOOLBAR_MENU_SIDE_OFFSET"
      :side-flip="false"
      :collision-padding="READER_TOOLBAR_MENU_COLLISION_PADDING"
      :reference="menuAnchor"
      class="w-56 rounded-2xl border-border/70"
      data-reader-settings-menu
      @click.stop
    >
      <DropdownMenuLabel class="font-normal text-muted-foreground">
        {{ t("bookBrowser.readerSettings") }}
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <p class="px-2 pb-1 pt-1.5 text-xs text-muted-foreground">{{ t(modeLabelKey) }}</p>
      <DropdownMenuGroup>
        <DropdownMenuRadioGroup :model-value="mode" @update:model-value="selectMode">
          <DropdownMenuRadioItem value="page">{{ t(modePageKey) }}</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="scroll">{{ t(modeScrollKey) }}</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuGroup>
      <DropdownMenuSeparator />
      <p class="px-2 pb-1 pt-1.5 text-xs text-muted-foreground">{{ t(fitLabelKey) }}</p>
      <DropdownMenuGroup>
        <DropdownMenuRadioGroup :model-value="fit" @update:model-value="selectFit">
          <DropdownMenuRadioItem value="contain">{{ t(fitContainKey) }}</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="width">{{ t(fitWidthKey) }}</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuGroup>
      <DropdownMenuSeparator />
      <p class="px-2 pb-1 pt-1.5 text-xs text-muted-foreground">{{ t(directionLabelKey) }}</p>
      <DropdownMenuGroup>
        <DropdownMenuRadioGroup :model-value="direction" @update:model-value="selectDirection">
          <DropdownMenuRadioItem value="ltr">{{ t(directionLtrKey) }}</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="rtl">{{ t(directionRtlKey) }}</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuGroup>
      <template v-if="stitchVisible">
        <DropdownMenuSeparator />
        <p class="px-2 pb-1 pt-1.5 text-xs text-muted-foreground">{{ t("comics.readerStitch") }}</p>
        <DropdownMenuGroup>
          <button
            type="button"
            data-reader-stitch-previous
            class="flex w-full items-center rounded-sm px-2 py-1.5 text-left text-sm outline-hidden hover:bg-accent"
            :aria-label="t('comics.readerStitchPrevious')"
            :title="t('comics.readerStitchPrevious')"
            @click="emit('stitchPrevious')"
          >
            {{ t("comics.readerStitchPreviousShort") }}
          </button>
          <button
            type="button"
            data-reader-stitch-next
            class="flex w-full items-center rounded-sm px-2 py-1.5 text-left text-sm outline-hidden hover:bg-accent"
            :aria-label="t('comics.readerStitchNext')"
            :title="t('comics.readerStitchNext')"
            @click="emit('stitchNext')"
          >
            {{ t("comics.readerStitchNextShort") }}
          </button>
          <button
            v-if="stitched"
            type="button"
            data-reader-clear-stitch
            class="flex w-full items-center rounded-sm px-2 py-1.5 text-left text-sm outline-hidden hover:bg-accent"
            :aria-label="t('comics.readerClearStitch')"
            :title="t('comics.readerClearStitch')"
            @click="emit('clearStitch')"
          >
            {{ t("comics.readerClearStitchShort") }}
          </button>
        </DropdownMenuGroup>
      </template>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

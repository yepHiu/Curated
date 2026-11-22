<script setup lang="ts">
import { useEventListener } from "@vueuse/core"
import { RefreshCw } from "lucide-vue-next"
import { computed, ref } from "vue"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem } from "@/components/ui/dropdown-menu"

defineProps<{ label: string }>()

const open = ref(false)
const point = ref({ x: 0, y: 0 })
const reference = computed(() => ({
  getBoundingClientRect: () => new DOMRect(point.value.x, point.value.y, 0, 0),
}))
const excluded = [
  "input", "textarea", "select", "option", "button", "a", "label", "summary",
  "[contenteditable]:not([contenteditable='false'])", "[role='textbox']",
  "[role='button']", "[role='link']", "[role='menu']", "[role='menuitem']",
  "[role='slider']", "[role='checkbox']", "[role='switch']", "[role='tab']",
  "[tabindex]:not([tabindex='-1'])", "img", "video", "audio", "canvas", "iframe",
  "[data-slot='context-menu-trigger']", "[data-desktop-reload-exclude]",
].join(",")

useEventListener(document, "contextmenu", (event: MouseEvent) => {
  if (event.defaultPrevented || !(event.target instanceof Element)) return
  if (event.target.closest(excluded) || window.getSelection()?.toString()) return
  event.preventDefault()
  point.value = { x: event.clientX, y: event.clientY }
  open.value = true
})

function close() {
  open.value = false
}

function reload() {
  window.location.reload()
}

useEventListener(window, "blur", close)
useEventListener(window, "resize", close)
useEventListener(document, "scroll", close, true)
</script>

<template>
  <DropdownMenu v-model:open="open" :modal="false">
    <DropdownMenuContent
      :reference="reference"
      :side-offset="0"
      :collision-padding="8"
      align="start"
      :aria-label="label"
    >
      <DropdownMenuGroup>
        <DropdownMenuItem @select="reload">
          <RefreshCw aria-hidden="true" />
          {{ label }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

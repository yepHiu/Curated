<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { Info, Sparkles } from "lucide-vue-next"
import { ContextMenu, ContextMenuTrigger, ContextMenuContent, ContextMenuGroup, ContextMenuItem } from "@/components/ui/context-menu"

defineProps<{ canOrganize?: boolean; disabled?: boolean }>()
const emit = defineEmits<{ organize: []; openDetails: [] }>()
const { t } = useI18n()
const openingDialog = ref(false)
function organize() { openingDialog.value = true; emit("organize") }
</script>

<template>
  <ContextMenu @update:open="value => { if (value) openingDialog = false }">
    <ContextMenuTrigger as-child :disabled="disabled"><slot /></ContextMenuTrigger>
    <ContextMenuContent class="min-w-[11rem]" @close-auto-focus="openingDialog && $event.preventDefault()">
      <ContextMenuGroup>
        <ContextMenuItem v-if="canOrganize" data-book-context-organize @select="organize"><Sparkles aria-hidden="true" />{{ t('topics.action') }}</ContextMenuItem>
        <ContextMenuItem @select="emit('openDetails')"><Info aria-hidden="true" />{{ t('bookBrowser.viewDetails') }}</ContextMenuItem>
      </ContextMenuGroup>
    </ContextMenuContent>
  </ContextMenu>
</template>

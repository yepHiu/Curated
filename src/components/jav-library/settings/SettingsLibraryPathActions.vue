<script setup lang="ts">
import {
  FolderOpen,
  MoreVertical,
  Pencil,
  RefreshCw,
  Trash2,
} from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type LibraryPathActionTarget = {
  id: string
  title: string
  path: string
}

defineProps<{
  path: LibraryPathActionTarget
  revealBusy?: boolean
  scanBusy?: boolean
}>()

const emit = defineEmits<{
  reveal: [path: LibraryPathActionTarget]
  edit: [path: LibraryPathActionTarget]
  rescan: [path: LibraryPathActionTarget]
  remove: [path: LibraryPathActionTarget]
}>()

const { t } = useI18n()
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        type="button"
        data-overflow-menu-trigger
        variant="ghost"
        size="icon"
        :aria-label="t('settings.moreActions')"
        class="border-0 bg-transparent text-muted-foreground shadow-none ring-0 transition-colors hover:bg-muted/50 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/45 data-[state=open]:bg-muted/55 data-[state=open]:text-foreground"
      >
        <MoreVertical class="size-4" aria-hidden="true" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="min-w-[11rem]">
      <DropdownMenuGroup>
        <DropdownMenuItem
          :disabled="revealBusy"
          @click="emit('reveal', path)"
        >
          <FolderOpen
            class="size-4 shrink-0"
            :class="revealBusy ? 'animate-pulse' : ''"
            aria-hidden="true"
          />
          {{ t("settings.openPath") }}
        </DropdownMenuItem>
        <DropdownMenuItem @click="emit('edit', path)">
          <Pencil class="size-4 shrink-0" aria-hidden="true" />
          {{ t("settings.editTitle") }}
        </DropdownMenuItem>
        <DropdownMenuItem
          :disabled="scanBusy"
          @click="emit('rescan', path)"
        >
          <RefreshCw
            class="size-4 shrink-0"
            :class="scanBusy ? 'animate-spin' : ''"
            aria-hidden="true"
          />
          {{ t("settings.rescan") }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <DropdownMenuGroup>
        <DropdownMenuItem
          variant="destructive"
          @click="emit('remove', path)"
        >
          <Trash2 class="size-4 shrink-0" aria-hidden="true" />
          {{ t("settings.removePathConfirmAction") }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

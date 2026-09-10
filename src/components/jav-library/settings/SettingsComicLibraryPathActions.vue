<script setup lang="ts">
import { MoreVertical, RefreshCw, Trash2 } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import type { ComicLibrarySetting } from "@/domain/comic/types"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

const props = defineProps<{
  path: ComicLibrarySetting
  scanBusy?: boolean
}>()

const emit = defineEmits<{
  scan: [path: ComicLibrarySetting]
  remove: [id: string]
}>()

const { t } = useI18n()
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        type="button"
        data-comic-path-actions-trigger
        variant="ghost"
        size="icon"
        :aria-label="t('settings.moreActions')"
        class="border-0 bg-transparent text-muted-foreground shadow-none ring-0 transition-colors hover:bg-muted/50 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/45 data-[state=open]:bg-muted/55 data-[state=open]:text-foreground"
      >
        <MoreVertical class="size-4" aria-hidden="true" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="min-w-[10rem]">
      <DropdownMenuGroup>
        <DropdownMenuItem
          :disabled="props.scanBusy"
          :data-scan-comic-path="props.path.id"
          @click="emit('scan', props.path)"
        >
          <RefreshCw
            class="size-4 shrink-0"
            :class="props.scanBusy ? 'animate-spin' : ''"
            aria-hidden="true"
          />
          {{ t("settings.comicLibraryPathScan") }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <DropdownMenuGroup>
        <DropdownMenuItem
          variant="destructive"
          :data-remove-comic-path="props.path.id"
          @click="emit('remove', props.path.id)"
        >
          <Trash2 class="size-4 shrink-0" aria-hidden="true" />
          {{ t("settings.comicLibraryPathRemove") }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

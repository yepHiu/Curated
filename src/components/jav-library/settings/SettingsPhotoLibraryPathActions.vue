<script setup lang="ts">
import { Check, Download, MoreVertical, RefreshCw, Trash2 } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import type { PhotoLibrarySetting } from "@/domain/photo/types"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

const props = defineProps<{
  path: PhotoLibrarySetting
  scanBusy?: boolean
  isDefault?: boolean
  defaultImportPathSaving?: boolean
}>()

const emit = defineEmits<{
  scan: [path: PhotoLibrarySetting]
  remove: [id: string]
  setDefault: [id: string]
}>()

const { t } = useI18n()
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        type="button"
        data-photo-path-actions-trigger
        variant="ghost"
        size="icon"
        :aria-label="t('settings.moreActions')"
        class="shrink-0 border-0 bg-transparent text-muted-foreground shadow-none ring-0 transition-colors hover:bg-muted/50 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/45 data-[state=open]:bg-muted/55 data-[state=open]:text-foreground"
      >
        <MoreVertical aria-hidden="true" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="min-w-[10rem]">
      <DropdownMenuGroup>
        <DropdownMenuItem
          :disabled="props.isDefault || props.defaultImportPathSaving"
          data-set-default-import-path
          @click="emit('setDefault', props.path.id)"
        >
          <Check v-if="props.isDefault" aria-hidden="true" />
          <Download v-else aria-hidden="true" />
          {{ props.isDefault ? t('settings.defaultImportPathLabel') : t('settings.setDefaultImportPath') }}
        </DropdownMenuItem>
        <DropdownMenuItem
          :disabled="props.scanBusy"
          :data-scan-photo-path="props.path.id"
          @click="emit('scan', props.path)"
        >
          <RefreshCw
            class="size-4 shrink-0"
            :class="props.scanBusy ? 'animate-spin' : ''"
            aria-hidden="true"
          />
          {{ t("settings.photoLibraryPathScan") }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <DropdownMenuGroup>
        <DropdownMenuItem
          variant="destructive"
          :data-remove-photo-path="props.path.id"
          @click="emit('remove', props.path.id)"
        >
          <Trash2 class="size-4 shrink-0" aria-hidden="true" />
          {{ t("settings.photoLibraryPathRemove") }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

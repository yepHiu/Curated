<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

type LibraryPathRemoveTarget = {
  id: string
  title: string
  path: string
}

defineProps<{
  open: boolean
  pending: LibraryPathRemoveTarget | null
  busy: boolean
  contentClass: string
}>()

const emit = defineEmits<{
  "update:open": [open: boolean]
  confirm: []
}>()

const { t } = useI18n()
</script>

<template>
  <Dialog
    :open="open"
    @update:open="emit('update:open', $event)"
  >
    <DialogContent :class="contentClass">
      <DialogHeader>
        <DialogTitle>{{ t("settings.removePathConfirmTitle") }}</DialogTitle>
        <DialogDescription>
          <div class="space-y-2">
            <p class="text-pretty">
              {{
                t("settings.removePathConfirmDesc", {
                  title: pending?.title ?? "—",
                })
              }}
            </p>
            <p
              v-if="pending?.path"
              class="break-all font-mono text-xs text-muted-foreground"
            >
              {{ pending.path }}
            </p>
          </div>
        </DialogDescription>
      </DialogHeader>
      <DialogFooter class="gap-3">
        <DialogClose as-child>
          <Button
            type="button"
            variant="outline"
            class="rounded-2xl"
            :disabled="busy"
          >
            {{ t("common.cancel") }}
          </Button>
        </DialogClose>
        <Button
          type="button"
          variant="destructive"
          class="rounded-2xl"
          :disabled="busy || !pending"
          data-remove-path-confirm
          @click="emit('confirm')"
        >
          {{ busy ? t("settings.removePathConfirmWorking") : t("settings.removePathConfirmAction") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

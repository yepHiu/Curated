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

const open = defineModel<boolean>("open", { required: true })

const emit = defineEmits<{
  confirm: []
}>()

const { t } = useI18n()

function onConfirm() {
  open.value = false
  emit("confirm")
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t("comics.deleteComic") }}</DialogTitle>
        <DialogDescription class="text-pretty">
          {{ t("comics.deleteComicConfirm") }}
        </DialogDescription>
      </DialogHeader>
      <DialogFooter class="gap-3">
        <DialogClose as-child>
          <Button type="button" variant="outline" class="rounded-2xl">
            {{ t("common.cancel") }}
          </Button>
        </DialogClose>
        <Button
          type="button"
          variant="destructive"
          class="rounded-2xl"
          data-comic-delete-confirm
          @click="onConfirm"
        >
          {{ t("comics.confirmDeleteComic") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

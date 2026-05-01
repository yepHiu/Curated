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

defineProps<{
  open: boolean
  label: string
  error: string
  busy: boolean
}>()

const emit = defineEmits<{
  "update:open": [open: boolean]
  confirm: []
}>()

const { t } = useI18n()
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t("curated.deleteCard") }}</DialogTitle>
        <DialogDescription class="text-pretty">
          {{ t("curated.deleteConfirm", { label }) }}
        </DialogDescription>
      </DialogHeader>
      <p v-if="error" class="text-sm text-destructive" role="alert">
        {{ error }}
      </p>
      <DialogFooter class="gap-3">
        <DialogClose as-child>
          <Button type="button" variant="outline" class="rounded-2xl" :disabled="busy">
            {{ t("curated.cancel") }}
          </Button>
        </DialogClose>
        <Button
          type="button"
          variant="destructive"
          class="rounded-2xl"
          :disabled="busy"
          @click="emit('confirm')"
        >
          {{ busy ? t("curated.deleteWorking") : t("curated.confirmDelete") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

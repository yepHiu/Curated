<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"

const props = defineProps<{
  book: {
    title: string
    sourceFileName: string
    location: string
    pageCount: number
    addedAt: string
    updatedAt: string
  }
}>()
const open = defineModel<boolean>("open", { required: true })
const { t } = useI18n()

function dateLabel(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString()
}

const rows = computed(() => [
  { label: t("bookBrowser.sourceFile"), value: props.book.sourceFileName || "—" },
  { label: t("bookBrowser.location"), value: props.book.location || "—" },
  { label: t("bookBrowser.fileFormat"), value: props.book.sourceFileName.match(/\.([^.]+)$/)?.[1]?.toUpperCase() || "—" },
  { label: t("bookBrowser.pages"), value: String(props.book.pageCount) },
  { label: t("bookBrowser.addedAt"), value: dateLabel(props.book.addedAt) },
  { label: t("bookBrowser.updatedAt"), value: dateLabel(props.book.updatedAt) },
])
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent data-book-media-info class="max-h-[min(90dvh,40rem)] min-w-0 overflow-y-auto rounded-3xl border-border/70 sm:max-w-lg">
      <DialogHeader class="min-w-0 pr-5">
        <DialogTitle>{{ t("bookBrowser.mediaInfo") }}</DialogTitle>
        <DialogDescription class="break-words">{{ book.title }}</DialogDescription>
      </DialogHeader>
      <dl class="flex min-w-0 flex-col gap-4 py-2 text-sm">
        <div v-for="row in rows" :key="row.label" class="grid min-w-0 gap-1 sm:grid-cols-[6rem_minmax(0,1fr)] sm:gap-4">
          <dt class="text-muted-foreground">{{ row.label }}</dt>
          <dd class="min-w-0 select-text break-all leading-relaxed">{{ row.value }}</dd>
        </div>
      </dl>
      <DialogFooter>
        <DialogClose as-child>
          <Button type="button" variant="outline" class="rounded-full">{{ t("common.close") }}</Button>
        </DialogClose>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

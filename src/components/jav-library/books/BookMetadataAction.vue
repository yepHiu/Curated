<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Sparkles, Check } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Toggle } from "@/components/ui/toggle"
import { Badge } from "@/components/ui/badge"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog"
import BookMetadataFacts from "./BookMetadataFacts.vue"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { BOOK_METADATA_GROUP_SIZE, useBookMetadata, type MetadataBook } from "@/composables/use-book-metadata"
import { useAIService } from "@/services/ai-service"
import { useComicLibraryService } from "@/services/comic-library-service"
import { usePhotoLibraryService } from "@/services/photo-library-service"

const props = defineProps<{ kind: "comic" | "photo"; books: readonly MetadataBook[] }>()
const emit = defineEmits<{ applied: [] }>()
const { t, locale } = useI18n()
const { writeEnabled } = useExperimentalAgent()
const open = ref(false)
const comics = useComicLibraryService()
const photos = usePhotoLibraryService()
const { rows, busy, page, currentRows, reset, cancel, generate, apply, regenerate } = useBookMetadata(useAIService(), async id => {
  if (props.kind === "comic") await comics.loadComicDetail(id)
  else await photos.loadPhotoDetail(id)
  emit("applied")
})
const pageCount = computed(() => Math.ceil(rows.value.length / BOOK_METADATA_GROUP_SIZE))
const readyCount = computed(() => currentRows.value.filter(row => row.selected && row.status === "ready").length)
const canGenerate = computed(() => currentRows.value.some(row => row.selected && ["pending", "failed"].includes(row.status)))
const mockMode = import.meta.env.VITE_USE_WEB_API !== "true"
watch(open, value => { if (value) reset(props.books); else cancel() })
watch(writeEnabled, value => { if (!value) cancel() })
function changePage(delta: number) { if (!busy.value) page.value += delta }
</script>

<template>
  <Button v-if="writeEnabled" variant="outline" class="min-h-11 shrink-0 rounded-full sm:min-h-8" :disabled="books.length === 0" @click="open = true">
    <Sparkles data-icon="inline-start" aria-hidden="true" />{{ t('bookAI.action') }}
  </Button>
  <Dialog :open="open" @update:open="value => { if (!busy) open = value }">
    <DialogContent class="flex max-h-[85dvh] flex-col sm:max-w-2xl" @escape-key-down="event => { if (busy) event.preventDefault() }" @interact-outside="event => { if (busy) event.preventDefault() }">
      <DialogHeader>
        <DialogTitle>{{ t('bookAI.action') }}</DialogTitle>
        <DialogDescription>{{ t('bookAI.description', { count: rows.length, size: BOOK_METADATA_GROUP_SIZE }) }}</DialogDescription>
      </DialogHeader>
      <Alert v-if="mockMode"><AlertDescription>{{ t('bookAI.mock') }}</AlertDescription></Alert>
      <div class="flex min-h-0 flex-col gap-4 overflow-y-auto pr-1" aria-live="polite">
        <article v-for="row in currentRows" :key="row.book.id" class="flex flex-col gap-3 rounded-xl border border-border p-4">
          <div class="flex items-start gap-3">
            <Toggle variant="outline" :model-value="row.selected" :disabled="busy || row.status === 'applied'" :aria-label="t('bookAI.select', { title: row.book.title })" @update:model-value="value => row.selected = value === true"><Check aria-hidden="true" /></Toggle>
            <div class="min-w-0 flex-1"><p class="break-words font-medium">{{ row.book.title }}</p><p class="break-all text-xs text-muted-foreground">{{ row.book.sourceFileName }}</p></div>
            <Badge variant="secondary">{{ t(`bookAI.status.${row.status}`) }}</Badge>
          </div>
          <template v-if="row.metadata">
            <div class="flex flex-col gap-2 text-sm">
              <p class="break-words text-muted-foreground">{{ t('bookAI.before') }} · {{ row.preview?.originalText }}</p>
              <p class="break-words font-medium">{{ t('bookAI.after') }} · {{ row.metadata.translatedTitle }}</p>
            </div>
            <details v-if="row.book.metadata?.cleanTitle" class="text-sm"><summary>{{ t('bookAI.saved') }}</summary><BookMetadataFacts :metadata="row.book.metadata" evidence /></details>
            <BookMetadataFacts :metadata="row.metadata" evidence />
          </template>
          <Alert v-if="row.error" variant="destructive"><AlertDescription>{{ row.error }}</AlertDescription></Alert>
          <Button v-if="['ready', 'failed', 'unchanged'].includes(row.status)" variant="ghost" class="w-fit" :disabled="busy" @click="regenerate(row)">{{ t('bookAI.regenerate') }}</Button>
        </article>
      </div>
      <div v-if="pageCount > 1" class="flex items-center justify-between gap-2">
        <Button variant="ghost" :disabled="busy || page === 0" @click="changePage(-1)">{{ t('bookAI.previous') }}</Button>
        <span class="text-sm text-muted-foreground">{{ page + 1 }} / {{ pageCount }}</span>
        <Button variant="ghost" :disabled="busy || page + 1 >= pageCount" @click="changePage(1)">{{ t('bookAI.next') }}</Button>
      </div>
      <DialogFooter class="flex-wrap">
        <Button v-if="busy" variant="outline" @click="cancel">{{ t('bookAI.stop') }}</Button>
        <Button v-else variant="outline" @click="open = false">{{ t('common.close') }}</Button>
        <Button variant="outline" :disabled="busy || !canGenerate || !writeEnabled" @click="generate(kind, locale)">{{ t('bookAI.generate') }}</Button>
        <Button :disabled="busy || !readyCount || !writeEnabled" @click="apply">{{ t('bookAI.apply', { count: readyCount }) }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import type { ComicBook } from "@/domain/comic/types"
import BookPagePreviewGrid from "@/components/jav-library/books/BookPagePreviewGrid.vue"
defineProps<{ comic: ComicBook }>()
const emit = defineEmits<{ openReader: [pageIndex: number] }>()
</script>
<template>
  <BookPagePreviewGrid :book-id="comic.id" :pages="comic.pages ?? []" :total="comic.pageCount" kind="comics" @open="emit('openReader', $event)" />
</template>

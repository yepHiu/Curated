<script setup lang="ts">
import type { PhotoBook } from "@/domain/photo/types"
import BookPagePreviewGrid from "@/components/jav-library/books/BookPagePreviewGrid.vue"
defineProps<{ photo: PhotoBook }>()
const emit = defineEmits<{ openViewer: [pageIndex: number] }>()
</script>
<template>
  <BookPagePreviewGrid :book-id="photo.id" :pages="photo.pages ?? []" :total="photo.pageCount" kind="photos" @open="emit('openViewer', $event)" />
</template>

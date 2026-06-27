<script setup lang="ts">
import { computed, onMounted } from "vue"
import MovieImportDialog from "@/components/jav-library/MovieImportDialog.vue"
import ComicImportDialog from "@/components/jav-library/ComicImportDialog.vue"
import { useComicLibraryService } from "@/services/comic-library-service"

const comicService = useComicLibraryService()
const comicLibraryEnabled = computed(() => comicService.comicLibraryEnabled.value)

onMounted(() => {
  void Promise.resolve(comicService.refreshSettings()).catch((error) => {
    console.warn("[import-menu] comic settings refresh failed", error)
  })
})
</script>

<template>
  <div data-import-menu class="flex min-w-0 flex-wrap items-center justify-end gap-2">
    <MovieImportDialog />
    <ComicImportDialog v-if="comicLibraryEnabled" />
  </div>
</template>

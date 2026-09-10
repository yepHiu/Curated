<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { FilePlus2 } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import MovieImportDialog from "./MovieImportDialog.vue"
import { useComicLibraryService } from "@/services/comic-library-service"
import { usePhotoLibraryService } from "@/services/photo-library-service"

const ComicImportDialog = defineAsyncComponent(() => import("./ComicImportDialog.vue"))
const PhotoImportPanel = defineAsyncComponent(() => import("./PhotoImportPanel.vue"))
const { t } = useI18n()
const comicService = useComicLibraryService()
const photoService = usePhotoLibraryService()
const open = ref(false)
const active = ref("movie")
const busy = ref(false)
const comicEnabled = computed(() => comicService.comicLibraryEnabled.value)
const photoEnabled = computed(() => photoService.photoLibraryEnabled.value)

/** Refresh gates independently; a failed settings request cannot enable a Tab. */
async function refreshGates() {
  await Promise.allSettled([comicService.refreshSettings(), photoService.refreshSettings()])
}
onMounted(refreshGates)
watch(open, (value) => { if (value) { active.value = "movie"; void refreshGates() } })
watch([comicEnabled, photoEnabled, busy], () => {
  if (busy.value) return
  if ((active.value === "comic" && !comicEnabled.value) || (active.value === "photo" && !photoEnabled.value)) active.value = "movie"
})
/** Keep the current upload mounted until it has returned its result. */
function updateOpen(value: boolean) { if (!busy.value) open.value = value }
function selectMedia(value: string | number) { if (!busy.value) active.value = String(value) }
function finishImport() { busy.value = false; open.value = false }
</script>

<template>
  <div data-import-menu>
    <Dialog :open="open" @update:open="updateOpen">
      <DialogTrigger as-child>
        <Button data-import-trigger type="button" variant="ghost" class="min-h-11 rounded-full" :aria-label="t('import.mediaTrigger')">
          <FilePlus2 data-icon="inline-start" />{{ t("import.mediaTrigger") }}
        </Button>
      </DialogTrigger>
      <DialogContent class="max-h-[calc(100dvh-2rem)] min-w-0 overflow-y-auto rounded-3xl sm:max-w-2xl" :show-close-button="!busy"
        @escape-key-down="busy && $event.preventDefault()" @interact-outside="busy && $event.preventDefault()">
        <DialogHeader>
          <DialogTitle>{{ t("import.mediaTrigger") }}</DialogTitle>
          <DialogDescription class="sr-only">{{ t("import.mediaDescription") }}</DialogDescription>
        </DialogHeader>
        <Tabs :model-value="active" class="min-w-0 gap-4" @update:model-value="selectMedia">
          <TabsList class="w-full" :aria-label="t('import.mediaType')">
            <TabsTrigger value="movie" class="flex-1" :disabled="busy">{{ t("import.trigger") }}</TabsTrigger>
            <TabsTrigger v-if="comicEnabled" value="comic" class="flex-1" :disabled="busy">{{ t("import.comicTrigger") }}</TabsTrigger>
            <TabsTrigger v-if="photoEnabled" value="photo" class="flex-1" :disabled="busy">{{ t("import.photoTrigger") }}</TabsTrigger>
          </TabsList>
          <TabsContent value="movie" force-mount v-show="active === 'movie'">
            <MovieImportDialog embedded :open="open && active === 'movie'" @busy="busy = $event" @completed="finishImport" />
          </TabsContent>
          <TabsContent v-if="comicEnabled || (busy && active === 'comic')" value="comic" force-mount v-show="active === 'comic'">
            <ComicImportDialog embedded :open="open && active === 'comic'" @busy="busy = $event" @completed="finishImport" />
          </TabsContent>
          <TabsContent v-if="photoEnabled || (busy && active === 'photo')" value="photo" force-mount v-show="active === 'photo'">
            <PhotoImportPanel :active="open && active === 'photo'" @busy="busy = $event" @completed="finishImport" />
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  </div>
</template>

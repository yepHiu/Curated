<script setup lang="ts">
import type { WishlistItem } from "@/domain/wishlist/types"
import { Badge } from "@/components/ui/badge"
import MediaStill from "@/components/jav-library/MediaStill.vue"
import { useI18n } from "vue-i18n"
defineProps<{ item: WishlistItem; image?: string }>()
defineEmits<{ open: [id: string] }>()
const { t } = useI18n()
</script>
<template>
  <button type="button" class="group flex min-w-0 flex-col gap-2 rounded-lg text-left focus-visible:outline-2 focus-visible:outline-ring" @click="$emit('open', item.id)">
    <div class="relative aspect-[2/3] w-full overflow-hidden rounded-lg bg-muted">
      <MediaStill v-if="image" :src="image" :alt="item.code" fit="contain" />
      <div v-else class="flex size-full items-center justify-center p-4 text-center font-medium text-muted-foreground">{{ item.code }}</div>
      <Badge v-if="item.enrichmentState !== 'ready'" class="absolute bottom-2 left-2 max-w-[calc(100%-1rem)] truncate" variant="secondary">{{ t(`wishlist.states.${item.enrichmentState}`) }}</Badge>
    </div>
    <span class="w-full truncate font-medium">{{ item.code }}</span>
    <span class="w-full truncate text-xs text-muted-foreground">{{ item.metadata.title || t('wishlist.awaiting') }}</span>
  </button>
</template>

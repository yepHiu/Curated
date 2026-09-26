<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { LibraryPathStorageStatusDTO } from "@/api/types"
import { Badge } from "@/components/ui/badge"

const props = defineProps<{
  title: string
  paths: readonly { id: string; title: string; path: string }[]
  defaultImportLibraryPathId: string
  storageStatuses?: readonly LibraryPathStorageStatusDTO[]
}>()
const { t } = useI18n()
const statuses = computed(() => new Map(props.storageStatuses?.map((item) => [item.libraryPathId, item])))
</script>

<template>
  <section class="flex flex-col gap-3 rounded-xl border border-border bg-card p-4" data-readonly-library-paths>
    <div class="flex flex-wrap items-center gap-2">
      <h3 class="text-sm font-semibold text-foreground">{{ title }}</h3>
      <Badge variant="secondary">{{ t('settings.libraryPathsReadOnly') }}</Badge>
    </div>
    <div v-for="path in paths" :key="path.id" class="flex flex-col gap-2 rounded-lg border border-border/50 bg-muted/5 p-4">
      <div class="flex flex-wrap items-center gap-2">
        <p class="min-w-0 break-words text-sm font-medium">{{ path.title || path.path }}</p>
        <Badge v-if="path.id === defaultImportLibraryPathId" variant="secondary">{{ t('settings.defaultImportPathLabel') }}</Badge>
      </div>
      <p class="break-all text-sm text-muted-foreground">{{ path.path }}</p>
      <p v-if="statuses.has(path.id)" class="text-xs leading-relaxed text-muted-foreground">
        {{ t(`settings.storageStatusMessages.${statuses.get(path.id)!.status}`) }}
      </p>
    </div>
    <p v-if="paths.length === 0" class="text-sm text-muted-foreground">{{ t('settings.libraryPathsReadOnlyEmpty') }}</p>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"

const { t } = useI18n()
</script>

<template>
  <div class="h-full overflow-y-auto pr-2">
    <NotFoundState
      :title="t('notFound.title')"
      :description="t('notFound.description')"
    />
  </div>
</template>

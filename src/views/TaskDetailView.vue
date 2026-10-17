<script setup lang="ts">
import { RouterLink, useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { ArrowLeft } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import TaskDetailContent from "@/components/jav-library/TaskDetailContent.vue"

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
</script>

<template>
  <div class="h-full min-h-0 overflow-y-auto">
    <div class="mx-auto flex max-w-6xl flex-col gap-5 px-1 py-3 sm:px-4 sm:py-5">
      <Button as-child variant="ghost" size="sm" class="self-start"><RouterLink :to="{ path: '/tasks', query: route.query }"><ArrowLeft data-icon="inline-start" />{{ t('tasks.back') }}</RouterLink></Button>
      <TaskDetailContent :task-key="String(route.params.taskKey ?? '')" @restarted="router.replace({ params: { taskKey: $event }, query: route.query })" />
    </div>
  </div>
</template>

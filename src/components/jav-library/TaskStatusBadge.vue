<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Badge } from "@/components/ui/badge"
import type { TaskState } from "@/services/contracts/task-center-service"
const props = defineProps<{ state: TaskState }>()
const { t } = useI18n()
const variant = computed(() => props.state === "succeeded" ? "success" : ["failed", "blocked", "partial"].includes(props.state) ? "warning" : props.state === "running" ? "info" : "secondary")
</script>
<template><Badge :variant="variant">{{ t(`tasks.states.${state}`) }}</Badge></template>

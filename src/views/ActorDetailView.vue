<script setup lang="ts">
import { computed } from "vue"
import { useRoute } from "vue-router"
import ActorDetailPage from "@/components/jav-library/ActorDetailPage.vue"

const route = useRoute()

const actorName = computed(() => {
  const raw = route.params.actorName
  if (Array.isArray(raw)) {
    return raw[0] ?? ""
  }
  return raw ?? ""
})
</script>

<template>
  <ActorDetailPage :actor-name="actorName" />
</template>

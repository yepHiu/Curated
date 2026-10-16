<script setup lang="ts">
import { computed } from "vue"
import PlaybackInfoPanel from "@/components/jav-library/PlaybackInfoPanel.vue"
import { nativePlaybackInfoMessages, playbackInfoGroups, transferPolyline, type NativeInfoLanguage } from "@/lib/native-playback-info"
import type { NativePlayerState } from "../../../electron/native-player-contract"

const props = defineProps<{ state?: NativePlayerState; lang: NativeInfoLanguage; busy: boolean; feedback: string }>()
const emit = defineEmits<{ close: []; copy: []; save: [] }>()
const labels = computed(() => nativePlaybackInfoMessages[props.lang])
const groups = computed(() => playbackInfoGroups(props.state, props.lang))
const history = computed(() => transferPolyline(props.state?.diagnostics?.network?.history ?? []))
</script>

<template>
  <PlaybackInfoPanel data-native-playback-info :labels="labels" :groups="groups" :history="history" :busy="busy" :feedback="feedback"
    @close="emit('close')" @copy="emit('copy')" @save="emit('save')" />
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { Copy, Download, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { nativePlaybackInfoMessages, playbackInfoGroups, transferPolyline, type NativeInfoLanguage } from "@/lib/native-playback-info"
import type { NativePlayerState } from "../../../electron/native-player-contract"

const props = defineProps<{ state?: NativePlayerState; lang: NativeInfoLanguage; busy: boolean; feedback: string }>()
const emit = defineEmits<{ close: []; copy: []; save: [] }>()
const panel = ref<HTMLElement>()
const labels = computed(() => nativePlaybackInfoMessages[props.lang])
const groups = computed(() => playbackInfoGroups(props.state, props.lang))
const history = computed(() => transferPolyline(props.state?.diagnostics?.network?.history ?? []))
onMounted(() => panel.value?.focus({ preventScroll: true }))
</script>

<template>
  <aside ref="panel" data-native-playback-info tabindex="-1" :aria-label="labels.title"
    class="absolute left-4 top-20 flex w-[30rem] max-w-[calc(100%-2rem)] flex-col overflow-hidden rounded-xl border border-border bg-background text-foreground shadow-lg outline-none sm:left-5 [@media(max-height:560px)]:top-4"
    @click.stop @dblclick.stop @keydown.stop @keydown.esc.prevent="emit('close')" @keydown.d.prevent="emit('close')">
    <div class="flex shrink-0 items-center justify-between gap-3 px-4 py-2">
      <h2 class="text-sm font-semibold">{{ labels.title }}</h2>
      <div class="flex items-center gap-1">
        <Button variant="ghost" size="icon-sm" class="rounded-full" :disabled="busy" :aria-label="labels.copy" :title="labels.copy" @click="emit('copy')"><Copy /></Button>
        <Button variant="ghost" size="icon-sm" class="rounded-full" :disabled="busy" :aria-label="labels.save" :title="labels.save" @click="emit('save')"><Download /></Button>
        <Button variant="ghost" size="icon-sm" class="rounded-full" :aria-label="labels.close" :title="labels.close" @click="emit('close')"><X /></Button>
      </div>
    </div>
    <Separator />
    <div class="min-h-0 overflow-y-auto overscroll-contain px-4 py-3 text-xs [overflow-anchor:none] [scrollbar-gutter:stable]">
      <p role="status" aria-live="polite" class="text-muted-foreground" :class="feedback ? 'mb-3' : ''">{{ feedback }}</p>
      <section v-for="(group, index) in groups" :key="group.title" :aria-label="group.title" class="flex flex-col gap-2">
        <Separator v-if="index" class="my-3" />
        <h3 class="font-medium text-muted-foreground">{{ group.title }}</h3>
        <dl class="grid grid-cols-[minmax(0,0.9fr)_minmax(0,1.5fr)] gap-x-4 gap-y-1.5">
          <template v-for="row in group.rows" :key="row.label">
            <dt class="text-muted-foreground">{{ row.label }}</dt>
            <dd class="break-words font-mono tabular-nums">
              <span v-if="row.metrics" class="inline-flex items-center gap-2 whitespace-nowrap">
                <template v-for="(measurement, measurementIndex) in row.metrics" :key="measurementIndex">
                  <span v-if="measurementIndex" class="text-muted-foreground">/</span>
                  <span class="inline-flex items-center gap-1">
                    <span data-info-number class="inline-block h-4 w-[9ch] shrink-0 text-left">{{ measurement.number }}</span>
                    <span v-if="measurement.reserveUnit" data-info-unit class="inline-block h-4 w-[5ch] shrink-0">{{ measurement.unit }}</span>
                  </span>
                </template>
              </span>
              <template v-else>{{ row.value }}</template>
            </dd>
          </template>
        </dl>
      </section>
      <svg viewBox="0 0 240 36" role="img" :aria-label="labels.history" class="mt-3 h-10 w-full text-primary" preserveAspectRatio="none">
        <polyline :points="history" fill="none" stroke="currentColor" stroke-width="1.5" vector-effect="non-scaling-stroke" />
      </svg>
    </div>
  </aside>
</template>

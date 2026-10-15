import { computed, onBeforeUnmount, ref, shallowRef, watch, type Ref } from "vue"
import { useSubtitleService } from "@/services/subtitle-service"
import { readSubtitlePreferences, saveSubtitlePreferences } from "@/lib/subtitle-preferences"
import { activeSubtitleText, parseSRT, type SubtitleCue } from "../../electron/subtitle-format"
import { resolveApiBaseUrl } from "@/api/http-client"
export function usePlayerSubtitles(movieId: Ref<string>, fileId: Ref<string | undefined>, time: Ref<number>) {
  const service = useSubtitleService()
  const preferences = ref(readSubtitlePreferences())
  const zh = shallowRef<SubtitleCue[]>([]), ja = shallowRef<SubtitleCue[]>([])
  const available = ref<string[]>([]), error = ref("")
  let revision = 0
  let preferenceScope: string | undefined
  watch([movieId, fileId], async ([movie, file]) => {
    const seq = ++revision; zh.value = []; ja.value = []; available.value = []; error.value = ""
    preferenceScope = undefined; preferences.value = readSubtitlePreferences()
    try {
      const assets = await service.assets(movie, file)
      const loaded = await Promise.all(assets.filter(asset => asset.format === "srt" && !asset.missing).map(async asset => ({ language: asset.language, cues: parseSRT(await service.content(asset.id)) })))
      if (seq !== revision) return
      preferenceScope = `${new URL(resolveApiBaseUrl(import.meta.env), window.location.origin).origin}:${file ?? movie}:${assets.filter(asset => asset.format === "srt" && !asset.missing).map(asset => asset.id).sort().join(",")}`
      preferences.value = readSubtitlePreferences(preferenceScope)
      for (const track of loaded) { if (track.language === "zh-CN") zh.value = track.cues; else if (track.language === "ja") ja.value = track.cues }
      available.value = loaded.map(track => track.language)
    } catch (e) { if (seq === revision) error.value = e instanceof Error ? e.message : String(e) }
  }, { immediate: true })
  watch(preferences, value => saveSubtitlePreferences(value, preferenceScope), { deep: true })
  onBeforeUnmount(() => { revision++ })
  const text = computed(() => {
    const { mode, offset } = preferences.value
    const current = time.value - offset // Positive offset means the subtitle appears later.
    return [mode === "zh-CN" || mode === "bilingual" ? activeSubtitleText(zh.value, current) : "", mode === "ja" || mode === "bilingual" ? activeSubtitleText(ja.value, current) : ""].filter(Boolean).join("\n")
  })
  return { preferences, available, text, error }
}

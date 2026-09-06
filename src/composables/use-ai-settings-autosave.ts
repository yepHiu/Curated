import { computed, onBeforeUnmount, ref, watch } from "vue"

/** Serializes writes and keeps the latest draft editable while a request is pending. */
export function useAISettingsAutosave<T>(options: {
  read: () => T
  enabled: () => boolean
  valid: () => boolean
  save: (value: T) => Promise<void>
  delay?: (next: T, previous: T) => number
  onDetachedError: (error: string) => void
}) {
  const baseline = ref("")
  const saving = ref(false)
  const error = ref("")
  const saved = ref(false)
  const draft = computed(() => JSON.stringify(options.read()))
  const dirty = computed(() => draft.value !== baseline.value)
  let timer: ReturnType<typeof setTimeout> | undefined
  let pending: Promise<boolean> | null = null
  let detached = false

  function initialize(value: T) {
    baseline.value = JSON.stringify(value)
  }
  function clearTimer() {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
  }
  async function drain(): Promise<boolean> {
    while (dirty.value && options.enabled()) {
      if (!options.valid()) return false
      const value = options.read()
      const snapshot = JSON.stringify(value)
      saving.value = true
      error.value = ""
      try {
        await options.save(value)
        baseline.value = snapshot
        saved.value = true
      } catch (err) {
        error.value = err instanceof Error ? err.message : String(err)
        if (detached) options.onDetachedError(error.value)
        return false
      } finally {
        saving.value = false
      }
    }
    return !dirty.value
  }
  function flush(): Promise<boolean> {
    clearTimer()
    if (pending) return pending
    if (!options.enabled() || !dirty.value) return Promise.resolve(true)
    pending = drain().finally(() => { pending = null; clearTimer() })
    return pending
  }
  watch(draft, (next, previous) => {
    clearTimer()
    if (!options.enabled() || !dirty.value) return
    saved.value = false
    error.value = ""
    if (!options.valid() || pending) return
    const delay = options.delay?.(JSON.parse(next) as T, JSON.parse(previous) as T) ?? 550
    if (delay === 0) void flush()
    else timer = setTimeout(() => { void flush() }, delay)
  }, { flush: "sync" })
  onBeforeUnmount(() => {
    detached = true
    // Tab switches must not discard an edit still inside the debounce window.
    void flush()
  })
  return { saving, error, saved, dirty, initialize, flush }
}

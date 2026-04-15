import { readonly, ref, watch, type Ref } from "vue"
import type { HomepageDailyRecommendationsDTO } from "@/api/types"
import { useCurrentUtcDayKey } from "@/lib/current-utc-day-key"
import { useLibraryService } from "@/services/library-service"

type HomepageDailyRecommendationLoader = Pick<
  ReturnType<typeof useLibraryService>,
  "getHomepageDailyRecommendations"
>

export interface UseHomepageDailyRecommendationsOptions {
  libraryService?: HomepageDailyRecommendationLoader
  utcDayKey?: Ref<string>
}

export function useHomepageDailyRecommendations(
  options: UseHomepageDailyRecommendationsOptions = {},
) {
  const libraryService = options.libraryService ?? useLibraryService()
  const currentUtcDayKey = options.utcDayKey ?? useCurrentUtcDayKey().dayKey
  const snapshot = ref<HomepageDailyRecommendationsDTO | null>(null)
  const loading = ref(false)
  const error = ref<unknown>(null)
  let requestSeq = 0

  const refresh = async (options?: { preserveOnError?: boolean }) => {
    const preserveOnError = options?.preserveOnError ?? snapshot.value !== null
    const requestId = ++requestSeq
    loading.value = true

    try {
      const next = await libraryService.getHomepageDailyRecommendations()
      if (requestId !== requestSeq) {
        return snapshot.value
      }
      snapshot.value = next
      error.value = null
      return next
    } catch (err) {
      if (requestId !== requestSeq) {
        return snapshot.value
      }
      error.value = err
      if (!preserveOnError) {
        snapshot.value = null
      }
      return snapshot.value
    } finally {
      if (requestId === requestSeq) {
        loading.value = false
      }
    }
  }

  watch(
    currentUtcDayKey,
    (_next, previous) => {
      void refresh({
        preserveOnError: Boolean(previous && snapshot.value),
      })
    },
    { immediate: true },
  )

  return {
    utcDayKey: readonly(currentUtcDayKey),
    snapshot: readonly(snapshot),
    loading: readonly(loading),
    error: readonly(error),
    refresh,
  }
}

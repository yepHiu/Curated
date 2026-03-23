import { ref, watch, type Ref } from "vue"
import { useResizeObserver } from "@vueuse/core"

const MAX_RELATED_VISIBLE = 6

/** 主内容区宽度 → 推荐区横向展示张数（1–6） */
export function widthToRelatedVisibleCount(width: number): number {
  if (!Number.isFinite(width) || width <= 0) return MAX_RELATED_VISIBLE
  if (width < 640) return 2
  if (width < 768) return 3
  if (width < 1024) return 4
  if (width < 1280) return 5
  return MAX_RELATED_VISIBLE
}

/**
 * 监听主内容区容器宽度，实时更新可见推荐条数（不设 debounce）。
 */
export function useRelatedVisibleCount(targetRef: Ref<HTMLElement | null>) {
  const visibleCount = ref(MAX_RELATED_VISIBLE)

  function applyFromEntries(entries: readonly ResizeObserverEntry[]) {
    const w = entries[0]?.contentRect.width
    if (w != null && w > 0) {
      visibleCount.value = widthToRelatedVisibleCount(w)
    }
  }

  useResizeObserver(targetRef, (entries) => applyFromEntries(entries))

  watch(
    targetRef,
    (el) => {
      if (!el) return
      const w = el.getBoundingClientRect().width
      if (w > 0) {
        visibleCount.value = widthToRelatedVisibleCount(w)
      }
    },
    { flush: "post" },
  )

  return { visibleCount }
}

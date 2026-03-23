import type { MaybeRefOrGetter, Ref } from "vue"
import { nextTick, ref, toValue, watch } from "vue"

export type UseUserTagSuggestKeyboardOptions = {
  showSuggestions: MaybeRefOrGetter<boolean>
  suggestions: MaybeRefOrGetter<readonly string[]>
  /** 联想列表根节点（用于 scrollIntoView），可与输入框分处 Teleport 内外 */
  listRootRef: Ref<HTMLElement | null>
  /** 回车且已高亮某项：直接提交该标签 */
  commitTag: (tag: string) => void | Promise<void>
  /** 无高亮时回车：提交输入框当前文本 */
  commitDraft: () => void | Promise<void>
}

/**
 * 标签模糊联想：↑/↓ 选择，Enter 提交选中项或当前输入；Escape 清除高亮。
 */
export function useUserTagSuggestKeyboard(options: UseUserTagSuggestKeyboardOptions) {
  const highlightIndex = ref(-1)

  function resetHighlight() {
    highlightIndex.value = -1
  }

  watch(
    () => ({
      show: toValue(options.showSuggestions),
      sig: toValue(options.suggestions).join("\u0001"),
    }),
    () => {
      resetHighlight()
    },
  )

  async function scrollHighlightIntoView() {
    const idx = highlightIndex.value
    const root = options.listRootRef.value
    if (idx < 0 || !root) {
      return
    }
    await nextTick()
    root
      .querySelector<HTMLElement>(`[data-tag-suggest-idx="${idx}"]`)
      ?.scrollIntoView({ block: "nearest" })
  }

  function onTagSuggestKeydown(e: KeyboardEvent) {
    const show = toValue(options.showSuggestions)
    const list = toValue(options.suggestions)
    if (!show || list.length === 0) {
      if (e.key === "Enter") {
        e.preventDefault()
        void options.commitDraft()
      }
      return
    }
    const n = list.length
    if (e.key === "ArrowDown") {
      e.preventDefault()
      if (highlightIndex.value < 0) {
        highlightIndex.value = 0
      } else {
        highlightIndex.value = Math.min(highlightIndex.value + 1, n - 1)
      }
      void scrollHighlightIntoView()
      return
    }
    if (e.key === "ArrowUp") {
      e.preventDefault()
      if (highlightIndex.value <= 0) {
        highlightIndex.value = -1
      } else {
        highlightIndex.value -= 1
      }
      void scrollHighlightIntoView()
      return
    }
    if (e.key === "Enter") {
      e.preventDefault()
      const idx = highlightIndex.value
      if (idx >= 0 && idx < n) {
        void options.commitTag(list[idx]!)
        resetHighlight()
      } else {
        void options.commitDraft()
      }
      return
    }
    if (e.key === "Escape") {
      resetHighlight()
    }
  }

  return { highlightIndex, onTagSuggestKeydown, resetHighlight }
}

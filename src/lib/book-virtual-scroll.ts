/**
 * 漫画/写真墙虚拟滚动分块。行高与海报 loading 策略复用影片墙辅助函数，
 * 不改 VirtualMovieMasonry。
 */

export const BOOK_VIRTUAL_ROWS_PER_CHUNK = 4
export const BOOK_VIRTUAL_BUFFER_CHUNKS = 8
export const BOOK_VIRTUAL_BUFFER_PX = 1400

export interface BookVirtualChunk<T> {
  id: string
  items: T[]
  /** 稳定串，供 DynamicScrollerItem 测量缓存。 */
  sizeKey: string
}

/**
 * 按「列数 × 4 行」容量把书目切成虚拟滚动块。
 * 除最后一块外每块都是整行满列，避免跨块阶梯断行。
 */
export function buildBookVirtualChunks<T>(
  items: readonly T[],
  chunkCapacity: number,
  sizeKeyOf: (item: T) => string,
): BookVirtualChunk<T>[] {
  const total = items.length
  if (total === 0) {
    return []
  }
  const size = Math.max(1, chunkCapacity)
  const chunks: BookVirtualChunk<T>[] = []
  for (let offset = 0; offset < total; offset += size) {
    const slice = items.slice(offset, offset + size)
    chunks.push({
      id: `chunk-${offset}-${size}`,
      items: slice,
      sizeKey: slice.map(sizeKeyOf).join("|"),
    })
  }
  return chunks
}

/**
 * 从 computed grid-template-columns 解析真实列数，与 auto-fill 实际布局对齐。
 */
export function parseBookGridColumnCount(el: HTMLElement): number {
  const raw = getComputedStyle(el).gridTemplateColumns
  if (!raw || raw === "none") return 0
  const trimmed = raw.trim()
  const repeatMatch = trimmed.match(/repeat\s*\(\s*(\d+)/i)
  if (repeatMatch) {
    return Math.max(1, parseInt(repeatMatch[1]!, 10))
  }
  const parts = trimmed.split(/\s+/).filter((part) => part.length > 0)
  return parts.length
}

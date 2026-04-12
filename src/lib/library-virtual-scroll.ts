const POSTER_ASPECT_RATIO = 537 / 358
const CARD_MAX_WIDTH_PX = 304
const CARD_MIN_WIDTH_PX = 188
const CARD_VERTICAL_CHROME_PX = 100

type PosterFetchPriority = "high" | "low" | "auto"

export function estimateVirtualMovieCardHeight(trackWidthPx: number): number {
  const width = Math.max(CARD_MIN_WIDTH_PX, Math.min(trackWidthPx, CARD_MAX_WIDTH_PX))
  return Math.ceil(width * POSTER_ASPECT_RATIO + CARD_VERTICAL_CHROME_PX)
}

export function estimateVirtualMovieChunkHeight(options: {
  containerWidth: number
  columnCount: number
  rowsPerChunk: number
  gapPx: number
}): number {
  const { containerWidth, columnCount, rowsPerChunk, gapPx } = options
  const safeColumns = Math.max(1, columnCount)
  const safeRows = Math.max(1, rowsPerChunk)
  const safeGap = Math.max(0, gapPx)
  const usableWidth = Math.max(0, containerWidth - safeGap * Math.max(0, safeColumns - 1))
  const trackWidth = usableWidth > 0 ? usableWidth / safeColumns : CARD_MIN_WIDTH_PX
  const cardHeight = estimateVirtualMovieCardHeight(trackWidth)

  return safeRows * cardHeight + safeRows * safeGap
}

export function getVirtualMovieFocusChunkIndex(options: {
  scrollTop: number
  viewportHeight: number
  chunkHeight: number
}): number {
  const { scrollTop, viewportHeight, chunkHeight } = options
  const safeChunkHeight = Math.max(1, chunkHeight)
  const focusTop = Math.max(0, scrollTop) + Math.max(0, viewportHeight) * 0.5
  return Math.max(0, Math.floor(focusTop / safeChunkHeight))
}

export function resolveVirtualMoviePosterLoadPolicy(
  chunkIndex: number,
  focusChunkIndex: number,
): { loading: "lazy" | "eager"; fetchPriority: PosterFetchPriority } {
  const distance = Math.abs(chunkIndex - focusChunkIndex)

  if (distance === 0) {
    return { loading: "eager", fetchPriority: "high" }
  }
  if (distance <= 1) {
    return { loading: "eager", fetchPriority: "auto" }
  }
  return { loading: "lazy", fetchPriority: "low" }
}

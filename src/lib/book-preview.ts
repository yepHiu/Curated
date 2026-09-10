/** Two rows sized to the actual content width; DOM and decoded images stay bounded. */
export function previewColumns(width: number): number {
  return Math.max(2, Math.min(10, Math.floor((Math.max(0, width) + 12) / 132)))
}
export function previewStart(index: number, size: number, total: number): number {
  const safe = Math.min(Math.max(0, total - 1), Math.max(0, Math.floor(index) || 0))
  return Math.floor(safe / size) * size
}

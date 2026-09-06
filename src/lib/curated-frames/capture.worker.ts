/// <reference lib="webworker" />
declare const self: DedicatedWorkerGlobalScope
self.onmessage = async (event: MessageEvent<{ id: number; bitmap: ImageBitmap }>) => {
  const { id, bitmap } = event.data
  try {
    const canvas = new OffscreenCanvas(bitmap.width, bitmap.height)
    const context = canvas.getContext('2d')
    if (!context) throw new Error('2d context unavailable')
    context.drawImage(bitmap, 0, 0)
    bitmap.close()
    const blob = await canvas.convertToBlob({ type: 'image/png' })
    self.postMessage({ id, blob })
    canvas.width = canvas.height = 1
  } catch {
    bitmap.close()
    self.postMessage({ id, error: true })
  }
}
export {}

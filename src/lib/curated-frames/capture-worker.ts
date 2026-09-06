let worker: Worker | undefined
let nextId = 0
let idleTimer: ReturnType<typeof setTimeout> | undefined
const pending = new Map<number, { resolve: (blob: Blob) => void; reject: () => void; timer: ReturnType<typeof setTimeout> }>()

function dispose() {
  worker?.terminate(); worker = undefined
  for (const item of pending.values()) { clearTimeout(item.timer); item.reject() }
  pending.clear()
}

export async function encodeCaptureOffThread(canvas: HTMLCanvasElement): Promise<Blob> {
  // The canvas is already frozen at gesture time; never re-read a playing video.
  const bitmap = await createImageBitmap(canvas)
  clearTimeout(idleTimer)
  try {
    worker ??= new Worker(new URL('./capture.worker.ts', import.meta.url), { type: 'module' })
    worker.onerror = dispose
    worker.onmessage = (event: MessageEvent<{ id: number; blob?: Blob }>) => {
      const item = pending.get(event.data.id)
      if (!item) return
      clearTimeout(item.timer); pending.delete(event.data.id)
      if (event.data.blob) item.resolve(event.data.blob); else item.reject()
      if (!pending.size) idleTimer = setTimeout(dispose, 30_000)
    }
    return await new Promise<Blob>((resolve, reject) => {
      const id = ++nextId
      pending.set(id, { resolve, reject: () => reject(new Error('capture worker unavailable')), timer: setTimeout(dispose, 15_000) })
      try { worker!.postMessage({ id, bitmap }, [bitmap]) }
      catch { bitmap.close(); dispose() }
    })
  } catch (error) { bitmap.close(); throw error }
}

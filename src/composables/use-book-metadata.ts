import { computed, onBeforeUnmount, ref } from "vue"
import type { AIActionPreviewDTO, BookMetadata } from "@/api/types"
import type { AIService } from "@/services/contracts/ai-service"
import { isBookMetadata } from "@/lib/book-metadata"

export type MetadataBook = { id: string; title: string; originalTitle?: string; sourceFileName: string; metadata?: BookMetadata }
export type MetadataRow = {
  book: MetadataBook
  selected: boolean
  status: "pending" | "generating" | "ready" | "applying" | "applied" | "unchanged" | "failed"
  preview?: AIActionPreviewDTO
  metadata?: BookMetadata
  error?: string
}

// Small review groups keep the oldest preview within the existing confirmation TTL.
export const BOOK_METADATA_GROUP_SIZE = 5

export function useBookMetadata(service: Pick<AIService, "runAction" | "confirmTool">, refresh: (id: string) => Promise<unknown>) {
  const rows = ref<MetadataRow[]>([])
  const busy = ref(false)
  const page = ref(0)
  const currentRows = computed(() => rows.value.slice(page.value * BOOK_METADATA_GROUP_SIZE, (page.value + 1) * BOOK_METADATA_GROUP_SIZE))
  let controller: AbortController | undefined
  let stopRequested = false

  function reset(books: readonly MetadataBook[]) {
    cancel()
    rows.value = books.map(book => ({ book: { id: book.id, title: book.title, originalTitle: book.originalTitle, sourceFileName: book.sourceFileName, metadata: book.metadata ? JSON.parse(JSON.stringify(book.metadata)) as BookMetadata : undefined }, selected: true, status: "pending" }))
    page.value = 0
  }

  function cancel() {
    stopRequested = true
    controller?.abort()
  }

  async function generate(kind: "comic" | "photo", locale: string) {
    if (busy.value) return
    busy.value = true
    stopRequested = false
    controller = new AbortController()
    const signal = controller.signal
    try {
      for (const row of currentRows.value) {
        if (stopRequested) break
        if (!row.selected || ["applied", "unchanged", "ready"].includes(row.status)) continue
        row.status = "generating"
        row.error = undefined
        row.preview = undefined
        try {
          const preview = await service.runAction("recognize_book_metadata", { [kind === "comic" ? "comicId" : "photoId"]: row.book.id, targetLocale: locale }, signal)
          if (signal.aborted) { row.status = "pending"; break }
          const metadata = preview.arguments?.metadata
          if (!isBookMetadata(metadata)) throw new Error("Invalid book metadata response")
          // The server may have fresher values than the list used to open the dialog.
          const snapshot = preview.arguments?.snapshot
          if (typeof snapshot === "string") {
            const source: unknown = JSON.parse(snapshot)
            if (source && typeof source === "object") {
              const values = source as Record<string, unknown>
              if (typeof values.title === "string") row.book.title = values.title
              if (typeof values.originalTitle === "string") row.book.originalTitle = values.originalTitle
              if (typeof values.sourceFileName === "string") row.book.sourceFileName = values.sourceFileName
              row.book.metadata = isBookMetadata(values.metadata) ? values.metadata : undefined
            }
          }
          row.preview = preview
          row.metadata = metadata
          row.status = preview.noop ? "unchanged" : "ready"
        } catch (error) {
          row.status = signal.aborted ? "pending" : "failed"
          if (!signal.aborted) row.error = error instanceof Error ? error.message : String(error)
        }
      }
    } finally { busy.value = false; controller = undefined }
  }

  async function apply() {
    if (busy.value) return
    busy.value = true
    stopRequested = false
    try {
      for (const row of currentRows.value) {
        if (stopRequested) break
        if (!row.selected || row.status !== "ready" || !row.preview) continue
        const preview = row.preview
        row.status = "applying"
        row.error = undefined
        try {
          if (!preview.confirmToken || !preview.arguments) throw new Error("Missing confirmation")
          const result = await service.confirmTool({ name: preview.name, sessionId: preview.sessionId, confirmToken: preview.confirmToken, arguments: preview.arguments })
          if (!result.ok) throw new Error("Could not apply metadata")
          row.status = "applied"
        } catch (error) {
          // Keep the exact preview for an idempotent retry when the response was lost.
          row.status = "ready"
          row.error = error instanceof Error ? error.message : String(error)
          continue
        }
        try { await refresh(row.book.id) }
        catch (error) { row.error = error instanceof Error ? error.message : String(error) }
      }
    } finally { busy.value = false }
  }

  function regenerate(row: MetadataRow) {
    if (busy.value || row.status === "applied") return
    row.status = "pending"
    row.preview = undefined
    row.metadata = undefined
    row.error = undefined
  }

  onBeforeUnmount(cancel)
  return { rows, busy, page, currentRows, reset, cancel, generate, apply, regenerate }
}

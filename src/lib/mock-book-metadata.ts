import type { BookMetadata } from "@/api/types"
import { isBookMetadata } from "@/lib/book-metadata"

const key = "curated-mock-book-metadata-v1"
export function readMockBookMetadata(kind: string, id: string): BookMetadata | undefined {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(key) ?? "{}")[`${kind}:${id}`]
    return isBookMetadata(value) ? value : undefined
  } catch { return undefined }
}
export function writeMockBookMetadata(kind: string, id: string, metadata: BookMetadata) {
  let values: Record<string, unknown> = {}
  try { values = JSON.parse(localStorage.getItem(key) ?? "{}") } catch { /* Reset malformed mock storage. */ }
  localStorage.setItem(key, JSON.stringify({ ...values, [`${kind}:${id}`]: metadata }))
}
